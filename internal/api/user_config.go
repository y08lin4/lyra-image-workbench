package api

import (
	"encoding/json"
	"net/http"

	"github.com/y08lin4/lyra-image-workbench/internal/spaceconfig"
)

type UserConfigHandler struct {
	store *spaceconfig.Store
}

func NewUserConfigHandler(store *spaceconfig.Store) UserConfigHandler {
	return UserConfigHandler{store: store}
}

func (h UserConfigHandler) Get(w http.ResponseWriter, r *http.Request) {
	cfg, err := h.store.Public(r.Header.Get("X-Space-Token"))
	if err != nil {
		writeSpaceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":     true,
		"config": cfg,
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
		"config": cfg,
	})
}
