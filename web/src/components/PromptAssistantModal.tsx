import { useEffect, useMemo, useState } from 'react'
import { formatError } from '../api/client'
import {
  deletePromptHistory,
  deletePromptSession,
  expandInspirationIdea,
  generateInspirationIdeas,
  getPromptSession,
  imageToPrompt,
  listPromptHistory,
  listPromptSessions,
  refinePromptSession,
  textToPrompt,
} from '../api/promptTools'
import { uploadReferenceImages } from '../api/uploads'
import type { InspirationIdea, ModelProvider, PromptRecord, PromptSession, PromptVersion, ReferenceUpload, Task } from '../types'
import { BANANA_MODEL_OPTIONS, BANANA_PROVIDER, DEFAULT_BANANA_MODEL, DEFAULT_IMAGE2_MODEL, providerLabel } from '../lib/models'

type Tab = 'text' | 'image' | 'inspiration' | 'history'

type Props = {
  tasks: Task[]
  uploads: ReferenceUpload[]
  provider: ModelProvider
  bananaModel: string
  onClose: () => void
  embedded?: boolean
  onUsePrompt: (prompt: string, options: { provider: ModelProvider; model: string }) => void
  onRefreshUploads: () => Promise<void>
}

const styleOptions = [
  { value: 'auto', label: '自动判断' },
  { value: 'cinematic', label: '电影感' },
  { value: 'photo', label: '写实摄影' },
  { value: 'poster', label: '海报设计' },
  { value: 'anime', label: '二次元' },
  { value: 'product', label: '产品图' },
]

const ratioOptions = ['auto', '1:1', '2:3', '3:2', '3:4', '4:3', '9:16', '16:9']
const categoryOptions = ['随机', '人像', '场景', '产品', '海报', '插画', '壁纸', '建筑', '美食']
const moodOptions = ['随机', '治愈', '孤独', '高级', '梦幻', '压迫感', '温暖', '荒诞', '浪漫']
const inspirationStyleOptions = ['随机', '写实摄影', '电影感', '日系胶片', '二次元', '3D 渲染', '极简设计', '国风']
const quickRefines = ['更写实', '更电影感', '更简洁', '更高级', '更梦幻', '增强光影', '减少元素', '改成竖屏构图', '改成商业海报']

