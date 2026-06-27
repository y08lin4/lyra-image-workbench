package promptlibrary

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	defaultTTL                  = 20 * time.Minute
	backgroundRefreshTimeout    = 30 * time.Second
	backgroundImageCacheTimeout = 90 * time.Second
)

type Service struct {
	owner  string
	repo   string
	branch string
	ttl    time.Duration
	client *http.Client
	store  *Store

	mu       sync.Mutex
	cache    map[string]Library
	inFlight map[string]*inFlightCall
	warmOnce sync.Once
}

type inFlightCall struct {
	done chan struct{}
}

func NewService(cacheDir string) *Service {
	return &Service{
		owner:    DefaultOwner,
		repo:     DefaultRepo,
		branch:   DefaultBranch,
		ttl:      defaultTTL,
		client:   &http.Client{Timeout: 15 * time.Second},
		store:    NewStore(cacheDir),
		cache:    make(map[string]Library),
		inFlight: make(map[string]*inFlightCall),
	}
}

func (s *Service) StartWarmCache(ctx context.Context) {
	if s == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	s.warmOnce.Do(func() {
		go s.warmCache(ctx, DefaultLang)
	})
}

func (s *Service) warmCache(ctx context.Context, langs ...string) {
	for _, rawLang := range langs {
		if err := ctx.Err(); err != nil {
			return
		}
		lang, readmePath := normalizeLang(rawLang)
		cached, ok, err := s.loadCached(lang)
		if err != nil {
			continue
		}
		if !ok || len(cached.Items) == 0 {
			s.syncInBackground(lang, readmePath)
			continue
		}
		cached.Stale = time.Since(cached.FetchedAt) >= s.ttl
		if cached.Stale {
			s.syncInBackground(lang, readmePath)
		}
		s.cacheImagesInBackground(lang, cached)
	}
}

func (s *Service) List(ctx context.Context, query Query) (Library, error) {
	if s == nil || s.store == nil {
		return Library{}, fmt.Errorf("提示词库服务未初始化")
	}
	lang, readmePath := normalizeLang(query.Lang)
	cached, ok, err := s.loadCached(lang)
	if err != nil {
		return Library{}, err
	}
	if ok && !query.Force && len(cached.Items) > 0 {
		cached.Stale = time.Since(cached.FetchedAt) >= s.ttl
		cached.FetchError = ""
		if cached.Stale {
			s.syncInBackground(lang, readmePath)
		}
		s.cacheImagesInBackground(lang, cached)
		return filterLibrary(cached, query), nil
	}
	fresh, err := s.sync(ctx, lang, readmePath, cached, ok)
	if err != nil {
		if ok && len(cached.Items) > 0 {
			cached.Stale = true
			cached.FetchError = err.Error()
			s.cacheImagesInBackground(lang, cached)
			return filterLibrary(cached, query), nil
		}
		return Library{}, err
	}
	return filterLibrary(fresh, query), nil
}

func (s *Service) loadCached(lang string) (Library, bool, error) {
	s.mu.Lock()
	if s.cache != nil {
		if cached, ok := s.cache[lang]; ok {
			s.mu.Unlock()
			return cloneLibrary(cached), true, nil
		}
	}
	s.mu.Unlock()

	cached, ok, err := s.store.Load(lang)
	if err != nil || !ok {
		return cached, ok, err
	}
	s.rememberCached(lang, cached)
	return cloneLibrary(cached), true, nil
}

func (s *Service) saveCached(lang string, lib Library) error {
	if err := s.store.Save(lang, lib); err != nil {
		return err
	}
	s.rememberCached(lang, lib)
	return nil
}

func (s *Service) rememberCached(lang string, lib Library) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cache == nil {
		s.cache = make(map[string]Library)
	}
	s.cache[lang] = cloneLibrary(lib)
}

