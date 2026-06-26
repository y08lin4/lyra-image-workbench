import { requestJson } from './client'
import type { BillingTopUp, BillingTopUpOptions, CreateEpayOrderRequest, EpayMethod, EpayOrder, TopUpOption } from '../types'

type TopUpOptionsResponse = { ok: boolean; enabled?: boolean; methods?: EpayMethod[]; options?: TopUpOption[]; topupOptions?: TopUpOption[] }
type EpayOrderResponse = { ok: boolean; order?: EpayOrder; tradeNo?: string; payUrl?: string; credits?: number; amountCents?: number; status?: string }
type TopUpsResponse = { ok: boolean; topups?: BillingTopUp[]; orders?: BillingTopUp[] }

export async function getTopUpOptions(): Promise<BillingTopUpOptions> {
  const data = await requestJson<TopUpOptionsResponse>('/api/billing/topup/options')
  const options = data.options || data.topupOptions || []
  return {
    enabled: Boolean(data.enabled),
    methods: normalizeMethods(data.methods),
    options: options.map((option) => ({
      ...option,
      methods: option.methods?.length ? normalizeMethods(option.methods) : undefined,
    })),
  }
}

export async function listTopUpOptions() {
  return (await getTopUpOptions()).options
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

function normalizeMethods(methods: EpayMethod[] | undefined): EpayMethod[] {
  return Array.from(new Set((methods || []).map((method) => String(method).trim()).filter(Boolean)))
}