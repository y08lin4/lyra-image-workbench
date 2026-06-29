package canvas

import (
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/y08lin4/lyra-image-workbench/internal/jobs"
	"github.com/y08lin4/lyra-image-workbench/internal/spaces"
)

func TestServiceProjectCRUDUsesRevisionLock(t *testing.T) {
	service := NewService(NewMemoryStore())
	created, err := service.CreateProject("space-a", CreateProjectRequest{
		Title: "  Draft canvas  ",
		Nodes: []Node{{
			ID:     "txt-1",
			Type:   NodeTypeText,
			Text:   "  first idea  ",
			Width:  120,
			Height: 80,
		}},
	})
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}
	if created.Revision != 1 || created.Title != "Draft canvas" || created.Viewport.Zoom != 1 {
		t.Fatalf("created project was not normalized: %+v", created)
	}
	if created.Nodes[0].Text != "first idea" {
		t.Fatalf("node text was not normalized: %+v", created.Nodes[0])
	}

	title := "Updated canvas"
	updated, err := service.UpdateProject("space-a", created.ID, UpdateProjectRequest{
		ExpectedRevision: created.Revision,
		Title:            &title,
	})
	if err != nil {
		t.Fatalf("UpdateProject() error = %v", err)
	}
	if updated.Revision != 2 || updated.Title != title {
		t.Fatalf("updated project mismatch: %+v", updated)
	}

	staleTitle := "Stale write"
	_, err = service.UpdateProject("space-a", created.ID, UpdateProjectRequest{
		ExpectedRevision: created.Revision,
		Title:            &staleTitle,
	})
	if !errors.Is(err, ErrRevisionConflict) {
		t.Fatalf("UpdateProject(stale) error = %v, want ErrRevisionConflict", err)
	}
	got, ok, err := service.GetProject("space-a", created.ID)
	if err != nil || !ok {
		t.Fatalf("GetProject() ok=%v err=%v", ok, err)
	}
	if got.Title != title || got.Revision != 2 {
		t.Fatalf("stale update changed stored project: %+v", got)
	}

	deleted, ok, err := service.DeleteProject("space-a", created.ID)
	if err != nil || !ok {
		t.Fatalf("DeleteProject() ok=%v err=%v", ok, err)
	}
	if deleted.ID != created.ID {
		t.Fatalf("deleted project mismatch: %+v", deleted)
	}
	if _, ok, err := service.GetProject("space-a", created.ID); err != nil || ok {
		t.Fatalf("project still exists after delete: ok=%v err=%v", ok, err)
	}
}

func TestFileStorePersistsProjectsBySpace(t *testing.T) {
	root := t.TempDir()
	spaceStore, err := spaces.NewFileStore(filepath.Join(root, "data"))
	if err != nil {
		t.Fatalf("spaces.NewFileStore() error = %v", err)
	}
	session, err := spaceStore.CreateOrOpenByPassword("R7!Blue#Vault$2026")
	if err != nil {
		t.Fatalf("CreateOrOpenByPassword() error = %v", err)
	}
	service := NewService(NewFileStore(spaceStore))
	created, err := service.CreateProject(session.Token, CreateProjectRequest{Title: "Persistent canvas"})
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	reopened := NewService(NewFileStore(spaceStore))
	got, ok, err := reopened.GetProject(session.Token, created.ID)
	if err != nil || !ok {
		t.Fatalf("GetProject(reopened) ok=%v err=%v", ok, err)
	}
	if got.ID != created.ID || got.SpaceToken != session.Token || got.Title != "Persistent canvas" {
		t.Fatalf("persisted project mismatch: %+v", got)
	}
	list, err := reopened.ListProjects(session.Token, 10)
	if err != nil {
		t.Fatalf("ListProjects() error = %v", err)
	}
	if len(list) != 1 || list[0].ID != created.ID {
		t.Fatalf("ListProjects() = %+v", list)
	}
}

