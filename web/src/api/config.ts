import { requestJson } from './client'
import type { UserConfig } from '../types'

export async function getUserConfig() {
  const data = await requestJson<{ ok: boolean; config: UserConfig }>('/api/config')
  return data.config
}

export async function saveApiKey(apiKey: string) {
  return saveUserConfig({ apiKey })
}

export async function saveUserConfig(payload: { apiKey?: string; bananaApiKey?: string; defaultCount?: number; defaultConcurrency?: number; autoUploadPixhost?: boolean }) {
  const data = await requestJson<{ ok: boolean; config: UserConfig }>('/api/config', {
    method: 'POST',
    body: JSON.stringify(payload),
  })
  return data.config
}
