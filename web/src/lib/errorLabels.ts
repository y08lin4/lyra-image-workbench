import type { TaskResult } from '../types'

export function errorReasonLabel(result: TaskResult) {
  const reason = normalizeErrorReason(result)
  return [reason.chinese, reason.code, reason.english].filter(Boolean).join(' / ')
}

export function normalizeErrorReason(result: TaskResult) {
  if (result.errorText || result.errorCode || result.errorEnglish) {
    return {
      chinese: result.errorText || '任务执行失败',
      code: result.errorCode || result.statusCode,
      english: result.errorEnglish || result.status,
    }
  }
  const raw = (result.error || '').trim().toLowerCase()
  if (raw.includes('unexpected eof')) {
    return { chinese: '上游响应提前结束', code: 'E_UPSTREAM_EOF', english: 'upstream_response_truncated' }
  }
  if (raw === 'eof' || raw.includes('empty response')) {
    return { chinese: '上游返回空响应', code: 'E_UPSTREAM_EMPTY', english: 'upstream_empty_response' }
  }
  const http = raw.match(/http\s+(\d{3})/)
  if (http) return httpErrorReason(http[1], raw)
  if (raw.includes('context deadline exceeded') || raw.includes('timeout')) {
    return { chinese: '上游请求超时', code: 'E_UPSTREAM_TIMEOUT', english: 'upstream_timeout' }
  }
  if (raw.includes('unauthorized') || raw.includes('invalid api key') || raw.includes('invalid key') || raw.includes('forbidden')) {
    return { chinese: '上游鉴权失败', code: 'E_UPSTREAM_AUTH', english: 'upstream_auth_failed' }
  }
  if (raw.includes('rate limit') || raw.includes('too many requests') || raw.includes('429')) {
    return { chinese: '上游请求限流', code: 'E_UPSTREAM_RATE_LIMIT', english: 'upstream_rate_limited' }
  }
  if (raw.includes('quota') || raw.includes('insufficient') || raw.includes('balance') || raw.includes('billing')) {
    return { chinese: '上游额度或余额不足', code: 'E_UPSTREAM_QUOTA', english: 'upstream_quota_or_balance_insufficient' }
  }
  if (raw.includes('unsupported parameter') || raw.includes('unknown parameter') || raw.includes('invalid parameter')) {
    return { chinese: '上游不支持当前参数', code: 'E_PROVIDER_UNSUPPORTED_PARAM', english: 'provider_unsupported_parameter' }
  }
  if (raw.includes('output_format') || raw.includes('unsupported format')) {
    return { chinese: '上游不支持当前输出格式', code: 'E_OUTPUT_FORMAT_UNSUPPORTED', english: 'output_format_unsupported' }
  }
  if (raw.includes('connection refused') || raw.includes('no such host') || raw.includes('connection reset') || raw.includes('tls')) {
    return { chinese: '上游网络连接失败', code: 'E_UPSTREAM_NETWORK', english: 'upstream_network_error' }
  }
  if (raw.includes('payload too large') || raw.includes('request body too large') || raw.includes('file too large')) {
    return { chinese: '图片或请求体过大', code: 'E_IMAGE_TOO_LARGE', english: 'image_too_large' }
  }
  if (raw.includes('invalid character') || raw.includes('cannot unmarshal') || raw.includes('bad json')) {
    return { chinese: '上游返回不是有效 JSON', code: 'E_UPSTREAM_BAD_JSON', english: 'upstream_bad_json' }
  }
  if (raw.includes('没有返回可用图片') || raw.includes('no usable image')) {
    return { chinese: '上游没有返回可用图片', code: 'E_UPSTREAM_NO_IMAGE', english: 'upstream_no_image' }
  }
  return { chinese: '任务执行失败', code: result.statusCode || 'J500', english: result.status || 'failed' }
}

function httpErrorReason(status: string, raw: string) {
  switch (status) {
    case '400':
      if (raw.includes('unsupported') || raw.includes('unknown parameter') || raw.includes('invalid parameter')) {
        return { chinese: '上游不支持当前参数', code: 'E_PROVIDER_UNSUPPORTED_PARAM', english: 'provider_unsupported_parameter' }
      }
      return { chinese: '上游认为请求参数无效', code: 'E_UPSTREAM_BAD_REQUEST', english: 'upstream_bad_request' }
    case '401':
    case '403':
      return { chinese: '上游鉴权失败', code: 'E_UPSTREAM_AUTH', english: 'upstream_auth_failed' }
    case '402':
      return { chinese: '上游额度或余额不足', code: 'E_UPSTREAM_QUOTA', english: 'upstream_quota_or_balance_insufficient' }
    case '404':
      return { chinese: '上游接口路径不存在', code: 'E_UPSTREAM_ROUTE_NOT_FOUND', english: 'upstream_route_not_found' }
    case '405':
      return { chinese: '上游接口不支持当前请求方法或路径', code: 'E_UPSTREAM_METHOD_NOT_ALLOWED', english: 'upstream_method_not_allowed' }
    case '408':
      return { chinese: '上游请求超时', code: 'E_UPSTREAM_TIMEOUT', english: 'upstream_timeout' }
    case '413':
      return { chinese: '图片或请求体过大', code: 'E_IMAGE_TOO_LARGE', english: 'image_or_payload_too_large' }
    case '415':
      return { chinese: '上游不支持当前图片或输出格式', code: 'E_OUTPUT_FORMAT_UNSUPPORTED', english: 'output_format_or_media_type_unsupported' }
    case '422':
      return { chinese: '上游无法处理当前请求', code: 'E_UPSTREAM_UNPROCESSABLE', english: 'upstream_unprocessable_request' }
    case '429':
      return { chinese: '上游请求限流', code: 'E_UPSTREAM_RATE_LIMIT', english: 'upstream_rate_limited' }
    case '500':
      return { chinese: '上游服务内部错误', code: 'E_UPSTREAM_SERVER', english: 'upstream_server_error' }
    case '502':
      return { chinese: '上游请求失败，可能触发敏感词或上游服务暂不可用', code: 'E_UPSTREAM_GATEWAY', english: 'upstream_gateway_error' }
    case '503':
    case '504':
      return { chinese: '上游服务暂不可用或等待超时', code: 'E_UPSTREAM_GATEWAY', english: 'upstream_gateway_error' }
    case '524':
      return { chinese: '上游网关等待超时', code: 'E_UPSTREAM_GATEWAY_TIMEOUT', english: 'upstream_gateway_timeout' }
    default:
      return { chinese: `上游接口返回 HTTP ${status}`, code: `E_UPSTREAM_HTTP_${status}`, english: `upstream_http_${status}` }
  }
}
