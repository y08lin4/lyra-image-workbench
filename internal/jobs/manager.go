package jobs

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/y08lin4/lyra-image-workbench/internal/config"
	"github.com/y08lin4/lyra-image-workbench/internal/events"
	"github.com/y08lin4/lyra-image-workbench/internal/newapi"
	"github.com/y08lin4/lyra-image-workbench/internal/output"
	"github.com/y08lin4/lyra-image-workbench/internal/pixhost"
	"github.com/y08lin4/lyra-image-workbench/internal/settings"
	"github.com/y08lin4/lyra-image-workbench/internal/spaceconfig"
	"github.com/y08lin4/lyra-image-workbench/internal/uploads"
)

type Manager struct {
	store       *Store
	hub         *events.Hub
	settings    *settings.FileStore
	spaceConfig *spaceconfig.Store
	uploads     *uploads.Store
	output      *output.Store
	newapi      *newapi.Client
	pixhost     *pixhost.Client
	queue       chan jobRef
	mu          sync.Mutex
	cancels     map[string]context.CancelFunc
	secrets     map[string]jobSecret
}

type jobRef struct {
	SpaceToken string
	JobID      string
}

type jobSecret struct {
	APIKey string
}

func NewManager(store *Store, hub *events.Hub, settingsStore *settings.FileStore, spaceConfig *spaceconfig.Store, uploadStore *uploads.Store, outputStore *output.Store, newapiClient *newapi.Client) *Manager {
	m := &Manager{
		store:       store,
		hub:         hub,
		settings:    settingsStore,
		spaceConfig: spaceConfig,
		uploads:     uploadStore,
		output:      outputStore,
		newapi:      newapiClient,
		pixhost:     pixhost.NewClient(),
		queue:       make(chan jobRef, 256),
		cancels:     make(map[string]context.CancelFunc),
		secrets:     make(map[string]jobSecret),
	}
	go m.worker()
	return m
}

func (m *Manager) Recover() error {
	bySpace, err := m.store.AllSpacesJobs()
	if err != nil {
		return err
	}
	for token, list := range bySpace {
		for _, job := range list {
			switch job.Status {
			case StatusQueued, StatusRunning:
				_, _, _ = m.store.Update(token, job.ID, func(j *Job) {
					ApplyStatus(j, StatusInterrupted)
					ApplyStage(j, StageInterrupted)
					j.Progress = 100
					j.Error = "Runtime API key is only stored in the browser and is unavailable after server restart. Please retry from the browser."
					j.FinishedAt = time.Now()
				})
			}
		}
	}
	return nil
}

