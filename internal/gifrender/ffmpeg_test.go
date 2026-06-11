package gifrender

import (
	"context"
	"errors"
	"image"
	"image/color"
	"image/gif"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type fakeRunner struct {
	lookPath string
	lookErr  error
	run      func(ctx context.Context, dir string, name string, args []string) ([]byte, error)
	args     []string
	dir      string
	name     string
}

func TestRenderIntegrationWithFFmpeg(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not installed")
	}
	root := t.TempDir()
	frames := makePNGFrames(t, root, 3)
	renderer := NewFFmpegRenderer(Config{Enabled: true, FFmpegBin: "ffmpeg", WorkDir: filepath.Join(root, "work"), MaxFrames: 4, MaxFPS: 12, MaxSize: 128, Timeout: 10 * time.Second})
	artifact, err := renderer.RenderGIF(context.Background(), RenderRequest{JobID: "integration", Frames: frames, FPS: 6, Width: 64, Loop: true})
	if err != nil {
		t.Fatalf("RenderGIF() error = %v", err)
	}
	file, err := os.Open(artifact.Path)
	if err != nil {
		t.Fatalf("open output gif: %v", err)
	}
	defer file.Close()
	decoded, err := gif.DecodeAll(file)
	if err != nil {
		t.Fatalf("DecodeAll(output.gif) error = %v", err)
	}
	if len(decoded.Image) < 3 {
		t.Fatalf("expected at least 3 frames, got %d", len(decoded.Image))
	}
	if len(decoded.Delay) == 0 {
		t.Fatalf("expected GIF delays")
	}
	if decoded.LoopCount != 0 {
		t.Fatalf("expected infinite loop count 0, got %d", decoded.LoopCount)
	}
}

func (f *fakeRunner) LookPath(file string) (string, error) {
	if f.lookErr != nil {
		return "", f.lookErr
	}
	if f.lookPath != "" {
		return f.lookPath, nil
	}
	return file, nil
}

func (f *fakeRunner) CombinedOutput(ctx context.Context, dir string, name string, args []string) ([]byte, error) {
	f.dir = dir
	f.name = name
	f.args = append([]string{}, args...)
	if f.run != nil {
		return f.run(ctx, dir, name, args)
	}
	return []byte{}, nil
}

func TestBuildFFmpegArgs(t *testing.T) {
	args := BuildFFmpegArgs(8, 512, true)
	joined := strings.Join(args, " ")
	for _, want := range []string{"-framerate 8", "-i frame_%04d.png", "scale=512:-1", "palettegen", "paletteuse", "-loop 0", "output.gif"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("args missing %q: %v", want, args)
		}
	}
	args = BuildFFmpegArgs(6, 320, false)
	if !strings.Contains(strings.Join(args, " "), "-loop -1") {
		t.Fatalf("loop=false should use -loop -1: %v", args)
	}
}

func TestRenderRejectsInvalidValues(t *testing.T) {
	root := t.TempDir()
	frames := makeFrames(t, root, 4)
	cfg := Config{Enabled: true, FFmpegBin: "ffmpeg", WorkDir: filepath.Join(root, "work"), MaxFrames: 3, MaxFPS: 10, MaxSize: 512, Timeout: time.Second}
	renderer := NewFFmpegRendererWithRunner(cfg, &fakeRunner{})
	tests := []struct {
		name string
		req  RenderRequest
		code string
	}{
		{"fps zero", RenderRequest{JobID: "a", Frames: frames[:2], FPS: 0, Width: 256}, "GIF_FPS_INVALID"},
		{"fps high", RenderRequest{JobID: "a", Frames: frames[:2], FPS: 11, Width: 256}, "GIF_FPS_TOO_HIGH"},
		{"too few frames", RenderRequest{JobID: "a", Frames: frames[:1], FPS: 8, Width: 256}, "GIF_FRAME_COUNT_TOO_LOW"},
		{"too many frames", RenderRequest{JobID: "a", Frames: frames, FPS: 8, Width: 256}, "GIF_FRAME_COUNT_TOO_HIGH"},
		{"width high", RenderRequest{JobID: "a", Frames: frames[:2], FPS: 8, Width: 2048}, "GIF_WIDTH_TOO_HIGH"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := renderer.RenderGIF(context.Background(), tt.req)
			var renderErr RenderError
			if !errors.As(err, &renderErr) || renderErr.Code != tt.code {
				t.Fatalf("expected %s, got %#v", tt.code, err)
			}
		})
	}
}

