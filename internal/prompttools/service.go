package prompttools

import (
	"bytes"
	"compress/zlib"
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/y08lin4/lyra-image-workbench/internal/config"
	"github.com/y08lin4/lyra-image-workbench/internal/jobs"
	"github.com/y08lin4/lyra-image-workbench/internal/llm"
	"github.com/y08lin4/lyra-image-workbench/internal/output"
	"github.com/y08lin4/lyra-image-workbench/internal/settings"
	"github.com/y08lin4/lyra-image-workbench/internal/spaceconfig"
	"github.com/y08lin4/lyra-image-workbench/internal/uploads"
)

type Service struct {
	store       *Store
	settings    *settings.FileStore
	spaceConfig *spaceconfig.Store
	uploads     *uploads.Store
	jobs        *jobs.Manager
	output      *output.Store
	llm         *llm.Client
}

func NewService(store *Store, settingsStore *settings.FileStore, spaceConfig *spaceconfig.Store, uploadStore *uploads.Store, jobManager *jobs.Manager, outputStore *output.Store, llmClient *llm.Client) *Service {
	return &Service{
		store:       store,
		settings:    settingsStore,
		spaceConfig: spaceConfig,
		uploads:     uploadStore,
		jobs:        jobManager,
		output:      outputStore,
		llm:         llmClient,
	}
}

func (s *Service) TextToPrompt(ctx context.Context, spaceToken string, req TextRequest) (Record, error) {
	input := strings.TrimSpace(req.Input)
	if input == "" {
		return Record{}, errors.New("请输入需要扩写的文字想法")
	}
	apiKey, err := s.apiKey(spaceToken, req.RuntimeAPIKey)
	if err != nil {
		return Record{}, err
	}
	started := time.Now()
	resp, err := s.llm.Complete(ctx, llm.Request{
		BaseURL:     s.settings.Get().NewAPIBaseURL,
		APIKey:      apiKey,
		Model:       config.DefaultPromptModel,
		System:      textSystemPrompt(),
		User:        textUserPrompt(input, strings.TrimSpace(req.Style), strings.TrimSpace(req.Ratio), strings.TrimSpace(req.Target)),
		TimeoutSec:  config.DefaultPromptTimeoutSec,
		Temperature: 0.4,
	})
	if err != nil {
		return Record{}, err
	}
	parsed := parsePromptJSON(resp.Text)
	ratio := resolvedPromptRatio(firstString(parsed, "ratio", ""), req.Ratio)
	record, err := s.newRecord(spaceToken, Record{
		Mode:           ModeTextToPrompt,
		Input:          input,
		Style:          firstString(parsed, "style", strings.TrimSpace(req.Style)),
		Ratio:          ratio,
		Language:       defaultString(req.Language, "zh"),
		Target:         defaultString(req.Target, "image-2"),
		FlatPrompt:     firstString(parsed, "flatPrompt", resp.Text),
		NegativePrompt: firstString(parsed, "negativePrompt", ""),
		MustKeep:       stringSlice(parsed["mustKeep"]),
		Raw:            resp.Text,
		Model:          config.DefaultPromptModel,
		ElapsedMs:      time.Since(started).Milliseconds(),
	})
	if err != nil {
		return Record{}, err
	}
	session, err := s.sessionFromRecord(record, SessionKindText, input)
	if err != nil {
		return Record{}, err
	}
	record.SessionID = session.ID
	record.VersionID = session.ActiveVersionID
	session.Versions[0].SourceRecordID = record.ID
	session.Messages[1].VersionID = record.VersionID
	if err := s.store.SaveSession(spaceToken, session); err != nil {
		return Record{}, err
	}
	return record, s.store.Save(spaceToken, record)
}

