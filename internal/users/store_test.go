package users

import (
	"path/filepath"
	"testing"
	"time"
)

const testPassword = "R7!Blue#Vault$2026"

func newTestStore(t *testing.T) *Store {
	t.Helper()
	store, err := NewStore(filepath.Join(t.TempDir(), "users.json"))
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	return store
}

func TestLoginRequiresTOTPWhenEnabled(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.Register("alice01", "", testPassword, "", ""); err != nil {
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
	if _, err := store.Login("alice01", testPassword, ""); err == nil {
		t.Fatal("Login without TOTP code should fail")
	}
	if _, err := store.Login("alice01", testPassword, code); err != nil {
		t.Fatalf("Login with TOTP code error = %v", err)
	}
}

func TestRegisterAllowsUppercaseUsername(t *testing.T) {
	store := newTestStore(t)
	session, err := store.Register("Alice_01", "", testPassword, "", "")
	if err != nil {
		t.Fatalf("Register() uppercase error = %v", err)
	}
	if session.User.Username != "Alice_01" || session.User.DisplayName != "Alice_01" {
		t.Fatalf("uppercase username was not preserved: %+v", session.User)
	}
	if _, err := store.Login("alice_01", testPassword, ""); err != nil {
		t.Fatalf("Login() should remain case-insensitive, got %v", err)
	}
	if _, err := store.Register("ALICE_01", "", testPassword, "", ""); err == nil {
		t.Fatal("Register() should reject case-insensitive duplicate username")
	}
}

func TestRegisterFirstUserAdminEmailLoginAndProfile(t *testing.T) {
	store := newTestStore(t)
	first, err := store.Register("Alice_01", "Alice@Example.com", testPassword, "", "")
	if err != nil {
		t.Fatalf("Register() first user error = %v", err)
	}
	if !first.User.IsAdmin {
		t.Fatal("first registered user should be admin")
	}
	if first.User.Email != "alice@example.com" {
		t.Fatalf("email was not normalized: %q", first.User.Email)
	}
	if first.User.ReferralCode == "" {
		t.Fatal("registered user should receive referral code")
	}
	second, err := store.Register("Bob_01", "bob@example.com", testPassword, "", "")
	if err != nil {
		t.Fatalf("Register() second user error = %v", err)
	}
	if second.User.IsAdmin {
		t.Fatal("second registered user should not be admin")
	}
	if _, err := store.Login("ALICE@EXAMPLE.COM", testPassword, ""); err != nil {
		t.Fatalf("Login() should accept email identifier, got %v", err)
	}
	if _, err := store.Register("Alice_02", "alice@example.com", testPassword, "", ""); err == nil {
		t.Fatal("Register() should reject duplicate email")
	}
	profile, err := store.UpdateProfile("Alice_01", ProfileUpdate{
		DisplayName: "Alice Display",
		Email:       "alice.new@example.com",
		AvatarURL:   "https://example.com/avatar.png",
	})
	if err != nil {
		t.Fatalf("UpdateProfile() error = %v", err)
	}
	if profile.DisplayName != "Alice Display" || profile.Email != "alice.new@example.com" || profile.AvatarURL == "" {
		t.Fatalf("profile update mismatch: %+v", profile)
	}
	if _, err := store.Login("alice.new@example.com", testPassword, ""); err != nil {
		t.Fatalf("Login() should use updated email, got %v", err)
	}
}

func TestAdminCreditsRequireReasonAndAppendLedger(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.Register("Alice_01", "", testPassword, "", ""); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if _, _, err := store.AddCreditsByAdmin("alice_01", 5, "", "root"); err == nil {
		t.Fatal("AddCreditsByAdmin() should require reason")
	}
	user, entry, err := store.AddCreditsByAdmin("alice_01", 5, "线下付款补录", "root")
	if err != nil {
		t.Fatalf("AddCreditsByAdmin() error = %v", err)
	}
	if user.CreditsBalance != 5 {
		t.Fatalf("credit balance after admin add = %+v", user)
	}
	if entry.Type != creditLedgerTypeAdminAdd || entry.Reason != "线下付款补录" || entry.AdminActor != "root" || entry.BalanceAfter != 5 {
		t.Fatalf("admin ledger entry mismatch: %+v", entry)
	}
	ledger, err := store.ListCreditLedger("ALICE_01")
	if err != nil {
		t.Fatalf("ListCreditLedger() error = %v", err)
	}
	if len(ledger) != 1 {
		t.Fatalf("ledger length = %d, want 1", len(ledger))
	}
}

func TestPurchaseCreditsIdempotentAndReferralRewardOnce(t *testing.T) {
	store := newTestStore(t)
	inviter, err := store.Register("Inviter_01", "inviter@example.com", testPassword, "", "")
	if err != nil {
		t.Fatalf("Register() inviter error = %v", err)
	}
	buyer, err := store.Register("Buyer_01", "buyer@example.com", testPassword, inviter.User.ReferralCode, "")
	if err != nil {
		t.Fatalf("Register() buyer error = %v", err)
	}
	if buyer.User.ReferredByUsername != "Inviter_01" {
		t.Fatalf("buyer referredByUsername = %q", buyer.User.ReferredByUsername)
	}
	result, err := store.AddPurchaseCredits("buyer_01", 10, "order_1", 3)
	if err != nil {
		t.Fatalf("AddPurchaseCredits() error = %v", err)
	}
	if !result.Created || result.Entry.Type != creditLedgerTypePurchase || result.Entry.BalanceAfter != 10 {
		t.Fatalf("purchase result mismatch: %+v", result)
	}
	if result.ReferralEntry == nil || result.ReferralEntry.Type != creditLedgerTypeReferralReward || result.ReferralEntry.BalanceAfter != 3 {
		t.Fatalf("referral reward missing or wrong: %+v", result.ReferralEntry)
	}
	duplicate, err := store.AddPurchaseCredits("Buyer_01", 10, "order_1", 3)
	if err != nil {
		t.Fatalf("duplicate AddPurchaseCredits() error = %v", err)
	}
	if duplicate.Created {
		t.Fatal("duplicate purchase sourceId should be idempotent")
	}
	buyerLedger, err := store.ListCreditLedger("buyer_01")
	if err != nil {
		t.Fatalf("buyer ListCreditLedger() error = %v", err)
	}
	if len(buyerLedger) != 1 {
		t.Fatalf("buyer ledger length = %d, want 1", len(buyerLedger))
	}
	inviterLedger, err := store.ListCreditLedger("inviter_01")
	if err != nil {
		t.Fatalf("inviter ListCreditLedger() error = %v", err)
	}
	if len(inviterLedger) != 1 {
		t.Fatalf("inviter ledger length = %d, want 1", len(inviterLedger))
	}
	inviterProfile, err := store.Profile("Inviter_01")
	if err != nil {
		t.Fatalf("Profile() inviter error = %v", err)
	}
	if inviterProfile.CreditsBalance != 3 {
		t.Fatalf("inviter credits = %d, want 3", inviterProfile.CreditsBalance)
	}
}
