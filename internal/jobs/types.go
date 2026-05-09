package jobs

import (
	"fmt"
	"time"
)

type Status string

type Stage string

type Mode string

const (
	ModeTextToImage  Mode = "text-to-image"
	ModeImageToImage Mode = "image-to-image"

	StatusQueued        Status = "queued"
	StatusRunning       Status = "running"
	StatusSucceeded     Status = "succeeded"
	StatusPartialFailed Status = "partial_failed"
	StatusFailed        Status = "failed"
	StatusCancelled     Status = "cancelled"
	StatusInterrupted   Status = "interrupted"

	StageQueued          Stage = "queued"
	StagePreparing       Stage = "preparing"
	StageSubmitting      Stage = "submitting"
	StageWaitingUpstream Stage = "waiting_upstream"
	StageDownloading     Stage = "downloading"
	StageSaving          Stage = "saving"
	StageSucceeded       Stage = "succeeded"
	StagePartialFailed   Stage = "partial_failed"
	StageFailed          Stage = "failed"
	StageCancelled       Stage = "cancelled"
	StageInterrupted     Stage = "interrupted"
)

type Meta struct {
	Code    string `json:"code"`
	English string `json:"english"`
	Chinese string `json:"chinese"`
}

type Job struct {
	ID          string    `json:"id"`
	SpaceToken  string    `json:"spaceToken"`
	Mode        Mode      `json:"mode"`
	Prompt      string    `json:"prompt"`
	Ratio       string    `json:"ratio"`
	Resolution  string    `json:"resolution"`
	Size        string    `json:"size"`
	Count       int       `json:"count"`
	Concurrency int       `json:"concurrency"`
	UploadIDs   []string  `json:"uploadIds,omitempty"`
	Status      Status    `json:"status"`
	StatusText  string    `json:"statusText"`
	StatusCode  string    `json:"statusCode"`
	Stage       Stage     `json:"stage"`
	StageText   string    `json:"stageText"`
	StageCode   string    `json:"stageCode"`
	Progress    int       `json:"progress"`
	Results     []Result  `json:"results"`
	Error       string    `json:"error,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
	StartedAt   time.Time `json:"startedAt,omitempty"`
	FinishedAt  time.Time `json:"finishedAt,omitempty"`
}

type Result struct {
	Index      int    `json:"index"`
	OK         bool   `json:"ok"`
	Status     Status `json:"status"`
	StatusText string `json:"statusText"`
	StatusCode string `json:"statusCode"`
	ImageURL   string `json:"imageUrl,omitempty"`
	Mime       string `json:"mime,omitempty"`
	Bytes      int64  `json:"bytes,omitempty"`
	Error      string `json:"error,omitempty"`
	ElapsedMs  int64  `json:"elapsedMs,omitempty"`
}

type CreateRequest struct {
	Mode        Mode     `json:"mode"`
	Prompt      string   `json:"prompt"`
	Ratio       string   `json:"ratio"`
	Resolution  string   `json:"resolution"`
	Count       int      `json:"count"`
	Concurrency int      `json:"concurrency"`
	UploadIDs   []string `json:"uploadIds"`
}

type Stats struct {
	TotalTasks     int `json:"totalTasks"`
	RunningTasks   int `json:"runningTasks"`
	SucceededTasks int `json:"succeededTasks"`
	FailedTasks    int `json:"failedTasks"`
	TotalImages    int `json:"totalImages"`
}

func ApplyStatus(job *Job, status Status) {
	meta := StatusMeta(status)
	job.Status = status
	job.StatusText = meta.Chinese
	job.StatusCode = meta.Code
	job.UpdatedAt = time.Now()
}

func ApplyStage(job *Job, stage Stage) {
	meta := StageMeta(stage)
	job.Stage = stage
	job.StageText = meta.Chinese
	job.StageCode = meta.Code
	job.UpdatedAt = time.Now()
}

func NewResult(index int, status Status, err string) Result {
	meta := StatusMeta(status)
	return Result{Index: index, OK: status == StatusSucceeded, Status: status, StatusText: meta.Chinese, StatusCode: meta.Code, Error: err}
}

func StatusMeta(status Status) Meta {
	switch status {
	case StatusQueued:
		return Meta{"J100", string(status), "排队中"}
	case StatusRunning:
		return Meta{"J200", string(status), "运行中"}
	case StatusSucceeded:
		return Meta{"J300", string(status), "已成功"}
	case StatusPartialFailed:
		return Meta{"J206", string(status), "部分成功"}
	case StatusCancelled:
		return Meta{"J499", string(status), "已取消"}
	case StatusInterrupted:
		return Meta{"J520", string(status), "已中断"}
	default:
		return Meta{"J500", string(status), "已失败"}
	}
}

func StageMeta(stage Stage) Meta {
	switch stage {
	case StageQueued:
		return Meta{"S100", string(stage), "排队中"}
	case StagePreparing:
		return Meta{"S110", string(stage), "准备中"}
	case StageSubmitting:
		return Meta{"S120", string(stage), "提交中"}
	case StageWaitingUpstream:
		return Meta{"S130", string(stage), "等待上游"}
	case StageDownloading:
		return Meta{"S140", string(stage), "下载图片"}
	case StageSaving:
		return Meta{"S150", string(stage), "保存本机"}
	case StageSucceeded:
		return Meta{"S300", string(stage), "已成功"}
	case StagePartialFailed:
		return Meta{"S206", string(stage), "部分成功"}
	case StageCancelled:
		return Meta{"S499", string(stage), "已取消"}
	case StageInterrupted:
		return Meta{"S520", string(stage), "已中断"}
	default:
		return Meta{"S500", string(stage), "已失败"}
	}
}

func EventMeta(name string) Meta {
	switch name {
	case "snapshot":
		return Meta{"E100", name, "任务快照"}
	case "progress":
		return Meta{"E110", name, "进度更新"}
	case "result":
		return Meta{"E120", name, "单图结果"}
	case "heartbeat":
		return Meta{"E130", name, "心跳保活"}
	case "done":
		return Meta{"E300", name, "任务结束"}
	default:
		return Meta{"E500", name, "错误事件"}
	}
}

func (j Job) Final() bool {
	return j.Status == StatusSucceeded || j.Status == StatusPartialFailed || j.Status == StatusFailed || j.Status == StatusCancelled || j.Status == StatusInterrupted
}

func (j Job) ElapsedMs() int64 {
	if j.StartedAt.IsZero() {
		return 0
	}
	end := time.Now()
	if !j.FinishedAt.IsZero() {
		end = j.FinishedAt
	}
	return end.Sub(j.StartedAt).Milliseconds()
}

func eventPayload(job Job) map[string]any {
	return map[string]any{"job": job, "elapsedMs": job.ElapsedMs(), "label": fmt.Sprintf("%s / %s / %s", job.StageText, job.Stage, job.StageCode)}
}
