package api

import (
	"encoding/json"
	"net/http"

	"github.com/y08lin4/image-Workbench-Localhost-Version/internal/settings"
)

type AdminConfigHandler struct {
	store *settings.FileStore
}

func NewAdminConfigHandler(store *settings.FileStore) AdminConfigHandler {
	return AdminConfigHandler{store: store}
}

func (h AdminConfigHandler) Get(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":     true,
		"config": h.store.Public(),
	})
}

func (h AdminConfigHandler) Update(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var payload settings.Update
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_JSON", "请求体不是有效 JSON")
		return
	}
	if _, err := h.store.Update(payload); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_ADMIN_CONFIG", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":     true,
		"config": h.store.Public(),
	})
}
