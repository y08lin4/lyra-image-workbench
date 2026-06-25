package gifrender

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/y08lin4/lyra-image-workbench/internal/config"
)

const (
	DefaultWidth             = 512
	DefaultFPS               = 8
	FFmpegMinimumSafeVersion = "8.1.2"
)

var ErrRendererUnavailable = errors.New("GIF_RENDERER_UNAVAILABLE")

type Config struct {
	Enabled   bool
	FFmpegBin string
	WorkDir   string
	MaxFrames int
	MaxFPS    int
	MaxSize   int
	Timeout   time.Duration
}

type Limits struct {
	MaxFrames int `json:"maxFrames"`
	MaxFPS    int `json:"maxFPS"`
	MaxSize   int `json:"maxSize"`
}

type FFmpegStatus struct {
	Enabled        bool   `json:"enabled"`
	Available      bool   `json:"available"`
	Safe           bool   `json:"safe"`
	Bin            string `json:"bin"`
	Version        string `json:"version,omitempty"`
	MinimumVersion string `json:"minimumVersion"`
	Code           string `json:"code,omitempty"`
	Message        string `json:"message,omitempty"`
}

type Renderer interface {
	RenderGIF(ctx context.Context, req RenderRequest) (*RenderArtifact, error)
	Available() (bool, string)
	Status() FFmpegStatus
	Limits() Limits
}

type RenderRequest struct {
	JobID   string
	WorkDir string
	Frames  []string
	FPS     int
	Width   int
	Loop    bool
	Timeout time.Duration
}

type RenderArtifact struct {
	Path   string
	Mime   string
	Bytes  int64
	Width  int
	Height int
}

type RenderError struct {
	Code    string
	Message string
	Err     error
}

func (e RenderError) Error() string {
	if e.Message == "" {
		return e.Code
	}
	return e.Code + ": " + e.Message
}

func (e RenderError) Unwrap() error { return e.Err }

func (e RenderError) Is(target error) bool {
	return target == ErrRendererUnavailable && e.Code == "GIF_RENDERER_UNAVAILABLE"
}

func ConfigFromApp(cfg config.Config) Config {
	return NormalizeConfig(Config{
		Enabled:   cfg.GIFEnabled,
		FFmpegBin: cfg.FFmpegBin,
		WorkDir:   cfg.GIFWorkDir,
		MaxFrames: cfg.GIFMaxFrames,
		MaxFPS:    cfg.GIFMaxFPS,
		MaxSize:   cfg.GIFMaxSize,
		Timeout:   time.Duration(cfg.GIFRenderTimeoutSec) * time.Second,
	})
}

func NormalizeConfig(cfg Config) Config {
	if strings.TrimSpace(cfg.FFmpegBin) == "" {
		cfg.FFmpegBin = "ffmpeg"
	}
	if strings.TrimSpace(cfg.WorkDir) == "" {
		cfg.WorkDir = "data/gif_work"
	}
	if cfg.MaxFrames <= 0 {
		cfg.MaxFrames = 24
	}
	if cfg.MaxFPS <= 0 {
		cfg.MaxFPS = 15
	}
	if cfg.MaxSize <= 0 {
		cfg.MaxSize = 1024
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 60 * time.Second
	}
	return cfg
}

func (c Config) Limits() Limits {
	c = NormalizeConfig(c)
	return Limits{MaxFrames: c.MaxFrames, MaxFPS: c.MaxFPS, MaxSize: c.MaxSize}
}
