package adminauth

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/y08lin4/lyra-image-workbench/internal/passwordhash"
)

const testAdminPassword = "R7!Orchid#Vault$2026"

func TestStoreSetupLoginValidateAndPersist(t *testing.T) {
	path := filepath.Join(t.TempDir(), "admin.auth.json")
	store, err := NewStore(path)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	if store.Status().PasswordSet {
		t.Fatal("password should not be set initially")
	}

	session, err := store.Setup(testAdminPassword)
	if err != nil {
		t.Fatalf("Setup() error = %v", err)
	}
	if session.Token == "" || !store.ValidateToken(session.Token) {
		t.Fatalf("setup session should be valid: %+v", session)
	}
	if !store.Status().PasswordSet {
		t.Fatal("password should be set after setup")
	}
	if _, err := store.Setup(testAdminPassword); err == nil {
		t.Fatal("second setup should be rejected")
	}

	reopened, err := NewStore(path)
	if err != nil {
		t.Fatalf("reopen NewStore() error = %v", err)
	}
	if !reopened.Status().PasswordSet {
		t.Fatal("password should persist")
	}
	if reopened.ValidateToken(session.Token) {
		t.Fatal("in-memory session should not survive restart")
	}

	login, err := reopened.Login(testAdminPassword)
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if !reopened.ValidateToken(login.Token) {
		t.Fatal("login session should be valid")
	}
	reopened.Logout(login.Token)
	if reopened.ValidateToken(login.Token) {
		t.Fatal("session should be invalid after logout")
	}
}

func TestStoreRejectsWeakOrWrongPassword(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "admin.auth.json"))
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	if _, err := store.Setup("short"); err == nil {
		t.Fatal("weak admin password should be rejected")
	}
	if _, err := store.Setup(testAdminPassword); err != nil {
		t.Fatalf("Setup() error = %v", err)
	}
	if _, err := store.Login("Wrong!Password123"); err == nil {
		t.Fatal("wrong admin password should be rejected")
	}
}

func TestStoreUpgradesLegacyPasswordHash(t *testing.T) {
	path := filepath.Join(t.TempDir(), "admin.auth.json")
	store, err := NewStore(path)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	if _, err := store.Setup(testAdminPassword); err != nil {
		t.Fatalf("Setup() error = %v", err)
	}
	store.mu.Lock()
	legacySalt := "00112233445566778899aabbccddeeff"
	store.current.SaltHex = legacySalt
	store.current.HashHex = passwordhash.LegacyHash(legacySalt, testAdminPassword)
	if err := store.saveLocked(); err != nil {
		store.mu.Unlock()
		t.Fatalf("save legacy hash: %v", err)
	}
	store.mu.Unlock()

	reopened, err := NewStore(path)
	if err != nil {
		t.Fatalf("reopen NewStore() error = %v", err)
	}
	if _, err := reopened.Login(testAdminPassword); err != nil {
		t.Fatalf("Login() legacy hash error = %v", err)
	}
	if !strings.HasPrefix(reopened.current.HashHex, passwordhash.Scheme+"$") {
		t.Fatalf("legacy hash was not upgraded: %s", reopened.current.HashHex)
	}
}
