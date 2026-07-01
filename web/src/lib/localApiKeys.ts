import { getLocalKeyScope } from '../api/client'
import type { UserConfig } from '../types'

const LOCAL_API_KEYS_STORAGE_KEY = 'image-workbench:local-api-keys:v1'
const DEFAULT_SCOPE = '__default__'

export const LOCAL_API_KEY_HEADER = 'X-Image-Workbench-API-Key'

export type LocalApiKeyUpdate = {
  apiKey?: string
}

type LocalApiKeys = {
  apiKey?: string
}

type LocalApiKeyStore = Record<string, LocalApiKeys>

export function mergeLocalApiKeys(config: UserConfig, scope = getLocalKeyScope()): UserConfig {
  const keys = getLocalApiKeys(scope)
  const localApiKeySet = Boolean(keys.apiKey)
  const cloudApiKeySet = Boolean(config.cloudApiKeySet ?? config.apiKeySet)
  const systemApiKeySet = Boolean(config.systemApiKeySet)
  return {
    ...config,
    localApiKeySet,
    localApiKeyPreview: maskSecret(keys.apiKey || ''),
    cloudApiKeySet,
    cloudApiKeyPreview: config.cloudApiKeyPreview || config.apiKeyPreview || '',
    systemApiKeySet,
    systemApiKeyPreview: '',
    apiKeySet: localApiKeySet || cloudApiKeySet || systemApiKeySet,
    apiKeyPreview: localApiKeySet ? maskSecret(keys.apiKey || '') : cloudApiKeySet ? (config.cloudApiKeyPreview || config.apiKeyPreview || '') : '',
    apiKeySource: localApiKeySet ? 'local' : cloudApiKeySet ? 'cloud' : systemApiKeySet ? 'system' : 'none',
  }
}

export function saveLocalApiKeys(update: LocalApiKeyUpdate, scope = getLocalKeyScope()) {
  const current = getLocalApiKeys(scope)
  const next = { ...current }
  if (update.apiKey !== undefined && update.apiKey.trim()) next.apiKey = update.apiKey.trim()
  writeLocalApiKeys(scope, next)
}

export function clearLocalApiKeys(update: { apiKey?: boolean }, scope = getLocalKeyScope()) {
  const current = getLocalApiKeys(scope)
  const next = { ...current }
  if (update.apiKey) delete next.apiKey
  writeLocalApiKeys(scope, next)
}

export function withLocalApiKeyHeaders(headers?: HeadersInit, scope = getLocalKeyScope()) {
  const next = new Headers(headers)
  const keys = getLocalApiKeys(scope)
  if (keys.apiKey) next.set(LOCAL_API_KEY_HEADER, keys.apiKey)
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
  }
  if (next.apiKey) {
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