package api

import (
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/y08lin4/lyra-image-workbench/internal/spaces"
	"github.com/y08lin4/lyra-image-workbench/internal/users"
)

const userSessionCookie = "image_workbench_user_session"

type UserHandler struct {
	store  *users.Store
	spaces *spaces.FileStore
}

type userRegisterRequest struct {
	Username            string `json:"username"`
	Email               string `json:"email"`
	Password            string `json:"password"`
	ReferralCode        string `json:"referralCode"`
	LegacySpacePassword string `json:"legacySpacePassword"`
}

type userLoginRequest struct {
	Identifier    string `json:"identifier"`
	Username      string `json:"username"`
	Password      string `json:"password"`
	TwoFactorCode string `json:"twoFactorCode"`
	TOTPCode      string `json:"totpCode"`
}

type userProfileUpdateRequest struct {
	DisplayName string `json:"displayName"`
	Email       string `json:"email"`
	AvatarURL   string `json:"avatarUrl"`
}

type twoFactorCodeRequest struct {
	Code string `json:"code"`
}

func NewUserHandler(store *users.Store, spaceStore *spaces.FileStore) UserHandler {
	return UserHandler{store: store, spaces: spaceStore}
}

func (h UserHandler) Register(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var payload userRegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_JSON", "请求体不是有效 JSON")
		return
	}
	storageToken := ""
	if payload.LegacySpacePassword != "" {
		token, err := spaces.DeriveToken(payload.LegacySpacePassword)
		if err != nil {
			writeSpaceError(w, err)
			return
		}
		if _, err := h.spaces.OpenByToken(token); err != nil {
			writeSpaceError(w, err)
			return
		}
		storageToken = token
	}
	session, err := h.store.Register(payload.Username, payload.Email, payload.Password, payload.ReferralCode, storageToken)
	if err != nil {
		writeUserError(w, err)
		return
	}
	setUserSessionCookie(w, r, session)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "session": session})
}

func (h UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var payload userLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_JSON", "请求体不是有效 JSON")
		return
	}
	identifier := payload.Identifier
	if identifier == "" {
		identifier = payload.Username
	}
	twoFactorCode := payload.TwoFactorCode
	if twoFactorCode == "" {
		twoFactorCode = payload.TOTPCode
	}
	session, err := h.store.Login(identifier, payload.Password, twoFactorCode)
	if err != nil {
		writeUserError(w, err)
		return
	}
	setUserSessionCookie(w, r, session)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "session": session})
}

func (h UserHandler) Current(w http.ResponseWriter, r *http.Request) {
	session, ok := currentUserSession(h.store, r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "USER_AUTH_REQUIRED", "请先登录")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "session": session})
}

func (h UserHandler) Profile(w http.ResponseWriter, r *http.Request) {
	session, ok := currentUserSession(h.store, r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "USER_AUTH_REQUIRED", "请先登录")
		return
	}
	profile, err := h.store.Profile(session.User.Username)
	if err != nil {
		writeUserError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "user": profile})
}

func (h UserHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	session, ok := currentUserSession(h.store, r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "USER_AUTH_REQUIRED", "请先登录")
		return
	}
	var payload userProfileUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_JSON", "请求体不是有效 JSON")
		return
	}
	profile, err := h.store.UpdateProfile(session.User.Username, users.ProfileUpdate{
		DisplayName: payload.DisplayName,
		Email:       payload.Email,
		AvatarURL:   payload.AvatarURL,
	})
	if err != nil {
		writeUserError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "user": profile})
}

func (h UserHandler) Ledger(w http.ResponseWriter, r *http.Request) {
	session, ok := currentUserSession(h.store, r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "USER_AUTH_REQUIRED", "请先登录")
		return
	}
	entries, err := h.store.ListCreditLedger(session.User.Username)
	if err != nil {
		writeUserError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "ledger": entries})
}

func (h UserHandler) ReferralCode(w http.ResponseWriter, r *http.Request) {
	session, ok := currentUserSession(h.store, r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "USER_AUTH_REQUIRED", "请先登录")
		return
	}
	profile, err := h.store.EnsureReferralCode(session.User.Username)
	if err != nil {
		writeUserError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "referralCode": profile.ReferralCode, "user": profile})
}

func (h UserHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if token := userSessionToken(r); token != "" {
		h.store.Logout(token)
	}
	clearUserSessionCookie(w, r)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h UserHandler) SetupTwoFactor(w http.ResponseWriter, r *http.Request) {
	session, ok := currentUserSession(h.store, r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "USER_AUTH_REQUIRED", "请先登录")
		return
	}
	setup, err := h.store.BeginTOTPSetup(session.User.Username)
	if err != nil {
		writeUserError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "setup": setup})
}

func (h UserHandler) EnableTwoFactor(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	session, ok := currentUserSession(h.store, r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "USER_AUTH_REQUIRED", "请先登录")
		return
	}
	var payload twoFactorCodeRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_JSON", "请求体不是有效 JSON")
		return
	}
	if err := h.store.EnableTOTP(session.User.Username, payload.Code); err != nil {
		writeUserError(w, err)
		return
	}
	current, _ := currentUserSession(h.store, r)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "session": current})
}

func (h UserHandler) DisableTwoFactor(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	session, ok := currentUserSession(h.store, r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "USER_AUTH_REQUIRED", "请先登录")
		return
	}
	var payload twoFactorCodeRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_JSON", "请求体不是有效 JSON")
		return
	}
	if err := h.store.DisableTOTP(session.User.Username, payload.Code); err != nil {
		writeUserError(w, err)
		return
	}
	current, _ := currentUserSession(h.store, r)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "session": current})
}

func currentUserSession(store *users.Store, r *http.Request) (users.Session, bool) {
	if store == nil {
		return users.Session{}, false
	}
	return store.Current(userSessionToken(r))
}

func userSessionToken(r *http.Request) string {
	if cookie, err := r.Cookie(userSessionCookie); err == nil {
		return cookie.Value
	}
	return r.Header.Get("X-User-Token")
}

func setUserSessionCookie(w http.ResponseWriter, r *http.Request, session users.Session) {
	maxAge := int(users.SessionTTL.Seconds())
	http.SetCookie(w, &http.Cookie{
		Name:     userSessionCookie,
		Value:    session.Token,
		Path:     "/",
		MaxAge:   maxAge,
		Expires:  time.Now().Add(users.SessionTTL),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   r.TLS != nil,
	})
}

func clearUserSessionCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     userSessionCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   r.TLS != nil,
	})
}

func writeUserError(w http.ResponseWriter, err error) {
	status := http.StatusBadRequest
	code := "USER_ERROR"
	message := err.Error()
	var userErr users.Error
	if users.AsError(err, &userErr) {
		code = userErr.Code
		message = userErr.Chinese
		if code == "USER_LOGIN_INVALID" || code == "USER_AUTH_REQUIRED" || code == "USER_TOTP_REQUIRED" || code == "USER_TOTP_INVALID" {
			status = http.StatusUnauthorized
		}
		if code == "USER_ALREADY_EXISTS" || code == "USER_EMAIL_ALREADY_EXISTS" || code == "USER_CREDITS_SOURCE_CONFLICT" {
			status = http.StatusConflict
		}
	}
	var validationErr spaces.ValidationError
	if spaces.AsValidationError(err, &validationErr) {
		code = validationErr.Code
		message = validationErr.Chinese
	}
	if os.IsNotExist(err) {
		code = "LEGACY_SPACE_NOT_FOUND"
		message = "旧空间不存在，请确认旧空间密码是否正确"
		status = http.StatusNotFound
	}
	writeError(w, status, code, message)
}
