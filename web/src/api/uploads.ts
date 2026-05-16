import { requestJson } from './client'
import type { ReferenceUpload } from '../types'

export async function listReferenceUploads() {
  const data = await requestJson<{ ok: boolean; uploads: ReferenceUpload[] }>('/api/uploads/reference')
  return data.uploads
}

export async function uploadReferenceImages(files: File[]) {
  const form = new FormData()
  for (const file of files) form.append('image[]', file, file.name)
  const data = await requestJson<{ ok: boolean; uploads: ReferenceUpload[] }>('/api/uploads/reference', {
    method: 'POST',
    body: form,
  })
  return data.uploads
}

export async function deleteReferenceUpload(id: string) {
  await requestJson<{ ok: boolean }>(`/api/uploads/reference/${encodeURIComponent(id)}`, { method: 'DELETE' })
}

export async function getReferenceUploadBlob(id: string) {
  const response = await fetch(`/api/uploads/reference/${encodeURIComponent(id)}/image`, {
    cache: 'no-store',
    credentials: 'same-origin',
  })
  if (!response.ok) throw new Error(`读取参考图失败：HTTP ${response.status}`)
  return response.blob()
}