export function PromptAssistantModal({ tasks, uploads, provider, bananaModel, onClose, onUsePrompt, onRefreshUploads, embedded = false }: Props) {
  const [tab, setTab] = useState<Tab>('text')
  const [idea, setIdea] = useState('')
  const [style, setStyle] = useState('auto')
  const [ratio, setRatio] = useState('auto')
  const [applyProvider, setApplyProvider] = useState<ModelProvider>(provider || 'image-2')
  const [applyBananaModel, setApplyBananaModel] = useState(bananaModel || DEFAULT_BANANA_MODEL)
  const [sourceType, setSourceType] = useState<'upload' | 'result'>('upload')
  const [uploadId, setUploadId] = useState('')
  const [resultKey, setResultKey] = useState('')
  const [records, setRecords] = useState<PromptRecord[]>([])
  const [sessions, setSessions] = useState<PromptSession[]>([])
  const [activeRecord, setActiveRecord] = useState<PromptRecord | null>(null)
  const [activeSession, setActiveSession] = useState<PromptSession | null>(null)
  const [activeVersionId, setActiveVersionId] = useState('')
  const [refineText, setRefineText] = useState('')
  const [inspirationCategory, setInspirationCategory] = useState('随机')
  const [inspirationMood, setInspirationMood] = useState('随机')
  const [inspirationStyle, setInspirationStyle] = useState('随机')
  const [inspirationSeed, setInspirationSeed] = useState('')
  const [inspirationRatio, setInspirationRatio] = useState('auto')
  const [ideas, setIdeas] = useState<InspirationIdea[]>([])
  const [loading, setLoading] = useState(false)
  const [message, setMessage] = useState('')
  const [error, setError] = useState('')

  const resultOptions = useMemo(() => tasks.flatMap((task) => task.results
    .filter((result) => result.ok && result.imageUrl)
    .map((result) => ({
      key: `${task.id}:${result.index}`,
      taskId: task.id,
      index: result.index,
      label: `${task.prompt.slice(0, 34) || task.id} · 第 ${result.index + 1} 张`,
      url: result.imageUrl!,
    }))), [tasks])

  const activeVersion = useMemo(() => pickVersion(activeSession, activeVersionId), [activeSession, activeVersionId])

  useEffect(() => {
    void refreshHistory()
  }, [])

  useEffect(() => {
    if (!uploadId && uploads[0]) setUploadId(uploads[0].id)
  }, [uploads, uploadId])

  useEffect(() => {
    if (!resultKey && resultOptions[0]) setResultKey(resultOptions[0].key)
  }, [resultOptions, resultKey])

  useEffect(() => {
    setApplyProvider(provider || 'image-2')
    setApplyBananaModel(bananaModel || DEFAULT_BANANA_MODEL)
  }, [provider, bananaModel])

  async function refreshHistory() {
    try {
      const [nextRecords, nextSessions] = await Promise.all([listPromptHistory(), listPromptSessions()])
      setRecords(nextRecords)
      setSessions(nextSessions)
    } catch {
      // 历史不是主链路，失败时不阻塞弹窗。
    }
  }



  function selectedModel(nextProvider = applyProvider) {
    return nextProvider === BANANA_PROVIDER ? applyBananaModel : DEFAULT_IMAGE2_MODEL
  }

  async function loadSession(id: string) {
    const session = await getPromptSession(id)
    setActiveSession(session)
    setActiveVersionId(session.activeVersionId)
    return session
  }

  async function generateTextPrompt() {
    setError('')
    setMessage('')
    if (!idea.trim()) {
      setError('先输入一句简单想法')
      return
    }
    setLoading(true)
    try {
      const record = await textToPrompt({
        input: idea,
        style,
        ratio,
        language: 'zh',
        target: '通用图片模型',
      })
      setActiveRecord(record)
      if (record.sessionId) await loadSession(record.sessionId)
      setMessage('文字提示词已生成，可以继续对话修改')
      await refreshHistory()
    } catch (err) {
      setError(formatError(err, '生成失败'))
    } finally {
      setLoading(false)
    }
  }

  async function generateImagePrompt() {
    setError('')
    setMessage('')
    const source = selectedSource()
    if (!source) {
      setError(sourceType === 'upload' ? '请先选择或上传参考图' : '请先选择一张历史结果图')
      return
    }
    setLoading(true)
    try {
      const record = await imageToPrompt({ source, language: 'zh', target: '通用图片模型' })
      setActiveRecord(record)
      if (record.sessionId) await loadSession(record.sessionId)
      setMessage('图片还原提示词已生成，可以继续对话修改')
      await refreshHistory()
    } catch (err) {
      setError(formatError(err, '图片分析失败'))
    } finally {
      setLoading(false)
    }
  }

  async function handleLocalUpload(files: FileList | null) {
    if (!files?.length) return
    setError('')
    setMessage('')
    setLoading(true)
    try {
      const created = await uploadReferenceImages(Array.from(files))
      if (created[0]) setUploadId(created[0].id)
      await onRefreshUploads()
      setMessage('参考图已上传，可直接还原提示词')
    } catch (err) {
      setError(formatError(err, '上传失败'))
    } finally {
      setLoading(false)
    }
  }

  async function makeIdeas() {
    setError('')
    setMessage('')
    setLoading(true)
    try {
      const nextIdeas = await generateInspirationIdeas({
        category: normalizeRandom(inspirationCategory),
        mood: normalizeRandom(inspirationMood),
        style: normalizeRandom(inspirationStyle),
        target: `${providerLabel(applyProvider)} / ${selectedModel()}`,
        count: 6,
        seed: inspirationSeed,
      })
      setIdeas(nextIdeas)
      setMessage('灵感已生成，选择一个即可扩写成提示词')
    } catch (err) {
      setError(formatError(err, '生成灵感失败'))
    } finally {
      setLoading(false)
    }
  }

  async function expandIdea(ideaItem: InspirationIdea) {
    setError('')
    setMessage('')
    setLoading(true)
    try {
      const session = await expandInspirationIdea({
        idea: ideaItem,
        ratio: inspirationRatio,
        target: `${providerLabel(applyProvider)} / ${selectedModel()}`,
        provider: applyProvider,
        model: selectedModel(),
      })
      setActiveRecord(null)
      setActiveSession(session)
      setActiveVersionId(session.activeVersionId)
      setMessage('灵感已扩写成提示词，可以继续对话修改')
      await refreshHistory()
    } catch (err) {
      setError(formatError(err, '扩写灵感失败'))
    } finally {
      setLoading(false)
    }
  }

  async function refineActiveSession() {
    setError('')
    setMessage('')
    if (!activeSession) {
      setError('请先生成或选择一个提示词会话')
      return
    }
    if (!refineText.trim()) {
      setError('请输入修改要求')
      return
    }
    setLoading(true)
    try {
      const session = await refinePromptSession(activeSession.id, {
        message: refineText,
        currentVersionId: activeVersion?.id || activeSession.activeVersionId,
        provider: applyProvider,
        model: selectedModel(),
      })
      setActiveSession(session)
      setActiveVersionId(session.activeVersionId)
      setRefineText('')
      setMessage('提示词已按要求生成新版本')
      await refreshHistory()
    } catch (err) {
      setError(formatError(err, '修改失败'))
    } finally {
      setLoading(false)
    }
  }

  async function copyPrompt(prompt: string) {
    await navigator.clipboard.writeText(prompt)
    setMessage('提示词已复制')
  }

  async function deleteRecord(id: string) {
    await deletePromptHistory(id)
    await refreshHistory()
    setActiveRecord((current) => current?.id === id ? null : current)
    setMessage('提示词历史已删除')
  }

  async function deleteSessionItem(id: string) {
    await deletePromptSession(id)
    await refreshHistory()
    setActiveSession((current) => current?.id === id ? null : current)
    setMessage('提示词会话已删除')
  }

  function selectedSource() {
    if (sourceType === 'upload') {
      return uploadId ? { type: 'upload' as const, uploadId } : null
    }
    const selected = resultOptions.find((item) => item.key === resultKey)
    return selected ? { type: 'result' as const, taskId: selected.taskId, index: selected.index } : null
  }

  function openRecord(record: PromptRecord) {
    setActiveRecord(record)
    if (record.sessionId) {
      void loadSession(record.sessionId).catch(() => {
        setActiveSession(null)
        setActiveVersionId('')
      })
    } else {
      setActiveSession(null)
      setActiveVersionId('')
    }
  }

  function openSession(session: PromptSession) {
    setActiveRecord(null)
    setActiveSession(session)
    setActiveVersionId(session.activeVersionId)
  }

  const content = (
      <section className={`prompt-assistant ${embedded ? 'prompt-assistant-inline' : ''}`} role={embedded ? undefined : 'dialog'} aria-modal={embedded ? undefined : true} aria-label="提示词助手" onMouseDown={(event) => event.stopPropagation()}>
        <header className="prompt-assistant-header">
          <div>
            <p className="eyebrow">Prompt Assistant</p>
            <h2>提示词助手</h2>
            <p>调用 gpt-5.5 生成、还原、找灵感，也可以像聊天一样继续改提示词。</p>
          </div>
          {embedded ? null : <button type="button" onClick={onClose}>关闭</button>}
        </header>

        <div className="prompt-tabs" role="tablist" aria-label="提示词工具">
          <button type="button" className={tab === 'text' ? 'active' : ''} onClick={() => setTab('text')}>文字生成图片提示词</button>
          <button type="button" className={tab === 'image' ? 'active' : ''} onClick={() => setTab('image')}>图片还原提示词</button>
          <button type="button" className={tab === 'inspiration' ? 'active' : ''} onClick={() => setTab('inspiration')}>灵感模式</button>
          <button type="button" className={tab === 'history' ? 'active' : ''} onClick={() => setTab('history')}>历史/会话</button>
        </div>

        <div className={`prompt-assistant-body ${tab === 'inspiration' ? 'is-inspiration' : ''}`}>
          {tab === 'text' ? (
            <section className="prompt-tool-panel">
              <label>
                <span>一句话想法</span>
                <textarea value={idea} onChange={(event) => setIdea(event.target.value)} placeholder="例如：雨夜东京街头的赛博朋克少女" rows={4} />
              </label>
              <div className="prompt-tool-grid">
                <label>
                  <span>风格</span>
                  <select value={style} onChange={(event) => setStyle(event.target.value)}>
                    {styleOptions.map((item) => <option key={item.value} value={item.value}>{item.label}</option>)}
                  </select>
                </label>
                <label>
                  <span>比例</span>
                  <select value={ratio} onChange={(event) => setRatio(event.target.value)}>
                    {ratioOptions.map((item) => <option key={item} value={item}>{item === 'auto' ? '自动' : item}</option>)}
                  </select>
                </label>
              </div>
              <button type="button" className="primary" disabled={loading} onClick={generateTextPrompt}>{loading ? '生成中...' : '生成专业提示词'}</button>
            </section>
          ) : null}

          {tab === 'image' ? (
            <section className="prompt-tool-panel">
              <div className="prompt-source-tabs">
                <button type="button" className={sourceType === 'upload' ? 'active' : ''} onClick={() => setSourceType('upload')}>参考图</button>
                <button type="button" className={sourceType === 'result' ? 'active' : ''} onClick={() => setSourceType('result')}>历史结果图</button>
              </div>
              {sourceType === 'upload' ? (
                <>
                  <label>
                    <span>上传新参考图</span>
                    <input type="file" accept="image/png,image/jpeg,image/webp" onChange={(event) => void handleLocalUpload(event.target.files)} />
                  </label>
                  <label>
                    <span>选择已上传参考图</span>
                    <select value={uploadId} onChange={(event) => setUploadId(event.target.value)}>
                      <option value="">请选择</option>
                      {uploads.map((item) => <option key={item.id} value={item.id}>{item.originalName} · {Math.round(item.size / 1024)}KB</option>)}
                    </select>
                  </label>
                </>
              ) : (
                <label>
                  <span>选择历史结果图</span>
                  <select value={resultKey} onChange={(event) => setResultKey(event.target.value)}>
                    <option value="">请选择</option>
                    {resultOptions.map((item) => <option key={item.key} value={item.key}>{item.label}</option>)}
                  </select>
                </label>
              )}
              <button type="button" className="primary" disabled={loading} onClick={generateImagePrompt}>{loading ? '分析中...' : '还原图片提示词'}</button>
            </section>
          ) : null}

          {tab === 'inspiration' ? (
            <section className="prompt-tool-panel inspiration-panel">
              <div className="prompt-tool-grid">
                <label>
                  <span>类别</span>
                  <select value={inspirationCategory} onChange={(event) => setInspirationCategory(event.target.value)}>
                    {categoryOptions.map((item) => <option key={item} value={item}>{item}</option>)}
                  </select>
                </label>
                <label>
                  <span>情绪</span>
                  <select value={inspirationMood} onChange={(event) => setInspirationMood(event.target.value)}>
                    {moodOptions.map((item) => <option key={item} value={item}>{item}</option>)}
                  </select>
                </label>
                <label>
                  <span>风格</span>
                  <select value={inspirationStyle} onChange={(event) => setInspirationStyle(event.target.value)}>
                    {inspirationStyleOptions.map((item) => <option key={item} value={item}>{item}</option>)}
                  </select>
                </label>
                <label>
                  <span>比例</span>
                  <select value={inspirationRatio} onChange={(event) => setInspirationRatio(event.target.value)}>
                    {ratioOptions.map((item) => <option key={item} value={item}>{item === 'auto' ? '自动' : item}</option>)}
                  </select>
                </label>
              </div>
              <label>
                <span>补充方向（可选）</span>
                <textarea value={inspirationSeed} onChange={(event) => setInspirationSeed(event.target.value)} placeholder="例如：想做手机壁纸，干净一点，有孤独感" rows={3} />
              </label>
              <button type="button" className="primary" disabled={loading} onClick={makeIdeas}>{loading ? '生成中...' : '生成 6 个灵感'}</button>
              {ideas.length ? (
                <div className="prompt-idea-head">
                  <strong>已生成 {ideas.length} 个灵感</strong>
                  <span>点击「扩写成提示词」后，右侧会生成可直接应用的完整提示词。</span>
                </div>
              ) : null}
              <div className="prompt-idea-grid">
                {ideas.map((item) => (
                  <article key={item.id || item.title} className="prompt-idea-item">
                    <strong>{item.title}</strong>
                    <p>{item.summary}</p>
                    <div className="prompt-chips">{item.tags?.map((tag) => <span key={tag}>{tag}</span>)}</div>
                    <button type="button" onClick={() => void expandIdea(item)} disabled={loading}>扩写成提示词</button>
                  </article>
                ))}
              </div>
            </section>
          ) : null}

          {tab === 'history' ? (
            <section className="prompt-history-list">
              {!sessions.length && !records.length ? <div className="prompt-empty">还没有提示词历史</div> : null}
              {sessions.length ? <p className="prompt-history-heading">可继续修改的会话</p> : null}
              {sessions.map((session) => (
                <article key={session.id} className="prompt-history-item" onClick={() => openSession(session)}>
                  <strong>{kindLabel(session.kind)} · {session.title}</strong>
                  <p>{pickVersion(session, session.activeVersionId)?.prompt || session.seed}</p>
                  <footer>
                    <span>{session.versions.length} 个版本</span>
                    <button type="button" onClick={(event) => { event.stopPropagation(); void deleteSessionItem(session.id) }}>删除</button>
                  </footer>
                </article>
              ))}
              {records.length ? <p className="prompt-history-heading">生成记录</p> : null}
              {records.map((record) => (
                <article key={record.id} className="prompt-history-item" onClick={() => openRecord(record)}>
                  <strong>{record.mode === 'image-to-prompt' ? '图片还原' : '文字扩写'}</strong>
                  <p>{record.flatPrompt}</p>
                  <footer>
                    <span>{record.model}</span>
                    <button type="button" onClick={(event) => { event.stopPropagation(); void deleteRecord(record.id) }}>删除</button>
                  </footer>
                </article>
              ))}
            </section>
          ) : null}

          <PromptResult
            record={activeRecord}
            session={activeSession}
            activeVersion={activeVersion}
            activeVersionId={activeVersionId}
            provider={applyProvider}
            bananaModel={applyBananaModel}
            refineText={refineText}
            loading={loading}
            onVersionChange={setActiveVersionId}
            onProviderChange={setApplyProvider}
            onBananaModelChange={setApplyBananaModel}
            onRefineTextChange={setRefineText}
            onQuickRefine={(text) => setRefineText((current) => current ? `${current}，${text}` : text)}
            onRefine={() => void refineActiveSession()}
            onCopy={(prompt) => void copyPrompt(prompt)}
            onUse={(prompt, options) => { onUsePrompt(prompt, options); setMessage(`已填入生成页，并切到 ${providerLabel(options.provider)}`) }}
          />
        </div>

        {message ? <div className="ok">{message}</div> : null}
        {error ? <div className="error">{error}</div> : null}
      </section>
  )

  if (embedded) return content
  return (
    <div className="prompt-assistant-mask" role="presentation" onMouseDown={onClose}>
      {content}
    </div>
  )
}
function PromptResult({
  record,
  session,
  activeVersion,
  activeVersionId,
  provider,
  bananaModel,
  refineText,
  loading,
  onVersionChange,
  onProviderChange,
  onBananaModelChange,
  onRefineTextChange,
  onQuickRefine,
  onRefine,
  onCopy,
  onUse,
}: {
  record: PromptRecord | null
  session: PromptSession | null
  activeVersion: PromptVersion | null
  activeVersionId: string
  provider: ModelProvider
  bananaModel: string
  refineText: string
  loading: boolean
  onVersionChange: (id: string) => void
  onProviderChange: (provider: ModelProvider) => void
  onBananaModelChange: (model: string) => void
  onRefineTextChange: (value: string) => void
  onQuickRefine: (value: string) => void
  onRefine: () => void
  onCopy: (prompt: string) => void
  onUse: (prompt: string, options: { provider: ModelProvider; model: string }) => void
}) {
  const model = provider === BANANA_PROVIDER ? bananaModel : DEFAULT_IMAGE2_MODEL
  const prompt = activeVersion?.prompt || record?.flatPrompt || ''
  const negativePrompt = activeVersion?.negativePrompt || record?.negativePrompt || ''
  const mustKeep = activeVersion?.mustKeep?.length ? activeVersion.mustKeep : record?.mustKeep
  const title = session ? `${kindLabel(session.kind)} · ${session.title}` : record ? (record.mode === 'image-to-prompt' ? '图片还原提示词' : '文字生成图片提示词') : '结果预览'
  const elapsedMs = activeVersion?.elapsedMs || record?.elapsedMs || 0
  const promptModel = activeVersion?.model || record?.model || 'gpt-5.5'

  if (!prompt) {
    return (
      <aside className="prompt-result empty">
        <strong>结果预览</strong>
        <span>生成后会在这里显示，可继续对话修改、复制或填入主输入框。</span>
      </aside>
    )
  }
  return (
    <aside className="prompt-result">
      <div className="prompt-result-title">
        <div className="prompt-result-title-main">
          <strong>{title}</strong>
          <span>{promptModel} · {elapsedMs ? `${(elapsedMs / 1000).toFixed(1)}s` : '会话'}</span>
        </div>
        <div className="prompt-result-title-actions">
          <button type="button" onClick={() => onCopy(prompt)}>复制</button>
          <button type="button" className="primary" onClick={() => onUse(prompt, { provider, model })}>应用此提示词</button>
        </div>
      </div>

      {session?.versions.length ? (
        <section className="prompt-version-list" aria-label="提示词版本">
          {session.versions.map((version) => (
            <button key={version.id} type="button" className={version.id === activeVersionId ? 'active' : ''} onClick={() => onVersionChange(version.id)}>
              V{version.index}
            </button>
          ))}
        </section>
      ) : null}

      <div className="prompt-result-actions prompt-result-actions-top">
        <button type="button" onClick={() => onCopy(prompt)}>复制提示词</button>
        <button type="button" className="primary" onClick={() => onUse(prompt, { provider, model })}>填入并使用该模型</button>
      </div>

      <label>
        <span>正向提示词</span>
        <textarea value={prompt} readOnly rows={8} />
      </label>
      {negativePrompt ? (
        <label>
          <span>负面提示词</span>
          <textarea value={negativePrompt} readOnly rows={3} />
        </label>
      ) : null}
      {mustKeep?.length ? (
        <div className="prompt-chips">
          {mustKeep.map((item) => <span key={item}>{item}</span>)}
        </div>
      ) : null}
      {activeVersion?.notes ? <p className="prompt-version-note">{activeVersion.notes}</p> : null}

      {session ? (
        <section className="prompt-refine-box">
          <div className="section-title">
            <span>继续对话修改</span>
            <small>会生成新版本，不覆盖旧版本</small>
          </div>
          <div className="prompt-quick-refines">
            {quickRefines.map((item) => <button key={item} type="button" onClick={() => onQuickRefine(item)}>{item}</button>)}
          </div>
          <textarea value={refineText} onChange={(event) => onRefineTextChange(event.target.value)} placeholder="例如：更写实一点，减少动漫感；主体换成猫；保留雨夜和霓虹灯" rows={3} />
          <button type="button" className="primary" disabled={loading} onClick={onRefine}>{loading ? '修改中...' : '生成新版本'}</button>
          {session.messages.length ? (
            <details className="prompt-chat-log">
              <summary>查看最近对话</summary>
              {session.messages.slice(-8).map((item) => <p key={item.id}><b>{item.role === 'user' ? '你' : '助手'}：</b>{item.content}</p>)}
            </details>
          ) : null}
        </section>
      ) : null}

      <section className="prompt-apply-model" aria-label="选择应用模型">
        <div className="section-title">
          <span>应用到模型</span>
          <small>生成完再选择</small>
        </div>
        <div className="mode-tabs provider-tabs">
          <button type="button" className={provider === 'image-2' ? 'active' : ''} onClick={() => onProviderChange('image-2')}>Image-2</button>
          <button type="button" className={provider === BANANA_PROVIDER ? 'active' : ''} onClick={() => onProviderChange(BANANA_PROVIDER)}>Banana</button>
        </div>
        {provider === BANANA_PROVIDER ? (
          <label>
            <span>Banana 模型 ID</span>
            <select value={bananaModel} onChange={(event) => onBananaModelChange(event.target.value)}>
              {BANANA_MODEL_OPTIONS.map((item) => <option key={item.id} value={item.id}>{item.label} · {item.id}</option>)}
            </select>
          </label>
        ) : (
          <div className="status-line">模型：{DEFAULT_IMAGE2_MODEL}</div>
        )}
      </section>
      <div className="prompt-result-actions">
        <button type="button" onClick={() => onCopy(prompt)}>复制提示词</button>
        <button type="button" className="primary" onClick={() => onUse(prompt, { provider, model })}>填入并使用该模型</button>
      </div>
    </aside>
  )
}


function pickVersion(session: PromptSession | null, versionId: string) {
  if (!session?.versions.length) return null
  return session.versions.find((item) => item.id === versionId) || session.versions.find((item) => item.id === session.activeVersionId) || session.versions[session.versions.length - 1]
}

function kindLabel(kind?: string) {
  if (kind === 'image') return '图片还原'
  if (kind === 'inspiration') return '灵感扩写'
  if (kind === 'manual') return '手动会话'
  return '文字扩写'
}

function normalizeRandom(value: string) {
  return value === '随机' ? '' : value
}
