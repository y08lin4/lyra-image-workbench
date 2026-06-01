package api

import (
	"net/http"

	"github.com/y08lin4/lyra-image-workbench/internal/statusmeta"
)

type StatusMetadataHandler struct{}

func NewStatusMetadataHandler() StatusMetadataHandler {
	return StatusMetadataHandler{}
}

func (h StatusMetadataHandler) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":       true,
		"metadata": statusmeta.All(),
	})
}
