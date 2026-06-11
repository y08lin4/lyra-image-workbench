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
	"time"
)

const defaultTTL = 20 * time.Minute

type Service struct {
	owner  string
	repo   string
	branch string
	ttl    time.Duration
	client *http.Client
	store  *Store
}

func NewService(cacheDir string) *Service {
	return &Service{
		owner:  DefaultOwner,
		repo:   DefaultRepo,
		branch: DefaultBranch,
		ttl:    defaultTTL,
		client: &http.Client{Timeout: 15 * time.Second},
		store:  NewStore(cacheDir),
	}
}

func (s *Service) List(ctx context.Context, query Query) (Library, error) {
	if s == nil || s.store == nil {
		return Library{}, fmt.Errorf("提示词库服务未初始化")
	}
	lang, readmePath := normalizeLang(query.Lang)
	cached, ok, err := s.store.Load(lang)
	if err != nil {
		return Library{}, err
	}
	if ok && !query.Force && time.Since(cached.FetchedAt) < s.ttl && len(cached.Items) > 0 {
		cached.Stale = false
		cached.FetchError = ""
		return filterLibrary(cached, query), nil
	}
	fresh, err := s.fetch(ctx, lang, readmePath, cached, ok)
	if err != nil {
		if ok && len(cached.Items) > 0 {
			cached.Stale = true
			cached.FetchError = err.Error()
			return filterLibrary(cached, query), nil
		}
		return Library{}, err
	}
	if err := s.store.Save(lang, fresh); err != nil {
		return Library{}, err
	}
	return filterLibrary(fresh, query), nil
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
