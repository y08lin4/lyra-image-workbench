import { useEffect, useRef, useState } from 'react'
import { THEME_OPTIONS, type ThemeMode } from '../lib/themes'

export type { ThemeMode }

export function ThemeToggle({ theme, onToggle }: { theme: ThemeMode; onToggle: (theme?: ThemeMode) => void }) {
  const [open, setOpen] = useState(false)
  const rootRef = useRef<HTMLDivElement | null>(null)
  const active = THEME_OPTIONS.find((item) => item.key === theme) || THEME_OPTIONS[0]

  useEffect(() => {
    if (!open) return
    function handlePointerDown(event: PointerEvent) {
      if (!rootRef.current?.contains(event.target as Node)) setOpen(false)
    }
    function handleKeyDown(event: KeyboardEvent) {
      if (event.key === 'Escape') setOpen(false)
    }
    document.addEventListener('pointerdown', handlePointerDown)
    document.addEventListener('keydown', handleKeyDown)
    return () => {
      document.removeEventListener('pointerdown', handlePointerDown)
      document.removeEventListener('keydown', handleKeyDown)
    }
  }, [open])

  return (
    <div ref={rootRef} className="theme-toggle" title="选择界面主题">
      <span>主题</span>
      <button type="button" className="theme-toggle-trigger" aria-haspopup="menu" aria-expanded={open} onClick={() => setOpen((value) => !value)}>
        {active.label}
      </button>
      {open ? (
        <div className="theme-toggle-menu" role="menu">
          {THEME_OPTIONS.map((item) => (
            <button
              key={item.key}
              type="button"
              role="menuitemradio"
              aria-checked={item.key === theme}
              className={item.key === theme ? 'active' : ''}
              onClick={() => {
                onToggle(item.key)
                setOpen(false)
              }}
            >
              {item.label}
            </button>
          ))}
        </div>
      ) : null}
    </div>
  )
}