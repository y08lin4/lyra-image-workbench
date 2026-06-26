import { type FormEvent, useEffect, useState } from 'react'
import { createEpayOrder, getTopUpOptions, listTopUps } from '../api/billing'
import { formatError } from '../api/client'
import { listMyPromptSquareItems } from '../api/promptSquare'
import { claimDailyCredits, createReferralCode, getUserProfile, listUserLedger, updateUserProfile } from '../api/users'
import type { BillingTopUp, CreditLedgerEntry, EpayMethod, EpayOrder, PromptSquareItem, PublicUser, TopUpOption, UserSession } from '../types'

type ProfileForm = {
  displayName: string
  email: string
  avatarUrl: string
}

type ProfilePageProps = {
  session: UserSession
  onSessionChange?: (session: UserSession) => void
}

const DEFAULT_EPAY_METHODS: EpayMethod[] = ['alipay', 'wxpay']

export function ProfilePage({ session, onSessionChange }: ProfilePageProps) {
  const [profile, setProfile] = useState<PublicUser>(session.user)
  const [form, setForm] = useState<ProfileForm>(() => profileToForm(session.user))
  const [ledger, setLedger] = useState<CreditLedgerEntry[]>([])
  const [items, setItems] = useState<PromptSquareItem[]>([])
  const [topupOptions, setTopupOptions] = useState<TopUpOption[]>([])
  const [billingEnabled, setBillingEnabled] = useState(false)
  const [topupMethods, setTopupMethods] = useState<EpayMethod[]>(DEFAULT_EPAY_METHODS)
  const [topups, setTopups] = useState<BillingTopUp[]>([])
  const [selectedCredits, setSelectedCredits] = useState(0)
  const [selectedMethod, setSelectedMethod] = useState<EpayMethod>('alipay')
  const [createdOrder, setCreatedOrder] = useState<EpayOrder | null>(null)
  const [loading, setLoading] = useState(true)
  const [billingLoading, setBillingLoading] = useState(true)
  const [billingSubmitting, setBillingSubmitting] = useState(false)
  const [dailyClaiming, setDailyClaiming] = useState(false)
  const [saving, setSaving] = useState(false)
  const [message, setMessage] = useState('')
  const [error, setError] = useState('')
  const [billingError, setBillingError] = useState('')

  useEffect(() => {
    void refresh()
  }, [session.user.username])

  async function refresh() {
    setLoading(true)
    setBillingLoading(true)
    setError('')
    setBillingError('')
    const [profileResult, ledgerResult, itemsResult, optionsResult, topupsResult] = await Promise.allSettled([
      getUserProfile(),
      listUserLedger(),
      listMyPromptSquareItems(),
      getTopUpOptions(),
      listTopUps(),
    ])

    const failures: string[] = []
    const billingFailures: string[] = []
    if (profileResult.status === 'fulfilled') {
      applyProfile(profileResult.value)
    } else {
      failures.push(formatError(profileResult.reason, '资料接口暂不可用'))
    }
    if (ledgerResult.status === 'fulfilled') {
      setLedger(ledgerResult.value)
    } else {
      failures.push(formatError(ledgerResult.reason, '额度流水暂不可用'))
    }
    if (itemsResult.status === 'fulfilled') {
      setItems(itemsResult.value)
    } else {
      failures.push(formatError(itemsResult.reason, '我的投稿暂不可用'))
    }
    if (optionsResult.status === 'fulfilled') {
      setBillingEnabled(optionsResult.value.enabled)
      setTopupMethods(optionsResult.value.methods.length ? optionsResult.value.methods : DEFAULT_EPAY_METHODS)
      setTopupOptions(optionsResult.value.options)
      syncTopUpSelection(optionsResult.value.options, optionsResult.value.methods)
    } else {
      billingFailures.push(formatError(optionsResult.reason, '充值档位暂不可用'))
    }
    if (topupsResult.status === 'fulfilled') {
      setTopups(topupsResult.value)
    } else {
      billingFailures.push(formatError(topupsResult.reason, '充值订单暂不可用'))
    }
    setError(failures[0] || '')
    setBillingError(billingFailures[0] || '')
    setLoading(false)
    setBillingLoading(false)
  }

  async function submit(event: FormEvent) {
    event.preventDefault()
    setSaving(true)
    setError('')
    setMessage('')
    try {
      const nextProfile = await updateUserProfile({
        displayName: form.displayName.trim(),
        email: form.email.trim(),
        avatarUrl: form.avatarUrl.trim(),
      })
      applyProfile(nextProfile)
      setMessage('资料已保存')
    } catch (err) {
      setError(formatError(err, '资料保存失败'))
    } finally {
      setSaving(false)
    }
  }

  async function copyReferral() {
    setError('')
    setMessage('')
    try {
      let code = profile.referralCode || ''
      if (!code) {
        const next = await createReferralCode()
        code = next.referralCode
        if (next.user) applyProfile(next.user)
        else if (code) setProfile((current) => ({ ...current, referralCode: code }))
      }
      if (!code) throw new Error('邀请码暂不可用')
      await navigator.clipboard.writeText(code)
      setMessage('邀请码已复制')
    } catch (err) {
      setError(formatError(err, '复制邀请码失败'))
    }
  }

  async function claimDaily() {
    setDailyClaiming(true)
    setError('')
    setMessage('')
    try {
      const result = await claimDailyCredits()
      const entry = result.entry || null
      if (result.user) applyProfile(result.user)
      if (entry) setLedger((current) => [entry, ...current.filter((item) => item.id !== entry.id)])
      if (result.claimed) setMessage(`已领取 ${formatCredits(result.amount)} 次每日免费额度`)
      else if (result.alreadyClaimed) setMessage('今日免费额度已领取')
      else setMessage('每日免费额度暂未开放')
    } catch (err) {
      setError(formatError(err, '领取每日免费额度失败'))
    } finally {
      setDailyClaiming(false)
    }
  }

  async function refreshTopUpsOnly() {
    setBillingLoading(true)
    setBillingError('')
    try {
      setTopups(await listTopUps())
    } catch (err) {
      setBillingError(formatError(err, '充值订单刷新失败'))
    } finally {
      setBillingLoading(false)
    }
  }

  async function submitTopUp(event: FormEvent) {
    event.preventDefault()
    if (!selectedTopUpOption || !billingEnabled) return
    setBillingSubmitting(true)
    setBillingError('')
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
      setBillingError(formatError(err, '创建充值订单失败'))
    } finally {
      setBillingSubmitting(false)
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
    setForm(profileToForm(nextProfile))
    onSessionChange?.({ ...session, user: { ...session.user, ...nextProfile } })
  }

  const selectedTopUpOption = topupOptions.find((option) => option.credits === selectedCredits) || topupOptions[0] || null
  const paymentMethods = billingEnabled ? (selectedTopUpOption ? normalizeMethods(selectedTopUpOption.methods, topupMethods) : normalizeMethods(topupMethods, [])) : []

  return (
    <section className="workflow-page profile-page" aria-labelledby="profile-title">
      <header className="workflow-page-header profile-page-header">
        <div className="profile-identity">
          <div className="profile-avatar" aria-hidden="true">
            {profile.avatarUrl ? <img src={profile.avatarUrl} alt="" /> : <span>{initials(profile.displayName || profile.username)}</span>}
          </div>
          <div>
            <p className="eyebrow">Profile</p>
            <h2 id="profile-title">我的资料</h2>
            <p>{profile.displayName || profile.username} · {profile.username}</p>
          </div>
        </div>
        <button type="button" onClick={() => void refresh()} disabled={loading}>{loading ? '刷新中...' : '刷新资料'}</button>
      </header>

      <div className="profile-summary-grid">
        <section className="profile-summary-item" aria-label="额度余额">
          <span>当前余额</span>
          <strong>{formatCredits(profile.creditsBalance)}</strong>
          <small>生成次数余额</small>
          <button type="button" onClick={() => void claimDaily()} disabled={dailyClaiming}>{dailyClaiming ? '领取中...' : '领取每日免费'}</button>
        </section>
        <section className="profile-summary-item" aria-label="邀请码">
          <span>邀请码</span>
          <strong>{profile.referralCode || '待生成'}</strong>
          <button type="button" onClick={() => void copyReferral()}>{profile.referralCode ? '复制邀请码' : '生成并复制'}</button>
        </section>
        <section className="profile-summary-item" aria-label="邀请来源">
          <span>邀请来源</span>
          <strong>{profile.referredByUsername || '无'}</strong>
          <small>{profile.isAdmin ? '管理员账号' : '普通账号'}</small>
        </section>
      </div>

      <section className="profile-panel profile-topup" aria-labelledby="topup-title">
        <div className="panel-title">
          <strong id="topup-title">充值次数</strong>
          <span>{billingLoading ? '同步中' : billingEnabled ? (topupOptions.length ? `${topupOptions.length} 个档位` : '未配置档位') : '暂未开放'}</span>
        </div>
        {billingError ? <div className="error profile-inline-alert">{billingError}</div> : null}
        <div className="topup-layout">
          <form className="topup-form" onSubmit={submitTopUp}>
            <fieldset>
              <legend>选择次数</legend>
              {billingLoading ? (
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
            <button className="primary" type="submit" disabled={!billingEnabled || !selectedTopUpOption || billingSubmitting || billingLoading}>
              {billingSubmitting ? '创建中...' : '创建充值订单'}
            </button>
          </form>

          <div className="topup-order-panel">
            <strong>当前订单</strong>
            {createdOrder ? (
              <div className="topup-current-order">
                <span className={`status-pill ${createdOrder.status}`}>{topUpStatusLabel(createdOrder.status)}</span>
                <p>{createdOrder.tradeNo} · {formatCredits(createdOrder.credits)} 次 · {formatMoney(createdOrder.amountCents)}</p>
                {createdOrder.payUrl ? <a href={createdOrder.payUrl} target="_blank" rel="noreferrer">打开支付链接</a> : <small>后端未返回支付链接</small>}
              </div>
            ) : (
              <div className="prompt-empty">创建订单后会在这里显示支付链接</div>
            )}
          </div>
        </div>

        <div className="topup-history-head">
          <strong>充值订单</strong>
          <button type="button" onClick={() => void refreshTopUpsOnly()} disabled={billingLoading}>{billingLoading ? '刷新中...' : '刷新订单'}</button>
        </div>
        {billingLoading ? (
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

      <div className="profile-page-grid">
        <form className="profile-panel profile-form" onSubmit={submit}>
          <div className="panel-title">
            <strong>资料编辑</strong>
            <span>修改公开显示资料和联系邮箱</span>
          </div>
          <label>
            <span>用户名</span>
            <input value={profile.username} readOnly />
          </label>
          <label>
            <span>显示名</span>
            <input value={form.displayName} onChange={(event) => setForm({ ...form, displayName: event.target.value })} placeholder="显示在广场和个人页" />
          </label>
          <label>
            <span>邮箱</span>
            <input type="email" value={form.email} onChange={(event) => setForm({ ...form, email: event.target.value })} placeholder="alice@example.com" />
          </label>
          <label>
            <span>头像 URL</span>
            <input value={form.avatarUrl} onChange={(event) => setForm({ ...form, avatarUrl: event.target.value })} placeholder="https://example.com/avatar.png" />
          </label>
          <button className="primary" type="submit" disabled={saving}>{saving ? '保存中...' : '保存资料'}</button>
        </form>

        <section className="profile-panel profile-ledger" aria-labelledby="ledger-title" aria-busy={loading}>
          <div className="panel-title">
            <strong id="ledger-title">额度流水</strong>
            <span>{ledger.length} 条记录</span>
          </div>
          {loading ? (
            <div className="prompt-empty">正在加载额度流水...</div>
          ) : ledger.length ? (
            <div className="profile-table-wrap">
              <table>
                <thead>
                  <tr>
                    <th>时间</th>
                    <th>类型</th>
                    <th>变动</th>
                    <th>余额</th>
                    <th>原因</th>
                  </tr>
                </thead>
                <tbody>
                  {ledger.map((entry) => (
                    <tr key={entry.id}>
                      <td>{formatDate(entry.createdAt)}</td>
                      <td>{ledgerTypeLabel(entry.type)}</td>
                      <td className={entry.delta >= 0 ? 'positive' : 'negative'}>{formatDelta(entry.delta)}</td>
                      <td>{formatCredits(entry.balanceAfter)}</td>
                      <td>{entry.reason || entry.sourceId || '-'}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : (
            <div className="prompt-empty">暂无额度流水</div>
          )}
        </section>
      </div>

      <section className="profile-panel profile-submissions" aria-labelledby="submissions-title" aria-busy={loading}>
        <div className="panel-title">
          <strong id="submissions-title">我的投稿</strong>
          <span>{items.length} 个作品</span>
        </div>
        {loading ? (
          <div className="prompt-empty">正在加载我的投稿...</div>
        ) : items.length ? (
          <div className="profile-submission-list">
            {items.map((item) => (
              <article key={item.id} className="profile-submission-item">
                {item.thumbnailUrl || item.imageUrl ? <img src={item.thumbnailUrl || item.imageUrl} alt={item.title} loading="lazy" /> : <div className="prompt-square-placeholder">Prompt</div>}
                <div>
                  <strong>{item.title || '未命名作品'}</strong>
                  <p>{item.prompt}</p>
                  <small>{item.model || 'unknown model'} · {item.ratio || item.params?.ratio || 'ratio'} · {formatDate(item.submittedAt || item.createdAt)}</small>
                </div>
                <span>{item.likeCount ?? item.likes ?? 0} likes</span>
              </article>
            ))}
          </div>
        ) : (
          <div className="prompt-empty">还没有投稿到广场的作品</div>
        )}
      </section>

      {message ? <div className="ok">{message}</div> : null}
      {error ? <div className="error">{error}</div> : null}
    </section>
  )
}

function profileToForm(user: PublicUser): ProfileForm {
  return {
    displayName: user.displayName || user.username || '',
    email: user.email || '',
    avatarUrl: user.avatarUrl || '',
  }
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
  return String(value ?? 0)
}

function formatMoney(value: number | undefined) {
  const cents = Number(value || 0)
  return `¥${(cents / 100).toFixed(2)}`
}

function formatDelta(value: number) {
  return value > 0 ? '+' + value : String(value)
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

function ledgerTypeLabel(type: string) {
  const labels: Record<string, string> = {
    initial_free: '新用户赠送',
    daily_free: '每日赠送',
    admin_add: '管理员增加',
    purchase: '充值',
    referral_reward: '邀请奖励',
    task_charge: '任务消耗',
    task_refund: '任务退回',
    refund: '退回',
  }
  return labels[type] || type
}
