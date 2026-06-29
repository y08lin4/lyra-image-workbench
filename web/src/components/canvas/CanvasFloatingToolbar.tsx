import type { ReactNode } from 'react'
import './CanvasComponents.css'

export type CanvasToolbarActionTone = 'default' | 'primary' | 'danger'

export type CanvasToolbarAction = {
  id: string
  label: string
  icon?: ReactNode
  active?: boolean
  disabled?: boolean
  tone?: CanvasToolbarActionTone
  onClick?: () => void
}

export type CanvasFloatingToolbarProps = {
  actions?: CanvasToolbarAction[]
  align?: 'start' | 'center' | 'end'
  orientation?: 'horizontal' | 'vertical'
  ariaLabel?: string
  className?: string
  children?: ReactNode
}

const DEFAULT_ACTIONS: CanvasToolbarAction[] = [
  { id: 'add-image', label: '添加图片', icon: '+' },
  { id: 'add-text', label: '添加文字', icon: 'T' },
  { id: 'connect', label: '连接节点', icon: '~' },
  { id: 'generate', label: '生成', icon: '>', tone: 'primary' },
]

export function CanvasFloatingToolbar({
  actions = DEFAULT_ACTIONS,
  align = 'center',
  orientation = 'horizontal',
  ariaLabel = '画布工具',
  className,
  children,
}: CanvasFloatingToolbarProps) {
  const rootClassName = ['canvas-floating-toolbar', className].filter(Boolean).join(' ')

  return (
    <div className={rootClassName} data-align={align} data-orientation={orientation} role="toolbar" aria-label={ariaLabel}>
      {actions.map((action) => (
        <button
          key={action.id}
          type="button"
          className={`canvas-toolbar-action ${action.active ? 'is-active' : ''}`}
          data-tone={action.tone || 'default'}
          aria-label={action.label}
          aria-pressed={action.active || undefined}
          title={action.label}
          disabled={action.disabled}
          onClick={action.onClick}
        >
          <span className="canvas-toolbar-action-glyph" aria-hidden="true">{action.icon || action.label.slice(0, 1)}</span>
          <span className="canvas-sr-only">{action.label}</span>
        </button>
      ))}
      {children}
    </div>
  )
}
