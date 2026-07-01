package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/y08lin4/lyra-image-workbench/internal/config"
	"github.com/y08lin4/lyra-image-workbench/internal/events"
	"github.com/y08lin4/lyra-image-workbench/internal/jobs"
	"github.com/y08lin4/lyra-image-workbench/internal/output"
	"github.com/y08lin4/lyra-image-workbench/internal/spaceconfig"
	"github.com/y08lin4/lyra-image-workbench/internal/users"
)

type TaskHandler struct {
	manager     *jobs.Manager
	output      *output.Store
	spaceConfig *spaceconfig.Store
	users       *users.Store
}

func NewTaskHandler(manager *jobs.Manager, outputStore *output.Store, spaceConfigStore *spaceconfig.Store, userStores ...*users.Store) TaskHandler {
	var userStore *users.Store
	if len(userStores) > 0 {
		userStore = userStores[0]
	}
	return TaskHandler{manager: manager, output: outputStore, spaceConfig: spaceConfigStore, users: userStore}
}

func (h TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	payload, err := decodeCreateRequest(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "BAD_JSON", "请求体不是有效 JSON")
		return
	}
	payload.RuntimeSecrets = runtimeSecretsFromRequest(r)
	payload.Source = jobs.JobSourceWeb
	spaceToken := r.Header.Get("X-Space-Token")
	username := r.Header.Get("X-User-Name")
	payload.WaiveCredits = h.waiveTaskCredits(spaceToken, payload)
	if err := h.ensureTaskCredits(username, billableTaskCredits(payload)); err != nil {
		writeUserCreditError(w, err)
		return
	}
	payload.BeforeEnqueue = func(job jobs.Job) error { return h.chargeTask(username, job) }
	job, err := h.manager.Create(spaceToken, payload)
	if err != nil {
		if isUserCreditError(err) {
			writeUserCreditError(w, err)
			return
		}
		writeError(w, http.StatusBadRequest, "TASK_CREATE_FAILED", err.Error())
		return
	}

	public := jobs.PublicJob(job)
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":              true,
		"job":             public,
		"taskId":          public.ID,
		"consumedCredits": public.ConsumedCredits,
	})
}

func decodeCreateRequest(body io.Reader) (jobs.CreateRequest, error) {
	var payload struct {
		jobs.CreateRequest
		Profile           string         `json:"profile"`
		ModelProfile      string         `json:"modelProfile"`
		ModelProfileSnake string         `json:"model_profile"`
		ChannelID         string         `json:"channel_id"`
		ExtraParamsSnake  map[string]any `json:"extra_params"`
		ExtraBody         map[string]any `json:"extra_body"`
	}
	if err := json.NewDecoder(body).Decode(&payload); err != nil {
		return jobs.CreateRequest{}, err
	}
	req := normalizeCreateRequestCompatibility(payload.CreateRequest, payload.Profile, payload.ModelProfile, payload.ModelProfileSnake, payload.ChannelID)
	req.ExtraParams = mergeExtraParamAliases(req.ExtraParams, payload.ExtraParamsSnake, payload.ExtraBody)
	return req, nil
}

func mergeExtraParamAliases(values ...map[string]any) map[string]any {
	var out map[string]any
	for _, value := range values {
		if len(value) == 0 {
			continue
		}
		if out == nil {
			out = make(map[string]any, len(value))
		}
		for key, item := range value {
			trimmed := strings.TrimSpace(key)
			if trimmed == "" || isCoreAPIExtraParamKey(trimmed) {
				continue
			}
			out[trimmed] = item
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func normalizeCreateRequestCompatibility(req jobs.CreateRequest, profiles ...string) jobs.CreateRequest {
	profile := firstNonEmpty(profiles...)
	if isImage2ModelProfile(profile) {
		req.Model = canonicalImage2Model(profile)
	}
	if isImage2LegacyProviderProfile(req.Provider) {
		if !isImage2ModelProfile(profile) && isDefaultImage2Model(req.Model) {
			req.Model = canonicalImage2Model(req.Provider)
		}
		req.Provider = config.DefaultProvider
		return req
	}
	if provider, ok := openAIImageProvider(req.Provider); ok {
		req.Provider = provider
	}
	return req
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func isImage2LegacyProviderProfile(value string) bool {
	return strings.EqualFold(strings.TrimSpace(value), "image-2-4k")
}

func isImage2ModelProfile(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "image-2", "image-2-4k":
		return true
	default:
		return false
	}
}

func canonicalImage2Model(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "image-2-4k":
		return "image-2-4k"
	case "image-2":
		return "image-2"
	default:
		return strings.TrimSpace(value)
	}
}

func isDefaultImage2Model(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "image-2", config.DefaultModel:
		return true
	default:
		return false
	}
}

func openAIImageProvider(value string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", config.DefaultProvider, "image2", "gpt-image-2", "image-2-4k", "openai", "openai-compatible", "openai_compatible":
		return config.DefaultProvider, true
	default:
		return "", false
	}
}

