package promptlibrary

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const maxPromptLibraryImageBytes = 12 * 1024 * 1024

var promptLibraryImageNameRe = regexp.MustCompile(`^[a-f0-9]{40}\.(jpg|jpeg|png|webp|gif)$`)

func (s *Service) cacheLibraryImages(ctx context.Context, lib Library) (Library, bool) {
	if s == nil || s.store == nil || s.client == nil {
		return lib, false
	}
	changed := false
	for itemIndex := range lib.Items {
		for imageIndex := range lib.Items[itemIndex].Images {
			image := lib.Items[itemIndex].Images[imageIndex]
			imageURL := strings.TrimSpace(image.URL)
			if imageURL == "" {
				continue
			}

			if strings.HasPrefix(imageURL, "/api/prompt-library/images/") {
				file := strings.TrimPrefix(imageURL, "/api/prompt-library/images/")
				if _, ok := s.CachedImagePath(file); ok {
					continue
				}
				if image.OriginalURL == "" || !isPromptLibraryCacheableURL(image.OriginalURL) {
					continue
				}
				localURL, err := s.cacheRemoteImage(ctx, image.OriginalURL)
				if err != nil || localURL == "" {
					continue
				}
				lib.Items[itemIndex].Images[imageIndex].URL = localURL
				changed = true
				continue
			}

			if !isPromptLibraryCacheableURL(imageURL) {
				continue
			}
			localURL, err := s.cacheRemoteImage(ctx, imageURL)
			if err != nil || localURL == "" {
				continue
			}
			if lib.Items[itemIndex].Images[imageIndex].OriginalURL == "" {
				lib.Items[itemIndex].Images[imageIndex].OriginalURL = imageURL
			}
			lib.Items[itemIndex].Images[imageIndex].URL = localURL
			changed = true
		}
	}
	return lib, changed
}

func (s *Service) cacheRemoteImage(ctx context.Context, rawURL string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	if !isPromptLibraryCacheableURL(parsed.String()) {
		return "", fmt.Errorf("prompt library image host is not cacheable")
	}
	nameBase := hashPromptLibraryImageURL(rawURL)
	imageDir := filepath.Join(s.store.dir, "images")
	if existing, ok := findExistingPromptLibraryImage(imageDir, nameBase); ok {
		return "/api/prompt-library/images/" + existing, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Lyra-Image-Workbench")
	client := promptLibraryImageHTTPClient(s.client)
	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return "", fmt.Errorf("image cache HTTP %d", res.StatusCode)
	}

	limited := io.LimitReader(res.Body, maxPromptLibraryImageBytes+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return "", err
	}
	if len(body) == 0 || len(body) > maxPromptLibraryImageBytes {
		return "", fmt.Errorf("prompt library image too large")
	}
	_, ext, ok := allowedPromptLibraryImageType(res.Header.Get("Content-Type"), body)
	if !ok {
		return "", fmt.Errorf("prompt library image type is not allowed")
	}
	filename := nameBase + ext
	if !promptLibraryImageNameRe.MatchString(filename) {
		return "", fmt.Errorf("prompt library image filename is invalid")
	}
	if err := os.MkdirAll(imageDir, 0o700); err != nil {
		return "", err
	}
	finalPath := filepath.Join(imageDir, filename)
	if _, err := os.Stat(finalPath); err == nil {
		return "/api/prompt-library/images/" + filename, nil
	}
	tmp := fmt.Sprintf("%s.%d.tmp", finalPath, time.Now().UnixNano())
	if err := os.WriteFile(tmp, body, 0o600); err != nil {
		return "", err
	}
	if err := os.Rename(tmp, finalPath); err != nil {
		_ = os.Remove(tmp)
		return "", err
	}
	return "/api/prompt-library/images/" + filename, nil
}

func (s *Service) CachedImagePath(file string) (string, bool) {
	if s == nil || s.store == nil {
		return "", false
	}
	name := strings.TrimSpace(file)
	if name == "" || filepath.Base(name) != name || strings.HasSuffix(name, ".tmp") || !promptLibraryImageNameRe.MatchString(name) {
		return "", false
	}
	root := filepath.Join(s.store.dir, "images")
	path := filepath.Join(root, name)
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", false
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", false
	}
	rel, err := filepath.Rel(absRoot, absPath)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", false
	}
	info, err := os.Stat(absPath)
	if err != nil || info.IsDir() {
		return "", false
	}
	return absPath, true
}

func isPromptLibraryCacheableURL(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	if parsed.Scheme != "https" {
		return false
	}
	host := strings.ToLower(parsed.Hostname())
	return host == "github.com" || host == "raw.githubusercontent.com" || host == "objects.githubusercontent.com" || strings.HasSuffix(host, ".githubusercontent.com")
}

func promptLibraryImageHTTPClient(base *http.Client) http.Client {
	client := http.Client{Timeout: 8 * time.Second}
	if base != nil {
		client.Transport = base.Transport
	}
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return http.ErrUseLastResponse
		}
		if req == nil || req.URL == nil || !isPromptLibraryCacheableURL(req.URL.String()) {
			return http.ErrUseLastResponse
		}
		return nil
	}
	return client
}

func allowedPromptLibraryImageType(header string, body []byte) (string, string, bool) {
	mediaType := strings.ToLower(strings.TrimSpace(header))
	if parsed, _, err := mime.ParseMediaType(mediaType); err == nil {
		mediaType = parsed
	}
	detected := strings.ToLower(http.DetectContentType(body))
	switch detected {
	case "image/jpeg":
		return "image/jpeg", ".jpg", true
	case "image/png":
		return "image/png", ".png", true
	case "image/gif":
		return "image/gif", ".gif", true
	}
	if mediaType == "image/webp" && hasWEBPMagic(body) {
		return "image/webp", ".webp", true
	}
	return "", "", false
}

func hasWEBPMagic(body []byte) bool {
	return len(body) >= 12 && string(body[:4]) == "RIFF" && string(body[8:12]) == "WEBP"
}

func hashPromptLibraryImageURL(rawURL string) string {
	sum := sha1.Sum([]byte(strings.TrimSpace(rawURL)))
	return hex.EncodeToString(sum[:])
}

func findExistingPromptLibraryImage(dir string, nameBase string) (string, bool) {
	matches, err := filepath.Glob(filepath.Join(dir, nameBase+".*"))
	if err != nil || len(matches) == 0 {
		return "", false
	}
	for _, match := range matches {
		name := filepath.Base(match)
		if !promptLibraryImageNameRe.MatchString(name) {
			continue
		}
		info, err := os.Stat(match)
		if err == nil && !info.IsDir() {
			return name, true
		}
	}
	return "", false
}
