import { useEffect, useState } from 'react'
import ReactDOM from 'react-dom/client'
import { AdminPage } from './components/AdminPage'
import { WorkbenchPage } from './components/WorkbenchPage'
import type { ThemeMode } from './components/ThemeToggle'
import './styles.css'

const THEME_KEY = 'image-workbench-theme'

function App() {
  const [theme, setTheme] = useState<ThemeMode>(() => {
    const saved = window.localStorage.getItem(THEME_KEY)
    if (saved === 'dark' || saved === 'light') return saved
    return window.matchMedia?.('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
  })

  useEffect(() => {
    document.documentElement.dataset.theme = theme
    window.localStorage.setItem(THEME_KEY, theme)
  }, [theme])

  const toggleTheme = () => setTheme((current) => current === 'dark' ? 'light' : 'dark')
  return window.location.pathname === '/admin'
    ? <AdminPage theme={theme} onToggleTheme={toggleTheme} />
    : <WorkbenchPage theme={theme} onToggleTheme={toggleTheme} />
}

ReactDOM.createRoot(document.getElementById('root')!).render(<App />)
