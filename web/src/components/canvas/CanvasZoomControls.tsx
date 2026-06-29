import { useId } from 'react'
import './CanvasComponents.css'

export type CanvasZoomControlsProps = {
  zoom: number
  minZoom?: number
  maxZoom?: number
  step?: number
  disabled?: boolean
  className?: string
  ariaLabel?: string
  onZoomChange?: (zoom: number) => void
  onFitToView?: () => void
  onReset?: () => void
}

function clampZoom(value: number, minZoom: number, maxZoom: number) {
  return Math.min(maxZoom, Math.max(minZoom, value))
}

export function CanvasZoomControls({
  zoom,
  minZoom = 25,
  maxZoom = 400,
  step = 10,
  disabled = false,
  className,
  ariaLabel = '画布缩放',
  onZoomChange,
  onFitToView,
  onReset,
}: CanvasZoomControlsProps) {
  const rangeId = useId()
  const safeZoom = clampZoom(Math.round(zoom), minZoom, maxZoom)
  const rootClassName = ['canvas-zoom-controls', className].filter(Boolean).join(' ')
  const updateZoom = (nextZoom: number) => onZoomChange?.(clampZoom(nextZoom, minZoom, maxZoom))
  const canChangeZoom = !disabled && Boolean(onZoomChange)
  const handleReset = () => {
    if (onReset) {
      onReset()
      return
    }
    updateZoom(100)
  }

  return (
    <div className={rootClassName} role="group" aria-label={ariaLabel}>
      <button
        type="button"
        className="canvas-zoom-button"
        aria-label="缩小"
        title="缩小"
        disabled={!canChangeZoom || safeZoom <= minZoom}
        onClick={() => updateZoom(safeZoom - step)}
      >
        -
      </button>
      <label className="canvas-sr-only" htmlFor={rangeId}>缩放比例</label>
      <input
        id={rangeId}
        className="canvas-zoom-slider"
        type="range"
        min={minZoom}
        max={maxZoom}
        step={step}
        value={safeZoom}
        disabled={!canChangeZoom}
        aria-valuetext={`${safeZoom}%`}
        onChange={(event) => updateZoom(Number(event.currentTarget.value))}
      />
      <span className="canvas-zoom-value" aria-live="polite">{safeZoom}%</span>
      <button
        type="button"
        className="canvas-zoom-button"
        aria-label="放大"
        title="放大"
        disabled={!canChangeZoom || safeZoom >= maxZoom}
        onClick={() => updateZoom(safeZoom + step)}
      >
        +
      </button>
      <button type="button" className="canvas-zoom-button" aria-label="重置缩放" title="重置缩放" disabled={disabled || (!onReset && !onZoomChange)} onClick={handleReset}>
        1:1
      </button>
      <button type="button" className="canvas-zoom-button" aria-label="适应画布" title="适应画布" disabled={disabled || !onFitToView} onClick={onFitToView}>
        []
      </button>
    </div>
  )
}
