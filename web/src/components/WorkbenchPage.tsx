import { type ComponentProps, type FormEvent, useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { cancelTask, createTask, deleteTask, listTasks, retryTask, setTaskFavorite, uploadTaskImageToPixhost } from '../api/tasks'
import { createGifTask, type GifTaskDraft } from '../api/gifTasks'
import { getCurrentUser, logoutUser } from '../api/users'
import { deleteReferenceUpload, listReferenceUploads, uploadReferenceImages } from '../api/uploads'
import { getUserConfig } from '../api/config'
import { submitPromptSquareFromResult, type PromptSquareReferencePayload, type SubmitPromptSquareFromResultRequest } from '../api/promptSquare'
import type { CreateTaskRequest, Mode, ModelProvider, PromptLibraryReferenceImage, PromptLibraryUsePromptOptions, PromptSquareItem, ReferenceUpload, Task, TaskEvent, TaskResult, UserConfig, UserSession } from '../types'
import { SpaceLogin } from './SpaceLogin'
import { GenerationPanel } from './GenerationPanel'
import { SettingsPanel } from './SettingsPanel'
import { TaskDetailModal } from './TaskDetailModal'
import { TaskSidebar } from './TaskSidebar'
import { PromptAssistantModal } from './PromptAssistantModal'
import { PromptLibraryPage } from './PromptLibraryPage'
import { NodeWorkflowPage, type CanvasHistoryImage } from './NodeWorkflowPage'
import { PromptSquarePanel } from './PromptSquarePanel'
import { ModelSquarePage } from './ModelSquarePage'
import { ProfilePage } from './ProfilePage'
import { TopUpPage } from './TopUpPage'
import { ApiDocsPage } from './ApiDocsPage'
import { AdminPage } from './AdminPage'
import { AgentPage } from './AgentPage'
import { GifPage } from './GifPage'
import { ResultCanvas } from './ResultCanvas'
import type { ThemeMode } from './ThemeToggle'
import { WorkbenchMobileTabs } from './workbench/WorkbenchMobileTabs'
import { WorkbenchSidebar } from './workbench/WorkbenchSidebar'
import {
  buildWorkbenchMobileMoreSummary,
  buildWorkbenchMobileMoreTabs,
  buildWorkbenchMobilePrimaryTabs,
  buildWorkbenchMobileTabs,
  buildWorkbenchNavGroups,
  buildWorkbenchTabItems,
  type WorkbenchNavId,
  type WorkbenchTab,
} from './workbench/nav'
import { useTaskEvents } from '../hooks/useTaskEvents'
import { useSubmittedSquareKeys } from '../hooks/useSubmittedSquareKeys'
import {
  DEFAULT_IMAGE2_MODEL,
  IMAGE2_PROVIDER,
  getImage2ModelOption,
  image2ModelSubmissionSpec,
  image2SelectableRatio,
  normalizeImage2Model,
} from '../lib/models'
import { formatBytes } from '../lib/format'
import { nativeExitApp, nativeSaveImage } from '../lib/nativeBridge'
import { ensureAppBackBridge, installEdgeBackGesture, registerAppBackHandler } from '../lib/appBack'

type NumericInputValue = number | ''
type SquareSubmitResultOptions = {
  title?: string
  tags?: string[]
  referenceUsageNote?: string
}
const MAX_REFERENCE_IMAGES = 8
const MAX_REFERENCE_IMAGE_BYTES = 12 * 1024 * 1024
const MAX_REFERENCE_UPLOAD_BYTES = 50 * 1024 * 1024
const ALLOWED_REFERENCE_TYPES = new Set(['image/png', 'image/jpeg', 'image/webp'])

type AgentTaskBackflowPayload = Task | Task[] | {
  task?: Task
  job?: Task
  tasks?: Task[]
  taskId?: string
  taskIds?: string[]
}

type AgentPageBridgeProps = ComponentProps<typeof AgentPage> & {
  onTaskConfirmed?: (payload: AgentTaskBackflowPayload) => void
  onTaskCreated?: (payload: AgentTaskBackflowPayload) => void
  onConfirmTask?: (payload: AgentTaskBackflowPayload) => void
  onConfirmedTasks?: (payload: AgentTaskBackflowPayload) => void
}

export function WorkbenchPage({ theme, onToggleTheme }: { theme: ThemeMode; onToggleTheme: () => void }) {
  const [session, setSession] = useState<UserSession | null>(null)
  const [spaceReady, setSpaceReady] = useState(false)
  const [activeTab, setActiveTab] = useState<WorkbenchTab>('nodes')
  const [mobileMoreOpen, setMobileMoreOpen] = useState(false)
  const [keyReady, setKeyReady] = useState(false)
  const [keyPreview, setKeyPreview] = useState('')
  const [tasks, setTasks] = useState<Task[]>([])
  const [activeId, setActiveId] = useState<string | null>(null)
  const [detailId, setDetailId] = useState<string | null>(null)
  const [searchQuery, setSearchQuery] = useState('')
  const [selectedIds, setSelectedIds] = useState<Set<string>>(() => new Set())
  const [uploads, setUploads] = useState<ReferenceUpload[]>([])
  const [mode, setMode] = useState<Mode>('text-to-image')
  const [provider, setProvider] = useState<ModelProvider>(IMAGE2_PROVIDER)
  const [imageModel, setImageModel] = useState(DEFAULT_IMAGE2_MODEL)
  const [prompt, setPrompt] = useState('')
  const [canvasPromptInjection, setCanvasPromptInjection] = useState<{ revision: number; prompt: string; provider: ModelProvider; model: string; ratio: string } | null>(null)
  const canvasPromptInjectionRevisionRef = useRef(0)
  const [ratio, setRatio] = useState('auto')
  const [resolution, setResolution] = useState('auto')
  const [imageSize, setImageSize] = useState('')
  const [quality, setQuality] = useState('high')
  const [outputFormat, setOutputFormat] = useState('png')
  const [count, setCount] = useState<NumericInputValue>(1)
  const [concurrency, setConcurrency] = useState<NumericInputValue>(1)
  const [message, setMessage] = useState('')
  const [error, setError] = useState('')
  const [toast, setToast] = useState('')
  const { submittedSquareKeys, markSubmittedSquareKey } = useSubmittedSquareKeys()
  const activeTabRef = useRef(activeTab)
  const detailIdRef = useRef(detailId)
  const mobileMoreOpenRef = useRef(mobileMoreOpen)
  const tabHistoryRef = useRef<WorkbenchTab[]>([])
  const lastExitBackAtRef = useRef(0)
  const mobileMoreSheetRef = useRef<HTMLElement | null>(null)

  activeTabRef.current = activeTab
  detailIdRef.current = detailId
  mobileMoreOpenRef.current = mobileMoreOpen

  const detailTask = useMemo(() => tasks.find((task) => task.id === detailId), [tasks, detailId])
  const activeTask = useMemo(() => tasks.find((task) => task.id === activeId), [tasks, activeId])
  const recentResultImages = useMemo<CanvasHistoryImage[]>(() => tasks.flatMap((task) => task.results.filter((result) => result.ok && result.imageUrl).map((result) => ({
    id: `${task.id}:${result.index}`,
    src: result.imageUrl!,
    title: `生成结果 ${result.index + 1}`,
    subtitle: `${task.statusText} · #${result.index + 1}`,
    taskId: task.id,
    index: result.index,
    prompt: result.revisedPrompt || task.prompt,
  }))).slice(0, 12), [tasks])
  const favoriteIds = useMemo(() => new Set(tasks.filter((task) => task.favorite).map((task) => task.id)), [tasks])
  const currentKeyReady = keyReady
  const currentKeyPreview = keyPreview
  const activeCount = useMemo(() => tasks.filter((task) => !isFinal(task)).length, [tasks])
  const missingKeyCount = keyReady ? 0 : 1
  const tabItems = useMemo(() => buildWorkbenchTabItems({
    currentKeyReady,
    activeTask: activeTask ? { statusText: activeTask.statusText, progress: activeTask.progress, isFinal: isFinal(activeTask) } : null,
    activeCount,
    creditsBalance: session?.user.creditsBalance,
    isAdmin: session?.user.isAdmin,
    missingKeyCount,
  }), [activeCount, activeTask, currentKeyReady, missingKeyCount, session?.user.creditsBalance, session?.user.isAdmin])

  const navGroups = useMemo(() => buildWorkbenchNavGroups(tabItems), [tabItems])
  const mobilePrimaryTabs = useMemo(() => buildWorkbenchMobilePrimaryTabs(tabItems), [tabItems])
  const mobileMoreTabs = useMemo(() => buildWorkbenchMobileMoreTabs(tabItems), [tabItems])
  const mobileMoreActive = mobileMoreOpen || mobileMoreTabs.some((tab) => tab.id === activeTab)
  const mobileMoreSummary = useMemo(() => buildWorkbenchMobileMoreSummary(mobileMoreTabs), [mobileMoreTabs])
  const mobileTabs = useMemo(() => buildWorkbenchMobileTabs(mobilePrimaryTabs, mobileMoreSummary), [mobileMoreSummary, mobilePrimaryTabs])
  const goToTab = useCallback((nextTab: WorkbenchNavId, options?: { replace?: boolean }) => {
    const targetTab: WorkbenchTab = nextTab === 'admin' && !session?.user.isAdmin ? 'profile' : nextTab
    setActiveTab((current) => {
      if (current === targetTab) return current
      if (!options?.replace) {
        const history = tabHistoryRef.current
        if (history[history.length - 1] !== current) history.push(current)
        if (history.length > 32) history.splice(0, history.length - 32)
      }
      return targetTab
    })
  }, [session?.user.isAdmin])

  const handleMobileNavChange = useCallback((nextTab: WorkbenchNavId) => {
    setMobileMoreOpen(false)
    goToTab(nextTab)
  }, [goToTab])

  const toggleMobileMore = useCallback(() => {
    setMobileMoreOpen((open) => !open)
  }, [])

  const upsertTask = useCallback((task: Task) => {
    setTasks((prev) => {
      const index = prev.findIndex((item) => item.id === task.id)
      if (index < 0) return [task, ...prev]
      const next = [...prev]
      next[index] = task
      return next
    })
  }, [])

  const handleTaskEvent = useCallback((event: TaskEvent) => {
    if (event.event !== 'heartbeat') setMessage(`${event.chinese} / ${event.code}`)
  }, [])

  useTaskEvents(activeId, upsertTask, handleTaskEvent)

  useEffect(() => {
    ensureAppBackBridge()
    const removeEdgeGesture = installEdgeBackGesture()
    const unregister = registerAppBackHandler(() => {
      if (mobileMoreOpenRef.current) {
        setMobileMoreOpen(false)
        return true
      }

      if (detailIdRef.current) {
        setDetailId(null)
        return true
      }

      const current = activeTabRef.current
      const history = tabHistoryRef.current
      while (history.length) {
        const previous = history.pop()
        if (previous && previous !== current) {
          setActiveTab(previous)
          return true
        }
      }

      if (current !== 'nodes') {
        setActiveTab('nodes')
        return true
      }

      const now = Date.now()
      if (now - lastExitBackAtRef.current <= 2000) {
        void nativeExitApp().then((result) => {
          if (!result.handled || !result.ok) setToast(result.message || '请使用系统手势退出 APP')
        })
        return true
      }

      lastExitBackAtRef.current = now
      setToast('再返回一次退出 APP')
      return true
    })

    return () => {
      unregister()
      removeEdgeGesture()
    }
  }, [])

  useEffect(() => {
    if (!mobileMoreOpen) return
    const previousFocus = document.activeElement instanceof HTMLElement ? document.activeElement : null
    const focusableSelector = 'a[href], button:not([disabled]), textarea:not([disabled]), input:not([disabled]), select:not([disabled]), [tabindex]:not([tabindex="-1"])'
    window.setTimeout(() => mobileMoreSheetRef.current?.focus(), 0)
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        event.preventDefault()
        setMobileMoreOpen(false)
        return
      }
      if (event.key !== 'Tab') return
      const sheet = mobileMoreSheetRef.current
      if (!sheet) return
      const focusable = Array.from(sheet.querySelectorAll<HTMLElement>(focusableSelector))
      if (!focusable.length) {
        event.preventDefault()
        sheet.focus()
        return
      }
      const first = focusable[0]
      const last = focusable[focusable.length - 1]
      if (event.shiftKey && document.activeElement === first) {
        event.preventDefault()
        last.focus()
      } else if (!event.shiftKey && document.activeElement === last) {
        event.preventDefault()
        first.focus()
      }
    }
    document.addEventListener('keydown', handleKeyDown)
    return () => {
      document.removeEventListener('keydown', handleKeyDown)
      previousFocus?.focus()
    }
  }, [mobileMoreOpen])

  useEffect(() => {
    void getCurrentUser().then((next) => { setSession(next); setSpaceReady(true) }).catch(() => { setSpaceReady(false) })
  }, [])

  useEffect(() => {
    if (activeTab === 'admin' && session?.user.isAdmin === false) {
      tabHistoryRef.current = tabHistoryRef.current.filter((tab) => tab !== 'admin')
      setActiveTab('profile')
    }
  }, [activeTab, session?.user.isAdmin])

  useEffect(() => {
    if (!spaceReady) return
    void refreshTasks()
    void refreshUploads()
    void refreshUserConfig()
  }, [spaceReady])

  useEffect(() => {
    if (!spaceReady) return
    const pollMs = activeCount || activeTab === 'result' || activeTab === 'nodes' ? 5000 : 15000
    const timer = window.setInterval(() => {
      void refreshTasks()
    }, pollMs)
    return () => window.clearInterval(timer)
  }, [activeCount, activeTab, spaceReady])

  useEffect(() => {
    const liveIds = new Set(tasks.map((task) => task.id))
    setSelectedIds((prev) => {
      const next = new Set(Array.from(prev).filter((id) => liveIds.has(id)))
      return next.size === prev.size ? prev : next
    })
  }, [tasks])

  useEffect(() => {
    if (!toast) return
    const timer = window.setTimeout(() => setToast(''), 3600)
    return () => window.clearTimeout(timer)
  }, [toast])

  async function refreshTasks() {
    const items = await listTasks()
    setTasks(items)
    setActiveId((current) => current || items[0]?.id || null)
  }

  async function refreshUploads() {
    setUploads(await listReferenceUploads())
  }

  async function refreshUserConfig() {
    const cfg = await getUserConfig()
    applyUserConfig(cfg)
  }

  async function refreshSession() {
    try {
      setSession(await getCurrentUser())
    } catch {
      // Keep the current session visible when the balance refresh endpoint is temporarily unavailable.
    }
  }

  const applyUserConfig = useCallback((cfg: UserConfig) => {
    setKeyReady(cfg.apiKeySet)
    setKeyPreview(cfg.apiKeyPreview)

    setCount(cfg.defaultCount || 1)
    setConcurrency(cfg.defaultConcurrency || 1)
  }, [])

  function applyImageModelSelection(nextModel: string, nextRatio?: string, nextResolution?: string, nextSize?: string) {
    const normalizedModel = normalizeImage2Model(nextModel)
    const option = getImage2ModelOption(normalizedModel)
    const ratioValue = option.ratioSelectable ? image2SelectableRatio(option, nextRatio) : option.defaultRatio
    const resolutionValue = nextResolution && nextResolution !== 'auto' ? nextResolution : option.defaultResolution
    setProvider(IMAGE2_PROVIDER)
    setImageModel(normalizedModel)
    setRatio(ratioValue)
    setResolution(resolutionValue)
    setImageSize(option.ratioSelectable ? normalizeImageSizeForUI(nextSize) : '')
  }

  async function submit(event: FormEvent) {
    event.preventDefault()
    setError('')
    if (!keyReady) {
      setError('请先保存 codex-key，或确认已上传到云端')
      return
    }
    if (!prompt.trim()) { setError('请先输入提示词'); return }
    if (mode === 'gif') {
      setError('GIF 动图请在 GIF 动图页面创建单图动效任务')
      goToTab('gif')
      return
    }
    if (mode === 'image-to-image' && uploads.length === 0) { setError('参考图生成需要先上传参考图'); return }
    const submittedUploads = mode === 'image-to-image' ? [...uploads] : []
    const imageSpec = image2ModelSubmissionSpec(imageModel, ratio, resolution, imageSize)
    const payload: CreateTaskRequest = {
      provider: IMAGE2_PROVIDER,
      model: imageSpec.model,
      mode,
      prompt,
      ratio: imageSpec.ratio,
      resolution: imageSpec.resolution,
      size: imageSpec.size,
      quality,
      outputFormat,
      count: numericOrDefault(count, 1),
      concurrency: numericOrDefault(concurrency, 1),
      uploadIds: submittedUploads.map((item) => item.id),
    }
    try {
      const job = await createTask(payload)
      upsertTask(job)
      setActiveId(job.id)
      setPrompt('')
      goToTab('result')
      void refreshSession()
      setMessage(submittedUploads.length ? '任务已提交，参考图已保留，可继续作为素材使用' : '任务已提交，可稍后回来查看结果')
    } catch (err) {
      setError(err instanceof Error ? err.message : '提交失败')
    }
  }

  async function handleCreateGifTask(draft: GifTaskDraft) {
    setError('')
    try {
      const job = await createGifTask(draft)
      upsertTask(job)
      setActiveId(job.id)
      goToTab('result')
      void refreshSession()
      setMessage(`GIF 动图任务已创建：${draft.preset.title} · ${draft.reference.originalName}。正在本地生成动图，结果会进入历史。`)
      setToast('GIF 任务已创建，已进入结果历史')
    } catch (err) {
      const message = err instanceof Error ? err.message : 'GIF 任务创建失败'
      setError(message)
      setToast(message)
      throw err
    }
  }

  async function handleCreateNodeWorkflowTask(payload: CreateTaskRequest) {
    if (!keyReady) {
      setToast('请先保存 codex-key，或确认已上传到云端')
      goToTab('settings')
      throw new Error('codex-key 未设置')
    }
    if (!payload.prompt.trim()) throw new Error('请先输入提示词')
    if (payload.mode === 'gif') {
      throw new Error('GIF 动图请在 GIF 动图页面创建单图动效任务')
    }
    if (payload.mode === 'image-to-image' && payload.uploadIds.length === 0) {
      throw new Error('参考图生成需要先选择参考图后再提交')
    }

    const imageSpec = image2ModelSubmissionSpec(payload.model || imageModel, payload.ratio, payload.resolution, payload.size || '')
    const imagePayload: CreateTaskRequest = {
      ...payload,
      provider: IMAGE2_PROVIDER,
      model: imageSpec.model,
      ratio: imageSpec.ratio,
      resolution: imageSpec.resolution,
      size: imageSpec.size,
    }
    const job = await createTask(imagePayload)
    upsertTask(job)
    setActiveId(job.id)
    applyImageModelSelection(imagePayload.model, imagePayload.ratio || 'auto', imagePayload.resolution || 'auto', imagePayload.size || '')
    setMode(imagePayload.mode)
    setQuality(imagePayload.quality || 'high')
    setOutputFormat(imagePayload.outputFormat || 'png')
    setCount(imagePayload.count || 1)
    setConcurrency(imagePayload.concurrency || 1)
    setPrompt('')
    goToTab('result')
    void refreshSession()
    setMessage('创作画布任务已提交，已进入结果历史')
    setToast('创作画布任务已创建，已进入结果页')
  }

  function handleAgentTaskBackflow(payload: AgentTaskBackflowPayload) {
    const confirmedTasks = agentTasksFromBackflow(payload)
    const confirmedIds = agentTaskIdsFromBackflow(payload)
    confirmedTasks.forEach(upsertTask)

    const activeTaskId = confirmedTasks[0]?.id || confirmedIds[0]
    if (activeTaskId) setActiveId(activeTaskId)
    goToTab('result')
    void refreshTasks()
    void refreshSession()

    const total = confirmedTasks.length || confirmedIds.length
    setMessage(total > 1 ? `Agent 已创建 ${total} 个任务，已进入结果历史` : 'Agent 已创建任务，已进入结果历史')
  }

  async function handleUpload(files: File[]) {
    if (!files.length) return []
    const validation = validateReferenceFiles(files, uploads.length)
    if (validation) {
      setToast(validation)
      return []
    }
    try {
      const created = await uploadReferenceImages(files)
      await refreshUploads()
      setToast(`已上传 ${created.length} 张参考图`)
      return created
    } catch (err) {
      setToast(formatReferenceUploadError(err))
      return []
    }
  }

  async function handleDeleteUpload(id: string) {
    await deleteReferenceUpload(id)
    await refreshUploads()
  }

  async function uploadResultImageAsReference(src: string, index: number) {
    const response = await fetch(src, { cache: 'no-store' })
    if (!response.ok) throw new Error(`读取结果图失败：HTTP ${response.status}`)
    const blob = await response.blob()
    const file = new File([blob], `result-reference-${index + 1}.${extensionFromMime(blob.type)}`, { type: blob.type || 'image/png' })
    const created = await uploadReferenceImages([file])
    await refreshUploads()
    if (!created[0]) throw new Error('参考图上传失败：未收到文件信息')
    return created[0]
  }

  async function handleUseResultAsReference(src: string, index: number) {
    const created = await uploadResultImageAsReference(src, index)
    setMode('image-to-image')
    goToTab('nodes')
    setMessage('已加入创作画布参考图')
    return created
  }

  async function handleUseResultAsGifReference(src: string, index: number) {
    const created = await uploadResultImageAsReference(src, index)
    setMessage('已加入 GIF 参考图')
    return created
  }

  async function handleUseResultAsReferenceVoid(src: string, index: number) {
    await handleUseResultAsReference(src, index)
  }

  async function handleUploadPixhost(taskId: string, index: number) {
    const data = await uploadTaskImageToPixhost(taskId, index)
    upsertTask(data.job)
    setMessage(data.result.remoteUrl ? 'PiXhost 图床上传成功' : 'PiXhost 图床上传完成')
  }

  function handleUseAssistantPrompt(nextPrompt: string, options?: { provider: ModelProvider; model: string; ratio?: string }) {
    setPrompt(nextPrompt)
    if (options) applyImageModelSelection(options.model, options.ratio || ratio)
    goToTab('nodes')
    setMessage('提示词已填入创作画布')
    window.setTimeout(() => {
      document.querySelector('[data-generation-composer] textarea')?.scrollIntoView({ behavior: 'smooth', block: 'center' })
    }, 0)
  }

  function promptOptionsWithRatio(options: { provider: ModelProvider; model: string; ratio?: string }) {
    return { provider: IMAGE2_PROVIDER, model: normalizeImage2Model(options.model), ratio: options.ratio || ratio }
  }

  function applyPromptModelOptions(options: { provider: ModelProvider; model: string; ratio: string }) {
    applyImageModelSelection(options.model, options.ratio)
  }

  function applyPromptToGenerate(nextPrompt: string, options: { provider: ModelProvider; model: string; ratio?: string }) {
    const nextOptions = promptOptionsWithRatio(options)
    setPrompt(nextPrompt)
    applyPromptModelOptions(nextOptions)
    goToTab('generate')
    window.setTimeout(() => {
      document.querySelector('[data-generation-composer] textarea')?.scrollIntoView({ behavior: 'smooth', block: 'center' })
    }, 0)
  }

  function injectPromptToCanvas(nextPrompt: string, options: { provider: ModelProvider; model: string; ratio?: string }) {
    const nextOptions = promptOptionsWithRatio(options)
    const revision = canvasPromptInjectionRevisionRef.current + 1
    canvasPromptInjectionRevisionRef.current = revision
    setPrompt(nextPrompt)
    applyPromptModelOptions(nextOptions)
    setCanvasPromptInjection({
      revision,
      prompt: nextPrompt,
      provider: nextOptions.provider,
      model: nextOptions.model,
      ratio: nextOptions.ratio || 'auto',
    })
    goToTab('nodes')
  }

  async function importPromptLibraryReferenceImage(referenceImage: PromptLibraryReferenceImage) {
    const response = await fetch(referenceImage.url, { cache: 'no-store' })
    if (!response.ok) throw new Error(`例图下载失败：HTTP ${response.status}`)
    const blob = await response.blob()
    const mime = normalizePromptLibraryReferenceMime(blob.type, referenceImage.url)
    const file = new File([blob], promptLibraryReferenceFileName(referenceImage, mime), { type: mime })
    const validation = validateReferenceFiles([file], uploads.length)
    if (validation) throw new Error(validation)
    const created = await uploadReferenceImages([file])
    await refreshUploads()
    if (!created[0]) throw new Error('参考图上传失败：未收到文件信息')
    return created[0]
  }

  async function handleUseLibraryPrompt(nextPrompt: string, options: PromptLibraryUsePromptOptions) {
    injectPromptToCanvas(nextPrompt, options)
    if (!options.referenceImage) {
      setMessage('提示词库已发送到创作画布，并同步模型选择')
      return
    }

    setMessage('提示词库已发送到创作画布，正在导入例图参考...')
    try {
      const uploaded = await importPromptLibraryReferenceImage(options.referenceImage)
      setMode('image-to-image')
      setMessage(`提示词库已发送到创作画布，并加入参考图：${uploaded.originalName}`)
    } catch (err) {
      setToast(formatPromptLibraryReferenceError(err))
      setMessage('提示词库已发送到创作画布；例图暂未加入参考图')
    }
  }

  function handleUseCanvasPrompt(nextPrompt: string, options: { provider: ModelProvider; model: string; ratio?: string }) {
    applyPromptToGenerate(nextPrompt, options)
    setMessage('创作画布已填入主输入框，并同步模型选择')
  }

  function handleUseAgentPromptToCanvas(nextPrompt: string, options: { provider: ModelProvider; model: string; ratio?: string }) {
    injectPromptToCanvas(nextPrompt, options)
    setMessage('Agent 已发送到创作画布')
  }
  async function handleUseSquarePrompt(nextPrompt: string, item?: PromptSquareItem) {
    setPrompt(nextPrompt)
    if (item) {
      const model = (item.model || item.params?.model || '').trim()
      const ratioValue = (item.ratio || item.params?.ratio || '').trim()
      const resolutionValue = (item.resolution || item.params?.resolution || '').trim()
      const qualityValue = (item.quality || item.params?.quality || '').trim()
      const formatValue = (item.outputFormat || item.params?.outputFormat || '').trim()
      const sizeValue = (item.params?.size || '').trim()
      applyImageModelSelection(model || DEFAULT_IMAGE2_MODEL, ratioValue || ratio, resolutionValue || resolution, sizeValue)
      if (qualityValue) setQuality(qualityValue)
      if (formatValue) setOutputFormat(formatValue)
    }
    goToTab('generate')
    window.setTimeout(() => {
      document.querySelector('[data-generation-composer] textarea')?.scrollIntoView({ behavior: 'smooth', block: 'center' })
    }, 0)

    const reference = item?.references?.find((entry) => entry.imageUrl || entry.thumbnailUrl)
    if (!reference) {
      setMessage(item ? '已从提示词广场填入主输入框，并同步可用参数' : '已从提示词广场填入主输入框')
      return
    }

    setMessage('已从提示词广场填入主输入框，正在导入参考图...')
    try {
      const uploaded = await importPromptLibraryReferenceImage({
        url: reference.imageUrl || reference.thumbnailUrl || '',
        alt: reference.usageNote || reference.originalName || item?.title || '广场参考图',
        itemId: item?.id || 'prompt-square',
        itemTitle: item?.title || '广场投稿',
        itemCategory: '提示词广场',
      })
      setMode('image-to-image')
      setMessage(`已从提示词广场应用提示词、参数和参考图：${uploaded.originalName}`)
    } catch (err) {
      setToast(formatPromptLibraryReferenceError(err))
      setMessage('已从提示词广场填入主输入框；参考图暂未导入')
    }
  }

  function handleSelectTask(task: Task) {
    setActiveId(task.id)
    goToTab('result')
  }

  function handleOpenTaskDetail(task: Task) {
    setActiveId(task.id)
    setDetailId(task.id)
  }

  async function handleSubmitResultToSquare(task: Task, result: TaskResult, index: number, options?: SquareSubmitResultOptions) {
    if (!result.ok || !result.imageUrl) throw new Error('只有成功生成的图片可以提交到广场')
    const defaultTitle = compactSquareText(result.revisedPrompt || task.prompt || `生成结果 ${index + 1}`, 80)
    const title = (options?.title || defaultTitle).trim() || defaultTitle
    const tags = options?.tags?.length ? options.tags : defaultSquareTags(task)
    const referencePayload = buildSquareReferenceSubmitPayload(task, options?.referenceUsageNote)

    const item = await submitPromptSquareFromResult({ taskId: task.id, imageIndex: index, title, tags, ...referencePayload })
    markSubmittedSquareKey(task.id, index)
    setMessage(`已提交到广场：${item.title || title || defaultTitle}`)
    setToast('已提交到广场，会生成广场展示副本')
    return true
  }
  function handleReuseTask(task: Task) {
    if (task.mode === 'gif') {
      setActiveId(task.id)
      goToTab('gif')
      setMessage('已打开 GIF 任务；GIF 参数保留在独立模块和任务历史中，不会写入快捷生成')
      window.setTimeout(() => {
        document.querySelector('[data-gif-composer] textarea')?.scrollIntoView({ behavior: 'smooth', block: 'center' })
      }, 0)
      return
    }
    applyImageModelSelection(task.model || DEFAULT_IMAGE2_MODEL, task.ratio || 'auto', task.resolution || 'auto', task.size || '')
    setMode(task.mode)
    setPrompt(task.prompt)
    setQuality(task.quality || 'high')
    setOutputFormat(task.outputFormat || 'png')
    setCount(task.count || 1)
    setConcurrency(task.concurrency || 1)
    goToTab('generate')
    setMessage('已复用该任务的提示词和参数')
    window.setTimeout(() => {
      document.querySelector('[data-generation-composer] textarea')?.scrollIntoView({ behavior: 'smooth', block: 'center' })
    }, 0)
  }

  async function toggleFavorite(id: string) {
    const current = tasks.find((task) => task.id === id)
    const job = await setTaskFavorite(id, !(current?.favorite ?? false))
    upsertTask(job)
    setMessage(job.favorite ? '已收藏任务' : '已取消收藏任务')
  }

  async function handleRetry(id: string) {
    if (!keyReady) {
      setToast('请先保存 codex-key，或确认已上传到云端')
      goToTab('settings')
      return
    }
    try {
      const job = await retryTask(id)
      upsertTask(job)
      setActiveId(job.id)
      goToTab('result')
      setDetailId(job.id)
      setMessage('已重新提交任务')
    } catch (err) {
      setToast(err instanceof Error ? err.message : '重试失败')
    }
  }

  async function handleCancel(id: string) {
    const job = await cancelTask(id)
    upsertTask(job)
    setMessage('已取消任务')
  }

  function toggleSelectTask(id: string) {
    setSelectedIds((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  function selectVisibleTasks(ids: string[]) {
    setSelectedIds((prev) => new Set([...Array.from(prev), ...ids]))
  }

  async function handleDelete(id: string) {
    const task = tasks.find((item) => item.id === id)
    if (!window.confirm(`确认删除这条生成记录？${task?.results?.some((result) => result.ok) ? '本地图片文件会先保留。' : ''}`)) return
    await deleteTask(id)
    setTasks((prev) => prev.filter((item) => item.id !== id))
    setSelectedIds((prev) => {
      const next = new Set(prev)
      next.delete(id)
      return next
    })
    setDetailId((current) => (current === id ? null : current))
    setActiveId((current) => {
      if (current !== id) return current
      const next = tasks.find((item) => item.id !== id)
      return next?.id || null
    })
    setMessage('已删除任务记录')
  }

  async function handleBatchFavorite(favorite: boolean) {
    const ids = Array.from(selectedIds)
    if (!ids.length) return
    let ok = 0
    let failed = 0
    for (const id of ids) {
      try {
        upsertTask(await setTaskFavorite(id, favorite))
        ok += 1
      } catch {
        failed += 1
      }
    }
    setMessage(`${favorite ? '批量收藏' : '取消收藏'}完成：成功 ${ok}，失败 ${failed}`)
  }

  async function handleBatchDelete() {
    const ids = Array.from(selectedIds)
    if (!ids.length) return
    if (!window.confirm(`确认删除选中的 ${ids.length} 条任务记录？本地图片文件会先保留。`)) return
    const deleted = new Set<string>()
    let failed = 0
    for (const id of ids) {
      try {
        await deleteTask(id)
        deleted.add(id)
      } catch {
        failed += 1
      }
    }
    setTasks((prev) => prev.filter((task) => !deleted.has(task.id)))
    setSelectedIds((prev) => new Set(Array.from(prev).filter((id) => !deleted.has(id))))
    setDetailId((current) => (current && deleted.has(current) ? null : current))
    setActiveId((current) => {
      if (!current || !deleted.has(current)) return current
      return tasks.find((task) => !deleted.has(task.id))?.id || null
    })
    setMessage(`批量删除完成：成功 ${deleted.size}，失败 ${failed}`)
  }

  async function handleBatchDownload() {
    const selected = selectedIds
    const items = tasks
      .filter((task) => selected.has(task.id))
      .flatMap((task) => task.results
        .filter((result) => result.ok && result.imageUrl)
        .map((result) => ({
          url: result.imageUrl!,
          name: `${task.id}-${result.index + 1}.${extensionFromMime(result.mime || 'image/png')}`,
        })))
    if (!items.length) {
      setMessage('选中的任务没有可下载图片')
      return
    }
    let ok = 0
    let failed = 0
    for (const item of items) {
      try {
        await downloadURL(item.url, item.name)
        ok += 1
      } catch {
        failed += 1
      }
      await delay(120)
    }
    setMessage(failed ? `下载完成：成功 ${ok}，失败 ${failed}` : `已保存/下载 ${ok} 张图片`)
  }

  const handleCanvasPromptInjectionApplied = useCallback((revision: number) => {
    setCanvasPromptInjection((current) => (current?.revision === revision ? null : current))
  }, [])

  async function logout() {
    await logoutUser()
    setSession(null)
    setSpaceReady(false)
    setTasks([])
    setUploads([])
    setActiveId(null)
    setDetailId(null)
    setSelectedIds(new Set())
  }

  if (!session) return <SpaceLogin theme={theme} onToggleTheme={onToggleTheme} onSession={(next) => { setSession(next); setSpaceReady(true) }} />

  const agentPageProps: AgentPageBridgeProps = {
    provider: IMAGE2_PROVIDER,
    model: imageModel,
    ratio,
    onCopyPrompt: () => setMessage('Agent 提示词已复制'),
    onUsePromptToCanvas: (payload) => handleUseAgentPromptToCanvas(payload.prompt, { provider: payload.provider, model: payload.model, ratio: payload.ratio }),
    onTaskConfirmed: handleAgentTaskBackflow,
    onTaskCreated: handleAgentTaskBackflow,
    onConfirmTask: handleAgentTaskBackflow,
    onConfirmedTasks: handleAgentTaskBackflow,
  }

  return (
    <div className="app-shell gallery-shell tabbed-workbench workbench-with-sidebar">
      <div className="workbench-desktop-frame">
        <WorkbenchSidebar
          navGroups={navGroups}
          activeTab={activeTab}
          user={session.user}
          creditsBalance={session.user.creditsBalance}
          theme={theme}
          onThemeChange={onToggleTheme}
          onLogout={() => void logout()}
          onChange={goToTab}
        />
        <div className="workbench-main-region">
      <main className={`workflow-content workflow-${activeTab}`}>
        {activeTab === 'generate' ? (
          <section className="workflow-page generate-page" data-generation-composer>
            <PageHeader eyebrow="Quick Generate" title="快捷生成" description="适合快速输入提示词；复杂参考图组织、预览和关系表达请在创作画布完成。" />
            <div className="generate-combined-layout">
              <div className="generate-main-column">
                {!currentKeyReady ? (
                  <div className="key-warning">
                    <strong>codex-key 未设置</strong>
                    <span>当前模型还没有可用 Key，先去设置保存，或主动上传到云端后再生成。</span>
                    <button type="button" onClick={() => goToTab('settings')}>去设置</button>
                  </div>
                ) : null}
                <GenerationPanel
                  mode={mode}
                  provider={provider}
                  prompt={prompt}
                  ratio={ratio}
                  resolution={resolution}
                  size={imageSize}
                  quality={quality}
                  outputFormat={outputFormat}
                  imageModel={imageModel}
                  count={count}
                  concurrency={concurrency}
                  uploads={uploads}
                  keyReady={currentKeyReady}
                  keyPreview={currentKeyPreview}
                  message={message}
                  error={error}
                  onModeChange={setMode}
                  onImageModelChange={applyImageModelSelection}
                  onPromptChange={setPrompt}
                  onRatioChange={setRatio}
                  onResolutionChange={setResolution}
                  onSizeChange={setImageSize}
                  onQualityChange={setQuality}
                  onOutputFormatChange={setOutputFormat}
                  onCountChange={setCount}
                  onConcurrencyChange={setConcurrency}
                  onOpenSettings={() => goToTab('settings')}
                  onUpload={handleUpload}
                  onDeleteUpload={handleDeleteUpload}
                  onSubmit={submit}
                />
              </div>
            </div>
          </section>
        ) : null}

        {activeTab === 'library' ? (
          <PromptLibraryPage
            provider={provider}
            onUsePrompt={handleUseLibraryPrompt}
          />
        ) : null}

        {activeTab === 'nodes' ? (
          <NodeWorkflowPage
            provider={provider}
            prompt={prompt}
            injectedPrompt={canvasPromptInjection?.prompt}
            injectedPromptRevision={canvasPromptInjection?.revision ?? 0}
            injectedPromptProvider={canvasPromptInjection?.provider}
            injectedPromptModel={canvasPromptInjection?.model}
            injectedPromptRatio={canvasPromptInjection?.ratio}
            onPromptInjectionApplied={handleCanvasPromptInjectionApplied}
            onUsePrompt={handleUseCanvasPrompt}
            onCreateTask={handleCreateNodeWorkflowTask}
            referenceUploads={uploads}
            recentResults={recentResultImages}
            latestTask={activeTask}
            onUploadReferences={handleUpload}
            onDeleteReferenceUpload={handleDeleteUpload}
            onUseHistoryImageAsReference={handleUseResultAsReference}
          />
        ) : null}

        {activeTab === 'gif' ? (
          <section className="workflow-page gif-page">
            <PageHeader eyebrow="GIF Motion" title="GIF 动图" description="上传或选择一张历史图片，创建循环动效并进入结果历史。" />
            <GifPage
              uploads={uploads}
              recentResults={recentResultImages}
              keyReady={keyReady}
              keyPreview={keyPreview}
              message={activeTab === 'gif' ? message : ''}
              error={activeTab === 'gif' ? error : ''}
              onUpload={handleUpload}
              onDeleteUpload={handleDeleteUpload}
              onUseHistoryImageAsReference={handleUseResultAsGifReference}
              onOpenSettings={() => goToTab('settings')}
              onSubmitTask={handleCreateGifTask}
            />
          </section>
        ) : null}

        {activeTab === 'agent' ? (
          <section className="workflow-page agent-workflow-page">
            <AgentPage {...agentPageProps} />
          </section>
        ) : null}

        {activeTab === 'assistant' ? (
          <section className="workflow-page assistant-page">
            <PromptAssistantModal
              embedded
              tasks={tasks}
              uploads={uploads}
              provider={provider}
              onClose={() => goToTab('nodes')}
              onUsePrompt={handleUseAssistantPrompt}
              onRefreshUploads={refreshUploads}
            />
          </section>
        ) : null}

        {activeTab === 'result' ? (
          <section className="workflow-page result-page">
            <div className="result-queue-layout">
              <TaskSidebar
                tasks={tasks}
                activeId={activeId || undefined}
                query={searchQuery}
                favoriteIds={favoriteIds}
                selectedIds={selectedIds}
                onQueryChange={setSearchQuery}
                onToggleSelect={toggleSelectTask}
                onSelectVisible={selectVisibleTasks}
                onClearSelection={() => setSelectedIds(new Set())}
                onBatchFavorite={(favorite) => void handleBatchFavorite(favorite)}
                onBatchDelete={() => void handleBatchDelete()}
                onBatchDownload={handleBatchDownload}
                onSelect={handleSelectTask}
                onOpenDetail={handleOpenTaskDetail}
                onRetry={(id) => void handleRetry(id)}
                onCancel={(id) => void handleCancel(id)}
                onDelete={(id) => void handleDelete(id)}
                onReuse={handleReuseTask}
                onToggleFavorite={(id) => void toggleFavorite(id)}
              />
              <ResultCanvas
                task={activeTask}
                onUseAsReference={handleUseResultAsReferenceVoid}
                onUploadPixhost={handleUploadPixhost}
                onOpenGenerate={() => goToTab('nodes')}
                onReuse={handleReuseTask}
                onRetry={(id) => void handleRetry(id)}
                submittedSquareKeys={submittedSquareKeys}
                onSubmitToSquare={handleSubmitResultToSquare}
              />
            </div>
          </section>
        ) : null}

        {activeTab === 'square' ? (
          <section className="workflow-page prompt-square-page">
            <PromptSquarePanel onUsePrompt={handleUseSquarePrompt} />
          </section>
        ) : null}

        {activeTab === 'modelSquare' ? (
          <section className="workflow-page model-square-workflow-page">
            <ModelSquarePage />
          </section>
        ) : null}


        {activeTab === 'profile' ? (
          <ProfilePage session={session} onSessionChange={(nextSession) => setSession(nextSession)} onOpenTopUp={() => goToTab('topup')} />
        ) : null}

        {activeTab === 'topup' ? (
          <TopUpPage session={session} onSessionChange={(nextSession) => setSession(nextSession)} />
        ) : null}

        {activeTab === 'apiDocs' ? (
          <ApiDocsPage />
        ) : null}

        {activeTab === 'settings' ? (
          <section className="workflow-page settings-page-inline">
            <PageHeader eyebrow="Settings" title="设置" description="管理账号安全、codex-pro 推荐、Image-2 / image-2-4k 生图分组提示、默认数量、默认并发和图床偏好。" />
            <div className="settings-inline-grid settings-only-grid">
              <SettingsPanel onConfig={applyUserConfig} />
            </div>
          </section>
        ) : null}

        {activeTab === 'admin' && session.user.isAdmin ? (
          <AdminPage theme={theme} onToggleTheme={onToggleTheme} embedded />
        ) : null}
          </main>
        </div>
      </div>

      <WorkbenchMobileTabs tabs={mobileTabs} activeTab={activeTab} moreActive={mobileMoreActive} moreOpen={mobileMoreOpen} onChange={handleMobileNavChange} onMore={toggleMobileMore} />

      {mobileMoreOpen ? (
        <div className="mobile-more-backdrop" role="presentation" onMouseDown={(event) => event.target === event.currentTarget && setMobileMoreOpen(false)}>
          <section ref={mobileMoreSheetRef} className="mobile-more-sheet" id="mobile-more-sheet" role="dialog" aria-modal="true" aria-labelledby="mobile-more-title" tabIndex={-1} onMouseDown={(event) => event.stopPropagation()}>
            <header className="mobile-more-header">
              <strong id="mobile-more-title">更多</strong>
              <button type="button" className="mobile-more-close" aria-label="关闭更多导航" onClick={() => setMobileMoreOpen(false)}>×</button>
            </header>
            <div className="mobile-more-list">
              {mobileMoreTabs.map((tab) => (
                <button key={tab.id} type="button" className={`mobile-more-item ${activeTab === tab.id ? 'active' : ''} ${tab.tone ? `tone-${tab.tone}` : ''}`} aria-current={activeTab === tab.id ? 'page' : undefined} onClick={() => handleMobileNavChange(tab.id)}>
                  <strong>{tab.label}</strong>
                </button>
              ))}
            </div>
          </section>
        </div>
      ) : null}

      {toast ? <div className="workbench-toast" role="status">{toast}</div> : null}

      {detailTask ? (
        <TaskDetailModal
          task={detailTask}
          favorite={favoriteIds.has(detailTask.id)}
          onClose={() => setDetailId(null)}
          onRetry={(id) => void handleRetry(id)}
          onCancel={(id) => void handleCancel(id)}
          onDelete={(id) => void handleDelete(id)}
          onReuse={handleReuseTask}
          onToggleFavorite={(id) => void toggleFavorite(id)}
          onUseAsReference={handleUseResultAsReferenceVoid}
          onUploadPixhost={handleUploadPixhost}
        />
      ) : null}
    </div>
  )
}

function normalizeImageSizeForUI(value?: string) {
  const normalized = (value || '').trim().toLowerCase().replace(/×/g, 'x').replace(/\s+/g, '')
  return normalized === 'auto' || normalized === '自动' ? '' : normalized
}
function defaultSquareTags(task: Task) {
  return [
    task.mode === 'gif' ? 'GIF动图' : task.mode === 'image-to-image' ? '图生图' : '文生图',
    squareModelTag(task),
    !task.ratio || task.ratio === 'auto' ? '' : task.ratio,
  ].filter(Boolean).slice(0, 6)
}

function squareModelTag(task: Task) {
  return image2SquareModelTag(task.model)
}

function image2SquareModelTag(model?: string) {
  const normalized = (model || '').trim()
  if (!normalized || normalized === 'image-2' || normalized === 'gpt-image-2' || normalized === DEFAULT_IMAGE2_MODEL) return DEFAULT_IMAGE2_MODEL
  return normalized
}

function buildSquareReferenceSubmitPayload(task: Task, referenceUsageNote?: string): Pick<SubmitPromptSquareFromResultRequest, 'referenceUploadIds' | 'references' | 'referenceUsageNote'> {
  const references = squareReferenceSubmitPayloads(task)
  const referenceUploadIds = uniqueStrings([
    ...(task.uploadIds || []),
    ...references.map((item) => item.uploadId || ''),
  ])
  const note = (referenceUsageNote || defaultSquareReferenceUsageNote(task, referenceUploadIds.length || references.length)).trim()
  if (!referenceUploadIds.length && !references.length) return {}

  const referencesWithNotes = references.map((item) => ({
    ...item,
    usageNote: item.usageNote || note || undefined,
  }))
  // Preserve reference metadata so published works can keep their source context.
  return {
    referenceUploadIds: referenceUploadIds.length ? referenceUploadIds : undefined,
    references: referencesWithNotes.length ? referencesWithNotes : undefined,
    referenceUsageNote: note || undefined,
  }
}

function squareReferenceSubmitPayloads(task: Task): PromptSquareReferencePayload[] {
  const out: PromptSquareReferencePayload[] = []
  const seen = new Set<string>()
  for (const reference of task.references || []) {
    const item: PromptSquareReferencePayload = {
      uploadId: reference.uploadId,
      originalName: reference.originalName,
      fileName: reference.fileName,
      mime: reference.mime,
      size: reference.size,
    }
    const key = item.uploadId || item.fileName || item.originalName || ''
    if (!key || seen.has(key)) continue
    seen.add(key)
    out.push(item)
  }
  for (const uploadId of task.uploadIds || []) {
    if (!uploadId || seen.has(uploadId)) continue
    seen.add(uploadId)
    out.push({ uploadId })
  }
  return out.slice(0, 12)
}

function defaultSquareReferenceUsageNote(task: Task, count: number) {
  if (!count) return ''
  return task.mode === 'image-to-image'
    ? '这些原始参考图用于图生图参考，可能影响主体、风格、构图和整体画面。'
    : '这些原始参考图随公开作品包保存，供别人理解和复用生成过程。'
}

function uniqueStrings(values: string[]) {
  const seen = new Set<string>()
  const out: string[] = []
  for (const value of values) {
    const normalized = value.trim()
    if (!normalized || seen.has(normalized)) continue
    seen.add(normalized)
    out.push(normalized)
  }
  return out.slice(0, 12)
}
function splitSubmitTags(value: string) {
  return value.split(/[,，\s]+/).map((item) => item.trim()).filter(Boolean).slice(0, 12)
}


function compactSquareText(value: string, max = 96) {
  const text = value.trim().replace(/\s+/g, ' ')
  if (text.length <= max) return text
  return `${text.slice(0, max)}...`
}

function PageHeader({ eyebrow, title, description }: { eyebrow: string; title: string; description: string }) {
  return (
    <header className="workflow-page-header">
      <p className="eyebrow">{eyebrow}</p>
      <h2>{title}</h2>
      <p>{description}</p>
    </header>
  )
}

function extensionFromMime(mime: string) {
  if (mime.includes('jpeg')) return 'jpg'
  if (mime.includes('webp')) return 'webp'
  if (mime.includes('gif')) return 'gif'
  return 'png'
}

function normalizePromptLibraryReferenceMime(mime: string, url: string) {
  const cleanMime = mime.split(';')[0].trim().toLowerCase()
  if (ALLOWED_REFERENCE_TYPES.has(cleanMime)) return cleanMime
  if (cleanMime) return cleanMime
  return promptLibraryReferenceMimeFromUrl(url)
}

function promptLibraryReferenceMimeFromUrl(url: string) {
  try {
    const pathname = new URL(url, window.location.origin).pathname.toLowerCase()
    if (pathname.endsWith('.jpg') || pathname.endsWith('.jpeg')) return 'image/jpeg'
    if (pathname.endsWith('.webp')) return 'image/webp'
  } catch {
    // Fall through to PNG; upload validation will catch unsupported responses.
  }
  return 'image/png'
}

function promptLibraryReferenceFileName(referenceImage: PromptLibraryReferenceImage, mime: string) {
  const baseName = sanitizeReferenceFilePart(referenceImage.itemTitle || referenceImage.itemId || 'prompt-library')
  return `prompt-library-${baseName}.${extensionFromMime(mime)}`
}

function sanitizeReferenceFilePart(value: string) {
  return value
    .trim()
    .replace(/[\\/:*?"<>|]+/g, '-')
    .replace(/\s+/g, '-')
    .slice(0, 54) || 'reference'
}

function formatPromptLibraryReferenceError(err: unknown) {
  const message = formatReferenceUploadError(err)
  if (/failed to fetch|load failed|networkerror|fetch/i.test(message)) {
    return '例图下载失败：当前图片源不允许浏览器直接读取，请先保存后再上传为参考图'
  }
  return message
}

async function downloadURL(url: string, filename: string) {
  const absoluteURL = new URL(url, window.location.origin).href
  const nativeResult = await nativeSaveImage(absoluteURL, filename)
  if (nativeResult.handled) {
    if (nativeResult.ok) return
    throw new Error(nativeResult.message || '保存图片失败')
  }
  const anchor = document.createElement('a')
  anchor.href = absoluteURL
  anchor.download = filename
  anchor.rel = 'noopener'
  document.body.appendChild(anchor)
  anchor.click()
  anchor.remove()
}

function delay(ms: number) {
  return new Promise<void>((resolve) => window.setTimeout(resolve, ms))
}

function numericOrDefault(value: NumericInputValue, fallback: number) {
  return value === '' ? fallback : value
}

function validateReferenceFiles(files: File[], existingCount: number) {
  if (existingCount + files.length > MAX_REFERENCE_IMAGES) {
    return `参考图最多 ${MAX_REFERENCE_IMAGES} 张，请先删除旧图`
  }
  const totalBytes = files.reduce((sum, file) => sum + file.size, 0)
  if (totalBytes > MAX_REFERENCE_UPLOAD_BYTES) {
    return `图片过大：一次上传总大小不能超过 ${formatBytes(MAX_REFERENCE_UPLOAD_BYTES)}`
  }
  const oversized = files.find((file) => file.size > MAX_REFERENCE_IMAGE_BYTES)
  if (oversized) {
    return `图片过大：${oversized.name || '参考图'} 超过 ${formatBytes(MAX_REFERENCE_IMAGE_BYTES)}`
  }
  const unsupported = files.find((file) => !ALLOWED_REFERENCE_TYPES.has(file.type))
  if (unsupported) {
    return `格式错误：${unsupported.name || '参考图'} 仅支持 PNG、JPG、WEBP`
  }
  return ''
}

function formatReferenceUploadError(err: unknown) {
  const message = err instanceof Error ? err.message : '参考图上传失败'
  if (message.includes('REFERENCE_IMAGE_TOO_LARGE') || message.includes('12MB')) return '图片过大：单张参考图不能超过 12MB'
  if (message.includes('REFERENCE_UPLOAD_INVALID') || message.includes('50MB')) return '图片过大：一次上传总大小不能超过 50MB'
  if (message.includes('REFERENCE_IMAGE_TYPE_UNSUPPORTED')) return '格式错误：参考图仅支持 PNG、JPG、WEBP'
  if (message.includes('REFERENCE_IMAGE_TOO_MANY')) return `参考图最多 ${MAX_REFERENCE_IMAGES} 张，请先删除旧图`
  return message
}

function isFinal(task: Task) {
  return ['succeeded', 'partial_failed', 'failed', 'cancelled', 'interrupted'].includes(task.status)
}

function agentTasksFromBackflow(payload: AgentTaskBackflowPayload): Task[] {
  if (Array.isArray(payload)) return payload.filter(isTaskPayload)
  if (isTaskPayload(payload)) return [payload]
  return [payload.task, payload.job, ...(Array.isArray(payload.tasks) ? payload.tasks : [])].filter(isTaskPayload)
}

function agentTaskIdsFromBackflow(payload: AgentTaskBackflowPayload) {
  const ids = agentTasksFromBackflow(payload).map((task) => task.id)
  if (!Array.isArray(payload) && !isTaskPayload(payload)) {
    if (payload.taskId) ids.push(payload.taskId)
    if (Array.isArray(payload.taskIds)) ids.push(...payload.taskIds)
  }
  return Array.from(new Set(ids.map((id) => id.trim()).filter(Boolean)))
}

function isTaskPayload(value: unknown): value is Task {
  if (!value || typeof value !== 'object') return false
  const candidate = value as Partial<Task>
  return typeof candidate.id === 'string' && typeof candidate.status === 'string' && Array.isArray(candidate.results)
}
