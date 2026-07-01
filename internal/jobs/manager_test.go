package jobs

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/y08lin4/lyra-image-workbench/internal/config"
	"github.com/y08lin4/lyra-image-workbench/internal/events"
	"github.com/y08lin4/lyra-image-workbench/internal/newapi"
	"github.com/y08lin4/lyra-image-workbench/internal/output"
	"github.com/y08lin4/lyra-image-workbench/internal/settings"
	"github.com/y08lin4/lyra-image-workbench/internal/spaceconfig"
	"github.com/y08lin4/lyra-image-workbench/internal/spaces"
	"github.com/y08lin4/lyra-image-workbench/internal/uploads"
)

func TestManagerCreateBeforeEnqueueFailureDoesNotStartWorker(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]string{{
			"b64_json": base64.StdEncoding.EncodeToString([]byte("should-not-run")),
		}}})
	}))
	defer server.Close()
	env := newManagerTestEnv(t, server.URL+"/v1")
	chargeErr := errors.New("credits not enough")

	created, err := env.manager.Create(env.token, CreateRequest{
		RuntimeSecrets: RuntimeSecrets{APIKey: "sk-test"},
		Mode:           ModeTextToImage,
		Prompt:         "cat",
		Ratio:          "1:1",
		Resolution:     "standard",
		Count:          1,
		Concurrency:    1,
		BeforeEnqueue: func(Job) error {
			return chargeErr
		},
	})
	if !errors.Is(err, chargeErr) {
		t.Fatalf("Create() error = %v, want %v", err, chargeErr)
	}
	if created.ID != "" {
		t.Fatalf("Create() should not return a created job on BeforeEnqueue failure: %+v", created)
	}
	jobs, listErr := env.store.List(env.token, 10)
	if listErr != nil {
		t.Fatalf("List() error = %v", listErr)
	}
	if len(jobs) != 0 {
		t.Fatalf("failed BeforeEnqueue job should be removed, got %+v", jobs)
	}
	time.Sleep(100 * time.Millisecond)
	if requests.Load() != 0 {
		t.Fatalf("worker should not call upstream after BeforeEnqueue failure, got %d requests", requests.Load())
	}
}
func TestManagerCreateReturnsQueuedAndCompletesInBackground(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.URL.Path != "/v1/images/generations" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if payload["n"].(float64) != 1 {
			t.Fatalf("count should be split into n=1 calls, got body %+v", payload)
		}
		if payload["output_format"] != "jpeg" {
			t.Fatalf("output_format should be jpeg, got body %+v", payload)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]string{{
			"b64_json":       base64.StdEncoding.EncodeToString([]byte("image")),
			"revised_prompt": "revised cat",
			"size":           "1024x1024",
			"quality":        "high",
			"output_format":  "jpeg",
		}}})
	}))
	defer server.Close()
	env := newManagerTestEnv(t, server.URL+"/v1")

	created, err := env.manager.Create(env.token, CreateRequest{
		RuntimeSecrets: RuntimeSecrets{APIKey: "sk-test"},
		Mode:           ModeTextToImage,
		Prompt:         "cat",
		Ratio:          "1:1",
		Resolution:     "standard",
		OutputFormat:   "jpg",
		Count:          2,
		Concurrency:    1,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.Status != StatusQueued || created.Progress != 0 || created.OutputFormat != "jpeg" {
		t.Fatalf("Create() should return queued snapshot, got %+v", created)
	}

	final := waitForJobStatus(t, env.store, env.token, created.ID, StatusSucceeded, 3*time.Second)
	if len(final.Results) != 2 {
		t.Fatalf("expected two results, got %+v", final.Results)
	}
	for _, result := range final.Results {
		if !result.OK || result.ImageURL == "" {
			t.Fatalf("unexpected result: %+v", result)
		}
		if result.RevisedPrompt != "revised cat" || result.ActualSize != "1024x1024" || result.ActualQuality != "high" || result.OutputFormat != "jpeg" {
			t.Fatalf("unexpected result metadata: %+v", result)
		}
	}
	if requests.Load() != 2 {
		t.Fatalf("expected two upstream calls, got %d", requests.Load())
	}
}

func TestManagerWorkerPoolRunsQueuedJobsConcurrently(t *testing.T) {
	t.Setenv(jobWorkerCountEnv, "2")
	var requests atomic.Int32
	started := make(chan struct{}, 2)
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		started <- struct{}{}
		select {
		case <-release:
		case <-r.Context().Done():
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]string{{
			"b64_json": base64.StdEncoding.EncodeToString([]byte("image")),
		}}})
	}))
	defer server.Close()
	defer closeOnce(release)
	env := newManagerTestEnv(t, server.URL)

	var ids []string
	for i := 0; i < 2; i++ {
		created, err := env.manager.Create(env.token, CreateRequest{
			RuntimeSecrets: RuntimeSecrets{APIKey: "sk-test"},
			Mode:           ModeTextToImage,
			Prompt:         fmt.Sprintf("cat %d", i),
			Ratio:          "1:1",
			Resolution:     "standard",
			Count:          1,
			Concurrency:    1,
		})
		if err != nil {
			t.Fatalf("Create(%d) error = %v", i, err)
		}
		ids = append(ids, created.ID)
	}

	for i := 0; i < 2; i++ {
		select {
		case <-started:
		case <-time.After(2 * time.Second):
			t.Fatalf("timed out waiting for concurrent worker start %d; requests = %d", i+1, requests.Load())
		}
	}
	closeOnce(release)
	for _, id := range ids {
		final := waitForJobStatus(t, env.store, env.token, id, StatusSucceeded, 3*time.Second)
		if len(final.Results) != 1 || !final.Results[0].OK {
			t.Fatalf("unexpected final result for %s: %+v", id, final.Results)
		}
	}
	if requests.Load() != 2 {
		t.Fatalf("expected two upstream calls, got %d", requests.Load())
	}
}

