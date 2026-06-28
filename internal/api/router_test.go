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

func TestUserConfigShowsSystemKeyStatusWithoutPreview(t *testing.T) {
	env := newTestAPIEnv(t)
	token := createTestSession(t, env.Router)
	imageKey := "sk-system-config-preview-hidden-123456"
	bananaKey := "sk-system-banana-preview-hidden-123456"
	if _, err := env.Settings.Update(settings.Update{SystemAPIKey: &imageKey, SystemBananaAPIKey: &bananaKey}); err != nil {
		t.Fatalf("settings.Update() system keys error = %v", err)
	}

	body := doJSON(t, env.Router, http.MethodGet, "/api/config", token, nil)
	if !strings.Contains(body, `"systemApiKeySet":true`) || !strings.Contains(body, `"systemBananaApiKeySet":true`) {
		t.Fatalf("user config missing system key status: %s", body)
	}
	if strings.Contains(body, imageKey) || strings.Contains(body, bananaKey) || strings.Contains(body, "systemApiKeyPreview") || strings.Contains(body, "systemBananaApiKeyPreview") {
		t.Fatalf("user config leaked system key preview: %s", body)
	}
}

func TestDeveloperAPIKeysAllowSystemUpstreamKey(t *testing.T) {
	env := newTestAPIEnv(t)
	token := createTestSession(t, env.Router)
	systemKey := "sk-system-for-developer-key-123456"
	if _, err := env.Settings.Update(settings.Update{SystemAPIKey: &systemKey}); err != nil {
		t.Fatalf("settings.Update() system key error = %v", err)
	}

	body := doJSON(t, env.Router, http.MethodPost, "/api/developer/api-keys", token, map[string]string{"name": "system sdk"})
	if strings.Contains(body, systemKey) || !strings.Contains(body, "lyra_sk_") {
		t.Fatalf("developer key with system upstream invalid or leaked upstream key: %s", body)
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
	grantTestCredits(t, router, token, "testuser01", 3)
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
	env := newTestAPIEnv(t)
	router := env.Router
	token := createTestSession(t, router)

	code, body := doJSONStatus(t, router, http.MethodPost, "/v1/image-tasks", "", map[string]any{"mode": "text-to-image", "prompt": "hello"}, "")
	if code != http.StatusUnauthorized || !strings.Contains(body, "UNAUTHORIZED") {
		t.Fatalf("v1 create without bearer code=%d body=%s", code, body)
	}

	_ = doJSON(t, router, http.MethodPost, "/api/config", token, map[string]any{"apiKey": "sk-cloud-v1-1234567890", "saveApiKeyToCloud": true})
	grantTestCredits(t, router, token, "testuser01", 3)
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
		Task            jobs.Job `json:"task"`
		TaskID          string   `json:"taskId"`
		ConsumedCredits int      `json:"consumedCredits"`
	}
	if err := json.Unmarshal([]byte(body), &taskResponse); err != nil || taskResponse.Task.ID == "" {
		t.Fatalf("decode v1 task response error=%v body=%s", err, body)
	}
	if taskResponse.TaskID != taskResponse.Task.ID || taskResponse.Task.ConsumedCredits != 0 || taskResponse.ConsumedCredits != 0 {
		t.Fatalf("v1 personal-key task should not consume credits: %+v", taskResponse)
	}
	requireUserCredits(t, env, "testuser01", 3)
	requireNoTaskCharge(t, env, "testuser01", taskResponse.Task.ID)

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
	requireUserCredits(t, env, "testuser01", 3)

	systemKey := "sk-system-v1-charge-1234567890"
	if _, err := env.Settings.Update(settings.Update{SystemAPIKey: &systemKey}); err != nil {
		t.Fatalf("settings.Update() system key error = %v", err)
	}
	code, body = doJSONStatus(t, router, http.MethodPost, "/v1/image-tasks", "", createPayload, "Bearer "+created.Secret)
	if code != http.StatusOK || !strings.Contains(body, `"status":"queued"`) {
		t.Fatalf("v1 create with system key code=%d body=%s", code, body)
	}
	var systemTaskResponse struct {
		Task            jobs.Job `json:"task"`
		TaskID          string   `json:"taskId"`
		ConsumedCredits int      `json:"consumedCredits"`
	}
	if err := json.Unmarshal([]byte(body), &systemTaskResponse); err != nil || systemTaskResponse.Task.ID == "" {
		t.Fatalf("decode v1 system-key task response error=%v body=%s", err, body)
	}
	if systemTaskResponse.Task.ConsumedCredits != 1 || systemTaskResponse.ConsumedCredits != 1 {
		t.Fatalf("v1 system-key task should consume credits: %+v", systemTaskResponse)
	}
	requireUserCredits(t, env, "testuser01", 2)
	requireLatestTaskCharge(t, env, "testuser01", systemTaskResponse.Task.ID, -1, 2)
	waitForTestTaskFinal(t, router, token, systemTaskResponse.Task.ID)

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

func TestTaskAPIsScopeHistoryStatusAndHideInternalFields(t *testing.T) {
	env := newTestAPIEnv(t)
	ownerToken := createNamedUserSession(t, env.Router, "taskowner01", "R7!Owner#Vault$2026", "")
	ownerSession, ok := env.Users.Current(ownerToken)
	if !ok {
		t.Fatal("owner session missing")
	}
	intruderToken := createNamedUserSession(t, env.Router, "taskintruder01", "R7!Intruder#Vault$2026", "")
	intruderSession, ok := env.Users.Current(intruderToken)
	if !ok {
		t.Fatal("intruder session missing")
	}
	_, ownerSecret, err := env.APIKeys.Create("owner sdk", ownerSession.User.Username, ownerSession.StorageToken)
	if err != nil {
		t.Fatalf("owner APIKeys.Create() error = %v", err)
	}
	_, intruderSecret, err := env.APIKeys.Create("intruder sdk", intruderSession.User.Username, intruderSession.StorageToken)
	if err != nil {
		t.Fatalf("intruder APIKeys.Create() error = %v", err)
	}

	now := time.Now().UTC()
	olderOwnerJob := seededRouterTask(ownerSession.StorageToken, "img_history_old", jobs.StatusSucceeded, jobs.StageSucceeded, now.Add(-time.Hour))
	result := jobs.NewResult(0, jobs.StatusSucceeded, "")
	result.ImageURL = "/internal/should-be-rewritten"
	result.OutputDate = "2026-06-26"
	result.OutputFileName = "private.png"
	olderOwnerJob.Results = []jobs.Result{result}
	if err := env.Jobs.Save(olderOwnerJob); err != nil {
		t.Fatalf("Save(older owner job) error = %v", err)
	}
	newerOwnerJob := seededRouterTask(ownerSession.StorageToken, "img_history_new", jobs.StatusRunning, jobs.StageWaitingUpstream, now)
	if err := env.Jobs.Save(newerOwnerJob); err != nil {
		t.Fatalf("Save(newer owner job) error = %v", err)
	}
	intruderJob := seededRouterTask(intruderSession.StorageToken, "img_history_intruder", jobs.StatusFailed, jobs.StageFailed, now.Add(time.Minute))
	if err := env.Jobs.Save(intruderJob); err != nil {
		t.Fatalf("Save(intruder job) error = %v", err)
	}

	historyBody := doJSON(t, env.Router, http.MethodGet, "/api/background-tasks?limit=10", ownerToken, nil)
	if strings.Contains(historyBody, intruderJob.ID) || strings.Contains(historyBody, "spaceToken") || strings.Contains(historyBody, "outputDate") || strings.Contains(historyBody, "outputFileName") {
		t.Fatalf("history response leaked another space or internal fields: %s", historyBody)
	}
	var history struct {
		Tasks []jobs.Job `json:"tasks"`
	}
	if err := json.Unmarshal([]byte(historyBody), &history); err != nil {
		t.Fatalf("decode history response: %v body=%s", err, historyBody)
	}
	if len(history.Tasks) != 2 || history.Tasks[0].ID != newerOwnerJob.ID || history.Tasks[1].ID != olderOwnerJob.ID {
		t.Fatalf("history order/items mismatch: %+v", history.Tasks)
	}
	if history.Tasks[0].Status != jobs.StatusRunning || history.Tasks[0].StatusCode != "J200" || history.Tasks[0].Stage != jobs.StageWaitingUpstream || history.Tasks[0].StageCode != "S130" {
		t.Fatalf("running task status metadata missing: %+v", history.Tasks[0])
	}
	if len(history.Tasks[1].Results) != 1 || history.Tasks[1].Results[0].ImageURL != "/api/background-tasks/img_history_old/images/0" {
		t.Fatalf("internal history result URL not rewritten: %+v", history.Tasks[1].Results)
	}

	code, body := doJSONStatus(t, env.Router, http.MethodGet, "/api/background-tasks/"+intruderJob.ID, ownerToken, nil, "")
	if code != http.StatusNotFound || !strings.Contains(body, "TASK_NOT_FOUND") {
		t.Fatalf("owner should not see intruder task via internal API, code=%d body=%s", code, body)
	}
	code, body = doJSONStatus(t, env.Router, http.MethodGet, "/v1/image-tasks/"+olderOwnerJob.ID, "", nil, "Bearer "+ownerSecret)
	if code != http.StatusOK || !strings.Contains(body, `"/v1/image-tasks/img_history_old/images/0"`) || strings.Contains(body, "/api/background-tasks/") || strings.Contains(body, "outputDate") {
		t.Fatalf("owner v1 get should expose v1 image URL only, code=%d body=%s", code, body)
	}
	code, body = doJSONStatus(t, env.Router, http.MethodGet, "/v1/image-tasks/"+olderOwnerJob.ID, "", nil, "Bearer "+intruderSecret)
	if code != http.StatusNotFound || !strings.Contains(body, "TASK_NOT_FOUND") {
		t.Fatalf("intruder should not see owner task via v1 API, code=%d body=%s", code, body)
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

func TestGIFAPIsAreRemoved(t *testing.T) {
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
		if tc.body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		res := httptest.NewRecorder()
		router.ServeHTTP(res, req)
		if res.Code != http.StatusNotFound && res.Code != http.StatusMethodNotAllowed {
			t.Fatalf("%s %s should be removed, code=%d body=%s", tc.method, tc.path, res.Code, res.Body.String())
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
	if strings.Contains(strings.ToLower(body), "mini"+"max") {
		t.Fatalf("admin config response should not include removed video config: %s", body)
	}
}

func TestFreeCreditsConfigGrantsRegistrationAndDailyClaim(t *testing.T) {
	env := newTestAPIEnv(t)
	adminToken := createAdminToken(t, env.Router)
	body := doAdminJSON(t, env.Router, http.MethodPut, "/api/admin/config", adminToken, map[string]any{
		"newUserInitialCredits": 7,
		"dailyFreeCredits":      2,
	})
	if !strings.Contains(body, `"newUserInitialCredits":7`) || !strings.Contains(body, `"dailyFreeCredits":2`) {
		t.Fatalf("admin config response missing free credit settings: %s", body)
	}

	body, cookies := doJSONWithCookies(t, env.Router, http.MethodPost, "/api/users/register", "", map[string]string{
		"username": "freeUser01",
		"password": "R7!Free#Vault$2026",
	})
	token := userSessionFromCookies(t, cookies)
	var registered struct {
		Session users.Session `json:"session"`
	}
	if err := json.Unmarshal([]byte(body), &registered); err != nil {
		t.Fatalf("decode register response: %v body=%s", err, body)
	}
	if registered.Session.User.CreditsBalance != 7 {
		t.Fatalf("registered creditsBalance=%d, want 7 body=%s", registered.Session.User.CreditsBalance, body)
	}

	body = doJSON(t, env.Router, http.MethodGet, "/api/users/ledger", token, nil)
	if !strings.Contains(body, `"type":"initial_free"`) || !strings.Contains(body, `"balanceAfter":7`) {
		t.Fatalf("initial free credit ledger missing: %s", body)
	}

	body = doJSON(t, env.Router, http.MethodPost, "/api/users/credits/daily", token, nil)
	var firstClaim struct {
		Claimed        bool                    `json:"claimed"`
		AlreadyClaimed bool                    `json:"alreadyClaimed"`
		Amount         int                     `json:"amount"`
		User           users.PublicUser        `json:"user"`
		Entry          users.CreditLedgerEntry `json:"entry"`
	}
	if err := json.Unmarshal([]byte(body), &firstClaim); err != nil {
		t.Fatalf("decode first daily claim: %v body=%s", err, body)
	}
	if !firstClaim.Claimed || firstClaim.AlreadyClaimed || firstClaim.Amount != 2 || firstClaim.User.CreditsBalance != 9 || firstClaim.Entry.Type != "daily_free" || firstClaim.Entry.BalanceAfter != 9 {
		t.Fatalf("first daily claim mismatch: %+v body=%s", firstClaim, body)
	}

	body = doJSON(t, env.Router, http.MethodPost, "/api/users/credits/daily", token, nil)
	var duplicateClaim struct {
		Claimed        bool                    `json:"claimed"`
		AlreadyClaimed bool                    `json:"alreadyClaimed"`
		User           users.PublicUser        `json:"user"`
		Entry          users.CreditLedgerEntry `json:"entry"`
	}
	if err := json.Unmarshal([]byte(body), &duplicateClaim); err != nil {
		t.Fatalf("decode duplicate daily claim: %v body=%s", err, body)
	}
	if duplicateClaim.Claimed || !duplicateClaim.AlreadyClaimed || duplicateClaim.User.CreditsBalance != 9 || duplicateClaim.Entry.ID != firstClaim.Entry.ID {
		t.Fatalf("duplicate daily claim should not credit again: %+v body=%s", duplicateClaim, body)
	}
}
func TestAdminUsersCanAddCreditsAndReadLedger(t *testing.T) {
	router := newTestRouter(t)
	createNamedUserSession(t, router, "creditUser01", "R7!Credit#Vault$2026", "")
	adminToken := createAdminToken(t, router)

	body := doAdminJSON(t, router, http.MethodPost, "/api/admin/users/credits/add", adminToken, map[string]any{
		"username": "CREDITUSER01",
		"amount":   4,
		"reason":   "test credit grant",
	})
	if strings.Contains(body, "storageToken") {
		t.Fatalf("admin users response leaked storage token: %s", body)
	}
	if !strings.Contains(body, `"creditsBalance":4`) {
		t.Fatalf("admin users response missing updated credits: %s", body)
	}

	body = doAdminJSON(t, router, http.MethodGet, "/api/admin/users/creditUser01/ledger", adminToken, nil)
	if !strings.Contains(body, `"balanceAfter":4`) || !strings.Contains(body, `"reason":"test credit grant"`) {
		t.Fatalf("admin ledger response missing credit entry: %s", body)
	}
}

func TestFullAdminSetupAttributesCreditGrantToAdminUser(t *testing.T) {
	env := newTestAPIEnv(t)
	setupPayload := map[string]any{
		"siteName": "Lyra Test",
		"admin": map[string]string{
			"username": "rootAdmin01",
			"email":    "root@example.com",
			"password": "R7!Root#Vault$2026",
		},
		"config": map[string]any{
			"newUserInitialCredits": 3,
			"dailyFreeCredits":      1,
		},
	}
	var setupBody bytes.Buffer
	if err := json.NewEncoder(&setupBody).Encode(setupPayload); err != nil {
		t.Fatalf("encode full setup payload: %v", err)
	}
	setupReq := httptest.NewRequest(http.MethodPost, "/api/admin/auth/setup", &setupBody)
	setupReq.Header.Set("Content-Type", "application/json")
	setupReq.Header.Set("X-Admin-Setup-Token", "test-admin-setup-token")
	setupRes := httptest.NewRecorder()
	env.Router.ServeHTTP(setupRes, setupReq)
	if setupRes.Code != http.StatusOK {
		t.Fatalf("full admin setup failed: code=%d body=%s", setupRes.Code, setupRes.Body.String())
	}
	var setupResponse struct {
		Session   adminauth.Session `json:"session"`
		AdminUser users.AdminUser   `json:"adminUser"`
	}
	if err := json.Unmarshal(setupRes.Body.Bytes(), &setupResponse); err != nil {
		t.Fatalf("decode full setup response: %v body=%s", err, setupRes.Body.String())
	}
	if setupResponse.Session.Token == "" || setupResponse.AdminUser.Username != "rootAdmin01" || !setupResponse.AdminUser.IsAdmin {
		t.Fatalf("full setup response missing admin session/user: %+v", setupResponse)
	}

	body, cookies := doJSONWithCookies(t, env.Router, http.MethodPost, "/api/users/register", "", map[string]string{
		"username": "creditTarget01",
		"password": "R7!Target#Vault$2026",
	})
	targetToken := userSessionFromCookies(t, cookies)
	var registered struct {
		Session users.Session `json:"session"`
	}
	if err := json.Unmarshal([]byte(body), &registered); err != nil {
		t.Fatalf("decode target register response: %v body=%s", err, body)
	}
	if registered.Session.User.CreditsBalance != 3 {
		t.Fatalf("target initial credits = %d, want 3", registered.Session.User.CreditsBalance)
	}

	grantBody := doAdminJSON(t, env.Router, http.MethodPost, "/api/admin/users/credits/add", setupResponse.Session.Token, map[string]any{
		"username": "creditTarget01",
		"amount":   4,
		"reason":   "manual back office grant",
	})
	var grantResponse struct {
		User  users.AdminUser         `json:"user"`
		Entry users.CreditLedgerEntry `json:"entry"`
	}
	if err := json.Unmarshal([]byte(grantBody), &grantResponse); err != nil {
		t.Fatalf("decode grant response: %v body=%s", err, grantBody)
	}
	if grantResponse.User.CreditsBalance != 7 || grantResponse.Entry.AdminActor != "rootAdmin01" || grantResponse.Entry.Reason != "manual back office grant" {
		t.Fatalf("grant response did not attribute admin actor: %+v body=%s", grantResponse, grantBody)
	}

	ledgerBody := doJSON(t, env.Router, http.MethodGet, "/api/users/ledger", targetToken, nil)
	if !strings.Contains(ledgerBody, `"adminActor":"rootAdmin01"`) || !strings.Contains(ledgerBody, `"type":"admin_add"`) {
		t.Fatalf("user ledger missing attributed admin grant: %s", ledgerBody)
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

func TestAdminSetupRequiresTokenOnPublicHost(t *testing.T) {
	router := newTestRouter(t)
	payload := map[string]string{"password": "R7!Orchid#Vault$2026"}

	var firstBody bytes.Buffer
	if err := json.NewEncoder(&firstBody).Encode(payload); err != nil {
		t.Fatalf("encode setup payload: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "https://image.example.com/api/admin/auth/setup", &firstBody)
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != http.StatusForbidden || !strings.Contains(res.Body.String(), "ADMIN_SETUP_TOKEN_REQUIRED") {
		t.Fatalf("setup without token code=%d body=%s", res.Code, res.Body.String())
	}

	var secondBody bytes.Buffer
	if err := json.NewEncoder(&secondBody).Encode(payload); err != nil {
		t.Fatalf("encode setup payload: %v", err)
	}
	req = httptest.NewRequest(http.MethodPost, "https://image.example.com/api/admin/auth/setup", &secondBody)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Admin-Setup-Token", "test-admin-setup-token")
	res = httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != http.StatusOK || !strings.Contains(res.Body.String(), "token") {
		t.Fatalf("setup with token code=%d body=%s", res.Code, res.Body.String())
	}
}

func TestAdminSetupRequiresTokenOnLoopback(t *testing.T) {
	store, err := adminauth.NewStore(filepath.Join(t.TempDir(), "admin-auth.json"))
	if err != nil {
		t.Fatalf("adminauth.NewStore() error = %v", err)
	}
	handler := NewAdminAuthHandler(store)
	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(map[string]string{"password": "R7!Orchid#Vault$2026"}); err != nil {
		t.Fatalf("encode setup payload: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "http://localhost/api/admin/auth/setup", &body)
	req.RemoteAddr = "127.0.0.1:51000"
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://localhost:5173")
	res := httptest.NewRecorder()
	handler.Setup(res, req)
	if res.Code != http.StatusForbidden || !strings.Contains(res.Body.String(), "ADMIN_SETUP_TOKEN_REQUIRED") {
		t.Fatalf("loopback setup without install token should be rejected, code=%d body=%s", res.Code, res.Body.String())
	}
}

func TestAdminSetupInvalidTokenRateLimited(t *testing.T) {
	router := newTestRouter(t)
	payload := map[string]string{"password": "R7!Orchid#Vault$2026"}

	for i := 0; i < 20; i++ {
		var body bytes.Buffer
		if err := json.NewEncoder(&body).Encode(payload); err != nil {
			t.Fatalf("encode setup payload: %v", err)
		}
		req := httptest.NewRequest(http.MethodPost, "https://image.example.com/api/admin/auth/setup", &body)
		req.RemoteAddr = "198.51.100.72:51004"
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Admin-Setup-Token", "wrong-token-"+stringInt(i))
		res := httptest.NewRecorder()
		router.ServeHTTP(res, req)
		if res.Code != http.StatusForbidden || !strings.Contains(res.Body.String(), "ADMIN_SETUP_TOKEN_REQUIRED") {
			t.Fatalf("invalid setup token attempt %d code=%d body=%s", i+1, res.Code, res.Body.String())
		}
	}

	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(payload); err != nil {
		t.Fatalf("encode setup payload: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "https://image.example.com/api/admin/auth/setup", &body)
	req.RemoteAddr = "198.51.100.72:51004"
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Admin-Setup-Token", "wrong-token-final")
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != http.StatusTooManyRequests || !strings.Contains(res.Body.String(), "AUTH_RATE_LIMITED") {
		t.Fatalf("setup should be rate limited after repeated invalid tokens, code=%d body=%s", res.Code, res.Body.String())
	}
}

func TestV1BearerInvalidSecretsShareRateLimitByClient(t *testing.T) {
	env := newTestAPIEnv(t)
	router := env.Router
	token := createTestSession(t, router)
	session, ok := env.Users.Current(token)
	if !ok {
		t.Fatal("test session missing")
	}
	_, validSecret, err := env.APIKeys.Create("valid sdk", session.User.Username, session.StorageToken)
	if err != nil {
		t.Fatalf("APIKeys.Create() error = %v", err)
	}

	for i := 0; i < 20; i++ {
		req := httptest.NewRequest(http.MethodGet, "/v1/image-tasks/nonexistent", nil)
		req.RemoteAddr = "198.51.100.73:51005"
		req.Header.Set("Authorization", "Bearer lyra_sk_invalid_"+stringInt(i))
		req.Header.Set("X-Forwarded-For", "127.0.0."+stringInt(i+1))
		req.Header.Set("CF-Connecting-IP", "10.0.0."+stringInt(i+1))
		res := httptest.NewRecorder()
		router.ServeHTTP(res, req)
		if res.Code != http.StatusUnauthorized || !strings.Contains(res.Body.String(), "UNAUTHORIZED") {
			t.Fatalf("invalid bearer attempt %d code=%d body=%s", i+1, res.Code, res.Body.String())
		}
	}

	validReq := httptest.NewRequest(http.MethodGet, "/v1/image-tasks/nonexistent", nil)
	validReq.RemoteAddr = "198.51.100.73:51005"
	validReq.Header.Set("Authorization", "Bearer "+validSecret)
	validRes := httptest.NewRecorder()
	router.ServeHTTP(validRes, validReq)
	if validRes.Code != http.StatusNotFound || !strings.Contains(validRes.Body.String(), "TASK_NOT_FOUND") {
		t.Fatalf("valid bearer should not be blocked by invalid bearer bucket, code=%d body=%s", validRes.Code, validRes.Body.String())
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/image-tasks/nonexistent", nil)
	req.RemoteAddr = "198.51.100.73:51005"
	req.Header.Set("Authorization", "Bearer lyra_sk_invalid_final")
	req.Header.Set("X-Forwarded-For", "127.0.0.250")
	req.Header.Set("CF-Connecting-IP", "10.0.0.250")
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != http.StatusTooManyRequests || !strings.Contains(res.Body.String(), "AUTH_RATE_LIMITED") {
		t.Fatalf("invalid bearer should still be rate limited by RemoteAddr after rotating invalid secrets and forwarded headers, code=%d body=%s", res.Code, res.Body.String())
	}
}
func TestUserLoginInvalidIdentifiersShareRateLimitByClient(t *testing.T) {
	router := newTestRouter(t)
	for i := 0; i < 20; i++ {
		var body bytes.Buffer
		if err := json.NewEncoder(&body).Encode(map[string]string{
			"identifier": "missing-user-" + stringInt(i),
			"password":   "wrong-password",
		}); err != nil {
			t.Fatalf("encode login payload: %v", err)
		}
		req := httptest.NewRequest(http.MethodPost, "/api/users/session", &body)
		req.RemoteAddr = "198.51.100.74:51006"
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Forwarded-For", "127.0.0."+stringInt(i+1))
		res := httptest.NewRecorder()
		router.ServeHTTP(res, req)
		if res.Code != http.StatusUnauthorized || !strings.Contains(res.Body.String(), "USER_LOGIN_INVALID") {
			t.Fatalf("invalid user login attempt %d code=%d body=%s", i+1, res.Code, res.Body.String())
		}
	}

	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(map[string]string{"identifier": "missing-user-final", "password": "wrong-password"}); err != nil {
		t.Fatalf("encode login payload: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/users/session", &body)
	req.RemoteAddr = "198.51.100.74:51006"
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Forwarded-For", "127.0.0.250")
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != http.StatusTooManyRequests || !strings.Contains(res.Body.String(), "AUTH_RATE_LIMITED") {
		t.Fatalf("rotating user login identifiers should rate limit by client, code=%d body=%s", res.Code, res.Body.String())
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
	Router   http.Handler
	APIKeys  *apikeys.Store
	Billing  *billing.Store
	Settings *settings.FileStore
	Users    *users.Store
	Spaces   *spaces.FileStore
	Jobs     *jobs.Store
	Output   *output.Store
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
	cfg.AdminSetupToken = "test-admin-setup-token"

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
	billingStore, err := billing.NewStore(filepath.Join(cfg.DataDir, "topups.json"))
	if err != nil {
		t.Fatalf("billing.NewStore() error = %v", err)
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

	router := NewRouter(Dependencies{
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
	return testAPIEnv{Router: router, APIKeys: apiKeyStore, Billing: billingStore, Settings: settingsStore, Users: userStore, Spaces: spaceStore, Jobs: jobStore, Output: outputStore}
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
	password := "R7!Orchid#Vault$2026"
	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(map[string]string{"password": password}); err != nil {
		t.Fatalf("encode admin setup payload: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/admin/auth/setup", &body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Admin-Setup-Token", "test-admin-setup-token")
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code == http.StatusConflict {
		var loginBody bytes.Buffer
		if err := json.NewEncoder(&loginBody).Encode(map[string]string{"password": password}); err != nil {
			t.Fatalf("encode admin login payload: %v", err)
		}
		req = httptest.NewRequest(http.MethodPost, "/api/admin/auth/session", &loginBody)
		req.Header.Set("Content-Type", "application/json")
		res = httptest.NewRecorder()
		router.ServeHTTP(res, req)
	}
	if res.Code < 200 || res.Code >= 300 {
		t.Fatalf("admin auth failed: code=%d body=%s", res.Code, res.Body.String())
	}
	bodyText := res.Body.String()
	var payload struct {
		Session struct {
			Token string `json:"token"`
		} `json:"session"`
	}
	if err := json.Unmarshal([]byte(bodyText), &payload); err != nil {
		t.Fatalf("decode admin auth response: %v body=%s", err, bodyText)
	}
	if payload.Session.Token == "" {
		t.Fatalf("admin token missing: %s", bodyText)
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

func grantTestCredits(t *testing.T, router http.Handler, token string, username string, amount int) {
	t.Helper()
	adminToken := createAdminToken(t, router)
	body := doAdminJSON(t, router, http.MethodPost, "/api/admin/users/credits/add", adminToken, map[string]any{
		"username": username,
		"amount":   amount,
		"reason":   "test credit grant",
	})
	if !strings.Contains(body, `"creditsBalance"`) {
		t.Fatalf("grant credits response invalid: %s", body)
	}
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

func seededRouterTask(storageToken string, id string, status jobs.Status, stage jobs.Stage, createdAt time.Time) jobs.Job {
	job := jobs.Job{
		ID:           id,
		SpaceToken:   storageToken,
		Provider:     "image-2",
		Model:        "gpt-image-2",
		Mode:         jobs.ModeTextToImage,
		Prompt:       "seeded task",
		Ratio:        "1:1",
		Resolution:   "standard",
		Quality:      "auto",
		OutputFormat: "png",
		Size:         "1024x1024",
		Count:        1,
		Concurrency:  1,
		Progress:     50,
		Results:      []jobs.Result{},
		CreatedAt:    createdAt,
		UpdatedAt:    createdAt,
	}
	jobs.ApplyStatus(&job, status)
	jobs.ApplyStage(&job, stage)
	job.UpdatedAt = createdAt
	return job
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
