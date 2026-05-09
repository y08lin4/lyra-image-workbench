package adminauth

import (
	"path/filepath"
	"testing"
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
