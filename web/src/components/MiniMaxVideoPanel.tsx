import { type FormEvent, useEffect, useMemo, useState } from 'react'
import { createMiniMaxVideo, getMiniMaxApiKey, maskMiniMaxKey, queryMiniMaxVideo, retrieveMiniMaxFile, saveMiniMaxApiKey } from '../api/minimaxVideos'
import type { MiniMaxFileResult, MiniMaxVideoStatus } from '../types'

const statusLabels: Record<string, string> = {
  Preparing: '排队/准备中',
  Queueing: '排队中',
  Processing: '生成中',
  Success: '成功',
  Fail: '失败',
}

export function MiniMaxVideoPanel({ seedPrompt }: { seedPrompt: string }) {
  const [apiKeyInput, setApiKeyInput] = useState('')
  const [apiKeyReady, setApiKeyReady] = useState(false)
  const [model, setModel] = useState('MiniMax-Hailuo-02')
  const [prompt, setPrompt] = useState(seedPrompt)
  const [duration, setDuration] = useState(6)
  const [resolution, setResolution] = useState('1080P')
  const [promptOptimizer, setPromptOptimizer] = useState(true)
  const [fastPretreatment, setFastPretreatment] = useState(false)
  const [aigcWatermark, setAigcWatermark] = useState(false)
  const [taskID, setTaskID] = useState('')
  const [status, setStatus] = useState<MiniMaxVideoStatus | null>(null)
  const [file, setFile] = useState<MiniMaxFileResult | null>(null)
  const [message, setMessage] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const downloadURL = file?.file?.download_url || ''

  useEffect(() => {
    const key = getMiniMaxApiKey()
    setApiKeyReady(Boolean(key))
    setApiKeyInput('')
  }, [])

  useEffect(() => {
    if (!seedPrompt || prompt.trim()) return
    setPrompt(seedPrompt)
  }, [seedPrompt, prompt])

  const statusText = useMemo(() => {
    if (!status) return '还没有视频任务'
    return `${statusLabels[status.status] || status.status || '未知状态'}${status.file_id ? ` · file_id=${status.file_id}` : ''}`
  }, [status])

  function saveKey() {
    saveMiniMaxApiKey(apiKeyInput)
    setApiKeyReady(Boolean(apiKeyInput.trim()))
    setApiKeyInput('')
    setMessage(apiKeyInput.trim() ? `MiniMax Key 已保存到当前浏览器：${maskMiniMaxKey(apiKeyInput)}` : 'MiniMax Key 已清除')
  }

  async function submit(event: FormEvent) {
    event.preventDefault()
    setError('')
    setMessage('')
    setFile(null)
    setStatus(null)
    if (!getMiniMaxApiKey()) {
      setError('请先保存 MiniMax API Key')
      return
    }
    if (!prompt.trim()) {
      setError('请先输入视频提示词')
      return
    }
    setBusy(true)
    try {
      const task = await createMiniMaxVideo({
        model,
        prompt,
        duration,
        resolution,
        prompt_optimizer: promptOptimizer,
        fast_pretreatment: fastPretreatment,
        aigc_watermark: aigcWatermark,
      })
      setTaskID(task.task_id)
      setMessage('视频任务已提交，稍后可点击查询状态')
    } catch (err) {
      setError(err instanceof Error ? err.message : '创建视频任务失败')
    } finally {
      setBusy(false)
    }
  }

  async function poll() {
    const id = taskID.trim()
    if (!id) {
      setError('没有可查询的 task_id')
      return
    }
    setBusy(true)
    setError('')
    setMessage('')
    try {
      const next = await queryMiniMaxVideo(id)
      setStatus(next)
      if (next.file_id) {
        const fileResult = await retrieveMiniMaxFile(next.file_id)
        setFile(fileResult)
        setMessage('视频已生成，可以预览或下载')
      } else {
        setMessage(`当前状态：${statusLabels[next.status] || next.status || '未知'}`)
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : '查询视频任务失败')
    } finally {
      setBusy(false)
    }
  }

  function copyTaskID() {
    if (!taskID) return
    void navigator.clipboard.writeText(taskID)
    setMessage('已复制 task_id')
  }

  return (
    <section className="video-panel">
      <header className="workflow-page-header">
        <p className="eyebrow">MiniMax Video / Dev Preview</p>
        <h2>MiniMax 文生视频</h2>
        <p>按 MiniMax T2V 接口创建视频任务，再通过 task_id 查询状态，成功后读取 file_id 的下载地址。</p>
      </header>

      <div className="video-layout">
        <form className="video-card video-form" onSubmit={submit}>
          <div className="panel-title">
            <strong>提交视频任务</strong>
            <span>{apiKeyReady ? 'MiniMax Key 已保存在当前浏览器' : 'MiniMax Key 未设置'}</span>
          </div>

          <label>
            MiniMax API Key
            <div className="video-key-row">
              <input type="password" value={apiKeyInput} onChange={(event) => setApiKeyInput(event.target.value)} placeholder={apiKeyReady ? '已保存，输入新 Key 可覆盖；留空保存会清除' : '填入 MiniMax API Key'} />
              <button type="button" onClick={saveKey}>保存</button>
            </div>
          </label>

          <label>
            视频提示词
            <textarea value={prompt} onChange={(event) => setPrompt(event.target.value)} rows={8} placeholder="描述你想生成的视频画面、镜头运动、风格、主体动作..." />
          </label>

          <div className="video-grid">
            <label>
              模型
              <input value={model} onChange={(event) => setModel(event.target.value)} />
            </label>
            <label>
              时长
              <select value={duration} onChange={(event) => setDuration(Number(event.target.value))}>
                <option value={6}>6 秒</option>
                <option value={10}>10 秒</option>
              </select>
            </label>
            <label>
              分辨率
              <select value={resolution} onChange={(event) => setResolution(event.target.value)}>
                <option value="1080P">1080P</option>
                <option value="768P">768P</option>
                <option value="">默认</option>
              </select>
            </label>
          </div>

          <div className="video-checks">
            <label><input type="checkbox" checked={promptOptimizer} onChange={(event) => setPromptOptimizer(event.target.checked)} /> 启用提示词优化</label>
            <label><input type="checkbox" checked={fastPretreatment} onChange={(event) => setFastPretreatment(event.target.checked)} /> 快速预处理</label>
            <label><input type="checkbox" checked={aigcWatermark} onChange={(event) => setAigcWatermark(event.target.checked)} /> 添加 AIGC 水印</label>
          </div>

          <button type="submit" disabled={busy || !apiKeyReady}>{busy ? '处理中...' : '生成视频'}</button>
          {message ? <p className="video-message">{message}</p> : null}
          {error ? <p className="video-error">{error}</p> : null}
        </form>

        <aside className="video-card video-result">
          <div className="panel-title">
            <strong>任务状态</strong>
            <span>{statusText}</span>
          </div>
          <label>
            task_id
            <div className="video-key-row">
              <input value={taskID} onChange={(event) => setTaskID(event.target.value)} placeholder="提交后自动填入，也可以粘贴历史 task_id" />
              <button type="button" onClick={copyTaskID}>复制</button>
            </div>
          </label>
          <button type="button" onClick={poll} disabled={busy || !taskID.trim()}>{busy ? '查询中...' : '查询状态 / 获取视频'}</button>

          {status ? (
            <div className="video-meta">
              <span>状态：{statusLabels[status.status] || status.status}</span>
              {status.video_width && status.video_height ? <span>尺寸：{status.video_width}×{status.video_height}</span> : null}
              {status.file_id ? <span>file_id：{status.file_id}</span> : null}
            </div>
          ) : null}

          {downloadURL ? (
            <div className="video-preview-box">
              <video src={downloadURL} controls playsInline />
              <div className="video-actions">
                <a href={downloadURL} target="_blank" rel="noreferrer">打开视频</a>
                <a href={downloadURL} download>下载视频</a>
              </div>
            </div>
          ) : (
            <div className="video-empty">
              <strong>还没有生成结果</strong>
              <span>提交任务后等待几分钟，再点击查询状态。</span>
            </div>
          )}
        </aside>
      </div>
    </section>
  )
}
