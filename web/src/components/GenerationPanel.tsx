import { type FormEvent } from 'react'
import type { Mode, ReferenceUpload } from '../types'
import { SettingsPanel } from './SettingsPanel'
import { UploadPanel } from './UploadPanel'

type Props = {
  mode: Mode
  prompt: string
  ratio: string
  resolution: string
  count: number
  concurrency: number
  uploads: ReferenceUpload[]
  message: string
  error: string
  onModeChange: (mode: Mode) => void
  onPromptChange: (value: string) => void
  onRatioChange: (value: string) => void
  onResolutionChange: (value: string) => void
  onCountChange: (value: number) => void
  onConcurrencyChange: (value: number) => void
  onKeyReady: (ready: boolean) => void
  onUpload: (files: File[]) => void
  onDeleteUpload: (id: string) => void
  onSubmit: (event: FormEvent) => void
}

export function GenerationPanel({
  mode,
  prompt,
  ratio,
  resolution,
  count,
  concurrency,
  uploads,
  message,
  error,
  onModeChange,
  onPromptChange,
  onRatioChange,
  onResolutionChange,
  onCountChange,
  onConcurrencyChange,
  onKeyReady,
  onUpload,
  onDeleteUpload,
  onSubmit,
}: Props) {
  return (
    <aside className="generation-panel">
      <div className="section-head">
        <p className="eyebrow">Generate</p>
        <h2>生成设置</h2>
        <p className="muted">配置从上到下完成后提交；后端会在本机持续执行任务。</p>
      </div>

      <SettingsPanel onReady={onKeyReady} />

      <form onSubmit={onSubmit} className="generation-form">
        <section className="form-section">
          <div className="section-title">
            <span>模式</span>
            <small>选择输入方式</small>
          </div>
          <div className="mode-tabs" role="tablist" aria-label="生成模式">
            <button type="button" className={mode === 'text-to-image' ? 'active' : ''} onClick={() => onModeChange('text-to-image')}>文生图</button>
            <button type="button" className={mode === 'image-to-image' ? 'active' : ''} onClick={() => onModeChange('image-to-image')}>图生图</button>
          </div>
        </section>

        <section className="form-section">
          <label className="field">
            <span>提示词</span>
            <textarea value={prompt} onChange={(event) => onPromptChange(event.target.value)} placeholder="描述你想生成的画面" rows={6} />
          </label>
        </section>

        <section className="form-section">
          <div className="section-title">
            <span>图片规格</span>
            <small>比例和清晰度</small>
          </div>
          <div className="grid-2">
            <label className="field">
              <span>比例</span>
              <select value={ratio} onChange={(event) => onRatioChange(event.target.value)}>
                <option value="1:1">1:1</option>
                <option value="16:9">16:9</option>
                <option value="9:16">9:16</option>
                <option value="3:4">3:4</option>
                <option value="4:3">4:3</option>
                <option value="3:2">3:2</option>
                <option value="2:3">2:3</option>
                <option value="auto">auto</option>
              </select>
            </label>
            <label className="field">
              <span>清晰度</span>
              <select value={resolution} onChange={(event) => onResolutionChange(event.target.value)}>
                <option value="standard">标准</option>
                <option value="2k">2K</option>
                <option value="4k">4K</option>
                <option value="auto">自动</option>
              </select>
            </label>
          </div>
          <div className="grid-2">
            <label className="field">
              <span>数量</span>
              <input type="number" min={1} max={12} value={count} onChange={(event) => onCountChange(Number(event.target.value))} />
            </label>
            <label className="field">
              <span>并发</span>
              <input type="number" min={1} max={4} value={concurrency} onChange={(event) => onConcurrencyChange(Number(event.target.value))} />
            </label>
          </div>
        </section>

        {mode === 'image-to-image' ? <UploadPanel uploads={uploads} onUpload={onUpload} onDelete={onDeleteUpload} /> : null}

        <button className="primary generate-submit" type="submit">开始生成</button>
      </form>

      {message ? <div className="ok">{message}</div> : null}
      {error ? <div className="error">{error}</div> : null}
    </aside>
  )
}
