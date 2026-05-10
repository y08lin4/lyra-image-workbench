import { type FormEvent } from 'react'
import type { Mode, ModelProvider, ReferenceUpload } from '../types'
import { QualityPicker } from './QualityPicker'
import { ImageSpecPicker } from './ImageSpecPicker'
import { OutputFormatPicker } from './OutputFormatPicker'
import { UploadPanel } from './UploadPanel'
import { BananaModelPicker } from './BananaModelPicker'

type NumericInputValue = number | ''

type Props = {
  mode: Mode
  provider: ModelProvider
  prompt: string
  ratio: string
  resolution: string
  quality: string
  outputFormat: string
  bananaModel: string
  count: NumericInputValue
  concurrency: NumericInputValue
  uploads: ReferenceUpload[]
  primaryUploadId: string
  keyReady: boolean
  keyPreview: string
  message: string
  error: string
  onModeChange: (mode: Mode) => void
  onProviderChange: (provider: ModelProvider) => void
  onPromptChange: (value: string) => void
  onRatioChange: (value: string) => void
  onResolutionChange: (value: string) => void
  onQualityChange: (value: string) => void
  onOutputFormatChange: (value: string) => void
  onBananaModelChange: (value: string) => void
  onCountChange: (value: NumericInputValue) => void
  onConcurrencyChange: (value: NumericInputValue) => void
  onPrimaryUploadChange: (id: string) => void
  onOpenSettings: () => void
  onOpenPromptAssistant: () => void
  onUpload: (files: File[]) => void
  onDeleteUpload: (id: string) => void
  onSubmit: (event: FormEvent) => void
}

export function GenerationPanel({
  mode,
  provider,
  prompt,
  ratio,
  resolution,
  quality,
  outputFormat,
  bananaModel,
  count,
  concurrency,
  uploads,
  primaryUploadId,
  keyReady,
  keyPreview,
  message,
  error,
  onModeChange,
  onProviderChange,
  onPromptChange,
  onRatioChange,
  onResolutionChange,
  onQualityChange,
  onOutputFormatChange,
  onBananaModelChange,
  onCountChange,
  onConcurrencyChange,
  onPrimaryUploadChange,
  onOpenSettings,
  onOpenPromptAssistant,
  onUpload,
  onDeleteUpload,
  onSubmit,
}: Props) {
  return (
    <aside className="generation-panel">
      <section className="request-status-row">
        <div>
          <strong>请求</strong>
          <span>{provider === 'banana' ? 'Banana Nano' : 'Image-2'} · {mode === 'image-to-image' ? '图生图' : '文生图'}</span>
        </div>
        <button type="button" className={keyReady ? 'key-ready' : 'key-missing'} onClick={onOpenSettings}>
          {keyReady ? `Key ${keyPreview || '已设置'}` : '设置 Key'}
        </button>
      </section>

      <form onSubmit={onSubmit} className="generation-form composer-form">
        {mode === 'image-to-image' ? <UploadPanel uploads={uploads} primaryUploadId={primaryUploadId} onPrimaryChange={onPrimaryUploadChange} onUpload={onUpload} onDelete={onDeleteUpload} /> : null}

        <div className="composer-primary-row">
          <label className="composer-prompt">
            <span>提示词</span>
            <textarea value={prompt} onChange={(event) => onPromptChange(event.target.value)} placeholder="描述你想生成的图片..." rows={2} />
          </label>
          <button className="primary generate-submit" type="submit">生成</button>
        </div>

        <div className="composer-control-row">
          <div className="composer-control-left">
            <div className="mode-tabs provider-tabs" role="tablist" aria-label="模型分组">
              <button type="button" className={provider === 'image-2' ? 'active' : ''} onClick={() => onProviderChange('image-2')}>Image-2</button>
              <button type="button" className={provider === 'banana' ? 'active' : ''} onClick={() => onProviderChange('banana')}>Banana</button>
            </div>
            <div className="mode-tabs" role="tablist" aria-label="生成模式">
              <button type="button" className={mode === 'text-to-image' ? 'active' : ''} onClick={() => onModeChange('text-to-image')}>文生图</button>
              <button type="button" className={mode === 'image-to-image' ? 'active' : ''} onClick={() => onModeChange('image-to-image')}>图生图</button>
            </div>
            {provider === 'banana' ? (
              <BananaModelPicker value={bananaModel} onChange={onBananaModelChange} />
            ) : (
              <>
                <ImageSpecPicker ratio={ratio} resolution={resolution} onRatioChange={onRatioChange} onResolutionChange={onResolutionChange} />
                <QualityPicker value={quality} onChange={onQualityChange} />
                <OutputFormatPicker value={outputFormat} onChange={onOutputFormatChange} />
              </>
            )}
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
