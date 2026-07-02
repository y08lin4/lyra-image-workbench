import type {
  CanvasAssetRef,
  CanvasEdge,
  CanvasGenerationParameters,
  CanvasNode,
  CanvasProject,
  CanvasSnapshot,
  CreateCanvasProjectRequest,
  CreateCanvasSnapshotRequest,
  UpdateCanvasProjectRequest,
} from '../../api/contracts/canvas'
import type { Mode, ModelProvider } from '../../types'
import type { CanvasConnection, CanvasImageItem, CanvasItem, CanvasTextItem, ReferenceRole } from './types'

export const CLOUD_GENERATION_NODE_ID = 'creative-canvas-generation'

export type CreativeCanvasCloudState = {
  items: CanvasItem[]
  connections: CanvasConnection[]
  prompt: string
  mode: Mode
  provider: ModelProvider
  imageModel: string
  ratio: string
  resolution: string
  quality: string
  outputFormat: string
  count: number
  concurrency: number
}

type CloudDraftMetadata = Partial<Omit<CreativeCanvasCloudState, 'items' | 'connections' | 'provider'>> & {
  provider?: string
}

export function buildCanvasProjectPayload(state: CreativeCanvasCloudState): CreateCanvasProjectRequest {
  return {
    title: canvasProjectTitle(state.prompt),
    nodes: buildCloudNodes(state),
    edges: buildCloudEdges(state.connections, state.items),
    viewport: { x: 0, y: 0, zoom: 1 },
    assets: buildCloudAssets(state.items),
  }
}

export function buildCanvasProjectUpdate(project: CanvasProject, state: CreativeCanvasCloudState): UpdateCanvasProjectRequest {
  return {
    ...buildCanvasProjectPayload(state),
    revision: project.revision,
  }
}

export function buildCanvasSnapshotPayload(state: CreativeCanvasCloudState): CreateCanvasSnapshotRequest {
  const promptParts = state.items
    .filter((item): item is CanvasTextItem => item.type === 'text' && item.text.trim().length > 0)
    .map((item) => ({ nodeId: item.id, text: item.text.trim() }))
  const references = state.items
    .filter((item): item is CanvasImageItem => item.type === 'image' && item.isReference)
    .map((item) => ({
      nodeId: item.id,
      assetId: canvasAssetId(item),
      role: canvasRole(item.role),
      source: item.uploadId ? 'upload' as const : 'result' as const,
      uploadId: item.uploadId,
      originalName: item.name,
      promptSnapshot: state.prompt,
    }))

  return {
    generationNodeId: CLOUD_GENERATION_NODE_ID,
    resolvedPrompt: state.prompt,
    promptParts,
    references,
    parameters: buildGenerationParameters(state),
    uploadIds: state.items
      .filter((item): item is CanvasImageItem & { uploadId: string } => item.type === 'image' && Boolean(item.uploadId))
      .map((item) => item.uploadId),
    sourceNodeIds: state.items.map((item) => item.id),
    metadata: {
      prompt: state.prompt,
      mode: state.mode,
      imageModel: state.imageModel,
      ratio: state.ratio,
      resolution: state.resolution,
      quality: state.quality,
      outputFormat: state.outputFormat,
      count: state.count,
      concurrency: state.concurrency,
    },
  }
}

export function restoreCreativeCanvasProject(project: CanvasProject): Partial<CreativeCanvasCloudState> {
  const nodes = Array.isArray(project.nodes) ? project.nodes : []
  const items = nodes.map(restoreCanvasItem).filter((item): item is CanvasItem => Boolean(item))
  const itemIds = new Set(items.map((item) => item.id))
  const connections = (Array.isArray(project.edges) ? project.edges : [])
    .map(restoreCanvasConnection)
    .filter((item): item is CanvasConnection => Boolean(item && itemIds.has(item.fromId) && itemIds.has(item.toId)))
  const snapshot = latestSnapshot(project.snapshots)
  const metadata = snapshot?.metadata as CloudDraftMetadata | undefined
  const parameters = snapshot?.parameters

  return {
    items,
    connections,
    prompt: stringValue(metadata?.prompt) || snapshot?.resolvedPrompt || '',
    mode: metadata?.mode === 'image-to-image' || parameters?.mode === 'image-to-image' ? 'image-to-image' : 'text-to-image',
    imageModel: stringValue(metadata?.imageModel || parameters?.model),
    ratio: stringValue(metadata?.ratio || parameters?.ratio),
    resolution: stringValue(metadata?.resolution || parameters?.resolution),
    quality: stringValue(metadata?.quality || parameters?.quality),
    outputFormat: stringValue(metadata?.outputFormat || parameters?.outputFormat),
    count: numberValue(metadata?.count || parameters?.count),
    concurrency: numberValue(metadata?.concurrency || parameters?.concurrency),
  }
}

