import type { ModelProvider } from '../types'

export const IMAGE2_PROVIDER: ModelProvider = 'image-2'
export const BANANA_PROVIDER: ModelProvider = 'banana'
export const DEFAULT_IMAGE2_MODEL = 'gpt-image-2'
export const DEFAULT_BANANA_MODEL = 'gemini-3.1-flash-image-preview'

export type BananaModelOption = {
  id: string
  label: string
  ratio: string
  resolution: string
  size: string
  hint: string
}

export const BANANA_MODEL_OPTIONS: BananaModelOption[] = [
  { id: 'gemini-3.1-flash-image-preview', label: '自动', ratio: 'auto', resolution: 'auto', size: '自动', hint: '模型自动决定比例和尺寸' },
  { id: 'gemini-3.1-flash-image-preview-2k', label: '自动 · 2K', ratio: 'auto', resolution: '2k', size: '自动', hint: '2K，比例自动' },
  { id: 'gemini-3.1-flash-image-preview-4k', label: '自动 · 4K', ratio: 'auto', resolution: '4k', size: '自动', hint: '4K，比例自动' },
  { id: 'gemini-3.1-flash-image-preview-16x9-2k', label: '16:9 · 2K', ratio: '16:9', resolution: '2k', size: '2048x1152', hint: '横屏 2K' },
  { id: 'gemini-3.1-flash-image-preview-16x9-4k', label: '16:9 · 4K', ratio: '16:9', resolution: '4k', size: '3840x2160', hint: '横屏 4K' },
  { id: 'gemini-3.1-flash-image-preview-9x16-2k', label: '9:16 · 2K', ratio: '9:16', resolution: '2k', size: '1152x2048', hint: '竖屏 2K' },
  { id: 'gemini-3.1-flash-image-preview-9x16-4k', label: '9:16 · 4K', ratio: '9:16', resolution: '4k', size: '2160x3840', hint: '竖屏 4K' },
  { id: 'gemini-3.1-flash-image-preview-4x3-2k', label: '4:3 · 2K', ratio: '4:3', resolution: '2k', size: '2048x1536', hint: '横向 4:3 2K' },
  { id: 'gemini-3.1-flash-image-preview-4x3-4k', label: '4:3 · 4K', ratio: '4:3', resolution: '4k', size: '3264x2448', hint: '横向 4:3 4K' },
  { id: 'gemini-3.1-flash-image-preview-3x4-2k', label: '3:4 · 2K', ratio: '3:4', resolution: '2k', size: '1536x2048', hint: '竖向 3:4 2K' },
  { id: 'gemini-3.1-flash-image-preview-3x4-4k', label: '3:4 · 4K', ratio: '3:4', resolution: '4k', size: '2448x3264', hint: '竖向 3:4 4K' },
  { id: 'gemini-3.1-flash-image-preview-3x2-2k', label: '3:2 · 2K', ratio: '3:2', resolution: '2k', size: '2016x1344', hint: '横版照片 2K' },
  { id: 'gemini-3.1-flash-image-preview-3x2-4k', label: '3:2 · 4K', ratio: '3:2', resolution: '4k', size: '3504x2336', hint: '横版照片 4K' },
  { id: 'gemini-3.1-flash-image-preview-2x3-2k', label: '2:3 · 2K', ratio: '2:3', resolution: '2k', size: '1344x2016', hint: '竖版海报 2K' },
  { id: 'gemini-3.1-flash-image-preview-2x3-4k', label: '2:3 · 4K', ratio: '2:3', resolution: '4k', size: '2336x3504', hint: '竖版海报 4K' },
  { id: 'gemini-3.1-flash-image-preview-1x1-2k', label: '1:1 · 2K', ratio: '1:1', resolution: '2k', size: '2048x2048', hint: '方图 2K' },
  { id: 'gemini-3.1-flash-image-preview-1x1-4k', label: '1:1 · 4K', ratio: '1:1', resolution: '4k', size: '2880x2880', hint: '方图 4K' },
]

export function getBananaModelOption(id: string) {
  return BANANA_MODEL_OPTIONS.find((item) => item.id === id) || BANANA_MODEL_OPTIONS[0]
}

export function getBananaModelForRatio(ratio?: string, preferredResolution = '2k') {
  const normalizedRatio = (ratio || '').trim()
  const normalizedResolution = preferredResolution === '4k' ? '4k' : preferredResolution === '2k' ? '2k' : ''
  if (!normalizedRatio || normalizedRatio === 'auto') {
    return normalizedResolution
      ? BANANA_MODEL_OPTIONS.find((item) => item.ratio === 'auto' && item.resolution === normalizedResolution) || getBananaModelOption(DEFAULT_BANANA_MODEL)
      : getBananaModelOption(DEFAULT_BANANA_MODEL)
  }
  return (normalizedResolution ? BANANA_MODEL_OPTIONS.find((item) => item.ratio === normalizedRatio && item.resolution === normalizedResolution) : undefined)
    || BANANA_MODEL_OPTIONS.find((item) => item.ratio === normalizedRatio)
    || getBananaModelOption(DEFAULT_BANANA_MODEL)
}

export function providerLabel(provider?: string) {
  return provider === BANANA_PROVIDER ? 'Banana Nano' : 'Image-2'
}
