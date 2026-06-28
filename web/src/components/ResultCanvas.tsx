import { type FormEvent, useState } from 'react'
import type { Task, TaskResult } from '../types'
import { formatBytes } from '../lib/format'
import { errorReasonLabel } from '../lib/errorLabels'
import { ImagePreviewModal } from './ImagePreviewModal'
import { BANANA_PROVIDER, DEFAULT_IMAGE2_MODEL, getBananaModelOption, providerLabel } from '../lib/models'
import { nativeCopyImage, nativeCopyText, nativeSaveImage } from '../lib/nativeBridge'

type SquareSubmitOptions = {
  title?: string
  tags?: string[]
  referenceUsageNote?: string
}

type SquareSubmitReference = {
  uploadId?: string
  originalName?: string
  fileName?: string
  mime?: string
  size?: number
}

type ResultCanvasProps = {
  task?: Task
  onUseAsReference?: (src: string, index: number) => Promise<unknown>
  onUploadPixhost?: (taskId: string, index: number) => Promise<unknown>
  onOpenGenerate?: () => void
  onReuse?: (task: Task) => void
  onRetry?: (id: string) => void
  submittedSquareKeys?: Set<string>
  onSubmitToSquare?: (task: Task, result: TaskResult, index: number, options?: SquareSubmitOptions) => Promise<boolean>
}

export function ResultCanvas({ task, onUseAsReference, onUploadPixhost, onOpenGenerate, onReuse, onRetry, submittedSquareKeys, onSubmitToSquare }: ResultCanvasProps) {
  const okCount = task?.results.filter((item) => item.ok).length || 0
  const hasFailed = task?.results.some((item) => !item.ok) || ['failed', 'partial_failed', 'cancelled', 'interrupted'].includes(task?.status || '')
  const hasTaskActions = Boolean(onReuse || (hasFailed && onRetry))
  const layoutClass = task ? resultLayoutClass(task.count) : ''
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

      <p className="result-retention-note">普通结果按当前保留策略处理；提交到广场后会生成用于广场展示的副本。</p>

      {!task ? (
        <div className="empty-state">
          <strong>先到“创作画布”提交任务</strong>
          <span>提交文字生成或参考图生成后，当前任务的图片、进度和操作会固定显示在这里。刷新页面也能恢复历史结果。</span>
          {onOpenGenerate ? (
            <div className="empty-actions">
              <button type="button" className="primary" onClick={onOpenGenerate}>去创作画布</button>
            </div>
          ) : null}
        </div>
      ) : (
        <>
          <ResultContext task={task} />
          {hasTaskActions ? (
            <div className="result-action-row">
              {onReuse ? <button type="button" onClick={() => onReuse(task)}>复用参数</button> : null}
              {hasFailed && onRetry ? <button type="button" onClick={() => onRetry(task.id)}>重试失败任务</button> : null}
            </div>
          ) : null}
          <div className={`result-grid ${layoutClass}`}>
            {Array.from({ length: task.count }, (_, index) => {
              const result = task.results.find((item) => item.index === index)
              return (
                <ResultCard
                  key={index}
                  task={task}
                  index={index}
                  result={result}
                  submittedToSquare={submittedSquareKeys?.has(`${task.id}:${index}`) || false}
                  onUseAsReference={onUseAsReference}
                  onUploadPixhost={onUploadPixhost}
                  onSubmitToSquare={onSubmitToSquare}
                />
              )
            })}
          </div>
          <DiagnosticLogPanel task={task} />
        </>
      )}
    </section>
  )
}

