import { type FormEvent, useCallback, useEffect, useMemo, useState } from 'react'
import { cancelTask, createTask, deleteTask, listTasks, retryTask, setTaskFavorite, uploadTaskImageToPixhost } from '../api/tasks'
import { clearSpaceToken, getSpaceToken } from '../api/client'
import { getCurrentSpace, leaveSpace } from '../api/spaces'
import { deleteReferenceUpload, listReferenceUploads, uploadReferenceImages } from '../api/uploads'
import { getUserConfig } from '../api/config'
import type { CreateTaskRequest, Mode, ModelProvider, ReferenceUpload, SpaceSession, Task, TaskEvent, TaskStatus, UserConfig } from '../types'
import { SpaceLogin } from './SpaceLogin'
import { GenerationPanel } from './GenerationPanel'
import { SettingsWindow } from './SettingsWindow'
import { TaskDetailModal } from './TaskDetailModal'
import { TaskGallery } from './TaskGallery'
import { PromptAssistantModal } from './PromptAssistantModal'
import { useTaskEvents } from '../hooks/useTaskEvents'
import { BANANA_PROVIDER, DEFAULT_BANANA_MODEL, DEFAULT_IMAGE2_MODEL, getBananaModelOption } from '../lib/models'

type NumericInputValue = number | ''
type TaskFilter = TaskStatus | 'all'

