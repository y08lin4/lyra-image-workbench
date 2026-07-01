export type Mode = 'text-to-image' | 'image-to-image' | 'gif'
export type ModelProvider = 'image-2' | 'banana' | (string & {})
export type TaskSource = 'web' | 'api' | (string & {})
export type TaskStatus = 'queued' | 'running' | 'succeeded' | 'partial_failed' | 'failed' | 'cancelled' | 'interrupted'

export interface TaskResult {
  index: number
  ok: boolean
  status: TaskStatus
  statusText: string
  statusCode: string
  imageUrl?: string
  outputDate?: string
  outputFileName?: string
  remoteUrl?: string
  remoteThumbUrl?: string
  uploadError?: string
  mime?: string
  bytes?: number
  revisedPrompt?: string
  actualSize?: string
  actualQuality?: string
  outputFormat?: string
  error?: string
  errorText?: string
  errorCode?: string
  errorEnglish?: string
  elapsedMs?: number
}

export interface DebugLog {
  time: string
  level: string
  stage: string
  message: string
  imageIndex: number
  fields?: Record<string, unknown>
}

export interface TaskReference {
  uploadId?: string
  originalName: string
  fileName: string
  mime: string
  size?: number
}

export interface Task {
  id: string
  provider?: ModelProvider
  model?: string
  mode: Mode
  source?: TaskSource
  prompt: string
  framePrompts?: string[]
  ratio: string
  resolution: string
  quality: string
  outputFormat: string
  size: string
  count: number
  consumedCredits?: number
  concurrency: number
  uploadIds?: string[]
  references?: TaskReference[]
  status: TaskStatus
  statusText: string
  statusCode: string
  stage: string
  stageText: string
  stageCode: string
  progress: number
  results: TaskResult[]
  debugEnabled?: boolean
  debugLogs?: DebugLog[]
  favorite?: boolean
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
  provider: ModelProvider
  model: string
  mode: Mode
  prompt: string
  framePrompts?: string[]
  ratio: string
  resolution: string
  size?: string
  quality: string
  outputFormat: string
  count: number
  concurrency: number
  uploadIds: string[]
}

export type TaskMutationResponse = { ok: boolean; job: Task; taskId?: string; consumedCredits?: number }
export type TaskListResponse = { ok: boolean; tasks: Task[] }
export type TaskResponse = { ok: boolean; task: Task }
export type TaskJobResponse = { ok: boolean; job: Task }
export type TaskImageUploadResponse = { ok: boolean; job: Task; result: Task['results'][number] }