func (m *Manager) Create(spaceToken string, req CreateRequest) (Job, error) {
	if err := validateCreate(req); err != nil {
		return Job{}, err
	}
	provider, err := resolveProvider(req.Provider)
	if err != nil {
		return Job{}, err
	}
	model, err := resolveModel(provider, req.Model)
	if err != nil {
		return Job{}, err
	}
	spaceCfg, err := m.spaceConfig.Get(spaceToken)
	if err != nil {
		return Job{}, err
	}
	apiKey := runtimeAPIKey(req.RuntimeSecrets, provider)
	if apiKey == "" {
		apiKey = cloudAPIKey(spaceCfg, provider)
	}
	if apiKey == "" {
		if provider == config.BananaProvider {
			return Job{}, errors.New("banana API key is not configured; save it locally or upload it to cloud after enabling account protection")
		}
		return Job{}, errors.New("codex-key is not configured; save it locally or upload it to cloud after enabling account protection")
	}
	if req.Mode == ModeImageToImage {
		if len(req.UploadIDs) == 0 && len(req.References) == 0 {
			return Job{}, errors.New("图生图需要先上传参考图")
		}
		if len(req.References) == 0 {
			for _, id := range req.UploadIDs {
				if _, _, err := m.uploads.GetReferenceImage(spaceToken, id); err != nil {
					return Job{}, err
				}
			}
		}
	}
	now := time.Now()
	id, err := newJobID()
	if err != nil {
		return Job{}, err
	}
	var references []ReferenceSnapshot
	if req.Mode == ModeImageToImage {
		if len(req.References) > 0 {
			references, err = m.copyReferenceSnapshots(spaceToken, id, req.References)
		} else {
			references, err = m.snapshotReferenceImages(spaceToken, id, req.UploadIDs)
		}
		if err != nil {
			return Job{}, err
		}
	}
	adminCfg := m.settings.Get()
	ratio := normalizeRatio(req.Ratio)
	resolution := normalizeResolution(req.Resolution)
	quality := normalizeQuality(req.Quality)
	outputFormat := normalizeOutputFormat(req.OutputFormat)
	size := imageSize(ratio, resolution)
	if provider == config.BananaProvider {
		spec := bananaModelSpec(model)
		ratio = spec.Ratio
		resolution = spec.Resolution
		quality = "auto"
		outputFormat = "auto"
		size = spec.Size
	}
	job := Job{
		ID:           id,
		SpaceToken:   spaceToken,
		Provider:     provider,
		Model:        model,
		Mode:         req.Mode,
		Prompt:       strings.TrimSpace(req.Prompt),
		FramePrompts: normalizeFramePrompts(req.FramePrompts, clamp(req.Count, 1, 24, 1)),
		Ratio:        ratio,
		Resolution:   resolution,
		Quality:      quality,
		OutputFormat: outputFormat,
		Size:         size,
		Count:        clamp(req.Count, 1, 24, 1),
		Concurrency:  clamp(req.Concurrency, 1, 0, 1),
		UploadIDs:    append([]string{}, req.UploadIDs...),
		References:   references,
		Progress:     0,
		Results:      []Result{},
		DebugEnabled: adminCfg.DebugEnabled,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	ApplyStatus(&job, StatusQueued)
	ApplyStage(&job, StageQueued)
	if job.DebugEnabled {
		job.DebugLogs = append(job.DebugLogs, newDebugLog(-1, "info", "create", "任务已创建，Debug 日志已开启", map[string]any{
			"provider":    job.Provider,
			"model":       job.Model,
			"mode":        job.Mode,
			"ratio":       job.Ratio,
			"resolution":  job.Resolution,
			"quality":     job.Quality,
			"format":      job.OutputFormat,
			"size":        job.Size,
			"count":       job.Count,
			"concurrency": job.Concurrency,
		}))
	}
	if err := m.store.Save(job); err != nil {
		return Job{}, err
	}
	m.setJobSecret(job.ID, apiKey)
	m.enqueue(spaceToken, job.ID)
	m.publish(job.ID, "progress", eventPayload(job))
	return job, nil
}

func (m *Manager) List(spaceToken string, limit int) ([]Job, error) {
	return m.store.List(spaceToken, limit)
}

func (m *Manager) Get(spaceToken string, id string) (Job, bool, error) {
	return m.store.Get(spaceToken, id)
}

func (m *Manager) Stats(spaceToken string) (Stats, error) { return m.store.Stats(spaceToken) }

func (m *Manager) SetFavorite(spaceToken string, id string, favorite bool) (Job, error) {
	job, ok, err := m.store.Update(spaceToken, id, func(j *Job) {
		j.Favorite = favorite
	})
	if err != nil {
		return Job{}, err
	}
	if !ok {
		return Job{}, errors.New("任务不存在")
	}
	m.publish(id, "progress", eventPayload(job))
	return job, nil
}

func (m *Manager) UploadResultToPixhost(ctx context.Context, spaceToken string, id string, index int) (Job, Result, error) {
	job, ok, err := m.store.Get(spaceToken, id)
	if err != nil {
		return Job{}, Result{}, err
	}
	if !ok {
		return Job{}, Result{}, errors.New("任务不存在")
	}
	resultIndex := -1
	for i := range job.Results {
		if job.Results[i].Index == index {
			resultIndex = i
			break
		}
	}
	if resultIndex < 0 || !job.Results[resultIndex].OK || strings.TrimSpace(job.Results[resultIndex].ImageURL) == "" {
		return Job{}, Result{}, errors.New("任务图片不存在")
	}
	current := job.Results[resultIndex]
	if current.RemoteURL != "" {
		return job, current, nil
	}
	path, mime, err := m.resolveResultOutput(job, current)
	if err != nil {
		return Job{}, Result{}, err
	}
	uploaded, err := m.pixhost.UploadFile(ctx, path, mime, "image-2-"+id+"-"+strconv.Itoa(index+1)+"."+output.ExtensionFromMime(mime))
	if err != nil {
		var updated Result
		job, _, _ = m.store.Update(spaceToken, id, func(j *Job) {
			for i := range j.Results {
				if j.Results[i].Index == index {
					j.Results[i].UploadError = err.Error()
					updated = j.Results[i]
					break
				}
			}
		})
		if updated.Index == 0 && index != 0 {
			updated = current
			updated.UploadError = err.Error()
		}
		m.publish(id, "result", map[string]any{"result": updated, "job": job})
		return job, updated, err
	}
	var updated Result
	job, _, err = m.store.Update(spaceToken, id, func(j *Job) {
		for i := range j.Results {
			if j.Results[i].Index == index {
				j.Results[i].RemoteURL = uploaded.ShowURL
				j.Results[i].RemoteThumbURL = uploaded.ThumbURL
				j.Results[i].UploadError = ""
				updated = j.Results[i]
				break
			}
		}
	})
	if err != nil {
		return Job{}, Result{}, err
	}
	m.publish(id, "result", map[string]any{"result": updated, "job": job})
	return job, updated, nil
}

func (m *Manager) Retry(spaceToken string, id string, secrets RuntimeSecrets) (Job, error) {
	old, ok, err := m.store.Get(spaceToken, id)
	if err != nil {
		return Job{}, err
	}
	if !ok {
		return Job{}, errors.New("任务不存在")
	}
	return m.Create(spaceToken, CreateRequest{
		RuntimeSecrets: secrets,
		Provider:       old.Provider,
		Model:          old.Model,
		Mode:           old.Mode,
		Prompt:         old.Prompt,
		FramePrompts:   old.FramePrompts,
		Ratio:          old.Ratio,
		Resolution:     old.Resolution,
		Quality:        old.Quality,
		OutputFormat:   old.OutputFormat,
		Count:          old.Count,
		Concurrency:    old.Concurrency,
		UploadIDs:      old.UploadIDs,
		References:     old.References,
	})
}

func (m *Manager) Cancel(spaceToken string, id string) (Job, error) {
	running := false
	m.mu.Lock()
	if cancel := m.cancels[id]; cancel != nil {
		running = true
		cancel()
	}
	m.mu.Unlock()
	job, ok, err := m.store.Update(spaceToken, id, func(j *Job) {
		if j.Final() {
			return
		}
		ApplyStatus(j, StatusCancelled)
		ApplyStage(j, StageCancelled)
		j.Progress = 100
		j.FinishedAt = time.Now()
	})
	if err != nil {
		return Job{}, err
	}
	if !ok {
		return Job{}, errors.New("任务不存在")
	}
	if !running {
		m.clearJobSecret(id)
	}
	m.publish(id, "done", eventPayload(job))
	return job, nil
}

func (m *Manager) Delete(spaceToken string, id string) (Job, error) {
	m.mu.Lock()
	if cancel := m.cancels[id]; cancel != nil {
		cancel()
		delete(m.cancels, id)
	}
	delete(m.secrets, id)
	m.mu.Unlock()
	job, ok, err := m.store.Delete(spaceToken, id)
	if err != nil {
		return Job{}, err
	}
	if !ok {
		return Job{}, errors.New("任务不存在")
	}
	m.deleteReferenceSnapshots(spaceToken, job)
	m.publish(id, "done", map[string]any{"deleted": true, "job": job})
	return job, nil
}

func (m *Manager) Subscribe(jobID string) (<-chan events.Event, func()) {
	return m.hub.Subscribe(jobID)
}

func (m *Manager) enqueue(spaceToken string, jobID string) {
	m.queue <- jobRef{SpaceToken: spaceToken, JobID: jobID}
}

func (m *Manager) setJobSecret(jobID string, apiKey string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.secrets[jobID] = jobSecret{APIKey: strings.TrimSpace(apiKey)}
}

func (m *Manager) jobAPIKey(jobID string) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return strings.TrimSpace(m.secrets[jobID].APIKey)
}

