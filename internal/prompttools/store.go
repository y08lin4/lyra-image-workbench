package prompttools

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

type persisted struct {
	Records []Record `json:"records"`
}

type persistedSessions struct {
	Sessions []PromptSession `json:"sessions"`
}

func NewStore(spaceStore *spaces.FileStore) *Store {
	return &Store{spaces: spaceStore}
}

func (s *Store) Save(spaceToken string, record Record) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	records, err := s.loadLocked(spaceToken)
	if err != nil {
		return err
	}
	found := false
	for i := range records {
		if records[i].ID == record.ID {
			records[i] = record
			found = true
			break
		}
	}
	if !found {
		records = append(records, record)
	}
	sortRecords(records)
	if len(records) > 200 {
		records = records[:200]
	}
	return s.saveLocked(spaceToken, records)
}

func (s *Store) List(spaceToken string, limit int) ([]Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	records, err := s.loadLocked(spaceToken)
	if err != nil {
		return nil, err
	}
	sortRecords(records)
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if len(records) > limit {
		records = records[:limit]
	}
	return records, nil
}

func (s *Store) Delete(spaceToken string, id string) (Record, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	records, err := s.loadLocked(spaceToken)
	if err != nil {
		return Record{}, false, err
	}
	for i := range records {
		if records[i].ID == id {
			deleted := records[i]
			records = append(records[:i], records[i+1:]...)
			if err := s.saveLocked(spaceToken, records); err != nil {
				return Record{}, false, err
			}
			return deleted, true, nil
		}
	}
	return Record{}, false, nil
}

func (s *Store) SaveSession(spaceToken string, session PromptSession) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	sessions, err := s.loadSessionsLocked(spaceToken)
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
	if len(sessions) > 200 {
		sessions = sessions[:200]
	}
	return s.saveSessionsLocked(spaceToken, sessions)
}

func (s *Store) ListSessions(spaceToken string, limit int) ([]PromptSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sessions, err := s.loadSessionsLocked(spaceToken)
	if err != nil {
		return nil, err
	}
	sortSessions(sessions)
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if len(sessions) > limit {
		sessions = sessions[:limit]
	}
	return sessions, nil
}

func (s *Store) GetSession(spaceToken string, id string) (PromptSession, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sessions, err := s.loadSessionsLocked(spaceToken)
	if err != nil {
		return PromptSession{}, false, err
	}
	for _, session := range sessions {
		if session.ID == id {
			return session, true, nil
		}
	}
	return PromptSession{}, false, nil
}

func (s *Store) DeleteSession(spaceToken string, id string) (PromptSession, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sessions, err := s.loadSessionsLocked(spaceToken)
	if err != nil {
		return PromptSession{}, false, err
	}
	for i := range sessions {
		if sessions[i].ID == id {
			deleted := sessions[i]
			sessions = append(sessions[:i], sessions[i+1:]...)
			if err := s.saveSessionsLocked(spaceToken, sessions); err != nil {
				return PromptSession{}, false, err
			}
			return deleted, true, nil
		}
	}
	return PromptSession{}, false, nil
}

func (s *Store) loadLocked(spaceToken string) ([]Record, error) {
	file, err := s.path(spaceToken)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(file)
	if err != nil {
		if os.IsNotExist(err) {
			return []Record{}, nil
		}
		return nil, err
	}
	var payload persisted
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("读取提示词历史失败：%w", err)
	}
	return payload.Records, nil
}

func (s *Store) saveLocked(spaceToken string, records []Record) error {
	file, err := s.path(spaceToken)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(file), 0o700); err != nil {
		return err
	}
	payload, err := json.MarshalIndent(persisted{Records: records}, "", "  ")
	if err != nil {
		return err
	}
	tmp := fmt.Sprintf("%s.%d.tmp", file, time.Now().UnixNano())
	if err := os.WriteFile(tmp, append(payload, '\n'), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, file)
}

func (s *Store) loadSessionsLocked(spaceToken string) ([]PromptSession, error) {
	file, err := s.sessionsPath(spaceToken)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(file)
	if err != nil {
		if os.IsNotExist(err) {
			return []PromptSession{}, nil
		}
		return nil, err
	}
	var payload persistedSessions
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("读取提示词会话失败：%w", err)
	}
	return payload.Sessions, nil
}

func (s *Store) saveSessionsLocked(spaceToken string, sessions []PromptSession) error {
	file, err := s.sessionsPath(spaceToken)
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
	return filepath.Join(spaceDir, "prompt_records.json"), nil
}

func (s *Store) sessionsPath(spaceToken string) (string, error) {
	spaceDir, err := s.spaces.SpaceDir(spaceToken)
	if err != nil {
		return "", err
	}
	return filepath.Join(spaceDir, "prompt_sessions.json"), nil
}

func sortRecords(records []Record) {
	sort.Slice(records, func(i, j int) bool {
		return records[i].CreatedAt.After(records[j].CreatedAt)
	})
}

func sortSessions(sessions []PromptSession) {
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].UpdatedAt.After(sessions[j].UpdatedAt)
	})
}
