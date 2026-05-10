import { useMemo, useState } from 'react'
import { createPortal } from 'react-dom'
import { BANANA_MODEL_OPTIONS, getBananaModelOption } from '../lib/models'

type Props = {
  value: string
  onChange: (model: string) => void
}

export function BananaModelPicker({ value, onChange }: Props) {
  const [open, setOpen] = useState(false)
  const current = useMemo(() => getBananaModelOption(value), [value])

  function choose(model: string) {
    onChange(model)
    setOpen(false)
  }

  const modal = open ? (
    <div className="size-modal-mask" onMouseDown={(event) => event.target === event.currentTarget && setOpen(false)}>
      <section className="size-modal banana-model-modal" role="dialog" aria-modal="true" aria-label="选择 Banana 模型规格">
        <header>
          <div>
            <h3>Banana 模型规格</h3>
            <p>比例和清晰度不作为参数发送，而是路由到对应模型 ID。</p>
          </div>
          <button type="button" onClick={() => setOpen(false)} aria-label="关闭 Banana 模型规格">×</button>
        </header>

        <div className="banana-model-grid">
          {BANANA_MODEL_OPTIONS.map((option) => (
            <button key={option.id} type="button" className={option.id === current.id ? 'active' : ''} onClick={() => choose(option.id)}>
              <span className="banana-ratio-preview" style={ratioPreviewStyle(option.ratio)}>
                <i>{option.ratio === 'auto' ? 'AUTO' : ''}</i>
              </span>
              <span className="banana-model-text">
                <strong>{option.label}</strong>
                <span>{option.hint}</span>
                <small title={option.id}>{compactModelID(option.id)}</small>
              </span>
            </button>
          ))}
        </div>
      </section>
    </div>
  ) : null

  return (
    <>
      <button type="button" className="size-selector-button banana-model-button" onClick={() => setOpen(true)}>
        <span>
          <strong>Banana 规格</strong>
          <small>{current.label} · {current.size}</small>
        </span>
        <b>模型</b>
      </button>
      {modal ? createPortal(modal, document.body) : null}
    </>
  )
}

function compactModelID(id: string) {
  return id.replace('gemini-3.1-flash-image-preview', 'gemini-preview')
}

function ratioPreviewStyle(ratio: string) {
  if (ratio === 'auto') return { aspectRatio: '1 / 1' }
  const [w, h] = ratio.split(':').map(Number)
  return { aspectRatio: `${w} / ${h}` }
}
