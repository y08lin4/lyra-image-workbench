import { getLocalKeyScope } from '../api/client'
import type { UserConfig } from '../types'

const LOCAL_API_KEYS_STORAGE_KEY = 'image-workbench:local-api-keys:v1'
const DEFAULT_SCOPE = '__default__'

export const LOCAL_API_KEY_HEADER = 'X-Image-Workbench-API-Key'
export const LOCAL_BANANA_API_KEY_HEADER = 'X-Image-Workbench-Banana-API-Key'

export type LocalApiKeyUpdate = {
  apiKey?: string
  bananaApiKey?: string
}

type LocalApiKeys = {
  apiKey?: string
  bananaApiKey?: string
}

type LocalApiKeyStore = Record<string, LocalApiKeys>

export function mergeLocalApiKeys(config: UserConfig, scope = getLocalKeyScope()): UserConfig {
  const keys = getLocalApiKeys(scope)
  return {
    ...config,
    apiKeySet: Boolean(keys.apiKey),
    apiKeyPreview: maskSecret(keys.apiKey || ''),
    bananaApiKeySet: Boolean(keys.bananaApiKey),
    bananaApiKeyPreview: maskSecret(keys.bananaApiKey || ''),
  }
}

export function saveLocalApiKeys(update: LocalApiKeyUpdate, scope = getLocalKeyScope()) {
  const current = getLocalApiKeys(scope)
  const next = { ...current }
  if (update.apiKey !== undefined && update.apiKey.trim()) next.apiKey = update.apiKey.trim()
  if (update.bananaApiKey !== undefined && update.bananaApiKey.trim()) next.bananaApiKey = update.bananaApiKey.trim()
  writeLocalApiKeys(scope, next)
}

export function clearLocalApiKeys(update: { apiKey?: boolean; bananaApiKey?: boolean }, scope = getLocalKeyScope()) {
  const current = getLocalApiKeys(scope)
  const next = { ...current }
  if (update.apiKey) delete next.apiKey
  if (update.bananaApiKey) delete next.bananaApiKey
  writeLocalApiKeys(scope, next)
}

export function withLocalApiKeyHeaders(headers?: HeadersInit, scope = getLocalKeyScope()) {
  const next = new Headers(headers)
  const keys = getLocalApiKeys(scope)
  if (keys.apiKey) next.set(LOCAL_API_KEY_HEADER, keys.apiKey)
  if (keys.bananaApiKey) next.set(LOCAL_BANANA_API_KEY_HEADER, keys.bananaApiKey)
  return next
}

function getLocalApiKeys(scope = getLocalKeyScope()): LocalApiKeys {
  return readLocalApiKeyStore()[scopeForUser(scope)] || {}
}

function writeLocalApiKeys(scopeValue: string, keys: LocalApiKeys) {
  const store = readLocalApiKeyStore()
  const scope = scopeForUser(scopeValue)
  const next = {
    apiKey: keys.apiKey?.trim() || undefined,
    bananaApiKey: keys.bananaApiKey?.trim() || undefined,
  }
  if (next.apiKey || next.bananaApiKey) {
    store[scope] = next
  } else {
    delete store[scope]
  }
  if (Object.keys(store).length) {
    localStorage.setItem(LOCAL_API_KEYS_STORAGE_KEY, JSON.stringify(store))
  } else {
    localStorage.removeItem(LOCAL_API_KEYS_STORAGE_KEY)
  }
}

function readLocalApiKeyStore(): LocalApiKeyStore {
  const raw = localStorage.getItem(LOCAL_API_KEYS_STORAGE_KEY)
  if (!raw) return {}
  try {
    const parsed = JSON.parse(raw) as LocalApiKeyStore
    return parsed && typeof parsed === 'object' ? parsed : {}
  } catch {
    return {}
  }
}

function scopeForUser(scope: string) {
  return scope.trim().toLowerCase() || DEFAULT_SCOPE
}

function maskSecret(value: string) {
  const trimmed = value.trim()
  if (!trimmed) return ''
  if (trimmed.length <= 8) return '********'
  return `${trimmed.slice(0, 4)}********${trimmed.slice(-4)}`
}