func TestConfiguredWorkerCount(t *testing.T) {
	t.Setenv(jobWorkerCountEnv, "")
	if got := configuredWorkerCount(); got != defaultWorkerCount {
		t.Fatalf("configuredWorkerCount(empty) = %d, want %d", got, defaultWorkerCount)
	}
	t.Setenv(jobWorkerCountEnv, "4")
	if got := configuredWorkerCount(); got != 4 {
		t.Fatalf("configuredWorkerCount(4) = %d", got)
	}
	t.Setenv(jobWorkerCountEnv, "999")
	if got := configuredWorkerCount(); got != maxWorkerCount {
		t.Fatalf("configuredWorkerCount(999) = %d, want %d", got, maxWorkerCount)
	}
	t.Setenv(jobWorkerCountEnv, "nope")
	if got := configuredWorkerCount(); got != defaultWorkerCount {
		t.Fatalf("configuredWorkerCount(nope) = %d, want %d", got, defaultWorkerCount)
	}
}

func TestManagerSendsExpectedImage2SizesForAllRatiosAndResolutions(t *testing.T) {
	type upstreamRequest struct {
		size string
	}
	captured := make(chan upstreamRequest, 63)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/images/generations" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			http.Error(w, "unexpected path", http.StatusNotFound)
			return
		}
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("decode body: %v", err)
			http.Error(w, "bad json", http.StatusBadRequest)
			return
		}
		size, _ := payload["size"].(string)
		captured <- upstreamRequest{size: size}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]string{{
			"b64_json": base64.StdEncoding.EncodeToString([]byte("image")),
			"size":     size,
		}}})
	}))
	defer server.Close()
	env := newManagerTestEnv(t, server.URL+"/v1")

	cases := []struct {
		ratio      string
		resolution string
		wantSize   string
	}{
		{ratio: "1:1", resolution: "standard", wantSize: "1024x1024"},
		{ratio: "2:3", resolution: "standard", wantSize: "1024x1536"},
		{ratio: "3:2", resolution: "standard", wantSize: "1536x1024"},
		{ratio: "3:4", resolution: "standard", wantSize: "768x1024"},
		{ratio: "4:3", resolution: "standard", wantSize: "1024x768"},
		{ratio: "9:16", resolution: "standard", wantSize: "1008x1792"},
		{ratio: "16:9", resolution: "standard", wantSize: "1792x1008"},
		{ratio: "1:1", resolution: "2k", wantSize: "2048x2048"},
		{ratio: "2:3", resolution: "2k", wantSize: "1344x2016"},
		{ratio: "3:2", resolution: "2k", wantSize: "2016x1344"},
		{ratio: "3:4", resolution: "2k", wantSize: "1536x2048"},
		{ratio: "4:3", resolution: "2k", wantSize: "2048x1536"},
		{ratio: "9:16", resolution: "2k", wantSize: "1152x2048"},
		{ratio: "16:9", resolution: "2k", wantSize: "2048x1152"},
		{ratio: "1:1", resolution: "4k", wantSize: "2880x2880"},
		{ratio: "2:3", resolution: "4k", wantSize: "2336x3504"},
		{ratio: "3:2", resolution: "4k", wantSize: "3504x2336"},
		{ratio: "3:4", resolution: "4k", wantSize: "2448x3264"},
		{ratio: "4:3", resolution: "4k", wantSize: "3264x2448"},
		{ratio: "9:16", resolution: "4k", wantSize: "2160x3840"},
		{ratio: "16:9", resolution: "4k", wantSize: "3840x2160"},
	}

	for _, tc := range cases {
		t.Run(tc.resolution+"_"+strings.ReplaceAll(tc.ratio, ":", "x"), func(t *testing.T) {
			for run := 1; run <= 3; run++ {
				t.Run(fmt.Sprintf("run_%d", run), func(t *testing.T) {
					created, err := env.manager.Create(env.token, CreateRequest{
						RuntimeSecrets: RuntimeSecrets{APIKey: "sk-test"},
						Mode:           ModeTextToImage,
						Prompt:         "size mapping",
						Ratio:          tc.ratio,
						Resolution:     tc.resolution,
						OutputFormat:   "png",
						Count:          1,
						Concurrency:    1,
					})
					if err != nil {
						t.Fatalf("Create() error = %v", err)
					}
					if created.Size != tc.wantSize {
						t.Fatalf("created.Size = %q, want %q", created.Size, tc.wantSize)
					}
					final := waitForJobStatus(t, env.store, env.token, created.ID, StatusSucceeded, 3*time.Second)
					if len(final.Results) != 1 || !final.Results[0].OK {
						t.Fatalf("unexpected final result: %+v", final.Results)
					}
					if final.Results[0].ActualSize != tc.wantSize {
						t.Fatalf("actual size = %q, want %q", final.Results[0].ActualSize, tc.wantSize)
					}
					select {
					case request := <-captured:
						if request.size != tc.wantSize {
							t.Fatalf("upstream size = %q, want %q", request.size, tc.wantSize)
						}
					case <-time.After(2 * time.Second):
						t.Fatal("timed out waiting for captured upstream request")
					}
				})
			}
		})
	}
}
func TestManagerImage2SkipsSpecParamsAndPassesExtraParams(t *testing.T) {
	captured := make(chan map[string]any, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		captured <- payload
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]string{{
			"b64_json": base64.StdEncoding.EncodeToString([]byte("image-2")),
		}}})
	}))
	defer server.Close()
	env := newManagerTestEnv(t, server.URL+"/v1")

	created, err := env.manager.Create(env.token, CreateRequest{
		RuntimeSecrets: RuntimeSecrets{APIKey: "sk-test"},
		Model:          "image-2",
		Mode:           ModeTextToImage,
		Prompt:         "image 2 cat",
		Ratio:          "16:9",
		Resolution:     "4k",
		Quality:        "high",
		OutputFormat:   "webp",
		ExtraParams: map[string]any{
			"negative_prompt": "blur",
			"seed":            123,
			"model":           "bad-model",
			"size":            "1x1",
			"quality":         "low",
			"output_format":   "jpeg",
		},
		Count:       1,
		Concurrency: 1,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.Model != "image-2" {
		t.Fatalf("created model = %q", created.Model)
	}
	if _, ok := created.ExtraParams["model"]; ok {
		t.Fatalf("core model should be filtered from extra params: %+v", created.ExtraParams)
	}
	if created.ExtraParams["negative_prompt"] != "blur" || created.ExtraParams["seed"] != 123 {
		t.Fatalf("unexpected created extra params: %+v", created.ExtraParams)
	}
	final := waitForJobStatus(t, env.store, env.token, created.ID, StatusSucceeded, 3*time.Second)
	if len(final.Results) != 1 || !final.Results[0].OK {
		t.Fatalf("unexpected final result: %+v", final.Results)
	}
	select {
	case payload := <-captured:
		if payload["model"] != "image-2" {
			t.Fatalf("upstream model = %+v", payload)
		}
		if _, ok := payload["size"]; ok {
			t.Fatalf("size should not be sent for image-2: %+v", payload)
		}
		if payload["quality"] != "high" || payload["output_format"] != "webp" {
			t.Fatalf("quality/output_format should still be sent for image-2: %+v", payload)
		}
		if payload["negative_prompt"] != "blur" || payload["seed"].(float64) != 123 {
			t.Fatalf("extra params were not sent upstream: %+v", payload)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for captured upstream request")
	}
}

func TestManagerImage2FullUsesSelectedSizeMapping(t *testing.T) {
	captured := make(chan map[string]any, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		captured <- payload
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]string{{
			"b64_json": base64.StdEncoding.EncodeToString([]byte("image-2-4k")),
			"size":     "1792x1008",
		}}})
	}))
	defer server.Close()
	env := newManagerTestEnv(t, server.URL+"/v1")

	created, err := env.manager.Create(env.token, CreateRequest{
		RuntimeSecrets: RuntimeSecrets{APIKey: "sk-test"},
		Model:          "image-2-4k",
		Mode:           ModeTextToImage,
		Prompt:         "4k cat",
		Ratio:          "16:9",
		Resolution:     "standard",
		OutputFormat:   "png",
		Count:          1,
		Concurrency:    1,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.Model != "image-2-4k" || created.Resolution != "standard" || created.Size != "1792x1008" {
		t.Fatalf("unexpected image-2-4k create fields: %+v", created)
	}
	final := waitForJobStatus(t, env.store, env.token, created.ID, StatusSucceeded, 3*time.Second)
	if len(final.Results) != 1 || final.Results[0].ActualSize != "1792x1008" {
		t.Fatalf("unexpected final result: %+v", final.Results)
	}
	select {
	case payload := <-captured:
		if payload["model"] != "image-2-4k" || payload["size"] != "1792x1008" {
			t.Fatalf("unexpected upstream request: %+v", payload)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for captured upstream request")
	}
}

func TestManagerImage2FullAcceptsCustomPixelSize(t *testing.T) {
	captured := make(chan map[string]any, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		captured <- payload
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]string{{
			"b64_json": base64.StdEncoding.EncodeToString([]byte("custom-size")),
			"size":     "1536x864",
		}}})
	}))
	defer server.Close()
	env := newManagerTestEnv(t, server.URL+"/v1")

	created, err := env.manager.Create(env.token, CreateRequest{
		RuntimeSecrets: RuntimeSecrets{APIKey: "sk-test"},
		Model:          "image-2-4k",
		Mode:           ModeTextToImage,
		Prompt:         "custom size cat",
		Ratio:          "1:1",
		Resolution:     "4k",
		Size:           "1536x864",
		OutputFormat:   "png",
		Count:          1,
		Concurrency:    1,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.Ratio != "auto" || created.Resolution != "auto" || created.Size != "1536x864" {
		t.Fatalf("unexpected custom size fields: %+v", created)
	}
	final := waitForJobStatus(t, env.store, env.token, created.ID, StatusSucceeded, 3*time.Second)
	if len(final.Results) != 1 || final.Results[0].ActualSize != "1536x864" {
		t.Fatalf("unexpected final result: %+v", final.Results)
	}
	select {
	case payload := <-captured:
		if payload["model"] != "image-2-4k" || payload["size"] != "1536x864" {
			t.Fatalf("unexpected upstream request: %+v", payload)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for captured upstream request")
	}
}
func TestManagerImage24KUsesConfiguredChannelBaseURLAndKey(t *testing.T) {
	defaultServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("default channel should not be used for image-2-4k, got %s", r.URL.Path)
	}))
	defer defaultServer.Close()

	captured := make(chan map[string]any, 1)
	server4K := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer sk-4k" {
			t.Fatalf("Authorization = %q", got)
		}
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		captured <- payload
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]string{{
			"b64_json": base64.StdEncoding.EncodeToString([]byte("image-2-4k-channel")),
			"size":     "1024x1024",
		}}})
	}))
	defer server4K.Close()

	env := newManagerTestEnv(t, defaultServer.URL+"/v1")
	if _, err := env.settings.Update(settings.Update{ImageChannels: []settings.ImageChannelConfig{
		{Type: settings.DefaultImageChannelType, Name: config.DefaultProvider, BaseURL: defaultServer.URL + "/v1", Key: "sk-default", Enabled: true},
		{Type: settings.DefaultImageChannelType, Name: "image-2-4k", BaseURL: server4K.URL + "/v1", Key: "sk-4k", Enabled: true},
	}}); err != nil {
		t.Fatalf("settings.Update(image channels) error = %v", err)
	}

	created, err := env.manager.Create(env.token, CreateRequest{
		Model:        "image-2-4k",
		Mode:         ModeTextToImage,
		Prompt:       "4k channel cat",
		Ratio:        "1:1",
		Resolution:   "auto",
		Count:        1,
		Concurrency:  1,
		OutputFormat: "png",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	final := waitForJobStatus(t, env.store, env.token, created.ID, StatusSucceeded, 3*time.Second)
	if len(final.Results) != 1 || final.Results[0].ActualSize != "1024x1024" {
		t.Fatalf("unexpected final result: %+v", final.Results)
	}
	select {
	case payload := <-captured:
		if payload["model"] != "image-2-4k" || payload["size"] != "1024x1024" {
			t.Fatalf("unexpected upstream request: %+v", payload)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for captured upstream request")
	}
}
func TestManagerImage2FullAutoRatioKeepsAutoSize(t *testing.T) {
	captured := make(chan map[string]any, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		captured <- payload
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]string{{
			"b64_json": base64.StdEncoding.EncodeToString([]byte("image-2-4k-auto")),
			"size":     "1024x1024",
		}}})
	}))
	defer server.Close()
	env := newManagerTestEnv(t, server.URL+"/v1")

	created, err := env.manager.Create(env.token, CreateRequest{
		RuntimeSecrets: RuntimeSecrets{APIKey: "sk-test"},
		Model:          "image-2-4k",
		Mode:           ModeTextToImage,
		Prompt:         "4k auto cat",
		Ratio:          "auto",
		Resolution:     "auto",
		OutputFormat:   "png",
		Count:          1,
		Concurrency:    1,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.Ratio != "auto" || created.Resolution != "auto" || created.Size != "自动" {
		t.Fatalf("unexpected image-2-4k auto fields: %+v", created)
	}
	final := waitForJobStatus(t, env.store, env.token, created.ID, StatusSucceeded, 3*time.Second)
	if len(final.Results) != 1 || !final.Results[0].OK {
		t.Fatalf("unexpected final result: %+v", final.Results)
	}
	select {
	case payload := <-captured:
		if payload["model"] != "image-2-4k" {
			t.Fatalf("unexpected upstream request: %+v", payload)
		}
		if _, ok := payload["size"]; ok {
			t.Fatalf("auto size should not be sent upstream: %+v", payload)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for captured upstream request")
	}
}

func TestResolveModelPassesThroughCustomOpenAICompatibleIDs(t *testing.T) {
	model, err := resolveModel(config.DefaultProvider, "z-image-turbo")
	if err != nil {
		t.Fatalf("resolveModel() error = %v", err)
	}
	if model != "z-image-turbo" {
		t.Fatalf("resolveModel() = %q, want custom model id", model)
	}
}
func TestManagerUsesCloudKeyWhenRuntimeKeyMissing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer sk-cloud" {
			t.Fatalf("Authorization = %q", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]string{{
			"b64_json": base64.StdEncoding.EncodeToString([]byte("cloud-image")),
		}}})
	}))
	defer server.Close()
	env := newManagerTestEnv(t, server.URL+"/v1")
	enabled := true
	cloudKey := "sk-cloud"
	if _, err := env.spaceConfig.Update(env.token, spaceconfig.Update{APIKey: &cloudKey, SaveAPIKeyToCloud: &enabled}); err != nil {
		t.Fatalf("spaceConfig.Update(cloud key) error = %v", err)
	}

	created, err := env.manager.Create(env.token, CreateRequest{
		Mode:        ModeTextToImage,
		Prompt:      "cloud cat",
		Ratio:       "1:1",
		Resolution:  "standard",
		Count:       1,
		Concurrency: 1,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	final := waitForJobStatus(t, env.store, env.token, created.ID, StatusSucceeded, 3*time.Second)
	if len(final.Results) != 1 || !final.Results[0].OK {
		t.Fatalf("unexpected final cloud-key result: %+v", final.Results)
	}
}

func TestManagerRecordsDebugLogsWhenEnabled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]string{{
			"b64_json": base64.StdEncoding.EncodeToString([]byte("debug-image")),
		}}})
	}))
	defer server.Close()
	env := newManagerTestEnv(t, server.URL+"/v1")
	debugEnabled := true
	if _, err := env.settings.Update(settings.Update{DebugEnabled: &debugEnabled}); err != nil {
		t.Fatalf("settings.Update(debug) error = %v", err)
	}

	created, err := env.manager.Create(env.token, CreateRequest{
		RuntimeSecrets: RuntimeSecrets{APIKey: "sk-test"},
		Mode:           ModeTextToImage,
		Prompt:         "debug cat",
		Ratio:          "1:1",
		Resolution:     "standard",
		Count:          1,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if !created.DebugEnabled || len(created.DebugLogs) == 0 {
		t.Fatalf("created job should include debug log: %+v", created)
	}

	final := waitForJobStatus(t, env.store, env.token, created.ID, StatusSucceeded, 3*time.Second)
	if len(final.DebugLogs) < 3 {
		t.Fatalf("expected request/response/save debug logs, got %+v", final.DebugLogs)
	}
	encoded, _ := json.Marshal(final.DebugLogs)
	if strings.Contains(string(encoded), "sk-test") {
		t.Fatalf("debug logs leaked raw api key: %s", encoded)
	}
	if !strings.Contains(string(encoded), "upstream_request") || !strings.Contains(string(encoded), "upstream_response") {
		t.Fatalf("debug logs missing upstream stages: %s", encoded)
	}
}

func TestManagerAllowsPartialFailed(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if requests.Add(1) == 2 {
			http.Error(w, `{"error":{"message":"boom"}}`, http.StatusBadGateway)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]string{{"b64_json": base64.StdEncoding.EncodeToString([]byte("ok"))}}})
	}))
	defer server.Close()
	env := newManagerTestEnv(t, server.URL)

	created, err := env.manager.Create(env.token, CreateRequest{
		RuntimeSecrets: RuntimeSecrets{APIKey: "sk-test"},
		Mode:           ModeTextToImage,
		Prompt:         "cat",
		Ratio:          "1:1",
		Resolution:     "standard",
		Count:          2,
		Concurrency:    1,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	final := waitForJobStatus(t, env.store, env.token, created.ID, StatusPartialFailed, 3*time.Second)
	okCount := 0
	for _, result := range final.Results {
		if result.OK {
			okCount++
		}
	}
	if okCount != 1 {
		t.Fatalf("expected exactly one success, got %+v", final.Results)
	}
}

func TestManagerSnapshotsImageReferencesForSubmittedTask(t *testing.T) {
	var editCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		editCalls.Add(1)
		if r.URL.Path != "/images/edits" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if err := r.ParseMultipartForm(20 << 20); err != nil {
			t.Fatalf("ParseMultipartForm() error = %v", err)
		}
		if got := r.FormValue("prompt"); got != "edit cat" {
			t.Fatalf("prompt = %q", got)
		}
		files := r.MultipartForm.File["image[]"]
		if len(files) != 1 {
			t.Fatalf("expected one reference image, got %d", len(files))
		}
		file, err := files[0].Open()
		if err != nil {
			t.Fatalf("open multipart image: %v", err)
		}
		defer file.Close()
		data, err := io.ReadAll(file)
		if err != nil {
			t.Fatalf("read multipart image: %v", err)
		}
		if !bytes.HasPrefix(data, pngHeader()) {
			t.Fatalf("reference image was not sent from snapshot, got %x", data)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]string{{"b64_json": base64.StdEncoding.EncodeToString([]byte("edited"))}}})
	}))
	defer server.Close()
	env := newManagerTestEnv(t, server.URL)
	reference := savePNGReference(t, env.uploads, env.token)

	created, err := env.manager.Create(env.token, CreateRequest{
		RuntimeSecrets: RuntimeSecrets{APIKey: "sk-test"},
		Mode:           ModeImageToImage,
		Prompt:         "edit cat",
		Ratio:          "1:1",
		Resolution:     "standard",
		Count:          1,
		Concurrency:    1,
		UploadIDs:      []string{reference.ID},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if len(created.References) != 1 {
		t.Fatalf("created job should snapshot references: %+v", created)
	}
	if err := env.uploads.DeleteReferenceImage(env.token, reference.ID); err != nil {
		t.Fatalf("DeleteReferenceImage() error = %v", err)
	}

	final := waitForJobStatus(t, env.store, env.token, created.ID, StatusSucceeded, 3*time.Second)
	if len(final.Results) != 1 || !final.Results[0].OK {
		t.Fatalf("unexpected final image-to-image result: %+v", final.Results)
	}
	if editCalls.Load() != 1 {
		t.Fatalf("expected one edit call, got %d", editCalls.Load())
	}
}

func TestManagerGeneratesGIFWithoutUpstream(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		t.Fatalf("GIF task should not call upstream, got %s", r.URL.Path)
	}))
	defer server.Close()
	env := newManagerTestEnv(t, server.URL)
	reference := savePNGReference(t, env.uploads, env.token)

	created, err := env.manager.Create(env.token, CreateRequest{
		Mode:         ModeGIF,
		Prompt:       "animate hair with gentle wind",
		FramePrompts: []string{"motion strength: standard", "loop rhythm: smooth"},
		OutputFormat: "gif",
		Count:        3,
		Concurrency:  2,
		UploadIDs:    []string{reference.ID},
	})
	if err != nil {
		t.Fatalf("Create() GIF task error = %v", err)
	}
	if created.Mode != ModeGIF || created.OutputFormat != "gif" || created.Count != 1 || created.ConsumedCredits != 0 {
		t.Fatalf("unexpected GIF create fields: %+v", created)
	}
	if len(created.References) != 1 {
		t.Fatalf("GIF task should snapshot one reference: %+v", created)
	}
	if err := env.uploads.DeleteReferenceImage(env.token, reference.ID); err != nil {
		t.Fatalf("DeleteReferenceImage() error = %v", err)
	}

	final := waitForJobStatus(t, env.store, env.token, created.ID, StatusSucceeded, 3*time.Second)
	if requests.Load() != 0 {
		t.Fatalf("GIF task should not call upstream, got %d requests", requests.Load())
	}
	if len(final.Results) != 1 || !final.Results[0].OK || final.Results[0].Mime != "image/gif" || final.Results[0].OutputFormat != "gif" {
		t.Fatalf("unexpected GIF final result: %+v", final.Results)
	}
	path, mime, err := env.output.Resolve(env.token, final.Results[0].OutputDate, final.Results[0].OutputFileName)
	if err != nil {
		t.Fatalf("Resolve() GIF output error = %v", err)
	}
	if mime != "image/gif" || filepath.Ext(path) != ".gif" {
		t.Fatalf("unexpected GIF output path/mime: %s %s", path, mime)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() GIF output error = %v", err)
	}
	if !bytes.HasPrefix(data, []byte("GIF")) {
		t.Fatalf("saved GIF output has unexpected prefix: %x", data[:min(8, len(data))])
	}
}

