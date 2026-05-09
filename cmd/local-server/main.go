package main

import (
	"errors"
	"log"
	"net/http"

	"github.com/y08lin4/image-Workbench-Localhost-Version/internal/adminauth"
	"github.com/y08lin4/image-Workbench-Localhost-Version/internal/api"
	"github.com/y08lin4/image-Workbench-Localhost-Version/internal/config"
	"github.com/y08lin4/image-Workbench-Localhost-Version/internal/events"
	"github.com/y08lin4/image-Workbench-Localhost-Version/internal/jobs"
	"github.com/y08lin4/image-Workbench-Localhost-Version/internal/newapi"
	"github.com/y08lin4/image-Workbench-Localhost-Version/internal/output"
	"github.com/y08lin4/image-Workbench-Localhost-Version/internal/server"
	"github.com/y08lin4/image-Workbench-Localhost-Version/internal/settings"
	"github.com/y08lin4/image-Workbench-Localhost-Version/internal/spaceconfig"
	"github.com/y08lin4/image-Workbench-Localhost-Version/internal/spaces"
	"github.com/y08lin4/image-Workbench-Localhost-Version/internal/uploads"
)

func main() {
	cfg := config.Load()
	settingsStore, err := settings.NewFileStore(cfg.RuntimeConfigPath(), settings.DefaultsFromConfig(cfg))
	if err != nil {
		log.Fatalf("加载本机配置失败：%v", err)
	}
	adminAuthStore, err := adminauth.NewStore(cfg.AdminAuthPath())
	if err != nil {
		log.Fatalf("加载 Admin 鉴权配置失败：%v", err)
	}
	spaceStore, err := spaces.NewFileStore(cfg.DataDir)
	if err != nil {
		log.Fatalf("加载个人空间存储失败：%v", err)
	}
	spaceConfigStore := spaceconfig.NewStore(spaceStore)
	uploadStore := uploads.NewStore(spaceStore)
	outputStore, err := output.NewStore("outputs")
	if err != nil {
		log.Fatalf("加载输出目录失败：%v", err)
	}
	eventHub := events.NewHub()
	jobStore := jobs.NewStore(spaceStore)
	jobManager := jobs.NewManager(jobStore, eventHub, settingsStore, spaceConfigStore, uploadStore, outputStore, newapi.NewClient())
	if err := jobManager.Recover(); err != nil {
		log.Printf("恢复任务状态失败：%v", err)
	}

	router := api.NewRouter(api.Dependencies{
		Config:      cfg,
		AdminAuth:   adminAuthStore,
		Settings:    settingsStore,
		Spaces:      spaceStore,
		SpaceConfig: spaceConfigStore,
		Uploads:     uploadStore,
		Jobs:        jobManager,
		Output:      outputStore,
	})
	httpServer := server.New(cfg, router)

	log.Printf("本机生图工作台后端启动：http://%s", cfg.Addr)
	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
