package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/y08lin4/lyra-image-workbench/internal/activitylog"
	"github.com/y08lin4/lyra-image-workbench/internal/adminauth"
	"github.com/y08lin4/lyra-image-workbench/internal/users"
)

type AdminActivityHandler struct {
	store activitylog.Reader
	auth  *adminauth.Store
	users *users.Store
}

func NewAdminActivityHandler(store activitylog.Reader, auth *adminauth.Store, userStore *users.Store) AdminActivityHandler {
	return AdminActivityHandler{store: store, auth: auth, users: userStore}
}

func (h AdminActivityHandler) List(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdminAccess(w, r) {
		return
	}
	if h.store == nil {
		writeError(w, http.StatusServiceUnavailable, "ACTIVITY_LOG_UNAVAILABLE", "活动日志服务未初始化")
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	entries := h.store.Recent(activitylog.Query{
		Limit: limit,
		Type:  activitylog.Type(strings.TrimSpace(r.URL.Query().Get("type"))),
		Level: activitylog.Level(strings.TrimSpace(r.URL.Query().Get("level"))),
	})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "activity": entries, "logs": entries})
}

func (h AdminActivityHandler) requireAdminAccess(w http.ResponseWriter, r *http.Request) bool {
	if h.auth == nil || !h.auth.Status().PasswordSet {
		writeError(w, http.StatusForbidden, "ADMIN_PASSWORD_NOT_SET", "请先初始化站点和 Admin 管理密码")
		return false
	}
	token := adminTokenFromRequest(r)
	if h.auth.ValidateToken(token) {
		return true
	}
	if session, ok := currentUserSession(h.users, r); ok && session.User.IsAdmin {
		return true
	}
	writeError(w, http.StatusUnauthorized, "ADMIN_AUTH_REQUIRED", "需要管理员权限")
	return false
}
