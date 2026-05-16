package api

import (
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/y08lin4/image-Workbench-Localhost-Version/internal/spaces"
	"github.com/y08lin4/image-Workbench-Localhost-Version/internal/users"
)

const userSessionCookie = "image_workbench_user_session"

type UserHandler struct {
	store  *users.Store
	spaces *spaces.FileStore
}

type userRegisterRequest struct {
	Username            string `json:"username"`
	Password            string `json:"password"`
	LegacySpacePassword string `json:"legacySpacePassword"`
}

type userLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
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
	session, err := h.store.Register(payload.Username, payload.Password, storageToken)
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
	session, err := h.store.Login(payload.Username, payload.Password)
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

func (h UserHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if token := userSessionToken(r); token != "" {
		h.store.Logout(token)
	}
	clearUserSessionCookie(w, r)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
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
		if code == "USER_LOGIN_INVALID" || code == "USER_AUTH_REQUIRED" {
			status = http.StatusUnauthorized
		}
		if code == "USER_ALREADY_EXISTS" {
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
