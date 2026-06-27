import { requestJson } from './client'
import type { PromptLibrary } from '../types'

export type PromptLibraryParams = {
  lang?: string
  q?: string
  category?: string
  limit?: number
}

type PromptLibraryCacheRecord = {
  savedAt: string
  library: PromptLibrary
}

const PROMPT_LIBRARY_CACHE_PREFIX = 'lyra:prompt-library:v1:'
const memoryCache = new Map<string, PromptLibrary>()

function toQuery(params: PromptLibraryParams = {}) {
  const query = new URLSearchParams()
  if (params.lang) query.set('lang', params.lang)
  if (params.q) query.set('q', params.q)
  if (params.category) query.set('category', params.category)
  if (params.limit) query.set('limit', String(params.limit))
  const suffix = query.toString()
  return suffix ? `?${suffix}` : ''
}

function cacheKey(params: PromptLibraryParams = {}) {
  return `${PROMPT_LIBRARY_CACHE_PREFIX}${JSON.stringify({
    lang: params.lang || '',
    q: params.q || '',
    category: params.category || '',
    limit: params.limit ?? null,
  })}`
}

function browserStorage() {
  if (typeof window === 'undefined') return null
  try {
    return window.localStorage
  } catch {
    return null
  }
}

export function getCachedPromptLibrary(params: PromptLibraryParams = {}) {
  const key = cacheKey(params)
  const cached = memoryCache.get(key)
  if (cached) return cached
  const storage = browserStorage()
  if (!storage) return null
  try {
    const raw = storage.getItem(key)
    if (!raw) return null
    const record = JSON.parse(raw) as PromptLibraryCacheRecord
    if (!record?.library?.items) return null
    memoryCache.set(key, record.library)
    return record.library
  } catch {
    storage.removeItem(key)
    return null
  }
}

function rememberPromptLibrary(params: PromptLibraryParams, library: PromptLibrary) {
  const key = cacheKey(params)
  memoryCache.set(key, library)
  const storage = browserStorage()
  if (!storage) return
  try {
    const record: PromptLibraryCacheRecord = { savedAt: new Date().toISOString(), library }
    storage.setItem(key, JSON.stringify(record))
  } catch {
    // localStorage may be unavailable or full; memory cache still keeps this tab fast.
  }
}

export async function listPromptLibrary(params: PromptLibraryParams = {}) {
  const data = await requestJson<{ ok: boolean; library: PromptLibrary }>(`/api/prompt-library${toQuery(params)}`)
  rememberPromptLibrary(params, data.library)
  return data.library
}

export async function refreshPromptLibrary(params: PromptLibraryParams = {}) {
  const data = await requestJson<{ ok: boolean; library: PromptLibrary }>(`/api/prompt-library/refresh${toQuery(params)}`, {
    method: 'POST',
  })
  rememberPromptLibrary(params, data.library)
  return data.library
}
