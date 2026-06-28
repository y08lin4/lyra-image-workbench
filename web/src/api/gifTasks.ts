import { createTask } from './tasks'
import type { CreateTaskRequest, ReferenceUpload } from '../types'
import { DEFAULT_IMAGE2_MODEL } from '../lib/models'

export type MotionStrength = 'subtle' | 'standard' | 'bold'
export type LoopRhythm = 'smooth' | 'breathing' | 'snappy'
export type GifPresetId = 'hair-sway' | 'camera-push' | 'blink' | 'poster-loop' | 'cloth-breeze' | 'product-turn'

export type GifMotionPreset = {
  id: GifPresetId
  title: string
  summary: string
  prompt: string
}

export type GifTaskDraft = {
  payload: CreateTaskRequest
  preset: GifMotionPreset
  reference: ReferenceUpload
  strength: MotionStrength
  rhythm: LoopRhythm
}

export const GIF_PRESETS: GifMotionPreset[] = [
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

export const STRENGTH_LABELS: Record<MotionStrength, string> = {
  subtle: '轻微',
  standard: '标准',
  bold: '明显',
}

export const RHYTHM_LABELS: Record<LoopRhythm, string> = {
  smooth: '平滑循环',
  breathing: '呼吸节奏',
  snappy: '短促活泼',
}

export function buildGifTaskDraft(options: {
  preset: GifMotionPreset
  reference: ReferenceUpload
  description: string
  strength: MotionStrength
  rhythm: LoopRhythm
}): GifTaskDraft {
  const { preset, reference, description, strength, rhythm } = options
  const payload: CreateTaskRequest = {
    provider: 'image-2',
    model: DEFAULT_IMAGE2_MODEL,
    mode: 'gif',
    prompt: buildGifPrompt(preset, description, strength, rhythm),
    framePrompts: [
      preset.prompt,
      `motion strength: ${strength}`,
      `loop rhythm: ${rhythm}`,
      `user intent: ${description.trim() || preset.title}`,
    ],
    ratio: 'auto',
    resolution: 'auto',
    quality: 'auto',
    outputFormat: 'gif',
    count: 1,
    concurrency: 1,
    uploadIds: [reference.id],
  }
  return { payload, preset, reference, strength, rhythm }
}

export function buildGifPrompt(preset: GifMotionPreset, description: string, strength: MotionStrength, rhythm: LoopRhythm) {
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

export async function createGifTask(draft: GifTaskDraft) {
  return createTask(draft.payload)
}
