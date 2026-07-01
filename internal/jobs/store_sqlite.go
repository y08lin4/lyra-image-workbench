package jobs

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/y08lin4/lyra-image-workbench/internal/spaces"
)

type SQLiteStore struct {
	mu     sync.Mutex
	db     *sql.DB
	spaces *spaces.FileStore
}

var sqliteJobStoreSchema = []string{
	`CREATE TABLE IF NOT EXISTS jobs (
		space_token TEXT NOT NULL,
		id TEXT NOT NULL,
		payload_json TEXT NOT NULL,
		status TEXT NOT NULL,
		stage TEXT NOT NULL,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL,
		finished_at TEXT,
		created_unix_nano INTEGER NOT NULL,
		updated_unix_nano INTEGER NOT NULL,
		finished_unix_nano INTEGER,
		favorite INTEGER NOT NULL DEFAULT 0,
		PRIMARY KEY (space_token, id)
	)`,
	`CREATE INDEX IF NOT EXISTS idx_jobs_space_created ON jobs (space_token, created_unix_nano DESC)`,
	`CREATE INDEX IF NOT EXISTS idx_jobs_space_status ON jobs (space_token, status)`,
	`CREATE INDEX IF NOT EXISTS idx_jobs_updated ON jobs (updated_unix_nano DESC)`,
}

func SQLiteJobStoreSchema() []string {
	return append([]string{}, sqliteJobStoreSchema...)
}

func NewStoreWithSQLiteDB(spaceStore *spaces.FileStore, db *sql.DB) (*Store, error) {
	sqliteStore, err := NewSQLiteStore(spaceStore, db)
	if err != nil {
		return nil, err
	}
	return &Store{spaces: spaceStore, sqlite: sqliteStore}, nil
}

