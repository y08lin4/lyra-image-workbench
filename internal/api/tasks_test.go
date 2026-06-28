package api

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/y08lin4/lyra-image-workbench/internal/jobs"
	"github.com/y08lin4/lyra-image-workbench/internal/settings"
)

func TestBackgroundTaskWithPersonalCloudKeyDoesNotConsumeCredits(t *testing.T) {
	env := newTestAPIEnv(t)
	router := env.Router
	token := createTestSession(t, router)
	_ = doJSON(t, router, http.MethodPost, "/api/config", token, map[string]any{
		"apiKey":            "sk-cloud-task-source-contract-1234567890",
		"saveApiKeyToCloud": true,
	})
	grantTestCredits(t, router, token, "testuser01", 6)

	code, body := doJSONStatus(t, router, http.MethodPost, "/api/background-tasks", token, map[string]any{
		"mode":         "text-to-image",
		"prompt":       "a task source contract image",
		"ratio":        "1:1",
		"resolution":   "standard",
		"quality":      "auto",
		"outputFormat": "png",
		"count":        3,
		"concurrency":  1,
	}, "")
	if code != http.StatusOK {
		t.Fatalf("web task create code=%d body=%s", code, body)
	}
	var response struct {
		Job             jobs.Job `json:"job"`
		TaskID          string   `json:"taskId"`
		ConsumedCredits int      `json:"consumedCredits"`
	}
	if err := json.Unmarshal([]byte(body), &response); err != nil {
		t.Fatalf("decode web task create response: %v body=%s", err, body)
	}
	if response.TaskID == "" || response.TaskID != response.Job.ID {
		t.Fatalf("taskId should mirror job id: %+v", response)
	}
	if response.Job.Source != jobs.JobSourceWeb {
		t.Fatalf("web task source=%q, want %q", response.Job.Source, jobs.JobSourceWeb)
	}
	if response.Job.ConsumedCredits != 0 || response.ConsumedCredits != 0 {
		t.Fatalf("personal-key task should not consume credits: %+v", response)
	}
	requireUserCredits(t, env, "testuser01", 6)
	requireNoTaskCharge(t, env, "testuser01", response.TaskID)

	waitForTestTaskFinal(t, router, token, response.TaskID)
	time.Sleep(200 * time.Millisecond)

	code, body = doJSONStatus(t, router, http.MethodPost, "/api/background-tasks/"+response.TaskID+"/retry", token, nil, "")
	if code != http.StatusOK {
		t.Fatalf("web task retry code=%d body=%s", code, body)
	}
	var retryResponse struct {
		Job             jobs.Job `json:"job"`
		TaskID          string   `json:"taskId"`
		ConsumedCredits int      `json:"consumedCredits"`
	}
	if err := json.Unmarshal([]byte(body), &retryResponse); err != nil {
		t.Fatalf("decode web task retry response: %v body=%s", err, body)
	}
	if retryResponse.TaskID == "" || retryResponse.TaskID == response.TaskID || retryResponse.TaskID != retryResponse.Job.ID {
		t.Fatalf("retry should create a distinct task id: first=%s retry=%+v", response.TaskID, retryResponse)
	}
	if retryResponse.Job.Source != jobs.JobSourceWeb {
		t.Fatalf("retry task source=%q, want %q", retryResponse.Job.Source, jobs.JobSourceWeb)
	}
	if retryResponse.Job.ConsumedCredits != 0 || retryResponse.ConsumedCredits != 0 {
		t.Fatalf("personal-key retry should not consume credits: %+v", retryResponse)
	}
	requireUserCredits(t, env, "testuser01", 6)
	requireNoTaskCharge(t, env, "testuser01", retryResponse.TaskID)
	waitForTestTaskFinal(t, router, token, retryResponse.TaskID)
	time.Sleep(200 * time.Millisecond)
}

func TestAdminBackgroundTaskWithSystemKeyConsumesCredits(t *testing.T) {
	env := newTestAPIEnv(t)
	router := env.Router
	token := createTestSession(t, router)
	if _, err := env.Users.SetAdmin("testuser01", true); err != nil {
		t.Fatalf("SetAdmin() error = %v", err)
	}
	systemKey := "sk-system-admin-task-1234567890"
	if _, err := env.Settings.Update(settings.Update{SystemAPIKey: &systemKey}); err != nil {
		t.Fatalf("settings.Update() system key error = %v", err)
	}
	grantTestCredits(t, router, token, "testuser01", 3)

	code, body := doJSONStatus(t, router, http.MethodPost, "/api/background-tasks", token, map[string]any{
		"mode":         "text-to-image",
		"prompt":       "admin task should use system key and credits",
		"ratio":        "1:1",
		"resolution":   "standard",
		"quality":      "auto",
		"outputFormat": "png",
		"count":        2,
		"concurrency":  1,
	}, "")
	if code != http.StatusOK {
		t.Fatalf("admin web task create code=%d body=%s", code, body)
	}
	var response struct {
		Job             jobs.Job `json:"job"`
		TaskID          string   `json:"taskId"`
		ConsumedCredits int      `json:"consumedCredits"`
	}
	if err := json.Unmarshal([]byte(body), &response); err != nil {
		t.Fatalf("decode admin web task response: %v body=%s", err, body)
	}
	if response.TaskID == "" || response.TaskID != response.Job.ID {
		t.Fatalf("task id mismatch: %+v", response)
	}
	if response.Job.ConsumedCredits != 2 || response.ConsumedCredits != 2 {
		t.Fatalf("system-key task should consume credits even for admin: %+v", response)
	}
	requireUserCredits(t, env, "testuser01", 1)
	requireLatestTaskCharge(t, env, "testuser01", response.TaskID, -2, 1)
	waitForTestTaskFinal(t, router, token, response.TaskID)
}

