import { requestJson } from './client'
import type {
  CreatePromptSessionRequest,
  ImageToPromptRequest,
  InspirationExpandRequest,
  InspirationIdea,
  InspirationIdeasRequest,
  PromptRecord,
  PromptSession,
  RefinePromptSessionRequest,
  TextToPromptRequest,
} from '../types'

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

export async function createPromptSession(payload: CreatePromptSessionRequest) {
  const data = await requestJson<{ ok: boolean; session: PromptSession }>('/api/prompt-tools/sessions', {
    method: 'POST',
    body: JSON.stringify(payload),
  })
  return data.session
}

export async function listPromptSessions() {
  const data = await requestJson<{ ok: boolean; sessions: PromptSession[] }>('/api/prompt-tools/sessions?limit=50')
  return data.sessions
}

export async function getPromptSession(id: string) {
  const data = await requestJson<{ ok: boolean; session: PromptSession }>(`/api/prompt-tools/sessions/${encodeURIComponent(id)}`)
  return data.session
}

export async function refinePromptSession(id: string, payload: RefinePromptSessionRequest) {
  const data = await requestJson<{ ok: boolean; session: PromptSession }>(`/api/prompt-tools/sessions/${encodeURIComponent(id)}/messages`, {
    method: 'POST',
    body: JSON.stringify(payload),
  })
  return data.session
}

export async function deletePromptSession(id: string) {
  const data = await requestJson<{ ok: boolean; session: PromptSession }>(`/api/prompt-tools/sessions/${encodeURIComponent(id)}`, {
    method: 'DELETE',
  })
  return data.session
}

export async function generateInspirationIdeas(payload: InspirationIdeasRequest) {
  const data = await requestJson<{ ok: boolean; ideas: InspirationIdea[] }>('/api/prompt-tools/inspiration/ideas', {
    method: 'POST',
    body: JSON.stringify(payload),
  })
  return data.ideas
}

export async function expandInspirationIdea(payload: InspirationExpandRequest) {
  const data = await requestJson<{ ok: boolean; session: PromptSession }>('/api/prompt-tools/inspiration/expand', {
    method: 'POST',
    body: JSON.stringify(payload),
  })
  return data.session
}
