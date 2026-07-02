package users

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/y08lin4/lyra-image-workbench/internal/passwordhash"
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
func TestSetDisabledBlocksLoginCurrentAndStorageToken(t *testing.T) {
	store := newTestStore(t)
	session, err := store.Register("Alice_01", "", testPassword, "", "")
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	adminUser, err := store.SetDisabled("alice_01", true)
	if err != nil {
		t.Fatalf("SetDisabled(true) error = %v", err)
	}
	if !adminUser.Disabled {
		t.Fatalf("SetDisabled(true) did not mark user disabled: %+v", adminUser)
	}
	if _, ok := store.Current(session.Token); ok {
		t.Fatal("Current() should reject an existing disabled-user session")
	}
	_, err = store.Login("alice_01", testPassword, "")
	assertUserErrorCode(t, err, "USER_DISABLED")
	if _, ok := store.FindByStorageToken(session.StorageToken); ok {
		t.Fatal("FindByStorageToken() should hide disabled users")
	}
	_, err = store.ProfileByStorageToken(session.StorageToken)
	assertUserErrorCode(t, err, "USER_DISABLED")

	adminUser, err = store.SetDisabled("Alice_01", false)
	if err != nil {
		t.Fatalf("SetDisabled(false) error = %v", err)
	}
	if adminUser.Disabled {
		t.Fatalf("SetDisabled(false) did not enable user: %+v", adminUser)
	}
	if _, err := store.Login("alice_01", testPassword, ""); err != nil {
		t.Fatalf("Login() after enabling user error = %v", err)
	}
}

func TestSetDisabledKeepsAtLeastOneActiveAdmin(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.Register("Admin_01", "", testPassword, "", ""); err != nil {
		t.Fatalf("Register(admin) error = %v", err)
	}
	if _, err := store.SetAdmin("admin_01", true); err != nil {
		t.Fatalf("SetAdmin(admin) error = %v", err)
	}
	_, err := store.SetDisabled("admin_01", true)
	assertUserErrorCode(t, err, "USER_LAST_ADMIN_REQUIRED")

	if _, err := store.Register("Admin_02", "", testPassword, "", ""); err != nil {
		t.Fatalf("Register(second admin) error = %v", err)
	}
	if _, err := store.SetAdmin("admin_02", true); err != nil {
		t.Fatalf("SetAdmin(second admin) error = %v", err)
	}
	adminUser, err := store.SetDisabled("admin_01", true)
	if err != nil {
		t.Fatalf("SetDisabled(admin with backup) error = %v", err)
	}
	if !adminUser.Disabled {
		t.Fatalf("admin should be disabled when another active admin exists: %+v", adminUser)
	}
}

