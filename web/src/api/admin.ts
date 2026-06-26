import { requestJson } from './client'
import type { AdminAuthStatus, AdminBillingConfig, AdminConfig, AdminSession, AdminSetupRequest, AdminSetupResponse, AdminUser, CreditLedgerEntry } from '../types'

export const ADMIN_TOKEN_KEY = 'image-workbench:admin-token:v1'

export type AdminConfigPatch = Partial<AdminConfig> & Record<string, unknown>
export type AdminBillingConfigPatch = AdminBillingConfig & {
  epayKey?: string
  clearEpayKey?: boolean
}
export type GrantCreditsResponse = {
  ok: boolean
  user?: AdminUser
  users?: AdminUser[]
  entry?: CreditLedgerEntry
  ledger?: CreditLedgerEntry[]
}
export type SetAdminUserRoleResponse = { ok: boolean; user?: AdminUser; users?: AdminUser[] }

export function getAdminToken() {
  return localStorage.getItem(ADMIN_TOKEN_KEY) || ''
}

export function saveAdminToken(token: string) {
  localStorage.setItem(ADMIN_TOKEN_KEY, token)
}

export function clearAdminToken() {
  localStorage.removeItem(ADMIN_TOKEN_KEY)
}

function adminHeaders(): Record<string, string> {
  const token = getAdminToken()
  return token ? { 'X-Admin-Token': token } : {}
}

export async function getAdminAuthStatus() {
  const data = await requestJson<{ ok: boolean; auth: AdminAuthStatus }>('/api/admin/auth', {}, '')
  return data.auth
}

export async function setupAdminSite(payload: AdminSetupRequest, setupToken = '') {
  const headers: Record<string, string> = {}
  if (setupToken.trim()) headers['X-Admin-Setup-Token'] = setupToken.trim()
  const data = await requestJson<AdminSetupResponse>('/api/admin/auth/setup', {
    method: 'POST',
    headers,
    body: JSON.stringify(payload),
  }, '')
  saveAdminToken(data.session.token)
  return data
}

export async function setupAdminPassword(password: string, setupToken = '') {
  const headers: Record<string, string> = {}
  if (setupToken.trim()) headers['X-Admin-Setup-Token'] = setupToken.trim()
  const data = await requestJson<{ ok: boolean; session: AdminSession; auth: AdminAuthStatus }>('/api/admin/auth/setup', {
    method: 'POST',
    headers,
    body: JSON.stringify({ password }),
  }, '')
  saveAdminToken(data.session.token)
  return data
}
export async function loginAdmin(password: string) {
  const data = await requestJson<{ ok: boolean; session: AdminSession; auth: AdminAuthStatus }>('/api/admin/auth/session', {
    method: 'POST',
    body: JSON.stringify({ password }),
  }, '')
  saveAdminToken(data.session.token)
  return data
}

export async function logoutAdmin() {
  await requestJson<{ ok: boolean }>('/api/admin/auth/session', {
    method: 'DELETE',
    headers: adminHeaders(),
  }, '')
  clearAdminToken()
}

export async function getAdminConfig() {
  const data = await requestJson<{ ok: boolean; config: AdminConfig }>('/api/admin/config', {
    headers: adminHeaders(),
  }, '')
  return data.config
}

export async function updateAdminConfig(patch: AdminConfigPatch) {
  const data = await requestJson<{ ok: boolean; config: AdminConfig }>('/api/admin/config', {
    method: 'PUT',
    headers: adminHeaders(),
    body: JSON.stringify(patch),
  }, '')
  return data.config
}

export async function saveAdminBillingConfig(config: AdminBillingConfigPatch) {
  return updateAdminConfig(config as AdminConfigPatch)
}

export async function saveAdminConfig(siteName: string, newApiBaseUrl: string, timeoutSec: number, publicBaseUrl: string, debugEnabled: boolean) {
  const body: Record<string, unknown> = { siteName, newApiBaseUrl, timeoutSec, publicBaseUrl, debugEnabled }
  const data = await requestJson<{ ok: boolean; config: AdminConfig }>('/api/admin/config', {
    method: 'POST',
    headers: adminHeaders(),
    body: JSON.stringify(body),
  }, '')
  return data.config
}

export async function listAdminUsers() {
  const data = await requestJson<{ ok: boolean; users: AdminUser[] }>('/api/admin/users', {
    headers: adminHeaders(),
  }, '')
  return data.users || []
}

export async function grantUserCredits(username: string, amount: number, reason: string) {
  const data = await requestJson<GrantCreditsResponse>('/api/admin/users/credits/add', {
    method: 'POST',
    headers: adminHeaders(),
    body: JSON.stringify({ username, amount, reason }),
  }, '')
  return data
}

export async function listAdminUserLedger(username: string) {
  const data = await requestJson<{ ok: boolean; ledger?: CreditLedgerEntry[]; entries?: CreditLedgerEntry[] }>(`/api/admin/users/${encodeURIComponent(username)}/ledger`, {
    headers: adminHeaders(),
  }, '')
  return data.ledger || data.entries || []
}

export async function setAdminUserRole(username: string, role: string | boolean) {
  const body = typeof role === 'boolean' ? { isAdmin: role } : { role }
  const data = await requestJson<SetAdminUserRoleResponse>(`/api/admin/users/${encodeURIComponent(username)}/role`, {
    method: 'POST',
    headers: adminHeaders(),
    body: JSON.stringify(body),
  }, '')
  return data
}
