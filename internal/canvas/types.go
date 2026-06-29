package canvas

import (
	"errors"
	"fmt"
	"time"

	"github.com/y08lin4/lyra-image-workbench/internal/jobs"
)

var (
	ErrStoreNotConfigured = errors.New("canvas store not configured")
	ErrProjectNotFound    = errors.New("canvas project not found")
	ErrInvalidProject     = errors.New("canvas project is invalid")
	ErrRevisionConflict   = errors.New("canvas project revision conflict")
)

type RevisionConflictError struct {
	ProjectID string
	Expected  int64
	Actual    int64
}

func (e RevisionConflictError) Error() string {
	if e.ProjectID == "" {
		return fmt.Sprintf("%s: expected revision %d, actual revision %d", ErrRevisionConflict, e.Expected, e.Actual)
	}
	return fmt.Sprintf("%s: project %s expected revision %d, actual revision %d", ErrRevisionConflict, e.ProjectID, e.Expected, e.Actual)
}

func (e RevisionConflictError) Is(target error) bool {
	return target == ErrRevisionConflict
}

type NodeType string

const (
	NodeTypeText       NodeType = "text"
	NodeTypeImage      NodeType = "image"
	NodeTypeGeneration NodeType = "generation"
	NodeTypeResult     NodeType = "result"
	NodeTypeGroup      NodeType = "group"
)

type EdgeRole string

const (
	EdgeRoleReference EdgeRole = "reference"
	EdgeRoleSubject   EdgeRole = "subject"
	EdgeRoleStyle     EdgeRole = "style"
	EdgeRoleDetail    EdgeRole = "detail"
	EdgeRoleCopy      EdgeRole = "copy"
	EdgeRoleCustom    EdgeRole = "custom"
)

type AssetSource string

const (
	AssetSourceUpload       AssetSource = "upload"
	AssetSourceHistory      AssetSource = "history"
	AssetSourceResult       AssetSource = "result"
	AssetSourceClipboard    AssetSource = "clipboard"
	AssetSourcePromptSquare AssetSource = "prompt-square"
)

type GenerationStatus string

const (
	GenerationStatusIdle      GenerationStatus = "idle"
	GenerationStatusReady     GenerationStatus = "ready"
	GenerationStatusCreating  GenerationStatus = "creating"
	GenerationStatusRunning   GenerationStatus = "running"
	GenerationStatusCompleted GenerationStatus = "completed"
	GenerationStatusFailed    GenerationStatus = "failed"
)

