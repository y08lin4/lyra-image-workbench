import { type ClipboardEvent as ReactClipboardEvent, type DragEvent as ReactDragEvent, type MouseEvent as ReactMouseEvent, useEffect, useMemo, useRef, useState } from 'react'
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
import { deleteReferenceUpload, uploadReferenceImages } from '../api/uploads'
import type { InspirationIdea, ModelProvider, PromptRecord, PromptSession, ReferenceUpload, Task } from '../types'
import { BANANA_PROVIDER, DEFAULT_BANANA_MODEL, DEFAULT_IMAGE2_MODEL, providerLabel } from '../lib/models'
import { PromptResultPanel } from './promptAssistant/PromptResultPanel'
import {
  categoryOptions,
  emptyInspirationAnswers,
  emptyInspirationSkipped,
  inspirationSteps,
  inspirationStyleOptions,
  kindLabel,
  moodOptions,
  promptTabs,
  ratioOptions,
  styleOptions,
  type InspirationSkipState,
  type InspirationStepId,
  type Tab,
} from './promptAssistant/constants'
import {
  buildInspirationBrief,
  buildInspirationMessages,
  countInspirationProgress,
  isInspirationStepComplete,
  normalizeRandom,
} from './promptAssistant/inspiration'
import { buildPromptOptimizationTarget } from './promptAssistant/templates'


