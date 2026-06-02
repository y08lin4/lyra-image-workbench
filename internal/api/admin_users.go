package api

import (
	"encoding/json"
	"net/http"

	"github.com/y08lin4/lyra-image-workbench/internal/adminauth"
	"github.com/y08lin4/lyra-image-workbench/internal/users"
)

type AdminUsersHandler struct {
	store *users.Store
	auth  *adminauth.Store
}

func NewAdminUsersHandler(store *users.Store, auth *adminauth.Store) AdminUsersHandler {
	return AdminUsersHandler{store: store, auth: auth}
}

func (h AdminUsersHandler) List(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r, h.auth) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "users": h.store.ListAdminUsers()})
}

func (h AdminUsersHandler) AddVideoQuota(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r, h.auth) {
		return
	}
	defer r.Body.Close()
	var payload struct {
		Username string `json:"username"`
		Delta    int    `json:"delta"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_JSON", "请求体不是有效 JSON")
		return
	}
	user, err := h.store.AddVideoQuota(payload.Username, payload.Delta)
	if err != nil {
		writeUserError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "user": user, "users": h.store.ListAdminUsers()})
}
