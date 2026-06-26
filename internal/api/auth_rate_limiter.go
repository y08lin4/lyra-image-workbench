package api

import (
	"crypto/sha256"
	"encoding/hex"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type authRateLimiter struct {
	mu       sync.Mutex
	maxFails int
	window   time.Duration
	lockout  time.Duration
	entries  map[string]authRateEntry
}

type authRateEntry struct {
	firstFail time.Time
	fails     int
	blocked   time.Time
}

var defaultAuthRateLimiter = newAuthRateLimiter(20, 10*time.Minute, 2*time.Minute)

func newAuthRateLimiter(maxFails int, window time.Duration, lockout time.Duration) *authRateLimiter {
	return &authRateLimiter{maxFails: maxFails, window: window, lockout: lockout, entries: make(map[string]authRateEntry)}
}

func authAttemptAllowed(kind string, r *http.Request, identity string) (bool, time.Duration) {
	return defaultAuthRateLimiter.allow(authRateKey(kind, r, identity), time.Now())
}

func recordAuthAttempt(kind string, r *http.Request, identity string, success bool) {
	defaultAuthRateLimiter.record(authRateKey(kind, r, identity), success, time.Now())
}

func (l *authRateLimiter) allow(key string, now time.Time) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.pruneExpiredLocked(now)
	entry, ok := l.entries[key]
	if !ok {
		return true, 0
	}
	if !entry.blocked.IsZero() && now.Before(entry.blocked) {
		return false, time.Until(entry.blocked)
	}
	return true, 0
}

func (l *authRateLimiter) record(key string, success bool, now time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.pruneExpiredLocked(now)
	if success {
		delete(l.entries, key)
		return
	}
	entry := l.entries[key]
	if entry.firstFail.IsZero() || now.Sub(entry.firstFail) > l.window {
		entry = authRateEntry{firstFail: now}
	}
	entry.fails++
	if entry.fails >= l.maxFails {
		entry.blocked = now.Add(l.lockout)
	}
	l.entries[key] = entry
}

func (l *authRateLimiter) pruneExpiredLocked(now time.Time) {
	for key, entry := range l.entries {
		if !entry.blocked.IsZero() {
			if !now.Before(entry.blocked) {
				delete(l.entries, key)
			}
			continue
		}
		if entry.firstFail.IsZero() || now.Sub(entry.firstFail) > l.window {
			delete(l.entries, key)
		}
	}
}

func authRateKey(kind string, r *http.Request, identity string) string {
	identity = strings.TrimSpace(strings.ToLower(identity))
	if identity == "" {
		identity = "anonymous"
	}
	hash := sha256.Sum256([]byte(identity))
	return strings.TrimSpace(kind) + "|" + clientIP(r) + "|" + hex.EncodeToString(hash[:8])
}

func clientIP(r *http.Request) string {
	if r == nil {
		return "unknown"
	}
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil && host != "" {
		return host
	}
	if strings.TrimSpace(r.RemoteAddr) != "" {
		return strings.TrimSpace(r.RemoteAddr)
	}
	return "unknown"
}

func hasForwardedClientHeaders(r *http.Request) bool {
	if r == nil {
		return false
	}
	for _, header := range []string{"CF-Connecting-IP", "X-Real-IP", "X-Forwarded-For"} {
		if strings.TrimSpace(r.Header.Get(header)) != "" {
			return true
		}
	}
	return false
}

func writeAuthRateLimited(w http.ResponseWriter, retryAfter time.Duration) {
	if retryAfter > 0 {
		seconds := int(retryAfter.Seconds())
		if seconds < 1 {
			seconds = 1
		}
		w.Header().Set("Retry-After", stringInt(seconds))
	}
	writeError(w, http.StatusTooManyRequests, "AUTH_RATE_LIMITED", "认证失败次数过多，请稍后再试")
}

func stringInt(value int) string {
	if value == 0 {
		return "0"
	}
	digits := [20]byte{}
	i := len(digits)
	negative := value < 0
	if negative {
		value = -value
	}
	for value > 0 {
		i--
		digits[i] = byte('0' + value%10)
		value /= 10
	}
	if negative {
		i--
		digits[i] = '-'
	}
	return string(digits[i:])
}
