package main

import (
	"errors"
	"log"
	"net/http"
	"path/filepath"

	"github.com/y08lin4/lyra-image-workbench/internal/adminauth"
	"github.com/y08lin4/lyra-image-workbench/internal/api"
	"github.com/y08lin4/lyra-image-workbench/internal/config"
	"github.com/y08lin4/lyra-image-workbench/internal/events"
	"github.com/y08lin4/lyra-image-workbench/internal/jobs"
	"github.com/y08lin4/lyra-image-workbench/internal/llm"
	"github.com/y08lin4/lyra-image-workbench/internal/newapi"
	"github.com/y08lin4/lyra-image-workbench/internal/output"
	"github.com/y08lin4/lyra-image-workbench/internal/promptlibrary"
	"github.com/y08lin4/lyra-image-workbench/internal/prompttools"
	"github.com/y08lin4/lyra-image-workbench/internal/server"
	"github.com/y08lin4/lyra-image-workbench/internal/settings"
	"github.com/y08lin4/lyra-image-workbench/internal/spaceconfig"
	"github.com/y08lin4/lyra-image-workbench/internal/spaces"
	"github.com/y08lin4/lyra-image-workbench/internal/uploads"
	"github.com/y08lin4/lyra-image-workbench/internal/users"
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
	userStore, err := users.NewStore(cfg.UsersPath())
	if err != nil {
		log.Fatalf("加载用户配置失败：%v", err)
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
	promptStore := prompttools.NewStore(spaceStore)
	promptService := prompttools.NewService(promptStore, settingsStore, spaceConfigStore, uploadStore, jobManager, outputStore, llm.NewClient())
	promptLibraryService := promptlibrary.NewService(filepath.Join(cfg.DataDir, "cache", "prompt-library"))
	if err := jobManager.Recover(); err != nil {
		log.Printf("恢复任务状态失败：%v", err)
	}

	router := api.NewRouter(api.Dependencies{
		Config:        cfg,
		AdminAuth:     adminAuthStore,
		Users:         userStore,
		Settings:      settingsStore,
		Spaces:        spaceStore,
		SpaceConfig:   spaceConfigStore,
		Uploads:       uploadStore,
		Jobs:          jobManager,
		Output:        outputStore,
		PromptLibrary: promptLibraryService,
		PromptTools:   promptService,
	})
	httpServer := server.New(cfg, router)

	log.Printf("Lyra Image Workbench 后端启动：http://%s", cfg.Addr)
	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
