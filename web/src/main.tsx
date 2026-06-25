import { useEffect, useState } from 'react'
import ReactDOM from 'react-dom/client'
import { AdminPage } from './components/AdminPage'
import { GifWorkbenchPage } from './components/GifWorkbenchPage'
import { WorkbenchPage } from './components/WorkbenchPage'
import { nextTheme, resolveTheme, type ThemeMode } from './lib/themes'
import './styles.css'

const THEME_KEY = 'image-workbench-theme'

function readInitialTheme() {
  const saved = window.localStorage.getItem(THEME_KEY)
  return resolveTheme(saved, window.matchMedia?.('(prefers-color-scheme: dark)').matches)
}

const initialTheme = readInitialTheme()
document.documentElement.dataset.theme = initialTheme

function App() {
  const [theme, setTheme] = useState<ThemeMode>(initialTheme)

  useEffect(() => {
    document.documentElement.dataset.theme = theme
    window.localStorage.setItem(THEME_KEY, theme)
  }, [theme])

  const toggleTheme = (next?: ThemeMode) => setTheme((current) => next || nextTheme(current))
  if (window.location.pathname === '/admin') return <AdminPage theme={theme} onToggleTheme={toggleTheme} />
  if (window.location.pathname === '/gif') return <GifWorkbenchPage theme={theme} onToggleTheme={toggleTheme} />
  return <WorkbenchPage theme={theme} onToggleTheme={toggleTheme} />
}

ReactDOM.createRoot(document.getElementById('root')!).render(<App />)
