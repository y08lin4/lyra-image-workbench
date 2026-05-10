import { useEffect, useState } from 'react'
import { createPortal } from 'react-dom'
import { formatBytes } from '../../lib/format'
import { PreviewImageStage, type ImageDimensions } from './PreviewImageStage'
import { PreviewMetaBar } from './PreviewMetaBar'
import { PreviewToolbar, type PreviewAction } from './PreviewToolbar'
import './preview.css'

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

  function flash(value: string) {
    setNotice(value)
    window.setTimeout(() => setNotice(''), 1800)
  }

  const sizeLabel = byteSize ? formatBytes(byteSize) : '\u8bfb\u53d6\u5927\u5c0f\u4e2d'

  return createPortal(
    <div className="image-preview-mask" onMouseDown={(event) => event.target === event.currentTarget && onClose()}>
      <section className="image-preview-shell" role="dialog" aria-modal="true" aria-label={title}>
        <button type="button" className="image-preview-close" onClick={onClose} aria-label={'\u5173\u95ed\u9884\u89c8'}>{'\u00d7'}</button>
        <PreviewMetaBar dimensions={dimensions} byteSizeLabel={sizeLabel} />
        <PreviewImageStage src={src} title={title} onDimensions={setDimensions} onClose={onClose} />
        {notice ? <div className="image-preview-toast" role="status">{notice}</div> : null}
        <PreviewToolbar
          onDownload={onDownload}
          onCopyImage={onCopyImage}
          onCopyUrl={onCopyUrl}
          onUseAsReference={onUseAsReference}
          onNotice={flash}
        />
      </section>
    </div>,
    document.body,
  )
}
