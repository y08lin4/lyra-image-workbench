package gifrender

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/color/palette"
	stdgif "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"
	"strings"

	"image/draw"
)

const (
	defaultFrames = 12
	maxLongEdge   = 1024
)

type Options struct {
	Prompt       string
	FramePrompts []string
}

type Result struct {
	Bytes  []byte
	Width  int
	Height int
	Frames int
	Effect string
}

func RenderFile(ctx context.Context, path string, options Options) (Result, error) {
	if strings.TrimSpace(path) == "" {
		return Result{}, errors.New("GIF 参考图路径为空")
	}
	file, err := os.Open(path)
	if err != nil {
		return Result{}, err
	}
	defer file.Close()

	source, format, err := image.Decode(file)
	if err != nil {
		return Result{}, fmt.Errorf("GIF 动图目前支持 PNG/JPG/GIF 参考图，当前图片无法解码: %w", err)
	}
	source = fitLongEdge(source, maxLongEdge)
	effect := detectEffect(options.Prompt, options.FramePrompts)
	strength := detectStrength(options.Prompt, options.FramePrompts)
	delay := detectDelay(options.Prompt, options.FramePrompts)
	rendered, err := renderLoop(ctx, source, effect, strength, delay)
	if err != nil {
		return Result{}, err
	}
	var out bytes.Buffer
	if err := stdgif.EncodeAll(&out, rendered); err != nil {
		return Result{}, err
	}
	bounds := source.Bounds()
	result := Result{
		Bytes:  out.Bytes(),
		Width:  bounds.Dx(),
		Height: bounds.Dy(),
		Frames: len(rendered.Image),
		Effect: effect,
	}
	if format == "gif" && effect == "still" {
		result.Effect = "gif-repack"
	}
	return result, nil
}

func renderLoop(ctx context.Context, source image.Image, effect string, strength float64, delay int) (*stdgif.GIF, error) {
	bounds := source.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	if width <= 0 || height <= 0 {
		return nil, errors.New("GIF 参考图尺寸无效")
	}
	anim := &stdgif.GIF{LoopCount: 0}
	for i := 0; i < defaultFrames; i++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		phase := 2 * math.Pi * float64(i) / float64(defaultFrames)
		frame := transformFrame(source, effect, phase, strength)
		paletted := image.NewPaletted(image.Rect(0, 0, width, height), palette.Plan9)
		draw.FloydSteinberg.Draw(paletted, paletted.Bounds(), frame, image.Point{})
		anim.Image = append(anim.Image, paletted)
		anim.Delay = append(anim.Delay, delay)
		anim.Disposal = append(anim.Disposal, stdgif.DisposalNone)
	}
	return anim, nil
}

func transformFrame(source image.Image, effect string, phase float64, strength float64) *image.RGBA {
	bounds := source.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	frame := image.NewRGBA(image.Rect(0, 0, width, height))
	cx := float64(width-1) / 2
	cy := float64(height-1) / 2
	wave := math.Sin(phase)
	scale := 1.0
	shiftX := 0.0
	shiftY := 0.0
	brightness := 1.0
	switch effect {
	case "camera-push":
		scale = 1 + strength*(0.5+0.5*wave)
	case "product-turn":
		scale = 1 + strength*0.55
		shiftX = wave * strength * float64(width) * 0.22
		brightness = 1 + 0.05*math.Sin(phase-math.Pi/3)
	case "poster-loop":
		scale = 1 + strength*0.3
		shiftY = wave * strength * float64(height) * 0.12
		brightness = 1 + 0.06*math.Sin(phase)
	case "blink":
		scale = 1 + strength*0.12
		brightness = 1 - 0.08*math.Max(0, math.Sin(phase))
	case "hair-sway", "cloth-breeze":
		scale = 1 + strength*0.18
		shiftX = wave * strength * float64(width) * 0.14
	default:
		scale = 1 + strength*0.22
		shiftX = wave * strength * float64(width) * 0.08
		shiftY = math.Cos(phase) * strength * float64(height) * 0.05
	}
	for y := 0; y < height; y++ {
		rowWave := 0.0
		if effect == "hair-sway" || effect == "cloth-breeze" {
			rowWave = math.Sin(float64(y)/28+phase) * strength * float64(width) * 0.035
		}
		for x := 0; x < width; x++ {
			srcX := (float64(x)-cx)/scale + cx - shiftX - rowWave
			srcY := (float64(y)-cy)/scale + cy - shiftY
			pixel := sampleNearest(source, bounds, srcX, srcY)
			frame.SetRGBA(x, y, adjustBrightness(pixel, brightness))
		}
	}
	return frame
}

