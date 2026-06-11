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

	"github.com/y08lin4/lyra-image-workbench/internal/adminauth"
	"github.com/y08lin4/lyra-image-workbench/internal/config"
	"github.com/y08lin4/lyra-image-workbench/internal/events"
	"github.com/y08lin4/lyra-image-workbench/internal/jobs"
	"github.com/y08lin4/lyra-image-workbench/internal/llm"
	"github.com/y08lin4/lyra-image-workbench/internal/newapi"
	"github.com/y08lin4/lyra-image-workbench/internal/output"
	"github.com/y08lin4/lyra-image-workbench/internal/promptlibrary"
	"github.com/y08lin4/lyra-image-workbench/internal/prompttools"
	"github.com/y08lin4/lyra-image-workbench/internal/settings"
	"github.com/y08lin4/lyra-image-workbench/internal/spaceconfig"
	"github.com/y08lin4/lyra-image-workbench/internal/spaces"
	"github.com/y08lin4/lyra-image-workbench/internal/uploads"
	"github.com/y08lin4/lyra-image-workbench/internal/users"
)

func TestConfigAPIDoesNotPersistAPIKeys(t *testing.T) {
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
	if !strings.Contains(body, `"apiKeySet":false`) || !strings.Contains(body, `"bananaApiKeySet":false`) {
		t.Fatalf("GET /api/config should not report server-side keys: %s", body)
	}
}

func TestConfigAPIOptionallyPersistsCloudKeys(t *testing.T) {
	router := newTestRouter(t)
	token := createTestSession(t, router)
	rawKey := "sk-cloud-secret-1234567890"

	body := doJSON(t, router, http.MethodPost, "/api/config", token, map[string]any{"apiKey": rawKey, "saveApiKeyToCloud": true})
	if strings.Contains(body, rawKey) {
		t.Fatalf("POST /api/config leaked raw cloud key: %s", body)
	}
	if !strings.Contains(body, `"cloudApiKeySet":true`) || !strings.Contains(body, `"apiKeySet":true`) {
		t.Fatalf("cloud key should be reported as set: %s", body)
	}
	body = doJSON(t, router, http.MethodGet, "/api/config", token, nil)
	if strings.Contains(body, rawKey) || !strings.Contains(body, `"cloudApiKeySet":true`) {
		t.Fatalf("GET /api/config cloud key status invalid: %s", body)
	}

	body = doJSON(t, router, http.MethodPost, "/api/config", token, map[string]any{"clearCloudApiKey": true})
	if !strings.Contains(body, `"cloudApiKeySet":false`) || !strings.Contains(body, `"apiKeySet":false`) {
		t.Fatalf("cloud key should be cleared: %s", body)
	}
}

func TestUserConfigRequiresLogin(t *testing.T) {
	router := newTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("GET /api/config without login code=%d body=%s", res.Code, res.Body.String())
	}
}

func TestPromptLibraryAPIRequiresLogin(t *testing.T) {
	router := newTestRouter(t)
	for _, tc := range []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/prompt-library"},
		{http.MethodPost, "/api/prompt-library/refresh"},
	} {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		res := httptest.NewRecorder()
		router.ServeHTTP(res, req)
		if res.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s without login code=%d body=%s", tc.method, tc.path, res.Code, res.Body.String())
		}
	}
}

func TestUserSessionDoesNotExposeStorageToken(t *testing.T) {
	router := newTestRouter(t)
	body, cookies := doJSONWithCookies(t, router, http.MethodPost, "/api/users/register", "", map[string]string{
		"username": "private01",
		"password": "R7!Private#Vault$2026",
	})
	if userSessionFromCookies(t, cookies) == "" {
		t.Fatal("session cookie missing")
	}
	for _, forbidden := range []string{"storageToken", "tokenPreview", `"token"`} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("user session response leaked %s: %s", forbidden, body)
		}
	}
}

func TestConfigAPISavesDefaultCountAndConcurrency(t *testing.T) {
	router := newTestRouter(t)
	token := createTestSession(t, router)

	body := doJSON(t, router, http.MethodPost, "/api/config", token, map[string]any{"defaultCount": 4, "defaultConcurrency": 3, "autoUploadPixhost": true})
	if !strings.Contains(body, `"defaultCount":4`) {
		t.Fatalf("POST /api/config did not save default count: %s", body)
	}
	if !strings.Contains(body, `"defaultConcurrency":3`) {
		t.Fatalf("POST /api/config did not save default concurrency: %s", body)
	}
	if !strings.Contains(body, `"autoUploadPixhost":true`) {
		t.Fatalf("POST /api/config did not save pixhost setting: %s", body)
	}
	body = doJSON(t, router, http.MethodGet, "/api/config", token, nil)
	if !strings.Contains(body, `"defaultCount":4`) || !strings.Contains(body, `"defaultConcurrency":3`) || !strings.Contains(body, `"autoUploadPixhost":true`) {
		t.Fatalf("GET /api/config did not return default settings: %s", body)
	}
}

func TestUserLoginReusesAccountStorage(t *testing.T) {
	router := newTestRouter(t)
	password := "R7!Blue#Vault$2026"
	first := createNamedUserSession(t, router, "alice01", password, "")

	body := doJSON(t, router, http.MethodPost, "/api/config", first, map[string]any{"defaultCount": 4, "defaultConcurrency": 3})
	if !strings.Contains(body, `"defaultCount":4`) || !strings.Contains(body, `"defaultConcurrency":3`) {
		t.Fatalf("POST /api/config did not save account defaults: %s", body)
	}

	second := loginTestSession(t, router, "alice01", password)
	body = doJSON(t, router, http.MethodGet, "/api/config", second, nil)
	if !strings.Contains(body, `"defaultCount":4`) || !strings.Contains(body, `"defaultConcurrency":3`) {
		t.Fatalf("same user login did not reuse account storage: %s", body)
	}

	other := createNamedUserSession(t, router, "bob01", "R7!Green#Vault$2026", "")
	body = doJSON(t, router, http.MethodGet, "/api/config", other, nil)
	if strings.Contains(body, `"defaultCount":4`) || strings.Contains(body, `"defaultConcurrency":3`) {
		t.Fatalf("different user should not see alice settings: %s", body)
	}
}

