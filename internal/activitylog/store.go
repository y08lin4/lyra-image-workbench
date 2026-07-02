package activitylog

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	DefaultLimit      = 100
	MaxLimit          = 500
	DefaultMaxEntries = 5000
)

type Level string

const (
	LevelInfo    Level = "info"
	LevelWarning Level = "warning"
	LevelError   Level = "error"
)

type Type string

const (
	TypeUserRegistration  Type = "user_registration"
	TypeTopUpOrderCreated Type = "topup_order_created"
	TypeTopUpPaidSuccess  Type = "topup_paid_success"
	TypeAdminCreditGrant  Type = "admin_credit_grant"
	TypeAdminUserStatus   Type = "admin_user_status"
	TypeTaskFailed        Type = "task_failed"
	TypeResultFailed      Type = "result_failed"
	TypeSystem            Type = "system"
)

type Entry struct {
	ID           string         `json:"id"`
	CreatedAt    time.Time      `json:"createdAt"`
	Level        Level          `json:"level"`
	Type         Type           `json:"type"`
	Message      string         `json:"message"`
	Actor        string         `json:"actor,omitempty"`
	Username     string         `json:"username,omitempty"`
	ResourceType string         `json:"resourceType,omitempty"`
	ResourceID   string         `json:"resourceId,omitempty"`
	Fields       map[string]any `json:"fields,omitempty"`
}

type EntryInput struct {
	Level        Level
	Type         Type
	Message      string
	Actor        string
	Username     string
	ResourceType string
	ResourceID   string
	Fields       map[string]any
}

type Query struct {
	Limit int
	Type  Type
	Level Level
}

type Recorder interface {
	Append(EntryInput) (Entry, error)
}

type Reader interface {
	Recent(Query) []Entry
}

type Store struct {
	mu         sync.Mutex
	path       string
	entries    []Entry
	now        func() time.Time
	maxEntries int
}

type persisted struct {
	Entries []Entry `json:"entries"`
}

func NewStore(path string) (*Store, error) {
	return NewStoreWithClock(path, time.Now)
}

func NewStoreWithClock(path string, now func() time.Time) (*Store, error) {
	if now == nil {
		now = time.Now
	}
	store := &Store{path: path, now: now, maxEntries: DefaultMaxEntries}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return store, nil
		}
		return nil, err
	}
	var current persisted
	if err := json.Unmarshal(data, &current); err != nil {
		return nil, fmt.Errorf("读取活动日志失败：%w", err)
	}
	store.entries = append([]Entry{}, current.Entries...)
	store.trimLocked()
	sort.SliceStable(store.entries, func(i int, j int) bool {
		return store.entries[i].CreatedAt.Before(store.entries[j].CreatedAt)
	})
	return store, nil
}

func (s *Store) Append(input EntryInput) (Entry, error) {
	if s == nil {
		return Entry{}, nil
	}
	entry := Entry{
		CreatedAt:    s.now().UTC(),
		Level:        normalizeLevel(input.Level),
		Type:         normalizeType(input.Type),
		Message:      compactString(input.Message, 300),
		Actor:        compactString(input.Actor, 120),
		Username:     compactString(input.Username, 120),
		ResourceType: compactString(input.ResourceType, 80),
		ResourceID:   compactString(input.ResourceID, 160),
		Fields:       sanitizeFields(input.Fields),
	}
	id, err := newEntryID()
	if err != nil {
		return Entry{}, err
	}
	entry.ID = id

	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries = append(s.entries, entry)
	s.trimLocked()
	if err := s.saveLocked(); err != nil {
		return Entry{}, err
	}
	return entry, nil
}

func (s *Store) Recent(query Query) []Entry {
	if s == nil {
		return []Entry{}
	}
	limit := normalizeLimit(query.Limit)
	filterType := normalizeFilterType(query.Type)
	filterLevel := normalizeFilterLevel(query.Level)

	s.mu.Lock()
	defer s.mu.Unlock()
	items := make([]Entry, 0, minInt(limit, len(s.entries)))
	for i := len(s.entries) - 1; i >= 0 && len(items) < limit; i-- {
		entry := s.entries[i]
		if filterType != "" && entry.Type != filterType {
			continue
		}
		if filterLevel != "" && entry.Level != filterLevel {
			continue
		}
		items = append(items, entry)
	}
	return items
}

func (s *Store) trimLocked() {
	if s.maxEntries <= 0 {
		s.maxEntries = DefaultMaxEntries
	}
	if len(s.entries) > s.maxEntries {
		s.entries = append([]Entry{}, s.entries[len(s.entries)-s.maxEntries:]...)
	}
}

