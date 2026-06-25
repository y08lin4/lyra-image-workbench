package billing

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type TopUpStatus string

const (
	TopUpStatusPending TopUpStatus = "pending"
	TopUpStatusSuccess TopUpStatus = "success"
	TopUpStatusFailed  TopUpStatus = "failed"
)

var (
	ErrOrderNotFound      = errors.New("top-up order not found")
	ErrDuplicateTradeNo   = errors.New("top-up trade number already exists")
	ErrInvalidOrder       = errors.New("invalid top-up order")
	ErrOrderStatusInvalid = errors.New("top-up order status invalid")
)

type TopUpOrder struct {
	TradeNo         string      `json:"tradeNo"`
	Username        string      `json:"username"`
	Credits         int         `json:"credits"`
	AmountCents     int64       `json:"amountCents"`
	Method          string      `json:"method"`
	Status          TopUpStatus `json:"status"`
	ProviderTradeNo string      `json:"providerTradeNo,omitempty"`
	CreatedAt       time.Time   `json:"createdAt"`
	UpdatedAt       time.Time   `json:"updatedAt"`
	PaidAt          *time.Time  `json:"paidAt,omitempty"`
	FailedAt        *time.Time  `json:"failedAt,omitempty"`
}

type CreateOrderInput struct {
	Username    string
	Credits     int
	AmountCents int64
	Method      string
}

type CreditGrant struct {
	Username    string `json:"username"`
	Credits     int    `json:"credits"`
	SourceID    string `json:"sourceId"`
	AmountCents int64  `json:"amountCents"`
}

type CreditRecorder interface {
	RecordPurchaseCredit(ctx context.Context, grant CreditGrant) error
}

type Store struct {
	mu      sync.Mutex
	path    string
	current persisted
	now     func() time.Time
}

type persisted struct {
	Orders []TopUpOrder `json:"orders"`
}

func NewStore(path string) (*Store, error) {
	return NewStoreWithClock(path, time.Now)
}

func NewStoreWithClock(path string, now func() time.Time) (*Store, error) {
	if now == nil {
		now = time.Now
	}
	store := &Store{path: path, now: now}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return store, nil
		}
		return nil, err
	}
	if err := json.Unmarshal(data, &store.current); err != nil {
		return nil, fmt.Errorf("读取充值订单失败：%w", err)
	}
	return store, nil
}

