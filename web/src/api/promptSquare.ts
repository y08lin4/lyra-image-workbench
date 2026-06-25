import { requestJson } from './client'
import type { CreatePromptSquareItemRequest, PromptSquareItem } from '../types'

export async function listPromptSquareItems() {
  const data = await requestJson<{ ok: boolean; items: PromptSquareItem[] }>('/api/prompt-square/items')
  return data.items || []
}

export async function createPromptSquareItem(payload: CreatePromptSquareItemRequest) {
  const form = new FormData()
  appendIfPresent(form, 'title', payload.title)
  appendIfPresent(form, 'prompt', payload.prompt)
  appendIfPresent(form, 'negativePrompt', payload.negativePrompt)
  appendIfPresent(form, 'model', payload.model)
  appendIfPresent(form, 'imageUrl', payload.imageUrl)
  appendIfPresent(form, 'sourceName', payload.sourceName)
  appendIfPresent(form, 'sourceUrl', payload.sourceUrl)
  appendIfPresent(form, 'license', payload.license)
  appendIfPresent(form, 'authorName', payload.authorName)
  appendIfPresent(form, 'authorUrl', payload.authorUrl)
  appendIfPresent(form, 'ratio', payload.ratio)
  appendIfPresent(form, 'resolution', payload.resolution)
  appendIfPresent(form, 'quality', payload.quality)
  appendIfPresent(form, 'outputFormat', payload.outputFormat)
  for (const tag of splitTags(payload.tags || '')) {
    form.append('tags', tag)
  }
  if (payload.image) {
    form.append('image', payload.image)
  }
  const data = await requestJson<{ ok: boolean; item: PromptSquareItem }>('/api/prompt-square/items', {
    method: 'POST',
    body: form,
  })
  return data.item
}

function appendIfPresent(form: FormData, key: string, value: string | undefined) {
  const normalized = (value || '').trim()
  if (normalized) form.append(key, normalized)
}

function splitTags(value: string) {
  return value.split(/[,，\s]+/).map((item) => item.trim()).filter(Boolean).slice(0, 12)
}
