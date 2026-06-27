package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/y08lin4/lyra-image-workbench/internal/billing"
	"github.com/y08lin4/lyra-image-workbench/internal/settings"
)

func TestAdminConfigDoesNotLeakEpayKey(t *testing.T) {
	router := newTestRouter(t)
	adminToken := createAdminToken(t, router)
	rawKey := "epay-secret-1234567890"

	body := doAdminJSON(t, router, http.MethodPost, "/api/admin/config", adminToken, map[string]any{
		"epayEnabled":           true,
		"epayApiUrl":            "https://pay.example.com/submit.php",
		"epayPid":               "1001",
		"epayKey":               rawKey,
		"epayMethods":           []string{"alipay", "wxpay"},
		"creditPriceCents":      10,
		"minTopUpCredits":       10,
		"referralRewardCredits": 3,
	})
	assertEpayKeyHidden(t, body, rawKey)
	if !strings.Contains(body, `"epayKeySet":true`) || !strings.Contains(body, `"epayKeyPreview":"epay********7890"`) {
		t.Fatalf("admin config response missing epay key status/preview: %s", body)
	}

	body = doAdminJSON(t, router, http.MethodGet, "/api/admin/config", adminToken, nil)
	assertEpayKeyHidden(t, body, rawKey)

	body = doAdminJSON(t, router, http.MethodPut, "/api/admin/config", adminToken, map[string]any{
		"epayEnabled":  false,
		"clearEpayKey": true,
	})
	if strings.Contains(body, rawKey) || !strings.Contains(body, `"epayKeySet":false`) {
		t.Fatalf("cleared epay key response invalid: %s", body)
	}
}

func TestBillingAuthMatrix(t *testing.T) {
	router := newTestRouter(t)
	for _, tc := range []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodGet, "/api/billing/topup/options", ""},
		{http.MethodPost, "/api/billing/epay/orders", `{"credits":10,"method":"alipay"}`},
		{http.MethodGet, "/api/billing/topups", ""},
	} {
		req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
		if tc.body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		res := httptest.NewRecorder()
		router.ServeHTTP(res, req)
		if res.Code != http.StatusUnauthorized || !strings.Contains(res.Body.String(), "USER_AUTH_REQUIRED") {
			t.Fatalf("%s %s without login code=%d body=%s", tc.method, tc.path, res.Code, res.Body.String())
		}
	}

	for _, method := range []string{http.MethodGet, http.MethodPost} {
		req := httptest.NewRequest(method, "/api/billing/epay/notify", nil)
		res := httptest.NewRecorder()
		router.ServeHTTP(res, req)
		if res.Code != http.StatusOK || strings.TrimSpace(res.Body.String()) != "fail" {
			t.Fatalf("%s notify without login code=%d body=%s", method, res.Code, res.Body.String())
		}
	}
}

func TestBillingDisabledRejectsOrder(t *testing.T) {
	router := newTestRouter(t)
	token := createTestSession(t, router)

	code, body := doJSONStatus(t, router, http.MethodPost, "/api/billing/epay/orders", token, map[string]any{
		"credits": 10,
		"method":  "alipay",
	}, "")
	if code != http.StatusBadRequest || !strings.Contains(body, "BILLING_DISABLED") {
		t.Fatalf("disabled billing order code=%d body=%s", code, body)
	}
}

