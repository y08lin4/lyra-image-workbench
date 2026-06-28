package jobs

import (
	"context"
	"fmt"
	"time"

	"github.com/y08lin4/lyra-image-workbench/internal/gifrender"
)

func (m *Manager) generateGIF(ctx context.Context, spaceToken string, job Job, index int, prompt string, started time.Time) Result {
	jobID := job.ID
	spaceCfg, err := m.spaceConfig.Get(spaceToken)
	if err != nil {
		return withElapsed(NewResult(index, StatusFailed, err.Error()), started)
	}
	inputs, err := m.inputImagesForJob(spaceToken, job)
	if err != nil {
		return withElapsed(NewResult(index, StatusFailed, err.Error()), started)
	}
	if len(inputs) != 1 {
		return withElapsed(NewResult(index, StatusFailed, "GIF 动图需要且只能使用一张参考图"), started)
	}
	job, _, _ = m.store.Update(spaceToken, jobID, func(j *Job) {
		ApplyStage(j, StagePreparing)
		if j.Progress < 24 {
			j.Progress = 24
		}
	})
	m.publish(jobID, "progress", eventPayload(job))
	m.appendDebugLog(spaceToken, jobID, index, "info", "gif_render_start", "开始本地生成 GIF 动图", map[string]any{
		"inputImages":   debugInputImages(inputs),
		"promptLength":  len([]rune(prompt)),
		"promptPreview": compactDebugText(prompt, 120),
		"framePrompts":  len(job.FramePrompts),
	})
	rendered, err := gifrender.RenderFile(ctx, inputs[0].Path, gifrender.Options{Prompt: prompt, FramePrompts: job.FramePrompts})
	if err != nil {
		m.appendDebugLog(spaceToken, jobID, index, "error", "gif_render", "GIF 动图生成失败", map[string]any{
			"error":     err.Error(),
			"errorCode": ErrorMeta(err.Error()).Code,
			"elapsedMs": time.Since(started).Milliseconds(),
		})
		return withElapsed(NewResult(index, StatusFailed, err.Error()), started)
	}
	m.appendDebugLog(spaceToken, jobID, index, "info", "gif_render_done", "GIF 动图已生成，准备保存", map[string]any{
		"bytes":      len(rendered.Bytes),
		"frames":     rendered.Frames,
		"effect":     rendered.Effect,
		"actualSize": fmt.Sprintf("%dx%d", rendered.Width, rendered.Height),
		"elapsedMs":  time.Since(started).Milliseconds(),
	})
	job, _, _ = m.store.Update(spaceToken, jobID, func(j *Job) {
		ApplyStage(j, StageSaving)
		if j.Progress < 92 {
			j.Progress = 92
		}
	})
	m.publish(jobID, "progress", eventPayload(job))
	saved, err := m.output.Save(spaceToken, jobID, index, rendered.Bytes, "image/gif")
	if err != nil {
		m.appendDebugLog(spaceToken, jobID, index, "error", "save_output", "保存 GIF 到本机失败", map[string]any{
			"error":     err.Error(),
			"errorCode": ErrorMeta(err.Error()).Code,
			"elapsedMs": time.Since(started).Milliseconds(),
		})
		return withElapsed(NewResult(index, StatusFailed, err.Error()), started)
	}
	m.appendDebugLog(spaceToken, jobID, index, "info", "save_output", "GIF 已保存到本机", map[string]any{
		"url":      saved.URL,
		"fileName": saved.FileName,
		"mime":     saved.Mime,
		"bytes":    saved.Bytes,
	})
	result := withElapsed(NewResult(index, StatusSucceeded, ""), started)
	result.ImageURL = fmt.Sprintf("/api/background-tasks/%s/images/%d", jobID, index)
	result.OutputDate = saved.Date
	result.OutputFileName = saved.FileName
	result.Mime = saved.Mime
	result.Bytes = saved.Bytes
	result.RevisedPrompt = prompt
	result.ActualSize = fmt.Sprintf("%dx%d", rendered.Width, rendered.Height)
	result.ActualQuality = fmt.Sprintf("%d frames", rendered.Frames)
	result.OutputFormat = "gif"
	if spaceCfg.AutoUploadPixhost {
		if uploaded, err := m.pixhost.UploadFile(ctx, saved.Path, saved.Mime, saved.FileName); err == nil {
			result.RemoteURL = uploaded.ShowURL
			result.RemoteThumbURL = uploaded.ThumbURL
		} else {
			result.UploadError = err.Error()
		}
	}
	return result
}
