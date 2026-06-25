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
	"github.com/y08lin4/lyra-image-workbench/internal/spaceconfig"
)

type V1ImageTaskHandler struct {
	apiKeys     *apikeys.Store
	spaceConfig *spaceconfig.Store
	manager     *jobs.Manager
	output      *output.Store
}

func NewV1ImageTaskHandler(apiKeyStore *apikeys.Store, spaceConfigStore *spaceconfig.Store, manager *jobs.Manager, outputStore *output.Store) V1ImageTaskHandler {
	return V1ImageTaskHandler{apiKeys: apiKeyStore, spaceConfig: spaceConfigStore, manager: manager, output: outputStore}
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
	if err := h.requireCloudUpstreamKey(scope.storageToken, payload.Provider); err != nil {
		writeError(w, http.StatusBadRequest, "UPSTREAM_KEY_REQUIRED", err.Error())
		return
	}
	job, err := h.manager.Create(scope.storageToken, payload)
	if err != nil {
		writeError(w, http.StatusBadRequest, "TASK_CREATE_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "task": publicV1Job(job)})
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
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Bearer API Key 缺失或无效")
		return v1Scope{}, false
	}
	record, ok, err := h.apiKeys.Authenticate(secret)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return v1Scope{}, false
	}
	if !ok {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Bearer API Key 缺失或无效")
		return v1Scope{}, false
	}
	return v1Scope{storageToken: record.StorageToken, apiKeyID: record.ID}, true
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
	if provider == config.BananaProvider {
		if cfg.CloudBananaAPIKeyEnabled && strings.TrimSpace(cfg.BananaAPIKey) != "" {
			return nil
		}
		return errUpstreamKeyRequired("请先在设置中确认保存 Banana 云端上游 Key，再使用外部 API 创建任务")
	}
	if cfg.CloudAPIKeyEnabled && strings.TrimSpace(cfg.APIKey) != "" {
		return nil
	}
	return errUpstreamKeyRequired("请先在设置中确认保存 codex-key 云端上游 Key，再使用外部 API 创建任务")
}

type v1Scope struct {
	storageToken string
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
