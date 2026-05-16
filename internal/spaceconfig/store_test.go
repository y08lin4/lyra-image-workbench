package spaceconfig

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/y08lin4/image-Workbench-Localhost-Version/internal/spaces"
)

func TestStoreDoesNotPersistAPIKey(t *testing.T) {
	spaceStore, err := spaces.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}
	session, err := spaceStore.CreateOrOpenByPassword("R7!Blue#Vault$2026")
	if err != nil {
		t.Fatalf("CreateOrOpenByPassword() error = %v", err)
	}
	store := NewStore(spaceStore)

	rawKey := "  sk-test-secret-1234567890  "
	public, err := store.Update(session.Token, Update{APIKey: &rawKey})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if public.APIKeySet || public.APIKeyPreview != "" {
		t.Fatalf("API key should not be reported from server config: %+v", public)
	}
	encoded, _ := json.Marshal(public)
	if strings.Contains(string(encoded), "sk-test-secret-1234567890") {
		t.Fatalf("public config leaked raw API key: %s", encoded)
	}

	private, err := store.Get(session.Token)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if private.APIKey != "" {
		t.Fatalf("private API key should not be persisted: %q", private.APIKey)
	}
}

func TestStoreDoesNotPersistBananaAPIKey(t *testing.T) {
	spaceStore, err := spaces.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}
	session, err := spaceStore.CreateOrOpenByPassword("R7!Banana#Vault$2026")
	if err != nil {
		t.Fatalf("CreateOrOpenByPassword() error = %v", err)
	}
	store := NewStore(spaceStore)

	image2Key := "sk-image2-secret-1234567890"
	bananaKey := "  sk-banana-secret-0987654321  "
	public, err := store.Update(session.Token, Update{APIKey: &image2Key, BananaAPIKey: &bananaKey})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if public.APIKeySet || public.BananaAPIKeySet || public.APIKeyPreview != "" || public.BananaAPIKeyPreview != "" {
		t.Fatalf("key flags should stay false for browser-local keys: %+v", public)
	}
	encoded, _ := json.Marshal(public)
	if strings.Contains(string(encoded), "sk-banana-secret-0987654321") {
		t.Fatalf("public config leaked raw banana API key: %s", encoded)
	}

	private, err := store.Get(session.Token)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if private.APIKey != "" {
		t.Fatalf("image-2 key should not be persisted: %q", private.APIKey)
	}
	if private.BananaAPIKey != "" {
		t.Fatalf("banana API key should not be persisted: %q", private.BananaAPIKey)
	}
}

func TestStorePersistsDefaultConcurrency(t *testing.T) {
	spaceStore, err := spaces.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}
	session, err := spaceStore.CreateOrOpenByPassword("R7!Green#Vault$2026")
	if err != nil {
		t.Fatalf("CreateOrOpenByPassword() error = %v", err)
	}
	store := NewStore(spaceStore)

	defaultPublic, err := store.Public(session.Token)
	if err != nil {
		t.Fatalf("Public() error = %v", err)
	}
	if defaultPublic.DefaultConcurrency != 1 {
		t.Fatalf("default concurrency = %d", defaultPublic.DefaultConcurrency)
	}

	value := 6
	public, err := store.Update(session.Token, Update{DefaultConcurrency: &value})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if public.DefaultConcurrency != 6 {
		t.Fatalf("DefaultConcurrency = %d", public.DefaultConcurrency)
	}

	larger := 99
	public, err = store.Update(session.Token, Update{DefaultConcurrency: &larger})
	if err != nil {
		t.Fatalf("Update(larger) error = %v", err)
	}
	if public.DefaultConcurrency != 99 {
		t.Fatalf("DefaultConcurrency should keep values above 4, got %d", public.DefaultConcurrency)
	}
}

func TestStorePersistsDefaultCount(t *testing.T) {
	spaceStore, err := spaces.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}
	session, err := spaceStore.CreateOrOpenByPassword("R7!Count#Vault$2026")
	if err != nil {
		t.Fatalf("CreateOrOpenByPassword() error = %v", err)
	}
	store := NewStore(spaceStore)

	defaultPublic, err := store.Public(session.Token)
	if err != nil {
		t.Fatalf("Public() error = %v", err)
	}
	if defaultPublic.DefaultCount != 1 {
		t.Fatalf("default count = %d", defaultPublic.DefaultCount)
	}

	value := 4
	public, err := store.Update(session.Token, Update{DefaultCount: &value})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if public.DefaultCount != 4 {
		t.Fatalf("DefaultCount = %d", public.DefaultCount)
	}

	larger := 99
	public, err = store.Update(session.Token, Update{DefaultCount: &larger})
	if err != nil {
		t.Fatalf("Update(larger) error = %v", err)
	}
	if public.DefaultCount != 12 {
		t.Fatalf("DefaultCount should clamp to 12, got %d", public.DefaultCount)
	}
}
