package users

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
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/y08lin4/lyra-image-workbench/internal/spaces"
)

const SessionTTL = 30 * 24 * time.Hour

var usernamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{2,31}$`)

type Store struct {
	mu       sync.Mutex
	path     string
	current  persisted
	sessions map[string]sessionRecord
}

type persisted struct {
	Users []record `json:"users"`
}

type record struct {
	Username     string `json:"username"`
	DisplayName  string `json:"displayName"`
	StorageToken string `json:"storageToken"`
	SaltHex      string `json:"saltHex"`
	HashHex      string `json:"hashHex"`
	TOTPSecret   string `json:"totpSecret,omitempty"`
	TOTPEnabled  bool   `json:"totpEnabled,omitempty"`
	VideoQuota   int    `json:"videoQuota,omitempty"`
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt"`
	LastLoginAt  string `json:"lastLoginAt,omitempty"`
}

type sessionRecord struct {
	Username  string
	ExpiresAt time.Time
}

type PublicUser struct {
	Username         string `json:"username"`
	DisplayName      string `json:"displayName"`
	TwoFactorEnabled bool   `json:"twoFactorEnabled"`
	VideoQuota       int    `json:"videoQuota"`
	CreatedAt        string `json:"createdAt"`
	LastLoginAt      string `json:"lastLoginAt,omitempty"`
}

type AdminUser struct {
	Username         string `json:"username"`
	DisplayName      string `json:"displayName"`
	TwoFactorEnabled bool   `json:"twoFactorEnabled"`
	VideoQuota       int    `json:"videoQuota"`
	CreatedAt        string `json:"createdAt"`
	LastLoginAt      string `json:"lastLoginAt,omitempty"`
}

type Session struct {
	User         PublicUser `json:"user"`
	ExpiresAt    string     `json:"expiresAt"`
	Token        string     `json:"-"`
	StorageToken string     `json:"-"`
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
		return nil, fmt.Errorf("读取用户配置失败：%w", err)
	}
	return store, nil
}

