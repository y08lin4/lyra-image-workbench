package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/y08lin4/lyra-image-workbench/internal/agents"
	"github.com/y08lin4/lyra-image-workbench/internal/jobs"
	"github.com/y08lin4/lyra-image-workbench/internal/spaces"
)

type AgentHandler struct {
	service *agents.Service
	task    TaskHandler
}

func NewAgentHandler(service *agents.Service, taskHandler TaskHandler) AgentHandler {
	return AgentHandler{service: service, task: taskHandler}
}

func (h AgentHandler) ListSessions(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	sessions, err := h.service.List(r.Header.Get("X-Space-Token"), limit)
	if err != nil {
		writeAgentError(w, "AGENT_SESSIONS_FAILED", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "sessions": sessions})
}

func (h AgentHandler) CreateSession(w http.ResponseWriter, r *http.Request) {
	payload, ok := decodeAgentPayload[agents.CreateSessionRequest](w, r)
	if !ok {
		return
	}
	session, err := h.service.CreateSession(r.Header.Get("X-Space-Token"), payload)
	if err != nil {
		writeAgentError(w, "AGENT_SESSION_CREATE_FAILED", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "session": session})
}

func (h AgentHandler) GetSession(w http.ResponseWriter, r *http.Request) {
	session, ok, err := h.service.Get(r.Header.Get("X-Space-Token"), agentSessionID(r))
	if err != nil {
		writeAgentError(w, "AGENT_SESSION_GET_FAILED", err)
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, "AGENT_SESSION_NOT_FOUND", "Agent 会话不存在")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "session": session})
}

func (h AgentHandler) DeleteSession(w http.ResponseWriter, r *http.Request) {
	session, ok, err := h.service.Delete(r.Header.Get("X-Space-Token"), agentSessionID(r))
	if err != nil {
		writeAgentError(w, "AGENT_SESSION_DELETE_FAILED", err)
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, "AGENT_SESSION_NOT_FOUND", "Agent 会话不存在")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "session": session})
}

func (h AgentHandler) SubmitMessage(w http.ResponseWriter, r *http.Request) {
	payload, ok := decodeAgentPayload[agents.MessageRequest](w, r)
	if !ok {
		return
	}
	payload.RuntimeAPIKey = runtimeAPIKeyFromRequest(r)
	session, err := h.service.SubmitMessage(r.Context(), r.Header.Get("X-Space-Token"), agentSessionID(r), payload)
	if err != nil {
		writeAgentError(w, "AGENT_MESSAGE_FAILED", err)
		return
	}
	round, hasRound := latestAgentRound(session)
	response := map[string]any{"ok": true, "session": session}
	if hasRound {
		response["round"] = round
		response["blocks"] = round.Blocks
	}
	writeJSON(w, http.StatusOK, response)
}

func (h AgentHandler) ConfirmRound(w http.ResponseWriter, r *http.Request) {
	payload, ok := decodeAgentPayload[agents.ConfirmRequest](w, r)
	if !ok {
		return
	}
	payload.RuntimeSecrets = runtimeSecretsFromRequest(r)
	spaceToken := r.Header.Get("X-Space-Token")
	sessionID := agentSessionID(r)
	roundID := r.PathValue("roundId")
	createReq, err := h.service.ConfirmRound(spaceToken, sessionID, roundID, payload)
	if err != nil {
		writeAgentError(w, "AGENT_CONFIRM_FAILED", err)
		return
	}
	job, err := h.createAgentJob(r, spaceToken, createReq)
	if err != nil {
		if isUserCreditError(err) {
			writeUserCreditError(w, err)
			return
		}
		writeAgentError(w, "AGENT_TASK_CREATE_FAILED", err)
		return
	}

	publicJob := jobs.PublicJob(job)
	response := map[string]any{
		"ok":              true,
		"taskId":          publicJob.ID,
		"taskIds":         []string{publicJob.ID},
		"task":            publicJob,
		"tasks":           []jobs.Job{publicJob},
		"job":             publicJob,
		"consumedCredits": publicJob.ConsumedCredits,
	}
	if session, ok, err := h.service.Get(spaceToken, sessionID); err == nil && ok {
		response["session"] = session
		if round, ok := agentRoundByID(session, roundID); ok {
			response["round"] = round
		}
	}
	writeJSON(w, http.StatusOK, response)
}

func (h AgentHandler) createAgentJob(r *http.Request, spaceToken string, req jobs.CreateRequest) (jobs.Job, error) {
	if h.task.manager == nil {
		return jobs.Job{}, errors.New("Agent 任务服务未初始化")
	}
	req.RuntimeSecrets = runtimeSecretsFromRequest(r)
	req.Source = jobs.JobSourceAgent
	req.WaiveCredits = h.task.waiveTaskCredits(spaceToken, req)
	username := r.Header.Get("X-User-Name")
	if err := h.task.ensureTaskCredits(username, billableTaskCredits(req)); err != nil {
		return jobs.Job{}, err
	}
	beforeEnqueue := req.BeforeEnqueue
	req.BeforeEnqueue = func(job jobs.Job) error {
		if beforeEnqueue != nil {
			if err := beforeEnqueue(job); err != nil {
				return err
			}
		}
		return h.task.chargeTask(username, job)
	}
	return h.task.manager.Create(spaceToken, req)
}

func decodeAgentPayload[T any](w http.ResponseWriter, r *http.Request) (T, bool) {
	defer r.Body.Close()
	var payload T
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_JSON", "请求体不是有效 JSON")
		return payload, false
	}
	return payload, true
}

func writeAgentError(w http.ResponseWriter, fallbackCode string, err error) {
	var validationErr spaces.ValidationError
	if spaces.AsValidationError(err, &validationErr) {
		writeSpaceError(w, err)
		return
	}
	message := err.Error()
	status := http.StatusBadRequest
	code := fallbackCode
	switch {
	case strings.Contains(message, "Agent store 未配置"):
		status = http.StatusServiceUnavailable
		code = "AGENT_STORE_UNAVAILABLE"
	case strings.Contains(message, "任务服务未初始化") || strings.Contains(message, "任务创建函数未配置"):
		status = http.StatusServiceUnavailable
		code = "AGENT_TASK_SERVICE_UNAVAILABLE"
	case strings.Contains(message, "Agent 会话不存在"):
		status = http.StatusNotFound
		code = "AGENT_SESSION_NOT_FOUND"
	case strings.Contains(message, "Agent 轮次不存在"):
		status = http.StatusNotFound
		code = "AGENT_ROUND_NOT_FOUND"
	case strings.Contains(message, "参考图不存在"):
		status = http.StatusNotFound
		code = "AGENT_REFERENCE_NOT_FOUND"
	case strings.Contains(message, "请输入创作需求"):
		code = "AGENT_MESSAGE_EMPTY"
	case strings.Contains(message, "还没有可确认"):
		code = "AGENT_ROUND_NOT_CONFIRMABLE"
	}
	writeError(w, status, code, message)
}

func agentSessionID(r *http.Request) string {
	if id := r.PathValue("sessionId"); id != "" {
		return id
	}
	return r.PathValue("id")
}

func latestAgentRound(session agents.Session) (agents.Round, bool) {
	if len(session.Rounds) == 0 {
		return agents.Round{}, false
	}
	return session.Rounds[len(session.Rounds)-1], true
}

func agentRoundByID(session agents.Session, id string) (agents.Round, bool) {
	id = strings.TrimSpace(id)
	if id == "" {
		return latestAgentRound(session)
	}
	for _, round := range session.Rounds {
		if round.ID == id {
			return round, true
		}
	}
	return agents.Round{}, false
}