func (s *Service) ImageToPrompt(ctx context.Context, spaceToken string, req ImageRequest) (Record, error) {
	started := time.Now()
	path, mime, sourceURL, err := s.resolveImageSource(spaceToken, req.Source)
	if err != nil {
		return Record{}, err
	}
	apiKey, err := s.apiKey(spaceToken, req.RuntimeAPIKey)
	if err != nil {
		return Record{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Record{}, err
	}
	if len(data) == 0 {
		return Record{}, errors.New("图片内容为空")
	}
	metrics := inspectImageMetrics(data)
	metadata := inspectImageMetadata(data, mime)
	resp, err := s.llm.Complete(ctx, llm.Request{
		BaseURL:    s.settings.Get().NewAPIBaseURL,
		APIKey:     apiKey,
		Model:      config.DefaultPromptModel,
		System:     imageSystemPrompt(),
		User:       imageUserPrompt(strings.TrimSpace(req.Target), metrics, metadata),
		Image:      &llm.ImagePart{Mime: mime, Data: data},
		TimeoutSec: config.DefaultPromptTimeoutSec,
	})
	if err != nil {
		return Record{}, err
	}
	parsed := parsePromptJSON(resp.Text)
	ratio := metrics.Ratio
	if ratio == "" {
		ratio = normalizePromptRatio(firstString(parsed, "ratio", ""))
	}
	if ratio == "" {
		ratio = "auto"
	}
	description := objectMap(parsed["jsonDescription"])
	description = withImageMetrics(description, metrics, ratio)
	flatPrompt := ensurePromptRatio(firstString(parsed, "flatPrompt", resp.Text), ratio)
	record, err := s.newRecord(spaceToken, Record{
		Mode:            ModeImageToPrompt,
		Language:        defaultString(req.Language, "zh"),
		Target:          defaultString(req.Target, "image-2"),
		Source:          req.Source,
		SourceImageURL:  sourceURL,
		Ratio:           ratio,
		FlatPrompt:      flatPrompt,
		NegativePrompt:  firstString(parsed, "negativePrompt", ""),
		MustKeep:        stringSlice(parsed["mustKeep"]),
		Avoid:           stringSlice(parsed["avoid"]),
		JSONDescription: description,
		Metadata:        metadata,
		Raw:             resp.Text,
		Model:           config.DefaultPromptModel,
		ElapsedMs:       time.Since(started).Milliseconds(),
	})
	if err != nil {
		return Record{}, err
	}
	session, err := s.sessionFromRecord(record, SessionKindImage, "图片还原提示词")
	if err != nil {
		return Record{}, err
	}
	record.SessionID = session.ID
	record.VersionID = session.ActiveVersionID
	session.Versions[0].SourceRecordID = record.ID
	session.Messages[1].VersionID = record.VersionID
	if err := s.store.SaveSession(spaceToken, session); err != nil {
		return Record{}, err
	}
	return record, s.store.Save(spaceToken, record)
}

func (s *Service) List(spaceToken string, limit int) ([]Record, error) {
	return s.store.List(spaceToken, limit)
}

func (s *Service) Delete(spaceToken string, id string) (Record, bool, error) {
	return s.store.Delete(spaceToken, id)
}

func (s *Service) CreateSession(spaceToken string, req CreateSessionRequest) (PromptSession, error) {
	if _, err := s.spaceConfig.Get(spaceToken); err != nil {
		return PromptSession{}, err
	}
	prompt := strings.TrimSpace(req.InitialPrompt)
	if prompt == "" {
		return PromptSession{}, errors.New("请输入初始提示词")
	}
	now := time.Now()
	sessionID, err := newPromptID("psn")
	if err != nil {
		return PromptSession{}, err
	}
	versionID, err := newPromptID("ver")
	if err != nil {
		return PromptSession{}, err
	}
	userMessageID, err := newPromptID("msg")
	if err != nil {
		return PromptSession{}, err
	}
	assistantMessageID, err := newPromptID("msg")
	if err != nil {
		return PromptSession{}, err
	}
	session := PromptSession{
		ID:              sessionID,
		Kind:            SessionKindManual,
		Title:           defaultTitle(req.Title, prompt),
		Seed:            prompt,
		Target:          defaultString(req.Target, "image-2"),
		Ratio:           resolvedPromptRatio(req.Ratio, ""),
		Provider:        strings.TrimSpace(req.Provider),
		Model:           strings.TrimSpace(req.Model),
		ActiveVersionID: versionID,
		CreatedAt:       now,
		UpdatedAt:       now,
		Versions: []PromptVersion{{
			ID:             versionID,
			Index:          1,
			Prompt:         prompt,
			NegativePrompt: strings.TrimSpace(req.NegativePrompt),
			Ratio:          resolvedPromptRatio(req.Ratio, ""),
			MustKeep:       cleanStringSlice(req.MustKeep),
			Model:          config.DefaultPromptModel,
			CreatedAt:      now,
		}},
		Messages: []PromptMessage{
			{ID: userMessageID, Role: "user", Content: "创建提示词会话", CreatedAt: now},
			{ID: assistantMessageID, Role: "assistant", Content: prompt, VersionID: versionID, CreatedAt: now},
		},
	}
	return session, s.store.SaveSession(spaceToken, session)
}

func (s *Service) ListSessions(spaceToken string, limit int) ([]PromptSession, error) {
	return s.store.ListSessions(spaceToken, limit)
}

func (s *Service) GetSession(spaceToken string, id string) (PromptSession, bool, error) {
	return s.store.GetSession(spaceToken, id)
}

func (s *Service) DeleteSession(spaceToken string, id string) (PromptSession, bool, error) {
	return s.store.DeleteSession(spaceToken, id)
}

func (s *Service) RefineSession(ctx context.Context, spaceToken string, id string, req RefineRequest) (PromptSession, error) {
	message := strings.TrimSpace(req.Message)
	if message == "" {
		return PromptSession{}, errors.New("请输入修改要求")
	}
	session, ok, err := s.store.GetSession(spaceToken, id)
	if err != nil {
		return PromptSession{}, err
	}
	if !ok {
		return PromptSession{}, errors.New("提示词会话不存在")
	}
	current, ok := session.version(req.CurrentVersionID)
	if !ok {
		return PromptSession{}, errors.New("提示词版本不存在")
	}
	apiKey, err := s.apiKey(spaceToken, req.RuntimeAPIKey)
	if err != nil {
		return PromptSession{}, err
	}
	started := time.Now()
	resp, err := s.llm.Complete(ctx, llm.Request{
		BaseURL:     s.settings.Get().NewAPIBaseURL,
		APIKey:      apiKey,
		Model:       config.DefaultPromptModel,
		System:      refineSystemPrompt(),
		User:        refineUserPrompt(session, current, message, strings.TrimSpace(req.Provider), strings.TrimSpace(req.Model)),
		TimeoutSec:  config.DefaultPromptTimeoutSec,
		Temperature: 0.35,
	})
	if err != nil {
		return PromptSession{}, err
	}
	parsed := parsePromptJSON(resp.Text)
	nextPrompt := firstString(parsed, "flatPrompt", resp.Text)
	if strings.TrimSpace(nextPrompt) == "" {
		return PromptSession{}, errors.New("提示词模型没有返回可用提示词")
	}
	nextRatio := resolvedPromptRatio(firstString(parsed, "ratio", ""), firstNonEmpty(current.Ratio, session.Ratio))
	now := time.Now()
	versionID, err := newPromptID("ver")
	if err != nil {
		return PromptSession{}, err
	}
	userMessageID, err := newPromptID("msg")
	if err != nil {
		return PromptSession{}, err
	}
	assistantMessageID, err := newPromptID("msg")
	if err != nil {
		return PromptSession{}, err
	}
	version := PromptVersion{
		ID:             versionID,
		Index:          len(session.Versions) + 1,
		Prompt:         strings.TrimSpace(nextPrompt),
		NegativePrompt: firstString(parsed, "negativePrompt", current.NegativePrompt),
		Ratio:          nextRatio,
		MustKeep:       fallbackSlice(stringSlice(parsed["mustKeep"]), current.MustKeep),
		Avoid:          fallbackSlice(stringSlice(parsed["avoid"]), current.Avoid),
		Notes:          firstString(parsed, "notes", ""),
		Model:          config.DefaultPromptModel,
		ElapsedMs:      time.Since(started).Milliseconds(),
		CreatedAt:      now,
	}
	session.Messages = append(session.Messages,
		PromptMessage{ID: userMessageID, Role: "user", Content: message, CreatedAt: now},
		PromptMessage{ID: assistantMessageID, Role: "assistant", Content: version.Prompt, VersionID: version.ID, CreatedAt: now},
	)
	session.Versions = append(session.Versions, version)
	session.ActiveVersionID = version.ID
	session.Ratio = version.Ratio
	session.Provider = strings.TrimSpace(req.Provider)
	session.Model = strings.TrimSpace(req.Model)
	session.UpdatedAt = now
	if strings.TrimSpace(session.Title) == "" {
		session.Title = defaultTitle("", version.Prompt)
	}
	if err := s.store.SaveSession(spaceToken, session); err != nil {
		return PromptSession{}, err
	}
	return session, nil
}

func (s *Service) GenerateIdeas(ctx context.Context, spaceToken string, req InspirationIdeasRequest) ([]InspirationIdea, error) {
	apiKey, err := s.apiKey(spaceToken, req.RuntimeAPIKey)
	if err != nil {
		return nil, err
	}
	count := req.Count
	if count <= 0 {
		count = 6
	}
	if count > 12 {
		count = 12
	}
	resp, err := s.llm.Complete(ctx, llm.Request{
		BaseURL:     s.settings.Get().NewAPIBaseURL,
		APIKey:      apiKey,
		Model:       config.DefaultPromptModel,
		System:      inspirationSystemPrompt(),
		User:        inspirationIdeasUserPrompt(req, count),
		TimeoutSec:  config.DefaultPromptTimeoutSec,
		Temperature: 0.75,
	})
	if err != nil {
		return nil, err
	}
	ideas := parseIdeas(resp.Text)
	if len(ideas) == 0 {
		return nil, errors.New("提示词模型没有返回灵感")
	}
	now := time.Now().Format(time.RFC3339)
	for i := range ideas {
		if strings.TrimSpace(ideas[i].ID) == "" {
			id, err := newPromptID("idea")
			if err != nil {
				return nil, err
			}
			ideas[i].ID = id
		}
		ideas[i].Category = defaultString(ideas[i].Category, req.Category)
		ideas[i].Mood = defaultString(ideas[i].Mood, req.Mood)
		ideas[i].Style = defaultString(ideas[i].Style, req.Style)
		ideas[i].CreatedAt = now
	}
	return ideas, nil
}

func (s *Service) ExpandIdea(ctx context.Context, spaceToken string, req InspirationExpandRequest) (PromptSession, error) {
	if strings.TrimSpace(req.Idea.Title) == "" && strings.TrimSpace(req.Idea.Summary) == "" {
		return PromptSession{}, errors.New("请先选择一个灵感")
	}
	apiKey, err := s.apiKey(spaceToken, req.RuntimeAPIKey)
	if err != nil {
		return PromptSession{}, err
	}
	started := time.Now()
	resp, err := s.llm.Complete(ctx, llm.Request{
		BaseURL:     s.settings.Get().NewAPIBaseURL,
		APIKey:      apiKey,
		Model:       config.DefaultPromptModel,
		System:      textSystemPrompt(),
		User:        inspirationExpandUserPrompt(req),
		TimeoutSec:  config.DefaultPromptTimeoutSec,
		Temperature: 0.45,
	})
	if err != nil {
		return PromptSession{}, err
	}
	parsed := parsePromptJSON(resp.Text)
	ratio := resolvedPromptRatio(firstString(parsed, "ratio", ""), req.Ratio)
	record, err := s.newRecord(spaceToken, Record{
		Mode:           ModeTextToPrompt,
		Input:          strings.TrimSpace(req.Idea.Title + " " + req.Idea.Summary),
		Style:          firstString(parsed, "style", strings.TrimSpace(req.Idea.Style)),
		Ratio:          ratio,
		Language:       "zh",
		Target:         defaultString(req.Target, "image-2"),
		FlatPrompt:     firstString(parsed, "flatPrompt", resp.Text),
		NegativePrompt: firstString(parsed, "negativePrompt", ""),
		MustKeep:       stringSlice(parsed["mustKeep"]),
		Raw:            resp.Text,
		Model:          config.DefaultPromptModel,
		ElapsedMs:      time.Since(started).Milliseconds(),
	})
	if err != nil {
		return PromptSession{}, err
	}
	session, err := s.sessionFromRecord(record, SessionKindInspiration, strings.TrimSpace(req.Idea.Title))
	if err != nil {
		return PromptSession{}, err
	}
	session.Provider = strings.TrimSpace(req.Provider)
	session.Model = strings.TrimSpace(req.Model)
	record.SessionID = session.ID
	record.VersionID = session.ActiveVersionID
	session.Versions[0].SourceRecordID = record.ID
	session.Versions[0].ElapsedMs = record.ElapsedMs
	session.Messages[0].Content = "灵感扩写：" + strings.TrimSpace(req.Idea.Title+" "+req.Idea.Summary)
	session.Messages[1].VersionID = record.VersionID
	if err := s.store.SaveSession(spaceToken, session); err != nil {
		return PromptSession{}, err
	}
	if err := s.store.Save(spaceToken, record); err != nil {
		return PromptSession{}, err
	}
	return session, nil
}

func (s *Service) apiKey(spaceToken string, runtimeAPIKey string) (string, error) {
	cfg, err := s.spaceConfig.Get(spaceToken)
	if err != nil {
		return "", err
	}
	apiKey := strings.TrimSpace(runtimeAPIKey)
	if apiKey == "" {
		apiKey = strings.TrimSpace(cfg.APIKey)
	}
	if apiKey == "" && s.settings != nil {
		apiKey = strings.TrimSpace(s.settings.Get().SystemAPIKey)
	}
	if apiKey == "" {
		return "", errors.New("codex-key is not configured; save it locally, upload it to cloud, or ask an admin to configure the system upstream key")
	}
	return apiKey, nil
}

func (s *Service) resolveImageSource(spaceToken string, source Source) (string, string, string, error) {
	switch strings.TrimSpace(source.Type) {
	case "upload":
		item, path, err := s.uploads.GetReferenceImage(spaceToken, source.UploadID)
		if err != nil {
			return "", "", "", err
		}
		return path, item.Mime, "", nil
	case "result":
		job, ok, err := s.jobs.Get(spaceToken, source.TaskID)
		if err != nil {
			return "", "", "", err
		}
		if !ok {
			return "", "", "", errors.New("任务不存在")
		}
		for _, result := range job.Results {
			if result.Index == source.Index && result.OK && result.ImageURL != "" {
				path, mime, err := s.resolveTaskResultImage(job, result)
				return path, mime, publicSourceImageURL(source, result.ImageURL), err
			}
		}
		return "", "", "", errors.New("任务图片不存在")
	default:
		return "", "", "", errors.New("图片来源无效")
	}
}

func (s *Service) resolveTaskResultImage(job jobs.Job, result jobs.Result) (string, string, error) {
	if result.OutputDate != "" && result.OutputFileName != "" {
		return s.output.Resolve(job.SpaceToken, result.OutputDate, result.OutputFileName)
	}
	return s.output.ResolveURL(result.ImageURL)
}

func (s *Service) newRecord(spaceToken string, record Record) (Record, error) {
	if _, err := s.spaceConfig.Get(spaceToken); err != nil {
		return Record{}, err
	}
	id, err := newRecordID()
	if err != nil {
		return Record{}, err
	}
	record.ID = id
	record.CreatedAt = time.Now()
	record.FlatPrompt = strings.TrimSpace(record.FlatPrompt)
	record.NegativePrompt = strings.TrimSpace(record.NegativePrompt)
	if record.FlatPrompt == "" {
		return Record{}, errors.New("提示词模型没有返回可用提示词")
	}
	return record, nil
}

func newRecordID() (string, error) {
	return newPromptID("ppt")
}

func newPromptID(prefix string) (string, error) {
	var bytes [8]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", err
	}
	return prefix + "_" + time.Now().Format("20060102150405") + "_" + hex.EncodeToString(bytes[:]), nil
}