function ResultContext({ task }: { task: Task }) {
  const revisedPrompts = uniqueRevisedPrompts(task)
  return (
    <section className="result-context" aria-label="生成请求信息">
      <div className="result-task-summary" aria-label="任务摘要">
        <div><span>任务 ID</span><strong>{compactTaskId(task.id)}</strong></div>
        <div><span>来源</span><strong>{sourceLabel(task.source)}</strong></div>
        <div><span>模型</span><strong>{taskModelLabel(task)}</strong></div>
        <div><span>消耗</span><strong>{task.count || 1} 次</strong></div>
        <div><span>创建时间</span><strong>{formatTaskTime(task.createdAt)}</strong></div>
      </div>
      <details className="result-prompt" open={(task.prompt || '').length <= 120}>
        <summary>
          <span>提示词</span>
          <strong>{task.prompt ? compactPrompt(task.prompt) : '（无提示词）'}</strong>
        </summary>
        <p>{task.prompt || '（无提示词）'}</p>
      </details>
      {revisedPrompts.length ? (
        <div className="result-revised-prompts" aria-label="模型返回提示词">
          {revisedPrompts.map((nextPrompt, index) => (
            <details key={`${index}-${nextPrompt}`} className="result-prompt result-revised-prompt" open={revisedPrompts.length === 1}>
              <summary>
                <span>{revisedPrompts.length > 1 ? `模型返回提示词 ${index + 1}` : '模型返回提示词'}</span>
                <strong>{compactPrompt(nextPrompt)}</strong>
              </summary>
              <p>{nextPrompt}</p>
            </details>
          ))}
        </div>
      ) : null}
      <div className="param-chips" aria-label="生成参数">
        {taskParameters(task).map((item) => <span key={item}>{item}</span>)}
      </div>
    </section>
  )
}

function compactTaskId(id: string) {
  if (id.length <= 18) return id
  return `${id.slice(0, 10)}...${id.slice(-5)}`
}

function sourceLabel(source?: string) {
  return source === 'api' ? 'API' : 'Web'
}

