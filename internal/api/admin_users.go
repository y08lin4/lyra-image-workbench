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
	if !h.requireAdminAccess(w, r) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "users": h.store.ListAdminUsers()})
}

func (h AdminUsersHandler) AddCredits(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdminAccess(w, r) {
		return
	}
	defer r.Body.Close()
	var payload struct {
		Username string `json:"username"`
		Amount   int    `json:"amount"`
		Reason   string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_JSON", "请求体不是有效 JSON")
		return
	}
	user, entry, err := h.store.AddCreditsByAdmin(payload.Username, payload.Amount, payload.Reason, h.adminActorFromRequest(r))
	if err != nil {
		writeUserError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "user": user, "entry": entry, "users": h.store.ListAdminUsers()})
}

func (h AdminUsersHandler) Ledger(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdminAccess(w, r) {
		return
	}
	username := r.PathValue("username")
	if username == "" {
		username = r.URL.Query().Get("username")
	}
	entries, err := h.store.ListCreditLedger(username)
	if err != nil {
		writeUserError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "ledger": entries})
}

func (h AdminUsersHandler) SetRole(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdminAccess(w, r) {
		return
	}
	defer r.Body.Close()
	var payload struct {
		Username string `json:"username"`
		IsAdmin  bool   `json:"isAdmin"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_JSON", "请求体不是有效 JSON")
		return
	}
	username := r.PathValue("username")
	if username == "" {
		username = payload.Username
	}
	user, err := h.store.SetAdmin(username, payload.IsAdmin)
	if err != nil {
		writeUserError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "user": user, "users": h.store.ListAdminUsers()})
}

func (h AdminUsersHandler) requireAdminAccess(w http.ResponseWriter, r *http.Request) bool {
	if h.auth == nil || !h.auth.Status().PasswordSet {
		writeError(w, http.StatusForbidden, "ADMIN_PASSWORD_NOT_SET", "请先初始化站点和 Admin 管理密码")
		return false
	}
	token := r.Header.Get("X-Admin-Token")
	if token == "" {
		token = bearerToken(r.Header.Get("Authorization"))
	}
	if h.auth.ValidateToken(token) {
		return true
	}
	if session, ok := currentUserSession(h.store, r); ok && session.User.IsAdmin {
		return true
	}
	writeError(w, http.StatusUnauthorized, "ADMIN_AUTH_REQUIRED", "需要管理员权限")
	return false
}

func (h AdminUsersHandler) adminActorFromRequest(r *http.Request) string {
	if session, ok := currentUserSession(h.store, r); ok && session.User.IsAdmin {
		return session.User.Username
	}
	return "admin"
}
