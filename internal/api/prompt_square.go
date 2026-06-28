package api

import (
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/y08lin4/lyra-image-workbench/internal/jobs"
	"github.com/y08lin4/lyra-image-workbench/internal/promptsquare"
)

type promptSquareJobReader interface {
	Get(spaceToken string, id string) (jobs.Job, bool, error)
}

type promptSquareOutputResolver interface {
	Resolve(spaceToken string, date string, fileName string) (string, string, error)
	ResolveURL(outputURL string) (string, string, error)
}

type PromptSquareHandler struct {
	store  *promptsquare.Store
	jobs   promptSquareJobReader
	output promptSquareOutputResolver
}

func NewPromptSquareHandler(store *promptsquare.Store) PromptSquareHandler {
	return PromptSquareHandler{store: store}
}

func NewPromptSquareHandlerWithResults(store *promptsquare.Store, jobReader promptSquareJobReader, outputResolver promptSquareOutputResolver) PromptSquareHandler {
	return PromptSquareHandler{store: store, jobs: jobReader, output: outputResolver}
}

func (h PromptSquareHandler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.store.ListForUser(promptSquareUsername(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "PROMPT_SQUARE_LIST_FAILED", "读取提示词广场失败")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "items": items})
}

func (h PromptSquareHandler) Create(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, promptsquare.MaxImageBytes+2*1024*1024)
	if err := r.ParseMultipartForm(promptsquare.MaxImageBytes + 2*1024*1024); err != nil {
		writeError(w, http.StatusBadRequest, "PROMPT_SQUARE_UPLOAD_INVALID", "上传请求无效或图片过大")
		return
	}

	var image = firstFile(r, "image")
	tags := []string{}
	if r.MultipartForm != nil && r.MultipartForm.Value != nil {
		tags = r.MultipartForm.Value["tags"]
	}
	authorName := strings.TrimSpace(r.FormValue("authorName"))
	username := promptSquareUsername(r)
	if authorName == "" {
		authorName = username
	}
	item, err := h.store.Create(promptsquare.CreateRequest{
		Title:             r.FormValue("title"),
		Prompt:            r.FormValue("prompt"),
		Negative:          r.FormValue("negativePrompt"),
		Model:             r.FormValue("model"),
		Tags:              tags,
		ImageURL:          r.FormValue("imageUrl"),
		SourceName:        r.FormValue("sourceName"),
		SourceURL:         r.FormValue("sourceUrl"),
		License:           r.FormValue("license"),
		AuthorName:        authorName,
		AuthorUsername:    username,
		AuthorDisplayName: promptSquareDisplayName(r, authorName),
		AuthorURL:         r.FormValue("authorUrl"),
		ImageHeader:       image,
		Params: map[string]string{
			"ratio":        r.FormValue("ratio"),
			"resolution":   r.FormValue("resolution"),
			"quality":      r.FormValue("quality"),
			"outputFormat": r.FormValue("outputFormat"),
		},
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, "PROMPT_SQUARE_CREATE_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "item": item})
}

type promptSquareFromResultRequest struct {
	TaskID             string                            `json:"taskId"`
	ImageIndex         int                               `json:"imageIndex"`
	Title              string                            `json:"title"`
	Tags               []string                          `json:"tags"`
	ReferenceUploadIDs []string                          `json:"referenceUploadIds"`
	References         []promptSquareFromResultReference `json:"references"`
	ReferenceUsageNote string                            `json:"referenceUsageNote"`
}

type promptSquareFromResultReference struct {
	UploadID       string `json:"uploadId"`
	SourceUploadID string `json:"sourceUploadId"`
	OriginalName   string `json:"originalName"`
	FileName       string `json:"fileName"`
	Mime           string `json:"mime"`
	Size           int64  `json:"size"`
	UsageNote      string `json:"usageNote"`
}

func (h PromptSquareHandler) FromResult(w http.ResponseWriter, r *http.Request) {
	if h.jobs == nil || h.output == nil {
		writeError(w, http.StatusServiceUnavailable, "PROMPT_SQUARE_RESULT_DEPENDENCY_MISSING", "广场投稿缺少任务或输出读取依赖")
		return
	}
	defer r.Body.Close()
	var payload promptSquareFromResultRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_JSON", "请求体不是有效 JSON")
		return
	}
	payload.TaskID = strings.TrimSpace(payload.TaskID)
	if payload.TaskID == "" || payload.ImageIndex < 0 {
		writeError(w, http.StatusBadRequest, "PROMPT_SQUARE_FROM_RESULT_INVALID", "任务 ID 或图片序号无效")
		return
	}

	spaceToken := r.Header.Get(userStorageTokenHeader)
	job, ok, err := h.jobs.Get(spaceToken, payload.TaskID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "PROMPT_SQUARE_TASK_READ_FAILED", "读取任务失败")
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, "PROMPT_SQUARE_TASK_NOT_FOUND", "任务不存在")
		return
	}
	result, ok := promptSquareResultByIndex(job, payload.ImageIndex)
	if !ok || !result.OK {
		writeError(w, http.StatusBadRequest, "PROMPT_SQUARE_RESULT_NOT_READY", "任务图片不存在或尚未成功")
		return
	}
	path, mime, err := h.resolveResultImage(job, result)
	if err != nil {
		writeError(w, http.StatusNotFound, "PROMPT_SQUARE_RESULT_IMAGE_NOT_FOUND", "任务图片文件不存在")
		return
	}
	references, err := promptSquareReferencesForSubmit(job, payload, spaceToken)
	if err != nil {
		writeError(w, http.StatusBadRequest, "PROMPT_SQUARE_REFERENCE_INVALID", "任务参考图无效")
		return
	}

	username := promptSquareUsername(r)
	item, err := h.store.SubmitFromResult(promptsquare.SubmitFromResultRequest{
		Title:              payload.Title,
		Prompt:             firstPromptSquareText(result.RevisedPrompt, job.Prompt),
		Model:              job.Model,
		Ratio:              job.Ratio,
		ActualSize:         result.ActualSize,
		Quality:            firstPromptSquareText(result.ActualQuality, job.Quality),
		OutputFormat:       firstPromptSquareText(result.OutputFormat, job.OutputFormat),
		Tags:               payload.Tags,
		Author:             username,
		AuthorDisplayName:  promptSquareDisplayName(r, username),
		SourceTaskID:       job.ID,
		SourceImagePath:    path,
		SourceImageMime:    firstPromptSquareText(mime, result.Mime),
		ReferenceUploadIDs: payload.ReferenceUploadIDs,
		References:         references,
		ReferenceUsageNote: payload.ReferenceUsageNote,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, "PROMPT_SQUARE_FROM_RESULT_FAILED", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "item": item})
}