func TestRenderUnavailable(t *testing.T) {
	root := t.TempDir()
	frames := makeFrames(t, root, 2)
	runner := &fakeRunner{lookErr: os.ErrNotExist}
	renderer := NewFFmpegRendererWithRunner(Config{Enabled: true, FFmpegBin: "missing-ffmpeg", WorkDir: filepath.Join(root, "work"), MaxFrames: 4, MaxFPS: 12, MaxSize: 512, Timeout: time.Second}, runner)
	_, err := renderer.RenderGIF(context.Background(), RenderRequest{JobID: "a", Frames: frames, FPS: 8, Width: 256})
	if !errors.Is(err, ErrRendererUnavailable) {
		t.Fatalf("expected ErrRendererUnavailable, got %v", err)
	}
}

func TestRenderTimeoutCleansWorkdir(t *testing.T) {
	root := t.TempDir()
	frames := makeFrames(t, root, 2)
	runner := &fakeRunner{run: func(ctx context.Context, dir string, name string, args []string) ([]byte, error) {
		<-ctx.Done()
		return []byte("slow ffmpeg"), ctx.Err()
	}}
	renderer := NewFFmpegRendererWithRunner(Config{Enabled: true, FFmpegBin: "ffmpeg", WorkDir: filepath.Join(root, "work"), MaxFrames: 4, MaxFPS: 12, MaxSize: 512, Timeout: 20 * time.Millisecond}, runner)
	_, err := renderer.RenderGIF(context.Background(), RenderRequest{JobID: "timeout", Frames: frames, FPS: 8, Width: 256})
	var renderErr RenderError
	if !errors.As(err, &renderErr) || renderErr.Code != "GIF_RENDER_TIMEOUT" {
		t.Fatalf("expected timeout error, got %#v", err)
	}
	if _, statErr := os.Stat(filepath.Join(root, "work", "timeout_work")); !os.IsNotExist(statErr) {
		t.Fatalf("temporary workdir was not cleaned, statErr=%v", statErr)
	}
}

func TestRenderSuccessCreatesArtifactAndCleansTemp(t *testing.T) {
	root := t.TempDir()
	frames := makeFrames(t, root, 2)
	runner := &fakeRunner{lookPath: "ffmpeg", run: func(ctx context.Context, dir string, name string, args []string) ([]byte, error) {
		if err := os.WriteFile(filepath.Join(dir, "output.gif"), []byte("GIF89a"), 0o600); err != nil {
			return nil, err
		}
		return []byte{}, nil
	}}
	renderer := NewFFmpegRendererWithRunner(Config{Enabled: true, FFmpegBin: "ffmpeg", WorkDir: filepath.Join(root, "work"), MaxFrames: 4, MaxFPS: 12, MaxSize: 512, Timeout: time.Second}, runner)
	artifact, err := renderer.RenderGIF(context.Background(), RenderRequest{JobID: "ok", Frames: frames, FPS: 8, Width: 256, Loop: true})
	if err != nil {
		t.Fatalf("RenderGIF failed: %v", err)
	}
	if artifact.Mime != "image/gif" || artifact.Bytes == 0 {
		t.Fatalf("bad artifact: %+v", artifact)
	}
	if _, err := os.Stat(artifact.Path); err != nil {
		t.Fatalf("artifact missing: %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(root, "work", "ok_work")); !os.IsNotExist(statErr) {
		t.Fatalf("temporary workdir was not cleaned, statErr=%v", statErr)
	}
	if !strings.Contains(strings.Join(runner.args, " "), "frame_%04d.png") {
		t.Fatalf("ffmpeg input pattern missing: %v", runner.args)
	}
}

func makeFrames(t *testing.T, dir string, count int) []string {
	t.Helper()
	frames := make([]string, 0, count)
	for i := 0; i < count; i++ {
		path := filepath.Join(dir, "src"+string(rune('a'+i))+".png")
		if err := os.WriteFile(path, []byte{0x89, 'P', 'N', 'G', byte(i)}, 0o600); err != nil {
			t.Fatal(err)
		}
		frames = append(frames, path)
	}
	return frames
}

func makePNGFrames(t *testing.T, dir string, count int) []string {
	t.Helper()
	frames := make([]string, 0, count)
	for i := 0; i < count; i++ {
		img := image.NewRGBA(image.Rect(0, 0, 32, 32))
		c := color.RGBA{R: uint8(40 + i*40), G: uint8(80 + i*30), B: uint8(160 - i*30), A: 255}
		for y := 0; y < 32; y++ {
			for x := 0; x < 32; x++ {
				img.SetRGBA(x, y, c)
			}
		}
		path := filepath.Join(dir, "valid-frame-"+string(rune('0'+i))+".png")
		file, err := os.Create(path)
		if err != nil {
			t.Fatal(err)
		}
		if err := png.Encode(file, img); err != nil {
			_ = file.Close()
			t.Fatal(err)
		}
		if err := file.Close(); err != nil {
			t.Fatal(err)
		}
		frames = append(frames, path)
	}
	return frames
}
