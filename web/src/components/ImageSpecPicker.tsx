import { useMemo, useState } from 'react'
import { createPortal } from 'react-dom'
import { FIXED_RATIOS, getImageSize, getResolutionLabel, RESOLUTION_TIERS } from '../lib/ratios'

type Props = {
  ratio: string
  resolution: string
  onRatioChange: (value: string) => void
  onResolutionChange: (value: string) => void
}

const RESOLUTION_OPTIONS = RESOLUTION_TIERS.filter((item) => item !== 'auto')

export function ImageSpecPicker({ ratio, resolution, onRatioChange, onResolutionChange }: Props) {
  const [open, setOpen] = useState(false)
  const [mode, setMode] = useState<'auto' | 'ratio'>(() => ratio === 'auto' ? 'auto' : 'ratio')
  const [draftRatio, setDraftRatio] = useState(() => ratio === 'auto' ? '1:1' : ratio)
  const [draftResolution, setDraftResolution] = useState(() => resolution === 'auto' ? 'standard' : resolution)

  const currentLabel = useMemo(() => {
    if (ratio === 'auto') return '自动尺寸'
    const tier = resolution === 'auto' ? 'standard' : resolution
    return `${resolutionTitle(tier)} · ${ratio} · ${getImageSize(ratio, tier)}`
  }, [ratio, resolution])

  function openDialog() {
    setMode(ratio === 'auto' ? 'auto' : 'ratio')
    setDraftRatio(ratio === 'auto' ? '1:1' : ratio)
    setDraftResolution(resolution === 'auto' ? 'standard' : resolution)
    setOpen(true)
  }

  function apply() {
    if (mode === 'auto') {
      onResolutionChange('auto')
      onRatioChange('auto')
    } else {
      onResolutionChange(draftResolution)
      onRatioChange(draftRatio)
    }
    setOpen(false)
  }

  const previewSize = mode === 'auto' ? 'auto' : getImageSize(draftRatio, draftResolution)
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

        <div className="size-tabs" role="tablist" aria-label="尺寸模式">
          <button type="button" className={mode === 'auto' ? 'active' : ''} onClick={() => setMode('auto')}>自动</button>
          <button type="button" className={mode === 'ratio' ? 'active' : ''} onClick={() => setMode('ratio')}>按比例</button>
        </div>

        {mode === 'auto' ? (
          <div className="image-ratio-grid image-ratio-grid-auto">
            <button type="button" className="image-ratio-card active">
              <span className="image-ratio-preview auto-preview">
                <i>AUTO</i>
              </span>
              <span className="image-ratio-text">
                <strong>自动尺寸</strong>
                <span>不传具体 size</span>
                <small>由模型或上游自动决定</small>
              </span>
            </button>
          </div>
        ) : (
          <div className="size-option-groups">
            <section>
              <span>基准分辨率</span>
              <div className="size-choice-grid three">
                {RESOLUTION_OPTIONS.map((item) => (
                  <button key={item} type="button" className={draftResolution === item ? 'active' : ''} onClick={() => setDraftResolution(item)}>
                    {resolutionTitle(item)}
                  </button>
                ))}
              </div>
            </section>
            <section>
              <span>图像比例</span>
              <div className="image-ratio-grid">
                {FIXED_RATIOS.map((item) => (
                  <button key={item} type="button" className={`image-ratio-card ${draftRatio === item ? 'active' : ''}`} onClick={() => setDraftRatio(item)}>
                    <span className="image-ratio-preview" style={ratioPreviewStyle(item)} />
                    <span className="image-ratio-text">
                      <strong>{item}</strong>
                      <span>{ratioHint(item)}</span>
                      <small>{getImageSize(item, draftResolution)}</small>
                    </span>
                  </button>
                ))}
              </div>
            </section>
          </div>
        )}

        <div className="size-preview">
          <span>将使用</span>
          <strong>{mode === 'auto' ? 'auto' : previewSize}</strong>
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
  if (value === 'standard') return '标准 / 1K'
  return getResolutionLabel(value)
}

function ratioPreviewStyle(ratio: string) {
  const [w, h] = ratio.split(':').map(Number)
  return { aspectRatio: `${w} / ${h}` }
}

function ratioHint(ratio: string) {
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
