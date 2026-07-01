package settings

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/mail"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/y08lin4/lyra-image-workbench/internal/config"
)

type RuntimeConfig struct {
	SiteName              string               `json:"siteName"`
	NewAPIBaseURL         string               `json:"newApiBaseUrl"`
	SystemAPIKey          string               `json:"systemApiKey,omitempty"`
	ImageChannels         []ImageChannelConfig `json:"imageChannels"`
	PublicBaseURL         string               `json:"publicBaseUrl"`
	DebugEnabled          bool                 `json:"debugEnabled"`
	TimeoutSec            int                  `json:"timeoutSec"`
	Model                 string               `json:"model"`
	EpayEnabled           bool                 `json:"epayEnabled"`
	EpayAPIURL            string               `json:"epayApiUrl"`
	EpayPID               string               `json:"epayPid"`
	EpayKey               string               `json:"epayKey"`
	EpayMethods           []string             `json:"epayMethods"`
	CreditPriceCents      int                  `json:"creditPriceCents"`
	MinTopUpCredits       int                  `json:"minTopUpCredits"`
	ReferralRewardCredits int                  `json:"referralRewardCredits"`
	SMTPEnabled           bool                 `json:"smtpEnabled"`
	SMTPHost              string               `json:"smtpHost"`
	SMTPPort              int                  `json:"smtpPort"`
	SMTPUser              string               `json:"smtpUser"`
	SMTPPassword          string               `json:"smtpPassword"`
	SMTPFrom              string               `json:"smtpFrom"`
	SMTPSecure            bool                 `json:"smtpSecure"`
	NewUserInitialCredits int                  `json:"newUserInitialCredits"`
	DailyFreeCredits      int                  `json:"dailyFreeCredits"`
	UpdatedAt             string               `json:"updatedAt"`
}

type PublicRuntimeConfig struct {
	SiteName              string                     `json:"siteName"`
	NewAPIBaseURL         string                     `json:"newApiBaseUrl"`
	SystemAPIKeySet       bool                       `json:"systemApiKeySet"`
	SystemAPIKeyPreview   string                     `json:"systemApiKeyPreview"`
	ImageChannels         []PublicImageChannelConfig `json:"imageChannels"`
	PublicBaseURL         string                     `json:"publicBaseUrl"`
	DebugEnabled          bool                       `json:"debugEnabled"`
	TimeoutSec            int                        `json:"timeoutSec"`
	Model                 string                     `json:"model"`
	ModelLocked           bool                       `json:"modelLocked"`
	EpayEnabled           bool                       `json:"epayEnabled"`
	EpayAPIURL            string                     `json:"epayApiUrl"`
	EpayPID               string                     `json:"epayPid"`
	EpayKeySet            bool                       `json:"epayKeySet"`
	EpayKeyPreview        string                     `json:"epayKeyPreview"`
	EpayMethods           []string                   `json:"epayMethods"`
	CreditPriceCents      int                        `json:"creditPriceCents"`
	MinTopUpCredits       int                        `json:"minTopUpCredits"`
	ReferralRewardCredits int                        `json:"referralRewardCredits"`
	SMTPEnabled           bool                       `json:"smtpEnabled"`
	SMTPHost              string                     `json:"smtpHost"`
	SMTPPort              int                        `json:"smtpPort"`
	SMTPUser              string                     `json:"smtpUser"`
	SMTPPasswordSet       bool                       `json:"smtpPasswordSet"`
	SMTPPasswordPreview   string                     `json:"smtpPasswordPreview"`
	SMTPFrom              string                     `json:"smtpFrom"`
	SMTPSecure            bool                       `json:"smtpSecure"`
	Email                 PublicEmailConfig          `json:"email"`
	NewUserInitialCredits int                        `json:"newUserInitialCredits"`
	DailyFreeCredits      int                        `json:"dailyFreeCredits"`
	Billing               PublicBillingConfig        `json:"billing"`
	TimeoutCode           string                     `json:"timeoutCode"`
	UpdatedAt             string                     `json:"updatedAt"`
	Limits                Limits                     `json:"limits"`
}

type PublicBillingConfig struct {
	EpayEnabled           bool     `json:"epayEnabled"`
	EpayAPIURL            string   `json:"epayApiUrl"`
	EpayPID               string   `json:"epayPid"`
	EpayKeySet            bool     `json:"epayKeySet"`
	EpayKeyPreview        string   `json:"epayKeyPreview"`
	EpayMethods           []string `json:"epayMethods"`
	CreditPriceCents      int      `json:"creditPriceCents"`
	MinTopUpCredits       int      `json:"minTopUpCredits"`
	ReferralRewardCredits int      `json:"referralRewardCredits"`
	NewUserInitialCredits int      `json:"newUserInitialCredits"`
	DailyFreeCredits      int      `json:"dailyFreeCredits"`
}

type PublicEmailConfig struct {
	SMTPEnabled         bool   `json:"smtpEnabled"`
	SMTPHost            string `json:"smtpHost"`
	SMTPPort            int    `json:"smtpPort"`
	SMTPUser            string `json:"smtpUser"`
	SMTPPasswordSet     bool   `json:"smtpPasswordSet"`
	SMTPPasswordPreview string `json:"smtpPasswordPreview"`
	SMTPFrom            string `json:"smtpFrom"`
	SMTPSecure          bool   `json:"smtpSecure"`
}

