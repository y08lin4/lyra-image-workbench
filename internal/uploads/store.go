package uploads

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
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/y08lin4/lyra-image-workbench/internal/spaces"
)

const (
	MaxReferenceImages      = 8
	MaxReferenceImageBytes  = 12 * 1024 * 1024
	MaxReferenceUploadBytes = 50 * 1024 * 1024
)

var allowedImageTypes = map[string]string{
	"image/png":  "png",
	"image/jpeg": "jpg",
	"image/webp": "webp",
}

type ReferenceImage struct {
	ID           string `json:"id"`
	OriginalName string `json:"originalName"`
	FileName     string `json:"fileName"`
	Mime         string `json:"mime"`
	Size         int64  `json:"size"`
	CreatedAt    string `json:"createdAt"`
}

type Store struct {
	spaces *spaces.FileStore
}

func NewStore(spaceStore *spaces.FileStore) *Store {
	return &Store{spaces: spaceStore}
}

func (s *Store) SaveReferenceImages(spaceToken string, headers []*multipart.FileHeader) ([]ReferenceImage, error) {
	if len(headers) == 0 {
		return nil, NewUploadError("REFERENCE_IMAGE_MISSING", "请先上传图生图参考图")
	}
	if len(headers) > MaxReferenceImages {
		return nil, NewUploadError("REFERENCE_IMAGE_TOO_MANY", "参考图最多 8 张")
	}

	spaceDir, err := s.spaces.SpaceDir(spaceToken)
	if err != nil {
		return nil, err
	}
	uploadDir := filepath.Join(spaceDir, "uploads")
	if err := os.MkdirAll(uploadDir, 0o700); err != nil {
		return nil, err
	}
	existing, err := s.listReferenceImagesInDir(uploadDir)
	if err != nil {
		return nil, err
	}
	if len(existing)+len(headers) > MaxReferenceImages {
		return nil, NewUploadError("REFERENCE_IMAGE_TOO_MANY", "当前空间参考图最多保留 8 张，请先删除旧图")
	}

	items := make([]ReferenceImage, 0, len(headers))
	for _, header := range headers {
		item, err := s.saveOne(uploadDir, header)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *Store) saveOne(uploadDir string, header *multipart.FileHeader) (ReferenceImage, error) {
	if header.Size > MaxReferenceImageBytes {
		return ReferenceImage{}, NewUploadError("REFERENCE_IMAGE_TOO_LARGE", "单张参考图不能超过 12MB")
	}

	file, err := header.Open()
	if err != nil {
		return ReferenceImage{}, err
	}
	defer file.Close()

	limited := io.LimitReader(file, MaxReferenceImageBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return ReferenceImage{}, err
	}
	if int64(len(data)) > MaxReferenceImageBytes {
		return ReferenceImage{}, NewUploadError("REFERENCE_IMAGE_TOO_LARGE", "单张参考图不能超过 12MB")
	}

	mime := detectMime(data, header.Header.Get("Content-Type"))
	ext, ok := allowedImageTypes[mime]
	if !ok {
		return ReferenceImage{}, NewUploadError("REFERENCE_IMAGE_TYPE_UNSUPPORTED", "参考图仅支持 PNG、JPG、WEBP")
	}

	id, err := newID()
	if err != nil {
		return ReferenceImage{}, err
	}
	now := time.Now().Format(time.RFC3339)
	originalName := safeOriginalName(header.Filename, ext)
	fileName := fmt.Sprintf("%s.%s", id, ext)
	imagePath := filepath.Join(uploadDir, fileName)
	metaPath := filepath.Join(uploadDir, fmt.Sprintf("%s.json", id))

	if err := writeFileAtomic(imagePath, data, 0o600); err != nil {
		return ReferenceImage{}, err
	}
	item := ReferenceImage{
		ID:           id,
		OriginalName: originalName,
		FileName:     fileName,
		Mime:         mime,
		Size:         int64(len(data)),
		CreatedAt:    now,
	}
	meta, err := json.MarshalIndent(item, "", "  ")
	if err != nil {
		return ReferenceImage{}, err
	}
	if err := writeFileAtomic(metaPath, append(meta, '\n'), 0o600); err != nil {
		return ReferenceImage{}, err
	}
	return item, nil
}

func detectMime(data []byte, _ string) string {
	if len(data) >= 8 && bytesHasPrefix(data, []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}) {
		return "image/png"
	}
	if len(data) >= 3 && bytesHasPrefix(data, []byte{0xff, 0xd8, 0xff}) {
		return "image/jpeg"
	}
	if len(data) >= 12 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WEBP" {
		return "image/webp"
	}
	return http.DetectContentType(data)
}

