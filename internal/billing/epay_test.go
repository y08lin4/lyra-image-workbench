package billing

import (
	"errors"
	"net/url"
	"testing"
	"time"
)

func TestSignParamsStable(t *testing.T) {
	params := map[string]string{
		"type":         "alipay",
		"pid":          "1001",
		"out_trade_no": "LYRA202606260001",
		"money":        "10.00",
		"name":         "Lyra credits x 100",
		"sign":         "ignored",
		"sign_type":    "MD5",
	}

	payload := SignPayload(params, "secret-key")
	if payload != "money=10.00&name=Lyra credits x 100&out_trade_no=LYRA202606260001&pid=1001&type=alipaysecret-key" {
		t.Fatalf("SignPayload() = %q", payload)
	}
	if got := SignParams(params, "secret-key"); got != "6f8347369c22ce56efccd3b3aec91138" {
		t.Fatalf("SignParams() = %q", got)
	}
}

func TestBuildEpayPaymentURL(t *testing.T) {
	order := TopUpOrder{
		TradeNo:     "LYRA202606260001",
		Username:    "alice",
		Credits:     100,
		AmountCents: 1000,
		Method:      "alipay",
		Status:      TopUpStatusPending,
		CreatedAt:   time.Date(2026, 6, 26, 0, 0, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2026, 6, 26, 0, 0, 0, 0, time.UTC),
	}
	payURL, err := BuildEpayPaymentURL(EpayConfig{
		APIURL:    "https://pay.example.com/submit.php",
		PID:       "1001",
		Key:       "secret-key",
		NotifyURL: "https://lyra.example.com/api/billing/epay/notify",
		ReturnURL: "https://lyra.example.com/profile",
		SiteName:  "Lyra",
	}, order)
	if err != nil {
		t.Fatalf("BuildEpayPaymentURL() error = %v", err)
	}

	parsed, err := url.Parse(payURL)
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}
	query := parsed.Query()
	if parsed.Scheme != "https" || parsed.Host != "pay.example.com" || parsed.Path != "/submit.php" {
		t.Fatalf("endpoint = %s", parsed.String())
	}
	for key, want := range map[string]string{
		"pid":          "1001",
		"type":         "alipay",
		"out_trade_no": "LYRA202606260001",
		"money":        "10.00",
		"name":         "Lyra credits x 100",
		"notify_url":   "https://lyra.example.com/api/billing/epay/notify",
		"return_url":   "https://lyra.example.com/profile",
		"sitename":     "Lyra",
		"sign_type":    EpaySignTypeMD5,
	} {
		if got := query.Get(key); got != want {
			t.Fatalf("query[%s] = %q, want %q", key, got, want)
		}
	}
	if query.Get("sign") == "" {
		t.Fatal("query sign is empty")
	}
}

func TestValidateEpayCallbackRejectsBadSignature(t *testing.T) {
	order := testOrder()
	params := validCallbackParams(order)
	params["sign"] = "bad-signature"

	if _, err := ValidateEpayCallback(params, "secret-key", order); !errors.Is(err, ErrInvalidEpaySignature) {
		t.Fatalf("ValidateEpayCallback() error = %v, want %v", err, ErrInvalidEpaySignature)
	}
}

func TestValidateEpayCallbackRejectsAmountMismatch(t *testing.T) {
	order := testOrder()
	params := validCallbackParams(order)
	params["money"] = "10.01"
	params["sign"] = SignParams(params, "secret-key")

	if _, err := ValidateEpayCallback(params, "secret-key", order); !errors.Is(err, ErrEpayAmountMismatch) {
		t.Fatalf("ValidateEpayCallback() error = %v, want %v", err, ErrEpayAmountMismatch)
	}
}

func TestValidateEpayCallbackAcceptsSignedSuccess(t *testing.T) {
	order := testOrder()
	params := validCallbackParams(order)

	callback, err := ValidateEpayCallback(params, "secret-key", order)
	if err != nil {
		t.Fatalf("ValidateEpayCallback() error = %v", err)
	}
	if callback.TradeNo != order.TradeNo || callback.ProviderTradeNo != "E202606260001" || callback.AmountCents != order.AmountCents {
		t.Fatalf("callback = %+v", callback)
	}
}

func testOrder() TopUpOrder {
	now := time.Date(2026, 6, 26, 0, 0, 0, 0, time.UTC)
	return TopUpOrder{
		TradeNo:     "LYRA202606260001",
		Username:    "alice",
		Credits:     100,
		AmountCents: 1000,
		Method:      "alipay",
		Status:      TopUpStatusPending,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

func validCallbackParams(order TopUpOrder) map[string]string {
	params := map[string]string{
		"pid":          "1001",
		"type":         order.Method,
		"out_trade_no": order.TradeNo,
		"trade_no":     "E202606260001",
		"trade_status": EpayStatusTradeSuccess,
		"money":        FormatCents(order.AmountCents),
	}
	params["sign"] = SignParams(params, "secret-key")
	params["sign_type"] = EpaySignTypeMD5
	return params
}