func TestManagerRejectsBananaProvider(t *testing.T) {
	env := newManagerTestEnv(t, "http://127.0.0.1:1")
	_, err := env.manager.Create(env.token, CreateRequest{
		Provider: "banana",
		Model:    "gemini-3.1-flash-image-preview",
		Mode:     ModeTextToImage,
		Prompt:   "banana",
		Count:    1,
	})
	if err == nil || !strings.Contains(err.Error(), "模型分组无效") {
		t.Fatalf("expected invalid provider error, got %v", err)
	}
}

func TestManagerCancelDoesNotWaitForUpstreamCompletion(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		closeOnce(started)
		select {
		case <-r.Context().Done():
			return
		case <-release:
			_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]string{{"b64_json": base64.StdEncoding.EncodeToString([]byte("late"))}}})
		}
	}))
	defer server.Close()
	defer close(release)
	env := newManagerTestEnv(t, server.URL)

	created, err := env.manager.Create(env.token, CreateRequest{
		RuntimeSecrets: RuntimeSecrets{APIKey: "sk-test"},
		Mode:           ModeTextToImage,
		Prompt:         "slow",
		Ratio:          "1:1",
		Resolution:     "standard",
		Count:          1,
		Concurrency:    1,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("upstream request did not start")
	}
	cancelled, err := env.manager.Cancel(env.token, created.ID)
	if err != nil {
		t.Fatalf("Cancel() error = %v", err)
	}
	if cancelled.Status != StatusCancelled {
		t.Fatalf("Cancel() status = %s", cancelled.Status)
	}
	final := waitForJobStatus(t, env.store, env.token, created.ID, StatusCancelled, 2*time.Second)
	if final.Status != StatusCancelled {
		t.Fatalf("final status = %s", final.Status)
	}
}

