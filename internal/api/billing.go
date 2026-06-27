package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/y08lin4/lyra-image-workbench/internal/billing"
	"github.com/y08lin4/lyra-image-workbench/internal/settings"
	"github.com/y08lin4/lyra-image-workbench/internal/users"
)

type BillingHandler struct {
	settings *settings.FileStore
	topups   *billing.Store
	users    *users.Store
}

type topUpOptionResponse struct {
	Credits     int      `json:"credits"`
	AmountCents int64    `json:"amountCents"`
	Label       string   `json:"label,omitempty"`
	Methods     []string `json:"methods,omitempty"`
}

type createEpayOrderRequest struct {
	Credits int    `json:"credits"`
	Method  string `json:"method"`
}

type billingTopUpResponse struct {
	TradeNo           string              `json:"tradeNo"`
	PayURL            string              `json:"payUrl,omitempty"`
	Credits           int                 `json:"credits"`
	AmountCents       int64               `json:"amountCents"`
	Status            billing.TopUpStatus `json:"status"`
	Method            string              `json:"method,omitempty"`
	CreatedAt         string              `json:"createdAt"`
	PaidAt            string              `json:"paidAt,omitempty"`
	ThirdPartyTradeNo string              `json:"thirdPartyTradeNo,omitempty"`
	ProviderTradeNo   string              `json:"providerTradeNo,omitempty"`
}

func NewBillingHandler(settingsStore *settings.FileStore, topups *billing.Store, userStore *users.Store) BillingHandler {
	return BillingHandler{settings: settingsStore, topups: topups, users: userStore}
}

func (h BillingHandler) Options(w http.ResponseWriter, r *http.Request) {
	cfg, ok := h.currentConfig(w)
	if !ok {
		return
	}
	options := buildTopUpOptions(cfg)
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":           true,
		"enabled":      cfg.EpayEnabled,
		"methods":      append([]string{}, cfg.EpayMethods...),
		"options":      options,
		"topupOptions": options,
	})
}

func (h BillingHandler) CreateEpayOrder(w http.ResponseWriter, r *http.Request) {
	if h.topups == nil {
		writeError(w, http.StatusServiceUnavailable, "BILLING_STORE_UNAVAILABLE", "充值订单服务未初始化")
		return
	}
	cfg, ok := h.currentConfig(w)
	if !ok {
		return
	}
	if !cfg.EpayEnabled {
		writeError(w, http.StatusBadRequest, "BILLING_DISABLED", "在线充值暂未开启")
		return
	}
	if err := validateBillingConfig(cfg); err != nil {
		writeError(w, http.StatusBadRequest, "BILLING_CONFIG_INVALID", err.Error())
		return
	}
	defer r.Body.Close()
	var payload createEpayOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_JSON", "请求体不是有效 JSON")
		return
	}
	method := strings.ToLower(strings.TrimSpace(payload.Method))
	if payload.Credits < cfg.MinTopUpCredits {
		writeError(w, http.StatusBadRequest, "TOPUP_CREDITS_TOO_LOW", fmt.Sprintf("充值次数不能低于 %d", cfg.MinTopUpCredits))
		return
	}
	if !epayMethodAllowed(method, cfg.EpayMethods) {
		writeError(w, http.StatusBadRequest, "EPAY_METHOD_UNSUPPORTED", "不支持该支付方式")
		return
	}
	username := strings.TrimSpace(r.Header.Get("X-User-Name"))
	if username == "" {
		if session, ok := currentUserSession(h.users, r); ok {
			username = session.User.Username
		}
	}
	if username == "" {
		writeError(w, http.StatusUnauthorized, "USER_AUTH_REQUIRED", "请先登录")
		return
	}
	amountCents := int64(payload.Credits) * int64(cfg.CreditPriceCents)
	order, err := h.topups.CreateOrder(billing.CreateOrderInput{
		Username:    username,
		Credits:     payload.Credits,
		AmountCents: amountCents,
		Method:      method,
	})
	if err != nil {
		writeBillingStoreError(w, err)
		return
	}
	payURL, err := billing.BuildEpayPaymentURL(epayConfigFromRequest(r, cfg), order)
	if err != nil {
		writeBillingStoreError(w, err)
		return
	}
	response := topUpResponse(order)
	response.PayURL = payURL
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":          true,
		"order":       response,
		"tradeNo":     response.TradeNo,
		"payUrl":      response.PayURL,
		"credits":     response.Credits,
		"amountCents": response.AmountCents,
		"status":      response.Status,
	})
}

