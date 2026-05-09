package main

import (
	"errors"
	"log"
	"net/http"

	"github.com/y08lin4/image-Workbench-Localhost-Version/internal/api"
	"github.com/y08lin4/image-Workbench-Localhost-Version/internal/config"
	"github.com/y08lin4/image-Workbench-Localhost-Version/internal/server"
	"github.com/y08lin4/image-Workbench-Localhost-Version/internal/settings"
)

func main() {
	cfg := config.Load()
	settingsStore, err := settings.NewFileStore(cfg.RuntimeConfigPath(), settings.DefaultsFromConfig(cfg))
	if err != nil {
		log.Fatalf("加载本机配置失败：%v", err)
	}

	router := api.NewRouter(api.Dependencies{Config: cfg, Settings: settingsStore})
	httpServer := server.New(cfg, router)

	log.Printf("本机生图工作台后端启动：http://%s", cfg.Addr)
	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
