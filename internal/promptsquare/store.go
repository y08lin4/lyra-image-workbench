package promptsquare

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	MaxImageBytes = 8 * 1024 * 1024
)

var allowedImageTypes = map[string]string{
	"image/png":  ".png",
	"image/jpeg": ".jpg",
	"image/webp": ".webp",
	"image/gif":  ".gif",
	"image/avif": ".avif",
}

var (
	ErrItemNotFound     = errors.New("广场作品不存在")
	ErrUsernameRequired = errors.New("用户名不能为空")
)

type Item struct {
	ID                string            `json:"id"`
	Title             string            `json:"title"`
	Prompt            string            `json:"prompt"`
	Negative          string            `json:"negativePrompt,omitempty"`
	Model             string            `json:"model,omitempty"`
	Ratio             string            `json:"ratio,omitempty"`
	Quality           string            `json:"quality,omitempty"`
	OutputFormat      string            `json:"outputFormat,omitempty"`
	Params            map[string]string `json:"params,omitempty"`
	ImageURL          string            `json:"imageUrl,omitempty"`
	ThumbnailURL      string            `json:"thumbnailUrl,omitempty"`
	Tags              []string          `json:"tags,omitempty"`
	Author            AuthorName        `json:"author"`
	AuthorDisplayName string            `json:"authorDisplayName,omitempty"`
	AuthorURL         string            `json:"authorUrl,omitempty"`
	Source            Source            `json:"source"`
	Status            string            `json:"status"`
	LikeCount         int               `json:"likeCount"`
	LikedByMe         bool              `json:"likedByMe"`
	DailyRank         int               `json:"dailyRank"`
	Permanent         bool              `json:"permanent"`
	SourceTaskID      string            `json:"sourceTaskId,omitempty"`
	LikedBy           []string          `json:"likedBy,omitempty"`
	CreatedAt         string            `json:"createdAt"`
	UpdatedAt         string            `json:"updatedAt"`
}

type AuthorName string

func (a AuthorName) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(a))
}

func (a *AuthorName) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err == nil {
		*a = AuthorName(strings.TrimSpace(value))
		return nil
	}
	var legacy Author
	if err := json.Unmarshal(data, &legacy); err != nil {
		return err
	}
	*a = AuthorName(strings.TrimSpace(legacy.Name))
	return nil
}

type Author struct {
	Name string `json:"name"`
	URL  string `json:"url,omitempty"`
}

type Source struct {
	Type    string `json:"type"`
	Name    string `json:"name,omitempty"`
	URL     string `json:"url,omitempty"`
	License string `json:"license,omitempty"`
}

type CreateRequest struct {
	Title             string
	Prompt            string
	Negative          string
	Model             string
	Tags              []string
	ImageURL          string
	SourceName        string
	SourceURL         string
	License           string
	AuthorName        string
	AuthorUsername    string
	AuthorDisplayName string
	AuthorURL         string
	Params            map[string]string
	ImageHeader       *multipart.FileHeader
}

type SubmitFromResultRequest struct {
	Title             string
	Prompt            string
	Negative          string
	Model             string
	Ratio             string
	Quality           string
	OutputFormat      string
	Tags              []string
	Author            string
	AuthorDisplayName string
	SourceTaskID      string
	SourceImagePath   string
	SourceImageMime   string
}

type Store struct {
	root string
	mu   sync.Mutex
	now  func() time.Time
}

func NewStore(dataDir string) (*Store, error) {
	root := filepath.Join(dataDir, "prompt_square")
	if err := os.MkdirAll(filepath.Join(root, "images"), 0o755); err != nil {
		return nil, err
	}
	return &Store{root: root, now: time.Now}, nil
}

func (s *Store) List() ([]Item, error) {
	return s.ListForUser("")
}

func (s *Store) ListForUser(username string) ([]Item, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	items, err := s.loadLocked()
	if err != nil {
		return nil, err
	}
	sort.SliceStable(items, func(i, j int) bool {
		return createdAfter(items[i], items[j])
	})
	return publicItems(items, username), nil
}