func isCoreAPIExtraParamKey(key string) bool {
	switch canonicalAPIExtraParamKey(key) {
	case "model", "prompt", "n", "responseformat", "size", "quality", "outputformat", "provider", "profile", "modelprofile", "channelid", "mode", "source", "ratio", "resolution", "count", "concurrency", "uploadids", "references", "runtimesecrets", "apikey", "authorization", "extraparams", "extrabody":
		return true
	default:
		return false
	}
}

func canonicalAPIExtraParamKey(key string) string {
	key = strings.ToLower(strings.TrimSpace(key))
	key = strings.ReplaceAll(key, "_", "")
	key = strings.ReplaceAll(key, "-", "")
	key = strings.ReplaceAll(key, ".", "")
	return key
}

func (h TaskHandler) List(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	items, err := h.manager.List(r.Header.Get("X-Space-Token"), limit)
	if err != nil {
		writeSpaceError(w, err)
		return
	}
	for i := range items {
		items[i] = jobs.PublicJob(items[i])
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "tasks": items})
}

func (h TaskHandler) Get(w http.ResponseWriter, r *http.Request) {
	job, ok, err := h.manager.Get(r.Header.Get("X-Space-Token"), r.PathValue("id"))
	if err != nil {
		writeSpaceError(w, err)
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, "TASK_NOT_FOUND", "任务不存在")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "task": jobs.PublicJob(job)})
}

func (h TaskHandler) Retry(w http.ResponseWriter, r *http.Request) {
	spaceToken := r.Header.Get("X-Space-Token")
	username := r.Header.Get("X-User-Name")
	secrets := runtimeSecretsFromRequest(r)
	waiveCredits := h.waiveRetryCredits(spaceToken, r.PathValue("id"), secrets)
	job, err := h.manager.Retry(spaceToken, r.PathValue("id"), secrets, waiveCredits, func(job jobs.Job) error { return h.chargeTask(username, job) })
	if err != nil {
		if isUserCreditError(err) {
			writeUserCreditError(w, err)
			return
		}
		writeError(w, http.StatusBadRequest, "TASK_RETRY_FAILED", err.Error())
		return
	}

	public := jobs.PublicJob(job)
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":              true,
		"job":             public,
		"taskId":          public.ID,
		"consumedCredits": public.ConsumedCredits,
	})
}

