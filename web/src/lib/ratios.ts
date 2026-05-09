export const RATIOS = ['auto', '1:1', '2:3', '3:2', '3:4', '4:3', '9:16', '16:9'] as const
export const FIXED_RATIOS = ['1:1', '2:3', '3:2', '3:4', '4:3', '9:16', '16:9'] as const
export const RESOLUTION_TIERS = ['auto', 'standard', '2k', '4k'] as const
export const QUALITY_LEVELS = ['auto', 'low', 'medium', 'high'] as const

export type AspectRatio = typeof RATIOS[number]
export type FixedRatio = typeof FIXED_RATIOS[number]
export type ResolutionTier = typeof RESOLUTION_TIERS[number]
export type QualityLevel = typeof QUALITY_LEVELS[number]

export const RESOLUTION_LABEL: Record<ResolutionTier, string> = {
  auto: '自动',
  standard: '标准',
  '2k': '2K',
  '4k': '4K',
}

export const QUALITY_LABEL: Record<QualityLevel, string> = {
  auto: '自动',
  low: '低',
  medium: '中',
  high: '高',
}

export const SIZE_MAP: Record<Exclude<ResolutionTier, 'auto'>, Record<FixedRatio, string>> = {
  standard: {
    '1:1': '1024x1024',
    '2:3': '1024x1536',
    '3:2': '1536x1024',
    '3:4': '768x1024',
    '4:3': '1024x768',
    '9:16': '1008x1792',
    '16:9': '1792x1008',
  },
  '2k': {
    '1:1': '2048x2048',
    '2:3': '1344x2016',
    '3:2': '2016x1344',
    '3:4': '1536x2048',
    '4:3': '2048x1536',
    '9:16': '1152x2048',
    '16:9': '2048x1152',
  },
  '4k': {
    '1:1': '2880x2880',
    '2:3': '2336x3504',
    '3:2': '3504x2336',
    '3:4': '2448x3264',
    '4:3': '3264x2448',
    '9:16': '2160x3840',
    '16:9': '3840x2160',
  },
}

export function isAspectRatio(value: string): value is AspectRatio {
  return RATIOS.includes(value as AspectRatio)
}

export function isResolutionTier(value: string): value is ResolutionTier {
  return RESOLUTION_TIERS.includes(value as ResolutionTier)
}

export function isFixedRatio(value: string): value is FixedRatio {
  return FIXED_RATIOS.includes(value as FixedRatio)
}

export function getResolutionLabel(resolution: string) {
  return isResolutionTier(resolution) ? RESOLUTION_LABEL[resolution] : resolution
}

export function getQualityLabel(quality: string) {
  return QUALITY_LEVELS.includes(quality as QualityLevel) ? QUALITY_LABEL[quality as QualityLevel] : quality
}

export function getAvailableRatios(resolution: string): readonly AspectRatio[] {
  return resolution === 'auto' ? RATIOS : FIXED_RATIOS
}

export function normalizeRatioForResolution(ratio: string, resolution: string): AspectRatio {
  const available = getAvailableRatios(resolution)
  return available.includes(ratio as AspectRatio) ? ratio as AspectRatio : available[0]
}

export function getImageSize(ratio: string, resolution: string) {
  if (!isFixedRatio(ratio)) return '自动'
  const tier = isResolutionTier(resolution) && resolution !== 'auto' ? resolution : 'standard'
  return SIZE_MAP[tier][ratio]
}

export function getRatioPreviewStyle(ratio: string) {
  if (!isFixedRatio(ratio)) {
    return { width: '18px', height: '18px' }
  }
  const [w, h] = ratio.split(':').map(Number)
  const maxW = 20
  const maxH = 20
  const scale = Math.min(maxW / w, maxH / h)
  return {
    width: `${Math.max(6, w * scale)}px`,
    height: `${Math.max(6, h * scale)}px`,
  }
}
