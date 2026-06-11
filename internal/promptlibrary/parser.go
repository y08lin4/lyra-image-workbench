package promptlibrary

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"html"
	"net/url"
	"regexp"
	"sort"
	"strings"
)

var (
	htmlImageRe    = regexp.MustCompile(`(?i)<img\b[^>]*\bsrc=["']([^"']+)["'][^>]*>`)
	htmlAltRe      = regexp.MustCompile(`(?i)\balt=["']([^"']*)["']`)
	markdownImgRe  = regexp.MustCompile(`!\[([^\]]*)\]\(([^)\s]+)(?:\s+"[^"]*")?\)`)
	markdownLinkRe = regexp.MustCompile(`\[([^\]]+)\]\(([^)\s]+)(?:\s+"[^"]*")?\)`)
)

type parseOptions struct {
	owner      string
	repo       string
	branch     string
	readmePath string
	readmeURL  string
}

func parseMarkdown(text string, opts parseOptions) []Item {
	if opts.owner == "" {
		opts.owner = DefaultOwner
	}
	if opts.repo == "" {
		opts.repo = DefaultRepo
	}
	if opts.branch == "" {
		opts.branch = DefaultBranch
	}
	if opts.readmePath == "" {
		opts.readmePath = "README.zh-CN.md"
	}
	if opts.readmeURL == "" {
		opts.readmeURL = fmt.Sprintf("https://github.com/%s/%s/blob/%s/%s", opts.owner, opts.repo, opts.branch, opts.readmePath)
	}

	var items []Item
	category := ""
	var current *Item
	var promptLines []string
	waitingPromptBlock := false
	inPromptBlock := false

	finish := func() {
		if current == nil {
			return
		}
		current.Prompt = strings.TrimSpace(current.Prompt)
		if current.Prompt == "" && len(promptLines) > 0 {
			current.Prompt = strings.TrimSpace(strings.Join(promptLines, "\n"))
		}
		if current.Prompt != "" {
			current.ID = stableID(current.Category, current.Title, current.Prompt)
			current.Images = dedupeImages(current.Images)
			current.Sources = dedupeSources(current.Sources)
			if current.RepoURL == "" {
				current.RepoURL = opts.readmeURL
			}
			items = append(items, *current)
		}
		current = nil
		promptLines = nil
		waitingPromptBlock = false
		inPromptBlock = false
	}

	for _, raw := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		line := strings.TrimSpace(raw)

		if inPromptBlock {
			if strings.HasPrefix(line, "```") {
				if current != nil {
					current.Prompt = strings.TrimSpace(strings.Join(promptLines, "\n"))
				}
				promptLines = nil
				inPromptBlock = false
				continue
			}
			promptLines = append(promptLines, raw)
			continue
		}

		if strings.HasPrefix(line, "## ") && !strings.HasPrefix(line, "### ") {
			finish()
			category = cleanHeading(strings.TrimSpace(strings.TrimPrefix(line, "##")))
			continue
		}

		if strings.HasPrefix(line, "### ") {
			finish()
			title := cleanHeading(strings.TrimSpace(strings.TrimPrefix(line, "###")))
			if title == "" {
				continue
			}
			current = &Item{Title: title, Category: category, RepoURL: opts.readmeURL}
			continue
		}

		if current == nil {
			continue
		}

		for _, img := range extractImages(line, opts) {
			current.Images = append(current.Images, img)
		}

		if isPromptLabel(line) {
			waitingPromptBlock = true
			continue
		}

		if waitingPromptBlock {
			if strings.HasPrefix(line, "```") {
				inPromptBlock = true
				promptLines = nil
				waitingPromptBlock = false
				continue
			}
			if line != "" && !strings.HasPrefix(line, "**") && !strings.HasPrefix(line, "*") {
				current.Prompt = strings.TrimSpace(stripMarkdown(line))
				waitingPromptBlock = false
			}
		}

		if isSourceLine(line) {
			current.Sources = append(current.Sources, extractSources(line, opts)...)
		}
	}
	finish()
	return items
}

