export type EpayMethod = 'alipay' | 'wxpay' | 'qqpay' | string

export interface TopUpOption {
  credits: number
  amountCents: number
  label?: string
  bonusCredits?: number
  methods?: EpayMethod[]
}

export interface BillingTopUpOptions {
  enabled: boolean
  methods: EpayMethod[]
  options: TopUpOption[]
}

export interface CreateEpayOrderRequest {
  credits: number
  method: EpayMethod
}

export interface EpayOrder {
  tradeNo: string
  payUrl: string
  credits: number
  amountCents: number
  status: string
  method?: EpayMethod
  createdAt?: string
  paidAt?: string
}

export interface BillingTopUp {
  tradeNo: string
  payUrl?: string
  credits: number
  amountCents: number
  status: string
  method?: EpayMethod
  createdAt: string
  paidAt?: string
  thirdPartyTradeNo?: string
}

export type TopUpOptionsResponse = {
  ok: boolean
  enabled?: boolean
  methods?: EpayMethod[]
  options?: TopUpOption[]
  topupOptions?: TopUpOption[]
}

export type EpayOrderResponse = {
  ok: boolean
  order?: EpayOrder
  tradeNo?: string
  payUrl?: string
  credits?: number
  amountCents?: number
  status?: string
}

export type TopUpsResponse = { ok: boolean; topups?: BillingTopUp[]; orders?: BillingTopUp[] }