func (m *Manager) clearJobSecret(jobID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.secrets, jobID)
}

func (m *Manager) worker() {
	for ref := range m.queue {
		m.run(ref.SpaceToken, ref.JobID)
	}
}

func (m *Manager) run(spaceToken string, jobID string) {
	job, ok, err := m.store.Get(spaceToken, jobID)
	if err != nil || !ok || job.Final() {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	m.mu.Lock()
	m.cancels[jobID] = cancel
	m.mu.Unlock()
	defer func() {
		cancel()
		m.mu.Lock()
		delete(m.cancels, jobID)
		delete(m.secrets, jobID)
		m.mu.Unlock()
	}()

	job, _, _ = m.store.Update(spaceToken, jobID, func(j *Job) {
		ApplyStatus(j, StatusRunning)
		ApplyStage(j, StagePreparing)
		j.Progress = 5
		j.StartedAt = time.Now()
	})
	m.publish(jobID, "progress", eventPayload(job))
	progressDone := make(chan struct{})
	go m.fakeProgress(spaceToken, jobID, progressDone)
	defer close(progressDone)

	results := m.runImages(ctx, spaceToken, jobID)
	select {
	case <-ctx.Done():
		job, _, _ = m.store.Get(spaceToken, jobID)
		if job.Status == StatusCancelled {
			m.publish(jobID, "done", eventPayload(job))
			return
		}
	default:
	}
	okCount := 0
	for _, result := range results {
		if result.OK {
			okCount++
		}
	}
	job, _, _ = m.store.Update(spaceToken, jobID, func(j *Job) {
		j.Results = results
		j.Progress = 100
		j.FinishedAt = time.Now()
		if okCount == j.Count {
			ApplyStatus(j, StatusSucceeded)
			ApplyStage(j, StageSucceeded)
		} else if okCount > 0 {
			ApplyStatus(j, StatusPartialFailed)
			ApplyStage(j, StagePartialFailed)
		} else if j.Status != StatusCancelled {
			ApplyStatus(j, StatusFailed)
			ApplyStage(j, StageFailed)
			j.Error = "没有图片生成成功"
		}
	})
	m.publish(jobID, "done", eventPayload(job))
}

func (m *Manager) runImages(ctx context.Context, spaceToken string, jobID string) []Result {
	job, _, _ := m.store.Get(spaceToken, jobID)
	results := make([]Result, job.Count)
	sem := make(chan struct{}, job.Concurrency)
	var wg sync.WaitGroup
	for i := 0; i < job.Count; i++ {
		idx := i
		results[idx] = NewResult(idx, StatusQueued, "")
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			result := m.generateOne(ctx, spaceToken, jobID, idx)
			results[idx] = result
			job, ok, _ := m.store.Update(spaceToken, jobID, func(j *Job) {
				upsertResult(j, result)
			})
			if ok {
				m.publish(jobID, "result", map[string]any{"result": result, "job": job})
			}
		}()
	}
	wg.Wait()
	return results
}

