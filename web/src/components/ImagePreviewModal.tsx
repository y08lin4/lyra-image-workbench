import { useEffect, useMemo, useState } from 'react'
import { createPortal } from 'react-dom'
import { formatBytes } from '../lib/format'

type ImageDimensions = { width: number; height: number }

type Props = {
  src: string
  title: string
  requestedSize?: string
  ratio?: string
  bytes?: number
  onCopyImage?: () => void | Promise<void>
  onCopyUrl?: () => void | Promise<void>
  onDownload?: () => void | Promise<void>
  onUseAsReference?: () => void | Promise<void>
  onClose: () => void
}

export function ImagePreviewModal({ src, title, bytes, onCopyImage, onCopyUrl, onDownload, onUseAsReference, onClose }: Props) {
  const [dimensions, setDimensions] = useState<ImageDimensions>()
  const [byteSize, setByteSize] = useState(bytes || 0)

  useEffect(() => {
    setDimensions(undefined)
  }, [src])

  useEffect(() => {
    if (bytes) {
      setByteSize(bytes)
      return
    }
    let cancelled = false
    setByteSize(0)
    void fetch(src)
      .then((response) => response.blob())
      .then((blob) => {
        if (!cancelled) setByteSize(blob.size)
      })
      .catch(() => {
        if (!cancelled) setByteSize(0)
      })
    return () => {
      cancelled = true
    }
  }, [src, bytes])

  useEffect(() => {
    const previousOverflow = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') onClose()
    }
    window.addEventListener('keydown', onKeyDown)
    return () => {
      document.body.style.overflow = previousOverflow
      window.removeEventListener('keydown', onKeyDown)
    }
  }, [onClose])

  const actualRatio = useMemo(() => formatActualRatio(dimensions), [dimensions])

  return createPortal(
    <div className="preview-mask" onMouseDown={(event) => event.target === event.currentTarget && onClose()}>
      <div className="preview-dialog" role="dialog" aria-modal="true" aria-label={title}>
        <button type="button" className="preview-close" onClick={onClose} aria-label="关闭预览">×</button>
        <div className="preview-info">
          <span>{formatDimensions(dimensions)}</span>
          <span>{actualRatio}</span>
          <span>{byteSize ? formatBytes(byteSize) : '读取大小中'}</span>
        </div>
        <div className="preview-stage">
          <img
            src={src}
            alt={title}
            onLoad={(event) => setDimensions({
              width: event.currentTarget.naturalWidth,
              height: event.currentTarget.naturalHeight,
            })}
          />
        </div>
        <div className="preview-actions">
          {onDownload ? <button type="button" onClick={() => void onDownload()}>下载</button> : null}
          {onCopyImage ? <button type="button" onClick={() => void onCopyImage()}>复制图片</button> : null}
          {onCopyUrl ? <button type="button" onClick={() => void onCopyUrl()}>复制链接</button> : null}
          {onUseAsReference ? <button type="button" onClick={() => void onUseAsReference()}>作为参考图</button> : null}
        </div>
      </div>
    </div>,
    document.body,
  )
}

function formatDimensions(dimensions?: ImageDimensions) {
  if (!dimensions) return '读取尺寸中'
  return `${dimensions.width}×${dimensions.height}`
}

function formatActualRatio(dimensions?: ImageDimensions) {
  if (!dimensions) return '读取比例中'
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