func TestRegisterUserEmailLoginAndProfile(t *testing.T) {
	store := newTestStore(t)
	first, err := store.Register("Alice_01", "Alice@Example.com", testPassword, "", "")
	if err != nil {
		t.Fatalf("Register() first user error = %v", err)
	}
	if first.User.IsAdmin {
		t.Fatal("public registration should not auto-grant admin")
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

func TestRegisterWithInitialCreditsAppendsLedger(t *testing.T) {
	store := newTestStore(t)
	session, err := store.RegisterWithInitialCredits("Alice_01", "", testPassword, "", "", 7)
	if err != nil {
		t.Fatalf("RegisterWithInitialCredits() error = %v", err)
	}
	if session.User.CreditsBalance != 7 {
		t.Fatalf("initial credits balance = %d, want 7", session.User.CreditsBalance)
	}
	ledger, err := store.ListCreditLedger("alice_01")
	if err != nil {
		t.Fatalf("ListCreditLedger() error = %v", err)
	}
	if len(ledger) != 1 {
		t.Fatalf("ledger length = %d, want 1", len(ledger))
	}
	entry := ledger[0]
	if entry.Type != creditLedgerTypeInitialFree || entry.Delta != 7 || entry.BalanceAfter != 7 || entry.SourceID != "initial:alice_01" {
		t.Fatalf("initial ledger entry mismatch: %+v", entry)
	}
}

func TestClaimDailyCreditsIdempotentByDate(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.Register("Alice_01", "", testPassword, "", ""); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	day := time.Date(2026, 6, 26, 9, 30, 0, 0, time.Local)
	first, err := store.ClaimDailyCredits("alice_01", 3, day)
	if err != nil {
		t.Fatalf("ClaimDailyCredits(first) error = %v", err)
	}
	if !first.Created || first.Entry.Type != creditLedgerTypeDailyFree || first.Entry.Delta != 3 || first.Entry.BalanceAfter != 3 || first.ClaimDate != "2026-06-26" {
		t.Fatalf("first daily claim mismatch: %+v", first)
	}
	duplicate, err := store.ClaimDailyCredits("Alice_01", 3, day.Add(2*time.Hour))
	if err != nil {
		t.Fatalf("ClaimDailyCredits(duplicate) error = %v", err)
	}
	if duplicate.Created || duplicate.Entry.ID != first.Entry.ID || duplicate.User.CreditsBalance != 3 {
		t.Fatalf("same-day duplicate should return existing entry without crediting again: %+v", duplicate)
	}
	next, err := store.ClaimDailyCredits("Alice_01", 3, day.AddDate(0, 0, 1))
	if err != nil {
		t.Fatalf("ClaimDailyCredits(next day) error = %v", err)
	}
	if !next.Created || next.Entry.BalanceAfter != 6 || next.User.CreditsBalance != 6 {
		t.Fatalf("next-day claim mismatch: %+v", next)
	}
	ledger, err := store.ListCreditLedger("alice_01")
	if err != nil {
		t.Fatalf("ListCreditLedger() error = %v", err)
	}
	if len(ledger) != 2 {
		t.Fatalf("ledger length = %d, want 2", len(ledger))
	}
}
func TestPurchaseCreditsIdempotentAndReferralRewardOnce(t *testing.T) {
	store := newTestStore(t)
	inviter, err := store.Register("Inviter_01", "inviter@example.com", testPassword, "", "")
	if err != nil {
		t.Fatalf("Register() inviter error = %v", err)
	}
	inviteLink := "https://image.example.com/?ref=" + inviter.User.ReferralCode
	buyer, err := store.Register("Buyer_01", "buyer@example.com", testPassword, inviteLink, "")
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

func TestTaskCreditsChargeRefundAndLedgerOrder(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.RegisterWithInitialCredits("Alice_01", "", testPassword, "", "", 5); err != nil {
		t.Fatalf("RegisterWithInitialCredits() error = %v", err)
	}
	charge, err := store.ChargeTaskCredits("alice_01", 2, "task_1", "")
	if err != nil {
		t.Fatalf("ChargeTaskCredits() error = %v", err)
	}
	if !charge.Created || charge.Entry.Type != creditLedgerTypeTaskCharge || charge.Entry.Delta != -2 || charge.Entry.BalanceAfter != 3 || charge.User.CreditsBalance != 3 {
		t.Fatalf("charge result mismatch: %+v", charge)
	}
	duplicateCharge, err := store.ChargeTaskCredits("Alice_01", 2, "task_1", "duplicate retry")
	if err != nil {
		t.Fatalf("duplicate ChargeTaskCredits() error = %v", err)
	}
	if duplicateCharge.Created || duplicateCharge.Entry.ID != charge.Entry.ID || duplicateCharge.User.CreditsBalance != 3 {
		t.Fatalf("duplicate charge should be idempotent: %+v", duplicateCharge)
	}
	if _, err := store.ChargeTaskCredits("Alice_01", 3, "task_1", "changed amount"); err == nil {
		t.Fatal("same task ID with different amount should conflict")
	}
	if _, err := store.ChargeTaskCredits("Alice_01", 10, "task_2", "too much"); err == nil {
		t.Fatal("charge should fail when balance is not enough")
	}
	refund, err := store.RefundTaskCredits("Alice_01", "task_1", "")
	if err != nil {
		t.Fatalf("RefundTaskCredits() error = %v", err)
	}
	if !refund.Created || refund.Entry.Type != creditLedgerTypeTaskRefund || refund.Entry.Delta != 2 || refund.Entry.BalanceAfter != 5 || refund.User.CreditsBalance != 5 {
		t.Fatalf("refund result mismatch: %+v", refund)
	}
	duplicateRefund, err := store.RefundTaskCredits("alice_01", "task_1", "duplicate retry")
	if err != nil {
		t.Fatalf("duplicate RefundTaskCredits() error = %v", err)
	}
	if duplicateRefund.Created || duplicateRefund.Entry.ID != refund.Entry.ID || duplicateRefund.User.CreditsBalance != 5 {
		t.Fatalf("duplicate refund should be idempotent: %+v", duplicateRefund)
	}
	if _, err := store.RefundTaskCredits("Alice_01", "task_missing", ""); err == nil {
		t.Fatal("refund without a task charge should fail")
	}
	ledger, err := store.ListCreditLedger("alice_01")
	if err != nil {
		t.Fatalf("ListCreditLedger() error = %v", err)
	}
	if len(ledger) != 3 {
		t.Fatalf("ledger length = %d, want 3: %+v", len(ledger), ledger)
	}
	if ledger[0].Type != creditLedgerTypeTaskRefund || ledger[1].Type != creditLedgerTypeTaskCharge || ledger[2].Type != creditLedgerTypeInitialFree {
		t.Fatalf("ledger should be newest first, got %+v", ledger)
	}
}

func TestLoginUpgradesLegacyPasswordHash(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.Register("Alice_01", "", testPassword, "", ""); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	store.mu.Lock()
	index, ok := store.findLocked("alice_01")
	if !ok {
		store.mu.Unlock()
		t.Fatal("registered user missing")
	}
	legacySalt := "00112233445566778899aabbccddeeff"
	store.current.Users[index].SaltHex = legacySalt
	store.current.Users[index].HashHex = passwordhash.LegacyHash(legacySalt, testPassword)
	if err := store.saveLocked(); err != nil {
		store.mu.Unlock()
		t.Fatalf("save legacy hash: %v", err)
	}
	store.mu.Unlock()

	if _, err := store.Login("alice_01", testPassword, ""); err != nil {
		t.Fatalf("Login() legacy hash error = %v", err)
	}
	store.mu.Lock()
	upgraded := store.current.Users[index].HashHex
	store.mu.Unlock()
	if !strings.HasPrefix(upgraded, passwordhash.Scheme+"$") {
		t.Fatalf("legacy hash was not upgraded: %s", upgraded)
	}
}

func assertUserErrorCode(t *testing.T, err error, code string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected %s error, got nil", code)
	}
	var userErr Error
	if !AsError(err, &userErr) || userErr.Code != code {
		t.Fatalf("error code = %v, want %s", err, code)
	}
}
