import { type FormEvent, useState } from 'react'
import { openSpace } from '../api/spaces'
import type { SpaceSession } from '../types'
import { ThemeToggle, type ThemeMode } from './ThemeToggle'

export function SpaceLogin({ onSession, theme, onToggleTheme }: { onSession: (session: SpaceSession) => void; theme: ThemeMode; onToggleTheme: () => void }) {
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  async function submit(event: FormEvent) {
    event.preventDefault()
    setError('')
    try {
      onSession(await openSpace(password))
    } catch (err) {
      setError(err instanceof Error ? err.message : '进入空间失败')
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
            <p className="eyebrow">个人空间</p>
            <h1>LyAI生图工作台</h1>
          </div>
        </div>
        <h2>进入个人空间</h2>
        <p className="muted">请输入你自己设置的空间密码后进入工作台。</p>
        <div className="identity-help">
          <strong>空间密码说明</strong>
          <ul>
            <li>这个密码由你自行设置，不需要注册，也没有默认密码。</li>
            <li>建议使用复杂密码，至少 10 位，最好包含大小写字母、数字和符号。</li>
            <li>在这台电脑输入完全相同的密码，会进入同一个本机任务空间。</li>
            <li>输入不同密码，会进入不同空间，任务互相隔离。</li>
            <li>浏览器和后端只保存不可逆算法处理后的结果，不保存明文密码。</li>
            <li>请自己保存好这个密码；忘记后无法找回原空间。</li>
          </ul>
        </div>
        <input type="password" value={password} onChange={(e) => setPassword(e.target.value)} placeholder="自行设置复杂密码，至少 10 位" autoFocus />
        <small className="muted">只有输入完全相同的空间密码，才会进入同一个本机任务空间。</small>
        <button className="primary" type="submit">进入这个空间</button>
        {error ? <div className="error">{error}</div> : null}
      </form>
    </main>
  )
}