func (h BillingHandler) GetEpayOrder(w http.ResponseWriter, r *http.Request) {
	if h.topups == nil {
		writeError(w, http.StatusServiceUnavailable, "BILLING_STORE_UNAVAILABLE", "充值订单服务未初始化")
		return
	}
	username := strings.TrimSpace(r.Header.Get("X-User-Name"))
	if username == "" {
		if session, ok := currentUserSession(h.users, r); ok {
			username = session.User.Username
		}
	}
	if username == "" {
		writeError(w, http.StatusUnauthorized, "USER_AUTH_REQUIRED", "请先登录")
		return
	}
	tradeNo := strings.TrimSpace(r.PathValue("tradeNo"))
	if tradeNo == "" {
		tradeNo = strings.TrimSpace(r.URL.Query().Get("tradeNo"))
	}
	if tradeNo == "" {
		writeError(w, http.StatusBadRequest, "TOPUP_TRADE_NO_REQUIRED", "缺少充值订单号")
		return
	}
	order, found := h.topups.GetByTradeNo(tradeNo)
	if !found || order.Username != username {
		writeError(w, http.StatusNotFound, "TOPUP_ORDER_NOT_FOUND", "充值订单不存在")
		return
	}
	response := topUpResponse(order)
	if order.Status == billing.TopUpStatusPending && h.settings != nil {
		cfg := h.settings.Get()
		if cfg.EpayEnabled && validateBillingConfig(cfg) == nil && epayMethodAllowed(order.Method, cfg.EpayMethods) {
			if payURL, err := billing.BuildEpayPaymentURL(epayConfigFromRequest(r, cfg), order); err == nil {
				response.PayURL = payURL
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "order": response, "tradeNo": response.TradeNo, "status": response.Status})
}

func (h BillingHandler) Notify(w http.ResponseWriter, r *http.Request) {
	if h.topups == nil || h.users == nil {
		writeEpayNotify(w, false)
		return
	}
	cfg, ok := h.currentConfigForNotify()
	if !ok {
		writeEpayNotify(w, false)
		return
	}
	params, err := epayNotifyParams(r)
	if err != nil {
		writeEpayNotify(w, false)
		return
	}
	tradeNo := strings.TrimSpace(params["out_trade_no"])
	order, found := h.topups.GetByTradeNo(tradeNo)
	if !found {
		writeEpayNotify(w, false)
		return
	}
	callback, err := billing.ValidateEpayCallback(params, cfg.EpayKey, order)
	if err != nil {
		writeEpayNotify(w, false)
		return
	}
	if strings.TrimSpace(params["pid"]) != cfg.EpayPID || strings.TrimSpace(params["type"]) != order.Method || callback.Method != order.Method {
		writeEpayNotify(w, false)
		return
	}
	if order.Status == billing.TopUpStatusFailed {
		writeEpayNotify(w, false)
		return
	}
	if order.Status != billing.TopUpStatusSuccess {
		if _, _, err := h.topups.MarkSuccess(order.TradeNo, callback.ProviderTradeNo, time.Now()); err != nil {
			if errors.Is(err, billing.ErrOrderStatusInvalid) {
				if latest, found := h.topups.GetByTradeNo(order.TradeNo); found && latest.Status == billing.TopUpStatusSuccess {
					order = latest
				} else {
					writeEpayNotify(w, false)
					return
				}
			} else {
				writeEpayNotify(w, false)
				return
			}
		}
	}
	if _, err := h.users.AddPurchaseCredits(order.Username, order.Credits, order.TradeNo, cfg.ReferralRewardCredits); err != nil {
		writeEpayNotify(w, false)
		return
	}
	writeEpayNotify(w, true)
}

func (h BillingHandler) ListTopUps(w http.ResponseWriter, r *http.Request) {
	if h.topups == nil {
		writeError(w, http.StatusServiceUnavailable, "BILLING_STORE_UNAVAILABLE", "充值订单服务未初始化")
		return
	}
	username := strings.TrimSpace(r.Header.Get("X-User-Name"))
	if username == "" {
		if session, ok := currentUserSession(h.users, r); ok {
			username = session.User.Username
		}
	}
	if username == "" {
		writeError(w, http.StatusUnauthorized, "USER_AUTH_REQUIRED", "请先登录")
		return
	}
	orders := h.topups.ListByUsername(username)
	items := make([]billingTopUpResponse, 0, len(orders))
	for _, order := range orders {
		items = append(items, topUpResponse(order))
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "topups": items, "orders": items})
}

func (h BillingHandler) currentConfig(w http.ResponseWriter) (settings.RuntimeConfig, bool) {
	if h.settings == nil {
		writeError(w, http.StatusServiceUnavailable, "SETTINGS_UNAVAILABLE", "系统设置服务未初始化")
		return settings.RuntimeConfig{}, false
	}
	return h.settings.Get(), true
}

