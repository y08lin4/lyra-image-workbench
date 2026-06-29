import type {
  CanvasEdge,
  CanvasGenerationParameters,
  CanvasNode,
  CanvasProject,
  CanvasSnapshotReference,
  CreateCanvasSnapshotRequest,
} from '../../../api/contracts/canvas'
import { buildCanvasTaskDraft, type CanvasTaskDraft, type CanvasTaskEdge, type CanvasTaskNode, type CanvasTaskParameters } from './context'

export type CanvasProjectTaskSource = Pick<CanvasProject, 'revision' | 'nodes' | 'edges'>

export function buildCanvasTaskDraftFromProject(project: Pick<CanvasProject, 'nodes' | 'edges'>, generationNodeId: string) {
  return buildCanvasTaskDraft({
    generationNodeId,
    nodes: project.nodes.map(canvasNodeToTaskNode),
    edges: project.edges.map(canvasEdgeToTaskEdge),
  })
}

export function canvasNodeToTaskNode(node: CanvasNode): CanvasTaskNode {
  return {
    id: node.id,
    promptParts: promptPartsForCanvasNode(node),
    referenceAssetIds: referenceAssetIdsForCanvasNode(node),
    parameters: parametersForCanvasNode(node),
  }
}

export function canvasEdgeToTaskEdge(edge: CanvasEdge): CanvasTaskEdge {
  return {
    id: edge.id,
    fromNodeId: edge.fromNodeId,
    toNodeId: edge.toNodeId,
    label: edge.label,
    text: edge.text,
  }
}

export function buildCanvasSnapshotRequest(
  project: CanvasProjectTaskSource,
  generationNodeId: string,
  metadata?: Record<string, unknown>,
): CreateCanvasSnapshotRequest {
  const draft = buildCanvasTaskDraftFromProject(project, generationNodeId)
  return canvasTaskDraftToSnapshotRequest(project, draft, metadata)
}

export function canvasTaskDraftToSnapshotRequest(
  project: CanvasProjectTaskSource,
  draft: CanvasTaskDraft,
  metadata?: Record<string, unknown>,
): CreateCanvasSnapshotRequest {
  const nodeById = new Map(project.nodes.map((node) => [node.id, node]))
  const generationNode = nodeById.get(draft.generationNodeId)
  const references = snapshotReferencesForDraft(draft, nodeById)
  const parameters = generationParametersForNode(generationNode, draft.parameters)
  const uploadIds = uniqueStrings([
    ...(parameters.uploadIds ?? []),
    ...references.map((reference) => reference.uploadId),
  ])

  return {
    revision: project.revision,
    generationNodeId: draft.generationNodeId,
    resolvedPrompt: draft.prompt,
    promptParts: draft.promptParts.map((part) => ({
      nodeId: part.sourceNodeId,
      text: part.text,
      edgeLabel: edgeLabelForNode(draft, part.sourceNodeId),
    })),
    references,
    parameters: uploadIds.length ? { ...parameters, uploadIds } : parameters,
    uploadIds,
    sourceNodeIds: draft.contextNodeIds.filter((nodeId) => nodeId !== draft.generationNodeId),
    metadata,
  }
}

function promptPartsForCanvasNode(node: CanvasNode) {
  const text = compactText(node.text ?? node.promptSnapshot)
  return text ? [text] : []
}

function referenceAssetIdsForCanvasNode(node: CanvasNode) {
  return uniqueStrings([node.assetId])
}

function parametersForCanvasNode(node: CanvasNode): CanvasTaskParameters {
  const parameters: CanvasTaskParameters = {}
  assignDefined(parameters, 'mode', node.mode)
  assignDefined(parameters, 'provider', node.provider)
  assignDefined(parameters, 'model', node.model)
  assignDefined(parameters, 'ratio', node.ratio)
  assignDefined(parameters, 'resolution', node.resolution)
  assignDefined(parameters, 'quality', node.quality)
  assignDefined(parameters, 'outputFormat', node.outputFormat)
  assignPositiveInteger(parameters, 'count', node.count)
  assignPositiveInteger(parameters, 'concurrency', node.concurrency)

  const uploadIds = uniqueStrings([node.uploadId])
  if (uploadIds.length) parameters.uploadIds = uploadIds

  return parameters
}

