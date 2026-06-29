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
  CanvasProjectStatus,
  CanvasSize,
  CanvasSnapshot,
  CanvasSnapshotReference,
  CanvasTaskBinding,
  CanvasViewport,
  GenerationNode,
  GenerationNodeData,
  ResultNode,
  ResultNodeData,
} from '../../../api/contracts/canvas'

import type { CanvasEdge, CanvasNode, CanvasProject } from '../../../api/contracts/canvas'

export interface CanvasSelection {
  nodeIds: string[]
  edgeIds: string[]
}

export interface CanvasModelState {
  project: CanvasProject | null
  selection: CanvasSelection
  activeNodeId?: string
  isLoading: boolean
  isSaving: boolean
  error?: string
}

export interface CanvasGraphSnapshot {
  projectId?: string
  nodes: CanvasNode[]
  edges: CanvasEdge[]
}
