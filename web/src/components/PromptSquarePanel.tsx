import { type FormEvent, useEffect, useMemo, useState } from 'react'
import { createPromptSquareItem, listPromptSquareItems } from '../api/promptSquare'
import type { CreatePromptSquareItemRequest, PromptSquareItem } from '../types'

const demoItems: PromptSquareItem[] = [
  {
    id: 'demo_cinematic_portrait',
    title: '电影感人物海报',
    prompt: 'Cinematic portrait of a confident traveler standing under neon rain, dramatic rim light, shallow depth of field, ultra detailed, editorial poster composition',
    model: 'gpt-image-2',
    params: { ratio: '9:16', quality: 'high' },
    tags: ['portrait', 'cinematic', 'neon'],
    author: { name: 'Lyra Demo' },
    source: { type: 'demo', license: 'example_only' },
    status: 'published',
    createdAt: '2026-06-01T00:00:00Z',
    updatedAt: '2026-06-01T00:00:00Z',
  },
  {
    id: 'demo_product_render',
    title: '极简产品渲染',
    prompt: 'Minimal product render of a translucent smart speaker on a matte stone pedestal, soft studio lighting, clean background, premium commercial photography',
    model: 'gpt-image-2',
    params: { ratio: '1:1', quality: 'high' },
    tags: ['product', 'minimal', 'studio'],
    author: { name: 'Lyra Demo' },
    source: { type: 'demo', license: 'example_only' },
    status: 'published',
    createdAt: '2026-06-01T00:00:00Z',
    updatedAt: '2026-06-01T00:00:00Z',
  },
]

type PromptSquarePanelProps = {
  onUsePrompt: (prompt: string, item?: PromptSquareItem) => void
}

