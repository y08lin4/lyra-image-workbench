package agents

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/y08lin4/lyra-image-workbench/internal/config"
	"github.com/y08lin4/lyra-image-workbench/internal/jobs"
	"github.com/y08lin4/lyra-image-workbench/internal/llm"
	"github.com/y08lin4/lyra-image-workbench/internal/settings"
	"github.com/y08lin4/lyra-image-workbench/internal/spaceconfig"
)

type JobCreator func(spaceToken string, req jobs.CreateRequest) (jobs.Job, error)

type Service struct {
	store       *Store
	settings    *settings.FileStore
	spaceConfig *spaceconfig.Store
	llm         *llm.Client
	createJob   JobCreator
}

type planningResponse struct {
	Action      Action   `json:"action"`
	Question    string   `json:"question"`
	Assumptions []string `json:"assumptions"`
	Plan        *Plan    `json:"plan"`
}

func NewService(store *Store, settingsStore *settings.FileStore, spaceConfigStore *spaceconfig.Store, llmClient *llm.Client, creators ...JobCreator) *Service {
	if llmClient == nil {
		llmClient = llm.NewClient()
	}
	var creator JobCreator
	if len(creators) > 0 {
		creator = creators[0]
	}
	return &Service{
		store:       store,
		settings:    settingsStore,
		spaceConfig: spaceConfigStore,
		llm:         llmClient,
		createJob:   creator,
	}
}

func (s *Service) SetJobCreator(createJob JobCreator) {
	s.createJob = createJob
}

func (s *Service) CreateSession(spaceToken string, req CreateSessionRequest) (Session, error) {
	if err := s.ensureReady(); err != nil {
		return Session{}, err
	}
	if err := s.ensureSpace(spaceToken); err != nil {
		return Session{}, err
	}
	now := time.Now()
	id, err := newAgentID("asn")
	if err != nil {
		return Session{}, err
	}
	session := Session{
		ID:        id,
		Title:     defaultAgentTitle(req.Title, "Agent 创作会话"),
		Status:    SessionDraft,
		Rounds:    []Round{},
		CreatedAt: now,
		UpdatedAt: now,
	}
	return session, s.store.Save(spaceToken, session)
}

func (s *Service) List(spaceToken string, limit int) ([]Session, error) {
	if err := s.ensureReady(); err != nil {
		return nil, err
	}
	return s.store.List(spaceToken, limit)
}

func (s *Service) Get(spaceToken string, id string) (Session, bool, error) {
	if err := s.ensureReady(); err != nil {
		return Session{}, false, err
	}
	return s.store.Get(spaceToken, strings.TrimSpace(id))
}

func (s *Service) Delete(spaceToken string, id string) (Session, bool, error) {
	if err := s.ensureReady(); err != nil {
		return Session{}, false, err
	}
	return s.store.Delete(spaceToken, strings.TrimSpace(id))
}

