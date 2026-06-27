import { type FormEvent, useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { cancelTask, createTask, deleteTask, listTasks, retryTask, setTaskFavorite, uploadTaskImageToPixhost } from '../api/tasks'
import { getCurrentUser, logoutUser } from '../api/users'
import { deleteReferenceUpload, listReferenceUploads, uploadReferenceImages } from '../api/uploads'
import { getUserConfig } from '../api/config'
import { submitPromptSquareFromResult } from '../api/promptSquare'
import type { CreateTaskRequest, Mode, ModelProvider, ReferenceUpload, Task, TaskEvent, TaskResult, UserConfig, UserSession } from '../types'
import { SpaceLogin } from './SpaceLogin'
import { GenerationPanel } from './GenerationPanel'
import { SettingsPanel } from './SettingsPanel'
import { TaskDetailModal } from './TaskDetailModal'
import { TaskSidebar } from './TaskSidebar'
import { PromptAssistantModal } from './PromptAssistantModal'
import { PromptLibraryPage } from './PromptLibraryPage'
import { NodeWorkflowPage, type CanvasHistoryImage } from './NodeWorkflowPage'
import { PromptSquarePanel } from './PromptSquarePanel'
import { ProfilePage } from './ProfilePage'
import { TopUpPage } from './TopUpPage'
import { ApiDocsPage } from './ApiDocsPage'
import { AgentPage } from './AgentPage'
import { GifPage, type GifDraftSubmission } from './GifPage'
import { ResultCanvas } from './ResultCanvas'
import { ThemeToggle, type ThemeMode } from './ThemeToggle'
import { GitHubLink } from './GitHubLink'
import { useTaskEvents } from '../hooks/useTaskEvents'
import { useSubmittedSquareKeys } from '../hooks/useSubmittedSquareKeys'
import { BANANA_PROVIDER, DEFAULT_BANANA_MODEL, DEFAULT_IMAGE2_MODEL, getBananaModelForRatio, getBananaModelOption } from '../lib/models'
import { formatBytes } from '../lib/format'
import { nativeExitApp, nativeSaveImage } from '../lib/nativeBridge'
import { ensureAppBackBridge, installEdgeBackGesture, registerAppBackHandler } from '../lib/appBack'
import './WorkbenchSidebar.css'

type NumericInputValue = number | ''
type WorkbenchTab = 'generate' | 'gif' | 'assistant' | 'agent' | 'nodes' | 'library' | 'square' | 'result' | 'profile' | 'topup' | 'apiDocs' | 'settings'
type WorkbenchNavId = WorkbenchTab | 'admin'
type WorkbenchMobileNavId = WorkbenchNavId | 'more'
type WorkbenchTabItem = { id: WorkbenchNavId; label: string; hint: string; badge?: string; tone?: 'normal' | 'danger' | 'active' | 'admin' }
type WorkbenchMobileTabItem = Omit<WorkbenchTabItem, 'id'> & { id: WorkbenchMobileNavId }

const MAX_REFERENCE_IMAGES = 8
const MAX_REFERENCE_IMAGE_BYTES = 12 * 1024 * 1024
const MAX_REFERENCE_UPLOAD_BYTES = 50 * 1024 * 1024
const ALLOWED_REFERENCE_TYPES = new Set(['image/png', 'image/jpeg', 'image/webp'])

const workflowTabs: WorkbenchTabItem[] = [
  { id: 'nodes', label: '创作画布', hint: 'Canvas' },
  { id: 'gif', label: 'GIF 动图', hint: '动效' },
  { id: 'agent', label: 'Agent 创作', hint: '多轮' },
  { id: 'generate', label: '快捷生成', hint: '兼容' },
  { id: 'assistant', label: '提示词助手', hint: '四栏' },
  { id: 'library', label: '提示词库', hint: '灵感' },
  { id: 'square', label: '广场', hint: 'Prompt' },
  { id: 'result', label: '结果', hint: '队列' },
  { id: 'profile', label: '我的', hint: '账号' },
  { id: 'topup', label: '充值', hint: '充值' },
  { id: 'apiDocs', label: 'API 文档', hint: 'Bearer' },
  { id: 'settings', label: '设置', hint: 'Key' },
]
const mobilePrimaryTabIds: WorkbenchTab[] = ['nodes', 'result', 'square', 'profile']
const mobileMoreTabIds: WorkbenchNavId[] = ['gif', 'agent', 'generate', 'assistant', 'library', 'topup', 'apiDocs', 'settings', 'admin']

export function WorkbenchPage({ theme, onToggleTheme }: { theme: ThemeMode; onToggleTheme: () => void }) {
  const [session, setSession] = useState<UserSession | null>(null)
  const [spaceReady, setSpaceReady] = useState(false)
  const [activeTab, setActiveTab] = useState<WorkbenchTab>('nodes')
  const [mobileMoreOpen, setMobileMoreOpen] = useState(false)
  const [keyReady, setKeyReady] = useState(false)
  const [keyPreview, setKeyPreview] = useState('')
  const [bananaKeyReady, setBananaKeyReady] = useState(false)
  const [bananaKeyPreview, setBananaKeyPreview] = useState('')
  const [tasks, setTasks] = useState<Task[]>([])
  const [activeId, setActiveId] = useState<string | null>(null)
  const [detailId, setDetailId] = useState<string | null>(null)
  const [searchQuery, setSearchQuery] = useState('')
  const [selectedIds, setSelectedIds] = useState<Set<string>>(() => new Set())
  const [uploads, setUploads] = useState<ReferenceUpload[]>([])
  const [mode, setMode] = useState<Mode>('text-to-image')
  const [provider, setProvider] = useState<ModelProvider>('image-2')
  const [bananaModel, setBananaModel] = useState(DEFAULT_BANANA_MODEL)
  const [prompt, setPrompt] = useState('')
  const [ratio, setRatio] = useState('auto')
  const [resolution, setResolution] = useState('auto')
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
  const currentKeyReady = provider === BANANA_PROVIDER ? bananaKeyReady : keyReady
  const currentKeyPreview = provider === BANANA_PROVIDER ? bananaKeyPreview : keyPreview
  const activeCount = useMemo(() => tasks.filter((task) => !isFinal(task)).length, [tasks])
  const missingKeyCount = (keyReady ? 0 : 1) + (bananaKeyReady ? 0 : 1)
  const tabItems = useMemo<WorkbenchTabItem[]>(() => {
    const items = workflowTabs.map<WorkbenchTabItem>((tab) => {
      if (tab.id === 'generate') return { ...tab, hint: currentKeyReady ? '可提交' : '缺 Key', tone: currentKeyReady ? 'normal' : 'danger' }
      if (tab.id === 'gif') return { ...tab, hint: '单图动效' }
      if (tab.id === 'agent') return { ...tab, hint: '多轮' }
      if (tab.id === 'assistant') return { ...tab, hint: '四栏' }
      if (tab.id === 'library') return { ...tab, hint: '自动同步', tone: 'normal' }
      if (tab.id === 'square') return { ...tab, hint: '试验版' }
      if (tab.id === 'result') return { ...tab, hint: activeTask ? activeTask.statusText : activeCount ? `${activeCount} 进行中` : '队列', badge: activeTask ? `${activeTask.progress}%` : activeCount ? String(activeCount) : undefined, tone: activeTask && !isFinal(activeTask) ? 'active' : activeCount ? 'active' : 'normal' }
      if (tab.id === 'profile') return { ...tab, hint: session?.user.creditsBalance != null ? `${session.user.creditsBalance} 次` : '账号' }
      if (tab.id === 'apiDocs') return { ...tab, hint: 'Bearer' }
      if (tab.id === 'settings') return { ...tab, hint: missingKeyCount ? `${missingKeyCount} 个未设` : '已配置', badge: missingKeyCount ? '!' : undefined, tone: missingKeyCount ? 'danger' : 'normal' }
      return tab
    })
    if (session?.user.isAdmin) {
      items.push({ id: 'admin', label: '管理员后台', hint: '管理', tone: 'admin' })
    }
    return items
  }, [activeCount, activeTask, currentKeyReady, missingKeyCount, session?.user.creditsBalance, session?.user.isAdmin])

  const tabItemById = useMemo(() => new Map<WorkbenchNavId, WorkbenchTabItem>(tabItems.map((tab) => [tab.id, tab])), [tabItems])
  const mobilePrimaryTabs = useMemo(() => mobilePrimaryTabIds.map((id) => tabItemById.get(id)).filter((tab): tab is WorkbenchTabItem => Boolean(tab)), [tabItemById])
  const mobileMoreTabs = useMemo(() => mobileMoreTabIds.map((id) => tabItemById.get(id)).filter((tab): tab is WorkbenchTabItem => Boolean(tab)), [tabItemById])
  const mobileMoreActive = mobileMoreOpen || mobileMoreTabs.some((tab) => tab.id === activeTab)
  const mobileMoreSummary = useMemo<WorkbenchMobileTabItem>(() => {
    const hiddenDanger = mobileMoreTabs.find((tab) => tab.tone === 'danger' || tab.badge === '!')
    if (hiddenDanger) return { id: 'more', label: '更多', hint: hiddenDanger.label, badge: hiddenDanger.badge || '!', tone: 'danger' }
    const hiddenActive = mobileMoreTabs.find((tab) => tab.tone === 'active' || tab.badge)
    if (hiddenActive) return { id: 'more', label: '更多', hint: hiddenActive.label, badge: hiddenActive.badge, tone: hiddenActive.tone }
    const adminTab = mobileMoreTabs.find((tab) => tab.id === 'admin')
    if (adminTab) return { id: 'more', label: '更多', hint: '含后台', badge: adminTab.badge, tone: adminTab.tone }
    return { id: 'more', label: '更多', hint: '菜单' }
  }, [mobileMoreTabs])
  const mobileTabs = useMemo<WorkbenchMobileTabItem[]>(() => [
    ...mobilePrimaryTabs,
    mobileMoreSummary,
  ], [mobileMoreSummary, mobilePrimaryTabs])

  const goToTab = useCallback((nextTab: WorkbenchNavId, options?: { replace?: boolean }) => {
    if (nextTab === 'admin') {
      window.location.assign('/admin')
      return
    }
    setActiveTab((current) => {
      if (current === nextTab) return current
      if (!options?.replace) {
        const history = tabHistoryRef.current
        if (history[history.length - 1] !== current) history.push(current)
        if (history.length > 32) history.splice(0, history.length - 32)
      }
      return nextTab
    })
  }, [])

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
    setBananaKeyReady(Boolean(cfg.bananaApiKeySet))
    setBananaKeyPreview(cfg.bananaApiKeyPreview || '')
    setCount(cfg.defaultCount || 1)
    setConcurrency(cfg.defaultConcurrency || 1)
  }, [])

  async function submit(event: FormEvent) {
    event.preventDefault()
    setError('')
    const ready = provider === BANANA_PROVIDER ? bananaKeyReady : keyReady
    if (!ready) {
      setError(provider === BANANA_PROVIDER ? '请先保存 banana 分组 API Key，或确认已上传到云端' : '请先保存 codex-key，或确认已上传到云端')
      return
    }
    if (!prompt.trim()) { setError('请先输入提示词'); return }
    if (mode === 'gif') {
      setError('GIF 动图后端尚未接入，请在 GIF 动图页面先准备占位任务参数')
      goToTab('gif')
      return
    }
    if (mode === 'image-to-image' && uploads.length === 0) { setError('参考图生成需要先上传参考图'); return }
    const submittedUploads = mode === 'image-to-image' ? [...uploads] : []
    const payload: CreateTaskRequest = {
      provider,
      model: provider === BANANA_PROVIDER ? bananaModel : DEFAULT_IMAGE2_MODEL,
      mode,
      prompt,
      ratio: provider === BANANA_PROVIDER ? getBananaModelOption(bananaModel).ratio : ratio,
      resolution: provider === BANANA_PROVIDER ? getBananaModelOption(bananaModel).resolution : resolution,
      quality: provider === BANANA_PROVIDER ? 'auto' : quality,
      outputFormat: provider === BANANA_PROVIDER ? 'auto' : outputFormat,
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
      setMessage(submittedUploads.length ? '任务已提交，参考图已保留，可继续作为素材使用' : '任务已提交，后端会继续执行，前端可刷新或断开')
    } catch (err) {
      setError(err instanceof Error ? err.message : '提交失败')
    }
  }

  function handleCreateGifDraft(draft: GifDraftSubmission) {
    setError('')
    setProvider(draft.payload.provider)
    setMode(draft.payload.mode)
    setPrompt(draft.payload.prompt)
    setRatio(draft.payload.ratio || 'auto')
    setResolution(draft.payload.resolution || 'auto')
    setQuality(draft.payload.quality || 'auto')
    setOutputFormat(draft.payload.outputFormat || 'gif')
    setCount(draft.payload.count || 1)
    setConcurrency(draft.payload.concurrency || 1)
    setMessage(`GIF 动图参数已准备：${draft.preset.title} · ${draft.reference.originalName}。真实 GIF 后端尚未接入，未创建生成任务。`)
    setToast('GIF 参数已准备，尚未创建真实后端任务')
  }

  async function handleCreateNodeWorkflowTask(payload: CreateTaskRequest) {
    const ready = payload.provider === BANANA_PROVIDER ? bananaKeyReady : keyReady
    if (!ready) {
      const message = payload.provider === BANANA_PROVIDER ? 'Banana Key 未设置' : 'codex-key 未设置'
      setToast(payload.provider === BANANA_PROVIDER ? '请先保存 Banana API Key，或确认已上传到云端' : '请先保存 codex-key，或确认已上传到云端')
      goToTab('settings')
      throw new Error(message)
    }
    if (!payload.prompt.trim()) throw new Error('请先输入提示词')
    if (payload.mode === 'gif') {
      throw new Error('GIF 动图后端尚未接入，请在 GIF 动图页面先准备占位任务参数')
    }
    if (payload.mode === 'image-to-image' && payload.uploadIds.length === 0) {
      throw new Error('参考图生成需要接入参考图后再提交')
    }

    const job = await createTask(payload)
    upsertTask(job)
    setActiveId(job.id)
    setProvider(payload.provider)
    setMode(payload.mode)
    if (payload.provider === BANANA_PROVIDER) {
      setBananaModel(getBananaModelOption(payload.model).id)
    } else {
      setRatio(payload.ratio || 'auto')
      setResolution(payload.resolution || 'auto')
      setQuality(payload.quality || 'high')
      setOutputFormat(payload.outputFormat || 'png')
    }
    setCount(payload.count || 1)
    setConcurrency(payload.concurrency || 1)
    setPrompt('')
    void refreshSession()
    setMessage('创作画布任务已提交，右侧预览会持续更新')
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

  async function handleUseResultAsReference(src: string, index: number) {
    const response = await fetch(src, { cache: 'no-store' })
    if (!response.ok) throw new Error(`读取结果图失败：HTTP ${response.status}`)
    const blob = await response.blob()
    const file = new File([blob], `result-reference-${index + 1}.${extensionFromMime(blob.type)}`, { type: blob.type || 'image/png' })
    const created = await uploadReferenceImages([file])
    await refreshUploads()
    setMode('image-to-image')
    goToTab('nodes')
    setMessage('已加入创作画布参考图')
    return created[0]
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
    if (options) {
      setProvider(options.provider)
      if (options.provider === BANANA_PROVIDER) {
        const preferredResolution = getBananaModelOption(options.model).resolution
        const nextBanana = options.ratio && options.ratio !== 'auto'
          ? getBananaModelForRatio(options.ratio, preferredResolution === 'auto' ? '2k' : preferredResolution)
          : getBananaModelOption(options.model)
        setBananaModel(nextBanana.id)
      } else if (options.ratio && options.ratio !== 'auto') {
        setRatio(options.ratio)
      }
    }
    goToTab('generate')
    setMessage(options ? '提示词助手已填入主输入框，并同步模型选择' : '提示词助手已填入主输入框')
    window.setTimeout(() => {
      document.querySelector('[data-generation-composer] textarea')?.scrollIntoView({ behavior: 'smooth', block: 'center' })
    }, 0)
  }

  function applyPromptToGenerate(nextPrompt: string, options: { provider: ModelProvider; model: string; ratio?: string }) {
    setPrompt(nextPrompt)
    setProvider(options.provider)
    if (options.provider === BANANA_PROVIDER) {
      const preferredResolution = getBananaModelOption(options.model).resolution
      const nextBanana = options.ratio && options.ratio !== 'auto'
        ? getBananaModelForRatio(options.ratio, preferredResolution === 'auto' ? '2k' : preferredResolution)
        : getBananaModelOption(options.model)
      setBananaModel(nextBanana.id)
    } else if (options.ratio && options.ratio !== 'auto') {
      setRatio(options.ratio)
    }
    goToTab('generate')
    window.setTimeout(() => {
      document.querySelector('[data-generation-composer] textarea')?.scrollIntoView({ behavior: 'smooth', block: 'center' })
    }, 0)
  }

  function handleUseLibraryPrompt(nextPrompt: string, options: { provider: ModelProvider; model: string; ratio?: string }) {
    applyPromptToGenerate(nextPrompt, options)
    setMessage('提示词库已填入主输入框，并同步模型选择')
  }

  function handleUseCanvasPrompt(nextPrompt: string, options: { provider: ModelProvider; model: string; ratio?: string }) {
    applyPromptToGenerate(nextPrompt, options)
    setMessage('创作画布已填入主输入框，并同步模型选择')
  }

  function handleUseAgentPromptToCanvas(nextPrompt: string, options: { provider: ModelProvider; model: string; ratio?: string }) {
    setPrompt(nextPrompt)
    setProvider(options.provider)
    if (options.provider === BANANA_PROVIDER) {
      const preferredResolution = getBananaModelOption(options.model).resolution
      const nextBanana = options.ratio && options.ratio !== 'auto'
        ? getBananaModelForRatio(options.ratio, preferredResolution === 'auto' ? '2k' : preferredResolution)
        : getBananaModelOption(options.model)
      setBananaModel(nextBanana.id)
    } else if (options.ratio && options.ratio !== 'auto') {
      setRatio(options.ratio)
    }
    goToTab('nodes')
    setMessage('Agent 已发送到创作画布')
  }

  function handleUseSquarePrompt(nextPrompt: string) {
    setPrompt(nextPrompt)
    goToTab('generate')
    setMessage('已从提示词广场填入主输入框')
    window.setTimeout(() => {
      document.querySelector('[data-generation-composer] textarea')?.scrollIntoView({ behavior: 'smooth', block: 'center' })
    }, 0)
  }

  function handleSelectTask(task: Task) {
    setActiveId(task.id)
    goToTab('result')
  }

  function handleOpenTaskDetail(task: Task) {
    setActiveId(task.id)
    setDetailId(task.id)
  }

  async function handleSubmitResultToSquare(task: Task, result: TaskResult, index: number, options?: { title?: string; tags?: string[] }) {
    if (!result.ok || !result.imageUrl) throw new Error('只有成功生成的图片可以提交到广场')
    const defaultTitle = compactSquareText(result.revisedPrompt || task.prompt || `生成结果 ${index + 1}`, 80)
    const title = (options?.title || defaultTitle).trim() || defaultTitle
    const tags = options?.tags?.length ? options.tags : defaultSquareTags(task)

    const item = await submitPromptSquareFromResult({ taskId: task.id, imageIndex: index, title, tags })
    markSubmittedSquareKey(task.id, index)
    setMessage(`已提交到广场：${item.title || title || defaultTitle}`)
    setToast('已提交到广场，会生成广场展示副本')
    return true
  }

  function handleReuseTask(task: Task) {
    const nextProvider = task.provider || 'image-2'
    setProvider(nextProvider)
    setBananaModel(nextProvider === BANANA_PROVIDER ? getBananaModelOption(task.model || '').id : bananaModel)
    setMode(task.mode)
    setPrompt(task.prompt)
    setRatio(task.ratio || 'auto')
    setResolution(task.resolution || 'auto')
    setQuality(task.quality || 'high')
    setOutputFormat(task.outputFormat || 'png')
    setCount(task.count || 1)
    setConcurrency(task.concurrency || 1)
    goToTab(task.mode === 'gif' ? 'gif' : 'generate')
    setMessage('已复用该任务的提示词和参数')
    window.setTimeout(() => {
      document.querySelector(task.mode === 'gif' ? '[data-gif-composer] textarea' : '[data-generation-composer] textarea')?.scrollIntoView({ behavior: 'smooth', block: 'center' })
    }, 0)
  }

  async function toggleFavorite(id: string) {
    const current = tasks.find((task) => task.id === id)
    const job = await setTaskFavorite(id, !(current?.favorite ?? false))
    upsertTask(job)
    setMessage(job.favorite ? '已收藏任务' : '已取消收藏任务')
  }

  async function handleRetry(id: string) {
    const task = tasks.find((item) => item.id === id)
    const nextProvider = task?.provider || 'image-2'
    const ready = nextProvider === BANANA_PROVIDER ? bananaKeyReady : keyReady
    if (!ready) {
      setToast(nextProvider === BANANA_PROVIDER ? '请先保存 Banana API Key，或确认已上传到云端' : '请先保存 codex-key，或确认已上传到云端')
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

  return (
    <div className="app-shell gallery-shell tabbed-workbench workbench-with-sidebar">
      <div className="workbench-desktop-frame">
        <WorkbenchSidebar
          tabs={tabItems}
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
            <PageHeader eyebrow="Quick Generate" title="快捷生成" description="保留兼容入口；完整参考图组织、预览和关系表达请在创作画布完成。" />
            <div className="generate-combined-layout">
              <div className="generate-main-column">
                {!currentKeyReady ? (
                  <div className="key-warning">
                    <strong>{provider === BANANA_PROVIDER ? 'Banana Key 未设置' : 'codex-key 未设置'}</strong>
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
                  quality={quality}
                  outputFormat={outputFormat}
                  bananaModel={bananaModel}
                  count={count}
                  concurrency={concurrency}
                  uploads={uploads}
                  keyReady={currentKeyReady}
                  keyPreview={currentKeyPreview}
                  message={message}
                  error={error}
                  onModeChange={setMode}
                  onProviderChange={setProvider}
                  onPromptChange={setPrompt}
                  onRatioChange={setRatio}
                  onResolutionChange={setResolution}
                  onQualityChange={setQuality}
                  onOutputFormatChange={setOutputFormat}
                  onBananaModelChange={setBananaModel}
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
            bananaModel={bananaModel}
            onUsePrompt={handleUseLibraryPrompt}
          />
        ) : null}

        {activeTab === 'nodes' ? (
          <NodeWorkflowPage
            provider={provider}
            bananaModel={bananaModel}
            prompt={prompt}
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
            <PageHeader eyebrow="GIF Motion" title="GIF 动图" description="上传一张图片，选择动效预设并描述想法；当前先准备任务参数，不调用视频或 FFmpeg。" />
            <GifPage
              uploads={uploads}
              keyReady={keyReady}
              keyPreview={keyPreview}
              message={activeTab === 'gif' ? message : ''}
              error={activeTab === 'gif' ? error : ''}
              onUpload={handleUpload}
              onDeleteUpload={handleDeleteUpload}
              onOpenSettings={() => goToTab('settings')}
              onSubmitDraft={handleCreateGifDraft}
            />
          </section>
        ) : null}

        {activeTab === 'agent' ? (
          <section className="workflow-page agent-workflow-page">
            <AgentPage
              provider={provider}
              model={provider === BANANA_PROVIDER ? bananaModel : DEFAULT_IMAGE2_MODEL}
              onCopyPrompt={() => setMessage('Agent 提示词已复制')}
              onSendToCanvas={(payload) => handleUseAgentPromptToCanvas(payload.prompt, payload)}
              onQuickGenerate={(payload) => {
                applyPromptToGenerate(payload.prompt, payload)
                setMessage('Agent 已填入快捷生成')
              }}
            />
          </section>
        ) : null}

        {activeTab === 'assistant' ? (
          <section className="workflow-page assistant-page">
            <PromptAssistantModal
              embedded
              tasks={tasks}
              uploads={uploads}
              provider={provider}
              bananaModel={bananaModel}
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
            <PageHeader eyebrow="Settings" title="设置" description="Key 保存在当前浏览器本地；默认数量、默认并发和图床设置随账号保存。" />
            <div className="settings-inline-grid settings-only-grid">
              <SettingsPanel onConfig={applyUserConfig} />
            </div>
          </section>
        ) : null}
          </main>
        </div>
      </div>

      <MobileWorkbenchTabs tabs={mobileTabs} activeTab={activeTab} moreActive={mobileMoreActive} moreOpen={mobileMoreOpen} onChange={handleMobileNavChange} onMore={toggleMobileMore} />

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

function defaultSquareTags(task: Task) {
  return [
    task.mode === 'gif' ? 'GIF动图' : task.mode === 'image-to-image' ? '图生图' : '文生图',
    task.provider || 'image-2',
    task.model || '',
  ].filter(Boolean).slice(0, 6)
}

function splitSubmitTags(value: string) {
  return value.split(/[,，\s]+/).map((item) => item.trim()).filter(Boolean).slice(0, 12)
}

function formatCredits(value: number | undefined) {
  return Number.isFinite(value) ? Number(value).toLocaleString() : '0'
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

function WorkbenchSidebar({
  tabs,
  activeTab,
  user,
  creditsBalance,
  theme,
  onThemeChange,
  onLogout,
  onChange,
}: {
  tabs: WorkbenchTabItem[]
  activeTab: WorkbenchTab
  user: UserSession['user']
  creditsBalance: number
  theme: ThemeMode
  onThemeChange: (theme?: ThemeMode) => void
  onLogout: () => void
  onChange: (tab: WorkbenchNavId) => void
}) {
  return (
    <aside className="workbench-sidebar" aria-label="桌面端工作台导航">
      <div className="workbench-sidebar-brand" aria-label="Lyra Image Workbench">
        <span>Ly</span>
        <strong>Lyra Image Workbench</strong>
      </div>
      <nav className="workbench-sidebar-nav" aria-label="工作台导航">
        {tabs.map((tab) => {
          const active = activeTab === tab.id
          const className = `workbench-sidebar-button ${active ? 'active' : ''} ${tab.tone ? `tone-${tab.tone}` : ''}`.trim()
          return (
            <button key={tab.id} type="button" className={className} aria-current={active ? 'page' : undefined} onClick={() => onChange(tab.id)}>
              <span className="workbench-sidebar-indicator" aria-hidden="true" />
              <span className="workbench-sidebar-label">
                <strong>{tab.label}</strong>
              </span>
            </button>
          )
        })}
      </nav>
      <div className="workbench-sidebar-footer" aria-label="工作台工具">
        <div className="workbench-sidebar-account">
          <div>
            <span>当前登录</span>
            <strong>{user.displayName || user.username}</strong>
            {user.displayName && user.displayName !== user.username ? <small>{user.username}</small> : null}
          </div>
          <button type="button" className="workbench-sidebar-logout" onClick={onLogout}>退出登录</button>
        </div>
        <div className="workbench-sidebar-balance">
          <span>余额</span>
          <strong>{formatCredits(creditsBalance)} 次</strong>
        </div>
        <div className="workbench-sidebar-tools">
          <GitHubLink />
          <ThemeToggle theme={theme} onToggle={onThemeChange} />
        </div>
      </div>
    </aside>
  )
}
function MobileWorkbenchTabs({ tabs, activeTab, moreActive, moreOpen, onChange, onMore }: { tabs: WorkbenchMobileTabItem[]; activeTab: WorkbenchTab; moreActive: boolean; moreOpen: boolean; onChange: (tab: WorkbenchNavId) => void; onMore: () => void }) {
  return (
    <nav className="workflow-tabs mobile-tabs" aria-label="移动端工作流导航">
      {tabs.map((tab) => {
        const isMore = tab.id === 'more'
        const active = isMore ? moreActive : activeTab === tab.id
        const className = `${active ? 'active' : ''} ${tab.tone ? `tone-${tab.tone}` : ''}`.trim()
        return (
          <button
            key={tab.id}
            type="button"
            aria-current={active ? 'page' : undefined}
            aria-expanded={isMore ? moreOpen : undefined}
            aria-haspopup={isMore ? 'dialog' : undefined}
            aria-controls={isMore ? 'mobile-more-sheet' : undefined}
            className={className}
            onClick={() => { if (tab.id === 'more') onMore(); else onChange(tab.id) }}
          >
            <strong>{tab.label}</strong>
          </button>
        )
      })}
    </nav>
  )
}

function extensionFromMime(mime: string) {
  if (mime.includes('jpeg')) return 'jpg'
  if (mime.includes('webp')) return 'webp'
  if (mime.includes('gif')) return 'gif'
  return 'png'
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