func (s *Store) Create(req CreateRequest) (Item, error) {
	item, err := s.newItem(itemDraft{
		Title:             req.Title,
		Prompt:            req.Prompt,
		Negative:          req.Negative,
		Model:             req.Model,
		Ratio:             req.Params["ratio"],
		Quality:           req.Params["quality"],
		OutputFormat:      req.Params["outputFormat"],
		Tags:              req.Tags,
		Author:            firstNonEmpty(req.AuthorUsername, req.AuthorName),
		AuthorDisplayName: firstNonEmpty(req.AuthorDisplayName, req.AuthorName, req.AuthorUsername),
		AuthorURL:         req.AuthorURL,
	})
	if err != nil {
		return Item{}, err
	}
	license := strings.TrimSpace(req.License)
	if license == "" {
		license = "user_submitted"
	}

	item.ImageURL = strings.TrimSpace(req.ImageURL)
	item.ThumbnailURL = item.ImageURL
	item.Source = Source{
		Type:    "user_upload",
		Name:    strings.TrimSpace(req.SourceName),
		URL:     strings.TrimSpace(req.SourceURL),
		License: license,
	}
	if item.Source.URL != "" {
		item.Source.Type = "external"
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if req.ImageHeader != nil {
		url, err := s.saveImageLocked(item.ID, req.ImageHeader)
		if err != nil {
			return Item{}, err
		}
		item.ImageURL = url
		item.ThumbnailURL = url
	}

	items, err := s.loadLocked()
	if err != nil {
		return Item{}, err
	}
	items = append(items, item)
	if err := s.saveLocked(items); err != nil {
		return Item{}, err
	}
	return publicItem(item, string(item.Author)), nil
}

type itemDraft struct {
	Title             string
	Prompt            string
	Negative          string
	Model             string
	Ratio             string
	Quality           string
	OutputFormat      string
	Tags              []string
	Author            string
	AuthorDisplayName string
	AuthorURL         string
	SourceTaskID      string
	Params            map[string]string
}

func (s *Store) SubmitFromResult(req SubmitFromResultRequest) (Item, error) {
	item, err := s.newItem(itemDraft{
		Title:             req.Title,
		Prompt:            req.Prompt,
		Negative:          req.Negative,
		Model:             req.Model,
		Ratio:             req.Ratio,
		Quality:           req.Quality,
		OutputFormat:      req.OutputFormat,
		Tags:              req.Tags,
		Author:            req.Author,
		AuthorDisplayName: req.AuthorDisplayName,
		SourceTaskID:      req.SourceTaskID,
	})
	if err != nil {
		return Item{}, err
	}
	if strings.TrimSpace(req.SourceImagePath) == "" {
		return Item{}, errors.New("任务图片不存在")
	}
	item.Permanent = true
	item.SourceTaskID = strings.TrimSpace(req.SourceTaskID)
	item.Source = Source{Type: "task_result", Name: item.SourceTaskID, License: "user_submitted"}

	s.mu.Lock()
	defer s.mu.Unlock()

	url, err := s.copyImageLocked(item.ID, req.SourceImagePath, req.SourceImageMime)
	if err != nil {
		return Item{}, err
	}
	item.ImageURL = url
	item.ThumbnailURL = url

	items, err := s.loadLocked()
	if err != nil {
		_ = s.removeImageURLLocked(item.ImageURL)
		return Item{}, err
	}
	items = append(items, item)
	if err := s.saveLocked(items); err != nil {
		_ = s.removeImageURLLocked(item.ImageURL)
		return Item{}, err
	}
	return publicItem(item, string(item.Author)), nil
}

func (s *Store) SetLike(id string, username string, liked bool) (Item, error) {
	user := normalizeUsername(username)
	if user == "" {
		return Item{}, ErrUsernameRequired
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	items, err := s.loadLocked()
	if err != nil {
		return Item{}, err
	}
	for i := range items {
		if items[i].ID != id {
			continue
		}
		items[i].LikedBy = setLikedBy(items[i].LikedBy, user, liked)
		items[i].LikeCount = len(items[i].LikedBy)
		items[i].UpdatedAt = s.now().UTC().Format(time.RFC3339)
		if err := s.saveLocked(items); err != nil {
			return Item{}, err
		}
		return publicItem(items[i], user), nil
	}
	return Item{}, ErrItemNotFound
}

func (s *Store) DailyForUser(username string, now time.Time) ([]Item, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	items, err := s.loadLocked()
	if err != nil {
		return nil, err
	}
	start := localDayStart(now)
	end := start.AddDate(0, 0, 1)
	daily := make([]Item, 0, len(items))
	for _, item := range items {
		created, err := time.Parse(time.RFC3339, item.CreatedAt)
		if err != nil {
			continue
		}
		localCreated := created.In(start.Location())
		if !localCreated.Before(start) && localCreated.Before(end) {
			daily = append(daily, item)
		}
	}
	sort.SliceStable(daily, func(i, j int) bool {
		if daily[i].LikeCount != daily[j].LikeCount {
			return daily[i].LikeCount > daily[j].LikeCount
		}
		return createdBefore(daily[i], daily[j])
	})
	for i := range daily {
		daily[i].DailyRank = i + 1
	}
	return publicItems(daily, username), nil
}

func (s *Store) Daily(username string) ([]Item, error) {
	return s.DailyForUser(username, s.now())
}

func (s *Store) MineForUser(username string) ([]Item, error) {
	user := normalizeUsername(username)
	if user == "" {
		return nil, ErrUsernameRequired
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	items, err := s.loadLocked()
	if err != nil {
		return nil, err
	}
	mine := make([]Item, 0, len(items))
	for _, item := range items {
		if normalizeUsername(string(item.Author)) == user {
			mine = append(mine, item)
		}
	}
	sort.SliceStable(mine, func(i, j int) bool {
		return createdAfter(mine[i], mine[j])
	})
	return publicItems(mine, user), nil
}

func (s *Store) newItem(req itemDraft) (Item, error) {
	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		return Item{}, errors.New("提示词不能为空")
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = firstPromptLine(prompt)
	}
	if len([]rune(title)) > 80 {
		title = string([]rune(title)[:80])
	}

	now := s.now().UTC()
	author := firstNonEmpty(req.Author, "anonymous")
	displayName := firstNonEmpty(req.AuthorDisplayName, author, "匿名用户")
	params := compactParams(req.Params)
	ratio := firstNonEmpty(req.Ratio, params["ratio"])
	quality := firstNonEmpty(req.Quality, params["quality"])
	outputFormat := firstNonEmpty(req.OutputFormat, params["outputFormat"])
	params = withParam(params, "ratio", ratio)
	params = withParam(params, "quality", quality)
	params = withParam(params, "outputFormat", outputFormat)

	return Item{
		ID:                "prompt_" + now.Format("20060102_150405") + "_" + randomHex(4),
		Title:             title,
		Prompt:            prompt,
		Negative:          strings.TrimSpace(req.Negative),
		Model:             strings.TrimSpace(req.Model),
		Ratio:             ratio,
		Quality:           quality,
		OutputFormat:      outputFormat,
		Params:            params,
		Tags:              normalizeTags(req.Tags),
		Author:            AuthorName(author),
		AuthorDisplayName: displayName,
		AuthorURL:         strings.TrimSpace(req.AuthorURL),
		SourceTaskID:      strings.TrimSpace(req.SourceTaskID),
		Source:            Source{Type: "user_upload", License: "user_submitted"},
		Status:            "published",
		Permanent:         false,
		CreatedAt:         now.Format(time.RFC3339),
		UpdatedAt:         now.Format(time.RFC3339),
	}, nil
}

func (s *Store) ResolveImage(file string) (string, string, error) {
	clean := filepath.Base(file)
	if clean == "." || clean == string(filepath.Separator) || clean == "" {
		return "", "", os.ErrNotExist
	}
	path := filepath.Join(s.root, "images", clean)
	info, err := os.Stat(path)
	if err != nil {
		return "", "", err
	}
	if info.IsDir() {
		return "", "", os.ErrNotExist
	}
	mime := mimeFromExt(filepath.Ext(clean))
	return path, mime, nil
}

func (s *Store) copyImageLocked(id string, sourcePath string, sourceMime string) (string, error) {
	info, err := os.Stat(sourcePath)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return "", errors.New("任务图片不存在")
	}
	if info.Size() > MaxImageBytes {
		return "", fmt.Errorf("图片不能超过 %dMB", MaxImageBytes/1024/1024)
	}

	src, err := os.Open(sourcePath)
	if err != nil {
		return "", err
	}
	defer src.Close()

	head := make([]byte, 512)
	n, _ := io.ReadFull(src, head)
	head = head[:n]
	mime := normalizeImageMime(sourceMime)
	if mime == "" {
		mime = normalizeImageMime(http.DetectContentType(head))
	}
	ext, ok := allowedImageTypes[mime]
	if !ok {
		return "", errors.New("图片仅支持 PNG、JPG、WEBP、GIF、AVIF")
	}
	if _, err := src.Seek(0, io.SeekStart); err != nil {
		return "", err
	}

	name := id + ext
	dstPath := filepath.Join(s.root, "images", name)
	dst, err := os.OpenFile(dstPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	limited := &io.LimitedReader{R: src, N: MaxImageBytes + 1}
	written, err := io.Copy(dst, limited)
	if err != nil {
		return "", err
	}
	if written > MaxImageBytes {
		_ = os.Remove(dstPath)
		return "", fmt.Errorf("图片不能超过 %dMB", MaxImageBytes/1024/1024)
	}
	return "/api/prompt-square/images/" + name, nil
}

func (s *Store) removeImageURLLocked(imageURL string) error {
	name := filepath.Base(strings.TrimSpace(imageURL))
	if name == "." || name == "" || strings.Contains(name, "..") {
		return nil
	}
	return os.Remove(filepath.Join(s.root, "images", name))
}

func publicItems(items []Item, username string) []Item {
	out := make([]Item, len(items))
	for i, item := range items {
		out[i] = publicItem(item, username)
	}
	return out
}

func publicItem(item Item, username string) Item {
	viewer := normalizeUsername(username)
	item.LikedBy = normalizeLikedBy(item.LikedBy)
	item.LikeCount = len(item.LikedBy)
	item.LikedByMe = viewer != "" && containsUser(item.LikedBy, viewer)
	item.LikedBy = nil
	if item.AuthorDisplayName == "" {
		item.AuthorDisplayName = string(item.Author)
	}
	item.Ratio = firstNonEmpty(item.Ratio, item.Params["ratio"])
	item.Quality = firstNonEmpty(item.Quality, item.Params["quality"])
	item.OutputFormat = firstNonEmpty(item.OutputFormat, item.Params["outputFormat"])
	return item
}

func normalizeLikedBy(users []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(users))
	for _, user := range users {
		normalized := normalizeUsername(user)
		if normalized == "" || seen[normalized] {
			continue
		}
		seen[normalized] = true
		out = append(out, normalized)
	}
	sort.Strings(out)
	return out
}

func setLikedBy(users []string, username string, liked bool) []string {
	normalized := normalizeUsername(username)
	current := normalizeLikedBy(users)
	if normalized == "" {
		return current
	}
	if liked {
		if containsUser(current, normalized) {
			return current
		}
		current = append(current, normalized)
		sort.Strings(current)
		return current
	}
	out := current[:0]
	for _, user := range current {
		if user != normalized {
			out = append(out, user)
		}
	}
	return out
}

func containsUser(users []string, username string) bool {
	normalized := normalizeUsername(username)
	for _, user := range users {
		if normalizeUsername(user) == normalized {
			return true
		}
	}
	return false
}

func normalizeUsername(username string) string {
	return strings.ToLower(strings.TrimSpace(username))
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func withParam(params map[string]string, key string, value string) map[string]string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return params
	}
	if params == nil {
		params = map[string]string{}
	}
	params[key] = trimmed
	return params
}

