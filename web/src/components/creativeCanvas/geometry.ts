import type { CSSProperties, DragEvent as ReactDragEvent } from 'react'
import type { CanvasInteraction, CanvasItem, CanvasPoint } from './types'

export function updateCanvasItemsForInteraction(items: CanvasItem[], interaction: CanvasInteraction, event: PointerEvent, stage: HTMLElement | null): CanvasItem[] {
  const rect = stage?.getBoundingClientRect()
  return items.map((item) => {
    if (item.id !== interaction.itemId) return item
    if (interaction.type === 'move') {
      const nextX = interaction.origin.x + event.clientX - interaction.startClientX
      const nextY = interaction.origin.y + event.clientY - interaction.startClientY
      return {
        ...item,
        x: rect ? clamp(nextX, 8, Math.max(8, rect.width - item.width - 8)) : nextX,
        y: rect ? clamp(nextY, 8, Math.max(8, rect.height - item.height - 8)) : nextY,
      }
    }
    if (interaction.type === 'resize') {
      const dx = event.clientX - interaction.startClientX
      const dy = event.clientY - interaction.startClientY
      const radians = -interaction.origin.rotation * Math.PI / 180
      const localDx = dx * Math.cos(radians) - dy * Math.sin(radians)
      const localDy = dx * Math.sin(radians) + dy * Math.cos(radians)
      const nextWidth = clamp(interaction.origin.width + localDx, 92, 520)
      const nextHeight = clamp(interaction.origin.height + localDy, 72, 420)
      return { ...item, width: nextWidth, height: nextHeight }
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
    transform: `translate(${item.x}px, ${item.y}px)`,
  }
}

export function canvasItemContentStyle(item: CanvasItem): CSSProperties {
  return {
    transform: `rotate(${item.rotation}deg)`,
  }
}

export function canvasControlStyle(item: CanvasItem): CSSProperties {
  const bounds = rotatedItemBounds(item)
  return {
    left: bounds.x,
    top: bounds.y,
    right: 'auto',
    bottom: 'auto',
    width: bounds.width,
    height: bounds.height,
  }
}

export function rotatedItemBounds(item: CanvasItem): { x: number; y: number; width: number; height: number } {
  const radians = item.rotation * Math.PI / 180
  const cos = Math.abs(Math.cos(radians))
  const sin = Math.abs(Math.sin(radians))
  const width = roundCanvasMetric(item.width * cos + item.height * sin)
  const height = roundCanvasMetric(item.width * sin + item.height * cos)
  return {
    x: roundCanvasMetric((item.width - width) / 2),
    y: roundCanvasMetric((item.height - height) / 2),
    width,
    height,
  }
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

export function canvasPointFromClient(clientX: number, clientY: number, stage: HTMLElement | null): CanvasPoint {
  const rect = stage?.getBoundingClientRect()
  if (!rect) return autoItemPosition(0)
  return {
    x: clamp(clientX - rect.left - 110, 8, Math.max(8, rect.width - 240)),
    y: clamp(clientY - rect.top - 78, 8, Math.max(8, rect.height - 180)),
  }
}

export function pointerAngleForItem(item: CanvasItem, clientX: number, clientY: number, stage: HTMLElement | null) {
  const rect = stage?.getBoundingClientRect()
  const centerX = (rect?.left || 0) + item.x + item.width / 2
  const centerY = (rect?.top || 0) + item.y + item.height / 2
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


