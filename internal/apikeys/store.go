package apikeys

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

const SecretPrefix = "lyra_sk_"

var namePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_. -]{0,63}$`)

type Store struct {
	mu      sync.Mutex
	path    string
	current persisted
}

type persisted struct {
	Keys []Record `json:"keys"`
}

type Record struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Prefix       string `json:"prefix"`
	Hash         string `json:"hash"`
	Username     string `json:"username"`
	StorageToken string `json:"storageToken"`
	CreatedAt    string `json:"createdAt"`
	LastUsedAt   string `json:"lastUsedAt,omitempty"`
	RevokedAt    string `json:"revokedAt,omitempty"`
}

type PublicRecord struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Prefix     string `json:"prefix"`
	CreatedAt  string `json:"createdAt"`
	LastUsedAt string `json:"lastUsedAt,omitempty"`
}

func NewStore(path string) (*Store, error) {
	store := &Store{path: path}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return store, nil
		}
		return nil, err
	}
	if err := json.Unmarshal(data, &store.current); err != nil {
		return nil, fmt.Errorf("读取开发者 API Key 失败：%w", err)
	}
	return store, nil
}

func (s *Store) Create(name string, username string, storageToken string) (Record, string, error) {
	name, err := normalizeName(name)
	if err != nil {
		return Record{}, "", err
	}
	secret, err := newSecret()
	if err != nil {
		return Record{}, "", err
	}
	id, err := newID()
	if err != nil {
		return Record{}, "", err
	}
	now := time.Now().Format(time.RFC3339)
	record := Record{
		ID:           id,
		Name:         name,
		Prefix:       secretPrefix(secret),
		Hash:         hashSecret(secret),
		Username:     strings.TrimSpace(username),
		StorageToken: strings.TrimSpace(storageToken),
		CreatedAt:    now,
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.current.Keys = append(s.current.Keys, record)
	if err := s.saveLocked(); err != nil {
		return Record{}, "", err
	}
	return record, secret, nil
}

func (s *Store) List(storageToken string) []PublicRecord {
	s.mu.Lock()
	defer s.mu.Unlock()
	storageToken = strings.TrimSpace(storageToken)
	items := make([]PublicRecord, 0)
	for _, record := range s.current.Keys {
		if record.StorageToken == storageToken && strings.TrimSpace(record.RevokedAt) == "" {
			items = append(items, ToPublic(record))
		}
	}
	sort.Slice(items, func(i int, j int) bool { return items[i].CreatedAt > items[j].CreatedAt })
	return items
}

func (s *Store) Delete(storageToken string, id string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	storageToken = strings.TrimSpace(storageToken)
	id = strings.TrimSpace(id)
	for i, record := range s.current.Keys {
		if record.ID == id && record.StorageToken == storageToken {
			s.current.Keys = append(s.current.Keys[:i], s.current.Keys[i+1:]...)
			if err := s.saveLocked(); err != nil {
				return false, err
			}
			return true, nil
		}
	}
	return false, nil
}

func (s *Store) Authenticate(secret string) (Record, bool, error) {
	secret = strings.TrimSpace(secret)
	if !strings.HasPrefix(secret, SecretPrefix) {
		return Record{}, false, nil
	}
	hash := hashSecret(secret)
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, record := range s.current.Keys {
		if strings.TrimSpace(record.RevokedAt) != "" {
			continue
		}
		if subtle.ConstantTimeCompare([]byte(record.Hash), []byte(hash)) == 1 {
			now := time.Now().Format(time.RFC3339)
			s.current.Keys[i].LastUsedAt = now
			if err := s.saveLocked(); err != nil {
				return Record{}, false, err
			}
			record.LastUsedAt = now
			return record, true, nil
		}
	}
	return Record{}, false, nil
}

func ToPublic(record Record) PublicRecord {
	return PublicRecord{
		ID:         record.ID,
		Name:       record.Name,
		Prefix:     record.Prefix,
		CreatedAt:  record.CreatedAt,
		LastUsedAt: record.LastUsedAt,
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

func normalizeName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "default"
	}
	if !namePattern.MatchString(name) {
		return "", errors.New("API Key 名称只能使用 1-64 位字母、数字、空格、下划线、点或短横线，并且必须以字母或数字开头")
	}
	return name, nil
}

func newSecret() (string, error) {
	data := make([]byte, 32)
	if _, err := rand.Read(data); err != nil {
		return "", err
	}
	return SecretPrefix + base64.RawURLEncoding.EncodeToString(data), nil
}

func newID() (string, error) {
	var bytes [6]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", err
	}
	return "ak_" + time.Now().Format("20060102150405") + "_" + hex.EncodeToString(bytes[:]), nil
}

func hashSecret(secret string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(secret)))
	return hex.EncodeToString(sum[:])
}

func secretPrefix(secret string) string {
	if len(secret) <= 16 {
		return secret
	}
	return secret[:16]
}
