import { type FormEvent, useEffect, useMemo, useState } from 'react'
import type { CreateTaskRequest, ReferenceUpload } from '../types'
import { DEFAULT_IMAGE2_MODEL } from '../lib/models'
import { formatBytes } from '../lib/format'
import { UploadPanel } from './UploadPanel'
import './GifPage.css'

type MotionStrength = 'subtle' | 'standard' | 'bold'
type LoopRhythm = 'smooth' | 'breathing' | 'snappy'

type GifPresetId = 'hair-sway' | 'camera-push' | 'blink' | 'poster-loop' | 'cloth-breeze' | 'product-turn'

type GifMotionPreset = {
  id: GifPresetId
  title: string
  summary: string
  prompt: string
}

export type GifDraftSubmission = {
  payload: CreateTaskRequest
  preset: GifMotionPreset
  reference: ReferenceUpload
  strength: MotionStrength
  rhythm: LoopRhythm
}

type Props = {
  uploads: ReferenceUpload[]
  keyReady: boolean
  keyPreview: string
  message: string
  error: string
  onUpload: (files: File[]) => void
  onDeleteUpload: (id: string) => void
  onOpenSettings: () => void
  onSubmitDraft: (draft: GifDraftSubmission) => void
}

const GIF_PRESETS: GifMotionPreset[] = [
  {
    id: 'hair-sway',
    title: '头发飘动',
    summary: '适合人像、二次元头像，让发丝轻轻摆动。',
    prompt: 'animate hair with gentle wind, keep the face identity and background stable',
  },
  {
    id: 'camera-push',
    title: '镜头推近',
    summary: '轻微放大主体，制造短循环镜头感。',
    prompt: 'subtle camera push-in toward the main subject, maintain image composition',
  },
  {
    id: 'blink',
    title: '眨眼微笑',
    summary: '人像轻眨眼，表情只做细微变化。',
    prompt: 'natural blink and slight smile micro-expression, preserve facial identity',
  },
  {
    id: 'poster-loop',
    title: '海报轻动效',
    summary: '让光影、烟雾、背景元素做小幅循环。',
    prompt: 'cinemagraph poster loop with subtle light, smoke, or background motion',
  },
  {
    id: 'cloth-breeze',
    title: '衣摆微风',
    summary: '服饰、裙摆、披风等轻轻摆动。',
    prompt: 'gentle breeze moving clothing edges, keep body pose stable',
  },
  {
    id: 'product-turn',
    title: '产品呼吸感',
    summary: '商品图做轻微高光和小幅转动。',
    prompt: 'premium product cinemagraph with subtle highlight sweep and tiny parallax',
  },
]

const STRENGTH_LABELS: Record<MotionStrength, string> = {
  subtle: '轻微',
  standard: '标准',
  bold: '明显',
}

const RHYTHM_LABELS: Record<LoopRhythm, string> = {
  smooth: '平滑循环',
  breathing: '呼吸节奏',
  snappy: '短促活泼',
}