func (s *Service) sessionFromRecord(record Record, kind SessionKind, seed string) (PromptSession, error) {
	now := record.CreatedAt
	if now.IsZero() {
		now = time.Now()
	}
	sessionID, err := newPromptID("psn")
	if err != nil {
		return PromptSession{}, err
	}
	versionID, err := newPromptID("ver")
	if err != nil {
		return PromptSession{}, err
	}
	userMessageID, err := newPromptID("msg")
	if err != nil {
		return PromptSession{}, err
	}
	assistantMessageID, err := newPromptID("msg")
	if err != nil {
		return PromptSession{}, err
	}
	userContent := strings.TrimSpace(seed)
	if userContent == "" {
		userContent = "生成提示词"
	}
	return PromptSession{
		ID:              sessionID,
		Kind:            kind,
		Title:           defaultTitle(record.Input, record.FlatPrompt),
		Seed:            userContent,
		Source:          record.Source,
		SourceImageURL:  record.SourceImageURL,
		Target:          record.Target,
		Ratio:           strings.TrimSpace(record.Ratio),
		ActiveVersionID: versionID,
		CreatedAt:       now,
		UpdatedAt:       now,
		Versions: []PromptVersion{{
			ID:             versionID,
			Index:          1,
			Prompt:         strings.TrimSpace(record.FlatPrompt),
			NegativePrompt: strings.TrimSpace(record.NegativePrompt),
			Ratio:          strings.TrimSpace(record.Ratio),
			MustKeep:       cleanStringSlice(record.MustKeep),
			Avoid:          cleanStringSlice(record.Avoid),
			SourceRecordID: record.ID,
			Model:          record.Model,
			ElapsedMs:      record.ElapsedMs,
			CreatedAt:      now,
		}},
		Messages: []PromptMessage{
			{ID: userMessageID, Role: "user", Content: userContent, CreatedAt: now},
			{ID: assistantMessageID, Role: "assistant", Content: strings.TrimSpace(record.FlatPrompt), VersionID: versionID, CreatedAt: now},
		},
	}, nil
}