func (m *Manager) generateOne(ctx context.Context, spaceToken string, jobID string, index int) Result {
	started := time.Now()
	job, _, _ := m.store.Update(spaceToken, jobID, func(j *Job) {
		ApplyStage(j, StageSubmitting)
		if j.Progress < 12 {
			j.Progress = 12
		}
	})
	m.publish(jobID, "progress", eventPayload(job))
	admin := m.settings.Get()
	prompt := promptForImage(job, index)
	spaceCfg, err := m.spaceConfig.Get(spaceToken)
	if err != nil {
		return NewResult(index, StatusFailed, err.Error())
	}
	provider, err := resolveProvider(job.Provider)
	if err != nil {
		return NewResult(index, StatusFailed, err.Error())
	}
	model, err := resolveModel(provider, job.Model)
	if err != nil {
		return NewResult(index, StatusFailed, err.Error())
	}
	apiKey := m.jobAPIKey(jobID)
	skipImageParams := provider == config.BananaProvider
	if apiKey == "" {
		apiKey = cloudAPIKey(spaceCfg, provider)
		if apiKey != "" {
			m.setJobSecret(jobID, apiKey)
		}
	}
	if apiKey == "" {
		if provider == config.BananaProvider {
			return NewResult(index, StatusFailed, "banana API key is not configured; save it locally or upload it to cloud after enabling account protection")
		}
		return NewResult(index, StatusFailed, "codex-key is not configured; save it locally or upload it to cloud after enabling account protection")
	}
	inputs, err := m.inputImagesForJob(spaceToken, job)
	if err != nil {
		return NewResult(index, StatusFailed, err.Error())
	}
	job, _, _ = m.store.Update(spaceToken, jobID, func(j *Job) {
		ApplyStage(j, StageWaitingUpstream)
		if j.Progress < 15 {
			j.Progress = 15
		}
	})
	m.publish(jobID, "progress", eventPayload(job))
	m.appendDebugLog(spaceToken, jobID, index, "info", "upstream_request", "准备向 NewAPI 提交生图请求", map[string]any{
		"method":          "POST",
		"url":             upstreamEndpoint(admin.NewAPIBaseURL, job.Mode),
		"baseUrl":         admin.NewAPIBaseURL,
		"timeoutSec":      admin.TimeoutSec,
		"provider":        provider,
		"model":           model,
		"mode":            job.Mode,
		"authorization":   "Bearer " + maskSecret(apiKey),
		"contentType":     requestContentType(job.Mode),
		"payload":         debugPayload(job, model, skipImageParams),
		"inputImages":     debugInputImages(inputs),
		"skipImageParams": skipImageParams,
		"promptLength":    len([]rune(prompt)),
		"promptPreview":   compactDebugText(prompt, 120),
	})
	image, err := m.newapi.Generate(ctx, newapi.Request{Mode: string(job.Mode), BaseURL: admin.NewAPIBaseURL, APIKey: apiKey, Model: model, Prompt: prompt, Size: job.Size, Quality: job.Quality, OutputFormat: job.OutputFormat, SkipImageParams: skipImageParams, TimeoutSec: admin.TimeoutSec, InputImages: inputs})
	if err != nil {
		fields := map[string]any{
			"error":     err.Error(),
			"errorCode": ErrorMeta(err.Error()).Code,
			"errorText": ErrorMeta(err.Error()).Chinese,
			"errorType": fmt.Sprintf("%T", err),
			"elapsedMs": time.Since(started).Milliseconds(),
		}
		var upstreamErr newapi.UpstreamError
		if errors.As(err, &upstreamErr) {
			fields["httpStatus"] = upstreamErr.StatusCode
			fields["upstreamMessage"] = upstreamErr.Message
		}
		m.appendDebugLog(spaceToken, jobID, index, "error", "upstream_response", "NewAPI 请求失败", fields)
		return withElapsed(NewResult(index, StatusFailed, err.Error()), started)
	}
	m.appendDebugLog(spaceToken, jobID, index, "info", "upstream_response", "NewAPI 已返回图片数据", map[string]any{
		"httpStatus":    image.StatusCode,
		"contentType":   image.ContentType,
		"mime":          image.Mime,
		"bytes":         len(image.Bytes),
		"actualSize":    image.ActualSize,
		"actualQuality": image.ActualQuality,
		"outputFormat":  image.OutputFormat,
		"elapsedMs":     time.Since(started).Milliseconds(),
	})
	job, _, _ = m.store.Update(spaceToken, jobID, func(j *Job) {
		ApplyStage(j, StageSaving)
		if j.Progress < 92 {
			j.Progress = 92
		}
	})
	m.publish(jobID, "progress", eventPayload(job))
	saved, err := m.output.Save(spaceToken, jobID, index, image.Bytes, image.Mime)
	if err != nil {
		m.appendDebugLog(spaceToken, jobID, index, "error", "save_output", "保存图片到本机失败", map[string]any{
			"error":     err.Error(),
			"errorCode": ErrorMeta(err.Error()).Code,
			"elapsedMs": time.Since(started).Milliseconds(),
		})
		return withElapsed(NewResult(index, StatusFailed, err.Error()), started)
	}
	m.appendDebugLog(spaceToken, jobID, index, "info", "save_output", "图片已保存到本机", map[string]any{
		"url":      saved.URL,
		"fileName": saved.FileName,
		"mime":     saved.Mime,
		"bytes":    saved.Bytes,
	})
	result := withElapsed(NewResult(index, StatusSucceeded, ""), started)
	result.ImageURL = fmt.Sprintf("/api/background-tasks/%s/images/%d", jobID, index)
	result.OutputDate = saved.Date
	result.OutputFileName = saved.FileName
	result.Mime = saved.Mime
	result.Bytes = saved.Bytes
	result.RevisedPrompt = image.RevisedPrompt
	result.ActualSize = image.ActualSize
	result.ActualQuality = image.ActualQuality
	result.OutputFormat = image.OutputFormat
	if spaceCfg.AutoUploadPixhost {
		if uploaded, err := m.pixhost.UploadFile(ctx, saved.Path, saved.Mime, saved.FileName); err == nil {
			result.RemoteURL = uploaded.ShowURL
			result.RemoteThumbURL = uploaded.ThumbURL
		} else {
			result.UploadError = err.Error()
		}
	}
	return result
}

