package settings

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/y08lin4/lyra-image-workbench/internal/config"
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
	publicURL := "https://image.example.com/"
	timeout := 601
	debugEnabled := true
	initialCredits := 7
	dailyCredits := 2
	updated, err := store.Update(Update{
		NewAPIBaseURL:         &baseURL,
		PublicBaseURL:         &publicURL,
		DebugEnabled:          &debugEnabled,
		TimeoutSec:            &timeout,
		NewUserInitialCredits: &initialCredits,
		DailyFreeCredits:      &dailyCredits,
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.NewAPIBaseURL != "http://127.0.0.1:3010/v1" {
		t.Fatalf("NewAPIBaseURL = %q", updated.NewAPIBaseURL)
	}
	if updated.TimeoutSec != timeout {
		t.Fatalf("TimeoutSec = %d", updated.TimeoutSec)
	}
	if updated.PublicBaseURL != "https://image.example.com" {
		t.Fatalf("PublicBaseURL = %q", updated.PublicBaseURL)
	}
	if !updated.DebugEnabled {
		t.Fatalf("DebugEnabled = false")
	}
	if updated.Model != config.DefaultModel {
		t.Fatalf("Model = %q", updated.Model)
	}
	if updated.NewUserInitialCredits != initialCredits || updated.DailyFreeCredits != dailyCredits {
		t.Fatalf("free credit settings = initial %d daily %d", updated.NewUserInitialCredits, updated.DailyFreeCredits)
	}

	reopened, err := NewFileStore(path, DefaultsFromConfig(config.Load()))
	if err != nil {
		t.Fatalf("reopen NewFileStore() error = %v", err)
	}
	public := reopened.Public()
	if public.NewAPIBaseURL != updated.NewAPIBaseURL || public.PublicBaseURL != updated.PublicBaseURL || public.TimeoutSec != timeout || !public.DebugEnabled {
		t.Fatalf("Public() = %+v", public)
	}
	if !public.ModelLocked || public.Model != config.DefaultModel {
		t.Fatalf("model should be locked to %s, got %+v", config.DefaultModel, public)
	}
	if public.NewUserInitialCredits != initialCredits || public.DailyFreeCredits != dailyCredits || public.Billing.NewUserInitialCredits != initialCredits || public.Billing.DailyFreeCredits != dailyCredits {
		t.Fatalf("public free credit settings mismatch: %+v", public)
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

	badPublicURL := "ftp://example.com"
	if _, err := store.Update(Update{PublicBaseURL: &badPublicURL}); err == nil {
		t.Fatal("expected invalid public base URL error")
	}

	negative := -1
	if _, err := store.Update(Update{NewUserInitialCredits: &negative}); err == nil {
		t.Fatal("expected invalid initial free credits error")
	}
	if _, err := store.Update(Update{DailyFreeCredits: &negative}); err == nil {
		t.Fatal("expected invalid daily free credits error")
	}
}

func TestFileStoreEpaySettingsMaskAndClearKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.local.json")
	store, err := NewFileStore(path, RuntimeConfig{
		NewAPIBaseURL: config.DefaultNewAPIBaseURL,
		TimeoutSec:    config.DefaultTimeoutSec,
		Model:         config.DefaultModel,
	})
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}

	enabled := true
	apiURL := "https://pay.example.com/submit.php/"
	pid := " 1001 "
	key := "merchant-secret-123456"
	price := 25
	minimum := 20
	reward := 5
	updated, err := store.Update(Update{
		EpayEnabled:           &enabled,
		EpayAPIURL:            &apiURL,
		EpayPID:               &pid,
		EpayKey:               &key,
		EpayMethods:           []string{" ALIPAY ", "wxpay", "alipay", ""},
		CreditPriceCents:      &price,
		MinTopUpCredits:       &minimum,
		ReferralRewardCredits: &reward,
	})
	if err != nil {
		t.Fatalf("Update(epay) error = %v", err)
	}
	if !updated.EpayEnabled || updated.EpayAPIURL != "https://pay.example.com/submit.php" || updated.EpayPID != "1001" || updated.EpayKey != key {
		t.Fatalf("epay settings not normalized/persisted privately: %+v", updated)
	}
	if len(updated.EpayMethods) != 2 || updated.EpayMethods[0] != "alipay" || updated.EpayMethods[1] != "wxpay" {
		t.Fatalf("EpayMethods = %+v", updated.EpayMethods)
	}

	public := store.Public()
	if !public.EpayKeySet || public.EpayKeyPreview != "merc********3456" || public.Billing.EpayKeyPreview != public.EpayKeyPreview {
		t.Fatalf("Public() epay key status invalid: %+v", public)
	}
	payload, err := json.Marshal(public)
	if err != nil {
		t.Fatalf("Marshal public config: %v", err)
	}
	if strings.Contains(string(payload), key) || strings.Contains(string(payload), `"epayKey":`) {
		t.Fatalf("public config leaked epay key: %s", payload)
	}

	reopened, err := NewFileStore(path, DefaultsFromConfig(config.Load()))
	if err != nil {
		t.Fatalf("reopen NewFileStore() error = %v", err)
	}
	if reopened.Get().EpayKey != key {
		t.Fatalf("private epay key not persisted")
	}
	disabled := false
	if _, err := reopened.Update(Update{EpayEnabled: &disabled, ClearEpayKey: true}); err != nil {
		t.Fatalf("Update(clear epay key) error = %v", err)
	}
	cleared := reopened.Public()
	if cleared.EpayKeySet || cleared.EpayKeyPreview != "" {
		t.Fatalf("epay key should be cleared: %+v", cleared)
	}
}

