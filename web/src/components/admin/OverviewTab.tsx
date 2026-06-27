import type { AdminBillingConfig, AdminConfig, AdminEmailConfig } from '../../types'
import { formatCredits, numericOrDefault, type NumericInputValue } from './adminHelpers'

type OverviewTabProps = {
  siteName: string
  newUserInitialCredits: NumericInputValue
  dailyFreeCredits: NumericInputValue
  epayEnabled: boolean
  smtpEnabled: boolean
  usersCount: number
  adminCount: number
  billingConfig: AdminBillingConfig
  emailConfig: AdminEmailConfig
  config: AdminConfig | null
}

export function OverviewTab({
  siteName,
  newUserInitialCredits,
  dailyFreeCredits,
  epayEnabled,
  smtpEnabled,
  usersCount,
  adminCount,
  billingConfig,
  emailConfig,
  config,
}: OverviewTabProps) {
  return (
    <section className="admin-tab-panel admin-overview-panel" id="admin-panel-overview" role="tabpanel" aria-labelledby="admin-tab-overview">
      <div className="admin-overview-grid" aria-label="后台运营概览">
        <section>
          <span>站点</span>
          <strong>{siteName || 'Lyra Image Workbench'}</strong>
          <small>公开显示名称</small>
        </section>
        <section>
          <span>新用户免费</span>
          <strong>{formatCredits(numericOrDefault(newUserInitialCredits, 0))}</strong>
          <small>注册后初始次数</small>
        </section>
        <section>
          <span>每日免费</span>
          <strong>{formatCredits(numericOrDefault(dailyFreeCredits, 0))}</strong>
          <small>自然日可领取次数</small>
        </section>
        <section>
          <span>易支付</span>
          <strong>{epayEnabled ? '已开启' : '未开启'}</strong>
          <small>{billingConfig.epayKeySet ? '商户 Key 已保存' : '商户 Key 未设置'}</small>
        </section>
        <section>
          <span>邮件</span>
          <strong>{smtpEnabled ? '已开启' : '未开启'}</strong>
          <small>{emailConfig.smtpPasswordSet ? `SMTP 密码已保存（${emailConfig.smtpPasswordPreview || '已隐藏'}）` : 'SMTP 密码未设置'}</small>
        </section>
        <section>
          <span>用户</span>
          <strong>{formatCredits(usersCount)}</strong>
          <small>{adminCount} 个管理员</small>
        </section>
      </div>
      <div className="admin-overview-summary">
        <div className="status-line">当前对外域名：{config?.publicBaseUrl || '未设置'}。用于记录部署域名，反代仍在宝塔/Nginx 里配置。</div>
        <div className="status-line">默认 Image-2 模型：{config?.model || 'gpt-image-2'}；Banana Nano 在工作台按规格路由到独立模型 ID。</div>
        <div className="status-line">易支付 Key：{billingConfig.epayKeySet ? `已设置（${billingConfig.epayKeyPreview || '已隐藏'}）` : '未设置'}</div>
        <div className="status-line">邮件发件：{smtpEnabled ? '已开启' : '未开启'}；SMTP 密码：{emailConfig.smtpPasswordSet ? `已设置（${emailConfig.smtpPasswordPreview || '已隐藏'}）` : '未设置'}</div>
      </div>
    </section>
  )
}
