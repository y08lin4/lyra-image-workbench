package spaces

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Space struct {
	ID           string `json:"id"`
	DisplayName  string `json:"displayName"`
	CreatedAt    string `json:"createdAt"`
	LastOpenedAt string `json:"lastOpenedAt"`
}

type Session struct {
	Space        Space  `json:"space"`
	Token        string `json:"token"`
	TokenPreview string `json:"tokenPreview"`
	Created      bool   `json:"created"`
}

type FileStore struct {
	mu   sync.Mutex
	root string
}

func NewFileStore(root string) (*FileStore, error) {
	spacesRoot := filepath.Join(root, "spaces")
	if err := os.MkdirAll(spacesRoot, 0o755); err != nil {
		return nil, err
	}
	return &FileStore{root: spacesRoot}, nil
}

func (s *FileStore) CreateOrOpenByPassword(password string) (Session, error) {
	token, err := DeriveToken(password)
	if err != nil {
		return Session{}, err
	}
	return s.createOrOpenByToken(token, true)
}

func (s *FileStore) OpenByToken(token string) (Session, error) {
	normalized, err := NormalizeToken(token)
	if err != nil {
		return Session{}, err
	}
	return s.createOrOpenByToken(normalized, false)
}

func (s *FileStore) SpaceDir(token string) (string, error) {
	normalized, err := NormalizeToken(token)
	if err != nil {
		return "", err
	}
	return filepath.Join(s.root, normalized), nil
}

func (s *FileStore) createOrOpenByToken(token string, allowCreate bool) (Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	dir := filepath.Join(s.root, token)
	file := filepath.Join(dir, "space.json")
	now := time.Now().Format(time.RFC3339)
	created := false

	space, err := readSpace(file)
	if err != nil {
		if !os.IsNotExist(err) {
			return Session{}, err
		}
		if !allowCreate {
			return Session{}, NewValidationError("SPACE_NOT_FOUND", "个人空间不存在，请先输入空间密码创建")
		}
		created = true
		space = Space{
			ID:           token,
			DisplayName:  fmt.Sprintf("个人空间 %s", token[:8]),
			CreatedAt:    now,
			LastOpenedAt: now,
		}
	} else {
		space.LastOpenedAt = now
	}

	if err := os.MkdirAll(dir, 0o700); err != nil {
		return Session{}, err
	}
	if err := writeSpace(file, space); err != nil {
		return Session{}, err
	}

	return Session{
		Space:        space,
		Token:        token,
		TokenPreview: previewToken(token),
		Created:      created,
	}, nil
}

func readSpace(file string) (Space, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return Space{}, err
	}
	var space Space
	if err := json.Unmarshal(data, &space); err != nil {
		return Space{}, err
	}
	if _, err := NormalizeToken(space.ID); err != nil {
		return Space{}, NewValidationError("SPACE_FILE_INVALID", "个人空间文件无效")
	}
	return space, nil
}

func writeSpace(file string, space Space) error {
	payload, err := json.MarshalIndent(space, "", "  ")
	if err != nil {
		return err
	}
	tmp := fmt.Sprintf("%s.%d.tmp", file, time.Now().UnixNano())
	if err := os.WriteFile(tmp, append(payload, '\n'), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, file)
}

func previewToken(token string) string {
	if len(token) <= 12 {
		return token
	}
	return token[:8] + "…" + token[len(token)-4:]
}