func TestManagerRecoverInterruptsQueuedAndRunningWithoutBrowserKeys(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]string{{"b64_json": base64.StdEncoding.EncodeToString([]byte("recovered"))}}})
	}))
	defer server.Close()
	env := newManagerTestEnvWithoutManager(t, server.URL)

	queued := newPersistedJob(env.token, "img_queued", StatusQueued, StageQueued)
	if err := env.store.Save(queued); err != nil {
		t.Fatalf("Save(queued) error = %v", err)
	}
	running := newPersistedJob(env.token, "img_running", StatusRunning, StageWaitingUpstream)
	if err := env.store.Save(running); err != nil {
		t.Fatalf("Save(running) error = %v", err)
	}

	env.manager = NewManager(env.store, events.NewHub(), env.settings, env.spaceConfig, env.uploads, env.output, newapi.NewClient())
	if err := env.manager.Recover(); err != nil {
		t.Fatalf("Recover() error = %v", err)
	}

	queuedInterrupted := waitForJobStatus(t, env.store, env.token, "img_queued", StatusInterrupted, 2*time.Second)
	if queuedInterrupted.Stage != StageInterrupted || queuedInterrupted.Progress != 100 {
		t.Fatalf("queued job was not interrupted cleanly: %+v", queuedInterrupted)
	}
	interrupted := waitForJobStatus(t, env.store, env.token, "img_running", StatusInterrupted, 2*time.Second)
	if interrupted.Stage != StageInterrupted || interrupted.Progress != 100 {
		t.Fatalf("running job was not interrupted cleanly: %+v", interrupted)
	}
}