export function GifPage({
  uploads,
  keyReady,
  keyPreview,
  message,
  error,
  onUpload,
  onDeleteUpload,
  onOpenSettings,
  onSubmitDraft,
}: Props) {
  const [selectedUploadId, setSelectedUploadId] = useState('')
  const [presetId, setPresetId] = useState<GifPresetId>('hair-sway')
  const [description, setDescription] = useState('头发动起来，背景保持不变')
  const [strength, setStrength] = useState<MotionStrength>('standard')
  const [rhythm, setRhythm] = useState<LoopRhythm>('smooth')
  const [localError, setLocalError] = useState('')
  const [draft, setDraft] = useState<GifDraftSubmission | null>(null)

  useEffect(() => {
    if (selectedUploadId && uploads.some((item) => item.id === selectedUploadId)) return
    setSelectedUploadId(uploads[0]?.id || '')
  }, [selectedUploadId, uploads])

  const selectedReference = useMemo(
    () => uploads.find((item) => item.id === selectedUploadId),
    [selectedUploadId, uploads],
  )
  const selectedPreset = useMemo(
    () => GIF_PRESETS.find((item) => item.id === presetId) || GIF_PRESETS[0],
    [presetId],
  )

  function submitDraft(event: FormEvent) {
    event.preventDefault()
    setLocalError('')

    if (!selectedReference) {
      setLocalError('请先上传并选择一张参考图')
      return
    }

    const payload: CreateTaskRequest = {
      provider: 'image-2',
      model: DEFAULT_IMAGE2_MODEL,
      mode: 'gif',
      prompt: buildGifPrompt(selectedPreset, description, strength, rhythm),
      framePrompts: [
        selectedPreset.prompt,
        `motion strength: ${strength}`,
        `loop rhythm: ${rhythm}`,
        `user intent: ${description.trim() || selectedPreset.title}`,
      ],
      ratio: 'auto',
      resolution: 'auto',
      quality: 'auto',
      outputFormat: 'gif',
      count: 1,
      concurrency: 1,
      uploadIds: [selectedReference.id],
    }
    const nextDraft = { payload, preset: selectedPreset, reference: selectedReference, strength, rhythm }
    setDraft(nextDraft)
    onSubmitDraft(nextDraft)
  }

  const requestPreview = draft?.payload || {
    provider: 'image-2',
    model: DEFAULT_IMAGE2_MODEL,
    mode: 'gif',
    prompt: buildGifPrompt(selectedPreset, description, strength, rhythm),
    framePrompts: [
      selectedPreset.prompt,
      `motion strength: ${strength}`,
      `loop rhythm: ${rhythm}`,
      `user intent: ${description.trim() || selectedPreset.title}`,
    ],
    ratio: 'auto',
    resolution: 'auto',
    quality: 'auto',
    outputFormat: 'gif',
    count: 1,
    concurrency: 1,
    uploadIds: selectedReference ? [selectedReference.id] : [],
  }

  return (
    <section className="gif-page-shell" data-gif-composer>
      <div className="request-status-row gif-status-row">
        <div>
          <strong>当前模式</strong>
          <span>GIF 动图 · 单图动效参数</span>
        </div>
        <button type="button" className={keyReady ? 'key-ready' : 'key-missing'} onClick={onOpenSettings}>
          {keyReady ? `Image-2 Key ${keyPreview || '已设置'}` : '去设置 Key'}
        </button>
      </div>

      <form className="gif-workflow-grid" onSubmit={submitDraft}>
        <div className="gif-main-column">
          <section className="generate-step gif-step">
            <StepTitle index="①" title="参考图" note="上传图片后选择一张作为动图来源；当前不接收视频输入。" />
            <UploadPanel uploads={uploads} onUpload={onUpload} onDelete={onDeleteUpload} />
            {uploads.length ? (
              <fieldset className="gif-reference-picker">
                <legend>选择要动起来的图片</legend>
                <div className="gif-reference-list">
                  {uploads.map((item, index) => (
                    <label key={item.id} className={selectedUploadId === item.id ? 'active' : ''}>
                      <input
                        type="radio"
                        name="gif-reference"
                        value={item.id}
                        checked={selectedUploadId === item.id}
                        onChange={() => setSelectedUploadId(item.id)}
                      />
                      <span>
                        <strong>参考图 {index + 1}</strong>
                        <small>{item.originalName} · {formatBytes(item.size)}</small>
                      </span>
                    </label>
                  ))}
                </div>
              </fieldset>
            ) : null}
          </section>

          <section className="generate-step gif-step">
            <StepTitle index="②" title="动效预设" note="先选一个基础动作，再用描述补充局部变化和保留区域。" />
            <div className="gif-preset-grid" role="radiogroup" aria-label="GIF 动效预设">
              {GIF_PRESETS.map((preset) => (
                <button
                  key={preset.id}
                  type="button"
                  className={presetId === preset.id ? 'active' : ''}
                  role="radio"
                  aria-checked={presetId === preset.id}
                  onClick={() => setPresetId(preset.id)}
                >
                  <strong>{preset.title}</strong>
                  <span>{preset.summary}</span>
                </button>
              ))}
            </div>
          </section>

          <section className="generate-step gif-step">
            <StepTitle index="③" title="描述想法" note="用自然语言说明哪里动、哪里保持不变，例如“头发和耳饰动起来，脸不要变”。" />
            <label className="composer-prompt gif-description-field">
              <span>动效描述</span>
              <textarea
                value={description}
                onChange={(event) => setDescription(event.target.value)}
                placeholder="例如：头发动起来，眼睛眨一下，背景和脸型保持不变"
                rows={4}
              />
            </label>
            <div className="gif-options-row">
              <fieldset>
                <legend>动效幅度</legend>
                <div className="mode-tabs gif-segmented" role="radiogroup" aria-label="动效幅度">
                  {(Object.keys(STRENGTH_LABELS) as MotionStrength[]).map((item) => (
                    <button key={item} type="button" className={strength === item ? 'active' : ''} onClick={() => setStrength(item)}>
                      {STRENGTH_LABELS[item]}
                    </button>
                  ))}
                </div>
              </fieldset>
              <label className="gif-select-field">
                <span>循环节奏</span>
                <select value={rhythm} onChange={(event) => setRhythm(event.target.value as LoopRhythm)}>
                  {(Object.keys(RHYTHM_LABELS) as LoopRhythm[]).map((item) => (
                    <option key={item} value={item}>{RHYTHM_LABELS[item]}</option>
                  ))}
                </select>
              </label>
            </div>
          </section>

          <section className="generate-step gif-step gif-submit-step">
            <StepTitle index="④" title="提交" note="当前先完成 GIF 任务参数闭环；真实 GIF 后端接入前不会生成动图文件。" />
            <div className="gif-submit-row">
              <div className={`submit-readiness ${selectedReference ? 'ready' : 'missing'}`}>
                <strong>{selectedReference ? '参数可准备' : '需要参考图'}</strong>
                <span>{selectedReference ? `${selectedPreset.title} · ${STRENGTH_LABELS[strength]} · ${RHYTHM_LABELS[rhythm]}` : '上传并选择一张图片'}</span>
              </div>
              <button className="primary generate-submit" type="submit">准备 GIF 任务参数</button>
            </div>
            <p className="gif-backend-note">真实 GIF API 尚未接入，本按钮不会创建后端生成任务，也不会调用视频或 FFmpeg 流程。</p>
            {draft ? <div className="ok">GIF 参数已准备：{draft.preset.title}，等待真实后端接入。</div> : null}
            {message ? <div className="ok">{message}</div> : null}
            {localError || error ? <div className="error">{localError || error}</div> : null}
          </section>
        </div>

        <aside className="gif-draft-panel" aria-label="GIF 任务参数预览">
          <div>
            <span>Draft payload</span>
            <strong>mode: gif</strong>
          </div>
          <dl className="gif-draft-summary">
            <dt>参考图</dt>
            <dd>{selectedReference ? selectedReference.originalName : '未选择'}</dd>
            <dt>预设</dt>
            <dd>{selectedPreset.title}</dd>
            <dt>输出</dt>
            <dd>GIF · 占位参数</dd>
          </dl>
          <pre>{JSON.stringify(requestPreview, null, 2)}</pre>
        </aside>
      </form>
    </section>
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

function buildGifPrompt(preset: GifMotionPreset, description: string, strength: MotionStrength, rhythm: LoopRhythm) {
  const intent = description.trim() || preset.summary
  return [
    `[GIF 动图] ${preset.title}`,
    intent,
    `preset: ${preset.prompt}`,
    `motion strength: ${STRENGTH_LABELS[strength]}`,
    `loop rhythm: ${RHYTHM_LABELS[rhythm]}`,
    'preserve identity, composition, and non-moving regions',
  ].join('\n')
}
