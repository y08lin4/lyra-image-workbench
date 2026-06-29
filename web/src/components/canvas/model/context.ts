export type CanvasTaskParameters = Record<string, unknown>

export type CanvasTaskPromptPartInput =
  | string
  | {
      id?: string
      text?: string | null
      value?: string | null
      content?: string | null
    }

export type CanvasTaskNodeContext = {
  promptParts?: readonly CanvasTaskPromptPartInput[]
  referenceAssetIds?: readonly (string | null | undefined)[]
  parameters?: CanvasTaskParameters
}

export type CanvasTaskNode = CanvasTaskNodeContext & {
  id: string
  data?: CanvasTaskNodeContext
}

export type CanvasTaskEdgeContext = {
  label?: string | null
  text?: string | null
  parameters?: CanvasTaskParameters
}

export type CanvasTaskEdge = CanvasTaskEdgeContext & {
  id?: string
  sourceNodeId?: string
  targetNodeId?: string
  fromNodeId?: string
  toNodeId?: string
  source?: string
  target?: string
  fromId?: string
  toId?: string
  data?: CanvasTaskEdgeContext
}

export type CanvasTaskDraftPromptPart = {
  text: string
  sourceNodeId: string
  partId?: string
}

export type CanvasTaskDraftEdgeLabel = {
  label: string
  sourceNodeId: string
  targetNodeId: string
  edgeId?: string
}

export type CanvasTaskDraftDiagnosticCode =
  | 'generation-node-missing'
  | 'edge-endpoint-missing'
  | 'edge-node-missing'
  | 'cycle-detected'

export type CanvasTaskDraftDiagnostic = {
  code: CanvasTaskDraftDiagnosticCode
  message: string
  nodeId?: string
  edgeId?: string
}

export type CanvasTaskDraft = {
  generationNodeId: string
  contextNodeIds: string[]
  contextEdgeIds: string[]
  promptParts: CanvasTaskDraftPromptPart[]
  prompt: string
  referenceAssetIds: string[]
  edgeLabels: CanvasTaskDraftEdgeLabel[]
  parameters: CanvasTaskParameters
  diagnostics: CanvasTaskDraftDiagnostic[]
}

export type BuildCanvasTaskDraftInput = {
  nodes: readonly CanvasTaskNode[]
  edges: readonly CanvasTaskEdge[]
  generationNodeId: string
}

type NormalizedEdge = {
  edge: CanvasTaskEdge
  index: number
  edgeId?: string
  sourceNodeId: string
  targetNodeId: string
}

export function buildCanvasTaskDraft(input: BuildCanvasTaskDraftInput): CanvasTaskDraft {
  const { nodes, edges, generationNodeId } = input
  const diagnostics: CanvasTaskDraftDiagnostic[] = []
  const nodeById = new Map(nodes.map((node) => [node.id, node]))
  const nodeIndexById = new Map(nodes.map((node, index) => [node.id, index]))
  const generationNode = nodeById.get(generationNodeId)

  if (!generationNode) {
    return {
      generationNodeId,
      contextNodeIds: [],
      contextEdgeIds: [],
      promptParts: [],
      prompt: '',
      referenceAssetIds: [],
      edgeLabels: [],
      parameters: {},
      diagnostics: [
        {
          code: 'generation-node-missing',
          message: `Generation node "${generationNodeId}" was not found.`,
          nodeId: generationNodeId,
        },
      ],
    }
  }

  const normalizedEdges = normalizeEdges(edges, nodeById, diagnostics)
  const incomingEdgesByTarget = groupIncomingEdges(normalizedEdges)
  const reachableNodeIds = collectAncestorNodeIds(generationNodeId, incomingEdgesByTarget)
  const contextEdges = normalizedEdges.filter((edge) => (
    reachableNodeIds.has(edge.sourceNodeId) && reachableNodeIds.has(edge.targetNodeId)
  ))
  const orderedNodeIds = orderNodeIds(reachableNodeIds, contextEdges, nodeIndexById, diagnostics)
  const orderedEdges = orderEdges(contextEdges, orderedNodeIds)
  const sourceNodeIds = orderedNodeIds.filter((nodeId) => nodeId !== generationNodeId)

  const promptParts = orderedNodeIds.flatMap((nodeId) => {
    const node = nodeById.get(nodeId)
    return node ? promptPartsForNode(node) : []
  })
  const referenceAssetIds = uniqueStrings(orderedNodeIds.flatMap((nodeId) => {
    const node = nodeById.get(nodeId)
    return node ? referenceAssetIdsForNode(node) : []
  }))
  const edgeLabels = orderedEdges
    .map((edge) => edgeLabelForEdge(edge))
    .filter((label): label is CanvasTaskDraftEdgeLabel => Boolean(label))
  const parameters = mergeDraftParameters(sourceNodeIds, generationNodeId, orderedEdges, nodeById)

  return {
    generationNodeId,
    contextNodeIds: orderedNodeIds,
    contextEdgeIds: orderedEdges.map((edge) => edge.edgeId ?? edge.index.toString()),
    promptParts,
    prompt: promptParts.map((part) => part.text).join('\n'),
    referenceAssetIds,
    edgeLabels,
    parameters,
    diagnostics,
  }
}

