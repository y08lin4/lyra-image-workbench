package jobs

import (
	"fmt"
	"regexp"
	"strings"
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
	ID           string    `json:"id"`
	SpaceToken   string    `json:"spaceToken"`
	Mode         Mode      `json:"mode"`
	Prompt       string    `json:"prompt"`
	Ratio        string    `json:"ratio"`
	Resolution   string    `json:"resolution"`
	Quality      string    `json:"quality"`
	OutputFormat string    `json:"outputFormat"`
	Size         string    `json:"size"`
	Count        int       `json:"count"`
	Concurrency  int       `json:"concurrency"`
	UploadIDs    []string  `json:"uploadIds,omitempty"`
	Status       Status    `json:"status"`
	StatusText   string    `json:"statusText"`
	StatusCode   string    `json:"statusCode"`
	Stage        Stage     `json:"stage"`
	StageText    string    `json:"stageText"`
	StageCode    string    `json:"stageCode"`
	Progress     int       `json:"progress"`
	Results      []Result  `json:"results"`
	Favorite     bool      `json:"favorite,omitempty"`
	Error        string    `json:"error,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
	StartedAt    time.Time `json:"startedAt,omitempty"`
	FinishedAt   time.Time `json:"finishedAt,omitempty"`
}

type Result struct {
	Index          int    `json:"index"`
	OK             bool   `json:"ok"`
	Status         Status `json:"status"`
	StatusText     string `json:"statusText"`
	StatusCode     string `json:"statusCode"`
	ImageURL       string `json:"imageUrl,omitempty"`
	RemoteURL      string `json:"remoteUrl,omitempty"`
	RemoteThumbURL string `json:"remoteThumbUrl,omitempty"`
	UploadError    string `json:"uploadError,omitempty"`
	Mime           string `json:"mime,omitempty"`
	Bytes          int64  `json:"bytes,omitempty"`
	RevisedPrompt  string `json:"revisedPrompt,omitempty"`
	ActualSize     string `json:"actualSize,omitempty"`
	ActualQuality  string `json:"actualQuality,omitempty"`
	OutputFormat   string `json:"outputFormat,omitempty"`
	Error          string `json:"error,omitempty"`
	ErrorText      string `json:"errorText,omitempty"`
	ErrorCode      string `json:"errorCode,omitempty"`
	ErrorEnglish   string `json:"errorEnglish,omitempty"`
	ElapsedMs      int64  `json:"elapsedMs,omitempty"`
}

type CreateRequest struct {
	Mode         Mode     `json:"mode"`
	Prompt       string   `json:"prompt"`
	Ratio        string   `json:"ratio"`
	Resolution   string   `json:"resolution"`
	Quality      string   `json:"quality"`
	OutputFormat string   `json:"outputFormat"`
	Count        int      `json:"count"`
	Concurrency  int      `json:"concurrency"`
	UploadIDs    []string `json:"uploadIds"`
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
	result := Result{Index: index, OK: status == StatusSucceeded, Status: status, StatusText: meta.Chinese, StatusCode: meta.Code, Error: err}
	if strings.TrimSpace(err) != "" {
		errorMeta := ErrorMeta(err)
		result.ErrorText = errorMeta.Chinese
		result.ErrorCode = errorMeta.Code
		result.ErrorEnglish = errorMeta.English
	}
	return result
}

func ErrorMeta(raw string) Meta {
	raw = strings.TrimSpace(raw)
	lower := strings.ToLower(raw)
	if raw == "" {
		return Meta{}
	}
	if strings.Contains(lower, "unexpected eof") {
		return Meta{"E_UPSTREAM_EOF", "upstream_response_truncated", "上游响应提前结束"}
	}
	if lower == "eof" || strings.Contains(lower, "empty response") {
		return Meta{"E_UPSTREAM_EMPTY", "upstream_empty_response", "上游返回空响应"}
	}
	if match := regexp.MustCompile(`(?i)http\s+(\d{3})`).FindStringSubmatch(raw); len(match) == 2 {
		return httpErrorMeta(match[1], lower)
	}
	if strings.Contains(lower, "context deadline exceeded") || strings.Contains(lower, "timeout") {
		return Meta{"E_UPSTREAM_TIMEOUT", "upstream_timeout", "上游请求超时"}
	}
	if strings.Contains(lower, "context canceled") || strings.Contains(raw, "已取消") {
		return Meta{"E_TASK_CANCELLED", "task_cancelled", "任务已取消"}
	}
	if strings.Contains(lower, "unauthorized") || strings.Contains(lower, "invalid api key") || strings.Contains(lower, "invalid key") || strings.Contains(lower, "forbidden") {
		return Meta{"E_UPSTREAM_AUTH", "upstream_auth_failed", "上游鉴权失败"}
	}
	if strings.Contains(lower, "rate limit") || strings.Contains(lower, "too many requests") || strings.Contains(lower, "429") {
		return Meta{"E_UPSTREAM_RATE_LIMIT", "upstream_rate_limited", "上游请求限流"}
	}
	if strings.Contains(lower, "quota") || strings.Contains(lower, "insufficient") || strings.Contains(lower, "balance") || strings.Contains(lower, "billing") {
		return Meta{"E_UPSTREAM_QUOTA", "upstream_quota_or_balance_insufficient", "上游额度或余额不足"}
	}
	if strings.Contains(lower, "unsupported parameter") || strings.Contains(lower, "unsupported_param") || strings.Contains(lower, "unknown parameter") || strings.Contains(lower, "invalid parameter") {
		return Meta{"E_PROVIDER_UNSUPPORTED_PARAM", "provider_unsupported_parameter", "上游不支持当前参数"}
	}
	if strings.Contains(lower, "unsupported output") || strings.Contains(lower, "output_format") || strings.Contains(lower, "unsupported format") {
		return Meta{"E_OUTPUT_FORMAT_UNSUPPORTED", "output_format_unsupported", "上游不支持当前输出格式"}
	}
	if strings.Contains(lower, "connection refused") || strings.Contains(lower, "no such host") || strings.Contains(lower, "tls") || strings.Contains(lower, "connection reset") {
		return Meta{"E_UPSTREAM_NETWORK", "upstream_network_error", "上游网络连接失败"}
	}
	if strings.Contains(lower, "request body too large") || strings.Contains(lower, "payload too large") || strings.Contains(lower, "file too large") {
		return Meta{"E_IMAGE_TOO_LARGE", "image_too_large", "图片或请求体过大"}
	}
	if strings.Contains(raw, "没有返回可用图片") || strings.Contains(lower, "no usable image") {
		return Meta{"E_UPSTREAM_NO_IMAGE", "upstream_no_image", "上游没有返回可用图片"}
	}
	if strings.Contains(lower, "invalid character") || strings.Contains(lower, "cannot unmarshal") || strings.Contains(lower, "bad json") {
		return Meta{"E_UPSTREAM_BAD_JSON", "upstream_bad_json", "上游返回不是有效 JSON"}
	}
	if strings.Contains(raw, "保存") || strings.Contains(lower, "permission denied") || strings.Contains(lower, "access is denied") {
		return Meta{"E_SAVE_IMAGE_FAILED", "save_image_failed", "保存图片失败"}
	}
	if strings.Contains(raw, "参考图") {
		return Meta{"E_REFERENCE_IMAGE", "reference_image_error", "参考图处理失败"}
	}
	return Meta{"E_UNKNOWN", "unknown_error", "任务执行失败"}
}

func httpErrorMeta(statusCode string, lower string) Meta {
	switch statusCode {
	case "400":
		if strings.Contains(lower, "unsupported") || strings.Contains(lower, "unknown parameter") || strings.Contains(lower, "invalid parameter") {
			return Meta{"E_PROVIDER_UNSUPPORTED_PARAM", "provider_unsupported_parameter", "上游不支持当前参数"}
		}
		return Meta{"E_UPSTREAM_BAD_REQUEST", "upstream_bad_request", "上游认为请求参数无效"}
	case "401", "403":
		return Meta{"E_UPSTREAM_AUTH", "upstream_auth_failed", "上游鉴权失败"}
	case "402":
		return Meta{"E_UPSTREAM_QUOTA", "upstream_quota_or_balance_insufficient", "上游额度或余额不足"}
	case "404":
		return Meta{"E_UPSTREAM_ROUTE_NOT_FOUND", "upstream_route_not_found", "上游接口路径不存在"}
	case "405":
		return Meta{"E_UPSTREAM_METHOD_NOT_ALLOWED", "upstream_method_not_allowed", "上游接口不支持当前请求方法或路径"}
	case "408":
		return Meta{"E_UPSTREAM_TIMEOUT", "upstream_timeout", "上游请求超时"}
	case "413":
		return Meta{"E_IMAGE_TOO_LARGE", "image_or_payload_too_large", "图片或请求体过大"}
	case "415":
		return Meta{"E_OUTPUT_FORMAT_UNSUPPORTED", "output_format_or_media_type_unsupported", "上游不支持当前图片或输出格式"}
	case "422":
		return Meta{"E_UPSTREAM_UNPROCESSABLE", "upstream_unprocessable_request", "上游无法处理当前请求"}
	case "429":
		return Meta{"E_UPSTREAM_RATE_LIMIT", "upstream_rate_limited", "上游请求限流"}
	case "500":
		return Meta{"E_UPSTREAM_SERVER", "upstream_server_error", "上游服务内部错误"}
	case "502", "503", "504":
		return Meta{"E_UPSTREAM_GATEWAY", "upstream_gateway_error", "上游网关或服务暂不可用"}
	case "524":
		return Meta{"E_UPSTREAM_GATEWAY_TIMEOUT", "upstream_gateway_timeout", "上游网关等待超时"}
	default:
		return Meta{Code: "E_UPSTREAM_HTTP_" + statusCode, English: "upstream_http_" + statusCode, Chinese: "上游接口返回 HTTP " + statusCode}
	}
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