func createdAfter(a Item, b Item) bool {
	at, aOK := parseCreatedAt(a.CreatedAt)
	bt, bOK := parseCreatedAt(b.CreatedAt)
	if aOK && bOK {
		return at.After(bt)
	}
	return a.CreatedAt > b.CreatedAt
}

func createdBefore(a Item, b Item) bool {
	at, aOK := parseCreatedAt(a.CreatedAt)
	bt, bOK := parseCreatedAt(b.CreatedAt)
	if aOK && bOK {
		return at.Before(bt)
	}
	return a.CreatedAt < b.CreatedAt
}

func parseCreatedAt(value string) (time.Time, bool) {
	created, err := time.Parse(time.RFC3339, strings.TrimSpace(value))
	if err != nil {
		return time.Time{}, false
	}
	return created, true
}

func localDayStart(now time.Time) time.Time {
	local := now.In(time.Local)
	year, month, day := local.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.Local)
}

func normalizeImageMime(value string) string {
	mime := strings.ToLower(strings.TrimSpace(strings.Split(value, ";")[0]))
	switch mime {
	case "image/jpg":
		return "image/jpeg"
	case "image/png", "image/jpeg", "image/webp", "image/gif", "image/avif":
		return mime
	default:
		return ""
	}
}

func (s *Store) saveImageLocked(id string, header *multipart.FileHeader) (string, error) {
	if header.Size > MaxImageBytes {
		return "", fmt.Errorf("图片不能超过 %dMB", MaxImageBytes/1024/1024)
	}
	src, err := header.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	head := make([]byte, 512)
	n, _ := io.ReadFull(src, head)
	head = head[:n]
	mime := http.DetectContentType(head)
	ext, ok := allowedImageTypes[mime]
	if !ok {
		return "", errors.New("图片仅支持 PNG、JPG、WEBP、GIF、AVIF")
	}
	if _, err := src.Seek(0, io.SeekStart); err != nil {
		return "", err
	}

	name := id + ext
	dstPath := filepath.Join(s.root, "images", name)
	dst, err := os.OpenFile(dstPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	limited := &io.LimitedReader{R: src, N: MaxImageBytes + 1}
	written, err := io.Copy(dst, limited)
	if err != nil {
		return "", err
	}
	if written > MaxImageBytes {
		return "", fmt.Errorf("图片不能超过 %dMB", MaxImageBytes/1024/1024)
	}
	return "/api/prompt-square/images/" + name, nil
}

func (s *Store) loadLocked() ([]Item, error) {
	path := filepath.Join(s.root, "items.json")
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return []Item{}, nil
	}
	if err != nil {
		return nil, err
	}
	var items []Item
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func (s *Store) saveLocked(items []Item) error {
	path := filepath.Join(s.root, "items.json")
	tmp := path + ".tmp"
	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(tmp, append(data, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func normalizeTags(tags []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, tag := range tags {
		for _, part := range strings.Split(tag, ",") {
			normalized := strings.TrimSpace(part)
			if normalized == "" {
				continue
			}
			key := strings.ToLower(normalized)
			if seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, normalized)
			if len(out) >= 12 {
				return out
			}
		}
	}
	return out
}

func compactParams(params map[string]string) map[string]string {
	out := map[string]string{}
	for key, value := range params {
		k := strings.TrimSpace(key)
		v := strings.TrimSpace(value)
		if k != "" && v != "" {
			out[k] = v
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func firstPromptLine(prompt string) string {
	line := strings.TrimSpace(strings.Split(prompt, "\n")[0])
	if line == "" {
		return "未命名提示词"
	}
	runes := []rune(line)
	if len(runes) > 36 {
		return string(runes[:36]) + "..."
	}
	return line
}

func randomHex(size int) string {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%04x", time.Now().UnixNano()%0xffff)
	}
	return hex.EncodeToString(buf)
}

func mimeFromExt(ext string) string {
	switch strings.ToLower(ext) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".webp":
		return "image/webp"
	case ".gif":
		return "image/gif"
	case ".avif":
		return "image/avif"
	default:
		return "image/png"
	}
}
