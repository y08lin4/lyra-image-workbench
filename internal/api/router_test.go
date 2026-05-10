package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/y08lin4/image-Workbench-Localhost-Version/internal/adminauth"
	"github.com/y08lin4/image-Workbench-Localhost-Version/internal/config"
	"github.com/y08lin4/image-Workbench-Localhost-Version/internal/events"
	"github.com/y08lin4/image-Workbench-Localhost-Version/internal/jobs"
	"github.com/y08lin4/image-Workbench-Localhost-Version/internal/llm"
	"github.com/y08lin4/image-Workbench-Localhost-Version/internal/newapi"
	"github.com/y08lin4/image-Workbench-Localhost-Version/internal/output"
	"github.com/y08lin4/image-Workbench-Localhost-Version/internal/prompttools"
	"github.com/y08lin4/image-Workbench-Localhost-Version/internal/settings"
	"github.com/y08lin4/image-Workbench-Localhost-Version/internal/spaceconfig"
	"github.com/y08lin4/image-Workbench-Localhost-Version/internal/spaces"
	"github.com/y08lin4/image-Workbench-Localhost-Version/internal/uploads"
)

func TestConfigAPIDoesNotReturnRawAPIKey(t *testing.T) {
	router := newTestRouter(t)
	token := createTestSession(t, router)
	rawKey := "sk-router-secret-1234567890"
	rawBananaKey := "sk-router-banana-secret-0987654321"

	body := doJSON(t, router, http.MethodPost, "/api/config", token, map[string]string{"apiKey": rawKey, "bananaApiKey": rawBananaKey})
	if strings.Contains(body, rawKey) || strings.Contains(body, rawBananaKey) {
		t.Fatalf("POST /api/config leaked raw key: %s", body)
	}
	body = doJSON(t, router, http.MethodGet, "/api/config", token, nil)
	if strings.Contains(body, rawKey) || strings.Contains(body, rawBananaKey) {
		t.Fatalf("GET /api/config leaked raw key: %s", body)
	}
	if !strings.Contains(body, `"apiKeySet":true`) || !strings.Contains(body, `"bananaApiKeySet":true`) {
		t.Fatalf("GET /api/config did not report key set: %s", body)
	}
}

func TestConfigAPISavesDefaultConcurrency(t *testing.T) {
	router := newTestRouter(t)
	token := createTestSession(t, router)

	body := doJSON(t, router, http.MethodPost, "/api/config", token, map[string]any{"defaultConcurrency": 3, "autoUploadPixhost": true})
	if !strings.Contains(body, `"defaultConcurrency":3`) {
		t.Fatalf("POST /api/config did not save default concurrency: %s", body)
	}
	if !strings.Contains(body, `"autoUploadPixhost":true`) {
		t.Fatalf("POST /api/config did not save pixhost setting: %s", body)
	}
	body = doJSON(t, router, http.MethodGet, "/api/config", token, nil)
	if !strings.Contains(body, `"defaultConcurrency":3`) || !strings.Contains(body, `"autoUploadPixhost":true`) {
		t.Fatalf("GET /api/config did not return default concurrency: %s", body)
	}
}

func TestAdminConfigAPIUpdatesURLAndTimeout(t *testing.T) {
	router := newTestRouter(t)
	adminToken := createAdminToken(t, router)
	body := doAdminJSON(t, router, http.MethodPost, "/api/admin/config", adminToken, map[string]any{
		"newApiBaseUrl": "http://127.0.0.1:3010/v1/images/edits",
		"timeoutSec":    600,
	})
	if !strings.Contains(body, `"newApiBaseUrl":"http://127.0.0.1:3010/v1"`) {
		t.Fatalf("admin URL was not normalized: %s", body)
	}
	if !strings.Contains(body, `"timeoutSec":600`) || !strings.Contains(body, `"model":"gpt-image-2"`) {
		t.Fatalf("admin response missing timeout/model: %s", body)
	}
}

func TestAdminConfigRequiresPasswordSession(t *testing.T) {
	router := newTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/config", nil)
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != http.StatusForbidden {
		t.Fatalf("before setup should require initial password, code=%d body=%s", res.Code, res.Body.String())
	}

	adminToken := createAdminToken(t, router)
	body := doAdminJSON(t, router, http.MethodGet, "/api/admin/config", adminToken, nil)
	if !strings.Contains(body, `"model":"gpt-image-2"`) {
		t.Fatalf("admin config response missing model: %s", body)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/admin/config", nil)
	res = httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("after setup should require admin token, code=%d body=%s", res.Code, res.Body.String())
	}
}

