import type { PromptSquareItem } from './promptSquare'

export interface PublicUser {
  username: string
  displayName: string
  email: string
  avatarUrl: string
  isAdmin: boolean
  creditsBalance: number
  referralCode: string
  referralLink?: string
  inviteLink?: string
  referredByUsername?: string
  createdAt: string
  lastLoginAt?: string
  twoFactorEnabled: boolean
}

export interface UserSession {
  user: PublicUser
  expiresAt: string
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

export interface DailyCreditClaim {
  claimed: boolean
  alreadyClaimed: boolean
  amount: number
  claimDate?: string
  user?: PublicUser
  entry?: CreditLedgerEntry | null
}

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

export type UserSessionResponse = {
  ok: boolean
  session: UserSession
  referralLink?: string
  inviteLink?: string
}

export type UserProfileResponse = {
  ok: boolean
  user?: PublicUser
  profile?: PublicUser
  session?: UserSession
  referralLink?: string
  inviteLink?: string
}

export type UserLedgerResponse = { ok: boolean; ledger?: CreditLedgerEntry[]; entries?: CreditLedgerEntry[] }
export type UserPromptSquareItemsResponse = { ok: boolean; items?: PromptSquareItem[] }
export type DailyCreditClaimResponse = DailyCreditClaim & { ok: boolean }
export type ReferralCodeResponse = {
  ok: boolean
  referralCode?: string
  referralLink?: string
  inviteLink?: string
  user?: PublicUser
  profile?: PublicUser
}

export type TwoFactorSetupResponse = { ok: boolean; setup: TwoFactorSetup }
