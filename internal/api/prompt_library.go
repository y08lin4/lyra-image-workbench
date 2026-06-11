package api

import (
	"net/http"
	"strconv"

	"github.com/y08lin4/lyra-image-workbench/internal/promptlibrary"
)

type PromptLibraryHandler struct {
	service *promptlibrary.Service
}

func NewPromptLibraryHandler(service *promptlibrary.Service) PromptLibraryHandler {
	return PromptLibraryHandler{service: service}
}

func (h PromptLibraryHandler) List(w http.ResponseWriter, r *http.Request) {
	if h.service == nil {
		writeError(w, http.StatusServiceUnavailable, "PROMPT_LIBRARY_DISABLED", "提示词库服务未启用")
		return
	}
	query := promptlibrary.Query{
		Lang:     r.URL.Query().Get("lang"),
		Q:        r.URL.Query().Get("q"),
		Category: r.URL.Query().Get("category"),
		Limit:    parsePromptLibraryLimit(r.URL.Query().Get("limit")),
	}
	library, err := h.service.List(r.Context(), query)
	if err != nil {
		writeError(w, http.StatusBadGateway, "PROMPT_LIBRARY_FETCH_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "library": library})
}

func (h PromptLibraryHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	if h.service == nil {
		writeError(w, http.StatusServiceUnavailable, "PROMPT_LIBRARY_DISABLED", "提示词库服务未启用")
		return
	}
	query := promptlibrary.Query{
		Lang:     r.URL.Query().Get("lang"),
		Q:        r.URL.Query().Get("q"),
		Category: r.URL.Query().Get("category"),
		Limit:    parsePromptLibraryLimit(r.URL.Query().Get("limit")),
		Force:    true,
	}
	library, err := h.service.List(r.Context(), query)
	if err != nil {
		writeError(w, http.StatusBadGateway, "PROMPT_LIBRARY_FETCH_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "library": library})
}

func parsePromptLibraryLimit(value string) int {
	if value == "" {
		return 80
	}
	limit, err := strconv.Atoi(value)
	if err != nil {
		return 80
	}
	return limit
}
