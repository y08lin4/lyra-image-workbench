package api

import (
	"encoding/json"
	"net/http"

	"github.com/y08lin4/image-Workbench-Localhost-Version/internal/spaces"
)

type SpaceHandler struct {
	store *spaces.FileStore
}

type spaceSessionRequest struct {
	Password string `json:"password"`
}

func NewSpaceHandler(store *spaces.FileStore) SpaceHandler {
	return SpaceHandler{store: store}
}

func (h SpaceHandler) CreateSession(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var payload spaceSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_JSON", "请求体不是有效 JSON")
		return
	}
	session, err := h.store.CreateOrOpenByPassword(payload.Password)
	if err != nil {
		writeSpaceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"session": session,
	})
}

func (h SpaceHandler) CurrentSession(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("X-Space-Token")
	session, err := h.store.OpenByToken(token)
	if err != nil {
		writeSpaceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"session": session,
	})
}

func (h SpaceHandler) DeleteSession(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"message": "已退出个人空间；前端需要清理本地保存的空间令牌",
	})
}

func writeSpaceError(w http.ResponseWriter, err error) {
	status := http.StatusUnauthorized
	code := "SPACE_AUTH_ERROR"
	message := err.Error()
	var validationErr spaces.ValidationError
	if spaces.AsValidationError(err, &validationErr) {
		code = validationErr.Code
		message = validationErr.Chinese
		if code == "SPACE_NOT_FOUND" {
			status = http.StatusNotFound
		} else {
			status = http.StatusBadRequest
		}
	}
	writeError(w, status, code, message)
}
