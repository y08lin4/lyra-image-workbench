import { THEME_OPTIONS, type ThemeMode } from '../lib/themes'

export type { ThemeMode }

export function ThemeToggle({ theme, onToggle }: { theme: ThemeMode; onToggle: (theme?: ThemeMode) => void }) {
  return (
    <label className="theme-toggle" title="选择界面主题">
      <span>主题</span>
      <select value={theme} aria-label="选择界面主题" onChange={(event) => onToggle(event.target.value as ThemeMode)}>
        {THEME_OPTIONS.map((item) => (
          <option key={item.key} value={item.key}>{item.label}</option>
        ))}
      </select>
    </label>
  )
}
