package prompttools

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

type Mode string

const (
	ModeTextToPrompt  Mode = "text-to-prompt"
	ModeImageToPrompt Mode = "image-to-prompt"
)

type Source struct {
	Type     string `json:"type"`
	UploadID string `json:"uploadId,omitempty"`
	TaskID   string `json:"taskId,omitempty"`
	Index    int    `json:"index,omitempty"`
}

type TextRequest struct {
	RuntimeAPIKey string `json:"-"`
	Input         string `json:"input"`
	Style         string `json:"style"`
	Ratio         string `json:"ratio"`
	Language      string `json:"language"`
	Target        string `json:"target"`
}

type ImageRequest struct {
	RuntimeAPIKey string `json:"-"`
	Source        Source `json:"source"`
	Language      string `json:"language"`
	Target        string `json:"target"`
}

type Record struct {
	ID              string         `json:"id"`
	SessionID       string         `json:"sessionId,omitempty"`
	VersionID       string         `json:"versionId,omitempty"`
	Mode            Mode           `json:"mode"`
	Input           string         `json:"input,omitempty"`
	Style           string         `json:"style,omitempty"`
	Ratio           string         `json:"ratio,omitempty"`
	Language        string         `json:"language,omitempty"`
	Target          string         `json:"target,omitempty"`
	Source          Source         `json:"source,omitempty"`
	SourceImageURL  string         `json:"sourceImageUrl,omitempty"`
	FlatPrompt      string         `json:"flatPrompt"`
	NegativePrompt  string         `json:"negativePrompt,omitempty"`
	MustKeep        []string       `json:"mustKeep,omitempty"`
	Avoid           []string       `json:"avoid,omitempty"`
	JSONDescription map[string]any `json:"jsonDescription,omitempty"`
	Raw             string         `json:"raw,omitempty"`
	Model           string         `json:"model"`
	ElapsedMs       int64          `json:"elapsedMs"`
	CreatedAt       time.Time      `json:"createdAt"`
}

type SessionKind string

const (
	SessionKindText        SessionKind = "text"
	SessionKindImage       SessionKind = "image"
	SessionKindInspiration SessionKind = "inspiration"
	SessionKindManual      SessionKind = "manual"
)

type PromptSession struct {
	ID              string          `json:"id"`
	Kind            SessionKind     `json:"kind"`
	Title           string          `json:"title"`
	Seed            string          `json:"seed,omitempty"`
	Source          Source          `json:"source,omitempty"`
	SourceImageURL  string          `json:"sourceImageUrl,omitempty"`
	Target          string          `json:"target,omitempty"`
	Provider        string          `json:"provider,omitempty"`
	Model           string          `json:"model,omitempty"`
	Messages        []PromptMessage `json:"messages"`
	Versions        []PromptVersion `json:"versions"`
	ActiveVersionID string          `json:"activeVersionId"`
	CreatedAt       time.Time       `json:"createdAt"`
	UpdatedAt       time.Time       `json:"updatedAt"`
}

type PromptMessage struct {
	ID        string    `json:"id"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	VersionID string    `json:"versionId,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

type PromptVersion struct {
	ID             string    `json:"id"`
	Index          int       `json:"index"`
	Prompt         string    `json:"prompt"`
	NegativePrompt string    `json:"negativePrompt,omitempty"`
	MustKeep       []string  `json:"mustKeep,omitempty"`
	Avoid          []string  `json:"avoid,omitempty"`
	Notes          string    `json:"notes,omitempty"`
	SourceRecordID string    `json:"sourceRecordId,omitempty"`
	Model          string    `json:"model"`
	ElapsedMs      int64     `json:"elapsedMs"`
	CreatedAt      time.Time `json:"createdAt"`
}

type CreateSessionRequest struct {
	Title          string   `json:"title"`
	InitialPrompt  string   `json:"initialPrompt"`
	NegativePrompt string   `json:"negativePrompt"`
	MustKeep       []string `json:"mustKeep"`
	Target         string   `json:"target"`
	Provider       string   `json:"provider"`
	Model          string   `json:"model"`
}

type RefineRequest struct {
	RuntimeAPIKey    string `json:"-"`
	Message          string `json:"message"`
	CurrentVersionID string `json:"currentVersionId"`
	Provider         string `json:"provider"`
	Model            string `json:"model"`
}

type InspirationIdeasRequest struct {
	RuntimeAPIKey string `json:"-"`
	Category      string `json:"category"`
	Mood          string `json:"mood"`
	Style         string `json:"style"`
	Target        string `json:"target"`
	Count         int    `json:"count"`
	Seed          string `json:"seed"`
}

type InspirationIdea struct {
	ID        string   `json:"id"`
	Title     string   `json:"title"`
	Summary   string   `json:"summary"`
	Tags      []string `json:"tags"`
	Category  string   `json:"category,omitempty"`
	Mood      string   `json:"mood,omitempty"`
	Style     string   `json:"style,omitempty"`
	CreatedAt string   `json:"createdAt,omitempty"`
}

type InspirationExpandRequest struct {
	RuntimeAPIKey string          `json:"-"`
	Idea          InspirationIdea `json:"idea"`
	Ratio         string          `json:"ratio"`
	Target        string          `json:"target"`
	Provider      string          `json:"provider"`
	Model         string          `json:"model"`
}

func PublicRecord(record Record) Record {
	record.SourceImageURL = publicSourceImageURL(record.Source, record.SourceImageURL)
	return record
}

func PublicRecords(records []Record) []Record {
	out := append([]Record{}, records...)
	for i := range out {
		out[i] = PublicRecord(out[i])
	}
	return out
}

func PublicSession(session PromptSession) PromptSession {
	session.SourceImageURL = publicSourceImageURL(session.Source, session.SourceImageURL)
	return session
}

func PublicSessions(sessions []PromptSession) []PromptSession {
	out := append([]PromptSession{}, sessions...)
	for i := range out {
		out[i] = PublicSession(out[i])
	}
	return out
}

func publicSourceImageURL(source Source, current string) string {
	if strings.TrimSpace(source.Type) == "result" && strings.TrimSpace(source.TaskID) != "" {
		return fmt.Sprintf("/api/background-tasks/%s/images/%d", url.PathEscape(strings.TrimSpace(source.TaskID)), source.Index)
	}
	return current
}
