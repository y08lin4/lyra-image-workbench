package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/y08lin4/lyra-image-workbench/internal/apikeys"
	"github.com/y08lin4/lyra-image-workbench/internal/config"
	"github.com/y08lin4/lyra-image-workbench/internal/jobs"
	"github.com/y08lin4/lyra-image-workbench/internal/output"
	"github.com/y08lin4/lyra-image-workbench/internal/settings"
	"github.com/y08lin4/lyra-image-workbench/internal/spaceconfig"
	"github.com/y08lin4/lyra-image-workbench/internal/users"
)

type V1ImageTaskHandler struct {
	apiKeys     *apikeys.Store
	spaceConfig *spaceconfig.Store
	settings    *settings.FileStore
	manager     *jobs.Manager
	output      *output.Store
	users       *users.Store
}

func NewV1ImageTaskHandler(apiKeyStore *apikeys.Store, spaceConfigStore *spaceconfig.Store, settingsStore *settings.FileStore, manager *jobs.Manager, outputStore *output.Store, userStores ...*users.Store) V1ImageTaskHandler {
	var userStore *users.Store
	if len(userStores) > 0 {
		userStore = userStores[0]
	}
	return V1ImageTaskHandler{apiKeys: apiKeyStore, spaceConfig: spaceConfigStore, settings: settingsStore, manager: manager, output: outputStore, users: userStore}
}

func (h V1ImageTaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	scope, ok := h.authenticate(w, r)
	if !ok {
		return
	}
	defer r.Body.Close()
	var payload jobs.CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_JSON", "请求体不是有效 JSON")
		return
	}
	if payload.Mode != jobs.ModeTextToImage {
		writeError(w, http.StatusBadRequest, "TASK_CREATE_FAILED", "外部 API 暂只支持 text-to-image")
		return
	}
	payload.RuntimeSecrets = jobs.RuntimeSecrets{}
	payload.Source = jobs.JobSourceAPI
	payload.WaiveCredits = h.waiveTaskCredits(scope.storageToken, payload)
	payload.BeforeEnqueue = func(job jobs.Job) error { return h.chargeTask(scope.username, job) }
	if err := h.ensureTaskCredits(scope.username, billableTaskCredits(payload)); err != nil {
		writeUserCreditError(w, err)
		return
	}
	if err := h.requireCloudUpstreamKey(scope.storageToken, payload.Provider); err != nil {
		writeError(w, http.StatusBadRequest, "UPSTREAM_KEY_REQUIRED", err.Error())
		return
	}
	job, err := h.manager.Create(scope.storageToken, payload)
	if err != nil {
		if isUserCreditError(err) {
			writeUserCreditError(w, err)
			return
		}
		writeError(w, http.StatusBadRequest, "TASK_CREATE_FAILED", err.Error())
		return
	}

	public := publicV1Job(job)
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":              true,
		"task":            public,
		"taskId":          public.ID,
		"consumedCredits": public.ConsumedCredits,
	})
}

func (h V1ImageTaskHandler) CreateGeneration(w http.ResponseWriter, r *http.Request) {
	scope, ok := h.authenticate(w, r)
	if !ok {
		return
	}
	defer r.Body.Close()
	var payload v1ImageGenerationRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_JSON", "请求体不是有效 JSON")
		return
	}
	createPayload := payload.createRequest()
	createPayload.Source = jobs.JobSourceAPI
	createPayload.WaiveCredits = h.waiveTaskCredits(scope.storageToken, createPayload)
	createPayload.BeforeEnqueue = func(job jobs.Job) error { return h.chargeTask(scope.username, job) }
	if err := h.ensureTaskCredits(scope.username, billableTaskCredits(createPayload)); err != nil {
		writeUserCreditError(w, err)
		return
	}
	if err := h.requireCloudUpstreamKey(scope.storageToken, createPayload.Provider); err != nil {
		writeError(w, http.StatusBadRequest, "UPSTREAM_KEY_REQUIRED", err.Error())
		return
	}
	job, err := h.manager.Create(scope.storageToken, createPayload)
	if err != nil {
		if isUserCreditError(err) {
			writeUserCreditError(w, err)
			return
		}
		writeError(w, http.StatusBadRequest, "TASK_CREATE_FAILED", err.Error())
		return
	}

	public := publicV1Job(job)
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":              true,
		"task":            public,
		"taskId":          public.ID,
		"consumedCredits": public.ConsumedCredits,
	})
}

func (h V1ImageTaskHandler) ensureTaskCredits(username string, amount int) error {
	return ensureTaskCredits(h.users, username, amount)
}

func (h V1ImageTaskHandler) chargeTask(username string, job jobs.Job) error {
	return chargeTaskCredits(h.users, username, job)
}