func (session PromptSession) version(id string) (PromptVersion, bool) {
	id = strings.TrimSpace(id)
	if id == "" {
		id = session.ActiveVersionID
	}
	for _, version := range session.Versions {
		if version.ID == id {
			return version, true
		}
	}
	if len(session.Versions) > 0 {
		return session.Versions[len(session.Versions)-1], true
	}
	return PromptVersion{}, false
}

func defaultTitle(seed string, prompt string) string {
	title := strings.TrimSpace(seed)
	if title == "" {
		title = strings.TrimSpace(prompt)
	}
	title = strings.Join(strings.Fields(title), " ")
	runes := []rune(title)
	if len(runes) > 24 {
		return string(runes[:24]) + "…"
	}
	if title == "" {
		return "提示词会话"
	}
	return title
}

func cleanStringSlice(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func fallbackSlice(values []string, fallback []string) []string {
	values = cleanStringSlice(values)
	if len(values) > 0 {
		return values
	}
	return cleanStringSlice(fallback)
}

type ImageMetrics struct {
	Width       int
	Height      int
	Ratio       string
	Orientation string
}

func inspectImageMetrics(data []byte) ImageMetrics {
	width, height := imageDimensions(data)
	ratio := closestSupportedRatio(width, height)
	return ImageMetrics{
		Width:       width,
		Height:      height,
		Ratio:       ratio,
		Orientation: orientationLabel(width, height),
	}
}

func imageDimensions(data []byte) (int, int) {
	if width, height, ok := pngDimensions(data); ok {
		return width, height
	}
	if width, height, ok := jpegDimensions(data); ok {
		return width, height
	}
	if width, height, ok := webpDimensions(data); ok {
		return width, height
	}
	return 0, 0
}

func pngDimensions(data []byte) (int, int, bool) {
	if len(data) < 24 || string(data[:8]) != "\x89PNG\r\n\x1a\n" || string(data[12:16]) != "IHDR" {
		return 0, 0, false
	}
	width := int(binary.BigEndian.Uint32(data[16:20]))
	height := int(binary.BigEndian.Uint32(data[20:24]))
	return width, height, width > 0 && height > 0
}

func jpegDimensions(data []byte) (int, int, bool) {
	if len(data) < 4 || data[0] != 0xff || data[1] != 0xd8 {
		return 0, 0, false
	}
	for i := 2; i+3 < len(data); {
		if data[i] != 0xff {
			i++
			continue
		}
		for i < len(data) && data[i] == 0xff {
			i++
		}
		if i >= len(data) {
			break
		}
		marker := data[i]
		i++
		if marker == 0xd8 || marker == 0xd9 {
			continue
		}
		if marker == 0xda || i+1 >= len(data) {
			break
		}
		size := int(binary.BigEndian.Uint16(data[i : i+2]))
		if size < 2 || i+size > len(data) {
			break
		}
		if isJPEGSOFMarker(marker) && size >= 7 {
			height := int(binary.BigEndian.Uint16(data[i+3 : i+5]))
			width := int(binary.BigEndian.Uint16(data[i+5 : i+7]))
			return width, height, width > 0 && height > 0
		}
		i += size
	}
	return 0, 0, false
}

func isJPEGSOFMarker(marker byte) bool {
	switch marker {
	case 0xc0, 0xc1, 0xc2, 0xc3, 0xc5, 0xc6, 0xc7, 0xc9, 0xca, 0xcb, 0xcd, 0xce, 0xcf:
		return true
	default:
		return false
	}
}

func webpDimensions(data []byte) (int, int, bool) {
	if len(data) < 30 || string(data[:4]) != "RIFF" || string(data[8:12]) != "WEBP" {
		return 0, 0, false
	}
	for offset := 12; offset+8 <= len(data); {
		chunkType := string(data[offset : offset+4])
		chunkSize := int(binary.LittleEndian.Uint32(data[offset+4 : offset+8]))
		payload := offset + 8
		if chunkSize < 0 || payload+chunkSize > len(data) {
			break
		}
		switch chunkType {
		case "VP8X":
			if chunkSize >= 10 {
				width := 1 + int(uint32(data[payload+4])|uint32(data[payload+5])<<8|uint32(data[payload+6])<<16)
				height := 1 + int(uint32(data[payload+7])|uint32(data[payload+8])<<8|uint32(data[payload+9])<<16)
				return width, height, width > 0 && height > 0
			}
		case "VP8 ":
			if chunkSize >= 10 && payload+10 <= len(data) {
				width := int(binary.LittleEndian.Uint16(data[payload+6:payload+8]) & 0x3fff)
				height := int(binary.LittleEndian.Uint16(data[payload+8:payload+10]) & 0x3fff)
				return width, height, width > 0 && height > 0
			}
		case "VP8L":
			if chunkSize >= 5 && data[payload] == 0x2f {
				bits := uint32(data[payload+1]) | uint32(data[payload+2])<<8 | uint32(data[payload+3])<<16 | uint32(data[payload+4])<<24
				width := int(bits&0x3fff) + 1
				height := int((bits>>14)&0x3fff) + 1
				return width, height, width > 0 && height > 0
			}
		}
		offset = payload + chunkSize
		if chunkSize%2 == 1 {
			offset++
		}
	}
	return 0, 0, false
}

func metricsPrompt(metrics ImageMetrics) string {
	if metrics.Width <= 0 || metrics.Height <= 0 {
		return "未知，请按图片视觉画幅判断最接近比例"
	}
	ratio := valueOr(metrics.Ratio, "auto")
	orientation := valueOr(metrics.Orientation, "未知画幅")
	return fmt.Sprintf("%dx%d，%s，最接近支持比例：%s", metrics.Width, metrics.Height, orientation, ratio)
}

func withImageMetrics(description map[string]any, metrics ImageMetrics, ratio string) map[string]any {
	out := map[string]any{}
	for key, value := range description {
		out[key] = value
	}
	if ratio != "" {
		out["ratio"] = ratio
	}
	if metrics.Width > 0 && metrics.Height > 0 {
		out["sourceSize"] = fmt.Sprintf("%dx%d", metrics.Width, metrics.Height)
		out["orientation"] = metrics.Orientation
	}
	return out
}

func inspectImageMetadata(data []byte, mime string) map[string]any {
	values := map[string]any{}
	mime = strings.ToLower(strings.TrimSpace(strings.Split(mime, ";")[0]))
	if mime == "image/png" || bytes.HasPrefix(data, []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}) {
		for key, value := range pngTextMetadata(data) {
			values[key] = value
		}
	} else if mime == "image/jpeg" || bytes.HasPrefix(data, []byte{0xff, 0xd8}) {
		for key, value := range jpegMetadata(data) {
			values[key] = value
		}
	} else if mime == "image/webp" || (len(data) >= 12 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WEBP") {
		for key, value := range webpMetadata(data) {
			values[key] = value
		}
	}
	return normalizeImageMetadata(values)
}

