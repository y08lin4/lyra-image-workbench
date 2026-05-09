import { requestJson } from './client'
import type { ImageToPromptRequest, PromptRecord, TextToPromptRequest } from '../types'

export async function textToPrompt(payload: TextToPromptRequest) {
  const data = await requestJson<{ ok: boolean; record: PromptRecord }>('/api/prompt-tools/text-to-prompt', {
    method: 'POST',
    body: JSON.stringify(payload),
  })
  return data.record
}

export async function imageToPrompt(payload: ImageToPromptRequest) {
  const data = await requestJson<{ ok: boolean; record: PromptRecord }>('/api/prompt-tools/image-to-prompt', {
    method: 'POST',
    body: JSON.stringify(payload),
  })
  return data.record
}

export async function listPromptHistory() {
  const data = await requestJson<{ ok: boolean; records: PromptRecord[] }>('/api/prompt-tools/history?limit=50')
  return data.records
}

export async function deletePromptHistory(id: string) {
  const data = await requestJson<{ ok: boolean; record: PromptRecord }>(`/api/prompt-tools/history/${encodeURIComponent(id)}`, {
    method: 'DELETE',
  })
  return data.record
}