func (h V1ImageTaskHandler) waiveTaskCredits(spaceToken string, req jobs.CreateRequest) bool {
	return requestHasInvalidProvider(req.Provider) || requestUsesPersonalUpstreamKey(h.spaceConfig, spaceToken, req)
}

type v1ImageGenerationRequest struct {
	Provider          string `json:"provider,omitempty"`
	Model             string `json:"model"`
	Prompt            string `json:"prompt"`
	Size              string `json:"size,omitempty"`
	Ratio             string `json:"ratio,omitempty"`
	Resolution        string `json:"resolution,omitempty"`
	Quality           string `json:"quality,omitempty"`
	OutputFormat      string `json:"output_format,omitempty"`
	OutputFormatCamel string `json:"outputFormat,omitempty"`
	Count             int    `json:"count,omitempty"`
	N                 int    `json:"n,omitempty"`
	Concurrency       int    `json:"concurrency,omitempty"`
}

func (req v1ImageGenerationRequest) createRequest() jobs.CreateRequest {
	ratio, resolution := v1SizeSpec(req.Size)
	if strings.TrimSpace(req.Ratio) != "" {
		ratio = strings.TrimSpace(req.Ratio)
	}
	if strings.TrimSpace(req.Resolution) != "" {
		resolution = strings.TrimSpace(req.Resolution)
	}
	outputFormat := strings.TrimSpace(req.OutputFormat)
	if outputFormat == "" {
		outputFormat = strings.TrimSpace(req.OutputFormatCamel)
	}
	count := req.N
	if count == 0 {
		count = req.Count
	}
	return jobs.CreateRequest{
		Provider:     strings.TrimSpace(req.Provider),
		Model:        strings.TrimSpace(req.Model),
		Mode:         jobs.ModeTextToImage,
		Prompt:       strings.TrimSpace(req.Prompt),
		Ratio:        ratio,
		Resolution:   resolution,
		Quality:      strings.TrimSpace(req.Quality),
		OutputFormat: outputFormat,
		Count:        count,
		Concurrency:  req.Concurrency,
	}
}

func v1SizeSpec(size string) (string, string) {
	switch strings.ToLower(strings.TrimSpace(size)) {
	case "auto":
		return "auto", "auto"
	case "1024x1024":
		return "1:1", "standard"
	case "1024x1536":
		return "2:3", "standard"
	case "1536x1024":
		return "3:2", "standard"
	case "768x1024":
		return "3:4", "standard"
	case "1024x768":
		return "4:3", "standard"
	case "1008x1792":
		return "9:16", "standard"
	case "1792x1008":
		return "16:9", "standard"
	case "2048x2048":
		return "1:1", "2k"
	case "1344x2016":
		return "2:3", "2k"
	case "2016x1344":
		return "3:2", "2k"
	case "1536x2048":
		return "3:4", "2k"
	case "2048x1536":
		return "4:3", "2k"
	case "1152x2048":
		return "9:16", "2k"
	case "2048x1152":
		return "16:9", "2k"
	case "2880x2880":
		return "1:1", "4k"
	case "2336x3504":
		return "2:3", "4k"
	case "3504x2336":
		return "3:2", "4k"
	case "2448x3264":
		return "3:4", "4k"
	case "3264x2448":
		return "4:3", "4k"
	case "2160x3840":
		return "9:16", "4k"
	case "3840x2160":
		return "16:9", "4k"
	default:
		return "", ""
	}
}

func (h V1ImageTaskHandler) Get(w http.ResponseWriter, r *http.Request) {
	scope, ok := h.authenticate(w, r)
	if !ok {
		return
	}
	job, found, err := h.manager.Get(scope.storageToken, r.PathValue("id"))
	if err != nil {
		writeSpaceError(w, err)
		return
	}
	if !found {
		writeError(w, http.StatusNotFound, "TASK_NOT_FOUND", "任务不存在")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "task": publicV1Job(job)})
}

func (h V1ImageTaskHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	scope, ok := h.authenticate(w, r)
	if !ok {
		return
	}
	if _, found, err := h.manager.Get(scope.storageToken, r.PathValue("id")); err != nil {
		writeSpaceError(w, err)
		return
	} else if !found {
		writeError(w, http.StatusNotFound, "TASK_NOT_FOUND", "任务不存在")
		return
	}
	job, err := h.manager.Cancel(scope.storageToken, r.PathValue("id"))
	if err != nil {
		if strings.Contains(err.Error(), "任务不存在") {
			writeError(w, http.StatusNotFound, "TASK_NOT_FOUND", "任务不存在")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "task": publicV1Job(job)})
}

