package activitylog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestStoreAppendRecentFiltersAndPersists(t *testing.T) {
	path := filepath.Join(t.TempDir(), "activity.json")
	now := time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC)
	store, err := NewStoreWithClock(path, func() time.Time { return now })
	if err != nil {
		t.Fatalf("NewStoreWithClock() error = %v", err)
	}

	if _, err := store.Append(EntryInput{Type: TypeUserRegistration, Level: LevelInfo, Username: "alice", Message: "registered"}); err != nil {
		t.Fatalf("Append(register) error = %v", err)
	}
	now = now.Add(time.Minute)
	if _, err := store.Append(EntryInput{Type: TypeResultFailed, Level: LevelError, ResourceID: "img_1", Message: "result failed"}); err != nil {
		t.Fatalf("Append(result failed) error = %v", err)
	}

	recent := store.Recent(Query{Limit: 10})
	if len(recent) != 2 || recent[0].Type != TypeResultFailed || recent[1].Type != TypeUserRegistration {
		t.Fatalf("Recent() order/items = %+v", recent)
	}
	errorsOnly := store.Recent(Query{Limit: 10, Level: LevelError})
	if len(errorsOnly) != 1 || errorsOnly[0].Type != TypeResultFailed {
		t.Fatalf("Recent(LevelError) = %+v", errorsOnly)
	}
	registrations := store.Recent(Query{Limit: 10, Type: TypeUserRegistration})
	if len(registrations) != 1 || registrations[0].Username != "alice" {
		t.Fatalf("Recent(TypeUserRegistration) = %+v", registrations)
	}

	reloaded, err := NewStore(path)
	if err != nil {
		t.Fatalf("NewStore(reload) error = %v", err)
	}
	if got := reloaded.Recent(Query{Limit: 2}); len(got) != 2 {
		t.Fatalf("reloaded Recent() len = %d, want 2", len(got))
	}
}

func TestStoreWhitelistsActivityFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "activity.json")
	store, err := NewStore(path)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	if _, err := store.Append(EntryInput{
		Type:    TypeTaskFailed,
		Level:   LevelError,
		Message: "failed",
		Fields: map[string]any{
			"amount":    5,
			"target":    "alice",
			"apiKey":    "sk-secret",
			"token":     "session-token-secret",
			"password":  "p@ssw0rd",
			"prompt":    "paint a detailed secret prompt",
			"imageData": "data:image/png;base64,abc123",
			"payUrl":    "https://pay.example.test/?sign=secret-signature",
			"sign":      "secret-signature",
			"reason":    "free-form admin reason should not be globally logged",
		},
	}); err != nil {
		t.Fatalf("Append() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	for _, leaked := range []string{"sk-secret", "session-token-secret", "p@ssw0rd", "paint a detailed", "data:image", "pay.example.test", "secret-signature", "free-form admin reason"} {
		if strings.Contains(string(data), leaked) {
			t.Fatalf("activity log leaked %q: %s", leaked, string(data))
		}
	}
	var persisted struct {
		Entries []Entry `json:"entries"`
	}
	if err := json.Unmarshal(data, &persisted); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	fields := persisted.Entries[0].Fields
	if got := fields["amount"]; got != float64(5) && got != 5 {
		t.Fatalf("amount field = %v", got)
	}
	if got := fields["target"]; got != "alice" {
		t.Fatalf("target field = %v", got)
	}
	for _, dropped := range []string{"apiKey", "token", "password", "prompt", "imageData", "payUrl", "sign", "reason"} {
		if _, ok := fields[dropped]; ok {
			t.Fatalf("non-whitelisted field %q was persisted: %+v", dropped, fields)
		}
	}
}
