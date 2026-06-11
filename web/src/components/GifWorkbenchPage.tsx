import { type ChangeEvent, useCallback, useEffect, useMemo, useState } from 'react'
import { createTask } from '../api/tasks'
import { getCurrentUser, logoutUser } from '../api/users'
import { getUserConfig } from '../api/config'
import { deleteReferenceUpload, getReferenceUploadBlob, uploadReferenceImages } from '../api/uploads'
import { createGifPlan, createGifRender, getGifStatus, type GifMotionType, type GifPlan, type GifStrength, type GifStatus, type GifRender } from '../api/gif'
import type { ReferenceUpload, Task, TaskEvent, UserSession } from '../types'
import { DEFAULT_IMAGE2_MODEL } from '../lib/models'
import { formatBytes } from '../lib/format'
import { useTaskEvents } from '../hooks/useTaskEvents'
import { SpaceLogin } from './SpaceLogin'
import { ThemeToggle, type ThemeMode } from './ThemeToggle'
import { GitHubLink } from './GitHubLink'

const MOTIONS: Array<{ id: GifMotionType; label: string; hint: string }> = [
  { id: 'blink', label: '眨眼', hint: '睁眼 → 闭眼 → 睁眼' },
  { id: 'smile', label: '微笑', hint: '中性 → 微笑 → 回落' },
  { id: 'turn_head', label: '轻微转头', hint: '正面轻转再回正' },
  { id: 'hair_flow', label: '头发飘动', hint: '发丝轻微摆动' },
  { id: 'custom', label: '自定义', hint: '输入你的动作' },
]

const STRENGTHS: Array<{ id: GifStrength; label: string }> = [
  { id: 'subtle', label: '轻微' },
  { id: 'medium', label: '中等' },
  { id: 'strong', label: '明显' },
]

const ALLOWED_TYPES = new Set(['image/png', 'image/jpeg', 'image/webp'])
const MAX_IMAGE_BYTES = 12 * 1024 * 1024

type WorkflowStep = 'editing_image' | 'editing_prompt' | 'planning' | 'frames_generating' | 'frames_ready' | 'rendering' | 'gif_ready' | 'failed'

