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

	"github.com/y08lin4/lyra-image-workbench/internal/activitylog"
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
	store         *Store
	hub           *events.Hub
	settings      *settings.FileStore
	spaceConfig   *spaceconfig.Store
	uploads       *uploads.Store
	output        *output.Store
	newapi        *newapi.Client
	pixhost       *pixhost.Client
	activity      activitylog.Recorder
	ownerForSpace func(string) string
	queue         chan jobRef
	workerCount   int
	mu            sync.Mutex
	cancels       map[string]*jobRun
	secrets       map[string]jobSecret
}

const (
	defaultWorkerCount = 2
	maxWorkerCount     = 16
	jobWorkerCountEnv  = "LYRA_JOB_WORKERS"
)

type jobRef struct {
	SpaceToken string
	JobID      string
}

type jobRun struct {
	cancel context.CancelFunc
}

type jobSecret struct {
	APIKey string
}

type RecoverOptions struct {
	RefundQueued func(Job) error
}

func NewManager(store *Store, hub *events.Hub, settingsStore *settings.FileStore, spaceConfig *spaceconfig.Store, uploadStore *uploads.Store, outputStore *output.Store, newapiClient *newapi.Client) *Manager {
	workerCount := configuredWorkerCount()
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
		workerCount: workerCount,
		cancels:     make(map[string]*jobRun),
		secrets:     make(map[string]jobSecret),
	}
	for i := 0; i < workerCount; i++ {
		go m.worker()
	}
	return m
}

func configuredWorkerCount() int {
	value := strings.TrimSpace(os.Getenv(jobWorkerCountEnv))
	if value == "" {
		return defaultWorkerCount
	}
	count, err := strconv.Atoi(value)
	if err != nil || count < 1 {
		return defaultWorkerCount
	}
	if count > maxWorkerCount {
		return maxWorkerCount
	}
	return count
}

func (m *Manager) SetActivityRecorder(recorder activitylog.Recorder, ownerForSpace func(string) string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.activity = recorder
	m.ownerForSpace = ownerForSpace
}

