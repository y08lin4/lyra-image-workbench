export type ThemeMode = 'light' | 'dark'

export function ThemeToggle({ theme, onToggle }: { theme: ThemeMode; onToggle: () => void }) {
  const isDark = theme === 'dark'
  return (
    <button type="button" className="theme-toggle" onClick={onToggle} aria-pressed={isDark} title={isDark ? '切换到白天模式' : '切换到黑夜模式'}>
      <span aria-hidden="true">{isDark ? '☀' : '☾'}</span>
      <strong>{isDark ? '白天模式' : '黑夜模式'}</strong>
    </button>
  )
}
