package output

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/y08lin4/image-Workbench-Localhost-Version/internal/spaces"
)

func TestStoreSaveAndResolve(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "outputs"))
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	token, err := spaces.DeriveToken("R7!Blue#Vault$2026")
	if err != nil {
		t.Fatalf("DeriveToken() error = %v", err)
	}

	saved, err := store.Save(token, "img_test", 0, []byte("image-bytes"), "image/webp")
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if saved.URL == "" || saved.Mime != "image/webp" || filepath.Ext(saved.FileName) != ".webp" {
		t.Fatalf("unexpected saved image: %+v", saved)
	}
	if _, err := os.Stat(saved.Path); err != nil {
		t.Fatalf("saved file missing: %v", err)
	}

	path, mime, err := store.Resolve(token, saved.Date, saved.FileName)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if path != saved.Path || mime != "image/webp" {
		t.Fatalf("Resolve() = path %q mime %q", path, mime)
	}
}

func TestStoreResolveRejectsUnsafePath(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "outputs"))
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	token, err := spaces.DeriveToken("R7!Blue#Vault$2026")
	if err != nil {
		t.Fatalf("DeriveToken() error = %v", err)
	}

	if _, _, err := store.Resolve(token, "20260509", "x.png"); err == nil {
		t.Fatal("expected invalid date error")
	}
	if _, _, err := store.Resolve(token, "2026-05-09", "../x.png"); err == nil {
		t.Fatal("expected invalid file name error")
	}
	if _, _, err := store.Resolve("bad-token", "2026-05-09", "x.png"); err == nil {
		t.Fatal("expected invalid token error")
	}
}
