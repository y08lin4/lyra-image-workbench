import React from 'react'
import ReactDOM from 'react-dom/client'
import './styles.css'

function App() {
  return (
    <main className="app-shell">
      <section className="hero-card">
        <p className="eyebrow">Localhost Image Workbench</p>
        <h1>Go 后端 + React 前端</h1>
        <p>
          前端只负责提交任务和观察 SSE 进度；真正的 NewAPI 请求会由本机 Go 后端任务队列执行，页面刷新不会中断生图。
        </p>
      </section>
    </main>
  )
}

ReactDOM.createRoot(document.getElementById('root')!).render(<App />)
