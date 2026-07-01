import type { CSSProperties, DragEvent as ReactDragEvent } from 'react'
import type { CanvasInteraction, CanvasItem, CanvasPoint } from './types'

type CanvasPointSize = {
  width?: number
  height?: number
}

const CANVAS_SAFE_INSET = 8
const DEFAULT_CANVAS_POINT_SIZE = { width: 220, height: 156 }

export function updateCanvasItemsForInteraction(items: CanvasItem[], interaction: CanvasInteraction, event: PointerEvent, stage: HTMLElement | null): CanvasItem[] {
  const metrics = canvasStageMetrics(stage)
  return items.map((item) => {
    if (item.id !== interaction.itemId) return item
    if (interaction.type === 'move') {
      const nextX = interaction.origin.x + event.clientX - interaction.startClientX
      const nextY = interaction.origin.y + event.clientY - interaction.startClientY
      return {
        ...item,
        x: metrics ? clampCanvasStart(nextX, metrics.width, item.width) : nextX,
        y: metrics ? clampCanvasStart(nextY, metrics.height, item.height) : nextY,
      }
    }
    if (interaction.type === 'resize') {
      return item.type === 'image'
        ? resizeImageItemByPointer(item, interaction, event, metrics)
        : resizeFreeformItemByPointer(item, interaction, event)
    }
    const nextAngle = pointerAngleForItem(item, event.clientX, event.clientY, stage)
    const startAngle = interaction.startAngle ?? nextAngle
    return { ...item, rotation: normalizeRotation(interaction.origin.rotation + nextAngle - startAngle) }
  })
}

export function canvasItemStyle(item: CanvasItem): CSSProperties {
  return {
    width: item.width,
    height: item.height,
    transform: `translate(${item.x}px, ${item.y}px) rotate(${item.rotation}deg)`,
  }
}

export function canvasItemContentStyle(_item: CanvasItem): CSSProperties {
  return {}
}

export function canvasControlStyle(_item: CanvasItem): CSSProperties {
  return {}
}

export function scaleCanvasItemByWheel(item: CanvasItem, deltaY: number, stage: HTMLElement | null): CanvasItem {
  const metrics = canvasStageMetrics(stage)
  const maxWidth = metrics ? Math.max(160, metrics.width - 24) : 900
  const maxHeight = metrics ? Math.max(120, metrics.height - 24) : 720
  const minWidth = item.type === 'text' ? 96 : 64
  const minHeight = item.type === 'text' ? 64 : 48
  const requestedScale = deltaY < 0 ? 1.08 : 0.92
  const minScale = Math.max(minWidth / item.width, minHeight / item.height)
  const maxScale = Math.min(maxWidth / item.width, maxHeight / item.height)
  const scale = clamp(requestedScale, minScale, maxScale)
  const nextSize = item.type === 'image'
    ? imageSizeForScale(item, scale, maxWidth, maxHeight, minWidth, minHeight)
    : {
        width: roundCanvasMetric(item.width * scale),
        height: roundCanvasMetric(item.height * scale),
      }
  const nextWidth = nextSize.width
  const nextHeight = nextSize.height
  const centerX = item.x + item.width / 2
  const centerY = item.y + item.height / 2
  const nextX = centerX - nextWidth / 2
  const nextY = centerY - nextHeight / 2

  return {
    ...item,
    x: metrics ? clampCanvasStart(nextX, metrics.width, nextWidth) : nextX,
    y: metrics ? clampCanvasStart(nextY, metrics.height, nextHeight) : nextY,
    width: nextWidth,
    height: nextHeight,
  }
}

function resizeFreeformItemByPointer(item: CanvasItem, interaction: CanvasInteraction, event: PointerEvent): CanvasItem {
  const dx = event.clientX - interaction.startClientX
  const dy = event.clientY - interaction.startClientY
  const radians = -interaction.origin.rotation * Math.PI / 180
  const localDx = dx * Math.cos(radians) - dy * Math.sin(radians)
  const localDy = dx * Math.sin(radians) + dy * Math.cos(radians)
  const nextWidth = clamp(interaction.origin.width + localDx, 92, 520)
  const nextHeight = clamp(interaction.origin.height + localDy, 72, 420)
  return { ...item, width: nextWidth, height: nextHeight }
}

