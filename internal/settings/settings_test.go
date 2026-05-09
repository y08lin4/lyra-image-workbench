package settings

import (
	"path/filepath"
	"testing"

	"github.com/y08lin4/image-Workbench-Localhost-Version/internal/config"
)

func TestFileStoreUpdatePersistsAdminConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.local.json")
	store, err := NewFileStore(path, RuntimeConfig{
		NewAPIBaseURL: "http://127.0.0.1:3000/v1/",
		TimeoutSec:    config.DefaultTimeoutSec,
		Model:         config.DefaultModel,
	})
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}

	baseURL := "http://127.0.0.1:3010/v1/images/generations"
	timeout := 601
	updated, err := store.Update(Update{NewAPIBaseURL: &baseURL, TimeoutSec: &timeout})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.NewAPIBaseURL != "http://127.0.0.1:3010/v1" {
		t.Fatalf("NewAPIBaseURL = %q", updated.NewAPIBaseURL)
	}
	if updated.TimeoutSec != timeout {
		t.Fatalf("TimeoutSec = %d", updated.TimeoutSec)
	}
	if updated.Model != config.DefaultModel {
		t.Fatalf("Model = %q", updated.Model)
	}

	reopened, err := NewFileStore(path, DefaultsFromConfig(config.Load()))
	if err != nil {
		t.Fatalf("reopen NewFileStore() error = %v", err)
	}
	public := reopened.Public()
	if public.NewAPIBaseURL != updated.NewAPIBaseURL || public.TimeoutSec != timeout {
		t.Fatalf("Public() = %+v", public)
	}
	if !public.ModelLocked || public.Model != config.DefaultModel {
		t.Fatalf("model should be locked to %s, got %+v", config.DefaultModel, public)
	}
}

func TestFileStoreRejectsInvalidAdminConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.local.json")
	store, err := NewFileStore(path, RuntimeConfig{
		NewAPIBaseURL: config.DefaultNewAPIBaseURL,
		TimeoutSec:    config.DefaultTimeoutSec,
		Model:         config.DefaultModel,
	})
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}

	badURL := "ftp://127.0.0.1/v1"
	if _, err := store.Update(Update{NewAPIBaseURL: &badURL}); err == nil {
		t.Fatal("expected invalid URL error")
	}

	tooShort := config.MinTimeoutSec - 1
	if _, err := store.Update(Update{TimeoutSec: &tooShort}); err == nil {
		t.Fatal("expected invalid timeout error")
	}
}
