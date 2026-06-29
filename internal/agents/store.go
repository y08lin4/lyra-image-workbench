package agents

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/y08lin4/lyra-image-workbench/internal/spaces"
)

type Store struct {
	mu     sync.Mutex
	spaces *spaces.FileStore
}

type persistedSessions struct {
	Sessions []Session `json:"sessions"`
}

func NewStore(spaceStore *spaces.FileStore) *Store {
	return &Store{spaces: spaceStore}
}

func (s *Store) Save(spaceToken string, session Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	sessions, err := s.loadLocked(spaceToken)
	if err != nil {
		return err
	}
	found := false
	for i := range sessions {
		if sessions[i].ID == session.ID {
			sessions[i] = session
			found = true
			break
		}
	}
	if !found {
		sessions = append(sessions, session)
	}
	sortSessions(sessions)
	if len(sessions) > 120 {
		sessions = sessions[:120]
	}
	return s.saveLocked(spaceToken, sessions)
}

func (s *Store) List(spaceToken string, limit int) ([]Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sessions, err := s.loadLocked(spaceToken)
	if err != nil {
		return nil, err
	}
	sortSessions(sessions)
	if limit <= 0 || limit > 120 {
		limit = 30
	}
	if len(sessions) > limit {
		sessions = sessions[:limit]
	}
	return sessions, nil
}

func (s *Store) Get(spaceToken string, id string) (Session, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sessions, err := s.loadLocked(spaceToken)
	if err != nil {
		return Session{}, false, err
	}
	for _, session := range sessions {
		if session.ID == id {
			return session, true, nil
		}
	}
	return Session{}, false, nil
}

func (s *Store) Delete(spaceToken string, id string) (Session, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sessions, err := s.loadLocked(spaceToken)
	if err != nil {
		return Session{}, false, err
	}
	for i := range sessions {
		if sessions[i].ID == id {
			deleted := sessions[i]
			sessions = append(sessions[:i], sessions[i+1:]...)
			if err := s.saveLocked(spaceToken, sessions); err != nil {
				return Session{}, false, err
			}
			return deleted, true, nil
		}
	}
	return Session{}, false, nil
}

func (s *Store) loadLocked(spaceToken string) ([]Session, error) {
	file, err := s.path(spaceToken)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(file)
	if err != nil {
		if os.IsNotExist(err) {
			return []Session{}, nil
		}
		return nil, err
	}
	var payload persistedSessions
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("读取 Agent 会话失败：%w", err)
	}
	return payload.Sessions, nil
}

func (s *Store) saveLocked(spaceToken string, sessions []Session) error {
	file, err := s.path(spaceToken)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(file), 0o700); err != nil {
		return err
	}
	payload, err := json.MarshalIndent(persistedSessions{Sessions: sessions}, "", "  ")
	if err != nil {
		return err
	}
	tmp := fmt.Sprintf("%s.%d.tmp", file, time.Now().UnixNano())
	if err := os.WriteFile(tmp, append(payload, '\n'), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, file)
}

func (s *Store) path(spaceToken string) (string, error) {
	spaceDir, err := s.spaces.SpaceDir(spaceToken)
	if err != nil {
		return "", err
	}
	return filepath.Join(spaceDir, "agent_sessions.json"), nil
}

func sortSessions(sessions []Session) {
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].UpdatedAt.After(sessions[j].UpdatedAt)
	})
}