func (m *Manager) snapshotReferenceImages(spaceToken string, jobID string, uploadIDs []string) ([]ReferenceSnapshot, error) {
	snapshots := make([]ReferenceSnapshot, 0, len(uploadIDs))
	for index, id := range uploadIDs {
		item, sourcePath, err := m.uploads.GetReferenceImage(spaceToken, id)
		if err != nil {
			return nil, err
		}
		uploadDir := filepath.Dir(sourcePath)
		spaceDir := filepath.Dir(uploadDir)
		relDir := filepath.Join("job_refs", jobID)
		absDir := filepath.Join(spaceDir, relDir)
		if err := os.MkdirAll(absDir, 0o700); err != nil {
			return nil, err
		}
		ext := filepath.Ext(item.FileName)
		if ext == "" {
			ext = "." + referenceExtension(item.Mime)
		}
		fileName := fmt.Sprintf("%02d-%s%s", index+1, item.ID, ext)
		relPath := filepath.Join(relDir, fileName)
		destPath := filepath.Join(absDir, fileName)
		if err := copyFileAtomic(sourcePath, destPath, 0o600); err != nil {
			return nil, err
		}
		snapshots = append(snapshots, ReferenceSnapshot{
			UploadID:     item.ID,
			OriginalName: item.OriginalName,
			FileName:     filepath.ToSlash(relPath),
			Mime:         item.Mime,
			Size:         item.Size,
		})
	}
	return snapshots, nil
}

func (m *Manager) copyReferenceSnapshots(spaceToken string, jobID string, refs []ReferenceSnapshot) ([]ReferenceSnapshot, error) {
	spaceDir, err := m.store.spaces.SpaceDir(spaceToken)
	if err != nil {
		return nil, err
	}
	relDir := filepath.Join("job_refs", jobID)
	absDir := filepath.Join(spaceDir, relDir)
	if err := os.MkdirAll(absDir, 0o700); err != nil {
		return nil, err
	}
	copied := make([]ReferenceSnapshot, 0, len(refs))
	for index, ref := range refs {
		relSource, err := cleanReferenceSnapshotPath(ref.FileName)
		if err != nil {
			return nil, err
		}
		sourcePath := filepath.Join(spaceDir, relSource)
		ext := filepath.Ext(ref.FileName)
		if ext == "" {
			ext = "." + referenceExtension(ref.Mime)
		}
		fileName := fmt.Sprintf("%02d%s", index+1, ext)
		relPath := filepath.Join(relDir, fileName)
		destPath := filepath.Join(absDir, fileName)
		if err := copyFileAtomic(sourcePath, destPath, 0o600); err != nil {
			return nil, err
		}
		size := ref.Size
		if info, err := os.Stat(destPath); err == nil {
			size = info.Size()
		}
		copied = append(copied, ReferenceSnapshot{
			UploadID:     ref.UploadID,
			OriginalName: ref.OriginalName,
			FileName:     filepath.ToSlash(relPath),
			Mime:         ref.Mime,
			Size:         size,
		})
	}
	return copied, nil
}

func (m *Manager) inputImagesForJob(spaceToken string, job Job) ([]newapi.InputImage, error) {
	if len(job.References) > 0 {
		spaceDir, err := m.store.spaces.SpaceDir(spaceToken)
		if err != nil {
			return nil, err
		}
		inputs := make([]newapi.InputImage, 0, len(job.References))
		for _, ref := range job.References {
			rel, err := cleanJobReferencePath(job.ID, ref.FileName)
			if err != nil {
				return nil, err
			}
			inputs = append(inputs, newapi.InputImage{Name: ref.OriginalName, Path: filepath.Join(spaceDir, rel), Mime: ref.Mime})
		}
		return inputs, nil
	}

	inputs := make([]newapi.InputImage, 0, len(job.UploadIDs))
	for _, id := range job.UploadIDs {
		item, path, err := m.uploads.GetReferenceImage(spaceToken, id)
		if err != nil {
			return nil, err
		}
		inputs = append(inputs, newapi.InputImage{Name: item.OriginalName, Path: path, Mime: item.Mime})
	}
	return inputs, nil
}

