import { useState } from 'react'
import type { Task, TaskResult } from '../types'
import { formatBytes } from '../lib/format'
import { errorReasonLabel } from '../lib/errorLabels'
import { ImagePreviewModal } from './ImagePreviewModal'
import { BANANA_PROVIDER, getBananaModelOption, providerLabel } from '../lib/models'

type ResultCanvasProps = {
  task?: Task
  onUseAsReference?: (src: string, index: number) => Promise<void>
  onUploadPixhost?: (taskId: string, index: number) => Promise<void>
  onOpenGenerate?: () => void
  onOpenQueue?: () => void
  onReuse?: (task: Task) => void
  onRetry?: (id: string) => void
}

export function ResultCanvas({ task, onUseAsReference, onUploadPixhost, onOpenGenerate, onOpenQueue, onReuse, onRetry }: ResultCanvasProps) {
  const okCount = task?.results.filter((item) => item.ok).length || 0
  const hasFailed = task?.results.some((item) => !item.ok) || ['failed', 'partial_failed', 'cancelled', 'interrupted'].includes(task?.status || '')
  const hasTaskActions = Boolean(onReuse || onOpenQueue || (hasFailed && onRetry))
  return (
    <section className="result-canvas">
      <header className="canvas-header">
        <div>
          <p className="eyebrow">结果区</p>
          <h2>生成结果</h2>
          {task ? <p>{task.stageText} / {task.stage} / {task.stageCode} · {task.progress}% · {okCount}/{task.count}</p> : <p>选择或创建一个任务后查看结果。</p>}
        </div>
        {task ? <span className={`status-pill ${task.status}`}>{task.statusText} / {task.statusCode}</span> : null}
      </header>

      {!task ? (
        <div className="empty-state">
          <strong>先到“生成”标签提交任务</strong>
          <span>提交文生图或图生图后，当前任务的图片、进度和操作会固定显示在这里。刷新页面也能恢复历史结果。</span>
          {onOpenGenerate || onOpenQueue ? (
            <div className="empty-actions">
              {onOpenGenerate ? <button type="button" className="primary" onClick={onOpenGenerate}>去生成</button> : null}
              {onOpenQueue ? <button type="button" onClick={onOpenQueue}>查看队列</button> : null}
            </div>
          ) : null}
        </div>
      ) : (
        <>
          <ResultContext task={task} />
          <DebugLogPanel task={task} />
          {hasTaskActions ? (
            <div className="result-action-row">
              {onReuse ? <button type="button" onClick={() => onReuse(task)}>复用参数</button> : null}
              {onOpenQueue ? <button type="button" onClick={onOpenQueue}>查看队列</button> : null}
              {hasFailed && onRetry ? <button type="button" onClick={() => onRetry(task.id)}>重试失败任务</button> : null}
            </div>
          ) : null}
          <div className="result-grid">
            {Array.from({ length: task.count }, (_, index) => {
              const result = task.results.find((item) => item.index === index)
              return <ResultCard key={index} task={task} index={index} result={result} onUseAsReference={onUseAsReference} onUploadPixhost={onUploadPixhost} />
            })}
          </div>
        </>
      )}
    </section>
  )
}

function ResultContext({ task }: { task: Task }) {
  return (
    <section className="result-context" aria-label="生成请求信息">
      <details className="result-prompt" open={(task.prompt || '').length <= 120}>
        <summary>
          <span>提示词</span>
          <strong>{task.prompt ? compactPrompt(task.prompt) : '（无提示词）'}</strong>
        </summary>
        <p>{task.prompt || '（无提示词）'}</p>
      </details>
      <div className="param-chips" aria-label="生成参数">
        {taskParameters(task).map((item) => <span key={item}>{item}</span>)}
      </div>
    </section>
  )
}

