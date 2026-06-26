import { useEffect, useMemo, useState } from 'react'
import { likePromptSquareItem, listPromptSquareItems, type PromptSquareListOptions } from '../api/promptSquare'
import type { PromptSquareItem } from '../types'

type PromptSquarePanelProps = {
  onUsePrompt: (prompt: string, item?: PromptSquareItem) => void
}

type PromptSquareView = 'latest' | 'daily' | 'mine'

type RuntimePromptSquareItem = Omit<PromptSquareItem, 'author'> & {
  author?: PromptSquareItem['author'] | string
  authorDisplayName?: string
  authorUrl?: string
  dailyRank?: number
  sourceTaskId?: string
}

const viewOptions: Array<{ value: PromptSquareView; label: string; description: string; emptyTitle: string; emptyBody: string }> = [
  {
    value: 'latest',
    label: '最新',
    description: '按发布时间查看公开作品',
    emptyTitle: '还没有公开投稿',
    emptyBody: '生成结果投稿入口会出现在结果卡片里；后端准备好后这里会显示公开作品。',
  },
  {
    value: 'daily',
    label: '每日榜',
    description: '查看今天点赞最高的投稿',
    emptyTitle: '今日榜单暂无作品',
    emptyBody: '每日榜只统计服务器当天 00:00-24:00 的投稿。',
  },
  {
    value: 'mine',
    label: '我的投稿',
    description: '只看当前登录用户的作品',
    emptyTitle: '你还没有投稿',
    emptyBody: '从生成结果提交作品后，它们会出现在这里并永久保留。',
  },
]

export function PromptSquarePanel({ onUsePrompt }: PromptSquarePanelProps) {
  const [view, setView] = useState<PromptSquareView>('latest')
  const [items, setItems] = useState<PromptSquareItem[]>([])
  const [query, setQuery] = useState('')
  const [selectedTag, setSelectedTag] = useState('')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const [likingIds, setLikingIds] = useState<Record<string, boolean>>({})

  useEffect(() => {
    void refresh(view)
  }, [view])

  async function refresh(targetView = view) {
    setLoading(true)
    setError('')
    setNotice('')
    try {
      const nextItems = await listPromptSquareItems(listOptionsForView(targetView))
      setItems(nextItems)
    } catch (err) {
      setItems([])
      setError(`${viewLabel(targetView)}加载失败：${formatErrorMessage(err, '读取提示词广场失败')}`)
    } finally {
      setLoading(false)
    }
  }

  function switchView(nextView: PromptSquareView) {
    if (nextView === view) {
      void refresh(nextView)
      return
    }
    setQuery('')
    setSelectedTag('')
    setView(nextView)
  }

  async function copyPrompt(prompt: string) {
    setError('')
    setNotice('')
    try {
      await navigator.clipboard.writeText(prompt)
      setNotice('提示词已复制')
    } catch (err) {
      setError(formatErrorMessage(err, '复制失败'))
    }
  }

  async function toggleLike(item: PromptSquareItem) {
    const liked = !Boolean(item.likedByMe)
    const previous = item
    setError('')
    setNotice('')
    setLikingIds((prev) => ({ ...prev, [item.id]: true }))
    setItems((prev) => prev.map((entry) => entry.id === item.id ? withLocalLike(entry, liked) : entry))
    try {
      const updated = await likePromptSquareItem(item.id, liked)
      setItems((prev) => prev.map((entry) => entry.id === item.id ? updated : entry))
      setNotice(liked ? '已点赞' : '已取消点赞')
    } catch (err) {
      setItems((prev) => prev.map((entry) => entry.id === item.id ? previous : entry))
      setError(`点赞失败：${formatErrorMessage(err, '请确认后端点赞接口已完成')}`)
    } finally {
      setLikingIds((prev) => {
        const next = { ...prev }
        delete next[item.id]
        return next
      })
    }
  }

  const activeView = viewOptions.find((option) => option.value === view) || viewOptions[0]

  const tags = useMemo(() => {
    const set = new Set<string>()
    for (const item of items) {
      for (const tag of item.tags || []) set.add(tag)
    }
    return Array.from(set).slice(0, 24)
  }, [items])

  const filtered = useMemo(() => {
    const keyword = query.trim().toLowerCase()
    return items.filter((item) => {
      const runtime = asRuntimeItem(item)
      const text = [
        runtime.title,
        runtime.prompt,
        runtime.model,
        runtime.ratio,
        runtime.quality,
        runtime.outputFormat,
        runtime.params?.ratio,
        runtime.params?.quality,
        runtime.params?.outputFormat,
        authorName(runtime),
        ...(runtime.tags || []),
      ].join(' ').toLowerCase()
      const matchQuery = !keyword || text.includes(keyword)
      const matchTag = !selectedTag || runtime.tags?.includes(selectedTag)
      return matchQuery && matchTag
    })
  }, [items, query, selectedTag])

  const emptyTitle = query || selectedTag ? '没有匹配的作品' : activeView.emptyTitle
  const emptyBody = query || selectedTag ? '换个关键词或清除标签筛选后再试。' : activeView.emptyBody

  return (
    <div className="prompt-square">
      <section className="prompt-square-hero">
        <div>
          <p className="eyebrow">Prompt Square</p>
          <h3>提示词广场</h3>
          <p>{activeView.description}</p>
        </div>
        <div className="prompt-square-header-actions">
          <button type="button" onClick={() => void refresh(view)} disabled={loading}>{loading ? '刷新中' : '刷新'}</button>
        </div>
      </section>

      <section className="prompt-square-feed" aria-busy={loading}>
        <div className="prompt-square-viewbar">
          <div className="prompt-square-segments" role="tablist" aria-label="广场视图">
            {viewOptions.map((option) => (
              <button
                key={option.value}
                type="button"
                role="tab"
                aria-selected={view === option.value}
                className={view === option.value ? 'active' : ''}
                onClick={() => switchView(option.value)}
              >
                {option.label}
              </button>
            ))}
          </div>
          <span>{loading ? '加载中' : `${filtered.length} / ${items.length} 个作品`}</span>
        </div>

        <div className="prompt-square-toolbar">
          <input value={query} onChange={(event) => setQuery(event.target.value)} placeholder="搜索标题、提示词、作者、模型、标签" />
          <div className="prompt-square-tags" aria-label="标签筛选">
            <button type="button" className={!selectedTag ? 'active' : ''} onClick={() => setSelectedTag('')} aria-pressed={!selectedTag}>全部</button>
            {tags.map((tag) => (
              <button key={tag} type="button" className={selectedTag === tag ? 'active' : ''} onClick={() => setSelectedTag(tag)} aria-pressed={selectedTag === tag}>{tag}</button>
            ))}
          </div>
        </div>

        {notice ? <p className="prompt-square-alert success" role="status">{notice}</p> : null}
        {error ? <p className="prompt-square-alert error" role="alert">{error}</p> : null}

        {loading ? (
          <PromptSquareSkeleton />
        ) : error && !items.length ? (
          <PromptSquareState title={`${activeView.label}暂不可用`} body="后端广场接口可能还在接线中，请稍后刷新。" actionLabel="重试" onAction={() => void refresh(view)} />
        ) : filtered.length ? (
          <div className="prompt-square-grid">
            {filtered.map((item, index) => (
              <PromptSquareCard
                key={item.id}
                item={item}
                dailyRank={view === 'daily' ? index + 1 : 0}
                liking={Boolean(likingIds[item.id])}
                onUsePrompt={onUsePrompt}
                onCopyPrompt={copyPrompt}
                onToggleLike={toggleLike}
              />
            ))}
          </div>
        ) : (
          <PromptSquareState title={emptyTitle} body={emptyBody} />
        )}
      </section>
    </div>
  )
}