type ImageChannelConfig struct {
	Type    string                    `json:"type"`
	Name    string                    `json:"name"`
	BaseURL string                    `json:"baseURL"`
	Key     string                    `json:"key,omitempty"`
	Enabled bool                      `json:"enabled"`
	Models  []ImageChannelModelConfig `json:"models"`
}

type PublicImageChannelConfig struct {
	Type       string                    `json:"type"`
	Name       string                    `json:"name"`
	BaseURL    string                    `json:"baseURL"`
	KeySet     bool                      `json:"keySet"`
	KeyPreview string                    `json:"keyPreview"`
	Enabled    bool                      `json:"enabled"`
	Models     []ImageChannelModelConfig `json:"models"`
}

type ImageChannelModelConfig struct {
	ID                string `json:"id"`
	Label             string `json:"label"`
	Enabled           bool   `json:"enabled"`
	Price             int    `json:"price"`
	RatioSelectable   bool   `json:"ratioSelectable"`
	DefaultResolution string `json:"defaultResolution"`
}
type Limits struct {
	MinTimeoutSec int `json:"minTimeoutSec"`
	MaxTimeoutSec int `json:"maxTimeoutSec"`
}

type Update struct {
	SiteName              *string              `json:"siteName"`
	NewAPIBaseURL         *string              `json:"newApiBaseUrl"`
	SystemAPIKey          *string              `json:"systemApiKey"`
	ClearSystemAPIKey     bool                 `json:"clearSystemApiKey"`
	ImageChannels         []ImageChannelConfig `json:"imageChannels"`
	PublicBaseURL         *string              `json:"publicBaseUrl"`
	DebugEnabled          *bool                `json:"debugEnabled"`
	TimeoutSec            *int                 `json:"timeoutSec"`
	EpayEnabled           *bool                `json:"epayEnabled"`
	EpayAPIURL            *string              `json:"epayApiUrl"`
	EpayPID               *string              `json:"epayPid"`
	EpayKey               *string              `json:"epayKey"`
	ClearEpayKey          bool                 `json:"clearEpayKey"`
	EpayMethods           []string             `json:"epayMethods"`
	CreditPriceCents      *int                 `json:"creditPriceCents"`
	MinTopUpCredits       *int                 `json:"minTopUpCredits"`
	ReferralRewardCredits *int                 `json:"referralRewardCredits"`
	SMTPEnabled           *bool                `json:"smtpEnabled"`
	SMTPHost              *string              `json:"smtpHost"`
	SMTPPort              *int                 `json:"smtpPort"`
	SMTPUser              *string              `json:"smtpUser"`
	SMTPPassword          *string              `json:"smtpPassword"`
	SMTPPass              *string              `json:"smtpPass"`
	ClearSMTPPassword     bool                 `json:"clearSmtpPassword"`
	ClearSMTPPass         bool                 `json:"clearSmtpPass"`
	SMTPFrom              *string              `json:"smtpFrom"`
	SMTPSecure            *bool                `json:"smtpSecure"`
	NewUserInitialCredits *int                 `json:"newUserInitialCredits"`
	DailyFreeCredits      *int                 `json:"dailyFreeCredits"`
}

const (
	DefaultSiteName              = "Lyra Image Workbench"
	DefaultCreditPriceCents      = 10
	DefaultMinTopUpCredits       = 10
	DefaultReferralRewardCredits = 0
	DefaultSMTPPort              = 587
	DefaultNewUserInitialCredits = 0
	DefaultDailyFreeCredits      = 0
	DefaultImageChannelType      = "openai-compatible"
	DefaultImage4KChannelName    = "image-2-4k"
	DefaultImageModelPrice       = 1
	DefaultImageResolution       = "auto"
	DefaultImage4KResolution     = "auto"
)

type FileStore struct {
	mu      sync.RWMutex
	path    string
	current RuntimeConfig
}

func NewFileStore(path string, defaults RuntimeConfig) (*FileStore, error) {
	store := &FileStore{path: path, current: normalize(defaults)}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err == nil {
		var loaded RuntimeConfig
		if err := json.Unmarshal(data, &loaded); err != nil {
			return nil, fmt.Errorf("读取本机配置失败：%w", err)
		}
		hasLegacyBananaKey := hasLegacyBananaConfig(data)
		store.current = normalize(merge(store.current, loaded))
		if hasLegacyBananaKey {
			if err := store.saveLocked(); err != nil {
				return nil, fmt.Errorf("清理旧 Banana 配置失败：%w", err)
			}
		}
		return store, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if err := store.saveLocked(); err != nil {
		return nil, err
	}
	return store, nil
}

func DefaultsFromConfig(cfg config.Config) RuntimeConfig {
	return normalize(RuntimeConfig{
		SiteName:              DefaultSiteName,
		NewAPIBaseURL:         cfg.BuiltinNewAPIBaseURL,
		TimeoutSec:            cfg.DefaultTimeoutSec,
		Model:                 config.DefaultModel,
		EpayMethods:           defaultEpayMethods(),
		CreditPriceCents:      DefaultCreditPriceCents,
		MinTopUpCredits:       DefaultMinTopUpCredits,
		ReferralRewardCredits: DefaultReferralRewardCredits,
		SMTPPort:              DefaultSMTPPort,
		NewUserInitialCredits: DefaultNewUserInitialCredits,
		DailyFreeCredits:      DefaultDailyFreeCredits,
		UpdatedAt:             time.Now().Format(time.RFC3339),
	})
}

func (s *FileStore) Get() RuntimeConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.current
}

