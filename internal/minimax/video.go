package minimax

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const DefaultBaseURL = "https://api.minimaxi.com"

type Client struct {
	httpClient *http.Client
	baseURL    string
}

func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 90 * time.Second},
		baseURL:    DefaultBaseURL,
	}
}

type CreateVideoRequest struct {
	Model            string `json:"model"`
	Prompt           string `json:"prompt"`
	Duration         int    `json:"duration,omitempty"`
	Resolution       string `json:"resolution,omitempty"`
	PromptOptimizer  bool   `json:"prompt_optimizer"`
	FastPretreatment bool   `json:"fast_pretreatment,omitempty"`
	AIGCWatermark    bool   `json:"aigc_watermark,omitempty"`
}

type CreateVideoResponse struct {
	TaskID string `json:"task_id"`
	Base   Base   `json:"base"`
	Raw    any    `json:"raw,omitempty"`
}

type QueryVideoResponse struct {
	TaskID      string `json:"task_id"`
	Status      string `json:"status"`
	FileID      string `json:"file_id,omitempty"`
	VideoWidth  int    `json:"video_width,omitempty"`
	VideoHeight int    `json:"video_height,omitempty"`
	Base        Base   `json:"base"`
	Raw         any    `json:"raw,omitempty"`
}

type FileResponse struct {
	File FileObject `json:"file"`
	Base Base       `json:"base"`
	Raw  any        `json:"raw,omitempty"`
}

type FileObject struct {
	FileID      string `json:"file_id,omitempty"`
	Bytes       int64  `json:"bytes,omitempty"`
	CreatedAt   int64  `json:"created_at,omitempty"`
	Filename    string `json:"filename,omitempty"`
	Purpose     string `json:"purpose,omitempty"`
	DownloadURL string `json:"download_url,omitempty"`
}

type Base struct {
	StatusCode int    `json:"status_code"`
	StatusMsg  string `json:"status_msg"`
}

func (c *Client) CreateVideo(ctx context.Context, apiKey string, req CreateVideoRequest) (CreateVideoResponse, error) {
	if strings.TrimSpace(apiKey) == "" {
		return CreateVideoResponse{}, errors.New("MiniMax API Key 为空")
	}
	if strings.TrimSpace(req.Model) == "" {
		req.Model = "MiniMax-Hailuo-02"
	}
	if strings.TrimSpace(req.Prompt) == "" {
		return CreateVideoResponse{}, errors.New("视频提示词不能为空")
	}
	var payload map[string]any
	data, _ := json.Marshal(req)
	if err := json.Unmarshal(data, &payload); err != nil {
		return CreateVideoResponse{}, err
	}
	compactPayload(payload)

	var out struct {
		TaskID   string `json:"task_id"`
		Base     Base   `json:"base"`
		BaseResp Base   `json:"base_resp"`
	}
	raw, err := c.doJSON(ctx, http.MethodPost, "/v1/video_generation", apiKey, payload, &out)
	if err != nil {
		return CreateVideoResponse{}, err
	}
	base := pickBase(out.Base, out.BaseResp)
	if base.StatusCode != 0 {
		return CreateVideoResponse{}, fmt.Errorf("MiniMax 创建视频失败：%s (%d)", firstNonEmpty(base.StatusMsg, "unknown error"), base.StatusCode)
	}
	if out.TaskID == "" {
		return CreateVideoResponse{}, errors.New("MiniMax 未返回 task_id")
	}
	return CreateVideoResponse{TaskID: out.TaskID, Base: base, Raw: raw}, nil
}

func (c *Client) QueryVideo(ctx context.Context, apiKey, taskID string) (QueryVideoResponse, error) {
	if strings.TrimSpace(apiKey) == "" {
		return QueryVideoResponse{}, errors.New("MiniMax API Key 为空")
	}
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return QueryVideoResponse{}, errors.New("task_id 为空")
	}
	var out struct {
		TaskID      string `json:"task_id"`
		Status      string `json:"status"`
		FileID      string `json:"file_id"`
		VideoWidth  int    `json:"video_width"`
		VideoHeight int    `json:"video_height"`
		Base        Base   `json:"base"`
		BaseResp    Base   `json:"base_resp"`
	}
	raw, err := c.doJSON(ctx, http.MethodGet, "/v1/query/video_generation?task_id="+url.QueryEscape(taskID), apiKey, nil, &out)
	if err != nil {
		return QueryVideoResponse{}, err
	}
	base := pickBase(out.Base, out.BaseResp)
	if base.StatusCode != 0 {
		return QueryVideoResponse{}, fmt.Errorf("MiniMax 查询视频失败：%s (%d)", firstNonEmpty(base.StatusMsg, "unknown error"), base.StatusCode)
	}
	return QueryVideoResponse{TaskID: firstNonEmpty(out.TaskID, taskID), Status: out.Status, FileID: out.FileID, VideoWidth: out.VideoWidth, VideoHeight: out.VideoHeight, Base: base, Raw: raw}, nil
}

func (c *Client) RetrieveFile(ctx context.Context, apiKey, fileID string) (FileResponse, error) {
	if strings.TrimSpace(apiKey) == "" {
		return FileResponse{}, errors.New("MiniMax API Key 为空")
	}
	fileID = strings.TrimSpace(fileID)
	if fileID == "" {
		return FileResponse{}, errors.New("file_id 为空")
	}
	var out struct {
		File     FileObject `json:"file"`
		Base     Base       `json:"base"`
		BaseResp Base       `json:"base_resp"`
	}
	raw, err := c.doJSON(ctx, http.MethodGet, "/v1/files/retrieve?file_id="+url.QueryEscape(fileID), apiKey, nil, &out)
	if err != nil {
		return FileResponse{}, err
	}
	base := pickBase(out.Base, out.BaseResp)
	if base.StatusCode != 0 {
		return FileResponse{}, fmt.Errorf("MiniMax 获取文件失败：%s (%d)", firstNonEmpty(base.StatusMsg, "unknown error"), base.StatusCode)
	}
	return FileResponse{File: out.File, Base: base, Raw: raw}, nil
}

func (c *Client) doJSON(ctx context.Context, method, path, apiKey string, payload any, out any) (any, error) {
	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(c.baseURL, "/")+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(apiKey))
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	data, err := io.ReadAll(io.LimitReader(res.Body, 4*1024*1024))
	if err != nil {
		return nil, err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("MiniMax HTTP %d：%s", res.StatusCode, strings.TrimSpace(string(data)))
	}
	if err := json.Unmarshal(data, out); err != nil {
		return nil, err
	}
	var raw any
	_ = json.Unmarshal(data, &raw)
	return raw, nil
}

func compactPayload(payload map[string]any) {
	for key, value := range payload {
		switch typed := value.(type) {
		case string:
			if strings.TrimSpace(typed) == "" {
				delete(payload, key)
			}
		case float64:
			if typed == 0 {
				delete(payload, key)
			}
		case int:
			if typed == 0 {
				delete(payload, key)
			}
		}
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func pickBase(values ...Base) Base {
	for _, value := range values {
		if value.StatusCode != 0 || strings.TrimSpace(value.StatusMsg) != "" {
			return value
		}
	}
	return Base{}
}
