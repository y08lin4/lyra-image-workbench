package api

import (
	"net/http"

	"github.com/y08lin4/lyra-image-workbench/internal/adminauth"
	"github.com/y08lin4/lyra-image-workbench/internal/apikeys"
	"github.com/y08lin4/lyra-image-workbench/internal/billing"
	"github.com/y08lin4/lyra-image-workbench/internal/config"
	"github.com/y08lin4/lyra-image-workbench/internal/jobs"
	"github.com/y08lin4/lyra-image-workbench/internal/llm"
	"github.com/y08lin4/lyra-image-workbench/internal/output"
	"github.com/y08lin4/lyra-image-workbench/internal/promptlibrary"
	"github.com/y08lin4/lyra-image-workbench/internal/promptsquare"
	"github.com/y08lin4/lyra-image-workbench/internal/prompttools"
	"github.com/y08lin4/lyra-image-workbench/internal/settings"
	"github.com/y08lin4/lyra-image-workbench/internal/spaceconfig"
	"github.com/y08lin4/lyra-image-workbench/internal/spaces"
	"github.com/y08lin4/lyra-image-workbench/internal/uploads"
	"github.com/y08lin4/lyra-image-workbench/internal/users"
)

type Dependencies struct {
	Config        config.Config
	AdminAuth     *adminauth.Store
	Users         *users.Store
	APIKeys       *apikeys.Store
	Billing       *billing.Store
	Settings      *settings.FileStore
	Spaces        *spaces.FileStore
	SpaceConfig   *spaceconfig.Store
	Uploads       *uploads.Store
	Jobs          *jobs.Manager
	Output        *output.Store
	PromptLibrary *promptlibrary.Service
	PromptSquare  *promptsquare.Store
	PromptTools   *prompttools.Service
	LLM           *llm.Client
}

