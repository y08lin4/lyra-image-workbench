package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/y08lin4/lyra-image-workbench/internal/jobs"
)

func TestV1ImagesGenerationsCreatesImageTask(t *testing.T) {
	env := newTestAPIEnv(t)
	router := env.Router
	token := createTestSession(t, router)
	secret := createV1BearerSecret(t, router, token)
	grantTestCredits(t, router, token, "testuser01", 4)

	code, body := doJSONStatus(t, router, http.MethodPost, "/v1/images/generations", "", map[string]any{
		"model":         "gpt-image-2",
		"prompt":        "a generated compatibility image",
		"n":             2,
		"size":          "1536x1024",
		"quality":       "high",
		"output_format": "webp",
		"concurrency":   2,
	}, "Bearer "+secret)
	if code != http.StatusOK || !strings.Contains(body, `"status":"queued"`) {
		t.Fatalf("v1 generations create code=%d body=%s", code, body)
	}
	var response struct {
		Task            jobs.Job `json:"task"`
		TaskID          string   `json:"taskId"`
		ConsumedCredits int      `json:"consumedCredits"`
	}
	if err := json.Unmarshal([]byte(body), &response); err != nil {
		t.Fatalf("decode v1 generations response: %v body=%s", err, body)
	}
	task := response.Task
	if task.ID == "" || response.TaskID != task.ID || task.Mode != jobs.ModeTextToImage || task.Prompt != "a generated compatibility image" {
		t.Fatalf("unexpected task identity: task=%+v response=%+v", task, response)
	}
	if task.Source != jobs.JobSourceAPI {
		t.Fatalf("task source=%q, want %q", task.Source, jobs.JobSourceAPI)
	}
	if task.Count != 2 || task.Concurrency != 2 || task.ConsumedCredits != 0 || response.ConsumedCredits != 0 {
		t.Fatalf("personal-key task should keep count/concurrency and consume zero credits: task=%+v response=%+v", task, response)
	}
	if task.Ratio != "3:2" || task.Resolution != "standard" || task.Size != "1536x1024" {
		t.Fatalf("unexpected size mapping: %+v", task)
	}
	if task.Quality != "high" || task.OutputFormat != "webp" {
		t.Fatalf("unexpected output settings: %+v", task)
	}
	requireUserCredits(t, env, "testuser01", 4)
	requireNoTaskCharge(t, env, "testuser01", task.ID)
	waitForTestTaskFinal(t, router, token, task.ID)
}
