package api

import (
	"encoding/json"
	"net/http"

	"github.com/y08lin4/lyra-image-workbench/internal/adminauth"
	"github.com/y08lin4/lyra-image-workbench/internal/settings"
	"github.com/y08lin4/lyra-image-workbench/internal/users"
)

type AdminConfigHandler struct {
	store *settings.FileStore
	auth  *adminauth.Store
	users *users.Store
}

func NewAdminConfigHandler(store *settings.FileStore, auth *adminauth.Store, userStore *users.Store) AdminConfigHandler {
	return AdminConfigHandler{store: store, auth: auth, users: userStore}
}

func (h AdminConfigHandler) Get(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":     true,
		"config": h.store.Public(),
	})
}

func (h AdminConfigHandler) Update(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}
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

func (h AdminConfigHandler) requireAdmin(w http.ResponseWriter, r *http.Request) bool {
	if adminTokenAuthorized(r, h.auth) {
		return true
	}
	if session, ok := currentUserSession(h.users, r); ok && session.User.IsAdmin {
		return true
	}
	if h.auth == nil || !h.auth.Status().PasswordSet {
		writeError(w, http.StatusForbidden, "ADMIN_PASSWORD_NOT_SET", "请先设置 Admin 管理密码")
		return false
	}
	writeError(w, http.StatusUnauthorized, "ADMIN_AUTH_REQUIRED", "需要先输入 Admin 管理密码")
	return false
}

func adminTokenAuthorized(r *http.Request, auth *adminauth.Store) bool {
	if auth == nil || !auth.Status().PasswordSet {
		return false
	}
	token := r.Header.Get("X-Admin-Token")
	if token == "" {
		token = bearerToken(r.Header.Get("Authorization"))
	}
	return auth.ValidateToken(token)
}

func bearerToken(value string) string {
	const prefix = "Bearer "
	if len(value) > len(prefix) && value[:len(prefix)] == prefix {
		return value[len(prefix):]
	}
	return ""
}
