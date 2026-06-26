package api

import (
	"testing"
	"time"
)

func TestAuthRateLimiterPrunesExpiredEntries(t *testing.T) {
	limiter := newAuthRateLimiter(2, time.Minute, time.Minute)
	start := time.Date(2026, 6, 26, 12, 0, 0, 0, time.UTC)

	limiter.record("stale-window", false, start)
	limiter.record("blocked", false, start)
	limiter.record("blocked", false, start.Add(time.Second))
	if len(limiter.entries) != 2 {
		t.Fatalf("entries before prune=%d, want 2", len(limiter.entries))
	}

	allowed, _ := limiter.allow("fresh", start.Add(3*time.Minute))
	if !allowed {
		t.Fatal("unrelated fresh key should be allowed")
	}
	if len(limiter.entries) != 0 {
		t.Fatalf("expired entries should be pruned, got %d", len(limiter.entries))
	}
}
