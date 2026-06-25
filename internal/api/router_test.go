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
	"time"

	"github.com/y08lin4/lyra-image-workbench/internal/adminauth"
	"github.com/y08lin4/lyra-image-workbench/internal/apikeys"
	"github.com/y08lin4/lyra-image-workbench/internal/config"
	"github.com/y08lin4/lyra-image-workbench/internal/events"
	"github.com/y08lin4/lyra-image-workbench/internal/gifrender"
	"github.com/y08lin4/lyra-image-workbench/internal/jobs"
	"github.com/y08lin4/lyra-image-workbench/internal/llm"
	"github.com/y08lin4/lyra-image-workbench/internal/minimax"
	"github.com/y08lin4/lyra-image-workbench/internal/newapi"
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

func TestDeveloperAPIKeysRequireCloudUpstreamKey(t *testing.T) {
	router := newTestRouter(t)
	token := createTestSession(t, router)

	body := doJSON(t, router, http.MethodGet, "/api/developer/api-keys", token, nil)
	if !strings.Contains(body, `"apiKeys":[]`) {
		t.Fatalf("new account should have no developer keys: %s", body)
	}

	code, body := doJSONStatus(t, router, http.MethodPost, "/api/developer/api-keys", token, map[string]string{"name": "sdk"}, "")
	if code != http.StatusBadRequest || !strings.Contains(body, "UPSTREAM_KEY_REQUIRED") {
		t.Fatalf("developer key without cloud key code=%d body=%s", code, body)
	}

	rawKey := "sk-cloud-for-developer-key-123456"
	_ = doJSON(t, router, http.MethodPost, "/api/config", token, map[string]any{"apiKey": rawKey, "saveApiKeyToCloud": true})
	body = doJSON(t, router, http.MethodPost, "/api/developer/api-keys", token, map[string]string{"name": "sdk"})
	if strings.Contains(body, rawKey) || !strings.Contains(body, "lyra_sk_") {
		t.Fatalf("developer key response invalid or leaked upstream key: %s", body)
	}
	var created struct {
		APIKey struct {
			ID string `json:"id"`
		} `json:"apiKey"`
		Secret string `json:"secret"`
	}
	if err := json.Unmarshal([]byte(body), &created); err != nil {
		t.Fatalf("decode developer key response: %v body=%s", err, body)
	}
	if created.APIKey.ID == "" || !strings.HasPrefix(created.Secret, "lyra_sk_") {
		t.Fatalf("developer key id/secret missing: %s", body)
	}

	body = doJSON(t, router, http.MethodGet, "/api/developer/api-keys", token, nil)
	if strings.Contains(body, created.Secret) || !strings.Contains(body, created.APIKey.ID) {
		t.Fatalf("developer key list leaked secret or missed key: %s", body)
	}

	_ = doJSON(t, router, http.MethodDelete, "/api/developer/api-keys/"+created.APIKey.ID, token, nil)
	body = doJSON(t, router, http.MethodGet, "/api/developer/api-keys", token, nil)
	if strings.Contains(body, created.APIKey.ID) {
		t.Fatalf("deleted developer key still listed: %s", body)
	}
}

