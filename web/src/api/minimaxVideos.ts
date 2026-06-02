import { requestJson } from './client'
import type { MiniMaxFileResult, MiniMaxVideoCreateRequest, MiniMaxVideoStatus, MiniMaxVideoTask } from '../types'

const MINIMAX_KEY_STORAGE = 'image-workbench:minimax-api-key:v1'
const MINIMAX_KEY_HEADER = 'X-Image-Workbench-Minimax-API-Key'

export function getMiniMaxApiKey() {
  return localStorage.getItem(MINIMAX_KEY_STORAGE) || ''
}

export function saveMiniMaxApiKey(value: string) {
  const key = value.trim()
  if (key) localStorage.setItem(MINIMAX_KEY_STORAGE, key)
  else localStorage.removeItem(MINIMAX_KEY_STORAGE)
}

export function maskMiniMaxKey(value: string) {
  const key = value.trim()
  if (!key) return ''
  if (key.length <= 8) return '********'
  return `${key.slice(0, 4)}********${key.slice(-4)}`
}

export async function createMiniMaxVideo(payload: MiniMaxVideoCreateRequest, apiKey = getMiniMaxApiKey()) {
  const data = await requestJson<{ ok: boolean; task: MiniMaxVideoTask }>('/api/minimax/videos', {
    method: 'POST',
    headers: withMiniMaxKey(apiKey),
    body: JSON.stringify(payload),
  })
  return data.task
}

export async function queryMiniMaxVideo(taskID: string, apiKey = getMiniMaxApiKey()) {
  const data = await requestJson<{ ok: boolean; task: MiniMaxVideoStatus }>(`/api/minimax/videos/${encodeURIComponent(taskID)}`, {
    headers: withMiniMaxKey(apiKey),
  })
  return data.task
}

export async function retrieveMiniMaxFile(fileID: string, apiKey = getMiniMaxApiKey()) {
  const data = await requestJson<{ ok: boolean; file: MiniMaxFileResult }>(`/api/minimax/files/${encodeURIComponent(fileID)}`, {
    headers: withMiniMaxKey(apiKey),
  })
  return data.file
}

function withMiniMaxKey(apiKey: string) {
  const headers = new Headers()
  if (apiKey.trim()) headers.set(MINIMAX_KEY_HEADER, apiKey.trim())
  return headers
}
