package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/y08lin4/lyra-image-workbench/internal/config"
	"github.com/y08lin4/lyra-image-workbench/internal/gifrender"
	"github.com/y08lin4/lyra-image-workbench/internal/jobs"
	"github.com/y08lin4/lyra-image-workbench/internal/llm"
	"github.com/y08lin4/lyra-image-workbench/internal/output"
	"github.com/y08lin4/lyra-image-workbench/internal/settings"
	"github.com/y08lin4/lyra-image-workbench/internal/spaceconfig"
	"github.com/y08lin4/lyra-image-workbench/internal/uploads"
)

type GIFHandler struct {
	cfg         config.Config
	jobs        *jobs.Manager
	output      *output.Store
	uploads     *uploads.Store
	settings    *settings.FileStore
	spaceConfig *spaceconfig.Store
	llm         *llm.Client
	service     *gifrender.Service
}

type gifPlanRequest struct {
	UploadID   string `json:"uploadId"`
	MotionType string `json:"motionType"`
	Prompt     string `json:"prompt"`
	FrameCount int    `json:"frameCount"`
	FPS        int    `json:"fps"`
	Strength   string `json:"strength"`
}

type gifPlan struct {
	BasePrompt     string         `json:"basePrompt"`
	NegativePrompt string         `json:"negativePrompt"`
	StyleLock      string         `json:"styleLock"`
	FrameCount     int            `json:"frameCount"`
	FPS            int            `json:"fps"`
	Frames         []gifPlanFrame `json:"frames"`
	RenderHints    gifRenderHints `json:"renderHints"`
}

type gifPlanFrame struct {
	Index  int    `json:"index"`
	Action string `json:"action"`
	Prompt string `json:"prompt"`
}

type gifRenderHints struct {
	FPS              int  `json:"fps"`
	Loop             bool `json:"loop"`
	RecommendedWidth int  `json:"recommendedWidth"`
}

type gifRenderRequest struct {
	SourceTaskID string `json:"sourceTaskId"`
	FrameIndexes []int  `json:"frameIndexes"`
	FPS          int    `json:"fps"`
	Loop         bool   `json:"loop"`
	Width        int    `json:"width"`
}

func NewGIFHandler(cfg config.Config, jobManager *jobs.Manager, outputStore *output.Store, uploadStore *uploads.Store, settingsStore *settings.FileStore, spaceConfigStore *spaceconfig.Store, llmClient *llm.Client, service *gifrender.Service) GIFHandler {
	return GIFHandler{cfg: cfg, jobs: jobManager, output: outputStore, uploads: uploadStore, settings: settingsStore, spaceConfig: spaceConfigStore, llm: llmClient, service: service}
}

func (h GIFHandler) Status(w http.ResponseWriter, r *http.Request) {
	available, bin := h.service.Renderer.Available()
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":              true,
		"gifEnabled":      h.cfg.GIFEnabled,
		"ffmpegAvailable": available,
		"ffmpegBin":       bin,
		"limits":          h.service.Renderer.Limits(),
	})
}

func (h GIFHandler) Plan(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var payload gifPlanRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_JSON", "请求体不是有效 JSON")
		return
	}
	spaceToken := r.Header.Get("X-Space-Token")
	if strings.TrimSpace(payload.UploadID) != "" {
		if _, _, err := h.uploads.GetReferenceImage(spaceToken, payload.UploadID); err != nil {
			writeError(w, http.StatusBadRequest, "GIF_UPLOAD_NOT_FOUND", err.Error())
			return
		}
	}
	payload = h.normalizePlanRequest(payload)
	plan := fallbackGIFPlan(payload)
	fallback := true
	warning := ""
	if apiKey, err := h.promptAPIKey(spaceToken, runtimeAPIKeyFromRequest(r)); err == nil && apiKey != "" {
		if generated, err := h.generateGIFPlanWithLLM(r.Context(), payload, apiKey); err == nil {
			plan = normalizeGIFPlan(generated, payload)
			fallback = false
		} else {
			warning = err.Error()
		}
	} else if err != nil {
		warning = err.Error()
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "plan": plan, "fallback": fallback, "warning": warning})
}

