import { useState } from 'react'
import type { Task, TaskResult } from '../types'
import { formatBytes } from '../lib/format'
import { ImagePreviewModal } from './ImagePreviewModal'

export function ResultCanvas({ task, onUseAsReference, onUploadPixhost }: { task?: Task; onUseAsReference?: (src: string, index: number) => Promise<void>; onUploadPixhost?: (taskId: string, index: number) => Promise<void> }) {
  const okCount = task?.results.filter((item) => item.ok).length || 0
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
          <strong>还没有选择任务</strong>
          <span>提交任务后，生成结果会出现在这里。后端继续执行，刷新页面也可以恢复。</span>
        </div>
      ) : (
        <>
          <ResultContext task={task} />
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
      <div className="result-prompt">
        <span>提示词</span>
        <p>{task.prompt || '（无提示词）'}</p>
      </div>
      <div className="param-chips" aria-label="生成参数">
        {taskParameters(task).map((item) => <span key={item}>{item}</span>)}
      </div>
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
            <p>{result.error || result.statusText}</p>
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
          prompt={task.prompt}
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

function taskParameters(task: Task) {
  return [
    task.mode === 'image-to-image' ? '图生图' : '文生图',
    !task.ratio || task.ratio === 'auto' ? '自动比例' : `比例 ${task.ratio}`,
    `清晰度 ${resolutionLabel(task.resolution)}`,
    `质量 ${qualityLabel(task.quality)}`,
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

function flash(setNotice: (value: string) => void, value: string) {
  setNotice(value)
  window.setTimeout(() => setNotice(''), 1600)
}
