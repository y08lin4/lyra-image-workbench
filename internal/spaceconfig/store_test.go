package spaceconfig

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/y08lin4/lyra-image-workbench/internal/spaces"
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

func TestStorePersistsCloudAPIKeysOnlyWhenExplicitlyEnabled(t *testing.T) {
	spaceStore, err := spaces.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}
	session, err := spaceStore.CreateOrOpenByPassword("R7!Cloud#Vault$2026")
	if err != nil {
		t.Fatalf("CreateOrOpenByPassword() error = %v", err)
	}
	store := NewStore(spaceStore)

	rawKey := "  sk-cloud-secret-1234567890  "
	enabled := true
	public, err := store.Update(session.Token, Update{APIKey: &rawKey, SaveAPIKeyToCloud: &enabled})
	if err != nil {
		t.Fatalf("Update(cloud key) error = %v", err)
	}
	if !public.APIKeySet || !public.CloudAPIKeySet || public.APIKeyPreview == "" || public.CloudAPIKeyPreview == "" {
		t.Fatalf("cloud API key should be reported as set: %+v", public)
	}
	encoded, _ := json.Marshal(public)
	if strings.Contains(string(encoded), "sk-cloud-secret-1234567890") {
		t.Fatalf("public config leaked raw cloud API key: %s", encoded)
	}
	private, err := store.Get(session.Token)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if private.APIKey != "sk-cloud-secret-1234567890" || !private.CloudAPIKeyEnabled {
		t.Fatalf("cloud API key was not persisted with explicit consent: %+v", private)
	}

	clear := true
	public, err = store.Update(session.Token, Update{ClearCloudAPIKey: &clear})
	if err != nil {
		t.Fatalf("Update(clear cloud key) error = %v", err)
	}
	if public.APIKeySet || public.CloudAPIKeySet {
		t.Fatalf("cloud API key should be cleared: %+v", public)
	}
}

func TestStoreScrubsLegacyPersistedAPIKeysOnRead(t *testing.T) {
	spaceStore, err := spaces.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}
	session, err := spaceStore.CreateOrOpenByPassword("R7!Legacy#Vault$2026")
	if err != nil {
		t.Fatalf("CreateOrOpenByPassword() error = %v", err)
	}
	dir, err := spaceStore.SpaceDir(session.Token)
	if err != nil {
		t.Fatalf("SpaceDir() error = %v", err)
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatalf("MkdirAll(space dir) error = %v", err)
	}
	legacy := map[string]any{
		"apiKey":                    "sk-legacy-secret-1234567890",
		"bananaApiKey":              "sk-legacy-banana-secret-1234567890",
		"cloudBananaApiKeyEnabled":  true,
		"defaultCount":              2,
		"defaultConcurrency":        3,
		"autoUploadPixhost":         true,
		"updatedAt":                 "2026-05-17T00:00:00Z",
	}
	payload, err := json.MarshalIndent(legacy, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent(legacy) error = %v", err)
	}
	configFile := filepath.Join(dir, "config.json")
	if err := os.WriteFile(configFile, append(payload, '\n'), 0o600); err != nil {
		t.Fatalf("WriteFile(legacy config) error = %v", err)
	}
	store := NewStore(spaceStore)

	public, err := store.Public(session.Token)
	if err != nil {
		t.Fatalf("Public() error = %v", err)
	}
	if public.APIKeySet || public.APIKeyPreview != "" {
		t.Fatalf("legacy keys should be hidden from public config: %+v", public)
	}
	encoded, _ := json.Marshal(public)
	if strings.Contains(string(encoded), "bananaApiKey") {
		t.Fatalf("public config should not expose banana key fields: %s", encoded)
	}
	if public.DefaultCount != 2 || public.DefaultConcurrency != 3 || !public.AutoUploadPixhost {
		t.Fatalf("non-secret settings should survive scrub: %+v", public)
	}
	scrubbed, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatalf("ReadFile(scrubbed config) error = %v", err)
	}
	if strings.Contains(string(scrubbed), "sk-legacy-secret") || strings.Contains(string(scrubbed), "sk-legacy-banana") || strings.Contains(string(scrubbed), "bananaApiKey") {
		t.Fatalf("legacy API keys were not removed from disk: %s", scrubbed)
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