func (h GIFHandler) CreateRender(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var payload gifRenderRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_JSON", "请求体不是有效 JSON")
		return
	}
	spaceToken := r.Header.Get("X-Space-Token")
	if strings.TrimSpace(payload.SourceTaskID) == "" {
		writeError(w, http.StatusBadRequest, "GIF_SOURCE_TASK_REQUIRED", "请选择已生成动画帧的任务")
		return
	}
	job, ok, err := h.jobs.Get(spaceToken, payload.SourceTaskID)
	if err != nil {
		writeSpaceError(w, err)
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, "TASK_NOT_FOUND", "任务不存在")
		return
	}
	if len(payload.FrameIndexes) == 0 {
		writeError(w, http.StatusBadRequest, "GIF_FRAME_INDEXES_EMPTY", "请选择要合成的帧")
		return
	}
	limits := h.service.Renderer.Limits()
	fps := payload.FPS
	if fps <= 0 {
		fps = gifrender.DefaultFPS
	}
	if fps > limits.MaxFPS {
		writeError(w, http.StatusBadRequest, "GIF_FPS_TOO_HIGH", fmt.Sprintf("FPS 不能超过 %d", limits.MaxFPS))
		return
	}
	width := payload.Width
	if width <= 0 {
		width = gifrender.DefaultWidth
	}
	if width > limits.MaxSize {
		writeError(w, http.StatusBadRequest, "GIF_WIDTH_TOO_HIGH", fmt.Sprintf("导出宽度不能超过 %d", limits.MaxSize))
		return
	}
	paths, indexes, err := h.resolveFramePaths(job, payload.FrameIndexes, limits.MaxFrames)
	if err != nil {
		writeError(w, http.StatusBadRequest, gifAPIErrorCode(err), err.Error())
		return
	}
	renderID, err := gifrender.NewRenderID()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "GIF_RENDER_ID_FAILED", err.Error())
		return
	}
	artifact, err := h.service.Renderer.RenderGIF(r.Context(), gifrender.RenderRequest{
		JobID:   renderID,
		WorkDir: h.cfg.GIFWorkDir,
		Frames:  paths,
		FPS:     fps,
		Width:   width,
		Loop:    payload.Loop,
		Timeout: time.Duration(h.cfg.GIFRenderTimeoutSec) * time.Second,
	})
	if err != nil {
		h.writeRenderError(w, err)
		return
	}
	defer os.Remove(artifact.Path)
	data, err := os.ReadFile(artifact.Path)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "GIF_ARTIFACT_READ_FAILED", err.Error())
		return
	}
	saved, err := h.output.Save(spaceToken, renderID, 0, data, "image/gif")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "GIF_ARTIFACT_SAVE_FAILED", err.Error())
		return
	}
	now := time.Now()
	render := gifrender.Render{
		ID:             renderID,
		SpaceToken:     spaceToken,
		SourceTaskID:   job.ID,
		Status:         gifrender.RenderStatusSucceeded,
		FPS:            fps,
		FrameIndexes:   indexes,
		Loop:           payload.Loop,
		Width:          width,
		OutputDate:     saved.Date,
		OutputFileName: saved.FileName,
		Bytes:          saved.Bytes,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := h.service.Store.Save(render); err != nil {
		writeError(w, http.StatusInternalServerError, "GIF_RENDER_SAVE_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "render": gifrender.PublicRender(render)})
}

func (h GIFHandler) GetRender(w http.ResponseWriter, r *http.Request) {
	render, ok, err := h.service.Store.Get(r.Header.Get("X-Space-Token"), r.PathValue("id"))
	if err != nil {
		writeSpaceError(w, err)
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, "GIF_RENDER_NOT_FOUND", "GIF 渲染记录不存在")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "render": gifrender.PublicRender(render)})
}

