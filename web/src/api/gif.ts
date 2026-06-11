import { requestJson } from './client'
import { withLocalApiKeyHeaders } from '../lib/localApiKeys'

export type GifMotionType = 'blink' | 'smile' | 'turn_head' | 'hair_flow' | 'custom'
export type GifStrength = 'subtle' | 'medium' | 'strong'

export interface GifStatus {
  ok: boolean
  gifEnabled: boolean
  ffmpegAvailable: boolean
  ffmpegBin: string
  limits: {
    maxFrames: number
    maxFPS: number
    maxSize: number
  }
}

export interface GifPlanFrame {
  index: number
  action: string
  prompt: string
}

export interface GifPlan {
  basePrompt: string
  negativePrompt: string
  styleLock: string
  frameCount: number
  fps: number
  frames: GifPlanFrame[]
  renderHints: {
    fps: number
    loop: boolean
    recommendedWidth: number
  }
}

export interface GifPlanRequest {
  uploadId: string
  motionType: GifMotionType
  prompt: string
  frameCount: number
  fps: number
  strength: GifStrength
}

export interface GifRender {
  id: string
  sourceTaskId: string
  status: 'succeeded' | 'failed'
  fps: number
  frameIndexes: number[]
  loop: boolean
  width: number
  gifUrl?: string
  bytes?: number
  error?: string
  createdAt: string
  updatedAt: string
}

export interface GifRenderRequest {
  sourceTaskId: string
  frameIndexes: number[]
  fps: number
  loop: boolean
  width: number
}

export async function getGifStatus() {
  const data = await requestJson<GifStatus>('/api/gif/status')
  return data
}

export async function createGifPlan(payload: GifPlanRequest) {
  const data = await requestJson<{ ok: boolean; plan: GifPlan; fallback?: boolean; warning?: string }>('/api/gif/plans', {
    method: 'POST',
    headers: withLocalApiKeyHeaders(),
    body: JSON.stringify(payload),
  })
  return data
}

export async function createGifRender(payload: GifRenderRequest) {
  const data = await requestJson<{ ok: boolean; render: GifRender }>('/api/gif-renders', {
    method: 'POST',
    body: JSON.stringify(payload),
  })
  return data.render
}

export async function getGifRender(id: string) {
  const data = await requestJson<{ ok: boolean; render: GifRender }>(`/api/gif-renders/${encodeURIComponent(id)}`)
  return data.render
}