func sampleNearest(source image.Image, bounds image.Rectangle, x float64, y float64) color.RGBA {
	ix := int(math.Round(x)) + bounds.Min.X
	iy := int(math.Round(y)) + bounds.Min.Y
	if ix < bounds.Min.X {
		ix = bounds.Min.X
	}
	if ix >= bounds.Max.X {
		ix = bounds.Max.X - 1
	}
	if iy < bounds.Min.Y {
		iy = bounds.Min.Y
	}
	if iy >= bounds.Max.Y {
		iy = bounds.Max.Y - 1
	}
	return color.RGBAModel.Convert(source.At(ix, iy)).(color.RGBA)
}

func adjustBrightness(c color.RGBA, factor float64) color.RGBA {
	return color.RGBA{R: clampByte(float64(c.R) * factor), G: clampByte(float64(c.G) * factor), B: clampByte(float64(c.B) * factor), A: c.A}
}

func clampByte(v float64) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}

func fitLongEdge(source image.Image, maxEdge int) image.Image {
	bounds := source.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	if width <= maxEdge && height <= maxEdge {
		return source
	}
	scale := float64(maxEdge) / float64(max(width, height))
	targetW := max(1, int(math.Round(float64(width)*scale)))
	targetH := max(1, int(math.Round(float64(height)*scale)))
	target := image.NewRGBA(image.Rect(0, 0, targetW, targetH))
	for y := 0; y < targetH; y++ {
		for x := 0; x < targetW; x++ {
			srcX := bounds.Min.X + min(width-1, int(float64(x)/scale))
			srcY := bounds.Min.Y + min(height-1, int(float64(y)/scale))
			target.Set(x, y, source.At(srcX, srcY))
		}
	}
	return target
}

func detectEffect(prompt string, framePrompts []string) string {
	text := strings.ToLower(prompt + "\n" + strings.Join(framePrompts, "\n"))
	switch {
	case strings.Contains(text, "camera push") || strings.Contains(text, "push-in") || strings.Contains(text, "镜头"):
		return "camera-push"
	case strings.Contains(text, "product") || strings.Contains(text, "highlight") || strings.Contains(text, "商品") || strings.Contains(text, "产品"):
		return "product-turn"
	case strings.Contains(text, "blink") || strings.Contains(text, "眨眼"):
		return "blink"
	case strings.Contains(text, "poster") || strings.Contains(text, "smoke") || strings.Contains(text, "light") || strings.Contains(text, "海报") || strings.Contains(text, "光影"):
		return "poster-loop"
	case strings.Contains(text, "cloth") || strings.Contains(text, "clothing") || strings.Contains(text, "衣"):
		return "cloth-breeze"
	case strings.Contains(text, "hair") || strings.Contains(text, "头发") || strings.Contains(text, "发丝"):
		return "hair-sway"
	default:
		return "subtle-loop"
	}
}

func detectStrength(prompt string, framePrompts []string) float64 {
	text := strings.ToLower(prompt + "\n" + strings.Join(framePrompts, "\n"))
	switch {
	case strings.Contains(text, "bold") || strings.Contains(text, "明显"):
		return 0.06
	case strings.Contains(text, "subtle") || strings.Contains(text, "轻微"):
		return 0.022
	default:
		return 0.038
	}
}

func detectDelay(prompt string, framePrompts []string) int {
	text := strings.ToLower(prompt + "\n" + strings.Join(framePrompts, "\n"))
	switch {
	case strings.Contains(text, "snappy") || strings.Contains(text, "短促"):
		return 7
	case strings.Contains(text, "breathing") || strings.Contains(text, "呼吸"):
		return 12
	default:
		return 9
	}
}