function resizeImageItemByPointer(item: CanvasItem, interaction: CanvasInteraction, event: PointerEvent, metrics: CanvasStageMetrics | null): CanvasItem {
  const maxWidth = metrics ? Math.max(160, metrics.width - 24) : 900
  const maxHeight = metrics ? Math.max(120, metrics.height - 24) : 720
  const minWidth = 64
  const minHeight = 48
  const centerX = (metrics?.left || 0) + interaction.origin.x + interaction.origin.width / 2
  const centerY = (metrics?.top || 0) + interaction.origin.y + interaction.origin.height / 2
  const startDistance = Math.max(1, Math.hypot(interaction.startClientX - centerX, interaction.startClientY - centerY))
  const currentDistance = Math.max(1, Math.hypot(event.clientX - centerX, event.clientY - centerY))
  const scale = clamp(currentDistance / startDistance, 0.05, 20)
  const { width, height } = imageSizeForScale(item, scale, maxWidth, maxHeight, minWidth, minHeight, interaction.origin)

  const nextX = interaction.origin.x + (interaction.origin.width - width) / 2
  const nextY = interaction.origin.y + (interaction.origin.height - height) / 2

  return {
    ...item,
    x: metrics ? clampCanvasStart(nextX, metrics.width, width) : nextX,
    y: metrics ? clampCanvasStart(nextY, metrics.height, height) : nextY,
    width,
    height,
  }
}

function imageSizeForScale(
  item: CanvasItem,
  scale: number,
  maxWidth: number,
  maxHeight: number,
  minWidth: number,
  minHeight: number,
  origin: Pick<CanvasItem, 'width' | 'height'> = item,
) {
  const aspectRatio = imageAspectRatio(item)
  let width = origin.width * scale
  let height = width / aspectRatio
  const growScale = Math.max(1, minWidth / Math.max(width, 1), minHeight / Math.max(height, 1))
  width *= growScale
  height *= growScale
  const shrinkScale = Math.min(1, maxWidth / Math.max(width, 1), maxHeight / Math.max(height, 1))
  width *= shrinkScale
  height *= shrinkScale

  return {
    width: roundCanvasMetric(width),
    height: roundCanvasMetric(height),
  }
}

function imageAspectRatio(item: CanvasItem) {
  if (item.type !== 'image') return Math.max(0.1, item.width / Math.max(1, item.height))
  const ratio = item.aspectRatio || (item.naturalWidth && item.naturalHeight ? item.naturalWidth / item.naturalHeight : 0) || item.width / Math.max(1, item.height)
  return clamp(ratio, 0.05, 20)
}

export function rotatedItemBounds(item: CanvasItem): { x: number; y: number; width: number; height: number } {
  const { cos, sin } = rotationComponents(item.rotation)
  const width = roundCanvasMetric(item.width * cos + item.height * sin)
  const height = roundCanvasMetric(item.width * sin + item.height * cos)
  return {
    x: roundCanvasMetric((item.width - width) / 2),
    y: roundCanvasMetric((item.height - height) / 2),
    width,
    height,
  }
}

function rotationComponents(rotation: number) {
  const radians = normalizeRotation(rotation) * Math.PI / 180
  return {
    cos: snapUnit(Math.abs(Math.cos(radians))),
    sin: snapUnit(Math.abs(Math.sin(radians))),
  }
}

function snapUnit(value: number) {
  if (value < 0.000001) return 0
  if (1 - value < 0.000001) return 1
  return value
}

function roundCanvasMetric(value: number) {
  return Math.round(value * 1000) / 1000
}

export function autoItemPosition(index: number): CanvasPoint {
  return {
    x: 80 + (index % 4) * 42,
    y: 78 + (index % 5) * 34,
  }
}

export function spreadPoint(point: CanvasPoint, index: number): CanvasPoint {
  return {
    x: point.x + index * 28,
    y: point.y + index * 24,
  }
}

export function dropPointFromEvent(event: ReactDragEvent<HTMLElement>, stage: HTMLElement | null): CanvasPoint {
  return canvasPointFromClient(event.clientX, event.clientY, stage)
}

export function canvasPointFromClient(clientX: number, clientY: number, stage: HTMLElement | null, size: CanvasPointSize = DEFAULT_CANVAS_POINT_SIZE): CanvasPoint {
  const metrics = canvasStageMetrics(stage)
  if (!metrics) return autoItemPosition(0)
  return canvasPointFromStageMetrics(clientX, clientY, metrics, size)
}

