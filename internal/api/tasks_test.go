package api

import (
	"encoding/json"
	"net/http"
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

func requireUserCredits(t *testing.T, env testAPIEnv, username string, want int) {
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