func TestGIFTasksAreLocalAndDoNotConsumeCredits(t *testing.T) {
	req := jobs.CreateRequest{Mode: jobs.ModeGIF, Count: 6, Concurrency: 3}
	if got := billableTaskCredits(req); got != 0 {
		t.Fatalf("billableTaskCredits(GIF) = %d, want 0", got)
	}
	if !requestUsesPersonalUpstreamKey(nil, "", req) {
		t.Fatal("GIF task should be treated as local/personal-key backed even without an upstream key store")
	}
}
func TestBackgroundTaskWithRuntimePersonalKeyDoesNotConsumeCredits(t *testing.T) {
	env := newTestAPIEnv(t)
	router := env.Router
	token := createTestSession(t, router)
	grantTestCredits(t, router, token, "testuser01", 2)

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer sk-runtime-personal-task-1234567890" {
			t.Fatalf("Authorization = %q", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]string{{
			"b64_json": base64.StdEncoding.EncodeToString([]byte("runtime-key-image")),
		}}})
	}))
	defer upstream.Close()
	baseURL := upstream.URL + "/v1"
	timeoutSec := 60
	if _, err := env.Settings.Update(settings.Update{NewAPIBaseURL: &baseURL, TimeoutSec: &timeoutSec}); err != nil {
		t.Fatalf("settings.Update() upstream error = %v", err)
	}

	var reqBody bytes.Buffer
	if err := json.NewEncoder(&reqBody).Encode(map[string]any{
		"mode":         "text-to-image",
		"prompt":       "runtime personal key should be free",
		"ratio":        "1:1",
		"resolution":   "standard",
		"quality":      "auto",
		"outputFormat": "png",
		"count":        1,
		"concurrency":  1,
	}); err != nil {
		t.Fatalf("encode create payload: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/background-tasks", &reqBody)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(runtimeAPIKeyHeader, "sk-runtime-personal-task-1234567890")
	req.AddCookie(&http.Cookie{Name: userSessionCookie, Value: token})
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("runtime-key task create code=%d body=%s", res.Code, res.Body.String())
	}
	var response struct {
		Job             jobs.Job `json:"job"`
		ConsumedCredits int      `json:"consumedCredits"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode runtime-key response: %v body=%s", err, res.Body.String())
	}
	if response.Job.ConsumedCredits != 0 || response.ConsumedCredits != 0 {
		t.Fatalf("runtime personal-key task should not consume credits: %+v", response)
	}
	requireUserCredits(t, env, "testuser01", 2)
	requireNoTaskCharge(t, env, "testuser01", response.Job.ID)
	waitForTestTaskFinal(t, router, token, response.Job.ID)
}

func TestGIFBackgroundTaskCreatesLocalGIF(t *testing.T) {
	env := newTestAPIEnv(t)
	router := env.Router
	token := createTestSession(t, router)
	uploadID := uploadAPITestPNGReference(t, router, token)

	code, body := doJSONStatus(t, router, http.MethodPost, "/api/background-tasks", token, map[string]any{
		"mode":         "gif",
		"prompt":       "animate the reference image with a soft loop",
		"framePrompts": []string{"gentle motion", "smooth loop"},
		"outputFormat": "gif",
		"count":        4,
		"concurrency":  2,
		"uploadIds":    []string{uploadID},
	}, "")
	if code != http.StatusOK {
		t.Fatalf("GIF task create code=%d body=%s", code, body)
	}
	assertNoGIFPlaceholder(t, body)
	var response struct {
		Job             jobs.Job `json:"job"`
		TaskID          string   `json:"taskId"`
		ConsumedCredits int      `json:"consumedCredits"`
	}
	if err := json.Unmarshal([]byte(body), &response); err != nil {
		t.Fatalf("decode GIF create response: %v body=%s", err, body)
	}
	if response.Job.Mode != jobs.ModeGIF || response.Job.OutputFormat != "gif" || response.Job.Count != 1 || response.Job.ConsumedCredits != 0 || response.ConsumedCredits != 0 {
		t.Fatalf("unexpected GIF create response: %+v", response)
	}

	waitForTestTaskFinal(t, router, token, response.TaskID)
	body = doJSON(t, router, http.MethodGet, "/api/background-tasks/"+response.TaskID, token, nil)
	assertNoGIFPlaceholder(t, body)
	var final struct {
		Task jobs.Job `json:"task"`
	}
	if err := json.Unmarshal([]byte(body), &final); err != nil {
		t.Fatalf("decode GIF final response: %v body=%s", err, body)
	}
	if final.Task.Status != jobs.StatusSucceeded || len(final.Task.Results) != 1 || !final.Task.Results[0].OK || final.Task.Results[0].Mime != "image/gif" || final.Task.Results[0].OutputFormat != "gif" {
		t.Fatalf("unexpected GIF final task: %+v", final.Task)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/background-tasks/"+response.TaskID+"/images/0", nil)
	req.AddCookie(&http.Cookie{Name: userSessionCookie, Value: token})
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != http.StatusOK || !strings.HasPrefix(res.Header().Get("Content-Type"), "image/gif") || !bytes.HasPrefix(res.Body.Bytes(), []byte("GIF")) {
		t.Fatalf("GIF image response invalid: code=%d contentType=%q prefix=%x", res.Code, res.Header().Get("Content-Type"), res.Body.Bytes()[:minInt(len(res.Body.Bytes()), 8)])
	}
}

func uploadAPITestPNGReference(t *testing.T, router http.Handler, token string) string {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("image", "gif-reference.png")
	if err != nil {
		t.Fatalf("CreateFormFile() error = %v", err)
	}
	if _, err := part.Write(apiTestPNGBytes(t)); err != nil {
		t.Fatalf("write PNG reference: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("multipart close: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/uploads/reference", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.AddCookie(&http.Cookie{Name: userSessionCookie, Value: token})
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("upload reference code=%d body=%s", res.Code, res.Body.String())
	}
	var payload struct {
		Uploads []struct {
			ID string `json:"id"`
		} `json:"uploads"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode upload response: %v body=%s", err, res.Body.String())
	}
	if len(payload.Uploads) != 1 || payload.Uploads[0].ID == "" {
		t.Fatalf("upload response missing id: %s", res.Body.String())
	}
	return payload.Uploads[0].ID
}

