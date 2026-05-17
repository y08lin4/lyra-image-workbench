import { useMemo, useState } from 'react'
import { createPortal } from 'react-dom'
import { getImageSize, getResolutionLabel, RATIOS, RESOLUTION_TIERS } from '../lib/ratios'

type Props = {
  ratio: string
  resolution: string
  onRatioChange: (value: string) => void
  onResolutionChange: (value: string) => void
}

export function ImageSpecPicker({ ratio, resolution, onRatioChange, onResolutionChange }: Props) {
  const [open, setOpen] = useState(false)
  const [draftRatio, setDraftRatio] = useState(() => ratio || 'auto')
  const [draftResolution, setDraftResolution] = useState(() => resolution || 'auto')

  const currentLabel = useMemo(() => {
    return specLabel(ratio || 'auto', resolution || 'auto')
  }, [ratio, resolution])

  function openDialog() {
    setDraftRatio(ratio || 'auto')
    setDraftResolution(resolution || 'auto')
    setOpen(true)
  }

  function apply() {
    onResolutionChange(draftResolution)
    onRatioChange(draftRatio)
    setOpen(false)
  }

  const previewSize = previewSizeLabel(draftRatio, draftResolution)
  const previewNote = previewSizeNote(draftRatio, draftResolution)
  const modal = open ? (
    <div className="size-modal-mask" onMouseDown={(event) => event.target === event.currentTarget && setOpen(false)}>
      <section className="size-modal image-spec-modal" role="dialog" aria-modal="true" aria-label="设置图像尺寸">
        <header>
          <div>
            <h3>设置图像尺寸</h3>
            <p>当前：{currentLabel}</p>
          </div>
          <button type="button" onClick={() => setOpen(false)} aria-label="关闭尺寸设置">×</button>
        </header>

        <div className="size-modal-body">
          <div className="size-option-groups">
            <section>
              <span>图像比例</span>
              <div className="image-ratio-grid">
                {RATIOS.map((item) => (
                  <button key={item} type="button" className={`image-ratio-card ${draftRatio === item ? 'active' : ''}`} onClick={() => setDraftRatio(item)}>
                    <span className={`image-ratio-preview ${item === 'auto' ? 'auto-preview' : ''}`} style={ratioPreviewStyle(item)}>
                      {item === 'auto' ? <i>AUTO</i> : null}
                    </span>
                    <span className="image-ratio-text">
                      <strong>{ratioTitle(item)}</strong>
                      <span>{ratioHint(item)}</span>
                      <small>{ratioSmall(item, draftResolution)}</small>
                    </span>
                  </button>
                ))}
              </div>
            </section>
            <section>
              <span>分辨率</span>
              <div className="size-choice-grid four">
                {RESOLUTION_TIERS.map((item) => (
                  <button key={item} type="button" className={draftResolution === item ? 'active' : ''} onClick={() => setDraftResolution(item)}>
                    {resolutionTitle(item)}
                  </button>
                ))}
              </div>
            </section>
          </div>

          <div className="size-preview">
            <span>将使用</span>
            <strong>{previewSize}</strong>
            <small>{previewNote}</small>
          </div>
        </div>

        <footer>
          <button type="button" onClick={() => setOpen(false)}>取消</button>
          <button type="button" className="primary" onClick={apply}>确定</button>
        </footer>
      </section>
    </div>
  ) : null

  return (
    <>
      <button type="button" className="size-selector-button" onClick={openDialog}>
        <span>
          <strong>图像尺寸</strong>
          <small>{currentLabel}</small>
        </span>
        <b>设置</b>
      </button>
      {modal ? createPortal(modal, document.body) : null}
    </>
  )
}

function resolutionTitle(value: string) {
  if (value === 'auto') return '自动'
  if (value === 'standard') return '标准 / 1K'
  return getResolutionLabel(value)
}

function ratioPreviewStyle(ratio: string) {
  if (ratio === 'auto') return undefined
  const [w, h] = ratio.split(':').map(Number)
  return { aspectRatio: `${w} / ${h}` }
}

function ratioTitle(ratio: string) {
  return ratio === 'auto' ? '自动比例' : ratio
}

function ratioHint(ratio: string) {
  if (ratio === 'auto') return '不固定画幅'
  const labels: Record<string, string> = {
    '1:1': '方图',
    '2:3': '竖版海报',
    '3:2': '横版照片',
    '3:4': '竖版构图',
    '4:3': '横版构图',
    '9:16': '手机竖屏',
    '16:9': '宽屏横图',
  }
  return labels[ratio] || '固定比例'
}

function ratioSmall(ratio: string, resolution: string) {
  if (ratio === 'auto') return '由模型或上游决定'
  if (resolution === 'auto') return `按标准 ${getImageSize(ratio, resolution)}`
  return getImageSize(ratio, resolution)
}

function specLabel(ratio: string, resolution: string) {
  if (ratio === 'auto' && resolution === 'auto') return '自动比例 · 自动分辨率 · 自动尺寸'
  if (ratio === 'auto') return `${resolutionTitle(resolution)} · 自动比例 · 自动尺寸`
  if (resolution === 'auto') return `自动分辨率 · ${ratio} · 标准 ${getImageSize(ratio, resolution)}`
  return `${resolutionTitle(resolution)} · ${ratio} · ${getImageSize(ratio, resolution)}`
}

function previewSizeLabel(ratio: string, resolution: string) {
  if (ratio === 'auto') return '自动尺寸'
  if (resolution === 'auto') return `标准 ${getImageSize(ratio, resolution)}`
  return getImageSize(ratio, resolution)
}

function previewSizeNote(ratio: string, resolution: string) {
  if (ratio === 'auto' && resolution === 'auto') return '不传具体 size，由模型或上游自动决定'
  if (ratio === 'auto') return `分辨率记录为${resolutionTitle(resolution)}，但比例自动时不传具体 size`
  if (resolution === 'auto') return '分辨率自动时沿用当前后端逻辑，按标准尺寸提交'
  return `${resolutionTitle(resolution)} + ${ratio}`
}