func pngTextMetadata(data []byte) map[string]any {
	out := map[string]any{}
	if len(data) < 12 || string(data[:8]) != "\x89PNG\r\n\x1a\n" {
		return out
	}
	for offset := 8; offset+12 <= len(data); {
		chunkLen := int(binary.BigEndian.Uint32(data[offset : offset+4]))
		chunkType := string(data[offset+4 : offset+8])
		payloadStart := offset + 8
		payloadEnd := payloadStart + chunkLen
		if chunkLen < 0 || payloadEnd+4 > len(data) {
			break
		}
		payload := data[payloadStart:payloadEnd]
		switch chunkType {
		case "tEXt":
			if key, value, ok := parsePNGText(payload); ok {
				out[key] = value
			}
		case "zTXt":
			if key, value, ok := parsePNGZText(payload); ok {
				out[key] = value
			}
		case "iTXt":
			if key, value, ok := parsePNGIText(payload); ok {
				out[key] = value
			}
		}
		offset = payloadEnd + 4
		if chunkType == "IEND" {
			break
		}
	}
	return out
}

func parsePNGText(payload []byte) (string, string, bool) {
	parts := bytes.SplitN(payload, []byte{0}, 2)
	if len(parts) != 2 {
		return "", "", false
	}
	key := cleanMetadataKey(string(parts[0]))
	value := cleanMetadataText(string(parts[1]))
	return key, value, key != "" && value != ""
}

func parsePNGZText(payload []byte) (string, string, bool) {
	parts := bytes.SplitN(payload, []byte{0}, 2)
	if len(parts) != 2 || len(parts[1]) < 2 || parts[1][0] != 0 {
		return "", "", false
	}
	key := cleanMetadataKey(string(parts[0]))
	value, err := inflateZlib(parts[1][1:])
	if err != nil {
		return "", "", false
	}
	value = cleanMetadataText(value)
	return key, value, key != "" && value != ""
}