func apiTestPNGBytes(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 48, 32))
	for y := 0; y < 32; y++ {
		for x := 0; x < 48; x++ {
			img.SetRGBA(x, y, color.RGBA{R: uint8(80 + x), G: uint8(40 + y*3), B: 190, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("png.Encode() error = %v", err)
	}
	return buf.Bytes()
}

func assertNoGIFPlaceholder(t *testing.T, body string) {
	t.Helper()
	if strings.Contains(body, "E_GIF_BACKEND_UNAVAILABLE") || strings.Contains(body, "GIF 动图生成"+"后端"+"尚未"+"接入") {
		t.Fatalf("GIF response used legacy placeholder: %s", body)
	}
}

func minInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}
func requireUserCredits(t *testing.T, env testAPIEnv, username string, want int) {
	time.Sleep(200 * time.Millisecond)
	t.Helper()
	profile, err := env.Users.Profile(username)
	if err != nil {
		t.Fatalf("Profile(%q) error = %v", username, err)
	}
	if profile.CreditsBalance != want {
		t.Fatalf("creditsBalance=%d, want %d", profile.CreditsBalance, want)
	}
}

func requireNoTaskCharge(t *testing.T, env testAPIEnv, username string, taskID string) {
	t.Helper()
	ledger, err := env.Users.ListCreditLedger(username)
	if err != nil {
		t.Fatalf("ListCreditLedger(%q) error = %v", username, err)
	}
	for _, entry := range ledger {
		if entry.Type == "task_charge" && entry.SourceID == taskID {
			t.Fatalf("unexpected task charge for %s: %+v", taskID, entry)
		}
	}
}

func requireLatestTaskCharge(t *testing.T, env testAPIEnv, username string, taskID string, delta int, balanceAfter int) {
	t.Helper()
	ledger, err := env.Users.ListCreditLedger(username)
	if err != nil {
		t.Fatalf("ListCreditLedger(%q) error = %v", username, err)
	}
	if len(ledger) == 0 {
		t.Fatalf("ledger for %q is empty", username)
	}
	latest := ledger[0]
	if latest.Type != "task_charge" || latest.SourceID != taskID || latest.Delta != delta || latest.BalanceAfter != balanceAfter {
		t.Fatalf("latest task charge mismatch: got %+v want source=%s delta=%d balance=%d", latest, taskID, delta, balanceAfter)
	}
}
