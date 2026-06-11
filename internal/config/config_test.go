package config

import "testing"

func TestLoadGIFDefaults(t *testing.T) {
	clearGIFEnv(t)
	cfg := Load()
	if !cfg.GIFEnabled || cfg.FFmpegBin != "ffmpeg" || cfg.GIFWorkDir != "data\\gif_work" && cfg.GIFWorkDir != "data/gif_work" {
		t.Fatalf("unexpected GIF defaults: %+v", cfg)
	}
	if cfg.GIFMaxFrames != 24 || cfg.GIFMaxFPS != 15 || cfg.GIFMaxSize != 1024 || cfg.GIFRenderTimeoutSec != 60 {
		t.Fatalf("unexpected GIF limits: %+v", cfg)
	}
}

func TestLoadGIFEnvOverridesAndBounds(t *testing.T) {
	clearGIFEnv(t)
	t.Setenv("GIF_ENABLED", "0")
	t.Setenv("FFMPEG_BIN", "/opt/bin/ffmpeg")
	t.Setenv("GIF_WORK_DIR", "tmp/gif")
	t.Setenv("GIF_MAX_FRAMES", "500")
	t.Setenv("GIF_MAX_FPS", "0")
	t.Setenv("GIF_MAX_SIZE", "64")
	t.Setenv("GIF_RENDER_TIMEOUT_SEC", "999")
	cfg := Load()
	if cfg.GIFEnabled {
		t.Fatalf("GIF_ENABLED=0 should disable GIF")
	}
	if cfg.FFmpegBin != "/opt/bin/ffmpeg" || cfg.GIFWorkDir != "tmp\\gif" && cfg.GIFWorkDir != "tmp/gif" {
		t.Fatalf("unexpected path overrides: %+v", cfg)
	}
	if cfg.GIFMaxFrames != 120 || cfg.GIFMaxFPS != 15 || cfg.GIFMaxSize != 128 || cfg.GIFRenderTimeoutSec != 600 {
		t.Fatalf("bounds not applied: %+v", cfg)
	}
}

func TestLoadGIFBoolParsing(t *testing.T) {
	truthy := []string{"1", "true", "yes", "on"}
	for _, value := range truthy {
		clearGIFEnv(t)
		t.Setenv("GIF_ENABLED", value)
		if !Load().GIFEnabled {
			t.Fatalf("GIF_ENABLED=%q should be true", value)
		}
	}
	falsy := []string{"0", "false", "no", "off"}
	for _, value := range falsy {
		clearGIFEnv(t)
		t.Setenv("GIF_ENABLED", value)
		if Load().GIFEnabled {
			t.Fatalf("GIF_ENABLED=%q should be false", value)
		}
	}
	clearGIFEnv(t)
	t.Setenv("GIF_ENABLED", "not-a-bool")
	if !Load().GIFEnabled {
		t.Fatalf("invalid bool should fall back to default true")
	}
}

func clearGIFEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{"GIF_ENABLED", "FFMPEG_BIN", "GIF_WORK_DIR", "GIF_MAX_FRAMES", "GIF_MAX_FPS", "GIF_MAX_SIZE", "GIF_RENDER_TIMEOUT_SEC"} {
		t.Setenv(key, "")
	}
}
