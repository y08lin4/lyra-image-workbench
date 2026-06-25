import { requestJson } from './client'
import type { DeveloperApiKey } from '../types'

export async function listDeveloperApiKeys() {
  const data = await requestJson<{ ok: boolean; apiKeys: DeveloperApiKey[] }>('/api/developer/api-keys')
  return data.apiKeys
}

export async function createDeveloperApiKey(name: string) {
  const data = await requestJson<{ ok: boolean; apiKey: DeveloperApiKey; secret: string }>('/api/developer/api-keys', {
    method: 'POST',
    body: JSON.stringify({ name }),
  })
  return data
}

export async function deleteDeveloperApiKey(id: string) {
  await requestJson<{ ok: boolean }>(`/api/developer/api-keys/${encodeURIComponent(id)}`, { method: 'DELETE' })
}
