package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/y08lin4/lyra-image-workbench/internal/apikeys"
	"github.com/y08lin4/lyra-image-workbench/internal/settings"
	"github.com/y08lin4/lyra-image-workbench/internal/spaceconfig"
)

type DeveloperAPIKeyHandler struct {
	store       *apikeys.Store
	spaceConfig *spaceconfig.Store
	settings    *settings.FileStore
}

func NewDeveloperAPIKeyHandler(store *apikeys.Store, spaceConfigStore *spaceconfig.Store, settingsStore *settings.FileStore) DeveloperAPIKeyHandler {
	return DeveloperAPIKeyHandler{store: store, spaceConfig: spaceConfigStore, settings: settingsStore}
}

func (h DeveloperAPIKeyHandler) List(w http.ResponseWriter, r *http.Request) {
	scope, ok := developerScopeFromRequest(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "USER_AUTH_REQUIRED", "请先登录")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "apiKeys": h.store.List(scope.storageToken)})
}

func (h DeveloperAPIKeyHandler) Create(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	scope, ok := developerScopeFromRequest(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "USER_AUTH_REQUIRED", "请先登录")
		return
	}
	var payload struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_JSON", "请求体不是有效 JSON")
		return
	}
	cfg, err := h.spaceConfig.Get(scope.storageToken)
	if err != nil {
		writeSpaceError(w, err)
		return
	}
	if !hasAnyCloudUpstreamKey(cfg) && !settings.HasAnySystemAPIKey(h.settings.Get()) {
		writeError(w, http.StatusBadRequest, "UPSTREAM_KEY_REQUIRED", "请先由管理员配置系统上游 Key，或在设置中保存个人云端上游 Key，再生成 Bearer API Key")
		return
	}
	record, secret, err := h.store.Create(payload.Name, scope.username, scope.storageToken)
	if err != nil {
		writeError(w, http.StatusBadRequest, "API_KEY_CREATE_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "apiKey": apikeys.ToPublic(record), "secret": secret})
}

func (h DeveloperAPIKeyHandler) Delete(w http.ResponseWriter, r *http.Request) {
	scope, ok := developerScopeFromRequest(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "USER_AUTH_REQUIRED", "请先登录")
		return
	}
	deleted, err := h.store.Delete(scope.storageToken, r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "API_KEY_DELETE_FAILED", err.Error())
		return
	}
	if !deleted {
		writeError(w, http.StatusNotFound, "API_KEY_NOT_FOUND", "API Key 不存在")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func hasAnyCloudUpstreamKey(cfg spaceconfig.Config) bool {
	return (cfg.CloudAPIKeyEnabled && cfg.APIKey != "") || (cfg.CloudBananaAPIKeyEnabled && cfg.BananaAPIKey != "")
}

type developerScope struct {
	storageToken string
	username     string
}

func developerScopeFromRequest(r *http.Request) (developerScope, bool) {
	scope := developerScope{
		storageToken: strings.TrimSpace(r.Header.Get(userStorageTokenHeader)),
		username:     strings.TrimSpace(r.Header.Get("X-User-Name")),
	}
	return scope, scope.storageToken != ""
}