func (s *Service) SubmitMessage(ctx context.Context, spaceToken string, sessionID string, req MessageRequest) (Session, error) {
	if err := s.ensureReady(); err != nil {
		return Session{}, err
	}
	content := strings.TrimSpace(req.Content)
	if content == "" {
		return Session{}, errors.New("请输入创作需求")
	}
	session, ok, err := s.store.Get(spaceToken, strings.TrimSpace(sessionID))
	if err != nil {
		return Session{}, err
	}
	if !ok {
		return Session{}, errors.New("Agent 会话不存在")
	}
	req.Content = content
	req.ReferenceIDs = cleanStrings(req.ReferenceIDs)
	req.Provider = normalizeProvider(req.Provider)
	req.Model = normalizeModel(req.Provider, req.Model)
	req.Ratio = normalizeAgentRatio(req.Ratio, "auto")

	refs, err := selectedReferences(session, req.ReferenceIDs)
	if err != nil {
		return Session{}, err
	}

	now := time.Now()
	userMessageID, err := newAgentID("msg")
	if err != nil {
		return Session{}, err
	}
	roundID, err := newAgentID("rnd")
	if err != nil {
		return Session{}, err
	}
	round := Round{
		ID:           roundID,
		Index:        len(session.Rounds) + 1,
		UserMessage:  Message{ID: userMessageID, Role: "user", Content: content, CreatedAt: now},
		Action:       ActionProposePlan,
		Status:       RoundPlanning,
		Blocks:       []Block{},
		ReferenceIDs: req.ReferenceIDs,
		CreatedAt:    now,
	}

	resp, raw, fallbackReason := s.planWithLLM(ctx, spaceToken, session, req, refs)
	round.Raw = raw
	round.Assumptions = cleanStrings(resp.Assumptions)

	if resp.Action == ActionAskQuestion && !req.SkipQuestions && strings.TrimSpace(resp.Question) != "" {
		round.Action = ActionAskQuestion
		round.Question = strings.TrimSpace(resp.Question)
		round.Status = RoundAsking
		round.Blocks = []Block{{Type: "question", Content: round.Question}}
	} else {
		plan := resp.Plan
		if plan == nil {
			plan = fallbackPlan(session, req, refs, fallbackReason)
		}
		normalized := normalizePlan(*plan, req, refs, content)
		if fallbackReason != "" {
			normalized.Notes = appendUnique(normalized.Notes, fallbackReason)
		}
		round.Action = ActionProposePlan
		round.Plan = &normalized
		round.Status = RoundAwaitingConfirmation
		round.Blocks = []Block{{Type: "plan", Plan: &normalized}}
		session.Status = SessionAwaitingConfirmation
		if shouldReplaceAgentTitle(session.Title) && strings.TrimSpace(normalized.Title) != "" {
			session.Title = defaultAgentTitle(normalized.Title, session.Title)
		}
	}
	finishedAt := time.Now()
	round.FinishedAt = &finishedAt
	session.Rounds = append(session.Rounds, round)
	session.UpdatedAt = finishedAt
	if session.Status == "" {
		session.Status = SessionDraft
	}
	return session, s.store.Save(spaceToken, session)
}

func (s *Service) ConfirmRound(spaceToken string, sessionID string, roundID string, req ConfirmRequest) (jobs.CreateRequest, error) {
	if err := s.ensureReady(); err != nil {
		return jobs.CreateRequest{}, err
	}
	session, ok, err := s.store.Get(spaceToken, strings.TrimSpace(sessionID))
	if err != nil {
		return jobs.CreateRequest{}, err
	}
	if !ok {
		return jobs.CreateRequest{}, errors.New("Agent 会话不存在")
	}
	roundIndex := findRound(session, roundID)
	if roundIndex < 0 {
		return jobs.CreateRequest{}, errors.New("Agent 轮次不存在")
	}
	round := session.Rounds[roundIndex]
	if round.Plan == nil {
		return jobs.CreateRequest{}, errors.New("当前轮次还没有可确认的生成计划")
	}
	return s.createRequestForRound(spaceToken, session.ID, round.ID, *round.Plan, req), nil
}

func (s *Service) ConfirmRoundAndCreate(spaceToken string, sessionID string, roundID string, req ConfirmRequest) (Session, jobs.Job, error) {
	if s.createJob == nil {
		return Session{}, jobs.Job{}, errors.New("Agent 任务创建函数未配置")
	}
	createReq, err := s.ConfirmRound(spaceToken, sessionID, roundID, req)
	if err != nil {
		return Session{}, jobs.Job{}, err
	}
	job, err := s.createJob(spaceToken, createReq)
	if err != nil {
		_ = s.markRoundFailed(spaceToken, sessionID, roundID, err)
		return Session{}, jobs.Job{}, err
	}
	if err := s.attachJob(spaceToken, sessionID, roundID, job); err != nil {
		return Session{}, jobs.Job{}, err
	}
	session, _, getErr := s.store.Get(spaceToken, sessionID)
	if getErr != nil {
		return Session{}, jobs.Job{}, getErr
	}
	return session, job, nil
}

