import { requestJson } from './client'
import type {
  CanvasProjectListOptions,
  CanvasProjectListResponse,
  CanvasProjectResponse,
  CanvasSnapshotResponse,
  CreateCanvasProjectRequest,
  CreateCanvasSnapshotRequest,
  DeleteCanvasProjectResponse,
  UpdateCanvasProjectRequest,
} from './contracts/canvas'

export type {
  CanvasAssetRef,
  CanvasAssetSource,
  CanvasEdge,
  CanvasEdgeKind,
  CanvasEdgeRole,
  CanvasExecutionStatus,
  CanvasGenerationParameters,
  CanvasGenerationStatus,
  CanvasNode,
  CanvasNodeBase,
  CanvasNodeKind,
  CanvasNodeType,
  CanvasPoint,
  CanvasPromptPart,
  CanvasProject,
  CanvasProjectListOptions,
  CanvasProjectStatus,
  CanvasSize,
  CanvasSnapshot,
  CanvasSnapshotReference,
  CanvasTaskBinding,
  CanvasViewport,
  CreateCanvasSnapshotRequest,
  GenerationNode,
  GenerationNodeData,
  ResultNode,
  ResultNodeData,
} from './contracts/canvas'

export async function listCanvasProjects(options: CanvasProjectListOptions = {}) {
  return requestJson<CanvasProjectListResponse>(canvasProjectListPath(options))
}

export async function createCanvasProject(payload: CreateCanvasProjectRequest = {}) {
  const data = await requestJson<CanvasProjectResponse>('/api/canvas/projects', {
    method: 'POST',
    body: JSON.stringify(payload),
  })
  return data.project
}

export async function getCanvasProject(projectId: string) {
  const data = await requestJson<CanvasProjectResponse>(canvasProjectPath(projectId))
  return data.project
}

export async function updateCanvasProject(projectId: string, payload: UpdateCanvasProjectRequest) {
  const data = await requestJson<CanvasProjectResponse>(canvasProjectPath(projectId), {
    method: 'PATCH',
    body: JSON.stringify(payload),
  })
  return data.project
}

export async function deleteCanvasProject(projectId: string) {
  const data = await requestJson<DeleteCanvasProjectResponse>(canvasProjectPath(projectId), {
    method: 'DELETE',
  })
  return data.project
}

export async function createCanvasSnapshot(projectId: string, payload: CreateCanvasSnapshotRequest) {
  const data = await requestJson<CanvasSnapshotResponse>(`${canvasProjectPath(projectId)}/snapshots`, {
    method: 'POST',
    body: JSON.stringify(payload),
  })
  return data.snapshot
}

function canvasProjectListPath(options: CanvasProjectListOptions) {
  const params = new URLSearchParams()
  if (typeof options.limit === 'number' && Number.isFinite(options.limit) && options.limit > 0) {
    params.set('limit', String(Math.floor(options.limit)))
  }
  const query = params.toString()
  return query ? `/api/canvas/projects?${query}` : '/api/canvas/projects'
}

function canvasProjectPath(projectId: string) {
  return `/api/canvas/projects/${encodeURIComponent(projectId)}`
}
