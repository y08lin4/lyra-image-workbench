package api

import (
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/y08lin4/lyra-image-workbench/internal/promptsquare"
)

type PromptSquareHandler struct {
	store *promptsquare.Store
}

func NewPromptSquareHandler(store *promptsquare.Store) PromptSquareHandler {
	return PromptSquareHandler{store: store}
}

func (h PromptSquareHandler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.store.List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "PROMPT_SQUARE_LIST_FAILED", "读取提示词广场失败")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "items": items})
}

func (h PromptSquareHandler) Create(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, promptsquare.MaxImageBytes+2*1024*1024)
	if err := r.ParseMultipartForm(promptsquare.MaxImageBytes + 2*1024*1024); err != nil {
		writeError(w, http.StatusBadRequest, "PROMPT_SQUARE_UPLOAD_INVALID", "上传请求无效或图片过大")
		return
	}

	var image = firstFile(r, "image")
	tags := []string{}
	if r.MultipartForm != nil && r.MultipartForm.Value != nil {
		tags = r.MultipartForm.Value["tags"]
	}
	author := strings.TrimSpace(r.FormValue("authorName"))
	if author == "" {
		author = strings.TrimSpace(r.Header.Get("X-User-Name"))
	}
	item, err := h.store.Create(promptsquare.CreateRequest{
		Title:       r.FormValue("title"),
		Prompt:      r.FormValue("prompt"),
		Negative:    r.FormValue("negativePrompt"),
		Model:       r.FormValue("model"),
		Tags:        tags,
		ImageURL:    r.FormValue("imageUrl"),
		SourceName:  r.FormValue("sourceName"),
		SourceURL:   r.FormValue("sourceUrl"),
		License:     r.FormValue("license"),
		AuthorName:  author,
		AuthorURL:   r.FormValue("authorUrl"),
		ImageHeader: image,
		Params: map[string]string{
			"ratio":        r.FormValue("ratio"),
			"resolution":   r.FormValue("resolution"),
			"quality":      r.FormValue("quality"),
			"outputFormat": r.FormValue("outputFormat"),
		},
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, "PROMPT_SQUARE_CREATE_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "item": item})
}

func (h PromptSquareHandler) Image(w http.ResponseWriter, r *http.Request) {
	path, mime, err := h.store.ResolveImage(r.PathValue("file"))
	if err != nil {
		writeError(w, http.StatusNotFound, "PROMPT_SQUARE_IMAGE_NOT_FOUND", "提示词广场图片不存在")
		return
	}
	w.Header().Set("Content-Type", mime)
	w.Header().Set("Cache-Control", "public, max-age=3600")
	http.ServeFile(w, r, path)
}

func firstFile(r *http.Request, name string) *multipart.FileHeader {
	if r.MultipartForm == nil || r.MultipartForm.File == nil {
		return nil
	}
	items := r.MultipartForm.File[name]
	if len(items) == 0 {
		return nil
	}
	return items[0]
}