func cleanRelativePath(value string) (string, error) {
	rel := filepath.Clean(filepath.FromSlash(strings.TrimSpace(value)))
	if rel == "." || rel == "" || filepath.IsAbs(rel) || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", errors.New("参考图快照路径无效")
	}
	return rel, nil
}

func cleanReferenceSnapshotPath(value string) (string, error) {
	rel, err := cleanRelativePath(value)
	if err != nil {
		return "", err
	}
	prefix := "job_refs" + string(filepath.Separator)
	if !strings.HasPrefix(rel, prefix) {
		return "", errors.New("参考图快照路径无效")
	}
	return rel, nil
}

func cleanJobReferencePath(jobID string, value string) (string, error) {
	rel, err := cleanRelativePath(value)
	if err != nil {
		return "", err
	}
	prefix := filepath.Join("job_refs", jobID)
	if rel != prefix && !strings.HasPrefix(rel, prefix+string(filepath.Separator)) {
		return "", errors.New("参考图快照路径无效")
	}
	return rel, nil
}

func copyFileAtomic(sourcePath string, destPath string, perm os.FileMode) error {
	source, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer source.Close()
	tmp := fmt.Sprintf("%s.%d.tmp", destPath, time.Now().UnixNano())
	dest, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(dest, source)
	closeErr := dest.Close()
	if copyErr != nil {
		_ = os.Remove(tmp)
		return copyErr
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return closeErr
	}
	return os.Rename(tmp, destPath)
}

func referenceExtension(mime string) string {
	switch strings.ToLower(strings.TrimSpace(mime)) {
	case "image/jpeg":
		return "jpg"
	case "image/webp":
		return "webp"
	default:
		return "png"
	}
}

func (m *Manager) deleteReferenceSnapshots(spaceToken string, job Job) {
	if len(job.References) == 0 {
		return
	}
	spaceDir, err := m.store.spaces.SpaceDir(spaceToken)
	if err != nil {
		return
	}
	dirs := make(map[string]struct{})
	for _, ref := range job.References {
		rel, err := cleanJobReferencePath(job.ID, ref.FileName)
		if err != nil {
			continue
		}
		dir := filepath.Dir(rel)
		if dir == "." || dir == "" {
			continue
		}
		dirs[dir] = struct{}{}
	}
	for dir := range dirs {
		_ = os.RemoveAll(filepath.Join(spaceDir, dir))
	}
}

func (m *Manager) resolveResultOutput(job Job, result Result) (string, string, error) {
	if result.OutputDate != "" && result.OutputFileName != "" {
		return m.output.Resolve(job.SpaceToken, result.OutputDate, result.OutputFileName)
	}
	return m.output.ResolveURL(result.ImageURL)
}

func upsertResult(job *Job, result Result) {
	for i := range job.Results {
		if job.Results[i].Index == result.Index {
			job.Results[i] = result
			return
		}
	}
	job.Results = append(job.Results, result)
}

func (m *Manager) fakeProgress(spaceToken string, jobID string, done <-chan struct{}) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			job, ok, _ := m.store.Update(spaceToken, jobID, func(j *Job) {
				if j.Final() || j.Progress >= 90 {
					return
				}
				if j.Stage == StageWaitingUpstream || j.Stage == StageSubmitting || j.Stage == StagePreparing {
					j.Progress += 3
					if j.Progress > 90 {
						j.Progress = 90
					}
				}
			})
			if ok && !job.Final() {
				m.publish(jobID, "progress", eventPayload(job))
			}
		}
	}
}

func (m *Manager) publish(jobID string, name string, data any) {
	meta := EventMeta(name)
	m.hub.Publish(jobID, events.Event{Event: name, Code: meta.Code, English: meta.English, Chinese: meta.Chinese, Data: data})
}

func (m *Manager) appendDebugLog(spaceToken string, jobID string, imageIndex int, level string, stage string, message string, fields map[string]any) {
	var logEntry DebugLog
	job, ok, _ := m.store.Update(spaceToken, jobID, func(j *Job) {
		if !j.DebugEnabled {
			return
		}
		logEntry = newDebugLog(imageIndex, level, stage, message, fields)
		j.DebugLogs = append(j.DebugLogs, logEntry)
		const maxDebugLogs = 200
		if len(j.DebugLogs) > maxDebugLogs {
			j.DebugLogs = append([]DebugLog{}, j.DebugLogs[len(j.DebugLogs)-maxDebugLogs:]...)
		}
	})
	if ok && logEntry.Time != "" {
		m.publish(jobID, "debug", map[string]any{"job": job, "log": logEntry})
	}
}

func newDebugLog(imageIndex int, level string, stage string, message string, fields map[string]any) DebugLog {
	if level == "" {
		level = "info"
	}
	return DebugLog{
		Time:       time.Now().Format(time.RFC3339Nano),
		Level:      level,
		Stage:      stage,
		Message:    message,
		ImageIndex: imageIndex,
		Fields:     sanitizeDebugFields(fields),
	}
}

func sanitizeDebugFields(fields map[string]any) map[string]any {
	if len(fields) == 0 {
		return nil
	}
	out := make(map[string]any, len(fields))
	for key, value := range fields {
		lower := strings.ToLower(key)
		if strings.Contains(lower, "apikey") || strings.Contains(lower, "api_key") || strings.Contains(lower, "token") || strings.Contains(lower, "secret") {
			out[key] = "***"
			continue
		}
		out[key] = value
	}
	return out
}