function formatTaskTime(value?: string) {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return new Intl.DateTimeFormat('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  }).format(date)
}

function squareDefaultTitle(task: Task, result: TaskResult, index: number) {
  return compactSquareText(result.revisedPrompt || task.prompt || `生成结果 ${index + 1}`, 80)
}

function squareDefaultTags(task: Task) {
  return [
    task.mode === 'gif' ? 'GIF 动图' : task.mode === 'image-to-image' ? '参考图生成' : '文字生成',
    taskModelTag(task),
    !task.ratio || task.ratio === 'auto' ? '' : task.ratio,
  ].filter(Boolean).slice(0, 6)
}

function splitSquareTags(value: string) {
  return value.split(/[,，\s]+/).map((item) => item.trim()).filter(Boolean).slice(0, 12)
}

function compactSquareText(value: string, max = 96) {
  const text = value.trim().replace(/\s+/g, ' ')
  if (text.length <= max) return text
  return `${text.slice(0, max)}...`
}

function imageAspectStyle(value: string) {
  const aspectRatio = ratioToCssAspect(value)
  return aspectRatio ? { aspectRatio } : undefined
}

function ratioToCssAspect(value: string) {
  const match = value.trim().match(/^(\d+(?:\.\d+)?)\s*[:/x×]\s*(\d+(?:\.\d+)?)$/i)
  if (!match) return ''
  const width = Number(match[1])
  const height = Number(match[2])
  if (!Number.isFinite(width) || !Number.isFinite(height) || width <= 0 || height <= 0) return ''
  return `${width} / ${height}`
}

function DiagnosticLogPanel({ task }: { task: Task }) {
  const logs = task.debugLogs || []
  if (!task.debugEnabled && logs.length === 0) return null
  return (
    <section className="debug-log-panel" aria-label="诊断日志">
      <details open={logs.length > 0 && task.status !== 'succeeded'}>
        <summary>
          <span>诊断日志</span>
          <strong>{logs.length ? `${logs.length} 条` : '已开启，等待新日志'}</strong>
        </summary>
        {logs.length ? (
          <div className="debug-log-list">
            {logs.map((item, index) => (
              <article key={`${item.time}-${index}`} className={`debug-log-item ${item.level || 'info'}`}>
                <header>
                  <b>{item.level || 'info'}</b>
                  <span>{item.stage || '诊断'}</span>
                  <time>{formatDiagnosticTime(item.time)}</time>
                  {item.imageIndex >= 0 ? <em>#{item.imageIndex + 1}</em> : null}
                </header>
                <p>{item.message}</p>
                {item.fields ? <pre>{renderDiagnosticFields(item.fields)}</pre> : null}
              </article>
            ))}
          </div>
        ) : (
          <p className="muted">开启后创建的新任务会记录诊断信息；不会显示 API Key 明文。</p>
        )}
      </details>
    </section>
  )
}

function ResultCard({ task, index, result, submittedToSquare, onUseAsReference, onUploadPixhost, onSubmitToSquare }: {
  task: Task
  index: number
  result?: TaskResult
  submittedToSquare: boolean
  onUseAsReference?: (src: string, index: number) => Promise<unknown>
  onUploadPixhost?: (taskId: string, index: number) => Promise<unknown>
  onSubmitToSquare?: (task: Task, result: TaskResult, index: number, options?: SquareSubmitOptions) => Promise<boolean>
}) {
  const [previewOpen, setPreviewOpen] = useState(false)
  const [squareDialogOpen, setSquareDialogOpen] = useState(false)
  const [notice, setNotice] = useState('')
  const [submittingToSquare, setSubmittingToSquare] = useState(false)

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
  const imageFrameStyle = task.count > 1 ? imageAspectStyle(result.actualSize || task.ratio) : undefined

  return (
    <>
      <article className={`result-card ${result.ok ? '' : 'is-error'}`}>
        {imageUrl ? (
          <>
            <div className="result-image-frame" style={imageFrameStyle}>
              <img
                src={imageUrl}
                alt={`生成结果 ${index + 1}`}
                draggable
                onDragStart={(event) => {
                  event.dataTransfer.effectAllowed = 'copy'
                  event.dataTransfer.setData('application/x-lyra-history-result', JSON.stringify({
                    id: `${task.id}:${index}`,
                    src: imageUrl,
                    title: `生成结果 ${index + 1}`,
                    subtitle: `${task.statusText} · #${index + 1}`,
                    taskId: task.id,
                    index,
                    prompt: result.revisedPrompt || task.prompt,
                  }))
                  event.dataTransfer.setData('text/plain', imageUrl)
                }}
              />
            </div>
            <div className="card-toolbar">
              <button type="button" onClick={() => setPreviewOpen(true)}>预览</button>
              <button type="button" onClick={() => void downloadImage(imageUrl, index, setNotice)}>下载</button>
              <button type="button" onClick={() => void copyImage(imageUrl, setNotice)}>复制图片</button>
              <button type="button" onClick={() => void copyURL(copyableURL, setNotice)}>复制链接</button>
              {onSubmitToSquare ? (
                <button type="button" className="primary square-submit-button" disabled={submittingToSquare || submittedToSquare} onClick={() => setSquareDialogOpen(true)}>
                  {submittedToSquare ? '已提交广场' : submittingToSquare ? '提交中...' : '提交到广场'}
                </button>
              ) : null}
              {!result.remoteUrl ? (
                <button type="button" className={result.uploadError ? 'danger-text' : ''} onClick={() => void uploadPixhost(task.id, index, onUploadPixhost, setNotice)}>
                  {result.uploadError ? '重试图床' : '上传图床'}
                </button>
              ) : null}
              <button type="button" onClick={() => void useAsReference(imageUrl, index, onUseAsReference, setNotice)}>@ 作为参考图</button>
            </div>
            <small className="card-meta">#{index + 1} · {result.elapsedMs ? `${(result.elapsedMs / 1000).toFixed(1)}s` : '完成'} · {formatBytes(result.bytes)}{result.remoteUrl ? ' · 已上传图床' : result.uploadError ? ` · 图床失败：${result.uploadError}` : ''}</small>
            <small className={`square-retention-state ${submittedToSquare ? 'is-square-copy' : ''}`}>{submittedToSquare ? '已提交到广场：广场展示副本' : '普通结果：按当前保留策略处理'}</small>
            {resultParameters(result).length ? (
              <div className="actual-chips" aria-label="上游实际参数">
                {resultParameters(result).map((item) => <span key={item}>{item}</span>)}
              </div>
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
          onCopyImage={() => copyImage(imageUrl, setNotice)}
          onCopyUrl={() => copyURL(copyableURL, setNotice)}
          onDownload={() => downloadImage(imageUrl, index, setNotice)}
          onUseAsReference={() => useAsReference(imageUrl, index, onUseAsReference, setNotice)}
          onClose={() => setPreviewOpen(false)}
        />
      ) : null}
      {squareDialogOpen && imageUrl && onSubmitToSquare ? (
        <SquareSubmitDialog
          task={task}
          result={result}
          index={index}
          imageUrl={imageUrl}
          submitting={submittingToSquare}
          onClose={() => setSquareDialogOpen(false)}
          onConfirm={(options) => submitResultToSquare(task, result, index, onSubmitToSquare, setNotice, setSubmittingToSquare, options)}
        />
      ) : null}
    </>
  )
}

function SquareSubmitDialog({ task, result, index, imageUrl, submitting, onClose, onConfirm }: {
  task: Task
  result: TaskResult
  index: number
  imageUrl: string
  submitting: boolean
  onClose: () => void
  onConfirm: (options: SquareSubmitOptions) => Promise<boolean>
}) {
  const promptText = result.revisedPrompt || task.prompt || '（无提示词）'
  const references = squareSubmitReferences(task)
  const defaultReferenceUsageNote = squareReferenceUsageNote(task, references.length)
  const [title, setTitle] = useState(squareDefaultTitle(task, result, index))
  const [tagInput, setTagInput] = useState(squareDefaultTags(task).join(', '))
  const [referenceUsageNote, setReferenceUsageNote] = useState(defaultReferenceUsageNote)

  async function submit(event: FormEvent) {
    event.preventDefault()
    const submitted = await onConfirm({
      title,
      tags: splitSquareTags(tagInput),
      referenceUsageNote: references.length ? referenceUsageNote : undefined,
    })
    if (submitted) onClose()
  }

  return (
    <div className="square-submit-backdrop" role="presentation" onMouseDown={(event) => { if (event.target === event.currentTarget && !submitting) onClose() }}>
      <form className="square-submit-dialog" role="dialog" aria-modal="true" aria-labelledby="square-submit-title" onSubmit={submit}>
        <header>
          <div>
            <span>提交到广场</span>
            <h3 id="square-submit-title">确认提交这张结果图</h3>
          </div>
          <button type="button" aria-label="关闭" onClick={onClose} disabled={submitting}>×</button>
        </header>
        <div className="square-submit-body">
          <div className="square-submit-preview" style={imageAspectStyle(result.actualSize || task.ratio)}>
            <img src={imageUrl} alt={`提交预览 ${index + 1}`} />
          </div>
          <div className="square-submit-form">
            <p className="square-submit-note square-submit-policy">
              <strong>公开作品包：</strong>提交后广场会公开保存结果图、提示词、模型/比例/质量等参数和原始参考图，供别人查看和复用。不想公开参考图就不要提交。
            </p>
            <label>
              <span>标题</span>
              <input value={title} onChange={(event) => setTitle(event.target.value)} placeholder="给这张作品起个短标题" autoFocus />
            </label>
            <label>
              <span>标签</span>
              <input value={tagInput} onChange={(event) => setTagInput(event.target.value)} placeholder={`参考图生成, ${taskModelTag(task)}, 1:1`} />
            </label>
            <dl className="square-submit-meta">
              <div><dt>模型</dt><dd>{taskModelLabel(task)}</dd></div>
              <div><dt>比例</dt><dd>{task.ratio || 'auto'}</dd></div>
              <div><dt>质量</dt><dd>{result.actualQuality || task.quality || 'auto'}</dd></div>
              <div><dt>图片</dt><dd>第 {index + 1} 张 · {result.actualSize || task.size || '自动尺寸'} · {formatBytes(result.bytes)}</dd></div>
              <div><dt>参考图</dt><dd>{references.length ? `${references.length} 张原始参考图` : '无参考图'}</dd></div>
            </dl>
            <div className="square-submit-prompt">
              <span>提示词</span>
              <p>{promptText}</p>
            </div>
            {references.length ? (
              <div className="square-submit-references">
                <span>原始参考图</span>
                <div className="square-submit-reference-grid">
                  {references.map((reference, referenceIndex) => (
                    <div className="square-submit-reference-item" key={`${reference.uploadId || reference.fileName || referenceIndex}`}>
                      <div className="square-submit-reference-thumb">
                        {reference.uploadId ? (
                          <img
                            src={`/api/uploads/reference/${encodeURIComponent(reference.uploadId)}/image`}
                            alt={squareReferenceLabel(reference, referenceIndex)}
                            onError={(event) => { event.currentTarget.hidden = true }}
                          />
                        ) : null}
                      </div>
                      <div>
                        <strong>{squareReferenceLabel(reference, referenceIndex)}</strong>
                        <small>{reference.uploadId ? `upload id ${compactUploadId(reference.uploadId)}` : '任务参考图快照'}{reference.size ? ` · ${formatBytes(reference.size)}` : ''}</small>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            ) : null}
            {references.length ? (
              <label className="square-submit-reference-note-field">
                <span>参考图用途备注</span>
                <textarea value={referenceUsageNote} onChange={(event) => setReferenceUsageNote(event.target.value)} rows={3} placeholder="说明参考图用于主体、风格、构图或整体画面" />
              </label>
            ) : null}
            <p className="square-submit-note">不提交广场时，这条结果仍是私有临时记录，会按当前保留策略清理。</p>
          </div>
        </div>
        <footer>
          <button type="button" onClick={onClose} disabled={submitting}>取消</button>
          <button type="submit" className="primary" disabled={submitting}>{submitting ? '提交中...' : '确认提交'}</button>
        </footer>
      </form>
    </div>
  )
}

async function submitResultToSquare(task: Task, result: TaskResult, index: number, onSubmitToSquare: (task: Task, result: TaskResult, index: number, options?: SquareSubmitOptions) => Promise<boolean>, setNotice: (value: string) => void, setSubmittingToSquare: (value: boolean) => void, options?: SquareSubmitOptions) {
  try {
    setSubmittingToSquare(true)
    flash(setNotice, '正在提交到广场...')
    const submitted = await onSubmitToSquare(task, result, index, options)
    flash(setNotice, submitted ? '已提交到广场，已生成广场展示副本' : '已取消提交')
    return submitted
  } catch (err) {
    flash(setNotice, err instanceof Error ? err.message : '提交到广场失败')
    return false
  } finally {
    setSubmittingToSquare(false)
  }
}

async function uploadPixhost(taskId: string, index: number, onUploadPixhost: ((taskId: string, index: number) => Promise<unknown>) | undefined, setNotice: (value: string) => void) {
  if (!onUploadPixhost) return
  try {
    flash(setNotice, '正在上传图床...')
    await onUploadPixhost(taskId, index)
    flash(setNotice, '图床上传成功')
  } catch (err) {
    flash(setNotice, err instanceof Error ? err.message : '图床上传失败')
  }
}

async function useAsReference(src: string, index: number, onUseAsReference: ((src: string, index: number) => Promise<unknown>) | undefined, setNotice: (value: string) => void) {
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
    const nativeResult = await nativeCopyText(url)
    if (nativeResult.handled) {
      if (!nativeResult.ok) throw new Error(nativeResult.message || '复制失败')
      flash(setNotice, '链接已复制')
      return '链接已复制'
    }
    await navigator.clipboard.writeText(url)
    flash(setNotice, '链接已复制')
    return '链接已复制'
  } catch {
    flash(setNotice, '复制失败')
    return '复制失败'
  }
}

async function copyImage(src: string, setNotice: (value: string) => void) {
  const absoluteURL = new URL(src, window.location.origin).href
  try {
    const nativeResult = await nativeCopyImage(absoluteURL, `lyai-image-${Date.now()}.png`)
    if (nativeResult.handled) {
      if (!nativeResult.ok) throw new Error(nativeResult.message || '复制图片失败')
      flash(setNotice, '图片已复制')
      return '图片已复制'
    }
    if (!window.isSecureContext) {
      if (copyImageAsHTML(absoluteURL)) {
        flash(setNotice, '图片已复制（备用方式）')
        return '图片已复制（备用方式）'
      }
      throw new Error('复制图片需要 HTTPS 或 localhost 环境')
    }
    if (!navigator.clipboard?.write || typeof ClipboardItem === 'undefined') {
      if (copyImageAsHTML(absoluteURL)) {
        flash(setNotice, '图片已复制（备用方式）')
        return '图片已复制（备用方式）'
      }
      throw new Error('当前浏览器不支持直接复制图片')
    }
    await navigator.clipboard.write([
      new ClipboardItem({ 'image/png': fetchClipboardPng(absoluteURL) }),
    ])
    flash(setNotice, '图片已复制')
    return '图片已复制'
  } catch (err) {
    if (copyImageAsHTML(absoluteURL)) {
      flash(setNotice, '图片已复制（备用方式）')
      return '图片已复制（备用方式）'
    }
    const message = err instanceof Error ? err.message : '复制图片失败'
    flash(setNotice, message)
    return message
  }
}

async function fetchClipboardPng(src: string) {
  const response = await fetch(src, { cache: 'no-store' })
  if (!response.ok) throw new Error(`读取图片失败：HTTP ${response.status}`)
  const blob = await response.blob()
  return ensureClipboardImageBlob(blob)
}

async function ensureClipboardImageBlob(blob: Blob) {
  const mime = (blob.type || '').toLowerCase()
  if (mime === 'image/png') return blob
  return convertImageBlobToPng(blob)
}

async function convertImageBlobToPng(blob: Blob): Promise<Blob> {
  const url = URL.createObjectURL(blob)
  try {
    const image = await loadImageElement(url)
    const width = image.naturalWidth || image.width
    const height = image.naturalHeight || image.height
    if (!width || !height) throw new Error('图片尺寸读取失败')
    const canvas = document.createElement('canvas')
    canvas.width = width
    canvas.height = height
    const context = canvas.getContext('2d')
    if (!context) throw new Error('浏览器无法创建图片画布')
    context.drawImage(image, 0, 0)
    return await new Promise<Blob>((resolve, reject) => {
      canvas.toBlob((next) => {
        if (next) resolve(next)
        else reject(new Error('图片转换失败'))
      }, 'image/png')
    })
  } finally {
    URL.revokeObjectURL(url)
  }
}

function loadImageElement(src: string): Promise<HTMLImageElement> {
  return new Promise((resolve, reject) => {
    const image = new Image()
    image.onload = () => resolve(image)
    image.onerror = () => reject(new Error('图片加载失败'))
    image.src = src
  })
}

function copyImageAsHTML(src: string) {
  const selection = window.getSelection()
  if (!selection) return false
  const host = document.createElement('div')
  host.contentEditable = 'true'
  host.style.position = 'fixed'
  host.style.left = '-9999px'
  host.style.top = '0'
  host.style.opacity = '0'
  host.style.pointerEvents = 'none'
  const image = document.createElement('img')
  image.src = src
  image.alt = 'generated image'
  host.appendChild(image)
  document.body.appendChild(host)
  const range = document.createRange()
  range.selectNode(image)
  selection.removeAllRanges()
  selection.addRange(range)
  let ok = false
  try {
    ok = document.execCommand('copy')
  } catch {
    ok = false
  } finally {
    selection.removeAllRanges()
    host.remove()
  }
  return ok
}

async function downloadImage(src: string, index: number, setNotice?: (value: string) => void) {
  const absoluteURL = new URL(src, window.location.origin).href
  const fileName = `lyai-image-${Date.now()}-${index + 1}.png`
  try {
    const nativeResult = await nativeSaveImage(absoluteURL, fileName)
    if (nativeResult.handled) {
      if (!nativeResult.ok) throw new Error(nativeResult.message || '保存图片失败')
      if (setNotice) flash(setNotice, '图片已保存到相册')
      return '图片已保存到相册'
    }
  } catch (error) {
    const message = error instanceof Error ? error.message : '保存图片失败'
    if (setNotice) flash(setNotice, message)
    return message
  }

  const response = await fetch(src)
  const blob = await response.blob()
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = `lyai-image-${Date.now()}-${index + 1}.${extensionFromMime(blob.type)}`
  document.body.appendChild(link)
  link.click()
  link.remove()
  URL.revokeObjectURL(url)
  if (setNotice) flash(setNotice, '下载已触发')
  return '下载已触发'
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

function resultLayoutClass(count: number) {
  if (count <= 1) return 'single-result'
  if (count <= 4) return 'few-results'
  return 'many-results'
}

function uniqueRevisedPrompts(task: Task) {
  const seen = new Set<string>()
  return task.results
    .map((result) => (result.revisedPrompt || '').trim())
    .filter((prompt) => {
      if (!prompt || seen.has(prompt)) return false
      seen.add(prompt)
      return true
    })
}

function taskModelLabel(task: Task) {
  if ((task.provider || 'image-2') === BANANA_PROVIDER) {
    const option = getBananaModelOption(task.model || '')
    return `${providerLabel(BANANA_PROVIDER)} · ${option.label}`
  }
  return image2ModelLabel(task.model)
}

function taskModelTag(task: Task) {
  if ((task.provider || 'image-2') === BANANA_PROVIDER) {
    return task.model || getBananaModelOption(task.model || '').id
  }
  return image2ModelLabel(task.model)
}

function image2ModelLabel(model?: string) {
  const normalized = (model || '').trim()
  if (!normalized || normalized === 'image-2' || normalized === DEFAULT_IMAGE2_MODEL) return DEFAULT_IMAGE2_MODEL
  return normalized
}

function squareSubmitReferences(task: Task): SquareSubmitReference[] {
  const references: SquareSubmitReference[] = (task.references || []).map((reference) => ({
    uploadId: reference.uploadId,
    originalName: reference.originalName,
    fileName: reference.fileName,
    mime: reference.mime,
    size: reference.size,
  }))
  const seen = new Set(references.map((reference) => reference.uploadId).filter(Boolean))
  for (const uploadId of task.uploadIds || []) {
    if (!uploadId || seen.has(uploadId)) continue
    seen.add(uploadId)
    references.push({ uploadId })
  }
  return references
}

function squareReferenceUsageNote(task: Task, count: number) {
  if (!count) return ''
  return task.mode === 'image-to-image'
    ? '这些原始参考图用于图生图参考，可能影响主体、风格、构图和整体画面。'
    : '这些原始参考图随公开作品包保存，供别人理解和复用生成过程。'
}

function squareReferenceLabel(reference: SquareSubmitReference, index: number) {
  return reference.originalName || reference.fileName || (reference.uploadId ? `参考图 ${index + 1}` : `参考图快照 ${index + 1}`)
}

function compactUploadId(id: string) {
  if (id.length <= 18) return id
  return `${id.slice(0, 8)}...${id.slice(-6)}`
}

function taskParameters(task: Task) {
  const provider = task.provider || 'image-2'
  if (provider === BANANA_PROVIDER) {
    const option = getBananaModelOption(task.model || '')
    return [
      task.mode === 'gif' ? 'GIF 动图' : task.mode === 'image-to-image' ? '参考图生成' : '文字生成',
      `模型分组 ${providerLabel(provider)}`,
      `规格 ${option.label}`,
      `模型 ID ${option.id}`,
      option.size !== '自动' ? `尺寸 ${option.size}` : '自动尺寸',
      `数量 ${task.count || 1}`,
      `并发 ${task.concurrency || 1}`,
    ]
  }
  return [
    task.mode === 'gif' ? 'GIF 动图' : task.mode === 'image-to-image' ? '参考图生成' : '文字生成',
    `模型 ${taskModelLabel(task)}`,
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
  const labels: Record<string, string> = { auto: '自动', png: 'PNG', jpeg: 'JPG', jpg: 'JPG', webp: 'WEBP', gif: 'GIF' }
  if (!value) return 'PNG'
  return labels[value] || value.toUpperCase()
}

function formatDiagnosticTime(value?: string) {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleTimeString()
}

function renderDiagnosticFields(fields: Record<string, unknown>) {
  return JSON.stringify(fields, null, 2)
}

function flash(setNotice: (value: string) => void, value: string) {
  setNotice(value)
  window.setTimeout(() => setNotice(''), 1600)
}
