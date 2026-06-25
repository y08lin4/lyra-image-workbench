import { clearLocalKeyScope, requestJson, saveLocalKeyScope } from './client'
import type { CreditLedgerEntry, PromptSquareItem, PublicUser, UserSession } from '../types'

type UserSessionResponse = { ok: boolean; session: UserSession }
type UserProfileResponse = { ok: boolean; user?: PublicUser; profile?: PublicUser; session?: UserSession }
type UserLedgerResponse = { ok: boolean; ledger?: CreditLedgerEntry[]; entries?: CreditLedgerEntry[] }
type UserPromptSquareItemsResponse = { ok: boolean; items?: PromptSquareItem[] }
type ReferralCodeResponse = { ok: boolean; referralCode?: string; user?: PublicUser; profile?: PublicUser }

export type TwoFactorSetup = { secret: string; otpauthUrl: string }

export type RegisterUserInput = {
  username: string
  email: string
  password: string
  referralCode?: string
  legacySpacePassword?: string
}

export type UpdateUserProfileInput = {
  displayName: string
  email: string
  avatarUrl: string
}

export async function registerUser(input: RegisterUserInput) {
  const body: Record<string, string> = {
    username: input.username,
    email: input.email,
    password: input.password,
  }
  if (input.referralCode?.trim()) body.referralCode = input.referralCode.trim()
  if (input.legacySpacePassword?.trim()) body.legacySpacePassword = input.legacySpacePassword
  const data = await requestJson<UserSessionResponse>('/api/users/register', {
    method: 'POST',
    body: JSON.stringify(body),
  })
  saveLocalKeyScope(data.session.user.username)
  return data.session
}

export async function loginUser(identifier: string, password: string, totpCode = '') {
  const body: Record<string, string> = { identifier, username: identifier, password }
  if (totpCode.trim()) {
    body.totpCode = totpCode.trim()
    body.twoFactorCode = totpCode.trim()
  }
  const data = await requestJson<UserSessionResponse>('/api/users/session', {
    method: 'POST',
    body: JSON.stringify(body),
  })
  saveLocalKeyScope(data.session.user.username)
  return data.session
}

export async function getCurrentUser() {
  const data = await requestJson<UserSessionResponse>('/api/users/session')
  saveLocalKeyScope(data.session.user.username)
  return data.session
}

export async function logoutUser() {
  await requestJson<{ ok: boolean }>('/api/users/session', { method: 'DELETE' })
  clearLocalKeyScope()
}

export async function getUserProfile() {
  const data = await requestJson<UserProfileResponse>('/api/users/profile')
  return readUserFromProfileResponse(data)
}

export async function updateUserProfile(payload: UpdateUserProfileInput) {
  const data = await requestJson<UserProfileResponse>('/api/users/profile', {
    method: 'PUT',
    body: JSON.stringify(payload),
  })
  return readUserFromProfileResponse(data)
}

export async function listUserLedger() {
  const data = await requestJson<UserLedgerResponse>('/api/users/ledger')
  return data.ledger || data.entries || []
}

export async function listMyPromptSquareItems() {
  const data = await requestJson<UserPromptSquareItemsResponse>('/api/users/me/prompt-square-items')
  return data.items || []
}

export async function createReferralCode() {
  const data = await requestJson<ReferralCodeResponse>('/api/users/referral-code', { method: 'POST' })
  return {
    referralCode: data.referralCode || data.user?.referralCode || data.profile?.referralCode || '',
    user: data.user || data.profile,
  }
}

export async function setupTwoFactor() {
  const data = await requestJson<{ ok: boolean; setup: TwoFactorSetup }>('/api/users/2fa/setup', { method: 'POST' })
  return data.setup
}

export async function enableTwoFactor(code: string) {
  const data = await requestJson<UserSessionResponse>('/api/users/2fa/enable', {
    method: 'POST',
    body: JSON.stringify({ code }),
  })
  return data.session
}

export async function disableTwoFactor(code: string) {
  const data = await requestJson<UserSessionResponse>('/api/users/2fa/disable', {
    method: 'POST',
    body: JSON.stringify({ code }),
  })
  return data.session
}

function readUserFromProfileResponse(data: UserProfileResponse) {
  const user = data.user || data.profile || data.session?.user
  if (!user) throw new Error('资料接口响应缺少用户信息')
  return user
}
