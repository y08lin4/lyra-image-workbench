import { useEffect } from 'react'
import type { DebugLog, Task, TaskResult } from '../types'
import { providerLabel } from '../lib/models'

type Props = {
  task: Task
  favorite: boolean
  onClose: () => void
  onRetry: (id: string) => void
  onCancel: (id: string) => void
  onDelete: (id: string) => void
  onReuse: (task: Task) => void
  onToggleFavorite: (id: string) => void
  onUseAsReference?: (src: string, index: number) => Promise<unknown>
  onUploadPixhost?: (taskId: string, index: number) => Promise<void>
}

export function TaskDetailModal({
  task,
  onClose,
}: Props) {
  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') onClose()
    }
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [onClose])

  const progress = clampProgress(task.progress)
  const resultSummary = summarizeResults(task)
  const errorSummary = taskErrorSummary(task)
  const parameterItems = taskParameterItems(task)
  const timeItems = taskTimeItems(task)
  const debugItems = taskDebugItems(task)

  return (
    <div className="task-detail-mask" onMouseDown={(event) => event.target === event.currentTarget && onClose()}>
      <section className="task-detail-dialog" role="dialog" aria-modal="true" aria-labelledby="task-detail-title">
        <header className="task-detail-header">
          <div className="task-detail-title">
            <p className="eyebrow">任务详情</p>
            <h2 id="task-detail-title">{task.mode === 'gif' ? 'GIF 动图任务' : task.mode === 'image-to-image' ? '图生图任务' : '文生图任务'}</h2>
            <span>{task.stageText} / {task.stageCode || task.stage} · ID {task.id}</span>
          </div>
          <div className="task-detail-header-actions">
            <span className={`status-pill task-detail-status ${task.status}`}>{task.statusText} / {task.statusCode}</span>
            <button type="button" className="task-detail-close" onClick={onClose} aria-label="关闭详情">×</button>
          </div>
        </header>
        <div className="task-detail-content">
          <section className="task-detail-overview" aria-label="任务概览">
            <DetailItem label="任务 ID" value={task.id} mono />
            <DetailItem label="来源" value={taskSourceLabel(task.source)} />
            <DetailItem label="模型分组" value={providerLabel(task.provider || 'image-2')} />
            <DetailItem label="模型" value={task.model || defaultModelLabel(task.provider)} mono />
          </section>

          <section className="task-detail-progress-panel" aria-label="任务进度">
            <div className="task-detail-progress-head">
              <span>进度</span>
              <strong>{progress}%</strong>
            </div>
            <progress value={progress} max={100} aria-label="任务进度" />
            <p>{task.stageText} / {task.stageCode || task.stage}</p>
          </section>

          <section className={`task-detail-alert ${errorSummary ? 'is-error' : 'is-neutral'}`} aria-label="错误摘要">
            <span>错误</span>
            <p>{errorSummary || '无'}</p>
          </section>

          <section className="task-detail-section" aria-label="提示词摘要">
            <div className="task-detail-section-head">
              <span>提示词摘要</span>
              {task.framePrompts?.length ? <strong>{task.framePrompts.length} 条分镜</strong> : null}
            </div>
            <p className="task-detail-prompt">{compactText(task.prompt || '（无提示词）', 180)}</p>
          </section>

          <section className="task-detail-section" aria-label="参数和结果数量">
            <div className="task-detail-section-head">
              <span>参数 / 结果数量</span>
              <strong>{resultSummary.ok}/{task.count || 1} 成功</strong>
            </div>
            <div className="task-detail-chip-grid">
              {parameterItems.map((item) => (
                <span key={item}>{item}</span>
              ))}
            </div>
            <div className="task-detail-count-grid">
              <DetailItem label="期望数量" value={`${task.count || 1} 张`} />
              <DetailItem label="已返回" value={`${resultSummary.total} 张`} />
              <DetailItem label="成功" value={`${resultSummary.ok} 张`} />
              <DetailItem label="失败" value={`${resultSummary.failed} 张`} tone={resultSummary.failed ? 'error' : undefined} />
              <DetailItem label="等待中" value={`${resultSummary.pending} 张`} />
              <DetailItem label="已上传图床" value={`${resultSummary.uploaded} 张`} />
            </div>
          </section>

          <section className="task-detail-section" aria-label="时间和调试摘要">
            <div className="task-detail-section-head">
              <span>时间 / 调试摘要</span>
              <strong>{task.debugEnabled ? 'Debug 已开启' : 'Debug 未开启'}</strong>
            </div>
            <div className="task-detail-two-column">
              <div className="task-detail-item-list">
                {timeItems.map((item) => (
                  <DetailItem key={item.label} label={item.label} value={item.value} />
                ))}
              </div>
              <div className="task-detail-item-list">
                {debugItems.map((item) => (
                  <DetailItem key={item.label} label={item.label} value={item.value} tone={item.tone} />
                ))}
              </div>
            </div>
          </section>
        </div>
      </section>
    </div>
  )
}