export function GifWorkbenchPage({ theme, onToggleTheme }: { theme: ThemeMode; onToggleTheme: () => void }) {
  const [session, setSession] = useState<UserSession | null>(null)
  const [status, setStatus] = useState<GifStatus | null>(null)
  const [keyReady, setKeyReady] = useState(false)
  const [upload, setUpload] = useState<ReferenceUpload | null>(null)
  const [previewUrl, setPreviewUrl] = useState('')
  const [motionType, setMotionType] = useState<GifMotionType>('blink')
  const [prompt, setPrompt] = useState('让人物自然眨眼并轻微微笑，保持同一个人、同一背景、同一构图')
  const [frameCount, setFrameCount] = useState(12)
  const [fps, setFPS] = useState(8)
  const [strength, setStrength] = useState<GifStrength>('subtle')
  const [width, setWidth] = useState(512)
  const [loop, setLoop] = useState(true)
  const [plan, setPlan] = useState<GifPlan | null>(null)
  const [planningFallback, setPlanningFallback] = useState(false)
  const [frameTask, setFrameTask] = useState<Task | null>(null)
  const [activeTaskId, setActiveTaskId] = useState<string | null>(null)
  const [render, setRender] = useState<GifRender | null>(null)
  const [step, setStep] = useState<WorkflowStep>('editing_image')
  const [message, setMessage] = useState('')
  const [error, setError] = useState('')

  const upsertTask = useCallback((task: Task) => {
    setFrameTask(task)
    if (isFinal(task)) setStep(task.results.filter((item) => item.ok).length >= 2 ? 'frames_ready' : 'failed')
  }, [])

  const handleTaskEvent = useCallback((event: TaskEvent) => {
    if (event.event !== 'heartbeat') setMessage(`${event.chinese} / ${event.code}`)
  }, [])

  useTaskEvents(activeTaskId, upsertTask, handleTaskEvent)

  useEffect(() => {
    void getCurrentUser().then(setSession).catch(() => setSession(null))
  }, [])

  useEffect(() => {
    if (!session) return
    void Promise.all([
      getGifStatus().then(setStatus),
      getUserConfig().then((cfg) => setKeyReady(cfg.apiKeySet)),
    ]).catch((err) => setError(err instanceof Error ? err.message : '初始化 GIF 页面失败'))
  }, [session])

  useEffect(() => {
    if (!upload) {
      setPreviewUrl('')
      return
    }
    let disposed = false
    let objectUrl = ''
    void getReferenceUploadBlob(upload.id).then((blob) => {
      if (disposed) return
      objectUrl = URL.createObjectURL(blob)
      setPreviewUrl(objectUrl)
    }).catch(() => setPreviewUrl(''))
    return () => {
      disposed = true
      if (objectUrl) URL.revokeObjectURL(objectUrl)
    }
  }, [upload])

  const successfulFrames = useMemo(() => {
    return (frameTask?.results || [])
      .filter((result) => result.ok && result.imageUrl)
      .sort((a, b) => a.index - b.index)
  }, [frameTask])

  const canRender = Boolean(status?.gifEnabled && status.ffmpegAvailable && successfulFrames.length >= 2 && frameTask && isFinal(frameTask))
  const ffmpegWarning = status && (!status.gifEnabled || !status.ffmpegAvailable)
    ? 'GIF 合成功能未启用：服务器未安装 FFmpeg 或 GIF_ENABLED=false。请安装 FFmpeg 后重启服务，或设置 FFMPEG_BIN。'
    : ''

  async function logout() {
    await logoutUser()
    setSession(null)
  }

  async function handleUpload(event: ChangeEvent<HTMLInputElement>) {
    const file = event.currentTarget.files?.[0]
    event.currentTarget.value = ''
    if (!file) return
    setError('')
    if (!ALLOWED_TYPES.has(file.type)) {
      setError('参考图仅支持 PNG / JPG / WEBP')
      return
    }
    if (file.size > MAX_IMAGE_BYTES) {
      setError('单张参考图不能超过 12MB')
      return
    }
    try {
      if (upload) await deleteReferenceUpload(upload.id).catch(() => undefined)
      const created = await uploadReferenceImages([file])
      setUpload(created[0] || null)
      setPlan(null)
      setFrameTask(null)
      setRender(null)
      setStep('editing_prompt')
      setMessage('参考图已上传，继续填写动作提示词')
    } catch (err) {
      setError(err instanceof Error ? err.message : '上传参考图失败')
    }
  }

  async function removeUpload() {
    if (upload) await deleteReferenceUpload(upload.id).catch(() => undefined)
    setUpload(null)
    setPlan(null)
    setFrameTask(null)
    setRender(null)
    setStep('editing_image')
  }

  async function planFrames() {
    setError('')
    if (!upload) {
      setError('请先上传 1 张参考图')
      return null
    }
    setStep('planning')
    try {
      const data = await createGifPlan({ uploadId: upload.id, motionType, prompt, frameCount, fps, strength })
      setPlan(data.plan)
      setFrameCount(data.plan.frameCount)
      setFPS(data.plan.fps)
      setPlanningFallback(Boolean(data.fallback))
      setMessage(data.fallback ? '已使用本地 fallback 规划帧提示词' : 'GPT-5.5 已完成帧提示词规划')
      setStep('editing_prompt')
      return data.plan
    } catch (err) {
      setStep('failed')
      setError(err instanceof Error ? err.message : '规划动画帧失败')
      return null
    }
  }

  async function generateFrames() {
    setError('')
    if (!upload) {
      setError('请先上传 1 张参考图')
      return
    }
    if (!keyReady) {
      setError('请先在设置页保存 codex-key，或确认已上传到云端')
      return
    }
    const currentPlan = plan || await planFrames()
    if (!currentPlan) return
    setStep('frames_generating')
    setRender(null)
    try {
      const framePrompts = currentPlan.frames.map((frame) => buildFramePrompt(currentPlan, frame.prompt))
      const task = await createTask({
        provider: 'image-2',
        model: DEFAULT_IMAGE2_MODEL,
        mode: 'image-to-image',
        prompt: framePrompts[0] || prompt,
        framePrompts,
        ratio: 'auto',
        resolution: 'standard',
        quality: 'high',
        outputFormat: 'png',
        count: currentPlan.frameCount,
        concurrency: 2,
        uploadIds: [upload.id],
      })
      setFrameTask(task)
      setActiveTaskId(task.id)
      setMessage('动画帧任务已提交，正在生成')
    } catch (err) {
      setStep('failed')
      setError(err instanceof Error ? err.message : '生成动画帧失败')
    }
  }

  async function renderGIF() {
    setError('')
    if (!frameTask) return
    const frameIndexes = successfulFrames.map((item) => item.index)
    if (frameIndexes.length < 2) {
      setError('至少需要 2 张成功帧才能合成 GIF')
      return
    }
    setStep('rendering')
    try {
      const item = await createGifRender({ sourceTaskId: frameTask.id, frameIndexes, fps, loop, width })
      setRender(item)
      setStep('gif_ready')
      setMessage('GIF 合成完成')
    } catch (err) {
      setStep('failed')
      setError(err instanceof Error ? err.message : '合成 GIF 失败')
    }
  }

  if (!session) return <SpaceLogin theme={theme} onToggleTheme={onToggleTheme} onSession={setSession} />

  return (
    <div className="app-shell gallery-shell gif-workbench-page">
      <header className="topbar workbench-topbar">
        <div className="brand">
          <div className="brand-mark">GIF</div>
          <div>
            <h1>GIF 工作流</h1>
            <p>单张图片动起来 · {session.user.displayName}</p>
          </div>
        </div>
        <div className="top-status">
          <span className={keyReady ? 'ready' : 'missing'}>codex-key {keyReady ? '已设置' : '未设置'}</span>
          <span className={status?.ffmpegAvailable ? 'ready' : 'missing'}>FFmpeg {status?.ffmpegAvailable ? '可用' : '不可用'}</span>
        </div>
        <nav className="top-actions">
          <a className="ghost-link" href="/">普通工作台</a>
          <GitHubLink />
          <ThemeToggle theme={theme} onToggle={onToggleTheme} />
          <button onClick={logout}>退出登录</button>
        </nav>
      </header>

      <main className="gif-layout">
        <section className="gif-panel gif-control-panel">
          <div className="page-heading">
            <span className="eyebrow">Image to GIF</span>
            <h2>图片动起来</h2>
            <p>上传 1 张参考图，规划小幅动作帧，用 image2 生成多张帧图，再用 FFmpeg 合成最终 GIF。</p>
          </div>

          {ffmpegWarning ? <div className="key-warning"><strong>FFmpeg 不可用</strong><span>{ffmpegWarning}</span></div> : null}
          {error ? <div className="error-banner">{error}</div> : null}
          {message ? <div className="api-service-banner"><strong>状态</strong><span>{message}</span></div> : null}

          <section className="gif-step-card">
            <div className="section-title"><span>Step 1 上传单张参考图</span><small>PNG / JPG / WEBP</small></div>
            {!upload ? (
              <label className="upload-dropzone gif-single-upload">
                <input type="file" accept="image/png,image/jpeg,image/webp" onChange={handleUpload} />
                <span>上传参考图</span>
                <small>只选择 1 张作为动画源图</small>
              </label>
            ) : (
              <article className="reference-card gif-source-card">
                <div className="reference-thumb">{previewUrl ? <img src={previewUrl} alt={upload.originalName} /> : <span>IMG</span>}</div>
                <div className="reference-info">
                  <strong>{upload.originalName}</strong>
                  <small>{upload.mime} · {formatBytes(upload.size)}</small>
                  <button type="button" className="danger-text" onClick={removeUpload}>更换图片</button>
                </div>
              </article>
            )}
          </section>

          <section className="gif-step-card">
            <div className="section-title"><span>Step 2 动作提示词</span><small>GPT-5.5 可选规划</small></div>
            <div className="gif-motion-grid">
              {MOTIONS.map((item) => (
                <button key={item.id} type="button" className={motionType === item.id ? 'active' : ''} onClick={() => { setMotionType(item.id); setPlan(null) }}>
                  <strong>{item.label}</strong><small>{item.hint}</small>
                </button>
              ))}
            </div>
            <textarea className="gif-prompt-input" value={prompt} onChange={(event) => { setPrompt(event.target.value); setPlan(null) }} rows={4} placeholder="例如：让人物自然眨眼并轻微微笑，保持同一个人、同一背景、同一构图" />
            <button type="button" className="secondary" onClick={planFrames} disabled={!upload || step === 'planning'}>{step === 'planning' ? '规划中...' : '使用 GPT-5.5 规划帧提示词'}</button>
            {planningFallback ? <small className="muted">当前规划来自本地 fallback，可直接用于生成帧。</small> : null}
          </section>

          <section className="gif-step-card">
            <div className="section-title"><span>Step 3 动画设置</span><small>{durationLabel(frameCount, fps)}</small></div>
            <div className="gif-settings-grid">
              <label>帧数<input type="number" min={2} max={status?.limits.maxFrames || 24} value={frameCount} onChange={(event) => { setFrameCount(numeric(event.target.value, 12)); setPlan(null) }} /></label>
              <label>FPS<input type="number" min={1} max={status?.limits.maxFPS || 15} value={fps} onChange={(event) => { setFPS(numeric(event.target.value, 8)); setPlan(null) }} /></label>
              <label>导出宽度<input type="number" min={128} max={status?.limits.maxSize || 1024} value={width} onChange={(event) => setWidth(numeric(event.target.value, 512))} /></label>
              <label>动作强度<select value={strength} onChange={(event) => { setStrength(event.target.value as GifStrength); setPlan(null) }}>{STRENGTHS.map((item) => <option key={item.id} value={item.id}>{item.label}</option>)}</select></label>
            </div>
            <label className="gif-checkbox"><input type="checkbox" checked={loop} onChange={(event) => setLoop(event.target.checked)} /> 循环播放</label>
          </section>

          <div className="gif-action-row">
            <button type="button" className="primary" onClick={generateFrames} disabled={!upload || !keyReady || step === 'frames_generating'}>{step === 'frames_generating' ? '生成中...' : '生成动画帧'}</button>
            <button type="button" className="primary" onClick={renderGIF} disabled={!canRender || step === 'rendering'}>{step === 'rendering' ? '合成中...' : '合并 GIF'}</button>
          </div>
        </section>

        <section className="gif-panel gif-preview-panel">
          <GifFramePreview frames={successfulFrames.map((item) => item.imageUrl!)} fps={fps} />
          <div className="gif-frame-summary">
            <strong>动画帧</strong>
            <span>{successfulFrames.length} / {frameTask?.count || frameCount} 张成功</span>
          </div>
          <div className="gif-frame-grid">
            {frameTask?.results?.map((result) => (
              <article key={result.index} className={result.ok ? 'gif-frame-card ok' : 'gif-frame-card failed'}>
                {result.ok && result.imageUrl ? <img src={result.imageUrl} alt={`frame ${result.index + 1}`} /> : <div className="gif-frame-error">失败</div>}
                <small>#{result.index + 1} {result.statusText}</small>
              </article>
            ))}
            {!frameTask ? <p className="muted">生成动画帧后，这里会显示帧网格和轮播预览。</p> : null}
          </div>

          <div className="gif-render-panel">
            <h3>最终 GIF</h3>
            {render?.gifUrl ? (
              <>
                <img className="gif-result-image" src={render.gifUrl} alt="final gif" />
                <div className="gif-action-row">
                  <a className="primary download-link" href={render.gifUrl} download={`${render.id}.gif`}>下载 GIF</a>
                  <small>{render.bytes ? formatBytes(render.bytes) : ''}</small>
                </div>
              </>
            ) : (
              <p className="muted">至少 2 张成功帧且 FFmpeg 可用时，可以合并 GIF。</p>
            )}
          </div>

          {plan ? <details className="gif-plan-details"><summary>查看帧提示词规划</summary><ol>{plan.frames.map((frame) => <li key={frame.index}><strong>{frame.action}</strong><p>{frame.prompt}</p></li>)}</ol></details> : null}
        </section>
      </main>
    </div>
  )
}

