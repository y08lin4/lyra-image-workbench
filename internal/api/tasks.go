package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/y08lin4/image-Workbench-Localhost-Version/internal/events"
	"github.com/y08lin4/image-Workbench-Localhost-Version/internal/jobs"
	"github.com/y08lin4/image-Workbench-Localhost-Version/internal/output"
)

type TaskHandler struct {
	manager *jobs.Manager
	output  *output.Store
}

func NewTaskHandler(manager *jobs.Manager, outputStore *output.Store) TaskHandler {
	return TaskHandler{manager: manager, output: outputStore}
}

func (h TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var payload jobs.CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_JSON", "请求体不是有效 JSON")
		return
	}
	payload.RuntimeSecrets = runtimeSecretsFromRequest(r)
	job, err := h.manager.Create(r.Header.Get("X-Space-Token"), payload)
	if err != nil {
		writeError(w, http.StatusBadRequest, "TASK_CREATE_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "job": job})
}

func (h TaskHandler) List(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	items, err := h.manager.List(r.Header.Get("X-Space-Token"), limit)
	if err != nil {
		writeSpaceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "tasks": items})
}

func (h TaskHandler) Get(w http.ResponseWriter, r *http.Request) {
	job, ok, err := h.manager.Get(r.Header.Get("X-Space-Token"), r.PathValue("id"))
	if err != nil {
		writeSpaceError(w, err)
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, "TASK_NOT_FOUND", "任务不存在")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "task": job})
}

func (h TaskHandler) Retry(w http.ResponseWriter, r *http.Request) {
	job, err := h.manager.Retry(r.Header.Get("X-Space-Token"), r.PathValue("id"), runtimeSecretsFromRequest(r))
	if err != nil {
		writeError(w, http.StatusBadRequest, "TASK_RETRY_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "job": job})
}

func (h TaskHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	job, err := h.manager.Cancel(r.Header.Get("X-Space-Token"), r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "TASK_CANCEL_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "job": job})
}

func (h TaskHandler) Favorite(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var payload struct {
		Favorite bool `json:"favorite"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_JSON", "请求体不是有效 JSON")
		return
	}
	job, err := h.manager.SetFavorite(r.Header.Get("X-Space-Token"), r.PathValue("id"), payload.Favorite)
	if err != nil {
		writeError(w, http.StatusBadRequest, "TASK_FAVORITE_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "job": job})
}

func (h TaskHandler) Delete(w http.ResponseWriter, r *http.Request) {
	job, err := h.manager.Delete(r.Header.Get("X-Space-Token"), r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "TASK_DELETE_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "job": job})
}

func (h TaskHandler) UploadPixhost(w http.ResponseWriter, r *http.Request) {
	idx, err := strconv.Atoi(r.PathValue("index"))
	if err != nil || idx < 0 {
		writeError(w, http.StatusBadRequest, "TASK_IMAGE_INDEX_INVALID", "图片序号无效")
		return
	}
	job, result, err := h.manager.UploadResultToPixhost(r.Context(), r.Header.Get("X-Space-Token"), r.PathValue("id"), idx)
	if err != nil {
		writeError(w, http.StatusBadRequest, "PIXHOST_UPLOAD_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "job": job, "result": result})
}

func (h TaskHandler) Events(w http.ResponseWriter, r *http.Request) {
	spaceToken := r.Header.Get("X-Space-Token")
	jobID := r.PathValue("id")
	job, ok, err := h.manager.Get(spaceToken, jobID)
	if err != nil {
		writeSpaceError(w, err)
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, "TASK_NOT_FOUND", "任务不存在")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	flusher, _ := w.(http.Flusher)
	sendSSE(w, events.Event{Event: "snapshot", Code: "E100", English: "snapshot", Chinese: "任务快照", Data: map[string]any{"job": job}})
	if flusher != nil {
		flusher.Flush()
	}
	ch, cancel := h.manager.Subscribe(jobID)
	defer cancel()
	heartbeat := time.NewTicker(5 * time.Second)
	defer heartbeat.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-heartbeat.C:
			sendSSE(w, events.Event{Event: "heartbeat", Code: "E130", English: "heartbeat", Chinese: "心跳保活", Data: map[string]any{"time": time.Now().Format(time.RFC3339)}})
			if flusher != nil {
				flusher.Flush()
			}
		case event, ok := <-ch:
			if !ok {
				return
			}
			sendSSE(w, event)
			if flusher != nil {
				flusher.Flush()
			}
			if event.Event == "done" {
				return
			}
		}
	}
}

func (h TaskHandler) Image(w http.ResponseWriter, r *http.Request) {
	spaceToken := r.Header.Get("X-Space-Token")
	job, ok, err := h.manager.Get(spaceToken, r.PathValue("id"))
	if err != nil {
		writeSpaceError(w, err)
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, "TASK_NOT_FOUND", "任务不存在")
		return
	}
	idx, err := strconv.Atoi(r.PathValue("index"))
	if err != nil || idx < 0 || idx >= len(job.Results) || !job.Results[idx].OK {
		writeError(w, http.StatusNotFound, "TASK_IMAGE_NOT_FOUND", "任务图片不存在")
		return
	}
	serveOutputURL(w, r, h.output, job.Results[idx].ImageURL)
}

func (h TaskHandler) Stats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.manager.Stats(r.Header.Get("X-Space-Token"))
	if err != nil {
		writeSpaceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "stats": stats})
}

func sendSSE(w http.ResponseWriter, event events.Event) {
	payload, _ := json.Marshal(event)
	fmt.Fprintf(w, "event: %s\n", event.Event)
	fmt.Fprintf(w, "data: %s\n\n", payload)
}

func serveOutputURL(w http.ResponseWriter, r *http.Request, store *output.Store, url string) {
	parts := strings.Split(strings.TrimPrefix(url, "/outputs/"), "/")
	if len(parts) != 3 {
		writeError(w, http.StatusNotFound, "OUTPUT_NOT_FOUND", "输出图片不存在")
		return
	}
	path, mime, err := store.Resolve(parts[0], parts[1], parts[2])
	if err != nil {
		writeError(w, http.StatusBadRequest, "OUTPUT_PATH_INVALID", err.Error())
		return
	}
	w.Header().Set("Content-Type", mime)
	w.Header().Set("Cache-Control", "private, max-age=86400")
	http.ServeFile(w, r, path)
}