func (s *FileStore) Public() PublicRuntimeConfig {
	return toPublic(s.Get())
}

func (s *FileStore) Update(update Update) (RuntimeConfig, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	next := s.current
	legacyBaseURLChanged := update.NewAPIBaseURL != nil
	legacyKeyChanged := update.SystemAPIKey != nil || update.ClearSystemAPIKey
	if update.SiteName != nil {
		next.SiteName = strings.TrimSpace(*update.SiteName)
	}
	if update.NewAPIBaseURL != nil {
		next.NewAPIBaseURL = strings.TrimSpace(*update.NewAPIBaseURL)
	}
	if update.SystemAPIKey != nil {
		next.SystemAPIKey = strings.TrimSpace(*update.SystemAPIKey)
	}
	if update.ClearSystemAPIKey {
		next.SystemAPIKey = ""
	}
	if update.PublicBaseURL != nil {
		next.PublicBaseURL = strings.TrimSpace(*update.PublicBaseURL)
	}
	if update.DebugEnabled != nil {
		next.DebugEnabled = *update.DebugEnabled
	}
	if update.TimeoutSec != nil {
		next.TimeoutSec = *update.TimeoutSec
	}
	if update.EpayEnabled != nil {
		next.EpayEnabled = *update.EpayEnabled
	}
	if update.EpayAPIURL != nil {
		next.EpayAPIURL = strings.TrimSpace(*update.EpayAPIURL)
	}
	if update.EpayPID != nil {
		next.EpayPID = strings.TrimSpace(*update.EpayPID)
	}
	if update.ClearEpayKey {
		next.EpayKey = ""
	}
	if update.EpayKey != nil {
		next.EpayKey = strings.TrimSpace(*update.EpayKey)
	}
	if update.EpayMethods != nil {
		next.EpayMethods = update.EpayMethods
	}
	if update.CreditPriceCents != nil {
		next.CreditPriceCents = *update.CreditPriceCents
	}
	if update.MinTopUpCredits != nil {
		next.MinTopUpCredits = *update.MinTopUpCredits
	}
	if update.ReferralRewardCredits != nil {
		next.ReferralRewardCredits = *update.ReferralRewardCredits
	}
	if update.SMTPEnabled != nil {
		next.SMTPEnabled = *update.SMTPEnabled
	}
	if update.SMTPHost != nil {
		next.SMTPHost = strings.TrimSpace(*update.SMTPHost)
	}
	if update.SMTPPort != nil {
		next.SMTPPort = *update.SMTPPort
	}
	if update.SMTPUser != nil {
		next.SMTPUser = strings.TrimSpace(*update.SMTPUser)
	}
	if update.ClearSMTPPassword || update.ClearSMTPPass {
		next.SMTPPassword = ""
	}
	if update.SMTPPassword != nil {
		next.SMTPPassword = strings.TrimSpace(*update.SMTPPassword)
	}
	if update.SMTPPass != nil {
		next.SMTPPassword = strings.TrimSpace(*update.SMTPPass)
	}
	if update.SMTPFrom != nil {
		next.SMTPFrom = strings.TrimSpace(*update.SMTPFrom)
	}
	if update.SMTPSecure != nil {
		next.SMTPSecure = *update.SMTPSecure
	}
	if update.NewUserInitialCredits != nil {
		next.NewUserInitialCredits = *update.NewUserInitialCredits
	}
	if update.DailyFreeCredits != nil {
		next.DailyFreeCredits = *update.DailyFreeCredits
	}
	if update.ImageChannels != nil {
		next.ImageChannels = mergeImageChannelsForUpdate(next.ImageChannels, update.ImageChannels)
	}
	if legacyBaseURLChanged || legacyKeyChanged {
		next.ImageChannels = syncLegacyImageChannels(next.ImageChannels, next.NewAPIBaseURL, next.SystemAPIKey)
	}
	normalized, err := validate(next)
	if err != nil {
		return RuntimeConfig{}, err
	}
	normalized.UpdatedAt = time.Now().Format(time.RFC3339)
	s.current = normalized
	if err := s.saveLocked(); err != nil {
		return RuntimeConfig{}, err
	}
	return s.current, nil
}