func (s *Store) CreateOrder(input CreateOrderInput) (TopUpOrder, error) {
	normalized, err := normalizeCreateOrderInput(input)
	if err != nil {
		return TopUpOrder{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now().UTC()
	tradeNo, err := s.nextTradeNoLocked(now)
	if err != nil {
		return TopUpOrder{}, err
	}
	order := TopUpOrder{
		TradeNo:     tradeNo,
		Username:    normalized.Username,
		Credits:     normalized.Credits,
		AmountCents: normalized.AmountCents,
		Method:      normalized.Method,
		Status:      TopUpStatusPending,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	s.current.Orders = append(s.current.Orders, order)
	if err := s.saveLocked(); err != nil {
		return TopUpOrder{}, err
	}
	return order, nil
}

func (s *Store) GetByTradeNo(tradeNo string) (TopUpOrder, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	tradeNo = strings.TrimSpace(tradeNo)
	for _, order := range s.current.Orders {
		if order.TradeNo == tradeNo {
			return order, true
		}
	}
	return TopUpOrder{}, false
}

func (s *Store) ListByUsername(username string) []TopUpOrder {
	s.mu.Lock()
	defer s.mu.Unlock()
	username = strings.TrimSpace(username)
	orders := make([]TopUpOrder, 0)
	for _, order := range s.current.Orders {
		if order.Username == username {
			orders = append(orders, order)
		}
	}
	sort.Slice(orders, func(i int, j int) bool {
		return orders[i].CreatedAt.After(orders[j].CreatedAt)
	})
	return orders
}

func (s *Store) MarkSuccess(tradeNo string, providerTradeNo string, paidAt time.Time) (TopUpOrder, *CreditGrant, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	index := s.indexByTradeNoLocked(tradeNo)
	if index < 0 {
		return TopUpOrder{}, nil, ErrOrderNotFound
	}
	order := &s.current.Orders[index]
	if order.Status == TopUpStatusSuccess {
		return *order, nil, nil
	}
	if order.Status != TopUpStatusPending {
		return TopUpOrder{}, nil, ErrOrderStatusInvalid
	}

	if paidAt.IsZero() {
		paidAt = s.now()
	}
	paidAt = paidAt.UTC()
	order.Status = TopUpStatusSuccess
	order.ProviderTradeNo = strings.TrimSpace(providerTradeNo)
	order.PaidAt = &paidAt
	order.UpdatedAt = paidAt
	if err := s.saveLocked(); err != nil {
		return TopUpOrder{}, nil, err
	}

	grant := &CreditGrant{
		Username:    order.Username,
		Credits:     order.Credits,
		SourceID:    order.TradeNo,
		AmountCents: order.AmountCents,
	}
	return *order, grant, nil
}

func (s *Store) MarkFailed(tradeNo string, providerTradeNo string, failedAt time.Time) (TopUpOrder, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	index := s.indexByTradeNoLocked(tradeNo)
	if index < 0 {
		return TopUpOrder{}, false, ErrOrderNotFound
	}
	order := &s.current.Orders[index]
	if order.Status == TopUpStatusFailed {
		return *order, false, nil
	}
	if order.Status != TopUpStatusPending {
		return TopUpOrder{}, false, ErrOrderStatusInvalid
	}

	if failedAt.IsZero() {
		failedAt = s.now()
	}
	failedAt = failedAt.UTC()
	order.Status = TopUpStatusFailed
	order.ProviderTradeNo = strings.TrimSpace(providerTradeNo)
	order.FailedAt = &failedAt
	order.UpdatedAt = failedAt
	if err := s.saveLocked(); err != nil {
		return TopUpOrder{}, false, err
	}
	return *order, true, nil
}

func (s *Store) nextTradeNoLocked(now time.Time) (string, error) {
	prefix := "LYRA" + now.Format("20060102")
	maxSeq := 0
	for _, order := range s.current.Orders {
		if !strings.HasPrefix(order.TradeNo, prefix) {
			continue
		}
		var seq int
		if _, err := fmt.Sscanf(strings.TrimPrefix(order.TradeNo, prefix), "%d", &seq); err == nil && seq > maxSeq {
			maxSeq = seq
		}
	}
	for seq := maxSeq + 1; seq < maxSeq+10000; seq++ {
		tradeNo := fmt.Sprintf("%s%04d", prefix, seq)
		if s.indexByTradeNoLocked(tradeNo) < 0 {
			return tradeNo, nil
		}
	}
	suffix, err := randomHex(4)
	if err != nil {
		return "", err
	}
	return prefix + suffix, nil
}

func (s *Store) indexByTradeNoLocked(tradeNo string) int {
	tradeNo = strings.TrimSpace(tradeNo)
	for i := range s.current.Orders {
		if s.current.Orders[i].TradeNo == tradeNo {
			return i
		}
	}
	return -1
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

func normalizeCreateOrderInput(input CreateOrderInput) (CreateOrderInput, error) {
	normalized := CreateOrderInput{
		Username:    strings.TrimSpace(input.Username),
		Credits:     input.Credits,
		AmountCents: input.AmountCents,
		Method:      strings.TrimSpace(input.Method),
	}
	if normalized.Username == "" || normalized.Credits <= 0 || normalized.AmountCents <= 0 || normalized.Method == "" {
		return CreateOrderInput{}, ErrInvalidOrder
	}
	return normalized, nil
}

func randomHex(size int) (string, error) {
	data := make([]byte, size)
	if _, err := rand.Read(data); err != nil {
		return "", err
	}
	return hex.EncodeToString(data), nil
}