func (s *Service) planWithLLM(ctx context.Context, spaceToken string, session Session, req MessageRequest, refs []Reference) (planningResponse, string, string) {
	apiKey, err := s.apiKey(spaceToken, req.RuntimeAPIKey)
	if err != nil {
		return planningResponse{Action: ActionProposePlan, Plan: fallbackPlan(session, req, refs, "未配置提示词模型 API Key，已使用本地兜底计划。")}, "", "未配置提示词模型 API Key，已使用本地兜底计划。"
	}
	resp, err := s.llm.Complete(ctx, llm.Request{
		BaseURL:     s.llmBaseURL(),
		APIKey:      apiKey,
		Model:       config.DefaultPromptModel,
		System:      planningSystemPrompt(),
		User:        planningUserPrompt(session, req, refs),
		TimeoutSec:  config.DefaultPromptTimeoutSec,
		Temperature: 0.35,
	})
	if err != nil {
		reason := "提示词模型暂不可用，已使用本地兜底计划。"
		return planningResponse{Action: ActionProposePlan, Plan: fallbackPlan(session, req, refs, reason)}, err.Error(), reason
	}
	parsed, err := parsePlanningResponse(resp.Text)
	if err != nil {
		reason := "提示词模型返回不是有效 JSON，已使用本地兜底计划。"
		return planningResponse{Action: ActionProposePlan, Plan: fallbackPlan(session, req, refs, reason)}, resp.Text, reason
	}
	if parsed.Action != ActionAskQuestion && parsed.Action != ActionProposePlan {
		parsed.Action = ActionProposePlan
	}
	return parsed, resp.Text, ""
}

func (s *Service) createRequestForRound(spaceToken string, sessionID string, roundID string, plan Plan, req ConfirmRequest) jobs.CreateRequest {
	params := normalizeParameters(plan.Parameters)
	provider := normalizeProvider(firstNonEmpty(req.Provider, params.Provider))
	model := normalizeModel(provider, firstNonEmpty(req.Model, params.Model))
	uploadIDs := cleanStrings(req.UploadIDs)
	if len(uploadIDs) == 0 {
		uploadIDs = uploadIDsFromPlan(plan)
	}
	mode := normalizeAgentMode(plan.Mode, len(uploadIDs) > 0)
	count := normalizeCount(firstNonZero(req.Count, params.Count))
	concurrency := normalizeConcurrency(firstNonZero(req.Concurrency, params.Concurrency), count)
	createReq := jobs.CreateRequest{
		RuntimeSecrets: req.RuntimeSecrets,
		Provider:       provider,
		Model:          model,
		Mode:           mode,
		Source:         jobs.JobSourceAgent,
		Prompt:         strings.TrimSpace(plan.GenerationPrompt),
		Ratio:          normalizeAgentRatio(firstNonEmpty(req.Ratio, params.Ratio), "auto"),
		Resolution:     normalizeResolution(firstNonEmpty(req.Resolution, params.Resolution)),
		Quality:        normalizeQuality(firstNonEmpty(req.Quality, params.Quality)),
		OutputFormat:   normalizeOutputFormat(firstNonEmpty(req.OutputFormat, params.OutputFormat)),
		Count:          count,
		Concurrency:    concurrency,
		UploadIDs:      uploadIDs,
	}
	if createReq.Prompt == "" {
		createReq.Prompt = strings.TrimSpace(plan.SceneBrief)
	}
	createReq.BeforeEnqueue = func(job jobs.Job) error {
		return s.attachJob(spaceToken, sessionID, roundID, job)
	}
	return createReq
}

