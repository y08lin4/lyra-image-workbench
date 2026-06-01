package prompttools

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
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
	record, err := s.newRecord(spaceToken, Record{
		Mode:           ModeTextToPrompt,
		Input:          input,
		Style:          firstString(parsed, "style", strings.TrimSpace(req.Style)),
		Ratio:          firstString(parsed, "ratio", strings.TrimSpace(req.Ratio)),
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
	resp, err := s.llm.Complete(ctx, llm.Request{
		BaseURL:    s.settings.Get().NewAPIBaseURL,
		APIKey:     apiKey,
		Model:      config.DefaultPromptModel,
		System:     imageSystemPrompt(),
		User:       imageUserPrompt(strings.TrimSpace(req.Target)),
		Image:      &llm.ImagePart{Mime: mime, Data: data},
		TimeoutSec: config.DefaultPromptTimeoutSec,
	})
	if err != nil {
		return Record{}, err
	}
	parsed := parsePromptJSON(resp.Text)
	record, err := s.newRecord(spaceToken, Record{
		Mode:            ModeImageToPrompt,
		Language:        defaultString(req.Language, "zh"),
		Target:          defaultString(req.Target, "image-2"),
		Source:          req.Source,
		SourceImageURL:  sourceURL,
		FlatPrompt:      firstString(parsed, "flatPrompt", resp.Text),
		NegativePrompt:  firstString(parsed, "negativePrompt", ""),
		MustKeep:        stringSlice(parsed["mustKeep"]),
		Avoid:           stringSlice(parsed["avoid"]),
		JSONDescription: objectMap(parsed["jsonDescription"]),
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
	record, err := s.newRecord(spaceToken, Record{
		Mode:           ModeTextToPrompt,
		Input:          strings.TrimSpace(req.Idea.Title + " " + req.Idea.Summary),
		Style:          firstString(parsed, "style", strings.TrimSpace(req.Idea.Style)),
		Ratio:          firstString(parsed, "ratio", strings.TrimSpace(req.Ratio)),
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
	if apiKey == "" {
		return "", errors.New("codex-key is not configured; save it locally or upload it to cloud after enabling account protection")
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
		ActiveVersionID: versionID,
		CreatedAt:       now,
		UpdatedAt:       now,
		Versions: []PromptVersion{{
			ID:             versionID,
			Index:          1,
			Prompt:         strings.TrimSpace(record.FlatPrompt),
			NegativePrompt: strings.TrimSpace(record.NegativePrompt),
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
