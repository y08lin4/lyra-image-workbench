package promptlibrary

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestListReturnsCachedLibraryAndRefreshesInBackground(t *testing.T) {
	service := NewService(t.TempDir())
	stale := Library{
		Repo:       "ZeroLu/awesome-gpt-image",
		Lang:       "zh-CN",
		SourceURL:  "https://github.com/ZeroLu/awesome-gpt-image",
		ReadmeURL:  "https://github.com/ZeroLu/awesome-gpt-image/blob/main/README.zh-CN.md",
		FetchedAt:  time.Now().Add(-time.Hour),
		ContentSHA: "cached",
		ETag:       `"cached"`,
		Categories: []string{"旧分类"},
		Total:      1,
		Matching:   1,
		Items: []Item{{
			ID:       "cached",
			Title:    "Cached",
			Category: "旧分类",
			Prompt:   "cached prompt",
			RepoURL:  "https://github.com/ZeroLu/awesome-gpt-image",
		}},
	}
	if err := service.store.Save("zh-CN", stale); err != nil {
		t.Fatalf("save stale cache: %v", err)
	}

	started := make(chan struct{}, 1)
	release := make(chan struct{})
	done := make(chan struct{})
	service.client = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		select {
		case started <- struct{}{}:
		default:
		}
		<-release
		defer close(done)
		markdown := strings.Join([]string{
			"# Awesome GPT Image 2 简体中文",
			"",
			"## 新分类",
			"",
			"### Fresh",
			"",
			"**提示词:**",
			"```text",
			"fresh prompt",
			"```",
		}, "\n")
		body := fmt.Sprintf(`{"sha":"fresh","content":%q,"encoding":"base64","html_url":"https://github.com/ZeroLu/awesome-gpt-image/blob/main/README.zh-CN.md"}`, base64.StdEncoding.EncodeToString([]byte(markdown)))
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"ETag": []string{`"fresh"`}},
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    req,
		}, nil
	})}

	type listResult struct {
		library Library
		err     error
	}
	result := make(chan listResult, 1)
	go func() {
		library, err := service.List(context.Background(), Query{Lang: "zh-CN", Limit: 10})
		result <- listResult{library: library, err: err}
	}()

	var got listResult
	select {
	case got = <-result:
	case <-time.After(150 * time.Millisecond):
		close(release)
		t.Fatal("List blocked on the stale cache refresh")
	}
	if got.err != nil {
		t.Fatalf("List returned error: %v", got.err)
	}
	if !got.library.Stale || got.library.Items[0].Title != "Cached" {
		t.Fatalf("List should return stale cached data immediately: %#v", got.library)
	}

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("background refresh did not start")
	}
	close(release)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("background refresh did not finish")
	}
	for deadline := time.After(time.Second); ; {
		current, ok, err := service.loadCached("zh-CN")
		if err != nil {
			t.Fatalf("load refreshed cache: %v", err)
		}
		if ok && current.ContentSHA == "fresh" && len(current.Items) == 1 && current.Items[0].Title == "Fresh" {
			return
		}
		select {
		case <-deadline:
			t.Fatalf("cache was not refreshed, current=%#v", current)
		case <-time.After(10 * time.Millisecond):
		}
	}
}

func TestStartWarmCachePrefetchesMissingCacheInBackground(t *testing.T) {
	service := NewService(t.TempDir())
	started := make(chan struct{}, 1)
	release := make(chan struct{})
	done := make(chan struct{}, 1)
	service.client = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		select {
		case started <- struct{}{}:
		default:
		}
		<-release
		defer func() { done <- struct{}{} }()
		return promptLibraryContentsResponse(req, "cold", "Cold", "冷启动", "cold prompt"), nil
	})}

	returned := make(chan struct{})
	go func() {
		service.StartWarmCache(context.Background())
		close(returned)
	}()

	select {
	case <-returned:
	case <-time.After(150 * time.Millisecond):
		close(release)
		t.Fatal("StartWarmCache blocked on cold prefetch")
	}
	select {
	case <-started:
	case <-time.After(time.Second):
		close(release)
		t.Fatal("cold prefetch did not start")
	}
	close(release)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("cold prefetch did not finish")
	}
	current := waitCachedLibrary(t, service, "zh-CN", "cold")
	if len(current.Items) != 1 || current.Items[0].Title != "Cold" {
		t.Fatalf("unexpected cold cache: %#v", current)
	}
}

func TestListReusesColdWarmCacheInFlight(t *testing.T) {
	service := NewService(t.TempDir())
	started := make(chan struct{}, 1)
	release := make(chan struct{})
	var requests int32
	service.client = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		atomic.AddInt32(&requests, 1)
		select {
		case started <- struct{}{}:
		default:
		}
		<-release
		return promptLibraryContentsResponse(req, "shared", "Shared", "冷启动", "shared prompt"), nil
	})}

	service.StartWarmCache(context.Background())
	select {
	case <-started:
	case <-time.After(time.Second):
		close(release)
		t.Fatal("cold prefetch did not start")
	}

	type listResult struct {
		library Library
		err     error
	}
	result := make(chan listResult, 1)
	go func() {
		library, err := service.List(context.Background(), Query{Lang: "zh-CN", Limit: 10})
		result <- listResult{library: library, err: err}
	}()

	select {
	case got := <-result:
		close(release)
		t.Fatalf("List returned before the in-flight cold prefetch completed: %#v", got)
	case <-time.After(100 * time.Millisecond):
	}
	if got := atomic.LoadInt32(&requests); got != 1 {
		close(release)
		t.Fatalf("expected one in-flight GitHub request, got %d", got)
	}
	close(release)

	select {
	case got := <-result:
		if got.err != nil {
			t.Fatalf("List returned error: %v", got.err)
		}
		if got.library.ContentSHA != "shared" || len(got.library.Items) != 1 || got.library.Items[0].Title != "Shared" {
			t.Fatalf("List returned unexpected library: %#v", got.library)
		}
	case <-time.After(time.Second):
		t.Fatal("List did not return after cold prefetch completed")
	}
	if got := atomic.LoadInt32(&requests); got != 1 {
		t.Fatalf("expected List to reuse the cold prefetch request, got %d requests", got)
	}
	waitCachedLibrary(t, service, "zh-CN", "shared")
}

func promptLibraryContentsResponse(req *http.Request, sha string, title string, category string, prompt string) *http.Response {
	markdown := strings.Join([]string{
		"# Awesome GPT Image 2 简体中文",
		"",
		"## " + category,
		"",
		"### " + title,
		"",
		"**提示词:**",
		"```text",
		prompt,
		"```",
	}, "\n")
	body := fmt.Sprintf(`{"sha":%q,"content":%q,"encoding":"base64","html_url":"https://github.com/ZeroLu/awesome-gpt-image/blob/main/README.zh-CN.md"}`, sha, base64.StdEncoding.EncodeToString([]byte(markdown)))
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"ETag": []string{fmt.Sprintf(`"%s"`, sha)}},
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    req,
	}
}

func waitCachedLibrary(t *testing.T, service *Service, lang string, sha string) Library {
	t.Helper()
	for deadline := time.After(time.Second); ; {
		current, ok, err := service.loadCached(lang)
		if err != nil {
			t.Fatalf("load cached library: %v", err)
		}
		if ok && current.ContentSHA == sha {
			return current
		}
		select {
		case <-deadline:
			t.Fatalf("cache did not reach sha %q, current=%#v", sha, current)
		case <-time.After(10 * time.Millisecond):
		}
	}
}