func (s *Service) attachJob(spaceToken string, sessionID string, roundID string, job jobs.Job) error {
	session, ok, err := s.store.Get(spaceToken, sessionID)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("Agent 会话不存在")
	}
	roundIndex := findRound(session, roundID)
	if roundIndex < 0 {
		return errors.New("Agent 轮次不存在")
	}
	now := time.Now()
	session.TaskIDs = appendUnique(session.TaskIDs, job.ID)
	session.Status = SessionGenerating
	session.UpdatedAt = now
	session.Rounds[roundIndex].TaskIDs = appendUnique(session.Rounds[roundIndex].TaskIDs, job.ID)
	session.Rounds[roundIndex].Status = RoundGenerating
	if !hasTaskBlock(session.Rounds[roundIndex].Blocks, job.ID) {
		session.Rounds[roundIndex].Blocks = append(session.Rounds[roundIndex].Blocks, Block{Type: "task", TaskID: job.ID})
	}
	return s.store.Save(spaceToken, session)
}

func (s *Service) markRoundFailed(spaceToken string, sessionID string, roundID string, cause error) error {
	session, ok, err := s.store.Get(spaceToken, sessionID)
	if err != nil || !ok {
		return err
	}
	roundIndex := findRound(session, roundID)
	if roundIndex < 0 {
		return nil
	}
	now := time.Now()
	session.Status = SessionFailed
	session.UpdatedAt = now
	session.Rounds[roundIndex].Status = RoundFailed
	session.Rounds[roundIndex].Error = cause.Error()
	session.Rounds[roundIndex].FinishedAt = &now
	return s.store.Save(spaceToken, session)
}

func (s *Service) apiKey(spaceToken string, runtimeAPIKey string) (string, error) {
	apiKey := strings.TrimSpace(runtimeAPIKey)
	if apiKey != "" {
		return apiKey, nil
	}
	if s.spaceConfig != nil {
		cfg, err := s.spaceConfig.Get(spaceToken)
		if err != nil {
			return "", err
		}
		if cfg.CloudAPIKeyEnabled {
			apiKey = strings.TrimSpace(cfg.APIKey)
		}
	}
	if apiKey == "" && s.settings != nil {
		apiKey = strings.TrimSpace(s.settings.Get().SystemAPIKey)
	}
	if apiKey == "" {
		return "", errors.New("codex-key is not configured")
	}
	return apiKey, nil
}

func (s *Service) llmBaseURL() string {
	if s.settings == nil {
		return config.DefaultNewAPIBaseURL
	}
	return s.settings.Get().NewAPIBaseURL
}

func (s *Service) ensureReady() error {
	if s == nil || s.store == nil {
		return errors.New("Agent store 未配置")
	}
	return nil
}

func (s *Service) ensureSpace(spaceToken string) error {
	if s.spaceConfig == nil {
		return nil
	}
	_, err := s.spaceConfig.Get(spaceToken)
	return err
}

func parsePlanningResponse(text string) (planningResponse, error) {
	candidates := planningJSONCandidates(text)
	var lastErr error
	for _, candidate := range candidates {
		var resp planningResponse
		if err := json.Unmarshal([]byte(candidate), &resp); err == nil {
			if resp.Plan == nil {
				if plan, ok := parseDirectPlan(candidate); ok {
					resp.Action = ActionProposePlan
					resp.Plan = &plan
				}
			}
			if resp.Action == "" && resp.Plan != nil {
				resp.Action = ActionProposePlan
			}
			if resp.Action != "" {
				return resp, nil
			}
			lastErr = errors.New("missing action")
			continue
		} else {
			lastErr = err
		}
		if plan, ok := parseDirectPlan(candidate); ok {
			return planningResponse{Action: ActionProposePlan, Plan: &plan}, nil
		}
	}
	if lastErr == nil {
		lastErr = errors.New("empty response")
	}
	return planningResponse{}, lastErr
}

func planningJSONCandidates(text string) []string {
	text = strings.TrimSpace(text)
	candidates := make([]string, 0, 3)
	if fenced := extractJSONFence(text); fenced != "" {
		candidates = append(candidates, fenced)
	}
	if object := extractJSONObject(text); object != "" && object != text {
		candidates = append(candidates, object)
	}
	if text != "" {
		candidates = append(candidates, text)
	}
	return candidates
}

