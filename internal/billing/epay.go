package billing

import (
	"crypto/md5"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

const (
	EpaySignTypeMD5        = "MD5"
	EpayStatusTradeSuccess = "TRADE_SUCCESS"
)

var (
	ErrInvalidEpayConfig      = errors.New("invalid epay config")
	ErrInvalidEpaySignature   = errors.New("invalid epay signature")
	ErrEpayTradeNoMismatch    = errors.New("epay trade number mismatch")
	ErrEpayAmountMismatch     = errors.New("epay amount mismatch")
	ErrEpayTradeStatusInvalid = errors.New("epay trade status invalid")
)

type EpayConfig struct {
	APIURL    string
	PID       string
	Key       string
	NotifyURL string
	ReturnURL string
	SiteName  string
}

type EpayCallback struct {
	TradeNo         string
	ProviderTradeNo string
	Method          string
	TradeStatus     string
	AmountCents     int64
}

func SignPayload(params map[string]string, key string) string {
	keys := make([]string, 0, len(params))
	for k, v := range params {
		if k == "" || v == "" || strings.EqualFold(k, "sign") || strings.EqualFold(k, "sign_type") {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+params[k])
	}
	return strings.Join(parts, "&") + strings.TrimSpace(key)
}

func SignParams(params map[string]string, key string) string {
	sum := md5.Sum([]byte(SignPayload(params, key)))
	return hex.EncodeToString(sum[:])
}

func BuildEpayPaymentURL(cfg EpayConfig, order TopUpOrder) (string, error) {
	cfg = normalizeEpayConfig(cfg)
	if cfg.APIURL == "" || cfg.PID == "" || cfg.Key == "" || cfg.NotifyURL == "" {
		return "", ErrInvalidEpayConfig
	}
	if order.TradeNo == "" || order.Credits <= 0 || order.AmountCents <= 0 || order.Method == "" {
		return "", ErrInvalidOrder
	}
	if order.Status != TopUpStatusPending {
		return "", ErrOrderStatusInvalid
	}

	endpoint, err := url.Parse(cfg.APIURL)
	if err != nil || endpoint.Scheme == "" || endpoint.Host == "" {
		return "", ErrInvalidEpayConfig
	}

	params := map[string]string{
		"pid":          cfg.PID,
		"type":         order.Method,
		"out_trade_no": order.TradeNo,
		"notify_url":   cfg.NotifyURL,
		"name":         fmt.Sprintf("Lyra credits x %d", order.Credits),
		"money":        FormatCents(order.AmountCents),
	}
	if cfg.ReturnURL != "" {
		params["return_url"] = cfg.ReturnURL
	}
	if cfg.SiteName != "" {
		params["sitename"] = cfg.SiteName
	}

	values := endpoint.Query()
	for k, v := range params {
		values.Set(k, v)
	}
	values.Set("sign", SignParams(params, cfg.Key))
	values.Set("sign_type", EpaySignTypeMD5)
	endpoint.RawQuery = values.Encode()
	return endpoint.String(), nil
}

func ValidateEpayCallback(params map[string]string, key string, order TopUpOrder) (EpayCallback, error) {
	if strings.TrimSpace(key) == "" || strings.TrimSpace(params["sign"]) == "" {
		return EpayCallback{}, ErrInvalidEpaySignature
	}
	expected := SignParams(params, key)
	received := strings.ToLower(strings.TrimSpace(params["sign"]))
	if subtle.ConstantTimeCompare([]byte(received), []byte(expected)) != 1 {
		return EpayCallback{}, ErrInvalidEpaySignature
	}

	tradeNo := strings.TrimSpace(params["out_trade_no"])
	if tradeNo == "" || tradeNo != order.TradeNo {
		return EpayCallback{}, ErrEpayTradeNoMismatch
	}

	status := strings.TrimSpace(params["trade_status"])
	if status != EpayStatusTradeSuccess {
		return EpayCallback{}, ErrEpayTradeStatusInvalid
	}

	amountCents, err := ParseMoneyCents(params["money"])
	if err != nil || amountCents != order.AmountCents {
		return EpayCallback{}, ErrEpayAmountMismatch
	}

	return EpayCallback{
		TradeNo:         tradeNo,
		ProviderTradeNo: strings.TrimSpace(params["trade_no"]),
		Method:          strings.TrimSpace(params["type"]),
		TradeStatus:     status,
		AmountCents:     amountCents,
	}, nil
}

func FormatCents(cents int64) string {
	sign := ""
	if cents < 0 {
		sign = "-"
		cents = -cents
	}
	return fmt.Sprintf("%s%d.%02d", sign, cents/100, cents%100)
}

func ParseMoneyCents(raw string) (int64, error) {
	value := strings.TrimSpace(raw)
	if value == "" || strings.HasPrefix(value, "-") || strings.HasPrefix(value, "+") {
		return 0, strconv.ErrSyntax
	}
	parts := strings.Split(value, ".")
	if len(parts) > 2 || parts[0] == "" || !isDigits(parts[0]) {
		return 0, strconv.ErrSyntax
	}
	fraction := "00"
	if len(parts) == 2 {
		if len(parts[1]) > 2 || !isDigits(parts[1]) {
			return 0, strconv.ErrSyntax
		}
		fraction = parts[1]
		if len(fraction) == 0 {
			fraction = "00"
		}
		if len(fraction) == 1 {
			fraction += "0"
		}
	}
	units, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, err
	}
	cents, err := strconv.ParseInt(fraction, 10, 64)
	if err != nil {
		return 0, err
	}
	return units*100 + cents, nil
}

func normalizeEpayConfig(cfg EpayConfig) EpayConfig {
	return EpayConfig{
		APIURL:    strings.TrimSpace(cfg.APIURL),
		PID:       strings.TrimSpace(cfg.PID),
		Key:       strings.TrimSpace(cfg.Key),
		NotifyURL: strings.TrimSpace(cfg.NotifyURL),
		ReturnURL: strings.TrimSpace(cfg.ReturnURL),
		SiteName:  strings.TrimSpace(cfg.SiteName),
	}
}

func isDigits(value string) bool {
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