func (s *FileStore) saveLocked() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	payload, err := json.MarshalIndent(s.current, "", "  ")
	if err != nil {
		return err
	}
	tmp := fmt.Sprintf("%s.%d.tmp", s.path, time.Now().UnixNano())
	if err := os.WriteFile(tmp, append(payload, '\n'), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func merge(base RuntimeConfig, loaded RuntimeConfig) RuntimeConfig {
	if strings.TrimSpace(loaded.SiteName) != "" {
		base.SiteName = loaded.SiteName
	}
	if strings.TrimSpace(loaded.NewAPIBaseURL) != "" {
		base.NewAPIBaseURL = loaded.NewAPIBaseURL
	}
	if strings.TrimSpace(loaded.PublicBaseURL) != "" {
		base.PublicBaseURL = loaded.PublicBaseURL
	}
	if strings.TrimSpace(loaded.SystemAPIKey) != "" {
		base.SystemAPIKey = loaded.SystemAPIKey
	}
	if len(loaded.ImageChannels) > 0 {
		base.ImageChannels = loaded.ImageChannels
	}
	base.DebugEnabled = loaded.DebugEnabled
	if loaded.TimeoutSec != 0 {
		base.TimeoutSec = loaded.TimeoutSec
	}
	base.EpayEnabled = loaded.EpayEnabled
	if strings.TrimSpace(loaded.EpayAPIURL) != "" {
		base.EpayAPIURL = loaded.EpayAPIURL
	}
	if strings.TrimSpace(loaded.EpayPID) != "" {
		base.EpayPID = loaded.EpayPID
	}
	if strings.TrimSpace(loaded.EpayKey) != "" {
		base.EpayKey = loaded.EpayKey
	}
	if len(loaded.EpayMethods) > 0 {
		base.EpayMethods = loaded.EpayMethods
	}
	if loaded.CreditPriceCents != 0 {
		base.CreditPriceCents = loaded.CreditPriceCents
	}
	if loaded.MinTopUpCredits != 0 {
		base.MinTopUpCredits = loaded.MinTopUpCredits
	}
	if loaded.ReferralRewardCredits != 0 {
		base.ReferralRewardCredits = loaded.ReferralRewardCredits
	}
	base.SMTPEnabled = loaded.SMTPEnabled
	if strings.TrimSpace(loaded.SMTPHost) != "" {
		base.SMTPHost = loaded.SMTPHost
	}
	if loaded.SMTPPort != 0 {
		base.SMTPPort = loaded.SMTPPort
	}
	if strings.TrimSpace(loaded.SMTPUser) != "" {
		base.SMTPUser = loaded.SMTPUser
	}
	if strings.TrimSpace(loaded.SMTPPassword) != "" {
		base.SMTPPassword = loaded.SMTPPassword
	}
	if strings.TrimSpace(loaded.SMTPFrom) != "" {
		base.SMTPFrom = loaded.SMTPFrom
	}
	base.SMTPSecure = loaded.SMTPSecure
	if loaded.NewUserInitialCredits != 0 {
		base.NewUserInitialCredits = loaded.NewUserInitialCredits
	}
	if loaded.DailyFreeCredits != 0 {
		base.DailyFreeCredits = loaded.DailyFreeCredits
	}
	if strings.TrimSpace(loaded.UpdatedAt) != "" {
		base.UpdatedAt = loaded.UpdatedAt
	}
	base.Model = config.DefaultModel
	return base
}

func hasLegacyBananaConfig(data []byte) bool {
	var raw map[string]json.RawMessage
	if json.Unmarshal(data, &raw) != nil {
		return false
	}
	_, ok := raw["systemBananaApiKey"]
	return ok
}

func normalize(value RuntimeConfig) RuntimeConfig {
	normalized, err := validate(value)
	if err != nil {
		return RuntimeConfig{
			SiteName:              DefaultSiteName,
			NewAPIBaseURL:         config.DefaultNewAPIBaseURL,
			ImageChannels:         defaultImageChannels(config.DefaultNewAPIBaseURL, ""),
			TimeoutSec:            config.DefaultTimeoutSec,
			Model:                 config.DefaultModel,
			EpayMethods:           defaultEpayMethods(),
			CreditPriceCents:      DefaultCreditPriceCents,
			MinTopUpCredits:       DefaultMinTopUpCredits,
			ReferralRewardCredits: DefaultReferralRewardCredits,
			SMTPPort:              DefaultSMTPPort,
			UpdatedAt:             time.Now().Format(time.RFC3339),
		}
	}
	if strings.TrimSpace(normalized.UpdatedAt) == "" {
		normalized.UpdatedAt = time.Now().Format(time.RFC3339)
	}
	return normalized
}

func validate(value RuntimeConfig) (RuntimeConfig, error) {
	siteName, err := normalizeSiteName(value.SiteName)
	if err != nil {
		return RuntimeConfig{}, err
	}
	if strings.TrimSpace(value.NewAPIBaseURL) == "" {
		if channelBaseURL := firstImageChannelBaseURL(value.ImageChannels); channelBaseURL != "" {
			value.NewAPIBaseURL = channelBaseURL
		}
	}
	baseURL, err := normalizeBaseURL(value.NewAPIBaseURL)
	if err != nil {
		return RuntimeConfig{}, err
	}
	publicBaseURL, err := normalizePublicBaseURL(value.PublicBaseURL)
	if err != nil {
		return RuntimeConfig{}, err
	}
	if value.TimeoutSec < config.MinTimeoutSec || value.TimeoutSec > config.MaxTimeoutSec {
		return RuntimeConfig{}, fmt.Errorf("超时时间必须在 %d 到 %d 秒之间", config.MinTimeoutSec, config.MaxTimeoutSec)
	}
	systemAPIKey := strings.TrimSpace(value.SystemAPIKey)
	if systemAPIKey == "" {
		systemAPIKey = firstImageChannelKey(value.ImageChannels)
	}
	imageChannels, err := normalizeImageChannels(value.ImageChannels, baseURL, systemAPIKey)
	if err != nil {
		return RuntimeConfig{}, err
	}
	baseURL, systemAPIKey = legacyImageChannelFields(imageChannels, baseURL, systemAPIKey)
	epayAPIURL, err := normalizeOptionalHTTPURL(value.EpayAPIURL, "易支付网关地址")
	if err != nil {
		return RuntimeConfig{}, err
	}
	epayPID := strings.TrimSpace(value.EpayPID)
	epayKey := strings.TrimSpace(value.EpayKey)
	epayMethods := normalizeEpayMethods(value.EpayMethods)
	creditPriceCents := value.CreditPriceCents
	if creditPriceCents == 0 {
		creditPriceCents = DefaultCreditPriceCents
	}
	if creditPriceCents < 0 {
		return RuntimeConfig{}, errors.New("次数单价不能小于 0")
	}
	minTopUpCredits := value.MinTopUpCredits
	if minTopUpCredits == 0 {
		minTopUpCredits = DefaultMinTopUpCredits
	}
	if minTopUpCredits < 0 {
		return RuntimeConfig{}, errors.New("最小充值次数不能小于 0")
	}
	if value.ReferralRewardCredits < 0 {
		return RuntimeConfig{}, errors.New("邀请奖励次数不能小于 0")
	}
	smtpHost, err := normalizeSMTPHost(value.SMTPHost)
	if err != nil {
		return RuntimeConfig{}, err
	}
	smtpPort := value.SMTPPort
	if smtpPort == 0 {
		smtpPort = DefaultSMTPPort
	}
	if smtpPort < 1 || smtpPort > 65535 {
		return RuntimeConfig{}, errors.New("SMTP 端口必须在 1 到 65535 之间")
	}
	smtpUser := strings.TrimSpace(value.SMTPUser)
	smtpPassword := strings.TrimSpace(value.SMTPPassword)
	smtpFrom, err := normalizeSMTPFrom(value.SMTPFrom)
	if err != nil {
		return RuntimeConfig{}, err
	}
	if value.SMTPEnabled && (smtpHost == "" || smtpFrom == "") {
		return RuntimeConfig{}, errors.New("启用邮件发件前必须填写 SMTP host 和 from")
	}
	if value.NewUserInitialCredits < 0 {
		return RuntimeConfig{}, errors.New("新用户初始免费次数不能小于 0")
	}
	if value.DailyFreeCredits < 0 {
		return RuntimeConfig{}, errors.New("每日免费次数不能小于 0")
	}
	if value.EpayEnabled && (epayAPIURL == "" || epayPID == "" || epayKey == "" || len(epayMethods) == 0 || creditPriceCents <= 0 || minTopUpCredits <= 0) {
		return RuntimeConfig{}, errors.New("启用易支付前必须填写网关地址、商户 PID、商户 Key、支付方式、次数单价和最小充值次数")
	}
	return RuntimeConfig{
		SiteName:              siteName,
		NewAPIBaseURL:         baseURL,
		PublicBaseURL:         publicBaseURL,
		SystemAPIKey:          systemAPIKey,
		ImageChannels:         imageChannels,
		DebugEnabled:          value.DebugEnabled,
		TimeoutSec:            value.TimeoutSec,
		Model:                 config.DefaultModel,
		EpayEnabled:           value.EpayEnabled,
		EpayAPIURL:            epayAPIURL,
		EpayPID:               epayPID,
		EpayKey:               epayKey,
		EpayMethods:           epayMethods,
		CreditPriceCents:      creditPriceCents,
		MinTopUpCredits:       minTopUpCredits,
		ReferralRewardCredits: value.ReferralRewardCredits,
		SMTPEnabled:           value.SMTPEnabled,
		SMTPHost:              smtpHost,
		SMTPPort:              smtpPort,
		SMTPUser:              smtpUser,
		SMTPPassword:          smtpPassword,
		SMTPFrom:              smtpFrom,
		SMTPSecure:            value.SMTPSecure,
		NewUserInitialCredits: value.NewUserInitialCredits,
		DailyFreeCredits:      value.DailyFreeCredits,
		UpdatedAt:             strings.TrimSpace(value.UpdatedAt),
	}, nil
}

func normalizeSiteName(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return DefaultSiteName, nil
	}
	if len([]rune(trimmed)) > 64 {
		return "", errors.New("站点名称不能超过 64 个字符")
	}
	return trimmed, nil
}
func normalizePublicBaseURL(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", nil
	}
	trimmed = strings.TrimRight(trimmed, "/")
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", errors.New("对外访问域名格式无效，请填写 http:// 或 https:// 开头的完整地址")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", errors.New("对外访问域名仅支持 http 或 https")
	}
	return strings.TrimRight(parsed.String(), "/"), nil
}

