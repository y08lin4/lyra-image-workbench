import { requestJson } from './client'
import type { UserConfig } from '../types'
import { mergeLocalApiKeys, saveLocalApiKeys } from '../lib/localApiKeys'

export async function getUserConfig() {
  const data = await requestJson<{ ok: boolean; config: UserConfig }>('/api/config')
  return mergeLocalApiKeys(data.config)
}

export async function saveApiKey(apiKey: string) {
  return saveUserConfig({ apiKey })
}

export type SaveUserConfigPayload = {
  apiKey?: string
  saveApiKeyToCloud?: boolean
  clearCloudApiKey?: boolean
  defaultCount?: number
  defaultConcurrency?: number
  autoUploadPixhost?: boolean
}

export async function saveUserConfig(payload: SaveUserConfigPayload) {
  const { apiKey, saveApiKeyToCloud, ...rest } = payload
  saveLocalApiKeys({ apiKey })
  const serverPayload = {
    ...rest,
    ...(saveApiKeyToCloud ? { apiKey, saveApiKeyToCloud } : {}),
  }
  const data = await requestJson<{ ok: boolean; config: UserConfig }>('/api/config', {
    method: 'POST',
    body: JSON.stringify(serverPayload),
  })
  return mergeLocalApiKeys(data.config)
}