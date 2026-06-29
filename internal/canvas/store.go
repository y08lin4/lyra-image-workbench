package canvas

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/y08lin4/lyra-image-workbench/internal/spaces"
)

type Store interface {
	Create(spaceToken string, project Project) error
	Get(spaceToken string, id string) (Project, bool, error)
	List(spaceToken string, limit int) ([]Project, error)
	Update(spaceToken string, id string, expectedRevision int64, mutate func(*Project) error) (Project, bool, error)
	Delete(spaceToken string, id string) (Project, bool, error)
}

type MemoryStore struct {
	mu       sync.Mutex
	projects map[string][]Project
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{projects: map[string][]Project{}}
}

func (s *MemoryStore) Create(spaceToken string, project Project) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.projects == nil {
		s.projects = map[string][]Project{}
	}
	spaceToken = strings.TrimSpace(spaceToken)
	project.SpaceToken = spaceToken
	for _, existing := range s.projects[spaceToken] {
		if existing.ID == project.ID {
			return fmt.Errorf("%w: duplicate project id %s", ErrInvalidProject, project.ID)
		}
	}
	cloned, err := cloneProject(project)
	if err != nil {
		return err
	}
	s.projects[spaceToken] = append(s.projects[spaceToken], cloned)
	return nil
}

func (s *MemoryStore) Get(spaceToken string, id string) (Project, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	spaceToken = strings.TrimSpace(spaceToken)
	id = strings.TrimSpace(id)
	for _, project := range s.projects[spaceToken] {
		if project.ID == id {
			cloned, err := cloneProject(project)
			return cloned, err == nil, err
		}
	}
	return Project{}, false, nil
}

func (s *MemoryStore) List(spaceToken string, limit int) ([]Project, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	spaceToken = strings.TrimSpace(spaceToken)
	projects, err := cloneProjects(s.projects[spaceToken])
	if err != nil {
		return nil, err
	}
	sortProjects(projects)
	limit = normalizedProjectLimit(limit)
	if len(projects) > limit {
		projects = projects[:limit]
	}
	return projects, nil
}

func (s *MemoryStore) Update(spaceToken string, id string, expectedRevision int64, mutate func(*Project) error) (Project, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	spaceToken = strings.TrimSpace(spaceToken)
	id = strings.TrimSpace(id)
	projects := s.projects[spaceToken]
	for i := range projects {
		if projects[i].ID != id {
			continue
		}
		current := projects[i]
		if expectedRevision > 0 && current.Revision != expectedRevision {
			cloned, cloneErr := cloneProject(current)
			if cloneErr != nil {
				return Project{}, true, cloneErr
			}
			return cloned, true, RevisionConflictError{ProjectID: id, Expected: expectedRevision, Actual: current.Revision}
		}
		next, err := cloneProject(current)
		if err != nil {
			return Project{}, true, err
		}
		if mutate != nil {
			if err := mutate(&next); err != nil {
				return Project{}, true, err
			}
		}
		next.ID = id
		next.SpaceToken = spaceToken
		projects[i] = next
		s.projects[spaceToken] = projects
		cloned, err := cloneProject(next)
		return cloned, err == nil, err
	}
	return Project{}, false, nil
}

func (s *MemoryStore) Delete(spaceToken string, id string) (Project, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	spaceToken = strings.TrimSpace(spaceToken)
	id = strings.TrimSpace(id)
	projects := s.projects[spaceToken]
	for i := range projects {
		if projects[i].ID == id {
			deleted := projects[i]
			projects = append(projects[:i], projects[i+1:]...)
			s.projects[spaceToken] = projects
			cloned, err := cloneProject(deleted)
			return cloned, err == nil, err
		}
	}
	return Project{}, false, nil
}

type FileStore struct {
	mu     sync.Mutex
	spaces *spaces.FileStore
}

type persistedProjects struct {
	Projects []Project `json:"projects"`
}

func NewFileStore(spaceStore *spaces.FileStore) *FileStore {
	return &FileStore{spaces: spaceStore}
}

func (s *FileStore) Create(spaceToken string, project Project) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	projects, err := s.loadLocked(spaceToken)
	if err != nil {
		return err
	}
	for _, existing := range projects {
		if existing.ID == project.ID {
			return fmt.Errorf("%w: duplicate project id %s", ErrInvalidProject, project.ID)
		}
	}
	project.SpaceToken = strings.TrimSpace(spaceToken)
	cloned, err := cloneProject(project)
	if err != nil {
		return err
	}
	projects = append(projects, cloned)
	return s.saveLocked(spaceToken, projects)
}

