package gifrender

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type commandRunner interface {
	LookPath(file string) (string, error)
	CombinedOutput(ctx context.Context, dir string, name string, args []string) ([]byte, error)
}

type execRunner struct{}

func (execRunner) LookPath(file string) (string, error) { return exec.LookPath(file) }

func (execRunner) CombinedOutput(ctx context.Context, dir string, name string, args []string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	return cmd.CombinedOutput()
}

type FFmpegRenderer struct {
	cfg    Config
	runner commandRunner
}

func NewFFmpegRenderer(cfg Config) *FFmpegRenderer {
	return &FFmpegRenderer{cfg: NormalizeConfig(cfg), runner: execRunner{}}
}

func NewFFmpegRendererWithRunner(cfg Config, runner commandRunner) *FFmpegRenderer {
	if runner == nil {
		runner = execRunner{}
	}
	return &FFmpegRenderer{cfg: NormalizeConfig(cfg), runner: runner}
}

func (r *FFmpegRenderer) Limits() Limits { return r.cfg.Limits() }

func (r *FFmpegRenderer) Available() (bool, string) {
	status := r.Status()
	return status.Available, status.Bin
}

func (r *FFmpegRenderer) Status() FFmpegStatus {
	cfg := NormalizeConfig(r.cfg)
	bin := strings.TrimSpace(cfg.FFmpegBin)
	status := FFmpegStatus{Enabled: cfg.Enabled, Bin: bin, MinimumVersion: FFmpegMinimumSafeVersion}
	if !cfg.Enabled {
		status.Code = "GIF_DISABLED"
		status.Message = "GIF rendering is disabled"
		return status
	}
	resolved, err := r.runner.LookPath(bin)
	if err != nil {
		status.Code = "FFMPEG_NOT_FOUND"
		status.Message = "FFmpeg executable was not found"
		return status
	}
	status.Bin = resolved
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	output, err := r.runner.CombinedOutput(ctx, "", resolved, []string{"-version"})
	if ctx.Err() != nil {
		status.Code = "FFMPEG_VERSION_CHECK_TIMEOUT"
		status.Message = ctx.Err().Error()
		return status
	}
	if err != nil {
		status.Code = "FFMPEG_VERSION_CHECK_FAILED"
		status.Message = truncateStderr(string(output), 512)
		if status.Message == "" {
			status.Message = err.Error()
		}
		return status
	}
	status.Version = parseFFmpegVersion(string(output))
	if status.Version == "" {
		status.Code = "FFMPEG_VERSION_UNKNOWN"
		status.Message = "FFmpeg version could not be parsed"
		return status
	}
	if !ffmpegVersionAtLeast(status.Version, FFmpegMinimumSafeVersion) {
		status.Code = "FFMPEG_VERSION_UNSAFE"
		status.Message = fmt.Sprintf("FFmpeg %s is below the required safe version %s", status.Version, FFmpegMinimumSafeVersion)
		return status
	}
	status.Available = true
	status.Safe = true
	status.Code = "OK"
	status.Message = "FFmpeg is available"
	return status
}

func parseFFmpegVersion(output string) string {
	match := regexp.MustCompile(`(?im)^ffmpeg version\s+([^\s]+)`).FindStringSubmatch(output)
	if len(match) < 2 {
		return ""
	}
	return strings.TrimPrefix(strings.TrimSpace(match[1]), "n")
}

func ffmpegVersionAtLeast(version string, minimum string) bool {
	current, ok := ffmpegVersionTriple(version)
	if !ok {
		return false
	}
	required, ok := ffmpegVersionTriple(minimum)
	if !ok {
		return false
	}
	for i := 0; i < len(current); i++ {
		if current[i] > required[i] {
			return true
		}
		if current[i] < required[i] {
			return false
		}
	}
	return true
}

func ffmpegVersionTriple(value string) ([3]int, bool) {
	var out [3]int
	match := regexp.MustCompile(`(?i)^n?(\d+)(?:\.(\d+))?(?:\.(\d+))?`).FindStringSubmatch(strings.TrimSpace(value))
	if len(match) == 0 {
		return out, false
	}
	for i := 0; i < len(out); i++ {
		if match[i+1] == "" {
			continue
		}
		parsed, err := strconv.Atoi(match[i+1])
		if err != nil {
			return out, false
		}
		out[i] = parsed
	}
	return out, true
}