func upstreamEndpoint(baseURL string, mode Mode) string {
	path := "images/generations"
	if mode == ModeImageToImage {
		path = "images/edits"
	}
	return strings.TrimRight(baseURL, "/") + "/" + path
}

func requestContentType(mode Mode) string {
	if mode == ModeImageToImage {
		return "multipart/form-data"
	}
	return "application/json"
}

func promptForImage(job Job, index int) string {
	if index >= 0 && index < len(job.FramePrompts) {
		if prompt := strings.TrimSpace(job.FramePrompts[index]); prompt != "" {
			return prompt
		}
	}
	return job.Prompt
}

func normalizeFramePrompts(values []string, count int) []string {
	if len(values) == 0 || count <= 0 {
		return nil
	}
	out := make([]string, 0, count)
	for i := 0; i < count && i < len(values); i++ {
		out = append(out, strings.TrimSpace(values[i]))
	}
	for len(out) > 0 && out[len(out)-1] == "" {
		out = out[:len(out)-1]
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func debugPayload(job Job, model string, skipImageParams bool) map[string]any {
	payload := map[string]any{
		"model":           model,
		"prompt":          fmt.Sprintf("<已脱敏，长度 %d 字符>", len([]rune(job.Prompt))),
		"n":               1,
		"response_format": "b64_json",
	}
	if !skipImageParams {
		payload["output_format"] = debugActualOutputFormat(job.OutputFormat)
		if job.Size != "" && job.Size != "自动" && job.Size != "auto" {
			payload["size"] = job.Size
		}
		if job.Quality != "" {
			payload["quality"] = debugActualQuality(job.Quality)
		}
	}
	return payload
}

func debugActualOutputFormat(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "jpg", "jpeg":
		return "jpeg"
	case "webp":
		return "webp"
	case "png":
		return "png"
	default:
		return "png"
	}
}

func debugActualQuality(value string) string {
	switch strings.TrimSpace(value) {
	case "low", "medium", "high":
		return strings.TrimSpace(value)
	default:
		return "auto"
	}
}

func debugInputImages(inputs []newapi.InputImage) []map[string]any {
	out := make([]map[string]any, 0, len(inputs))
	for index, input := range inputs {
		out = append(out, map[string]any{
			"index": index,
			"name":  input.Name,
			"mime":  input.Mime,
		})
	}
	return out
}

func maskSecret(value string) string {
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if len(runes) <= 8 {
		return "***"
	}
	return string(runes[:4]) + "..." + string(runes[len(runes)-4:])
}

func runtimeAPIKey(secrets RuntimeSecrets, provider string) string {
	if provider == config.BananaProvider {
		return strings.TrimSpace(secrets.BananaAPIKey)
	}
	return strings.TrimSpace(secrets.APIKey)
}

func cloudAPIKey(cfg spaceconfig.Config, provider string) string {
	if provider == config.BananaProvider {
		return strings.TrimSpace(cfg.BananaAPIKey)
	}
	return strings.TrimSpace(cfg.APIKey)
}

func compactDebugText(value string, limit int) string {
	text := strings.TrimSpace(strings.Join(strings.Fields(value), " "))
	runes := []rune(text)
	if limit <= 0 || len(runes) <= limit {
		return text
	}
	return string(runes[:limit]) + "..."
}

type bananaSpec struct {
	Ratio      string
	Resolution string
	Size       string
}

func resolveProvider(value string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", config.DefaultProvider, "image2", "gpt-image-2":
		return config.DefaultProvider, nil
	case config.BananaProvider, "banana-nano", "nano-banana":
		return config.BananaProvider, nil
	default:
		return "", errors.New("模型分组无效")
	}
}

func resolveModel(provider string, value string) (string, error) {
	value = strings.TrimSpace(value)
	if provider == config.BananaProvider {
		if value == "" {
			value = config.DefaultBananaModel
		}
		if !config.IsBananaModel(value) {
			return "", errors.New("Banana 模型 ID 无效")
		}
		return value, nil
	}
	return config.DefaultModel, nil
}