export function PromptSquarePanel({ onUsePrompt }: PromptSquarePanelProps) {
  const [items, setItems] = useState<PromptSquareItem[]>([])
  const [query, setQuery] = useState('')
  const [selectedTag, setSelectedTag] = useState('')
  const [loading, setLoading] = useState(true)
  const [message, setMessage] = useState('')
  const [error, setError] = useState('')
  const [form, setForm] = useState<CreatePromptSquareItemRequest>({
    title: '',
    prompt: '',
    model: 'gpt-image-2',
    tags: '',
    license: 'user_submitted',
    ratio: '',
    resolution: '',
    quality: 'high',
    outputFormat: 'png',
    image: null,
  })

  useEffect(() => {
    void refresh()
  }, [])

  async function refresh() {
    setLoading(true)
    setError('')
    try {
      setItems(await listPromptSquareItems())
    } catch (err) {
      setError(err instanceof Error ? err.message : '读取提示词广场失败')
    } finally {
      setLoading(false)
    }
  }

  const displayItems = items.length ? items : demoItems
  const tags = useMemo(() => {
    const set = new Set<string>()
    for (const item of displayItems) {
      for (const tag of item.tags || []) set.add(tag)
    }
    return Array.from(set).slice(0, 24)
  }, [displayItems])

  const filtered = useMemo(() => {
    const keyword = query.trim().toLowerCase()
    return displayItems.filter((item) => {
      const text = [item.title, item.prompt, item.model, ...(item.tags || [])].join(' ').toLowerCase()
      const matchQuery = !keyword || text.includes(keyword)
      const matchTag = !selectedTag || item.tags?.includes(selectedTag)
      return matchQuery && matchTag
    })
  }, [displayItems, query, selectedTag])

  async function submit(event: FormEvent) {
    event.preventDefault()
    setError('')
    setMessage('')
    if (!form.prompt.trim()) {
      setError('请先填写提示词')
      return
    }
    try {
      const item = await createPromptSquareItem(form)
      setItems((prev) => [item, ...prev])
      setForm({
        title: '',
        prompt: '',
        model: 'gpt-image-2',
        tags: '',
        license: 'user_submitted',
        ratio: '',
        resolution: '',
        quality: 'high',
        outputFormat: 'png',
        image: null,
      })
      setMessage('已提交到本地提示词广场。下一步可把 data/prompt_square 同步到 GitHub 数据仓库。')
    } catch (err) {
      setError(err instanceof Error ? err.message : '提交失败')
    }
  }

  return (
    <div className="prompt-square">
      <section className="prompt-square-hero">
        <div>
          <p className="eyebrow">Prompt Square / Dev Preview</p>
          <h3>提示词广场试验版</h3>
          <p>先跑通“用户上传提示词 + 图片/来源 + 本地结构化存储”的闭环；后续再把同一份 JSON 和图片同步到 GitHub 数据仓库并做 PR 审核。</p>
        </div>
        <button type="button" onClick={() => void refresh()} disabled={loading}>{loading ? '刷新中' : '刷新广场'}</button>
      </section>

      <section className="prompt-square-layout">
        <form className="prompt-square-submit" onSubmit={submit}>
          <div className="panel-title">
            <strong>上传提示词</strong>
            <span>当前 dev 版写入后端 `data/prompt_square`</span>
          </div>
          <label>
            标题
            <input value={form.title} onChange={(event) => setForm({ ...form, title: event.target.value })} placeholder="例如：电影感霓虹人像" />
          </label>
          <label>
            提示词 *
            <textarea value={form.prompt} onChange={(event) => setForm({ ...form, prompt: event.target.value })} placeholder="输入可复用的生图提示词" rows={5} />
          </label>
          <div className="prompt-square-form-grid">
            <label>
              模型
              <input value={form.model || ''} onChange={(event) => setForm({ ...form, model: event.target.value })} placeholder="gpt-image-2" />
            </label>
            <label>
              标签
              <input value={form.tags || ''} onChange={(event) => setForm({ ...form, tags: event.target.value })} placeholder="portrait, neon, 9:16" />
            </label>
            <label>
              比例
              <input value={form.ratio || ''} onChange={(event) => setForm({ ...form, ratio: event.target.value })} placeholder="9:16 / 1:1" />
            </label>
            <label>
              质量
              <input value={form.quality || ''} onChange={(event) => setForm({ ...form, quality: event.target.value })} placeholder="high" />
            </label>
          </div>
          <label>
            上传图片
            <input type="file" accept="image/png,image/jpeg,image/webp" onChange={(event) => setForm({ ...form, image: event.target.files?.[0] || null })} />
          </label>
          <label>
            或填写图片 URL
            <input value={form.imageUrl || ''} onChange={(event) => setForm({ ...form, imageUrl: event.target.value })} placeholder="https://..." />
          </label>
          <div className="prompt-square-form-grid">
            <label>
              来源链接
              <input value={form.sourceUrl || ''} onChange={(event) => setForm({ ...form, sourceUrl: event.target.value })} placeholder="https://..." />
            </label>
            <label>
              授权/许可
              <input value={form.license || ''} onChange={(event) => setForm({ ...form, license: event.target.value })} placeholder="user_submitted / CC-BY" />
            </label>
          </div>
          <button type="submit">提交到广场</button>
          {message ? <p className="prompt-square-message">{message}</p> : null}
          {error ? <p className="prompt-square-error">{error}</p> : null}
        </form>

        <section className="prompt-square-feed">
          <div className="prompt-square-toolbar">
            <input value={query} onChange={(event) => setQuery(event.target.value)} placeholder="搜索标题、提示词、模型、标签" />
            <div className="prompt-square-tags">
              <button type="button" className={!selectedTag ? 'active' : ''} onClick={() => setSelectedTag('')}>全部</button>
              {tags.map((tag) => (
                <button key={tag} type="button" className={selectedTag === tag ? 'active' : ''} onClick={() => setSelectedTag(tag)}>{tag}</button>
              ))}
            </div>
          </div>
          {!items.length ? <p className="prompt-square-hint">当前还没有真实上传数据，下方展示的是前端 demo 卡片。</p> : null}
          <div className="prompt-square-grid">
            {filtered.map((item) => (
              <PromptSquareCard key={item.id} item={item} onUsePrompt={onUsePrompt} />
            ))}
          </div>
        </section>
      </section>
    </div>
  )
}

function PromptSquareCard({ item, onUsePrompt }: { item: PromptSquareItem; onUsePrompt: PromptSquarePanelProps['onUsePrompt'] }) {
  return (
    <article className="prompt-square-card">
      <div className="prompt-square-image">
        {item.thumbnailUrl || item.imageUrl ? <img src={item.thumbnailUrl || item.imageUrl} alt={item.title} loading="lazy" /> : <div className="prompt-square-placeholder">Prompt</div>}
      </div>
      <div className="prompt-square-card-body">
        <div className="prompt-square-card-head">
          <strong>{item.title}</strong>
          <span>{item.model || 'unknown model'}</span>
        </div>
        <p>{item.prompt}</p>
        <div className="prompt-square-chip-row">
          {(item.tags || []).slice(0, 6).map((tag) => <span key={tag}>{tag}</span>)}
        </div>
        <footer>
          <span>{item.author?.name || '匿名'} · {item.source?.license || 'unknown'}</span>
          <div>
            {item.source?.url ? <a href={item.source.url} target="_blank" rel="noreferrer">来源</a> : null}
            <button type="button" onClick={() => void navigator.clipboard.writeText(item.prompt)}>复制</button>
            <button type="button" onClick={() => onUsePrompt(item.prompt, item)}>使用</button>
          </div>
        </footer>
      </div>
    </article>
  )
}