type promptSquareLikeRequest struct {
	Liked bool `json:"liked"`
}

func (h PromptSquareHandler) Like(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var payload promptSquareLikeRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_JSON", "请求体不是有效 JSON")
		return
	}
	item, err := h.store.SetLike(strings.TrimSpace(r.PathValue("id")), promptSquareUsername(r), payload.Liked)
	if err != nil {
		writePromptSquareStoreError(w, err, "PROMPT_SQUARE_LIKE_FAILED", "更新点赞失败")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "item": item})
}

func (h PromptSquareHandler) Daily(w http.ResponseWriter, r *http.Request) {
	items, err := h.store.Daily(promptSquareUsername(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "PROMPT_SQUARE_DAILY_FAILED", "读取每日榜失败")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "items": items})
}

func (h PromptSquareHandler) Mine(w http.ResponseWriter, r *http.Request) {
	items, err := h.store.MineForUser(promptSquareUsername(r))
	if err != nil {
		writePromptSquareStoreError(w, err, "PROMPT_SQUARE_MINE_FAILED", "读取我的作品失败")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "items": items})
}

func (h PromptSquareHandler) Image(w http.ResponseWriter, r *http.Request) {
	path, mime, err := h.store.ResolveImage(r.PathValue("file"))
	if err != nil {
		writeError(w, http.StatusNotFound, "PROMPT_SQUARE_IMAGE_NOT_FOUND", "提示词广场图片不存在")
		return
	}
	w.Header().Set("Content-Type", mime)
	w.Header().Set("Cache-Control", "public, max-age=3600")
	http.ServeFile(w, r, path)
}

func (h PromptSquareHandler) resolveResultImage(job jobs.Job, result jobs.Result) (string, string, error) {
	if result.OutputDate != "" && result.OutputFileName != "" {
		return h.output.Resolve(job.SpaceToken, result.OutputDate, result.OutputFileName)
	}
	return h.output.ResolveURL(result.ImageURL)
}

