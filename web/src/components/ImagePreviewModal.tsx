import { useEffect, useState } from 'react'
import { createPortal } from 'react-dom'
import { formatBytes } from '../lib/format'

type ImageDimensions = { width: number; height: number }
type PreviewAction = () => void | string | Promise<void | string>

type Props = {
  src: string
  title: string
  requestedSize?: string
  ratio?: string
  bytes?: number
  onCopyImage?: PreviewAction
  onCopyUrl?: PreviewAction
  onDownload?: PreviewAction
  onUseAsReference?: PreviewAction
  onClose: () => void
}

export function ImagePreviewModal({ src, title, bytes, onCopyImage, onCopyUrl, onDownload, onUseAsReference, onClose }: Props) {
  const [dimensions, setDimensions] = useState<ImageDimensions>()
  const [byteSize, setByteSize] = useState(bytes || 0)
  const [notice, setNotice] = useState('')

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

  const actualRatio = formatActualRatio(dimensions)

  async function runAction(action: PreviewAction | undefined, fallback: string) {
    if (!action) return
    try {
      const result = await action()
      setNotice(typeof result === 'string' && result ? result : fallback)
    } catch (err) {
      setNotice(err instanceof Error ? err.message : '操作失败')
    }
    window.setTimeout(() => setNotice(''), 1800)
  }

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
        {notice ? <div className="preview-notice" role="status">{notice}</div> : null}
        <div className="preview-actions">
          {onDownload ? <button type="button" onClick={() => void runAction(onDownload, '下载已触发')}>下载</button> : null}
          {onCopyImage ? <button type="button" onClick={() => void runAction(onCopyImage, '图片已复制')}>复制图片</button> : null}
          {onCopyUrl ? <button type="button" onClick={() => void runAction(onCopyUrl, '链接已复制')}>复制链接</button> : null}
          {onUseAsReference ? <button type="button" onClick={() => void runAction(onUseAsReference, '已加入参考图')}>作为参考图</button> : null}
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