func parsePNGIText(payload []byte) (string, string, bool) {
	keywordEnd := bytes.IndexByte(payload, 0)
	if keywordEnd < 0 || keywordEnd+3 > len(payload) {
		return "", "", false
	}
	key := cleanMetadataKey(string(payload[:keywordEnd]))
	compressed := payload[keywordEnd+1] == 1
	compressionMethod := payload[keywordEnd+2]
	rest := payload[keywordEnd+3:]
	langEnd := bytes.IndexByte(rest, 0)
	if langEnd < 0 {
		return "", "", false
	}
	rest = rest[langEnd+1:]
	translatedEnd := bytes.IndexByte(rest, 0)
	if translatedEnd < 0 {
		return "", "", false
	}
	textBytes := rest[translatedEnd+1:]
	var value string
	if compressed {
		if compressionMethod != 0 {
			return "", "", false
		}
		decoded, err := inflateZlib(textBytes)
		if err != nil {
			return "", "", false
		}
		value = decoded
	} else {
		value = string(textBytes)
	}
	value = cleanMetadataText(value)
	return key, value, key != "" && value != ""
}

func jpegMetadata(data []byte) map[string]any {
	out := map[string]any{}
	if len(data) < 4 || data[0] != 0xff || data[1] != 0xd8 {
		return out
	}
	var comments []string
	for i := 2; i+3 < len(data); {
		if data[i] != 0xff {
			i++
			continue
		}
		for i < len(data) && data[i] == 0xff {
			i++
		}
		if i >= len(data) {
			break
		}
		marker := data[i]
		i++
		if marker == 0xda || marker == 0xd9 {
			break
		}
		if marker == 0xd8 {
			continue
		}
		if i+1 >= len(data) {
			break
		}
		size := int(binary.BigEndian.Uint16(data[i : i+2]))
		if size < 2 || i+size > len(data) {
			break
		}
		payload := data[i+2 : i+size]
		switch marker {
		case 0xfe:
			if text := cleanMetadataText(string(payload)); text != "" {
				comments = append(comments, text)
			}
		case 0xe1:
			if bytes.HasPrefix(payload, []byte("Exif\x00\x00")) {
				for key, value := range parseExifMetadata(payload[6:]) {
					out[key] = value
				}
			} else if bytes.HasPrefix(payload, []byte("http://ns.adobe.com/xap/1.0/\x00")) {
				if text := cleanMetadataText(string(payload[len("http://ns.adobe.com/xap/1.0/\x00"):])); text != "" {
					out["xmp"] = limitMetadataText(text, 2000)
				}
			}
		}
		i += size
	}
	if len(comments) == 1 {
		out["comment"] = comments[0]
	} else if len(comments) > 1 {
		out["comments"] = comments
	}
	return out
}

func webpMetadata(data []byte) map[string]any {
	out := map[string]any{}
	if len(data) < 12 || string(data[:4]) != "RIFF" || string(data[8:12]) != "WEBP" {
		return out
	}
	for offset := 12; offset+8 <= len(data); {
		chunkType := string(data[offset : offset+4])
		chunkSize := int(binary.LittleEndian.Uint32(data[offset+4 : offset+8]))
		payload := offset + 8
		if chunkSize < 0 || payload+chunkSize > len(data) {
			break
		}
		switch chunkType {
		case "EXIF":
			for key, value := range parseExifMetadata(data[payload : payload+chunkSize]) {
				out[key] = value
			}
		case "XMP ":
			if text := cleanMetadataText(string(data[payload : payload+chunkSize])); text != "" {
				out["xmp"] = limitMetadataText(text, 2000)
			}
		}
		offset = payload + chunkSize
		if chunkSize%2 == 1 {
			offset++
		}
	}
	return out
}

func parseExifMetadata(data []byte) map[string]any {
	out := map[string]any{}
	if len(data) < 8 {
		return out
	}
	var order binary.ByteOrder
	switch string(data[:2]) {
	case "II":
		order = binary.LittleEndian
	case "MM":
		order = binary.BigEndian
	default:
		return out
	}
	if order.Uint16(data[2:4]) != 42 {
		return out
	}
	ifdOffset := int(order.Uint32(data[4:8]))
	parseExifIFD(data, order, ifdOffset, out)
	if exifOffset, ok := exifOffsetValue(data, order, ifdOffset); ok {
		parseExifIFD(data, order, exifOffset, out)
	}
	return out
}

func parseExifIFD(data []byte, order binary.ByteOrder, offset int, out map[string]any) {
	if offset < 0 || offset+2 > len(data) {
		return
	}
	count := int(order.Uint16(data[offset : offset+2]))
	pos := offset + 2
	for i := 0; i < count && pos+12 <= len(data); i++ {
		tag := order.Uint16(data[pos : pos+2])
		fieldType := order.Uint16(data[pos+2 : pos+4])
		numValues := int(order.Uint32(data[pos+4 : pos+8]))
		valueOffset := pos + 8
		raw := exifFieldBytes(data, order, fieldType, numValues, valueOffset)
		if len(raw) > 0 {
			switch tag {
			case 0x010e:
				setMetadataString(out, "imageDescription", decodeExifText(raw))
			case 0x0131:
				setMetadataString(out, "software", decodeExifText(raw))
			case 0x013b:
				setMetadataString(out, "artist", decodeExifText(raw))
			case 0x8298:
				setMetadataString(out, "copyright", decodeExifText(raw))
			case 0x9286:
				setMetadataString(out, "userComment", decodeExifUserComment(raw))
			}
		}
		pos += 12
	}
}

func exifOffsetValue(data []byte, order binary.ByteOrder, offset int) (int, bool) {
	if offset < 0 || offset+2 > len(data) {
		return 0, false
	}
	count := int(order.Uint16(data[offset : offset+2]))
	pos := offset + 2
	for i := 0; i < count && pos+12 <= len(data); i++ {
		tag := order.Uint16(data[pos : pos+2])
		fieldType := order.Uint16(data[pos+2 : pos+4])
		numValues := order.Uint32(data[pos+4 : pos+8])
		if tag == 0x8769 && fieldType == 4 && numValues == 1 {
			return int(order.Uint32(data[pos+8 : pos+12])), true
		}
		pos += 12
	}
	return 0, false
}