func TestFileStoreSMTPSettingsMaskAndClearPassword(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.local.json")
	store, err := NewFileStore(path, RuntimeConfig{
		NewAPIBaseURL: config.DefaultNewAPIBaseURL,
		TimeoutSec:    config.DefaultTimeoutSec,
		Model:         config.DefaultModel,
	})
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}

	enabled := true
	host := " smtp.example.com "
	port := 465
	user := " noreply@example.com "
	password := "smtp-secret-1234567890"
	from := "Lyra Mailer <noreply@example.com>"
	secure := true
	updated, err := store.Update(Update{
		SMTPEnabled:  &enabled,
		SMTPHost:     &host,
		SMTPPort:     &port,
		SMTPUser:     &user,
		SMTPPassword: &password,
		SMTPFrom:     &from,
		SMTPSecure:   &secure,
	})
	if err != nil {
		t.Fatalf("Update(smtp) error = %v", err)
	}
	if !updated.SMTPEnabled || updated.SMTPHost != "smtp.example.com" || updated.SMTPPort != port || updated.SMTPUser != "noreply@example.com" || updated.SMTPPassword != password || updated.SMTPFrom == "" || !updated.SMTPSecure {
		t.Fatalf("smtp settings not normalized/persisted privately: %+v", updated)
	}

	public := store.Public()
	if !public.SMTPPasswordSet || public.SMTPPasswordPreview != "smtp********7890" || !public.Email.SMTPPasswordSet || public.Email.SMTPPasswordPreview != public.SMTPPasswordPreview || !public.SMTPSecure || !public.Email.SMTPSecure {
		t.Fatalf("Public() smtp password status invalid: %+v", public)
	}
	payload, err := json.Marshal(public)
	if err != nil {
		t.Fatalf("Marshal public config: %v", err)
	}
	if strings.Contains(string(payload), password) || strings.Contains(string(payload), `"smtpPassword":`) || strings.Contains(string(payload), `"smtpPass":`) {
		t.Fatalf("public config leaked smtp password: %s", payload)
	}

	reopened, err := NewFileStore(path, DefaultsFromConfig(config.Load()))
	if err != nil {
		t.Fatalf("reopen NewFileStore() error = %v", err)
	}
	if reopened.Get().SMTPPassword != password || !reopened.Get().SMTPSecure {
		t.Fatalf("private smtp password not persisted")
	}
	if _, err := reopened.Update(Update{ClearSMTPPassword: true}); err != nil {
		t.Fatalf("Update(clear smtp password) error = %v", err)
	}
	cleared := reopened.Public()
	if cleared.SMTPPasswordSet || cleared.SMTPPasswordPreview != "" {
		t.Fatalf("smtp password should be cleared: %+v", cleared)
	}
}

func TestFileStoreSystemUpstreamKeysMaskAndClear(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.local.json")
	store, err := NewFileStore(path, RuntimeConfig{
		NewAPIBaseURL: config.DefaultNewAPIBaseURL,
		TimeoutSec:    config.DefaultTimeoutSec,
		Model:         config.DefaultModel,
	})
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}

	imageKey := "sk-system-image-1234567890"
	bananaKey := "sk-system-banana-0987654321"
	updated, err := store.Update(Update{SystemAPIKey: &imageKey, SystemBananaAPIKey: &bananaKey})
	if err != nil {
		t.Fatalf("Update(system keys) error = %v", err)
	}
	if updated.SystemAPIKey != imageKey || updated.SystemBananaAPIKey != bananaKey {
		t.Fatalf("system keys not persisted privately: %+v", updated)
	}
	public := store.Public()
	if !public.SystemAPIKeySet || !public.SystemBananaKeySet || public.SystemAPIKeyPreview == "" || public.SystemBananaKeyPreview == "" {
		t.Fatalf("system key public status invalid: %+v", public)
	}
	payload, err := json.Marshal(public)
	if err != nil {
		t.Fatalf("Marshal public config: %v", err)
	}
	if strings.Contains(string(payload), imageKey) || strings.Contains(string(payload), bananaKey) || strings.Contains(string(payload), `"systemApiKey":`) || strings.Contains(string(payload), `"systemBananaApiKey":`) {
		t.Fatalf("public config leaked system upstream key: %s", payload)
	}

	reopened, err := NewFileStore(path, DefaultsFromConfig(config.Load()))
	if err != nil {
		t.Fatalf("reopen NewFileStore() error = %v", err)
	}
	if reopened.Get().SystemAPIKey != imageKey || reopened.Get().SystemBananaAPIKey != bananaKey {
		t.Fatalf("private system keys not persisted")
	}
	if _, err := reopened.Update(Update{ClearSystemAPIKey: true, ClearSystemBananaKey: true}); err != nil {
		t.Fatalf("Update(clear system keys) error = %v", err)
	}
	cleared := reopened.Public()
	if cleared.SystemAPIKeySet || cleared.SystemBananaKeySet || cleared.SystemAPIKeyPreview != "" || cleared.SystemBananaKeyPreview != "" {
		t.Fatalf("system keys should be cleared: %+v", cleared)
	}
}