func extractJSONFence(text string) string {
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
		return strings.TrimSpace(text[start : end+1])
	}
	return ""
}

func parseDirectPlan(text string) (Plan, bool) {
	var plan Plan
	if err := json.Unmarshal([]byte(text), &plan); err != nil {
		return Plan{}, false
	}
	return plan, strings.TrimSpace(plan.GenerationPrompt) != "" || strings.TrimSpace(plan.SceneBrief) != ""
}

func selectedReferences(session Session, ids []string) ([]Reference, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	byID := make(map[string]Reference, len(session.References))
	for _, ref := range session.References {
		if ref.Removed {
			continue
		}
		byID[ref.ID] = ref
	}
	refs := make([]Reference, 0, len(ids))
	for _, id := range ids {
		ref, ok := byID[id]
		if !ok {
			return nil, fmt.Errorf("参考图不存在：%s", id)
		}
		refs = append(refs, ref)
	}
	return refs, nil
}

func normalizePlan(plan Plan, req MessageRequest, refs []Reference, userContent string) Plan {
	plan.Title = defaultAgentTitle(plan.Title, userContent)
	plan.SceneBrief = defaultString(plan.SceneBrief, "基于用户描述生成一张可直接执行的图像方案。")
	plan.GenerationPrompt = defaultString(plan.GenerationPrompt, userContent)
	plan.NegativePrompt = strings.TrimSpace(plan.NegativePrompt)
	plan.Mode = normalizeAgentMode(plan.Mode, len(refs) > 0)
	plan.Parameters = normalizeParameters(plan.Parameters)
	plan.Parameters.Provider = normalizeProvider(firstNonEmpty(req.Provider, plan.Parameters.Provider))
	plan.Parameters.Model = normalizeModel(plan.Parameters.Provider, firstNonEmpty(req.Model, plan.Parameters.Model))
	plan.Parameters.Ratio = normalizeAgentRatio(firstNonEmpty(req.Ratio, plan.Parameters.Ratio), "auto")
	if plan.Mode == jobs.ModeImageToImage {
		plan.ReferenceUsages = normalizeReferenceUsages(plan.ReferenceUsages, refs)
	} else {
		plan.ReferenceUsages = normalizeReferenceUsages(plan.ReferenceUsages, refs)
	}
	plan.MustKeep = cleanStrings(plan.MustKeep)
	plan.Avoid = cleanStrings(plan.Avoid)
	plan.Notes = cleanStrings(plan.Notes)
	plan.VisualPlan = normalizeVisualPlan(plan.VisualPlan, userContent)
	return plan
}

func normalizeVisualPlan(plan VisualPlan, fallbackSubject string) VisualPlan {
	plan.Subject = defaultString(plan.Subject, fallbackSubject)
	plan.Environment = defaultString(plan.Environment, "干净、有层次的背景环境")
	plan.Camera = defaultString(plan.Camera, "中景视角，主体清晰，透视自然")
	plan.Composition = defaultString(plan.Composition, "主体居中或略偏黄金分割位置，保留适度留白")
	plan.Lighting = defaultString(plan.Lighting, "柔和主光配合轻微轮廓光，明暗层次清楚")
	plan.Colors = defaultString(plan.Colors, "协调的主辅色，整体色调统一")
	plan.Materials = defaultString(plan.Materials, "材质细节真实，边缘和纹理清晰")
	plan.Mood = defaultString(plan.Mood, "清晰、精致、有完成度")
	plan.Style = defaultString(plan.Style, "高质量商业摄影或精致数字艺术")
	return plan
}

func normalizeParameters(params Parameters) Parameters {
	provider := normalizeProvider(params.Provider)
	count := normalizeCount(params.Count)
	return Parameters{
		Provider:     provider,
		Model:        normalizeModel(provider, params.Model),
		Ratio:        normalizeAgentRatio(params.Ratio, "auto"),
		Resolution:   normalizeResolution(params.Resolution),
		Quality:      normalizeQuality(params.Quality),
		OutputFormat: normalizeOutputFormat(params.OutputFormat),
		Count:        count,
		Concurrency:  normalizeConcurrency(params.Concurrency, count),
	}
}