func TestManagerRecoverRefundsQueuedChargedJobsOnly(t *testing.T) {
	env := newManagerTestEnvWithoutManager(t, "http://127.0.0.1:1")
	queued := newPersistedJob(env.token, "img_queued_charged", StatusQueued, StageQueued)
	queued.ConsumedCredits = 2
	if err := env.store.Save(queued); err != nil {
		t.Fatalf("Save(queued) error = %v", err)
	}
	running := newPersistedJob(env.token, "img_running_charged", StatusRunning, StageWaitingUpstream)
	running.ConsumedCredits = 3
	if err := env.store.Save(running); err != nil {
		t.Fatalf("Save(running) error = %v", err)
	}

	var refunded []string
	env.manager = NewManager(env.store, events.NewHub(), env.settings, env.spaceConfig, env.uploads, env.output, newapi.NewClient())
	if err := env.manager.Recover(RecoverOptions{RefundQueued: func(job Job) error {
		refunded = append(refunded, job.ID)
		return nil
	}}); err != nil {
		t.Fatalf("Recover() error = %v", err)
	}
	if len(refunded) != 1 || refunded[0] != queued.ID {
		t.Fatalf("Recover refunded jobs = %+v, want only %s", refunded, queued.ID)
	}

	queuedInterrupted := waitForJobStatus(t, env.store, env.token, queued.ID, StatusInterrupted, 2*time.Second)
	if queuedInterrupted.Stage != StageInterrupted || !strings.Contains(queuedInterrupted.Error, "refunded automatically") {
		t.Fatalf("queued job was not interrupted with refund message: %+v", queuedInterrupted)
	}
	runningInterrupted := waitForJobStatus(t, env.store, env.token, running.ID, StatusInterrupted, 2*time.Second)
	if runningInterrupted.Stage != StageInterrupted || strings.Contains(runningInterrupted.Error, "refunded automatically") {
		t.Fatalf("running job should not use queued refund message: %+v", runningInterrupted)
	}
}
func TestManagerSetFavoritePersists(t *testing.T) {
	env := newManagerTestEnvWithoutManager(t, "http://127.0.0.1:1")
	env.manager = NewManager(env.store, events.NewHub(), env.settings, env.spaceConfig, env.uploads, env.output, newapi.NewClient())
	job := newPersistedJob(env.token, "img_favorite", StatusSucceeded, StageSucceeded)
	if err := env.store.Save(job); err != nil {
		t.Fatalf("Save(job) error = %v", err)
	}

	favorited, err := env.manager.SetFavorite(env.token, job.ID, true)
	if err != nil {
		t.Fatalf("SetFavorite(true) error = %v", err)
	}
	if !favorited.Favorite {
		t.Fatalf("favorite flag was not set: %+v", favorited)
	}
	persisted, ok, err := env.store.Get(env.token, job.ID)
	if err != nil || !ok {
		t.Fatalf("Get(favorited) ok=%v err=%v", ok, err)
	}
	if !persisted.Favorite {
		t.Fatalf("favorite flag was not persisted: %+v", persisted)
	}

	unfavorited, err := env.manager.SetFavorite(env.token, job.ID, false)
	if err != nil {
		t.Fatalf("SetFavorite(false) error = %v", err)
	}
	if unfavorited.Favorite {
		t.Fatalf("favorite flag was not cleared: %+v", unfavorited)
	}
}

