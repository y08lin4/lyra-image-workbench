import type { CreditLedgerEntry, PublicUser, UserSession } from './users'

export interface AdminUser extends PublicUser {
  role?: string
  disabled?: boolean
  referredByCode?: string
  referralRewardedAt?: string
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

export interface AdminEmailConfig {
  smtpEnabled?: boolean
  smtpHost?: string
  smtpPort?: number
  smtpUser?: string
  smtpPasswordSet?: boolean
  smtpPasswordPreview?: string
  smtpFrom?: string
  smtpSecure?: boolean
}

export interface AdminConfig {
  siteName: string
  newApiBaseUrl: string
  publicBaseUrl: string
  debugEnabled: boolean
  timeoutSec: number
  model: string
  modelLocked: boolean
  billing?: AdminBillingConfig
  email?: AdminEmailConfig
  epayEnabled?: boolean
  epayApiUrl?: string
  epayPid?: string
  epayKeySet?: boolean
  epayKeyPreview?: string
  epayMethods?: string[]
  creditPriceCents?: number
  minTopUpCredits?: number
  referralRewardCredits?: number
  smtpEnabled?: boolean
  smtpHost?: string
  smtpPort?: number
  smtpUser?: string
  smtpPasswordSet?: boolean
  smtpPasswordPreview?: string
  smtpFrom?: string
  smtpSecure?: boolean
  newUserInitialCredits?: number
  dailyFreeCredits?: number
  timeoutCode: string
  updatedAt: string
  limits: { minTimeoutSec: number; maxTimeoutSec: number }
}

export interface AdminAuthStatus {
  passwordSet: boolean
  initialized?: boolean
  setupRequired?: boolean
  sessionTtlSec: number
  updatedAt: string
}

export interface AdminSession {
  token: string
  expiresAt: string
}

export interface AdminSetupRequest {
  siteName: string
  admin: {
    username: string
    email?: string
    password: string
  }
  config: {
    newApiBaseUrl: string
    publicBaseUrl?: string
    timeoutSec: number
    debugEnabled: boolean
    newUserInitialCredits?: number
    dailyFreeCredits?: number
  }
}

export interface AdminSetupResponse {
  ok: boolean
  session: AdminSession
  auth: AdminAuthStatus
  config?: AdminConfig
  adminUser?: AdminUser
  userSession?: UserSession
}

export type AdminConfigPatch = Partial<AdminConfig> & Record<string, unknown>
export type AdminBillingConfigPatch = AdminBillingConfig & {
  epayKey?: string
  clearEpayKey?: boolean
}
export type AdminEmailConfigPatch = AdminEmailConfig & {
  smtpPassword?: string
  smtpPass?: string
  clearSmtpPassword?: boolean
  clearSmtpPass?: boolean
}

export type AdminAuthStatusResponse = { ok: boolean; auth: AdminAuthStatus }
export type AdminSessionResponse = { ok: boolean; session: AdminSession; auth: AdminAuthStatus }
export type AdminConfigResponse = { ok: boolean; config: AdminConfig }
export type AdminUsersResponse = { ok: boolean; users: AdminUser[] }
export type AdminUserLedgerResponse = { ok: boolean; ledger?: CreditLedgerEntry[]; entries?: CreditLedgerEntry[] }

export type GrantUserCreditsRequest = {
  username: string
  amount: number
  reason: string
}

export type GrantCreditsResponse = {
  ok: boolean
  user?: AdminUser
  users?: AdminUser[]
  entry?: CreditLedgerEntry
  ledger?: CreditLedgerEntry[]
}

export type SetAdminUserRoleRequest = { isAdmin: boolean } | { role: string }
export type SetAdminUserRoleResponse = { ok: boolean; user?: AdminUser; users?: AdminUser[] }
