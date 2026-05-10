export const SPACE_TOKEN_KEY = 'image-workbench:space-token:v1'

export function getSpaceToken() {
  return localStorage.getItem(SPACE_TOKEN_KEY) || ''
}

export function saveSpaceToken(token: string) {
  localStorage.setItem(SPACE_TOKEN_KEY, token)
}

export function clearSpaceToken() {
  localStorage.removeItem(SPACE_TOKEN_KEY)
}

export class ApiError extends Error {
  status: number
  code: string
  english: string
  chinese: string
  detail: string

  constructor(status: number, payload: { code?: string; english?: string; chinese?: string; message?: string } | null) {
    const normalized = normalizeApiError(status, payload)
    super(formatApiError(normalized))
    this.name = 'ApiError'
    this.status = status
    this.code = normalized.code
    this.english = normalized.english
    this.chinese = normalized.chinese
    this.detail = normalized.detail
  }
}

export async function requestJson<T>(path: string, options: RequestInit = {}, token = getSpaceToken()): Promise<T> {
  const headers = new Headers(options.headers)
  if (token) headers.set('X-Space-Token', token)
  if (options.body && !(options.body instanceof FormData) && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json')
  }
  const response = await fetch(path, { ...options, headers, cache: 'no-store' })
  const data = await response.json().catch(() => null) as { message?: string; chinese?: string; code?: string; english?: string } | null
  if (!response.ok) throw new ApiError(response.status, data)
  return data as T
}

export function formatError(err: unknown, fallback: string) {
  if (err instanceof ApiError) return err.message
  return err instanceof Error ? err.message : fallback
}

function normalizeApiError(status: number, payload: { code?: string; english?: string; chinese?: string; message?: string } | null) {
  if (payload) {
    return {
      code: payload.code || `HTTP_${status}`,
      english: payload.english || httpEnglish(status),
      chinese: payload.chinese || payload.message || httpChinese(status),
      detail: payload.message || '',
    }
  }
  return {
    code: `HTTP_${status}`,
    english: httpEnglish(status),
    chinese: httpChinese(status),
    detail: '',
  }
}

function formatApiError(error: { chinese: string; code: string; english: string; detail: string }) {
  const base = [error.chinese, error.code, error.english].filter(Boolean).join(' / ')
  return error.detail && error.detail !== error.chinese ? `${base}：${error.detail}` : base
}

function httpEnglish(status: number) {
  const labels: Record<number, string> = {
    400: 'bad_request',
    401: 'unauthorized',
    403: 'forbidden',
    404: 'not_found',
    405: 'method_not_allowed',
    408: 'request_timeout',
    413: 'payload_too_large',
    415: 'unsupported_media_type',
    429: 'rate_limited',
    500: 'server_error',
    502: 'bad_gateway',
    503: 'service_unavailable',
    504: 'gateway_timeout',
  }
  return labels[status] || `http_${status}`
}

function httpChinese(status: number) {
  const labels: Record<number, string> = {
    400: '请求参数无效',
    401: '需要登录或 Key 无效',
    403: '没有访问权限',
    404: '接口或资源不存在',
    405: '当前后端不支持这个请求方法，请确认后端已重启到最新版本',
    408: '请求超时',
    413: '请求体过大',
    415: '媒体类型不支持',
    429: '请求过于频繁',
    500: '后端内部错误',
    502: '上游请求失败，可能触发敏感词或上游服务暂不可用',
    503: '服务暂不可用',
    504: '网关超时',
  }
  return labels[status] || `HTTP ${status}`
}