func TestManagerDeleteRemovesPersistedJob(t *testing.T) {
	env := newManagerTestEnvWithoutManager(t, "http://127.0.0.1:1")
	env.manager = NewManager(env.store, events.NewHub(), env.settings, env.spaceConfig, env.uploads, env.output, newapi.NewClient())
	job := newPersistedJob(env.token, "img_delete", StatusSucceeded, StageSucceeded)
	if err := env.store.Save(job); err != nil {
		t.Fatalf("Save(job) error = %v", err)
	}

	deleted, err := env.manager.Delete(env.token, job.ID)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if deleted.ID != job.ID {
		t.Fatalf("deleted job mismatch: %+v", deleted)
	}
	if _, ok, err := env.store.Get(env.token, job.ID); err != nil || ok {
		t.Fatalf("job still exists after delete: ok=%v err=%v", ok, err)
	}
}

func TestImageSizeKeepsAutoRatioAsAuto(t *testing.T) {
	if got := imageSize("auto", "4k"); got != "自动" {
		t.Fatalf("imageSize(auto, 4k) = %q", got)
	}
	if got := imageSize("16:9", "4k"); got != "3840x2160" {
		t.Fatalf("imageSize(16:9, 4k) = %q", got)
	}
}

type managerTestEnv struct {
	token       string
	store       *Store
	settings    *settings.FileStore
	spaceConfig *spaceconfig.Store
	uploads     *uploads.Store
	output      *output.Store
	manager     *Manager
}