func TestEpayNotifyValidatesPidMethodAndIsIdempotent(t *testing.T) {
	env := newTestAPIEnv(t)
	key := configureTestBilling(t, env)
	token := createNamedUserSession(t, env.Router, "buyer01", "R7!Buyer#Vault$2026", "")

	tradeNo := createTestEpayOrder(t, env.Router, token, 10, "alipay")
	order, ok := env.Billing.GetByTradeNo(tradeNo)
	if !ok {
		t.Fatalf("created order %s missing", tradeNo)
	}

	badPID := signedNotifyValues(order, key)
	badPID.Set("pid", "bad-pid")
	badPID.Set("sign", billing.SignParams(valuesToMap(badPID), key))
	code, body := postNotifyForm(t, env.Router, badPID)
	if code != http.StatusOK || strings.TrimSpace(body) != "fail" {
		t.Fatalf("bad pid notify code=%d body=%s", code, body)
	}
	ledger, err := env.Users.ListCreditLedger("buyer01")
	if err != nil {
		t.Fatalf("ListCreditLedger() error = %v", err)
	}
	if len(ledger) != 0 {
		t.Fatalf("bad pid should not grant credits: %+v", ledger)
	}

	badMethod := signedNotifyValues(order, key)
	badMethod.Set("type", "wxpay")
	badMethod.Set("sign", billing.SignParams(valuesToMap(badMethod), key))
	code, body = postNotifyForm(t, env.Router, badMethod)
	if code != http.StatusOK || strings.TrimSpace(body) != "fail" {
		t.Fatalf("bad method notify code=%d body=%s", code, body)
	}

	valid := signedNotifyValues(order, key)
	for i := 0; i < 2; i++ {
		code, body = postNotifyForm(t, env.Router, valid)
		if code != http.StatusOK || strings.TrimSpace(body) != "success" {
			t.Fatalf("valid notify #%d code=%d body=%s", i+1, code, body)
		}
	}

	ledger, err = env.Users.ListCreditLedger("buyer01")
	if err != nil {
		t.Fatalf("ListCreditLedger(after valid) error = %v", err)
	}
	if len(ledger) != 1 || ledger[0].Delta != 10 || ledger[0].SourceID != tradeNo {
		t.Fatalf("duplicate notify should grant once, ledger=%+v", ledger)
	}
	profile, err := env.Users.Profile("buyer01")
	if err != nil {
		t.Fatalf("Profile() error = %v", err)
	}
	if profile.CreditsBalance != 10 {
		t.Fatalf("creditsBalance=%d, want 10", profile.CreditsBalance)
	}
	updated, ok := env.Billing.GetByTradeNo(tradeNo)
	if !ok || updated.Status != billing.TopUpStatusSuccess {
		t.Fatalf("order after notify = %+v ok=%v", updated, ok)
	}
}

func TestEpayNotifyRejectsBadSignatureAndAmount(t *testing.T) {
	env := newTestAPIEnv(t)
	key := configureTestBilling(t, env)
	token := createNamedUserSession(t, env.Router, "buyer02", "R7!Buyer#Vault$2026", "")

	tradeNo := createTestEpayOrder(t, env.Router, token, 10, "alipay")
	order, ok := env.Billing.GetByTradeNo(tradeNo)
	if !ok {
		t.Fatalf("created order %s missing", tradeNo)
	}

	badSignature := signedNotifyValues(order, key)
	badSignature.Set("sign", "bad-signature")
	code, body := postNotifyForm(t, env.Router, badSignature)
	if code != http.StatusOK || strings.TrimSpace(body) != "fail" {
		t.Fatalf("bad signature notify code=%d body=%s", code, body)
	}

	badAmount := signedNotifyValues(order, key)
	badAmount.Set("money", billing.FormatCents(order.AmountCents+1))
	badAmount.Set("sign", billing.SignParams(valuesToMap(badAmount), key))
	code, body = postNotifyForm(t, env.Router, badAmount)
	if code != http.StatusOK || strings.TrimSpace(body) != "fail" {
		t.Fatalf("bad amount notify code=%d body=%s", code, body)
	}

	ledger, err := env.Users.ListCreditLedger("buyer02")
	if err != nil {
		t.Fatalf("ListCreditLedger() error = %v", err)
	}
	if len(ledger) != 0 {
		t.Fatalf("invalid notify should not grant credits: %+v", ledger)
	}
	updated, ok := env.Billing.GetByTradeNo(tradeNo)
	if !ok || updated.Status != billing.TopUpStatusPending {
		t.Fatalf("invalid notify should leave order pending, order=%+v ok=%v", updated, ok)
	}
}