func categoriesFromItems(items []Item) []string {
	seen := map[string]bool{}
	var categories []string
	for _, item := range items {
		category := strings.TrimSpace(item.Category)
		if category == "" || seen[category] {
			continue
		}
		seen[category] = true
		categories = append(categories, category)
	}
	return categories
}

func cleanHeading(value string) string {
	value = strings.TrimSpace(strings.Trim(value, "#"))
	value = html.UnescapeString(value)
	value = markdownLinkRe.ReplaceAllString(value, "$1")
	value = strings.Trim(value, " *`\t")
	return strings.Join(strings.Fields(value), " ")
}

func stripMarkdown(value string) string {
	value = html.UnescapeString(value)
	value = strings.Trim(value, " *`\t")
	return markdownLinkRe.ReplaceAllString(value, "$1")
}

func isPromptLabel(line string) bool {
	label := strings.Trim(strings.ToLower(strings.TrimSpace(line)), " *`?:")
	promptLabel := "提示词"
	return label == promptLabel || label == "prompt" || strings.HasPrefix(label, promptLabel) || strings.HasPrefix(label, "prompt")
}

func isSourceLine(line string) bool {
	lower := strings.ToLower(line)
	return strings.Contains(line, "来源") || strings.Contains(lower, "source") || strings.Contains(lower, "via")
}

func extractImages(line string, opts parseOptions) []Image {
	var out []Image
	for _, match := range htmlImageRe.FindAllStringSubmatch(line, -1) {
		urlValue := resolveURL(match[1], opts)
		if urlValue == "" {
			continue
		}
		alt := "image"
		if altMatch := htmlAltRe.FindStringSubmatch(match[0]); len(altMatch) > 1 {
			alt = strings.TrimSpace(html.UnescapeString(altMatch[1]))
		}
		out = append(out, Image{URL: urlValue, Alt: alt})
	}
	for _, match := range markdownImgRe.FindAllStringSubmatch(line, -1) {
		urlValue := resolveURL(match[2], opts)
		if urlValue == "" {
			continue
		}
		out = append(out, Image{URL: urlValue, Alt: strings.TrimSpace(html.UnescapeString(match[1]))})
	}
	return out
}

func extractSources(line string, opts parseOptions) []Source {
	var out []Source
	for _, match := range markdownLinkRe.FindAllStringSubmatch(line, -1) {
		label := strings.TrimSpace(stripMarkdown(match[1]))
		urlValue := resolveURL(match[2], opts)
		if label == "" || urlValue == "" {
			continue
		}
		out = append(out, Source{Label: label, URL: urlValue})
	}
	return out
}

func resolveURL(value string, opts parseOptions) string {
	value = strings.TrimSpace(strings.Trim(value, "<>\"'"))
	if value == "" || strings.HasPrefix(value, "#") || strings.HasPrefix(strings.ToLower(value), "data:") {
		return ""
	}
	if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
		return value
	}
	value = strings.TrimPrefix(value, "./")
	if strings.HasPrefix(value, "/") {
		value = strings.TrimPrefix(value, "/")
	}
	if value == "" {
		return ""
	}
	return fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s", opts.owner, opts.repo, opts.branch, escapePath(value))
}

func escapePath(path string) string {
	parts := strings.Split(path, "/")
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	return strings.Join(parts, "/")
}

func stableID(parts ...string) string {
	h := sha1.New()
	for _, part := range parts {
		_, _ = h.Write([]byte(strings.TrimSpace(part)))
		_, _ = h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))[:12]
}

func dedupeImages(images []Image) []Image {
	seen := map[string]bool{}
	var out []Image
	for _, image := range images {
		image.URL = strings.TrimSpace(image.URL)
		if image.URL == "" || seen[image.URL] {
			continue
		}
		seen[image.URL] = true
		out = append(out, image)
	}
	return out
}

func dedupeSources(sources []Source) []Source {
	seen := map[string]bool{}
	var out []Source
	for _, source := range sources {
		source.URL = strings.TrimSpace(source.URL)
		source.Label = strings.TrimSpace(source.Label)
		if source.URL == "" || seen[source.URL] {
			continue
		}
		seen[source.URL] = true
		out = append(out, source)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Label < out[j].Label })
	return out
}
