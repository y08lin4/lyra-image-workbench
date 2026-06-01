package spaceconfig

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/y08lin4/lyra-image-workbench/internal/spaces"
)

type Config struct {
	APIKey                   string `json:"apiKey,omitempty"`
	BananaAPIKey             string `json:"bananaApiKey,omitempty"`
	CloudAPIKeyEnabled       bool   `json:"cloudApiKeyEnabled,omitempty"`
	CloudBananaAPIKeyEnabled bool   `json:"cloudBananaApiKeyEnabled,omitempty"`
	DefaultCount             int    `json:"defaultCount,omitempty"`
	DefaultConcurrency       int    `json:"defaultConcurrency,omitempty"`
	AutoUploadPixhost        bool   `json:"autoUploadPixhost,omitempty"`
	UpdatedAt                string `json:"updatedAt"`
}

type PublicConfig struct {
	APIKeySet                bool   `json:"apiKeySet"`
	APIKeyPreview            string `json:"apiKeyPreview"`
	BananaAPIKeySet          bool   `json:"bananaApiKeySet"`
	BananaAPIKeyPreview      string `json:"bananaApiKeyPreview"`
	CloudAPIKeySet           bool   `json:"cloudApiKeySet"`
	CloudAPIKeyPreview       string `json:"cloudApiKeyPreview"`
	CloudBananaAPIKeySet     bool   `json:"cloudBananaApiKeySet"`
	CloudBananaAPIKeyPreview string `json:"cloudBananaApiKeyPreview"`
	DefaultCount             int    `json:"defaultCount"`
	DefaultConcurrency       int    `json:"defaultConcurrency"`
	AutoUploadPixhost        bool   `json:"autoUploadPixhost"`
	UpdatedAt                string `json:"updatedAt"`
}

type Update struct {
	APIKey                 *string `json:"apiKey"`
	BananaAPIKey           *string `json:"bananaApiKey"`
	SaveAPIKeyToCloud      *bool   `json:"saveApiKeyToCloud"`
	SaveBananaKeyToCloud   *bool   `json:"saveBananaKeyToCloud"`
	ClearCloudAPIKey       *bool   `json:"clearCloudApiKey"`
	ClearCloudBananaAPIKey *bool   `json:"clearCloudBananaApiKey"`
	DefaultCount           *int    `json:"defaultCount"`
	DefaultConcurrency     *int    `json:"defaultConcurrency"`
	AutoUploadPixhost      *bool   `json:"autoUploadPixhost"`
}

type Store struct {
	mu     sync.Mutex
	spaces *spaces.FileStore
}

func NewStore(spaceStore *spaces.FileStore) *Store {
	return &Store{spaces: spaceStore}
}

func (s *Store) Public(spaceToken string) (PublicConfig, error) {
	cfg, err := s.Get(spaceToken)
	if err != nil {
		return PublicConfig{}, err
	}
	return toPublic(cfg), nil
}

func (s *Store) Get(spaceToken string) (Config, error) {
	file, err := s.configPath(spaceToken)
	if err != nil {
		return Config{}, err
	}
	data, err := os.ReadFile(file)
	if err != nil {
		if os.IsNotExist(err) {
			return normalize(Config{}), nil
		}
		return Config{}, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("读取个人空间配置失败：%w", err)
	}
	normalized := normalize(cfg)
	if hasUnauthorizedSensitiveKeys(cfg) {
		if err := s.save(spaceToken, normalized); err != nil {
			return Config{}, fmt.Errorf("清理未授权云端 API Key 失败: %w", err)
		}
	}
	return normalized, nil
}