function normalizeEdges(
  edges: readonly CanvasTaskEdge[],
  nodeById: ReadonlyMap<string, CanvasTaskNode>,
  diagnostics: CanvasTaskDraftDiagnostic[],
) {
  return edges
    .map((edge, index): NormalizedEdge | null => {
      const sourceNodeId = compactId(edge.sourceNodeId ?? edge.fromNodeId ?? edge.source ?? edge.fromId)
      const targetNodeId = compactId(edge.targetNodeId ?? edge.toNodeId ?? edge.target ?? edge.toId)
      const edgeId = compactId(edge.id)

      if (!sourceNodeId || !targetNodeId) {
        diagnostics.push({
          code: 'edge-endpoint-missing',
          message: 'A canvas edge is missing a source or target node id.',
          edgeId,
        })
        return null
      }

      if (!nodeById.has(sourceNodeId) || !nodeById.has(targetNodeId)) {
        diagnostics.push({
          code: 'edge-node-missing',
          message: `Canvas edge "${edgeId ?? index}" references a node that was not found.`,
          edgeId,
        })
        return null
      }

      return { edge, index, edgeId, sourceNodeId, targetNodeId }
    })
    .filter((edge): edge is NormalizedEdge => Boolean(edge))
}

function groupIncomingEdges(edges: readonly NormalizedEdge[]) {
  const incoming = new Map<string, NormalizedEdge[]>()
  for (const edge of edges) {
    const current = incoming.get(edge.targetNodeId)
    if (current) {
      current.push(edge)
    } else {
      incoming.set(edge.targetNodeId, [edge])
    }
  }
  return incoming
}

function collectAncestorNodeIds(generationNodeId: string, incomingEdgesByTarget: ReadonlyMap<string, readonly NormalizedEdge[]>) {
  const reachableNodeIds = new Set<string>([generationNodeId])
  const stack = [generationNodeId]

  while (stack.length) {
    const targetNodeId = stack.pop()
    if (!targetNodeId) continue

    for (const edge of incomingEdgesByTarget.get(targetNodeId) ?? []) {
      if (reachableNodeIds.has(edge.sourceNodeId)) continue
      reachableNodeIds.add(edge.sourceNodeId)
      stack.push(edge.sourceNodeId)
    }
  }

  return reachableNodeIds
}

function orderNodeIds(
  reachableNodeIds: ReadonlySet<string>,
  edges: readonly NormalizedEdge[],
  nodeIndexById: ReadonlyMap<string, number>,
  diagnostics: CanvasTaskDraftDiagnostic[],
) {
  const indegree = new Map<string, number>()
  const outgoing = new Map<string, NormalizedEdge[]>()
  for (const nodeId of reachableNodeIds) indegree.set(nodeId, 0)

  for (const edge of edges) {
    if (!reachableNodeIds.has(edge.sourceNodeId) || !reachableNodeIds.has(edge.targetNodeId)) continue
    indegree.set(edge.targetNodeId, (indegree.get(edge.targetNodeId) ?? 0) + 1)
    const current = outgoing.get(edge.sourceNodeId)
    if (current) {
      current.push(edge)
    } else {
      outgoing.set(edge.sourceNodeId, [edge])
    }
  }

  const ordered: string[] = []
  const ready = Array.from(reachableNodeIds)
    .filter((nodeId) => (indegree.get(nodeId) ?? 0) === 0)
    .sort((a, b) => compareNodeIds(a, b, nodeIndexById))

  while (ready.length) {
    const nodeId = ready.shift()
    if (!nodeId) continue

    ordered.push(nodeId)
    const edgesFromNode = [...(outgoing.get(nodeId) ?? [])].sort((a, b) => a.index - b.index)
    for (const edge of edgesFromNode) {
      const nextIndegree = (indegree.get(edge.targetNodeId) ?? 0) - 1
      indegree.set(edge.targetNodeId, nextIndegree)
      if (nextIndegree === 0) {
        ready.push(edge.targetNodeId)
        ready.sort((a, b) => compareNodeIds(a, b, nodeIndexById))
      }
    }
  }

  if (ordered.length !== reachableNodeIds.size) {
    diagnostics.push({
      code: 'cycle-detected',
      message: 'A cycle was found in the generation context; cyclic nodes were appended in canvas order.',
    })
    const orderedSet = new Set(ordered)
    ordered.push(
      ...Array.from(reachableNodeIds)
        .filter((nodeId) => !orderedSet.has(nodeId))
        .sort((a, b) => compareNodeIds(a, b, nodeIndexById)),
    )
  }

  return ordered
}

