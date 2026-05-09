import { getAvailableRatios, getRatioPreviewStyle, normalizeRatioForResolution } from '../lib/ratios'

export function RatioPicker({ value, resolution, onChange }: { value: string; resolution: string; onChange: (ratio: string) => void }) {
  const ratios = getAvailableRatios(resolution)
  const normalized = normalizeRatioForResolution(value, resolution)
  return (
    <div className="ratio-list" role="radiogroup" aria-label="图片比例">
      {ratios.map((ratio) => (
        <button
          key={ratio}
          type="button"
          className={`ratio-btn ${ratio === normalized ? 'active' : ''}`}
          onClick={() => onChange(ratio)}
          aria-checked={ratio === normalized}
          role="radio"
        >
          <span className="ratio-icon" style={getRatioPreviewStyle(ratio)} />
          <span>{ratio === 'auto' ? '自动' : ratio}</span>
        </button>
      ))}
    </div>
  )
}
