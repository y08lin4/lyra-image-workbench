import { useEffect, useMemo, useRef, useState } from 'react'
import type { KeyboardEvent } from 'react'
import { formatError } from '../api/client'
import { getCachedPromptLibrary, listPromptLibrary, refreshPromptLibrary } from '../api/promptLibrary'
import type { ModelProvider, PromptLibrary, PromptLibraryItem } from '../types'
import { BANANA_PROVIDER, DEFAULT_IMAGE2_MODEL, providerLabel } from '../lib/models'

type Props = {
  provider: ModelProvider
  bananaModel: string
  onUsePrompt: (prompt: string, options: { provider: ModelProvider; model: string }) => void
}

const AUTO_REFRESH_MS = 10 * 60 * 1000
const DEFAULT_PROMPT_LIBRARY_PARAMS = { lang: 'zh-CN', limit: 120 }

function promptLibraryParams(query: string, category: string) {
  return { ...DEFAULT_PROMPT_LIBRARY_PARAMS, q: query.trim(), category }
}

export function PromptLibraryPage({ provider, bananaModel, onUsePrompt }: Props) {
  const cachedOnOpen = useMemo(() => getCachedPromptLibrary(DEFAULT_PROMPT_LIBRARY_PARAMS), [])
  const [library, setLibrary] = useState<PromptLibrary | null>(cachedOnOpen)
  const [query, setQuery] = useState('')
  const [category, setCategory] = useState('')
  const [activeItem, setActiveItem] = useState<PromptLibraryItem | null>(() => cachedOnOpen?.items[0] || null)
  const [loading, setLoading] = useState(false)
  const [loaded, setLoaded] = useState(Boolean(cachedOnOpen))
  const [autoCheckedAt, setAutoCheckedAt] = useState('')
  const [message, setMessage] = useState('')
  const [error, setError] = useState('')
  const loadingRef = useRef(false)
  const filterRefreshReadyRef = useRef(false)
  const previewRef = useRef<HTMLElement | null>(null)

  const selectedModel = useMemo(() => provider === BANANA_PROVIDER ? bananaModel : DEFAULT_IMAGE2_MODEL, [bananaModel, provider])

  useEffect(() => {
    void refresh(false)
  }, [])

  useEffect(() => {
    if (!loaded) return
    if (!filterRefreshReadyRef.current) {
      filterRefreshReadyRef.current = true
      return
    }
    const timer = window.setTimeout(() => {
      void refresh(false)
    }, 280)
    return () => window.clearTimeout(timer)
  }, [loaded, query, category])

  useEffect(() => {
    if (!loaded) return
    const timer = window.setInterval(() => {
      if (document.visibilityState === 'visible') void refresh(true)
    }, AUTO_REFRESH_MS)
    return () => window.clearInterval(timer)
  }, [loaded, query, category])
  useEffect(() => {
    previewRef.current?.scrollTo({ top: 0, behavior: 'smooth' })
  }, [activeItem?.id])

  async function refresh(force = false) {
    if (loadingRef.current) return
    const params = promptLibraryParams(query, category)
    const cached = force ? null : getCachedPromptLibrary(params)
    const hasVisibleLibrary = Boolean(cached || library)
    if (cached) applyLibrary(cached)
    loadingRef.current = true
    setLoading(true)
    setError('')
    try {
      const nextLibrary = force ? await refreshPromptLibrary(params) : await listPromptLibrary(params)
      applyLibrary(nextLibrary)
      if (force) setAutoCheckedAt(new Date().toISOString())
    } catch (err) {
      setLoaded(true)
      if (!hasVisibleLibrary) setError(formatError(err, '提示词库加载失败'))
    } finally {
      loadingRef.current = false
      setLoading(false)
    }
  }

  function applyLibrary(nextLibrary: PromptLibrary) {
    setLibrary(nextLibrary)
    setLoaded(true)
    setActiveItem((current) => {
      if (!current) return nextLibrary.items[0] || null
      return nextLibrary.items.find((item) => item.id === current.id) || nextLibrary.items[0] || null
    })
  }

  async function copyPrompt(prompt: string) {
    await navigator.clipboard.writeText(prompt)
    setMessage('提示词已复制')
  }

  function usePrompt(item: PromptLibraryItem) {
    onUsePrompt(item.prompt, { provider, model: selectedModel })
    setMessage(`已填入生成页，并切到 ${providerLabel(provider)}`)
  }

  function useWorkflowPrompt(item: PromptLibraryItem, workflow: 'template' | 'product-copy') {
    const prompt = workflow === 'template'
      ? [
        '请将下面的提示词作为可复用模板，结合我接下来提供的主体、风格或画面要求进行套用。',
        '保留模板里的构图、镜头、光线、材质与审美关键词；如信息冲突，以我的新要求为准。',
        '',
        `模板名称：${item.title}`,
        `模板分类：${item.category}`,
        '',
        item.prompt,
      ].join('\n')
      : [
        '请基于我上传的产品图，生成适合电商或社媒投放的产品图提示词，并改写一版广告文案。',
        '要求：保持产品外观真实，不改变品牌标识和关键结构；画面突出卖点、使用场景、光线质感与可读排版。',
        '',
        `参考模板：${item.title} / ${item.category}`,
        '',
        item.prompt,
        '',
        '待改写广告词：',
      ].join('\n')
    onUsePrompt(prompt, { provider, model: selectedModel })
    setMessage(workflow === 'template' ? '已填入“模型套模板”草稿' : '已填入“产品图+广告词改写”草稿')
  }

  function selectItem(item: PromptLibraryItem) {
    setActiveItem(item)
  }

  function handleItemKeyDown(event: KeyboardEvent<HTMLElement>, item: PromptLibraryItem) {
    if (event.target !== event.currentTarget) return
    if (event.key !== 'Enter' && event.key !== ' ') return
    event.preventDefault()
    selectItem(item)
  }

  const previewPrompt = activeItem?.prompt || ''

  return (
    <section className="workflow-page prompt-library-page">
      <header className="workflow-page-header prompt-library-page-header">
        <div>
          <p className="eyebrow">Prompt Library</p>
          <h2>提示词库</h2>
          <p>搜索高质量图像提示词，点击列表后在右侧固定预览，并一键填入生成页。</p>
        </div>
        <div className="prompt-library-auto-note" aria-live="polite">
          <span>{loading ? '正在更新' : '自动更新'}</span>
          <strong>每 10 分钟</strong>
        </div>
      </header>

      <div className="prompt-library-page-grid">
        <section className="prompt-library-panel prompt-library-browser">
          <div className="prompt-library-toolbar">
            <label>
              <span>搜索</span>
              <input value={query} onChange={(event) => setQuery(event.target.value)} placeholder="搜索标题、分类或提示词" />
            </label>
            <label>
              <span>分类</span>
              <select value={category} onChange={(event) => setCategory(event.target.value)}>
                <option value="">全部分类</option>
                {library?.categories.map((item) => <option key={item} value={item}>{item}</option>)}
              </select>
            </label>
          </div>

          {library ? (
            <div className={`prompt-library-status ${library.stale ? 'stale' : ''}`}>
              <span>{library.matching}/{library.total} 条</span>
              <span>更新：{formatLibraryTime(library.fetchedAt)}</span>
              {autoCheckedAt ? <span>最近检查：{formatLibraryTime(autoCheckedAt)}</span> : null}
              {library.stale ? <strong>当前显示缓存，后台更新暂不可用</strong> : null}
            </div>
          ) : null}
          {loading && !library ? <div className="prompt-empty">正在加载提示词库...</div> : null}
          {!loading && library && !library.items.length ? <div className="prompt-empty">没有匹配的提示词</div> : null}

          <div className="prompt-library-list prompt-library-list-large">
            {library?.items.map((item) => (
              <article
                key={item.id}
                className={`prompt-library-item ${activeItem?.id === item.id ? 'active' : ''}`}
                role="button"
                tabIndex={0}
                aria-pressed={activeItem?.id === item.id}
                onClick={() => selectItem(item)}
                onKeyDown={(event) => handleItemKeyDown(event, item)}
              >
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
                  </div>
                </div>
              </article>
            ))}
          </div>
        </section>

        <aside ref={previewRef} className="prompt-result prompt-library-preview">
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
                  {activeItem.images.slice(0, 4).map((image) => (
                    <figure key={image.url} className="prompt-library-preview-image">
                      <img src={image.url} alt={image.alt || activeItem.title} loading="lazy" referrerPolicy="no-referrer" />
                    </figure>
                  ))}
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
                <div className="prompt-library-workflows" aria-label="提示词工作流入口">
                  <button type="button" onClick={() => useWorkflowPrompt(activeItem, 'template')}>模型套模板</button>
                  <button type="button" onClick={() => useWorkflowPrompt(activeItem, 'product-copy')}>产品图+广告词改写</button>
                </div>
              </section>
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
