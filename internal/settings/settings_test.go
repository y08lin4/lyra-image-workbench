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

func TestFileStoreSystemUpstreamKeyMaskAndClear(t *testing.T) {
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
	updated, err := store.Update(Update{SystemAPIKey: &imageKey})
	if err != nil {
		t.Fatalf("Update(system key) error = %v", err)
	}
	if updated.SystemAPIKey != imageKey {
		t.Fatalf("system key not persisted privately: %+v", updated)
	}
	public := store.Public()
	if !public.SystemAPIKeySet || public.SystemAPIKeyPreview == "" {
		t.Fatalf("system key public status invalid: %+v", public)
	}
	payload, err := json.Marshal(public)
	if err != nil {
		t.Fatalf("Marshal public config: %v", err)
	}
	if strings.Contains(string(payload), imageKey) || strings.Contains(string(payload), `"systemApiKey":`) || strings.Contains(string(payload), "systemBananaApiKey") {
		t.Fatalf("public config leaked system upstream key: %s", payload)
	}

	reopened, err := NewFileStore(path, DefaultsFromConfig(config.Load()))
	if err != nil {
		t.Fatalf("reopen NewFileStore() error = %v", err)
	}
	if reopened.Get().SystemAPIKey != imageKey {
		t.Fatalf("private system key not persisted")
	}
	if _, err := reopened.Update(Update{ClearSystemAPIKey: true}); err != nil {
		t.Fatalf("Update(clear system key) error = %v", err)
	}
	cleared := reopened.Public()
	if cleared.SystemAPIKeySet || cleared.SystemAPIKeyPreview != "" {
		t.Fatalf("system key should be cleared: %+v", cleared)
	}
}

func TestFileStoreDefaultImageChannelsMirrorLegacySettings(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.local.json")
	systemKey := "sk-system-channel-1234567890"
	store, err := NewFileStore(path, RuntimeConfig{
		NewAPIBaseURL: "http://127.0.0.1:3000/v1/images/generations",
		SystemAPIKey:  " " + systemKey + " ",
		TimeoutSec:    config.DefaultTimeoutSec,
		Model:         config.DefaultModel,
	})
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}

	cfg := store.Get()
	if cfg.NewAPIBaseURL != "http://127.0.0.1:3000/v1" || cfg.SystemAPIKey != systemKey {
		t.Fatalf("legacy fields not normalized: %+v", cfg)
	}
	if len(cfg.ImageChannels) != 2 {
		t.Fatalf("ImageChannels length = %d, want 2: %+v", len(cfg.ImageChannels), cfg.ImageChannels)
	}
	image2, ok := imageChannelByName(cfg.ImageChannels, config.DefaultProvider)
	if !ok {
		t.Fatalf("default image-2 channel missing: %+v", cfg.ImageChannels)
	}
	assertImageChannel(t, image2, config.DefaultProvider, "http://127.0.0.1:3000/v1", systemKey, false, DefaultImageResolution)
	image4K, ok := imageChannelByName(cfg.ImageChannels, DefaultImage4KChannelName)
	if !ok {
		t.Fatalf("default image-2-4k channel missing: %+v", cfg.ImageChannels)
	}
	assertImageChannel(t, image4K, DefaultImage4KChannelName, "http://127.0.0.1:3000/v1", systemKey, true, DefaultImage4KResolution)

	public := store.Public()
	publicImage2, ok := publicImageChannelByName(public.ImageChannels, config.DefaultProvider)
	if !ok || !publicImage2.KeySet || publicImage2.KeyPreview != MaskSecret(systemKey) {
		t.Fatalf("public image-2 channel key status invalid: %+v", public.ImageChannels)
	}
	payload, err := json.Marshal(public)
	if err != nil {
		t.Fatalf("Marshal public config: %v", err)
	}
	if strings.Contains(string(payload), systemKey) || strings.Contains(string(payload), `"key":`) {
		t.Fatalf("public config leaked image channel key: %s", payload)
	}
}