func normalizeOptionalHTTPURL(raw string, label string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", nil
	}
	trimmed = strings.TrimRight(trimmed, "/")
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("%s格式无效，请填写 http:// 或 https:// 开头的完整地址", label)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("%s仅支持 http 或 https", label)
	}
	return strings.TrimRight(parsed.String(), "/"), nil
}

func normalizeSMTPHost(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", nil
	}
	if strings.Contains(trimmed, "://") || strings.ContainsAny(trimmed, "/\\") || len(strings.Fields(trimmed)) != 1 {
		return "", errors.New("SMTP host 只填写主机名或 IP，不要包含协议、路径或空格")
	}
	if len(trimmed) > 255 {
		return "", errors.New("SMTP host 不能超过 255 个字符")
	}
	return trimmed, nil
}

func normalizeSMTPFrom(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", nil
	}
	address, err := mail.ParseAddress(trimmed)
	if err != nil || address.Address == "" || !strings.Contains(address.Address, "@") {
		return "", errors.New("SMTP from 邮箱格式无效")
	}
	if len(address.String()) > 320 {
		return "", errors.New("SMTP from 不能超过 320 个字符")
	}
	return address.String(), nil
}

func normalizeBaseURL(raw string) (string, error) {
	return normalizeOpenAICompatibleBaseURL(raw, "NewAPI 请求 URL")
}