func TestServiceCreatesSnapshotsAndTaskBindings(t *testing.T) {
	service := NewService(NewMemoryStore())
	project, err := service.CreateProject("space-a", CreateProjectRequest{
		Title: "Snapshot project",
		Nodes: []Node{
			{ID: "text-1", Type: NodeTypeText, Text: "cat", Width: 120, Height: 80},
			{ID: "image-1", Type: NodeTypeImage, UploadID: "upload-1", Width: 220, Height: 156},
			{ID: "gen-1", Type: NodeTypeGeneration, Width: 260, Height: 180},
		},
		Edges: []Edge{{
			ID:         "edge-1",
			FromNodeID: "image-1",
			ToNodeID:   "gen-1",
			Role:       EdgeRoleSubject,
		}},
	})
	if err != nil {
		t.Fatalf("CreateProject() error = %v", err)
	}

	_, err = service.CreateSnapshot("space-a", project.ID, CreateSnapshotRequest{
		GenerationNodeID: "gen-1",
		ResolvedPrompt:   "missing revision",
	})
	if !errors.Is(err, ErrInvalidProject) {
		t.Fatalf("CreateSnapshot(missing revision) error = %v, want ErrInvalidProject", err)
	}

	snapshot, err := service.CreateSnapshot("space-a", project.ID, CreateSnapshotRequest{
		ExpectedRevision: project.Revision,
		GenerationNodeID: "gen-1",
		ResolvedPrompt:   "a sharp studio cat portrait",
		PromptParts: []PromptPart{{
			NodeID:    "text-1",
			Text:      "a sharp studio cat portrait",
			EdgeLabel: "prompt",
		}},
		References: []SnapshotReference{{
			NodeID:   "image-1",
			Role:     EdgeRoleSubject,
			UploadID: "upload-1",
		}},
		Parameters: GenerationParameters{
			Mode:        jobs.ModeImageToImage,
			Count:       2,
			Concurrency: 9,
			UploadIDs:   []string{"upload-1", "upload-2", "upload-1"},
		},
	})
	if err != nil {
		t.Fatalf("CreateSnapshot() error = %v", err)
	}
	if snapshot.ProjectRevision != project.Revision {
		t.Fatalf("snapshot revision = %d, want %d", snapshot.ProjectRevision, project.Revision)
	}
	if !strings.HasPrefix(snapshot.ContextHash, "sha256:") {
		t.Fatalf("snapshot context hash was not set: %+v", snapshot)
	}
	if !reflect.DeepEqual(snapshot.UploadIDs, []string{"upload-1", "upload-2"}) {
		t.Fatalf("snapshot upload IDs = %+v", snapshot.UploadIDs)
	}
	if !reflect.DeepEqual(snapshot.SourceNodeIDs, []string{"text-1", "image-1"}) {
		t.Fatalf("snapshot source nodes = %+v", snapshot.SourceNodeIDs)
	}
	if snapshot.Parameters.Concurrency != 2 {
		t.Fatalf("snapshot concurrency should be clamped to count: %+v", snapshot.Parameters)
	}

	got, ok, err := service.GetProject("space-a", project.ID)
	if err != nil || !ok {
		t.Fatalf("GetProject() ok=%v err=%v", ok, err)
	}
	if got.Revision != 2 || len(got.Snapshots) != 1 || got.Snapshots[0].ID != snapshot.ID {
		t.Fatalf("snapshot was not persisted on project: %+v", got)
	}

	binding := BindingFromSnapshot(snapshot, "gen-1", "result-1")
	if binding.ProjectID != project.ID || binding.SnapshotID != snapshot.ID || binding.ContextHash != snapshot.ContextHash {
		t.Fatalf("binding mismatch: %+v", binding)
	}
	if !reflect.DeepEqual(binding.SourceNodeIDs, snapshot.SourceNodeIDs) {
		t.Fatalf("binding source nodes = %+v, want %+v", binding.SourceNodeIDs, snapshot.SourceNodeIDs)
	}

	_, err = service.CreateSnapshot("space-a", project.ID, CreateSnapshotRequest{
		ExpectedRevision: project.Revision,
		GenerationNodeID: "gen-1",
		ResolvedPrompt:   "stale snapshot",
	})
	if !errors.Is(err, ErrRevisionConflict) {
		t.Fatalf("CreateSnapshot(stale) error = %v, want ErrRevisionConflict", err)
	}
}
