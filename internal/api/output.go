package api

import (
	"net/http"

	"github.com/y08lin4/image-Workbench-Localhost-Version/internal/output"
)

type OutputHandler struct {
	store *output.Store
}

func NewOutputHandler(store *output.Store) OutputHandler {
	return OutputHandler{store: store}
}

func (h OutputHandler) Serve(w http.ResponseWriter, r *http.Request) {
	if current := r.Header.Get(userStorageTokenHeader); current == "" || current != r.PathValue("space") {
		writeError(w, http.StatusForbidden, "OUTPUT_FORBIDDEN", "没有访问该输出图片的权限")
		return
	}
	path, mime, err := h.store.Resolve(r.PathValue("space"), r.PathValue("date"), r.PathValue("file"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "OUTPUT_PATH_INVALID", err.Error())
		return
	}
	w.Header().Set("Content-Type", mime)
	w.Header().Set("Cache-Control", "private, max-age=86400")
	http.ServeFile(w, r, path)
}