function PromptSquareCard({
  item,
  dailyRank,
  liking,
  onUsePrompt,
  onCopyPrompt,
  onToggleLike,
}: {
  item: PromptSquareItem
  dailyRank: number
  liking: boolean
  onUsePrompt: PromptSquarePanelProps['onUsePrompt']
  onCopyPrompt: (prompt: string) => Promise<void>
  onToggleLike: (item: PromptSquareItem) => Promise<void>
}) {
  const runtime = asRuntimeItem(item)
  const title = itemTitle(runtime)
  const ratio = itemParam(runtime, 'ratio')
  const quality = itemParam(runtime, 'quality')
  const outputFormat = itemParam(runtime, 'outputFormat')
  const likes = likeCount(runtime)
  const liked = Boolean(runtime.likedByMe)
  const sourceUrl = runtime.source?.url
  const imageUrl = runtime.thumbnailUrl || runtime.imageUrl
  const rank = Number(runtime.dailyRank || dailyRank || 0)

  return (
    <article className="prompt-square-card">
      <div className="prompt-square-image" style={imageAspectStyle(ratio)}>
        {rank > 0 ? <span className="prompt-square-rank">每日 #{rank}</span> : null}
        {imageUrl ? <img src={imageUrl} alt={title} loading="lazy" /> : <div className="prompt-square-placeholder">Prompt</div>}
      </div>
      <div className="prompt-square-card-body">
        <header className="prompt-square-card-head">
          <div className="prompt-square-title-block">
            <strong>{title}</strong>
            <span>{authorName(runtime)}</span>
          </div>
          <span className="prompt-square-like-count">{likes} 赞</span>
        </header>

        <p className="prompt-square-prompt">{runtime.prompt}</p>

        <dl className="prompt-square-meta-grid">
          <div><dt>模型</dt><dd>{runtime.model || '未标注'}</dd></div>
          <div><dt>比例</dt><dd>{ratio || '未标注'}</dd></div>
          <div><dt>质量</dt><dd>{quality || '未标注'}</dd></div>
          <div><dt>格式</dt><dd>{outputFormat || '未标注'}</dd></div>
        </dl>

        <div className="prompt-square-chip-row">
          {(runtime.tags || []).slice(0, 8).map((tag) => <span key={tag}>{tag}</span>)}
          {runtime.tags?.length ? null : <span>无标签</span>}
        </div>

        <footer className="prompt-square-card-footer">
          <span className="prompt-square-time">{formatDate(runtime.submittedAt || runtime.createdAt)}</span>
          <div className="prompt-square-actions">
            {sourceUrl ? <a href={sourceUrl} target="_blank" rel="noreferrer">来源</a> : null}
            <button type="button" onClick={() => void onCopyPrompt(runtime.prompt)}>复制</button>
            <button type="button" className="prompt-square-use" onClick={() => onUsePrompt(runtime.prompt, item)}>使用</button>
            <button
              type="button"
              className={liked ? 'prompt-square-like active' : 'prompt-square-like'}
              onClick={() => void onToggleLike(item)}
              disabled={liking}
              aria-pressed={liked}
            >
              {liking ? '处理中' : liked ? '已赞' : '点赞'}
            </button>
          </div>
        </footer>
      </div>
    </article>
  )
}

function PromptSquareSkeleton() {
  return (
    <div className="prompt-square-grid" aria-hidden="true">
      {Array.from({ length: 6 }, (_, index) => (
        <article className="prompt-square-card prompt-square-card-skeleton" key={index}>
          <div className="prompt-square-image" />
          <div className="prompt-square-card-body">
            <span />
            <p />
            <p />
            <div />
          </div>
        </article>
      ))}
    </div>
  )
}

function PromptSquareState({ title, body, actionLabel, onAction }: { title: string; body: string; actionLabel?: string; onAction?: () => void }) {
  return (
    <div className="prompt-square-state">
      <strong>{title}</strong>
      <p>{body}</p>
      {actionLabel && onAction ? <button type="button" onClick={onAction}>{actionLabel}</button> : null}
    </div>
  )
}

function listOptionsForView(view: PromptSquareView): PromptSquareListOptions {
  if (view === 'daily') return { daily: true, sort: 'daily' }
  if (view === 'mine') return { mine: true }
  return { sort: 'latest' }
}

function asRuntimeItem(item: PromptSquareItem) {
  return item as RuntimePromptSquareItem
}

function itemTitle(item: RuntimePromptSquareItem) {
  const title = item.title?.trim()
  if (title) return title
  const prompt = item.prompt?.trim() || ''
  return prompt ? prompt.slice(0, 48) : '未命名作品'
}

function authorName(item: RuntimePromptSquareItem) {
  if (item.authorDisplayName?.trim()) return item.authorDisplayName.trim()
  if (typeof item.author === 'string') return item.author.trim() || '匿名用户'
  return item.author?.name?.trim() || item.authorUsername || '匿名用户'
}

function itemParam(item: RuntimePromptSquareItem, key: 'ratio' | 'quality' | 'outputFormat') {
  return (item[key] || item.params?.[key] || '').trim()
}

function likeCount(item: RuntimePromptSquareItem) {
  return Math.max(0, Number(item.likeCount ?? item.likes ?? 0) || 0)
}

function withLocalLike(item: PromptSquareItem, liked: boolean): PromptSquareItem {
  const current = likeCount(asRuntimeItem(item))
  const wasLiked = Boolean(item.likedByMe)
  const delta = liked === wasLiked ? 0 : liked ? 1 : -1
  const nextCount = Math.max(0, current + delta)
  return { ...item, likedByMe: liked, likeCount: nextCount, likes: nextCount }
}

function imageAspectStyle(ratio: string) {
  const aspectRatio = ratioToCssAspect(ratio)
  return aspectRatio ? { aspectRatio } : undefined
}

function ratioToCssAspect(value: string) {
  const match = value.trim().match(/^(\d+(?:\.\d+)?)\s*[:/x×]\s*(\d+(?:\.\d+)?)$/i)
  if (!match) return ''
  const width = Number(match[1])
  const height = Number(match[2])
  if (!Number.isFinite(width) || !Number.isFinite(height) || width <= 0 || height <= 0) return ''
  return `${width} / ${height}`
}

function formatDate(value?: string) {
  if (!value) return '发布时间未知'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return `${new Intl.DateTimeFormat('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  }).format(date)} 发布`
}

function viewLabel(view: PromptSquareView) {
  return viewOptions.find((option) => option.value === view)?.label || '广场'
}

function formatErrorMessage(err: unknown, fallback: string) {
  return err instanceof Error ? err.message : fallback
}