func normalizeReferenceUsages(usages []ReferenceUsage, refs []Reference) []ReferenceUsage {
	byID := make(map[string]Reference, len(refs))
	for _, ref := range refs {
		byID[ref.ID] = ref
	}
	out := make([]ReferenceUsage, 0, len(refs))
	seen := map[string]bool{}
	for _, usage := range usages {
		usage.ReferenceID = strings.TrimSpace(usage.ReferenceID)
		if usage.ReferenceID == "" || seen[usage.ReferenceID] {
			continue
		}
		ref, ok := byID[usage.ReferenceID]
		if !ok {
			continue
		}
		usage.UploadID = firstNonEmpty(usage.UploadID, ref.UploadID)
		usage.Usage = normalizeReferenceUsage(usage.Usage)
		usage.MustKeep = cleanStrings(usage.MustKeep)
		usage.CanChange = cleanStrings(usage.CanChange)
		out = append(out, usage)
		seen[usage.ReferenceID] = true
	}
	for _, ref := range refs {
		if seen[ref.ID] {
			continue
		}
		out = append(out, ReferenceUsage{
			ReferenceID: ref.ID,
			UploadID:    ref.UploadID,
			Usage:       "loose_reference",
		})
	}
	return out
}

func fallbackPlan(session Session, req MessageRequest, refs []Reference, note string) *Plan {
	content := strings.TrimSpace(req.Content)
	if content == "" {
		content = "生成一张完整、有视觉重点的图像"
	}
	ratio := normalizeAgentRatio(req.Ratio, "auto")
	mode := normalizeAgentMode("", len(refs) > 0)
	provider := normalizeProvider(req.Provider)
	model := normalizeModel(provider, req.Model)
	notes := []string{"本地兜底计划，可在确认前继续修改。"}
	if strings.TrimSpace(note) != "" {
		notes = append(notes, strings.TrimSpace(note))
	}
	return &Plan{
		Title:      defaultAgentTitle(session.Title, content),
		Mode:       mode,
		SceneBrief: "根据用户输入整理出的基础创作方案，适合先生成一版用于确认方向。",
		VisualPlan: VisualPlan{
			Subject:     content,
			Environment: "简洁、有层次的场景背景，主体与环境关系清楚",
			Camera:      "中景或近中景视角，主体清晰，透视自然",
			Composition: "主体占据主要视觉区域，保留适度留白，画面重心稳定",
			Lighting:    "柔和主光，适度轮廓光，细节不过曝也不死黑",
			Colors:      "色彩协调，主色明确，整体观感干净",
			Materials:   "材质、纹理和边缘细节清晰",
			Mood:        "精致、清晰、有完成度",
			Style:       "高质量图像生成风格，偏商业摄影或数字艺术",
		},
		GenerationPrompt: fmt.Sprintf("%s。画面比例为 %s。主体清晰，构图稳定，光影自然，细节完整，整体具有高质量成片感。", content, ratio),
		NegativePrompt:   "低清晰度、模糊、畸形、重复肢体、文字乱码、水印、过曝、欠曝、杂乱背景",
		Parameters: Parameters{
			Provider:     provider,
			Model:        model,
			Ratio:        ratio,
			Resolution:   "standard",
			Quality:      "auto",
			OutputFormat: "png",
			Count:        1,
			Concurrency:  1,
		},
		ReferenceUsages: normalizeReferenceUsages(nil, refs),
		MustKeep:        cleanStrings([]string{content}),
		Avoid:           []string{"低清晰度", "画面杂乱", "主体不明确"},
		Notes:           notes,
	}
}

