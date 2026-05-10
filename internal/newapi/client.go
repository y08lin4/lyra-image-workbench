package newapi

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Client struct {
	httpClient *http.Client
}

type InputImage struct {
	Name string
	Path string
	Mime string
}

type Request struct {
	Mode            string
	BaseURL         string
	APIKey          string
	Model           string
	Prompt          string
	Size            string
	Quality         string
	OutputFormat    string
	SkipImageParams bool
	TimeoutSec      int
	InputImages     []InputImage
}

type Image struct {
	Bytes         []byte
	Mime          string
	RevisedPrompt string
	ActualSize    string
	ActualQuality string
	OutputFormat  string
}

type UpstreamError struct {
	StatusCode int
	Message    string
}

func (e UpstreamError) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("上游请求失败：HTTP %d", e.StatusCode)
	}
	return fmt.Sprintf("上游请求失败：HTTP %d：%s", e.StatusCode, e.Message)
}

func NewClient() *Client {
	return &Client{httpClient: &http.Client{}}
}

func (c *Client) Generate(ctx context.Context, req Request) (Image, error) {
	timeout := time.Duration(req.TimeoutSec) * time.Second
	if timeout <= 0 {
		timeout = 600 * time.Second
	}
	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if req.Mode == "image-to-image" {
		return c.editImage(callCtx, req)
	}
	return c.generateText(callCtx, req)
}

