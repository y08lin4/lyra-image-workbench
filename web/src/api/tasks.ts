import { requestJson } from './client'
import type {
  CreateTaskRequest,
  Task,
  TaskEvent,
  TaskImageUploadResponse,
  TaskJobResponse,
  TaskListResponse,
  TaskMutationResponse,
  TaskResponse,
} from './contracts/tasks'
import { withLocalApiKeyHeaders } from '../lib/localApiKeys'

export async function createTask(payload: CreateTaskRequest) {
  const data = await requestJson<TaskMutationResponse>('/api/background-tasks', {
    method: 'POST',
    headers: withLocalApiKeyHeaders(),
    body: JSON.stringify(payload),
  })
  return taskFromMutationResponse(data)
}

export async function listTasks() {
  const data = await requestJson<TaskListResponse>('/api/background-tasks?limit=50')
  return data.tasks
}

export async function getTask(id: string) {
  const data = await requestJson<TaskResponse>(`/api/background-tasks/${encodeURIComponent(id)}`)
  return data.task
}

export async function cancelTask(id: string) {
  const data = await requestJson<TaskJobResponse>(`/api/background-tasks/${encodeURIComponent(id)}/cancel`, { method: 'POST' })
  return data.job
}

export async function deleteTask(id: string) {
  const data = await requestJson<TaskJobResponse>(`/api/background-tasks/${encodeURIComponent(id)}`, { method: 'DELETE' })
  return data.job
}

export async function retryTask(id: string) {
  const data = await requestJson<TaskMutationResponse>(`/api/background-tasks/${encodeURIComponent(id)}/retry`, {
    method: 'POST',
    headers: withLocalApiKeyHeaders(),
  })
  return taskFromMutationResponse(data)
}

export async function setTaskFavorite(id: string, favorite: boolean) {
  const data = await requestJson<TaskJobResponse>(`/api/background-tasks/${encodeURIComponent(id)}/favorite`, {
    method: 'POST',
    body: JSON.stringify({ favorite }),
  })
  return data.job
}

export async function uploadTaskImageToPixhost(id: string, index: number) {
  const data = await requestJson<TaskImageUploadResponse>(`/api/background-tasks/${encodeURIComponent(id)}/images/${index}/pixhost`, { method: 'POST' })
  return data
}

export async function streamTaskEvents(id: string, onEvent: (event: TaskEvent) => void, signal: AbortSignal) {
  const response = await fetch(`/api/background-tasks/${encodeURIComponent(id)}/events`, {
    cache: 'no-store',
    credentials: 'same-origin',
    signal,
  })
  if (!response.ok || !response.body) throw new Error(`SSE HTTP ${response.status}`)
  const reader = response.body.getReader()
  const decoder = new TextDecoder()
  let buffer = ''
  const handleBlock = (block: string) => {
    const dataLine = block.split('\n').find((line) => line.startsWith('data:'))
    if (!dataLine) return
    onEvent(JSON.parse(dataLine.slice(5).trimStart()) as TaskEvent)
  }
  while (true) {
    const { value, done } = await reader.read()
    if (done) break
    buffer += decoder.decode(value, { stream: true }).replace(/\r\n/g, '\n')
    let index = buffer.indexOf('\n\n')
    while (index >= 0) {
      const block = buffer.slice(0, index).trim()
      buffer = buffer.slice(index + 2)
      if (block) handleBlock(block)
      index = buffer.indexOf('\n\n')
    }
  }
}

function taskFromMutationResponse(data: TaskMutationResponse): Task {
  if (typeof data.consumedCredits === 'number' && typeof data.job.consumedCredits !== 'number') {
    return { ...data.job, consumedCredits: data.consumedCredits }
  }
  return data.job
}
