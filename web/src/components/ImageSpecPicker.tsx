import { useMemo, useState } from 'react'
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

  return (
    <>
      <button type="button" className="size-selector-button" onClick={openDialog}>
        <span>
          <strong>图像尺寸</strong>
          <small>{currentLabel}</small>
        </span>
        <b>设置</b>
      </button>
      {open ? (
        <div className="size-modal-mask" onMouseDown={(event) => event.target === event.currentTarget && setOpen(false)}>
          <section className="size-modal" role="dialog" aria-modal="true" aria-label="设置图像尺寸">
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
              <div className="size-auto-state">
                <strong>自动尺寸</strong>
                <span>不向上游传递具体尺寸，由模型自行决定。</span>
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
                  <div className="size-choice-grid four">
                    {FIXED_RATIOS.map((item) => (
                      <button key={item} type="button" className={draftRatio === item ? 'active' : ''} onClick={() => setDraftRatio(item)}>
                        {item}
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
      ) : null}
    </>
  )
}

function resolutionTitle(value: string) {
  if (value === 'standard') return '标准 / 1K'
  return getResolutionLabel(value)
}