func (s *FileStore) Get(spaceToken string, id string) (Project, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	projects, err := s.loadLocked(spaceToken)
	if err != nil {
		return Project{}, false, err
	}
	id = strings.TrimSpace(id)
	for _, project := range projects {
		if project.ID == id {
			cloned, err := cloneProject(project)
			return cloned, err == nil, err
		}
	}
	return Project{}, false, nil
}

func (s *FileStore) List(spaceToken string, limit int) ([]Project, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	projects, err := s.loadLocked(spaceToken)
	if err != nil {
		return nil, err
	}
	projects, err = cloneProjects(projects)
	if err != nil {
		return nil, err
	}
	sortProjects(projects)
	limit = normalizedProjectLimit(limit)
	if len(projects) > limit {
		projects = projects[:limit]
	}
	return projects, nil
}

func (s *FileStore) Update(spaceToken string, id string, expectedRevision int64, mutate func(*Project) error) (Project, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	projects, err := s.loadLocked(spaceToken)
	if err != nil {
		return Project{}, false, err
	}
	id = strings.TrimSpace(id)
	for i := range projects {
		if projects[i].ID != id {
			continue
		}
		current := projects[i]
		if expectedRevision > 0 && current.Revision != expectedRevision {
			cloned, cloneErr := cloneProject(current)
			if cloneErr != nil {
				return Project{}, true, cloneErr
			}
			return cloned, true, RevisionConflictError{ProjectID: id, Expected: expectedRevision, Actual: current.Revision}
		}
		next, err := cloneProject(current)
		if err != nil {
			return Project{}, true, err
		}
		if mutate != nil {
			if err := mutate(&next); err != nil {
				return Project{}, true, err
			}
		}
		next.ID = id
		next.SpaceToken = strings.TrimSpace(spaceToken)
		projects[i] = next
		if err := s.saveLocked(spaceToken, projects); err != nil {
			return Project{}, true, err
		}
		cloned, err := cloneProject(next)
		return cloned, err == nil, err
	}
	return Project{}, false, nil
}

func (s *FileStore) Delete(spaceToken string, id string) (Project, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	projects, err := s.loadLocked(spaceToken)
	if err != nil {
		return Project{}, false, err
	}
	id = strings.TrimSpace(id)
	for i := range projects {
		if projects[i].ID == id {
			deleted := projects[i]
			projects = append(projects[:i], projects[i+1:]...)
			if err := s.saveLocked(spaceToken, projects); err != nil {
				return Project{}, true, err
			}
			cloned, err := cloneProject(deleted)
			return cloned, err == nil, err
		}
	}
	return Project{}, false, nil
}

func (s *FileStore) loadLocked(spaceToken string) ([]Project, error) {
	file, err := s.path(spaceToken)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(file)
	if err != nil {
		if os.IsNotExist(err) {
			return []Project{}, nil
		}
		return nil, err
	}
	var payload persistedProjects
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("read canvas projects: %w", err)
	}
	spaceToken = strings.TrimSpace(spaceToken)
	for i := range payload.Projects {
		payload.Projects[i].SpaceToken = spaceToken
	}
	return payload.Projects, nil
}

func (s *FileStore) saveLocked(spaceToken string, projects []Project) error {
	file, err := s.path(spaceToken)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(file), 0o700); err != nil {
		return err
	}
	payload, err := json.MarshalIndent(persistedProjects{Projects: projects}, "", "  ")
	if err != nil {
		return err
	}
	tmp := fmt.Sprintf("%s.%d.tmp", file, time.Now().UnixNano())
	if err := os.WriteFile(tmp, append(payload, '\n'), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, file)
}

func (s *FileStore) path(spaceToken string) (string, error) {
	if s == nil || s.spaces == nil {
		return "", ErrStoreNotConfigured
	}
	spaceDir, err := s.spaces.SpaceDir(spaceToken)
	if err != nil {
		return "", err
	}
	return filepath.Join(spaceDir, "canvas_projects.json"), nil
}

func cloneProject(project Project) (Project, error) {
	data, err := json.Marshal(project)
	if err != nil {
		return Project{}, err
	}
	var cloned Project
	if err := json.Unmarshal(data, &cloned); err != nil {
		return Project{}, err
	}
	return cloned, nil
}

func cloneProjects(projects []Project) ([]Project, error) {
	out := make([]Project, 0, len(projects))
	for _, project := range projects {
		cloned, err := cloneProject(project)
		if err != nil {
			return nil, err
		}
		out = append(out, cloned)
	}
	return out, nil
}

func sortProjects(projects []Project) {
	sort.Slice(projects, func(i int, j int) bool {
		return projects[i].UpdatedAt.After(projects[j].UpdatedAt)
	})
}

func normalizedProjectLimit(limit int) int {
	if limit <= 0 || limit > 200 {
		return 50
	}
	return limit
}
