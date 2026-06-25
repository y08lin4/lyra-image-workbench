package billing

import (
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestStoreCreateOrderGeneratesTradeNoAndPersists(t *testing.T) {
	now := time.Date(2026, 6, 26, 9, 30, 0, 0, time.FixedZone("CST", 8*60*60))
	path := filepath.Join(t.TempDir(), "topups.json")
	store, err := NewStoreWithClock(path, func() time.Time { return now })
	if err != nil {
		t.Fatalf("NewStoreWithClock() error = %v", err)
	}

	first, err := store.CreateOrder(CreateOrderInput{
		Username:    " alice ",
		Credits:     100,
		AmountCents: 1000,
		Method:      " alipay ",
	})
	if err != nil {
		t.Fatalf("CreateOrder(first) error = %v", err)
	}
	second, err := store.CreateOrder(CreateOrderInput{
		Username:    "alice",
		Credits:     200,
		AmountCents: 2000,
		Method:      "wxpay",
	})
	if err != nil {
		t.Fatalf("CreateOrder(second) error = %v", err)
	}

	if first.TradeNo != "LYRA202606260001" || second.TradeNo != "LYRA202606260002" {
		t.Fatalf("trade numbers = %q, %q", first.TradeNo, second.TradeNo)
	}
	if first.Username != "alice" || first.Method != "alipay" || first.Status != TopUpStatusPending {
		t.Fatalf("first order not normalized: %+v", first)
	}

	reopened, err := NewStore(path)
	if err != nil {
		t.Fatalf("reopen NewStore() error = %v", err)
	}
	got, ok := reopened.GetByTradeNo(first.TradeNo)
	if !ok {
		t.Fatalf("GetByTradeNo(%q) not found", first.TradeNo)
	}
	if got.Credits != first.Credits || got.AmountCents != first.AmountCents || got.Status != TopUpStatusPending {
		t.Fatalf("persisted order = %+v, want %+v", got, first)
	}
}

func TestStoreMarkSuccessReturnsCreditGrantOnce(t *testing.T) {
	now := time.Date(2026, 6, 26, 1, 0, 0, 0, time.UTC)
	store, err := NewStoreWithClock(filepath.Join(t.TempDir(), "topups.json"), func() time.Time { return now })
	if err != nil {
		t.Fatalf("NewStoreWithClock() error = %v", err)
	}
	order, err := store.CreateOrder(CreateOrderInput{
		Username:    "alice",
		Credits:     100,
		AmountCents: 1000,
		Method:      "alipay",
	})
	if err != nil {
		t.Fatalf("CreateOrder() error = %v", err)
	}

	paidAt := now.Add(3 * time.Minute)
	updated, grant, err := store.MarkSuccess(order.TradeNo, "E20260626001", paidAt)
	if err != nil {
		t.Fatalf("MarkSuccess(first) error = %v", err)
	}
	if updated.Status != TopUpStatusSuccess || updated.PaidAt == nil || updated.ProviderTradeNo != "E20260626001" {
		t.Fatalf("updated order = %+v", updated)
	}
	if grant == nil {
		t.Fatal("first MarkSuccess() returned nil grant")
	}
	if grant.Username != "alice" || grant.Credits != 100 || grant.SourceID != order.TradeNo || grant.AmountCents != 1000 {
		t.Fatalf("grant = %+v", grant)
	}

	updated, grant, err = store.MarkSuccess(order.TradeNo, "E20260626001", paidAt.Add(time.Minute))
	if err != nil {
		t.Fatalf("MarkSuccess(second) error = %v", err)
	}
	if updated.Status != TopUpStatusSuccess {
		t.Fatalf("second updated order = %+v", updated)
	}
	if grant != nil {
		t.Fatalf("second MarkSuccess() returned duplicate grant: %+v", grant)
	}
}

func TestStoreMarkFailedRejectsLaterSuccess(t *testing.T) {
	store, err := NewStoreWithClock(filepath.Join(t.TempDir(), "topups.json"), func() time.Time {
		return time.Date(2026, 6, 26, 2, 0, 0, 0, time.UTC)
	})
	if err != nil {
		t.Fatalf("NewStoreWithClock() error = %v", err)
	}
	order, err := store.CreateOrder(CreateOrderInput{
		Username:    "alice",
		Credits:     10,
		AmountCents: 100,
		Method:      "alipay",
	})
	if err != nil {
		t.Fatalf("CreateOrder() error = %v", err)
	}
	if _, changed, err := store.MarkFailed(order.TradeNo, "EFAILED", time.Time{}); err != nil || !changed {
		t.Fatalf("MarkFailed() changed=%v error=%v", changed, err)
	}
	if _, grant, err := store.MarkSuccess(order.TradeNo, "EFAILED", time.Time{}); !errors.Is(err, ErrOrderStatusInvalid) || grant != nil {
		t.Fatalf("MarkSuccess(after failed) grant=%+v error=%v", grant, err)
	}
}
