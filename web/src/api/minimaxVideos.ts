import { requestJson } from './client'
import type { MiniMaxFileResult, MiniMaxVideoCreateRequest, MiniMaxVideoQuota, MiniMaxVideoStatus, MiniMaxVideoTask } from '../types'

export async function getMiniMaxVideoQuota() {
  const data = await requestJson<{ ok: boolean; quota: MiniMaxVideoQuota }>('/api/minimax/video-quota')
  return data.quota
}

export async function createMiniMaxVideo(payload: MiniMaxVideoCreateRequest) {
  const data = await requestJson<{ ok: boolean; task: MiniMaxVideoTask; quota: { remaining: number } }>('/api/minimax/videos', {
    method: 'POST',
    body: JSON.stringify(payload),
  })
  return data
}

export async function queryMiniMaxVideo(taskID: string) {
  const data = await requestJson<{ ok: boolean; task: MiniMaxVideoStatus }>(`/api/minimax/videos/${encodeURIComponent(taskID)}`)
  return data.task
}

export async function retrieveMiniMaxFile(fileID: string) {
  const data = await requestJson<{ ok: boolean; file: MiniMaxFileResult }>(`/api/minimax/files/${encodeURIComponent(fileID)}`)
  return data.file
}