func NewSQLiteStore(spaceStore *spaces.FileStore, db *sql.DB) (*SQLiteStore, error) {
	if db == nil {
		return nil, errors.New("jobs sqlite store requires a non-nil *sql.DB")
	}
	store := &SQLiteStore{db: db, spaces: spaceStore}
	if err := store.migrate(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *SQLiteStore) Save(job Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	if err := s.saveTx(tx, job); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	committed = true
	return nil
}

func (s *SQLiteStore) Update(spaceToken string, id string, mutate func(*Job)) (Job, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	token, err := normalizeSQLiteSpaceToken(spaceToken)
	if err != nil {
		return Job{}, false, err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return Job{}, false, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	job, ok, err := s.getTx(tx, token, id)
	if err != nil || !ok {
		return job, ok, err
	}
	mutate(&job)
	job.UpdatedAt = time.Now()
	if err := s.saveTx(tx, job); err != nil {
		return Job{}, false, err
	}
	if err := tx.Commit(); err != nil {
		return Job{}, false, err
	}
	committed = true
	return job, true, nil
}

func (s *SQLiteStore) Delete(spaceToken string, id string) (Job, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	token, err := normalizeSQLiteSpaceToken(spaceToken)
	if err != nil {
		return Job{}, false, err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return Job{}, false, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	deleted, ok, err := s.getTx(tx, token, id)
	if err != nil || !ok {
		return deleted, ok, err
	}
	if _, err := tx.Exec(`DELETE FROM jobs WHERE space_token = ? AND id = ?`, token, id); err != nil {
		return Job{}, false, err
	}
	if err := tx.Commit(); err != nil {
		return Job{}, false, err
	}
	committed = true
	return deleted, true, nil
}

func (s *SQLiteStore) Get(spaceToken string, id string) (Job, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	token, err := normalizeSQLiteSpaceToken(spaceToken)
	if err != nil {
		return Job{}, false, err
	}
	return s.getDB(token, id)
}

func (s *SQLiteStore) List(spaceToken string, limit int) ([]Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	token, err := normalizeSQLiteSpaceToken(spaceToken)
	if err != nil {
		return nil, err
	}
	return s.list(token, normalizeSQLiteJobLimit(limit))
}

func (s *SQLiteStore) AllSpacesJobs() (map[string][]Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string][]Job)
	if s.spaces != nil {
		tokens, err := s.spaces.ListTokens()
		if err != nil {
			return nil, err
		}
		for _, token := range tokens {
			out[token] = []Job{}
		}
	}
	rows, err := s.db.Query(`SELECT space_token, payload_json FROM jobs ORDER BY space_token ASC, created_unix_nano DESC, id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var token string
		var payload string
		if err := rows.Scan(&token, &payload); err != nil {
			return nil, err
		}
		job, err := decodeSQLiteJob(token, payload)
		if err != nil {
			return nil, err
		}
		out[token] = append(out[token], job)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *SQLiteStore) Stats(spaceToken string) (Stats, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	token, err := normalizeSQLiteSpaceToken(spaceToken)
	if err != nil {
		return Stats{}, err
	}
	jobs, err := s.list(token, 0)
	if err != nil {
		return Stats{}, err
	}
	return statsFromJobs(jobs), nil
}

func (s *SQLiteStore) migrate() error {
	if _, err := s.db.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		return err
	}
	if _, err := s.db.Exec(`PRAGMA busy_timeout = 5000`); err != nil {
		return err
	}
	for _, stmt := range sqliteJobStoreSchema {
		if _, err := s.db.Exec(stmt); err != nil {
			return fmt.Errorf("初始化任务 SQLite 表失败：%w", err)
		}
	}
	return nil
}

func (s *SQLiteStore) getDB(spaceToken string, id string) (Job, bool, error) {
	var payload string
	err := s.db.QueryRow(`SELECT payload_json FROM jobs WHERE space_token = ? AND id = ?`, spaceToken, id).Scan(&payload)
	if errors.Is(err, sql.ErrNoRows) {
		return Job{}, false, nil
	}
	if err != nil {
		return Job{}, false, err
	}
	job, err := decodeSQLiteJob(spaceToken, payload)
	if err != nil {
		return Job{}, false, err
	}
	return job, true, nil
}

func (s *SQLiteStore) getTx(tx *sql.Tx, spaceToken string, id string) (Job, bool, error) {
	var payload string
	err := tx.QueryRow(`SELECT payload_json FROM jobs WHERE space_token = ? AND id = ?`, spaceToken, id).Scan(&payload)
	if errors.Is(err, sql.ErrNoRows) {
		return Job{}, false, nil
	}
	if err != nil {
		return Job{}, false, err
	}
	job, err := decodeSQLiteJob(spaceToken, payload)
	if err != nil {
		return Job{}, false, err
	}
	return job, true, nil
}

func (s *SQLiteStore) list(spaceToken string, limit int) ([]Job, error) {
	query := `SELECT payload_json FROM jobs WHERE space_token = ? ORDER BY created_unix_nano DESC, id DESC`
	args := []any{spaceToken}
	if limit > 0 {
		query += ` LIMIT ?`
		args = append(args, limit)
	}
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	jobs := []Job{}
	for rows.Next() {
		var payload string
		if err := rows.Scan(&payload); err != nil {
			return nil, err
		}
		job, err := decodeSQLiteJob(spaceToken, payload)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return jobs, nil
}

func (s *SQLiteStore) saveTx(tx *sql.Tx, job Job) error {
	spaceToken, err := normalizeSQLiteSpaceToken(job.SpaceToken)
	if err != nil {
		return err
	}
	job.SpaceToken = spaceToken
	payload, err := json.Marshal(job)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`INSERT INTO jobs (
		space_token,
		id,
		payload_json,
		status,
		stage,
		created_at,
		updated_at,
		finished_at,
		created_unix_nano,
		updated_unix_nano,
		finished_unix_nano,
		favorite
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(space_token, id) DO UPDATE SET
		payload_json = excluded.payload_json,
		status = excluded.status,
		stage = excluded.stage,
		created_at = excluded.created_at,
		updated_at = excluded.updated_at,
		finished_at = excluded.finished_at,
		created_unix_nano = excluded.created_unix_nano,
		updated_unix_nano = excluded.updated_unix_nano,
		finished_unix_nano = excluded.finished_unix_nano,
		favorite = excluded.favorite`,
		spaceToken,
		job.ID,
		string(payload),
		string(job.Status),
		string(job.Stage),
		formatSQLiteTime(job.CreatedAt),
		formatSQLiteTime(job.UpdatedAt),
		nullableSQLiteTime(job.FinishedAt),
		unixNanoOrZero(job.CreatedAt),
		unixNanoOrZero(job.UpdatedAt),
		nullableUnixNano(job.FinishedAt),
		boolToSQLiteInt(job.Favorite),
	)
	if err != nil {
		return fmt.Errorf("保存任务状态失败：%w", err)
	}
	return nil
}

func normalizeSQLiteSpaceToken(spaceToken string) (string, error) {
	return spaces.NormalizeToken(spaceToken)
}

func normalizeSQLiteJobLimit(limit int) int {
	if limit <= 0 || limit > 200 {
		return 50
	}
	return limit
}

func decodeSQLiteJob(spaceToken string, payload string) (Job, error) {
	var job Job
	if err := json.Unmarshal([]byte(payload), &job); err != nil {
		return Job{}, fmt.Errorf("读取任务状态失败：%w", err)
	}
	job.SpaceToken = spaceToken
	return job, nil
}

func statsFromJobs(jobs []Job) Stats {
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
	return stats
}

func formatSQLiteTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}

func nullableSQLiteTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return formatSQLiteTime(t)
}

func unixNanoOrZero(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.UnixNano()
}

func nullableUnixNano(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t.UnixNano()
}

func boolToSQLiteInt(value bool) int {
	if value {
		return 1
	}
	return 0
}