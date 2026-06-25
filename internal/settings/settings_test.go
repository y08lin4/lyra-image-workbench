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
	updated, err := store.Update(Update{NewAPIBaseURL: &baseURL, PublicBaseURL: &publicURL, DebugEnabled: &debugEnabled, TimeoutSec: &timeout})
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
