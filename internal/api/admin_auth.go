package api

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/y08lin4/lyra-image-workbench/internal/adminauth"
	"github.com/y08lin4/lyra-image-workbench/internal/settings"
	"github.com/y08lin4/lyra-image-workbench/internal/spaces"
	"github.com/y08lin4/lyra-image-workbench/internal/users"
)

type AdminAuthHandler struct {
	store      *adminauth.Store
	setupToken string
	settings   *settings.FileStore
	users      *users.Store
}

type adminPasswordRequest struct {
	Password string `json:"password"`
}

type adminSetupAdminRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type adminSetupRequest struct {
	Password string                 `json:"password"`
	SiteName string                 `json:"siteName"`
	Admin    adminSetupAdminRequest `json:"admin"`
	Config   settings.Update        `json:"config"`
}

type adminAuthStatusResponse struct {
	adminauth.PublicStatus
	Initialized   bool `json:"initialized"`
	SetupRequired bool `json:"setupRequired"`
}

func NewAdminAuthHandler(store *adminauth.Store, setupTokens ...string) AdminAuthHandler {
	setupToken := ""
	if len(setupTokens) > 0 {
		setupToken = strings.TrimSpace(setupTokens[0])
	}
	return AdminAuthHandler{store: store, setupToken: setupToken}
}

func (h AdminAuthHandler) WithInitialSetup(settingsStore *settings.FileStore, userStore *users.Store) AdminAuthHandler {
	h.settings = settingsStore
	h.users = userStore
	return h
}

func (h AdminAuthHandler) Status(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "auth": h.publicStatus()})
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
	var payload adminSetupRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_JSON", "请求体不是有效 JSON")
		return
	}

	password := strings.TrimSpace(payload.Admin.Password)
	if password == "" {
		password = strings.TrimSpace(payload.Password)
	}
	if err := spaces.ValidatePassword(password); err != nil {
		writeAdminAuthError(w, err)
		return
	}
	if h.store.Status().PasswordSet {
		writeAdminAuthError(w, adminauth.NewError("ADMIN_PASSWORD_ALREADY_SET", "Admin 密码已设置，请直接登录"))
		return
	}

	update := payload.Config
	if siteName := strings.TrimSpace(payload.SiteName); siteName != "" {
		update.SiteName = &siteName
	}
	fullSetup := hasAdminSetupDetails(payload)
	if fullSetup && strings.TrimSpace(payload.Admin.Username) == "" {
		writeError(w, http.StatusBadRequest, "ADMIN_USERNAME_REQUIRED", "初始化站点必须填写管理员用户名")
		return
	}
	if fullSetup && h.users == nil {
		writeError(w, http.StatusServiceUnavailable, "USER_STORE_UNAVAILABLE", "用户服务未初始化")
		return
	}
	if fullSetup && h.users.HasUsers() {
		writeError(w, http.StatusConflict, "ADMIN_SETUP_USERS_EXIST", "已有用户数据，不能覆盖初始化管理员账号")
		return
	}
	if update.HasChanges() {
		if h.settings == nil {
			writeError(w, http.StatusServiceUnavailable, "SETTINGS_UNAVAILABLE", "系统设置服务未初始化")
			return
		}
		if _, err := h.settings.Update(update); err != nil {
			writeError(w, http.StatusBadRequest, "BAD_ADMIN_CONFIG", err.Error())
			return
		}
	}

	var userSession users.Session
	var adminUser users.AdminUser
	createdAdminUser := false
	if fullSetup {
		var err error
		userSession, err = h.users.RegisterWithInitialCredits(payload.Admin.Username, payload.Admin.Email, password, "", "", 0)
		if err != nil {
			writeUserError(w, err)
			return
		}
		adminUser, err = h.users.SetAdmin(userSession.User.Username, true)
		if err != nil {
			writeUserError(w, err)
			return
		}
		createdAdminUser = true
	}

	session, err := h.store.Setup(password)
	if err != nil {
		writeAdminAuthError(w, err)
		return
	}
	if createdAdminUser {
		h.store.SetSessionActor(session.Token, adminUser.Username)
	}
	recordAuthAttempt("admin-setup", r, "setup", true)
	if createdAdminUser {
		setUserSessionCookie(w, r, userSession)
	}
	response := map[string]any{"ok": true, "session": session, "auth": h.publicStatus()}
	if h.settings != nil {
		response["config"] = h.settings.Public()
	}
	if createdAdminUser {
		response["adminUser"] = adminUser
		response["userSession"] = userSession
	}
	writeJSON(w, http.StatusOK, response)
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
	if userSession, ok := currentUserSession(h.users, r); ok && userSession.User.IsAdmin {
		h.store.SetSessionActor(session.Token, userSession.User.Username)
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "session": session, "auth": h.publicStatus()})
}

func (h AdminAuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("X-Admin-Token")
	if token == "" {
		token = bearerToken(r.Header.Get("Authorization"))
	}
	h.store.Logout(token)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h AdminAuthHandler) publicStatus() adminAuthStatusResponse {
	status := h.store.Status()
	return adminAuthStatusResponse{PublicStatus: status, Initialized: status.PasswordSet, SetupRequired: !status.PasswordSet}
}

func hasAdminSetupDetails(payload adminSetupRequest) bool {
	return strings.TrimSpace(payload.SiteName) != "" || strings.TrimSpace(payload.Admin.Username) != "" || strings.TrimSpace(payload.Admin.Email) != "" || strings.TrimSpace(payload.Admin.Password) != "" || payload.Config.HasChanges()
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
