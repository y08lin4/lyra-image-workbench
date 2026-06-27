import { type FormEvent, useState } from 'react'
import { ApiError } from '../api/client'
import { loginUser, registerUser } from '../api/users'
import type { UserSession } from '../types'
import { ThemeToggle, type ThemeMode } from './ThemeToggle'
import { GitHubLink } from './GitHubLink'

type Mode = 'login' | 'register'

export function SpaceLogin({ onSession, theme, onToggleTheme }: { onSession: (session: UserSession) => void; theme: ThemeMode; onToggleTheme: () => void }) {
  const initialReferralCode = readReferralCodeFromUrl()
  const [mode, setMode] = useState<Mode>(initialReferralCode ? 'register' : 'login')
  const [identifier, setIdentifier] = useState('')
  const [username, setUsername] = useState('')
  const [email, setEmail] = useState('')
  const [referralCode, setReferralCode] = useState(initialReferralCode)
  const [password, setPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [twoFactorCode, setTwoFactorCode] = useState('')
  const [twoFactorRequired, setTwoFactorRequired] = useState(false)
  const [passwordVisible, setPasswordVisible] = useState(false)
  const [confirmPasswordVisible, setConfirmPasswordVisible] = useState(false)
  const [legacySpacePassword, setLegacySpacePassword] = useState('')
  const [importLegacy, setImportLegacy] = useState(false)
  const [error, setError] = useState('')
  const isRegister = mode === 'register'

  async function submit(event: FormEvent) {
    event.preventDefault()
    setError('')
    if (isRegister && !username.trim()) {
      setError('请输入用户名')
      return
    }
    if (!isRegister && !identifier.trim()) {
      setError('请输入用户名或邮箱')
      return
    }
    if (isRegister && !email.trim()) {
      setError('请输入邮箱')
      return
    }
    if (isRegister && password !== confirmPassword) {
      setError('两次输入的密码不一致')
      return
    }
    try {
      const session = isRegister
        ? await registerUser({
          username: username.trim(),
          email: email.trim(),
          password,
          referralCode: referralCode.trim(),
          legacySpacePassword: importLegacy ? legacySpacePassword : '',
        })
        : await loginUser(identifier.trim(), password, twoFactorCode)
      onSession(session)
    } catch (err) {
      if (err instanceof ApiError && (err.code === 'USER_TOTP_REQUIRED' || err.code === 'USER_TOTP_INVALID')) {
        setTwoFactorRequired(true)
        setError(err.code === 'USER_TOTP_INVALID' ? '2FA 验证码无效或已过期' : '请输入 2FA 验证码')
        return
      }
      setError(err instanceof Error ? err.message : (isRegister ? '注册失败' : '登录失败'))
    }
  }

  return (
    <main className="center-shell">
      <div className="center-theme-action">
        <GitHubLink compact />
        <ThemeToggle theme={theme} onToggle={onToggleTheme} />
      </div>
      <form className="login-panel" onSubmit={submit}>
        <div className="brand login-brand">
          <div className="brand-mark">Ly</div>
          <div>
            <p className="eyebrow">用户账号</p>
            <h1>Lyra Image Workbench</h1>
          </div>
        </div>
        <h2>{isRegister ? '注册账号' : '登录账号'}</h2>
        <p className="muted">同一个账号在不同设备登录后，会同步任务历史、提示词历史、参考图和输出图。</p>
        <div className="identity-help">
          <strong>Key 与历史说明</strong>
          <ul>
            <li>用户名和密码用于进入同一个服务器账号空间。</li>
            <li>历史记录保存在服务器账号空间中，多设备登录可同步查看。</li>
            <li>API Key 默认只保存在当前浏览器本地，也可以在设置页确认风险后上传到云端。</li>
            <li>新设备登录后可重新填写本地 Key，或使用已授权保存的云端 Key。</li>
          </ul>
        </div>
        {isRegister ? (
          <>
            <input value={username} onChange={(e) => setUsername(e.target.value)} placeholder="用户名，大小写字母/数字/._-" autoComplete="username" autoFocus />
            <input type="email" value={email} onChange={(e) => setEmail(e.target.value)} placeholder="邮箱，用于账号和通知" autoComplete="email" />
            <input value={referralCode} onChange={(e) => setReferralCode(e.target.value)} placeholder="邀请码，可留空" autoComplete="off" />
          </>
        ) : (
          <input value={identifier} onChange={(e) => setIdentifier(e.target.value)} placeholder="用户名或邮箱" autoComplete="username" autoFocus />
        )}
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
        {!isRegister && twoFactorRequired ? (
          <input
            inputMode="numeric"
            value={twoFactorCode}
            onChange={(e) => setTwoFactorCode(e.target.value)}
            placeholder="2FA 验证码"
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
        <button type="button" onClick={() => { setMode(isRegister ? 'login' : 'register'); setError(''); setConfirmPassword(''); setTwoFactorCode(''); setTwoFactorRequired(false) }}>
          {isRegister ? '已有账号，去登录' : '没有账号，去注册'}
        </button>
        {error ? <div className="error">{error}</div> : null}
      </form>
    </main>
  )
}

function readReferralCodeFromUrl() {
  const params = new URLSearchParams(window.location.search)
  return (params.get('ref') || params.get('referralCode') || params.get('invite') || '').trim()
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
