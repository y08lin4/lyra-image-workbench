package canvas

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) CreateProject(spaceToken string, req CreateProjectRequest) (Project, error) {
	if err := s.ensureReady(); err != nil {
		return Project{}, err
	}
	id, err := newCanvasID("cvp")
	if err != nil {
		return Project{}, err
	}
	now := time.Now()
	project := Project{
		ID:          id,
		SpaceToken:  strings.TrimSpace(spaceToken),
		OwnerUserID: strings.TrimSpace(req.OwnerUserID),
		Title:       defaultProjectTitle(req.Title),
		Revision:    1,
		Viewport:    normalizeViewport(req.Viewport),
		Nodes:       req.Nodes,
		Edges:       req.Edges,
		Assets:      req.Assets,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := normalizeProject(&project, now); err != nil {
		return Project{}, err
	}
	return project, s.store.Create(spaceToken, project)
}

func (s *Service) ListProjects(spaceToken string, limit int) ([]Project, error) {
	if err := s.ensureReady(); err != nil {
		return nil, err
	}
	return s.store.List(spaceToken, limit)
}

func (s *Service) GetProject(spaceToken string, id string) (Project, bool, error) {
	if err := s.ensureReady(); err != nil {
		return Project{}, false, err
	}
	return s.store.Get(spaceToken, strings.TrimSpace(id))
}

func (s *Service) UpdateProject(spaceToken string, id string, req UpdateProjectRequest) (Project, error) {
	if err := s.ensureReady(); err != nil {
		return Project{}, err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return Project{}, fmt.Errorf("%w: project id is required", ErrInvalidProject)
	}
	if req.ExpectedRevision <= 0 {
		return Project{}, fmt.Errorf("%w: expected revision is required", ErrInvalidProject)
	}
	now := time.Now()
	project, found, err := s.store.Update(spaceToken, id, req.ExpectedRevision, func(project *Project) error {
		if req.Title != nil {
			project.Title = defaultProjectTitle(*req.Title)
		}
		if req.Viewport != nil {
			project.Viewport = normalizeViewport(*req.Viewport)
		}
		if req.Nodes != nil {
			project.Nodes = *req.Nodes
		}
		if req.Edges != nil {
			project.Edges = *req.Edges
		}
		if req.Assets != nil {
			project.Assets = *req.Assets
		}
		project.Revision++
		project.UpdatedAt = now
		return normalizeProject(project, now)
	})
	if err != nil {
		return project, err
	}
	if !found {
		return Project{}, ErrProjectNotFound
	}
	return project, nil
}

func (s *Service) DeleteProject(spaceToken string, id string) (Project, bool, error) {
	if err := s.ensureReady(); err != nil {
		return Project{}, false, err
	}
	return s.store.Delete(spaceToken, strings.TrimSpace(id))
}

func (s *Service) CreateSnapshot(spaceToken string, projectID string, req CreateSnapshotRequest) (Snapshot, error) {
	if err := s.ensureReady(); err != nil {
		return Snapshot{}, err
	}
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return Snapshot{}, fmt.Errorf("%w: project id is required", ErrInvalidProject)
	}
	if req.ExpectedRevision <= 0 {
		return Snapshot{}, fmt.Errorf("%w: expected revision is required", ErrInvalidProject)
	}
	now := time.Now()
	var snapshot Snapshot
	project, found, err := s.store.Update(spaceToken, projectID, req.ExpectedRevision, func(project *Project) error {
		next, err := newSnapshot(*project, req, now)
		if err != nil {
			return err
		}
		snapshot = next
		project.Snapshots = append(project.Snapshots, snapshot)
		project.Revision++
		project.UpdatedAt = now
		return normalizeProject(project, now)
	})
	if err != nil {
		return snapshot, err
	}
	if !found {
		return Snapshot{}, ErrProjectNotFound
	}
	_ = project
	return snapshot, nil
}

func BindingFromSnapshot(snapshot Snapshot, targetNodeID string, createdNodeID string) TaskBinding {
	return TaskBinding{
		ProjectID:     strings.TrimSpace(snapshot.ProjectID),
		SnapshotID:    strings.TrimSpace(snapshot.ID),
		SourceNodeIDs: cleanUniqueStrings(snapshot.SourceNodeIDs),
		TargetNodeID:  strings.TrimSpace(targetNodeID),
		CreatedNodeID: strings.TrimSpace(createdNodeID),
		ContextHash:   strings.TrimSpace(snapshot.ContextHash),
		CreatedAt:     time.Now(),
	}
}

