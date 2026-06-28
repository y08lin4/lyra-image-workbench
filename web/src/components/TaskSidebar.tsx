import { useMemo, useState } from 'react'
import type { Task } from '../types'
import { BANANA_PROVIDER, getBananaModelOption, providerLabel } from '../lib/models'

type SidebarFilter = 'all' | 'active' | 'succeeded' | 'failed' | 'api' | 'favorite'

type Props = {
  tasks: Task[]
  activeId?: string
  query: string
  favoriteIds: Set<string>
  selectedIds: Set<string>
  onQueryChange: (value: string) => void
  onToggleSelect: (id: string) => void
  onSelectVisible: (ids: string[]) => void
  onClearSelection: () => void
  onBatchFavorite: (favorite: boolean) => void
  onBatchDelete: () => void
  onBatchDownload: () => void
  onSelect: (task: Task) => void
  onOpenDetail: (task: Task) => void
  onRetry: (id: string) => void
  onCancel: (id: string) => void
  onDelete: (id: string) => void
  onReuse: (task: Task) => void
  onToggleFavorite: (id: string) => void
}

const filterOptions: Array<{ value: SidebarFilter; label: string }> = [
  { value: 'all', label: '全部' },
  { value: 'active', label: '进行中' },
  { value: 'succeeded', label: '成功' },
  { value: 'failed', label: '失败' },
  { value: 'api', label: 'API' },
  { value: 'favorite', label: '收藏' },
]

export function TaskSidebar({
  tasks,
  activeId,
  query,
  favoriteIds,
  selectedIds,
  onQueryChange,
  onToggleSelect,
  onSelectVisible,
  onClearSelection,
  onBatchFavorite,
  onBatchDelete,
  onBatchDownload,
  onSelect,
  onOpenDetail,
  onRetry,
  onCancel,
  onDelete,
  onReuse,
  onToggleFavorite,
}: Props) {
  const [filter, setFilter] = useState<SidebarFilter>('all')
  const [copiedTaskId, setCopiedTaskId] = useState('')
  const filteredTasks = useMemo(() => filterTasks(tasks, query, filter, favoriteIds), [tasks, query, filter, favoriteIds])
  const apiTasks = useMemo(() => tasks.filter((task) => task.source === 'api'), [tasks])
  const latestApiTask = apiTasks[0]
  const stats = useMemo(() => ({
    total: tasks.length,
    active: tasks.filter((task) => !isFinal(task)).length,
    succeeded: tasks.filter((task) => task.status === 'succeeded').length,
    failed: tasks.filter((task) => ['failed', 'partial_failed', 'cancelled', 'interrupted'].includes(task.status)).length,
    api: tasks.filter((task) => task.source === 'api').length,
  }), [tasks])
  const apiStats = useMemo(() => ({
    total: apiTasks.length,
    active: apiTasks.filter((task) => !isFinal(task)).length,
    succeeded: apiTasks.filter((task) => task.status === 'succeeded' || task.status === 'partial_failed').length,
  }), [apiTasks])

  async function copyTaskId(id: string) {
    try {
      await copyToClipboard(id)
      setCopiedTaskId(id)
      window.setTimeout(() => setCopiedTaskId((current) => current === id ? '' : current), 1800)
    } catch {
      setCopiedTaskId('')
    }
  }

  return (
    <section className={`queue-sidebar ${selectedIds.size ? 'has-selection' : ''}`} aria-label="任务队列">
      <header className="queue-header">
        <div>
          <p className="eyebrow">Queue</p>
          <h2>任务队列</h2>
        </div>
        <span>{stats.active ? `${stats.active} 个执行中` : '后台空闲'}</span>
      </header>

      <div className="queue-stat-grid">
        <div><strong>{stats.total}</strong><span>全部</span></div>
        <div><strong>{stats.active}</strong><span>进行中</span></div>
        <div><strong>{stats.succeeded}</strong><span>成功</span></div>
        <div><strong>{stats.failed}</strong><span>异常</span></div>
      </div>

      {latestApiTask ? (
        <div className="api-task-sync-card" aria-label="API 请求同步状态">
          <div className="api-task-sync-main">
            <span className="queue-source-badge">API</span>
            <div>
              <strong>API 请求同步</strong>
              <p>{apiStats.active ? `${apiStats.active} 个 API 任务执行中` : 'API 任务已同步到生成历史'}</p>
            </div>
          </div>
          <div className="api-task-sync-meta">
            <span className={`status-pill ${latestApiTask.status}`}>{latestApiTask.statusText} / {latestApiTask.statusCode}</span>
            <code title={latestApiTask.id}>ID {latestApiTask.id}</code>
            <small>历史 {apiStats.total} · 成功/部分成功 {apiStats.succeeded}</small>
          </div>
          <div className="api-task-sync-actions">
            <button type="button" onClick={() => onSelect(latestApiTask)}>查看最新</button>
            <button type="button" onClick={() => void copyTaskId(latestApiTask.id)}>{copiedTaskId === latestApiTask.id ? '已复制 ID' : '复制 ID'}</button>
          </div>
        </div>
      ) : null}

      <div className="queue-filter-row" role="tablist" aria-label="任务筛选">
        {filterOptions.map((item) => (
          <button key={item.value} type="button" className={filter === item.value ? 'active' : ''} onClick={() => setFilter(item.value)}>{item.label}</button>
        ))}
      </div>

      <label className="queue-search">
        <span>⌕</span>
        <input value={query} onChange={(event) => onQueryChange(event.target.value)} placeholder="搜索提示词、模型、错误码" />
      </label>

      {selectedIds.size ? (
        <div className="queue-batch-bar">
          <strong>已选 {selectedIds.size}</strong>
          <button type="button" onClick={() => onSelectVisible(filteredTasks.map((task) => task.id))}>选当前</button>
          <button type="button" onClick={() => onBatchFavorite(true)}>收藏</button>
          <button type="button" onClick={onBatchDownload}>下载</button>
          <button type="button" className="danger-text" onClick={onBatchDelete}>删除</button>
          <button type="button" onClick={onClearSelection}>取消</button>
        </div>
      ) : null}

      <div className="queue-list">
        {!filteredTasks.length ? (
          <div className="queue-empty">
            <strong>{tasks.length ? '没有匹配任务' : '还没有任务'}</strong>
            <span>{tasks.length ? '换个筛选或关键词试试。' : '还没有任务，去“生成”标签提交第一条请求。'}</span>
          </div>
        ) : filteredTasks.map((task) => (
          <TaskQueueItem
            key={task.id}
            task={task}
            active={task.id === activeId}
            favorite={favoriteIds.has(task.id)}
            selected={selectedIds.has(task.id)}
            onSelect={() => onSelect(task)}
            onOpenDetail={() => onOpenDetail(task)}
            onToggleSelect={() => onToggleSelect(task.id)}
            onRetry={() => onRetry(task.id)}
            onCancel={() => onCancel(task.id)}
            onDelete={() => onDelete(task.id)}
            onReuse={() => onReuse(task)}
            onToggleFavorite={() => onToggleFavorite(task.id)}
          />
        ))}
      </div>
    </section>
  )
}

