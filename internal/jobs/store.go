package jobs

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/y08lin4/image-Workbench-Localhost-Version/internal/spaces"
)

type Store struct {
	mu     sync.Mutex
	spaces *spaces.FileStore
}

type persisted struct {
	Jobs []Job `json:"jobs"`
}

func NewStore(spaceStore *spaces.FileStore) *Store {
	return &Store{spaces: spaceStore}
}

func (s *Store) Save(job Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	jobs, err := s.loadLocked(job.SpaceToken)
	if err != nil {
		return err
	}
	found := false
	for i := range jobs {
		if jobs[i].ID == job.ID {
			jobs[i] = job
			found = true
			break
		}
	}
	if !found {
		jobs = append(jobs, job)
	}
	return s.saveLocked(job.SpaceToken, jobs)
}

func (s *Store) Update(spaceToken string, id string, mutate func(*Job)) (Job, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	jobs, err := s.loadLocked(spaceToken)
	if err != nil {
		return Job{}, false, err
	}
	for i := range jobs {
		if jobs[i].ID == id {
			mutate(&jobs[i])
			jobs[i].UpdatedAt = time.Now()
			if err := s.saveLocked(spaceToken, jobs); err != nil {
				return Job{}, false, err
			}
			return jobs[i], true, nil
		}
	}
	return Job{}, false, nil
}

func (s *Store) Delete(spaceToken string, id string) (Job, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	jobs, err := s.loadLocked(spaceToken)
	if err != nil {
		return Job{}, false, err
	}
	for i := range jobs {
		if jobs[i].ID == id {
			deleted := jobs[i]
			jobs = append(jobs[:i], jobs[i+1:]...)
			if err := s.saveLocked(spaceToken, jobs); err != nil {
				return Job{}, false, err
			}
			return deleted, true, nil
		}
	}
	return Job{}, false, nil
}

func (s *Store) Get(spaceToken string, id string) (Job, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	jobs, err := s.loadLocked(spaceToken)
	if err != nil {
		return Job{}, false, err
	}
	for _, job := range jobs {
		if job.ID == id {
			return job, true, nil
		}
	}
	return Job{}, false, nil
}

func (s *Store) List(spaceToken string, limit int) ([]Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	jobs, err := s.loadLocked(spaceToken)
	if err != nil {
		return nil, err
	}
	sort.Slice(jobs, func(i int, j int) bool { return jobs[i].CreatedAt.After(jobs[j].CreatedAt) })
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if len(jobs) > limit {
		jobs = jobs[:limit]
	}
	return jobs, nil
}

func (s *Store) AllSpacesJobs() (map[string][]Job, error) {
	tokens, err := s.spaces.ListTokens()
	if err != nil {
		return nil, err
	}
	out := make(map[string][]Job)
	for _, token := range tokens {
		jobs, err := s.List(token, 1000)
		if err != nil {
			return nil, err
		}
		out[token] = jobs
	}
	return out, nil
}

func (s *Store) Stats(spaceToken string) (Stats, error) {
	jobs, err := s.List(spaceToken, 1000)
	if err != nil {
		return Stats{}, err
	}
	var stats Stats
	stats.TotalTasks = len(jobs)
	for _, job := range jobs {
		switch job.Status {
		case StatusQueued, StatusRunning:
			stats.RunningTasks++
		case StatusSucceeded, StatusPartialFailed:
			stats.SucceededTasks++
		case StatusFailed, StatusInterrupted:
			stats.FailedTasks++
		}
		for _, result := range job.Results {
			if result.OK {
				stats.TotalImages++
			}
		}
	}
	return stats, nil
}

func (s *Store) loadLocked(spaceToken string) ([]Job, error) {
	file, err := s.jobsPath(spaceToken)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(file)
	if err != nil {
		if os.IsNotExist(err) {
			return []Job{}, nil
		}
		return nil, err
	}
	var payload persisted
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("读取任务状态失败：%w", err)
	}
	for i := range payload.Jobs {
		payload.Jobs[i].SpaceToken = spaceToken
	}
	return payload.Jobs, nil
}

func (s *Store) saveLocked(spaceToken string, jobs []Job) error {
	file, err := s.jobsPath(spaceToken)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(file), 0o700); err != nil {
		return err
	}
	payload, err := json.MarshalIndent(persisted{Jobs: jobs}, "", "  ")
	if err != nil {
		return err
	}
	tmp := fmt.Sprintf("%s.%d.tmp", file, time.Now().UnixNano())
	if err := os.WriteFile(tmp, append(payload, '\n'), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, file)
}

func (s *Store) jobsPath(spaceToken string) (string, error) {
	spaceDir, err := s.spaces.SpaceDir(spaceToken)
	if err != nil {
		return "", err
	}
	return filepath.Join(spaceDir, "jobs.json"), nil
}
