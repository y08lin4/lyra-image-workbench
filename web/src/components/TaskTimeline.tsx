import type { Task } from '../types'

export function TaskTimeline({ tasks, activeId, onSelect, onRetry, onCancel }: { tasks: Task[]; activeId?: string; onSelect: (id: string) => void; onRetry: (id: string) => void; onCancel: (id: string) => void }) {
  return (
    <aside className="task-timeline">
      <header className="timeline-header">
        <p className="eyebrow">Tasks</p>
        <h2>任务</h2>
        <span>{tasks.length ? `${tasks.length} 个任务` : '暂无任务'}</span>
      </header>
      {!tasks.length ? <p className="muted">生成任务会按时间显示在这里。</p> : null}
      <div className="task-stack">
        {tasks.map((task) => (
          <article className={`timeline-item ${task.id === activeId ? 'active' : ''}`} key={task.id} onClick={() => onSelect(task.id)}>
            <div className="timeline-status">
              <strong>{task.statusText} / {task.status} / {task.statusCode}</strong>
              <span>{task.stageText} / {task.stage} / {task.stageCode}</span>
            </div>
            <progress value={task.progress} max={100} />
            <small>{task.mode} · {task.size} · {task.results.filter((result) => result.ok).length}/{task.count}</small>
            <div className="task-actions">
              {isFinal(task) ? <button type="button" onClick={(event) => { event.stopPropagation(); onRetry(task.id) }}>重试</button> : <button type="button" onClick={(event) => { event.stopPropagation(); onCancel(task.id) }}>取消</button>}
            </div>
          </article>
        ))}
      </div>
    </aside>
  )
}

function isFinal(task: Task) {
  return ['succeeded', 'partial_failed', 'failed', 'cancelled', 'interrupted'].includes(task.status)
}
