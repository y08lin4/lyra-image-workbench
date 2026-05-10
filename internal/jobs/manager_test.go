package jobs

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/y08lin4/image-Workbench-Localhost-Version/internal/config"
	"github.com/y08lin4/image-Workbench-Localhost-Version/internal/events"
	"github.com/y08lin4/image-Workbench-Localhost-Version/internal/newapi"
	"github.com/y08lin4/image-Workbench-Localhost-Version/internal/output"
	"github.com/y08lin4/image-Workbench-Localhost-Version/internal/settings"
	"github.com/y08lin4/image-Workbench-Localhost-Version/internal/spaceconfig"
	"github.com/y08lin4/image-Workbench-Localhost-Version/internal/spaces"
	"github.com/y08lin4/image-Workbench-Localhost-Version/internal/uploads"
)

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
		Mode:         ModeTextToImage,
		Prompt:       "cat",
		Ratio:        "1:1",
		Resolution:   "standard",
		OutputFormat: "jpg",
		Count:        2,
		Concurrency:  1,
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
		Mode:        ModeTextToImage,
		Prompt:      "cat",
		Ratio:       "1:1",
		Resolution:  "standard",
		Count:       2,
		Concurrency: 1,
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

func TestManagerRoutesBananaModelWithSeparateKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer sk-banana" {
			t.Fatalf("Authorization = %q", got)
		}
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if payload["model"] != "gemini-3.1-flash-image-preview-16x9-4k" {
			t.Fatalf("unexpected banana model: %+v", payload)
		}
		for _, key := range []string{"size", "quality", "output_format"} {
			if _, ok := payload[key]; ok {
				t.Fatalf("%s should not be sent for banana model-encoded specs: %+v", key, payload)
			}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]string{{"b64_json": base64.StdEncoding.EncodeToString([]byte("banana"))}}})
	}))
	defer server.Close()
	env := newManagerTestEnv(t, server.URL)
	bananaKey := "sk-banana"
	if _, err := env.spaceConfig.Update(env.token, spaceconfig.Update{BananaAPIKey: &bananaKey}); err != nil {
		t.Fatalf("space config banana Update() error = %v", err)
	}

	created, err := env.manager.Create(env.token, CreateRequest{
		Provider:    config.BananaProvider,
		Model:       "gemini-3.1-flash-image-preview-16x9-4k",
		Mode:        ModeTextToImage,
		Prompt:      "banana",
		Ratio:       "1:1",
		Resolution:  "standard",
		Quality:     "high",
		Count:       1,
		Concurrency: 1,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.Provider != config.BananaProvider || created.Model != "gemini-3.1-flash-image-preview-16x9-4k" {
		t.Fatalf("banana route fields were not stored: %+v", created)
	}
	if created.Ratio != "16:9" || created.Resolution != "4k" || created.Size != "3840x2160" || created.Quality != "auto" || created.OutputFormat != "auto" {
		t.Fatalf("banana model spec was not mapped onto task params: %+v", created)
	}
	final := waitForJobStatus(t, env.store, env.token, created.ID, StatusSucceeded, 3*time.Second)
	if len(final.Results) != 1 || !final.Results[0].OK {
		t.Fatalf("unexpected final banana result: %+v", final.Results)
	}
}

func TestManagerRequiresBananaKeyForBananaProvider(t *testing.T) {
	env := newManagerTestEnv(t, "http://127.0.0.1:1")
	_, err := env.manager.Create(env.token, CreateRequest{
		Provider: config.BananaProvider,
		Model:    config.DefaultBananaModel,
		Mode:     ModeTextToImage,
		Prompt:   "banana",
		Count:    1,
	})
	if err == nil || !strings.Contains(err.Error(), "banana 分组") {
		t.Fatalf("expected banana key error, got %v", err)
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

	created, err := env.manager.Create(env.token, CreateRequest{Mode: ModeTextToImage, Prompt: "slow", Ratio: "1:1", Resolution: "standard", Count: 1, Concurrency: 1})
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

func TestManagerRecoverRequeuesQueuedAndInterruptsRunning(t *testing.T) {
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

	waitForJobStatus(t, env.store, env.token, "img_queued", StatusSucceeded, 3*time.Second)
	interrupted := waitForJobStatus(t, env.store, env.token, "img_running", StatusInterrupted, 2*time.Second)
	if interrupted.Stage != StageInterrupted || interrupted.Progress != 100 {
		t.Fatalf("running job was not interrupted cleanly: %+v", interrupted)
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
	key := "sk-test"
	if _, err := spaceConfigStore.Update(session.Token, spaceconfig.Update{APIKey: &key}); err != nil {
		t.Fatalf("space config Update() error = %v", err)
	}
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

func closeOnce(ch chan struct{}) {
	defer func() { _ = recover() }()
	close(ch)
}
