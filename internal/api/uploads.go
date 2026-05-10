package api

import (
	"mime/multipart"
	"net/http"

	"github.com/y08lin4/image-Workbench-Localhost-Version/internal/spaces"
	"github.com/y08lin4/image-Workbench-Localhost-Version/internal/uploads"
)

type UploadHandler struct {
	store *uploads.Store
}

func NewUploadHandler(store *uploads.Store) UploadHandler {
	return UploadHandler{store: store}
}

func (h UploadHandler) SaveReferenceImages(w http.ResponseWriter, r *http.Request) {
	spaceToken := r.Header.Get("X-Space-Token")
	r.Body = http.MaxBytesReader(w, r.Body, uploads.MaxReferenceUploadBytes)
	if err := r.ParseMultipartForm(uploads.MaxReferenceUploadBytes); err != nil {
		writeError(w, http.StatusBadRequest, "REFERENCE_UPLOAD_INVALID", "参考图上传请求无效或超过 50MB")
		return
	}

	var files []*multipart.FileHeader
	if r.MultipartForm != nil && r.MultipartForm.File != nil {
		files = append(files, r.MultipartForm.File["image"]...)
		files = append(files, r.MultipartForm.File["image[]"]...)
	}

	items, err := h.store.SaveReferenceImages(spaceToken, files)
	if err != nil {
		writeUploadError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"uploads": items,
	})
}

func (h UploadHandler) ListReferenceImages(w http.ResponseWriter, r *http.Request) {
	items, err := h.store.ListReferenceImages(r.Header.Get("X-Space-Token"))
	if err != nil {
		writeUploadError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "uploads": items})
}

func (h UploadHandler) DeleteReferenceImage(w http.ResponseWriter, r *http.Request) {
	if err := h.store.DeleteReferenceImage(r.Header.Get("X-Space-Token"), r.PathValue("id")); err != nil {
		writeUploadError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h UploadHandler) ServeReferenceImage(w http.ResponseWriter, r *http.Request) {
	item, path, err := h.store.GetReferenceImage(r.Header.Get("X-Space-Token"), r.PathValue("id"))
	if err != nil {
		writeUploadError(w, err)
		return
	}
	w.Header().Set("Content-Type", item.Mime)
	w.Header().Set("Cache-Control", "no-store")
	http.ServeFile(w, r, path)
}

func writeUploadError(w http.ResponseWriter, err error) {
	status := http.StatusBadRequest
	code := "REFERENCE_UPLOAD_ERROR"
	message := err.Error()
	var uploadErr uploads.UploadError
	if uploads.AsUploadError(err, &uploadErr) {
		code = uploadErr.Code
		message = uploadErr.Chinese
	}
	var spaceErr spaces.ValidationError
	if spaces.AsValidationError(err, &spaceErr) {
		code = spaceErr.Code
		message = spaceErr.Chinese
	}
	writeError(w, status, code, message)
}
