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

export async function saveUserConfig(payload: { apiKey?: string; bananaApiKey?: string; defaultCount?: number; defaultConcurrency?: number; autoUploadPixhost?: boolean }) {
  const { apiKey, bananaApiKey, ...serverPayload } = payload
  saveLocalApiKeys({ apiKey, bananaApiKey })
  const data = await requestJson<{ ok: boolean; config: UserConfig }>('/api/config', {
    method: 'POST',
    body: JSON.stringify(serverPayload),
  })
  return mergeLocalApiKeys(data.config)
}
