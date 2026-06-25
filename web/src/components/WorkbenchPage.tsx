import { type FormEvent, useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { cancelTask, createTask, deleteTask, listTasks, retryTask, setTaskFavorite, uploadTaskImageToPixhost } from '../api/tasks'
import { getCurrentUser, logoutUser } from '../api/users'
import { deleteReferenceUpload, listReferenceUploads, uploadReferenceImages } from '../api/uploads'
import { getUserConfig } from '../api/config'
import type { CreateTaskRequest, Mode, ModelProvider, ReferenceUpload, Task, TaskEvent, UserConfig, UserSession } from '../types'
import { SpaceLogin } from './SpaceLogin'
import { GenerationPanel } from './GenerationPanel'
import { SettingsPanel } from './SettingsPanel'
import { TaskDetailModal } from './TaskDetailModal'
import { TaskSidebar } from './TaskSidebar'
import { PromptAssistantModal } from './PromptAssistantModal'
import { PromptLibraryPage } from './PromptLibraryPage'
import { PromptSquarePanel } from './PromptSquarePanel'
import { ResultCanvas } from './ResultCanvas'
import { ThemeToggle, type ThemeMode } from './ThemeToggle'
import { GitHubLink } from './GitHubLink'
import { useTaskEvents } from '../hooks/useTaskEvents'
import { BANANA_PROVIDER, DEFAULT_BANANA_MODEL, DEFAULT_IMAGE2_MODEL, getBananaModelForRatio, getBananaModelOption } from '../lib/models'
import { formatBytes } from '../lib/format'
import { nativeExitApp, nativeSaveImage } from '../lib/nativeBridge'
import { ensureAppBackBridge, installEdgeBackGesture, registerAppBackHandler } from '../lib/appBack'

type NumericInputValue = number | ''
type WorkbenchTab = 'generate' | 'library' | 'square' | 'result' | 'queue' | 'settings'
type WorkbenchTabItem = { id: WorkbenchTab; label: string; hint: string; badge?: string; tone?: 'normal' | 'danger' | 'active' }

const MAX_REFERENCE_IMAGES = 8
const MAX_REFERENCE_IMAGE_BYTES = 12 * 1024 * 1024
const MAX_REFERENCE_UPLOAD_BYTES = 50 * 1024 * 1024
const ALLOWED_REFERENCE_TYPES = new Set(['image/png', 'image/jpeg', 'image/webp'])

const workflowTabs: WorkbenchTabItem[] = [
  { id: 'generate', label: '生成', hint: '请求' },
  { id: 'library', label: '提示词库', hint: '灵感' },
  { id: 'square', label: '广场', hint: 'Prompt' },
  { id: 'result', label: '结果', hint: '图片' },
  { id: 'queue', label: '队列', hint: '历史' },
  { id: 'settings', label: '设置', hint: 'Key' },
]

export function WorkbenchPage({ theme, onToggleTheme }: { theme: ThemeMode; onToggleTheme: () => void }) {
  const [session, setSession] = useState<UserSession | null>(null)
  const [spaceReady, setSpaceReady] = useState(false)
  const [activeTab, setActiveTab] = useState<WorkbenchTab>('generate')
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
  const activeTabRef = useRef(activeTab)
  const detailIdRef = useRef(detailId)
  const tabHistoryRef = useRef<WorkbenchTab[]>([])
  const lastExitBackAtRef = useRef(0)

  activeTabRef.current = activeTab
  detailIdRef.current = detailId

  const detailTask = useMemo(() => tasks.find((task) => task.id === detailId), [tasks, detailId])
  const activeTask = useMemo(() => tasks.find((task) => task.id === activeId), [tasks, activeId])
  const favoriteIds = useMemo(() => new Set(tasks.filter((task) => task.favorite).map((task) => task.id)), [tasks])
  const currentKeyReady = provider === BANANA_PROVIDER ? bananaKeyReady : keyReady
  const currentKeyPreview = provider === BANANA_PROVIDER ? bananaKeyPreview : keyPreview
  const activeCount = useMemo(() => tasks.filter((task) => !isFinal(task)).length, [tasks])
  const missingKeyCount = (keyReady ? 0 : 1) + (bananaKeyReady ? 0 : 1)
  const tabItems = useMemo<WorkbenchTabItem[]>(() => workflowTabs.map((tab) => {
    if (tab.id === 'generate') return { ...tab, hint: currentKeyReady ? '可提交' : '缺 Key', tone: currentKeyReady ? 'normal' : 'danger' }
    if (tab.id === 'library') return { ...tab, hint: 'GitHub', tone: 'normal' }
    if (tab.id === 'square') return { ...tab, hint: '试验版' }
    if (tab.id === 'result') return { ...tab, hint: activeTask ? activeTask.statusText : '图片', badge: activeTask ? `${activeTask.progress}%` : undefined, tone: activeTask && !isFinal(activeTask) ? 'active' : 'normal' }
    if (tab.id === 'queue') return { ...tab, hint: activeCount ? `${activeCount} 进行中` : '历史', badge: activeCount ? String(activeCount) : undefined, tone: activeCount ? 'active' : 'normal' }
    if (tab.id === 'settings') return { ...tab, hint: missingKeyCount ? `${missingKeyCount} 个未设` : '已配置', badge: missingKeyCount ? '!' : undefined, tone: missingKeyCount ? 'danger' : 'normal' }
    return tab
  }), [activeCount, activeTask, currentKeyReady, missingKeyCount])

  const goToTab = useCallback((nextTab: WorkbenchTab, options?: { replace?: boolean }) => {
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

      if (current !== 'generate') {
        setActiveTab('generate')
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
    void getCurrentUser().then((next) => { setSession(next); setSpaceReady(true) }).catch(() => { setSpaceReady(false) })
  }, [])

  useEffect(() => {
    if (!spaceReady) return
    void refreshTasks()
    void refreshUploads()
    void refreshUserConfig()
  }, [spaceReady])

  useEffect(() => {
    if (!spaceReady || !tasks.some((task) => !isFinal(task))) return
    const timer = window.setInterval(() => {
      void refreshTasks()
    }, 5000)
    return () => window.clearInterval(timer)
  }, [spaceReady, tasks])

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
    if (mode === 'image-to-image' && uploads.length === 0) { setError('图生图需要先上传参考图'); return }
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
      setMessage(submittedUploads.length ? '任务已提交，参考图已保留，可在生成页手动删除' : '任务已提交，后端会继续执行，前端可刷新或断开')
    } catch (err) {
      setError(err instanceof Error ? err.message : '提交失败')
    }
  }

  async function handleUpload(files: File[]) {
    if (!files.length) return
    const validation = validateReferenceFiles(files, uploads.length)
    if (validation) {
      setToast(validation)
      return
    }
    try {
      const created = await uploadReferenceImages(files)
      await refreshUploads()
      setToast(`已上传 ${created.length} 张参考图`)
    } catch (err) {
      setToast(formatReferenceUploadError(err))
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
    await uploadReferenceImages([file])
    await refreshUploads()
    setMode('image-to-image')
    goToTab('generate')
    setMessage('已作为图生图参考图')
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

  function handleUseLibraryPrompt(nextPrompt: string, options: { provider: ModelProvider; model: string }) {
    setPrompt(nextPrompt)
    setProvider(options.provider)
    if (options.provider === BANANA_PROVIDER) {
      setBananaModel(getBananaModelOption(options.model).id)
    }
    goToTab('generate')
    setMessage('提示词库已填入主输入框，并同步模型选择')
    window.setTimeout(() => {
      document.querySelector('[data-generation-composer] textarea')?.scrollIntoView({ behavior: 'smooth', block: 'center' })
    }, 0)
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
    <div className="app-shell gallery-shell tabbed-workbench">
      <header className="topbar workbench-topbar">
        <div className="brand">
          <div className="brand-mark">Ly</div>
          <div>
            <h1>Lyra Image Workbench</h1>
            <p>{session.user.displayName} · {session.user.username}</p>
          </div>
        </div>
        <div className="top-status" aria-label="当前状态">
          <span className={keyReady ? 'ready' : 'missing'}>codex-key {keyReady ? '已设置' : '未设置'}</span>
          <span className={bananaKeyReady ? 'ready' : 'missing'}>Banana {bananaKeyReady ? '已设置' : '未设置'}</span>
          <span className="ready">后端在线</span>
        </div>
        <nav className="top-actions">
          <GitHubLink />
          <ThemeToggle theme={theme} onToggle={onToggleTheme} />
          <button onClick={logout}>退出登录</button>
        </nav>
      </header>

      <div className="api-service-banner">
        <strong>API 服务</strong>
        <span>当前前端通过同源 /api 访问本机后端；对外域名可在 Admin 页记录。</span>
      </div>

      <WorkbenchTabs tabs={tabItems} activeTab={activeTab} onChange={goToTab} className="workflow-tabs desktop-tabs" />

      <main className={`workflow-content workflow-${activeTab}`}>
        {activeTab === 'generate' ? (
          <section className="workflow-page generate-page" data-generation-composer>
            <PageHeader eyebrow="Generate" title="生成请求" description="生成和提示词助手已合并在同一页：左侧提交任务，右侧生成/修改提示词，填入后直接回到主输入框。" />
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
              <aside className="generate-assistant-panel" data-prompt-assistant-panel>
                <PromptAssistantModal
                  embedded
                  tasks={tasks}
                  uploads={uploads}
                  provider={provider}
                  bananaModel={bananaModel}
                  onClose={() => goToTab('generate')}
                  onUsePrompt={handleUseAssistantPrompt}
                  onRefreshUploads={refreshUploads}
                />
              </aside>
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

        {activeTab === 'result' ? (
          <section className="workflow-page result-page">
            <ResultCanvas
              task={activeTask}
              onUseAsReference={handleUseResultAsReference}
              onUploadPixhost={handleUploadPixhost}
              onOpenGenerate={() => goToTab('generate')}
              onOpenQueue={() => goToTab('queue')}
              onReuse={handleReuseTask}
              onRetry={(id) => void handleRetry(id)}
            />
          </section>
        ) : null}

        {activeTab === 'square' ? (
          <section className="workflow-page prompt-square-page">
            <PromptSquarePanel onUsePrompt={handleUseSquarePrompt} />
          </section>
        ) : null}

        {activeTab === 'queue' ? (
          <section className="workflow-page queue-page">
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
          </section>
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

      <WorkbenchTabs tabs={tabItems} activeTab={activeTab} onChange={goToTab} className="workflow-tabs mobile-tabs" />

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
          onUseAsReference={handleUseResultAsReference}
          onUploadPixhost={handleUploadPixhost}
        />
      ) : null}
    </div>
  )
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

function WorkbenchTabs({ tabs = workflowTabs, activeTab, onChange, className }: { tabs?: WorkbenchTabItem[]; activeTab: WorkbenchTab; onChange: (tab: WorkbenchTab) => void; className: string }) {
  return (
    <nav className={className} aria-label="工作流标签页">
      {tabs.map((tab) => (
        <button key={tab.id} type="button" className={`${activeTab === tab.id ? 'active' : ''} ${tab.tone ? `tone-${tab.tone}` : ''}`} onClick={() => onChange(tab.id)}>
          <strong>{tab.label}{tab.badge ? <i>{tab.badge}</i> : null}</strong>
          <span>{tab.hint}</span>
        </button>
      ))}
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
