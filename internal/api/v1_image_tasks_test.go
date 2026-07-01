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

func TestV1ImagesGenerationsAcceptsCustomImage2FullSize(t *testing.T) {
	env := newTestAPIEnv(t)
	router := env.Router
	token := createTestSession(t, router)
	secret := createV1BearerSecret(t, router, token)
	grantTestCredits(t, router, token, "testuser01", 1)

	code, body := doJSONStatus(t, router, http.MethodPost, "/v1/images/generations", "", map[string]any{
		"model":         "image-2-4k",
		"prompt":        "a custom full image",
		"size":          "1536x864",
		"quality":       "medium",
		"output_format": "png",
	}, "Bearer "+secret)
	if code != http.StatusOK || !strings.Contains(body, `"status":"queued"`) {
		t.Fatalf("v1 generations create code=%d body=%s", code, body)
	}
	var response struct {
		Task jobs.Job `json:"task"`
	}
	if err := json.Unmarshal([]byte(body), &response); err != nil {
		t.Fatalf("decode v1 generations response: %v body=%s", err, body)
	}
	task := response.Task
	if task.Model != "image-2-4k" || task.Ratio != "auto" || task.Resolution != "auto" || task.Size != "1536x864" {
		t.Fatalf("unexpected custom size task fields: %+v", task)
	}
	if task.Quality != "medium" || task.OutputFormat != "png" {
		t.Fatalf("unexpected custom output settings: %+v", task)
	}
	waitForTestTaskFinal(t, router, token, task.ID)
}

func TestV1ImagesGenerationsImage2OmitSizeKeepsQualityAndFormat(t *testing.T) {
	env := newTestAPIEnv(t)
	router := env.Router
	token := createTestSession(t, router)
	secret := createV1BearerSecret(t, router, token)
	grantTestCredits(t, router, token, "testuser01", 1)

	code, body := doJSONStatus(t, router, http.MethodPost, "/v1/images/generations", "", map[string]any{
		"model":         "image-2",
		"prompt":        "a base image 2 task",
		"size":          "3840x2160",
		"quality":       "high",
		"output_format": "webp",
	}, "Bearer "+secret)
	if code != http.StatusOK || !strings.Contains(body, `"status":"queued"`) {
		t.Fatalf("v1 generations create code=%d body=%s", code, body)
	}
	var response struct {
		Task jobs.Job `json:"task"`
	}
	if err := json.Unmarshal([]byte(body), &response); err != nil {
		t.Fatalf("decode v1 generations response: %v body=%s", err, body)
	}
	task := response.Task
	if task.Model != "image-2" || task.Quality != "high" || task.OutputFormat != "webp" {
		t.Fatalf("unexpected image-2 output settings: %+v", task)
	}
	if task.Size != "3840x2160" || task.Ratio != "16:9" || task.Resolution != "4k" {
		t.Fatalf("unexpected compatibility size mapping before upstream skip: %+v", task)
	}
	waitForTestTaskFinal(t, router, token, task.ID)
}

func TestV1ImagesGenerationsImage2FullAutoSize(t *testing.T) {
	env := newTestAPIEnv(t)
	router := env.Router
	token := createTestSession(t, router)
	secret := createV1BearerSecret(t, router, token)
	grantTestCredits(t, router, token, "testuser01", 1)

	code, body := doJSONStatus(t, router, http.MethodPost, "/v1/images/generations", "", map[string]any{
		"model":         "image-2-4k",
		"prompt":        "a full image 2 auto task",
		"size":          "auto",
		"quality":       "medium",
		"output_format": "png",
	}, "Bearer "+secret)
	if code != http.StatusOK || !strings.Contains(body, `"status":"queued"`) {
		t.Fatalf("v1 generations create code=%d body=%s", code, body)
	}
	var response struct {
		Task jobs.Job `json:"task"`
	}
	if err := json.Unmarshal([]byte(body), &response); err != nil {
		t.Fatalf("decode v1 generations response: %v body=%s", err, body)
	}
	task := response.Task
	if task.Model != "image-2-4k" || task.Ratio != "auto" || task.Resolution != "auto" || task.Size != "自动" {
		t.Fatalf("unexpected image-2-4k auto task fields: %+v", task)
	}
	if task.Quality != "medium" || task.OutputFormat != "png" {
		t.Fatalf("unexpected image-2-4k output settings: %+v", task)
	}
	waitForTestTaskFinal(t, router, token, task.ID)
}
func TestV1ImagesGenerationsAcceptsExtraBodyForImage24K(t *testing.T) {
	env := newTestAPIEnv(t)
	router := env.Router
	token := createTestSession(t, router)
	secret := createV1BearerSecret(t, router, token)
	grantTestCredits(t, router, token, "testuser01", 2)

	code, body := doJSONStatus(t, router, http.MethodPost, "/v1/images/generations", "", map[string]any{
		"provider": "image-2-4k",
		"model":    "image-2-4k",
		"prompt":   "a generated 4k compatibility image",
		"n":        1,
		"size":     "1024x1024",
		"extra_body": map[string]any{
			"seed":          77,
			"model":         "bad-model",
			"size":          "1x1",
			"output_format": "jpeg",
		},
	}, "Bearer "+secret)
	if code != http.StatusOK || !strings.Contains(body, `"status":"queued"`) {
		t.Fatalf("v1 generations create code=%d body=%s", code, body)
	}
	var response struct {
		Task jobs.Job `json:"task"`
	}
	if err := json.Unmarshal([]byte(body), &response); err != nil {
		t.Fatalf("decode v1 generations response: %v body=%s", err, body)
	}
	task := response.Task
	if task.Model != "image-2-4k" || task.Ratio != "1:1" || task.Resolution != "standard" || task.Size != "1024x1024" {
		t.Fatalf("unexpected image-2-4k task fields: %+v", task)
	}
	if task.ExtraParams["seed"].(float64) != 77 {
		t.Fatalf("seed extra param missing: %+v", task.ExtraParams)
	}
	for _, key := range []string{"model", "size", "output_format"} {
		if _, ok := task.ExtraParams[key]; ok {
			t.Fatalf("core field %s should be filtered from extra params: %+v", key, task.ExtraParams)
		}
	}
	waitForTestTaskFinal(t, router, token, task.ID)
}
