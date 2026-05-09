export type Mode = 'text-to-image' | 'image-to-image'
export type TaskStatus = 'queued' | 'running' | 'succeeded' | 'partial_failed' | 'failed' | 'cancelled' | 'interrupted'

export interface SpaceSession {
  space: { id: string; displayName: string; createdAt: string; lastOpenedAt: string }
  token: string
  tokenPreview: string
  created: boolean
}

export interface UserConfig {
  apiKeySet: boolean
  apiKeyPreview: string
  updatedAt: string
}

export interface AdminConfig {
  newApiBaseUrl: string
  timeoutSec: number
  model: string
  modelLocked: boolean
  timeoutCode: string
  updatedAt: string
  limits: { minTimeoutSec: number; maxTimeoutSec: number }
}

export interface AdminAuthStatus {
  passwordSet: boolean
  sessionTtlSec: number
  updatedAt: string
}

export interface AdminSession {
  token: string
  expiresAt: string
}

export interface ReferenceUpload {
  id: string
  originalName: string
  fileName: string
  mime: string
  size: number
  createdAt: string
}

export interface TaskResult {
  index: number
  ok: boolean
  status: TaskStatus
  statusText: string
  statusCode: string
  imageUrl?: string
  mime?: string
  bytes?: number
  error?: string
  elapsedMs?: number
}

export interface Task {
  id: string
  mode: Mode
  prompt: string
  ratio: string
  resolution: string
  quality: string
  size: string
  count: number
  concurrency: number
  uploadIds?: string[]
  status: TaskStatus
  statusText: string
  statusCode: string
  stage: string
  stageText: string
  stageCode: string
  progress: number
  results: TaskResult[]
  error?: string
  createdAt: string
  updatedAt: string
  startedAt?: string
  finishedAt?: string
}

export interface TaskEvent {
  event: string
  code: string
  english: string
  chinese: string
  data?: { job?: Task; result?: TaskResult; [key: string]: unknown }
}

export interface CreateTaskRequest {
  mode: Mode
  prompt: string
  ratio: string
  resolution: string
  quality: string
  count: number
  concurrency: number
  uploadIds: string[]
}
