package api

import (
	"encoding/json"
	"net/http"

	"github.com/y08lin4/lyra-image-workbench/internal/minimax"
)

type MiniMaxVideoHandler struct {
	client *minimax.Client
}

func NewMiniMaxVideoHandler(client *minimax.Client) MiniMaxVideoHandler {
	return MiniMaxVideoHandler{client: client}
}

func (h MiniMaxVideoHandler) Create(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var payload minimax.CreateVideoRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_JSON", "请求体不是有效 JSON")
		return
	}
	result, err := h.client.CreateVideo(r.Context(), minimaxAPIKeyFromRequest(r), payload)
	if err != nil {
		writeError(w, http.StatusBadRequest, "MINIMAX_VIDEO_CREATE_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "task": result})
}

func (h MiniMaxVideoHandler) Query(w http.ResponseWriter, r *http.Request) {
	result, err := h.client.QueryVideo(r.Context(), minimaxAPIKeyFromRequest(r), r.PathValue("taskID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "MINIMAX_VIDEO_QUERY_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "task": result})
}

func (h MiniMaxVideoHandler) File(w http.ResponseWriter, r *http.Request) {
	result, err := h.client.RetrieveFile(r.Context(), minimaxAPIKeyFromRequest(r), r.PathValue("fileID"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "MINIMAX_VIDEO_FILE_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "file": result})
}
