import { clearLocalKeyScope, requestJson, saveLocalKeyScope } from './client'
import type { UserSession } from '../types'

type UserSessionResponse = { ok: boolean; session: UserSession }

export async function registerUser(username: string, password: string, legacySpacePassword = '') {
  const data = await requestJson<UserSessionResponse>('/api/users/register', {
    method: 'POST',
    body: JSON.stringify({ username, password, legacySpacePassword }),
  })
  saveLocalKeyScope(data.session.user.username)
  return data.session
}

export async function loginUser(username: string, password: string) {
  const data = await requestJson<UserSessionResponse>('/api/users/session', {
    method: 'POST',
    body: JSON.stringify({ username, password }),
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
