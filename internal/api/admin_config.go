package api

import (
	"encoding/json"
	"net/http"

	"github.com/y08lin4/lyra-image-workbench/internal/adminauth"
	"github.com/y08lin4/lyra-image-workbench/internal/settings"
)

type AdminConfigHandler struct {
	store *settings.FileStore
	auth  *adminauth.Store
}

func NewAdminConfigHandler(store *settings.FileStore, auth *adminauth.Store) AdminConfigHandler {
	return AdminConfigHandler{store: store, auth: auth}
}

func (h AdminConfigHandler) Get(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r, h.auth) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":     true,
		"config": h.store.Public(),
	})
}

func (h AdminConfigHandler) Update(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r, h.auth) {
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

func requireAdmin(w http.ResponseWriter, r *http.Request, auth *adminauth.Store) bool {
	if auth == nil || !auth.Status().PasswordSet {
		writeError(w, http.StatusForbidden, "ADMIN_PASSWORD_NOT_SET", "请先设置 Admin 管理密码")
		return false
	}
	token := r.Header.Get("X-Admin-Token")
	if token == "" {
		token = bearerToken(r.Header.Get("Authorization"))
	}
	if !auth.ValidateToken(token) {
		writeError(w, http.StatusUnauthorized, "ADMIN_AUTH_REQUIRED", "需要先输入 Admin 管理密码")
		return false
	}
	return true
}

func bearerToken(value string) string {
	const prefix = "Bearer "
	if len(value) > len(prefix) && value[:len(prefix)] == prefix {
		return value[len(prefix):]
	}
	return ""
}