func (r *FFmpegRenderer) RenderGIF(ctx context.Context, req RenderRequest) (*RenderArtifact, error) {
	cfg := NormalizeConfig(r.cfg)
	if err := validateRenderRequest(cfg, &req); err != nil {
		return nil, err
	}
	status := r.Status()
	if !status.Available {
		message := status.Message
		if message == "" {
			message = "FFmpeg is not available"
		}
		return nil, RenderError{Code: "GIF_RENDERER_UNAVAILABLE", Message: message, Err: ErrRendererUnavailable}
	}
	bin := status.Bin
	root := strings.TrimSpace(req.WorkDir)
	if root == "" {
		root = cfg.WorkDir
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		return nil, RenderError{Code: "GIF_WORK_DIR_CREATE_FAILED", Message: err.Error(), Err: err}
	}
	jobID := safeJobID(req.JobID)
	workDir := filepath.Join(root, jobID+"_work")
	finalPath := filepath.Join(root, jobID+".gif")
	_ = os.RemoveAll(workDir)
	_ = os.Remove(finalPath)
	if err := os.MkdirAll(workDir, 0o700); err != nil {
		return nil, RenderError{Code: "GIF_WORK_DIR_CREATE_FAILED", Message: err.Error(), Err: err}
	}
	defer os.RemoveAll(workDir)

	for i, frame := range req.Frames {
		if err := copyFrame(frame, filepath.Join(workDir, fmt.Sprintf("frame_%04d.png", i))); err != nil {
			return nil, err
		}
	}

	timeout := req.Timeout
	if timeout <= 0 {
		timeout = cfg.Timeout
	}
	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	args := BuildFFmpegArgs(req.FPS, req.Width, req.Loop)
	output, err := r.runner.CombinedOutput(callCtx, workDir, bin, args)
	if callCtx.Err() != nil {
		return nil, RenderError{Code: "GIF_RENDER_TIMEOUT", Message: callCtx.Err().Error(), Err: callCtx.Err()}
	}
	if err != nil {
		message := truncateStderr(string(output), 4096)
		if message == "" {
			message = err.Error()
		}
		return nil, RenderError{Code: "GIF_RENDER_FAILED", Message: message, Err: err}
	}
	if err := moveFile(filepath.Join(workDir, "output.gif"), finalPath); err != nil {
		return nil, RenderError{Code: "GIF_OUTPUT_SAVE_FAILED", Message: err.Error(), Err: err}
	}
	info, err := os.Stat(finalPath)
	if err != nil {
		return nil, RenderError{Code: "GIF_OUTPUT_SAVE_FAILED", Message: err.Error(), Err: err}
	}
	if info.Size() <= 0 {
		return nil, RenderError{Code: "GIF_OUTPUT_EMPTY", Message: "FFmpeg generated an empty GIF"}
	}
	return &RenderArtifact{Path: finalPath, Mime: "image/gif", Bytes: info.Size(), Width: req.Width}, nil
}

func BuildFFmpegArgs(fps int, width int, loop bool) []string {
	loopValue := "-1"
	if loop {
		loopValue = "0"
	}
	filter := fmt.Sprintf("fps=%d,scale=%d:-1:flags=lanczos,split[s0][s1];[s0]palettegen=stats_mode=diff[p];[s1][p]paletteuse=dither=sierra2_4a:diff_mode=rectangle", fps, width)
	return []string{
		"-y",
		"-hide_banner",
		"-loglevel", "error",
		"-framerate", strconv.Itoa(fps),
		"-f", "image2",
		"-c:v", "png",
		"-i", "frame_%04d.png",
		"-filter_complex", filter,
		"-loop", loopValue,
		"output.gif",
	}
}

func validateRenderRequest(cfg Config, req *RenderRequest) error {
	if req.FPS <= 0 {
		return RenderError{Code: "GIF_FPS_INVALID", Message: "fps must be greater than 0"}
	}
	if req.FPS > cfg.MaxFPS {
		return RenderError{Code: "GIF_FPS_TOO_HIGH", Message: fmt.Sprintf("fps must be <= %d", cfg.MaxFPS)}
	}
	if len(req.Frames) < 2 {
		return RenderError{Code: "GIF_FRAME_COUNT_TOO_LOW", Message: "at least 2 frames are required"}
	}
	if len(req.Frames) > cfg.MaxFrames {
		return RenderError{Code: "GIF_FRAME_COUNT_TOO_HIGH", Message: fmt.Sprintf("frame count must be <= %d", cfg.MaxFrames)}
	}
	if req.Width <= 0 {
		req.Width = DefaultWidth
	}
	if req.Width > cfg.MaxSize {
		return RenderError{Code: "GIF_WIDTH_TOO_HIGH", Message: fmt.Sprintf("width must be <= %d", cfg.MaxSize)}
	}
	for _, frame := range req.Frames {
		if strings.TrimSpace(frame) == "" {
			return RenderError{Code: "GIF_FRAME_PATH_INVALID", Message: "frame path is empty"}
		}
		info, err := os.Stat(frame)
		if err != nil {
			return RenderError{Code: "GIF_FRAME_NOT_FOUND", Message: err.Error(), Err: err}
		}
		if info.IsDir() {
			return RenderError{Code: "GIF_FRAME_PATH_INVALID", Message: "frame path is a directory"}
		}
		if err := validatePNGFrame(frame); err != nil {
			return err
		}
	}
	return nil
}

func validatePNGFrame(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return RenderError{Code: "GIF_FRAME_NOT_FOUND", Message: err.Error(), Err: err}
	}
	defer file.Close()
	var signature [8]byte
	if _, err := io.ReadFull(file, signature[:]); err != nil {
		return RenderError{Code: "GIF_FRAME_NOT_PNG", Message: "GIF frames must be PNG files", Err: err}
	}
	if string(signature[:]) != "\x89PNG\r\n\x1a\n" {
		return RenderError{Code: "GIF_FRAME_NOT_PNG", Message: "GIF frames must be PNG files"}
	}
	return nil
}
func copyFrame(src string, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return RenderError{Code: "GIF_FRAME_NOT_FOUND", Message: err.Error(), Err: err}
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return RenderError{Code: "GIF_FRAME_COPY_FAILED", Message: err.Error(), Err: err}
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		return RenderError{Code: "GIF_FRAME_COPY_FAILED", Message: copyErr.Error(), Err: copyErr}
	}
	if closeErr != nil {
		return RenderError{Code: "GIF_FRAME_COPY_FAILED", Message: closeErr.Error(), Err: closeErr}
	}
	return nil
}

func moveFile(src string, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func truncateStderr(value string, max int) string {
	value = strings.TrimSpace(value)
	if max <= 0 || len(value) <= max {
		return value
	}
	return value[:max] + "..."
}

func safeJobID(value string) string {
	value = regexp.MustCompile(`[^a-zA-Z0-9_-]+`).ReplaceAllString(strings.TrimSpace(value), "-")
	value = strings.Trim(value, "-")
	if value == "" {
		value = fmt.Sprintf("gifrender_%d", time.Now().UnixNano())
	}
	if len(value) > 80 {
		value = value[:80]
	}
	return value
}
