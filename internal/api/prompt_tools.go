package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/y08lin4/image-Workbench-Localhost-Version/internal/jobs"
	"github.com/y08lin4/image-Workbench-Localhost-Version/internal/prompttools"
)

type PromptToolsHandler struct {
	service *prompttools.Service
}

func NewPromptToolsHandler(service *prompttools.Service) PromptToolsHandler {
	return PromptToolsHandler{service: service}
}

func (h PromptToolsHandler) TextToPrompt(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var payload prompttools.TextRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_JSON", "请求体不是有效 JSON")
		return
	}
	record, err := h.service.TextToPrompt(r.Context(), r.Header.Get("X-Space-Token"), payload)
	if err != nil {
		writePromptToolError(w, "text", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "record": record})
}

func (h PromptToolsHandler) ImageToPrompt(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var payload prompttools.ImageRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_JSON", "请求体不是有效 JSON")
		return
	}
	record, err := h.service.ImageToPrompt(r.Context(), r.Header.Get("X-Space-Token"), payload)
	if err != nil {
		writePromptToolError(w, "image", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "record": record})
}

func decodePromptPayload[T any](w http.ResponseWriter, r *http.Request) (T, bool) {
	defer r.Body.Close()
	var payload T
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_JSON", "请求体不是有效 JSON")
		return payload, false
	}
	return payload, true
}

func writePromptToolError(w http.ResponseWriter, mode string, err error) {
	status := http.StatusBadRequest
	meta := promptToolErrorMeta(mode, err)
	if strings.HasPrefix(meta.Code, "E_UPSTREAM_") || strings.HasPrefix(meta.Code, "E_PROVIDER_") || strings.HasPrefix(meta.Code, "E_OUTPUT_") {
		status = http.StatusBadGateway
	}
	writeErrorMeta(w, status, meta, err.Error())
}

func promptToolErrorMeta(mode string, err error) jobs.Meta {
	raw := strings.TrimSpace(err.Error())
	lower := strings.ToLower(raw)
	if strings.Contains(lower, "请先") && strings.Contains(lower, "key") {
		return jobs.Meta{Code: "P_CODEX_KEY_MISSING", English: "codex_key_missing", Chinese: "请先填写 codex-key"}
	}
	if strings.Contains(raw, "请输入需要扩写") {
		return jobs.Meta{Code: "P_TEXT_INPUT_EMPTY", English: "text_input_empty", Chinese: "请输入需要扩写的文字想法"}
	}
	if strings.Contains(raw, "请输入初始提示词") {
		return jobs.Meta{Code: "P_SESSION_INITIAL_PROMPT_EMPTY", English: "prompt_session_initial_prompt_empty", Chinese: "请输入初始提示词"}
	}
	if strings.Contains(raw, "请输入修改要求") {
		return jobs.Meta{Code: "P_REFINE_MESSAGE_EMPTY", English: "refine_message_empty", Chinese: "请输入修改要求"}
	}
	if strings.Contains(raw, "提示词会话不存在") {
		return jobs.Meta{Code: "P_SESSION_NOT_FOUND", English: "prompt_session_not_found", Chinese: "提示词会话不存在"}
	}
	if strings.Contains(raw, "提示词版本不存在") {
		return jobs.Meta{Code: "P_VERSION_NOT_FOUND", English: "prompt_version_not_found", Chinese: "提示词版本不存在"}
	}
	if strings.Contains(raw, "没有返回灵感") {
		return jobs.Meta{Code: "P_INSPIRATION_EMPTY", English: "inspiration_empty", Chinese: "提示词模型没有返回灵感"}
	}
	if strings.Contains(raw, "请先选择一个灵感") {
		return jobs.Meta{Code: "P_INSPIRATION_IDEA_EMPTY", English: "inspiration_idea_empty", Chinese: "请先选择一个灵感"}
	}
	if strings.Contains(raw, "图片来源无效") || strings.Contains(raw, "请先选择") {
		return jobs.Meta{Code: "P_IMAGE_SOURCE_INVALID", English: "image_source_invalid", Chinese: "图片来源无效"}
	}
	if strings.Contains(raw, "任务不存在") || strings.Contains(raw, "任务图片不存在") || strings.Contains(raw, "参考图不存在") {
		return jobs.Meta{Code: "P_SOURCE_IMAGE_NOT_FOUND", English: "source_image_not_found", Chinese: "来源图片不存在或已删除"}
	}
	if strings.Contains(raw, "提示词模型 URL 为空") {
		return jobs.Meta{Code: "P_MODEL_URL_EMPTY", English: "prompt_model_url_empty", Chinese: "提示词模型 URL 为空"}
	}
	if strings.Contains(raw, "提示词模型为空") {
		return jobs.Meta{Code: "P_MODEL_EMPTY", English: "prompt_model_empty", Chinese: "提示词模型为空"}
	}
	if strings.Contains(raw, "提示词模型没有返回内容") || strings.Contains(raw, "提示词模型返回内容为空") {
		return jobs.Meta{Code: "P_MODEL_EMPTY_RESPONSE", English: "prompt_model_empty_response", Chinese: "提示词模型返回内容为空"}
	}
	upstream := jobs.ErrorMeta(raw)
	if upstream.Code != "" && upstream.Code != "E_UNKNOWN" {
		return upstream
	}
	if mode == "image" {
		return jobs.Meta{Code: "P_IMAGE_TO_PROMPT_FAILED", English: "image_to_prompt_failed", Chinese: "图片还原提示词失败"}
	}
	return jobs.Meta{Code: "P_TEXT_TO_PROMPT_FAILED", English: "text_to_prompt_failed", Chinese: "文字生成图片提示词失败"}
}