func (s *Service) ensureReady() error {
	if s == nil || s.store == nil {
		return ErrStoreNotConfigured
	}
	return nil
}

func newSnapshot(project Project, req CreateSnapshotRequest, now time.Time) (Snapshot, error) {
	id, err := newCanvasID("cvs")
	if err != nil {
		return Snapshot{}, err
	}
	req.GenerationNodeID = strings.TrimSpace(req.GenerationNodeID)
	req.ResolvedPrompt = strings.TrimSpace(req.ResolvedPrompt)
	req.Parameters = normalizeGenerationParameters(req.Parameters)
	req.PromptParts = normalizePromptParts(req.PromptParts)
	req.References = normalizeSnapshotReferences(req.References)
	req.UploadIDs = cleanUniqueStrings(append(req.UploadIDs, req.Parameters.UploadIDs...))
	if len(req.UploadIDs) == 0 {
		req.UploadIDs = uploadIDsFromReferences(req.References)
	}
	req.SourceNodeIDs = cleanUniqueStrings(req.SourceNodeIDs)
	if len(req.SourceNodeIDs) == 0 {
		req.SourceNodeIDs = sourceNodeIDs(req.GenerationNodeID, req.PromptParts, req.References)
	}
	if req.GenerationNodeID == "" {
		return Snapshot{}, fmt.Errorf("%w: generation node id is required", ErrInvalidProject)
	}
	if req.ResolvedPrompt == "" {
		return Snapshot{}, fmt.Errorf("%w: resolved prompt is required", ErrInvalidProject)
	}
	snapshot := Snapshot{
		ID:               id,
		ProjectID:        project.ID,
		ProjectRevision:  project.Revision,
		GenerationNodeID: req.GenerationNodeID,
		ResolvedPrompt:   req.ResolvedPrompt,
		PromptParts:      req.PromptParts,
		References:       req.References,
		Parameters:       req.Parameters,
		UploadIDs:        req.UploadIDs,
		SourceNodeIDs:    req.SourceNodeIDs,
		Metadata:         req.Metadata,
		CreatedAt:        now,
	}
	hash, err := contextHash(snapshot)
	if err != nil {
		return Snapshot{}, err
	}
	snapshot.ContextHash = hash
	return snapshot, nil
}