function DebugLogPanel({ task }: { task: Task }) {
  const logs = task.debugLogs || []
  if (!task.debugEnabled && logs.length === 0) return null
  return (
    <section className="debug-log-panel" aria-label="Debug 日志">
      <details open={logs.length > 0 && task.status !== 'succeeded'}>
        <summary>
          <span>Debug 日志</span>
          <strong>{logs.length ? `${logs.length} 条` : '已开启，等待新日志'}</strong>
        </summary>
        {logs.length ? (
          <div className="debug-log-list">
            {logs.map((item, index) => (
              <article key={`${item.time}-${index}`} className={`debug-log-item ${item.level || 'info'}`}>
                <header>
                  <b>{item.level || 'info'}</b>
                  <span>{item.stage || 'debug'}</span>
                  <time>{formatDebugTime(item.time)}</time>
                  {item.imageIndex >= 0 ? <em>#{item.imageIndex + 1}</em> : null}
                </header>
                <p>{item.message}</p>
                {item.fields ? <pre>{renderDebugFields(item.fields)}</pre> : null}
              </article>
            ))}
          </div>
        ) : (
          <p className="muted">Debug 只对开启后创建的新任务记录；不会显示 API Key 明文。</p>
        )}
      </details>
    </section>
  )
}

function ResultCard({ task, index, result, onUseAsReference, onUploadPixhost }: { task: Task; index: number; result?: TaskResult; onUseAsReference?: (src: string, index: number) => Promise<void>; onUploadPixhost?: (taskId: string, index: number) => Promise<void> }) {
  const [previewOpen, setPreviewOpen] = useState(false)
  const [notice, setNotice] = useState('')

  if (!result) {
    return (
      <article className="result-card is-loading">
        <div className="skeleton">
          <div className="spinner" />
          <span>第 {index + 1} 张生成中...</span>
        </div>
        <footer>排队中 / queued / J100</footer>
      </article>
    )
  }

  const imageUrl = result.ok && result.imageUrl ? result.imageUrl : ''
  const copyableURL = result.remoteUrl || imageUrl

  return (
    <>
      <article className={`result-card ${result.ok ? '' : 'is-error'}`}>
        {imageUrl ? (
          <>
            <img src={imageUrl} alt={`生成结果 ${index + 1}`} />
            <div className="floating-actions">
              <button type="button" className="zoom-btn" onClick={() => setPreviewOpen(true)} title="放大预览">⛶</button>
              {result.remoteUrl ? (
                <button type="button" className="url-copy-btn" onClick={() => void copyURL(result.remoteUrl!, setNotice)}>复制URL</button>
              ) : (
                <button type="button" className={`url-copy-btn ${result.uploadError ? 'error' : ''}`} onClick={() => void uploadPixhost(task.id, index, onUploadPixhost, setNotice)}>
                  {result.uploadError ? '重试上传' : '上传图床'}
                </button>
              )}
            </div>
            <div className="card-toolbar">
              <button type="button" onClick={() => void downloadImage(imageUrl, index)}>下载</button>
              <button type="button" onClick={() => void copyImage(imageUrl, setNotice)}>复制图片</button>
              <button type="button" onClick={() => void useAsReference(imageUrl, index, onUseAsReference, setNotice)}>作为参考图</button>
            </div>
            <small className="card-meta">#{index + 1} · {result.elapsedMs ? `${(result.elapsedMs / 1000).toFixed(1)}s` : '完成'} · {formatBytes(result.bytes)}{result.remoteUrl ? ' · 已上传图床' : result.uploadError ? ` · 图床失败：${result.uploadError}` : ''}</small>
            {resultParameters(result).length ? (
              <div className="actual-chips" aria-label="上游实际参数">
                {resultParameters(result).map((item) => <span key={item}>{item}</span>)}
              </div>
            ) : null}
            {result.revisedPrompt ? (
              <details className="revised-prompt">
                <summary>上游改写提示词</summary>
                <p>{result.revisedPrompt}</p>
              </details>
            ) : null}
          </>
        ) : (
          <div className="error-card">
            <strong>第 {index + 1} 张失败</strong>
            <p>{errorReasonLabel(result)}</p>
            {result.error ? <small>原始错误：{result.error}</small> : null}
          </div>
        )}
        <footer>{result.statusText} / {result.status} / {result.statusCode}</footer>
        {notice ? <div className="card-notice">{notice}</div> : null}
      </article>
      {previewOpen && imageUrl ? (
        <ImagePreviewModal
          src={imageUrl}
          title={`生成结果 ${index + 1}`}
          requestedSize={task.size}
          ratio={task.ratio}
          bytes={result.bytes}
          parameters={[...taskParameters(task), ...resultParameters(result)]}
          onCopyImage={() => copyImage(imageUrl, setNotice)}
          onCopyUrl={() => copyURL(copyableURL, setNotice)}
          onDownload={() => downloadImage(imageUrl, index)}
          onUseAsReference={() => useAsReference(imageUrl, index, onUseAsReference, setNotice)}
          onClose={() => setPreviewOpen(false)}
        />
      ) : null}
    </>
  )
}

async function uploadPixhost(taskId: string, index: number, onUploadPixhost: ((taskId: string, index: number) => Promise<void>) | undefined, setNotice: (value: string) => void) {
  if (!onUploadPixhost) return
  try {
    flash(setNotice, '正在上传图床...')
    await onUploadPixhost(taskId, index)
    flash(setNotice, '图床上传成功')
  } catch (err) {
    flash(setNotice, err instanceof Error ? err.message : '图床上传失败')
  }
}

async function useAsReference(src: string, index: number, onUseAsReference: ((src: string, index: number) => Promise<void>) | undefined, setNotice: (value: string) => void) {
  if (!onUseAsReference) return
  try {
    await onUseAsReference(src, index)
    flash(setNotice, '已加入参考图')
  } catch (err) {
    flash(setNotice, err instanceof Error ? err.message : '加入参考图失败')
  }
}

async function copyURL(src: string, setNotice: (value: string) => void) {
  const url = new URL(src, window.location.origin).href
  try {
    await navigator.clipboard.writeText(url)
    flash(setNotice, '链接已复制')
  } catch {
    flash(setNotice, '复制失败')
  }
}

async function copyImage(src: string, setNotice: (value: string) => void) {
  try {
    const response = await fetch(src)
    const blob = await response.blob()
    await navigator.clipboard.write([
      new ClipboardItem({ [blob.type || 'image/png']: blob }),
    ])
    flash(setNotice, '图片已复制')
  } catch {
    await copyURL(src, setNotice)
  }
}

async function downloadImage(src: string, index: number) {
  const response = await fetch(src)
  const blob = await response.blob()
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = `image-2-${Date.now()}-${index + 1}.${extensionFromMime(blob.type)}`
  document.body.appendChild(link)
  link.click()
  link.remove()
  URL.revokeObjectURL(url)
}

function extensionFromMime(mime: string) {
  if (mime.includes('jpeg')) return 'jpg'
  if (mime.includes('webp')) return 'webp'
  if (mime.includes('gif')) return 'gif'
  return 'png'
}

function compactPrompt(prompt: string) {
  const text = prompt.trim().replace(/\s+/g, ' ')
  return text.length > 96 ? `${text.slice(0, 96)}...` : text
}

function taskParameters(task: Task) {
  const provider = task.provider || 'image-2'
  if (provider === BANANA_PROVIDER) {
    const option = getBananaModelOption(task.model || '')
    return [
      task.mode === 'image-to-image' ? '图生图' : '文生图',
      `模型分组 ${providerLabel(provider)}`,
      `规格 ${option.label}`,
      `模型 ID ${option.id}`,
      option.size !== '自动' ? `尺寸 ${option.size}` : '自动尺寸',
      `数量 ${task.count || 1}`,
      `并发 ${task.concurrency || 1}`,
    ]
  }
  return [
    task.mode === 'image-to-image' ? '图生图' : '文生图',
    `模型分组 ${providerLabel(provider)}`,
    `模型 ${task.model || 'gpt-image-2'}`,
    !task.ratio || task.ratio === 'auto' ? '自动比例' : `比例 ${task.ratio}`,
    `清晰度 ${resolutionLabel(task.resolution)}`,
    `质量 ${qualityLabel(task.quality)}`,
    `格式 ${outputFormatLabel(task.outputFormat)}`,
    task.size && task.size !== '自动' ? `尺寸 ${task.size}` : '自动尺寸',
    `数量 ${task.count || 1}`,
    `并发 ${task.concurrency || 1}`,
  ]
}

function resultParameters(result?: TaskResult) {
  if (!result) return []
  return [
    result.actualSize ? `实际尺寸 ${result.actualSize}` : '',
    result.actualQuality ? `实际质量 ${qualityLabel(result.actualQuality)}` : '',
    result.outputFormat ? `输出格式 ${result.outputFormat}` : '',
    result.revisedPrompt ? '有上游改写提示词' : '',
  ].filter(Boolean)
}

function resolutionLabel(value?: string) {
  const labels: Record<string, string> = { auto: '自动', standard: '标准', '2k': '2K', '4k': '4K' }
  if (!value) return '自动'
  return labels[value] || value
}

function qualityLabel(value?: string) {
  const labels: Record<string, string> = { auto: '自动', low: '低', medium: '中', high: '高' }
  if (!value) return '自动'
  return labels[value] || value
}

function outputFormatLabel(value?: string) {
  const labels: Record<string, string> = { auto: '自动', png: 'PNG', jpeg: 'JPG', jpg: 'JPG', webp: 'WEBP' }
  if (!value) return 'PNG'
  return labels[value] || value.toUpperCase()
}

function formatDebugTime(value?: string) {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleTimeString()
}

function renderDebugFields(fields: Record<string, unknown>) {
  return JSON.stringify(fields, null, 2)
}

function flash(setNotice: (value: string) => void, value: string) {
  setNotice(value)
  window.setTimeout(() => setNotice(''), 1600)
}
