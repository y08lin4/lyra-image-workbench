import type { Task } from '../types'
import { ResultCanvas } from './ResultCanvas'

type Props = {
  task: Task
  favorite: boolean
  onClose: () => void
  onRetry: (id: string) => void
  onCancel: (id: string) => void
  onDelete: (id: string) => void
  onReuse: (task: Task) => void
  onToggleFavorite: (id: string) => void
  onUseAsReference?: (src: string, index: number) => Promise<void>
  onUploadPixhost?: (taskId: string, index: number) => Promise<void>
}

export function TaskDetailModal({
  task,
  favorite,
  onClose,
  onRetry,
  onCancel,
  onDelete,
  onReuse,
  onToggleFavorite,
  onUseAsReference,
  onUploadPixhost,
}: Props) {
  return (
    <div className="task-detail-mask" onMouseDown={(event) => event.target === event.currentTarget && onClose()}>
      <section className="task-detail-dialog" role="dialog" aria-modal="true" aria-label="任务详情">
        <header className="task-detail-header">
          <div>
            <p className="eyebrow">任务详情</p>
            <h2>{task.mode === 'image-to-image' ? '图生图结果' : '文生图结果'}</h2>
            <span>{task.statusText} / {task.statusCode} · {task.stageText} / {task.stageCode} · 来源 {taskSourceLabel(task.source)} · ID {task.id}</span>
          </div>
          <div className="detail-actions">
            <button type="button" onClick={() => onReuse(task)}>复用配置</button>
            <button type="button" onClick={() => onToggleFavorite(task.id)}>{favorite ? '取消收藏' : '收藏'}</button>
            {isFinal(task)
              ? <button type="button" onClick={() => onRetry(task.id)}>重试</button>
              : <button type="button" onClick={() => onCancel(task.id)}>取消</button>}
            <button type="button" className="danger-text" onClick={() => onDelete(task.id)}>删除</button>
            <button type="button" className="detail-close" onClick={onClose} aria-label="关闭详情">×</button>
          </div>
        </header>
        <div className="task-detail-content">
          <ResultCanvas task={task} onUseAsReference={onUseAsReference} onUploadPixhost={onUploadPixhost} />
        </div>
      </section>
    </div>
  )
}

function isFinal(task: Task) {
  return ['succeeded', 'partial_failed', 'failed', 'cancelled', 'interrupted'].includes(task.status)
}

function taskSourceLabel(source?: string) {
  return source === 'api' ? 'API' : 'Web'
}
