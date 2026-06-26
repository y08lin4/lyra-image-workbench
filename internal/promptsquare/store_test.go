package promptsquare

import (
	"os"
	"path"
	"path/filepath"
	"testing"
	"time"
)

func TestPromptSquareStoreLikeIsIdempotent(t *testing.T) {
	store := newTestStore(t)
	item := createTestItem(t, store, "alice", "first", time.Date(2026, 6, 26, 9, 0, 0, 0, time.Local))

	liked, err := store.SetLike(item.ID, "Bob", true)
	if err != nil {
		t.Fatalf("SetLike(true) error = %v", err)
	}
	likedAgain, err := store.SetLike(item.ID, "bob", true)
	if err != nil {
		t.Fatalf("SetLike(true again) error = %v", err)
	}
	if liked.LikeCount != 1 || likedAgain.LikeCount != 1 || !likedAgain.LikedByMe {
		t.Fatalf("like should be idempotent and liked by viewer: first=%+v again=%+v", liked, likedAgain)
	}

	unliked, err := store.SetLike(item.ID, "BOB", false)
	if err != nil {
		t.Fatalf("SetLike(false) error = %v", err)
	}
	unlikedAgain, err := store.SetLike(item.ID, "bob", false)
	if err != nil {
		t.Fatalf("SetLike(false again) error = %v", err)
	}
	if unliked.LikeCount != 0 || unlikedAgain.LikeCount != 0 || unlikedAgain.LikedByMe {
		t.Fatalf("unlike should be idempotent and clear viewer like: first=%+v again=%+v", unliked, unlikedAgain)
	}
}

func TestPromptSquareStoreDailySortsByLikesThenEarlierCreated(t *testing.T) {
	store := newTestStore(t)
	day := time.Date(2026, 6, 26, 12, 0, 0, 0, time.Local)
	older := createTestItem(t, store, "alice", "older one like", time.Date(2026, 6, 26, 8, 0, 0, 0, time.Local))
	newer := createTestItem(t, store, "alice", "newer one like", time.Date(2026, 6, 26, 9, 0, 0, 0, time.Local))
	top := createTestItem(t, store, "alice", "top two likes", time.Date(2026, 6, 26, 10, 0, 0, 0, time.Local))
	_ = createTestItem(t, store, "alice", "yesterday", time.Date(2026, 6, 25, 23, 0, 0, 0, time.Local))

	mustLike(t, store, older.ID, "u1")
	mustLike(t, store, newer.ID, "u2")
	mustLike(t, store, top.ID, "u3")
	mustLike(t, store, top.ID, "u4")

	items, err := store.DailyForUser("u3", day)
	if err != nil {
		t.Fatalf("DailyForUser() error = %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("expected 3 daily items, got %d: %+v", len(items), items)
	}
	if items[0].ID != top.ID || items[0].DailyRank != 1 || items[0].LikeCount != 2 || !items[0].LikedByMe {
		t.Fatalf("top item/rank/like viewer mismatch: %+v", items[0])
	}
	if items[1].ID != older.ID || items[1].DailyRank != 2 {
		t.Fatalf("same-like older item should rank second: %+v", items)
	}
	if items[2].ID != newer.ID || items[2].DailyRank != 3 {
		t.Fatalf("same-like newer item should rank third: %+v", items)
	}
}

func TestPromptSquareStoreMineFiltersByAuthor(t *testing.T) {
	store := newTestStore(t)
	firstAlice := createTestItem(t, store, "Alice", "first alice", time.Date(2026, 6, 26, 8, 0, 0, 0, time.Local))
	_ = createTestItem(t, store, "bob", "bob", time.Date(2026, 6, 26, 9, 0, 0, 0, time.Local))
	secondAlice := createTestItem(t, store, "alice", "second alice", time.Date(2026, 6, 26, 10, 0, 0, 0, time.Local))

	items, err := store.MineForUser("ALICE")
	if err != nil {
		t.Fatalf("MineForUser() error = %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 alice items, got %d: %+v", len(items), items)
	}
	if items[0].ID != secondAlice.ID || items[1].ID != firstAlice.ID {
		t.Fatalf("mine should filter alice and sort newest first: %+v", items)
	}
}

func TestPromptSquareStoreSubmitFromResultCreatesPermanentCopy(t *testing.T) {
	store := newTestStore(t)
	store.now = func() time.Time { return time.Date(2026, 6, 26, 11, 0, 0, 0, time.UTC) }

	sourceDir := t.TempDir()
	sourcePath := filepath.Join(sourceDir, "result.png")
	sourceData := []byte("not-a-real-png-but-output-owned")
	if err := os.WriteFile(sourcePath, sourceData, 0o644); err != nil {
		t.Fatalf("write source image: %v", err)
	}

	item, err := store.SubmitFromResult(SubmitFromResultRequest{
		Title:             "result title",
		Prompt:            "result prompt",
		Model:             "gpt-image-2",
		Ratio:             "16:9",
		Quality:           "high",
		OutputFormat:      "png",
		Tags:              []string{"daily", "result"},
		Author:            "Alice",
		AuthorDisplayName: "Alice A.",
		SourceTaskID:      "img_task_01",
		SourceImagePath:   sourcePath,
		SourceImageMime:   "image/png",
	})
	if err != nil {
		t.Fatalf("SubmitFromResult() error = %v", err)
	}
	if !item.Permanent || item.Source.Type != "task_result" || item.SourceTaskID != "img_task_01" {
		t.Fatalf("submitted item should be permanent task result: %+v", item)
	}
	if item.ImageURL == "" || path.Dir(item.ImageURL) != "/api/prompt-square/images" {
		t.Fatalf("submitted item should point at prompt-square copy, got %q", item.ImageURL)
	}

	copyPath, mime, err := store.ResolveImage(filepath.Base(item.ImageURL))
	if err != nil {
		t.Fatalf("ResolveImage(copy) error = %v", err)
	}
	if mime != "image/png" {
		t.Fatalf("copy mime = %q, want image/png", mime)
	}
	copied, err := os.ReadFile(copyPath)
	if err != nil {
		t.Fatalf("read copied image: %v", err)
	}
	if string(copied) != string(sourceData) {
		t.Fatalf("copied image data mismatch: got %q want %q", copied, sourceData)
	}

	if err := os.Remove(sourcePath); err != nil {
		t.Fatalf("remove source image: %v", err)
	}
	if _, _, err := store.ResolveImage(filepath.Base(item.ImageURL)); err != nil {
		t.Fatalf("permanent copy should remain after source removal: %v", err)
	}
}

func newTestStore(t *testing.T) *Store {
	t.Helper()
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	return store
}

func createTestItem(t *testing.T, store *Store, author string, prompt string, now time.Time) Item {
	t.Helper()
	store.now = func() time.Time { return now }
	item, err := store.Create(CreateRequest{
		Title:             prompt,
		Prompt:            prompt,
		Model:             "gpt-image-2",
		AuthorUsername:    author,
		AuthorDisplayName: author,
		Params: map[string]string{
			"ratio":        "1:1",
			"quality":      "high",
			"outputFormat": "png",
		},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	return item
}

func mustLike(t *testing.T, store *Store, id string, username string) {
	t.Helper()
	if _, err := store.SetLike(id, username, true); err != nil {
		t.Fatalf("SetLike(%s, %s) error = %v", id, username, err)
	}
}