type Project struct {
	ID          string     `json:"id"`
	SpaceToken  string     `json:"spaceToken,omitempty"`
	OwnerUserID string     `json:"ownerUserId,omitempty"`
	Title       string     `json:"title"`
	Revision    int64      `json:"revision"`
	Viewport    Viewport   `json:"viewport"`
	Nodes       []Node     `json:"nodes"`
	Edges       []Edge     `json:"edges"`
	Assets      []AssetRef `json:"assets,omitempty"`
	Snapshots   []Snapshot `json:"snapshots,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

type Viewport struct {
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
	Zoom float64 `json:"zoom"`
}

type Node struct {
	ID                   string           `json:"id"`
	Type                 NodeType         `json:"type"`
	Name                 string           `json:"name,omitempty"`
	X                    float64          `json:"x"`
	Y                    float64          `json:"y"`
	Width                float64          `json:"width"`
	Height               float64          `json:"height"`
	Rotation             float64          `json:"rotation"`
	ZIndex               int              `json:"zIndex"`
	Text                 string           `json:"text,omitempty"`
	Role                 EdgeRole         `json:"role,omitempty"`
	IsReference          bool             `json:"isReference,omitempty"`
	AssetID              string           `json:"assetId,omitempty"`
	Source               AssetSource      `json:"source,omitempty"`
	UploadID             string           `json:"uploadId,omitempty"`
	TaskID               string           `json:"taskId,omitempty"`
	ResultIndex          int              `json:"resultIndex,omitempty"`
	ImageURL             string           `json:"imageUrl,omitempty"`
	ThumbnailURL         string           `json:"thumbnailUrl,omitempty"`
	OriginalURL          string           `json:"originalUrl,omitempty"`
	NaturalWidth         int              `json:"naturalWidth,omitempty"`
	NaturalHeight        int              `json:"naturalHeight,omitempty"`
	PromptSnapshot       string           `json:"promptSnapshot,omitempty"`
	ReferenceSnapshotIDs []string         `json:"referenceSnapshotIds,omitempty"`
	Mode                 jobs.Mode        `json:"mode,omitempty"`
	Provider             string           `json:"provider,omitempty"`
	Model                string           `json:"model,omitempty"`
	Ratio                string           `json:"ratio,omitempty"`
	Resolution           string           `json:"resolution,omitempty"`
	Quality              string           `json:"quality,omitempty"`
	OutputFormat         string           `json:"outputFormat,omitempty"`
	Count                int              `json:"count,omitempty"`
	Concurrency          int              `json:"concurrency,omitempty"`
	Status               GenerationStatus `json:"status,omitempty"`
	TaskIDs              []string         `json:"taskIds,omitempty"`
	CreatedAt            time.Time        `json:"createdAt"`
	UpdatedAt            time.Time        `json:"updatedAt"`
	Metadata             map[string]any   `json:"metadata,omitempty"`
}

type Edge struct {
	ID         string    `json:"id"`
	FromNodeID string    `json:"fromNodeId"`
	ToNodeID   string    `json:"toNodeId"`
	Role       EdgeRole  `json:"role"`
	Label      string    `json:"label,omitempty"`
	Text       string    `json:"text,omitempty"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

type AssetRef struct {
	ID             string      `json:"id"`
	Source         AssetSource `json:"source"`
	UploadID       string      `json:"uploadId,omitempty"`
	TaskID         string      `json:"taskId,omitempty"`
	ResultIndex    int         `json:"resultIndex,omitempty"`
	URL            string      `json:"url,omitempty"`
	ThumbnailURL   string      `json:"thumbnailUrl,omitempty"`
	Mime           string      `json:"mime,omitempty"`
	Size           int64       `json:"size,omitempty"`
	Width          int         `json:"width,omitempty"`
	Height         int         `json:"height,omitempty"`
	OriginalName   string      `json:"originalName,omitempty"`
	PromptSnapshot string      `json:"promptSnapshot,omitempty"`
	CreatedAt      time.Time   `json:"createdAt"`
}

type GenerationParameters struct {
	Mode         jobs.Mode `json:"mode"`
	Provider     string    `json:"provider,omitempty"`
	Model        string    `json:"model,omitempty"`
	Ratio        string    `json:"ratio,omitempty"`
	Resolution   string    `json:"resolution,omitempty"`
	Quality      string    `json:"quality,omitempty"`
	OutputFormat string    `json:"outputFormat,omitempty"`
	Count        int       `json:"count,omitempty"`
	Concurrency  int       `json:"concurrency,omitempty"`
	UploadIDs    []string  `json:"uploadIds,omitempty"`
}

type PromptPart struct {
	NodeID    string `json:"nodeId"`
	Text      string `json:"text"`
	EdgeLabel string `json:"edgeLabel,omitempty"`
}

type SnapshotReference struct {
	NodeID              string      `json:"nodeId"`
	AssetID             string      `json:"assetId,omitempty"`
	Role                EdgeRole    `json:"role,omitempty"`
	EdgeLabel           string      `json:"edgeLabel,omitempty"`
	ReferenceSnapshotID string      `json:"snapshotId,omitempty"`
	Source              AssetSource `json:"source,omitempty"`
	UploadID            string      `json:"uploadId,omitempty"`
	TaskID              string      `json:"taskId,omitempty"`
	ResultIndex         int         `json:"resultIndex,omitempty"`
	OriginalName        string      `json:"originalName,omitempty"`
	FileName            string      `json:"fileName,omitempty"`
	Mime                string      `json:"mime,omitempty"`
	Size                int64       `json:"size,omitempty"`
	PromptSnapshot      string      `json:"promptSnapshot,omitempty"`
}

type Snapshot struct {
	ID               string               `json:"id"`
	ProjectID        string               `json:"projectId"`
	ProjectRevision  int64                `json:"projectRevision"`
	GenerationNodeID string               `json:"generationNodeId"`
	ContextHash      string               `json:"contextHash"`
	ResolvedPrompt   string               `json:"resolvedPrompt"`
	PromptParts      []PromptPart         `json:"promptParts,omitempty"`
	References       []SnapshotReference  `json:"references,omitempty"`
	Parameters       GenerationParameters `json:"parameters"`
	UploadIDs        []string             `json:"uploadIds,omitempty"`
	SourceNodeIDs    []string             `json:"sourceNodeIds,omitempty"`
	Metadata         map[string]any       `json:"metadata,omitempty"`
	CreatedAt        time.Time            `json:"createdAt"`
}

type TaskBinding struct {
	ProjectID     string    `json:"projectId"`
	SnapshotID    string    `json:"snapshotId"`
	SourceNodeIDs []string  `json:"sourceNodeIds"`
	TargetNodeID  string    `json:"targetNodeId,omitempty"`
	CreatedNodeID string    `json:"createdNodeId,omitempty"`
	ContextHash   string    `json:"contextHash"`
	CreatedAt     time.Time `json:"createdAt"`
}

type CreateProjectRequest struct {
	OwnerUserID string     `json:"ownerUserId,omitempty"`
	Title       string     `json:"title"`
	Viewport    Viewport   `json:"viewport"`
	Nodes       []Node     `json:"nodes,omitempty"`
	Edges       []Edge     `json:"edges,omitempty"`
	Assets      []AssetRef `json:"assets,omitempty"`
}

type UpdateProjectRequest struct {
	ExpectedRevision int64       `json:"revision"`
	Title            *string     `json:"title,omitempty"`
	Viewport         *Viewport   `json:"viewport,omitempty"`
	Nodes            *[]Node     `json:"nodes,omitempty"`
	Edges            *[]Edge     `json:"edges,omitempty"`
	Assets           *[]AssetRef `json:"assets,omitempty"`
}

type CreateSnapshotRequest struct {
	ExpectedRevision int64                `json:"revision,omitempty"`
	GenerationNodeID string               `json:"generationNodeId"`
	ResolvedPrompt   string               `json:"resolvedPrompt"`
	PromptParts      []PromptPart         `json:"promptParts,omitempty"`
	References       []SnapshotReference  `json:"references,omitempty"`
	Parameters       GenerationParameters `json:"parameters"`
	UploadIDs        []string             `json:"uploadIds,omitempty"`
	SourceNodeIDs    []string             `json:"sourceNodeIds,omitempty"`
	Metadata         map[string]any       `json:"metadata,omitempty"`
}
