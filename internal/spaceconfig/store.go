package spaceconfig

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/y08lin4/image-Workbench-Localhost-Version/internal/spaces"
)

type Config struct {
	APIKey             string `json:"apiKey,omitempty"`
	BananaAPIKey       string `json:"bananaApiKey,omitempty"`
	DefaultConcurrency int    `json:"defaultConcurrency,omitempty"`
	AutoUploadPixhost  bool   `json:"autoUploadPixhost,omitempty"`
	UpdatedAt          string `json:"updatedAt"`
}

type PublicConfig struct {
	APIKeySet           bool   `json:"apiKeySet"`
	APIKeyPreview       string `json:"apiKeyPreview"`
	BananaAPIKeySet     bool   `json:"bananaApiKeySet"`
	BananaAPIKeyPreview string `json:"bananaApiKeyPreview"`
	DefaultConcurrency  int    `json:"defaultConcurrency"`
	AutoUploadPixhost   bool   `json:"autoUploadPixhost"`
	UpdatedAt           string `json:"updatedAt"`
}

type Update struct {
	APIKey             *string `json:"apiKey"`
	BananaAPIKey       *string `json:"bananaApiKey"`
	DefaultConcurrency *int    `json:"defaultConcurrency"`
	AutoUploadPixhost  *bool   `json:"autoUploadPixhost"`
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
	return normalize(cfg), nil
}

func (s *Store) Update(spaceToken string, update Update) (PublicConfig, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cfg, err := s.Get(spaceToken)
	if err != nil {
		return PublicConfig{}, err
	}
	if update.APIKey != nil {
		cfg.APIKey = strings.TrimSpace(*update.APIKey)
	}
	if update.BananaAPIKey != nil {
		cfg.BananaAPIKey = strings.TrimSpace(*update.BananaAPIKey)
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
	cfg.DefaultConcurrency = clamp(cfg.DefaultConcurrency, 1, 0, 1)
	cfg.UpdatedAt = strings.TrimSpace(cfg.UpdatedAt)
	return cfg
}

func toPublic(cfg Config) PublicConfig {
	return PublicConfig{
		APIKeySet:           cfg.APIKey != "",
		APIKeyPreview:       maskSecret(cfg.APIKey),
		BananaAPIKeySet:     cfg.BananaAPIKey != "",
		BananaAPIKeyPreview: maskSecret(cfg.BananaAPIKey),
		DefaultConcurrency:  cfg.DefaultConcurrency,
		AutoUploadPixhost:   cfg.AutoUploadPixhost,
		UpdatedAt:           cfg.UpdatedAt,
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
		return "••••••••"
	}
	return value[:4] + "••••••••" + value[len(value)-4:]
}
