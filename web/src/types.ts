export type Mode = 'text-to-image' | 'image-to-image'
export type ModelProvider = 'image-2' | 'banana' | (string & {})
export type TaskSource = 'web' | 'api' | (string & {})
export type TaskStatus = 'queued' | 'running' | 'succeeded' | 'partial_failed' | 'failed' | 'cancelled' | 'interrupted'

export interface PublicUser {
  username: string
  displayName: string
  email: string
  avatarUrl: string
  isAdmin: boolean
  creditsBalance: number
  referralCode: string
  referredByUsername?: string
  createdAt: string
  lastLoginAt?: string
  twoFactorEnabled: boolean
}

export interface UserSession {
  user: PublicUser
  expiresAt: string
}

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
  apiKeySource?: 'local' | 'cloud' | 'none'
  bananaApiKeySource?: 'local' | 'cloud' | 'none'
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

export type CreditLedgerType =
  | 'initial_free'
  | 'daily_free'
  | 'admin_add'
  | 'purchase'
  | 'referral_reward'
  | 'task_charge'
  | 'task_refund'
  | 'refund'
  | string

export interface CreditLedgerEntry {
  id: string
  username: string
  delta: number
  balanceAfter: number
  type: CreditLedgerType
  reason?: string
  sourceId?: string
  adminActor?: string
  relatedUsername?: string
  createdAt: string
}

export interface AdminUser extends PublicUser {
  role?: string
  disabled?: boolean
}

export interface AdminBillingConfig {
  epayEnabled?: boolean
  epayApiUrl?: string
  epayPid?: string
  epayKeySet?: boolean
  epayKeyPreview?: string
  epayMethods?: string[]
  creditPriceCents?: number
  minTopUpCredits?: number
  referralRewardCredits?: number
  newUserInitialCredits?: number
  dailyFreeCredits?: number
}

export interface AdminConfig {
  newApiBaseUrl: string
  publicBaseUrl: string
  debugEnabled: boolean
  timeoutSec: number
  model: string
  modelLocked: boolean
  billing?: AdminBillingConfig
  epayEnabled?: boolean
  epayApiUrl?: string
  epayPid?: string
  epayKeySet?: boolean
  epayKeyPreview?: string
  epayMethods?: string[]
  creditPriceCents?: number
  minTopUpCredits?: number
  referralRewardCredits?: number
  newUserInitialCredits?: number
  dailyFreeCredits?: number
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
  quality: string
  outputFormat: string
  count: number
  concurrency: number
  uploadIds: string[]
}

export interface PromptLibraryImage {
  url: string
  alt?: string
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

export interface PromptSquareItem {
  id: string
  title: string
  prompt: string
  negativePrompt?: string
  model?: string
  params?: Record<string, string>
  imageUrl?: string
  thumbnailUrl?: string
  ratio?: string
  resolution?: string
  quality?: string
  outputFormat?: string
  tags?: string[]
  authorUsername?: string
  authorDisplayName?: string
  authorUrl?: string
  author: string | {
    name: string
    url?: string
  }
  source: {
    type: 'user_upload' | 'external' | 'task_result' | (string & {})
    name?: string
    url?: string
    license?: string
  }
  likeCount?: number
  likes?: number
  likedByMe?: boolean
  dailyRank?: number
  permanent?: boolean
  submittedToSquare?: boolean
  taskId?: string
  sourceTaskId?: string
  imageIndex?: number
  submittedAt?: string
  status: string
  createdAt: string
  updatedAt: string
}

export interface CreatePromptSquareItemRequest {
  title: string
  prompt: string
  negativePrompt?: string
  model?: string
  tags?: string
  imageUrl?: string
  sourceName?: string
  sourceUrl?: string
  license?: string
  authorName?: string
  authorUrl?: string
  ratio?: string
  resolution?: string
  quality?: string
  outputFormat?: string
  image?: File | null
}

export type EpayMethod = 'alipay' | 'wxpay' | 'qqpay' | string

export interface TopUpOption {
  credits: number
  amountCents: number
  label?: string
  bonusCredits?: number
  methods?: EpayMethod[]
}

export interface BillingTopUpOptions {
  enabled: boolean
  methods: EpayMethod[]
  options: TopUpOption[]
}

export interface CreateEpayOrderRequest {
  credits: number
  method: EpayMethod
}

export interface EpayOrder {
  tradeNo: string
  payUrl: string
  credits: number
  amountCents: number
  status: string
  method?: EpayMethod
  createdAt?: string
  paidAt?: string
}

export interface BillingTopUp {
  tradeNo: string
  payUrl?: string
  credits: number
  amountCents: number
  status: string
  method?: EpayMethod
  createdAt: string
  paidAt?: string
  thirdPartyTradeNo?: string
}

export interface DailyCreditClaim {
  claimed: boolean
  alreadyClaimed: boolean
  amount: number
  claimDate?: string
  user?: PublicUser
  entry?: CreditLedgerEntry | null
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
