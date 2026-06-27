import type { ReferenceUpload } from '../../types'
import type { CanvasImageDisplaySize } from './imageSizing'
import type { CanvasHistoryImage, CanvasImageItem, CanvasItem, CanvasPoint, ReferenceRole } from './types'
import { REFERENCE_ROLES } from './types'

export function createCanvasItemFromUpload(upload: ReferenceUpload, point: CanvasPoint, index: number, localPreviewUrl?: string, size?: CanvasImageDisplaySize): CanvasImageItem {
  return {
    type: 'image',
    id: `upload-${upload.id}-${Date.now()}-${index}`,
    uploadId: upload.id,
    localPreviewUrl,
    name: upload.originalName || `参考图 ${index + 1}`,
    role: index === 0 ? 'subject' : 'reference',
    isReference: true,
    x: point.x,
    y: point.y,
    width: size?.width ?? 220,
    height: size?.height ?? 156,
    naturalWidth: size?.naturalWidth,
    naturalHeight: size?.naturalHeight,
    aspectRatio: size?.aspectRatio,
    rotation: 0,
  }
}

export function createCanvasItemFromHistory(image: CanvasHistoryImage, point: CanvasPoint, index: number, upload?: ReferenceUpload, size?: CanvasImageDisplaySize): CanvasImageItem {
  return {
    type: 'image',
    id: `history-${image.id}-${Date.now()}`,
    uploadId: upload?.id,
    resultSrc: image.src,
    name: image.title || `历史结果 ${index + 1}`,
    role: index === 0 ? 'subject' : 'reference',
    isReference: true,
    x: point.x,
    y: point.y,
    width: size?.width ?? 220,
    height: size?.height ?? 156,
    naturalWidth: size?.naturalWidth,
    naturalHeight: size?.naturalHeight,
    aspectRatio: size?.aspectRatio,
    rotation: 0,
  }
}

export function imageSrcForCanvasItem(item: CanvasItem, previewUrls: Record<string, string>) {
  if (item.type !== 'image') return ''
  return item.localPreviewUrl || (item.uploadId ? previewUrls[item.uploadId] : '') || item.resultSrc || ''
}

export function buildReferencePromptLine(index: number) {
  return `@${index}`
}

export function referenceIndexForItem(items: CanvasItem[], itemId: string) {
  const references = items.filter((item) => item.isReference || item.id === itemId)
  const index = references.findIndex((item) => item.id === itemId)
  return index >= 0 ? index + 1 : references.length + 1
}

export function imageFilesFromClipboard(data: DataTransfer) {
  const directFiles = Array.from(data.files || []).filter(isImageFile)
  if (directFiles.length) return directFiles
  return Array.from(data.items || [])
    .filter((item) => item.kind === 'file')
    .map((item) => item.getAsFile())
    .filter(isImageFile)
}

export function isImageFile(file: File | null): file is File {
  if (!file) return false
  return file.type.startsWith('image/') || /\.(png|jpe?g|webp|gif|avif)$/i.test(file.name)
}

export function roleMeta(role: ReferenceRole) {
  return REFERENCE_ROLES.find((item) => item.value === role) || REFERENCE_ROLES[0]
}

export function clampNumber(value: string, min: number, max: number) {
  const parsed = Number(value)
  if (!Number.isFinite(parsed)) return min
  return Math.min(max, Math.max(min, Math.round(parsed)))
}

export function unique(items: string[]) {
  return Array.from(new Set(items))
}

export function safeParseDragData<T>(value: string): T | null {
  try {
    return JSON.parse(value) as T
  } catch {
    return null
  }
}