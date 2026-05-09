import type { Task, TaskResult } from '../types'
import { formatBytes } from '../lib/format'

export function ResultCanvas({ task }: { task?: Task }) {
  const okCount = task?.results.filter((item) => item.ok).length || 0
  return (
    <section className="result-canvas">
      <header className="canvas-header">
        <div>
          <p className="eyebrow">Canvas</p>
          <h2>生成结果</h2>
          {task ? <p>{task.stageText} / {task.stage} / {task.stageCode} · {task.progress}% · {okCount}/{task.count}</p> : <p>选择或创建一个任务后查看结果。</p>}
        </div>
        {task ? <span className={`status-pill ${task.status}`}>{task.statusText} / {task.statusCode}</span> : null}
      </header>

      {!task ? (
        <div className="empty-state">
          <strong>还没有选择任务</strong>
          <span>提交任务后，生成结果会出现在这里。后端继续执行，刷新页面也可以恢复。</span>
        </div>
      ) : (
        <div className="result-grid">
          {Array.from({ length: task.count }, (_, index) => {
            const result = task.results.find((item) => item.index === index)
            return <ResultCard key={index} index={index} result={result} />
          })}
        </div>
      )}
    </section>
  )
}

function ResultCard({ index, result }: { index: number; result?: TaskResult }) {
  if (!result) {
    return (
      <article className="result-card pending">
        <div className="result-pending">等待生成 #{index + 1}</div>
        <footer>排队中 / queued / J100</footer>
      </article>
    )
  }
  return (
    <article className="result-card">
      {result.ok && result.imageUrl ? <img src={result.imageUrl} alt={`result-${index + 1}`} /> : <div className="result-error">{result.error || result.statusText}</div>}
      <footer>{result.statusText} / {result.status} / {result.statusCode} · {formatBytes(result.bytes)}</footer>
    </article>
  )
}