func (s *Store) saveLocked() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	payload, err := json.MarshalIndent(persisted{Entries: s.entries}, "", "  ")
	if err != nil {
		return err
	}
	tmp := fmt.Sprintf("%s.%d.tmp", s.path, time.Now().UnixNano())
	if err := os.WriteFile(tmp, append(payload, '\n'), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func normalizeLevel(level Level) Level {
	switch Level(strings.ToLower(strings.TrimSpace(string(level)))) {
	case LevelWarning:
		return LevelWarning
	case LevelError:
		return LevelError
	default:
		return LevelInfo
	}
}

func normalizeFilterLevel(level Level) Level {
	value := Level(strings.ToLower(strings.TrimSpace(string(level))))
	switch value {
	case LevelInfo, LevelWarning, LevelError:
		return value
	default:
		return ""
	}
}

func normalizeType(eventType Type) Type {
	value := normalizeFilterType(eventType)
	if value == "" {
		return TypeSystem
	}
	return value
}

func normalizeFilterType(eventType Type) Type {
	value := Type(strings.ToLower(strings.TrimSpace(string(eventType))))
	switch value {
	case TypeUserRegistration, TypeTopUpOrderCreated, TypeTopUpPaidSuccess, TypeAdminCreditGrant, TypeAdminUserStatus, TypeTaskFailed, TypeResultFailed, TypeSystem:
		return value
	default:
		return ""
	}
}

func normalizeLimit(limit int) int {
	if limit <= 0 {
		return DefaultLimit
	}
	if limit > MaxLimit {
		return MaxLimit
	}
	return limit
}

func sanitizeFields(fields map[string]any) map[string]any {
	if len(fields) == 0 {
		return nil
	}
	out := make(map[string]any, len(fields))
	for key, value := range fields {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		if !allowedFieldKey(key) {
			continue
		}
		out[key] = sanitizeValue(key, value, 0)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func allowedFieldKey(key string) bool {
	switch strings.TrimSpace(key) {
	case "amount",
		"amountCents",
		"balanceAfter",
		"count",
		"credits",
		"elapsedMs",
		"emailSet",
		"errorCode",
		"errorText",
		"failedCount",
		"imageIndex",
		"initialCredits",
		"legacySpaceLinked",
		"method",
		"mode",
		"model",
		"outputFormat",
		"provider",
		"providerTradeNo",
		"referralUsed",
		"status",
		"statusCode",
		"succeededCount",
		"target",
		"taskId":
		return true
	default:
		return false
	}
}

func sanitizeValue(key string, value any, depth int) any {
	if isSensitiveKey(key) {
		return "***"
	}
	if isPromptKey(key) {
		if text, ok := value.(string); ok {
			return fmt.Sprintf("<prompt omitted, %d chars>", len([]rune(text)))
		}
		return "<prompt omitted>"
	}
	if isImagePayloadKey(key) {
		return "<image data omitted>"
	}
	if depth >= 3 {
		return compactString(fmt.Sprint(value), 200)
	}
	switch typed := value.(type) {
	case nil:
		return nil
	case string:
		return compactString(typed, 512)
	case []byte:
		return "<binary data omitted>"
	case []string:
		limit := minInt(len(typed), 8)
		out := make([]any, 0, limit)
		for i := 0; i < limit; i++ {
			out = append(out, sanitizeValue(key, typed[i], depth+1))
		}
		return out
	case []any:
		limit := minInt(len(typed), 8)
		out := make([]any, 0, limit)
		for i := 0; i < limit; i++ {
			out = append(out, sanitizeValue(key, typed[i], depth+1))
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(typed))
		for childKey, childValue := range typed {
			childKey = strings.TrimSpace(childKey)
			if childKey == "" {
				continue
			}
			out[childKey] = sanitizeValue(childKey, childValue, depth+1)
		}
		return out
	default:
		return typed
	}
}

func isSensitiveKey(key string) bool {
	lower := strings.ToLower(strings.TrimSpace(key))
	return strings.Contains(lower, "apikey") ||
		strings.Contains(lower, "api_key") ||
		strings.Contains(lower, "password") ||
		strings.Contains(lower, "secret") ||
		strings.Contains(lower, "token") ||
		strings.Contains(lower, "authorization") ||
		strings.Contains(lower, "payurl") ||
		strings.Contains(lower, "pay_url") ||
		strings.Contains(lower, "cookie")
}

func isPromptKey(key string) bool {
	return strings.Contains(strings.ToLower(strings.TrimSpace(key)), "prompt")
}

func isImagePayloadKey(key string) bool {
	lower := strings.ToLower(strings.TrimSpace(key))
	return strings.Contains(lower, "imagedata") ||
		strings.Contains(lower, "image_data") ||
		strings.Contains(lower, "base64") ||
		strings.Contains(lower, "b64")
}

func compactString(value string, limit int) string {
	value = strings.TrimSpace(value)
	if limit <= 0 {
		return value
	}
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit]) + "..."
}

func newEntryID() (string, error) {
	data := make([]byte, 12)
	if _, err := rand.Read(data); err != nil {
		return "", err
	}
	return "act_" + hex.EncodeToString(data), nil
}

func minInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}
