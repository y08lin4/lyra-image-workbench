package users

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/subtle"
	"encoding/base32"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	totpIssuer = "Lyra Image Workbench"
	totpPeriod = int64(30)
)

type TOTPSetup struct {
	Secret     string `json:"secret"`
	OtpauthURL string `json:"otpauthUrl"`
}

func newTOTPSecret() (string, error) {
	var data [20]byte
	if _, err := rand.Read(data[:]); err != nil {
		return "", err
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(data[:]), nil
}

func setupFromSecret(username string, secret string) TOTPSetup {
	label := totpIssuer + ":" + username
	values := url.Values{}
	values.Set("secret", secret)
	values.Set("issuer", totpIssuer)
	values.Set("algorithm", "SHA1")
	values.Set("digits", "6")
	values.Set("period", strconv.FormatInt(totpPeriod, 10))
	return TOTPSetup{
		Secret:     secret,
		OtpauthURL: "otpauth://totp/" + url.PathEscape(label) + "?" + values.Encode(),
	}
}

func verifyTOTP(secret string, code string, now time.Time) bool {
	code = strings.TrimSpace(code)
	if len(code) != 6 {
		return false
	}
	for _, item := range code {
		if item < '0' || item > '9' {
			return false
		}
	}
	for offset := int64(-1); offset <= 1; offset++ {
		expected, err := totpCode(secret, now.Unix()/totpPeriod+offset)
		if err != nil {
			return false
		}
		if subtle.ConstantTimeCompare([]byte(expected), []byte(code)) == 1 {
			return true
		}
	}
	return false
}

func totpCode(secret string, counter int64) (string, error) {
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(strings.TrimSpace(secret)))
	if err != nil {
		return "", err
	}
	var msg [8]byte
	for i := 7; i >= 0; i-- {
		msg[i] = byte(counter)
		counter >>= 8
	}
	mac := hmac.New(sha1.New, key)
	if _, err := mac.Write(msg[:]); err != nil {
		return "", err
	}
	sum := mac.Sum(nil)
	offset := sum[len(sum)-1] & 0x0f
	value := (int(sum[offset])&0x7f)<<24 |
		(int(sum[offset+1])&0xff)<<16 |
		(int(sum[offset+2])&0xff)<<8 |
		(int(sum[offset+3]) & 0xff)
	return fmt.Sprintf("%06d", value%1000000), nil
}
