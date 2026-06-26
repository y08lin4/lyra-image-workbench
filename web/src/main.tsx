import { useEffect, useState } from 'react'
import ReactDOM from 'react-dom/client'
import { AdminPage } from './components/AdminPage'
import { ApiDocsPage } from './components/ApiDocsPage'
import { WorkbenchPage } from './components/WorkbenchPage'
import { ThemeToggle } from './components/ThemeToggle'
import { nextTheme, resolveTheme, type ThemeMode } from './lib/themes'
import { getAdminAuthStatus } from './api/admin'
import './styles.css'

const THEME_KEY = 'image-workbench-theme'

function readInitialTheme() {
  const saved = window.localStorage.getItem(THEME_KEY)
  return resolveTheme(saved, window.matchMedia?.('(prefers-color-scheme: dark)').matches)
}

const initialTheme = readInitialTheme()
document.documentElement.dataset.theme = initialTheme

function PublicApiDocsPage({ theme, onToggleTheme }: { theme: ThemeMode; onToggleTheme: (next?: ThemeMode) => void }) {
  return (
    <div className="app-shell gallery-shell api-docs-public-shell">
      <header className="topbar workbench-topbar">
        <div className="brand">
          <div className="brand-mark">API</div>
          <div>
            <h1>LyAi Image API</h1>
            <p>公开调用文档</p>
          </div>
        </div>
        <nav className="top-actions">
          <a className="ghost-link" href="/">进入工作台</a>
          <ThemeToggle theme={theme} onToggle={onToggleTheme} />
        </nav>
      </header>
      <ApiDocsPage />
    </div>
  )
}
function App() {
  const initialPath = window.location.pathname
  const [theme, setTheme] = useState<ThemeMode>(initialTheme)
  const [setupChecked, setSetupChecked] = useState(initialPath === '/admin' || initialPath === '/api-docs')
  const [forceAdminSetup, setForceAdminSetup] = useState(initialPath === '/admin')

  useEffect(() => {
    document.documentElement.dataset.theme = theme
    window.localStorage.setItem(THEME_KEY, theme)
  }, [theme])

  useEffect(() => {
    const path = window.location.pathname
    if (path === '/admin' || path === '/api-docs') {
      setSetupChecked(true)
      return
    }
    let alive = true
    getAdminAuthStatus()
      .then((status) => {
        if (!alive) return
        if (status.setupRequired || status.initialized === false) {
          window.history.replaceState(null, '', '/admin')
          setForceAdminSetup(true)
        }
      })
      .catch(() => undefined)
      .finally(() => {
        if (alive) setSetupChecked(true)
      })
    return () => {
      alive = false
    }
  }, [])

  const toggleTheme = (next?: ThemeMode) => setTheme((current) => next || nextTheme(current))
  if (!setupChecked) {
    return (
      <main className="center-shell">
        <section className="admin-panel">
          <div className="brand login-brand">
            <div className="brand-mark">Ly</div>
            <div>
              <p className="eyebrow">Setup</p>
              <h1>正在进入初始化设置</h1>
            </div>
          </div>
          <div className="info">正在检查站点初始化状态...</div>
        </section>
      </main>
    )
  }
  if (forceAdminSetup || window.location.pathname === '/admin') return <AdminPage theme={theme} onToggleTheme={toggleTheme} />
  if (window.location.pathname === '/api-docs') return <PublicApiDocsPage theme={theme} onToggleTheme={toggleTheme} />
  return <WorkbenchPage theme={theme} onToggleTheme={toggleTheme} />
}
ReactDOM.createRoot(document.getElementById('root')!).render(<App />)
