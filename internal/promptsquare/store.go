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
}

type Item struct {
	ID           string            `json:"id"`
	Title        string            `json:"title"`
	Prompt       string            `json:"prompt"`
	Negative     string            `json:"negativePrompt,omitempty"`
	Model        string            `json:"model,omitempty"`
	Params       map[string]string `json:"params,omitempty"`
	ImageURL     string            `json:"imageUrl,omitempty"`
	ThumbnailURL string            `json:"thumbnailUrl,omitempty"`
	Tags         []string          `json:"tags,omitempty"`
	Author       Author            `json:"author"`
	Source       Source            `json:"source"`
	Status       string            `json:"status"`
	CreatedAt    string            `json:"createdAt"`
	UpdatedAt    string            `json:"updatedAt"`
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
	Title       string
	Prompt      string
	Negative    string
	Model       string
	Tags        []string
	ImageURL    string
	SourceName  string
	SourceURL   string
	License     string
	AuthorName  string
	AuthorURL   string
	Params      map[string]string
	ImageHeader *multipart.FileHeader
}

type Store struct {
	root string
	mu   sync.Mutex
}

func NewStore(dataDir string) (*Store, error) {
	root := filepath.Join(dataDir, "prompt_square")
	if err := os.MkdirAll(filepath.Join(root, "images"), 0o755); err != nil {
		return nil, err
	}
	return &Store{root: root}, nil
}

func (s *Store) List() ([]Item, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	items, err := s.loadLocked()
	if err != nil {
		return nil, err
	}
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].CreatedAt > items[j].CreatedAt
	})
	return items, nil
}

func (s *Store) Create(req CreateRequest) (Item, error) {
	title := strings.TrimSpace(req.Title)
	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		return Item{}, errors.New("提示词不能为空")
	}
	if title == "" {
		title = firstPromptLine(prompt)
	}
	if len([]rune(title)) > 80 {
		title = string([]rune(title)[:80])
	}

	now := time.Now().UTC().Format(time.RFC3339)
	id := "prompt_" + time.Now().UTC().Format("20060102_150405") + "_" + randomHex(4)
	author := strings.TrimSpace(req.AuthorName)
	if author == "" {
		author = "匿名用户"
	}
	license := strings.TrimSpace(req.License)
	if license == "" {
		license = "user_submitted"
	}

	item := Item{
		ID:       id,
		Title:    title,
		Prompt:   prompt,
		Negative: strings.TrimSpace(req.Negative),
		Model:    strings.TrimSpace(req.Model),
		Params:   compactParams(req.Params),
		ImageURL: strings.TrimSpace(req.ImageURL),
		Tags:     normalizeTags(req.Tags),
		Author: Author{
			Name: author,
			URL:  strings.TrimSpace(req.AuthorURL),
		},
		Source: Source{
			Type:    "user_upload",
			Name:    strings.TrimSpace(req.SourceName),
			URL:     strings.TrimSpace(req.SourceURL),
			License: license,
		},
		Status:    "published",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if item.Source.URL != "" {
		item.Source.Type = "external"
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if req.ImageHeader != nil {
		url, err := s.saveImageLocked(id, req.ImageHeader)
		if err != nil {
			return Item{}, err
		}
		item.ImageURL = url
		item.ThumbnailURL = url
	} else {
		item.ThumbnailURL = item.ImageURL
	}

	items, err := s.loadLocked()
	if err != nil {
		return Item{}, err
	}
	items = append(items, item)
	if err := s.saveLocked(items); err != nil {
		return Item{}, err
	}
	return item, nil
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
		return "", errors.New("图片仅支持 PNG、JPG、WEBP")
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
	default:
		return "image/png"
	}
}
