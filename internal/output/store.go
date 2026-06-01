package output

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/y08lin4/lyra-image-workbench/internal/spaces"
)

type Store struct {
	root string
}

type SavedImage struct {
	URL      string `json:"url"`
	Path     string `json:"-"`
	Mime     string `json:"mime"`
	Bytes    int64  `json:"bytes"`
	FileName string `json:"fileName"`
	Date     string `json:"date"`
}

func NewStore(root string) (*Store, error) {
	clean := filepath.Clean(root)
	if err := os.MkdirAll(clean, 0o755); err != nil {
		return nil, err
	}
	return &Store{root: clean}, nil
}

func (s *Store) Save(spaceToken string, jobID string, index int, data []byte, mime string) (SavedImage, error) {
	token, err := spaces.NormalizeToken(spaceToken)
	if err != nil {
		return SavedImage{}, err
	}
	if len(data) == 0 {
		return SavedImage{}, errors.New("图片内容为空")
	}
	day := time.Now().Format("2006-01-02")
	dir := filepath.Join(s.root, token, day)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return SavedImage{}, err
	}
	ext := ExtensionFromMime(mime)
	fileName := fmt.Sprintf("%s-%02d.%s", safeSegment(jobID), index+1, ext)
	path := filepath.Join(dir, fileName)
	if err := writeAtomic(path, data, 0o644); err != nil {
		return SavedImage{}, err
	}
	return SavedImage{
		URL:      fmt.Sprintf("/outputs/%s/%s/%s", token, day, fileName),
		Path:     path,
		Mime:     NormalizeMime(mime),
		Bytes:    int64(len(data)),
		FileName: fileName,
		Date:     day,
	}, nil
}

func (s *Store) Resolve(spaceToken string, date string, fileName string) (string, string, error) {
	token, err := spaces.NormalizeToken(spaceToken)
	if err != nil {
		return "", "", err
	}
	if !regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`).MatchString(date) {
		return "", "", errors.New("输出日期无效")
	}
	if filepath.Base(fileName) != fileName || fileName == "" || strings.Contains(fileName, "..") {
		return "", "", errors.New("输出文件名无效")
	}
	root, err := filepath.Abs(s.root)
	if err != nil {
		return "", "", err
	}
	path, err := filepath.Abs(filepath.Join(s.root, token, date, fileName))
	if err != nil {
		return "", "", err
	}
	if !strings.HasPrefix(path, root+string(filepath.Separator)) && path != root {
		return "", "", errors.New("输出路径越界")
	}
	return path, MimeFromFileName(fileName), nil
}

func (s *Store) ResolveURL(outputURL string) (string, string, error) {
	parts := strings.Split(strings.TrimPrefix(outputURL, "/outputs/"), "/")
	if len(parts) != 3 {
		return "", "", errors.New("输出图片地址无效")
	}
	return s.Resolve(parts[0], parts[1], parts[2])
}

func ExtensionFromMime(mime string) string {
	switch NormalizeMime(mime) {
	case "image/jpeg":
		return "jpg"
	case "image/webp":
		return "webp"
	case "image/gif":
		return "gif"
	case "image/avif":
		return "avif"
	default:
		return "png"
	}
}

func MimeFromFileName(fileName string) string {
	switch strings.ToLower(filepath.Ext(fileName)) {
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

func NormalizeMime(mime string) string {
	mime = strings.ToLower(strings.TrimSpace(strings.Split(mime, ";")[0]))
	if strings.HasPrefix(mime, "image/") {
		return mime
	}
	return "image/png"
}

func safeSegment(value string) string {
	value = regexp.MustCompile(`[^a-zA-Z0-9_-]+`).ReplaceAllString(value, "-")
	value = strings.Trim(value, "-")
	if value == "" {
		return "image"
	}
	if len(value) > 80 {
		return value[:80]
	}
	return value
}

func writeAtomic(path string, data []byte, perm os.FileMode) error {
	tmp := fmt.Sprintf("%s.%d.tmp", path, time.Now().UnixNano())
	if err := os.WriteFile(tmp, data, perm); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