func normalizeProject(project *Project, now time.Time) error {
	project.ID = strings.TrimSpace(project.ID)
	project.SpaceToken = strings.TrimSpace(project.SpaceToken)
	project.OwnerUserID = strings.TrimSpace(project.OwnerUserID)
	project.Title = defaultProjectTitle(project.Title)
	project.Viewport = normalizeViewport(project.Viewport)
	if project.ID == "" {
		return fmt.Errorf("%w: project id is required", ErrInvalidProject)
	}
	if project.Revision <= 0 {
		project.Revision = 1
	}
	if project.CreatedAt.IsZero() {
		project.CreatedAt = now
	}
	if project.UpdatedAt.IsZero() {
		project.UpdatedAt = project.CreatedAt
	}
	nodeIDs := map[string]bool{}
	for i := range project.Nodes {
		node := &project.Nodes[i]
		node.ID = strings.TrimSpace(node.ID)
		node.Name = strings.TrimSpace(node.Name)
		node.Text = strings.TrimSpace(node.Text)
		node.AssetID = strings.TrimSpace(node.AssetID)
		node.UploadID = strings.TrimSpace(node.UploadID)
		node.TaskID = strings.TrimSpace(node.TaskID)
		node.ImageURL = strings.TrimSpace(node.ImageURL)
		node.ThumbnailURL = strings.TrimSpace(node.ThumbnailURL)
		node.OriginalURL = strings.TrimSpace(node.OriginalURL)
		node.PromptSnapshot = strings.TrimSpace(node.PromptSnapshot)
		node.Provider = strings.TrimSpace(node.Provider)
		node.Model = strings.TrimSpace(node.Model)
		node.Ratio = strings.TrimSpace(node.Ratio)
		node.Resolution = strings.TrimSpace(node.Resolution)
		node.Quality = strings.TrimSpace(node.Quality)
		node.OutputFormat = strings.TrimSpace(node.OutputFormat)
		node.TaskIDs = cleanUniqueStrings(node.TaskIDs)
		node.ReferenceSnapshotIDs = cleanUniqueStrings(node.ReferenceSnapshotIDs)
		if node.ID == "" {
			return fmt.Errorf("%w: node id is required", ErrInvalidProject)
		}
		if nodeIDs[node.ID] {
			return fmt.Errorf("%w: duplicate node id %s", ErrInvalidProject, node.ID)
		}
		if node.Type == "" {
			return fmt.Errorf("%w: node type is required", ErrInvalidProject)
		}
		if node.Width < 0 || node.Height < 0 {
			return fmt.Errorf("%w: node size must be non-negative", ErrInvalidProject)
		}
		if node.CreatedAt.IsZero() {
			node.CreatedAt = now
		}
		if node.UpdatedAt.IsZero() {
			node.UpdatedAt = node.CreatedAt
		}
		nodeIDs[node.ID] = true
	}
	edgeIDs := map[string]bool{}
	for i := range project.Edges {
		edge := &project.Edges[i]
		edge.ID = strings.TrimSpace(edge.ID)
		edge.FromNodeID = strings.TrimSpace(edge.FromNodeID)
		edge.ToNodeID = strings.TrimSpace(edge.ToNodeID)
		edge.Label = strings.TrimSpace(edge.Label)
		edge.Text = strings.TrimSpace(edge.Text)
		if edge.ID == "" {
			return fmt.Errorf("%w: edge id is required", ErrInvalidProject)
		}
		if edgeIDs[edge.ID] {
			return fmt.Errorf("%w: duplicate edge id %s", ErrInvalidProject, edge.ID)
		}
		if edge.FromNodeID == "" || edge.ToNodeID == "" || edge.FromNodeID == edge.ToNodeID {
			return fmt.Errorf("%w: edge endpoints are invalid", ErrInvalidProject)
		}
		if !nodeIDs[edge.FromNodeID] || !nodeIDs[edge.ToNodeID] {
			return fmt.Errorf("%w: edge references missing node", ErrInvalidProject)
		}
		if edge.Role == "" {
			edge.Role = EdgeRoleReference
		}
		if edge.CreatedAt.IsZero() {
			edge.CreatedAt = now
		}
		if edge.UpdatedAt.IsZero() {
			edge.UpdatedAt = edge.CreatedAt
		}
		edgeIDs[edge.ID] = true
	}
	assetIDs := map[string]bool{}
	for i := range project.Assets {
		asset := &project.Assets[i]
		asset.ID = strings.TrimSpace(asset.ID)
		asset.UploadID = strings.TrimSpace(asset.UploadID)
		asset.TaskID = strings.TrimSpace(asset.TaskID)
		asset.URL = strings.TrimSpace(asset.URL)
		asset.ThumbnailURL = strings.TrimSpace(asset.ThumbnailURL)
		asset.Mime = strings.TrimSpace(asset.Mime)
		asset.OriginalName = strings.TrimSpace(asset.OriginalName)
		asset.PromptSnapshot = strings.TrimSpace(asset.PromptSnapshot)
		if asset.ID == "" {
			return fmt.Errorf("%w: asset id is required", ErrInvalidProject)
		}
		if assetIDs[asset.ID] {
			return fmt.Errorf("%w: duplicate asset id %s", ErrInvalidProject, asset.ID)
		}
		if asset.CreatedAt.IsZero() {
			asset.CreatedAt = now
		}
		assetIDs[asset.ID] = true
	}
	return nil
}

func normalizeViewport(viewport Viewport) Viewport {
	if viewport.Zoom <= 0 {
		viewport.Zoom = 1
	}
	return viewport
}

func normalizeGenerationParameters(params GenerationParameters) GenerationParameters {
	params.Provider = strings.TrimSpace(params.Provider)
	params.Model = strings.TrimSpace(params.Model)
	params.Ratio = strings.TrimSpace(params.Ratio)
	params.Resolution = strings.TrimSpace(params.Resolution)
	params.Quality = strings.TrimSpace(params.Quality)
	params.OutputFormat = strings.TrimSpace(params.OutputFormat)
	params.UploadIDs = cleanUniqueStrings(params.UploadIDs)
	if params.Count <= 0 {
		params.Count = 1
	}
	if params.Concurrency <= 0 {
		params.Concurrency = 1
	}
	if params.Concurrency > params.Count {
		params.Concurrency = params.Count
	}
	return params
}