export function canvasPointFromStageMetrics(clientX: number, clientY: number, metrics: CanvasStageMetrics, size: CanvasPointSize = DEFAULT_CANVAS_POINT_SIZE): CanvasPoint {
  const width = canvasPointDimension(size.width, DEFAULT_CANVAS_POINT_SIZE.width)
  const height = canvasPointDimension(size.height, DEFAULT_CANVAS_POINT_SIZE.height)
  const inside = isClientPointInsideMetrics(clientX, clientY, metrics)
  const x = inside ? clientX - metrics.left - width / 2 : metrics.width / 2 - width / 2
  const y = inside ? clientY - metrics.top - height / 2 : metrics.height / 2 - height / 2
  return {
    x: clampCanvasStart(x, metrics.width, width),
    y: clampCanvasStart(y, metrics.height, height),
  }
}

export function canvasPointAtStageCenter(stage: HTMLElement | null, size: CanvasPointSize = DEFAULT_CANVAS_POINT_SIZE): CanvasPoint {
  const metrics = canvasStageMetrics(stage)
  if (!metrics) return autoItemPosition(0)
  return canvasPointFromStageMetrics(metrics.left + metrics.width / 2, metrics.top + metrics.height / 2, metrics, size)
}

export function snapshotCanvasStageMetrics(stage: HTMLElement | null): CanvasStageMetrics | null {
  return canvasStageMetrics(stage)
}

export function isClientPointInsideMetrics(clientX: number, clientY: number, metrics: CanvasStageMetrics | null) {
  if (!metrics) return false
  return clientX >= metrics.left && clientX <= metrics.left + metrics.width && clientY >= metrics.top && clientY <= metrics.top + metrics.height
}

export function pointerAngleForItem(item: CanvasItem, clientX: number, clientY: number, stage: HTMLElement | null) {
  const metrics = canvasStageMetrics(stage)
  const centerX = (metrics?.left || 0) + item.x + item.width / 2
  const centerY = (metrics?.top || 0) + item.y + item.height / 2
  return Math.atan2(clientY - centerY, clientX - centerX) * 180 / Math.PI
}

export function itemCenter(item: CanvasItem): CanvasPoint {
  return {
    x: item.x + item.width / 2,
    y: item.y + item.height / 2,
  }
}

export function clamp(value: number, min: number, max: number) {
  return Math.min(max, Math.max(min, value))
}

export function normalizeRotation(value: number) {
  const next = value % 360
  return next > 180 ? next - 360 : next < -180 ? next + 360 : next
}

export type CanvasStageMetrics = {
  left: number
  top: number
  width: number
  height: number
}

function canvasStageMetrics(stage: HTMLElement | null): CanvasStageMetrics | null {
  const rect = stage?.getBoundingClientRect()
  if (!stage || !rect) return null
  return {
    left: rect.left + stage.clientLeft,
    top: rect.top + stage.clientTop,
    width: stage.clientWidth || rect.width,
    height: stage.clientHeight || rect.height,
  }
}

function canvasPointDimension(value: number | undefined, fallback: number) {
  return typeof value === 'number' && Number.isFinite(value) && value > 0 ? value : fallback
}

function clampCanvasStart(value: number, available: number, size: number) {
  return clamp(value, CANVAS_SAFE_INSET, Math.max(CANVAS_SAFE_INSET, available - size - CANVAS_SAFE_INSET))
}

export function nearestConnectableItem(items: readonly CanvasItem[], sourceId: string, maxGap = 44): CanvasItem | null {
  const source = items.find((item) => item.id === sourceId)
  if (!source) return null
  let nearest: { item: CanvasItem; gap: number } | null = null
  for (const item of items) {
    if (item.id === sourceId) continue
    const gap = itemRectGap(source, item)
    if (gap > maxGap) continue
    if (!nearest || gap < nearest.gap) nearest = { item, gap }
  }
  return nearest?.item || null
}

function itemRectGap(a: CanvasItem, b: CanvasItem) {
  const aLeft = a.x
  const aRight = a.x + a.width
  const aTop = a.y
  const aBottom = a.y + a.height
  const bLeft = b.x
  const bRight = b.x + b.width
  const bTop = b.y
  const bBottom = b.y + b.height
  const dx = Math.max(0, Math.max(bLeft - aRight, aLeft - bRight))
  const dy = Math.max(0, Math.max(bTop - aBottom, aTop - bBottom))
  return Math.hypot(dx, dy)
}