func (h GIFHandler) File(w http.ResponseWriter, r *http.Request) {
	spaceToken := r.Header.Get("X-Space-Token")
	render, ok, err := h.service.Store.Get(spaceToken, r.PathValue("id"))
	if err != nil {
		writeSpaceError(w, err)
		return
	}
	if !ok || render.Status != gifrender.RenderStatusSucceeded || render.OutputDate == "" || render.OutputFileName == "" {
		writeError(w, http.StatusNotFound, "GIF_RENDER_FILE_NOT_FOUND", "GIF 文件不存在")
		return
	}
	path, mime, err := h.output.Resolve(spaceToken, render.OutputDate, render.OutputFileName)
	if err != nil {
		writeError(w, http.StatusBadRequest, "OUTPUT_PATH_INVALID", err.Error())
		return
	}
	w.Header().Set("Content-Type", mime)
	w.Header().Set("Cache-Control", "private, max-age=86400")
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=\"%s.gif\"", render.ID))
	http.ServeFile(w, r, path)
}

func (h GIFHandler) promptAPIKey(spaceToken string, runtimeKey string) (string, error) {
	if key := strings.TrimSpace(runtimeKey); key != "" {
		return key, nil
	}
	cfg, err := h.spaceConfig.Get(spaceToken)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(cfg.APIKey), nil
}

func (h GIFHandler) normalizePlanRequest(req gifPlanRequest) gifPlanRequest {
	req.MotionType = normalizeMotionType(req.MotionType)
	req.Strength = normalizeStrength(req.Strength)
	req.Prompt = strings.TrimSpace(req.Prompt)
	if req.FrameCount <= 0 {
		req.FrameCount = 12
	}
	if req.FrameCount < 2 {
		req.FrameCount = 2
	}
	maxFrames := h.service.Renderer.Limits().MaxFrames
	if req.FrameCount > maxFrames {
		req.FrameCount = maxFrames
	}
	if req.FPS <= 0 {
		req.FPS = 8
	}
	maxFPS := h.service.Renderer.Limits().MaxFPS
	if req.FPS > maxFPS {
		req.FPS = maxFPS
	}
	return req
}

func (h GIFHandler) generateGIFPlanWithLLM(ctx context.Context, req gifPlanRequest, apiKey string) (gifPlan, error) {
	resp, err := h.llm.Complete(ctx, llm.Request{
		BaseURL:     h.settings.Get().NewAPIBaseURL,
		APIKey:      apiKey,
		Model:       config.DefaultPromptModel,
		System:      gifPlanSystemPrompt(),
		User:        gifPlanUserPrompt(req),
		TimeoutSec:  config.DefaultPromptTimeoutSec,
		Temperature: 0.3,
	})
	if err != nil {
		return gifPlan{}, err
	}
	var plan gifPlan
	if err := json.Unmarshal([]byte(extractJSONObject(resp.Text)), &plan); err != nil {
		return gifPlan{}, err
	}
	return plan, nil
}

func (h GIFHandler) resolveFramePaths(job jobs.Job, requested []int, maxFrames int) ([]string, []int, error) {
	if len(requested) < 2 {
		return nil, nil, errors.New("至少需要 2 张成功帧才能合成 GIF")
	}
	if len(requested) > maxFrames {
		return nil, nil, fmt.Errorf("合成帧数不能超过 %d", maxFrames)
	}
	byIndex := make(map[int]jobs.Result, len(job.Results))
	for _, result := range job.Results {
		byIndex[result.Index] = result
	}
	seen := make(map[int]bool, len(requested))
	paths := make([]string, 0, len(requested))
	indexes := make([]int, 0, len(requested))
	for _, index := range requested {
		if index < 0 {
			return nil, nil, fmt.Errorf("帧序号 %d 无效", index)
		}
		if seen[index] {
			return nil, nil, fmt.Errorf("帧序号 %d 重复", index)
		}
		seen[index] = true
		result, ok := byIndex[index]
		if !ok {
			return nil, nil, fmt.Errorf("帧 %d 不存在", index)
		}
		if !result.OK {
			return nil, nil, fmt.Errorf("帧 %d 生成失败，不能合成 GIF", index)
		}
		if result.OutputDate == "" || result.OutputFileName == "" {
			return nil, nil, fmt.Errorf("帧 %d 缺少本地输出记录", index)
		}
		path, _, err := h.output.Resolve(job.SpaceToken, result.OutputDate, result.OutputFileName)
		if err != nil {
			return nil, nil, err
		}
		paths = append(paths, path)
		indexes = append(indexes, index)
	}
	return paths, indexes, nil
}