func (m *Manager) Recover(options ...RecoverOptions) error {
	var opts RecoverOptions
	if len(options) > 0 {
		opts = options[0]
	}
	bySpace, err := m.store.AllSpacesJobs()
	if err != nil {
		return err
	}
	for token, list := range bySpace {
		for _, job := range list {
			switch job.Status {
			case StatusQueued:
				if opts.RefundQueued != nil && job.ConsumedCredits > 0 {
					if err := opts.RefundQueued(job); err != nil {
						return fmt.Errorf("recover queued job %s refund failed: %w", job.ID, err)
					}
				}
				_, _, _ = m.store.Update(token, job.ID, func(j *Job) {
					ApplyStatus(j, StatusInterrupted)
					ApplyStage(j, StageInterrupted)
					j.Progress = 100
					j.Error = "Task was queued before the server restarted and has not been submitted upstream. Please retry. Charged credits are refunded automatically when account billing is enabled."
					j.FinishedAt = time.Now()
				})
			case StatusRunning:
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
	modelInput := modelInputForProvider(req.Provider, req.Model)
	model, err := resolveModel(provider, modelInput)
	if err != nil {
		return Job{}, err
	}
	spaceCfg, err := m.spaceConfig.Get(spaceToken)
	if err != nil {
		return Job{}, err
	}
	if modeRequiresReference(req.Mode) {
		if effectiveReferenceCount(req) == 0 {
			if req.Mode == ModeGIF {
				return Job{}, errors.New("GIF 动图需要先选择一张参考图")
			}
			return Job{}, errors.New("图生图需要先上传参考图")
		}
		if req.Mode == ModeGIF && effectiveReferenceCount(req) != 1 {
			return Job{}, errors.New("GIF 动图只能选择一张参考图")
		}
		if len(req.References) == 0 {
			for _, id := range req.UploadIDs {
				if _, _, err := m.uploads.GetReferenceImage(spaceToken, id); err != nil {
					return Job{}, err
				}
			}
		}
	}
	apiKey := ""
	if req.Mode != ModeGIF {
		channelProvider := channelProviderForModel(provider, model)
		apiKey = runtimeAPIKey(req.RuntimeSecrets)
		if apiKey == "" {
			apiKey = cloudAPIKey(spaceCfg)
		}
		if apiKey == "" {
			apiKey = settings.SystemAPIKeyForProvider(m.settings.Get(), channelProvider)
		}
		if apiKey == "" {
			return Job{}, errors.New("codex-key is not configured; save it locally or upload it to cloud after enabling account protection")
		}
	}
	now := time.Now()
	id, err := newJobID()
	if err != nil {
		return Job{}, err
	}
	var references []ReferenceSnapshot
	if modeRequiresReference(req.Mode) {
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
	resolution := normalizeResolution(req.Resolution)
	ratio := normalizeRatio(req.Ratio)
	size, err := imageSizeForRequest(req.Size, ratio, resolution)
	if err != nil {
		return Job{}, err
	}
	if requestedImageSizeIsExplicit(req.Size) {
		if mappedRatio, mappedResolution, ok := imageSpecFromSize(size); ok {
			ratio = mappedRatio
			resolution = mappedResolution
		} else {
			ratio = "auto"
			resolution = "auto"
		}
	}
	quality := normalizeQuality(req.Quality)
	outputFormat := normalizeOutputFormat(req.OutputFormat)
	if req.Mode == ModeGIF {
		ratio = "auto"
		resolution = "auto"
		quality = "auto"
		outputFormat = "gif"
		size = "自动"
	}
	count := clamp(req.Count, 1, 24, 1)
	if req.Mode == ModeGIF {
		count = 1
	}
	consumedCredits := CreditCostForCount(count)
	if req.Mode == ModeGIF || req.WaiveCredits {
		consumedCredits = 0
	}
	extraParams := filterExtraParams(req.ExtraParams)
	job := Job{
		ID:              id,
		SpaceToken:      spaceToken,
		Provider:        provider,
		Model:           model,
		Mode:            req.Mode,
		Source:          normalizeSource(req.Source),
		Prompt:          strings.TrimSpace(req.Prompt),
		FramePrompts:    normalizeFramePrompts(req.FramePrompts, count),
		Ratio:           ratio,
		Resolution:      resolution,
		Quality:         quality,
		OutputFormat:    outputFormat,
		Size:            size,
		ExtraParams:     extraParams,
		Count:           count,
		ConsumedCredits: consumedCredits,
		Concurrency:     clamp(req.Concurrency, 1, 0, 1),
		UploadIDs:       append([]string{}, req.UploadIDs...),
		References:      references,
		Progress:        0,
		Results:         []Result{},
		DebugEnabled:    adminCfg.DebugEnabled,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	ApplyStatus(&job, StatusQueued)
	ApplyStage(&job, StageQueued)
	if job.DebugEnabled {
		job.DebugLogs = append(job.DebugLogs, newDebugLog(-1, "info", "create", "任务已创建，诊断日志已开启", map[string]any{
			"provider":        job.Provider,
			"model":           job.Model,
			"mode":            job.Mode,
			"ratio":           job.Ratio,
			"resolution":      job.Resolution,
			"quality":         job.Quality,
			"format":          job.OutputFormat,
			"size":            job.Size,
			"extraParams":     job.ExtraParams,
			"count":           job.Count,
			"consumedCredits": job.ConsumedCredits,
			"concurrency":     job.Concurrency,
		}))
	}
	if err := m.store.Save(job); err != nil {
		return Job{}, err
	}
	if req.BeforeEnqueue != nil {
		if err := req.BeforeEnqueue(job); err != nil {
			if deleted, ok, deleteErr := m.store.Delete(spaceToken, job.ID); deleteErr == nil && ok {
				m.deleteReferenceSnapshots(spaceToken, deleted)
			}
			return Job{}, err
		}
	}
	if apiKey != "" {
		m.setJobSecret(job.ID, apiKey)
	}
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

func (m *Manager) Retry(spaceToken string, id string, secrets RuntimeSecrets, waiveCredits bool, beforeEnqueue ...func(Job) error) (Job, error) {
	old, ok, err := m.store.Get(spaceToken, id)
	if err != nil {
		return Job{}, err
	}
	if !ok {
		return Job{}, errors.New("任务不存在")
	}
	var callback func(Job) error
	if len(beforeEnqueue) > 0 {
		callback = beforeEnqueue[0]
	}
	return m.Create(spaceToken, CreateRequest{
		RuntimeSecrets: secrets,
		Provider:       old.Provider,
		Model:          old.Model,
		Mode:           old.Mode,
		Source:         sourceOrWeb(old.Source),
		Prompt:         old.Prompt,
		FramePrompts:   old.FramePrompts,
		Ratio:          old.Ratio,
		Resolution:     old.Resolution,
		Size:           old.Size,
		Quality:        old.Quality,
		OutputFormat:   old.OutputFormat,
		ExtraParams:    old.ExtraParams,
		Count:          old.Count,
		Concurrency:    old.Concurrency,
		UploadIDs:      old.UploadIDs,
		References:     old.References,
		WaiveCredits:   waiveCredits,
		BeforeEnqueue:  callback,
	})
}

func (m *Manager) Cancel(spaceToken string, id string) (Job, error) {
	running := false
	m.mu.Lock()
	if run := m.cancels[id]; run != nil {
		running = true
		run.cancel()
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
	if run := m.cancels[id]; run != nil {
		run.cancel()
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
func (m *Manager) finishRun(jobID string, run *jobRun) {
	run.cancel()
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cancels[jobID] != run {
		return
	}
	delete(m.cancels, jobID)
	delete(m.secrets, jobID)
}

func (m *Manager) run(spaceToken string, jobID string) {
	ctx, cancel := context.WithCancel(context.Background())
	run := &jobRun{cancel: cancel}
	m.mu.Lock()
	if m.cancels[jobID] != nil {
		m.mu.Unlock()
		cancel()
		return
	}
	m.cancels[jobID] = run
	m.mu.Unlock()

	claimed := false
	job, ok, err := m.store.Update(spaceToken, jobID, func(j *Job) {
		if j.Status != StatusQueued || j.Final() {
			return
		}
		ApplyStatus(j, StatusRunning)
		ApplyStage(j, StagePreparing)
		j.Progress = 5
		j.StartedAt = time.Now()
		claimed = true
	})
	if err != nil || !ok || !claimed {
		m.finishRun(jobID, run)
		return
	}
	defer m.finishRun(jobID, run)
	m.publish(jobID, "progress", eventPayload(job))
	progressDone := make(chan struct{})
	go m.fakeProgress(spaceToken, jobID, progressDone)
	defer close(progressDone)

	results := m.runImages(ctx, spaceToken, jobID)
	select {
	case <-ctx.Done():
		job, exists, _ := m.store.Get(spaceToken, jobID)
		if !exists {
			return
		}
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
	activityJob := job
	activityJob.Results = results
	activityJob.Progress = 100
	activityJob.FinishedAt = time.Now()
	if okCount == activityJob.Count {
		ApplyStatus(&activityJob, StatusSucceeded)
		ApplyStage(&activityJob, StageSucceeded)
	} else if okCount > 0 {
		ApplyStatus(&activityJob, StatusPartialFailed)
		ApplyStage(&activityJob, StagePartialFailed)
	} else if activityJob.Status != StatusCancelled {
		ApplyStatus(&activityJob, StatusFailed)
		ApplyStage(&activityJob, StageFailed)
		activityJob.Error = "没有图片生成成功"
	}
	if okCount < activityJob.Count && activityJob.Status != StatusCancelled {
		m.recordTaskFailure(activityJob, okCount)
	}
	job, _, _ = m.store.Update(spaceToken, jobID, func(j *Job) {
		j.Results = results
		j.Progress = 100
		j.FinishedAt = activityJob.FinishedAt
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
	concurrency := job.Concurrency
	if concurrency < 1 {
		concurrency = 1
	}
	if job.Count > 0 && concurrency > job.Count {
		concurrency = job.Count
	}
	sem := make(chan struct{}, concurrency)
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
				if !result.OK {
					m.recordResultFailure(job, result)
				}
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
	if job.Mode == ModeGIF {
		return m.generateGIF(ctx, spaceToken, job, index, prompt, started)
	}
	spaceCfg, err := m.spaceConfig.Get(spaceToken)
	if err != nil {
		return NewResult(index, StatusFailed, err.Error())
	}
	provider, err := resolveProvider(job.Provider)
	if err != nil {
		return NewResult(index, StatusFailed, err.Error())
	}
	modelInput := modelInputForProvider(job.Provider, job.Model)
	model, err := resolveModel(provider, modelInput)
	if err != nil {
		return NewResult(index, StatusFailed, err.Error())
	}
	channelProvider := channelProviderForModel(provider, model)
	baseURL := settings.SystemBaseURLForProvider(admin, channelProvider)
	if baseURL == "" {
		return NewResult(index, StatusFailed, "image channel is disabled or missing base URL")
	}
	apiKey := m.jobAPIKey(jobID)
	skipImageParams := shouldSkipImageParams(model)
	if apiKey == "" {
		apiKey = cloudAPIKey(spaceCfg)
		if apiKey == "" {
			apiKey = settings.SystemAPIKeyForProvider(admin, channelProvider)
		}
		if apiKey != "" {
			m.setJobSecret(jobID, apiKey)
		}
	}
	if apiKey == "" {
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
		"url":             upstreamEndpoint(baseURL, job.Mode),
		"baseUrl":         baseURL,
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
	image, err := m.newapi.Generate(ctx, newapi.Request{Mode: string(job.Mode), BaseURL: baseURL, APIKey: apiKey, Model: model, Prompt: prompt, Size: job.Size, Quality: job.Quality, OutputFormat: job.OutputFormat, ExtraParams: job.ExtraParams, SkipImageParams: skipImageParams, TimeoutSec: admin.TimeoutSec, InputImages: inputs})
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
			current, exists, _ := m.store.Get(spaceToken, jobID)
			if !exists || current.Final() {
				return
			}
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

func (m *Manager) activityContext(spaceToken string) (activitylog.Recorder, string) {
	m.mu.Lock()
	recorder := m.activity
	ownerForSpace := m.ownerForSpace
	m.mu.Unlock()
	username := ""
	if ownerForSpace != nil {
		username = strings.TrimSpace(ownerForSpace(spaceToken))
	}
	return recorder, username
}

func (m *Manager) recordResultFailure(job Job, result Result) {
	if result.OK {
		return
	}
	recorder, username := m.activityContext(job.SpaceToken)
	if recorder == nil {
		return
	}
	meta := ErrorMeta(result.Error)
	_, _ = recorder.Append(activitylog.EntryInput{
		Type:         activitylog.TypeResultFailed,
		Level:        activitylog.LevelError,
		Username:     username,
		ResourceType: "task_result",
		ResourceID:   fmt.Sprintf("%s:%d", job.ID, result.Index),
		Message:      "生成结果失败",
		Fields: map[string]any{
			"taskId":       job.ID,
			"imageIndex":   result.Index,
			"mode":         job.Mode,
			"provider":     job.Provider,
			"model":        job.Model,
			"outputFormat": job.OutputFormat,
			"status":       result.Status,
			"errorCode":    meta.Code,
			"errorText":    meta.Chinese,
			"elapsedMs":    result.ElapsedMs,
		},
	})
}

func (m *Manager) recordTaskFailure(job Job, okCount int) {
	recorder, username := m.activityContext(job.SpaceToken)
	if recorder == nil {
		return
	}
	failedCount := job.Count - okCount
	level := activitylog.LevelError
	message := "生成任务失败"
	if okCount > 0 {
		level = activitylog.LevelWarning
		message = "生成任务部分失败"
	}
	_, _ = recorder.Append(activitylog.EntryInput{
		Type:         activitylog.TypeTaskFailed,
		Level:        level,
		Username:     username,
		ResourceType: "task",
		ResourceID:   job.ID,
		Message:      message,
		Fields: map[string]any{
			"taskId":         job.ID,
			"mode":           job.Mode,
			"provider":       job.Provider,
			"model":          job.Model,
			"status":         job.Status,
			"statusCode":     job.StatusCode,
			"count":          job.Count,
			"succeededCount": okCount,
			"failedCount":    failedCount,
		},
	})
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
		if isSensitiveDebugKey(key) {
			out[key] = "***"
			continue
		}
		out[key] = sanitizeDebugValue(value)
	}
	return out
}

func sanitizeDebugValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, item := range typed {
			if isSensitiveDebugKey(key) {
				out[key] = "***"
				continue
			}
			out[key] = sanitizeDebugValue(item)
		}
		return out
	case []any:
		out := make([]any, len(typed))
		for i, item := range typed {
			out[i] = sanitizeDebugValue(item)
		}
		return out
	default:
		return value
	}
}

func isSensitiveDebugKey(key string) bool {
	lower := strings.ToLower(key)
	return strings.Contains(lower, "apikey") || strings.Contains(lower, "api_key") || strings.Contains(lower, "authorization") || strings.Contains(lower, "token") || strings.Contains(lower, "secret")
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
		"output_format":   debugActualOutputFormat(job.OutputFormat),
	}
	if !skipImageParams && job.Size != "" && job.Size != "自动" && job.Size != "auto" {
		payload["size"] = job.Size
	}
	if job.Quality != "" {
		payload["quality"] = debugActualQuality(job.Quality)
	}
	for key, value := range filterExtraParams(job.ExtraParams) {
		payload[key] = value
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

func runtimeAPIKey(secrets RuntimeSecrets) string {
	return strings.TrimSpace(secrets.APIKey)
}

func cloudAPIKey(cfg spaceconfig.Config) string {
	if !cfg.CloudAPIKeyEnabled {
		return ""
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

func resolveProvider(value string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", config.DefaultProvider, "image2", "gpt-image-2", "image-2-4k":
		return config.DefaultProvider, nil
	default:
		return "", errors.New("模型分组无效")
	}
}

func resolveModel(_ string, value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	switch strings.ToLower(trimmed) {
	case "image-2":
		return "image-2", nil
	case "image-2-4k":
		return "image-2-4k", nil
	case "", config.DefaultModel:
		return config.DefaultModel, nil
	default:
		return trimmed, nil
	}
}
func modelInputForProvider(provider string, model string) string {
	if strings.EqualFold(strings.TrimSpace(provider), "image-2-4k") && isDefaultModelInput(model) {
		return "image-2-4k"
	}
	return model
}

func channelProviderForModel(provider string, model string) string {
	if modelUsesFullImageChannel(model) {
		return "image-2-4k"
	}
	if strings.TrimSpace(provider) == "" {
		return config.DefaultProvider
	}
	return provider
}

func isDefaultModelInput(model string) bool {
	switch strings.ToLower(strings.TrimSpace(model)) {
	case "", "image-2", config.DefaultModel:
		return true
	default:
		return false
	}
}

func shouldSkipImageParams(model string) bool {
	return strings.EqualFold(strings.TrimSpace(model), "image-2")
}

func modelUsesFullImageChannel(model string) bool {
	return strings.EqualFold(strings.TrimSpace(model), "image-2-4k")
}

func filterExtraParams(values map[string]any) map[string]any {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]any, len(values))
	for key, value := range values {
		trimmed := strings.TrimSpace(key)
		if trimmed == "" || isCoreExtraParamKey(trimmed) {
			continue
		}
		out[trimmed] = value
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func isCoreExtraParamKey(key string) bool {
	switch canonicalExtraParamKey(key) {
	case "model", "prompt", "n", "responseformat", "size", "quality", "outputformat", "provider", "mode", "source", "ratio", "resolution", "count", "concurrency", "uploadids", "references", "runtimesecrets", "apikey", "authorization", "extraparams", "extrabody":
		return true
	default:
		return false
	}
}

func canonicalExtraParamKey(key string) string {
	key = strings.ToLower(strings.TrimSpace(key))
	key = strings.ReplaceAll(key, "_", "")
	key = strings.ReplaceAll(key, "-", "")
	key = strings.ReplaceAll(key, ".", "")
	return key
}

func validateCreate(req CreateRequest) error {
	if strings.TrimSpace(req.Prompt) == "" {
		return errors.New("提示词不能为空")
	}
	provider, err := resolveProvider(req.Provider)
	if err != nil {
		return err
	}
	if _, err := resolveModel(provider, modelInputForProvider(req.Provider, req.Model)); err != nil {
		return err
	}
	if req.Mode != ModeTextToImage && req.Mode != ModeImageToImage && req.Mode != ModeGIF {
		return errors.New("任务模式无效")
	}
	if modeRequiresReference(req.Mode) && len(req.UploadIDs) > uploads.MaxReferenceImages {
		return fmt.Errorf("参考图最多 %d 张", uploads.MaxReferenceImages)
	}
	if modeRequiresReference(req.Mode) && len(req.References) > uploads.MaxReferenceImages {
		return fmt.Errorf("参考图最多 %d 张", uploads.MaxReferenceImages)
	}
	return nil
}

func modeRequiresReference(mode Mode) bool {
	return mode == ModeImageToImage || mode == ModeGIF
}

func effectiveReferenceCount(req CreateRequest) int {
	if len(req.References) > 0 {
		return len(req.References)
	}
	return len(req.UploadIDs)
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

func imageSizeForRequest(requestedSize string, ratio string, resolution string) (string, error) {
	if strings.TrimSpace(requestedSize) == "" {
		return imageSize(ratio, resolution), nil
	}
	size, err := normalizeExplicitImageSize(requestedSize)
	if err != nil {
		return "", err
	}
	return size, nil
}

func requestedImageSizeIsExplicit(value string) bool {
	size, err := normalizeExplicitImageSize(value)
	return err == nil && size != "" && size != "自动"
}

func normalizeExplicitImageSize(value string) (string, error) {
	clean := strings.ToLower(strings.TrimSpace(value))
	clean = strings.ReplaceAll(clean, "×", "x")
	clean = strings.Join(strings.Fields(clean), "")
	if clean == "" || clean == "auto" || clean == "自动" {
		return "自动", nil
	}
	parts := strings.Split(clean, "x")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", errors.New("自定义尺寸必须使用 WIDTHxHEIGHT，例如 1536x864")
	}
	width, err := strconv.Atoi(parts[0])
	if err != nil {
		return "", errors.New("自定义尺寸宽度必须是整数")
	}
	height, err := strconv.Atoi(parts[1])
	if err != nil {
		return "", errors.New("自定义尺寸高度必须是整数")
	}
	if width <= 0 || height <= 0 {
		return "", errors.New("自定义尺寸宽高必须大于 0")
	}
	if width%16 != 0 || height%16 != 0 {
		return "", errors.New("自定义尺寸宽高必须都能被 16 整除")
	}
	ratio := float64(width) / float64(height)
	if ratio < 1.0/3.0 || ratio > 3.0 {
		return "", errors.New("自定义尺寸比例必须在 1:3 到 3:1 之间")
	}
	const maxEdge = 3840
	const maxPixels = 3840 * 2160
	if width > maxEdge || height > maxEdge || width*height > maxPixels {
		return "", errors.New("自定义尺寸最大边不能超过 3840，像素总量不能超过 3840x2160")
	}
	return fmt.Sprintf("%dx%d", width, height), nil
}

func imageSpecFromSize(size string) (string, string, bool) {
	for resolution, byRatio := range imageSizeTable() {
		for ratio, value := range byRatio {
			if value == size {
				return ratio, resolution, true
			}
		}
	}
	return "", "", false
}

func imageSize(ratio string, resolution string) string {
	if ratio == "auto" {
		return "自动"
	}
	if resolution == "auto" {
		resolution = "standard"
	}
	return imageSizeTable()[resolution][ratio]
}

func imageSizeTable() map[string]map[string]string {
	return map[string]map[string]string{
		"standard": {"1:1": "1024x1024", "2:3": "1024x1536", "3:2": "1536x1024", "3:4": "768x1024", "4:3": "1024x768", "9:16": "1008x1792", "16:9": "1792x1008"},
		"2k":       {"1:1": "2048x2048", "2:3": "1344x2016", "3:2": "2016x1344", "3:4": "1536x2048", "4:3": "2048x1536", "9:16": "1152x2048", "16:9": "2048x1152"},
		"4k":       {"1:1": "2880x2880", "2:3": "2336x3504", "3:2": "3504x2336", "3:4": "2448x3264", "4:3": "3264x2448", "9:16": "2160x3840", "16:9": "3840x2160"},
	}
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
