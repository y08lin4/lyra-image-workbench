import { type FormEvent, useCallback, useEffect, useMemo, useState } from 'react'
import { cancelTask, createTask, listTasks, retryTask, uploadTaskImageToPixhost } from '../api/tasks'
import { clearSpaceToken, getSpaceToken } from '../api/client'
import { getCurrentSpace, leaveSpace } from '../api/spaces'
import { deleteReferenceUpload, listReferenceUploads, uploadReferenceImages } from '../api/uploads'
import { getUserConfig } from '../api/config'
import type { CreateTaskRequest, Mode, ReferenceUpload, SpaceSession, Task, TaskEvent, UserConfig } from '../types'
import { SpaceLogin } from './SpaceLogin'
import { GenerationPanel } from './GenerationPanel'
import { ResultCanvas } from './ResultCanvas'
import { TaskTimeline } from './TaskTimeline'
import { SettingsWindow } from './SettingsWindow'
import { useTaskEvents } from '../hooks/useTaskEvents'

type NumericInputValue = number | ''

export function WorkbenchPage() {
  const [session, setSession] = useState<SpaceSession | null>(null)
  const [spaceReady, setSpaceReady] = useState(false)
  const [keyReady, setKeyReady] = useState(false)
  const [keyPreview, setKeyPreview] = useState('')
  const [settingsOpen, setSettingsOpen] = useState(false)
  const [tasks, setTasks] = useState<Task[]>([])
  const [activeId, setActiveId] = useState<string | null>(null)
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

  const activeTask = useMemo(() => tasks.find((task) => task.id === activeId), [tasks, activeId])
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
    if (event.event !== 'heartbeat') setMessage(`${event.chinese} / ${event.english} / ${event.code}`)
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

  async function refreshTasks() {
    const items = await listTasks()
    setTasks(items)
    if (!activeId && items[0]) setActiveId(items[0].id)
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
      setMessage('任务已提交，后端会继续执行')
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

  async function logout() {
    await leaveSpace()
    setSession(null)
    setSpaceReady(false)
  }

  if (!session) return <SpaceLogin onSession={(next) => { setSession(next); setSpaceReady(true) }} />

  return (
    <div className="app-shell">
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
      <main className="workspace">
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
        <ResultCanvas task={activeTask} onUseAsReference={handleUseResultAsReference} onUploadPixhost={handleUploadPixhost} />
        <TaskTimeline tasks={tasks} activeId={activeId || undefined} onSelect={setActiveId} onRetry={(id) => void retryTask(id).then(upsertTask)} onCancel={(id) => void cancelTask(id).then(upsertTask)} />
      </main>
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
