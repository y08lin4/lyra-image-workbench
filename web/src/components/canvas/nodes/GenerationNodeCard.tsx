import type { KeyboardEvent, ReactNode } from 'react'
import '../CanvasComponents.css'

export type GenerationNodeStatus = 'idle' | 'ready' | 'running' | 'blocked' | 'error'

export type GenerationNodeAction = {
  id: string
  label: string
  disabled?: boolean
  tone?: 'default' | 'primary' | 'danger'
  onClick?: () => void
}

export type GenerationNodeCardProps = {
  title?: string
  subtitle?: string
  prompt?: string
  status?: GenerationNodeStatus
  modelLabel?: string
  ratioLabel?: string
  referenceCount?: number
  selected?: boolean
  dimmed?: boolean
  className?: string
  actions?: GenerationNodeAction[]
  children?: ReactNode
  onSelect?: () => void
}

const STATUS_LABELS: Record<GenerationNodeStatus, string> = {
  idle: '待配置',
  ready: '可生成',
  running: '生成中',
  blocked: '需补全',
  error: '异常',
}

export function GenerationNodeCard({
  title = '生成节点',
  subtitle = 'Image generation',
  prompt,
  status = 'idle',
  modelLabel = 'Model',
  ratioLabel = '1:1',
  referenceCount = 0,
  selected = false,
  dimmed = false,
  className,
  actions = [],
  children,
  onSelect,
}: GenerationNodeCardProps) {
  const rootClassName = [
    'canvas-node-card',
    'canvas-generation-node-card',
    onSelect ? 'is-interactive' : '',
    selected ? 'is-selected' : '',
    dimmed ? 'is-dimmed' : '',
    className,
  ].filter(Boolean).join(' ')

  const handleKeyDown = (event: KeyboardEvent<HTMLElement>) => {
    if (!onSelect || (event.key !== 'Enter' && event.key !== ' ')) return
    event.preventDefault()
    onSelect()
  }

  return (
    <article
      className={rootClassName}
      tabIndex={onSelect ? 0 : undefined}
      aria-label={`${title}，${STATUS_LABELS[status]}`}
      onClick={onSelect}
      onKeyDown={handleKeyDown}
    >
      <header className="canvas-node-card-header">
        <div className="canvas-node-title">
          <strong>{title}</strong>
          <span>{subtitle}</span>
        </div>
        <span className="canvas-node-status" data-status={status}>{STATUS_LABELS[status]}</span>
      </header>

      <div className="canvas-node-prompt">
        {prompt ? prompt : <span className="canvas-node-empty">暂无提示词</span>}
      </div>

      <div className="canvas-node-meta" aria-label="生成参数">
        <span title={modelLabel}>{modelLabel}</span>
        <span title={ratioLabel}>{ratioLabel}</span>
        <span>{referenceCount} 张参考</span>
      </div>

      {children}

      {actions.length ? (
        <div className="canvas-node-actions">
          {actions.map((action) => (
            <button
              key={action.id}
              type="button"
              className="canvas-node-action"
              data-tone={action.tone || 'default'}
              disabled={action.disabled}
              onClick={(event) => {
                event.stopPropagation()
                action.onClick?.()
              }}
            >
              {action.label}
            </button>
          ))}
        </div>
      ) : null}
    </article>
  )
}