func promptSquareReferencesForSubmit(job jobs.Job, payload promptSquareFromResultRequest, spaceToken string) ([]promptsquare.Reference, error) {
	if len(job.References) == 0 {
		return fallbackPromptSquareReferences(payload), nil
	}
	usageByUpload := map[string]string{}
	usageByFile := map[string]string{}
	for _, ref := range payload.References {
		usage := strings.TrimSpace(ref.UsageNote)
		if usage == "" {
			continue
		}
		if id := firstPromptSquareText(ref.UploadID, ref.SourceUploadID); id != "" {
			usageByUpload[id] = usage
		}
		if file := strings.TrimSpace(ref.FileName); file != "" {
			usageByFile[filepath.ToSlash(filepath.Clean(filepath.FromSlash(file)))] = usage
		}
	}
	sourceSpaceToken := firstPromptSquareText(job.SpaceToken, spaceToken)
	out := make([]promptsquare.Reference, 0, len(job.References))
	for _, ref := range job.References {
		rel, err := cleanPromptSquareJobReferencePath(job.ID, ref.FileName)
		if err != nil {
			return nil, err
		}
		usage := firstPromptSquareText(usageByUpload[ref.UploadID], usageByFile[filepath.ToSlash(rel)], payload.ReferenceUsageNote)
		out = append(out, promptsquare.Reference{
			OriginalName:   ref.OriginalName,
			Mime:           ref.Mime,
			Size:           ref.Size,
			UsageNote:      usage,
			SourceUploadID: ref.UploadID,
			SourceTaskID:   job.ID,
			SourcePath:     filepath.ToSlash(filepath.Join("spaces", sourceSpaceToken, rel)),
		})
	}
	return out, nil
}

func fallbackPromptSquareReferences(payload promptSquareFromResultRequest) []promptsquare.Reference {
	seen := map[string]bool{}
	out := make([]promptsquare.Reference, 0, len(payload.References)+len(payload.ReferenceUploadIDs))
	add := func(ref promptsquare.Reference) {
		key := firstPromptSquareText(ref.SourceUploadID, ref.OriginalName)
		if key == "" || seen[key] || len(out) >= 12 {
			return
		}
		seen[key] = true
		out = append(out, ref)
	}
	for _, ref := range payload.References {
		add(promptsquare.Reference{
			OriginalName:   strings.TrimSpace(ref.OriginalName),
			Mime:           strings.TrimSpace(ref.Mime),
			Size:           ref.Size,
			UsageNote:      firstPromptSquareText(ref.UsageNote, payload.ReferenceUsageNote),
			SourceUploadID: firstPromptSquareText(ref.UploadID, ref.SourceUploadID),
		})
	}
	for _, id := range payload.ReferenceUploadIDs {
		add(promptsquare.Reference{
			UsageNote:      strings.TrimSpace(payload.ReferenceUsageNote),
			SourceUploadID: strings.TrimSpace(id),
		})
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func cleanPromptSquareJobReferencePath(jobID string, value string) (string, error) {
	rel := filepath.Clean(filepath.FromSlash(strings.TrimSpace(value)))
	if rel == "." || rel == "" || filepath.IsAbs(rel) || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", errors.New("参考图快照路径无效")
	}
	prefix := filepath.Join("job_refs", strings.TrimSpace(jobID))
	if prefix == "job_refs" || rel != prefix && !strings.HasPrefix(rel, prefix+string(filepath.Separator)) {
		return "", errors.New("参考图快照路径无效")
	}
	return rel, nil
}

func promptSquareResultByIndex(job jobs.Job, index int) (jobs.Result, bool) {
	for _, result := range job.Results {
		if result.Index == index {
			return result, true
		}
	}
	return jobs.Result{}, false
}

func promptSquareUsername(r *http.Request) string {
	return strings.TrimSpace(r.Header.Get("X-User-Name"))
}

func promptSquareDisplayName(r *http.Request, fallback string) string {
	if display := strings.TrimSpace(r.Header.Get("X-User-Display-Name")); display != "" {
		return display
	}
	return strings.TrimSpace(fallback)
}

func firstPromptSquareText(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func writePromptSquareStoreError(w http.ResponseWriter, err error, fallbackCode string, fallbackMessage string) {
	switch {
	case errors.Is(err, promptsquare.ErrItemNotFound):
		writeError(w, http.StatusNotFound, "PROMPT_SQUARE_ITEM_NOT_FOUND", "广场作品不存在")
	case errors.Is(err, promptsquare.ErrUsernameRequired):
		writeError(w, http.StatusUnauthorized, "USER_AUTH_REQUIRED", "请先登录")
	default:
		writeError(w, http.StatusBadRequest, fallbackCode, fallbackMessage)
	}
}

func firstFile(r *http.Request, name string) *multipart.FileHeader {
	if r.MultipartForm == nil || r.MultipartForm.File == nil {
		return nil
	}
	items := r.MultipartForm.File[name]
	if len(items) == 0 {
		return nil
	}
	return items[0]
}