func NewRouter(deps Dependencies) http.Handler {
	mux := http.NewServeMux()
	health := NewHealthHandler(deps.Config)
	adminAuth := NewAdminAuthHandler(deps.AdminAuth, deps.Config.AdminSetupToken).WithInitialSetup(deps.Settings, deps.Users)
	adminConfig := NewAdminConfigHandler(deps.Settings, deps.AdminAuth, deps.Users)
	adminUsers := NewAdminUsersHandler(deps.Users, deps.AdminAuth)
	userHandler := NewUserHandler(deps.Users, deps.Spaces, deps.Settings)
	userConfig := NewUserConfigHandler(deps.SpaceConfig)
	developerKeyHandler := NewDeveloperAPIKeyHandler(deps.APIKeys, deps.SpaceConfig)
	statusMetadata := NewStatusMetadataHandler()
	uploadHandler := NewUploadHandler(deps.Uploads)
	taskHandler := NewTaskHandler(deps.Jobs, deps.Output, deps.Users)
	v1ImageTaskHandler := NewV1ImageTaskHandler(deps.APIKeys, deps.SpaceConfig, deps.Jobs, deps.Output, deps.Users)
	promptToolsHandler := NewPromptToolsHandler(deps.PromptTools)
	promptLibraryHandler := NewPromptLibraryHandler(deps.PromptLibrary)
	outputHandler := NewOutputHandler(deps.Output)
	promptSquareHandler := NewPromptSquareHandlerWithResults(deps.PromptSquare, deps.Jobs, deps.Output)
	billingHandler := NewBillingHandler(deps.Settings, deps.Billing, deps.Users)
	staticHandler := NewStaticHandler(deps.Config.WebDir)

	mux.HandleFunc("GET /api/health", health.ServeHTTP)
	mux.HandleFunc("GET /api/admin/auth", adminAuth.Status)
	mux.HandleFunc("POST /api/admin/auth/setup", adminAuth.Setup)
	mux.HandleFunc("POST /api/admin/auth/session", adminAuth.Login)
	mux.HandleFunc("DELETE /api/admin/auth/session", adminAuth.Logout)
	mux.HandleFunc("GET /api/admin/config", adminConfig.Get)
	mux.HandleFunc("POST /api/admin/config", adminConfig.Update)
	mux.HandleFunc("PUT /api/admin/config", adminConfig.Update)
	mux.HandleFunc("GET /api/admin/users", adminUsers.List)
	mux.HandleFunc("POST /api/admin/users/credits/add", adminUsers.AddCredits)
	mux.HandleFunc("GET /api/admin/users/{username}/ledger", adminUsers.Ledger)
	mux.HandleFunc("POST /api/admin/users/{username}/role", adminUsers.SetRole)
	mux.HandleFunc("POST /api/users/register", userHandler.Register)
	mux.HandleFunc("POST /api/users/session", userHandler.Login)
	mux.HandleFunc("GET /api/users/session", userHandler.Current)
	mux.HandleFunc("DELETE /api/users/session", userHandler.Logout)
	mux.HandleFunc("GET /api/users/profile", userHandler.Profile)
	mux.HandleFunc("PUT /api/users/profile", userHandler.UpdateProfile)
	mux.HandleFunc("GET /api/users/ledger", userHandler.Ledger)
	mux.HandleFunc("POST /api/users/credits/daily", userHandler.ClaimDailyCredits)
	mux.HandleFunc("GET /api/billing/topup/options", billingHandler.Options)
	mux.HandleFunc("POST /api/billing/epay/orders", billingHandler.CreateEpayOrder)
	mux.HandleFunc("GET /api/billing/epay/notify", billingHandler.Notify)
	mux.HandleFunc("POST /api/billing/epay/notify", billingHandler.Notify)
	mux.HandleFunc("GET /api/billing/topups", billingHandler.ListTopUps)
	mux.HandleFunc("POST /api/users/referral-code", userHandler.ReferralCode)
	mux.HandleFunc("POST /api/users/2fa/setup", userHandler.SetupTwoFactor)
	mux.HandleFunc("POST /api/users/2fa/enable", userHandler.EnableTwoFactor)
	mux.HandleFunc("POST /api/users/2fa/disable", userHandler.DisableTwoFactor)
	mux.HandleFunc("GET /api/config", userConfig.Get)
	mux.HandleFunc("POST /api/config", userConfig.Update)
	mux.HandleFunc("GET /api/developer/api-keys", developerKeyHandler.List)
	mux.HandleFunc("POST /api/developer/api-keys", developerKeyHandler.Create)
	mux.HandleFunc("DELETE /api/developer/api-keys/{id}", developerKeyHandler.Delete)
	mux.HandleFunc("GET /api/status-metadata", statusMetadata.ServeHTTP)
	mux.HandleFunc("POST /api/uploads/reference", uploadHandler.SaveReferenceImages)
	mux.HandleFunc("GET /api/uploads/reference", uploadHandler.ListReferenceImages)
	mux.HandleFunc("GET /api/uploads/reference/{id}/image", uploadHandler.ServeReferenceImage)
	mux.HandleFunc("DELETE /api/uploads/reference/{id}", uploadHandler.DeleteReferenceImage)
	mux.HandleFunc("POST /api/background-tasks", taskHandler.Create)
	mux.HandleFunc("GET /api/background-tasks", taskHandler.List)
	mux.HandleFunc("GET /api/background-tasks/{id}", taskHandler.Get)
	mux.HandleFunc("GET /api/background-tasks/{id}/events", taskHandler.Events)
	mux.HandleFunc("POST /api/background-tasks/{id}/retry", taskHandler.Retry)
	mux.HandleFunc("POST /api/background-tasks/{id}/cancel", taskHandler.Cancel)
	mux.HandleFunc("POST /api/background-tasks/{id}/favorite", taskHandler.Favorite)
	mux.HandleFunc("DELETE /api/background-tasks/{id}", taskHandler.Delete)
	mux.HandleFunc("POST /api/background-tasks/{id}/images/{index}/pixhost", taskHandler.UploadPixhost)
	mux.HandleFunc("GET /api/background-tasks/{id}/images/{index}", taskHandler.Image)
	mux.HandleFunc("GET /api/stats", taskHandler.Stats)
	mux.HandleFunc("POST /v1/images/generations", v1ImageTaskHandler.CreateGeneration)
	mux.HandleFunc("POST /v1/image-tasks", v1ImageTaskHandler.Create)
	mux.HandleFunc("GET /v1/image-tasks/{id}", v1ImageTaskHandler.Get)
	mux.HandleFunc("POST /v1/image-tasks/{id}/cancel", v1ImageTaskHandler.Cancel)
	mux.HandleFunc("GET /v1/image-tasks/{id}/images/{index}", v1ImageTaskHandler.Image)
	mux.HandleFunc("POST /api/prompt-tools/text-to-prompt", promptToolsHandler.TextToPrompt)
	mux.HandleFunc("POST /api/prompt-tools/image-to-prompt", promptToolsHandler.ImageToPrompt)
	mux.HandleFunc("POST /api/prompt-tools/sessions", promptToolsHandler.CreateSession)
	mux.HandleFunc("GET /api/prompt-tools/sessions", promptToolsHandler.Sessions)
	mux.HandleFunc("GET /api/prompt-tools/sessions/{id}", promptToolsHandler.Session)
	mux.HandleFunc("POST /api/prompt-tools/sessions/{id}/messages", promptToolsHandler.RefineSession)
	mux.HandleFunc("DELETE /api/prompt-tools/sessions/{id}", promptToolsHandler.DeleteSession)
	mux.HandleFunc("POST /api/prompt-tools/inspiration/ideas", promptToolsHandler.InspirationIdeas)
	mux.HandleFunc("POST /api/prompt-tools/inspiration/expand", promptToolsHandler.InspirationExpand)
	mux.HandleFunc("GET /api/prompt-tools/history", promptToolsHandler.History)
	mux.HandleFunc("DELETE /api/prompt-tools/history/{id}", promptToolsHandler.Delete)
	mux.HandleFunc("GET /api/prompt-library", promptLibraryHandler.List)
	mux.HandleFunc("POST /api/prompt-library/refresh", promptLibraryHandler.Refresh)
	mux.HandleFunc("GET /api/prompt-library/images/{file}", promptLibraryHandler.Image)
	mux.HandleFunc("GET /api/prompt-square/items", promptSquareHandler.List)
	mux.HandleFunc("POST /api/prompt-square/items", promptSquareHandler.Create)
	mux.HandleFunc("POST /api/prompt-square/from-result", promptSquareHandler.FromResult)
	mux.HandleFunc("POST /api/prompt-square/items/{id}/like", promptSquareHandler.Like)
	mux.HandleFunc("GET /api/prompt-square/daily", promptSquareHandler.Daily)
	mux.HandleFunc("GET /api/prompt-square/mine", promptSquareHandler.Mine)
	mux.HandleFunc("GET /api/prompt-square/images/{file}", promptSquareHandler.Image)
	mux.HandleFunc("GET /outputs/{space}/{date}/{file}", outputHandler.Serve)
	mux.HandleFunc("GET /", staticHandler.Serve)

	return withCommonHeaders(withUserAuth(deps.Users, mux))
}

func withCommonHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		next.ServeHTTP(w, r)
	})
}