function buildCloudNodes(state: CreativeCanvasCloudState): CanvasNode[] {
  const itemNodes = state.items.map((item, index): CanvasNode => {
    const base = {
      id: item.id,
      type: item.type,
      name: item.name,
      x: item.x,
      y: item.y,
      width: item.width,
      height: item.height,
      rotation: item.rotation,
      zIndex: index,
      role: canvasRole(item.role),
      isReference: item.isReference,
    }
    if (item.type === 'text') {
      return { ...base, type: 'text', text: item.text }
    }
    return {
      ...base,
      type: 'image',
      assetId: canvasAssetId(item),
      source: item.uploadId ? 'upload' : 'result',
      uploadId: item.uploadId,
      imageUrl: persistentImageUrl(item),
      originalUrl: persistentImageUrl(item),
      naturalWidth: item.naturalWidth,
      naturalHeight: item.naturalHeight,
    }
  })

  return [
    ...itemNodes,
    {
      id: CLOUD_GENERATION_NODE_ID,
      type: 'generation',
      name: '画布生成参数',
      x: 32,
      y: 32,
      width: 1,
      height: 1,
      rotation: 0,
      zIndex: -1,
      status: 'ready',
      promptSnapshot: state.prompt,
      ...buildGenerationParameters(state),
    },
  ]
}

function buildCloudEdges(connections: CanvasConnection[], items: CanvasItem[]): CanvasEdge[] {
  const itemIds = new Set(items.map((item) => item.id))
  return connections
    .filter((item) => itemIds.has(item.fromId) && itemIds.has(item.toId) && item.fromId !== item.toId)
    .map((item) => ({
      id: item.id,
      fromNodeId: item.fromId,
      toNodeId: item.toId,
      role: 'custom',
      label: item.label,
      text: item.text,
    }))
}

function buildCloudAssets(items: CanvasItem[]): CanvasAssetRef[] {
  return items
    .filter((item): item is CanvasImageItem => item.type === 'image')
    .map((item) => ({
      id: canvasAssetId(item),
      source: item.uploadId ? 'upload' : 'result',
      uploadId: item.uploadId,
      url: persistentImageUrl(item),
      thumbnailUrl: persistentImageUrl(item),
      width: item.naturalWidth,
      height: item.naturalHeight,
      originalName: item.name,
      promptSnapshot: item.isReference ? item.name : undefined,
    }))
}

function buildGenerationParameters(state: CreativeCanvasCloudState): CanvasGenerationParameters {
  return {
    mode: state.mode,
    provider: state.provider,
    model: state.imageModel,
    ratio: state.ratio,
    resolution: state.resolution,
    quality: state.quality,
    outputFormat: state.outputFormat,
    count: state.count,
    concurrency: state.concurrency,
  }
}

function restoreCanvasItem(node: CanvasNode): CanvasItem | null {
  if (node.id === CLOUD_GENERATION_NODE_ID || node.type === 'generation' || node.width < 24 || node.height < 24) return null
  const base = {
    id: node.id,
    name: node.name || '画布元素',
    x: finiteNumber(node.x, 80),
    y: finiteNumber(node.y, 78),
    width: finiteNumber(node.width, 220),
    height: finiteNumber(node.height, 156),
    rotation: finiteNumber(node.rotation, 0),
    role: restoreRole(node.role),
    isReference: Boolean(node.isReference),
  }
  if (node.type === 'text') {
    return { ...base, type: 'text', text: node.text || '', uploadId: undefined, resultSrc: undefined, localPreviewUrl: undefined }
  }
  if (node.type === 'image' || node.imageUrl || node.uploadId) {
    return {
      ...base,
      type: 'image',
      uploadId: node.uploadId,
      resultSrc: node.imageUrl || node.thumbnailUrl || node.originalUrl,
      naturalWidth: node.naturalWidth,
      naturalHeight: node.naturalHeight,
      aspectRatio: node.naturalWidth && node.naturalHeight ? node.naturalWidth / node.naturalHeight : undefined,
    }
  }
  return null
}

function restoreCanvasConnection(edge: CanvasEdge): CanvasConnection | null {
  if (!edge.id || !edge.fromNodeId || !edge.toNodeId || edge.fromNodeId === edge.toNodeId) return null
  return {
    id: edge.id,
    fromId: edge.fromNodeId,
    toId: edge.toNodeId,
    label: edge.label,
    text: edge.text,
  }
}

function latestSnapshot(snapshots: CanvasSnapshot[] | undefined) {
  if (!snapshots?.length) return undefined
  return [...snapshots].sort((a, b) => Date.parse(b.createdAt) - Date.parse(a.createdAt))[0]
}

function canvasProjectTitle(prompt: string) {
  const title = prompt.trim().replace(/\s+/g, ' ').slice(0, 32)
  return title || '未命名画布'
}

function canvasAssetId(item: CanvasImageItem) {
  return item.uploadId ? `upload-${item.uploadId}` : `image-${item.id}`
}

function persistentImageUrl(item: CanvasImageItem) {
  return item.resultSrc && !item.resultSrc.startsWith('blob:') ? item.resultSrc : undefined
}

function canvasRole(role: ReferenceRole) {
  return role === 'subject' || role === 'style' ? role : 'reference'
}

function restoreRole(role: unknown): ReferenceRole {
  return role === 'subject' || role === 'style' ? role : 'reference'
}

function finiteNumber(value: unknown, fallback: number) {
  return typeof value === 'number' && Number.isFinite(value) ? value : fallback
}

function stringValue(value: unknown) {
  return typeof value === 'string' ? value : ''
}

function numberValue(value: unknown) {
  return typeof value === 'number' && Number.isFinite(value) ? value : undefined
}