func normalizeImageChannelBaseURL(raw string) (string, error) {
	return normalizeOpenAICompatibleBaseURL(raw, "图片渠道 baseURL")
}

func normalizeOpenAICompatibleBaseURL(raw string, label string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("%s不能为空", label)
	}
	trimmed = strings.TrimRight(trimmed, "/")
	trimmed = strings.TrimSuffix(trimmed, "/images/generations")
	trimmed = strings.TrimSuffix(trimmed, "/images/edits")
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("%s格式无效", label)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("%s仅支持 http 或 https", label)
	}
	return strings.TrimRight(parsed.String(), "/"), nil
}

func normalizeImageChannels(channels []ImageChannelConfig, legacyBaseURL string, legacyKey string) ([]ImageChannelConfig, error) {
	if len(channels) == 0 {
		return defaultImageChannels(legacyBaseURL, legacyKey), nil
	}
	normalized := make([]ImageChannelConfig, 0, len(channels)+2)
	seen := make(map[string]bool, len(channels)+2)
	for _, channel := range channels {
		item, err := normalizeImageChannel(channel, legacyBaseURL, legacyKey)
		if err != nil {
			return nil, err
		}
		identity := imageChannelIdentity(item)
		if seen[identity] {
			return nil, fmt.Errorf("图片渠道 %s 重复", item.Name)
		}
		seen[identity] = true
		normalized = append(normalized, item)
	}
	defaultBaseURL, defaultKey := legacyImageChannelFields(normalized, legacyBaseURL, legacyKey)
	for _, channel := range defaultImageChannels(defaultBaseURL, defaultKey) {
		identity := imageChannelIdentity(channel)
		if seen[identity] {
			continue
		}
		seen[identity] = true
		normalized = append(normalized, channel)
	}
	return normalized, nil
}

func normalizeImageChannel(channel ImageChannelConfig, legacyBaseURL string, legacyKey string) (ImageChannelConfig, error) {
	channelType := imageChannelType(channel.Type)
	name := strings.TrimSpace(channel.Name)
	if name == "" {
		return ImageChannelConfig{}, errors.New("图片渠道名称不能为空")
	}
	baseURL := strings.TrimSpace(channel.BaseURL)
	if baseURL == "" && isDefaultImageChannelName(name) {
		baseURL = legacyBaseURL
	}
	normalizedBaseURL, err := normalizeImageChannelBaseURL(baseURL)
	if err != nil {
		return ImageChannelConfig{}, err
	}
	key := strings.TrimSpace(channel.Key)
	if key == "" && isDefaultImageChannelName(name) {
		key = strings.TrimSpace(legacyKey)
	}
	models, err := normalizeImageChannelModels(name, channel.Models)
	if err != nil {
		return ImageChannelConfig{}, err
	}
	return ImageChannelConfig{
		Type:    channelType,
		Name:    name,
		BaseURL: normalizedBaseURL,
		Key:     key,
		Enabled: channel.Enabled,
		Models:  models,
	}, nil
}

func normalizeImageChannelModels(channelName string, models []ImageChannelModelConfig) ([]ImageChannelModelConfig, error) {
	if len(models) == 0 {
		return defaultImageChannelModels(channelName), nil
	}
	normalized := make([]ImageChannelModelConfig, 0, len(models))
	seen := make(map[string]bool, len(models))
	for _, model := range models {
		id := strings.TrimSpace(model.ID)
		if id == "" {
			return nil, errors.New("图片渠道模型 ID 不能为空")
		}
		identity := strings.ToLower(id)
		if seen[identity] {
			return nil, fmt.Errorf("图片渠道模型 %s 重复", id)
		}
		seen[identity] = true
		label := strings.TrimSpace(model.Label)
		if label == "" {
			label = id
		}
		price := model.Price
		if price < 0 {
			return nil, errors.New("图片渠道模型价格不能小于 0")
		}
		if price == 0 {
			price = DefaultImageModelPrice
		}
		defaultResolution := strings.ToLower(strings.TrimSpace(model.DefaultResolution))
		if defaultResolution == "" {
			defaultResolution = defaultImageChannelResolution(channelName)
		}
		normalized = append(normalized, ImageChannelModelConfig{
			ID:                id,
			Label:             label,
			Enabled:           model.Enabled,
			Price:             price,
			RatioSelectable:   model.RatioSelectable,
			DefaultResolution: defaultResolution,
		})
	}
	return normalized, nil
}

func defaultImageChannels(baseURL string, key string) []ImageChannelConfig {
	return []ImageChannelConfig{
		defaultImageChannel(config.DefaultProvider, baseURL, key),
		defaultImageChannel(DefaultImage4KChannelName, baseURL, key),
	}
}