function orderEdges(edges: readonly NormalizedEdge[], orderedNodeIds: readonly string[]) {
  const orderByNodeId = new Map(orderedNodeIds.map((nodeId, index) => [nodeId, index]))
  return [...edges].sort((a, b) => {
    const targetOrder = (orderByNodeId.get(a.targetNodeId) ?? Number.MAX_SAFE_INTEGER)
      - (orderByNodeId.get(b.targetNodeId) ?? Number.MAX_SAFE_INTEGER)
    if (targetOrder !== 0) return targetOrder

    const sourceOrder = (orderByNodeId.get(a.sourceNodeId) ?? Number.MAX_SAFE_INTEGER)
      - (orderByNodeId.get(b.sourceNodeId) ?? Number.MAX_SAFE_INTEGER)
    return sourceOrder || a.index - b.index
  })
}

function promptPartsForNode(node: CanvasTaskNode) {
  const promptParts = [...(node.promptParts ?? []), ...(node.data?.promptParts ?? [])]

  return promptParts
    .map((part): CanvasTaskDraftPromptPart | null => {
      if (typeof part === 'string') {
        const text = compactPromptText(part)
        return text ? { text, sourceNodeId: node.id } : null
      }

      const text = compactPromptText(part.text ?? part.value ?? part.content)
      return text ? { text, sourceNodeId: node.id, partId: compactId(part.id) } : null
    })
    .filter((part): part is CanvasTaskDraftPromptPart => Boolean(part))
}

function referenceAssetIdsForNode(node: CanvasTaskNode) {
  return uniqueStrings([...(node.referenceAssetIds ?? []), ...(node.data?.referenceAssetIds ?? [])])
}

function edgeLabelForEdge(edge: NormalizedEdge): CanvasTaskDraftEdgeLabel | null {
  const label = compactSingleLineText(edge.edge.label ?? edge.edge.text ?? edge.edge.data?.label)
  if (!label) return null

  return {
    label,
    sourceNodeId: edge.sourceNodeId,
    targetNodeId: edge.targetNodeId,
    edgeId: edge.edgeId,
  }
}

function mergeDraftParameters(
  sourceNodeIds: readonly string[],
  generationNodeId: string,
  orderedEdges: readonly NormalizedEdge[],
  nodeById: ReadonlyMap<string, CanvasTaskNode>,
) {
  const parameters: CanvasTaskParameters = {}

  for (const nodeId of sourceNodeIds) {
    mergeParameterRecords(parameters, parametersForNode(nodeById.get(nodeId)))
  }

  for (const edge of orderedEdges) {
    mergeParameterRecords(parameters, edge.edge.parameters)
    mergeParameterRecords(parameters, edge.edge.data?.parameters)
  }

  mergeParameterRecords(parameters, parametersForNode(nodeById.get(generationNodeId)))
  return parameters
}

function parametersForNode(node?: CanvasTaskNode) {
  if (!node) return undefined
  const parameters: CanvasTaskParameters = {}
  mergeParameterRecords(parameters, node.parameters)
  mergeParameterRecords(parameters, node.data?.parameters)
  return parameters
}

function mergeParameterRecords(target: CanvasTaskParameters, source?: CanvasTaskParameters) {
  if (!source) return target

  for (const [key, value] of Object.entries(source)) {
    if (typeof value === 'undefined') continue

    const current = target[key]
    target[key] = isPlainRecord(current) && isPlainRecord(value)
      ? mergeParameterRecords({ ...current }, value)
      : cloneParameterValue(value)
  }

  return target
}

function cloneParameterValue(value: unknown): unknown {
  if (Array.isArray(value)) return value.map(cloneParameterValue)
  if (isPlainRecord(value)) return mergeParameterRecords({}, value)
  return value
}

function isPlainRecord(value: unknown): value is CanvasTaskParameters {
  return Boolean(value) && typeof value === 'object' && !Array.isArray(value)
}

function compareNodeIds(a: string, b: string, nodeIndexById: ReadonlyMap<string, number>) {
  const indexDiff = (nodeIndexById.get(a) ?? Number.MAX_SAFE_INTEGER)
    - (nodeIndexById.get(b) ?? Number.MAX_SAFE_INTEGER)
  return indexDiff || a.localeCompare(b)
}

function compactId(value?: string | null) {
  const trimmed = value?.trim()
  return trimmed || undefined
}

function compactPromptText(value?: string | null) {
  return value?.trim() || ''
}

function compactSingleLineText(value?: string | null) {
  return value?.trim().replace(/\s+/g, ' ') || ''
}

function uniqueStrings(values: readonly (string | null | undefined)[]) {
  const seen = new Set<string>()
  const output: string[] = []

  for (const value of values) {
    const compactValue = compactId(value)
    if (!compactValue || seen.has(compactValue)) continue
    seen.add(compactValue)
    output.push(compactValue)
  }

  return output
}


