import { type FormEvent, useEffect, useState } from 'react'
import { createEpayOrder, getTopUpOptions, listTopUps } from '../api/billing'
import { formatError } from '../api/client'
import { getUserProfile } from '../api/users'
import type { BillingTopUp, EpayMethod, EpayOrder, PublicUser, TopUpOption, UserSession } from '../types'

type TopUpPageProps = {
  session: UserSession
  onSessionChange?: (session: UserSession) => void
}

const DEFAULT_EPAY_METHODS: EpayMethod[] = ['alipay', 'wxpay']

export function TopUpPage({ session, onSessionChange }: TopUpPageProps) {
  const [profile, setProfile] = useState<PublicUser>(session.user)
  const [topupOptions, setTopupOptions] = useState<TopUpOption[]>([])
  const [billingEnabled, setBillingEnabled] = useState(false)
  const [topupMethods, setTopupMethods] = useState<EpayMethod[]>(DEFAULT_EPAY_METHODS)
  const [topups, setTopups] = useState<BillingTopUp[]>([])
  const [selectedCredits, setSelectedCredits] = useState(0)
  const [selectedMethod, setSelectedMethod] = useState<EpayMethod>('alipay')
  const [createdOrder, setCreatedOrder] = useState<EpayOrder | null>(null)
  const [loading, setLoading] = useState(true)
  const [submitting, setSubmitting] = useState(false)
  const [message, setMessage] = useState('')
  const [error, setError] = useState('')

  useEffect(() => {
    void refresh()
  }, [session.user.username])

  async function refresh() {
    setLoading(true)
    setError('')
    const [profileResult, optionsResult, topupsResult] = await Promise.allSettled([
      getUserProfile(),
      getTopUpOptions(),
      listTopUps(),
    ])

    const failures: string[] = []
    if (profileResult.status === 'fulfilled') {
      applyProfile(profileResult.value)
    } else {
      failures.push(formatError(profileResult.reason, '账户资料暂不可用'))
    }
    if (optionsResult.status === 'fulfilled') {
      setBillingEnabled(optionsResult.value.enabled)
      setTopupMethods(optionsResult.value.methods.length ? optionsResult.value.methods : DEFAULT_EPAY_METHODS)
      syncTopUpSelection(optionsResult.value.options, optionsResult.value.methods)
      setTopupOptions(optionsResult.value.options)
    } else {
      failures.push(formatError(optionsResult.reason, '充值档位暂不可用'))
    }
    if (topupsResult.status === 'fulfilled') {
      setTopups(topupsResult.value)
    } else {
      failures.push(formatError(topupsResult.reason, '充值订单暂不可用'))
    }
    setError(failures[0] || '')
    setLoading(false)
  }

  async function refreshTopUpsOnly() {
    setLoading(true)
    setError('')
    try {
      setTopups(await listTopUps())
      const nextProfile = await getUserProfile()
      applyProfile(nextProfile)
    } catch (err) {
      setError(formatError(err, '充值订单刷新失败'))
    } finally {
      setLoading(false)
    }
  }

  async function submitTopUp(event: FormEvent) {
    event.preventDefault()
    if (!selectedTopUpOption || !billingEnabled) return
    setSubmitting(true)
    setError('')
    setMessage('')
    try {
      const methods = normalizeMethods(selectedTopUpOption.methods)
      const method = methods.includes(selectedMethod) ? selectedMethod : methods[0]
      const order = await createEpayOrder({ credits: selectedTopUpOption.credits, method })
      setCreatedOrder(order)
      setTopups((current) => [orderToTopUp(order), ...current.filter((item) => item.tradeNo !== order.tradeNo)])
      const payWindow = order.payUrl ? window.open(order.payUrl, '_blank', 'noopener,noreferrer') : null
      setMessage(payWindow ? '订单已创建，支付页已打开' : '订单已创建，请点击支付链接完成支付')
    } catch (err) {
      setError(formatError(err, '创建充值订单失败'))
    } finally {
      setSubmitting(false)
    }
  }

  function syncTopUpSelection(options: TopUpOption[], fallbackMethods: EpayMethod[] = DEFAULT_EPAY_METHODS) {
    if (!options.length) {
      setSelectedCredits(0)
      const methods = normalizeMethods(undefined, fallbackMethods)
      setSelectedMethod(methods[0] || 'alipay')
      return
    }
    const nextOption = options.find((option) => option.credits === selectedCredits) || options[0]
    const methods = normalizeMethods(nextOption.methods, fallbackMethods)
    setSelectedCredits(nextOption.credits)
    setSelectedMethod(methods.includes(selectedMethod) ? selectedMethod : methods[0] || 'alipay')
  }

  function selectTopUpOption(option: TopUpOption) {
    const methods = normalizeMethods(option.methods, topupMethods)
    setSelectedCredits(option.credits)
    setSelectedMethod((current) => methods.includes(current) ? current : methods[0])
  }

  function applyProfile(nextProfile: PublicUser) {
    setProfile(nextProfile)
    onSessionChange?.({ ...session, user: { ...session.user, ...nextProfile } })
  }

  const selectedTopUpOption = topupOptions.find((option) => option.credits === selectedCredits) || topupOptions[0] || null
  const paymentMethods = billingEnabled ? (selectedTopUpOption ? normalizeMethods(selectedTopUpOption.methods, topupMethods) : normalizeMethods(topupMethods, [])) : []

  return (
    <section className="workflow-page profile-page topup-page" aria-labelledby="topup-page-title">
      <header className="workflow-page-header profile-page-header topup-page-header">
        <div className="profile-identity">
          <div className="profile-avatar" aria-hidden="true">
            {profile.avatarUrl ? <img src={profile.avatarUrl} alt="" /> : <span>{initials(profile.displayName || profile.username)}</span>}
          </div>
          <div>
            <p className="eyebrow">Top Up</p>
            <h2 id="topup-page-title">充值次数</h2>
            <p>{profile.displayName || profile.username} · 当前余额 {formatCredits(profile.creditsBalance)} 次</p>
          </div>
        </div>
        <button type="button" onClick={() => void refresh()} disabled={loading}>{loading ? '刷新中...' : '刷新充值'}</button>
      </header>

      <div className="profile-summary-grid topup-summary-grid">
        <section className="profile-summary-item" aria-label="当前余额">
          <span>当前余额</span>
          <strong>{formatCredits(profile.creditsBalance)}</strong>
          <small>可用于图片生成的剩余次数</small>
        </section>
        <section className="profile-summary-item" aria-label="当前档位">
          <span>当前选择</span>
          <strong>{selectedTopUpOption ? topUpOptionLabel(selectedTopUpOption) : '-'}</strong>
          <small>{selectedTopUpOption ? formatMoney(selectedTopUpOption.amountCents) : '选择充值档位后显示金额'}</small>
        </section>
        <section className="profile-summary-item" aria-label="充值状态">
          <span>充值状态</span>
          <strong>{billingEnabled ? '已开放' : '未开放'}</strong>
          <small>{topupOptions.length ? `${topupOptions.length} 个档位可选` : '等待管理员配置'}</small>
        </section>
      </div>

      <section className="profile-panel profile-topup" aria-labelledby="topup-form-title">
        <div className="panel-title topup-panel-title">
          <strong id="topup-form-title">选择充值套餐</strong>
          <span>{loading ? '同步中' : billingEnabled ? (topupOptions.length ? `${topupOptions.length} 个档位` : '未配置档位') : '暂未开放'}</span>
        </div>
        {error ? <div className="error profile-inline-alert">{error}</div> : null}
        {message ? <div className="ok profile-inline-alert">{message}</div> : null}
        <div className="topup-layout">
          <form className="topup-form" onSubmit={submitTopUp}>
            <fieldset>
              <legend>选择次数</legend>
              {loading ? (
                <div className="topup-skeleton">正在读取充值档位...</div>
              ) : !billingEnabled ? (
                <div className="prompt-empty">充值暂未开放</div>
              ) : topupOptions.length ? (
                <div className="topup-option-list">
                  {topupOptions.map((option) => (
                    <button
                      type="button"
                      key={`${option.credits}-${option.amountCents}`}
                      className={selectedTopUpOption?.credits === option.credits ? 'active' : ''}
                      aria-pressed={selectedTopUpOption?.credits === option.credits}
                      onClick={() => selectTopUpOption(option)}
                    >
                      <strong>{topUpOptionLabel(option)}</strong>
                      <span>{formatMoney(option.amountCents)}</span>
                      {option.bonusCredits ? <small>赠送 {option.bonusCredits} 次</small> : null}
                    </button>
                  ))}
                </div>
              ) : (
                <div className="prompt-empty">暂未配置充值档位</div>
              )}
            </fieldset>
            <fieldset>
              <legend>支付方式</legend>
              {paymentMethods.length ? (
                <div className="topup-method-list">
                  {paymentMethods.map((method) => (
                    <label key={method}>
                      <input
                        type="radio"
                        name="topup-method"
                        value={method}
                        checked={selectedMethod === method}
                        onChange={() => setSelectedMethod(method)}
                      />
                      <span>{methodLabel(method)}</span>
                    </label>
                  ))}
                </div>
              ) : (
                <div className="prompt-empty">{billingEnabled ? '暂无可用支付方式' : '充值开启后显示支付方式'}</div>
              )}
            </fieldset>
            <div className="status-line topup-total">
              <span>应付金额</span>
              <strong>{formatMoney(selectedTopUpOption?.amountCents || 0)}</strong>
            </div>
            <button className="primary" type="submit" disabled={!billingEnabled || !selectedTopUpOption || submitting || loading}>
              {submitting ? '创建中...' : '创建充值订单'}
            </button>
          </form>

          <div className="topup-order-panel">
            <strong>当前订单</strong>
            {createdOrder ? (
              <div className="topup-current-order">
                <span className={`status-pill ${createdOrder.status}`}>{topUpStatusLabel(createdOrder.status)}</span>
                <p>{createdOrder.tradeNo} · {formatCredits(createdOrder.credits)} 次 · {formatMoney(createdOrder.amountCents)}</p>
                {createdOrder.payUrl ? <a href={createdOrder.payUrl} target="_blank" rel="noreferrer">打开支付链接</a> : <small>暂未获得支付链接</small>}
              </div>
            ) : (
              <div className="prompt-empty">创建订单后会在这里显示支付链接</div>
            )}
          </div>
        </div>

        <div className="topup-history-head">
          <strong>充值订单</strong>
          <button type="button" onClick={() => void refreshTopUpsOnly()} disabled={loading}>{loading ? '刷新中...' : '刷新订单'}</button>
        </div>
        {loading ? (
          <div className="prompt-empty">正在同步订单状态...</div>
        ) : topups.length ? (
          <div className="profile-table-wrap">
            <table>
              <thead>
                <tr>
                  <th>订单号</th>
                  <th>次数</th>
                  <th>金额</th>
                  <th>方式</th>
                  <th>状态</th>
                  <th>时间</th>
                </tr>
              </thead>
              <tbody>
                {topups.map((topup) => (
                  <tr key={topup.tradeNo}>
                    <td>{topup.tradeNo}</td>
                    <td>{formatCredits(topup.credits)}</td>
                    <td>{formatMoney(topup.amountCents)}</td>
                    <td>{topup.method ? methodLabel(topup.method) : '-'}</td>
                    <td><span className={`status-pill ${topup.status}`}>{topUpStatusLabel(topup.status)}</span></td>
                    <td>{formatDate(topup.paidAt || topup.createdAt)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : (
          <div className="prompt-empty">暂无充值订单</div>
        )}
      </section>
    </section>
  )
}

function initials(value: string) {
  return value.trim().slice(0, 2).toUpperCase() || 'LY'
}

function normalizeMethods(methods: EpayMethod[] | undefined, fallback: EpayMethod[] = DEFAULT_EPAY_METHODS): EpayMethod[] {
  const values = methods?.length ? methods : fallback
  return Array.from(new Set(values.map((method) => String(method).trim()).filter(Boolean)))
}

function methodLabel(method: EpayMethod | string) {
  const labels: Record<string, string> = {
    alipay: '支付宝',
    wxpay: '微信支付',
    qqpay: 'QQ 钱包',
  }
  return labels[String(method)] || String(method)
}

function topUpOptionLabel(option: TopUpOption) {
  return option.label || `${formatCredits(option.credits)} 次`
}

function formatCredits(value: number | undefined) {
  return Number(value ?? 0).toLocaleString()
}

function formatMoney(value: number | undefined) {
  const cents = Number(value || 0)
  return `¥${(cents / 100).toFixed(2)}`
}

function formatDate(value: string | undefined) {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString()
}

function topUpStatusLabel(status: string) {
  const labels: Record<string, string> = {
    pending: '待支付',
    paid: '已支付',
    succeeded: '已完成',
    success: '已完成',
    failed: '失败',
    cancelled: '已取消',
    canceled: '已取消',
  }
  return labels[status] || status
}

function orderToTopUp(order: EpayOrder): BillingTopUp {
  return {
    tradeNo: order.tradeNo,
    payUrl: order.payUrl,
    credits: order.credits,
    amountCents: order.amountCents,
    status: order.status,
    method: order.method,
    createdAt: order.createdAt || new Date().toISOString(),
    paidAt: order.paidAt,
  }
}
