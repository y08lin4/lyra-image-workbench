import type { Mode } from '../../types'
import type { CanvasConnection, CanvasImageItem, CanvasItem, CanvasTextItem, ReferenceRole } from './types'

const CREATIVE_CANVAS_DRAFT_KEY = 'lyra.creativeCanvas.draft.v1'

export type CreativeCanvasDraft = {
  items: CanvasItem[]
  connections: CanvasConnection[]
  prompt: string
  mode: Mode
  hasStoredDraft: boolean
}

type StoredCreativeCanvasDraft = Partial<CreativeCanvasDraft> & {
  version?: number
  savedAt?: string
}

const EMPTY_DRAFT: CreativeCanvasDraft = {
  items: [],
  connections: [],
  prompt: '',
  mode: 'text-to-image',
  hasStoredDraft: false,
}

export function loadCreativeCanvasDraft(): CreativeCanvasDraft {
  if (typeof window === 'undefined') return EMPTY_DRAFT

  try {
    const raw = window.localStorage.getItem(CREATIVE_CANVAS_DRAFT_KEY)
    if (!raw) return EMPTY_DRAFT
    const parsed = JSON.parse(raw) as StoredCreativeCanvasDraft
    const items = Array.isArray(parsed.items) ? parsed.items.map(sanitizeCanvasItem).filter(isCanvasItem) : []
    const itemIds = new Set(items.map((item) => item.id))
    const connections = Array.isArray(parsed.connections)
      ? parsed.connections.map(sanitizeConnection).filter((item): item is CanvasConnection => {
        if (!item) return false
        return itemIds.has(item.fromId) && itemIds.has(item.toId) && item.fromId !== item.toId
      })
      : []

    return {
      items,
      connections,
      prompt: typeof parsed.prompt === 'string' ? parsed.prompt : '',
      mode: parsed.mode === 'image-to-image' ? 'image-to-image' : 'text-to-image',
      hasStoredDraft: true,
    }
  } catch {
    window.localStorage.removeItem(CREATIVE_CANVAS_DRAFT_KEY)
    return EMPTY_DRAFT
  }
}

export function saveCreativeCanvasDraft(draft: CreativeCanvasDraft) {
  if (typeof window === 'undefined') return

  const items = draft.items.map(sanitizeCanvasItem).filter(isCanvasItem)
  const itemIds = new Set(items.map((item) => item.id))
  const connections = draft.connections.filter((item) => (
    itemIds.has(item.fromId) && itemIds.has(item.toId) && item.fromId !== item.toId
  ))
  const payload: StoredCreativeCanvasDraft = {
    version: 1,
    savedAt: new Date().toISOString(),
    items,
    connections,
    prompt: draft.prompt,
    mode: draft.mode,
  }

  try {
    window.localStorage.setItem(CREATIVE_CANVAS_DRAFT_KEY, JSON.stringify(payload))
  } catch {
    // localStorage may be full or disabled. Canvas editing should remain usable.
  }
}

function sanitizeCanvasItem(value: unknown): CanvasItem | null {
  if (!value || typeof value !== 'object') return null
  const item = value as Partial<CanvasItem>
  const role: ReferenceRole = item.role === 'subject' || item.role === 'style' ? item.role : 'reference'
  const base = {
    id: sanitizeString(item.id),
    name: sanitizeString(item.name) || '画布元素',
    x: sanitizeNumber(item.x, 80),
    y: sanitizeNumber(item.y, 78),
    width: sanitizeNumber(item.width, 220),
    height: sanitizeNumber(item.height, 156),
    rotation: sanitizeNumber(item.rotation, 0),
    role,
    isReference: Boolean(item.isReference),
  }
  if (!base.id || base.width < 24 || base.height < 24) return null

  if (item.type === 'text') {
    return {
      ...base,
      type: 'text',
      text: sanitizeString((item as Partial<CanvasTextItem>).text),
      uploadId: undefined,
      resultSrc: undefined,
      localPreviewUrl: undefined,
    }
  }

  if (item.type === 'image') {
    const image = item as Partial<CanvasImageItem>
    return {
      ...base,
      type: 'image',
      uploadId: sanitizeOptionalString(image.uploadId),
      resultSrc: sanitizePersistentUrl(image.resultSrc),
      naturalWidth: sanitizeOptionalNumber(image.naturalWidth),
      naturalHeight: sanitizeOptionalNumber(image.naturalHeight),
      aspectRatio: sanitizeOptionalNumber(image.aspectRatio),
      role: base.role,
      isReference: base.isReference,
    }
  }

  return null
}

function sanitizeConnection(value: unknown): CanvasConnection | null {
  if (!value || typeof value !== 'object') return null
  const connection = value as Partial<CanvasConnection>
  const id = sanitizeString(connection.id)
  const fromId = sanitizeString(connection.fromId)
  const toId = sanitizeString(connection.toId)
  if (!id || !fromId || !toId) return null
  const label = sanitizeOptionalString(connection.label)
  const text = sanitizeOptionalString(connection.text)
  return { id, fromId, toId, label, text }
}

function isCanvasItem(item: CanvasItem | null): item is CanvasItem {
  return Boolean(item)
}

function sanitizeString(value: unknown) {
  return typeof value === 'string' ? value : ''
}

function sanitizeOptionalString(value: unknown) {
  const next = sanitizeString(value)
  return next || undefined
}

function sanitizePersistentUrl(value: unknown) {
  const next = sanitizeOptionalString(value)
  return next && !next.startsWith('blob:') ? next : undefined
}

function sanitizeNumber(value: unknown, fallback: number) {
  return typeof value === 'number' && Number.isFinite(value) ? value : fallback
}

function sanitizeOptionalNumber(value: unknown) {
  return typeof value === 'number' && Number.isFinite(value) ? value : undefined
}