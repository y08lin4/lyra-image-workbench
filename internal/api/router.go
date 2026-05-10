package api

import (
	"net/http"

	"github.com/y08lin4/image-Workbench-Localhost-Version/internal/adminauth"
	"github.com/y08lin4/image-Workbench-Localhost-Version/internal/config"
	"github.com/y08lin4/image-Workbench-Localhost-Version/internal/jobs"
	"github.com/y08lin4/image-Workbench-Localhost-Version/internal/output"
	"github.com/y08lin4/image-Workbench-Localhost-Version/internal/prompttools"
	"github.com/y08lin4/image-Workbench-Localhost-Version/internal/settings"
	"github.com/y08lin4/image-Workbench-Localhost-Version/internal/spaceconfig"
	"github.com/y08lin4/image-Workbench-Localhost-Version/internal/spaces"
	"github.com/y08lin4/image-Workbench-Localhost-Version/internal/uploads"
)

type Dependencies struct {
	Config      config.Config
	AdminAuth   *adminauth.Store
	Settings    *settings.FileStore
	Spaces      *spaces.FileStore
	SpaceConfig *spaceconfig.Store
	Uploads     *uploads.Store
	Jobs        *jobs.Manager
	Output      *output.Store
	PromptTools *prompttools.Service
}

func NewRouter(deps Dependencies) http.Handler {
	mux := http.NewServeMux()
	health := NewHealthHandler(deps.Config)
	adminAuth := NewAdminAuthHandler(deps.AdminAuth)
	adminConfig := NewAdminConfigHandler(deps.Settings, deps.AdminAuth)
	userConfig := NewUserConfigHandler(deps.SpaceConfig)
	statusMetadata := NewStatusMetadataHandler()
	spaceHandler := NewSpaceHandler(deps.Spaces)
	uploadHandler := NewUploadHandler(deps.Uploads)
	taskHandler := NewTaskHandler(deps.Jobs, deps.Output)
	promptToolsHandler := NewPromptToolsHandler(deps.PromptTools)
	outputHandler := NewOutputHandler(deps.Output)
	staticHandler := NewStaticHandler(deps.Config.WebDir)

	mux.HandleFunc("GET /api/health", health.ServeHTTP)
	mux.HandleFunc("GET /api/admin/auth", adminAuth.Status)
	mux.HandleFunc("POST /api/admin/auth/setup", adminAuth.Setup)
	mux.HandleFunc("POST /api/admin/auth/session", adminAuth.Login)
	mux.HandleFunc("DELETE /api/admin/auth/session", adminAuth.Logout)
	mux.HandleFunc("GET /api/admin/config", adminConfig.Get)
	mux.HandleFunc("POST /api/admin/config", adminConfig.Update)
	mux.HandleFunc("GET /api/config", userConfig.Get)
	mux.HandleFunc("POST /api/config", userConfig.Update)
	mux.HandleFunc("GET /api/status-metadata", statusMetadata.ServeHTTP)
	mux.HandleFunc("POST /api/spaces/session", spaceHandler.CreateSession)
	mux.HandleFunc("GET /api/spaces/session", spaceHandler.CurrentSession)
	mux.HandleFunc("DELETE /api/spaces/session", spaceHandler.DeleteSession)
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
	mux.HandleFunc("GET /outputs/{space}/{date}/{file}", outputHandler.Serve)
	mux.HandleFunc("GET /", staticHandler.Serve)

	return withCommonHeaders(mux)
}

func withCommonHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		next.ServeHTTP(w, r)
	})
}