func exifFieldBytes(data []byte, order binary.ByteOrder, fieldType uint16, numValues int, valueOffset int) []byte {
	size := exifTypeSize(fieldType) * numValues
	if size <= 0 {
		return nil
	}
	if size <= 4 {
		if valueOffset+size > len(data) {
			return nil
		}
		return data[valueOffset : valueOffset+size]
	}
	if valueOffset+4 > len(data) {
		return nil
	}
	offset := int(order.Uint32(data[valueOffset : valueOffset+4]))
	if offset < 0 || offset+size > len(data) {
		return nil
	}
	return data[offset : offset+size]
}

func exifTypeSize(fieldType uint16) int {
	switch fieldType {
	case 1, 2, 6, 7:
		return 1
	case 3, 8:
		return 2
	case 4, 9:
		return 4
	case 5, 10:
		return 8
	default:
		return 0
	}
}

func decodeExifText(data []byte) string {
	return cleanMetadataText(strings.TrimRight(string(data), "\x00"))
}

func decodeExifUserComment(data []byte) string {
	if len(data) >= 8 {
		prefix := strings.TrimRight(string(data[:8]), "\x00 ")
		body := data[8:]
		if strings.EqualFold(prefix, "ASCII") || strings.EqualFold(prefix, "JIS") || strings.EqualFold(prefix, "UNICODE") || prefix == "" {
			return cleanMetadataText(strings.TrimRight(string(body), "\x00"))
		}
	}
	return decodeExifText(data)
}

func normalizeImageMetadata(values map[string]any) map[string]any {
	if len(values) == 0 {
		return nil
	}
	out := map[string]any{}
	for key, value := range values {
		cleanKey := cleanMetadataKey(key)
		if cleanKey == "" {
			continue
		}
		switch typed := value.(type) {
		case string:
			if text := cleanMetadataText(typed); text != "" {
				out[cleanKey] = limitMetadataText(text, 4000)
			}
		case []string:
			items := make([]string, 0, len(typed))
			for _, item := range typed {
				if text := cleanMetadataText(item); text != "" {
					items = append(items, limitMetadataText(text, 1000))
				}
			}
			if len(items) > 0 {
				out[cleanKey] = items
			}
		default:
			out[cleanKey] = typed
		}
	}
	enrichPromptMetadata(out)
	if len(out) == 0 {
		return nil
	}
	return out
}

func enrichPromptMetadata(out map[string]any) {
	for _, key := range []string{"parameters", "prompt", "comment", "description", "userComment", "imageDescription"} {
		raw, ok := out[key].(string)
		if !ok || strings.TrimSpace(raw) == "" {
			continue
		}
		if prompt, negative := splitStableDiffusionParameters(raw); prompt != "" || negative != "" {
			if prompt != "" {
				out["metadataPrompt"] = prompt
			}
			if negative != "" {
				out["metadataNegativePrompt"] = negative
			}
			return
		}
	}
}

func splitStableDiffusionParameters(text string) (string, string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return "", ""
	}
	lower := strings.ToLower(text)
	negIndex := strings.Index(lower, "negative prompt:")
	if negIndex < 0 {
		return limitMetadataText(text, 3000), ""
	}
	prompt := strings.TrimSpace(text[:negIndex])
	rest := strings.TrimSpace(text[negIndex+len("negative prompt:"):])
	paramsIndex := regexp.MustCompile(`(?i)\n\s*(steps|sampler|cfg scale|seed|size|model):`).FindStringIndex(rest)
	if paramsIndex != nil {
		rest = strings.TrimSpace(rest[:paramsIndex[0]])
	}
	return limitMetadataText(prompt, 3000), limitMetadataText(rest, 2000)
}

func metadataPrompt(metadata map[string]any) string {
	if len(metadata) == 0 {
		return "未发现可用 metadata"
	}
	preferred := []string{"metadataPrompt", "metadataNegativePrompt", "parameters", "prompt", "workflow", "software", "comment", "userComment", "imageDescription", "xmp"}
	lines := make([]string, 0, len(metadata))
	seen := map[string]bool{}
	for _, key := range preferred {
		if value, ok := metadata[key]; ok {
			lines = append(lines, fmt.Sprintf("- %s: %s", key, metadataValuePreview(value, 900)))
			seen[key] = true
		}
	}
	for key, value := range metadata {
		if seen[key] {
			continue
		}
		lines = append(lines, fmt.Sprintf("- %s: %s", key, metadataValuePreview(value, 500)))
	}
	if len(lines) == 0 {
		return "未发现可用 metadata"
	}
	return strings.Join(lines, "\n")
}

func metadataValuePreview(value any, limit int) string {
	switch typed := value.(type) {
	case string:
		return limitMetadataText(typed, limit)
	case []string:
		return limitMetadataText(strings.Join(typed, " / "), limit)
	default:
		data, _ := json.Marshal(typed)
		return limitMetadataText(string(data), limit)
	}
}

func setMetadataString(out map[string]any, key string, value string) {
	value = cleanMetadataText(value)
	if value != "" {
		out[key] = value
	}
}

func cleanMetadataKey(key string) string {
	key = strings.TrimSpace(strings.Trim(key, "\x00"))
	key = strings.ReplaceAll(key, " ", "")
	if key == "" {
		return ""
	}
	switch strings.ToLower(key) {
	case "parameters":
		return "parameters"
	case "prompt":
		return "prompt"
	case "workflow":
		return "workflow"
	case "negativeprompt":
		return "negativePrompt"
	case "description":
		return "description"
	default:
		return key
	}
}

func cleanMetadataText(value string) string {
	value = strings.ReplaceAll(value, "\x00", "")
	value = strings.TrimSpace(value)
	value = strings.Join(strings.Fields(value), " ")
	return value
}

func limitMetadataText(value string, limit int) string {
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if limit <= 0 || len(runes) <= limit {
		return value
	}
	return string(runes[:limit]) + "…"
}

func inflateZlib(data []byte) (string, error) {
	reader, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	defer reader.Close()
	decoded, err := io.ReadAll(io.LimitReader(reader, 1<<20))
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}

