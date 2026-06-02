import { requestJson } from './client'
import type { AdminAuthStatus, AdminConfig, AdminSession, AdminUser } from '../types'

export const ADMIN_TOKEN_KEY = 'image-workbench:admin-token:v1'

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

export async function setupAdminPassword(password: string) {
  const data = await requestJson<{ ok: boolean; session: AdminSession; auth: AdminAuthStatus }>('/api/admin/auth/setup', {
    method: 'POST',
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

export async function saveAdminConfig(newApiBaseUrl: string, timeoutSec: number, publicBaseUrl: string, debugEnabled: boolean, minimaxApiKey = '', clearMinimaxApiKey = false) {
  const body: Record<string, unknown> = { newApiBaseUrl, timeoutSec, publicBaseUrl, debugEnabled }
  if (minimaxApiKey.trim()) body.minimaxApiKey = minimaxApiKey.trim()
  if (clearMinimaxApiKey) body.clearMinimaxApiKey = true
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

export async function addUserVideoQuota(username: string, delta: number) {
  const data = await requestJson<{ ok: boolean; user: AdminUser; users: AdminUser[] }>('/api/admin/users/video-quota', {
    method: 'POST',
    headers: adminHeaders(),
    body: JSON.stringify({ username, delta }),
  }, '')
  return data
}
