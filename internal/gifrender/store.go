package gifrender

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/y08lin4/lyra-image-workbench/internal/spaces"
)

type RenderStatus string

const (
	RenderStatusSucceeded RenderStatus = "succeeded"
	RenderStatusFailed    RenderStatus = "failed"
)

type Render struct {
	ID             string       `json:"id"`
	SpaceToken     string       `json:"-"`
	SourceTaskID   string       `json:"sourceTaskId"`
	Status         RenderStatus `json:"status"`
	FPS            int          `json:"fps"`
	FrameIndexes   []int        `json:"frameIndexes"`
	Loop           bool         `json:"loop"`
	Width          int          `json:"width"`
	GifURL         string       `json:"gifUrl,omitempty"`
	OutputDate     string       `json:"outputDate,omitempty"`
	OutputFileName string       `json:"outputFileName,omitempty"`
	Bytes          int64        `json:"bytes,omitempty"`
	Error          string       `json:"error,omitempty"`
	CreatedAt      time.Time    `json:"createdAt"`
	UpdatedAt      time.Time    `json:"updatedAt"`
}

type Store struct {
	mu     sync.Mutex
	spaces *spaces.FileStore
}

type persistedRenders struct {
	Renders []Render `json:"renders"`
}

func NewStore(spaceStore *spaces.FileStore) *Store {
	return &Store{spaces: spaceStore}
}

func (s *Store) Save(render Render) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	renders, err := s.loadLocked(render.SpaceToken)
	if err != nil {
		return err
	}
	found := false
	for i := range renders {
		if renders[i].ID == render.ID {
			renders[i] = render
			found = true
			break
		}
	}
	if !found {
		renders = append(renders, render)
	}
	return s.saveLocked(render.SpaceToken, renders)
}

func (s *Store) Get(spaceToken string, id string) (Render, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	renders, err := s.loadLocked(spaceToken)
	if err != nil {
		return Render{}, false, err
	}
	for _, render := range renders {
		if render.ID == id {
			return render, true, nil
		}
	}
	return Render{}, false, nil
}

func (s *Store) List(spaceToken string, limit int) ([]Render, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	renders, err := s.loadLocked(spaceToken)
	if err != nil {
		return nil, err
	}
	sort.Slice(renders, func(i, j int) bool { return renders[i].CreatedAt.After(renders[j].CreatedAt) })
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if len(renders) > limit {
		renders = renders[:limit]
	}
	return renders, nil
}

func (s *Store) loadLocked(spaceToken string) ([]Render, error) {
	file, err := s.rendersPath(spaceToken)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(file)
	if err != nil {
		if os.IsNotExist(err) {
			return []Render{}, nil
		}
		return nil, err
	}
	var payload persistedRenders
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("读取 GIF 渲染记录失败：%w", err)
	}
	for i := range payload.Renders {
		payload.Renders[i].SpaceToken = spaceToken
	}
	return payload.Renders, nil
}

func (s *Store) saveLocked(spaceToken string, renders []Render) error {
	file, err := s.rendersPath(spaceToken)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(file), 0o700); err != nil {
		return err
	}
	payload, err := json.MarshalIndent(persistedRenders{Renders: renders}, "", "  ")
	if err != nil {
		return err
	}
	tmp := fmt.Sprintf("%s.%d.tmp", file, time.Now().UnixNano())
	if err := os.WriteFile(tmp, append(payload, '\n'), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, file)
}

func (s *Store) rendersPath(spaceToken string) (string, error) {
	spaceDir, err := s.spaces.SpaceDir(spaceToken)
	if err != nil {
		return "", err
	}
	return filepath.Join(spaceDir, "gif-renders.json"), nil
}

func NewRenderID() (string, error) {
	var bytes [8]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", err
	}
	return "gifrender_" + time.Now().Format("20060102150405") + "_" + hex.EncodeToString(bytes[:]), nil
}

func PublicRender(render Render) Render {
	render.SpaceToken = ""
	render.OutputDate = ""
	render.OutputFileName = ""
	if render.ID != "" {
		render.GifURL = "/api/gif-renders/" + render.ID + "/file"
	}
	return render
}