func newManagerTestEnv(t *testing.T, baseURL string) managerTestEnv {
	t.Helper()
	env := newManagerTestEnvWithoutManager(t, baseURL)
	env.manager = NewManager(env.store, events.NewHub(), env.settings, env.spaceConfig, env.uploads, env.output, newapi.NewClient())
	return env
}

func newManagerTestEnvWithoutManager(t *testing.T, baseURL string) managerTestEnv {
	t.Helper()
	root := t.TempDir()
	spaceStore, err := spaces.NewFileStore(filepath.Join(root, "data"))
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}
	session, err := spaceStore.CreateOrOpenByPassword("R7!Blue#Vault$2026")
	if err != nil {
		t.Fatalf("CreateOrOpenByPassword() error = %v", err)
	}
	settingsStore, err := settings.NewFileStore(filepath.Join(root, "data", "config.local.json"), settings.RuntimeConfig{
		NewAPIBaseURL: baseURL,
		TimeoutSec:    config.DefaultTimeoutSec,
		Model:         config.DefaultModel,
	})
	if err != nil {
		t.Fatalf("settings.NewFileStore() error = %v", err)
	}
	spaceConfigStore := spaceconfig.NewStore(spaceStore)
	outputStore, err := output.NewStore(filepath.Join(root, "outputs"))
	if err != nil {
		t.Fatalf("output.NewStore() error = %v", err)
	}
	return managerTestEnv{
		token:       session.Token,
		store:       NewStore(spaceStore),
		settings:    settingsStore,
		spaceConfig: spaceConfigStore,
		uploads:     uploads.NewStore(spaceStore),
		output:      outputStore,
	}
}

