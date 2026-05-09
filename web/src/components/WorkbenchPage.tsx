import { type FormEvent, useCallback, useEffect, useMemo, useState } from 'react'
import { cancelTask, createTask, listTasks, retryTask, setTaskFavorite, uploadTaskImageToPixhost } from '../api/tasks'
import { clearSpaceToken, getSpaceToken } from '../api/client'
import { getCurrentSpace, leaveSpace } from '../api/spaces'
import { deleteReferenceUpload, listReferenceUploads, uploadReferenceImages } from '../api/uploads'
import { getUserConfig } from '../api/config'
import type { CreateTaskRequest, Mode, ReferenceUpload, SpaceSession, Task, TaskEvent, TaskStatus, UserConfig } from '../types'
import { SpaceLogin } from './SpaceLogin'
import { GenerationPanel } from './GenerationPanel'
import { SettingsWindow } from './SettingsWindow'
import { TaskDetailModal } from './TaskDetailModal'
import { TaskGallery } from './TaskGallery'
import { useTaskEvents } from '../hooks/useTaskEvents'

type NumericInputValue = number | ''
type TaskFilter = TaskStatus | 'all'

export function WorkbenchPage() {
  const [session, setSession] = useState<SpaceSession | null>(null)
  const [spaceReady, setSpaceReady] = useState(false)
  const [keyReady, setKeyReady] = useState(false)
  const [keyPreview, setKeyPreview] = useState('')
  const [settingsOpen, setSettingsOpen] = useState(false)
  const [tasks, setTasks] = useState<Task[]>([])
  const [activeId, setActiveId] = useState<string | null>(null)
  const [detailId, setDetailId] = useState<string | null>(null)
  const [searchQuery, setSearchQuery] = useState('')
  const [statusFilter, setStatusFilter] = useState<TaskFilter>('all')
  const [favoriteOnly, setFavoriteOnly] = useState(false)
  const [uploads, setUploads] = useState<ReferenceUpload[]>([])
  const [mode, setMode] = useState<Mode>('text-to-image')
  const [prompt, setPrompt] = useState('')
  const [ratio, setRatio] = useState('1:1')
  const [resolution, setResolution] = useState('standard')
  const [quality, setQuality] = useState('auto')
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
    setConcurrency(cfg.defaultConcurrency || 1)
  }, [])

  async function submit(event: FormEvent) {
    event.preventDefault()
    setError('')
    if (!keyReady) { setError('请先保存当前空间的 Image-2 Key'); return }
    if (!prompt.trim()) { setError('请先输入提示词'); return }
    if (mode === 'image-to-image' && uploads.length === 0) { setError('图生图需要先上传参考图'); return }
    const payload: CreateTaskRequest = {
      mode,
      prompt,
      ratio,
      resolution,
      quality,
      count: numericOrDefault(count, 1),
      concurrency: numericOrDefault(concurrency, 1),
      uploadIds: mode === 'image-to-image' ? uploads.map((item) => item.id) : [],
    }
    try {
      const job = await createTask(payload)
      upsertTask(job)
      setActiveId(job.id)
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

  function handleSelectTask(task: Task) {
    setActiveId(task.id)
    setDetailId(task.id)
  }

  function handleReuseTask(task: Task) {
    setMode(task.mode)
    setPrompt(task.prompt)
    setRatio(task.ratio || '1:1')
    setResolution(task.resolution || 'standard')
    setQuality(task.quality || 'auto')
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
          onQueryChange={setSearchQuery}
          onStatusFilterChange={setStatusFilter}
          onFavoriteOnlyChange={setFavoriteOnly}
          onSelect={handleSelectTask}
          onRetry={(id) => void handleRetry(id)}
          onCancel={(id) => void handleCancel(id)}
          onReuse={handleReuseTask}
          onToggleFavorite={(id) => void toggleFavorite(id)}
        />
      </main>
      <div className="composer-dock" data-generation-composer>
        <GenerationPanel
            mode={mode}
            prompt={prompt}
            ratio={ratio}
            resolution={resolution}
            quality={quality}
            count={count}
            concurrency={concurrency}
            uploads={uploads}
            keyReady={keyReady}
            keyPreview={keyPreview}
            message={message}
            error={error}
            onModeChange={setMode}
            onPromptChange={setPrompt}
            onRatioChange={setRatio}
            onResolutionChange={setResolution}
            onQualityChange={setQuality}
            onCountChange={setCount}
            onConcurrencyChange={setConcurrency}
            onOpenSettings={() => setSettingsOpen(true)}
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
          onReuse={handleReuseTask}
          onToggleFavorite={(id) => void toggleFavorite(id)}
          onUseAsReference={handleUseResultAsReference}
          onUploadPixhost={handleUploadPixhost}
        />
      ) : null}
      {settingsOpen ? <SettingsWindow onClose={() => setSettingsOpen(false)} onConfig={applyUserConfig} /> : null}
    </div>
  )
}

function extensionFromMime(mime: string) {
  if (mime.includes('jpeg')) return 'jpg'
  if (mime.includes('webp')) return 'webp'
  if (mime.includes('gif')) return 'gif'
  return 'png'
}

function numericOrDefault(value: NumericInputValue, fallback: number) {
  return value === '' ? fallback : value
}

function isFinal(task: Task) {
  return ['succeeded', 'partial_failed', 'failed', 'cancelled', 'interrupted'].includes(task.status)
}
