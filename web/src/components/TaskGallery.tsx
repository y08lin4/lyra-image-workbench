import type { Task, TaskStatus } from '../types'

type TaskFilter = TaskStatus | 'all'

type Props = {
  tasks: Task[]
  activeId?: string
  query: string
  statusFilter: TaskFilter
  favoriteOnly: boolean
  favoriteIds: Set<string>
  onQueryChange: (value: string) => void
  onStatusFilterChange: (value: TaskFilter) => void
  onFavoriteOnlyChange: (value: boolean) => void
  onSelect: (task: Task) => void
  onRetry: (id: string) => void
  onCancel: (id: string) => void
  onReuse: (task: Task) => void
  onToggleFavorite: (id: string) => void
}

const statusOptions: Array<{ value: TaskFilter; label: string }> = [
  { value: 'all', label: '全部状态' },
  { value: 'queued', label: '排队中' },
  { value: 'running', label: '运行中' },
  { value: 'succeeded', label: '已成功' },
  { value: 'partial_failed', label: '部分成功' },
  { value: 'failed', label: '已失败' },
  { value: 'cancelled', label: '已取消' },
  { value: 'interrupted', label: '已中断' },
]

export function TaskGallery({
  tasks,
  activeId,
  query,
  statusFilter,
  favoriteOnly,
  favoriteIds,
  onQueryChange,
  onStatusFilterChange,
  onFavoriteOnlyChange,
  onSelect,
  onRetry,
  onCancel,
  onReuse,
  onToggleFavorite,
}: Props) {
  const filteredTasks = filterTasks(tasks, query, statusFilter, favoriteOnly, favoriteIds)
  const runningCount = tasks.filter((task) => !isFinal(task)).length

  return (
    <section className="gallery-area" aria-label="任务画廊">
      <div className="gallery-toolbar">
        <button
          type="button"
          className={`icon-filter ${favoriteOnly ? 'active' : ''}`}
          onClick={() => onFavoriteOnlyChange(!favoriteOnly)}
          aria-label={favoriteOnly ? '显示全部' : '只看收藏'}
          title={favoriteOnly ? '显示全部' : '只看收藏'}
        >
          ★
        </button>
        <select value={statusFilter} onChange={(event) => onStatusFilterChange(event.target.value as TaskFilter)} aria-label="状态筛选">
          {statusOptions.map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}
        </select>
        <label className="gallery-search">
          <span>⌕</span>
          <input value={query} onChange={(event) => onQueryChange(event.target.value)} placeholder="搜索提示词、参数、状态码..." />
        </label>
      </div>

      <div className="gallery-summary">
        <strong>{filteredTasks.length ? `${filteredTasks.length} 个任务` : '暂无匹配任务'}</strong>
        <span>{runningCount ? `${runningCount} 个任务仍在后台执行` : '后台空闲，结果会保存在本机空间'}</span>
      </div>

      {!filteredTasks.length ? (
        <div className="gallery-empty">
          <strong>{tasks.length ? '没有找到匹配的任务' : '还没有生成记录'}</strong>
          <span>{tasks.length ? '换个关键词或筛选条件试试。' : '在底部输入提示词并提交，结果会显示为画廊卡片。'}</span>
        </div>
      ) : (
        <div className="task-gallery-grid">
          {filteredTasks.map((task) => (
            <TaskGalleryCard
              key={task.id}
              task={task}
              active={task.id === activeId}
              favorite={favoriteIds.has(task.id)}
              onSelect={() => onSelect(task)}
              onRetry={() => onRetry(task.id)}
              onCancel={() => onCancel(task.id)}
              onReuse={() => onReuse(task)}
              onToggleFavorite={() => onToggleFavorite(task.id)}
            />
          ))}
        </div>
      )}
    </section>
  )
}

