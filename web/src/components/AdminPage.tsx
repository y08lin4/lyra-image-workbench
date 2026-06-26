import { type FormEvent, useEffect, useMemo, useState } from 'react'
import {
  clearAdminToken,
  getAdminAuthStatus,
  getAdminConfig,
  getAdminToken,
  grantUserCredits,
  listAdminUserLedger,
  listAdminUsers,
  loginAdmin,
  logoutAdmin,
  saveAdminBillingConfig,
  saveAdminConfig,
  setAdminUserRole,
  setupAdminPassword,
} from '../api/admin'
import type { AdminAuthStatus, AdminBillingConfig, AdminConfig, AdminUser, CreditLedgerEntry } from '../types'
import { ThemeToggle, type ThemeMode } from './ThemeToggle'
import { GitHubLink } from './GitHubLink'

type AdminMode = 'loading' | 'setup' | 'login' | 'config'
type NumericInputValue = number | ''

const EPAY_METHOD_CHOICES = [
  { value: 'alipay', label: '支付宝' },
  { value: 'wxpay', label: '微信支付' },
  { value: 'qqpay', label: 'QQ 钱包' },
]

export function AdminPage({ theme, onToggleTheme }: { theme: ThemeMode; onToggleTheme: () => void }) {
  const [mode, setMode] = useState<AdminMode>('loading')
  const [auth, setAuth] = useState<AdminAuthStatus | null>(null)
  const [config, setConfig] = useState<AdminConfig | null>(null)
  const [url, setUrl] = useState('')
  const [publicBaseUrl, setPublicBaseUrl] = useState('')
  const [debugEnabled, setDebugEnabled] = useState(false)
  const [timeout, setTimeoutSec] = useState<NumericInputValue>(600)
  const [epayEnabled, setEpayEnabled] = useState(false)
  const [epayApiUrl, setEpayApiUrl] = useState('')
  const [epayPid, setEpayPid] = useState('')
  const [epayKey, setEpayKey] = useState('')
  const [clearEpayKey, setClearEpayKey] = useState(false)
  const [epayMethods, setEpayMethods] = useState<string[]>(['alipay', 'wxpay'])
  const [creditPriceCents, setCreditPriceCents] = useState<NumericInputValue>(10)
  const [minTopUpCredits, setMinTopUpCredits] = useState<NumericInputValue>(10)
  const [referralRewardCredits, setReferralRewardCredits] = useState<NumericInputValue>(0)
  const [newUserInitialCredits, setNewUserInitialCredits] = useState<NumericInputValue>(0)
  const [dailyFreeCredits, setDailyFreeCredits] = useState<NumericInputValue>(0)
  const [savingBilling, setSavingBilling] = useState(false)
  const [password, setPassword] = useState('')
  const [setupToken, setSetupToken] = useState('')
  const [message, setMessage] = useState('')
  const [error, setError] = useState('')
  const [users, setUsers] = useState<AdminUser[]>([])
  const [usersLoading, setUsersLoading] = useState(false)
  const [userQuery, setUserQuery] = useState('')
  const [selectedLedgerUser, setSelectedLedgerUser] = useState('')
  const [ledger, setLedger] = useState<CreditLedgerEntry[]>([])
  const [ledgerLoading, setLedgerLoading] = useState(false)
  const [grantUsername, setGrantUsername] = useState('')
  const [grantAmount, setGrantAmount] = useState<NumericInputValue>(10)
  const [grantReason, setGrantReason] = useState('')
  const [grantSubmitting, setGrantSubmitting] = useState(false)
  const [roleBusyUser, setRoleBusyUser] = useState('')

  useEffect(() => {
    void boot()
  }, [])

  async function boot() {
    setError('')
    try {
      const status = await getAdminAuthStatus()
      setAuth(status)
      if (!status.passwordSet) {
        setMode('setup')
        return
      }
      if (!getAdminToken()) {
        setMode('login')
        return
      }
      await loadConfig()
    } catch (err) {
      setError(err instanceof Error ? err.message : '读取 Admin 状态失败')
      setMode('login')
    }
  }

  async function loadConfig() {
    try {
      const cfg = await getAdminConfig()
      setConfig(cfg)
      setUrl(cfg.newApiBaseUrl)
      setPublicBaseUrl(cfg.publicBaseUrl || '')
      setDebugEnabled(Boolean(cfg.debugEnabled))
      setTimeoutSec(cfg.timeoutSec)
      applyBillingConfig(cfg)
      setMode('config')
      await loadUsers()
    } catch (err) {
      clearAdminToken()
      setError(err instanceof Error ? err.message : 'Admin 登录已失效')
      setMode('login')
    }
  }

  async function loadUsers() {
    setUsersLoading(true)
    try {
      const nextUsers = await listAdminUsers()
      setUsers(nextUsers)
      if (!grantUsername && nextUsers.length > 0) {
        setGrantUsername(nextUsers[0].username)
      }
      if (selectedLedgerUser && !nextUsers.some((user) => user.username === selectedLedgerUser)) {
        setSelectedLedgerUser('')
        setLedger([])
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : '读取用户列表失败')
    } finally {
      setUsersLoading(false)
    }
  }

  async function refreshUsersFromResponse(nextUser?: AdminUser, nextUsers?: AdminUser[]) {
    if (nextUsers?.length) {
      setUsers(nextUsers)
      return
    }
    if (nextUser) {
      setUsers((current) => current.map((user) => user.username === nextUser.username ? nextUser : user))
      return
    }
    await loadUsers()
  }

  async function submitPassword(event: FormEvent) {
    event.preventDefault()
    setError('')
    setMessage('')
    try {
      if (mode === 'setup') {
        const next = await setupAdminPassword(password, setupToken)
        setAuth(next.auth)
        setMessage('Admin 管理密码已设置')
      } else {
        const next = await loginAdmin(password)
        setAuth(next.auth)
        setMessage('Admin 登录成功')
      }
      setPassword('')
      setSetupToken('')
      await loadConfig()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Admin 鉴权失败')
    }
  }

  async function submit(event: FormEvent) {
    event.preventDefault()
    setError('')
    try {
      const cfg = await saveAdminConfig(url, numericOrDefault(timeout, config?.timeoutSec || 600), publicBaseUrl, debugEnabled)
      setConfig(cfg)
      setPublicBaseUrl(cfg.publicBaseUrl || '')
      setDebugEnabled(Boolean(cfg.debugEnabled))
      applyBillingConfig(cfg)
      setMessage('管理配置已保存')
    } catch (err) {
      setError(err instanceof Error ? err.message : '保存失败')
    }
  }

  async function submitBillingConfig() {
    setSavingBilling(true)
    setError('')
    setMessage('')
    try {
      const cfg = await saveAdminBillingConfig({
        epayEnabled,
        epayApiUrl: epayApiUrl.trim(),
        epayPid: epayPid.trim(),
        epayMethods,
        creditPriceCents: numericOrDefault(creditPriceCents, 0),
        minTopUpCredits: numericOrDefault(minTopUpCredits, 0),
        referralRewardCredits: numericOrDefault(referralRewardCredits, 0),
        newUserInitialCredits: numericOrDefault(newUserInitialCredits, 0),
        dailyFreeCredits: numericOrDefault(dailyFreeCredits, 0),
        ...(epayKey.trim() ? { epayKey: epayKey.trim() } : {}),
        clearEpayKey,
      })
      setConfig(cfg)
      applyBillingConfig(cfg)
      setMessage('额度/易支付配置已保存')
    } catch (err) {
      setError(err instanceof Error ? err.message : '额度/易支付配置保存失败')
    } finally {
      setSavingBilling(false)
    }
  }

  function applyBillingConfig(cfg: AdminConfig) {
    const billing = billingConfigOf(cfg)
    setEpayEnabled(Boolean(billing.epayEnabled))
    setEpayApiUrl(billing.epayApiUrl || '')
    setEpayPid(billing.epayPid || '')
    setEpayMethods(normalizeAdminEpayMethods(billing.epayMethods))
    setCreditPriceCents(billing.creditPriceCents ?? 10)
    setMinTopUpCredits(billing.minTopUpCredits ?? 10)
    setReferralRewardCredits(billing.referralRewardCredits ?? 0)
    setNewUserInitialCredits(billing.newUserInitialCredits ?? 0)
    setDailyFreeCredits(billing.dailyFreeCredits ?? 0)
    setEpayKey('')
    setClearEpayKey(false)
  }

  function toggleEpayMethod(method: string, checked: boolean) {
    setEpayMethods((current) => {
      if (checked) return Array.from(new Set([...current, method]))
      return current.filter((item) => item !== method)
    })
  }

  async function handleLogout() {
    try {
      await logoutAdmin()
    } finally {
      setConfig(null)
      setUsers([])
      setLedger([])
      setSelectedLedgerUser('')
      setMode('login')
      setMessage('已退出 Admin')
    }
  }

  async function submitGrantCredits() {
    setError('')
    setMessage('')
    const username = grantUsername.trim()
    const amount = numericOrDefault(grantAmount, 0)
    const reason = grantReason.trim()
    if (!username) {
      setError('请选择要加次数的用户')
      return
    }
    if (!Number.isFinite(amount) || amount <= 0) {
      setError('加次数数量必须大于 0')
      return
    }
    if (!reason) {
      setError('加次数必须填写原因')
      return
    }
    if (!window.confirm(`确认给 ${username} 增加 ${amount} 次？\n原因：${reason}`)) {
      return
    }
    setGrantSubmitting(true)
    try {
      const result = await grantUserCredits(username, amount, reason)
      await refreshUsersFromResponse(result.user, result.users)
      if (selectedLedgerUser === username) {
        await loadLedger(username)
      }
      setGrantReason('')
      setMessage(`已给 ${username} 增加 ${amount} 次`)
    } catch (err) {
      setError(err instanceof Error ? err.message : '加次数失败')
    } finally {
      setGrantSubmitting(false)
    }
  }

  async function loadLedger(username: string) {
    setError('')
    setMessage('')
    setSelectedLedgerUser(username)
    setLedgerLoading(true)
    try {
      const entries = await listAdminUserLedger(username)
      setLedger(entries)
    } catch (err) {
      setLedger([])
      setError(err instanceof Error ? err.message : '读取用户流水失败')
    } finally {
      setLedgerLoading(false)
    }
  }

  async function toggleAdminRole(user: AdminUser) {
    setError('')
    setMessage('')
    const nextIsAdmin = !user.isAdmin
    const actionText = nextIsAdmin ? '设为管理员' : '取消管理员'
    if (!window.confirm(`确认${actionText}：${user.username}？`)) {
      return
    }
    setRoleBusyUser(user.username)
    try {
      const result = await setAdminUserRole(user.username, nextIsAdmin)
      await refreshUsersFromResponse(result.user, result.users)
      setMessage(`已${actionText}：${user.username}`)
    } catch (err) {
      setError(err instanceof Error ? err.message : `${actionText}失败`)
    } finally {
      setRoleBusyUser('')
    }
  }

  const filteredUsers = useMemo(() => filterAdminUsers(users, userQuery), [users, userQuery])
  const adminCount = useMemo(() => users.filter((user) => user.isAdmin).length, [users])

  if (mode === 'loading') {
    return (
      <main className="center-shell">
        <div className="center-theme-action">
          <GitHubLink compact />
          <ThemeToggle theme={theme} onToggle={onToggleTheme} />
        </div>
        <section className="admin-panel">
          <AdminBrand title="后台管理" />
          <div className="info">正在检查 Admin 鉴权状态...</div>
          {error ? <div className="error">{error}</div> : null}
        </section>
      </main>
    )
  }

  if (mode === 'setup' || mode === 'login') {
    return (
      <main className="center-shell">
        <div className="center-theme-action">
          <GitHubLink compact />
          <ThemeToggle theme={theme} onToggle={onToggleTheme} />
        </div>
        <form className="admin-panel" onSubmit={submitPassword}>
          <AdminBrand title={mode === 'setup' ? '初次设置 Admin 密码' : '输入 Admin 密码'} />
          <p className="muted">{mode === 'setup' ? '这是开放服务的管理入口，初次访问必须先设置管理密码。' : '后续访问 Admin 页面需要先输入管理密码。'}</p>
          <div className="identity-help">
            <strong>管理密码说明</strong>
            <ul>
              <li>用于保护 NewAPI URL、超时时间等本机服务管理配置。</li>
              <li>至少 10 位，建议包含大小写字母、数字和符号。</li>
              <li>后端只保存不可逆哈希，不保存明文密码。</li>
            </ul>
          </div>
          {mode === 'setup' ? (
            <label>安装令牌<input type="password" value={setupToken} onChange={(e) => setSetupToken(e.target.value)} placeholder="填写 LOCAL_IMAGE_ADMIN_SETUP_TOKEN 后再设置管理密码" autoComplete="one-time-code" /></label>
          ) : null}
          <label>Admin 密码<input type="password" value={password} onChange={(e) => setPassword(e.target.value)} placeholder="输入复杂管理密码" autoFocus /></label>
          <button className="primary" type="submit">{mode === 'setup' ? '设置并进入 Admin' : '登录 Admin'}</button>
          <a href="/">返回工作台</a>
          {message ? <div className="ok">{message}</div> : null}
          {error ? <div className="error">{error}</div> : null}
        </form>
      </main>
    )
  }

  return (
    <main className="center-shell">
      <div className="center-theme-action">
        <GitHubLink compact />
        <ThemeToggle theme={theme} onToggle={onToggleTheme} />
      </div>
      <form className="admin-panel admin-panel-wide" onSubmit={submit}>
        <AdminBrand title="后台管理" />
        <div className="status-line">Admin 已登录 · 密码状态：{auth?.passwordSet ? '已设置' : '未设置'}</div>
        <div className="admin-overview-grid" aria-label="后台运营概览">
          <section>
            <span>新用户免费</span>
            <strong>{formatCredits(numericOrDefault(newUserInitialCredits, 0))}</strong>
            <small>注册后初始次数</small>
          </section>
          <section>
            <span>每日免费</span>
            <strong>{formatCredits(numericOrDefault(dailyFreeCredits, 0))}</strong>
            <small>自然日可领取次数</small>
          </section>
          <section>
            <span>易支付</span>
            <strong>{epayEnabled ? '已开启' : '未开启'}</strong>
            <small>{billingConfigOf(config).epayKeySet ? '商户 Key 已保存' : '商户 Key 未设置'}</small>
          </section>
          <section>
            <span>用户</span>
            <strong>{formatCredits(users.length)}</strong>
            <small>{adminCount} 个管理员</small>
          </section>
        </div>
        <label>NewAPI 请求 URL<input value={url} onChange={(e) => setUrl(e.target.value)} placeholder="http://127.0.0.1:3000/v1" /></label>
        <label>对外访问域名<input value={publicBaseUrl} onChange={(e) => setPublicBaseUrl(e.target.value)} placeholder="https://image.example.com，可留空" /></label>
        <label>超时时间（秒）<input type="number" min={config?.limits.minTimeoutSec || 60} max={config?.limits.maxTimeoutSec || 3600} value={timeout} onChange={(e) => setTimeoutSec(readNumberInput(e.target.value))} /></label>
        <label className="check-row admin-debug-toggle">
          <input type="checkbox" checked={debugEnabled} onChange={(e) => setDebugEnabled(e.target.checked)} />
          <span>开启 Debug 日志：新任务会在前端结果页显示脱敏后的请求 URL、参数、上游状态和错误详情</span>
        </label>
        <div className="status-line">当前对外域名：{config?.publicBaseUrl || '未设置'}。用于记录部署域名，反代仍在宝塔/Nginx 里配置。</div>
        <div className="status-line">默认 Image-2 模型：{config?.model || 'gpt-image-2'}；Banana Nano 在工作台按规格路由到独立模型 ID。</div>
        <fieldset className="admin-billing-box">
          <legend>额度与易支付配置</legend>
          <div className="admin-section-title">
            <div>
              <strong>用户充值支付</strong>
              <span>配置易支付网关、免费次数和邀请奖励</span>
            </div>
            <span className={`admin-key-status ${billingConfigOf(config).epayKeySet ? 'ready' : 'missing'}`}>
              {billingConfigOf(config).epayKeySet ? `Key 已设置（${billingConfigOf(config).epayKeyPreview || '已隐藏'}）` : 'Key 未设置'}
            </span>
          </div>
          <label className="check-row admin-debug-toggle">
            <input type="checkbox" checked={epayEnabled} onChange={(e) => setEpayEnabled(e.target.checked)} />
            <span>启用易支付充值</span>
          </label>
          <div className="admin-billing-grid">
            <label>网关地址<input value={epayApiUrl} onChange={(e) => setEpayApiUrl(e.target.value)} placeholder="https://pay.example.com/" /></label>
            <label>商户 PID<input value={epayPid} onChange={(e) => setEpayPid(e.target.value)} placeholder="易支付商户 ID" /></label>
            <label>商户 Key<input type="password" value={epayKey} onChange={(e) => setEpayKey(e.target.value)} placeholder={billingConfigOf(config).epayKeySet ? '已保存，输入新 Key 可覆盖' : '仅保存，不会明文显示'} autoComplete="new-password" /></label>
            <label>次数单价（分）<input type="number" min={0} value={creditPriceCents} onChange={(e) => setCreditPriceCents(readNumberInput(e.target.value))} /></label>
            <label>最小充值次数<input type="number" min={0} value={minTopUpCredits} onChange={(e) => setMinTopUpCredits(readNumberInput(e.target.value))} /></label>
            <label>邀请奖励次数<input type="number" min={0} value={referralRewardCredits} onChange={(e) => setReferralRewardCredits(readNumberInput(e.target.value))} /></label>
            <label>新用户初始免费次数<input type="number" min={0} value={newUserInitialCredits} onChange={(e) => setNewUserInitialCredits(readNumberInput(e.target.value))} /></label>
            <label>每日免费次数<input type="number" min={0} value={dailyFreeCredits} onChange={(e) => setDailyFreeCredits(readNumberInput(e.target.value))} /></label>
          </div>
          <label className="check-row admin-debug-toggle">
            <input type="checkbox" checked={clearEpayKey} onChange={(e) => setClearEpayKey(e.target.checked)} />
            <span>清空已保存的商户 Key</span>
          </label>
          <div className="admin-method-list" role="group" aria-label="支付方式">
            {EPAY_METHOD_CHOICES.map((method) => (
              <label key={method.value} className="check-row">
                <input
                  type="checkbox"
                  checked={epayMethods.includes(method.value)}
                  onChange={(e) => toggleEpayMethod(method.value, e.target.checked)}
                />
                <span>{method.label}</span>
              </label>
            ))}
          </div>
          <div className="admin-billing-actions">
            <button className="primary" type="button" onClick={() => void submitBillingConfig()} disabled={savingBilling}>
              {savingBilling ? '保存中...' : '保存额度/易支付配置'}
            </button>
          </div>
        </fieldset>
        <button className="primary" type="submit">保存管理配置</button>
        <div className="admin-actions"><a href="/">返回工作台</a><button type="button" onClick={handleLogout}>退出 Admin</button></div>
        <section className="admin-users-section" aria-labelledby="admin-users-title">
          <div className="admin-section-heading admin-users-heading">
            <div>
              <h2 id="admin-users-title">用户管理</h2>
              <p className="muted">查看余额、加次数流水和管理员角色。当前展示 {filteredUsers.length} / {users.length} 个用户。</p>
            </div>
            <div className="admin-users-tools">
              <label className="admin-user-search">
                <span>搜索用户</span>
                <input value={userQuery} onChange={(event) => setUserQuery(event.target.value)} placeholder="用户名、显示名或邮箱" />
              </label>
              <button type="button" onClick={() => void loadUsers()} disabled={usersLoading}>{usersLoading ? '刷新中...' : '刷新用户'}</button>
            </div>
          </div>
          <div
            className="admin-grant-form"
            onKeyDown={(event) => {
              if (event.key === 'Enter') {
                event.preventDefault()
                void submitGrantCredits()
              }
            }}
          >
            <label>
              用户
              <select value={grantUsername} onChange={(event) => setGrantUsername(event.target.value)} disabled={grantSubmitting || usersLoading}>
                <option value="">选择用户</option>
                {users.map((user) => (
                  <option key={user.username} value={user.username}>{displayUserLabel(user)}</option>
                ))}
              </select>
            </label>
            <label>
              增加次数
              <input type="number" min={1} step={1} value={grantAmount} onChange={(event) => setGrantAmount(readNumberInput(event.target.value))} />
            </label>
            <label>
              原因 <span className="required-mark">必填</span>
              <input value={grantReason} onChange={(event) => setGrantReason(event.target.value)} placeholder="例如：线下付款补录" />
            </label>
            <button type="button" className="primary" onClick={() => void submitGrantCredits()} disabled={grantSubmitting || usersLoading}>{grantSubmitting ? '提交中...' : '增加次数'}</button>
          </div>
          <div className="profile-table-wrap admin-users-table-wrap">
            <table className="admin-users-table">
              <thead>
                <tr>
                  <th>用户</th>
                  <th>邮箱</th>
                  <th>余额</th>
                  <th>管理员</th>
                  <th>注册时间</th>
                  <th>操作</th>
                </tr>
              </thead>
              <tbody>
                {usersLoading && users.length === 0 ? (
                  <tr><td colSpan={6}>正在读取用户...</td></tr>
                ) : filteredUsers.length === 0 ? (
                  <tr><td colSpan={6}>{users.length ? '没有匹配用户' : '暂无用户数据'}</td></tr>
                ) : filteredUsers.map((user) => (
                  <tr key={user.username}>
                    <td>
                      <strong>{user.displayName || user.username}</strong>
                      <span>{user.username}</span>
                    </td>
                    <td>{user.email || '-'}</td>
                    <td className="numeric-cell">{formatCredits(user.creditsBalance)}</td>
                    <td><span className={user.isAdmin ? 'admin-role-badge admin-role-badge-on' : 'admin-role-badge'}>{user.isAdmin ? '管理员' : '普通用户'}</span></td>
                    <td>{formatDateTime(user.createdAt)}</td>
                    <td>
                      <div className="admin-row-actions">
                        <button type="button" onClick={() => void loadLedger(user.username)} disabled={ledgerLoading && selectedLedgerUser === user.username}>流水</button>
                        <button type="button" onClick={() => void toggleAdminRole(user)} disabled={roleBusyUser === user.username}>{roleBusyUser === user.username ? '处理中...' : user.isAdmin ? '取消管理员' : '设为管理员'}</button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <div className="admin-ledger-section">
            <div className="admin-section-heading compact">
              <div>
                <h2>用户流水</h2>
                <p className="muted">{selectedLedgerUser ? `${selectedLedgerUser} 的额度变动记录` : '从用户列表选择一个用户查看流水。'}</p>
              </div>
              {selectedLedgerUser ? <button type="button" onClick={() => void loadLedger(selectedLedgerUser)} disabled={ledgerLoading}>{ledgerLoading ? '刷新中...' : '刷新流水'}</button> : null}
            </div>
            <div className="profile-table-wrap admin-ledger-table-wrap">
              <table className="admin-ledger-table">
                <thead>
                  <tr>
                    <th>Delta</th>
                    <th>Type</th>
                    <th>Reason</th>
                    <th>AdminActor</th>
                    <th>SourceId</th>
                    <th>CreatedAt</th>
                  </tr>
                </thead>
                <tbody>
                  {ledgerLoading ? (
                    <tr><td colSpan={6}>正在读取流水...</td></tr>
                  ) : !selectedLedgerUser ? (
                    <tr><td colSpan={6}>请选择用户</td></tr>
                  ) : ledger.length === 0 ? (
                    <tr><td colSpan={6}>暂无流水</td></tr>
                  ) : ledger.map((entry) => (
                    <tr key={entry.id}>
                      <td className={entry.delta >= 0 ? 'positive numeric-cell' : 'negative numeric-cell'}>{formatDelta(entry.delta)}</td>
                      <td>{entry.type}</td>
                      <td>{entry.reason || '-'}</td>
                      <td>{entry.adminActor || '-'}</td>
                      <td>{entry.sourceId || '-'}</td>
                      <td>{formatDateTime(entry.createdAt)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        </section>
        {message ? <div className="ok">{message}</div> : null}
        {error ? <div className="error">{error}</div> : null}
      </form>
    </main>
  )
}

function AdminBrand({ title }: { title: string }) {
  return (
    <div className="brand login-brand">
      <div className="brand-mark">Ly</div>
      <div>
        <p className="eyebrow">Admin</p>
        <h1>{title}</h1>
      </div>
    </div>
  )
}

function billingConfigOf(config: AdminConfig | null): AdminBillingConfig {
  const billing = config?.billing || {}
  return {
    epayEnabled: billing.epayEnabled ?? config?.epayEnabled ?? false,
    epayApiUrl: billing.epayApiUrl ?? config?.epayApiUrl ?? '',
    epayPid: billing.epayPid ?? config?.epayPid ?? '',
    epayKeySet: billing.epayKeySet ?? config?.epayKeySet ?? false,
    epayKeyPreview: billing.epayKeyPreview ?? config?.epayKeyPreview ?? '',
    epayMethods: billing.epayMethods ?? config?.epayMethods ?? ['alipay', 'wxpay'],
    creditPriceCents: billing.creditPriceCents ?? config?.creditPriceCents ?? 10,
    minTopUpCredits: billing.minTopUpCredits ?? config?.minTopUpCredits ?? 10,
    referralRewardCredits: billing.referralRewardCredits ?? config?.referralRewardCredits ?? 0,
    newUserInitialCredits: billing.newUserInitialCredits ?? config?.newUserInitialCredits ?? 0,
    dailyFreeCredits: billing.dailyFreeCredits ?? config?.dailyFreeCredits ?? 0,
  }
}

function filterAdminUsers(users: AdminUser[], query: string) {
  const keyword = query.trim().toLowerCase()
  if (!keyword) return users
  return users.filter((user) => [
    user.username,
    user.displayName,
    user.email,
    user.role,
    user.isAdmin ? 'admin 管理员' : 'user 普通用户',
  ].join(' ').toLowerCase().includes(keyword))
}

function normalizeAdminEpayMethods(methods: string[] | undefined) {
  const normalized = (methods?.length ? methods : ['alipay', 'wxpay']).map((method) => method.trim()).filter(Boolean)
  return Array.from(new Set(normalized))
}

function readNumberInput(value: string): NumericInputValue {
  return value === '' ? '' : Number(value)
}

function numericOrDefault(value: NumericInputValue, fallback: number) {
  return value === '' ? fallback : value
}

function displayUserLabel(user: AdminUser) {
  const email = user.email ? ` · ${user.email}` : ''
  return `${user.username}${email}`
}

function formatCredits(value: number) {
  return Number.isFinite(value) ? value.toLocaleString() : '0'
}

function formatDelta(value: number) {
  const normalized = Number.isFinite(value) ? value : 0
  return normalized > 0 ? `+${normalized.toLocaleString()}` : normalized.toLocaleString()
}

function formatDateTime(value?: string) {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString()
}
