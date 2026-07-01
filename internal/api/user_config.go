package api

import (
	"encoding/json"
	"net/http"

	"github.com/y08lin4/lyra-image-workbench/internal/settings"
	"github.com/y08lin4/lyra-image-workbench/internal/spaceconfig"
)

type UserConfigHandler struct {
	store    *spaceconfig.Store
	settings *settings.FileStore
}

func NewUserConfigHandler(store *spaceconfig.Store, settingsStore *settings.FileStore) UserConfigHandler {
	return UserConfigHandler{store: store, settings: settingsStore}
}

func (h UserConfigHandler) Get(w http.ResponseWriter, r *http.Request) {
	cfg, err := h.store.Public(r.Header.Get("X-Space-Token"))
	if err != nil {
		writeSpaceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":     true,
		"config": h.withSystemKeyStatus(cfg),
	})
}

func (h UserConfigHandler) Update(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var payload spaceconfig.Update
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_JSON", "请求体不是有效 JSON")
		return
	}
	cfg, err := h.store.Update(r.Header.Get("X-Space-Token"), payload)
	if err != nil {
		writeSpaceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":     true,
		"config": h.withSystemKeyStatus(cfg),
	})
}

func (h UserConfigHandler) withSystemKeyStatus(cfg spaceconfig.PublicConfig) map[string]any {
	payload := map[string]any{
		"apiKeySet":                cfg.APIKeySet,
		"apiKeyPreview":            cfg.APIKeyPreview,
		"cloudApiKeySet":           cfg.CloudAPIKeySet,
		"cloudApiKeyPreview":       cfg.CloudAPIKeyPreview,
		"defaultCount":             cfg.DefaultCount,
		"defaultConcurrency":       cfg.DefaultConcurrency,
		"autoUploadPixhost":        cfg.AutoUploadPixhost,
		"updatedAt":                cfg.UpdatedAt,
	}
	if h.settings != nil {
		site := h.settings.Public()
		payload["systemApiKeySet"] = site.SystemAPIKeySet
	}
	return payload
}
