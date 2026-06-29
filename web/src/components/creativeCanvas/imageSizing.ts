import type { Mode } from '../../types'

export function aspectRatioValue(ratio: string) {
  if (!ratio || ratio === 'auto') return '1 / 1'
  return ratio.replace(':', ' / ')
}

export function extensionLabel(mime: string) {
  if (mime.includes('jpeg')) return 'JPG'
  if (mime.includes('webp')) return 'WEBP'
  return 'PNG'
}

export function modeLabel(mode: Mode) {
  return mode === 'image-to-image' ? '参考图生成' : '文字生成'
}

export type CanvasImageDisplaySize = {
  width: number
  height: number
  naturalWidth?: number
  naturalHeight?: number
  aspectRatio?: number
}

const DEFAULT_CANVAS_IMAGE_SIZE: CanvasImageDisplaySize = { width: 220, height: 156 }
const CANVAS_IMAGE_MIN_EDGE = 96
const CANVAS_IMAGE_MAX_EDGE = 260

export async function canvasImageSizeFromSrc(src?: string): Promise<CanvasImageDisplaySize> {
  const naturalSize = await loadImageNaturalSize(src)
  return fitCanvasImageSize(naturalSize?.width, naturalSize?.height)
}

export async function canvasImageSizeFromFile(file: File): Promise<CanvasImageDisplaySize> {
  if (typeof createImageBitmap === 'function') {
    try {
      const bitmap = await createImageBitmap(file)
      const size = fitCanvasImageSize(bitmap.width, bitmap.height)
      bitmap.close()
      return size
    } catch {
      // Fall back to an object URL decode below.
    }
  }

  if (typeof URL === 'undefined') return DEFAULT_CANVAS_IMAGE_SIZE
  const url = URL.createObjectURL(file)
  try {
    return await canvasImageSizeFromSrc(url)
  } finally {
    URL.revokeObjectURL(url)
  }
}

export function fitCanvasImageSize(naturalWidth?: number, naturalHeight?: number): CanvasImageDisplaySize {
  const width = Number(naturalWidth)
  const height = Number(naturalHeight)
  if (!Number.isFinite(width) || !Number.isFinite(height) || width <= 0 || height <= 0) {
    return DEFAULT_CANVAS_IMAGE_SIZE
  }

  const longestEdge = Math.max(width, height)
  const targetLongestEdge = clamp(longestEdge, CANVAS_IMAGE_MIN_EDGE, CANVAS_IMAGE_MAX_EDGE)
  const scale = targetLongestEdge / longestEdge

  return {
    width: Math.max(1, Math.round(width * scale)),
    height: Math.max(1, Math.round(height * scale)),
    naturalWidth: Math.round(width),
    naturalHeight: Math.round(height),
    aspectRatio: width / height,
  }
}

function loadImageNaturalSize(src?: string): Promise<{ width: number; height: number } | null> {
  if (!src || typeof window === 'undefined') return Promise.resolve(null)

  return new Promise((resolve) => {
    const image = new Image()
    image.onload = () => resolve({
      width: image.naturalWidth || image.width,
      height: image.naturalHeight || image.height,
    })
    image.onerror = () => resolve(null)
    image.src = src
  })
}

function clamp(value: number, min: number, max: number) {
  return Math.min(max, Math.max(min, value))
}