package promptlibrary

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
)

func TestCacheLibraryImagesCachesGitHubImagesLocally(t *testing.T) {
	var requests int32
	service := &Service{
		client: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			atomic.AddInt32(&requests, 1)
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"image/png"}},
				Body:       io.NopCloser(strings.NewReader("\x89PNG\r\n\x1a\ncache-test")),
				Request:    req,
			}, nil
		})},
		store: NewStore(t.TempDir()),
	}
	lib := Library{Items: []Item{{
		ID:     "one",
		Title:  "One",
		Prompt: "Prompt",
		Images: []Image{{URL: "https://raw.githubusercontent.com/y08lin4/repo/main/example.png"}},
	}}}

	cached, changed := service.cacheLibraryImages(context.Background(), lib)
	if !changed {
		t.Fatal("expected first cache pass to change image URL")
	}
	localURL := cached.Items[0].Images[0].URL
	if !strings.HasPrefix(localURL, "/api/prompt-library/images/") {
		t.Fatalf("cached URL = %q, want local prompt-library image URL", localURL)
	}
	if cached.Items[0].Images[0].OriginalURL == "" {
		t.Fatalf("expected original URL to be preserved: %+v", cached.Items[0].Images[0])
	}

	cachedAgain, changedAgain := service.cacheLibraryImages(context.Background(), cached)
	if changedAgain {
		t.Fatalf("second cache pass should reuse local file without changes: %+v", cachedAgain.Items[0].Images[0])
	}
	if got := atomic.LoadInt32(&requests); got != 1 {
		t.Fatalf("image should be downloaded once, got %d requests", got)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}
