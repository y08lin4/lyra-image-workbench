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

export type PromptSquareListOptions = {
  sort?: 'latest' | 'daily' | 'mine'
  mine?: boolean
  daily?: boolean
}

export type SubmitPromptSquareFromResultRequest = {
  taskId: string
  imageIndex: number
  title?: string
  tags?: string[] | string
}

export type PromptSquareItemsResponse = { ok: boolean; items?: PromptSquareItem[] }
export type PromptSquareItemResponse = { ok: boolean; item?: PromptSquareItem }
export type PromptSquareLikeRequest = { liked: boolean }