export function WorkbenchPage() {
  const [session, setSession] = useState<SpaceSession | null>(null)
  const [spaceReady, setSpaceReady] = useState(false)
  const [keyReady, setKeyReady] = useState(false)
  const [keyPreview, setKeyPreview] = useState('')
  const [bananaKeyReady, setBananaKeyReady] = useState(false)
  const [bananaKeyPreview, setBananaKeyPreview] = useState('')
  const [settingsOpen, setSettingsOpen] = useState(false)
  const [promptAssistantOpen, setPromptAssistantOpen] = useState(false)
  const [tasks, setTasks] = useState<Task[]>([])
  const [activeId, setActiveId] = useState<string | null>(null)
  const [detailId, setDetailId] = useState<string | null>(null)
  const [searchQuery, setSearchQuery] = useState('')
  const [statusFilter, setStatusFilter] = useState<TaskFilter>('all')
  const [favoriteOnly, setFavoriteOnly] = useState(false)
  const [selectedIds, setSelectedIds] = useState<Set<string>>(() => new Set())
  const [uploads, setUploads] = useState<ReferenceUpload[]>([])
  const [mode, setMode] = useState<Mode>('text-to-image')
  const [provider, setProvider] = useState<ModelProvider>('image-2')
  const [bananaModel, setBananaModel] = useState(DEFAULT_BANANA_MODEL)
  const [prompt, setPrompt] = useState('')
  const [ratio, setRatio] = useState('1:1')
  const [resolution, setResolution] = useState('standard')
  const [quality, setQuality] = useState('auto')
  const [outputFormat, setOutputFormat] = useState('png')
  const [count, setCount] = useState<NumericInputValue>(1)
  const [concurrency, setConcurrency] = useState<NumericInputValue>(1)
  const [message, setMessage] = useState('')
  const [error, setError] = useState('')

  const detailTask = useMemo(() => tasks.find((task) => task.id === detailId), [tasks, detailId])
  const favoriteIds = useMemo(() => new Set(tasks.filter((task) => task.favorite).map((task) => task.id)), [tasks])
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
    const token = getSpaceToken()
    if (!token) return
    void getCurrentSpace(token).then((next) => { setSession(next); setSpaceReady(true) }).catch(() => { clearSpaceToken(); setSpaceReady(false) })
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
    setConcurrency(cfg.defaultConcurrency || 1)
  }, [])

  async function submit(event: FormEvent) {
    event.preventDefault()
    setError('')
    const currentKeyReady = provider === BANANA_PROVIDER ? bananaKeyReady : keyReady
    if (!currentKeyReady) {
      setError(provider === BANANA_PROVIDER ? '请先在设置里保存 banana 分组的 API Key' : '请先保存当前空间的 codex-key')
      return
    }
    if (!prompt.trim()) { setError('请先输入提示词'); return }
    if (mode === 'image-to-image' && uploads.length === 0) { setError('图生图需要先上传参考图'); return }
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
      uploadIds: mode === 'image-to-image' ? uploads.map((item) => item.id) : [],
    }
    try {
      const job = await createTask(payload)
      upsertTask(job)
      setActiveId(job.id)
      setPrompt('')
      setMessage('任务已提交，后端会继续执行，前端可刷新或断开')
    } catch (err) {
      setError(err instanceof Error ? err.message : '提交失败')
    }
  }

  async function handleUpload(files: File[]) {
    if (!files.length) return
    setUploads([...(await uploadReferenceImages(files)), ...(await listReferenceUploads())])
    await refreshUploads()
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
    setMessage('已作为图生图参考图')
  }

  async function handleUploadPixhost(taskId: string, index: number) {
    const data = await uploadTaskImageToPixhost(taskId, index)
    upsertTask(data.job)
    setMessage(data.result.remoteUrl ? 'PiXhost 图床上传成功' : 'PiXhost 图床上传完成')
  }

  function handleUseAssistantPrompt(nextPrompt: string) {
    setPrompt(nextPrompt)
    setPromptAssistantOpen(false)
    setMessage('提示词助手已填入主输入框')
    window.setTimeout(() => {
      document.querySelector('[data-generation-composer] textarea')?.scrollIntoView({ behavior: 'smooth', block: 'center' })
    }, 0)
  }

  function handleSelectTask(task: Task) {
    setActiveId(task.id)
    setDetailId(task.id)
  }

  function handleReuseTask(task: Task) {
    const nextProvider = task.provider || 'image-2'
    setProvider(nextProvider)
    setBananaModel(nextProvider === BANANA_PROVIDER ? getBananaModelOption(task.model || '').id : bananaModel)
    setMode(task.mode)
    setPrompt(task.prompt)
    setRatio(task.ratio || '1:1')
    setResolution(task.resolution || 'standard')
    setQuality(task.quality || 'auto')
    setOutputFormat(task.outputFormat || 'png')
    setCount(task.count || 1)
    setConcurrency(task.concurrency || 1)
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
    const job = await retryTask(id)
    upsertTask(job)
    setActiveId(job.id)
    setDetailId(job.id)
    setMessage('已重新提交任务')
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

  function handleBatchDownload() {
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
    items.forEach((item, index) => {
      window.setTimeout(() => downloadURL(item.url, item.name), index * 120)
    })
    setMessage(`已触发 ${items.length} 张图片下载`)
  }

  async function logout() {
    await leaveSpace()
    setSession(null)
    setSpaceReady(false)
  }

  if (!session) return <SpaceLogin onSession={(next) => { setSession(next); setSpaceReady(true) }} />

  return (
    <div className="app-shell gallery-shell">
      <header className="topbar">
        <div className="brand">
          <div className="brand-mark">AI</div>
          <div>
            <h1>本机生图工作台</h1>
            <p>{session.space.displayName} · {session.tokenPreview}</p>
          </div>
        </div>
        <nav className="top-actions"><button type="button" onClick={() => setSettingsOpen(true)}>设置</button><a className="ghost-link" href="/admin">Admin</a><button onClick={logout}>退出空间</button></nav>
      </header>
      <main className="gallery-workspace">
        <TaskGallery
          tasks={tasks}
          activeId={activeId || undefined}
          query={searchQuery}
          statusFilter={statusFilter}
          favoriteOnly={favoriteOnly}
          favoriteIds={favoriteIds}
          selectedIds={selectedIds}
          onQueryChange={setSearchQuery}
          onStatusFilterChange={setStatusFilter}
          onFavoriteOnlyChange={setFavoriteOnly}
          onToggleSelect={toggleSelectTask}
          onSelectVisible={selectVisibleTasks}
          onClearSelection={() => setSelectedIds(new Set())}
          onBatchFavorite={(favorite) => void handleBatchFavorite(favorite)}
          onBatchDelete={() => void handleBatchDelete()}
          onBatchDownload={handleBatchDownload}
          onSelect={handleSelectTask}
          onRetry={(id) => void handleRetry(id)}
          onCancel={(id) => void handleCancel(id)}
          onDelete={(id) => void handleDelete(id)}
          onReuse={handleReuseTask}
          onToggleFavorite={(id) => void toggleFavorite(id)}
        />
      </main>
      <div className="composer-dock" data-generation-composer>
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
            keyReady={provider === BANANA_PROVIDER ? bananaKeyReady : keyReady}
            keyPreview={provider === BANANA_PROVIDER ? bananaKeyPreview : keyPreview}
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
            onOpenSettings={() => setSettingsOpen(true)}
            onOpenPromptAssistant={() => setPromptAssistantOpen(true)}
            onUpload={handleUpload}
            onDeleteUpload={handleDeleteUpload}
            onSubmit={submit}
          />
      </div>
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
      {settingsOpen ? <SettingsWindow onClose={() => setSettingsOpen(false)} onConfig={applyUserConfig} /> : null}
      {promptAssistantOpen ? (
        <PromptAssistantModal
          tasks={tasks}
          uploads={uploads}
          onClose={() => setPromptAssistantOpen(false)}
          onUsePrompt={handleUseAssistantPrompt}
          onRefreshUploads={refreshUploads}
        />
      ) : null}
    </div>
  )
}

function extensionFromMime(mime: string) {
  if (mime.includes('jpeg')) return 'jpg'
  if (mime.includes('webp')) return 'webp'
  if (mime.includes('gif')) return 'gif'
  return 'png'
}

function downloadURL(url: string, filename: string) {
  const anchor = document.createElement('a')
  anchor.href = url
  anchor.download = filename
  anchor.rel = 'noopener'
  document.body.appendChild(anchor)
  anchor.click()
  anchor.remove()
}

function numericOrDefault(value: NumericInputValue, fallback: number) {
  return value === '' ? fallback : value
}

function isFinal(task: Task) {
  return ['succeeded', 'partial_failed', 'failed', 'cancelled', 'interrupted'].includes(task.status)
}
