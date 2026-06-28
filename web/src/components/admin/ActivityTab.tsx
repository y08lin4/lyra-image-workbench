import { useMemo, useState } from 'react'
import type { AdminActivityEvent } from '../../types'
import { formatCredits, formatDateTime, formatDelta } from './adminHelpers'

type ActivityFilter = 'all' | 'registration' | 'recharge' | 'task_error'

type ActivityTabProps = {
  events: AdminActivityEvent[]
  loading: boolean
  error: string
  source: 'api' | 'derived' | 'none'
  onRefresh: () => void
}

const FILTERS: Array<{ id: ActivityFilter; label: string }> = [
  { id: 'all', label: '全部' },
  { id: 'registration', label: '注册' },
  { id: 'recharge', label: '充值' },
  { id: 'task_error', label: '错误' },
]

export function ActivityTab({ events, loading, error, source, onRefresh }: ActivityTabProps) {
  const [filter, setFilter] = useState<ActivityFilter>('all')
  const summary = useMemo(() => activitySummary(events), [events])
  const filteredEvents = useMemo(() => {
    if (filter === 'all') return events
    return events.filter((event) => event.kind === filter)
  }, [events, filter])

  return (
    <section className="admin-tab-panel admin-activity-panel" id="admin-panel-activity" role="tabpanel" aria-labelledby="admin-tab-activity">
      <div className="admin-section-heading admin-activity-heading">
        <div>
          <h2>全局活动</h2>
          <p className="muted">集中查看用户注册、充值入账和任务错误动态。</p>
        </div>
        <div className="admin-panel-actions">
          <button type="button" onClick={onRefresh} disabled={loading}>{loading ? '同步中...' : '刷新活动'}</button>
        </div>
      </div>

      {error ? (
        <div className="info admin-activity-note">
          {error}
        </div>
      ) : null}

      <div className="admin-activity-summary" aria-label="活动概览">
        <ActivityMetric label="新注册" value={summary.registration} />
        <ActivityMetric label="充值入账" value={summary.recharge} />
        <ActivityMetric label="任务错误" value={summary.taskError} tone={summary.taskError ? 'error' : undefined} />
        <ActivityMetric label="额度调整" value={summary.creditAdjustment} />
      </div>

      <div className="admin-activity-toolbar">
        <div className="admin-activity-filter" role="group" aria-label="筛选活动类型">
          {FILTERS.map((item) => (
            <button
              key={item.id}
              type="button"
              className={filter === item.id ? 'active' : undefined}
              aria-pressed={filter === item.id}
              onClick={() => setFilter(item.id)}
            >
              {item.label}
            </button>
          ))}
        </div>
        <span>{source === 'api' ? '实时活动源' : source === 'derived' ? '由用户与流水推导' : '等待同步'}</span>
      </div>

      <div className="profile-table-wrap admin-activity-table-wrap">
        <table className="admin-activity-table">
          <thead>
            <tr>
              <th>时间</th>
              <th>类型</th>
              <th>用户</th>
              <th>动态</th>
              <th>关联</th>
            </tr>
          </thead>
          <tbody>
            {loading && events.length === 0 ? (
              <tr><td colSpan={5}>正在读取活动...</td></tr>
            ) : filteredEvents.length === 0 ? (
              <tr><td colSpan={5}>{emptyActivityText(filter, source)}</td></tr>
            ) : filteredEvents.map((event) => (
              <tr key={event.id}>
                <td>{formatDateTime(activityTime(event))}</td>
                <td><span className={`admin-activity-badge ${activityTone(event.kind)}`}>{activityLabel(event.kind)}</span></td>
                <td>
                  <strong>{event.displayName || event.username || '-'}</strong>
                  {event.username && event.displayName && event.displayName !== event.username ? <span>{event.username}</span> : null}
                  {event.email ? <span>{event.email}</span> : null}
                </td>
                <td>
                  <strong>{event.title || defaultActivityTitle(event)}</strong>
                  <span>{activityDescription(event)}</span>
                </td>
                <td>{activityReference(event)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </section>
  )
}

function ActivityMetric({ label, value, tone }: { label: string; value: number; tone?: 'error' }) {
  return (
    <article className={`admin-activity-metric ${tone ? `is-${tone}` : ''}`}>
      <span>{label}</span>
      <strong>{value.toLocaleString()}</strong>
    </article>
  )
}

function activitySummary(events: AdminActivityEvent[]) {
  return {
    registration: events.filter((event) => event.kind === 'registration').length,
    recharge: events.filter((event) => event.kind === 'recharge').length,
    taskError: events.filter((event) => event.kind === 'task_error').length,
    creditAdjustment: events.filter((event) => event.kind === 'credit_adjustment').length,
  }
}

function activityTime(event: AdminActivityEvent) {
  return event.occurredAt || event.createdAt || ''
}

function activityLabel(kind: string) {
  if (kind === 'registration') return '注册'
  if (kind === 'recharge') return '充值'
  if (kind === 'task_error') return '任务错误'
  if (kind === 'credit_adjustment') return '额度调整'
  return kind || '活动'
}

function activityTone(kind: string) {
  if (kind === 'task_error') return 'is-error'
  if (kind === 'recharge') return 'is-success'
  if (kind === 'registration') return 'is-info'
  return 'is-neutral'
}

function defaultActivityTitle(event: AdminActivityEvent) {
  if (event.kind === 'registration') return '用户完成注册'
  if (event.kind === 'recharge') return '充值成功入账'
  if (event.kind === 'task_error') return '任务执行失败'
  if (event.kind === 'credit_adjustment') return '管理员调整额度'
  return '系统活动'
}

function activityDescription(event: AdminActivityEvent) {
  if (event.description) return event.description
  if (event.kind === 'registration') return event.email ? `邮箱：${event.email}` : '新用户账号已创建'
  if (event.kind === 'recharge') {
    const credits = event.credits ?? event.delta ?? 0
    const amount = typeof event.amountCents === 'number' ? ` · ${formatMoney(event.amountCents)}` : ''
    return `${formatCredits(credits)} 次${amount}`
  }
  if (event.kind === 'task_error') return [event.errorCode, event.errorText].filter(Boolean).join(' / ') || '任务返回错误'
  if (event.kind === 'credit_adjustment') {
    const actor = event.actor ? ` · 操作人：${event.actor}` : ''
    return `${formatDelta(event.delta ?? 0)} 次${actor}`
  }
  return '-'
}

function activityReference(event: AdminActivityEvent) {
  if (event.taskId) return event.taskId
  if (event.sourceId) return event.sourceId
  if (event.status) return event.status
  return '-'
}

function formatMoney(cents: number) {
  return `¥${(cents / 100).toFixed(2)}`
}

function emptyActivityText(filter: ActivityFilter, source: ActivityTabProps['source']) {
  if (filter === 'task_error' && source === 'derived') return '当前数据不包含全局任务错误，等待活动日志源返回。'
  if (filter === 'all') return '暂无活动记录'
  return `暂无${FILTERS.find((item) => item.id === filter)?.label || ''}记录`
}