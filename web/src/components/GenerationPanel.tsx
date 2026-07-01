import { type FormEvent } from 'react'
import type { Mode, ModelProvider, ReferenceUpload } from '../types'
import { QualityPicker } from './QualityPicker'
import { ImageSpecPicker } from './ImageSpecPicker'
import { OutputFormatPicker } from './OutputFormatPicker'
import { UploadPanel } from './UploadPanel'
import { IMAGE2_MODEL_OPTIONS, getImage2ModelOption, image2ModelAllowsRatio, providerLabel } from '../lib/models'

type NumericInputValue = number | ''

type Props = {
  mode: Mode
  provider: ModelProvider
  prompt: string
  ratio: string
  resolution: string
  size: string
  quality: string
  outputFormat: string
  imageModel: string
  count: NumericInputValue
  concurrency: NumericInputValue
  uploads: ReferenceUpload[]
  keyReady: boolean
  keyPreview: string
  message: string
  error: string
  onModeChange: (mode: Mode) => void
  onImageModelChange: (model: string) => void
  onPromptChange: (value: string) => void
  onRatioChange: (value: string) => void
  onResolutionChange: (value: string) => void
  onSizeChange: (value: string) => void
  onQualityChange: (value: string) => void
  onOutputFormatChange: (value: string) => void
  onCountChange: (value: NumericInputValue) => void
  onConcurrencyChange: (value: NumericInputValue) => void
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
  size,
  quality,
  outputFormat,
  imageModel,
  count,
  concurrency,
  uploads,
  keyReady,
  keyPreview,
  message,
  error,
  onModeChange,
  onImageModelChange,
  onPromptChange,
  onRatioChange,
  onResolutionChange,
  onSizeChange,
  onQualityChange,
  onOutputFormatChange,
  onCountChange,
  onConcurrencyChange,
  onOpenSettings,
  onUpload,
  onDeleteUpload,
  onSubmit,
}: Props) {
  const isImageToImage = mode === 'image-to-image'
  const imageModelOption = getImage2ModelOption(imageModel)
  const ratioSelectable = image2ModelAllowsRatio(imageModel)
  const specTitleNote = ratioSelectable ? '设置比例、清晰度、质量和输出格式。' : '当前模型自动决定画幅；质量和输出格式仍可设置。'
  const specStep = isImageToImage ? '④' : '③'
  const executeStep = isImageToImage ? '⑤' : '④'

  return (
    <aside className="generation-panel generate-flow">
      <section className="request-status-row">
        <div>
          <strong>当前请求</strong>
          <span>{providerLabel(provider)} · {isImageToImage ? '参考图生成' : '文字生成'}</span>
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
          <StepTitle index="②" title="模型与模式" note="选择生图模型和生成方式。" />
          <div className="generate-control-grid">
            <div className="mode-tabs provider-tabs" role="radiogroup" aria-label="生图模型">
              {IMAGE2_MODEL_OPTIONS.map((item) => (
                <button key={item.id} type="button" className={imageModelOption.id === item.id ? 'active' : ''} onClick={() => onImageModelChange(item.id)}>
                  {item.label}
                </button>
              ))}
            </div>
            <small className="muted">{imageModelOption.hint}</small>
            <div className="mode-tabs" role="tablist" aria-label="生成模式">
              <button type="button" className={mode === 'text-to-image' ? 'active' : ''} onClick={() => onModeChange('text-to-image')}>文字生成</button>
              <button type="button" className={mode === 'image-to-image' ? 'active' : ''} onClick={() => onModeChange('image-to-image')}>参考图生成</button>
            </div>
          </div>
        </section>

        {isImageToImage ? (
          <section className="generate-step reference-step">
            <StepTitle index="③" title="参考图" note="上传一张或多张参考图；提示词会按你输入的内容原样提交。" />
            <UploadPanel uploads={uploads} onUpload={onUpload} onDelete={onDeleteUpload} />
          </section>
        ) : null}

        <section className="generate-step spec-step">
          <StepTitle index={specStep} title="图片规格" note={specTitleNote} />
          <div className="generate-control-grid spec-grid">
            {ratioSelectable ? (
              <ImageSpecPicker ratio={ratio} resolution={resolution} size={size} allowAutoRatio={imageModelOption.defaultRatio === 'auto'} customSizeEnabled onRatioChange={onRatioChange} onResolutionChange={onResolutionChange} onSizeChange={onSizeChange} />
            ) : (
              <AutoImageSpecNotice hint={imageModelOption.hint} />
            )}
            <QualityPicker value={quality} onChange={onQualityChange} />
            <OutputFormatPicker value={outputFormat} onChange={onOutputFormatChange} />
          </div>
        </section>

        <section className="generate-step execute-step">
          <StepTitle index={executeStep} title="数量与执行" note="提交后会自动切到结果页，生成会在后台继续进行，关闭当前页面也不会影响。" />
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

function AutoImageSpecNotice({ hint }: { hint: string }) {
  const detail = hint ? `比例、尺寸和分辨率由模型自动决定；${hint}` : '比例、尺寸和分辨率由模型自动决定。'

  return (
    <div className="size-auto-notice" aria-label="自动画幅说明">
      <span>
        <strong>自动画幅</strong>
        <small>{detail}</small>
      </span>
      <b>自动</b>
    </div>
  )
}

function readNumberInput(value: string): NumericInputValue {
  return value === '' ? '' : Number(value)
}
