package main

import (
	"errors"
	"log"
	"net/http"
	"path/filepath"

	"github.com/y08lin4/lyra-image-workbench/internal/adminauth"
	"github.com/y08lin4/lyra-image-workbench/internal/api"
	"github.com/y08lin4/lyra-image-workbench/internal/apikeys"
	"github.com/y08lin4/lyra-image-workbench/internal/billing"
	"github.com/y08lin4/lyra-image-workbench/internal/config"
	"github.com/y08lin4/lyra-image-workbench/internal/events"
	"github.com/y08lin4/lyra-image-workbench/internal/jobs"
	"github.com/y08lin4/lyra-image-workbench/internal/llm"
	"github.com/y08lin4/lyra-image-workbench/internal/newapi"
	"github.com/y08lin4/lyra-image-workbench/internal/output"
	"github.com/y08lin4/lyra-image-workbench/internal/promptlibrary"
	"github.com/y08lin4/lyra-image-workbench/internal/promptsquare"
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
	apiKeyStore, err := apikeys.NewStore(cfg.APIKeysPath())
	if err != nil {
		log.Fatalf("加载开发者 API Key 失败：%v", err)
	}
	billingStore, err := billing.NewStore(filepath.Join(cfg.DataDir, "topups.json"))
	if err != nil {
		log.Fatalf("加载充值订单失败：%v", err)
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
	promptSquareStore, err := promptsquare.NewStore(cfg.DataDir)
	if err != nil {
		log.Fatalf("加载提示词广场失败：%v", err)
	}
	eventHub := events.NewHub()
	jobStore := jobs.NewStore(spaceStore)
	jobManager := jobs.NewManager(jobStore, eventHub, settingsStore, spaceConfigStore, uploadStore, outputStore, newapi.NewClient())
	llmClient := llm.NewClient()
	promptStore := prompttools.NewStore(spaceStore)
	promptService := prompttools.NewService(promptStore, settingsStore, spaceConfigStore, uploadStore, jobManager, outputStore, llmClient)
	promptLibraryService := promptlibrary.NewService(filepath.Join(cfg.DataDir, "cache", "prompt-library"))
	if err := jobManager.Recover(jobs.RecoverOptions{RefundQueued: func(job jobs.Job) error {
		owner, ok := userStore.FindByStorageToken(job.SpaceToken)
		if !ok {
			return nil
		}
		_, err := userStore.RefundTaskCredits(owner.Username, job.ID, "服务重启前任务尚未提交，自动退回次数")
		if err != nil {
			var userErr users.Error
			if users.AsError(err, &userErr) && userErr.Code == "USER_TASK_CHARGE_NOT_FOUND" {
				return nil
			}
		}
		return err
	}}); err != nil {
		log.Printf("恢复任务状态失败：%v", err)
	}

	router := api.NewRouter(api.Dependencies{
		Config:        cfg,
		AdminAuth:     adminAuthStore,
		Users:         userStore,
		APIKeys:       apiKeyStore,
		Billing:       billingStore,
		Settings:      settingsStore,
		Spaces:        spaceStore,
		SpaceConfig:   spaceConfigStore,
		Uploads:       uploadStore,
		Jobs:          jobManager,
		Output:        outputStore,
		PromptLibrary: promptLibraryService,
		PromptSquare:  promptSquareStore,
		PromptTools:   promptService,
		LLM:           llmClient})
	httpServer := server.New(cfg, router)

	log.Printf("Lyra Image Workbench 后端启动：http://%s", cfg.Addr)
	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
