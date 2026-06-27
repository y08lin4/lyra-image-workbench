import type { AdminEmailConfig } from '../../types'
import { readNumberInput, type NumericInputValue } from './adminHelpers'

type EmailTabProps = {
  emailConfig: AdminEmailConfig
  smtpEnabled: boolean
  smtpHost: string
  smtpPort: NumericInputValue
  smtpUser: string
  smtpPassword: string
  smtpFrom: string
  smtpSecure: boolean
  clearSmtpPassword: boolean
  savingEmail: boolean
  onSmtpEnabledChange: (value: boolean) => void
  onSmtpHostChange: (value: string) => void
  onSmtpPortChange: (value: NumericInputValue) => void
  onSmtpUserChange: (value: string) => void
  onSmtpPasswordChange: (value: string) => void
  onSmtpFromChange: (value: string) => void
  onSmtpSecureChange: (value: boolean) => void
  onClearSmtpPasswordChange: (value: boolean) => void
  onSave: () => void
}

export function EmailTab({
  emailConfig,
  smtpEnabled,
  smtpHost,
  smtpPort,
  smtpUser,
  smtpPassword,
  smtpFrom,
  smtpSecure,
  clearSmtpPassword,
  savingEmail,
  onSmtpEnabledChange,
  onSmtpHostChange,
  onSmtpPortChange,
  onSmtpUserChange,
  onSmtpPasswordChange,
  onSmtpFromChange,
  onSmtpSecureChange,
  onClearSmtpPasswordChange,
  onSave,
}: EmailTabProps) {
  return (
    <section className="admin-tab-panel admin-email-panel" id="admin-panel-email" role="tabpanel" aria-labelledby="admin-tab-email">
      <fieldset className="admin-billing-box admin-email-box">
        <legend>SMTP 邮件发件</legend>
        <div className="admin-section-title">
          <div>
            <strong>邮件发件设置</strong>
            <span>配置系统通知的 SMTP 连接参数</span>
          </div>
          <span className={`admin-key-status ${emailConfig.smtpPasswordSet ? 'ready' : 'missing'}`}>
            {emailConfig.smtpPasswordSet ? `密码已设置（${emailConfig.smtpPasswordPreview || '已隐藏'}）` : '密码未设置'}
          </span>
        </div>
        <label className="check-row admin-debug-toggle admin-debug-row">
          <input type="checkbox" checked={smtpEnabled} onChange={(e) => onSmtpEnabledChange(e.target.checked)} />
          <span>启用 SMTP 发件</span>
        </label>
        <div className="admin-billing-grid admin-email-grid">
          <label>SMTP Host<input value={smtpHost} onChange={(e) => onSmtpHostChange(e.target.value)} placeholder="smtp.example.com" /></label>
          <label>SMTP Port<input type="number" min={1} max={65535} value={smtpPort} onChange={(e) => onSmtpPortChange(readNumberInput(e.target.value))} /></label>
          <label>用户名<input value={smtpUser} onChange={(e) => onSmtpUserChange(e.target.value)} placeholder="noreply@example.com" autoComplete="username" /></label>
          <label>密码<input type="password" value={smtpPassword} onChange={(e) => onSmtpPasswordChange(e.target.value)} placeholder={emailConfig.smtpPasswordSet ? '已保存，输入新密码可覆盖' : '仅保存，不会明文显示'} autoComplete="new-password" /></label>
          <label className="wide">发件人<input value={smtpFrom} onChange={(e) => onSmtpFromChange(e.target.value)} placeholder="Lyra Mailer <noreply@example.com>" /></label>
        </div>
        <label className="check-row admin-debug-toggle admin-debug-row">
          <input type="checkbox" checked={smtpSecure} onChange={(e) => onSmtpSecureChange(e.target.checked)} />
          <span>使用 SSL/TLS 安全连接</span>
        </label>
        <label className="check-row admin-debug-toggle admin-debug-row">
          <input type="checkbox" checked={clearSmtpPassword} onChange={(e) => onClearSmtpPasswordChange(e.target.checked)} />
          <span>清空已保存的 SMTP 密码</span>
        </label>
        <div className="admin-inline-notes">
          <div className="status-line">当前版本仅保存发件配置，不会发送测试邮件或触发真实邮件服务。</div>
        </div>
        <div className="admin-billing-actions admin-email-actions">
          <button className="primary" type="button" onClick={onSave} disabled={savingEmail}>
            {savingEmail ? '保存中...' : '保存邮件配置'}
          </button>
        </div>
      </fieldset>
    </section>
  )
}