func defaultImageChannel(name string, baseURL string, key string) ImageChannelConfig {
	return ImageChannelConfig{
		Type:    DefaultImageChannelType,
		Name:    name,
		BaseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		Key:     strings.TrimSpace(key),
		Enabled: true,
		Models:  defaultImageChannelModels(name),
	}
}

func defaultImageChannelModels(channelName string) []ImageChannelModelConfig {
	return []ImageChannelModelConfig{
		{
			ID:                config.DefaultModel,
			Label:             config.DefaultModel,
			Enabled:           true,
			Price:             DefaultImageModelPrice,
			RatioSelectable:   defaultImageChannelRatioSelectable(channelName),
			DefaultResolution: defaultImageChannelResolution(channelName),
		},
	}
}

func defaultImageChannelRatioSelectable(channelName string) bool {
	return strings.EqualFold(strings.TrimSpace(channelName), DefaultImage4KChannelName)
}

func defaultImageChannelResolution(channelName string) string {
	if strings.EqualFold(strings.TrimSpace(channelName), DefaultImage4KChannelName) {
		return DefaultImage4KResolution
	}
	return DefaultImageResolution
}

func firstImageChannelBaseURL(channels []ImageChannelConfig) string {
	if channel, ok := findImageChannel(channels, config.DefaultProvider); ok {
		return strings.TrimSpace(channel.BaseURL)
	}
	for _, channel := range channels {
		if strings.TrimSpace(channel.BaseURL) != "" {
			return strings.TrimSpace(channel.BaseURL)
		}
	}
	return ""
}

func firstImageChannelKey(channels []ImageChannelConfig) string {
	if channel, ok := findImageChannel(channels, config.DefaultProvider); ok {
		return strings.TrimSpace(channel.Key)
	}
	for _, channel := range channels {
		if strings.TrimSpace(channel.Key) != "" {
			return strings.TrimSpace(channel.Key)
		}
	}
	return ""
}

func legacyImageChannelFields(channels []ImageChannelConfig, fallbackBaseURL string, fallbackKey string) (string, string) {
	if channel, ok := findImageChannel(channels, config.DefaultProvider); ok {
		return channel.BaseURL, channel.Key
	}
	return fallbackBaseURL, fallbackKey
}

func mergeImageChannelsForUpdate(existing []ImageChannelConfig, incoming []ImageChannelConfig) []ImageChannelConfig {
	existingKeys := make(map[string]string, len(existing))
	for _, channel := range existing {
		if key := strings.TrimSpace(channel.Key); key != "" {
			existingKeys[imageChannelIdentity(channel)] = key
		}
	}
	merged := append([]ImageChannelConfig(nil), incoming...)
	for i := range merged {
		if strings.TrimSpace(merged[i].Key) != "" {
			continue
		}
		if key := existingKeys[imageChannelIdentity(merged[i])]; key != "" {
			merged[i].Key = key
		}
	}
	return merged
}

func syncLegacyImageChannels(channels []ImageChannelConfig, legacyBaseURL string, legacyKey string) []ImageChannelConfig {
	if len(channels) == 0 {
		return channels
	}
	synced := append([]ImageChannelConfig(nil), channels...)
	for i := range synced {
		if imageChannelType(synced[i].Type) == DefaultImageChannelType && isDefaultImageChannelName(synced[i].Name) {
			synced[i].BaseURL = legacyBaseURL
			synced[i].Key = legacyKey
		}
	}
	return synced
}

func findImageChannel(channels []ImageChannelConfig, name string) (ImageChannelConfig, bool) {
	target := strings.ToLower(strings.TrimSpace(name))
	for _, channel := range channels {
		if imageChannelType(channel.Type) == DefaultImageChannelType && strings.ToLower(strings.TrimSpace(channel.Name)) == target {
			return channel, true
		}
	}
	return ImageChannelConfig{}, false
}

func imageChannelIdentity(channel ImageChannelConfig) string {
	return imageChannelType(channel.Type) + "\x00" + strings.ToLower(strings.TrimSpace(channel.Name))
}

func imageChannelType(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		return DefaultImageChannelType
	}
	return normalized
}

func isDefaultImageChannelName(name string) bool {
	normalized := strings.ToLower(strings.TrimSpace(name))
	return normalized == config.DefaultProvider || normalized == DefaultImage4KChannelName
}
func normalizeEpayMethods(methods []string) []string {
	seen := make(map[string]bool)
	normalized := make([]string, 0, len(methods))
	for _, method := range methods {
		method = strings.ToLower(strings.TrimSpace(method))
		if method == "" || seen[method] {
			continue
		}
		seen[method] = true
		normalized = append(normalized, method)
	}
	if len(normalized) == 0 {
		return defaultEpayMethods()
	}
	return normalized
}

func defaultEpayMethods() []string {
	return []string{"alipay", "wxpay"}
}

func SystemAPIKeyForProvider(value RuntimeConfig, provider string) string {
	provider = normalizeImageChannelProvider(provider)
	if channel, ok := findImageChannel(value.ImageChannels, provider); ok {
		if !channel.Enabled {
			return ""
		}
		if key := strings.TrimSpace(channel.Key); key != "" {
			return key
		}
	}
	return strings.TrimSpace(value.SystemAPIKey)
}