func (s *Service) syncInBackground(lang string, readmePath string) {
	key := syncInFlightKey(lang)
	if !s.beginInFlight(key) {
		return
	}
	go func() {
		defer s.endInFlight(key)
		ctx, cancel := context.WithTimeout(context.Background(), backgroundRefreshTimeout)
		defer cancel()
		cached, ok, err := s.loadCached(lang)
		if err != nil {
			return
		}
		fresh, err := s.fetch(ctx, lang, readmePath, cached, ok)
		if err != nil {
			return
		}
		if err := s.saveCached(lang, fresh); err != nil {
			return
		}
		s.cacheImagesInBackground(lang, fresh)
	}()
}

func (s *Service) sync(ctx context.Context, lang string, readmePath string, cached Library, hasCache bool) (Library, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	key := syncInFlightKey(lang)
	for {
		if s.beginInFlight(key) {
			defer s.endInFlight(key)
			fresh, err := s.fetch(ctx, lang, readmePath, cached, hasCache)
			if err != nil {
				return Library{}, err
			}
			if err := s.saveCached(lang, fresh); err != nil {
				return Library{}, err
			}
			s.cacheImagesInBackground(lang, fresh)
			return fresh, nil
		}
		if err := s.waitInFlight(ctx, key); err != nil {
			return Library{}, err
		}
		current, ok, err := s.loadCached(lang)
		if err != nil {
			return Library{}, err
		}
		if ok && len(current.Items) > 0 {
			current.Stale = time.Since(current.FetchedAt) >= s.ttl
			if !current.Stale {
				current.FetchError = ""
				s.cacheImagesInBackground(lang, current)
				return current, nil
			}
		}
		cached = current
		hasCache = ok
	}
}

func (s *Service) cacheImagesInBackground(lang string, lib Library) {
	if !s.hasCacheableRemoteImages(lib) {
		return
	}
	key := "images:" + lang
	if !s.beginInFlight(key) {
		return
	}
	go func() {
		defer s.endInFlight(key)
		ctx, cancel := context.WithTimeout(context.Background(), backgroundImageCacheTimeout)
		defer cancel()
		next, changed := s.cacheLibraryImages(ctx, lib)
		if !changed {
			return
		}
		current, ok, err := s.loadCached(lang)
		if err != nil || !ok || !sameLibraryVersion(current, lib) {
			return
		}
		next.FetchedAt = current.FetchedAt
		next.ETag = current.ETag
		next.Stale = current.Stale
		next.FetchError = current.FetchError
		_ = s.saveCached(lang, next)
	}()
}

func (s *Service) beginInFlight(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.inFlight == nil {
		s.inFlight = make(map[string]*inFlightCall)
	}
	if _, ok := s.inFlight[key]; ok {
		return false
	}
	s.inFlight[key] = &inFlightCall{done: make(chan struct{})}
	return true
}

