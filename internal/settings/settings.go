package settings

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/y08lin4/lyra-image-workbench/internal/config"
)

type RuntimeConfig struct {
	SiteName              string   `json:"siteName"`
	NewAPIBaseURL         string   `json:"newApiBaseUrl"`
	PublicBaseURL         string   `json:"publicBaseUrl"`
	DebugEnabled          bool     `json:"debugEnabled"`
	TimeoutSec            int      `json:"timeoutSec"`
	Model                 string   `json:"model"`
	EpayEnabled           bool     `json:"epayEnabled"`
	EpayAPIURL            string   `json:"epayApiUrl"`
	EpayPID               string   `json:"epayPid"`
	EpayKey               string   `json:"epayKey"`
	EpayMethods           []string `json:"epayMethods"`
	CreditPriceCents      int      `json:"creditPriceCents"`
	MinTopUpCredits       int      `json:"minTopUpCredits"`
	ReferralRewardCredits int      `json:"referralRewardCredits"`
	NewUserInitialCredits int      `json:"newUserInitialCredits"`
	DailyFreeCredits      int      `json:"dailyFreeCredits"`
	UpdatedAt             string   `json:"updatedAt"`
}

type PublicRuntimeConfig struct {
	SiteName              string              `json:"siteName"`
	NewAPIBaseURL         string              `json:"newApiBaseUrl"`
	PublicBaseURL         string              `json:"publicBaseUrl"`
	DebugEnabled          bool                `json:"debugEnabled"`
	TimeoutSec            int                 `json:"timeoutSec"`
	Model                 string              `json:"model"`
	ModelLocked           bool                `json:"modelLocked"`
	EpayEnabled           bool                `json:"epayEnabled"`
	EpayAPIURL            string              `json:"epayApiUrl"`
	EpayPID               string              `json:"epayPid"`
	EpayKeySet            bool                `json:"epayKeySet"`
	EpayKeyPreview        string              `json:"epayKeyPreview"`
	EpayMethods           []string            `json:"epayMethods"`
	CreditPriceCents      int                 `json:"creditPriceCents"`
	MinTopUpCredits       int                 `json:"minTopUpCredits"`
	ReferralRewardCredits int                 `json:"referralRewardCredits"`
	NewUserInitialCredits int                 `json:"newUserInitialCredits"`
	DailyFreeCredits      int                 `json:"dailyFreeCredits"`
	Billing               PublicBillingConfig `json:"billing"`
	TimeoutCode           string              `json:"timeoutCode"`
	UpdatedAt             string              `json:"updatedAt"`
	Limits                Limits              `json:"limits"`
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

type Limits struct {
	MinTimeoutSec int `json:"minTimeoutSec"`
	MaxTimeoutSec int `json:"maxTimeoutSec"`
}

type Update struct {
	SiteName              *string  `json:"siteName"`
	NewAPIBaseURL         *string  `json:"newApiBaseUrl"`
	PublicBaseURL         *string  `json:"publicBaseUrl"`
	DebugEnabled          *bool    `json:"debugEnabled"`
	TimeoutSec            *int     `json:"timeoutSec"`
	EpayEnabled           *bool    `json:"epayEnabled"`
	EpayAPIURL            *string  `json:"epayApiUrl"`
	EpayPID               *string  `json:"epayPid"`
	EpayKey               *string  `json:"epayKey"`
	ClearEpayKey          bool     `json:"clearEpayKey"`
	EpayMethods           []string `json:"epayMethods"`
	CreditPriceCents      *int     `json:"creditPriceCents"`
	MinTopUpCredits       *int     `json:"minTopUpCredits"`
	ReferralRewardCredits *int     `json:"referralRewardCredits"`
	NewUserInitialCredits *int     `json:"newUserInitialCredits"`
	DailyFreeCredits      *int     `json:"dailyFreeCredits"`
}

const (
	DefaultSiteName              = "Lyra Image Workbench"
	DefaultCreditPriceCents      = 10
	DefaultMinTopUpCredits       = 10
	DefaultReferralRewardCredits = 0
	DefaultNewUserInitialCredits = 0
	DefaultDailyFreeCredits      = 0
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
		store.current = normalize(merge(store.current, loaded))
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
	if update.SiteName != nil {
		next.SiteName = strings.TrimSpace(*update.SiteName)
	}
	if update.NewAPIBaseURL != nil {
		next.NewAPIBaseURL = strings.TrimSpace(*update.NewAPIBaseURL)
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
	if update.NewUserInitialCredits != nil {
		next.NewUserInitialCredits = *update.NewUserInitialCredits
	}
	if update.DailyFreeCredits != nil {
		next.DailyFreeCredits = *update.DailyFreeCredits
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

func normalize(value RuntimeConfig) RuntimeConfig {
	normalized, err := validate(value)
	if err != nil {
		return RuntimeConfig{
			SiteName:              DefaultSiteName,
			NewAPIBaseURL:         config.DefaultNewAPIBaseURL,
			TimeoutSec:            config.DefaultTimeoutSec,
			Model:                 config.DefaultModel,
			EpayMethods:           defaultEpayMethods(),
			CreditPriceCents:      DefaultCreditPriceCents,
			MinTopUpCredits:       DefaultMinTopUpCredits,
			ReferralRewardCredits: DefaultReferralRewardCredits,
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

func normalizeBaseURL(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", errors.New("NewAPI 请求 URL 不能为空")
	}
	trimmed = strings.TrimRight(trimmed, "/")
	trimmed = strings.TrimSuffix(trimmed, "/images/generations")
	trimmed = strings.TrimSuffix(trimmed, "/images/edits")
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", errors.New("NewAPI 请求 URL 格式无效")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", errors.New("NewAPI 请求 URL 仅支持 http 或 https")
	}
	return strings.TrimRight(parsed.String(), "/"), nil
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
	epayKeyPreview := MaskSecret(value.EpayKey)
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
	return PublicRuntimeConfig{
		SiteName:              value.SiteName,
		NewAPIBaseURL:         value.NewAPIBaseURL,
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