func SystemBaseURLForProvider(value RuntimeConfig, provider string) string {
	provider = normalizeImageChannelProvider(provider)
	if channel, ok := findImageChannel(value.ImageChannels, provider); ok {
		if !channel.Enabled {
			return ""
		}
		if baseURL := strings.TrimSpace(channel.BaseURL); baseURL != "" {
			return baseURL
		}
	}
	return strings.TrimSpace(value.NewAPIBaseURL)
}
func HasSystemAPIKeyForProvider(value RuntimeConfig, provider string) bool {
	return SystemAPIKeyForProvider(value, provider) != ""
}

func HasAnySystemAPIKey(value RuntimeConfig) bool {
	if strings.TrimSpace(value.SystemAPIKey) != "" {
		return true
	}
	for _, channel := range value.ImageChannels {
		if channel.Enabled && strings.TrimSpace(channel.Key) != "" {
			return true
		}
	}
	return false
}

func normalizeImageChannelProvider(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "", config.DefaultProvider, "image2", config.DefaultModel:
		return config.DefaultProvider
	case DefaultImage4KChannelName:
		return DefaultImage4KChannelName
	default:
		return strings.TrimSpace(provider)
	}
}
func MaskSecret(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if len(value) <= 8 {
		return "********"
	}
	return value[:4] + "********" + value[len(value)-4:]
}

func toPublic(value RuntimeConfig) PublicRuntimeConfig {
	systemAPIKeyPreview := MaskSecret(value.SystemAPIKey)
	imageChannels := publicImageChannels(value.ImageChannels)
	epayKeyPreview := MaskSecret(value.EpayKey)
	smtpPasswordPreview := MaskSecret(value.SMTPPassword)
	billing := PublicBillingConfig{
		EpayEnabled:           value.EpayEnabled,
		EpayAPIURL:            value.EpayAPIURL,
		EpayPID:               value.EpayPID,
		EpayKeySet:            value.EpayKey != "",
		EpayKeyPreview:        epayKeyPreview,
		EpayMethods:           append([]string{}, value.EpayMethods...),
		CreditPriceCents:      value.CreditPriceCents,
		MinTopUpCredits:       value.MinTopUpCredits,
		ReferralRewardCredits: value.ReferralRewardCredits,
		NewUserInitialCredits: value.NewUserInitialCredits,
		DailyFreeCredits:      value.DailyFreeCredits,
	}
	email := PublicEmailConfig{
		SMTPEnabled:         value.SMTPEnabled,
		SMTPHost:            value.SMTPHost,
		SMTPPort:            value.SMTPPort,
		SMTPUser:            value.SMTPUser,
		SMTPPasswordSet:     value.SMTPPassword != "",
		SMTPPasswordPreview: smtpPasswordPreview,
		SMTPFrom:            value.SMTPFrom,
		SMTPSecure:          value.SMTPSecure,
	}
	return PublicRuntimeConfig{
		SiteName:              value.SiteName,
		NewAPIBaseURL:         value.NewAPIBaseURL,
		SystemAPIKeySet:       value.SystemAPIKey != "",
		SystemAPIKeyPreview:   systemAPIKeyPreview,
		ImageChannels:         imageChannels,
		PublicBaseURL:         value.PublicBaseURL,
		DebugEnabled:          value.DebugEnabled,
		TimeoutSec:            value.TimeoutSec,
		Model:                 config.DefaultModel,
		ModelLocked:           true,
		EpayEnabled:           billing.EpayEnabled,
		EpayAPIURL:            billing.EpayAPIURL,
		EpayPID:               billing.EpayPID,
		EpayKeySet:            billing.EpayKeySet,
		EpayKeyPreview:        billing.EpayKeyPreview,
		EpayMethods:           append([]string{}, billing.EpayMethods...),
		CreditPriceCents:      billing.CreditPriceCents,
		MinTopUpCredits:       billing.MinTopUpCredits,
		ReferralRewardCredits: billing.ReferralRewardCredits,
		SMTPEnabled:           email.SMTPEnabled,
		SMTPHost:              email.SMTPHost,
		SMTPPort:              email.SMTPPort,
		SMTPUser:              email.SMTPUser,
		SMTPPasswordSet:       email.SMTPPasswordSet,
		SMTPPasswordPreview:   email.SMTPPasswordPreview,
		SMTPFrom:              email.SMTPFrom,
		SMTPSecure:            email.SMTPSecure,
		Email:                 email,
		NewUserInitialCredits: billing.NewUserInitialCredits,
		DailyFreeCredits:      billing.DailyFreeCredits,
		Billing:               billing,
		TimeoutCode:           fmt.Sprintf("TIMEOUT_%dS", value.TimeoutSec),
		UpdatedAt:             value.UpdatedAt,
		Limits: Limits{
			MinTimeoutSec: config.MinTimeoutSec,
			MaxTimeoutSec: config.MaxTimeoutSec,
		},
	}
}

func publicImageChannels(channels []ImageChannelConfig) []PublicImageChannelConfig {
	public := make([]PublicImageChannelConfig, 0, len(channels))
	for _, channel := range channels {
		key := strings.TrimSpace(channel.Key)
		public = append(public, PublicImageChannelConfig{
			Type:       channel.Type,
			Name:       channel.Name,
			BaseURL:    channel.BaseURL,
			KeySet:     key != "",
			KeyPreview: MaskSecret(key),
			Enabled:    channel.Enabled,
			Models:     append([]ImageChannelModelConfig{}, channel.Models...),
		})
	}
	return public
}
