import type { ModelProvider } from '../types'

export const IMAGE2_PROVIDER: ModelProvider = 'image-2'
export const IMAGE2_MODEL = 'image-2'
export const IMAGE2_4K_MODEL = 'image-2-4k'
export const LEGACY_GPT_IMAGE2_MODEL = 'gpt-image-2'
export const DEFAULT_IMAGE2_MODEL = IMAGE2_MODEL

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

export function providerLabel(_provider?: string) {
  return 'Image-2'
}