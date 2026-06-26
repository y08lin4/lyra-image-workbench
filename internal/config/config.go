package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultNewAPIBaseURL    = "http://127.0.0.1:3000/v1"
	DefaultModel            = "gpt-image-2"
	DefaultProvider         = "image-2"
	BananaProvider          = "banana"
	DefaultBananaModel      = "gemini-3.1-flash-image-preview"
	DefaultPromptModel      = "gpt-5.5"
	DefaultPromptTimeoutSec = 180
	DefaultTimeoutSec       = 600
	MinTimeoutSec           = 60
	MaxTimeoutSec           = 3600
)

var bananaModels = []string{
	"gemini-3.1-flash-image-preview",
	"gemini-3.1-flash-image-preview-16x9-4k",
	"gemini-3.1-flash-image-preview-9x16-4k",
	"gemini-3.1-flash-image-preview-16x9-2k",
	"gemini-3.1-flash-image-preview-9x16-2k",
	"gemini-3.1-flash-image-preview-2k",
	"gemini-3.1-flash-image-preview-4k",
	"gemini-3.1-flash-image-preview-4x3-4k",
	"gemini-3.1-flash-image-preview-4x3-2k",
	"gemini-3.1-flash-image-preview-3x2-4k",
	"gemini-3.1-flash-image-preview-3x2-2k",
	"gemini-3.1-flash-image-preview-2x3-4k",
	"gemini-3.1-flash-image-preview-2x3-2k",
	"gemini-3.1-flash-image-preview-1x1-4k",
	"gemini-3.1-flash-image-preview-3x4-2k",
	"gemini-3.1-flash-image-preview-3x4-4k",
	"gemini-3.1-flash-image-preview-1x1-2k",
}

func BananaModels() []string {
	return append([]string{}, bananaModels...)
}

func IsBananaModel(model string) bool {
	for _, item := range bananaModels {
		if model == item {
			return true
		}
	}
	return false
}

type Config struct {
	Host                 string
	Port                 int
	Addr                 string
	DataDir              string
	WebDir               string
	BuiltinNewAPIBaseURL string
	DefaultModel         string
	DefaultTimeoutSec    int
	ReadTimeout          time.Duration
	WriteTimeout         time.Duration
	IdleTimeout          time.Duration
	AdminSetupToken      string
}

func Load() Config {
	host := getenv("LOCAL_IMAGE_HOST", "0.0.0.0")
	port := getenvInt("LOCAL_IMAGE_PORT", 8787)
	return Config{
		Host:                 host,
		Port:                 port,
		Addr:                 fmt.Sprintf("%s:%d", host, port),
		DataDir:              filepath.Clean(getenv("LOCAL_IMAGE_DATA_DIR", "data")),
		WebDir:               filepath.Clean(getenv("LOCAL_IMAGE_WEB_DIR", filepath.Join("web", "dist"))),
		BuiltinNewAPIBaseURL: getenv("NEWAPI_BASE_URL", DefaultNewAPIBaseURL),
		DefaultModel:         DefaultModel,
		DefaultTimeoutSec:    getenvBoundedInt("NEWAPI_TIMEOUT_SEC", DefaultTimeoutSec, MinTimeoutSec, MaxTimeoutSec),
		ReadTimeout:          30 * time.Second,
		WriteTimeout:         0, // SSE 和长连接响应不能被固定写超时切断。
		IdleTimeout:          120 * time.Second,
		AdminSetupToken:      strings.TrimSpace(os.Getenv("LOCAL_IMAGE_ADMIN_SETUP_TOKEN")),
	}
}

func (c Config) RuntimeConfigPath() string {
	return filepath.Join(c.DataDir, "config.local.json")
}

func (c Config) AdminAuthPath() string {
	return filepath.Join(c.DataDir, "admin.auth.json")
}

func (c Config) UsersPath() string {
	return filepath.Join(c.DataDir, "users.json")
}

func (c Config) APIKeysPath() string {
	return filepath.Join(c.DataDir, "api_keys.json")
}

func getenv(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func getenvInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func getenvBool(key string, fallback bool) bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	if value == "" {
		return fallback
	}
	switch value {
	case "1", "true", "yes", "y", "on":
		return true
	case "0", "false", "no", "n", "off":
		return false
	default:
		return fallback
	}
}

func getenvBoundedInt(key string, fallback int, minValue int, maxValue int) int {
	parsed := getenvInt(key, fallback)
	if parsed < minValue {
		return minValue
	}
	if parsed > maxValue {
		return maxValue
	}
	return parsed
}
