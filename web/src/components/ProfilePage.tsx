import { type FormEvent, useEffect, useState } from 'react'
import { formatError } from '../api/client'
import { listMyPromptSquareItems } from '../api/promptSquare'
import { claimDailyCredits, createReferralCode, getUserProfile, listUserLedger, updateUserProfile } from '../api/users'
import type { CreditLedgerEntry, PromptSquareItem, PublicUser, UserSession } from '../types'

type ProfileForm = {
  displayName: string
  email: string
  avatarUrl: string
}

type ProfilePageProps = {
  session: UserSession
  onSessionChange?: (session: UserSession) => void
  onOpenTopUp?: () => void
}

export function ProfilePage({ session, onSessionChange, onOpenTopUp }: ProfilePageProps) {
  const [profile, setProfile] = useState<PublicUser>(session.user)
  const [form, setForm] = useState<ProfileForm>(() => profileToForm(session.user))
  const [ledger, setLedger] = useState<CreditLedgerEntry[]>([])
  const [items, setItems] = useState<PromptSquareItem[]>([])
  const [loading, setLoading] = useState(true)
  const [dailyClaiming, setDailyClaiming] = useState(false)
  const [saving, setSaving] = useState(false)
  const [message, setMessage] = useState('')
  const [error, setError] = useState('')

  useEffect(() => {
    void refresh()
  }, [session.user.username])

  async function refresh() {
    setLoading(true)
    setError('')
    const [profileResult, ledgerResult, itemsResult] = await Promise.allSettled([
      getUserProfile(),
      listUserLedger(),
      listMyPromptSquareItems(),
    ])

    const failures: string[] = []
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
    setError(failures[0] || '')
    setLoading(false)
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
      if (!code) throw new Error('邀请链接暂不可用')
      await navigator.clipboard.writeText(buildReferralLink(code))
      setMessage('邀请链接已复制')
    } catch (err) {
      setError(formatError(err, '复制邀请链接失败'))
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

  function applyProfile(nextProfile: PublicUser) {
    setProfile(nextProfile)
    setForm(profileToForm(nextProfile))
    onSessionChange?.({ ...session, user: { ...session.user, ...nextProfile } })
  }


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
        <section className="profile-summary-item" aria-label="邀请链接">
          <span>邀请链接</span>
          <strong>{profile.referralCode || '待生成'}</strong>
          <small className="referral-link-preview">{profile.referralCode ? buildReferralLink(profile.referralCode) : '生成后可复制注册链接'}</small>
          <button type="button" onClick={() => void copyReferral()}>{profile.referralCode ? '复制邀请链接' : '生成并复制'}</button>
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
          <span>独立页面</span>
        </div>
        <div className="prompt-empty">
          充值套餐、支付链接和订单记录已移到独立充值页。
          <button type="button" onClick={onOpenTopUp}>去充值</button>
        </div>
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

function buildReferralLink(code: string) {
  const trimmed = code.trim()
  if (!trimmed) return ''
  const url = new URL('/', window.location.origin)
  url.searchParams.set('ref', trimmed)
  return url.toString()
}

function initials(value: string) {
  return value.trim().slice(0, 2).toUpperCase() || 'LY'
}

function formatCredits(value: number | undefined) {
  return String(value ?? 0)
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
