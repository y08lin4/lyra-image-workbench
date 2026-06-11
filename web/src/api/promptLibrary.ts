import { requestJson } from './client'
import type { PromptLibrary } from '../types'

type PromptLibraryParams = {
  lang?: string
  q?: string
  category?: string
  limit?: number
}

function toQuery(params: PromptLibraryParams = {}) {
  const query = new URLSearchParams()
  if (params.lang) query.set('lang', params.lang)
  if (params.q) query.set('q', params.q)
  if (params.category) query.set('category', params.category)
  if (params.limit) query.set('limit', String(params.limit))
  const suffix = query.toString()
  return suffix ? `?${suffix}` : ''
}

export async function listPromptLibrary(params: PromptLibraryParams = {}) {
  const data = await requestJson<{ ok: boolean; library: PromptLibrary }>(`/api/prompt-library${toQuery(params)}`)
  return data.library
}

export async function refreshPromptLibrary(params: PromptLibraryParams = {}) {
  const data = await requestJson<{ ok: boolean; library: PromptLibrary }>(`/api/prompt-library/refresh${toQuery(params)}`, {
    method: 'POST',
  })
  return data.library
}