func waitForJobStatus(t *testing.T, store *Store, token string, id string, status Status, timeout time.Duration) Job {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	var last Job
	for {
		job, ok, err := store.Get(token, id)
		if err != nil {
			t.Fatalf("store.Get() error = %v", err)
		}
		if ok {
			last = job
			if job.Status == status {
				return job
			}
		}
		select {
		case <-ctx.Done():
			t.Fatalf("timed out waiting for %s; last job = %+v", status, last)
		case <-ticker.C:
		}
	}
}

func newPersistedJob(token string, id string, status Status, stage Stage) Job {
	now := time.Now()
	job := Job{
		ID:          id,
		SpaceToken:  token,
		Mode:        ModeTextToImage,
		Prompt:      "recover",
		Ratio:       "1:1",
		Resolution:  "standard",
		Size:        "1024x1024",
		Count:       1,
		Concurrency: 1,
		Results:     []Result{},
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	ApplyStatus(&job, status)
	ApplyStage(&job, stage)
	return job
}

func savePNGReference(t *testing.T, store *uploads.Store, token string) uploads.ReferenceImage {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("image", "Reference.PNG")
	if err != nil {
		t.Fatalf("CreateFormFile() error = %v", err)
	}
	if _, err := part.Write(validPNGBytes(t)); err != nil {
		t.Fatalf("write reference image: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("multipart close: %v", err)
	}
	request := httptest.NewRequest(http.MethodPost, "/upload", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	if err := request.ParseMultipartForm(uploads.MaxReferenceUploadBytes); err != nil {
		t.Fatalf("ParseMultipartForm() error = %v", err)
	}
	saved, err := store.SaveReferenceImages(token, request.MultipartForm.File["image"])
	if err != nil {
		t.Fatalf("SaveReferenceImages() error = %v", err)
	}
	if len(saved) != 1 {
		t.Fatalf("expected one saved reference, got %+v", saved)
	}
	return saved[0]
}

func pngHeader() []byte {
	return []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}
}

func validPNGBytes(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 40, 24))
	for y := 0; y < 24; y++ {
		for x := 0; x < 40; x++ {
			img.SetRGBA(x, y, color.RGBA{R: uint8(x * 5), G: uint8(y * 8), B: 180, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("png.Encode() error = %v", err)
	}
	return buf.Bytes()
}

func closeOnce(ch chan struct{}) {
	defer func() { _ = recover() }()
	close(ch)
}
