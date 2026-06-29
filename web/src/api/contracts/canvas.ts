import type { Mode, ModelProvider } from './tasks'

export type CanvasProjectStatus = 'draft' | 'ready' | 'generating' | 'archived' | (string & {})
export type CanvasNodeType = 'text' | 'image' | 'generation' | 'result' | 'group' | (string & {})
export type CanvasNodeKind = CanvasNodeType
export type CanvasEdgeRole = 'reference' | 'subject' | 'style' | 'detail' | 'copy' | 'custom' | (string & {})
export type CanvasEdgeKind = CanvasEdgeRole
export type CanvasAssetSource = 'upload' | 'history' | 'result' | 'clipboard' | 'prompt-square' | (string & {})
export type CanvasGenerationStatus = 'idle' | 'ready' | 'creating' | 'running' | 'completed' | 'failed' | (string & {})
export type CanvasExecutionStatus = CanvasGenerationStatus

export interface CanvasPoint {
  x: number
  y: number
}

export interface CanvasSize {
  width: number
  height: number
}

export interface CanvasViewport extends CanvasPoint {
  zoom: number
}

export interface CanvasProject {
  id: string
  spaceToken?: string
  ownerUserId?: string
  title: string
  revision: number
  viewport: CanvasViewport
  nodes: CanvasNode[]
  edges: CanvasEdge[]
  assets?: CanvasAssetRef[]
  snapshots?: CanvasSnapshot[]
  createdAt: string
  updatedAt: string
}

export interface CanvasNodeBase {
  id: string
  type: CanvasNodeType
  name?: string
  x: number
  y: number
  width: number
  height: number
  rotation: number
  zIndex: number
  text?: string
  role?: CanvasEdgeRole
  isReference?: boolean
  assetId?: string
  source?: CanvasAssetSource
  uploadId?: string
  taskId?: string
  resultIndex?: number
  imageUrl?: string
  thumbnailUrl?: string
  originalUrl?: string
  naturalWidth?: number
  naturalHeight?: number
  promptSnapshot?: string
  referenceSnapshotIds?: string[]
  mode?: Mode
  provider?: ModelProvider
  model?: string
  ratio?: string
  resolution?: string
  quality?: string
  outputFormat?: string
  count?: number
  concurrency?: number
  status?: CanvasGenerationStatus
  taskIds?: string[]
  metadata?: Record<string, unknown>
  createdAt?: string
  updatedAt?: string
}

export type GenerationNodeData = Pick<
  CanvasNodeBase,
  | 'mode'
  | 'provider'
  | 'model'
  | 'ratio'
  | 'resolution'
  | 'quality'
  | 'outputFormat'
  | 'count'
  | 'concurrency'
  | 'uploadId'
  | 'taskId'
  | 'taskIds'
  | 'promptSnapshot'
  | 'status'
>

export interface GenerationNode extends Omit<CanvasNodeBase, 'type'> {
  type: 'generation'
}

export type ResultNodeData = Pick<
  CanvasNodeBase,
  | 'taskId'
  | 'resultIndex'
  | 'imageUrl'
  | 'thumbnailUrl'
  | 'originalUrl'
  | 'naturalWidth'
  | 'naturalHeight'
  | 'promptSnapshot'
  | 'status'
>

export interface ResultNode extends Omit<CanvasNodeBase, 'type'> {
  type: 'result'
}

export interface GenericCanvasNode extends CanvasNodeBase {
  type: Exclude<CanvasNodeType, 'generation' | 'result'>
}

export type CanvasNode = GenerationNode | ResultNode | GenericCanvasNode

export interface CanvasEdge {
  id: string
  fromNodeId: string
  toNodeId: string
  role: CanvasEdgeRole
  label?: string
  text?: string
  createdAt?: string
  updatedAt?: string
}

export interface CanvasAssetRef {
  id: string
  source: CanvasAssetSource
  uploadId?: string
  taskId?: string
  resultIndex?: number
  url?: string
  thumbnailUrl?: string
  mime?: string
  size?: number
  width?: number
  height?: number
  originalName?: string
  promptSnapshot?: string
  createdAt?: string
}

export interface CanvasGenerationParameters {
  mode: Mode
  provider?: ModelProvider
  model?: string
  ratio?: string
  resolution?: string
  quality?: string
  outputFormat?: string
  count?: number
  concurrency?: number
  uploadIds?: string[]
}

export interface CanvasPromptPart {
  nodeId: string
  text: string
  edgeLabel?: string
}

export interface CanvasSnapshotReference {
  nodeId?: string
  assetId?: string
  role?: CanvasEdgeRole
  edgeLabel?: string
  snapshotId?: string
  source?: CanvasAssetSource
  uploadId?: string
  taskId?: string
  resultIndex?: number
  originalName?: string
  fileName?: string
  mime?: string
  size?: number
  promptSnapshot?: string
}

export interface CanvasSnapshot {
  id: string
  projectId: string
  projectRevision: number
  generationNodeId: string
  contextHash: string
  resolvedPrompt: string
  promptParts?: CanvasPromptPart[]
  references?: CanvasSnapshotReference[]
  parameters: CanvasGenerationParameters
  uploadIds?: string[]
  sourceNodeIds?: string[]
  metadata?: Record<string, unknown>
  createdAt: string
}

export interface CanvasTaskBinding {
  projectId: string
  snapshotId: string
  sourceNodeIds: string[]
  targetNodeId?: string
  createdNodeId?: string
  contextHash: string
  createdAt: string
}

export interface CanvasProjectListOptions {
  limit?: number
}

export interface CreateCanvasProjectRequest {
  title?: string
  nodes?: CanvasNode[]
  edges?: CanvasEdge[]
  viewport?: CanvasViewport
  assets?: CanvasAssetRef[]
}

export interface UpdateCanvasProjectRequest {
  revision: number
  title?: string
  nodes?: CanvasNode[]
  edges?: CanvasEdge[]
  viewport?: CanvasViewport
  assets?: CanvasAssetRef[]
}

export interface CreateCanvasSnapshotRequest {
  revision?: number
  generationNodeId: string
  resolvedPrompt: string
  promptParts?: CanvasPromptPart[]
  references?: CanvasSnapshotReference[]
  parameters: CanvasGenerationParameters
  uploadIds?: string[]
  sourceNodeIds?: string[]
  metadata?: Record<string, unknown>
}

export type CanvasProjectListResponse = {
  ok: boolean
  projects: CanvasProject[]
}

export type CanvasProjectResponse = { ok: boolean; project: CanvasProject }
export type DeleteCanvasProjectResponse = { ok: boolean; project: CanvasProject }
export type CanvasSnapshotResponse = { ok: boolean; snapshot: CanvasSnapshot }
