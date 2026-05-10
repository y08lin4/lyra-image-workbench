import type { ImageDimensions } from './PreviewImageStage'

type Props = {
  dimensions?: ImageDimensions
  byteSizeLabel: string
}

export function PreviewMetaBar({ dimensions, byteSizeLabel }: Props) {
  return (
    <div className="image-preview-meta" aria-label="????">
      <span>{formatDimensions(dimensions)}</span>
      <span>{formatActualRatio(dimensions)}</span>
      <span>{byteSizeLabel}</span>
    </div>
  )
}

function formatDimensions(dimensions?: ImageDimensions) {
  if (!dimensions) return '?????'
  return `${dimensions.width}?${dimensions.height}`
}

function formatActualRatio(dimensions?: ImageDimensions) {
  if (!dimensions) return '?????'
  const divisor = gcd(dimensions.width, dimensions.height)
  return `${dimensions.width / divisor}:${dimensions.height / divisor}`
}

function gcd(a: number, b: number): number {
  while (b) {
    const t = b
    b = a % b
    a = t
  }
  return Math.max(1, a)
}
