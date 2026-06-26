package api

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/y08lin4/lyra-image-workbench/internal/adminauth"
	"github.com/y08lin4/lyra-image-workbench/internal/spaces"
)

type AdminAuthHandler struct {
	store      *adminauth.Store
	setupToken string
}

type adminPasswordRequest struct {
	Password string `json:"password"`
}

func NewAdminAuthHandler(store *adminauth.Store, setupTokens ...string) AdminAuthHandler {
	setupToken := ""
	if len(setupTokens) > 0 {
		setupToken = strings.TrimSpace(setupTokens[0])
	}
	return AdminAuthHandler{store: store, setupToken: setupToken}
}

func (h AdminAuthHandler) Status(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "auth": h.store.Status()})
}

func (h AdminAuthHandler) Setup(w http.ResponseWriter, r *http.Request) {
	if ok, retryAfter := authAttemptAllowed("admin-setup", r, "setup"); !ok {
		writeAuthRateLimited(w, retryAfter)
		return
	}
	if !h.setupAllowed(r) {
		recordAuthAttempt("admin-setup", r, "setup", false)
		writeError(w, http.StatusForbidden, "ADMIN_SETUP_TOKEN_REQUIRED", "公网首次设置 Admin 密码需要安装令牌")
		return
	}
	defer r.Body.Close()
	var payload adminPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_JSON", "请求体不是有效 JSON")
		return
	}
	session, err := h.store.Setup(payload.Password)
	if err != nil {
		writeAdminAuthError(w, err)
		return
	}
	recordAuthAttempt("admin-setup", r, "setup", true)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "session": session, "auth": h.store.Status()})
}

func (h AdminAuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if ok, retryAfter := authAttemptAllowed("admin-login", r, "admin"); !ok {
		writeAuthRateLimited(w, retryAfter)
		return
	}
	defer r.Body.Close()
	var payload adminPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_JSON", "请求体不是有效 JSON")
		return
	}
	session, err := h.store.Login(payload.Password)
	recordAuthAttempt("admin-login", r, "admin", err == nil)
	if err != nil {
		writeAdminAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "session": session, "auth": h.store.Status()})
}

func (h AdminAuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("X-Admin-Token")
	if token == "" {
		token = bearerToken(r.Header.Get("Authorization"))
	}
	h.store.Logout(token)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func writeAdminAuthError(w http.ResponseWriter, err error) {
	status := http.StatusBadRequest
	code := "ADMIN_AUTH_ERROR"
	message := err.Error()
	var adminErr adminauth.Error
	if adminauth.AsError(err, &adminErr) {
		code = adminErr.Code
		message = adminErr.Chinese
		if code == "ADMIN_PASSWORD_ALREADY_SET" {
			status = http.StatusConflict
		}
		if code == "ADMIN_PASSWORD_INVALID" {
			status = http.StatusUnauthorized
		}
	}
	var validationErr spaces.ValidationError
	if spaces.AsValidationError(err, &validationErr) {
		code = validationErr.Code
		message = strings.ReplaceAll(validationErr.Chinese, "空间密码", "管理密码")
	}
	writeError(w, status, code, message)
}

func (h AdminAuthHandler) setupAllowed(r *http.Request) bool {
	expected := strings.TrimSpace(h.setupToken)
	if expected == "" {
		return false
	}
	supplied := strings.TrimSpace(r.Header.Get("X-Admin-Setup-Token"))
	if supplied == "" {
		supplied = bearerToken(r.Header.Get("Authorization"))
	}
	return subtle.ConstantTimeCompare([]byte(supplied), []byte(expected)) == 1
}