func (s *Service) waitInFlight(ctx context.Context, key string) error {
	s.mu.Lock()
	call := s.inFlight[key]
	s.mu.Unlock()
	if call == nil {
		return nil
	}
	select {
	case <-call.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *Service) endInFlight(key string) {
	s.mu.Lock()
	call := s.inFlight[key]
	delete(s.inFlight, key)
	s.mu.Unlock()
	if call != nil {
		close(call.done)
	}
}

func syncInFlightKey(lang string) string {
	return "sync:" + lang
}

func sameLibraryVersion(a Library, b Library) bool {
	if a.ContentSHA != "" || b.ContentSHA != "" {
		return a.ContentSHA == b.ContentSHA
	}
	return a.FetchedAt.Equal(b.FetchedAt)
}

type contentsResponse struct {
	SHA         string `json:"sha"`
	Content     string `json:"content"`
	Encoding    string `json:"encoding"`
	DownloadURL string `json:"download_url"`
	HTMLURL     string `json:"html_url"`
}

func (s *Service) fetch(ctx context.Context, lang string, readmePath string, cached Library, hasCache bool) (Library, error) {
	endpoint := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s?ref=%s", s.owner, s.repo, readmePath, s.branch)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return Library{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "Lyra-Image-Workbench")
	if hasCache && cached.ETag != "" {
		req.Header.Set("If-None-Match", cached.ETag)
	}
	res, err := s.client.Do(req)
	if err != nil {
		return Library{}, fmt.Errorf("同步 GitHub 提示词库失败：%w", err)
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusNotModified && hasCache {
		cached.FetchedAt = time.Now().UTC()
		cached.Stale = false
		cached.FetchError = ""
		return cached, nil
	}
	body, err := io.ReadAll(io.LimitReader(res.Body, 4*1024*1024))
	if err != nil {
		return Library{}, err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return Library{}, fmt.Errorf("GitHub 返回 HTTP %d：%s", res.StatusCode, strings.TrimSpace(string(body)))
	}
	var payload contentsResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return Library{}, fmt.Errorf("解析 GitHub 响应失败：%w", err)
	}
	if payload.Encoding != "base64" {
		return Library{}, fmt.Errorf("GitHub README 编码不支持：%s", payload.Encoding)
	}
	encoded := strings.ReplaceAll(payload.Content, "\n", "")
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return Library{}, fmt.Errorf("解码 GitHub README 失败：%w", err)
	}
	readmeURL := payload.HTMLURL
	if readmeURL == "" {
		readmeURL = fmt.Sprintf("https://github.com/%s/%s/blob/%s/%s", s.owner, s.repo, s.branch, readmePath)
	}
	items := parseMarkdown(string(decoded), parseOptions{owner: s.owner, repo: s.repo, branch: s.branch, readmePath: readmePath, readmeURL: readmeURL})
	etag := res.Header.Get("ETag")
	if etag == "" {
		etag = cached.ETag
	}
	lib := Library{
		Repo:       fmt.Sprintf("%s/%s", s.owner, s.repo),
		Lang:       lang,
		SourceURL:  fmt.Sprintf("https://github.com/%s/%s", s.owner, s.repo),
		ReadmeURL:  readmeURL,
		FetchedAt:  time.Now().UTC(),
		ContentSHA: payload.SHA,
		ETag:       etag,
		Stale:      false,
		Categories: categoriesFromItems(items),
		Total:      len(items),
		Matching:   len(items),
		Items:      items,
	}
	return lib, nil
}

func normalizeLang(raw string) (string, string) {
	lang := strings.TrimSpace(raw)
	if lang == "" {
		lang = DefaultLang
	}
	switch strings.ToLower(lang) {
	case "en", "en-us", "english":
		return "en", "README.md"
	case "zh", "zh-cn", "cn", "简体中文":
		return "zh-CN", "README.zh-CN.md"
	case "zh-tw", "tw", "繁體中文":
		return "zh-TW", "README.zh-TW.md"
	case "ja", "jp", "日本語":
		return "ja", "README.ja.md"
	case "ko", "kr", "한국어":
		return "ko", "README.ko.md"
	case "fr", "français":
		return "fr", "README.fr.md"
	case "de", "deutsch":
		return "de", "README.de.md"
	case "es", "español":
		return "es", "README.es.md"
	default:
		return DefaultLang, "README.zh-CN.md"
	}
}

func filterLibrary(lib Library, query Query) Library {
	q := strings.ToLower(strings.TrimSpace(query.Q))
	category := strings.TrimSpace(query.Category)
	limit := query.Limit
	if limit <= 0 || limit > 300 {
		limit = 80
	}
	items := make([]Item, 0, len(lib.Items))
	for _, item := range lib.Items {
		if category != "" && item.Category != category {
			continue
		}
		if q != "" {
			haystack := strings.ToLower(item.Title + "\n" + item.Category + "\n" + item.Prompt)
			if !strings.Contains(haystack, q) {
				continue
			}
		}
		items = append(items, item)
	}
	lib.Total = len(lib.Items)
	lib.Matching = len(items)
	if len(items) > limit {
		items = items[:limit]
	}
	lib.Items = items
	lib.Categories = append([]string{}, lib.Categories...)
	sort.Strings(lib.Categories)
	return lib
}
