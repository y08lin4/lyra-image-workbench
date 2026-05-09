package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Host         string
	Port         int
	Addr         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

func Load() Config {
	host := getenv("LOCAL_IMAGE_HOST", "127.0.0.1")
	port := getenvInt("LOCAL_IMAGE_PORT", 8787)
	return Config{
		Host:         host,
		Port:         port,
		Addr:         fmt.Sprintf("%s:%d", host, port),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 0, // SSE 和长连接响应不能被固定写超时切断。
		IdleTimeout:  120 * time.Second,
	}
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