func TestAdminConfigAPIUpdatesURLAndTimeout(t *testing.T) {
	router := newTestRouter(t)
	adminToken := createAdminToken(t, router)
	body := doAdminJSON(t, router, http.MethodPost, "/api/admin/config", adminToken, map[string]any{
		"newApiBaseUrl": "http://127.0.0.1:3010/v1/images/edits",
		"timeoutSec":    600,
		"debugEnabled":  true,
	})
	if !strings.Contains(body, `"newApiBaseUrl":"http://127.0.0.1:3010/v1"`) {
		t.Fatalf("admin URL was not normalized: %s", body)
	}
	if !strings.Contains(body, `"timeoutSec":600`) || !strings.Contains(body, `"model":"gpt-image-2"`) {
		t.Fatalf("admin response missing timeout/model: %s", body)
	}
	if !strings.Contains(body, `"debugEnabled":true`) {
		t.Fatalf("admin response missing debug flag: %s", body)
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

func TestOutputRouteRequiresAuthenticatedOwner(t *testing.T) {
	env := newTestAPIEnv(t)
	legacyPassword := "R7!Legacy#Vault$2026"
	legacy, err := env.Spaces.CreateOrOpenByPassword(legacyPassword)
	if err != nil {
		t.Fatalf("CreateOrOpenByPassword() error = %v", err)
	}
	saved, err := env.Output.Save(legacy.Token, "img_owner", 0, []byte("owner-image"), "image/png")
	if err != nil {
		t.Fatalf("output.Save() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, saved.URL, nil)
	res := httptest.NewRecorder()
	env.Router.ServeHTTP(res, req)
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated output code=%d body=%s", res.Code, res.Body.String())
	}

	owner := createNamedUserSession(t, env.Router, "owner01", "R7!Owner#Vault$2026", legacyPassword)
	req = httptest.NewRequest(http.MethodGet, saved.URL, nil)
	req.AddCookie(&http.Cookie{Name: userSessionCookie, Value: owner})
	res = httptest.NewRecorder()
	env.Router.ServeHTTP(res, req)
	if res.Code != http.StatusOK || res.Body.String() != "owner-image" {
		t.Fatalf("owner output code=%d body=%s", res.Code, res.Body.String())
	}

	intruder := createNamedUserSession(t, env.Router, "intruder01", "R7!Intruder#Vault$2026", "")
	req = httptest.NewRequest(http.MethodGet, saved.URL, nil)
	req.AddCookie(&http.Cookie{Name: userSessionCookie, Value: intruder})
	res = httptest.NewRecorder()
	env.Router.ServeHTTP(res, req)
	if res.Code != http.StatusForbidden {
		t.Fatalf("intruder output code=%d body=%s", res.Code, res.Body.String())
	}
}

type testAPIEnv struct {
	Router http.Handler
	Spaces *spaces.FileStore
	Output *output.Store
}

func newTestRouter(t *testing.T) http.Handler {
	return newTestAPIEnv(t).Router
}

func newTestAPIEnv(t *testing.T) testAPIEnv {
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
	userStore, err := users.NewStore(cfg.UsersPath())
	if err != nil {
		t.Fatalf("users.NewStore() error = %v", err)
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
	promptLibraryService := promptlibrary.NewService(filepath.Join(cfg.DataDir, "cache", "prompt-library"))

	router := NewRouter(Dependencies{
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
	return testAPIEnv{Router: router, Spaces: spaceStore, Output: outputStore}
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
	return createNamedUserSession(t, router, "testuser01", "R7!Blue#Vault$2026", "")
}

func createNamedUserSession(t *testing.T, router http.Handler, username string, password string, legacySpacePassword string) string {
	t.Helper()
	_, cookies := doJSONWithCookies(t, router, http.MethodPost, "/api/users/register", "", map[string]string{
		"username":            username,
		"password":            password,
		"legacySpacePassword": legacySpacePassword,
	})
	return userSessionFromCookies(t, cookies)
}

func loginTestSession(t *testing.T, router http.Handler, username string, password string) string {
	t.Helper()
	_, cookies := doJSONWithCookies(t, router, http.MethodPost, "/api/users/session", "", map[string]string{
		"username": username,
		"password": password,
	})
	return userSessionFromCookies(t, cookies)
}

func userSessionFromCookies(t *testing.T, cookies []*http.Cookie) string {
	t.Helper()
	for _, cookie := range cookies {
		if cookie.Name == userSessionCookie && cookie.Value != "" {
			return cookie.Value
		}
	}
	t.Fatalf("%s cookie missing", userSessionCookie)
	return ""
}

func doJSON(t *testing.T, router http.Handler, method string, path string, token string, payload any) string {
	t.Helper()
	body, _ := doJSONWithCookies(t, router, method, path, token, payload)
	return body
}

func doJSONWithCookies(t *testing.T, router http.Handler, method string, path string, token string, payload any) (string, []*http.Cookie) {
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
		req.AddCookie(&http.Cookie{Name: userSessionCookie, Value: token})
	}
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code < 200 || res.Code >= 300 {
		t.Fatalf("%s %s failed: code=%d body=%s", method, path, res.Code, res.Body.String())
	}
	return res.Body.String(), res.Result().Cookies()
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
