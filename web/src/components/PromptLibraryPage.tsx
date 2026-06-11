import { useEffect, useMemo, useState } from 'react'
import { formatError } from '../api/client'
import { listPromptLibrary, refreshPromptLibrary } from '../api/promptLibrary'
import type { ModelProvider, PromptLibrary, PromptLibraryItem } from '../types'
import { BANANA_PROVIDER, DEFAULT_IMAGE2_MODEL, providerLabel } from '../lib/models'

type Props = {
  provider: ModelProvider
  bananaModel: string
  onUsePrompt: (prompt: string, options: { provider: ModelProvider; model: string }) => void
}

export function PromptLibraryPage({ provider, bananaModel, onUsePrompt }: Props) {
  const [library, setLibrary] = useState<PromptLibrary | null>(null)
  const [query, setQuery] = useState('')
  const [category, setCategory] = useState('')
  const [activeItem, setActiveItem] = useState<PromptLibraryItem | null>(null)
  const [loading, setLoading] = useState(false)
  const [loaded, setLoaded] = useState(false)
  const [message, setMessage] = useState('')
  const [error, setError] = useState('')

  const selectedModel = useMemo(() => provider === BANANA_PROVIDER ? bananaModel : DEFAULT_IMAGE2_MODEL, [bananaModel, provider])

  useEffect(() => {
    if (!loaded && !loading) void refresh(false)
  }, [loaded, loading])

  useEffect(() => {
    if (!loaded) return
    const timer = window.setTimeout(() => {
      void refresh(false)
    }, 280)
    return () => window.clearTimeout(timer)
  }, [loaded, query, category])

  async function refresh(force = false) {
    setLoading(true)
    setError('')
    try {
      const params = { lang: 'zh-CN', q: query.trim(), category, limit: 120 }
      const nextLibrary = force ? await refreshPromptLibrary(params) : await listPromptLibrary(params)
      setLibrary(nextLibrary)
      setLoaded(true)
      setActiveItem((current) => {
        if (!current) return nextLibrary.items[0] || null
        return nextLibrary.items.find((item) => item.id === current.id) || nextLibrary.items[0] || null
      })
      if (force) setMessage(nextLibrary.stale ? '提示词库刷新失败，已显示本地缓存' : '提示词库已从 GitHub 同步')
    } catch (err) {
      setLoaded(true)
      setError(formatError(err, '提示词库加载失败'))
    } finally {
      setLoading(false)
    }
  }

  async function copyPrompt(prompt: string) {
    await navigator.clipboard.writeText(prompt)
    setMessage('提示词已复制')
  }

  function usePrompt(item: PromptLibraryItem) {
    onUsePrompt(item.prompt, { provider, model: selectedModel })
    setMessage(`已填入生成页，并切到 ${providerLabel(provider)}`)
  }

  const previewPrompt = activeItem?.prompt || ''

  return (
    <section className="workflow-page prompt-library-page">
      <header className="workflow-page-header prompt-library-page-header">
        <div>
          <p className="eyebrow">Prompt Library</p>
          <h2>提示词库</h2>
          <p>实时同步 ZeroLu/awesome-gpt-image，搜索高质量 GPT Image 示例并一键填入生成页。</p>
        </div>
        <button type="button" onClick={() => void refresh(true)} disabled={loading}>{loading ? '同步中...' : '从 GitHub 刷新'}</button>
      </header>

      <div className="prompt-library-page-grid">
        <section className="prompt-library-panel prompt-library-browser">
          <div className="prompt-library-toolbar">
            <label>
              <span>搜索</span>
              <input value={query} onChange={(event) => setQuery(event.target.value)} placeholder="搜索标题、分类或提示词" />
            </label>
            <label>
              <span>搜索</span>
              <select value={category} onChange={(event) => setCategory(event.target.value)}>
                <option value="">全部分类</option>
                {library?.categories.map((item) => <option key={item} value={item}>{item}</option>)}
              </select>
            </label>
          </div>

          {library ? (
            <div className={`prompt-library-status ${library.stale ? 'stale' : ''}`}>
              <span>{library.repo} · {library.matching}/{library.total} 条</span>
              <span>同步：{formatLibraryTime(library.fetchedAt)}</span>
              {library.stale ? <strong>当前显示缓存：{library.fetchError || 'GitHub 暂不可用'}</strong> : null}
            </div>
          ) : null}
          {loading && !library ? <div className="prompt-empty">正在从 GitHub 加载提示词库...</div> : null}
          {!loading && library && !library.items.length ? <div className="prompt-empty">没有匹配的提示词</div> : null}

          <div className="prompt-library-list prompt-library-list-large">
            {library?.items.map((item) => (
              <article key={item.id} className={`prompt-library-item ${activeItem?.id === item.id ? 'active' : ''}`} onClick={() => setActiveItem(item)}>
                <div className="prompt-library-thumb">
                  {item.images?.[0] ? <img src={item.images[0].url} alt={item.images[0].alt || item.title} loading="lazy" referrerPolicy="no-referrer" /> : <span>Prompt</span>}
                </div>
                <div className="prompt-library-copy">
                  <strong>{item.title}</strong>
                  <small>{item.category}</small>
                  <p>{item.prompt}</p>
                  <div className="prompt-library-actions">
                    <button type="button" onClick={(event) => { event.stopPropagation(); void copyPrompt(item.prompt) }}>复制</button>
                    <button type="button" className="primary" onClick={(event) => { event.stopPropagation(); usePrompt(item) }}>应用</button>
                    {item.sources?.[0] ? <a href={item.sources[0].url} target="_blank" rel="noopener noreferrer" onClick={(event) => event.stopPropagation()}>来源</a> : null}
                  </div>
                </div>
              </article>
            ))}
          </div>
        </section>

        <aside className="prompt-result prompt-library-preview">
          {activeItem ? (
            <>
              <div className="prompt-result-title">
                <div className="prompt-result-title-main">
                  <strong>{activeItem.title}</strong>
                  <span>{activeItem.category}</span>
                </div>
                <div className="prompt-result-title-actions">
                  <button type="button" onClick={() => void copyPrompt(activeItem.prompt)}>复制</button>
                  <button type="button" className="primary" onClick={() => usePrompt(activeItem)}>应用</button>
                </div>
              </div>

              {activeItem.images?.length ? (
                <div className="prompt-library-preview-images">
                  {activeItem.images.slice(0, 4).map((image) => <img key={image.url} src={image.url} alt={image.alt || activeItem.title} loading="lazy" referrerPolicy="no-referrer" />)}
                </div>
              ) : null}

              <label>
                <span>正向提示词</span>
                <textarea value={previewPrompt} readOnly rows={12} />
              </label>
              <section className="prompt-apply-model" aria-label="当前应用模型">
                <div className="section-title">
                  <span>应用到当前模型</span>
                  <small>{providerLabel(provider)} / {selectedModel}</small>
                </div>
              </section>
              {activeItem.sources?.length ? (
                <div className="prompt-chips">
                  {activeItem.sources.map((source) => <a key={source.url} href={source.url} target="_blank" rel="noopener noreferrer">{source.label}</a>)}
                </div>
              ) : null}
            </>
          ) : (
            <div className="prompt-empty">选择一个提示词查看完整内容</div>
          )}
        </aside>
      </div>

      {message ? <div className="ok">{message}</div> : null}
      {error ? <div className="error">{error}</div> : null}
    </section>
  )
}

function formatLibraryTime(value: string) {
  if (!value) return '未知'
  const time = new Date(value)
  if (Number.isNaN(time.getTime())) return value
  return time.toLocaleString()
}
