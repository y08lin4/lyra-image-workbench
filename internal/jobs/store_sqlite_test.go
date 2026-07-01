package jobs

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/y08lin4/lyra-image-workbench/internal/spaces"
)

func TestSQLiteStoreCRUD(t *testing.T) {
	root := t.TempDir()
	spaceStore, err := spaces.NewFileStore(filepath.Join(root, "data"))
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}
	session, err := spaceStore.CreateOrOpenByPassword("R7!Blue#Vault$2026")
	if err != nil {
		t.Fatalf("CreateOrOpenByPassword() error = %v", err)
	}
	db := newSQLiteStoreTestDB(t)
	store, err := NewStoreWithSQLiteDB(spaceStore, db)
	if err != nil {
		t.Fatalf("NewStoreWithSQLiteDB() error = %v", err)
	}

	older := newSQLiteStoreTestJob(session.Token, "img_old", time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC))
	newer := newSQLiteStoreTestJob(session.Token, "img_new", time.Date(2026, 6, 1, 11, 0, 0, 0, time.UTC))
	newer.Favorite = true
	newer.Results = []Result{{Index: 0, OK: true, Status: StatusSucceeded, OutputFileName: "img_new.png"}}
	if err := store.Save(older); err != nil {
		t.Fatalf("Save(older) error = %v", err)
	}
	if err := store.Save(newer); err != nil {
		t.Fatalf("Save(newer) error = %v", err)
	}

	got, ok, err := store.Get(session.Token, older.ID)
	if err != nil || !ok {
		t.Fatalf("Get(older) ok=%v err=%v", ok, err)
	}
	if got.ID != older.ID || got.SpaceToken != session.Token || got.Prompt != older.Prompt {
		t.Fatalf("Get(older) mismatch: %+v", got)
	}

	listed, err := store.List(session.Token, 10)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(listed) != 2 || listed[0].ID != newer.ID || listed[1].ID != older.ID {
		t.Fatalf("List() order = %+v", listed)
	}
	limited, err := store.List(session.Token, 1)
	if err != nil {
		t.Fatalf("List(limit) error = %v", err)
	}
	if len(limited) != 1 || limited[0].ID != newer.ID {
		t.Fatalf("List(limit) = %+v", limited)
	}

	updated, ok, err := store.Update(session.Token, older.ID, func(job *Job) {
		job.Progress = 100
		job.Favorite = true
		job.Results = []Result{{Index: 0, OK: true, Status: StatusSucceeded}}
		ApplyStatus(job, StatusSucceeded)
		ApplyStage(job, StageSucceeded)
	})
	if err != nil || !ok {
		t.Fatalf("Update(older) ok=%v err=%v", ok, err)
	}
	if updated.Status != StatusSucceeded || updated.Stage != StageSucceeded || !updated.Favorite || updated.Progress != 100 {
		t.Fatalf("Update(older) did not persist mutation: %+v", updated)
	}
	if !updated.UpdatedAt.After(older.UpdatedAt) {
		t.Fatalf("Update(older) did not refresh UpdatedAt: old=%s updated=%s", older.UpdatedAt, updated.UpdatedAt)
	}
	persisted, ok, err := store.Get(session.Token, older.ID)
	if err != nil || !ok {
		t.Fatalf("Get(updated) ok=%v err=%v", ok, err)
	}
	if persisted.Status != StatusSucceeded || len(persisted.Results) != 1 || !persisted.Results[0].OK {
		t.Fatalf("Get(updated) mismatch: %+v", persisted)
	}

	deleted, ok, err := store.Delete(session.Token, newer.ID)
	if err != nil || !ok {
		t.Fatalf("Delete(newer) ok=%v err=%v", ok, err)
	}
	if deleted.ID != newer.ID || !deleted.Favorite {
		t.Fatalf("Delete(newer) returned wrong job: %+v", deleted)
	}
	if _, ok, err := store.Get(session.Token, newer.ID); err != nil || ok {
		t.Fatalf("Get(deleted) ok=%v err=%v", ok, err)
	}
	remaining, err := store.List(session.Token, 10)
	if err != nil {
		t.Fatalf("List(remaining) error = %v", err)
	}
	if len(remaining) != 1 || remaining[0].ID != older.ID {
		t.Fatalf("List(remaining) = %+v", remaining)
	}

	if _, ok, err := store.Update(session.Token, "missing", func(*Job) {}); err != nil || ok {
		t.Fatalf("Update(missing) ok=%v err=%v", ok, err)
	}
	if _, ok, err := store.Delete(session.Token, "missing"); err != nil || ok {
		t.Fatalf("Delete(missing) ok=%v err=%v", ok, err)
	}
}