func TestFileStoreLegacyUpstreamFieldsSyncDefaultImageChannels(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.local.json")
	store, err := NewFileStore(path, RuntimeConfig{
		NewAPIBaseURL: config.DefaultNewAPIBaseURL,
		TimeoutSec:    config.DefaultTimeoutSec,
		Model:         config.DefaultModel,
	})
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}

	baseURL := "https://upstream.example/v1/images/edits"
	key := "sk-legacy-sync-1234567890"
	updated, err := store.Update(Update{NewAPIBaseURL: &baseURL, SystemAPIKey: &key})
	if err != nil {
		t.Fatalf("Update(legacy upstream) error = %v", err)
	}
	if updated.NewAPIBaseURL != "https://upstream.example/v1" || updated.SystemAPIKey != key {
		t.Fatalf("legacy fields not persisted: %+v", updated)
	}
	for _, name := range []string{config.DefaultProvider, DefaultImage4KChannelName} {
		channel, ok := imageChannelByName(updated.ImageChannels, name)
		if !ok {
			t.Fatalf("channel %s missing: %+v", name, updated.ImageChannels)
		}
		if channel.BaseURL != updated.NewAPIBaseURL || channel.Key != key {
			t.Fatalf("channel %s not synced with legacy fields: %+v", name, channel)
		}
	}
	if got := SystemAPIKeyForProvider(updated, DefaultImage4KChannelName); got != key {
		t.Fatalf("SystemAPIKeyForProvider(image-2-4k) = %q", got)
	}

	cleared, err := store.Update(Update{ClearSystemAPIKey: true})
	if err != nil {
		t.Fatalf("Update(clear system key) error = %v", err)
	}
	if HasAnySystemAPIKey(cleared) || SystemAPIKeyForProvider(cleared, config.DefaultProvider) != "" {
		t.Fatalf("system key should be cleared from legacy and channels: %+v", cleared)
	}
	for _, channel := range cleared.ImageChannels {
		if channel.Key != "" {
			t.Fatalf("channel key should be cleared: %+v", channel)
		}
	}
}

func TestFileStoreImageChannelsUpdateNormalizesAndBackfillsLegacyFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.local.json")
	store, err := NewFileStore(path, RuntimeConfig{
		NewAPIBaseURL: config.DefaultNewAPIBaseURL,
		TimeoutSec:    config.DefaultTimeoutSec,
		Model:         config.DefaultModel,
	})
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}

	channelKey := "sk-channel-update-1234567890"
	updated, err := store.Update(Update{ImageChannels: []ImageChannelConfig{
		{
			Type:    " OpenAI-Compatible ",
			Name:    " image-2 ",
			BaseURL: "https://channel.example/v1/images/generations",
			Key:     " " + channelKey + " ",
			Enabled: true,
			Models: []ImageChannelModelConfig{
				{ID: " gpt-image-2 ", Enabled: true, RatioSelectable: true, DefaultResolution: " 4K "},
			},
		},
	}})
	if err != nil {
		t.Fatalf("Update(image channels) error = %v", err)
	}
	if updated.NewAPIBaseURL != "https://channel.example/v1" || updated.SystemAPIKey != channelKey {
		t.Fatalf("legacy fields not backfilled from image channel: %+v", updated)
	}
	image2, ok := imageChannelByName(updated.ImageChannels, config.DefaultProvider)
	if !ok {
		t.Fatalf("image-2 channel missing: %+v", updated.ImageChannels)
	}
	if image2.Type != DefaultImageChannelType || image2.Name != config.DefaultProvider || image2.BaseURL != updated.NewAPIBaseURL || image2.Key != channelKey {
		t.Fatalf("image-2 channel not normalized: %+v", image2)
	}
	if len(image2.Models) != 1 || image2.Models[0].ID != config.DefaultModel || image2.Models[0].Label != config.DefaultModel || image2.Models[0].Price != DefaultImageModelPrice || image2.Models[0].DefaultResolution != "4k" {
		t.Fatalf("image-2 model not normalized: %+v", image2.Models)
	}
	image4K, ok := imageChannelByName(updated.ImageChannels, DefaultImage4KChannelName)
	if !ok {
		t.Fatalf("image-2-4k channel should be backfilled: %+v", updated.ImageChannels)
	}
	assertImageChannel(t, image4K, DefaultImage4KChannelName, updated.NewAPIBaseURL, channelKey, true, DefaultImage4KResolution)
}

func assertImageChannel(t *testing.T, channel ImageChannelConfig, name string, baseURL string, key string, ratioSelectable bool, defaultResolution string) {
	t.Helper()
	if channel.Type != DefaultImageChannelType || channel.Name != name || channel.BaseURL != baseURL || channel.Key != key || !channel.Enabled {
		t.Fatalf("channel %s not normalized: %+v", name, channel)
	}
	if len(channel.Models) != 1 {
		t.Fatalf("channel %s model count = %d", name, len(channel.Models))
	}
	model := channel.Models[0]
	if model.ID != config.DefaultModel || model.Label != config.DefaultModel || !model.Enabled || model.Price != DefaultImageModelPrice || model.RatioSelectable != ratioSelectable || model.DefaultResolution != defaultResolution {
		t.Fatalf("channel %s model invalid: %+v", name, model)
	}
}

func imageChannelByName(channels []ImageChannelConfig, name string) (ImageChannelConfig, bool) {
	for _, channel := range channels {
		if channel.Name == name {
			return channel, true
		}
	}
	return ImageChannelConfig{}, false
}

func publicImageChannelByName(channels []PublicImageChannelConfig, name string) (PublicImageChannelConfig, bool) {
	for _, channel := range channels {
		if channel.Name == name {
			return channel, true
		}
	}
	return PublicImageChannelConfig{}, false
}
