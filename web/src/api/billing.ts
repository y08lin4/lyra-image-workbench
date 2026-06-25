import { requestJson } from './client'
import type { BillingTopUp, CreateEpayOrderRequest, EpayOrder, TopUpOption } from '../types'

type TopUpOptionsResponse = { ok: boolean; options?: TopUpOption[]; topupOptions?: TopUpOption[] }
type EpayOrderResponse = { ok: boolean; order?: EpayOrder; tradeNo?: string; payUrl?: string; credits?: number; amountCents?: number; status?: string }
type TopUpsResponse = { ok: boolean; topups?: BillingTopUp[]; orders?: BillingTopUp[] }

export async function listTopUpOptions() {
  const data = await requestJson<TopUpOptionsResponse>('/api/billing/topup/options')
  return data.options || data.topupOptions || []
}

export async function createEpayOrder(payload: CreateEpayOrderRequest) {
  const data = await requestJson<EpayOrderResponse>('/api/billing/epay/orders', {
    method: 'POST',
    body: JSON.stringify(payload),
  })
  if (data.order) return data.order
  return {
    tradeNo: data.tradeNo || '',
    payUrl: data.payUrl || '',
    credits: data.credits || payload.credits,
    amountCents: data.amountCents || 0,
    status: data.status || 'pending',
    method: payload.method,
  }
}

export async function listTopUps() {
  const data = await requestJson<TopUpsResponse>('/api/billing/topups')
  return data.topups || data.orders || []
}