function DetailItem({ label, value, mono, tone }: { label: string; value: string; mono?: boolean; tone?: 'error' }) {
  return (
    <div className={`task-detail-item ${mono ? 'is-mono' : ''} ${tone ? `is-${tone}` : ''}`}>
      <span>{label}</span>
      <strong>{value}</strong>
    </div>
  )
}

function clampProgress(value?: number) {
  if (typeof value !== 'number' || Number.isNaN(value)) return 0
  return Math.max(0, Math.min(100, Math.round(value)))
}

function summarizeResults(task: Task) {
  const total = task.results.length
  const ok = task.results.filter((item) => item.ok).length
  const failed = task.results.filter((item) => !item.ok).length
  const uploaded = task.results.filter((item) => Boolean(item.remoteUrl)).length
  const pending = Math.max((task.count || 1) - total, 0)
  return { total, ok, failed, uploaded, pending }
}

function taskErrorSummary(task: Task) {
  if (task.error) return compactText(task.error, 180)
  const resultError = task.results.map(resultErrorText).find(Boolean)
  return resultError ? compactText(resultError, 180) : ''
}

function resultErrorText(result: TaskResult) {
  return [
    result.errorText,
    result.error,
    result.errorCode,
    result.errorEnglish,
    result.uploadError ? `图床：${result.uploadError}` : '',
  ].filter(Boolean).join(' / ')
}

function taskParameterItems(task: Task) {
  return [
    task.mode === 'gif' ? 'GIF 动图' : task.mode === 'image-to-image' ? '图生图' : '文生图',
    task.ratio && task.ratio !== 'auto' ? `比例 ${task.ratio}` : '自动比例',
    `清晰度 ${resolutionLabel(task.resolution)}`,
    `质量 ${qualityLabel(task.quality)}`,
    `格式 ${outputFormatLabel(task.outputFormat)}`,
    task.size && task.size !== '自动' ? `尺寸 ${task.size}` : '自动尺寸',
    `并发 ${task.concurrency || 1}`,
    `参考图 ${task.references?.length || task.uploadIds?.length || 0}`,
  ]
}

function taskTimeItems(task: Task) {
  return [
    { label: '创建时间', value: formatDateTime(task.createdAt) },
    { label: '开始时间', value: formatDateTime(task.startedAt) },
    { label: '完成时间', value: formatDateTime(task.finishedAt) },
    { label: '更新时间', value: formatDateTime(task.updatedAt) },
  ]
}

function taskDebugItems(task: Task): Array<{ label: string; value: string; tone?: 'error' }> {
  const logs = task.debugLogs || []
  const errorLogs = logs.filter(isErrorLog)
  const lastLog = logs.length ? logs[logs.length - 1] : undefined
  return [
    { label: 'Debug 状态', value: task.debugEnabled ? '已开启' : '未开启' },
    { label: '日志数量', value: `${logs.length} 条` },
    { label: '错误日志', value: `${errorLogs.length} 条`, tone: errorLogs.length ? 'error' : undefined },
    { label: '最后日志', value: lastLog ? formatDebugLog(lastLog) : '无' },
  ]
}

function isErrorLog(log: DebugLog) {
  return (log.level || '').toLowerCase() === 'error'
}

function formatDebugLog(log: DebugLog) {
  return compactText(`${formatDateTime(log.time)} · ${log.stage || 'debug'} · ${log.message}`, 120)
}

function formatDateTime(value?: string) {
  if (!value) return '未记录'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString()
}

function compactText(value: string, maxLength: number) {
  const text = value.trim().replace(/\s+/g, ' ')
  return text.length > maxLength ? `${text.slice(0, maxLength)}...` : text
}

function defaultModelLabel(provider?: string) {
  return provider === 'banana' ? 'gemini-3.1-flash-image-preview' : 'gpt-image-2'
}

function resolutionLabel(value?: string) {
  const labels: Record<string, string> = { auto: '自动', standard: '标准', '2k': '2K', '4k': '4K' }
  return value ? labels[value] || value : '自动'
}

function qualityLabel(value?: string) {
  const labels: Record<string, string> = { auto: '自动', low: '低', medium: '中', high: '高' }
  return value ? labels[value] || value : '自动'
}

function outputFormatLabel(value?: string) {
  const labels: Record<string, string> = { auto: '自动', png: 'PNG', jpeg: 'JPG', jpg: 'JPG', webp: 'WEBP', gif: 'GIF' }
  return value ? labels[value] || value.toUpperCase() : 'PNG'
}

function taskSourceLabel(source?: string) {
  if (source === 'api') return 'API'
  if (!source || source === 'web') return 'Web'
  return source
}
