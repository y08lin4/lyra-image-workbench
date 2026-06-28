import { type FormEvent, useEffect, useMemo, useState } from 'react'
import type { ReferenceUpload } from '../types'
import { DEFAULT_IMAGE2_MODEL } from '../lib/models'
import { formatBytes } from '../lib/format'
import {
  buildGifPrompt,
  buildGifTaskDraft,
  GIF_PRESETS,
  RHYTHM_LABELS,
  STRENGTH_LABELS,
  type GifPresetId,
  type GifTaskDraft,
  type LoopRhythm,
  type MotionStrength,
} from '../api/gifTasks'
import { UploadPanel } from './UploadPanel'
import './GifPage.css'

export type GifDraftSubmission = GifTaskDraft

export type GifHistoryImage = {
  id: string
  src: string
  title: string
  subtitle?: string
  taskId?: string
  index: number
  prompt?: string
}

type Props = {
  uploads: ReferenceUpload[]
  recentResults: GifHistoryImage[]
  keyReady: boolean
  keyPreview: string
  message: string
  error: string
  onUpload: (files: File[]) => void
  onDeleteUpload: (id: string) => void
  onUseHistoryImageAsReference: (src: string, index: number) => Promise<ReferenceUpload | undefined>
  onOpenSettings: () => void
  onSubmitTask: (draft: GifTaskDraft) => Promise<void>
}

export function GifPage({
  uploads,
  recentResults,
  keyReady,
  keyPreview,
  message,
  error,
  onUpload,
  onDeleteUpload,
  onUseHistoryImageAsReference,
  onOpenSettings,
  onSubmitTask,
}: Props) {
  const [selectedUploadId, setSelectedUploadId] = useState('')
  const [presetId, setPresetId] = useState<GifPresetId>('hair-sway')
  const [description, setDescription] = useState('头发动起来，背景保持不变')
  const [strength, setStrength] = useState<MotionStrength>('standard')
  const [rhythm, setRhythm] = useState<LoopRhythm>('smooth')
  const [localError, setLocalError] = useState('')
  const [draft, setDraft] = useState<GifTaskDraft | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const [historyImportingId, setHistoryImportingId] = useState('')

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

  async function submitTask(event: FormEvent) {
    event.preventDefault()
    setLocalError('')

    if (!selectedReference) {
      setLocalError('请先上传并选择一张参考图')
      return
    }

    const nextDraft = buildGifTaskDraft({
      preset: selectedPreset,
      reference: selectedReference,
      description,
      strength,
      rhythm,
    })
    setDraft(nextDraft)
    setSubmitting(true)
    try {
      await onSubmitTask(nextDraft)
    } catch (err) {
      const message = err instanceof Error ? err.message : 'GIF 任务创建失败'
      setLocalError(formatGifCreateError(message))
    } finally {
      setSubmitting(false)
    }
  }

  async function useHistoryImage(item: GifHistoryImage) {
    setLocalError('')
    setHistoryImportingId(item.id)
    try {
      const created = await onUseHistoryImageAsReference(item.src, item.index)
      if (created) {
        setSelectedUploadId(created.id)
        setDraft(null)
      }
    } catch (err) {
      setLocalError(err instanceof Error ? err.message : '历史图片加入参考图失败')
    } finally {
      setHistoryImportingId('')
    }
  }

  const requestPreview = draft?.payload || (selectedReference ? buildGifTaskDraft({
    preset: selectedPreset,
    reference: selectedReference,
    description,
    strength,
    rhythm,
  }).payload : {
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
    uploadIds: [],
  })

  return (
    <section className="gif-page-shell" data-gif-composer>
      <div className="request-status-row gif-status-row">
        <div>
          <strong>当前模式</strong>
          <span>GIF 动图 · 独立后端任务</span>
        </div>
        <button type="button" className={keyReady ? 'key-ready' : 'key-missing'} onClick={onOpenSettings}>
          {keyReady ? `Image-2 Key ${keyPreview || '已设置'}` : '设置 Image-2 Key'}
        </button>
      </div>

      <form className="gif-workflow-grid" onSubmit={submitTask}>
        <div className="gif-main-column">
          <section className="generate-step gif-step">
            <StepTitle index="①" title="参考图" note="上传图片或从历史结果加入一张作为动图来源；当前不接收视频输入。" />
            <UploadPanel uploads={uploads} onUpload={onUpload} onDelete={onDeleteUpload} />
            {recentResults.length ? (
              <section className="gif-history-picker" aria-label="历史图片参考">
                <div className="gif-history-picker-head">
                  <strong>历史图片</strong>
                  <span>从生成历史复制为 GIF 参考图</span>
                </div>
                <div className="gif-history-list">
                  {recentResults.map((item) => (
                    <button
                      key={item.id}
                      type="button"
                      onClick={() => void useHistoryImage(item)}
                      disabled={Boolean(historyImportingId)}
                    >
                      <img src={item.src} alt={item.title} />
                      <span>
                        <strong>{item.title}</strong>
                        <small>{historyImportingId === item.id ? '加入中...' : item.subtitle || '历史结果'}</small>
                      </span>
                    </button>
                  ))}
                </div>
              </section>
            ) : null}
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
            <StepTitle index="④" title="提交" note="创建后端 GIF 任务并进入结果历史。" />
            <div className="gif-submit-row">
              <div className={`submit-readiness ${selectedReference ? 'ready' : 'missing'}`}>
                <strong>{selectedReference ? '可以创建任务' : '需要参考图'}</strong>
                <span>{selectedReference ? `${selectedPreset.title} · ${STRENGTH_LABELS[strength]} · ${RHYTHM_LABELS[rhythm]}` : '上传并选择一张图片'}</span>
              </div>
              <button className="primary generate-submit" type="submit" disabled={submitting}>
                {submitting ? '创建中...' : '创建 GIF 任务'}
              </button>
            </div>
            <p className="gif-backend-note">会创建 /api/background-tasks GIF 任务并进入结果历史；后端会基于单张参考图生成循环动效。</p>
            {draft ? <div className="ok">GIF 任务参数已提交：{draft.preset.title} · {draft.reference.originalName}</div> : null}
            {message ? <div className="ok">{message}</div> : null}
            {localError || error ? <div className="error">{localError || error}</div> : null}
          </section>
        </div>

        <aside className="gif-draft-panel" aria-label="GIF 任务参数预览">
          <div>
            <span>Task payload</span>
            <strong>mode: gif</strong>
          </div>
          <dl className="gif-draft-summary">
            <dt>参考图</dt>
            <dd>{selectedReference ? selectedReference.originalName : '未选择'}</dd>
            <dt>预设</dt>
            <dd>{selectedPreset.title}</dd>
            <dt>输出</dt>
            <dd>GIF · 后端任务</dd>
          </dl>
          <pre>{JSON.stringify(requestPreview, null, 2)}</pre>
        </aside>
      </form>
    </section>
  )
}


function formatGifCreateError(message: string) {
  if (message.includes('任务模式无效')) return `${message}；请确认后端服务已重启到包含 GIF 模式的最新版本。`
  if (message.includes('参考图不存在') || message.includes('参考图 ID 无效')) return `${message}；这张参考图可能已经被清理，请重新上传或重新从历史选择。`
  return message
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
