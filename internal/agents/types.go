package agents

import (
	"time"

	"github.com/y08lin4/lyra-image-workbench/internal/jobs"
)

type SessionStatus string
type RoundStatus string
type Action string

const (
	SessionDraft                SessionStatus = "draft"
	SessionAwaitingConfirmation SessionStatus = "awaiting_confirmation"
	SessionGenerating           SessionStatus = "generating"
	SessionCompleted            SessionStatus = "completed"
	SessionFailed               SessionStatus = "failed"

	RoundPlanning             RoundStatus = "planning"
	RoundAsking               RoundStatus = "asking"
	RoundAwaitingConfirmation RoundStatus = "awaiting_confirmation"
	RoundGenerating           RoundStatus = "generating"
	RoundCompleted            RoundStatus = "completed"
	RoundFailed               RoundStatus = "failed"

	ActionAskQuestion Action = "ask_question"
	ActionProposePlan Action = "propose_plan"
)

type Session struct {
	ID         string        `json:"id"`
	Title      string        `json:"title"`
	Status     SessionStatus `json:"status"`
	Rounds     []Round       `json:"rounds"`
	References []Reference   `json:"references,omitempty"`
	TaskIDs    []string      `json:"taskIds,omitempty"`
	CreatedAt  time.Time     `json:"createdAt"`
	UpdatedAt  time.Time     `json:"updatedAt"`
}

type Round struct {
	ID           string      `json:"id"`
	Index        int         `json:"index"`
	UserMessage  Message     `json:"userMessage"`
	Action       Action      `json:"action"`
	Question     string      `json:"question,omitempty"`
	Assumptions  []string    `json:"assumptions,omitempty"`
	Plan         *Plan       `json:"plan,omitempty"`
	Blocks       []Block     `json:"blocks"`
	ReferenceIDs []string    `json:"referenceIds,omitempty"`
	TaskIDs      []string    `json:"taskIds,omitempty"`
	Status       RoundStatus `json:"status"`
	Error        string      `json:"error,omitempty"`
	Raw          string      `json:"raw,omitempty"`
	CreatedAt    time.Time   `json:"createdAt"`
	FinishedAt   *time.Time  `json:"finishedAt,omitempty"`
}

type Message struct {
	ID        string    `json:"id"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
}

type Block struct {
	Type    string `json:"type"`
	Content string `json:"content,omitempty"`
	Plan    *Plan  `json:"plan,omitempty"`
	TaskID  string `json:"taskId,omitempty"`
}

type Plan struct {
	Title            string           `json:"title"`
	Mode             jobs.Mode        `json:"mode"`
	SceneBrief       string           `json:"sceneBrief"`
	VisualPlan       VisualPlan       `json:"visualPlan"`
	GenerationPrompt string           `json:"generationPrompt"`
	NegativePrompt   string           `json:"negativePrompt,omitempty"`
	Parameters       Parameters       `json:"parameters"`
	ReferenceUsages  []ReferenceUsage `json:"referenceUsages,omitempty"`
	MustKeep         []string         `json:"mustKeep,omitempty"`
	Avoid            []string         `json:"avoid,omitempty"`
	Notes            []string         `json:"notes,omitempty"`
}

type VisualPlan struct {
	Subject     string `json:"subject"`
	Environment string `json:"environment"`
	Camera      string `json:"camera"`
	Composition string `json:"composition"`
	Lighting    string `json:"lighting"`
	Colors      string `json:"colors"`
	Materials   string `json:"materials"`
	Mood        string `json:"mood"`
	Style       string `json:"style"`
}

type Parameters struct {
	Provider     string `json:"provider"`
	Model        string `json:"model"`
	Ratio        string `json:"ratio"`
	Resolution   string `json:"resolution"`
	Size         string `json:"size,omitempty"`
	Quality      string `json:"quality"`
	OutputFormat string `json:"outputFormat"`
	Count        int    `json:"count"`
	Concurrency  int    `json:"concurrency"`
}

type ReferenceUsage struct {
	ReferenceID string   `json:"referenceId"`
	UploadID    string   `json:"uploadId,omitempty"`
	Usage       string   `json:"usage"`
	MustKeep    []string `json:"mustKeep,omitempty"`
	CanChange   []string `json:"canChange,omitempty"`
}

type Reference struct {
	ID           string    `json:"id"`
	SourceType   string    `json:"sourceType"`
	UploadID     string    `json:"uploadId,omitempty"`
	TaskID       string    `json:"taskId,omitempty"`
	ResultIndex  int       `json:"resultIndex,omitempty"`
	OriginalName string    `json:"originalName,omitempty"`
	ImageURL     string    `json:"imageUrl,omitempty"`
	Prompt       string    `json:"prompt,omitempty"`
	Removed      bool      `json:"removed,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
}

type CreateSessionRequest struct {
	Title string `json:"title"`
}

type MessageRequest struct {
	Content       string   `json:"content"`
	ReferenceIDs  []string `json:"referenceIds"`
	Provider      string   `json:"provider"`
	Model         string   `json:"model"`
	Ratio         string   `json:"ratio"`
	SkipQuestions bool     `json:"skipQuestions"`
	RuntimeAPIKey string   `json:"-"`
}

type ConfirmRequest struct {
	Provider       string              `json:"provider"`
	Model          string              `json:"model"`
	Ratio          string              `json:"ratio"`
	Resolution     string              `json:"resolution"`
	Size           string              `json:"size,omitempty"`
	Quality        string              `json:"quality"`
	OutputFormat   string              `json:"outputFormat"`
	Count          int                 `json:"count"`
	Concurrency    int                 `json:"concurrency"`
	UploadIDs      []string            `json:"uploadIds"`
	RuntimeSecrets jobs.RuntimeSecrets `json:"-"`
}
