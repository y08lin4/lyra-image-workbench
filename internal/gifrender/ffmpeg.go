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
	cfg := NormalizeConfig(r.cfg)
	bin := strings.TrimSpace(cfg.FFmpegBin)
	if !cfg.Enabled {
		return false, bin
	}
	resolved, err := r.runner.LookPath(bin)
	if err != nil {
		return false, bin
	}
	return true, resolved
}

func (r *FFmpegRenderer) RenderGIF(ctx context.Context, req RenderRequest) (*RenderArtifact, error) {
	cfg := NormalizeConfig(r.cfg)
	if err := validateRenderRequest(cfg, &req); err != nil {
		return nil, err
	}
	available, bin := r.Available()
	if !available {
		return nil, RenderError{Code: "GIF_RENDERER_UNAVAILABLE", Message: "FFmpeg is not available", Err: ErrRendererUnavailable}
	}
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
