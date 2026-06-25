package users

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/mail"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/y08lin4/lyra-image-workbench/internal/spaces"
)

const SessionTTL = 30 * 24 * time.Hour

const (
	creditLedgerTypeAdminAdd       = "admin_add"
	creditLedgerTypePurchase       = "purchase"
	creditLedgerTypeReferralReward = "referral_reward"
	creditLedgerTypeTaskCharge     = "task_charge"
	creditLedgerTypeRefund         = "refund"
)

var usernamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{2,31}$`)

type Store struct {
	mu       sync.Mutex
	path     string
	current  persisted
	sessions map[string]sessionRecord
}

type persisted struct {
	Users        []record            `json:"users"`
	CreditLedger []CreditLedgerEntry `json:"creditLedger,omitempty"`
}

type record struct {
	Username           string `json:"username"`
	DisplayName        string `json:"displayName"`
	Email              string `json:"email,omitempty"`
	AvatarURL          string `json:"avatarUrl,omitempty"`
	IsAdmin            bool   `json:"isAdmin,omitempty"`
	CreditsBalance     int    `json:"creditsBalance,omitempty"`
	ReferralCode       string `json:"referralCode,omitempty"`
	ReferredByCode     string `json:"referredByCode,omitempty"`
	ReferredByUsername string `json:"referredByUsername,omitempty"`
	ReferralRewardedAt string `json:"referralRewardedAt,omitempty"`
	StorageToken       string `json:"storageToken"`
	SaltHex            string `json:"saltHex"`
	HashHex            string `json:"hashHex"`
	TOTPSecret         string `json:"totpSecret,omitempty"`
	TOTPEnabled        bool   `json:"totpEnabled,omitempty"`
	CreatedAt          string `json:"createdAt"`
	UpdatedAt          string `json:"updatedAt"`
	LastLoginAt        string `json:"lastLoginAt,omitempty"`
}

type sessionRecord struct {
	Username  string
	ExpiresAt time.Time
}

type PublicUser struct {
	Username           string `json:"username"`
	DisplayName        string `json:"displayName"`
	Email              string `json:"email"`
	AvatarURL          string `json:"avatarUrl"`
	IsAdmin            bool   `json:"isAdmin"`
	CreditsBalance     int    `json:"creditsBalance"`
	ReferralCode       string `json:"referralCode"`
	ReferredByUsername string `json:"referredByUsername,omitempty"`
	TwoFactorEnabled   bool   `json:"twoFactorEnabled"`
	CreatedAt          string `json:"createdAt"`
	LastLoginAt        string `json:"lastLoginAt,omitempty"`
}

type AdminUser struct {
	Username           string `json:"username"`
	DisplayName        string `json:"displayName"`
	Email              string `json:"email"`
	AvatarURL          string `json:"avatarUrl"`
	IsAdmin            bool   `json:"isAdmin"`
	CreditsBalance     int    `json:"creditsBalance"`
	ReferralCode       string `json:"referralCode"`
	ReferredByCode     string `json:"referredByCode,omitempty"`
	ReferredByUsername string `json:"referredByUsername,omitempty"`
	ReferralRewardedAt string `json:"referralRewardedAt,omitempty"`
	TwoFactorEnabled   bool   `json:"twoFactorEnabled"`
	CreatedAt          string `json:"createdAt"`
	LastLoginAt        string `json:"lastLoginAt,omitempty"`
}

type ProfileUpdate struct {
	DisplayName string
	Email       string
	AvatarURL   string
}

type CreditLedgerEntry struct {
	ID              string `json:"id"`
	Username        string `json:"username"`
	Delta           int    `json:"delta"`
	BalanceAfter    int    `json:"balanceAfter"`
	Type            string `json:"type"`
	Reason          string `json:"reason,omitempty"`
	SourceID        string `json:"sourceId,omitempty"`
	AdminActor      string `json:"adminActor,omitempty"`
	RelatedUsername string `json:"relatedUsername,omitempty"`
	CreatedAt       string `json:"createdAt"`
}

type PurchaseCreditResult struct {
	User          AdminUser          `json:"user"`
	Entry         CreditLedgerEntry  `json:"entry"`
	Created       bool               `json:"created"`
	ReferralEntry *CreditLedgerEntry `json:"referralEntry,omitempty"`
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

func (s *Store) Register(username string, email string, password string, referralCode string, storageToken string) (Session, error) {
	normalized, displayName, err := normalizeUsername(username)
	if err != nil {
		return Session{}, err
	}
	normalizedEmail, err := normalizeEmail(email)
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
	if normalizedEmail != "" {
		if _, ok := s.findByEmailLocked(normalizedEmail); ok {
			return Session{}, NewError("USER_EMAIL_ALREADY_EXISTS", "邮箱已被使用，请直接登录或换一个邮箱")
		}
	}

	normalizedReferralCode := normalizeReferralCode(referralCode)
	referredByUsername := ""
	if normalizedReferralCode != "" {
		index, ok := s.findByReferralCodeLocked(normalizedReferralCode)
		if !ok {
			return Session{}, NewError("REFERRAL_CODE_INVALID", "邀请码无效")
		}
		referredByUsername = s.current.Users[index].Username
	}

	salt, err := randomHex(16)
	if err != nil {
		return Session{}, err
	}
	newReferralCode, err := s.generateReferralCodeLocked()
	if err != nil {
		return Session{}, err
	}
	now := time.Now().Format(time.RFC3339)
	s.current.Users = append(s.current.Users, record{
		Username:           displayName,
		DisplayName:        displayName,
		Email:              normalizedEmail,
		IsAdmin:            len(s.current.Users) == 0,
		CreditsBalance:     0,
		ReferralCode:       newReferralCode,
		ReferredByCode:     normalizedReferralCode,
		ReferredByUsername: referredByUsername,
		StorageToken:       storageToken,
		SaltHex:            salt,
		HashHex:            hashPassword(salt, password),
		CreatedAt:          now,
		UpdatedAt:          now,
		LastLoginAt:        now,
	})
	if err := s.saveLocked(); err != nil {
		return Session{}, err
	}
	return s.newSessionLocked(normalized)
}

func (s *Store) Login(identifier string, password string, twoFactorCode string) (Session, error) {
	identifier = strings.TrimSpace(identifier)
	if identifier == "" {
		return Session{}, NewError("USER_LOGIN_INVALID", "用户名或密码错误")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	index, ok := s.findByIdentifierLocked(identifier)
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
	return s.newSessionLocked(s.current.Users[index].Username)
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
	for i := range s.current.Users {
		items = append(items, adminUserFromRecord(s.current.Users[i]))
	}
	return items
}

func (s *Store) Profile(username string) (PublicUser, error) {
	normalized, _, err := normalizeUsername(username)
	if err != nil {
		return PublicUser{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	index, ok := s.findLocked(normalized)
	if !ok {
		return PublicUser{}, NewError("USER_NOT_FOUND", "用户不存在")
	}
	return publicUserFromRecord(s.current.Users[index]), nil
}

func (s *Store) UpdateProfile(username string, update ProfileUpdate) (PublicUser, error) {
	normalized, _, err := normalizeUsername(username)
	if err != nil {
		return PublicUser{}, err
	}
	displayName, err := normalizeDisplayName(update.DisplayName)
	if err != nil {
		return PublicUser{}, err
	}
	email, err := normalizeEmail(update.Email)
	if err != nil {
		return PublicUser{}, err
	}
	avatarURL, err := normalizeAvatarURL(update.AvatarURL)
	if err != nil {
		return PublicUser{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	index, ok := s.findLocked(normalized)
	if !ok {
		return PublicUser{}, NewError("USER_NOT_FOUND", "用户不存在")
	}
	if email != "" {
		if existing, ok := s.findByEmailLocked(email); ok && existing != index {
			return PublicUser{}, NewError("USER_EMAIL_ALREADY_EXISTS", "邮箱已被使用，请换一个邮箱")
		}
	}
	if displayName == "" {
		displayName = s.current.Users[index].Username
	}
	s.current.Users[index].DisplayName = displayName
	s.current.Users[index].Email = email
	s.current.Users[index].AvatarURL = avatarURL
	s.current.Users[index].UpdatedAt = time.Now().Format(time.RFC3339)
	if err := s.saveLocked(); err != nil {
		return PublicUser{}, err
	}
	return publicUserFromRecord(s.current.Users[index]), nil
}

func (s *Store) EnsureReferralCode(username string) (PublicUser, error) {
	normalized, _, err := normalizeUsername(username)
	if err != nil {
		return PublicUser{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	index, ok := s.findLocked(normalized)
	if !ok {
		return PublicUser{}, NewError("USER_NOT_FOUND", "用户不存在")
	}
	if strings.TrimSpace(s.current.Users[index].ReferralCode) == "" {
		code, err := s.generateReferralCodeLocked()
		if err != nil {
			return PublicUser{}, err
		}
		s.current.Users[index].ReferralCode = code
		s.current.Users[index].UpdatedAt = time.Now().Format(time.RFC3339)
		if err := s.saveLocked(); err != nil {
			return PublicUser{}, err
		}
	}
	return publicUserFromRecord(s.current.Users[index]), nil
}

func (s *Store) ListCreditLedger(username string) ([]CreditLedgerEntry, error) {
	normalized, _, err := normalizeUsername(username)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.findLocked(normalized); !ok {
		return nil, NewError("USER_NOT_FOUND", "用户不存在")
	}
	items := make([]CreditLedgerEntry, 0)
	for _, entry := range s.current.CreditLedger {
		if normalizeUsernameKey(entry.Username) == normalized {
			items = append(items, entry)
		}
	}
	return items, nil
}

func (s *Store) SetAdmin(username string, isAdmin bool) (AdminUser, error) {
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
	if !isAdmin && s.current.Users[index].IsAdmin && s.countAdminsLocked() == 1 {
		return AdminUser{}, NewError("USER_LAST_ADMIN_REQUIRED", "至少需要保留一个管理员")
	}
	s.current.Users[index].IsAdmin = isAdmin
	s.current.Users[index].UpdatedAt = time.Now().Format(time.RFC3339)
	if err := s.saveLocked(); err != nil {
		return AdminUser{}, err
	}
	return adminUserFromRecord(s.current.Users[index]), nil
}

func (s *Store) AddCreditsByAdmin(username string, amount int, reason string, adminActor string) (AdminUser, CreditLedgerEntry, error) {
	if amount <= 0 {
		return AdminUser{}, CreditLedgerEntry{}, NewError("USER_CREDITS_AMOUNT_INVALID", "增加次数必须大于 0")
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return AdminUser{}, CreditLedgerEntry{}, NewError("USER_CREDITS_REASON_REQUIRED", "管理员加次数必须填写原因")
	}
	normalized, _, err := normalizeUsername(username)
	if err != nil {
		return AdminUser{}, CreditLedgerEntry{}, err
	}
	adminActor = strings.TrimSpace(adminActor)
	if adminActor == "" {
		adminActor = "admin"
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	index, ok := s.findLocked(normalized)
	if !ok {
		return AdminUser{}, CreditLedgerEntry{}, NewError("USER_NOT_FOUND", "用户不存在")
	}
	now := time.Now().Format(time.RFC3339)
	entry, err := s.appendCreditLedgerLocked(index, amount, creditLedgerTypeAdminAdd, reason, "", adminActor, "", now)
	if err != nil {
		return AdminUser{}, CreditLedgerEntry{}, err
	}
	if err := s.saveLocked(); err != nil {
		return AdminUser{}, CreditLedgerEntry{}, err
	}
	return adminUserFromRecord(s.current.Users[index]), entry, nil
}

func (s *Store) AddPurchaseCredits(username string, amount int, sourceID string, referralRewardCredits int) (PurchaseCreditResult, error) {
	if amount <= 0 {
		return PurchaseCreditResult{}, NewError("USER_CREDITS_AMOUNT_INVALID", "入账次数必须大于 0")
	}
	sourceID = strings.TrimSpace(sourceID)
	if sourceID == "" {
		return PurchaseCreditResult{}, NewError("USER_CREDITS_SOURCE_REQUIRED", "购买入账必须包含订单号")
	}
	normalized, _, err := normalizeUsername(username)
	if err != nil {
		return PurchaseCreditResult{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.findLedgerBySourceLocked(creditLedgerTypePurchase, sourceID); ok {
		if normalizeUsernameKey(existing.Username) != normalized {
			return PurchaseCreditResult{}, NewError("USER_CREDITS_SOURCE_CONFLICT", "订单号已被其他用户使用")
		}
		index, ok := s.findLocked(existing.Username)
		if !ok {
			return PurchaseCreditResult{}, NewError("USER_NOT_FOUND", "用户不存在")
		}
		return PurchaseCreditResult{User: adminUserFromRecord(s.current.Users[index]), Entry: existing, Created: false}, nil
	}

	index, ok := s.findLocked(normalized)
	if !ok {
		return PurchaseCreditResult{}, NewError("USER_NOT_FOUND", "用户不存在")
	}
	now := time.Now().Format(time.RFC3339)
	entry, err := s.appendCreditLedgerLocked(index, amount, creditLedgerTypePurchase, "购买入账", sourceID, "", "", now)
	if err != nil {
		return PurchaseCreditResult{}, err
	}
	result := PurchaseCreditResult{Entry: entry, Created: true}
	buyerIndex := index
	buyer := s.current.Users[buyerIndex]
	if referralRewardCredits > 0 && buyer.ReferredByUsername != "" && buyer.ReferralRewardedAt == "" {
		inviterIndex, ok := s.findLocked(buyer.ReferredByUsername)
		if ok && inviterIndex != buyerIndex {
			reward, err := s.appendCreditLedgerLocked(inviterIndex, referralRewardCredits, creditLedgerTypeReferralReward, "邀请用户首次充值奖励", "referral:"+sourceID, "", buyer.Username, now)
			if err != nil {
				return PurchaseCreditResult{}, err
			}
			result.ReferralEntry = &reward
			s.current.Users[buyerIndex].ReferralRewardedAt = now
			s.current.Users[buyerIndex].UpdatedAt = now
		}
	}
	if err := s.saveLocked(); err != nil {
		return PurchaseCreditResult{}, err
	}
	result.User = adminUserFromRecord(s.current.Users[index])
	return result, nil
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
	s.sessions[token] = sessionRecord{Username: s.current.Users[index].Username, ExpiresAt: expires}
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

func (s *Store) findByIdentifierLocked(identifier string) (int, bool) {
	if index, ok := s.findLocked(identifier); ok {
		return index, true
	}
	return s.findByEmailLocked(normalizeEmailKey(identifier))
}

func (s *Store) findByEmailLocked(email string) (int, bool) {
	email = normalizeEmailKey(email)
	if email == "" {
		return -1, false
	}
	for i := range s.current.Users {
		if normalizeEmailKey(s.current.Users[i].Email) == email {
			return i, true
		}
	}
	return -1, false
}

func (s *Store) findByReferralCodeLocked(code string) (int, bool) {
	code = normalizeReferralCode(code)
	if code == "" {
		return -1, false
	}
	for i := range s.current.Users {
		if normalizeReferralCode(s.current.Users[i].ReferralCode) == code {
			return i, true
		}
	}
	return -1, false
}

func (s *Store) findLedgerBySourceLocked(entryType string, sourceID string) (CreditLedgerEntry, bool) {
	entryType = strings.TrimSpace(entryType)
	sourceID = strings.TrimSpace(sourceID)
	if sourceID == "" {
		return CreditLedgerEntry{}, false
	}
	for _, entry := range s.current.CreditLedger {
		if entry.Type == entryType && strings.TrimSpace(entry.SourceID) == sourceID {
			return entry, true
		}
	}
	return CreditLedgerEntry{}, false
}

func (s *Store) appendCreditLedgerLocked(index int, delta int, entryType string, reason string, sourceID string, adminActor string, relatedUsername string, now string) (CreditLedgerEntry, error) {
	entryType = strings.TrimSpace(entryType)
	if entryType == "" {
		return CreditLedgerEntry{}, NewError("USER_CREDIT_TYPE_REQUIRED", "额度流水类型不能为空")
	}
	if delta == 0 {
		return CreditLedgerEntry{}, NewError("USER_CREDIT_DELTA_INVALID", "额度变动不能为 0")
	}
	nextBalance := s.current.Users[index].CreditsBalance + delta
	if nextBalance < 0 {
		return CreditLedgerEntry{}, NewError("USER_CREDITS_NOT_ENOUGH", "次数不足")
	}
	id, err := newLedgerID()
	if err != nil {
		return CreditLedgerEntry{}, err
	}
	s.current.Users[index].CreditsBalance = nextBalance
	s.current.Users[index].UpdatedAt = now
	entry := CreditLedgerEntry{
		ID:              id,
		Username:        s.current.Users[index].Username,
		Delta:           delta,
		BalanceAfter:    nextBalance,
		Type:            entryType,
		Reason:          strings.TrimSpace(reason),
		SourceID:        strings.TrimSpace(sourceID),
		AdminActor:      strings.TrimSpace(adminActor),
		RelatedUsername: strings.TrimSpace(relatedUsername),
		CreatedAt:       now,
	}
	s.current.CreditLedger = append(s.current.CreditLedger, entry)
	return entry, nil
}

func (s *Store) generateReferralCodeLocked() (string, error) {
	for i := 0; i < 16; i++ {
		value, err := randomHex(4)
		if err != nil {
			return "", err
		}
		code := strings.ToUpper(value)
		if _, ok := s.findByReferralCodeLocked(code); !ok {
			return code, nil
		}
	}
	return "", NewError("REFERRAL_CODE_GENERATE_FAILED", "生成邀请码失败，请稍后重试")
}

func (s *Store) countAdminsLocked() int {
	count := 0
	for _, user := range s.current.Users {
		if user.IsAdmin {
			count++
		}
	}
	return count
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
		User:         publicUserFromRecord(user),
		ExpiresAt:    expires.Format(time.RFC3339),
		Token:        token,
		StorageToken: user.StorageToken,
	}
}

func publicUserFromRecord(user record) PublicUser {
	return PublicUser{
		Username:           user.Username,
		DisplayName:        user.DisplayName,
		Email:              user.Email,
		AvatarURL:          user.AvatarURL,
		IsAdmin:            user.IsAdmin,
		CreditsBalance:     user.CreditsBalance,
		ReferralCode:       user.ReferralCode,
		ReferredByUsername: user.ReferredByUsername,
		TwoFactorEnabled:   user.TOTPEnabled,
		CreatedAt:          user.CreatedAt,
		LastLoginAt:        user.LastLoginAt,
	}
}

func adminUserFromRecord(user record) AdminUser {
	return AdminUser{
		Username:           user.Username,
		DisplayName:        user.DisplayName,
		Email:              user.Email,
		AvatarURL:          user.AvatarURL,
		IsAdmin:            user.IsAdmin,
		CreditsBalance:     user.CreditsBalance,
		ReferralCode:       user.ReferralCode,
		ReferredByCode:     user.ReferredByCode,
		ReferredByUsername: user.ReferredByUsername,
		ReferralRewardedAt: user.ReferralRewardedAt,
		TwoFactorEnabled:   user.TOTPEnabled,
		CreatedAt:          user.CreatedAt,
		LastLoginAt:        user.LastLoginAt,
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

func normalizeEmail(email string) (string, error) {
	email = strings.TrimSpace(email)
	if email == "" {
		return "", nil
	}
	address, err := mail.ParseAddress(email)
	if err != nil || address.Name != "" || address.Address != email || !strings.Contains(address.Address, "@") {
		return "", NewError("EMAIL_INVALID", "邮箱格式无效")
	}
	if len(address.Address) > 254 {
		return "", NewError("EMAIL_INVALID", "邮箱长度不能超过 254 个字符")
	}
	return normalizeEmailKey(address.Address), nil
}

func normalizeEmailKey(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func normalizeDisplayName(displayName string) (string, error) {
	displayName = strings.TrimSpace(displayName)
	if len([]rune(displayName)) > 64 {
		return "", NewError("USER_DISPLAY_NAME_INVALID", "昵称不能超过 64 个字符")
	}
	return displayName, nil
}

func normalizeAvatarURL(avatarURL string) (string, error) {
	avatarURL = strings.TrimSpace(avatarURL)
	if len(avatarURL) > 2048 {
		return "", NewError("USER_AVATAR_URL_INVALID", "头像地址不能超过 2048 个字符")
	}
	return avatarURL, nil
}

func normalizeReferralCode(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
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

func newLedgerID() (string, error) {
	value, err := randomHex(12)
	if err != nil {
		return "", err
	}
	return "ledger_" + value, nil
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