func TestEpayNotifyRewardsInviterOnce(t *testing.T) {
	env := newTestAPIEnv(t)
	key := configureTestBilling(t, env)
	reward := 4
	if _, err := env.Settings.Update(settings.Update{ReferralRewardCredits: &reward}); err != nil {
		t.Fatalf("Settings.Update(referral reward) error = %v", err)
	}

	createNamedUserSession(t, env.Router, "inviter01", "R7!Invite#Vault$2026", "")
	inviter, err := env.Users.Profile("inviter01")
	if err != nil {
		t.Fatalf("Profile(inviter01) error = %v", err)
	}
	if inviter.ReferralCode == "" {
		t.Fatal("inviter referral code is empty")
	}

	_, cookies := doJSONWithCookies(t, env.Router, http.MethodPost, "/api/users/register", "", map[string]string{
		"username":     "buyer03",
		"password":     "R7!Buyer#Vault$2026",
		"referralCode": inviter.ReferralCode,
	})
	token := userSessionFromCookies(t, cookies)
	tradeNo := createTestEpayOrder(t, env.Router, token, 10, "alipay")
	order, ok := env.Billing.GetByTradeNo(tradeNo)
	if !ok {
		t.Fatalf("created order %s missing", tradeNo)
	}

	valid := signedNotifyValues(order, key)
	for i := 0; i < 2; i++ {
		code, body := postNotifyForm(t, env.Router, valid)
		if code != http.StatusOK || strings.TrimSpace(body) != "success" {
			t.Fatalf("valid notify #%d code=%d body=%s", i+1, code, body)
		}
	}

	buyerLedger, err := env.Users.ListCreditLedger("buyer03")
	if err != nil {
		t.Fatalf("buyer ListCreditLedger() error = %v", err)
	}
	if len(buyerLedger) != 1 || buyerLedger[0].Type != "purchase" || buyerLedger[0].Delta != 10 || buyerLedger[0].SourceID != tradeNo {
		t.Fatalf("buyer purchase ledger mismatch: %+v", buyerLedger)
	}
	inviterLedger, err := env.Users.ListCreditLedger("inviter01")
	if err != nil {
		t.Fatalf("inviter ListCreditLedger() error = %v", err)
	}
	if len(inviterLedger) != 1 || inviterLedger[0].Type != "referral_reward" || inviterLedger[0].Delta != reward || inviterLedger[0].SourceID != "referral:"+tradeNo {
		t.Fatalf("inviter reward ledger mismatch: %+v", inviterLedger)
	}
	inviter, err = env.Users.Profile("inviter01")
	if err != nil {
		t.Fatalf("Profile(inviter01 after notify) error = %v", err)
	}
	if inviter.CreditsBalance != reward {
		t.Fatalf("inviter creditsBalance=%d, want %d", inviter.CreditsBalance, reward)
	}
}

func TestAdminConfigDoesNotLeakSMTPPassword(t *testing.T) {
	router := newTestRouter(t)
	adminToken := createAdminToken(t, router)
	rawPassword := "smtp-secret-1234567890"

	body := doAdminJSON(t, router, http.MethodPost, "/api/admin/config", adminToken, map[string]any{
		"smtpEnabled":  true,
		"smtpHost":     "smtp.example.com",
		"smtpPort":     587,
		"smtpUser":     "noreply@example.com",
		"smtpPassword": rawPassword,
		"smtpFrom":     "noreply@example.com",
	})
	assertSMTPPasswordHidden(t, body, rawPassword)
	if !strings.Contains(body, `"smtpPasswordSet":true`) || !strings.Contains(body, `"smtpPasswordPreview":"smtp********7890"`) {
		t.Fatalf("admin config response missing smtp password status/preview: %s", body)
	}

	body = doAdminJSON(t, router, http.MethodGet, "/api/admin/config", adminToken, nil)
	assertSMTPPasswordHidden(t, body, rawPassword)

	body = doAdminJSON(t, router, http.MethodPut, "/api/admin/config", adminToken, map[string]any{
		"smtpEnabled":       false,
		"clearSmtpPassword": true,
	})
	if strings.Contains(body, rawPassword) || !strings.Contains(body, `"smtpPasswordSet":false`) {
		t.Fatalf("cleared smtp password response invalid: %s", body)
	}
}