type Props = {
  tasks: Task[]
  uploads: ReferenceUpload[]
  provider: ModelProvider
  bananaModel: string
  onClose: () => void
  embedded?: boolean
  onUsePrompt: (prompt: string, options: { provider: ModelProvider; model: string; ratio?: string }) => void
  onRefreshUploads: () => Promise<void>
}

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
  const [inspirationAnswers, setInspirationAnswers] = useState<Record<InspirationStepId, string>>({ ...emptyInspirationAnswers })
  const [inspirationSkipped, setInspirationSkipped] = useState<InspirationSkipState>({ ...emptyInspirationSkipped })
  const [inspirationDraft, setInspirationDraft] = useState('')
  const [inspirationStepIndex, setInspirationStepIndex] = useState(0)
  const [ideas, setIdeas] = useState<InspirationIdea[]>([])
  const [loading, setLoading] = useState(false)
  const [imageDropActive, setImageDropActive] = useState(false)
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
  const visibleRecord = tab === 'image' && activeRecord?.mode !== 'image-to-prompt' ? null : activeRecord
  const visibleSession = tab === 'image' && activeSession?.kind !== 'image' ? null : activeSession
  const visibleActiveVersion = visibleSession ? activeVersion : null
  const visibleActiveVersionId = visibleSession ? activeVersionId : ''
  const hasPromptResult = Boolean(visibleRecord || visibleSession)
  const resultScrollAnchorRef = useRef<HTMLDivElement | null>(null)
  const imageInputRef = useRef<HTMLInputElement | null>(null)
  const resultScrollKey = visibleActiveVersion?.id || visibleSession?.id || visibleRecord?.id || ''
  const inspirationCurrentStep = inspirationSteps[inspirationStepIndex] || inspirationSteps[0]
  const inspirationCurrentStepSkipped = Boolean(inspirationSkipped[inspirationCurrentStep.id])
  const inspirationCompletedCount = countInspirationProgress(inspirationAnswers, inspirationSkipped)
  const inspirationComplete = inspirationCompletedCount === inspirationSteps.length
  const inspirationProgress = `${inspirationCompletedCount}/${inspirationSteps.length}`
  const inspirationMessages = useMemo(() => buildInspirationMessages(inspirationAnswers, inspirationStepIndex, inspirationSkipped), [inspirationAnswers, inspirationSkipped, inspirationStepIndex])
  const textPromptInputCollapsed = tab === 'text' && hasPromptResult
  const imagePreviewUrl = sourceType === 'upload' && uploadId ? `/api/uploads/reference/${uploadId}/image` : ''

  useEffect(() => {
    if (!resultScrollKey || typeof window === 'undefined') return
    if (!window.matchMedia('(max-width: 980px)').matches) return
    window.requestAnimationFrame(() => {
      const prefersReducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches
      resultScrollAnchorRef.current?.scrollIntoView({ behavior: prefersReducedMotion ? 'auto' : 'smooth', block: 'start' })
    })
  }, [resultScrollKey])

  useEffect(() => {
    void refreshHistory()
  }, [])


  useEffect(() => {
    if (!resultKey && resultOptions[0]) setResultKey(resultOptions[0].key)
  }, [resultOptions, resultKey])

  useEffect(() => {
    setApplyProvider(provider || 'image-2')
    setApplyBananaModel(bananaModel || DEFAULT_BANANA_MODEL)
  }, [provider, bananaModel])
  useEffect(() => {
    if (tab !== 'image') return
    const handlePaste = (event: ClipboardEvent) => {
      const files = Array.from(event.clipboardData?.files || []).filter(isSupportedImageFile)
      if (!files.length) return
      event.preventDefault()
      void uploadImageFiles(files, '粘贴的图片')
    }
    window.addEventListener('paste', handlePaste)
    return () => window.removeEventListener('paste', handlePaste)
  }, [tab])

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

  function inspirationTarget() {
    return `${providerLabel(applyProvider)} / ${selectedModel()}`
  }

  function inspirationDraftForStep(index: number) {
    const stepItem = inspirationSteps[index]
    if (!stepItem || inspirationSkipped[stepItem.id]) return ''
    return inspirationAnswers[stepItem.id] || ''
  }

  function moveToInspirationStep(index: number) {
    const nextIndex = Math.max(0, Math.min(index, inspirationSteps.length - 1))
    setInspirationStepIndex(nextIndex)
    setInspirationDraft(inspirationDraftForStep(nextIndex))
  }

  function selectInspirationStep(index: number) {
    moveToInspirationStep(index)
  }

  function moveToNextInspirationStep() {
    if (inspirationStepIndex < inspirationSteps.length - 1) {
      moveToInspirationStep(inspirationStepIndex + 1)
      return
    }
    setInspirationDraft('')
    setMessage('追问信息已整理，可以生成完整提示词')
  }

  function updateInspirationDraft(value: string) {
    setInspirationDraft(value)
    if (!inspirationCurrentStepSkipped) return
    setInspirationSkipped((current) => ({ ...current, [inspirationCurrentStep.id]: false }))
  }

  function appendInspirationQuick(value: string) {
    if (inspirationCurrentStepSkipped) {
      setInspirationSkipped((current) => ({ ...current, [inspirationCurrentStep.id]: false }))
    }
    setInspirationDraft((current) => current.trim() ? `${current.trim()}，${value}` : value)
  }

  function submitInspirationAnswer() {
    const value = inspirationDraft.trim()
    if (!value) {
      setError(`先回答「${inspirationCurrentStep.label}」，或点“跳过此项”`)
      return
    }
    setError('')
    setMessage('')
    setInspirationAnswers((current) => ({ ...current, [inspirationCurrentStep.id]: value }))
    setInspirationSkipped((current) => ({ ...current, [inspirationCurrentStep.id]: false }))
    moveToNextInspirationStep()
  }

  function skipInspirationAnswer() {
    setError('')
    setMessage('')
    setInspirationAnswers((current) => ({ ...current, [inspirationCurrentStep.id]: '' }))
    setInspirationSkipped((current) => ({ ...current, [inspirationCurrentStep.id]: true }))
    moveToNextInspirationStep()
  }

  function resetInspirationConversation() {
    setInspirationAnswers({ ...emptyInspirationAnswers })
    setInspirationSkipped({ ...emptyInspirationSkipped })
    setInspirationDraft('')
    setInspirationStepIndex(0)
    setIdeas([])
    setMessage('已重开灵感对话')
    setError('')
  }

  function startNewTextOptimization() {
    setActiveRecord(null)
    setActiveSession(null)
    setActiveVersionId('')
    setRefineText('')
    setMessage('可以输入新的提示词优化想法')
    setError('')
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
        target: buildPromptOptimizationTarget(),
      })
      setActiveRecord(record)
      setActiveSession(null)
      setActiveVersionId('')
      if (record.sessionId) await loadSession(record.sessionId)
      setMessage('提示词优化结果已生成，可以继续对话修改')
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
      setActiveSession(null)
      setActiveVersionId('')
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
    await uploadImageFiles(files ? Array.from(files) : [], '选择的图片')
    if (imageInputRef.current) imageInputRef.current.value = ''
  }

  async function uploadImageFiles(files: File[], label = '图片') {
    const imageFiles = files.filter(isSupportedImageFile)
    if (!imageFiles.length) {
      setError('请粘贴或拖入 PNG、JPG、WEBP 图片')
      return
    }
    setError('')
    setMessage('')
    setSourceType('upload')
    setLoading(true)
    try {
      const created = await uploadReferenceImages(imageFiles.slice(0, 1))
      if (created[0]) setUploadId(created[0].id)
      await onRefreshUploads()
      setMessage(`${label}已载入，可以还原提示词`)
    } catch (err) {
      setError(formatError(err, '上传失败'))
    } finally {
      setLoading(false)
      setImageDropActive(false)
    }
  }

  async function clearReferenceImage() {
    if (!uploadId) return
    const currentUploadId = uploadId
    setError('')
    setMessage('')
    setLoading(true)
    try {
      await deleteReferenceUpload(currentUploadId)
      setUploadId('')
      if (imageInputRef.current) imageInputRef.current.value = ''
      await onRefreshUploads()
      setMessage('参考图已删除')
    } catch (err) {
      setError(formatError(err, '删除参考图失败'))
    } finally {
      setLoading(false)
    }
  }

  function handleImagePanelClick(event: ReactMouseEvent<HTMLElement>) {
    const target = event.target as HTMLElement | null
    if (target?.closest('button, input, select, textarea, a, .prompt-image-dropzone')) return
    imageInputRef.current?.click()
  }

  function handleImageDragEnter(event: ReactDragEvent<HTMLElement>) {
    if (!hasImageTransfer(event.dataTransfer)) return
    event.preventDefault()
    setImageDropActive(true)
  }

  function handleImageDragOver(event: ReactDragEvent<HTMLElement>) {
    if (!hasImageTransfer(event.dataTransfer)) return
    event.preventDefault()
    event.dataTransfer.dropEffect = 'copy'
    setImageDropActive(true)
  }

  function handleImageDragLeave(event: ReactDragEvent<HTMLElement>) {
    if (event.currentTarget.contains(event.relatedTarget as Node | null)) return
    setImageDropActive(false)
  }

  function handleImageDrop(event: ReactDragEvent<HTMLElement>) {
    if (!hasImageTransfer(event.dataTransfer)) return
    event.preventDefault()
    const files = Array.from(event.dataTransfer.files || [])
    void uploadImageFiles(files, '拖入的图片')
  }

  function handleImagePanelPaste(event: ReactClipboardEvent<HTMLElement>) {
    const files = Array.from(event.clipboardData.files || []).filter(isSupportedImageFile)
    if (!files.length) return
    event.preventDefault()
    void uploadImageFiles(files, '粘贴的图片')
  }

  async function makeIdeas() {
    setError('')
    setMessage('')
    const seed = buildInspirationBrief(inspirationAnswers, inspirationSeed, inspirationSkipped)
    if (!seed.trim()) {
      setError('先回答或跳过灵感对话里的问题')
      return
    }
    setLoading(true)
    try {
      const nextIdeas = await generateInspirationIdeas({
        category: normalizeRandom(inspirationCategory),
        mood: normalizeRandom(inspirationMood),
        style: normalizeRandom(inspirationStyle),
        target: inspirationTarget(),
        count: 6,
        seed,
      })
      setIdeas(nextIdeas)
      setMessage('已生成备选灵感，也可以直接生成完整提示词')
    } catch (err) {
      setError(formatError(err, '生成灵感失败'))
    } finally {
      setLoading(false)
    }
  }

  async function generateInspirationPrompt() {
    setError('')
    setMessage('')
    const seed = buildInspirationBrief(inspirationAnswers, inspirationSeed, inspirationSkipped)
    if (!seed.trim()) {
      setError('先回答或跳过灵感对话里的问题')
      return
    }
    if (!inspirationComplete) {
      setError(`继续回答或跳过「${inspirationCurrentStep.label}」后再生成完整提示词`)
      return
    }
    setLoading(true)
    try {
      const nextIdeas = await generateInspirationIdeas({
        category: normalizeRandom(inspirationCategory),
        mood: normalizeRandom(inspirationMood),
        style: normalizeRandom(inspirationStyle),
        target: inspirationTarget(),
        count: 4,
        seed,
      })
      const primaryIdea: InspirationIdea = nextIdeas[0] || {
        id: `guided-${Date.now()}`,
        title: inspirationAnswers.idea.trim().slice(0, 28) || '灵感方案',
        summary: seed,
        tags: ['对话灵感'],
      }
      const session = await expandInspirationIdea({
        idea: primaryIdea,
        ratio: inspirationRatio,
        target: inspirationTarget(),
        provider: applyProvider,
        model: selectedModel(),
      })
      setIdeas(nextIdeas)
      setActiveRecord(null)
      setActiveSession(session)
      setActiveVersionId(session.activeVersionId)
      setMessage('完整提示词已生成，可以继续对话优化')
      await refreshHistory()
    } catch (err) {
      setError(formatError(err, '生成完整提示词失败'))
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
        target: inspirationTarget(),
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
    setActiveSession(null)
    setActiveVersionId('')
    if (record.sessionId) {
      void loadSession(record.sessionId).catch(() => {
        setActiveSession(null)
        setActiveVersionId('')
      })
    }
  }

  function openSession(session: PromptSession) {
    setActiveRecord(null)
    setActiveSession(session)
    setActiveVersionId(session.activeVersionId)
  }

  const content = (
      <section className={`prompt-assistant ${embedded ? 'prompt-assistant-inline' : 'prompt-assistant-modal'}`} role={embedded ? undefined : 'dialog'} aria-modal={embedded ? undefined : true} aria-label="提示词助手" onMouseDown={(event) => event.stopPropagation()}>
        <header className="prompt-assistant-header">
          <div>
            <p className="eyebrow">Prompt Assistant</p>
            <h2>提示词工作台</h2>
            <p>从想法、图片和历史会话组织提示词，生成后可继续对话优化并直接填入生成页。</p>
          </div>
          {embedded ? null : <button type="button" onClick={onClose}>关闭</button>}
        </header>

        <div className="prompt-tabs" role="tablist" aria-label="提示词工具">
          {promptTabs.map((item) => (
            <button
              key={item.id}
              type="button"
              role="tab"
              id={`prompt-tab-${item.id}`}
              aria-selected={tab === item.id}
              aria-controls={`prompt-panel-${item.id}`}
              className={tab === item.id ? 'active' : ''}
              onClick={() => setTab(item.id)}
            >
              {item.label}
            </button>
          ))}
        </div>

        <div className={`prompt-assistant-body is-${tab}${tab === 'inspiration' ? ' is-inspiration' : ''}${hasPromptResult ? ' has-result' : ''}`}>
          {tab === 'text' ? (
            textPromptInputCollapsed ? (
              <section className="prompt-tool-panel prompt-text-collapsed" id="prompt-panel-text" role="tabpanel" aria-labelledby="prompt-tab-text">
                <div>
                  <strong>初始输入已收起</strong>
                  <span>当前结果请在“继续对话修改”里调整；需要新想法时可重新打开输入。</span>
                </div>
                <button type="button" onClick={startNewTextOptimization}>重新优化</button>
              </section>
            ) : (
              <section className="prompt-tool-panel" id="prompt-panel-text" role="tabpanel" aria-labelledby="prompt-tab-text">
                <label>
                  <span>原始想法</span>
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
                <button type="button" className="primary" disabled={loading} onClick={generateTextPrompt}>{loading ? '优化中...' : '优化提示词'}</button>
              </section>
            )
          ) : null}

          {tab === 'image' ? (
            <section
              className={`prompt-tool-panel prompt-image-restore-panel ${imageDropActive ? 'is-dragging' : ''} ${imagePreviewUrl ? 'has-preview' : ''}`}
              id="prompt-panel-image"
              role="tabpanel"
              aria-labelledby="prompt-tab-image"
              tabIndex={0}
              onDragEnter={handleImageDragEnter}
              onDragOver={handleImageDragOver}
              onDragLeave={handleImageDragLeave}
              onDrop={handleImageDrop}
              onPaste={handleImagePanelPaste}
              onClick={handleImagePanelClick}
            >
              <input ref={imageInputRef} className="prompt-image-hidden-input" type="file" accept="image/png,image/jpeg,image/webp" onChange={(event) => void handleLocalUpload(event.target.files)} />
              <div className="prompt-image-restore-controls">
                <div className="prompt-image-dropzone" role="button" tabIndex={0} onClick={() => imageInputRef.current?.click()} onKeyDown={(event) => { if (event.key === 'Enter' || event.key === ' ') { event.preventDefault(); imageInputRef.current?.click() } }}>
                  <div className="prompt-image-empty-state">
                    <strong>{imagePreviewUrl ? '参考图已载入' : '粘贴或拖入图片'}</strong>
                    <span>点击此区域选择文件，Ctrl+V 粘贴截图，或把图片拖到整块图片还原区域后松开。</span>
                  </div>
                  {imageDropActive ? <div className="prompt-image-drop-overlay">松开粘贴</div> : null}
                </div>
                <div className="prompt-image-restore-actions">
                  <button type="button" className="primary" disabled={loading || !uploadId} onClick={generateImagePrompt}>{loading ? '还原中...' : '还原提示词'}</button>
                  {imagePreviewUrl ? <button type="button" onClick={() => imageInputRef.current?.click()}>换一张</button> : null}
                  {imagePreviewUrl ? <button type="button" disabled={loading} onClick={() => void clearReferenceImage()}>删除参考图</button> : null}
                </div>
              </div>
              <figure className="prompt-image-preview">
                {imagePreviewUrl ? (
                  <img src={imagePreviewUrl} alt="待还原参考图预览" />
                ) : (
                  <figcaption>预览区会完整显示参考图</figcaption>
                )}
              </figure>
            </section>
          ) : null}

          {tab === 'inspiration' ? (
            <section className="prompt-tool-panel inspiration-panel inspiration-chat-panel" id="prompt-panel-inspiration" role="tabpanel" aria-labelledby="prompt-tab-inspiration">
              <div className="inspiration-chat-head">
                <div>
                  <strong>从一句话聊到完整提示词</strong>
                  <span>已完成 {inspirationProgress}</span>
                </div>
                <button type="button" onClick={resetInspirationConversation}>重开</button>
              </div>

              <div className="inspiration-stepper" aria-label="灵感追问进度">
                {inspirationSteps.map((stepItem, index) => {
                  const skipped = Boolean(inspirationSkipped[stepItem.id])
                  const completed = isInspirationStepComplete(inspirationAnswers, inspirationSkipped, stepItem.id)
                  return (
                    <button
                      key={stepItem.id}
                      type="button"
                      className={`${index === inspirationStepIndex ? 'active' : ''}${completed ? ' done' : ''}${skipped ? ' skipped' : ''}`}
                      onClick={() => selectInspirationStep(index)}
                    >
                      <span>{skipped ? '跳' : index + 1}</span>
                      <b>{stepItem.label}</b>
                    </button>
                  )
                })}
              </div>

              <div className="inspiration-chat-window" aria-live="polite">
                {inspirationMessages.map((item) => (
                  <div key={item.id} className={`inspiration-message ${item.role === 'user' ? 'from-user' : 'from-assistant'}`}>
                    {item.label ? <b>{item.label}</b> : null}
                    <p>{item.content}</p>
                  </div>
                ))}
              </div>

              <div className="inspiration-composer">
                <label>
                  <span>{inspirationCurrentStep.label}{inspirationCurrentStepSkipped ? <em>已跳过</em> : null}</span>
                  <textarea
                    value={inspirationDraft}
                    onChange={(event) => updateInspirationDraft(event.target.value)}
                    placeholder={inspirationCurrentStepSkipped ? '已跳过；输入内容可重新回答这一项' : inspirationCurrentStep.placeholder}
                    rows={3}
                  />
                </label>
                <div className="inspiration-quick-row">
                  {inspirationCurrentStep.quick.map((item) => (
                    <button key={item} type="button" onClick={() => appendInspirationQuick(item)}>{item}</button>
                  ))}
                </div>
                <div className="inspiration-action-row">
                  <button type="button" onClick={skipInspirationAnswer}>{inspirationStepIndex === inspirationSteps.length - 1 ? '跳过并完成' : '跳过此项'}</button>
                  <button type="button" onClick={submitInspirationAnswer}>{inspirationStepIndex === inspirationSteps.length - 1 ? '完成追问' : '继续追问'}</button>
                  <button type="button" className="primary" disabled={loading || !inspirationComplete} onClick={() => void generateInspirationPrompt()}>{loading ? '生成中...' : '生成完整提示词'}</button>
                </div>
              </div>

              <section className="inspiration-options-panel" aria-label="输出设置">
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
                <label className="inspiration-note-field">
                  <span>额外备注（可选）</span>
                  <textarea value={inspirationSeed} onChange={(event) => setInspirationSeed(event.target.value)} placeholder="例如：要适合商业投放，画面里不要出现可读文字" rows={2} />
                </label>
              </section>

              <div className="inspiration-secondary-actions">
                <button type="button" disabled={loading} onClick={() => void makeIdeas()}>{loading ? '生成中...' : '给我备选灵感'}</button>
              </div>

              {ideas.length ? (
                <div className="prompt-idea-head">
                  <strong>备选灵感 {ideas.length} 个</strong>
                  <span>点击任意方案可扩写成右侧完整提示词。</span>
                </div>
              ) : null}
              <div className="prompt-idea-grid inspiration-idea-grid">
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
            <section className="prompt-history-list" id="prompt-panel-history" role="tabpanel" aria-labelledby="prompt-tab-history">
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
                  <strong>{record.mode === 'image-to-prompt' ? '图片还原' : '提示词优化'}</strong>
                  <p>{record.flatPrompt}</p>
                  <footer>
                    <span>{record.model}</span>
                    {record.ratio ? <span>{record.ratio === 'auto' ? '自动比例' : `比例 ${record.ratio}`}</span> : null}
                    <button type="button" onClick={(event) => { event.stopPropagation(); void deleteRecord(record.id) }}>删除</button>
                  </footer>
                </article>
              ))}
            </section>
          ) : null}

          <div ref={resultScrollAnchorRef} className="prompt-result-scroll-anchor" aria-hidden="true" />
          <PromptResultPanel
            record={visibleRecord}
            session={visibleSession}
            activeVersion={visibleActiveVersion}
            activeVersionId={visibleActiveVersionId}
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
function isSupportedImageFile(file: File) {
  return ['image/png', 'image/jpeg', 'image/webp'].includes(file.type)
}

function hasImageTransfer(dataTransfer: DataTransfer) {
  return Array.from(dataTransfer.items || []).some((item) => item.kind === 'file' && item.type.startsWith('image/')) || Array.from(dataTransfer.files || []).some((file) => file.type.startsWith('image/'))
}
function pickVersion(session: PromptSession | null, versionId: string) {
  if (!session?.versions.length) return null
  return session.versions.find((item) => item.id === versionId) || session.versions.find((item) => item.id === session.activeVersionId) || session.versions[session.versions.length - 1]
}
