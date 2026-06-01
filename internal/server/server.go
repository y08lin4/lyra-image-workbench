package server

import (
	"net/http"

	"github.com/y08lin4/lyra-image-workbench/internal/config"
)

func New(cfg config.Config, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:         cfg.Addr,
		Handler:      handler,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}
}
