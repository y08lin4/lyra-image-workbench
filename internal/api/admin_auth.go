package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/y08lin4/lyra-image-workbench/internal/adminauth"
	"github.com/y08lin4/lyra-image-workbench/internal/spaces"
)

type AdminAuthHandler struct {
	store *adminauth.Store
}

type adminPasswordRequest struct {
	Password string `json:"password"`
}

func NewAdminAuthHandler(store *adminauth.Store) AdminAuthHandler {
	return AdminAuthHandler{store: store}
}

func (h AdminAuthHandler) Status(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "auth": h.store.Status()})
}

func (h AdminAuthHandler) Setup(w http.ResponseWriter, r *http.Request) {
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
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "session": session, "auth": h.store.Status()})
}

func (h AdminAuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var payload adminPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_JSON", "请求体不是有效 JSON")
		return
	}
	session, err := h.store.Login(payload.Password)
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
