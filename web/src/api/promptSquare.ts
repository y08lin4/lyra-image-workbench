import { requestJson } from './client'
import type { CreatePromptSquareItemRequest, PromptSquareItem } from '../types'

export type PromptSquareListOptions = {
  sort?: 'latest' | 'daily'
  mine?: boolean
  daily?: boolean
}

type PromptSquareItemsResponse = { ok: boolean; items?: PromptSquareItem[] }
type PromptSquareItemResponse = { ok: boolean; item?: PromptSquareItem }

export type SubmitPromptSquareFromResultRequest = {
  taskId: string
  imageIndex: number
  title?: string
  tags?: string[] | string
}

export async function listPromptSquareItems(options: PromptSquareListOptions = {}) {
  const data = await requestJson<PromptSquareItemsResponse>(promptSquareListPath(options))
  return data.items || []
}

export function listDailyPromptSquareItems() {
  return listPromptSquareItems({ daily: true })
}

export function listMyPromptSquareItems() {
  return listPromptSquareItems({ mine: true })
}

export async function likePromptSquareItem(id: string, liked: boolean) {
  const data = await requestJson<PromptSquareItemResponse>(`/api/prompt-square/items/${encodeURIComponent(id)}/like`, {
    method: 'POST',
    body: JSON.stringify({ liked }),
  })
  if (!data.item) throw new Error('点赞接口响应缺少作品信息')
  return data.item
}

export async function submitPromptSquareFromResult(payload: SubmitPromptSquareFromResultRequest) {
  const data = await requestJson<PromptSquareItemResponse>('/api/prompt-square/from-result', {
    method: 'POST',
    body: JSON.stringify({
      taskId: payload.taskId,
      imageIndex: payload.imageIndex,
      title: (payload.title || '').trim(),
      tags: normalizeSubmitTags(payload.tags),
    }),
  })
  if (!data.item) throw new Error('结果投稿接口响应缺少作品信息')
  return data.item
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
  const data = await requestJson<PromptSquareItemResponse>('/api/prompt-square/items', {
    method: 'POST',
    body: form,
  })
  if (!data.item) throw new Error('投稿接口响应缺少作品信息')
  return data.item
}

function promptSquareListPath(options: PromptSquareListOptions) {
  if (options.mine) return '/api/prompt-square/mine'
  if (options.daily || options.sort === 'daily') return '/api/prompt-square/daily'
  if (options.sort) return `/api/prompt-square/items?sort=${encodeURIComponent(options.sort)}`
  return '/api/prompt-square/items'
}

function appendIfPresent(form: FormData, key: string, value: string | undefined) {
  const normalized = (value || '').trim()
  if (normalized) form.append(key, normalized)
}

function splitTags(value: string) {
  return value.split(/[,，\s]+/).map((item) => item.trim()).filter(Boolean).slice(0, 12)
}

function normalizeSubmitTags(value: string[] | string | undefined) {
  if (Array.isArray(value)) {
    return value.flatMap((item) => splitTags(item)).slice(0, 12)
  }
  return splitTags(value || '')
}
