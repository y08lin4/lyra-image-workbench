import { useEffect, useMemo, useState } from 'react'
import { likePromptSquareItem, listPromptSquareItems, type PromptSquareListOptions } from '../api/promptSquare'
import type { PromptSquareItem } from '../types'
import { ImagePreviewModal } from './ImagePreviewModal'

type PromptSquarePanelProps = {
  onUsePrompt: (prompt: string, item?: PromptSquareItem) => void | Promise<void>
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
    emptyBody: '生成结果投稿入口会出现在结果卡片里；有公开作品后这里会自动展示。',
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
    emptyBody: '从生成结果提交作品后，它们会按当前广场展示策略出现在这里。',
  },
]

export function PromptSquarePanel({ onUsePrompt }: PromptSquarePanelProps) {
  const [view, setView] = useState<PromptSquareView>('latest')
  const [items, setItems] = useState<PromptSquareItem[]>([])
  const [query, setQuery] = useState('')
  const [selectedModel, setSelectedModel] = useState('')
  const [selectedType, setSelectedType] = useState('')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const [likingIds, setLikingIds] = useState<Record<string, boolean>>({})
  const [selectedItem, setSelectedItem] = useState<PromptSquareItem | null>(null)

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
    setSelectedModel('')
    setSelectedType('')
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
    setSelectedItem((current) => current?.id === item.id ? withLocalLike(current, liked) : current)
    try {
      const updated = await likePromptSquareItem(item.id, liked)
      setItems((prev) => prev.map((entry) => entry.id === item.id ? updated : entry))
      setSelectedItem((current) => current?.id === item.id ? updated : current)
      setNotice(liked ? '已点赞' : '已取消点赞')
    } catch (err) {
      setItems((prev) => prev.map((entry) => entry.id === item.id ? previous : entry))
      setSelectedItem((current) => current?.id === item.id ? previous : current)
      setError(`点赞失败：${formatErrorMessage(err, '请稍后再试')}`)
    } finally {
      setLikingIds((prev) => {
        const next = { ...prev }
        delete next[item.id]
        return next
      })
    }
  }

  const activeView = viewOptions.find((option) => option.value === view) || viewOptions[0]

  const modelOptions = useMemo(() => {
    const set = new Set<string>()
    for (const item of items) {
      const model = displayModelLabel(asRuntimeItem(item))
      if (model && model !== '未标注') set.add(model)
    }
    return Array.from(set).slice(0, 12)
  }, [items])

  const typeOptions = useMemo(() => {
    const set = new Set<string>()
    for (const item of items) set.add(itemTypeLabel(asRuntimeItem(item)))
    return Array.from(set).slice(0, 8)
  }, [items])
  const filtered = useMemo(() => {
    const keyword = query.trim().toLowerCase()
    return items.filter((item) => {
      const runtime = asRuntimeItem(item)
      const model = displayModelLabel(runtime)
      const type = itemTypeLabel(runtime)
      const text = [
        runtime.title,
        runtime.prompt,
        model,
        type,
        runtime.ratio,
        runtime.quality,
        runtime.outputFormat,
        runtime.params?.ratio,
        runtime.params?.quality,
        runtime.params?.outputFormat,
        runtime.params?.actualSize,
        authorName(runtime),
        ...(runtime.tags || []),
      ].join(' ').toLowerCase()
      const matchQuery = !keyword || text.includes(keyword)
      const matchModel = !selectedModel || model === selectedModel
      const matchType = !selectedType || type === selectedType
      return matchQuery && matchModel && matchType
    })
  }, [items, query, selectedModel, selectedType])
  const hasFilter = Boolean(query || selectedModel || selectedType)
  const emptyTitle = hasFilter ? '没有匹配的作品' : activeView.emptyTitle
  const emptyBody = hasFilter ? '换个关键词或清除筛选后再试。' : activeView.emptyBody

  return (
    <div className="prompt-square">
      <section className="prompt-square-hero">
        <div className="prompt-square-hero-copy">
          <h3>提示词广场</h3>
          <p>{activeView.description}</p>
        </div>
        <div className="prompt-square-header-actions">
          <button type="button" className="prompt-square-submit-hint" onClick={() => setNotice('投稿入口在结果页的结果卡片里，提交前会再次确认公开范围。')}>投稿入口</button>
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
          <label className="prompt-square-search">
            <span>搜索</span>
            <input value={query} onChange={(event) => setQuery(event.target.value)} placeholder="标题、提示词、作者、模型" />
          </label>
          <div className="prompt-square-filter-row">
            <div className="prompt-square-filter-group">
              <span>模型</span>
              <div className="prompt-square-tags" aria-label="模型筛选">
                <button type="button" className={!selectedModel ? 'active' : ''} onClick={() => setSelectedModel('')} aria-pressed={!selectedModel}>全部</button>
                {modelOptions.map((model) => (
                  <button key={model} type="button" className={selectedModel === model ? 'active' : ''} onClick={() => setSelectedModel(model)} aria-pressed={selectedModel === model}>{model}</button>
                ))}
              </div>
            </div>
            <div className="prompt-square-filter-group">
              <span>类型</span>
              <div className="prompt-square-tags" aria-label="类型筛选">
                <button type="button" className={!selectedType ? 'active' : ''} onClick={() => setSelectedType('')} aria-pressed={!selectedType}>全部</button>
                {typeOptions.map((type) => (
                  <button key={type} type="button" className={selectedType === type ? 'active' : ''} onClick={() => setSelectedType(type)} aria-pressed={selectedType === type}>{type}</button>
                ))}
              </div>
            </div>
          </div>
        </div>
        {notice ? <p className="prompt-square-alert success" role="status">{notice}</p> : null}
        {error ? <p className="prompt-square-alert error" role="alert">{error}</p> : null}

        {loading ? (
          <PromptSquareSkeleton />
        ) : error && !items.length ? (
          <PromptSquareState title={`${activeView.label}暂不可用`} body="提示词广场暂时无法加载，请稍后刷新。" actionLabel="重试" onAction={() => void refresh(view)} />
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
                onOpenDetails={setSelectedItem}
              />
            ))}
          </div>
        ) : (
          <PromptSquareState title={emptyTitle} body={emptyBody} />
        )}
        {selectedItem ? (
          <PromptSquareDetailModal
            item={selectedItem}
            liking={Boolean(likingIds[selectedItem.id])}
            onClose={() => setSelectedItem(null)}
            onUsePrompt={onUsePrompt}
            onCopyPrompt={copyPrompt}
            onToggleLike={toggleLike}
          />
        ) : null}
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
  onOpenDetails,
}: {
  item: PromptSquareItem
  dailyRank: number
  liking: boolean
  onUsePrompt: PromptSquarePanelProps['onUsePrompt']
  onCopyPrompt: (prompt: string) => Promise<void>
  onToggleLike: (item: PromptSquareItem) => Promise<void>
  onOpenDetails: (item: PromptSquareItem) => void
}) {
  const runtime = asRuntimeItem(item)
  const title = itemTitle(runtime)
  const ratio = itemParam(runtime, 'ratio')
  const actualSize = itemParam(runtime, 'actualSize')
  const quality = itemParam(runtime, 'quality')
  const outputFormat = itemParam(runtime, 'outputFormat')
  const likes = likeCount(runtime)
  const liked = Boolean(runtime.likedByMe)
  const imageUrl = runtime.thumbnailUrl || runtime.imageUrl
  const rank = Number(runtime.dailyRank || dailyRank || 0)

  const model = displayModelLabel(runtime)
  const type = itemTypeLabel(runtime)
  const refs = referenceCount(runtime)
  const referenceText = referenceSummary(runtime)
  const tagPreview = (runtime.tags || []).slice(0, 3)
  const extraTagCount = Math.max(0, (runtime.tags?.length || 0) - tagPreview.length)

  return (
    <article className="prompt-square-card">
      <div className="prompt-square-image">
        {rank > 0 ? <span className="prompt-square-rank">每日 #{rank}</span> : null}
        {refs > 0 ? <span className="prompt-square-ref-badge">{refs} 参考图</span> : null}
        {imageUrl ? <img src={imageUrl} alt={title} loading="lazy" /> : <div className="prompt-square-placeholder">Prompt</div>}
      </div>
      <div className="prompt-square-quick-actions">
        <button type="button" onClick={() => onOpenDetails(item)}>详情</button>
        <button type="button" className="prompt-square-use" onClick={() => void onUsePrompt(runtime.prompt, item)}>应用</button>
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
      <div className="prompt-square-card-body">
        <header className="prompt-square-card-head">
          <div className="prompt-square-title-block">
            <strong title={title}>{title}</strong>
            <span title={`${model} · ${type}`}>{model} · {type}</span>
          </div>
          <span className="prompt-square-like-count">{likes} 赞</span>
        </header>

        <p className="prompt-square-prompt" title={runtime.prompt}>{runtime.prompt}</p>
        {referenceText ? <p className="prompt-square-reference-note" title={referenceText}>参考图：{referenceText}</p> : null}

        <dl className="prompt-square-meta-grid">
          <div><dt>比例</dt><dd>{ratio || '未标注'}</dd></div>
          <div><dt>质量</dt><dd>{quality || '未标注'}</dd></div>
          <div><dt>格式</dt><dd>{outputFormat || '未标注'}</dd></div>
          <div><dt>尺寸</dt><dd>{actualSize || '未标注'}</dd></div>
        </dl>

        <div className="prompt-square-chip-row">
          <span>{type}</span>
          {tagPreview.map((tag) => <span key={tag} title={tag}>{tag}</span>)}
          {extraTagCount ? <span>+{extraTagCount}</span> : null}
          {runtime.tags?.length ? null : <span>无标签</span>}
        </div>

        <footer className="prompt-square-card-footer">
          <div className="prompt-square-author">
            <strong>{authorName(runtime)}</strong>
            <span>{formatDate(runtime.submittedAt || runtime.createdAt)}</span>
          </div>
          <button type="button" onClick={() => void onCopyPrompt(runtime.prompt)}>复制提示词</button>
        </footer>
      </div>
    </article>
  )
}

function PromptSquareDetailModal({
  item,
  liking,
  onClose,
  onUsePrompt,
  onCopyPrompt,
  onToggleLike,
}: {
  item: PromptSquareItem
  liking: boolean
  onClose: () => void
  onUsePrompt: PromptSquarePanelProps['onUsePrompt']
  onCopyPrompt: (prompt: string) => Promise<void>
  onToggleLike: (item: PromptSquareItem) => Promise<void>
}) {
  const runtime = asRuntimeItem(item)
  const title = itemTitle(runtime)
  const imageUrl = runtime.imageUrl || runtime.thumbnailUrl
  const references = runtime.references || []
  const hasReferenceRecord = references.length > 0 || referenceCount(runtime) > 0
  const expectsReferences = itemTypeLabel(runtime) === '图生图' || hasReferenceRecord
  const liked = Boolean(runtime.likedByMe)
  const likes = likeCount(runtime)
  const sourceUrl = runtime.source?.url
  const transformText = transformationSummary(runtime, references.length)
  const [previewImage, setPreviewImage] = useState<{ src: string; title: string } | null>(null)

  return (
    <>
      <div className="prompt-square-detail-backdrop" role="presentation" onMouseDown={onClose}>
        <section className="prompt-square-detail" role="dialog" aria-modal="true" aria-label="投稿详情" onMouseDown={(event) => event.stopPropagation()}>
          <header className="prompt-square-detail-head">
            <div>
              <strong>{title}</strong>
              <span>{displayModelLabel(runtime)} · {itemTypeLabel(runtime)} · {authorName(runtime)}</span>
            </div>
            <button type="button" onClick={onClose} aria-label="关闭详情">×</button>
          </header>
          <div className="prompt-square-detail-body">
            <figure className="prompt-square-detail-image">
              {imageUrl ? (
                <>
                  <img src={imageUrl} alt={title} />
                  <button type="button" onClick={() => setPreviewImage({ src: imageUrl, title: `${title} · 成品图` })}>放大查看</button>
                </>
              ) : <div>Prompt</div>}
            </figure>
            <div className="prompt-square-detail-info">
              <section className="prompt-square-detail-section">
                <strong>生成链路</strong>
                <p>{transformText}</p>
              </section>
              <section className="prompt-square-detail-section">
                <strong>生成提示词</strong>
                <p className="prompt-square-detail-prompt">{runtime.prompt}</p>
              </section>
              {runtime.referenceUsageNote ? <p className="prompt-square-detail-note">参考说明：{runtime.referenceUsageNote}</p> : null}
              <dl className="prompt-square-detail-meta">
                <div><dt>模型</dt><dd>{displayModelLabel(runtime)}</dd></div>
                <div><dt>类型</dt><dd>{itemTypeLabel(runtime)}</dd></div>
                <div><dt>比例</dt><dd>{itemParam(runtime, 'ratio') || '未标注'}</dd></div>
                <div><dt>质量</dt><dd>{itemParam(runtime, 'quality') || '未标注'}</dd></div>
                <div><dt>格式</dt><dd>{itemParam(runtime, 'outputFormat') || '未标注'}</dd></div>
                <div><dt>尺寸</dt><dd>{itemParam(runtime, 'actualSize') || runtime.resolution || '未标注'}</dd></div>
                <div><dt>参考图</dt><dd>{references.length ? `${references.length} 张` : expectsReferences ? '旧数据缺失' : '无'}</dd></div>
                <div><dt>点赞</dt><dd>{likes}</dd></div>
                <div><dt>时间</dt><dd>{formatDate(runtime.submittedAt || runtime.createdAt)}</dd></div>
              </dl>
              {references.length ? (
                <div className="prompt-square-detail-refs">
                  <strong>输入参考图</strong>
                  <div>
                    {references.slice(0, 6).map((reference, index) => {
                      const refUrl = reference.imageUrl || reference.thumbnailUrl
                      const label = referenceLabel(reference, index)
                      return (
                        <button key={`${refUrl || label}-${index}`} type="button" onClick={() => refUrl && setPreviewImage({ src: refUrl, title: label })} disabled={!refUrl}>
                          {refUrl ? <img src={refUrl} alt={label} /> : <span>参考图文件缺失</span>}
                          <small>{label}</small>
                        </button>
                      )
                    })}
                  </div>
                </div>
              ) : expectsReferences ? (
                <div className="prompt-square-detail-refs missing">
                  <strong>输入参考图</strong>
                  <p>这条历史投稿没有保存原始参考图副本，所以只能看到成品图和提示词，无法完整复原图生图链路。重新从结果页提交后，新作品会公开保存参考图、提示词和参数。</p>
                </div>
              ) : (
                <div className="prompt-square-detail-refs missing">
                  <strong>输入参考图</strong>
                  <p>这是一条文生图或无参考图投稿，没有使用原始参考图。</p>
                </div>
              )}
              <footer className="prompt-square-detail-actions">
                <button type="button" onClick={() => void onCopyPrompt(runtime.prompt)}>复制提示词</button>
                {sourceUrl ? <a href={sourceUrl} target="_blank" rel="noreferrer">来源</a> : null}
                <button type="button" onClick={() => void onUsePrompt(runtime.prompt, item)}>应用到创作</button>
                <button type="button" className={liked ? 'prompt-square-like active' : 'prompt-square-like'} onClick={() => void onToggleLike(item)} disabled={liking} aria-pressed={liked}>
                  {liking ? '处理中' : liked ? '已赞' : '点赞'}
                </button>
              </footer>
            </div>
          </div>
        </section>
      </div>
      {previewImage ? (
        <ImagePreviewModal src={previewImage.src} title={previewImage.title} onClose={() => setPreviewImage(null)} />
      ) : null}
    </>
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

function itemParam(item: RuntimePromptSquareItem, key: 'ratio' | 'quality' | 'outputFormat' | 'actualSize') {
  if (key === 'actualSize') return (item.params?.actualSize || '').trim()
  return (item[key] || item.params?.[key] || '').trim()
}

function displayModelLabel(item: RuntimePromptSquareItem) {
  const model = (item.model || '').trim()
  if (!model) return '未标注'
  if (model === 'image-2' || model === 'gpt-image-2') return 'gpt-image-2'
  return model
}

function itemTypeLabel(item: RuntimePromptSquareItem) {
  const tags = (item.tags || []).join(' ').toLowerCase()
  if (tags.includes('gif')) return 'GIF'
  if (tags.includes('图生图') || tags.includes('参考图') || tags.includes('image-to-image')) return '图生图'
  if (item.source?.type === 'task_result') return '结果投稿'
  return '文生图'
}

function referenceCount(item: RuntimePromptSquareItem) {
  const fromReferences = item.references?.length || 0
  const fromUploadIds = item.referenceUploadIds?.length || 0
  const fromParams = Number(item.params?.referenceCount || 0) || 0
  return Math.max(fromReferences, fromUploadIds, fromParams, 0)
}

function referenceSummary(item: RuntimePromptSquareItem) {
  const explicitNote = item.referenceUsageNote?.trim()
  if (explicitNote) return explicitNote
  const referenceNotes = (item.references || [])
    .map((reference) => reference.usageNote || reference.originalName || reference.fileName || '')
    .map((value) => value.trim())
    .filter(Boolean)
  if (referenceNotes.length) return Array.from(new Set(referenceNotes)).slice(0, 3).join('；')
  const count = referenceCount(item)
  return count ? `${count} 张参考图随作品公开` : ''
}

function referenceLabel(reference: { usageNote?: string; originalName?: string; fileName?: string }, index: number) {
  const explicit = (reference.usageNote || reference.originalName || reference.fileName || '').trim()
  return explicit || `参考图 ${index + 1}`
}

function transformationSummary(item: RuntimePromptSquareItem, referenceLength: number) {
  const count = referenceLength || referenceCount(item)
  const type = itemTypeLabel(item)
  const prompt = item.prompt?.trim()
  if (count > 0) {
    const note = item.referenceUsageNote?.trim() || referenceSummary(item)
    const prefix = `这张作品由 ${count} 张输入参考图 + 生成提示词 + ${displayModelLabel(item)} 参数生成。`
    return note ? `${prefix} 参考用途：${note}` : `${prefix} 下方会展示公开保存的原始参考图，便于理解主体、风格、构图或细节如何影响最终成品。`
  }
  if (type === '图生图') return '这条投稿标记为图生图，但历史数据没有保存参考图副本；当前只能展示成品图、提示词和参数。'
  return prompt ? `这张作品由提示词直接生成，未使用公开参考图。` : '这条投稿没有保存可展示的生成链路。'
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
