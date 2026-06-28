import type { ModelProvider } from './api/contracts/tasks'

export type {
  PublicUser,
  UserSession,
  CreditLedgerType,
  CreditLedgerEntry,
  DailyCreditClaim,
} from './api/contracts/users'
export type {
  AdminUser,
  AdminBillingConfig,
  AdminEmailConfig,
  AdminConfig,
  AdminAuthStatus,
  AdminSession,
  AdminSetupRequest,
  AdminSetupResponse,
} from './api/contracts/admin'
export type {
  Mode,
  ModelProvider,
  TaskSource,
  TaskStatus,
  TaskResult,
  DebugLog,
  TaskReference,
  Task,
  TaskEvent,
  CreateTaskRequest,
} from './api/contracts/tasks'
export type {
  PromptSquareItem,
  CreatePromptSquareItemRequest,
} from './api/contracts/promptSquare'
export type {
  EpayMethod,
  TopUpOption,
  BillingTopUpOptions,
  CreateEpayOrderRequest,
  EpayOrder,
  BillingTopUp,
} from './api/contracts/billing'

export interface UserConfig {
  apiKeySet: boolean
  apiKeyPreview: string
  bananaApiKeySet: boolean
  bananaApiKeyPreview: string
  localApiKeySet?: boolean
  localApiKeyPreview?: string
  localBananaApiKeySet?: boolean
  localBananaApiKeyPreview?: string
  cloudApiKeySet?: boolean
  cloudApiKeyPreview?: string
  cloudBananaApiKeySet?: boolean
  cloudBananaApiKeyPreview?: string
  systemApiKeySet?: boolean
  systemApiKeyPreview?: string
  systemBananaApiKeySet?: boolean
  systemBananaApiKeyPreview?: string
  apiKeySource?: 'local' | 'cloud' | 'system' | 'none'
  bananaApiKeySource?: 'local' | 'cloud' | 'system' | 'none'
  defaultCount: number
  defaultConcurrency: number
  autoUploadPixhost: boolean
  updatedAt: string
}

export interface DeveloperApiKey {
  id: string
  name: string
  prefix: string
  createdAt: string
  lastUsedAt?: string
}

export interface ReferenceUpload {
  id: string
  originalName: string
  fileName: string
  mime: string
  size: number
  createdAt: string
}

export interface PromptLibraryImage {
  url: string
  alt?: string
}

export interface PromptLibraryReferenceImage extends PromptLibraryImage {
  itemId: string
  itemTitle: string
  itemCategory: string
}

export interface PromptLibraryUsePromptOptions {
  provider: ModelProvider
  model: string
  ratio?: string
  referenceImage?: PromptLibraryReferenceImage
}

export interface PromptLibrarySource {
  label: string
  url: string
}

export interface PromptLibraryItem {
  id: string
  title: string
  category: string
  prompt: string
  images?: PromptLibraryImage[]
  sources?: PromptLibrarySource[]
  repoUrl: string
}

export interface PromptLibrary {
  repo: string
  lang: string
  sourceUrl: string
  readmeUrl: string
  fetchedAt: string
  contentSha?: string
  stale: boolean
  fetchError?: string
  categories: string[]
  total: number
  matching: number
  items: PromptLibraryItem[]
}

export type PromptToolMode = 'text-to-prompt' | 'image-to-prompt'

export interface PromptRecord {
  id: string
  sessionId?: string
  versionId?: string
  mode: PromptToolMode
  input?: string
  style?: string
  ratio?: string
  language?: string
  target?: string
  source?: {
    type: string
    uploadId?: string
    taskId?: string
    index?: number
  }
  sourceImageUrl?: string
  flatPrompt: string
  negativePrompt?: string
  mustKeep?: string[]
  avoid?: string[]
  jsonDescription?: Record<string, unknown>
  metadata?: Record<string, unknown>
  raw?: string
  model: string
  elapsedMs: number
  createdAt: string
}

export interface TextToPromptRequest {
  input: string
  style: string
  ratio: string
  language: string
  target: string
}

export interface ImageToPromptRequest {
  source: {
    type: 'upload' | 'result'
    uploadId?: string
    taskId?: string
    index?: number
  }
  language: string
  target: string
}

export type PromptSessionKind = 'text' | 'image' | 'inspiration' | 'manual'

export interface PromptMessage {
  id: string
  role: 'user' | 'assistant' | string
  content: string
  versionId?: string
  createdAt: string
}

export interface PromptVersion {
  id: string
  index: number
  prompt: string
  negativePrompt?: string
  ratio?: string
  mustKeep?: string[]
  avoid?: string[]
  notes?: string
  sourceRecordId?: string
  model: string
  elapsedMs: number
  createdAt: string
}

export interface PromptSession {
  id: string
  kind: PromptSessionKind
  title: string
  seed?: string
  source?: PromptRecord['source']
  sourceImageUrl?: string
  target?: string
  ratio?: string
  provider?: ModelProvider
  model?: string
  messages: PromptMessage[]
  versions: PromptVersion[]
  activeVersionId: string
  createdAt: string
  updatedAt: string
}

export interface CreatePromptSessionRequest {
  title?: string
  initialPrompt: string
  negativePrompt?: string
  ratio?: string
  mustKeep?: string[]
  target?: string
  provider?: ModelProvider
  model?: string
}

export interface RefinePromptSessionRequest {
  message: string
  currentVersionId: string
  provider: ModelProvider
  model: string
}

export interface InspirationIdea {
  id: string
  title: string
  summary: string
  tags: string[]
  category?: string
  mood?: string
  style?: string
  createdAt?: string
}

export interface InspirationIdeasRequest {
  category: string
  mood: string
  style: string
  target: string
  count: number
  seed: string
}

export interface InspirationExpandRequest {
  idea: InspirationIdea
  ratio: string
  target: string
  provider: ModelProvider
  model: string
}