function TaskGalleryCard({ task, active, favorite, onSelect, onRetry, onCancel, onReuse, onToggleFavorite }: {
  task: Task
  active: boolean
  favorite: boolean
  onSelect: () => void
  onRetry: () => void
  onCancel: () => void
  onReuse: () => void
  onToggleFavorite: () => void
}) {
  const cover = firstImage(task)
  const okCount = task.results.filter((result) => result.ok).length
  const hasError = task.results.some((result) => !result.ok)

  return (
    <article className={`gallery-card ${active ? 'active' : ''} ${!cover ? 'no-cover' : ''}`} onClick={onSelect}>
      <div className="gallery-cover">
        {cover ? <img src={cover.remoteThumbUrl || cover.imageUrl} alt={`任务 ${task.id} 的结果缩略图`} /> : <LoadingCover task={task} />}
        <div className="cover-badges">
          <span>{task.ratio === 'auto' ? '自动比例' : task.ratio}</span>
          <span>{task.size && task.size !== '自动' ? task.size : '自动尺寸'}</span>
        </div>
      </div>
      <div className="gallery-card-body">
        <div className="card-title-line">
          <h3>{task.prompt || '未填写提示词'}</h3>
          <button
            type="button"
            className={`favorite-btn ${favorite ? 'active' : ''}`}
            onClick={(event) => { event.stopPropagation(); onToggleFavorite() }}
            title={favorite ? '取消收藏' : '收藏'}
            aria-label={favorite ? '取消收藏' : '收藏'}
          >
            ★
          </button>
        </div>
        <div className="gallery-tags">
          <span>{task.mode === 'image-to-image' ? '图生图' : '文生图'}</span>
          <span>质量 {qualityLabel(task.quality)}</span>
          <span>{task.statusText} / {task.statusCode}</span>
          {hasError ? <span className="warn">含失败</span> : null}
        </div>
        <progress value={task.progress} max={100} />
        <div className="gallery-card-meta">
          <span>{okCount}/{task.count || 1}</span>
          <span>{task.concurrency || 1} 并发</span>
          <span>{formatElapsed(task)}</span>
        </div>
        <div className="gallery-card-actions">
          <button type="button" onClick={(event) => { event.stopPropagation(); onReuse() }}>复用</button>
          {isFinal(task)
            ? <button type="button" onClick={(event) => { event.stopPropagation(); onRetry() }}>重试</button>
            : <button type="button" onClick={(event) => { event.stopPropagation(); onCancel() }}>取消</button>}
          <button type="button" onClick={(event) => { event.stopPropagation(); onSelect() }}>详情</button>
        </div>
      </div>
    </article>
  )
}

function LoadingCover({ task }: { task: Task }) {
  if (task.status === 'failed' || task.status === 'cancelled' || task.status === 'interrupted') {
    return (
      <div className="gallery-placeholder error">
        <strong>{task.statusText}</strong>
        <span>{task.error || task.stageText}</span>
      </div>
    )
  }
  return (
    <div className="gallery-placeholder">
      <div className="spinner" />
      <span>{task.stageText} · {task.progress}%</span>
    </div>
  )
}

function filterTasks(tasks: Task[], query: string, statusFilter: TaskFilter, favoriteOnly: boolean, favoriteIds: Set<string>) {
  const q = query.trim().toLowerCase()
  return tasks.filter((task) => {
    if (favoriteOnly && !favoriteIds.has(task.id)) return false
    if (statusFilter !== 'all' && task.status !== statusFilter) return false
    if (!q) return true
    return [
      task.prompt,
      task.id,
      task.status,
      task.statusText,
      task.statusCode,
      task.stageText,
      task.stageCode,
      task.ratio,
      task.resolution,
      task.quality,
      task.size,
    ].join(' ').toLowerCase().includes(q)
  })
}

function firstImage(task: Task) {
  return task.results.find((result) => result.ok && (result.remoteThumbUrl || result.imageUrl))
}

function isFinal(task: Task) {
  return ['succeeded', 'partial_failed', 'failed', 'cancelled', 'interrupted'].includes(task.status)
}

function qualityLabel(value?: string) {
  const labels: Record<string, string> = { auto: '自动', low: '低', medium: '中', high: '高' }
  return value ? labels[value] || value : '自动'
}

function formatElapsed(task: Task) {
  const start = Date.parse(task.startedAt || task.createdAt)
  const end = task.finishedAt ? Date.parse(task.finishedAt) : Date.now()
  if (!Number.isFinite(start) || !Number.isFinite(end) || end <= start) return '刚刚'
  const seconds = Math.floor((end - start) / 1000)
  const mm = String(Math.floor(seconds / 60)).padStart(2, '0')
  const ss = String(seconds % 60).padStart(2, '0')
  return `${mm}:${ss}`
}