func (s *Store) Register(username string, password string, storageToken string) (Session, error) {
	normalized, displayName, err := normalizeUsername(username)
	if err != nil {
		return Session{}, err
	}
	if err := spaces.ValidatePassword(password); err != nil {
		return Session{}, err
	}
	if storageToken != "" {
		if storageToken, err = spaces.NormalizeToken(storageToken); err != nil {
			return Session{}, err
		}
	} else {
		storageToken, err = randomHex(32)
		if err != nil {
			return Session{}, err
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.findLocked(normalized); ok {
		return Session{}, NewError("USER_ALREADY_EXISTS", "用户名已存在，请直接登录或换一个用户名")
	}
	salt, err := randomHex(16)
	if err != nil {
		return Session{}, err
	}
	now := time.Now().Format(time.RFC3339)
	s.current.Users = append(s.current.Users, record{
		Username:     displayName,
		DisplayName:  displayName,
		StorageToken: storageToken,
		SaltHex:      salt,
		HashHex:      hashPassword(salt, password),
		CreatedAt:    now,
		UpdatedAt:    now,
		LastLoginAt:  now,
	})
	if err := s.saveLocked(); err != nil {
		return Session{}, err
	}
	return s.newSessionLocked(normalized)
}

func (s *Store) Login(username string, password string, twoFactorCode string) (Session, error) {
	normalized, _, err := normalizeUsername(username)
	if err != nil {
		return Session{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	index, ok := s.findLocked(normalized)
	if !ok {
		return Session{}, NewError("USER_LOGIN_INVALID", "用户名或密码错误")
	}
	user := s.current.Users[index]
	got := hashPassword(user.SaltHex, password)
	if subtle.ConstantTimeCompare([]byte(got), []byte(user.HashHex)) != 1 {
		return Session{}, NewError("USER_LOGIN_INVALID", "用户名或密码错误")
	}
	if user.TOTPEnabled {
		if strings.TrimSpace(twoFactorCode) == "" {
			return Session{}, NewError("USER_TOTP_REQUIRED", "请输入 2FA 验证码")
		}
		if !verifyTOTP(user.TOTPSecret, twoFactorCode, time.Now()) {
			return Session{}, NewError("USER_TOTP_INVALID", "2FA 验证码无效或已过期")
		}
	}
	s.current.Users[index].LastLoginAt = time.Now().Format(time.RFC3339)
	s.current.Users[index].UpdatedAt = s.current.Users[index].LastLoginAt
	if err := s.saveLocked(); err != nil {
		return Session{}, err
	}
	return s.newSessionLocked(normalized)
}

func (s *Store) Current(token string) (Session, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	token = strings.TrimSpace(token)
	if token == "" {
		return Session{}, false
	}
	now := time.Now()
	s.pruneLocked(now)
	session, ok := s.sessions[token]
	if !ok || !now.Before(session.ExpiresAt) {
		return Session{}, false
	}
	index, ok := s.findLocked(session.Username)
	if !ok {
		delete(s.sessions, token)
		return Session{}, false
	}
	return sessionFromRecord(s.current.Users[index], token, session.ExpiresAt), true
}

func (s *Store) Logout(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, strings.TrimSpace(token))
}

func (s *Store) ListAdminUsers() []AdminUser {
	s.mu.Lock()
	defer s.mu.Unlock()
	items := make([]AdminUser, 0, len(s.current.Users))
	for _, user := range s.current.Users {
		items = append(items, adminUserFromRecord(user))
	}
	return items
}

func (s *Store) AddVideoQuota(username string, delta int) (AdminUser, error) {
	if delta <= 0 {
		return AdminUser{}, NewError("USER_VIDEO_QUOTA_DELTA_INVALID", "增加额度必须大于 0")
	}
	normalized, _, err := normalizeUsername(username)
	if err != nil {
		return AdminUser{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	index, ok := s.findLocked(normalized)
	if !ok {
		return AdminUser{}, NewError("USER_NOT_FOUND", "用户不存在")
	}
	s.current.Users[index].VideoQuota += delta
	s.current.Users[index].UpdatedAt = time.Now().Format(time.RFC3339)
	if err := s.saveLocked(); err != nil {
		return AdminUser{}, err
	}
	return adminUserFromRecord(s.current.Users[index]), nil
}

func (s *Store) ConsumeVideoQuota(username string, amount int) (int, error) {
	if amount <= 0 {
		return 0, NewError("USER_VIDEO_QUOTA_AMOUNT_INVALID", "消耗额度必须大于 0")
	}
	normalized, _, err := normalizeUsername(username)
	if err != nil {
		return 0, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	index, ok := s.findLocked(normalized)
	if !ok {
		return 0, NewError("USER_NOT_FOUND", "用户不存在")
	}
	if s.current.Users[index].VideoQuota < amount {
		return s.current.Users[index].VideoQuota, NewError("USER_VIDEO_QUOTA_NOT_ENOUGH", "视频额度不足，请联系管理员增加额度")
	}
	s.current.Users[index].VideoQuota -= amount
	s.current.Users[index].UpdatedAt = time.Now().Format(time.RFC3339)
	if err := s.saveLocked(); err != nil {
		return 0, err
	}
	return s.current.Users[index].VideoQuota, nil
}

func (s *Store) RefundVideoQuota(username string, amount int) {
	if amount <= 0 {
		return
	}
	normalized, _, err := normalizeUsername(username)
	if err != nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	index, ok := s.findLocked(normalized)
	if !ok {
		return
	}
	s.current.Users[index].VideoQuota += amount
	s.current.Users[index].UpdatedAt = time.Now().Format(time.RFC3339)
	_ = s.saveLocked()
}

func (s *Store) VideoQuota(username string) (int, error) {
	normalized, _, err := normalizeUsername(username)
	if err != nil {
		return 0, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	index, ok := s.findLocked(normalized)
	if !ok {
		return 0, NewError("USER_NOT_FOUND", "用户不存在")
	}
	return s.current.Users[index].VideoQuota, nil
}

func (s *Store) BeginTOTPSetup(username string) (TOTPSetup, error) {
	normalized, _, err := normalizeUsername(username)
	if err != nil {
		return TOTPSetup{}, err
	}
	secret, err := newTOTPSecret()
	if err != nil {
		return TOTPSetup{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	index, ok := s.findLocked(normalized)
	if !ok {
		return TOTPSetup{}, NewError("USER_NOT_FOUND", "用户不存在")
	}
	if s.current.Users[index].TOTPEnabled {
		return TOTPSetup{}, NewError("USER_TOTP_ALREADY_ENABLED", "2FA 已开启")
	}
	s.current.Users[index].TOTPSecret = secret
	s.current.Users[index].TOTPEnabled = false
	s.current.Users[index].UpdatedAt = time.Now().Format(time.RFC3339)
	if err := s.saveLocked(); err != nil {
		return TOTPSetup{}, err
	}
	return setupFromSecret(normalized, secret), nil
}

func (s *Store) EnableTOTP(username string, code string) error {
	normalized, _, err := normalizeUsername(username)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	index, ok := s.findLocked(normalized)
	if !ok {
		return NewError("USER_NOT_FOUND", "用户不存在")
	}
	user := s.current.Users[index]
	if strings.TrimSpace(user.TOTPSecret) == "" {
		return NewError("USER_TOTP_SETUP_REQUIRED", "请先生成 2FA 密钥")
	}
	if !verifyTOTP(user.TOTPSecret, code, time.Now()) {
		return NewError("USER_TOTP_INVALID", "2FA 验证码无效或已过期")
	}
	s.current.Users[index].TOTPEnabled = true
	s.current.Users[index].UpdatedAt = time.Now().Format(time.RFC3339)
	return s.saveLocked()
}

func (s *Store) DisableTOTP(username string, code string) error {
	normalized, _, err := normalizeUsername(username)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	index, ok := s.findLocked(normalized)
	if !ok {
		return NewError("USER_NOT_FOUND", "用户不存在")
	}
	user := s.current.Users[index]
	if user.TOTPEnabled && !verifyTOTP(user.TOTPSecret, code, time.Now()) {
		return NewError("USER_TOTP_INVALID", "2FA 验证码无效或已过期")
	}
	s.current.Users[index].TOTPSecret = ""
	s.current.Users[index].TOTPEnabled = false
	s.current.Users[index].UpdatedAt = time.Now().Format(time.RFC3339)
	return s.saveLocked()
}

func (s *Store) newSessionLocked(username string) (Session, error) {
	index, ok := s.findLocked(username)
	if !ok {
		return Session{}, NewError("USER_NOT_FOUND", "用户不存在")
	}
	token, err := randomHex(32)
	if err != nil {
		return Session{}, err
	}
	expires := time.Now().Add(SessionTTL)
	s.sessions[token] = sessionRecord{Username: username, ExpiresAt: expires}
	return sessionFromRecord(s.current.Users[index], token, expires), nil
}

func (s *Store) findLocked(username string) (int, bool) {
	username = normalizeUsernameKey(username)
	for i := range s.current.Users {
		if normalizeUsernameKey(s.current.Users[i].Username) == username {
			return i, true
		}
	}
	return -1, false
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

func sessionFromRecord(user record, token string, expires time.Time) Session {
	return Session{
		User: PublicUser{
			Username:         user.Username,
			DisplayName:      user.DisplayName,
			TwoFactorEnabled: user.TOTPEnabled,
			VideoQuota:       user.VideoQuota,
			CreatedAt:        user.CreatedAt,
			LastLoginAt:      user.LastLoginAt,
		},
		ExpiresAt:    expires.Format(time.RFC3339),
		Token:        token,
		StorageToken: user.StorageToken,
	}
}

func adminUserFromRecord(user record) AdminUser {
	return AdminUser{
		Username:         user.Username,
		DisplayName:      user.DisplayName,
		TwoFactorEnabled: user.TOTPEnabled,
		VideoQuota:       user.VideoQuota,
		CreatedAt:        user.CreatedAt,
		LastLoginAt:      user.LastLoginAt,
	}
}

func normalizeUsername(username string) (string, string, error) {
	displayName := strings.TrimSpace(username)
	normalized := normalizeUsernameKey(displayName)
	if !usernamePattern.MatchString(displayName) {
		return "", "", NewError("USERNAME_INVALID", "用户名只能使用 3-32 位大小写字母、数字、下划线、点或短横线，并且必须以字母或数字开头")
	}
	return normalized, displayName, nil
}

func normalizeUsernameKey(username string) string {
	return strings.ToLower(strings.TrimSpace(username))
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
