package promptlibrary

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Store struct {
	mu  sync.Mutex
	dir string
}

type cachedLibrary struct {
	Library
	ETag string `json:"etag,omitempty"`
}

func NewStore(dir string) *Store {
	return &Store{dir: filepath.Clean(dir)}
}

func (s *Store) Load(lang string) (Library, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	file := s.path(lang)
	data, err := os.ReadFile(file)
	if err != nil {
		if os.IsNotExist(err) {
			return Library{}, false, nil
		}
		return Library{}, false, err
	}
	var cached cachedLibrary
	if err := json.Unmarshal(data, &cached); err != nil {
		return Library{}, false, fmt.Errorf("read prompt library cache: %w", err)
	}
	lib := cached.Library
	lib.ETag = cached.ETag
	return lib, true, nil
}

func (s *Store) Save(lang string, lib Library) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	file := s.path(lang)
	if err := os.MkdirAll(filepath.Dir(file), 0o700); err != nil {
		return err
	}
	payload, err := json.MarshalIndent(cachedLibrary{Library: lib, ETag: lib.ETag}, "", "  ")
	if err != nil {
		return err
	}
	tmp := fmt.Sprintf("%s.%d.tmp", file, time.Now().UnixNano())
	if err := os.WriteFile(tmp, append(payload, '\n'), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, file)
}

func (s *Store) path(lang string) string {
	name := strings.NewReplacer("/", "-", "\\", "-", "..", "-").Replace(strings.TrimSpace(lang))
	if name == "" {
		name = DefaultLang
	}
	return filepath.Join(s.dir, name+".json")
}
