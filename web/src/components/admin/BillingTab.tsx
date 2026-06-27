import type { AdminBillingConfig } from '../../types'
import { readNumberInput, type NumericInputValue } from './adminHelpers'

const EPAY_METHOD_CHOICES = [
  { value: 'alipay', label: '支付宝' },
  { value: 'wxpay', label: '微信支付' },
  { value: 'qqpay', label: 'QQ 钱包' },
]

type BillingTabProps = {
  billingConfig: AdminBillingConfig
  epayEnabled: boolean
  epayApiUrl: string
  epayPid: string
  epayKey: string
  clearEpayKey: boolean
  epayMethods: string[]
  creditPriceCents: NumericInputValue
  minTopUpCredits: NumericInputValue
  referralRewardCredits: NumericInputValue
  newUserInitialCredits: NumericInputValue
  dailyFreeCredits: NumericInputValue
  savingBilling: boolean
  onEpayEnabledChange: (value: boolean) => void
  onEpayApiUrlChange: (value: string) => void
  onEpayPidChange: (value: string) => void
  onEpayKeyChange: (value: string) => void
  onClearEpayKeyChange: (value: boolean) => void
  onCreditPriceCentsChange: (value: NumericInputValue) => void
  onMinTopUpCreditsChange: (value: NumericInputValue) => void
  onReferralRewardCreditsChange: (value: NumericInputValue) => void
  onNewUserInitialCreditsChange: (value: NumericInputValue) => void
  onDailyFreeCreditsChange: (value: NumericInputValue) => void
  onToggleEpayMethod: (method: string, checked: boolean) => void
  onSave: () => void
}

export function BillingTab({
  billingConfig,
  epayEnabled,
  epayApiUrl,
  epayPid,
  epayKey,
  clearEpayKey,
  epayMethods,
  creditPriceCents,
  minTopUpCredits,
  referralRewardCredits,
  newUserInitialCredits,
  dailyFreeCredits,
  savingBilling,
  onEpayEnabledChange,
  onEpayApiUrlChange,
  onEpayPidChange,
  onEpayKeyChange,
  onClearEpayKeyChange,
  onCreditPriceCentsChange,
  onMinTopUpCreditsChange,
  onReferralRewardCreditsChange,
  onNewUserInitialCreditsChange,
  onDailyFreeCreditsChange,
  onToggleEpayMethod,
  onSave,
}: BillingTabProps) {
  return (
    <section className="admin-tab-panel admin-billing-panel" id="admin-panel-billing" role="tabpanel" aria-labelledby="admin-tab-billing">
      <fieldset className="admin-billing-box">
        <legend>额度与易支付配置</legend>
        <div className="admin-section-title">
          <div>
            <strong>用户充值支付</strong>
            <span>配置易支付网关、免费次数和邀请奖励</span>
          </div>
          <span className={`admin-key-status ${billingConfig.epayKeySet ? 'ready' : 'missing'}`}>
            {billingConfig.epayKeySet ? `Key 已设置（${billingConfig.epayKeyPreview || '已隐藏'}）` : 'Key 未设置'}
          </span>
        </div>
        <label className="check-row admin-debug-toggle admin-debug-row">
          <input type="checkbox" checked={epayEnabled} onChange={(e) => onEpayEnabledChange(e.target.checked)} />
          <span>启用易支付充值</span>
        </label>
        <div className="admin-billing-grid">
          <label>网关地址<input value={epayApiUrl} onChange={(e) => onEpayApiUrlChange(e.target.value)} placeholder="https://pay.example.com/" /></label>
          <label>商户 PID<input value={epayPid} onChange={(e) => onEpayPidChange(e.target.value)} placeholder="易支付商户 ID" /></label>
          <label>商户 Key<input type="password" value={epayKey} onChange={(e) => onEpayKeyChange(e.target.value)} placeholder={billingConfig.epayKeySet ? '已保存，输入新 Key 可覆盖' : '仅保存，不会明文显示'} autoComplete="new-password" /></label>
          <label>次数单价（分）<input type="number" min={0} value={creditPriceCents} onChange={(e) => onCreditPriceCentsChange(readNumberInput(e.target.value))} /></label>
          <label>最小充值次数<input type="number" min={0} value={minTopUpCredits} onChange={(e) => onMinTopUpCreditsChange(readNumberInput(e.target.value))} /></label>
          <label>邀请奖励次数<input type="number" min={0} value={referralRewardCredits} onChange={(e) => onReferralRewardCreditsChange(readNumberInput(e.target.value))} /></label>
          <label>新用户初始免费次数<input type="number" min={0} value={newUserInitialCredits} onChange={(e) => onNewUserInitialCreditsChange(readNumberInput(e.target.value))} /></label>
          <label>每日免费次数<input type="number" min={0} value={dailyFreeCredits} onChange={(e) => onDailyFreeCreditsChange(readNumberInput(e.target.value))} /></label>
        </div>
        <label className="check-row admin-debug-toggle admin-debug-row">
          <input type="checkbox" checked={clearEpayKey} onChange={(e) => onClearEpayKeyChange(e.target.checked)} />
          <span>清空已保存的商户 Key</span>
        </label>
        <div className="admin-method-list" role="group" aria-label="支付方式">
          {EPAY_METHOD_CHOICES.map((method) => (
            <label key={method.value} className="check-row">
              <input
                type="checkbox"
                checked={epayMethods.includes(method.value)}
                onChange={(e) => onToggleEpayMethod(method.value, e.target.checked)}
              />
              <span>{method.label}</span>
            </label>
          ))}
        </div>
        <div className="admin-billing-actions">
          <button className="primary" type="button" onClick={onSave} disabled={savingBilling}>
            {savingBilling ? '保存中...' : '保存额度/易支付配置'}
          </button>
        </div>
      </fieldset>
    </section>
  )
}