func ensurePromptRatio(prompt string, ratio string) string {
	prompt = strings.TrimSpace(prompt)
	ratio = normalizePromptRatio(ratio)
	if prompt == "" || ratio == "" || ratio == "auto" {
		return prompt
	}
	if strings.Contains(prompt, ratio) {
		return prompt
	}
	suffix := fmt.Sprintf("画面比例为 %s。", ratio)
	if strings.HasSuffix(prompt, "。") || strings.HasSuffix(prompt, ".") || strings.HasSuffix(prompt, "！") || strings.HasSuffix(prompt, "!") {
		return prompt + suffix
	}
	return prompt + "，" + suffix
}

func normalizePromptRatio(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "：", ":")
	value = strings.ReplaceAll(value, "×", "x")
	if value == "" {
		return ""
	}
	if value == "auto" || value == "自动" || strings.Contains(value, "自动") {
		return "auto"
	}
	if width, height, ok := sizeLikeRatio(value); ok {
		return closestSupportedRatio(width, height)
	}
	re := regexp.MustCompile(`(?:^|[^0-9])((?:1\s*:\s*1)|(?:2\s*:\s*3)|(?:3\s*:\s*2)|(?:3\s*:\s*4)|(?:4\s*:\s*3)|(?:9\s*:\s*16)|(?:16\s*:\s*9))(?:[^0-9]|$)`)
	match := re.FindStringSubmatch(value)
	if len(match) > 1 {
		return strings.ReplaceAll(match[1], " ", "")
	}
	switch value {
	case "1:1", "2:3", "3:2", "3:4", "4:3", "9:16", "16:9":
		return value
	default:
		return ""
	}
}

func resolvedPromptRatio(value string, fallback string) string {
	ratio := normalizePromptRatio(value)
	if ratio != "" {
		return ratio
	}
	ratio = normalizePromptRatio(fallback)
	if ratio != "" {
		return ratio
	}
	return "auto"
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func sizeLikeRatio(value string) (int, int, bool) {
	re := regexp.MustCompile(`(\d{2,5})\s*x\s*(\d{2,5})`)
	match := re.FindStringSubmatch(value)
	if len(match) != 3 {
		return 0, 0, false
	}
	width, _ := strconv.Atoi(match[1])
	height, _ := strconv.Atoi(match[2])
	return width, height, width > 0 && height > 0
}

func closestSupportedRatio(width int, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	source := float64(width) / float64(height)
	type candidate struct {
		ratio string
		value float64
	}
	candidates := []candidate{
		{"1:1", 1.0 / 1.0},
		{"2:3", 2.0 / 3.0},
		{"3:2", 3.0 / 2.0},
		{"3:4", 3.0 / 4.0},
		{"4:3", 4.0 / 3.0},
		{"9:16", 9.0 / 16.0},
		{"16:9", 16.0 / 9.0},
	}
	best := candidates[0]
	bestDistance := math.Abs(math.Log(source / best.value))
	for _, item := range candidates[1:] {
		distance := math.Abs(math.Log(source / item.value))
		if distance < bestDistance {
			best = item
			bestDistance = distance
		}
	}
	return best.ratio
}

func orientationLabel(width int, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	if width == height {
		return "方图"
	}
	if width > height {
		return "横图"
	}
	return "竖图"
}

func parsePromptJSON(text string) map[string]any {
	candidates := []string{strings.TrimSpace(text)}
	if fenced := extractFence(text); fenced != "" {
		candidates = append([]string{fenced}, candidates...)
	}
	if object := extractJSONObject(text); object != "" {
		candidates = append([]string{object}, candidates...)
	}
	for _, candidate := range candidates {
		var out map[string]any
		if json.Unmarshal([]byte(candidate), &out) == nil {
			return out
		}
	}
	return map[string]any{"flatPrompt": strings.TrimSpace(text)}
}

func extractFence(text string) string {
	re := regexp.MustCompile("(?s)```(?:json)?\\s*(.*?)\\s*```")
	match := re.FindStringSubmatch(text)
	if len(match) == 2 {
		return strings.TrimSpace(match[1])
	}
	return ""
}

func extractJSONObject(text string) string {
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start >= 0 && end > start {
		return text[start : end+1]
	}
	return ""
}

func firstString(values map[string]any, key string, fallback string) string {
	if value, ok := values[key].(string); ok && strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	return strings.TrimSpace(fallback)
}

func stringSlice(value any) []string {
	switch typed := value.(type) {
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if text := fmt.Sprint(item); strings.TrimSpace(text) != "" {
				out = append(out, strings.TrimSpace(text))
			}
		}
		return out
	case []string:
		return typed
	case string:
		if strings.TrimSpace(typed) == "" {
			return nil
		}
		return []string{strings.TrimSpace(typed)}
	default:
		return nil
	}
}

func objectMap(value any) map[string]any {
	if typed, ok := value.(map[string]any); ok {
		return typed
	}
	return nil
}

func parseIdeas(text string) []InspirationIdea {
	parsed := parsePromptJSON(text)
	var rawIdeas any
	if value, ok := parsed["ideas"]; ok {
		rawIdeas = value
	} else if value, ok := parsed["items"]; ok {
		rawIdeas = value
	}
	items, ok := rawIdeas.([]any)
	if !ok {
		if title := firstString(parsed, "title", ""); title != "" {
			return []InspirationIdea{{
				Title:   title,
				Summary: firstString(parsed, "summary", firstString(parsed, "flatPrompt", strings.TrimSpace(text))),
				Tags:    stringSlice(parsed["tags"]),
			}}
		}
		return nil
	}
	ideas := make([]InspirationIdea, 0, len(items))
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		idea := InspirationIdea{
			ID:       firstString(m, "id", ""),
			Title:    firstString(m, "title", ""),
			Summary:  firstString(m, "summary", ""),
			Tags:     stringSlice(m["tags"]),
			Category: firstString(m, "category", ""),
			Mood:     firstString(m, "mood", ""),
			Style:    firstString(m, "style", ""),
		}
		if idea.Title == "" && idea.Summary == "" {
			continue
		}
		ideas = append(ideas, idea)
	}
	return ideas
}

func defaultString(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func ParseLimit(value string) int {
	limit, _ := strconv.Atoi(value)
	if limit <= 0 {
		return 50
	}
	return limit
}