function TaskQueueItem({ task, active, favorite, selected, onSelect, onOpenDetail, onToggleSelect, onRetry, onCancel, onDelete, onReuse, onToggleFavorite }: {
  task: Task
  active: boolean
  favorite: boolean
  selected: boolean
  onSelect: () => void
  onOpenDetail: () => void
  onToggleSelect: () => void
  onRetry: () => void
  onCancel: () => void
  onDelete: () => void
  onReuse: () => void
  onToggleFavorite: () => void
}) {
  const cover = firstImage(task)
  const okCount = task.results.filter((result) => result.ok).length
  const error = firstError(task)
  const modelLabel = task.provider === BANANA_PROVIDER ? getBananaModelOption(task.model || '').label : task.model || 'gpt-image-2'
  const sourceLabel = taskSourceLabel(task.source)
  const sourceIsApi = task.source === 'api'

  return (
    <article className={`queue-item ${active ? 'active' : ''} ${selected ? 'selected' : ''} ${sourceIsApi ? 'source-api' : ''}`} onClick={onSelect}>
      <div className="queue-thumb">
        {cover ? <img src={cover.remoteThumbUrl || cover.imageUrl} alt="任务缩略图" /> : <QueuePlaceholder task={task} />}
        <label className="queue-check" onClick={(event) => event.stopPropagation()}>
          <input type="checkbox" checked={selected} onChange={onToggleSelect} />
        </label>
      </div>
      <div className="queue-item-main">
        <div className="queue-status-line">
          <span className={`status-pill ${task.status}`}>{task.statusText} / {task.statusCode}</span>
          {sourceIsApi ? <span className="queue-source-badge">API</span> : null}
          <small>{okCount}/{task.count || 1} · {formatElapsed(task)}</small>
        </div>
        <div className="queue-title-line">
          <strong>{task.prompt || '未填写提示词'}</strong>
          <button type="button" className={`favorite-btn ${favorite ? 'active' : ''}`} onClick={(event) => { event.stopPropagation(); onToggleFavorite() }}>★</button>
        </div>
        <div className="queue-tags">
          <span>{task.mode === 'gif' ? 'GIF动图' : task.mode === 'image-to-image' ? '图生图' : '文生图'}</span>
          <span>{providerLabel(task.provider)}</span>
          <span>{modelLabel}</span>
          <span>来源 {sourceLabel}</span>
          <span title={task.id}>ID {compactTaskId(task.id)}</span>
        </div>
        <progress value={task.progress} max={100} />
        <div className="queue-meta">
          <span>{okCount}/{task.count || 1}</span>
          <span>{task.size && task.size !== '自动' ? task.size : task.ratio || '自动'}</span>
          <span>{formatElapsed(task)}</span>
        </div>
        {error ? <p className="queue-error">{error}</p> : null}
        <div className="queue-actions">
          <button type="button" onClick={(event) => { event.stopPropagation(); onReuse() }}>复用</button>
          {isFinal(task)
            ? <button type="button" onClick={(event) => { event.stopPropagation(); onRetry() }}>重试</button>
            : <button type="button" onClick={(event) => { event.stopPropagation(); onCancel() }}>取消</button>}
          <button type="button" onClick={(event) => { event.stopPropagation(); onOpenDetail() }}>详情</button>
          <button type="button" className="danger-text" onClick={(event) => { event.stopPropagation(); onDelete() }}>删除</button>
        </div>
      </div>
    </article>
  )
}

