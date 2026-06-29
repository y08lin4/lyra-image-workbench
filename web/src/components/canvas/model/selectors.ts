import type {
  CanvasEdge,
  CanvasGraphSnapshot,
  CanvasModelState,
  CanvasNode,
  CanvasProject,
  GenerationNode,
  ResultNode,
} from './types'

export function selectCanvasProject(state: CanvasModelState) {
  return state.project
}

export function selectCanvasSnapshot(project: CanvasProject | null | undefined): CanvasGraphSnapshot {
  return {
    projectId: project?.id,
    nodes: project?.nodes ?? [],
    edges: project?.edges ?? [],
  }
}

export function selectCanvasNodes(project: CanvasProject | null | undefined) {
  return project?.nodes ?? []
}

export function selectCanvasEdges(project: CanvasProject | null | undefined) {
  return project?.edges ?? []
}

export function selectCanvasNodeById(project: CanvasProject | null | undefined, nodeId: string) {
  return selectCanvasNodes(project).find((node) => node.id === nodeId)
}

export function selectCanvasEdgeById(project: CanvasProject | null | undefined, edgeId: string) {
  return selectCanvasEdges(project).find((edge) => edge.id === edgeId)
}

export function selectCanvasNodeMap(project: CanvasProject | null | undefined) {
  return new Map(selectCanvasNodes(project).map((node) => [node.id, node]))
}

export function selectGenerationNodes(project: CanvasProject | null | undefined) {
  return selectCanvasNodes(project).filter(isGenerationNode)
}

export function selectResultNodes(project: CanvasProject | null | undefined) {
  return selectCanvasNodes(project).filter(isResultNode)
}

export function selectSelectedNodes(state: CanvasModelState) {
  const nodeMap = selectCanvasNodeMap(state.project)
  return state.selection.nodeIds.map((nodeId) => nodeMap.get(nodeId)).filter(isCanvasNode)
}

export function selectSelectedEdges(state: CanvasModelState) {
  const edgeMap = new Map(selectCanvasEdges(state.project).map((edge) => [edge.id, edge]))
  return state.selection.edgeIds.map((edgeId) => edgeMap.get(edgeId)).filter(isCanvasEdge)
}

export function selectIncomingEdges(project: CanvasProject | null | undefined, nodeId: string) {
  return selectCanvasEdges(project).filter((edge) => edge.toNodeId === nodeId)
}

export function selectOutgoingEdges(project: CanvasProject | null | undefined, nodeId: string) {
  return selectCanvasEdges(project).filter((edge) => edge.fromNodeId === nodeId)
}

export function selectConnectedNodeIds(project: CanvasProject | null | undefined, nodeId: string) {
  const ids = new Set<string>()
  for (const edge of selectCanvasEdges(project)) {
    if (edge.fromNodeId === nodeId) ids.add(edge.toNodeId)
    if (edge.toNodeId === nodeId) ids.add(edge.fromNodeId)
  }
  return ids
}

export function isGenerationNode(node: CanvasNode | undefined): node is GenerationNode {
  return node?.type === 'generation'
}

export function isResultNode(node: CanvasNode | undefined): node is ResultNode {
  return node?.type === 'result'
}

function isCanvasNode(node: CanvasNode | undefined): node is CanvasNode {
  return Boolean(node)
}

function isCanvasEdge(edge: CanvasEdge | undefined): edge is CanvasEdge {
  return Boolean(edge)
}