func (h GIFHandler) writeRenderError(w http.ResponseWriter, err error) {
	var renderErr gifrender.RenderError
	if errors.As(err, &renderErr) {
		status := http.StatusBadRequest
		if renderErr.Code == "GIF_RENDERER_UNAVAILABLE" {
			status = http.StatusServiceUnavailable
		} else if strings.HasPrefix(renderErr.Code, "GIF_RENDER_") || strings.HasPrefix(renderErr.Code, "GIF_OUTPUT_") {
			status = http.StatusInternalServerError
		}
		writeError(w, status, renderErr.Code, renderErr.Message)
		return
	}
	if errors.Is(err, gifrender.ErrRendererUnavailable) {
		writeError(w, http.StatusServiceUnavailable, "GIF_RENDERER_UNAVAILABLE", "FFmpeg 不可用")
		return
	}
	writeError(w, http.StatusInternalServerError, "GIF_RENDER_FAILED", err.Error())
}

func gifAPIErrorCode(err error) string {
	text := err.Error()
	switch {
	case strings.Contains(text, "至少需要"):
		return "GIF_FRAME_COUNT_TOO_LOW"
	case strings.Contains(text, "不能超过"):
		return "GIF_FRAME_COUNT_TOO_HIGH"
	case strings.Contains(text, "重复") || strings.Contains(text, "无效"):
		return "GIF_FRAME_INDEX_INVALID"
	case strings.Contains(text, "失败"):
		return "GIF_FRAME_FAILED"
	case strings.Contains(text, "不存在"):
		return "GIF_FRAME_NOT_FOUND"
	default:
		return "GIF_FRAME_INVALID"
	}
}

func normalizeMotionType(value string) string {
	switch strings.TrimSpace(value) {
	case "blink", "smile", "turn_head", "hair_flow", "custom":
		return strings.TrimSpace(value)
	default:
		return "custom"
	}
}

func normalizeStrength(value string) string {
	switch strings.TrimSpace(value) {
	case "subtle", "medium", "strong":
		return strings.TrimSpace(value)
	default:
		return "subtle"
	}
}

func fallbackGIFPlan(req gifPlanRequest) gifPlan {
	base := "保持同一主体、同一身份、同一构图、同一背景、同一光照、同一画风。"
	negative := "不要改变人物身份，不要改变服装，不要改变背景，不要新增肢体，不要畸形，不要改变镜头角度。"
	style := "保持参考图中的主体身份、脸型、发型、服装、背景、光照和镜头角度完全一致。"
	actions := motionActions(req.MotionType, req.FrameCount)
	frames := make([]gifPlanFrame, 0, req.FrameCount)
	for i := 0; i < req.FrameCount; i++ {
		action := actions[i]
		motion := action
		if req.Prompt != "" {
			motion = req.Prompt + "；" + action
		}
		frames = append(frames, gifPlanFrame{Index: i, Action: action, Prompt: fmt.Sprintf("%s %s 动作强度：%s。只表达小幅变化，主体身份、背景、服装、构图、光照保持一致。", base, motion, strengthLabel(req.Strength))})
	}
	return gifPlan{BasePrompt: base, NegativePrompt: negative, StyleLock: style, FrameCount: req.FrameCount, FPS: req.FPS, Frames: frames, RenderHints: gifRenderHints{FPS: req.FPS, Loop: true, RecommendedWidth: gifrender.DefaultWidth}}
}

func normalizeGIFPlan(plan gifPlan, req gifPlanRequest) gifPlan {
	fallback := fallbackGIFPlan(req)
	if strings.TrimSpace(plan.BasePrompt) == "" {
		plan.BasePrompt = fallback.BasePrompt
	}
	if strings.TrimSpace(plan.NegativePrompt) == "" {
		plan.NegativePrompt = fallback.NegativePrompt
	}
	if strings.TrimSpace(plan.StyleLock) == "" {
		plan.StyleLock = fallback.StyleLock
	}
	plan.FrameCount = req.FrameCount
	plan.FPS = req.FPS
	if len(plan.Frames) != req.FrameCount {
		plan.Frames = fallback.Frames
	} else {
		for i := range plan.Frames {
			plan.Frames[i].Index = i
			if strings.TrimSpace(plan.Frames[i].Prompt) == "" {
				plan.Frames[i].Prompt = fallback.Frames[i].Prompt
			}
			plan.Frames[i].Prompt = ensurePromptConstraints(plan.Frames[i].Prompt)
			if strings.TrimSpace(plan.Frames[i].Action) == "" {
				plan.Frames[i].Action = fallback.Frames[i].Action
			}
		}
	}
	plan.RenderHints.FPS = req.FPS
	plan.RenderHints.Loop = true
	if plan.RenderHints.RecommendedWidth <= 0 {
		plan.RenderHints.RecommendedWidth = gifrender.DefaultWidth
	}
	return plan
}

