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
	NewAPIBaseURL string `json:"newApiBaseUrl"`
	PublicBaseURL string `json:"publicBaseUrl"`
	DebugEnabled  bool   `json:"debugEnabled"`
	TimeoutSec    int    `json:"timeoutSec"`
	Model         string `json:"model"`
	UpdatedAt     string `json:"updatedAt"`
}

type PublicRuntimeConfig struct {
	NewAPIBaseURL string `json:"newApiBaseUrl"`
	PublicBaseURL string `json:"publicBaseUrl"`
	DebugEnabled  bool   `json:"debugEnabled"`
	TimeoutSec    int    `json:"timeoutSec"`
	Model         string `json:"model"`
	ModelLocked   bool   `json:"modelLocked"`
	TimeoutCode   string `json:"timeoutCode"`
	UpdatedAt     string `json:"updatedAt"`
	Limits        Limits `json:"limits"`
}

type Limits struct {
	MinTimeoutSec int `json:"minTimeoutSec"`
	MaxTimeoutSec int `json:"maxTimeoutSec"`
}

type Update struct {
	NewAPIBaseURL *string `json:"newApiBaseUrl"`
	PublicBaseURL *string `json:"publicBaseUrl"`
	DebugEnabled  *bool   `json:"debugEnabled"`
	TimeoutSec    *int    `json:"timeoutSec"`
}

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
		NewAPIBaseURL: cfg.BuiltinNewAPIBaseURL,
		TimeoutSec:    cfg.DefaultTimeoutSec,
		Model:         config.DefaultModel,
		UpdatedAt:     time.Now().Format(time.RFC3339),
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
			NewAPIBaseURL: config.DefaultNewAPIBaseURL,
			TimeoutSec:    config.DefaultTimeoutSec,
			Model:         config.DefaultModel,
			UpdatedAt:     time.Now().Format(time.RFC3339),
		}
	}
	if strings.TrimSpace(normalized.UpdatedAt) == "" {
		normalized.UpdatedAt = time.Now().Format(time.RFC3339)
	}
	return normalized
}

func validate(value RuntimeConfig) (RuntimeConfig, error) {
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
	return RuntimeConfig{
		NewAPIBaseURL: baseURL,
		PublicBaseURL: publicBaseURL,
		DebugEnabled:  value.DebugEnabled,
		TimeoutSec:    value.TimeoutSec,
		Model:         config.DefaultModel,
		UpdatedAt:     strings.TrimSpace(value.UpdatedAt),
	}, nil
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

func toPublic(value RuntimeConfig) PublicRuntimeConfig {
	return PublicRuntimeConfig{
		NewAPIBaseURL: value.NewAPIBaseURL,
		PublicBaseURL: value.PublicBaseURL,
		DebugEnabled:  value.DebugEnabled,
		TimeoutSec:    value.TimeoutSec,
		Model:         config.DefaultModel,
		ModelLocked:   true,
		TimeoutCode:   fmt.Sprintf("TIMEOUT_%dS", value.TimeoutSec),
		UpdatedAt:     value.UpdatedAt,
		Limits: Limits{
			MinTimeoutSec: config.MinTimeoutSec,
			MaxTimeoutSec: config.MaxTimeoutSec,
		},
	}
}
