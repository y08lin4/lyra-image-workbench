import type { Mode, ModelProvider, Task } from './tasks'

export type AgentSessionStatus =
  | 'draft'
  | 'awaiting_confirmation'
  | 'generating'
  | 'completed'
  | 'failed'
  | (string & {})

export type AgentRoundStatus =
  | 'collecting'
  | 'planning'
  | 'asking'
  | 'awaiting_confirmation'
  | 'generating'
  | 'completed'
  | 'failed'
  | (string & {})

export type AgentAction = 'ask_question' | 'propose_plan' | (string & {})
export type AgentMessageRole = 'user' | 'assistant' | 'system' | (string & {})
export type AgentReferenceSourceType = 'upload' | 'task_result' | 'agent_result' | (string & {})
export type AgentBlockType =
  | 'text'
  | 'question'
  | 'plan'
  | 'task'
  | 'result'
  | 'error'
  | 'agent_question'
  | 'agent_plan'
  | 'agent_parameters'
  | 'task_card'
  | 'result_grid'
  | 'reference_notice'
  | (string & {})

export interface AgentMessage {
  id: string
  role: AgentMessageRole
  content: string
  createdAt: string
}

export interface AgentSession {
  id: string
  title: string
  status: AgentSessionStatus
  rounds: AgentRound[]
  references?: AgentReference[]
  taskIds?: string[]
  createdAt: string
  updatedAt: string
}

export interface AgentRound {
  id: string
  index: number
  userMessage: AgentMessage
  action?: AgentAction
  question?: string
  assumptions?: string[]
  plan?: AgentPlan
  blocks: AgentBlock[]
  assistantBlocks?: AgentBlock[]
  referenceIds?: string[]
  taskIds?: string[]
  status: AgentRoundStatus
  error?: string
  raw?: string
  createdAt: string
  finishedAt?: string
}

export interface AgentBlock {
  type: AgentBlockType
  content?: string
  plan?: AgentPlan
  taskId?: string
  taskIds?: string[]
  imageUrls?: string[]
  message?: string
  referenceId?: string
  fields?: Record<string, unknown>
}

export interface AgentPlan {
  title: string
  mode: Mode
  sceneBrief: string
  visualPlan: AgentVisualPlan
  generationPrompt: string
  negativePrompt?: string
  parameters: AgentParameters
  referenceUsages?: AgentReferenceUsage[]
  mustKeep?: string[]
  avoid?: string[]
  notes?: string[]
}

export interface AgentVisualPlan {
  subject: string
  environment: string
  camera: string
  composition: string
  lighting: string
  colors: string
  materials: string
  mood: string
  style: string
}

export interface AgentParameters {
  provider: ModelProvider
  model: string
  ratio: string
  resolution: string
  quality: string
  outputFormat: string
  count: number
  concurrency: number
}

export interface AgentReferenceUsage {
  referenceId: string
  uploadId?: string
  usage: string
  mustKeep?: string[]
  canChange?: string[]
}

export interface AgentReference {
  id: string
  sourceType: AgentReferenceSourceType
  uploadId?: string
  taskId?: string
  resultIndex?: number
  originalName?: string
  fileName?: string
  mime?: string
  size?: number
  imageUrl?: string
  thumbnailUrl?: string
  prompt?: string
  removed?: boolean
  createdAt: string
}

export interface AgentSessionListOptions {
  limit?: number
}

export interface CreateAgentSessionRequest {
  title?: string
}

export interface SendAgentMessageRequest {
  content: string
  referenceIds?: string[]
  provider?: ModelProvider
  model?: string
  ratio?: string
  skipQuestions?: boolean
}

export interface ConfirmAgentRoundRequest {
  provider: ModelProvider
  model: string
  ratio: string
  resolution: string
  quality: string
  outputFormat: string
  count: number
  concurrency: number
  uploadIds: string[]
}

export type AgentSessionsResponse = { ok: boolean; sessions: AgentSession[] }
export type AgentSessionResponse = { ok: boolean; session: AgentSession }
export type DeleteAgentSessionResponse = { ok: boolean }
export type SendAgentMessageResponse = {
  ok: boolean
  session: AgentSession
  round: AgentRound
  blocks?: AgentBlock[]
}
export type ConfirmAgentRoundResponse = {
  ok: boolean
  taskIds: string[]
  tasks?: Task[]
  session?: AgentSession
  task?: Task
  job?: Task
  round?: AgentRound
}