func TestEpayOrderStatusQuery(t *testing.T) {
	env := newTestAPIEnv(t)
	key := configureTestBilling(t, env)
	token := createNamedUserSession(t, env.Router, "buyer04", "R7!Buyer#Vault$2026", "")

	tradeNo := createTestEpayOrder(t, env.Router, token, 10, "alipay")
	body := doJSON(t, env.Router, http.MethodGet, "/api/billing/epay/orders/"+tradeNo, token, nil)
	if !strings.Contains(body, `"status":"pending"`) || !strings.Contains(body, `"payUrl"`) {
		t.Fatalf("pending order status response invalid: %s", body)
	}

	order, ok := env.Billing.GetByTradeNo(tradeNo)
	if !ok {
		t.Fatalf("created order %s missing", tradeNo)
	}
	code, notifyBody := postNotifyForm(t, env.Router, signedNotifyValues(order, key))
	if code != http.StatusOK || strings.TrimSpace(notifyBody) != "success" {
		t.Fatalf("valid notify code=%d body=%s", code, notifyBody)
	}

	body = doJSON(t, env.Router, http.MethodGet, "/api/billing/epay/orders?tradeNo="+url.QueryEscape(tradeNo), token, nil)
	if !strings.Contains(body, `"status":"success"`) || !strings.Contains(body, `"providerTradeNo":"E202606260001"`) {
		t.Fatalf("success order status response invalid: %s", body)
	}

	otherToken := createNamedUserSession(t, env.Router, "buyer05", "R7!Buyer#Vault$2026", "")
	status, otherBody := doJSONStatus(t, env.Router, http.MethodGet, "/api/billing/epay/orders/"+tradeNo, otherToken, nil, "")
	if status != http.StatusNotFound || !strings.Contains(otherBody, "TOPUP_ORDER_NOT_FOUND") {
		t.Fatalf("other user order lookup status=%d body=%s", status, otherBody)
	}
}

func assertSMTPPasswordHidden(t *testing.T, body string, rawPassword string) {
	t.Helper()
	if strings.Contains(body, rawPassword) || strings.Contains(body, `"smtpPassword":`) || strings.Contains(body, `"smtpPass":`) {
		t.Fatalf("admin config leaked smtp password: %s", body)
	}
}

func assertEpayKeyHidden(t *testing.T, body string, rawKey string) {
	t.Helper()
	if strings.Contains(body, rawKey) || strings.Contains(body, `"epayKey":`) {
		t.Fatalf("admin config leaked epay key: %s", body)
	}
}

func configureTestBilling(t *testing.T, env testAPIEnv) string {
	t.Helper()
	enabled := true
	apiURL := "https://pay.example.com/submit.php"
	pid := "1001"
	key := "secret-key"
	price := 10
	minimum := 10
	reward := 0
	if _, err := env.Settings.Update(settings.Update{
		EpayEnabled:           &enabled,
		EpayAPIURL:            &apiURL,
		EpayPID:               &pid,
		EpayKey:               &key,
		EpayMethods:           []string{"alipay"},
		CreditPriceCents:      &price,
		MinTopUpCredits:       &minimum,
		ReferralRewardCredits: &reward,
	}); err != nil {
		t.Fatalf("Settings.Update(epay) error = %v", err)
	}
	return key
}

func createTestEpayOrder(t *testing.T, router http.Handler, token string, credits int, method string) string {
	t.Helper()
	body := doJSON(t, router, http.MethodPost, "/api/billing/epay/orders", token, map[string]any{"credits": credits, "method": method})
	var payload struct {
		TradeNo string `json:"tradeNo"`
		Order   struct {
			TradeNo string `json:"tradeNo"`
		} `json:"order"`
	}
	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		t.Fatalf("decode epay order response: %v body=%s", err, body)
	}
	if payload.TradeNo != "" {
		return payload.TradeNo
	}
	if payload.Order.TradeNo != "" {
		return payload.Order.TradeNo
	}
	t.Fatalf("tradeNo missing from response: %s", body)
	return ""
}

func signedNotifyValues(order billing.TopUpOrder, key string) url.Values {
	params := map[string]string{
		"pid":          "1001",
		"type":         order.Method,
		"out_trade_no": order.TradeNo,
		"trade_no":     "E202606260001",
		"trade_status": billing.EpayStatusTradeSuccess,
		"money":        billing.FormatCents(order.AmountCents),
	}
	params["sign"] = billing.SignParams(params, key)
	params["sign_type"] = billing.EpaySignTypeMD5
	values := url.Values{}
	for k, v := range params {
		values.Set(k, v)
	}
	return values
}

func valuesToMap(values url.Values) map[string]string {
	params := make(map[string]string, len(values))
	for key, item := range values {
		if len(item) > 0 {
			params[key] = item[0]
		}
	}
	return params
}

func postNotifyForm(t *testing.T, router http.Handler, values url.Values) (int, string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/billing/epay/notify", strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	return res.Code, res.Body.String()
}
