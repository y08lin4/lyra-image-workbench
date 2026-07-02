import type { AdminBillingConfig, AdminConfig, AdminEmailConfig, AdminUser } from '../../types'

export type NumericInputValue = number | ''

export function billingConfigOf(config: AdminConfig | null): AdminBillingConfig {
  const billing = config?.billing || {}
  return {
    epayEnabled: billing.epayEnabled ?? config?.epayEnabled ?? false,
    epayApiUrl: billing.epayApiUrl ?? config?.epayApiUrl ?? '',
    epayPid: billing.epayPid ?? config?.epayPid ?? '',
    epayKeySet: billing.epayKeySet ?? config?.epayKeySet ?? false,
    epayKeyPreview: billing.epayKeyPreview ?? config?.epayKeyPreview ?? '',
    epayMethods: billing.epayMethods ?? config?.epayMethods ?? ['alipay', 'wxpay'],
    creditPriceCents: billing.creditPriceCents ?? config?.creditPriceCents ?? 10,
    minTopUpCredits: billing.minTopUpCredits ?? config?.minTopUpCredits ?? 10,
    referralRewardCredits: billing.referralRewardCredits ?? config?.referralRewardCredits ?? 0,
    newUserInitialCredits: billing.newUserInitialCredits ?? config?.newUserInitialCredits ?? 0,
    dailyFreeCredits: billing.dailyFreeCredits ?? config?.dailyFreeCredits ?? 0,
  }
}

export function emailConfigOf(config: AdminConfig | null): AdminEmailConfig {
  const email = config?.email || {}
  return {
    smtpEnabled: email.smtpEnabled ?? config?.smtpEnabled ?? false,
    smtpHost: email.smtpHost ?? config?.smtpHost ?? '',
    smtpPort: email.smtpPort ?? config?.smtpPort ?? 587,
    smtpUser: email.smtpUser ?? config?.smtpUser ?? '',
    smtpPasswordSet: email.smtpPasswordSet ?? config?.smtpPasswordSet ?? false,
    smtpPasswordPreview: email.smtpPasswordPreview ?? config?.smtpPasswordPreview ?? '',
    smtpFrom: email.smtpFrom ?? config?.smtpFrom ?? '',
    smtpSecure: email.smtpSecure ?? config?.smtpSecure ?? false,
  }
}

export function filterAdminUsers(users: AdminUser[], query: string) {
  const keyword = query.trim().toLowerCase()
  if (!keyword) return users
  return users.filter((user) => [
    user.username,
    user.displayName,
    user.email,
    user.role,
    user.disabled ? 'disabled 停用 禁用 已停用' : 'enabled 正常 启用',
    user.isAdmin ? 'admin 管理员' : 'user 普通用户',
  ].join(' ').toLowerCase().includes(keyword))
}

export function normalizeAdminEpayMethods(methods: string[] | undefined) {
  const normalized = (methods?.length ? methods : ['alipay', 'wxpay']).map((method) => method.trim()).filter(Boolean)
  return Array.from(new Set(normalized))
}

export function readNumberInput(value: string): NumericInputValue {
  return value === '' ? '' : Number(value)
}

export function numericOrDefault(value: NumericInputValue, fallback: number) {
  return value === '' ? fallback : value
}

export function displayUserLabel(user: AdminUser) {
  const email = user.email ? ` · ${user.email}` : ''
  return `${user.username}${email}`
}

export function formatCredits(value: number) {
  return Number.isFinite(value) ? value.toLocaleString() : '0'
}

export function formatDelta(value: number) {
  const normalized = Number.isFinite(value) ? value : 0
  return normalized > 0 ? `+${normalized.toLocaleString()}` : normalized.toLocaleString()
}

export function formatDateTime(value?: string) {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString()
}