function GifFramePreview({ frames, fps }: { frames: string[]; fps: number }) {
  const [frameIndex, setFrameIndex] = useState(0)
  const [playing, setPlaying] = useState(true)

  useEffect(() => {
    setFrameIndex(0)
  }, [frames.length])

  useEffect(() => {
    if (!playing || frames.length === 0) return
    const interval = window.setInterval(() => {
      setFrameIndex((i) => (i + 1) % frames.length)
    }, Math.max(50, 1000 / Math.max(1, fps)))
    return () => window.clearInterval(interval)
  }, [playing, frames.length, fps])

  useEffect(() => {
    if (!frames.length) return
    const next = frames[(frameIndex + 1) % frames.length]
    if (!next) return
    const image = new Image()
    image.src = next
  }, [frameIndex, frames])

  return (
    <div className="gif-player">
      <div className="gif-player-stage">
        {frames.length ? <img src={frames[frameIndex]} alt={`preview frame ${frameIndex + 1}`} /> : <span>等待动画帧</span>}
      </div>
      <div className="gif-player-controls">
        <button type="button" onClick={() => setPlaying((current) => !current)} disabled={!frames.length}>{playing ? '暂停' : '播放'}</button>
        <span>{frames.length ? `${frameIndex + 1} / ${frames.length}` : '0 / 0'} · {fps} fps</span>
      </div>
    </div>
  )
}

function buildFramePrompt(plan: GifPlan, framePrompt: string) {
  return [plan.basePrompt, plan.styleLock, framePrompt, `负面约束：${plan.negativePrompt}`].filter(Boolean).join('\n')
}

function numeric(value: string, fallback: number) {
  const parsed = Number.parseInt(value, 10)
  return Number.isFinite(parsed) && parsed > 0 ? parsed : fallback
}

function durationLabel(frameCount: number, fps: number) {
  const duration = fps > 0 ? frameCount / fps : 0
  return `约 ${duration.toFixed(1)} 秒`
}

function isFinal(task: Task) {
  return ['succeeded', 'partial_failed', 'failed', 'cancelled', 'interrupted'].includes(task.status)
}
