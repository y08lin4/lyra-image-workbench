package gifrender

import (
	"bytes"
	"context"
	"image"
	"image/color"
	stdgif "image/gif"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderFileCreatesAnimatedGIF(t *testing.T) {
	t.Parallel()
	path := writePNGFixture(t)
	result, err := RenderFile(context.Background(), path, Options{
		Prompt:       "animate hair with gentle wind",
		FramePrompts: []string{"motion strength: standard", "loop rhythm: smooth"},
	})
	if err != nil {
		t.Fatalf("RenderFile() error = %v", err)
	}
	if !bytes.HasPrefix(result.Bytes, []byte("GIF")) {
		t.Fatalf("RenderFile() did not return GIF data")
	}
	decoded, err := stdgif.DecodeAll(bytes.NewReader(result.Bytes))
	if err != nil {
		t.Fatalf("DecodeAll() error = %v", err)
	}
	if len(decoded.Image) < 2 {
		t.Fatalf("expected multiple frames, got %d", len(decoded.Image))
	}
	if result.Width != 40 || result.Height != 24 || result.Frames != defaultFrames {
		t.Fatalf("unexpected render metadata: %+v", result)
	}
}

func TestRenderFileRejectsUndecodableReference(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "bad.webp")
	if err := os.WriteFile(path, []byte("not an image"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	_, err := RenderFile(context.Background(), path, Options{})
	if err == nil || !strings.Contains(err.Error(), "无法解码") {
		t.Fatalf("expected decode error, got %v", err)
	}
}

func writePNGFixture(t *testing.T) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 40, 24))
	for y := 0; y < 24; y++ {
		for x := 0; x < 40; x++ {
			img.SetRGBA(x, y, color.RGBA{R: uint8(x * 6), G: uint8(y * 9), B: 160, A: 255})
		}
	}
	path := filepath.Join(t.TempDir(), "reference.png")
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if err := png.Encode(file, img); err != nil {
		_ = file.Close()
		t.Fatalf("png.Encode() error = %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	return path
}