func (h BillingHandler) currentConfigForNotify() (settings.RuntimeConfig, bool) {
	if h.settings == nil {
		return settings.RuntimeConfig{}, false
	}
	cfg := h.settings.Get()
	if strings.TrimSpace(cfg.EpayPID) == "" || strings.TrimSpace(cfg.EpayKey) == "" {
		return settings.RuntimeConfig{}, false
	}
	return cfg, true
}

func validateBillingConfig(cfg settings.RuntimeConfig) error {
	if strings.TrimSpace(cfg.EpayAPIURL) == "" || strings.TrimSpace(cfg.EpayPID) == "" || strings.TrimSpace(cfg.EpayKey) == "" {
		return errors.New("易支付网关地址、商户 PID 和商户 Key 未配置完整")
	}
	if cfg.CreditPriceCents <= 0 || cfg.MinTopUpCredits <= 0 {
		return errors.New("次数单价和最小充值次数必须大于 0")
	}
	if len(cfg.EpayMethods) == 0 {
		return errors.New("至少需要配置一种支付方式")
	}
	return nil
}

func buildTopUpOptions(cfg settings.RuntimeConfig) []topUpOptionResponse {
	credits := []int{cfg.MinTopUpCredits, cfg.MinTopUpCredits * 5, cfg.MinTopUpCredits * 10}
	seen := make(map[int]bool)
	options := make([]topUpOptionResponse, 0, len(credits))
	for _, credit := range credits {
		if credit <= 0 || seen[credit] {
			continue
		}
		seen[credit] = true
		options = append(options, topUpOptionResponse{
			Credits:     credit,
			AmountCents: int64(credit) * int64(cfg.CreditPriceCents),
			Label:       fmt.Sprintf("%d 次", credit),
			Methods:     append([]string{}, cfg.EpayMethods...),
		})
	}
	return options
}

func epayMethodAllowed(method string, methods []string) bool {
	for _, item := range methods {
		if method == strings.ToLower(strings.TrimSpace(item)) {
			return true
		}
	}
	return false
}

func epayConfigFromRequest(r *http.Request, cfg settings.RuntimeConfig) billing.EpayConfig {
	baseURL := strings.TrimRight(cfg.PublicBaseURL, "/")
	if baseURL == "" {
		baseURL = requestBaseURL(r)
	}
	siteName := strings.TrimSpace(cfg.SiteName)
	if siteName == "" {
		siteName = settings.DefaultSiteName
	}
	return billing.EpayConfig{
		APIURL:    cfg.EpayAPIURL,
		PID:       cfg.EpayPID,
		Key:       cfg.EpayKey,
		NotifyURL: baseURL + "/api/billing/epay/notify",
		ReturnURL: baseURL + "/profile",
		SiteName:  siteName,
	}
}

func requestBaseURL(r *http.Request) string {
	scheme := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto"))
	if scheme == "" {
		if r.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}
	host := strings.TrimSpace(r.Header.Get("X-Forwarded-Host"))
	if host == "" {
		host = r.Host
	}
	return strings.TrimRight(scheme+"://"+host, "/")
}

func epayNotifyParams(r *http.Request) (map[string]string, error) {
	if err := r.ParseForm(); err != nil {
		return nil, err
	}
	params := make(map[string]string, len(r.Form))
	for key, values := range r.Form {
		if len(values) == 0 {
			continue
		}
		params[key] = strings.TrimSpace(values[0])
	}
	return params, nil
}

func topUpResponse(order billing.TopUpOrder) billingTopUpResponse {
	response := billingTopUpResponse{
		TradeNo:           order.TradeNo,
		Credits:           order.Credits,
		AmountCents:       order.AmountCents,
		Status:            order.Status,
		Method:            order.Method,
		CreatedAt:         formatBillingTime(order.CreatedAt),
		ThirdPartyTradeNo: order.ProviderTradeNo,
		ProviderTradeNo:   order.ProviderTradeNo,
	}
	if order.PaidAt != nil {
		response.PaidAt = formatBillingTime(*order.PaidAt)
	}
	return response
}

func formatBillingTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

func writeBillingStoreError(w http.ResponseWriter, err error) {
	status := http.StatusBadRequest
	code := "BILLING_ERROR"
	message := err.Error()
	switch {
	case errors.Is(err, billing.ErrInvalidOrder):
		code = "TOPUP_ORDER_INVALID"
		message = "充值订单参数无效"
	case errors.Is(err, billing.ErrOrderStatusInvalid):
		code = "TOPUP_ORDER_STATUS_INVALID"
		message = "充值订单状态不允许该操作"
	case errors.Is(err, billing.ErrInvalidEpayConfig):
		code = "BILLING_CONFIG_INVALID"
		message = "易支付配置无效"
	}
	writeError(w, status, code, message)
}

func writeEpayNotify(w http.ResponseWriter, ok bool) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	if ok {
		_, _ = w.Write([]byte("success"))
		return
	}
	_, _ = w.Write([]byte("fail"))
}
