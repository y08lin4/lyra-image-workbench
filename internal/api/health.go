package api

import (
	"net/http"
	"time"

	"github.com/y08lin4/image-Workbench-Localhost-Version/internal/config"
)

type HealthHandler struct {
	cfg config.Config
}

type healthResponse struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
	Mode    string `json:"mode"`
	Time    string `json:"time"`
}

func NewHealthHandler(cfg config.Config) HealthHandler {
	return HealthHandler{cfg: cfg}
}

func (h HealthHandler) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, healthResponse{
		OK:      true,
		Message: "Local image workbench backend is ready",
		Mode:    "go-backend-owned-newapi",
		Time:    time.Now().Format(time.RFC3339),
	})
}