function QueuePlaceholder({ task }: { task: Task }) {
  if (isFinal(task)) return <span>{task.statusText}</span>
  return <div className="mini-spinner" />
}

function filterTasks(tasks: Task[], query: string, filter: SidebarFilter, favoriteIds: Set<string>) {
  const q = query.trim().toLowerCase()
  return tasks.filter((task) => {
    if (filter === 'favorite' && !favoriteIds.has(task.id)) return false
    if (filter === 'active' && isFinal(task)) return false
    if (filter === 'succeeded' && task.status !== 'succeeded') return false
    if (filter === 'failed' && !['failed', 'partial_failed', 'cancelled', 'interrupted'].includes(task.status)) return false
    if (filter === 'api' && task.source !== 'api') return false
    if (!q) return true
    return [
      task.prompt,
      task.id,
      task.statusText,
      task.statusCode,
      task.stageText,
      task.stageCode,
      task.provider,
      task.source,
      task.model,
      task.ratio,
      task.resolution,
      task.quality,
      task.outputFormat,
      task.size,
      ...task.results.flatMap((result) => [result.error, result.errorText, result.errorCode, result.errorEnglish]),
    ].join(' ').toLowerCase().includes(q)
  })
}

function firstImage(task: Task) {
  return task.results.find((result) => result.ok && (result.remoteThumbUrl || result.imageUrl))
}

function firstError(task: Task) {
  const result = task.results.find((item) => !item.ok && (item.errorText || item.errorCode || item.errorEnglish || item.error))
  if (result) return [result.errorText, result.errorCode, result.errorEnglish].filter(Boolean).join(' / ') || result.error
  return task.error || ''
}

function isFinal(task: Task) {
  return ['succeeded', 'partial_failed', 'failed', 'cancelled', 'interrupted'].includes(task.status)
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

function taskSourceLabel(source?: string) {
  return source === 'api' ? 'API' : 'Web'
}

function compactTaskId(id: string) {
  if (id.length <= 22) return id
  return `${id.slice(0, 12)}...${id.slice(-6)}`
}

async function copyToClipboard(value: string) {
  if (navigator.clipboard?.writeText) {
    await navigator.clipboard.writeText(value)
    return
  }
  const textArea = document.createElement('textarea')
  textArea.value = value
  textArea.setAttribute('readonly', '')
  textArea.style.position = 'fixed'
  textArea.style.left = '-9999px'
  textArea.style.top = '0'
  document.body.appendChild(textArea)
  try {
    textArea.focus()
    textArea.select()
    document.execCommand('copy')
  } finally {
    textArea.remove()
  }
}