func TestV1ImageTasksUseBearerAndCloudKey(t *testing.T) {
	router := newTestRouter(t)
	token := createTestSession(t, router)

	code, body := doJSONStatus(t, router, http.MethodPost, "/v1/image-tasks", "", map[string]any{"mode": "text-to-image", "prompt": "hello"}, "")
	if code != http.StatusUnauthorized || !strings.Contains(body, "UNAUTHORIZED") {
		t.Fatalf("v1 create without bearer code=%d body=%s", code, body)
	}

	_ = doJSON(t, router, http.MethodPost, "/api/config", token, map[string]any{"apiKey": "sk-cloud-v1-1234567890", "saveApiKeyToCloud": true})
	body = doJSON(t, router, http.MethodPost, "/api/developer/api-keys", token, map[string]string{"name": "sdk"})
	var created struct {
		APIKey struct {
			ID string `json:"id"`
		} `json:"apiKey"`
		Secret string `json:"secret"`
	}
	if err := json.Unmarshal([]byte(body), &created); err != nil {
		t.Fatalf("decode developer key response: %v body=%s", err, body)
	}

	createPayload := map[string]any{
		"provider":     "image-2",
		"model":        "gpt-image-2",
		"mode":         "text-to-image",
		"prompt":       "a test image",
		"ratio":        "1:1",
		"resolution":   "standard",
		"quality":      "auto",
		"outputFormat": "png",
		"count":        1,
		"concurrency":  1,
		"apiKey":       "sk-should-not-be-used",
	}
	code, body = doJSONStatus(t, router, http.MethodPost, "/v1/image-tasks", "", createPayload, "Bearer "+created.Secret)
	if code != http.StatusOK || !strings.Contains(body, `"status":"queued"`) {
		t.Fatalf("v1 create with bearer code=%d body=%s", code, body)
	}
	if strings.Contains(body, "sk-should-not-be-used") {
		t.Fatalf("v1 response leaked runtime key: %s", body)
	}
	var taskResponse struct {
		Task struct {
			ID string `json:"id"`
		} `json:"task"`
	}
	if err := json.Unmarshal([]byte(body), &taskResponse); err != nil || taskResponse.Task.ID == "" {
		t.Fatalf("decode v1 task response error=%v body=%s", err, body)
	}

	code, body = doJSONStatus(t, router, http.MethodGet, "/v1/image-tasks/"+taskResponse.Task.ID, "", nil, "Bearer "+created.Secret)
	if code != http.StatusOK || !strings.Contains(body, taskResponse.Task.ID) {
		t.Fatalf("v1 get created task code=%d body=%s", code, body)
	}
	waitForTestTaskFinal(t, router, token, taskResponse.Task.ID)

	_ = doJSON(t, router, http.MethodPost, "/api/config", token, map[string]any{"clearCloudApiKey": true})
	code, body = doJSONStatus(t, router, http.MethodPost, "/v1/image-tasks", "", createPayload, "Bearer "+created.Secret)
	if code != http.StatusBadRequest || !strings.Contains(body, "UPSTREAM_KEY_REQUIRED") {
		t.Fatalf("v1 create after clearing cloud key code=%d body=%s", code, body)
	}

	_ = doJSON(t, router, http.MethodDelete, "/api/developer/api-keys/"+created.APIKey.ID, token, nil)
	code, body = doJSONStatus(t, router, http.MethodGet, "/v1/image-tasks/"+taskResponse.Task.ID, "", nil, "Bearer "+created.Secret)
	if code != http.StatusUnauthorized || !strings.Contains(body, "UNAUTHORIZED") {
		t.Fatalf("v1 get with deleted bearer code=%d body=%s", code, body)
	}
}

func TestV1ImageTasksRejectUnsupportedMode(t *testing.T) {
	router := newTestRouter(t)
	token := createTestSession(t, router)
	secret := createV1BearerSecret(t, router, token)

	code, body := doJSONStatus(t, router, http.MethodPost, "/v1/image-tasks", "", map[string]any{
		"provider":     "image-2",
		"model":        "gpt-image-2",
		"mode":         "image-to-image",
		"prompt":       "edit this image",
		"ratio":        "1:1",
		"resolution":   "standard",
		"quality":      "auto",
		"outputFormat": "png",
		"count":        1,
		"concurrency":  1,
	}, "Bearer "+secret)
	if code != http.StatusBadRequest || !strings.Contains(body, "TASK_CREATE_FAILED") || !strings.Contains(body, "text-to-image") {
		t.Fatalf("v1 image-to-image code=%d body=%s", code, body)
	}
}