func (c *Client) generateText(ctx context.Context, req Request) (Image, error) {
	outputFormat := normalizeOutputFormat(req.OutputFormat)
	body := map[string]any{
		"model":           req.Model,
		"prompt":          req.Prompt,
		"n":               1,
		"response_format": "b64_json",
	}
	if !req.SkipImageParams {
		body["output_format"] = outputFormat
		if req.Size != "" && req.Size != "自动" && req.Size != "auto" {
			body["size"] = req.Size
		}
		if req.Quality != "" {
			body["quality"] = normalizeQuality(req.Quality)
		}
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return Image{}, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, buildURL(req.BaseURL, "images/generations"), bytes.NewReader(payload))
	if err != nil {
		return Image{}, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+req.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Cache-Control", "no-store")
	return c.doAndParse(ctx, httpReq, outputFormat)
}

func (c *Client) editImage(ctx context.Context, req Request) (Image, error) {
	if len(req.InputImages) == 0 {
		return Image{}, errors.New("图生图缺少参考图")
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	outputFormat := normalizeOutputFormat(req.OutputFormat)
	_ = writer.WriteField("model", req.Model)
	_ = writer.WriteField("prompt", req.Prompt)
	_ = writer.WriteField("n", "1")
	_ = writer.WriteField("response_format", "b64_json")
	if !req.SkipImageParams {
		_ = writer.WriteField("output_format", outputFormat)
		if req.Size != "" && req.Size != "自动" && req.Size != "auto" {
			_ = writer.WriteField("size", req.Size)
		}
		if req.Quality != "" {
			_ = writer.WriteField("quality", normalizeQuality(req.Quality))
		}
	}
	for idx, input := range req.InputImages {
		if err := addImagePart(writer, input, idx); err != nil {
			return Image{}, err
		}
	}
	if err := writer.Close(); err != nil {
		return Image{}, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, buildURL(req.BaseURL, "images/edits"), &body)
	if err != nil {
		return Image{}, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+req.APIKey)
	httpReq.Header.Set("Content-Type", writer.FormDataContentType())
	httpReq.Header.Set("Cache-Control", "no-store")
	return c.doAndParse(ctx, httpReq, outputFormat)
}

func addImagePart(writer *multipart.Writer, input InputImage, idx int) error {
	file, err := os.Open(input.Path)
	if err != nil {
		return err
	}
	defer file.Close()
	name := input.Name
	if name == "" {
		name = fmt.Sprintf("input-%d%s", idx+1, filepath.Ext(input.Path))
	}
	mime := strings.ToLower(strings.TrimSpace(strings.Split(input.Mime, ";")[0]))
	if !strings.HasPrefix(mime, "image/") {
		mime = "application/octet-stream"
	}
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", fmt.Sprintf(`form-data; name="image[]"; filename="%s"`, escapeMultipartFilename(name)))
	header.Set("Content-Type", mime)
	part, err := writer.CreatePart(header)
	if err != nil {
		return err
	}
	_, err = io.Copy(part, file)
	return err
}

func escapeMultipartFilename(value string) string {
	return strings.NewReplacer("\\", "\\\\", `"`, "\\\"").Replace(value)
}

func (c *Client) doAndParse(ctx context.Context, req *http.Request, requestedFormat string) (Image, error) {
	res, err := c.httpClient.Do(req)
	if err != nil {
		return Image{}, err
	}
	defer res.Body.Close()
	if !statusCodeOK(res.StatusCode) {
		return Image{}, UpstreamError{StatusCode: res.StatusCode, Message: readErrorMessage(res)}
	}
	contentType := strings.ToLower(strings.TrimSpace(strings.Split(res.Header.Get("Content-Type"), ";")[0]))
	if strings.HasPrefix(contentType, "image/") {
		data, err := io.ReadAll(res.Body)
		if err != nil {
			return Image{}, err
		}
		return Image{Bytes: data, Mime: contentType, OutputFormat: outputFormatFromMime(contentType)}, nil
	}
	var payload struct {
		RevisedPrompt string `json:"revised_prompt"`
		Size          string `json:"size"`
		Quality       string `json:"quality"`
		OutputFormat  string `json:"output_format"`
		Data          []struct {
			B64JSON       string `json:"b64_json"`
			URL           string `json:"url"`
			RevisedPrompt string `json:"revised_prompt"`
			Size          string `json:"size"`
			Quality       string `json:"quality"`
			OutputFormat  string `json:"output_format"`
		} `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		return Image{}, err
	}
	for _, item := range payload.Data {
		if strings.TrimSpace(item.B64JSON) != "" {
			data, err := decodeBase64Image(item.B64JSON)
			if err != nil {
				return Image{}, err
			}
			outputFormat := normalizeOutputFormat(firstNonEmpty(item.OutputFormat, payload.OutputFormat, requestedFormat, "png"))
			return Image{
				Bytes:         data,
				Mime:          mimeFromOutputFormat(outputFormat),
				RevisedPrompt: firstNonEmpty(item.RevisedPrompt, payload.RevisedPrompt),
				ActualSize:    firstNonEmpty(item.Size, payload.Size),
				ActualQuality: firstNonEmpty(item.Quality, payload.Quality),
				OutputFormat:  outputFormat,
			}, nil
		}
		if strings.HasPrefix(item.URL, "http://") || strings.HasPrefix(item.URL, "https://") {
			image, err := c.fetchImageURL(ctx, item.URL)
			if err != nil {
				return Image{}, err
			}
			image.RevisedPrompt = firstNonEmpty(item.RevisedPrompt, payload.RevisedPrompt)
			image.ActualSize = firstNonEmpty(item.Size, payload.Size)
			image.ActualQuality = firstNonEmpty(item.Quality, payload.Quality)
			image.OutputFormat = firstNonEmpty(item.OutputFormat, payload.OutputFormat, image.OutputFormat)
			return image, nil
		}
	}
	return Image{}, errors.New("上游没有返回可用图片")
}

func (c *Client) fetchImageURL(ctx context.Context, url string) (Image, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Image{}, err
	}
	req.Header.Set("Accept", "image/avif,image/webp,image/png,image/jpeg,image/*,*/*;q=0.8")
	res, err := c.httpClient.Do(req)
	if err != nil {
		return Image{}, err
	}
	defer res.Body.Close()
	if !statusCodeOK(res.StatusCode) {
		return Image{}, UpstreamError{StatusCode: res.StatusCode, Message: "图片 URL 下载失败"}
	}
	mime := strings.ToLower(strings.TrimSpace(strings.Split(res.Header.Get("Content-Type"), ";")[0]))
	if !strings.HasPrefix(mime, "image/") {
		mime = "image/png"
	}
	data, err := io.ReadAll(res.Body)
	if err != nil {
		return Image{}, err
	}
	return Image{Bytes: data, Mime: mime, OutputFormat: outputFormatFromMime(mime)}, nil
}

func decodeBase64Image(value string) ([]byte, error) {
	if strings.HasPrefix(value, "data:") {
		comma := strings.Index(value, ",")
		if comma < 0 {
			return nil, errors.New("data URL 无效")
		}
		value = value[comma+1:]
	}
	return base64.StdEncoding.DecodeString(strings.ReplaceAll(value, "\n", ""))
}

func buildURL(baseURL string, path string) string {
	return strings.TrimRight(baseURL, "/") + "/" + strings.TrimLeft(path, "/")
}

func readErrorMessage(res *http.Response) string {
	data, err := io.ReadAll(io.LimitReader(res.Body, 4096))
	if err != nil {
		return ""
	}
	var payload map[string]any
	if json.Unmarshal(data, &payload) == nil {
		if errValue, ok := payload["error"].(map[string]any); ok {
			if msg, ok := errValue["message"].(string); ok {
				return msg
			}
		}
		if msg, ok := payload["message"].(string); ok {
			return msg
		}
	}
	return strings.TrimSpace(string(data))
}

func statusCodeOK(status int) bool {
	return status >= 200 && status < 300
}

func normalizeQuality(value string) string {
	switch strings.TrimSpace(value) {
	case "low", "medium", "high":
		return strings.TrimSpace(value)
	default:
		return "auto"
	}
}

func normalizeOutputFormat(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "jpg", "jpeg":
		return "jpeg"
	case "webp":
		return "webp"
	case "png":
		return "png"
	default:
		return "png"
	}
}

func mimeFromOutputFormat(format string) string {
	switch normalizeOutputFormat(format) {
	case "jpeg":
		return "image/jpeg"
	case "webp":
		return "image/webp"
	default:
		return "image/png"
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

func outputFormatFromMime(mime string) string {
	switch strings.ToLower(strings.TrimSpace(strings.Split(mime, ";")[0])) {
	case "image/jpeg", "image/jpg":
		return "jpeg"
	case "image/webp":
		return "webp"
	case "image/gif":
		return "gif"
	case "image/avif":
		return "avif"
	case "image/png":
		return "png"
	default:
		return ""
	}
}
