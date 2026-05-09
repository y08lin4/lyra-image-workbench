import { type FormEvent } from 'react'
import type { Mode, ReferenceUpload } from '../types'
import { QualityPicker } from './QualityPicker'
import { ImageSpecPicker } from './ImageSpecPicker'
import { OutputFormatPicker } from './OutputFormatPicker'
import { UploadPanel } from './UploadPanel'

type NumericInputValue = number | ''

type Props = {
  mode: Mode
  prompt: string
  ratio: string
  resolution: string
  quality: string
  outputFormat: string
  count: NumericInputValue
  concurrency: NumericInputValue
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
  onOutputFormatChange: (value: string) => void
  onCountChange: (value: NumericInputValue) => void
  onConcurrencyChange: (value: NumericInputValue) => void
  onOpenSettings: () => void
  onOpenPromptAssistant: () => void
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
  outputFormat,
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
  onOutputFormatChange,
  onCountChange,
  onConcurrencyChange,
  onOpenSettings,
  onOpenPromptAssistant,
  onUpload,
  onDeleteUpload,
  onSubmit,
}: Props) {
  return (
    <aside className="generation-panel">
      <section className="form-section key-summary">
        <div className="section-title">
          <span>Image-2 Key</span>
          <small>{keyReady ? '已就绪' : '需要设置'}</small>
        </div>
        <div className="key-row">
          <div className="status-line">当前：{keyReady ? `已设置 ${keyPreview}` : '未设置'}</div>
          <button type="button" onClick={onOpenSettings}>设置</button>
        </div>
      </section>

      <form onSubmit={onSubmit} className="generation-form composer-form">
        {mode === 'image-to-image' ? <UploadPanel uploads={uploads} onUpload={onUpload} onDelete={onDeleteUpload} /> : null}

        <label className="composer-prompt">
          <textarea value={prompt} onChange={(event) => onPromptChange(event.target.value)} placeholder="描述你想生成的图片..." rows={2} />
        </label>

        <div className="composer-control-row">
          <div className="composer-control-left">
            <div className="mode-tabs" role="tablist" aria-label="生成模式">
              <button type="button" className={mode === 'text-to-image' ? 'active' : ''} onClick={() => onModeChange('text-to-image')}>文生图</button>
              <button type="button" className={mode === 'image-to-image' ? 'active' : ''} onClick={() => onModeChange('image-to-image')}>图生图</button>
            </div>
            <ImageSpecPicker ratio={ratio} resolution={resolution} onRatioChange={onRatioChange} onResolutionChange={onResolutionChange} />
            <QualityPicker value={quality} onChange={onQualityChange} />
            <OutputFormatPicker value={outputFormat} onChange={onOutputFormatChange} />
            <button type="button" className="prompt-assistant-trigger" onClick={onOpenPromptAssistant}>提示词助手</button>
            <label className="composer-mini-field">
              <span>数量</span>
              <input type="number" min={1} max={12} value={count} onChange={(event) => onCountChange(readNumberInput(event.target.value))} />
            </label>
            <label className="composer-mini-field">
              <span>并发</span>
              <input type="number" min={1} value={concurrency} onChange={(event) => onConcurrencyChange(readNumberInput(event.target.value))} />
            </label>
          </div>

          <button className="primary generate-submit" type="submit">生成</button>
        </div>
      </form>

      {message ? <div className="ok">{message}</div> : null}
      {error ? <div className="error">{error}</div> : null}
    </aside>
  )
}

function readNumberInput(value: string): NumericInputValue {
  return value === '' ? '' : Number(value)
}