func TestStaticFallbackSkipsAPIPrefixes(t *testing.T) {
	router := newTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != http.StatusOK || !strings.Contains(res.Body.String(), "test index") {
		t.Fatalf("/admin should serve SPA index, code=%d body=%s", res.Code, res.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/not-found", nil)
	res = httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != http.StatusNotFound {
		t.Fatalf("/api/not-found should not fall back to SPA, code=%d body=%s", res.Code, res.Body.String())
	}
}

func newTestRouter(t *testing.T) http.Handler {
	t.Helper()
	root := t.TempDir()
	webDir := filepath.Join(root, "web", "dist")
	if err := os.MkdirAll(webDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(webDir) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(webDir, "index.html"), []byte("test index"), 0o644); err != nil {
		t.Fatalf("WriteFile(index) error = %v", err)
	}

	cfg := config.Load()
	cfg.DataDir = filepath.Join(root, "data")
	cfg.WebDir = webDir
	cfg.BuiltinNewAPIBaseURL = config.DefaultNewAPIBaseURL
	cfg.DefaultTimeoutSec = config.DefaultTimeoutSec

	settingsStore, err := settings.NewFileStore(cfg.RuntimeConfigPath(), settings.DefaultsFromConfig(cfg))
	if err != nil {
		t.Fatalf("settings.NewFileStore() error = %v", err)
	}
	adminAuthStore, err := adminauth.NewStore(cfg.AdminAuthPath())
	if err != nil {
		t.Fatalf("adminauth.NewStore() error = %v", err)
	}
	spaceStore, err := spaces.NewFileStore(cfg.DataDir)
	if err != nil {
		t.Fatalf("spaces.NewFileStore() error = %v", err)
	}
	spaceConfigStore := spaceconfig.NewStore(spaceStore)
	uploadStore := uploads.NewStore(spaceStore)
	outputStore, err := output.NewStore(filepath.Join(root, "outputs"))
	if err != nil {
		t.Fatalf("output.NewStore() error = %v", err)
	}
	jobStore := jobs.NewStore(spaceStore)
	jobManager := jobs.NewManager(jobStore, events.NewHub(), settingsStore, spaceConfigStore, uploadStore, outputStore, newapi.NewClient())
	promptStore := prompttools.NewStore(spaceStore)
	promptService := prompttools.NewService(promptStore, settingsStore, spaceConfigStore, uploadStore, jobManager, outputStore, llm.NewClient())

	return NewRouter(Dependencies{
		Config:      cfg,
		AdminAuth:   adminAuthStore,
		Settings:    settingsStore,
		Spaces:      spaceStore,
		SpaceConfig: spaceConfigStore,
		Uploads:     uploadStore,
		Jobs:        jobManager,
		Output:      outputStore,
		PromptTools: promptService,
	})
}

func createAdminToken(t *testing.T, router http.Handler) string {
	t.Helper()
	body := doJSON(t, router, http.MethodPost, "/api/admin/auth/setup", "", map[string]string{"password": "R7!Orchid#Vault$2026"})
	var payload struct {
		Session struct {
			Token string `json:"token"`
		} `json:"session"`
	}
	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		t.Fatalf("decode admin setup response: %v body=%s", err, body)
	}
	if payload.Session.Token == "" {
		t.Fatalf("admin token missing: %s", body)
	}
	return payload.Session.Token
}

func createTestSession(t *testing.T, router http.Handler) string {
	t.Helper()
	body := doJSON(t, router, http.MethodPost, "/api/spaces/session", "", map[string]string{"password": "R7!Blue#Vault$2026"})
	var payload struct {
		Session struct {
			Token string `json:"token"`
		} `json:"session"`
	}
	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		t.Fatalf("decode session response: %v body=%s", err, body)
	}
	if payload.Session.Token == "" {
		t.Fatalf("session token missing: %s", body)
	}
	return payload.Session.Token
}

func doJSON(t *testing.T, router http.Handler, method string, path string, token string, payload any) string {
	t.Helper()
	var body bytes.Buffer
	if payload != nil {
		if err := json.NewEncoder(&body).Encode(payload); err != nil {
			t.Fatalf("encode payload: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, &body)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("X-Space-Token", token)
	}
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code < 200 || res.Code >= 300 {
		t.Fatalf("%s %s failed: code=%d body=%s", method, path, res.Code, res.Body.String())
	}
	return res.Body.String()
}

func doAdminJSON(t *testing.T, router http.Handler, method string, path string, adminToken string, payload any) string {
	t.Helper()
	var body bytes.Buffer
	if payload != nil {
		if err := json.NewEncoder(&body).Encode(payload); err != nil {
			t.Fatalf("encode payload: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, &body)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if adminToken != "" {
		req.Header.Set("X-Admin-Token", adminToken)
	}
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code < 200 || res.Code >= 300 {
		t.Fatalf("%s %s failed: code=%d body=%s", method, path, res.Code, res.Body.String())
	}
	return res.Body.String()
}