function snapshotReferencesForDraft(
  draft: CanvasTaskDraft,
  nodeById: ReadonlyMap<string, CanvasNode>,
): CanvasSnapshotReference[] {
  return draft.contextNodeIds
    .filter((nodeId) => nodeId !== draft.generationNodeId)
    .map((nodeId): CanvasSnapshotReference | null => {
      const node = nodeById.get(nodeId)
      if (!node) return null
      const snapshotId = node.referenceSnapshotIds?.find((id) => Boolean(compactText(id)))
      const edgeLabel = edgeLabelForNode(draft, nodeId)

      if (!node.assetId && !node.uploadId && !node.taskId && !snapshotId && !node.source) return null

      return {
        nodeId: node.id,
        assetId: compactText(node.assetId) || undefined,
        role: node.role,
        edgeLabel,
        snapshotId,
        source: node.source,
        uploadId: compactText(node.uploadId) || undefined,
        taskId: compactText(node.taskId) || undefined,
        resultIndex: nonNegativeIntegerOrUndefined(node.resultIndex),
        mime: compactText(node.metadata?.mime) || undefined,
        size: positiveIntegerOrUndefined(node.metadata?.size),
        promptSnapshot: compactText(node.promptSnapshot) || undefined,
      }
    })
    .filter((reference): reference is CanvasSnapshotReference => Boolean(reference))
}

function generationParametersForNode(node: CanvasNode | undefined, draftParameters: CanvasTaskParameters): CanvasGenerationParameters {
  const uploadIds = arrayOfStrings(draftParameters.uploadIds)
  const count = positiveIntegerOrUndefined(draftParameters.count)
  const concurrency = positiveIntegerOrUndefined(draftParameters.concurrency)

  return {
    mode: modeOrUndefined(draftParameters.mode) ?? node?.mode ?? 'text-to-image',
    provider: stringOrUndefined(draftParameters.provider) ?? node?.provider,
    model: stringOrUndefined(draftParameters.model) ?? node?.model,
    ratio: stringOrUndefined(draftParameters.ratio) ?? node?.ratio,
    resolution: stringOrUndefined(draftParameters.resolution) ?? node?.resolution,
    quality: stringOrUndefined(draftParameters.quality) ?? node?.quality,
    outputFormat: stringOrUndefined(draftParameters.outputFormat) ?? node?.outputFormat,
    count: count ?? node?.count,
    concurrency: concurrency ?? node?.concurrency,
    uploadIds: uploadIds.length ? uploadIds : undefined,
  }
}

function edgeLabelForNode(draft: CanvasTaskDraft, nodeId: string) {
  return draft.edgeLabels.find((edgeLabel) => edgeLabel.sourceNodeId === nodeId || edgeLabel.targetNodeId === nodeId)?.label
}

function assignDefined(target: CanvasTaskParameters, key: string, value: unknown) {
  if (typeof value !== 'undefined' && value !== null && value !== '') target[key] = value
}

function assignPositiveInteger(target: CanvasTaskParameters, key: string, value: unknown) {
  const next = positiveIntegerOrUndefined(value)
  if (typeof next === 'number') target[key] = next
}

function positiveIntegerOrUndefined(value: unknown) {
  return typeof value === 'number' && Number.isFinite(value) && value > 0 ? Math.floor(value) : undefined
}

function nonNegativeIntegerOrUndefined(value: unknown) {
  return typeof value === 'number' && Number.isFinite(value) && value >= 0 ? Math.floor(value) : undefined
}

function stringOrUndefined(value: unknown) {
  return typeof value === 'string' && value.trim() ? value.trim() : undefined
}

function modeOrUndefined(value: unknown): CanvasGenerationParameters['mode'] | undefined {
  const mode = stringOrUndefined(value)
  return mode === 'text-to-image' || mode === 'image-to-image' || mode === 'gif' ? mode : undefined
}

function arrayOfStrings(value: unknown) {
  return Array.isArray(value) ? uniqueStrings(value) : []
}

function compactText(value: unknown) {
  return typeof value === 'string' ? value.trim() : ''
}

function uniqueStrings(values: readonly unknown[]) {
  const seen = new Set<string>()
  const output: string[] = []
  for (const value of values) {
    const text = compactText(value)
    if (!text || seen.has(text)) continue
    seen.add(text)
    output.push(text)
  }
  return output
}