func (h V1ImageTaskHandler) Image(w http.ResponseWriter, r *http.Request) {
	scope, ok := h.authenticate(w, r)
	if !ok {
		return
	}
	job, found, err := h.manager.Get(scope.storageToken, r.PathValue("id"))
	if err != nil {
		writeSpaceError(w, err)
		return
	}
	if !found {
		writeError(w, http.StatusNotFound, "TASK_NOT_FOUND", "任务不存在")
		return
	}
	index, err := strconv.Atoi(r.PathValue("index"))
	if err != nil || index < 0 {
		writeError(w, http.StatusNotFound, "TASK_IMAGE_NOT_FOUND", "任务图片不存在")
		return
	}
	result, ok := resultByIndex(job, index)
	if !ok || !result.OK {
		writeError(w, http.StatusNotFound, "TASK_IMAGE_NOT_FOUND", "任务图片不存在")
		return
	}
	serveV1ResultImage(w, r, h.output, job, result)
}

func (h V1ImageTaskHandler) authenticate(w http.ResponseWriter, r *http.Request) (v1Scope, bool) {
	secret := bearerSecret(r.Header.Get("Authorization"))
	if secret == "" {
		h.writeInvalidBearer(w, r)
		return v1Scope{}, false
	}
	record, ok, err := h.apiKeys.Authenticate(secret)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return v1Scope{}, false
	}
	if !ok {
		h.writeInvalidBearer(w, r)
		return v1Scope{}, false
	}
	username := strings.TrimSpace(record.Username)
	if username == "" && h.users != nil {
		if owner, found := h.users.FindByStorageToken(record.StorageToken); found {
			username = owner.Username
		}
	}
	return v1Scope{storageToken: record.StorageToken, username: username, apiKeyID: record.ID}, true
}

func (h V1ImageTaskHandler) writeInvalidBearer(w http.ResponseWriter, r *http.Request) {
	identity := "api-bearer-invalid"
	if ok, retryAfter := authAttemptAllowed("api-bearer", r, identity); !ok {
		writeAuthRateLimited(w, retryAfter)
		return
	}
	recordAuthAttempt("api-bearer", r, identity, false)
	writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Bearer API Key 缺失或无效")
}

func (h V1ImageTaskHandler) requireCloudUpstreamKey(storageToken string, providerValue string) error {
	provider, ok := providerForV1(providerValue)
	if !ok {
		return nil
	}
	cfg, err := h.spaceConfig.Get(storageToken)
	if err != nil {
		return err
	}
	if h.settings != nil && settings.HasSystemAPIKeyForProvider(h.settings.Get(), provider) {
		return nil
	}
	if provider == config.BananaProvider {
		if cfg.CloudBananaAPIKeyEnabled && strings.TrimSpace(cfg.BananaAPIKey) != "" {
			return nil
		}
		return errUpstreamKeyRequired("请先由管理员配置系统 Banana 上游 Key，或在设置中保存个人 Banana 云端上游 Key，再使用外部 API 创建任务")
	}
	if cfg.CloudAPIKeyEnabled && strings.TrimSpace(cfg.APIKey) != "" {
		return nil
	}
	return errUpstreamKeyRequired("请先由管理员配置系统 codex-key 上游 Key，或在设置中保存个人 codex-key 云端上游 Key，再使用外部 API 创建任务")
}

type v1Scope struct {
	storageToken string
	username     string
	apiKeyID     string
}

type errUpstreamKeyRequired string

func (e errUpstreamKeyRequired) Error() string { return string(e) }

func bearerSecret(header string) string {
	const prefix = "Bearer "
	header = strings.TrimSpace(header)
	if !strings.HasPrefix(header, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(header, prefix))
}

func providerForV1(value string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", config.DefaultProvider, "image2", "gpt-image-2":
		return config.DefaultProvider, true
	case config.BananaProvider, "banana-nano", "nano-banana":
		return config.BananaProvider, true
	default:
		return "", false
	}
}

func publicV1Job(job jobs.Job) jobs.Job {
	public := jobs.PublicJob(job)
	for i := range public.Results {
		if public.Results[i].OK {
			public.Results[i].ImageURL = "/v1/image-tasks/" + public.ID + "/images/" + strconv.Itoa(public.Results[i].Index)
		}
	}
	return public
}

func resultByIndex(job jobs.Job, index int) (jobs.Result, bool) {
	for _, result := range job.Results {
		if result.Index == index {
			return result, true
		}
	}
	return jobs.Result{}, false
}
func serveV1ResultImage(w http.ResponseWriter, r *http.Request, store *output.Store, job jobs.Job, result jobs.Result) {
	var path, mime string
	var err error
	if result.OutputDate != "" && result.OutputFileName != "" {
		path, mime, err = store.Resolve(job.SpaceToken, result.OutputDate, result.OutputFileName)
	} else {
		path, mime, err = store.ResolveURL(result.ImageURL)
	}
	if err != nil {
		writeError(w, http.StatusNotFound, "TASK_IMAGE_NOT_FOUND", "任务图片不存在")
		return
	}
	w.Header().Set("Content-Type", mime)
	w.Header().Set("Cache-Control", "private, max-age=86400")
	http.ServeFile(w, r, path)
}
