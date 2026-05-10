import { useEffect, useMemo, useState } from 'react'
import { formatError } from '../api/client'
import { deletePromptHistory, imageToPrompt, listPromptHistory, textToPrompt } from '../api/promptTools'
import { uploadReferenceImages } from '../api/uploads'
import type { ModelProvider, PromptRecord, ReferenceUpload, Task } from '../types'
import { BANANA_MODEL_OPTIONS, BANANA_PROVIDER, DEFAULT_BANANA_MODEL, DEFAULT_IMAGE2_MODEL, providerLabel } from '../lib/models'

type Tab = 'text' | 'image' | 'history'

type Props = {
  tasks: Task[]
  uploads: ReferenceUpload[]
  provider: ModelProvider
  bananaModel: string
  onClose: () => void
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

export function PromptAssistantModal({ tasks, uploads, provider, bananaModel, onClose, onUsePrompt, onRefreshUploads }: Props) {
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
  const [activeRecord, setActiveRecord] = useState<PromptRecord | null>(null)
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
      setRecords(await listPromptHistory())
    } catch {
      // 历史不是主链路，失败时不阻塞弹窗。
    }
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
        target: 'image-2',
      })
      setActiveRecord(record)
      setMessage('文字提示词已生成')
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
      const record = await imageToPrompt({ source, language: 'zh', target: 'image-2' })
      setActiveRecord(record)
      setMessage('图片还原提示词已生成')
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

  function selectedSource() {
    if (sourceType === 'upload') {
      return uploadId ? { type: 'upload' as const, uploadId } : null
    }
    const selected = resultOptions.find((item) => item.key === resultKey)
    return selected ? { type: 'result' as const, taskId: selected.taskId, index: selected.index } : null
  }

  return (
    <div className="prompt-assistant-mask" role="presentation" onMouseDown={onClose}>
      <section className="prompt-assistant" role="dialog" aria-modal="true" aria-label="提示词助手" onMouseDown={(event) => event.stopPropagation()}>
        <header className="prompt-assistant-header">
          <div>
            <p className="eyebrow">Prompt Assistant</p>
            <h2>提示词助手</h2>
            <p>全局调用 gpt-5.5 生成提示词，生成后可选择填入 Image-2 或 Banana 模型。</p>
          </div>
          <button type="button" onClick={onClose}>关闭</button>
        </header>

        <div className="prompt-tabs" role="tablist" aria-label="提示词工具">
          <button type="button" className={tab === 'text' ? 'active' : ''} onClick={() => setTab('text')}>文字生成图片提示词</button>
          <button type="button" className={tab === 'image' ? 'active' : ''} onClick={() => setTab('image')}>图片还原提示词</button>
          <button type="button" className={tab === 'history' ? 'active' : ''} onClick={() => setTab('history')}>历史</button>
        </div>

        <div className="prompt-assistant-body">
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

          {tab === 'history' ? (
            <section className="prompt-history-list">
              {!records.length ? <div className="prompt-empty">还没有提示词历史</div> : records.map((record) => (
                <article key={record.id} className="prompt-history-item" onClick={() => setActiveRecord(record)}>
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
            provider={applyProvider}
            bananaModel={applyBananaModel}
            onProviderChange={setApplyProvider}
            onBananaModelChange={setApplyBananaModel}
            onCopy={(prompt) => void copyPrompt(prompt)}
            onUse={(prompt, options) => { onUsePrompt(prompt, options); setMessage(`已填入主提示词输入框，并切到 ${providerLabel(options.provider)}`) }}
          />
        </div>

        {message ? <div className="ok">{message}</div> : null}
        {error ? <div className="error">{error}</div> : null}
      </section>
    </div>
  )
}

function PromptResult({
  record,
  provider,
  bananaModel,
  onProviderChange,
  onBananaModelChange,
  onCopy,
  onUse,
}: {
  record: PromptRecord | null
  provider: ModelProvider
  bananaModel: string
  onProviderChange: (provider: ModelProvider) => void
  onBananaModelChange: (model: string) => void
  onCopy: (prompt: string) => void
  onUse: (prompt: string, options: { provider: ModelProvider; model: string }) => void
}) {
  const model = provider === BANANA_PROVIDER ? bananaModel : DEFAULT_IMAGE2_MODEL
  if (!record) {
    return (
      <aside className="prompt-result empty">
        <strong>结果预览</strong>
        <span>生成后会在这里显示，可复制或填入主输入框。</span>
      </aside>
    )
  }
  return (
    <aside className="prompt-result">
      <div className="prompt-result-title">
        <strong>{record.mode === 'image-to-prompt' ? '图片还原提示词' : '文字生成图片提示词'}</strong>
        <span>{record.model} · {(record.elapsedMs / 1000).toFixed(1)}s</span>
      </div>
      <label>
        <span>正向提示词</span>
        <textarea value={record.flatPrompt} readOnly rows={8} />
      </label>
      {record.negativePrompt ? (
        <label>
          <span>负面提示词</span>
          <textarea value={record.negativePrompt} readOnly rows={3} />
        </label>
      ) : null}
      {record.mustKeep?.length ? (
        <div className="prompt-chips">
          {record.mustKeep.map((item) => <span key={item}>{item}</span>)}
        </div>
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
        <button type="button" onClick={() => onCopy(record.flatPrompt)}>复制提示词</button>
        <button type="button" className="primary" onClick={() => onUse(record.flatPrompt, { provider, model })}>填入并使用该模型</button>
      </div>
    </aside>
  )
}
