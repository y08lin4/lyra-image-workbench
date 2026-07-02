export type AdminTab = 'overview' | 'activity' | 'system' | 'channels' | 'billing' | 'email' | 'users' | 'ledger'

export const ADMIN_TABS: Array<{ id: AdminTab; label: string; description: string }> = [
  { id: 'overview', label: '总览', description: '状态与关键指标' },
  { id: 'activity', label: '活动日志', description: '注册、充值、错误' },
  { id: 'system', label: '系统配置', description: '站点与上游' },
  { id: 'channels', label: '图片渠道', description: '模型与密钥' },
  { id: 'billing', label: '额度支付', description: '次数与易支付' },
  { id: 'email', label: '邮件', description: 'SMTP 发件' },
  { id: 'users', label: '用户管理', description: '余额与角色' },
  { id: 'ledger', label: '用户流水', description: '额度变动记录' },
]

type AdminTabsProps = {
  activeTab: AdminTab
  onSelectTab: (tab: AdminTab) => void
}

export function AdminTabs({ activeTab, onSelectTab }: AdminTabsProps) {
  return (
    <nav className="admin-tabs" role="tablist" aria-label="后台管理分区">
      {ADMIN_TABS.map((tab) => (
        <button
          key={tab.id}
          id={`admin-tab-${tab.id}`}
          type="button"
          role="tab"
          aria-selected={activeTab === tab.id}
          aria-controls={`admin-panel-${tab.id}`}
          className={activeTab === tab.id ? 'active' : undefined}
          onClick={() => onSelectTab(tab.id)}
        >
          <span>{tab.label}</span>
          <small>{tab.description}</small>
        </button>
      ))}
    </nav>
  )
}
