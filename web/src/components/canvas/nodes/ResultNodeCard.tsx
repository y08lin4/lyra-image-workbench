import type { KeyboardEvent, ReactNode } from 'react'
import '../CanvasComponents.css'

export type ResultNodeStatus = 'empty' | 'queued' | 'running' | 'succeeded' | 'failed'

export type ResultNodeAction = {
  id: string
  label: string
  disabled?: boolean
  tone?: 'default' | 'primary' | 'danger'
  onClick?: () => void
}

export type ResultNodeCardProps = {
  title?: string
  subtitle?: string
  imageSrc?: string
  imageAlt?: string
  status?: ResultNodeStatus
  progress?: number
  selected?: boolean
  dimmed?: boolean
  aspectRatio?: string
  className?: string
  actions?: ResultNodeAction[]
  children?: ReactNode
  onSelect?: () => void
}

const STATUS_LABELS: Record<ResultNodeStatus, string> = {
  empty: '待生成',
  queued: '排队中',
  running: '生成中',
  succeeded: '已完成',
  failed: '失败',
}

function clampProgress(progress?: number) {
  if (typeof progress !== 'number' || Number.isNaN(progress)) return undefined
  return Math.min(100, Math.max(0, progress))
}

export function ResultNodeCard({
  title = '结果节点',
  subtitle = 'Generated result',
  imageSrc,
  imageAlt,
  status = imageSrc ? 'succeeded' : 'empty',
  progress,
  selected = false,
  dimmed = false,
  aspectRatio = '1 / 1',
  className,
  actions = [],
  children,
  onSelect,
}: ResultNodeCardProps) {
  const safeProgress = clampProgress(progress)
  const rootClassName = [
    'canvas-node-card',
    'canvas-result-node-card',
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

      <div className="canvas-result-frame" style={{ aspectRatio }}>
        {imageSrc ? (
          <img src={imageSrc} alt={imageAlt || title} draggable={false} />
        ) : (
          <div className="canvas-result-placeholder">
            <strong>暂无结果</strong>
            <span>{STATUS_LABELS[status]}</span>
          </div>
        )}
      </div>

      {safeProgress !== undefined ? (
        <div className="canvas-result-progress" role="progressbar" aria-valuemin={0} aria-valuemax={100} aria-valuenow={safeProgress}>
          <span style={{ width: `${safeProgress}%` }} />
        </div>
      ) : null}

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
