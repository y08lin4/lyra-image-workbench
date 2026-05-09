import { getQualityLabel, QUALITY_LEVELS } from '../lib/ratios'

export function QualityPicker({ value, onChange }: { value: string; onChange: (quality: string) => void }) {
  return (
    <div className="quality-list" role="radiogroup" aria-label="质量档位">
      {QUALITY_LEVELS.map((quality) => (
        <button
          key={quality}
          type="button"
          className={`quality-btn ${quality === value ? 'active' : ''}`}
          onClick={() => onChange(quality)}
          aria-checked={quality === value}
          role="radio"
        >
          {getQualityLabel(quality)}
        </button>
      ))}
    </div>
  )
}