func normalizePromptParts(parts []PromptPart) []PromptPart {
	out := make([]PromptPart, 0, len(parts))
	for _, part := range parts {
		part.NodeID = strings.TrimSpace(part.NodeID)
		part.Text = strings.TrimSpace(part.Text)
		part.EdgeLabel = strings.TrimSpace(part.EdgeLabel)
		if part.NodeID == "" || part.Text == "" {
			continue
		}
		out = append(out, part)
	}
	return out
}

func normalizeSnapshotReferences(refs []SnapshotReference) []SnapshotReference {
	out := make([]SnapshotReference, 0, len(refs))
	for _, ref := range refs {
		ref.NodeID = strings.TrimSpace(ref.NodeID)
		ref.AssetID = strings.TrimSpace(ref.AssetID)
		ref.EdgeLabel = strings.TrimSpace(ref.EdgeLabel)
		ref.ReferenceSnapshotID = strings.TrimSpace(ref.ReferenceSnapshotID)
		ref.UploadID = strings.TrimSpace(ref.UploadID)
		ref.TaskID = strings.TrimSpace(ref.TaskID)
		ref.OriginalName = strings.TrimSpace(ref.OriginalName)
		ref.FileName = strings.TrimSpace(ref.FileName)
		ref.Mime = strings.TrimSpace(ref.Mime)
		ref.PromptSnapshot = strings.TrimSpace(ref.PromptSnapshot)
		if ref.NodeID == "" && ref.AssetID == "" && ref.UploadID == "" && ref.TaskID == "" {
			continue
		}
		if ref.Role == "" {
			ref.Role = EdgeRoleReference
		}
		out = append(out, ref)
	}
	return out
}

func uploadIDsFromReferences(refs []SnapshotReference) []string {
	values := make([]string, 0, len(refs))
	for _, ref := range refs {
		values = append(values, ref.UploadID)
	}
	return cleanUniqueStrings(values)
}

func sourceNodeIDs(generationNodeID string, parts []PromptPart, refs []SnapshotReference) []string {
	values := make([]string, 0, len(parts)+len(refs))
	for _, part := range parts {
		if part.NodeID != generationNodeID {
			values = append(values, part.NodeID)
		}
	}
	for _, ref := range refs {
		if ref.NodeID != generationNodeID {
			values = append(values, ref.NodeID)
		}
	}
	return cleanUniqueStrings(values)
}

func contextHash(snapshot Snapshot) (string, error) {
	payload := struct {
		ProjectID        string               `json:"projectId"`
		ProjectRevision  int64                `json:"projectRevision"`
		GenerationNodeID string               `json:"generationNodeId"`
		ResolvedPrompt   string               `json:"resolvedPrompt"`
		PromptParts      []PromptPart         `json:"promptParts,omitempty"`
		References       []SnapshotReference  `json:"references,omitempty"`
		Parameters       GenerationParameters `json:"parameters"`
		UploadIDs        []string             `json:"uploadIds,omitempty"`
		SourceNodeIDs    []string             `json:"sourceNodeIds,omitempty"`
	}{
		ProjectID:        snapshot.ProjectID,
		ProjectRevision:  snapshot.ProjectRevision,
		GenerationNodeID: snapshot.GenerationNodeID,
		ResolvedPrompt:   snapshot.ResolvedPrompt,
		PromptParts:      snapshot.PromptParts,
		References:       snapshot.References,
		Parameters:       snapshot.Parameters,
		UploadIDs:        snapshot.UploadIDs,
		SourceNodeIDs:    snapshot.SourceNodeIDs,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func cleanUniqueStrings(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func defaultProjectTitle(value string) string {
	title := strings.Join(strings.Fields(value), " ")
	if title == "" {
		return "Untitled canvas"
	}
	runes := []rune(title)
	if len(runes) > 64 {
		return string(runes[:64]) + "..."
	}
	return title
}

func newCanvasID(prefix string) (string, error) {
	var bytes [8]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", err
	}
	return prefix + "_" + time.Now().Format("20060102150405") + "_" + hex.EncodeToString(bytes[:]), nil
}
