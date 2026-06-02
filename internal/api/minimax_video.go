package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/y08lin4/lyra-image-workbench/internal/minimax"
	"github.com/y08lin4/lyra-image-workbench/internal/settings"
	"github.com/y08lin4/lyra-image-workbench/internal/users"
)

type MiniMaxVideoHandler struct {
	client   *minimax.Client
	settings *settings.FileStore
	users    *users.Store
}

func NewMiniMaxVideoHandler(client *minimax.Client, settingsStore *settings.FileStore, userStore *users.Store) MiniMaxVideoHandler {
	return MiniMaxVideoHandler{client: client, settings: settingsStore, users: userStore}
}

func (h MiniMaxVideoHandler) Create(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var payload minimax.CreateVideoRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_JSON", "请求体不是有效 JSON")
		return
	}
	apiKey := strings.TrimSpace(h.settings.Get().MiniMaxAPIKey)
	if apiKey == "" {
		writeError(w, http.StatusBadRequest, "MINIMAX_API_KEY_NOT_CONFIGURED", "管理员还没有配置 MiniMax API Key")
		return
	}
	username := r.Header.Get("X-User-Name")
	remaining, err := h.users.ConsumeVideoQuota(username, 1)
	if err != nil {
		writeUserError(w, err)
		return
	}
	result, err := h.client.CreateVideo(r.Context(), apiKey, payload)
	if err != nil {
		h.users.RefundVideoQuota(username, 1)
		writeError(w, http.StatusBadRequest, "MINIMAX_VIDEO_CREATE_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "task": result, "quota": map[string]any{"remaining": remaining}})
}

func (h MiniMaxVideoHandler) Query(w http.ResponseWriter, r *http.Request) {
	apiKey := strings.TrimSpace(h.settings.Get().MiniMaxAPIKey)
	if apiKey == "" {
		writeError(w, http.StatusBadRequest, "MINIMAX_API_KEY_NOT_CONFIGURED", "管理员还没有配置 MiniMax API Key")
		return
	}
	result, err := h.client.QueryVideo(r.Context(), apiKey, r.PathValue("taskID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "MINIMAX_VIDEO_QUERY_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "task": result})
}

func (h MiniMaxVideoHandler) File(w http.ResponseWriter, r *http.Request) {
	apiKey := strings.TrimSpace(h.settings.Get().MiniMaxAPIKey)
	if apiKey == "" {
		writeError(w, http.StatusBadRequest, "MINIMAX_API_KEY_NOT_CONFIGURED", "管理员还没有配置 MiniMax API Key")
		return
	}
	result, err := h.client.RetrieveFile(r.Context(), apiKey, r.PathValue("fileID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "MINIMAX_VIDEO_FILE_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "file": result})
}

func (h MiniMaxVideoHandler) Quota(w http.ResponseWriter, r *http.Request) {
	quota, err := h.users.VideoQuota(r.Header.Get("X-User-Name"))
	if err != nil {
		writeUserError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok": true,
		"quota": map[string]any{
			"remaining":        quota,
			"costPerVideo":     1,
			"minimaxApiKeySet": strings.TrimSpace(h.settings.Get().MiniMaxAPIKey) != "",
		},
	})
}