func newSQLiteStoreTestJob(spaceToken string, id string, createdAt time.Time) Job {
	job := Job{
		ID:           id,
		SpaceToken:   spaceToken,
		Provider:     "test-provider",
		Model:        "test-model",
		Mode:         ModeTextToImage,
		Source:       JobSourceAPI,
		Prompt:       "prompt for " + id,
		Ratio:        "1:1",
		Resolution:   "standard",
		OutputFormat: "png",
		Size:         "1024x1024",
		Count:        1,
		Concurrency:  1,
		Results:      []Result{},
		CreatedAt:    createdAt,
		UpdatedAt:    createdAt,
	}
	ApplyStatus(&job, StatusQueued)
	ApplyStage(&job, StageQueued)
	job.CreatedAt = createdAt
	job.UpdatedAt = createdAt
	return job
}

const sqliteStoreTestDriverName = "jobs-sqlite-store-test"

var (
	sqliteStoreTestDriverOnce sync.Once
	sqliteStoreTestDatabases  sync.Map
)

func newSQLiteStoreTestDB(t *testing.T) *sql.DB {
	t.Helper()
	sqliteStoreTestDriverOnce.Do(func() {
		sql.Register(sqliteStoreTestDriverName, sqliteStoreTestDriver{})
	})
	name := strings.ReplaceAll(t.Name(), "/", "_")
	sqliteStoreTestDatabases.Store(name, &sqliteStoreTestState{rows: make(map[sqliteStoreTestKey]sqliteStoreTestRow)})
	t.Cleanup(func() {
		sqliteStoreTestDatabases.Delete(name)
	})
	db, err := sql.Open(sqliteStoreTestDriverName, name)
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() {
		_ = db.Close()
	})
	return db
}

type sqliteStoreTestDriver struct{}

func (sqliteStoreTestDriver) Open(name string) (driver.Conn, error) {
	value, ok := sqliteStoreTestDatabases.Load(name)
	if !ok {
		return nil, fmt.Errorf("unknown sqlite store test database %q", name)
	}
	return &sqliteStoreTestConn{state: value.(*sqliteStoreTestState)}, nil
}

type sqliteStoreTestState struct {
	mu   sync.Mutex
	rows map[sqliteStoreTestKey]sqliteStoreTestRow
}

type sqliteStoreTestKey struct {
	spaceToken string
	id         string
}

type sqliteStoreTestRow struct {
	spaceToken      string
	id              string
	payload         string
	createdUnixNano int64
}

type sqliteStoreTestConn struct {
	state *sqliteStoreTestState
}

func (c *sqliteStoreTestConn) Prepare(query string) (driver.Stmt, error) {
	return nil, fmt.Errorf("Prepare(%q) is not implemented in sqlite store test driver", query)
}

func (c *sqliteStoreTestConn) Close() error {
	return nil
}

func (c *sqliteStoreTestConn) Begin() (driver.Tx, error) {
	return &sqliteStoreTestTx{}, nil
}

func (c *sqliteStoreTestConn) BeginTx(context.Context, driver.TxOptions) (driver.Tx, error) {
	return &sqliteStoreTestTx{}, nil
}

func (c *sqliteStoreTestConn) ExecContext(_ context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	sqlText := normalizeSQLiteStoreTestSQL(query)
	switch {
	case strings.HasPrefix(sqlText, "pragma ") || strings.HasPrefix(sqlText, "create table ") || strings.HasPrefix(sqlText, "create index "):
		return driver.RowsAffected(0), nil
	case strings.HasPrefix(sqlText, "insert into jobs "):
		if len(args) != 12 {
			return nil, fmt.Errorf("insert args = %d, want 12", len(args))
		}
		spaceToken, err := sqliteStoreTestStringArg(args, 0)
		if err != nil {
			return nil, err
		}
		id, err := sqliteStoreTestStringArg(args, 1)
		if err != nil {
			return nil, err
		}
		payload, err := sqliteStoreTestStringArg(args, 2)
		if err != nil {
			return nil, err
		}
		createdUnixNano, err := sqliteStoreTestInt64Arg(args, 8)
		if err != nil {
			return nil, err
		}
		c.state.mu.Lock()
		c.state.rows[sqliteStoreTestKey{spaceToken: spaceToken, id: id}] = sqliteStoreTestRow{
			spaceToken:      spaceToken,
			id:              id,
			payload:         payload,
			createdUnixNano: createdUnixNano,
		}
		c.state.mu.Unlock()
		return driver.RowsAffected(1), nil
	case strings.HasPrefix(sqlText, "delete from jobs where space_token = ? and id = ?"):
		spaceToken, err := sqliteStoreTestStringArg(args, 0)
		if err != nil {
			return nil, err
		}
		id, err := sqliteStoreTestStringArg(args, 1)
		if err != nil {
			return nil, err
		}
		c.state.mu.Lock()
		_, existed := c.state.rows[sqliteStoreTestKey{spaceToken: spaceToken, id: id}]
		delete(c.state.rows, sqliteStoreTestKey{spaceToken: spaceToken, id: id})
		c.state.mu.Unlock()
		if existed {
			return driver.RowsAffected(1), nil
		}
		return driver.RowsAffected(0), nil
	default:
		return nil, fmt.Errorf("unexpected Exec query: %s", query)
	}
}

