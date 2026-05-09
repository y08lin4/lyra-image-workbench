import { type FormEvent } from 'react'
import type { Mode, ReferenceUpload } from '../types'
import { getImageSize, normalizeRatioForResolution } from '../lib/ratios'
import { QualityPicker } from './QualityPicker'
import { RatioPicker } from './RatioPicker'
import { ResolutionPicker } from './ResolutionPicker'
import { UploadPanel } from './UploadPanel'

type Props = {
  mode: Mode
  prompt: string
  ratio: string
  resolution: string
  quality: string
  count: number
  concurrency: number
  uploads: ReferenceUpload[]
  keyReady: boolean
  keyPreview: string
  message: string
  error: string
  onModeChange: (mode: Mode) => void
  onPromptChange: (value: string) => void
  onRatioChange: (value: string) => void
  onResolutionChange: (value: string) => void
  onQualityChange: (value: string) => void
  onCountChange: (value: number) => void
  onConcurrencyChange: (value: number) => void
  onOpenSettings: () => void
  onUpload: (files: File[]) => void
  onDeleteUpload: (id: string) => void
  onSubmit: (event: FormEvent) => void
}

export function GenerationPanel({
  mode,
  prompt,
  ratio,
  resolution,
  quality,
  count,
  concurrency,
  uploads,
  keyReady,
  keyPreview,
  message,
  error,
  onModeChange,
  onPromptChange,
  onRatioChange,
  onResolutionChange,
  onQualityChange,
  onCountChange,
  onConcurrencyChange,
  onOpenSettings,
  onUpload,
  onDeleteUpload,
  onSubmit,
}: Props) {
  function changeResolution(next: string) {
    onResolutionChange(next)
    const normalizedRatio = normalizeRatioForResolution(ratio, next)
    if (normalizedRatio !== ratio) onRatioChange(normalizedRatio)
  }

  return (
    <aside className="generation-panel">
      <div className="section-head">
        <p className="eyebrow">Generate</p>
        <h2>生成设置</h2>
        <p className="muted">配置从上到下完成后提交；后端会在本机持续执行任务。</p>
      </div>

      <section className="form-section key-summary">
        <div className="section-title">
          <span>空间 Key</span>
          <small>{keyReady ? '已就绪' : '需要设置'}</small>
        </div>
        <div className="key-row">
          <div className="status-line">当前：{keyReady ? `已设置 ${keyPreview}` : '未设置'}</div>
          <button type="button" onClick={onOpenSettings}>设置</button>
        </div>
      </section>

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
            <small>{getImageSize(normalizeRatioForResolution(ratio, resolution), resolution)}</small>
          </div>
          <div className="field">
            <span>比例</span>
            <RatioPicker value={ratio} resolution={resolution} onChange={onRatioChange} />
          </div>
          <div className="field">
            <span>清晰度</span>
            <ResolutionPicker value={resolution} onChange={changeResolution} />
          </div>
          <div className="field">
            <span>质量</span>
            <QualityPicker value={quality} onChange={onQualityChange} />
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