func (h TaskHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	job, err := h.manager.Cancel(r.Header.Get("X-Space-Token"), r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "TASK_CANCEL_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "job": jobs.PublicJob(job)})
}

func (h TaskHandler) Favorite(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var payload struct {
		Favorite bool `json:"favorite"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_JSON", "请求体不是有效 JSON")
		return
	}
	job, err := h.manager.SetFavorite(r.Header.Get("X-Space-Token"), r.PathValue("id"), payload.Favorite)
	if err != nil {
		writeError(w, http.StatusBadRequest, "TASK_FAVORITE_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "job": jobs.PublicJob(job)})
}

func (h TaskHandler) Delete(w http.ResponseWriter, r *http.Request) {
	job, err := h.manager.Delete(r.Header.Get("X-Space-Token"), r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "TASK_DELETE_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "job": jobs.PublicJob(job)})
}

func (h TaskHandler) UploadPixhost(w http.ResponseWriter, r *http.Request) {
	idx, err := strconv.Atoi(r.PathValue("index"))
	if err != nil || idx < 0 {
		writeError(w, http.StatusBadRequest, "TASK_IMAGE_INDEX_INVALID", "图片序号无效")
		return
	}
	job, result, err := h.manager.UploadResultToPixhost(r.Context(), r.Header.Get("X-Space-Token"), r.PathValue("id"), idx)
	if err != nil {
		writeError(w, http.StatusBadRequest, "PIXHOST_UPLOAD_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "job": jobs.PublicJob(job), "result": publicResult(job, result)})
}

func (h TaskHandler) Events(w http.ResponseWriter, r *http.Request) {
	spaceToken := r.Header.Get("X-Space-Token")
	jobID := r.PathValue("id")
	job, ok, err := h.manager.Get(spaceToken, jobID)
	if err != nil {
		writeSpaceError(w, err)
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, "TASK_NOT_FOUND", "任务不存在")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	flusher, _ := w.(http.Flusher)
	sendSSE(w, events.Event{Event: "snapshot", Code: "E100", English: "snapshot", Chinese: "任务快照", Data: map[string]any{"job": jobs.PublicJob(job)}})
	if flusher != nil {
		flusher.Flush()
	}
	ch, cancel := h.manager.Subscribe(jobID)
	defer cancel()
	heartbeat := time.NewTicker(5 * time.Second)
	defer heartbeat.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-heartbeat.C:
			sendSSE(w, events.Event{Event: "heartbeat", Code: "E130", English: "heartbeat", Chinese: "心跳保活", Data: map[string]any{"time": time.Now().Format(time.RFC3339)}})
			if flusher != nil {
				flusher.Flush()
			}
		case event, ok := <-ch:
			if !ok {
				return
			}
			sendSSE(w, event)
			if flusher != nil {
				flusher.Flush()
			}
			if event.Event == "done" {
				return
			}
		}
	}
}

func (h TaskHandler) Image(w http.ResponseWriter, r *http.Request) {
	spaceToken := r.Header.Get("X-Space-Token")
	job, ok, err := h.manager.Get(spaceToken, r.PathValue("id"))
	if err != nil {
		writeSpaceError(w, err)
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, "TASK_NOT_FOUND", "任务不存在")
		return
	}
	idx, err := strconv.Atoi(r.PathValue("index"))
	if err != nil || idx < 0 || idx >= len(job.Results) || !job.Results[idx].OK {
		writeError(w, http.StatusNotFound, "TASK_IMAGE_NOT_FOUND", "任务图片不存在")
		return
	}
	serveResultImage(w, r, h.output, job, job.Results[idx])
}

func (h TaskHandler) Stats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.manager.Stats(r.Header.Get("X-Space-Token"))
	if err != nil {
		writeSpaceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "stats": stats})
}

func (h TaskHandler) ensureTaskCredits(username string, amount int) error {
	return ensureTaskCredits(h.users, username, amount)
}

func (h TaskHandler) chargeTask(username string, job jobs.Job) error {
	return chargeTaskCredits(h.users, username, job)
}

func (h TaskHandler) waiveTaskCredits(spaceToken string, req jobs.CreateRequest) bool {
	return requestHasInvalidProvider(req.Provider) || requestUsesPersonalUpstreamKey(h.spaceConfig, spaceToken, req)
}

func (h TaskHandler) waiveRetryCredits(spaceToken string, id string, secrets jobs.RuntimeSecrets) bool {
	old, ok, err := h.manager.Get(spaceToken, id)
	if err != nil || !ok {
		return false
	}
	return requestHasInvalidProvider(old.Provider) || requestUsesPersonalUpstreamKey(h.spaceConfig, spaceToken, jobs.CreateRequest{RuntimeSecrets: secrets, Provider: old.Provider, Model: old.Model, Mode: old.Mode})
}

func ensureTaskCredits(userStore *users.Store, username string, amount int) error {
	if userStore == nil {
		return nil
	}
	username = strings.TrimSpace(username)
	if username == "" {
		return users.NewError("USER_AUTH_REQUIRED", "请先登录")
	}
	if amount <= 0 {
		return nil
	}
	profile, err := userStore.Profile(username)
	if err != nil {
		return err
	}
	if profile.CreditsBalance < amount {
		return users.NewError("USER_CREDITS_NOT_ENOUGH", "次数不足")
	}
	return nil
}

func chargeTaskCredits(userStore *users.Store, username string, job jobs.Job) error {
	if userStore == nil {
		return nil
	}
	username = strings.TrimSpace(username)
	if username == "" {
		return users.NewError("USER_AUTH_REQUIRED", "请先登录")
	}
	amount := job.ConsumedCredits
	if amount <= 0 {
		return nil
	}
	_, err := userStore.ChargeTaskCredits(username, amount, job.ID, taskChargeReason(job))
	return err
}

func billableTaskCredits(req jobs.CreateRequest) int {
	if req.WaiveCredits {
		return 0
	}
	return jobs.CreditCostForRequest(req)
}

func requestUsesPersonalUpstreamKey(spaceConfigStore *spaceconfig.Store, spaceToken string, req jobs.CreateRequest) bool {
	if req.Mode == jobs.ModeGIF {
		return true
	}
	provider := requestProvider(req.Provider, req.Model)
	if runtimeAPIKeyForProvider(req.RuntimeSecrets, provider) != "" {
		return true
	}
	if spaceConfigStore == nil {
		return false
	}
	cfg, err := spaceConfigStore.Get(spaceToken)
	if err != nil {
		return false
	}
	return cloudAPIKeyForProvider(cfg, provider) != ""
}

func requestProvider(providerValue string, modelValue string) string {
	provider, ok := openAIImageProvider(providerValue)
	if !ok {
		return config.DefaultProvider
	}
	if strings.EqualFold(strings.TrimSpace(providerValue), "image-2-4k") || strings.EqualFold(strings.TrimSpace(modelValue), "image-2-4k") {
		return "image-2-4k"
	}
	return provider
}

func requestHasInvalidProvider(value string) bool {
	_, ok := openAIImageProvider(value)
	return !ok
}

func runtimeAPIKeyForProvider(secrets jobs.RuntimeSecrets, _ string) string {
	return strings.TrimSpace(secrets.APIKey)
}

func cloudAPIKeyForProvider(cfg spaceconfig.Config, _ string) string {
	if cfg.CloudAPIKeyEnabled {
		return strings.TrimSpace(cfg.APIKey)
	}
	return ""
}
func taskChargeReason(job jobs.Job) string {
	label := "生成任务扣减"
	if job.Source == jobs.JobSourceAPI {
		label = "API 生成任务扣减"
	}
	if job.Source == jobs.JobSourceAgent {
		label = "Agent 生成任务扣减"
	}
	if job.Count > 1 {
		return fmt.Sprintf("%s（%d 张）", label, job.Count)
	}
	return label
}

func isUserCreditError(err error) bool {
	var userErr users.Error
	return users.AsError(err, &userErr)
}

func writeUserCreditError(w http.ResponseWriter, err error) {
	var userErr users.Error
	if users.AsError(err, &userErr) {
		status := http.StatusBadRequest
		if userErr.Code == "USER_AUTH_REQUIRED" {
			status = http.StatusUnauthorized
		}
		if userErr.Code == "USER_CREDITS_NOT_ENOUGH" {
			status = http.StatusPaymentRequired
		}
		writeError(w, status, userErr.Code, userErr.Chinese)
		return
	}
	writeError(w, http.StatusBadRequest, "USER_CREDITS_ERROR", err.Error())
}

func sendSSE(w http.ResponseWriter, event events.Event) {
	event.Data = publicEventData(event.Data)
	payload, _ := json.Marshal(event)
	fmt.Fprintf(w, "event: %s\n", event.Event)
	fmt.Fprintf(w, "data: %s\n\n", payload)
}

func publicEventData(data any) any {
	payload, ok := data.(map[string]any)
	if !ok {
		return data
	}
	job, hasJob := payload["job"].(jobs.Job)
	if !hasJob {
		return data
	}
	next := make(map[string]any, len(payload))
	for key, value := range payload {
		next[key] = value
	}
	next["job"] = jobs.PublicJob(job)
	if result, ok := payload["result"].(jobs.Result); ok {
		next["result"] = publicResult(job, result)
	}
	return next
}

func serveResultImage(w http.ResponseWriter, r *http.Request, store *output.Store, job jobs.Job, result jobs.Result) {
	var path, mime string
	var err error
	if result.OutputDate != "" && result.OutputFileName != "" {
		path, mime, err = store.Resolve(job.SpaceToken, result.OutputDate, result.OutputFileName)
	} else {
		path, mime, err = store.ResolveURL(result.ImageURL)
	}
	if err != nil {
		writeError(w, http.StatusBadRequest, "OUTPUT_PATH_INVALID", err.Error())
		return
	}
	w.Header().Set("Content-Type", mime)
	w.Header().Set("Cache-Control", "private, max-age=86400")
	http.ServeFile(w, r, path)
}

func publicResult(job jobs.Job, result jobs.Result) jobs.Result {
	return jobs.PublicResult(job.ID, result)
}
