import { type FormEvent } from 'react'
import type { Mode, ModelProvider, ReferenceUpload } from '../types'
import { QualityPicker } from './QualityPicker'
import { ImageSpecPicker } from './ImageSpecPicker'
import { OutputFormatPicker } from './OutputFormatPicker'
import { UploadPanel } from './UploadPanel'
import { BananaModelPicker } from './BananaModelPicker'
import { BANANA_PROVIDER, providerLabel } from '../lib/models'

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
  onUpload,
  onDeleteUpload,
  onSubmit,
}: Props) {
  const isImageToImage = mode === 'image-to-image'
  const specStep = isImageToImage ? '④' : '③'
  const executeStep = isImageToImage ? '⑤' : '④'

  return (
    <aside className="generation-panel generate-flow">
      <section className="request-status-row">
        <div>
          <strong>当前请求</strong>
          <span>{providerLabel(provider)} · {isImageToImage ? '图生图' : '文生图'}</span>
        </div>
        <button type="button" className={keyReady ? 'key-ready' : 'key-missing'} onClick={onOpenSettings}>
          {keyReady ? `Key ${keyPreview || '已设置'}` : '去设置 Key'}
        </button>
      </section>

      <form onSubmit={onSubmit} className="generation-form composer-form generate-flow-form">
        <section className="generate-step prompt-step">
          <StepTitle index="①" title="提示词" note="先写清楚你想要的画面，不满意可以去助手继续改。" />
          <label className="composer-prompt">
            <span>主提示词</span>
            <textarea value={prompt} onChange={(event) => onPromptChange(event.target.value)} placeholder="描述你想生成的图片，例如：雨夜东京街头，霓虹灯，电影感，写实摄影..." rows={5} />
          </label>
        </section>

        <section className="generate-step model-step">
          <StepTitle index="②" title="模型与模式" note="先选模型分组，再选文生图或图生图。" />
          <div className="generate-control-grid two">
            <div className="mode-tabs provider-tabs" role="tablist" aria-label="模型分组">
              <button type="button" className={provider === 'image-2' ? 'active' : ''} onClick={() => onProviderChange('image-2')}>Image-2</button>
              <button type="button" className={provider === BANANA_PROVIDER ? 'active' : ''} onClick={() => onProviderChange(BANANA_PROVIDER)}>Banana</button>
            </div>
            <div className="mode-tabs" role="tablist" aria-label="生成模式">
              <button type="button" className={mode === 'text-to-image' ? 'active' : ''} onClick={() => onModeChange('text-to-image')}>文生图</button>
              <button type="button" className={mode === 'image-to-image' ? 'active' : ''} onClick={() => onModeChange('image-to-image')}>图生图</button>
            </div>
          </div>
        </section>

        {isImageToImage ? (
          <section className="generate-step reference-step">
            <StepTitle index="③" title="参考图 / 合一方向" note="多张图时可以指定主图，系统会把主图排在请求第一位。" />
            <UploadPanel uploads={uploads} primaryUploadId={primaryUploadId} onPrimaryChange={onPrimaryUploadChange} onUpload={onUpload} onDelete={onDeleteUpload} />
          </section>
        ) : null}

        <section className="generate-step spec-step">
          <StepTitle index={specStep} title={provider === BANANA_PROVIDER ? 'Banana 模型规格' : '图片规格'} note={provider === BANANA_PROVIDER ? 'Banana 的比例和清晰度由模型 ID 决定。' : '设置比例、清晰度、质量和输出格式。'} />
          <div className="generate-control-grid spec-grid">
            {provider === BANANA_PROVIDER ? (
              <BananaModelPicker value={bananaModel} onChange={onBananaModelChange} />
            ) : (
              <>
                <ImageSpecPicker ratio={ratio} resolution={resolution} onRatioChange={onRatioChange} onResolutionChange={onResolutionChange} />
                <QualityPicker value={quality} onChange={onQualityChange} />
                <OutputFormatPicker value={outputFormat} onChange={onOutputFormatChange} />
              </>
            )}
          </div>
        </section>

        <section className="generate-step execute-step">
          <StepTitle index={executeStep} title="数量与执行" note="提交后会自动切到结果页，后端继续跑，前端断开不影响任务。" />
          <div className="execute-grid">
            <label className="composer-mini-field">
              <span>数量</span>
              <input type="number" min={1} max={12} value={count} onChange={(event) => onCountChange(readNumberInput(event.target.value))} />
            </label>
            <label className="composer-mini-field">
              <span>并发</span>
              <input type="number" min={1} value={concurrency} onChange={(event) => onConcurrencyChange(readNumberInput(event.target.value))} />
            </label>
            <div className={`submit-readiness ${keyReady ? 'ready' : 'missing'}`}>
              <strong>{keyReady ? '可以提交' : '需要先设置 Key'}</strong>
              <span>{keyReady ? `${providerLabel(provider)} Key ${keyPreview || '已设置'}` : `当前 ${providerLabel(provider)} 没有可用 Key`}</span>
            </div>
            <button className="primary generate-submit" type="submit">生成</button>
          </div>
          {message ? <div className="ok">{message}</div> : null}
          {error ? <div className="error">{error}</div> : null}
        </section>
      </form>
    </aside>
  )
}

function StepTitle({ index, title, note }: { index: string; title: string; note: string }) {
  return (
    <div className="generate-step-title">
      <b>{index}</b>
      <div>
        <strong>{title}</strong>
        <span>{note}</span>
      </div>
    </div>
  )
}

function readNumberInput(value: string): NumericInputValue {
  return value === '' ? '' : Number(value)
}