func newID() (string, error) {
	var bytes [12]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", err
	}
	return "ref_" + hex.EncodeToString(bytes[:]), nil
}

func safeOriginalName(name string, ext string) string {
	base := filepath.Base(strings.TrimSpace(name))
	if base == "." || base == string(filepath.Separator) || base == "" {
		base = "reference." + ext
	}
	base = regexp.MustCompile(`[\\/:*?"<>|]+`).ReplaceAllString(base, "-")
	base = regexp.MustCompile(`\s+`).ReplaceAllString(base, "-")
	if len(base) > 96 {
		base = base[:96]
	}
	return base
}

func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	tmp := fmt.Sprintf("%s.%d.tmp", path, time.Now().UnixNano())
	if err := os.WriteFile(tmp, data, perm); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func (s *Store) ListReferenceImages(spaceToken string) ([]ReferenceImage, error) {
	spaceDir, err := s.spaces.SpaceDir(spaceToken)
	if err != nil {
		return nil, err
	}
	uploadDir := filepath.Join(spaceDir, "uploads")
	return s.listReferenceImagesInDir(uploadDir)
}

func (s *Store) listReferenceImagesInDir(uploadDir string) ([]ReferenceImage, error) {
	entries, err := os.ReadDir(uploadDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []ReferenceImage{}, nil
		}
		return nil, err
	}
	items := make([]ReferenceImage, 0)
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		item, err := readReferenceMeta(filepath.Join(uploadDir, entry.Name()))
		if err == nil {
			items = append(items, item)
		}
	}
	sort.Slice(items, func(i int, j int) bool {
		return items[i].CreatedAt > items[j].CreatedAt
	})
	return items, nil
}

func (s *Store) GetReferenceImage(spaceToken string, id string) (ReferenceImage, string, error) {
	spaceDir, err := s.spaces.SpaceDir(spaceToken)
	if err != nil {
		return ReferenceImage{}, "", err
	}
	id = strings.TrimSpace(id)
	if !regexp.MustCompile(`^ref_[a-f0-9]{24}$`).MatchString(id) {
		return ReferenceImage{}, "", NewUploadError("REFERENCE_IMAGE_ID_INVALID", "参考图 ID 无效")
	}
	uploadDir := filepath.Join(spaceDir, "uploads")
	item, err := readReferenceMeta(filepath.Join(uploadDir, id+".json"))
	if err != nil {
		if os.IsNotExist(err) {
			return ReferenceImage{}, "", NewUploadError("REFERENCE_IMAGE_NOT_FOUND", "参考图不存在或已删除")
		}
		return ReferenceImage{}, "", err
	}
	path := filepath.Join(uploadDir, item.FileName)
	return item, path, nil
}

func (s *Store) DeleteReferenceImage(spaceToken string, id string) error {
	item, path, err := s.GetReferenceImage(spaceToken, id)
	if err != nil {
		return err
	}
	_ = os.Remove(path)
	_ = os.Remove(filepath.Join(filepath.Dir(path), item.ID+".json"))
	return nil
}

func readReferenceMeta(path string) (ReferenceImage, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ReferenceImage{}, err
	}
	var item ReferenceImage
	if err := json.Unmarshal(data, &item); err != nil {
		return ReferenceImage{}, err
	}
	return item, nil
}

type UploadError struct {
	Code    string
	Chinese string
}

func NewUploadError(code string, chinese string) UploadError {
	return UploadError{Code: code, Chinese: chinese}
}

func (e UploadError) Error() string {
	return e.Chinese
}

func AsUploadError(err error, target *UploadError) bool {
	return errors.As(err, target)
}

func bytesHasPrefix(data []byte, prefix []byte) bool {
	if len(data) < len(prefix) {
		return false
	}
	for i := range prefix {
		if data[i] != prefix[i] {
			return false
		}
	}
	return true
}
