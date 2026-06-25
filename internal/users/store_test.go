package users

import (
	"path/filepath"
	"testing"
	"time"
)

func TestLoginRequiresTOTPWhenEnabled(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "users.json"))
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	if _, err := store.Register("alice01", "R7!Blue#Vault$2026", ""); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	setup, err := store.BeginTOTPSetup("alice01")
	if err != nil {
		t.Fatalf("BeginTOTPSetup() error = %v", err)
	}
	code, err := totpCode(setup.Secret, time.Now().Unix()/totpPeriod)
	if err != nil {
		t.Fatalf("totpCode() error = %v", err)
	}
	if err := store.EnableTOTP("alice01", code); err != nil {
		t.Fatalf("EnableTOTP() error = %v", err)
	}
	if _, err := store.Login("alice01", "R7!Blue#Vault$2026", ""); err == nil {
		t.Fatal("Login without TOTP code should fail")
	}
	if _, err := store.Login("alice01", "R7!Blue#Vault$2026", code); err != nil {
		t.Fatalf("Login with TOTP code error = %v", err)
	}
}

func TestRegisterAllowsUppercaseUsername(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "users.json"))
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	session, err := store.Register("Alice_01", "R7!Blue#Vault$2026", "")
	if err != nil {
		t.Fatalf("Register() uppercase error = %v", err)
	}
	if session.User.Username != "Alice_01" || session.User.DisplayName != "Alice_01" {
		t.Fatalf("uppercase username was not preserved: %+v", session.User)
	}
	if _, err := store.Login("alice_01", "R7!Blue#Vault$2026", ""); err != nil {
		t.Fatalf("Login() should remain case-insensitive, got %v", err)
	}
	if _, err := store.Register("ALICE_01", "R7!Blue#Vault$2026", ""); err == nil {
		t.Fatal("Register() should reject case-insensitive duplicate username")
	}
}

func TestVideoQuotaLifecycle(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "users.json"))
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	if _, err := store.Register("Alice_01", "R7!Blue#Vault$2026", ""); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	user, err := store.AddVideoQuota("alice_01", 3)
	if err != nil {
		t.Fatalf("AddVideoQuota() error = %v", err)
	}
	if user.VideoQuota != 3 {
		t.Fatalf("VideoQuota after add = %d", user.VideoQuota)
	}
	remaining, err := store.ConsumeVideoQuota("ALICE_01", 2)
	if err != nil {
		t.Fatalf("ConsumeVideoQuota() error = %v", err)
	}
	if remaining != 1 {
		t.Fatalf("remaining = %d", remaining)
	}
	if _, err := store.ConsumeVideoQuota("Alice_01", 2); err == nil {
		t.Fatal("ConsumeVideoQuota() should fail when quota is not enough")
	}
	store.RefundVideoQuota("Alice_01", 1)
	quota, err := store.VideoQuota("alice_01")
	if err != nil {
		t.Fatalf("VideoQuota() error = %v", err)
	}
	if quota != 2 {
		t.Fatalf("quota after refund = %d", quota)
	}
}
