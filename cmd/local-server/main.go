package main

import (
	"errors"
	"log"
	"net/http"

	"github.com/y08lin4/image-Workbench-Localhost-Version/internal/api"
	"github.com/y08lin4/image-Workbench-Localhost-Version/internal/config"
	"github.com/y08lin4/image-Workbench-Localhost-Version/internal/server"
)

func main() {
	cfg := config.Load()
	router := api.NewRouter(api.Dependencies{Config: cfg})
	httpServer := server.New(cfg, router)

	log.Printf("本机生图工作台后端启动：http://%s", cfg.Addr)
	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
