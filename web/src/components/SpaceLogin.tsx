import { type FormEvent, useState } from 'react'
import { loginUser, registerUser } from '../api/users'
import type { UserSession } from '../types'
import { ThemeToggle, type ThemeMode } from './ThemeToggle'

type Mode = 'login' | 'register'

export function SpaceLogin({ onSession, theme, onToggleTheme }: { onSession: (session: UserSession) => void; theme: ThemeMode; onToggleTheme: () => void }) {
  const [mode, setMode] = useState<Mode>('login')
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [passwordVisible, setPasswordVisible] = useState(false)
  const [confirmPasswordVisible, setConfirmPasswordVisible] = useState(false)
  const [legacySpacePassword, setLegacySpacePassword] = useState('')
  const [importLegacy, setImportLegacy] = useState(false)
  const [error, setError] = useState('')
  const isRegister = mode === 'register'

  async function submit(event: FormEvent) {
    event.preventDefault()
    setError('')
    if (isRegister && password !== confirmPassword) {
      setError('两次输入的密码不一致')
      return
    }
    try {
      const session = isRegister
        ? await registerUser(username, password, importLegacy ? legacySpacePassword : '')
        : await loginUser(username, password)
      onSession(session)
    } catch (err) {
      setError(err instanceof Error ? err.message : (isRegister ? '注册失败' : '登录失败'))
    }
  }

  return (
    <main className="center-shell">
      <div className="center-theme-action">
        <ThemeToggle theme={theme} onToggle={onToggleTheme} />
      </div>
      <form className="login-panel" onSubmit={submit}>
        <div className="brand login-brand">
          <div className="brand-mark">Ly</div>
          <div>
            <p className="eyebrow">用户账号</p>
            <h1>LyAI生图工作台</h1>
          </div>
        </div>
        <h2>{isRegister ? '注册账号' : '登录账号'}</h2>
        <p className="muted">同一个账号在不同设备登录后，会同步任务历史、提示词历史、参考图和输出图。</p>
        <div className="identity-help">
          <strong>Key 与历史说明</strong>
          <ul>
            <li>用户名和密码用于进入同一个服务器账号空间。</li>
            <li>历史记录保存在服务器账号空间中，多设备登录可同步查看。</li>
            <li>API Key 只保存在当前浏览器本地，不会随账号同步。</li>
            <li>新设备登录后需要在设置页重新填写 Key 才能生成或重试任务。</li>
          </ul>
        </div>
        <input value={username} onChange={(e) => setUsername(e.target.value)} placeholder="用户名，小写字母/数字/._-" autoFocus />
        <PasswordField
          value={password}
          onChange={setPassword}
          visible={passwordVisible}
          onToggle={() => setPasswordVisible((current) => !current)}
          placeholder="复杂密码，至少 10 位"
        />
        {isRegister ? (
          <PasswordField
            value={confirmPassword}
            onChange={setConfirmPassword}
            visible={confirmPasswordVisible}
            onToggle={() => setConfirmPasswordVisible((current) => !current)}
            placeholder="再次输入密码"
          />
        ) : null}
        {isRegister ? (
          <label className="check-row">
            <input type="checkbox" checked={importLegacy} onChange={(e) => setImportLegacy(e.target.checked)} />
            <span>导入旧空间历史</span>
          </label>
        ) : null}
        {isRegister && importLegacy ? (
          <input type="password" value={legacySpacePassword} onChange={(e) => setLegacySpacePassword(e.target.value)} placeholder="旧空间密码" />
        ) : null}
        <button className="primary" type="submit">{isRegister ? '注册并进入' : '登录'}</button>
        <button type="button" onClick={() => { setMode(isRegister ? 'login' : 'register'); setError(''); setConfirmPassword('') }}>
          {isRegister ? '已有账号，去登录' : '没有账号，去注册'}
        </button>
        {error ? <div className="error">{error}</div> : null}
      </form>
    </main>
  )
}

function PasswordField({ value, onChange, visible, onToggle, placeholder }: { value: string; onChange: (value: string) => void; visible: boolean; onToggle: () => void; placeholder: string }) {
  return (
    <div className="password-field">
      <input
        type={visible ? 'text' : 'password'}
        value={value}
        onChange={(event) => onChange(event.target.value)}
        placeholder={placeholder}
      />
      <button type="button" className="password-toggle" onClick={onToggle} aria-label={visible ? '隐藏密码' : '显示密码'} title={visible ? '隐藏密码' : '显示密码'}>
        {visible ? <EyeOffIcon /> : <EyeIcon />}
      </button>
    </div>
  )
}

function EyeIcon() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <path d="M2.25 12s3.5-6.25 9.75-6.25S21.75 12 21.75 12s-3.5 6.25-9.75 6.25S2.25 12 2.25 12Z" />
      <circle cx="12" cy="12" r="2.75" />
    </svg>
  )
}

function EyeOffIcon() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <path d="m3 3 18 18" />
      <path d="M10.6 5.9c.45-.1.92-.15 1.4-.15 6.25 0 9.75 6.25 9.75 6.25a17.9 17.9 0 0 1-2.9 3.55" />
      <path d="M6.45 7.15A18.7 18.7 0 0 0 2.25 12s3.5 6.25 9.75 6.25c1.55 0 2.92-.38 4.1-.95" />
      <path d="M9.9 9.9a2.75 2.75 0 0 0 3.9 3.9" />
    </svg>
  )
}