func (c *sqliteStoreTestConn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	sqlText := normalizeSQLiteStoreTestSQL(query)
	switch {
	case strings.HasPrefix(sqlText, "select payload_json from jobs where space_token = ? and id = ?"):
		spaceToken, err := sqliteStoreTestStringArg(args, 0)
		if err != nil {
			return nil, err
		}
		id, err := sqliteStoreTestStringArg(args, 1)
		if err != nil {
			return nil, err
		}
		c.state.mu.Lock()
		row, ok := c.state.rows[sqliteStoreTestKey{spaceToken: spaceToken, id: id}]
		c.state.mu.Unlock()
		if !ok {
			return newSQLiteStoreTestRows([]string{"payload_json"}, nil), nil
		}
		return newSQLiteStoreTestRows([]string{"payload_json"}, [][]driver.Value{{row.payload}}), nil
	case strings.HasPrefix(sqlText, "select payload_json from jobs where space_token = ? order by "):
		spaceToken, err := sqliteStoreTestStringArg(args, 0)
		if err != nil {
			return nil, err
		}
		limit := 0
		if len(args) > 1 {
			parsed, err := sqliteStoreTestInt64Arg(args, 1)
			if err != nil {
				return nil, err
			}
			limit = int(parsed)
		}
		rows := c.sortedRows(func(row sqliteStoreTestRow) bool {
			return row.spaceToken == spaceToken
		})
		if limit > 0 && len(rows) > limit {
			rows = rows[:limit]
		}
		values := make([][]driver.Value, 0, len(rows))
		for _, row := range rows {
			values = append(values, []driver.Value{row.payload})
		}
		return newSQLiteStoreTestRows([]string{"payload_json"}, values), nil
	case strings.HasPrefix(sqlText, "select space_token, payload_json from jobs order by "):
		rows := c.sortedRows(func(sqliteStoreTestRow) bool {
			return true
		})
		values := make([][]driver.Value, 0, len(rows))
		for _, row := range rows {
			values = append(values, []driver.Value{row.spaceToken, row.payload})
		}
		return newSQLiteStoreTestRows([]string{"space_token", "payload_json"}, values), nil
	default:
		return nil, fmt.Errorf("unexpected Query query: %s", query)
	}
}

func (c *sqliteStoreTestConn) sortedRows(include func(sqliteStoreTestRow) bool) []sqliteStoreTestRow {
	c.state.mu.Lock()
	defer c.state.mu.Unlock()
	rows := make([]sqliteStoreTestRow, 0, len(c.state.rows))
	for _, row := range c.state.rows {
		if include(row) {
			rows = append(rows, row)
		}
	}
	sort.Slice(rows, func(i int, j int) bool {
		if rows[i].spaceToken != rows[j].spaceToken {
			return rows[i].spaceToken < rows[j].spaceToken
		}
		if rows[i].createdUnixNano != rows[j].createdUnixNano {
			return rows[i].createdUnixNano > rows[j].createdUnixNano
		}
		return rows[i].id > rows[j].id
	})
	return rows
}

type sqliteStoreTestTx struct{}

func (sqliteStoreTestTx) Commit() error {
	return nil
}

func (sqliteStoreTestTx) Rollback() error {
	return nil
}

type sqliteStoreTestRows struct {
	columns []string
	values  [][]driver.Value
	index   int
}

func newSQLiteStoreTestRows(columns []string, values [][]driver.Value) *sqliteStoreTestRows {
	if values == nil {
		values = [][]driver.Value{}
	}
	return &sqliteStoreTestRows{columns: columns, values: values}
}

func (r *sqliteStoreTestRows) Columns() []string {
	return r.columns
}

func (r *sqliteStoreTestRows) Close() error {
	return nil
}

func (r *sqliteStoreTestRows) Next(dest []driver.Value) error {
	if r.index >= len(r.values) {
		return io.EOF
	}
	copy(dest, r.values[r.index])
	r.index++
	return nil
}

func normalizeSQLiteStoreTestSQL(query string) string {
	return strings.ToLower(strings.Join(strings.Fields(query), " "))
}

func sqliteStoreTestStringArg(args []driver.NamedValue, index int) (string, error) {
	if len(args) <= index {
		return "", fmt.Errorf("missing arg %d", index)
	}
	value, ok := args[index].Value.(string)
	if !ok {
		return "", fmt.Errorf("arg %d = %T, want string", index, args[index].Value)
	}
	return value, nil
}

func sqliteStoreTestInt64Arg(args []driver.NamedValue, index int) (int64, error) {
	if len(args) <= index {
		return 0, fmt.Errorf("missing arg %d", index)
	}
	switch value := args[index].Value.(type) {
	case int64:
		return value, nil
	case int:
		return int64(value), nil
	default:
		return 0, fmt.Errorf("arg %d = %T, want int64", index, args[index].Value)
	}
}
