import { useMemo, useState } from 'react'
import { createPortal } from 'react-dom'
import { FIXED_RATIOS, getImageSize, getResolutionLabel, RATIOS, RESOLUTION_TIERS } from '../lib/ratios'

type Props = {
  ratio: string
  resolution: string
  size?: string
  onRatioChange: (value: string) => void
  onResolutionChange: (value: string) => void
  onSizeChange?: (value: string) => void
  ratioSelectable?: boolean
  ratioLockedHint?: string
  allowAutoRatio?: boolean
  customSizeEnabled?: boolean
}

export function ImageSpecPicker({ ratio, resolution, size = '', onRatioChange, onResolutionChange, onSizeChange, ratioSelectable = true, ratioLockedHint, allowAutoRatio = true, customSizeEnabled = ratioSelectable }: Props) {
  const [open, setOpen] = useState(false)
  const [draftRatio, setDraftRatio] = useState(() => ratio || 'auto')
  const [draftResolution, setDraftResolution] = useState(() => resolution || 'auto')
  const [draftSize, setDraftSize] = useState(() => normalizeCustomSizeInput(size))

  const ratioOptions = useMemo(() => (allowAutoRatio ? RATIOS : FIXED_RATIOS), [allowAutoRatio])
  const normalizedRatio = normalizePickerRatio(ratio, ratioSelectable, allowAutoRatio)
  const normalizedSize = normalizeCustomSizeInput(size)
  const currentLabel = useMemo(() => {
    return normalizedSize ? `自定义 ${normalizedSize}` : specLabel(normalizedRatio, resolution || 'auto')
  }, [normalizedRatio, normalizedSize, resolution])

  function openDialog() {
    setDraftRatio(normalizedRatio)
    setDraftResolution(resolution || 'auto')
    setDraftSize(normalizedSize)
    setOpen(true)
  }

  function apply() {
    const nextSize = customSizeEnabled ? normalizeCustomSizeInput(draftSize) : ''
    onSizeChange?.(nextSize)
    if (nextSize) {
      onResolutionChange('auto')
      onRatioChange('auto')
    } else {
      onResolutionChange(draftResolution)
      onRatioChange(ratioSelectable ? normalizePickerRatio(draftRatio, ratioSelectable, allowAutoRatio) : 'auto')
    }
    setOpen(false)
  }

  function selectRatio(nextRatio: string) {
    setDraftRatio(nextRatio)
    setDraftSize('')
  }

  function selectResolution(nextResolution: string) {
    setDraftResolution(nextResolution)
    setDraftSize('')
  }

  const effectiveDraftRatio = ratioSelectable ? normalizePickerRatio(draftRatio, ratioSelectable, allowAutoRatio) : 'auto'
  const effectiveDraftSize = normalizeCustomSizeInput(draftSize)
  const previewSize = effectiveDraftSize || previewSizeLabel(effectiveDraftRatio, draftResolution)
  const previewNote = effectiveDraftSize ? customSizeNote(effectiveDraftSize) : previewSizeNote(effectiveDraftRatio, draftResolution)
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
            {ratioSelectable ? (
              <section>
                <span>图像比例</span>
                <div className="image-ratio-grid">
                  {ratioOptions.map((item) => (
                    <button key={item} type="button" className={`image-ratio-card ${!effectiveDraftSize && draftRatio === item ? 'active' : ''}`} onClick={() => selectRatio(item)}>
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
            ) : (
              <p className="muted">{ratioLockedHint || '当前模型会自动决定画幅，提交时 ratio=auto。'}</p>
            )}
            <section>
              <span>分辨率</span>
              <div className="size-choice-grid four">
                {RESOLUTION_TIERS.map((item) => (
                  <button key={item} type="button" className={!effectiveDraftSize && draftResolution === item ? 'active' : ''} onClick={() => selectResolution(item)}>
                    {resolutionTitle(item)}
                  </button>
                ))}
              </div>
            </section>
            {customSizeEnabled ? (
              <section className="custom-size-section">
                <span>自定义像素</span>
                <label className="custom-size-field">
                  <input value={draftSize} onChange={(event) => setDraftSize(event.target.value)} placeholder="1536x864" inputMode="text" spellCheck={false} />
                </label>
                <small>留空使用上方预设；自定义尺寸会优先作为 size 提交。</small>
              </section>
            ) : null}
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

function normalizePickerRatio(ratio: string, ratioSelectable: boolean, allowAutoRatio: boolean) {
  if (!ratioSelectable) return 'auto'
  if (!ratio || (ratio === 'auto' && !allowAutoRatio)) return '1:1'
  return ratio
}

function normalizeCustomSizeInput(value: string) {
  return value.trim().toLowerCase().replace(/×/g, 'x').replace(/\s+/g, '')
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
  if (resolution === 'auto') return '分辨率自动时按标准尺寸提交'
  return `${resolutionTitle(resolution)} + ${ratio}`
}

function customSizeNote(size: string) {
  return `${size} 会作为 OpenAI 兼容 size 直接提交；请确保宽高均可被 16 整除。`
}
