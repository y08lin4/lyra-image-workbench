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
  saveAdminEmailConfig,
  setAdminUserRole,
  setupAdminSite,
} from '../api/admin'
import type { AdminAuthStatus, AdminConfig, AdminUser, CreditLedgerEntry } from '../types'
import { ThemeToggle, type ThemeMode } from './ThemeToggle'
import { GitHubLink } from './GitHubLink'
import { AdminTabs, ADMIN_TABS, type AdminTab } from './admin/AdminTabs'
import { BillingTab } from './admin/BillingTab'
import { EmailTab } from './admin/EmailTab'
import { LedgerTab } from './admin/LedgerTab'
import { OverviewTab } from './admin/OverviewTab'
import { SystemTab } from './admin/SystemTab'
import { UsersTab } from './admin/UsersTab'
import {
  billingConfigOf,
  emailConfigOf,
  filterAdminUsers,
  normalizeAdminEpayMethods,
  numericOrDefault,
  readNumberInput,
  type NumericInputValue,
} from './admin/adminHelpers'
import './AdminPage.css'

type AdminMode = 'loading' | 'setup' | 'login' | 'config'
export function AdminPage({ theme, onToggleTheme }: { theme: ThemeMode; onToggleTheme: () => void }) {
  const [mode, setMode] = useState<AdminMode>('loading')
  const [activeTab, setActiveTab] = useState<AdminTab>('overview')
  const [auth, setAuth] = useState<AdminAuthStatus | null>(null)
  const [config, setConfig] = useState<AdminConfig | null>(null)
  const [siteName, setSiteName] = useState('Lyra Image Workbench')
  const [adminUsername, setAdminUsername] = useState('admin')
  const [adminEmail, setAdminEmail] = useState('')
  const [url, setUrl] = useState('http://127.0.0.1:3000/v1')
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
  const [smtpEnabled, setSmtpEnabled] = useState(false)
  const [smtpHost, setSmtpHost] = useState('')
  const [smtpPort, setSmtpPort] = useState<NumericInputValue>(587)
  const [smtpUser, setSmtpUser] = useState('')
  const [smtpPassword, setSmtpPassword] = useState('')
  const [smtpFrom, setSmtpFrom] = useState('')
  const [smtpSecure, setSmtpSecure] = useState(false)
  const [clearSmtpPassword, setClearSmtpPassword] = useState(false)
  const [savingEmail, setSavingEmail] = useState(false)
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
      setSiteName(cfg.siteName || 'Lyra Image Workbench')
      setUrl(cfg.newApiBaseUrl)
      setPublicBaseUrl(cfg.publicBaseUrl || '')
      setDebugEnabled(Boolean(cfg.debugEnabled))
      setTimeoutSec(cfg.timeoutSec)
      applyBillingConfig(cfg)
      applyEmailConfig(cfg)
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
        const next = await setupAdminSite({
          siteName: siteName.trim() || 'Lyra Image Workbench',
          admin: {
            username: adminUsername.trim(),
            email: adminEmail.trim() || undefined,
            password,
          },
          config: {
            newApiBaseUrl: url.trim(),
            publicBaseUrl: publicBaseUrl.trim() || undefined,
            timeoutSec: numericOrDefault(timeout, 600),
            debugEnabled,
            newUserInitialCredits: numericOrDefault(newUserInitialCredits, 0),
            dailyFreeCredits: numericOrDefault(dailyFreeCredits, 0),
          },
        }, setupToken)
        setAuth(next.auth)
        if (next.config) {
          setConfig(next.config)
          setSiteName(next.config.siteName || siteName)
          setUrl(next.config.newApiBaseUrl)
          setPublicBaseUrl(next.config.publicBaseUrl || '')
          setDebugEnabled(Boolean(next.config.debugEnabled))
          setTimeoutSec(next.config.timeoutSec)
          applyBillingConfig(next.config)
          applyEmailConfig(next.config)
        }
        setMessage('站点初始化完成')
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
      const cfg = await saveAdminConfig(siteName, url, numericOrDefault(timeout, config?.timeoutSec || 600), publicBaseUrl, debugEnabled)
      setConfig(cfg)
      setSiteName(cfg.siteName || 'Lyra Image Workbench')
      setPublicBaseUrl(cfg.publicBaseUrl || '')
      setDebugEnabled(Boolean(cfg.debugEnabled))
      applyBillingConfig(cfg)
      applyEmailConfig(cfg)
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

  async function submitEmailConfig() {
    setSavingEmail(true)
    setError('')
    setMessage('')
    try {
      const cfg = await saveAdminEmailConfig({
        smtpEnabled,
        smtpHost: smtpHost.trim(),
        smtpPort: numericOrDefault(smtpPort, 587),
        smtpUser: smtpUser.trim(),
        smtpFrom: smtpFrom.trim(),
        smtpSecure,
        ...(smtpPassword.trim() ? { smtpPassword: smtpPassword.trim() } : {}),
        clearSmtpPassword,
      })
      setConfig(cfg)
      applyEmailConfig(cfg)
      setMessage('邮件发件配置已保存')
    } catch (err) {
      setError(err instanceof Error ? err.message : '邮件发件配置保存失败')
    } finally {
      setSavingEmail(false)
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

  function applyEmailConfig(cfg: AdminConfig) {
    const email = emailConfigOf(cfg)
    setSmtpEnabled(Boolean(email.smtpEnabled))
    setSmtpHost(email.smtpHost || '')
    setSmtpPort(email.smtpPort ?? 587)
    setSmtpUser(email.smtpUser || '')
    setSmtpFrom(email.smtpFrom || '')
    setSmtpSecure(Boolean(email.smtpSecure))
    setSmtpPassword('')
    setClearSmtpPassword(false)
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
    setActiveTab('ledger')
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
  const billingConfig = billingConfigOf(config)
  const emailConfig = emailConfigOf(config)
  const activeTabMeta = ADMIN_TABS.find((tab) => tab.id === activeTab) ?? ADMIN_TABS[0]

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
        <form className={mode === 'setup' ? 'admin-panel admin-setup-panel' : 'admin-panel'} onSubmit={submitPassword}>
          <AdminBrand title={mode === 'setup' ? '初始化站点' : '输入 Admin 密码'} />
          {mode === 'setup' ? (
            <>
              <p className="muted">首次部署需要完成站点名称、管理员账号和基础配置。初始化完成后，只有管理员才能看到后台设置入口。</p>
              <div className="identity-help">
                <strong>初始化说明</strong>
                <ul>
                  <li>安装令牌来自服务端环境变量 LOCAL_IMAGE_ADMIN_SETUP_TOKEN。</li>
                  <li>服务端未设置 LOCAL_IMAGE_ADMIN_SETUP_TOKEN 时会拒绝初始化。</li>
                  <li>生产部署必须先设置 LOCAL_IMAGE_ADMIN_SETUP_TOKEN，再打开初始化页面。</li>
                  <li>管理员密码至少 10 位，建议包含大小写字母、数字和符号。</li>
                  <li>后端只保存不可逆哈希，不保存明文密码。</li>
                </ul>
              </div>
              <div className="admin-setup-grid">
                <label className="wide">安装令牌<input type="password" value={setupToken} onChange={(e) => setSetupToken(e.target.value)} placeholder="LOCAL_IMAGE_ADMIN_SETUP_TOKEN" autoComplete="one-time-code" /></label>
                <label>站点名称<input value={siteName} onChange={(e) => setSiteName(e.target.value)} placeholder="Lyra Image Workbench" /></label>
                <label>管理员用户名<input value={adminUsername} onChange={(e) => setAdminUsername(e.target.value)} placeholder="admin" autoComplete="username" /></label>
                <label>管理员邮箱<input type="email" value={adminEmail} onChange={(e) => setAdminEmail(e.target.value)} placeholder="admin@example.com，可留空" autoComplete="email" /></label>
                <label>管理员密码<input type="password" value={password} onChange={(e) => setPassword(e.target.value)} placeholder="输入复杂管理密码" autoComplete="new-password" autoFocus /></label>
                <label className="wide">NewAPI 请求 URL<input value={url} onChange={(e) => setUrl(e.target.value)} placeholder="http://127.0.0.1:3000/v1" /></label>
                <label className="wide">对外访问域名<input value={publicBaseUrl} onChange={(e) => setPublicBaseUrl(e.target.value)} placeholder="https://image.example.com，可留空" /></label>
                <label>超时时间（秒）<input type="number" min={60} max={3600} value={timeout} onChange={(e) => setTimeoutSec(readNumberInput(e.target.value))} /></label>
                <label>新用户初始免费次数<input type="number" min={0} value={newUserInitialCredits} onChange={(e) => setNewUserInitialCredits(readNumberInput(e.target.value))} /></label>
                <label>每日免费次数<input type="number" min={0} value={dailyFreeCredits} onChange={(e) => setDailyFreeCredits(readNumberInput(e.target.value))} /></label>
                <label className="check-row admin-debug-toggle wide">
                  <input type="checkbox" checked={debugEnabled} onChange={(e) => setDebugEnabled(e.target.checked)} />
                  <span>开启 Debug 日志：任务结果页会显示脱敏后的请求参数、上游状态和错误详情</span>
                </label>
              </div>
              <button className="primary" type="submit">初始化站点并进入 Admin</button>
            </>
          ) : (
            <>
              <p className="muted">后续访问 Admin 页面需要先输入管理密码。</p>
              <div className="identity-help">
                <strong>管理密码说明</strong>
                <ul>
                  <li>用于保护 NewAPI URL、额度、用户、支付等站点管理配置。</li>
                  <li>后端只保存不可逆哈希，不保存明文密码。</li>
                </ul>
              </div>
              <label>Admin 密码<input type="password" value={password} onChange={(e) => setPassword(e.target.value)} placeholder="输入管理密码" autoFocus /></label>
              <button className="primary" type="submit">登录 Admin</button>
            </>
          )}
          <a href="/">返回工作台</a>
          {message ? <div className="ok">{message}</div> : null}
          {error ? <div className="error">{error}</div> : null}
        </form>
      </main>
    )
  }

  return (
    <main className="center-shell admin-page-shell">
      <div className="center-theme-action">
        <GitHubLink compact />
        <ThemeToggle theme={theme} onToggle={onToggleTheme} />
      </div>
      <section className="admin-panel admin-panel-wide admin-console" aria-labelledby="admin-console-title">
        <header className="admin-console-head">
          <AdminBrand title="后台管理" />
          <div className="admin-console-actions">
            <a href="/">返回工作台</a>
            <button type="button" onClick={handleLogout}>退出 Admin</button>
          </div>
        </header>

        <div className="status-line admin-auth-status">Admin 已登录 · 密码状态：{auth?.passwordSet ? '已设置' : '未设置'}</div>

        <AdminTabs activeTab={activeTab} onSelectTab={setActiveTab} />

        <div className="admin-tab-intro">
          <h2 id="admin-console-title">{activeTabMeta.label}</h2>
          <p>{activeTabMeta.description}</p>
        </div>

        {message ? <div className="ok">{message}</div> : null}
        {error ? <div className="error">{error}</div> : null}

        {activeTab === 'overview' ? (
          <OverviewTab
            siteName={siteName}
            newUserInitialCredits={newUserInitialCredits}
            dailyFreeCredits={dailyFreeCredits}
            epayEnabled={epayEnabled}
            smtpEnabled={smtpEnabled}
            usersCount={users.length}
            adminCount={adminCount}
            billingConfig={billingConfig}
            emailConfig={emailConfig}
            config={config}
          />
        ) : null}

        {activeTab === 'system' ? (
          <SystemTab
            config={config}
            siteName={siteName}
            url={url}
            publicBaseUrl={publicBaseUrl}
            timeout={timeout}
            debugEnabled={debugEnabled}
            onSiteNameChange={setSiteName}
            onUrlChange={setUrl}
            onPublicBaseUrlChange={setPublicBaseUrl}
            onTimeoutChange={setTimeoutSec}
            onDebugEnabledChange={setDebugEnabled}
            onSubmit={submit}
          />
        ) : null}

        {activeTab === 'billing' ? (
          <BillingTab
            billingConfig={billingConfig}
            epayEnabled={epayEnabled}
            epayApiUrl={epayApiUrl}
            epayPid={epayPid}
            epayKey={epayKey}
            clearEpayKey={clearEpayKey}
            epayMethods={epayMethods}
            creditPriceCents={creditPriceCents}
            minTopUpCredits={minTopUpCredits}
            referralRewardCredits={referralRewardCredits}
            newUserInitialCredits={newUserInitialCredits}
            dailyFreeCredits={dailyFreeCredits}
            savingBilling={savingBilling}
            onEpayEnabledChange={setEpayEnabled}
            onEpayApiUrlChange={setEpayApiUrl}
            onEpayPidChange={setEpayPid}
            onEpayKeyChange={setEpayKey}
            onClearEpayKeyChange={setClearEpayKey}
            onCreditPriceCentsChange={setCreditPriceCents}
            onMinTopUpCreditsChange={setMinTopUpCredits}
            onReferralRewardCreditsChange={setReferralRewardCredits}
            onNewUserInitialCreditsChange={setNewUserInitialCredits}
            onDailyFreeCreditsChange={setDailyFreeCredits}
            onToggleEpayMethod={toggleEpayMethod}
            onSave={() => void submitBillingConfig()}
          />
        ) : null}

        {activeTab === 'email' ? (
          <EmailTab
            emailConfig={emailConfig}
            smtpEnabled={smtpEnabled}
            smtpHost={smtpHost}
            smtpPort={smtpPort}
            smtpUser={smtpUser}
            smtpPassword={smtpPassword}
            smtpFrom={smtpFrom}
            smtpSecure={smtpSecure}
            clearSmtpPassword={clearSmtpPassword}
            savingEmail={savingEmail}
            onSmtpEnabledChange={setSmtpEnabled}
            onSmtpHostChange={setSmtpHost}
            onSmtpPortChange={setSmtpPort}
            onSmtpUserChange={setSmtpUser}
            onSmtpPasswordChange={setSmtpPassword}
            onSmtpFromChange={setSmtpFrom}
            onSmtpSecureChange={setSmtpSecure}
            onClearSmtpPasswordChange={setClearSmtpPassword}
            onSave={() => void submitEmailConfig()}
          />
        ) : null}

        {activeTab === 'users' ? (
          <UsersTab
            users={users}
            filteredUsers={filteredUsers}
            usersLoading={usersLoading}
            userQuery={userQuery}
            selectedLedgerUser={selectedLedgerUser}
            ledgerLoading={ledgerLoading}
            grantUsername={grantUsername}
            grantAmount={grantAmount}
            grantReason={grantReason}
            grantSubmitting={grantSubmitting}
            roleBusyUser={roleBusyUser}
            onUserQueryChange={setUserQuery}
            onRefreshUsers={() => void loadUsers()}
            onGrantUsernameChange={setGrantUsername}
            onGrantAmountChange={setGrantAmount}
            onGrantReasonChange={setGrantReason}
            onSubmitGrantCredits={() => void submitGrantCredits()}
            onLoadLedger={(username) => void loadLedger(username)}
            onToggleAdminRole={(user) => void toggleAdminRole(user)}
          />
        ) : null}

        {activeTab === 'ledger' ? (
          <LedgerTab
            users={users}
            selectedLedgerUser={selectedLedgerUser}
            ledger={ledger}
            ledgerLoading={ledgerLoading}
            usersLoading={usersLoading}
            onSelectLedgerUser={(username) => void loadLedger(username)}
            onClearLedgerUser={() => {
              setSelectedLedgerUser('')
              setLedger([])
            }}
            onLoadLedger={(username) => void loadLedger(username)}
          />
        ) : null}
      </section>
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
