package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/y08lin4/lyra-image-workbench/internal/canvas"
)

type CanvasHandler struct {
	service *canvas.Service
}

func NewCanvasHandler(service *canvas.Service) CanvasHandler {
	return CanvasHandler{service: service}
}

func (h CanvasHandler) ListProjects(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	projects, err := h.service.ListProjects(r.Header.Get(userStorageTokenHeader), limit)
	if err != nil {
		writeCanvasError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "projects": publicCanvasProjects(projects)})
}

func (h CanvasHandler) CreateProject(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var payload canvas.CreateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_JSON", "请求体不是有效 JSON")
		return
	}
	payload.OwnerUserID = r.Header.Get("X-User-Name")
	project, err := h.service.CreateProject(r.Header.Get(userStorageTokenHeader), payload)
	if err != nil {
		writeCanvasError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "project": publicCanvasProject(project)})
}

func (h CanvasHandler) GetProject(w http.ResponseWriter, r *http.Request) {
	project, ok, err := h.service.GetProject(r.Header.Get(userStorageTokenHeader), r.PathValue("projectId"))
	if err != nil {
		writeCanvasError(w, err)
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, "CANVAS_PROJECT_NOT_FOUND", "画布项目不存在")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "project": publicCanvasProject(project)})
}

func (h CanvasHandler) UpdateProject(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var payload canvas.UpdateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_JSON", "请求体不是有效 JSON")
		return
	}
	project, err := h.service.UpdateProject(r.Header.Get(userStorageTokenHeader), r.PathValue("projectId"), payload)
	if err != nil {
		writeCanvasError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "project": publicCanvasProject(project)})
}

func (h CanvasHandler) DeleteProject(w http.ResponseWriter, r *http.Request) {
	project, ok, err := h.service.DeleteProject(r.Header.Get(userStorageTokenHeader), r.PathValue("projectId"))
	if err != nil {
		writeCanvasError(w, err)
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, "CANVAS_PROJECT_NOT_FOUND", "画布项目不存在")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "project": publicCanvasProject(project)})
}

func (h CanvasHandler) CreateSnapshot(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var payload canvas.CreateSnapshotRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_JSON", "请求体不是有效 JSON")
		return
	}
	snapshot, err := h.service.CreateSnapshot(r.Header.Get(userStorageTokenHeader), r.PathValue("projectId"), payload)
	if err != nil {
		writeCanvasError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "snapshot": snapshot})
}

func publicCanvasProject(project canvas.Project) canvas.Project {
	project.SpaceToken = ""
	return project
}

func publicCanvasProjects(projects []canvas.Project) []canvas.Project {
	out := make([]canvas.Project, len(projects))
	for i := range projects {
		out[i] = publicCanvasProject(projects[i])
	}
	return out
}

func writeCanvasError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, canvas.ErrStoreNotConfigured):
		writeError(w, http.StatusServiceUnavailable, "CANVAS_STORE_NOT_CONFIGURED", "画布存储尚未初始化")
	case errors.Is(err, canvas.ErrProjectNotFound):
		writeError(w, http.StatusNotFound, "CANVAS_PROJECT_NOT_FOUND", "画布项目不存在")
	case errors.Is(err, canvas.ErrRevisionConflict):
		writeError(w, http.StatusConflict, "CANVAS_REVISION_CONFLICT", "画布已在其他位置更新，请刷新后重试")
	case errors.Is(err, canvas.ErrInvalidProject):
		writeError(w, http.StatusBadRequest, "CANVAS_PROJECT_INVALID", err.Error())
	default:
		writeError(w, http.StatusBadRequest, "CANVAS_ERROR", err.Error())
	}
}
