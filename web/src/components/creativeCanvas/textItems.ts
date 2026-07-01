import { canvasPointFromClient } from './geometry'
import type { CanvasImageItem, CanvasItem, CanvasPoint, CanvasTextItem } from './types'

export const CANVAS_TEXT_ITEM_WIDTH = 280
export const CANVAS_TEXT_ITEM_HEIGHT = 118
export const DEFAULT_CANVAS_TEXT = ''

export function isCanvasImageItem(item: CanvasItem | null | undefined): item is CanvasImageItem {
  return item?.type === 'image'
}

export function isCanvasTextItem(item: CanvasItem | null | undefined): item is CanvasTextItem {
  return item?.type === 'text'
}

export function createCanvasTextItem(point: CanvasPoint, index: number, text = DEFAULT_CANVAS_TEXT): CanvasTextItem {
  return {
    type: 'text',
    id: `text-${Date.now()}-${index}`,
    name: `文字 ${index + 1}`,
    text,
    role: 'reference',
    isReference: false,
    x: point.x,
    y: point.y,
    width: CANVAS_TEXT_ITEM_WIDTH,
    height: CANVAS_TEXT_ITEM_HEIGHT,
    rotation: 0,
  }
}

export function updateCanvasTextItemText(items: CanvasItem[], itemId: string, text: string): CanvasItem[] {
  return items.map((item) => (isCanvasTextItem(item) && item.id === itemId ? { ...item, text } : item))
}

export function canvasTextPointFromClient(clientX: number, clientY: number, stage: HTMLElement | null): CanvasPoint {
  return canvasPointFromClient(clientX, clientY, stage, {
    width: CANVAS_TEXT_ITEM_WIDTH,
    height: CANVAS_TEXT_ITEM_HEIGHT,
  })
}