func bananaModelSpec(model string) bananaSpec {
	specs := map[string]bananaSpec{
		"gemini-3.1-flash-image-preview":         {Ratio: "auto", Resolution: "auto", Size: "自动"},
		"gemini-3.1-flash-image-preview-2k":      {Ratio: "auto", Resolution: "2k", Size: "自动"},
		"gemini-3.1-flash-image-preview-4k":      {Ratio: "auto", Resolution: "4k", Size: "自动"},
		"gemini-3.1-flash-image-preview-16x9-2k": {Ratio: "16:9", Resolution: "2k", Size: imageSize("16:9", "2k")},
		"gemini-3.1-flash-image-preview-16x9-4k": {Ratio: "16:9", Resolution: "4k", Size: imageSize("16:9", "4k")},
		"gemini-3.1-flash-image-preview-9x16-2k": {Ratio: "9:16", Resolution: "2k", Size: imageSize("9:16", "2k")},
		"gemini-3.1-flash-image-preview-9x16-4k": {Ratio: "9:16", Resolution: "4k", Size: imageSize("9:16", "4k")},
		"gemini-3.1-flash-image-preview-4x3-2k":  {Ratio: "4:3", Resolution: "2k", Size: imageSize("4:3", "2k")},
		"gemini-3.1-flash-image-preview-4x3-4k":  {Ratio: "4:3", Resolution: "4k", Size: imageSize("4:3", "4k")},
		"gemini-3.1-flash-image-preview-3x4-2k":  {Ratio: "3:4", Resolution: "2k", Size: imageSize("3:4", "2k")},
		"gemini-3.1-flash-image-preview-3x4-4k":  {Ratio: "3:4", Resolution: "4k", Size: imageSize("3:4", "4k")},
		"gemini-3.1-flash-image-preview-3x2-2k":  {Ratio: "3:2", Resolution: "2k", Size: imageSize("3:2", "2k")},
		"gemini-3.1-flash-image-preview-3x2-4k":  {Ratio: "3:2", Resolution: "4k", Size: imageSize("3:2", "4k")},
		"gemini-3.1-flash-image-preview-2x3-2k":  {Ratio: "2:3", Resolution: "2k", Size: imageSize("2:3", "2k")},
		"gemini-3.1-flash-image-preview-2x3-4k":  {Ratio: "2:3", Resolution: "4k", Size: imageSize("2:3", "4k")},
		"gemini-3.1-flash-image-preview-1x1-2k":  {Ratio: "1:1", Resolution: "2k", Size: imageSize("1:1", "2k")},
		"gemini-3.1-flash-image-preview-1x1-4k":  {Ratio: "1:1", Resolution: "4k", Size: imageSize("1:1", "4k")},
	}
	if spec, ok := specs[model]; ok {
		return spec
	}
	return specs[config.DefaultBananaModel]
}

func validateCreate(req CreateRequest) error {
	if strings.TrimSpace(req.Prompt) == "" {
		return errors.New("提示词不能为空")
	}
	provider, err := resolveProvider(req.Provider)
	if err != nil {
		return err
	}
	if _, err := resolveModel(provider, req.Model); err != nil {
		return err
	}
	if req.Mode != ModeTextToImage && req.Mode != ModeImageToImage {
		return errors.New("任务模式无效")
	}
	if req.Mode == ModeImageToImage && len(req.UploadIDs) > uploads.MaxReferenceImages {
		return fmt.Errorf("图生图参考图最多 %d 张", uploads.MaxReferenceImages)
	}
	if req.Mode == ModeImageToImage && len(req.References) > uploads.MaxReferenceImages {
		return fmt.Errorf("图生图参考图最多 %d 张", uploads.MaxReferenceImages)
	}
	return nil
}

func withElapsed(result Result, started time.Time) Result {
	result.ElapsedMs = time.Since(started).Milliseconds()
	return result
}

func newJobID() (string, error) {
	var bytes [8]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", err
	}
	return "img_" + time.Now().Format("20060102150405") + "_" + hex.EncodeToString(bytes[:]), nil
}

func normalizeRatio(value string) string {
	switch value {
	case "auto", "1:1", "2:3", "3:2", "3:4", "4:3", "9:16", "16:9":
		return value
	default:
		return "1:1"
	}
}

func normalizeResolution(value string) string {
	switch value {
	case "auto", "standard", "2k", "4k":
		return value
	default:
		return "standard"
	}
}

func normalizeQuality(value string) string {
	switch value {
	case "auto", "low", "medium", "high":
		return value
	default:
		return "auto"
	}
}

func normalizeOutputFormat(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "auto":
		return "auto"
	case "jpg", "jpeg":
		return "jpeg"
	case "png", "webp":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "png"
	}
}

func imageSize(ratio string, resolution string) string {
	if ratio == "auto" {
		return "自动"
	}
	if resolution == "auto" {
		resolution = "standard"
	}
	sizes := map[string]map[string]string{
		"standard": {"1:1": "1024x1024", "2:3": "1024x1536", "3:2": "1536x1024", "3:4": "768x1024", "4:3": "1024x768", "9:16": "1008x1792", "16:9": "1792x1008"},
		"2k":       {"1:1": "2048x2048", "2:3": "1344x2016", "3:2": "2016x1344", "3:4": "1536x2048", "4:3": "2048x1536", "9:16": "1152x2048", "16:9": "2048x1152"},
		"4k":       {"1:1": "2880x2880", "2:3": "2336x3504", "3:2": "3504x2336", "3:4": "2448x3264", "4:3": "3264x2448", "9:16": "2160x3840", "16:9": "3840x2160"},
	}
	return sizes[resolution][ratio]
}

func clamp(value int, min int, max int, fallback int) int {
	if value == 0 {
		value = fallback
	}
	if value < min {
		return min
	}
	if max > 0 && value > max {
		return max
	}
	return value
}

func ErrorResponse(err error) map[string]any {
	return map[string]any{"ok": false, "message": err.Error(), "status": 400}
}

func Label(meta Meta) string {
	return fmt.Sprintf("%s / %s / %s", meta.Chinese, meta.English, meta.Code)
}
