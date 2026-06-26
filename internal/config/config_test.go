package config

import (
	"path/filepath"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	clearConfigEnv(t)
	cfg := Load()
	if cfg.Host != "0.0.0.0" || cfg.Port != 8787 || cfg.Addr != "0.0.0.0:8787" {
		t.Fatalf("unexpected listen defaults: %+v", cfg)
	}
	if cfg.DataDir != "data" || cfg.WebDir != filepath.Join("web", "dist") {
		t.Fatalf("unexpected path defaults: %+v", cfg)
	}
	if cfg.BuiltinNewAPIBaseURL != DefaultNewAPIBaseURL || cfg.DefaultModel != DefaultModel || cfg.DefaultTimeoutSec != DefaultTimeoutSec {
		t.Fatalf("unexpected API defaults: %+v", cfg)
	}
}

func TestLoadEnvOverridesAndBounds(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("LOCAL_IMAGE_HOST", "127.0.0.1")
	t.Setenv("LOCAL_IMAGE_PORT", "9900")
	t.Setenv("LOCAL_IMAGE_DATA_DIR", "tmp/data")
	t.Setenv("LOCAL_IMAGE_WEB_DIR", "tmp/web")
	t.Setenv("NEWAPI_BASE_URL", "https://example.test/v1")
	t.Setenv("NEWAPI_TIMEOUT_SEC", "99999")

	cfg := Load()
	if cfg.Host != "127.0.0.1" || cfg.Port != 9900 || cfg.Addr != "127.0.0.1:9900" {
		t.Fatalf("unexpected listen overrides: %+v", cfg)
	}
	if cfg.DataDir != filepath.Clean("tmp/data") || cfg.WebDir != filepath.Clean("tmp/web") {
		t.Fatalf("unexpected path overrides: %+v", cfg)
	}
	if cfg.BuiltinNewAPIBaseURL != "https://example.test/v1" || cfg.DefaultTimeoutSec != MaxTimeoutSec {
		t.Fatalf("bounds not applied: %+v", cfg)
	}
}

func clearConfigEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"LOCAL_IMAGE_HOST",
		"LOCAL_IMAGE_PORT",
		"LOCAL_IMAGE_DATA_DIR",
		"LOCAL_IMAGE_WEB_DIR",
		"NEWAPI_BASE_URL",
		"NEWAPI_TIMEOUT_SEC",
	} {
		t.Setenv(key, "")
	}
}
