import { useEffect, useState } from 'react'
import ReactDOM from 'react-dom/client'
import { AdminPage } from './components/AdminPage'
import { ApiDocsPage } from './components/ApiDocsPage'
import { WorkbenchPage } from './components/WorkbenchPage'
import { ThemeToggle } from './components/ThemeToggle'
import { nextTheme, resolveTheme, type ThemeMode } from './lib/themes'
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
  const [theme, setTheme] = useState<ThemeMode>(initialTheme)

  useEffect(() => {
    document.documentElement.dataset.theme = theme
    window.localStorage.setItem(THEME_KEY, theme)
  }, [theme])

  const toggleTheme = (next?: ThemeMode) => setTheme((current) => next || nextTheme(current))
  if (window.location.pathname === '/admin') return <AdminPage theme={theme} onToggleTheme={toggleTheme} />
  if (window.location.pathname === '/api-docs') return <PublicApiDocsPage theme={theme} onToggleTheme={toggleTheme} />
  return <WorkbenchPage theme={theme} onToggleTheme={toggleTheme} />
}

ReactDOM.createRoot(document.getElementById('root')!).render(<App />)