func ensurePromptConstraints(prompt string) string {
	constraint := "保持同一主体身份、同一背景、同一构图、同一服装和同一光照，只做小幅动作变化。"
	if strings.Contains(prompt, "同一主体") || strings.Contains(prompt, "同一背景") {
		return prompt
	}
	return strings.TrimSpace(prompt) + " " + constraint
}

func motionActions(motionType string, count int) []string {
	if count < 2 {
		count = 2
	}
	patterns := map[string][]string{
		"blink":     {"初始睁眼", "眼睛略微变小", "半闭眼", "闭眼", "半睁眼", "回到自然睁眼"},
		"smile":     {"中性表情", "嘴角轻微上扬", "自然微笑", "微笑减弱", "回到中性表情"},
		"turn_head": {"正面看镜头", "头部轻微向一侧转动", "保持轻微转头", "逐渐回正", "回到正面"},
		"hair_flow": {"头发自然静止", "发梢轻微飘动", "头发柔和摆动", "发丝回落", "回到自然状态"},
		"custom":    {"初始状态", "动作轻微开始", "动作达到小幅峰值", "动作逐渐回落", "回到初始状态"},
	}
	base := patterns[motionType]
	if len(base) == 0 {
		base = patterns["custom"]
	}
	out := make([]string, count)
	for i := 0; i < count; i++ {
		pos := 0
		if count > 1 {
			pos = int(float64(i) / float64(count-1) * float64(len(base)-1))
		}
		out[i] = base[pos]
	}
	out[0] = base[0]
	out[count-1] = base[0] + "，接近首帧以便循环"
	return out
}

func strengthLabel(value string) string {
	switch value {
	case "medium":
		return "中等"
	case "strong":
		return "明显但仍保持克制"
	default:
		return "轻微"
	}
}

func gifPlanSystemPrompt() string {
	return "你是动画帧提示词规划器。只输出严格 JSON，不要输出 Markdown。规划单张参考图动起来的多帧提示词，必须保持主体身份、服装、背景、构图、光照和镜头角度一致。每帧只允许小幅变化，第一帧和最后一帧必须接近以便循环。"
}

func gifPlanUserPrompt(req gifPlanRequest) string {
	return fmt.Sprintf(`请为单张图片动起来生成 GIF 帧规划。
动作类型：%s
用户动作描述：%s
帧数：%d
FPS：%d
动作强度：%s

必须返回 JSON：
{
  "basePrompt": "...",
  "negativePrompt": "...",
  "styleLock": "...",
  "frameCount": %d,
  "fps": %d,
  "frames": [{"index":0,"action":"...","prompt":"..."}],
  "renderHints": {"fps": %d, "loop": true, "recommendedWidth": 512}
}
frames 数量必须正好为 %d，index 从 0 连续到 %d。`, req.MotionType, req.Prompt, req.FrameCount, req.FPS, req.Strength, req.FrameCount, req.FPS, req.FPS, req.FrameCount, req.FrameCount-1)
}

func extractJSONObject(text string) string {
	text = strings.TrimSpace(text)
	if strings.HasPrefix(text, "{") && strings.HasSuffix(text, "}") {
		return text
	}
	text = regexp.MustCompile("(?s)```(?:json)?\\s*(.*?)\\s*```").ReplaceAllString(text, "$1")
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start >= 0 && end > start {
		return text[start : end+1]
	}
	return text
}

func sortedIndexes(indexes []int) []int {
	out := append([]int{}, indexes...)
	sort.Ints(out)
	return out
}
