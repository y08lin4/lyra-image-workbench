import type { ModelProvider } from '../types'

export const IMAGE2_PROVIDER: ModelProvider = 'image-2'
export const BANANA_PROVIDER: ModelProvider = 'banana'
export const IMAGE2_MODEL = 'image-2'
export const IMAGE2_4K_MODEL = 'image-2-4k'
export const LEGACY_GPT_IMAGE2_MODEL = 'gpt-image-2'
export const DEFAULT_IMAGE2_MODEL = IMAGE2_MODEL
export const DEFAULT_BANANA_MODEL = 'gemini-3.1-flash-image-preview'

export type Image2ModelOption = {
  id: string
  label: string
  hint: string
  ratioSelectable: boolean
  defaultRatio: string
  defaultResolution: string
}

export const IMAGE2_MODEL_OPTIONS: Image2ModelOption[] = [
  {
    id: IMAGE2_MODEL,
    label: 'image-2',
    hint: '基础版不提交 size；质量和格式仍会提交',
    ratioSelectable: false,
    defaultRatio: 'auto',
    defaultResolution: 'auto',
  },
  {
    id: IMAGE2_4K_MODEL,
    label: 'image-2（满血版）',
    hint: '可选比例、分辨率，也支持自定义像素尺寸',
    ratioSelectable: true,
    defaultRatio: 'auto',
    defaultResolution: 'auto',
  },
]

export function normalizeImage2Model(id?: string) {
  const normalized = (id || '').trim()
  if (!normalized || normalized === LEGACY_GPT_IMAGE2_MODEL) return DEFAULT_IMAGE2_MODEL
  return IMAGE2_MODEL_OPTIONS.find((item) => item.id === normalized)?.id || DEFAULT_IMAGE2_MODEL
}

export function getImage2ModelOption(id?: string) {
  const normalized = normalizeImage2Model(id)
  return IMAGE2_MODEL_OPTIONS.find((item) => item.id === normalized) || IMAGE2_MODEL_OPTIONS[0]
}

export function image2ModelAllowsRatio(id?: string) {
  return getImage2ModelOption(id).ratioSelectable
}

export function image2ModelSubmissionSpec(id: string, ratio: string, resolution: string, size = '') {
  const option = getImage2ModelOption(id)
  const selectedResolution = resolution && resolution !== 'auto' ? resolution : option.defaultResolution
  return {
    model: option.id,
    ratio: option.ratioSelectable ? image2SelectableRatio(option, ratio) : option.defaultRatio,
    resolution: selectedResolution,
    size: option.ratioSelectable ? size.trim() : '',
  }
}

export function image2SelectableRatio(option: Image2ModelOption, ratio?: string) {
  const selected = (ratio || '').trim()
  if (!selected || (selected === 'auto' && option.defaultRatio !== 'auto')) return option.defaultRatio
  return selected
}

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
