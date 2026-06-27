package adminauth

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/y08lin4/lyra-image-workbench/internal/passwordhash"

	"github.com/y08lin4/lyra-image-workbench/internal/spaces"
)

const SessionTTL = 12 * time.Hour

type Store struct {
	mu       sync.Mutex
	path     string
	current  record
	sessions map[string]sessionRecord
}

type record struct {
	SaltHex   string `json:"saltHex"`
	HashHex   string `json:"hashHex"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

type sessionRecord struct {
	ExpiresAt time.Time
	Actor     string
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
	store := &Store{path: path, sessions: make(map[string]sessionRecord)}
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
	salt, encodedHash, err := passwordhash.New(password)
	if err != nil {
		return Session{}, err
	}
	now := time.Now().Format(time.RFC3339)
	s.current = record{
		SaltHex:   salt,
		HashHex:   encodedHash,
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
	passwordOK, needsPasswordUpgrade := s.matchPasswordLocked(password)
	if !passwordOK {
		return Session{}, NewError("ADMIN_PASSWORD_INVALID", "Admin 密码错误")
	}
	if needsPasswordUpgrade {
		if err := s.upgradePasswordLocked(password); err != nil {
			return Session{}, err
		}
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
	session, ok := s.sessions[token]
	return ok && now.Before(session.ExpiresAt)
}

func (s *Store) SetSessionActor(token string, actor string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	token = strings.TrimSpace(token)
	actor = strings.TrimSpace(actor)
	if token == "" || actor == "" {
		return
	}
	session, ok := s.sessions[token]
	if !ok || !time.Now().Before(session.ExpiresAt) {
		return
	}
	session.Actor = actor
	s.sessions[token] = session
}

func (s *Store) TokenActor(token string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	token = strings.TrimSpace(token)
	if token == "" || !s.passwordSetLocked() {
		return "", false
	}
	now := time.Now()
	s.pruneLocked(now)
	session, ok := s.sessions[token]
	if !ok || !now.Before(session.ExpiresAt) || strings.TrimSpace(session.Actor) == "" {
		return "", false
	}
	return session.Actor, true
}

func (s *Store) Logout(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, strings.TrimSpace(token))
}

func (s *Store) passwordSetLocked() bool {
	return strings.TrimSpace(s.current.SaltHex) != "" && strings.TrimSpace(s.current.HashHex) != ""
}

func (s *Store) matchPasswordLocked(password string) (bool, bool) {
	return passwordhash.Verify(s.current.SaltHex, s.current.HashHex, password)
}

func (s *Store) upgradePasswordLocked(password string) error {
	salt, encodedHash, err := passwordhash.New(password)
	if err != nil {
		return err
	}
	s.current.SaltHex = salt
	s.current.HashHex = encodedHash
	s.current.UpdatedAt = time.Now().Format(time.RFC3339)
	return s.saveLocked()
}

func (s *Store) newSessionLocked() (Session, error) {
	token, err := randomHex(32)
	if err != nil {
		return Session{}, err
	}
	expires := time.Now().Add(SessionTTL)
	s.sessions[token] = sessionRecord{ExpiresAt: expires}
	return Session{Token: token, ExpiresAt: expires.Format(time.RFC3339)}, nil
}

func (s *Store) pruneLocked(now time.Time) {
	for token, session := range s.sessions {
		if !now.Before(session.ExpiresAt) {
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