func TestV1ImageTaskCancelMissingReturnsNotFound(t *testing.T) {
	router := newTestRouter(t)
	token := createTestSession(t, router)
	secret := createV1BearerSecret(t, router, token)

	code, body := doJSONStatus(t, router, http.MethodPost, "/v1/image-tasks/img_missing/cancel", "", nil, "Bearer "+secret)
	if code != http.StatusNotFound || !strings.Contains(body, "TASK_NOT_FOUND") {
		t.Fatalf("v1 cancel missing task code=%d body=%s", code, body)
	}
}

func TestV1ImageDownloadHidesInternalOutputPathErrors(t *testing.T) {
	env := newTestAPIEnv(t)
	token := createTestSession(t, env.Router)
	session, ok := env.Users.Current(token)
	if !ok {
		t.Fatal("test session missing")
	}
	_, secret, err := env.APIKeys.Create("sdk", "testuser01", session.StorageToken)
	if err != nil {
		t.Fatalf("APIKeys.Create() error = %v", err)
	}

	result := jobs.NewResult(0, jobs.StatusSucceeded, "")
	result.ImageURL = "/bad-output-url"
	now := time.Now()
	job := jobs.Job{
		ID:           "img_bad_output",
		SpaceToken:   session.StorageToken,
		Provider:     "image-2",
		Model:        "gpt-image-2",
		Mode:         jobs.ModeTextToImage,
		Prompt:       "bad output fixture",
		Ratio:        "1:1",
		Resolution:   "standard",
		Quality:      "auto",
		OutputFormat: "png",
		Size:         "1024x1024",
		Count:        1,
		Concurrency:  1,
		Progress:     100,
		Results:      []jobs.Result{result},
		CreatedAt:    now,
		UpdatedAt:    now,
		FinishedAt:   now,
	}
	jobs.ApplyStatus(&job, jobs.StatusSucceeded)
	jobs.ApplyStage(&job, jobs.StageSucceeded)
	if err := env.Jobs.Save(job); err != nil {
		t.Fatalf("Jobs.Save() error = %v", err)
	}

	code, body := doJSONStatus(t, env.Router, http.MethodGet, "/v1/image-tasks/img_bad_output/images/0", "", nil, "Bearer "+secret)
	if code != http.StatusNotFound || !strings.Contains(body, "TASK_IMAGE_NOT_FOUND") || strings.Contains(body, "OUTPUT_PATH_INVALID") {
		t.Fatalf("v1 bad image output code=%d body=%s", code, body)
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

func TestGIFAPIsRequireLogin(t *testing.T) {
	router := newTestRouter(t)
	for _, tc := range []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodGet, "/api/gif/status", ""},
		{http.MethodPost, "/api/gif/plans", `{}`},
		{http.MethodPost, "/api/gif-renders", `{}`},
		{http.MethodGet, "/api/gif-renders/gifrender_missing", ""},
	} {
		req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
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
	minimaxKey := "minimax-secret-1234567890"
	body := doAdminJSON(t, router, http.MethodPost, "/api/admin/config", adminToken, map[string]any{
		"newApiBaseUrl": "http://127.0.0.1:3010/v1/images/edits",
		"timeoutSec":    600,
		"debugEnabled":  true,
		"minimaxApiKey": minimaxKey,
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
	if strings.Contains(body, minimaxKey) || !strings.Contains(body, `"minimaxApiKeySet":true`) {
		t.Fatalf("admin response should only expose MiniMax key status: %s", body)
	}
}

func TestAdminUsersCanAddVideoQuota(t *testing.T) {
	router := newTestRouter(t)
	userToken := createNamedUserSession(t, router, "videoUser01", "R7!Video#Vault$2026", "")
	adminToken := createAdminToken(t, router)

	body := doAdminJSON(t, router, http.MethodPost, "/api/admin/users/video-quota", adminToken, map[string]any{
		"username": "VIDEOUSER01",
		"delta":    4,
	})
	if strings.Contains(body, "storageToken") {
		t.Fatalf("admin users response leaked storage token: %s", body)
	}
	if !strings.Contains(body, `"videoQuota":4`) {
		t.Fatalf("admin users response missing updated quota: %s", body)
	}

	body = doJSON(t, router, http.MethodGet, "/api/minimax/video-quota", userToken, nil)
	if !strings.Contains(body, `"remaining":4`) || !strings.Contains(body, `"costPerVideo":1`) {
		t.Fatalf("video quota response missing remaining quota: %s", body)
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
	Router  http.Handler
	APIKeys *apikeys.Store
	Users   *users.Store
	Spaces  *spaces.FileStore
	Jobs    *jobs.Store
	Output  *output.Store
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
	apiKeyStore, err := apikeys.NewStore(cfg.APIKeysPath())
	if err != nil {
		t.Fatalf("apikeys.NewStore() error = %v", err)
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
	llmClient := llm.NewClient()
	promptStore := prompttools.NewStore(spaceStore)
	promptService := prompttools.NewService(promptStore, settingsStore, spaceConfigStore, uploadStore, jobManager, outputStore, llmClient)
	promptLibraryService := promptlibrary.NewService(filepath.Join(cfg.DataDir, "cache", "prompt-library"))
	promptSquareStore, err := promptsquare.NewStore(cfg.DataDir)
	if err != nil {
		t.Fatalf("promptsquare.NewStore() error = %v", err)
	}
	gifService := gifrender.NewService(gifrender.NewFFmpegRenderer(gifrender.ConfigFromApp(cfg)), gifrender.NewStore(spaceStore))

	router := NewRouter(Dependencies{
		Config:        cfg,
		AdminAuth:     adminAuthStore,
		Users:         userStore,
		APIKeys:       apiKeyStore,
		Settings:      settingsStore,
		Spaces:        spaceStore,
		SpaceConfig:   spaceConfigStore,
		Uploads:       uploadStore,
		Jobs:          jobManager,
		MiniMax:       minimax.NewClient(),
		Output:        outputStore,
		PromptLibrary: promptLibraryService,
		PromptSquare:  promptSquareStore,
		PromptTools:   promptService,
		LLM:           llmClient,
		GIF:           gifService})
	return testAPIEnv{Router: router, APIKeys: apiKeyStore, Users: userStore, Spaces: spaceStore, Jobs: jobStore, Output: outputStore}
}

func createV1BearerSecret(t *testing.T, router http.Handler, token string) string {
	t.Helper()
	_ = doJSON(t, router, http.MethodPost, "/api/config", token, map[string]any{"apiKey": "sk-cloud-v1-contract-1234567890", "saveApiKeyToCloud": true})
	body := doJSON(t, router, http.MethodPost, "/api/developer/api-keys", token, map[string]string{"name": "sdk"})
	var created struct {
		Secret string `json:"secret"`
	}
	if err := json.Unmarshal([]byte(body), &created); err != nil {
		t.Fatalf("decode developer key response: %v body=%s", err, body)
	}
	if created.Secret == "" {
		t.Fatalf("developer key secret missing: %s", body)
	}
	return created.Secret
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

func waitForTestTaskFinal(t *testing.T, router http.Handler, token string, id string) {
	t.Helper()
	for i := 0; i < 80; i++ {
		body := doJSON(t, router, http.MethodGet, "/api/background-tasks/"+id, token, nil)
		for _, status := range []string{"succeeded", "partial_failed", "failed", "cancelled", "interrupted"} {
			if strings.Contains(body, `"status":"`+status+`"`) {
				return
			}
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("task %s did not reach final state", id)
}

func doJSONStatus(t *testing.T, router http.Handler, method string, path string, token string, payload any, authorization string) (int, string) {
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
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	return res.Code, res.Body.String()
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