func (h PromptToolsHandler) CreateSession(w http.ResponseWriter, r *http.Request) {
	payload, ok := decodePromptPayload[prompttools.CreateSessionRequest](w, r)
	if !ok {
		return
	}
	session, err := h.service.CreateSession(r.Header.Get("X-Space-Token"), payload)
	if err != nil {
		writePromptToolError(w, "session", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "session": session})
}

func (h PromptToolsHandler) Sessions(w http.ResponseWriter, r *http.Request) {
	sessions, err := h.service.ListSessions(r.Header.Get("X-Space-Token"), prompttools.ParseLimit(r.URL.Query().Get("limit")))
	if err != nil {
		writeSpaceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "sessions": sessions})
}

func (h PromptToolsHandler) Session(w http.ResponseWriter, r *http.Request) {
	session, ok, err := h.service.GetSession(r.Header.Get("X-Space-Token"), r.PathValue("id"))
	if err != nil {
		writeSpaceError(w, err)
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, "P_SESSION_NOT_FOUND", "提示词会话不存在")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "session": session})
}

func (h PromptToolsHandler) RefineSession(w http.ResponseWriter, r *http.Request) {
	payload, ok := decodePromptPayload[prompttools.RefineRequest](w, r)
	if !ok {
		return
	}
	session, err := h.service.RefineSession(r.Context(), r.Header.Get("X-Space-Token"), r.PathValue("id"), payload)
	if err != nil {
		writePromptToolError(w, "refine", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "session": session})
}

func (h PromptToolsHandler) DeleteSession(w http.ResponseWriter, r *http.Request) {
	session, ok, err := h.service.DeleteSession(r.Header.Get("X-Space-Token"), r.PathValue("id"))
	if err != nil {
		writeSpaceError(w, err)
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, "P_SESSION_NOT_FOUND", "提示词会话不存在")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "session": session})
}

func (h PromptToolsHandler) InspirationIdeas(w http.ResponseWriter, r *http.Request) {
	payload, ok := decodePromptPayload[prompttools.InspirationIdeasRequest](w, r)
	if !ok {
		return
	}
	ideas, err := h.service.GenerateIdeas(r.Context(), r.Header.Get("X-Space-Token"), payload)
	if err != nil {
		writePromptToolError(w, "inspiration", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "ideas": ideas})
}

func (h PromptToolsHandler) InspirationExpand(w http.ResponseWriter, r *http.Request) {
	payload, ok := decodePromptPayload[prompttools.InspirationExpandRequest](w, r)
	if !ok {
		return
	}
	session, err := h.service.ExpandIdea(r.Context(), r.Header.Get("X-Space-Token"), payload)
	if err != nil {
		writePromptToolError(w, "inspiration", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "session": session})
}

func (h PromptToolsHandler) History(w http.ResponseWriter, r *http.Request) {
	records, err := h.service.List(r.Header.Get("X-Space-Token"), prompttools.ParseLimit(r.URL.Query().Get("limit")))
	if err != nil {
		writeSpaceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "records": records})
}

func (h PromptToolsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	record, ok, err := h.service.Delete(r.Header.Get("X-Space-Token"), r.PathValue("id"))
	if err != nil {
		writeSpaceError(w, err)
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, "PROMPT_RECORD_NOT_FOUND", "提示词记录不存在")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "record": record})
}
