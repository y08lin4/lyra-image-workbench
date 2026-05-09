package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

const (
	DefaultNewAPIBaseURL = "http://127.0.0.1:3000/v1"
	DefaultModel         = "gpt-image-2"
	DefaultTimeoutSec    = 600
	MinTimeoutSec        = 60
	MaxTimeoutSec        = 3600
)

type Config struct {
	Host                 string
	Port                 int
	Addr                 string
	DataDir              string
	BuiltinNewAPIBaseURL string
	DefaultModel         string
	DefaultTimeoutSec    int
	ReadTimeout          time.Duration
	WriteTimeout         time.Duration
	IdleTimeout          time.Duration
}

func Load() Config {
	host := getenv("LOCAL_IMAGE_HOST", "127.0.0.1")
	port := getenvInt("LOCAL_IMAGE_PORT", 8787)
	return Config{
		Host:                 host,
		Port:                 port,
		Addr:                 fmt.Sprintf("%s:%d", host, port),
		DataDir:              filepath.Clean(getenv("LOCAL_IMAGE_DATA_DIR", "data")),
		BuiltinNewAPIBaseURL: getenv("NEWAPI_BASE_URL", DefaultNewAPIBaseURL),
		DefaultModel:         DefaultModel,
		DefaultTimeoutSec:    getenvBoundedInt("NEWAPI_TIMEOUT_SEC", DefaultTimeoutSec, MinTimeoutSec, MaxTimeoutSec),
		ReadTimeout:          30 * time.Second,
		WriteTimeout:         0, // SSE 和长连接响应不能被固定写超时切断。
		IdleTimeout:          120 * time.Second,
	}
}

func (c Config) RuntimeConfigPath() string {
	return filepath.Join(c.DataDir, "config.local.json")
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