func uploadIDsFromPlan(plan Plan) []string {
	out := make([]string, 0, len(plan.ReferenceUsages))
	seen := map[string]bool{}
	for _, usage := range plan.ReferenceUsages {
		id := strings.TrimSpace(usage.UploadID)
		if id == "" || seen[id] || normalizeReferenceUsage(usage.Usage) == "ignore" {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}

func findRound(session Session, roundID string) int {
	roundID = strings.TrimSpace(roundID)
	if roundID == "" && len(session.Rounds) > 0 {
		return len(session.Rounds) - 1
	}
	for i := range session.Rounds {
		if session.Rounds[i].ID == roundID {
			return i
		}
	}
	return -1
}

func newAgentID(prefix string) (string, error) {
	var bytes [8]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", err
	}
	return prefix + "_" + time.Now().Format("20060102150405") + "_" + hex.EncodeToString(bytes[:]), nil
}

func normalizeAgentMode(value jobs.Mode, hasReferences bool) jobs.Mode {
	switch value {
	case jobs.ModeImageToImage:
		if hasReferences {
			return jobs.ModeImageToImage
		}
		return jobs.ModeTextToImage
	case jobs.ModeTextToImage:
		return jobs.ModeTextToImage
	default:
		if hasReferences {
			return jobs.ModeImageToImage
		}
		return jobs.ModeTextToImage
	}
}

func normalizeAgentRatio(value string, fallback string) string {
	value = strings.ReplaceAll(strings.TrimSpace(value), "：", ":")
	switch value {
	case "auto", "1:1", "2:3", "3:2", "3:4", "4:3", "9:16", "16:9":
		return value
	}
	fallback = strings.TrimSpace(fallback)
	switch fallback {
	case "auto", "1:1", "2:3", "3:2", "3:4", "4:3", "9:16", "16:9":
		return fallback
	default:
		return "auto"
	}
}

func normalizeCount(value int) int {
	if value <= 0 {
		return 1
	}
	if value > 3 {
		return 3
	}
	return value
}

func normalizeConcurrency(value int, count int) int {
	if value <= 0 {
		value = 1
	}
	if count <= 0 {
		count = 1
	}
	if value > count {
		return count
	}
	return value
}

func normalizeProvider(_ string) string {
	return config.DefaultProvider
}

func normalizeModel(_ string, _ string) string {
	return config.DefaultModel
}

func normalizeResolution(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "auto", "standard", "2k", "4k":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "standard"
	}
}

func normalizeQuality(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "auto", "low", "medium", "high":
		return strings.ToLower(strings.TrimSpace(value))
	case "standard":
		return "auto"
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
	case "webp":
		return "webp"
	default:
		return "png"
	}
}

func normalizeReferenceUsage(value string) string {
	switch strings.TrimSpace(value) {
	case "preserve_subject", "preserve_style", "preserve_composition", "style_reference", "content_reference", "loose_reference", "ignore":
		return strings.TrimSpace(value)
	default:
		return "loose_reference"
	}
}

func cleanStrings(values []string) []string {
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

func hasTaskBlock(blocks []Block, taskID string) bool {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return false
	}
	for _, block := range blocks {
		if block.Type == "task" && block.TaskID == taskID {
			return true
		}
	}
	return false
}
func appendUnique(values []string, next string) []string {
	next = strings.TrimSpace(next)
	if next == "" {
		return values
	}
	for _, value := range values {
		if value == next {
			return values
		}
	}
	return append(values, next)
}

func firstNonZero(values ...int) int {
	for _, value := range values {
		if value != 0 {
			return value
		}
	}
	return 0
}

func defaultString(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func defaultAgentTitle(value string, fallback string) string {
	title := strings.TrimSpace(value)
	if title == "" || title == "Agent 创作会话" {
		title = strings.TrimSpace(fallback)
	}
	if title == "" {
		title = "Agent 创作会话"
	}
	title = strings.Join(strings.Fields(title), " ")
	runes := []rune(title)
	if len(runes) > 32 {
		return string(runes[:32]) + "..."
	}
	return title
}

func shouldReplaceAgentTitle(value string) bool {
	value = strings.TrimSpace(value)
	return value == "" || value == "Agent 创作会话"
}
