package adminauth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/y08lin4/lyra-image-workbench/internal/spaces"
)

const SessionTTL = 12 * time.Hour

type Store struct {
	mu       sync.Mutex
	path     string
	current  record
	sessions map[string]time.Time
}

type record struct {
	SaltHex   string `json:"saltHex"`
	HashHex   string `json:"hashHex"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

type PublicStatus struct {
	PasswordSet   bool   `json:"passwordSet"`
	SessionTtlSec int    `json:"sessionTtlSec"`
	UpdatedAt     string `json:"updatedAt"`
}

type Session struct {
	Token     string `json:"token"`
	ExpiresAt string `json:"expiresAt"`
}

func NewStore(path string) (*Store, error) {
	store := &Store{path: path, sessions: make(map[string]time.Time)}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return store, nil
		}
		return nil, err
	}
	if err := json.Unmarshal(data, &store.current); err != nil {
		return nil, fmt.Errorf("读取 Admin 密码配置失败：%w", err)
	}
	return store, nil
}

func (s *Store) Status() PublicStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	return PublicStatus{
		PasswordSet:   s.passwordSetLocked(),
		SessionTtlSec: int(SessionTTL.Seconds()),
		UpdatedAt:     s.current.UpdatedAt,
	}
}

func (s *Store) Setup(password string) (Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.passwordSetLocked() {
		return Session{}, NewError("ADMIN_PASSWORD_ALREADY_SET", "Admin 密码已设置，请直接登录")
	}
	if err := spaces.ValidatePassword(password); err != nil {
		return Session{}, err
	}
	salt, err := randomHex(16)
	if err != nil {
		return Session{}, err
	}
	now := time.Now().Format(time.RFC3339)
	s.current = record{
		SaltHex:   salt,
		HashHex:   hashPassword(salt, password),
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.saveLocked(); err != nil {
		return Session{}, err
	}
	return s.newSessionLocked()
}

func (s *Store) Login(password string) (Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.passwordSetLocked() {
		return Session{}, NewError("ADMIN_PASSWORD_NOT_SET", "请先设置 Admin 管理密码")
	}
	if !s.matchPasswordLocked(password) {
		return Session{}, NewError("ADMIN_PASSWORD_INVALID", "Admin 密码错误")
	}
	return s.newSessionLocked()
}

func (s *Store) ValidateToken(token string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	token = strings.TrimSpace(token)
	if token == "" || !s.passwordSetLocked() {
		return false
	}
	now := time.Now()
	s.pruneLocked(now)
	expires, ok := s.sessions[token]
	return ok && now.Before(expires)
}

func (s *Store) Logout(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, strings.TrimSpace(token))
}

func (s *Store) passwordSetLocked() bool {
	return strings.TrimSpace(s.current.SaltHex) != "" && strings.TrimSpace(s.current.HashHex) != ""
}

func (s *Store) matchPasswordLocked(password string) bool {
	got := hashPassword(s.current.SaltHex, password)
	return subtle.ConstantTimeCompare([]byte(got), []byte(s.current.HashHex)) == 1
}

func (s *Store) newSessionLocked() (Session, error) {
	token, err := randomHex(32)
	if err != nil {
		return Session{}, err
	}
	expires := time.Now().Add(SessionTTL)
	s.sessions[token] = expires
	return Session{Token: token, ExpiresAt: expires.Format(time.RFC3339)}, nil
}

func (s *Store) pruneLocked(now time.Time) {
	for token, expires := range s.sessions {
		if !now.Before(expires) {
			delete(s.sessions, token)
		}
	}
}

func (s *Store) saveLocked() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	payload, err := json.MarshalIndent(s.current, "", "  ")
	if err != nil {
		return err
	}
	tmp := fmt.Sprintf("%s.%d.tmp", s.path, time.Now().UnixNano())
	if err := os.WriteFile(tmp, append(payload, '\n'), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func hashPassword(saltHex string, password string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(saltHex) + ":" + strings.TrimSpace(password)))
	return hex.EncodeToString(sum[:])
}

func randomHex(size int) (string, error) {
	data := make([]byte, size)
	if _, err := rand.Read(data); err != nil {
		return "", err
	}
	return hex.EncodeToString(data), nil
}

type Error struct {
	Code    string
	Chinese string
}

func NewError(code string, chinese string) Error {
	return Error{Code: code, Chinese: chinese}
}

func (e Error) Error() string { return e.Chinese }

func AsError(err error, target *Error) bool {
	return errors.As(err, target)
}
