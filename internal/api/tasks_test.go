package api

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/y08lin4/lyra-image-workbench/internal/jobs"
)

func TestBackgroundTaskCreateRecordsWebSourceAndConsumedCredits(t *testing.T) {
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
	if response.Job.ConsumedCredits != 3 || response.ConsumedCredits != 3 {
		t.Fatalf("consumed credits mismatch: %+v", response)
	}
	requireUserCredits(t, env, "testuser01", 3)
	requireLatestTaskCharge(t, env, "testuser01", response.TaskID, -3, 3)

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
	if retryResponse.Job.ConsumedCredits != 3 || retryResponse.ConsumedCredits != 3 {
		t.Fatalf("retry consumed credits mismatch: %+v", retryResponse)
	}
	requireUserCredits(t, env, "testuser01", 0)
	requireLatestTaskCharge(t, env, "testuser01", retryResponse.TaskID, -3, 0)
	waitForTestTaskFinal(t, router, token, retryResponse.TaskID)
	time.Sleep(200 * time.Millisecond)
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