func (s *Store) Update(spaceToken string, update Update) (PublicConfig, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cfg, err := s.Get(spaceToken)
	if err != nil {
		return PublicConfig{}, err
	}
	if update.ClearCloudAPIKey != nil && *update.ClearCloudAPIKey {
		cfg.APIKey = ""
		cfg.CloudAPIKeyEnabled = false
	}
	if update.ClearCloudBananaAPIKey != nil && *update.ClearCloudBananaAPIKey {
		cfg.BananaAPIKey = ""
		cfg.CloudBananaAPIKeyEnabled = false
	}
	if update.APIKey != nil && update.SaveAPIKeyToCloud != nil && *update.SaveAPIKeyToCloud {
		cfg.APIKey = strings.TrimSpace(*update.APIKey)
		cfg.CloudAPIKeyEnabled = cfg.APIKey != ""
	}
	if update.BananaAPIKey != nil && update.SaveBananaKeyToCloud != nil && *update.SaveBananaKeyToCloud {
		cfg.BananaAPIKey = strings.TrimSpace(*update.BananaAPIKey)
		cfg.CloudBananaAPIKeyEnabled = cfg.BananaAPIKey != ""
	}
	if update.DefaultCount != nil {
		cfg.DefaultCount = clamp(*update.DefaultCount, 1, 12, 1)
	}
	if update.DefaultConcurrency != nil {
		cfg.DefaultConcurrency = clamp(*update.DefaultConcurrency, 1, 0, 1)
	}
	if update.AutoUploadPixhost != nil {
		cfg.AutoUploadPixhost = *update.AutoUploadPixhost
	}
	cfg.UpdatedAt = time.Now().Format(time.RFC3339)
	cfg = normalize(cfg)
	if err := s.save(spaceToken, cfg); err != nil {
		return PublicConfig{}, err
	}
	return toPublic(cfg), nil
}

func (s *Store) save(spaceToken string, cfg Config) error {
	file, err := s.configPath(spaceToken)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(file), 0o700); err != nil {
		return err
	}
	payload, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	tmp := fmt.Sprintf("%s.%d.tmp", file, time.Now().UnixNano())
	if err := os.WriteFile(tmp, append(payload, '\n'), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, file)
}

func (s *Store) configPath(spaceToken string) (string, error) {
	spaceDir, err := s.spaces.SpaceDir(spaceToken)
	if err != nil {
		return "", err
	}
	return filepath.Join(spaceDir, "config.json"), nil
}

func normalize(cfg Config) Config {
	cfg.APIKey = strings.TrimSpace(cfg.APIKey)
	cfg.BananaAPIKey = strings.TrimSpace(cfg.BananaAPIKey)
	if !cfg.CloudAPIKeyEnabled {
		cfg.APIKey = ""
	}
	if cfg.APIKey == "" {
		cfg.CloudAPIKeyEnabled = false
	}
	if !cfg.CloudBananaAPIKeyEnabled {
		cfg.BananaAPIKey = ""
	}
	if cfg.BananaAPIKey == "" {
		cfg.CloudBananaAPIKeyEnabled = false
	}
	cfg.DefaultCount = clamp(cfg.DefaultCount, 1, 12, 1)
	cfg.DefaultConcurrency = clamp(cfg.DefaultConcurrency, 1, 0, 1)
	cfg.UpdatedAt = strings.TrimSpace(cfg.UpdatedAt)
	return cfg
}

func hasUnauthorizedSensitiveKeys(cfg Config) bool {
	return (strings.TrimSpace(cfg.APIKey) != "" && !cfg.CloudAPIKeyEnabled) ||
		(strings.TrimSpace(cfg.BananaAPIKey) != "" && !cfg.CloudBananaAPIKeyEnabled)
}

func toPublic(cfg Config) PublicConfig {
	cloudAPIKeySet := cfg.CloudAPIKeyEnabled && cfg.APIKey != ""
	cloudBananaKeySet := cfg.CloudBananaAPIKeyEnabled && cfg.BananaAPIKey != ""
	return PublicConfig{
		APIKeySet:                cloudAPIKeySet,
		APIKeyPreview:            maskSecret(cfg.APIKey),
		BananaAPIKeySet:          cloudBananaKeySet,
		BananaAPIKeyPreview:      maskSecret(cfg.BananaAPIKey),
		CloudAPIKeySet:           cloudAPIKeySet,
		CloudAPIKeyPreview:       maskSecret(cfg.APIKey),
		CloudBananaAPIKeySet:     cloudBananaKeySet,
		CloudBananaAPIKeyPreview: maskSecret(cfg.BananaAPIKey),
		DefaultCount:             cfg.DefaultCount,
		DefaultConcurrency:       cfg.DefaultConcurrency,
		AutoUploadPixhost:        cfg.AutoUploadPixhost,
		UpdatedAt:                cfg.UpdatedAt,
	}
}

func clamp(value int, min int, max int, fallback int) int {
	if value == 0 {
		value = fallback
	}
	if value < min {
		return min
	}
	if max > 0 && value > max {
		return max
	}
	return value
}

func maskSecret(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if len(value) <= 8 {
		return "********"
	}
	return value[:4] + "********" + value[len(value)-4:]
}
