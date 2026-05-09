import { type FormEvent, useEffect, useState } from 'react'
import {
  clearAdminToken,
  getAdminAuthStatus,
  getAdminConfig,
  getAdminToken,
  loginAdmin,
  logoutAdmin,
  saveAdminConfig,
  setupAdminPassword,
} from '../api/admin'
import type { AdminAuthStatus, AdminConfig } from '../types'

type AdminMode = 'loading' | 'setup' | 'login' | 'config'
type NumericInputValue = number | ''

export function AdminPage() {
  const [mode, setMode] = useState<AdminMode>('loading')
  const [auth, setAuth] = useState<AdminAuthStatus | null>(null)
  const [config, setConfig] = useState<AdminConfig | null>(null)
  const [url, setUrl] = useState('')
  const [timeout, setTimeoutSec] = useState<NumericInputValue>(600)
  const [password, setPassword] = useState('')
  const [message, setMessage] = useState('')
  const [error, setError] = useState('')

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
      setTimeoutSec(cfg.timeoutSec)
      setMode('config')
    } catch (err) {
      clearAdminToken()
      setError(err instanceof Error ? err.message : 'Admin 登录已失效')
      setMode('login')
    }
  }

  async function submitPassword(event: FormEvent) {
    event.preventDefault()
    setError('')
    setMessage('')
    try {
      if (mode === 'setup') {
        const next = await setupAdminPassword(password)
        setAuth(next.auth)
        setMessage('Admin 管理密码已设置')
      } else {
        const next = await loginAdmin(password)
        setAuth(next.auth)
        setMessage('Admin 登录成功')
      }
      setPassword('')
      await loadConfig()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Admin 鉴权失败')
    }
  }

  async function submit(event: FormEvent) {
    event.preventDefault()
    setError('')
    try {
      const cfg = await saveAdminConfig(url, numericOrDefault(timeout, config?.timeoutSec || 600))
      setConfig(cfg)
      setMessage('管理配置已保存')
    } catch (err) {
      setError(err instanceof Error ? err.message : '保存失败')
    }
  }

  async function handleLogout() {
    try {
      await logoutAdmin()
    } finally {
      setConfig(null)
      setMode('login')
      setMessage('已退出 Admin')
    }
  }

  if (mode === 'loading') {
    return (
      <main className="center-shell">
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
      <form className="admin-panel" onSubmit={submit}>
        <AdminBrand title="后台管理" />
        <div className="status-line">Admin 已登录 · 密码状态：{auth?.passwordSet ? '已设置' : '未设置'}</div>
        <label>NewAPI 请求 URL<input value={url} onChange={(e) => setUrl(e.target.value)} placeholder="http://127.0.0.1:3000/v1" /></label>
        <label>超时时间（秒）<input type="number" min={config?.limits.minTimeoutSec || 60} max={config?.limits.maxTimeoutSec || 3600} value={timeout} onChange={(e) => setTimeoutSec(readNumberInput(e.target.value))} /></label>
        <div className="status-line">模型：{config?.model || 'gpt-image-2'} {config?.modelLocked ? '（首版固定）' : ''}</div>
        <button className="primary" type="submit">保存管理配置</button>
        <div className="admin-actions"><a href="/">返回工作台</a><button type="button" onClick={handleLogout}>退出 Admin</button></div>
        {message ? <div className="ok">{message}</div> : null}
        {error ? <div className="error">{error}</div> : null}
      </form>
    </main>
  )
}

function AdminBrand({ title }: { title: string }) {
  return (
    <div className="brand login-brand">
      <div className="brand-mark">AI</div>
      <div>
        <p className="eyebrow">Admin</p>
        <h1>{title}</h1>
      </div>
    </div>
  )
}

function readNumberInput(value: string): NumericInputValue {
  return value === '' ? '' : Number(value)
}

function numericOrDefault(value: NumericInputValue, fallback: number) {
  return value === '' ? fallback : value
}
