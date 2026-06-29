import { requestJson } from './client'
import type {
  AgentSessionListOptions,
  AgentSessionResponse,
  AgentSessionsResponse,
  ConfirmAgentRoundRequest,
  ConfirmAgentRoundResponse,
  CreateAgentSessionRequest,
  DeleteAgentSessionResponse,
  SendAgentMessageRequest,
  SendAgentMessageResponse,
} from './contracts/agents'
import { withLocalApiKeyHeaders } from '../lib/localApiKeys'

export type {
  AgentSessionListOptions,
  ConfirmAgentRoundRequest,
  CreateAgentSessionRequest,
  SendAgentMessageRequest,
} from './contracts/agents'

export async function listAgentSessions(options: AgentSessionListOptions = {}) {
  const data = await requestJson<AgentSessionsResponse>(agentSessionListPath(options), {
    headers: withLocalApiKeyHeaders(),
  })
  return data.sessions
}

export async function createAgentSession(payload: CreateAgentSessionRequest = {}) {
  const data = await requestJson<AgentSessionResponse>('/api/agents/sessions', {
    method: 'POST',
    headers: withLocalApiKeyHeaders(),
    body: JSON.stringify(payload),
  })
  return data.session
}

export async function getAgentSession(sessionId: string) {
  const data = await requestJson<AgentSessionResponse>(agentSessionPath(sessionId), {
    headers: withLocalApiKeyHeaders(),
  })
  return data.session
}

export async function deleteAgentSession(sessionId: string) {
  await requestJson<DeleteAgentSessionResponse>(agentSessionPath(sessionId), {
    method: 'DELETE',
    headers: withLocalApiKeyHeaders(),
  })
}

export async function sendAgentMessage(sessionId: string, payload: SendAgentMessageRequest) {
  return requestJson<SendAgentMessageResponse>(`${agentSessionPath(sessionId)}/messages`, {
    method: 'POST',
    headers: withLocalApiKeyHeaders(),
    body: JSON.stringify(payload),
  })
}

export async function confirmAgentRound(sessionId: string, roundId: string, payload: ConfirmAgentRoundRequest) {
  return requestJson<ConfirmAgentRoundResponse>(`${agentSessionPath(sessionId)}/rounds/${encodeURIComponent(roundId)}/confirm`, {
    method: 'POST',
    headers: withLocalApiKeyHeaders(),
    body: JSON.stringify(payload),
  })
}

function agentSessionListPath(options: AgentSessionListOptions) {
  const params = new URLSearchParams()
  if (typeof options.limit === 'number' && Number.isFinite(options.limit) && options.limit > 0) {
    params.set('limit', String(Math.floor(options.limit)))
  }
  const query = params.toString()
  return query ? `/api/agents/sessions?${query}` : '/api/agents/sessions'
}

function agentSessionPath(sessionId: string) {
  return `/api/agents/sessions/${encodeURIComponent(sessionId)}`
}
