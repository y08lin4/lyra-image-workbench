package api

import (
	"encoding/json"
	"net/http"

	"github.com/y08lin4/lyra-image-workbench/internal/jobs"
)

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, code string, message string) {
	writeJSON(w, status, map[string]any{
		"ok":      false,
		"code":    code,
		"status":  status,
		"english": code,
		"chinese": message,
		"message": message,
	})
}

func writeErrorMeta(w http.ResponseWriter, status int, meta jobs.Meta, message string) {
	if meta.Code == "" {
		meta.Code = "ERROR"
	}
	if meta.English == "" {
		meta.English = meta.Code
	}
	if meta.Chinese == "" {
		meta.Chinese = message
	}
	writeJSON(w, status, map[string]any{
		"ok":      false,
		"code":    meta.Code,
		"status":  status,
		"english": meta.English,
		"chinese": meta.Chinese,
		"message": message,
	})
}
