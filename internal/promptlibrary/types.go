package promptlibrary

import "time"

const (
	DefaultOwner  = "ZeroLu"
	DefaultRepo   = "awesome-gpt-image"
	DefaultBranch = "main"
	DefaultLang   = "zh-CN"
)

type Image struct {
	URL         string `json:"url"`
	Alt         string `json:"alt,omitempty"`
	OriginalURL string `json:"originalUrl,omitempty"`
}

type Source struct {
	Label string `json:"label"`
	URL   string `json:"url"`
}

type Item struct {
	ID       string   `json:"id"`
	Title    string   `json:"title"`
	Category string   `json:"category"`
	Prompt   string   `json:"prompt"`
	Images   []Image  `json:"images,omitempty"`
	Sources  []Source `json:"sources,omitempty"`
	RepoURL  string   `json:"repoUrl"`
}

type Library struct {
	Repo       string    `json:"repo"`
	Lang       string    `json:"lang"`
	SourceURL  string    `json:"sourceUrl"`
	ReadmeURL  string    `json:"readmeUrl"`
	FetchedAt  time.Time `json:"fetchedAt"`
	ContentSHA string    `json:"contentSha,omitempty"`
	ETag       string    `json:"-"`
	Stale      bool      `json:"stale"`
	FetchError string    `json:"fetchError,omitempty"`
	Categories []string  `json:"categories"`
	Total      int       `json:"total"`
	Matching   int       `json:"matching"`
	Items      []Item    `json:"items"`
}

type Query struct {
	Lang     string
	Q        string
	Category string
	Limit    int
	Force    bool
}

func cloneLibrary(lib Library) Library {
	lib.Categories = append([]string(nil), lib.Categories...)
	if len(lib.Items) > 0 {
		items := make([]Item, len(lib.Items))
		for index, item := range lib.Items {
			item.Images = append([]Image(nil), item.Images...)
			item.Sources = append([]Source(nil), item.Sources...)
			items[index] = item
		}
		lib.Items = items
	}
	return lib
}
