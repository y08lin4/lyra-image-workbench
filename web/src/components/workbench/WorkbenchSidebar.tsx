import type { UserSession } from '../../types'
import { GitHubLink } from '../GitHubLink'
import { ThemeToggle, type ThemeMode } from '../ThemeToggle'
import { workbenchNavIconById, type WorkbenchNavGroup, type WorkbenchNavId, type WorkbenchTab } from './nav'

export function WorkbenchSidebar({
  navGroups,
  activeTab,
  user,
  creditsBalance,
  theme,
  onThemeChange,
  onLogout,
  onChange,
}: {
  navGroups: WorkbenchNavGroup[]
  activeTab: WorkbenchTab
  user: UserSession['user']
  creditsBalance: number
  theme: ThemeMode
  onThemeChange: (theme?: ThemeMode) => void
  onLogout: () => void
  onChange: (tab: WorkbenchNavId) => void
}) {
  return (
    <aside className="workbench-sidebar" aria-label="桌面端工作台导航">
      <div className="workbench-sidebar-brand" aria-label="Lyra Image Workbench">
        <span>Ly</span>
        <strong>Lyra Image Workbench</strong>
      </div>
      <nav className="workbench-sidebar-nav" aria-label="工作台导航">
        {navGroups.map((group, groupIndex) => {
          const groupTitleId = `workbench-sidebar-group-${groupIndex}`
          return (
            <section key={group.title} className="workbench-sidebar-group" aria-labelledby={groupTitleId}>
              <h2 id={groupTitleId} className="workbench-sidebar-group-title">{group.title}</h2>
              <div className="workbench-sidebar-group-list">
                {group.items.map((tab) => {
                  const active = activeTab === tab.id
                  const className = `workbench-sidebar-button ${active ? 'active' : ''} ${tab.tone ? `tone-${tab.tone}` : ''}`.trim()
                  return (
                    <button key={tab.id} type="button" className={className} aria-current={active ? 'page' : undefined} onClick={() => onChange(tab.id)}>
                      <span className="workbench-sidebar-icon" aria-hidden="true">{workbenchNavIconById[tab.id]}</span>
                      <span className="workbench-sidebar-label">
                        <strong>{tab.label}</strong>
                      </span>
                    </button>
                  )
                })}
              </div>
            </section>
          )
        })}
      </nav>
      <div className="workbench-sidebar-footer" aria-label="工作台工具">
        <div className="workbench-sidebar-tools">
          <GitHubLink />
          <ThemeToggle theme={theme} onToggle={onThemeChange} />
        </div>
        <div className="workbench-sidebar-balance">
          <span>余额</span>
          <strong>{formatCredits(creditsBalance)} 次</strong>
        </div>
        <div className="workbench-sidebar-account">
          <div>
            <span>当前登录</span>
            <strong>{user.displayName || user.username}</strong>
            {user.displayName && user.displayName !== user.username ? <small>{user.username}</small> : null}
          </div>
          <button type="button" className="workbench-sidebar-logout" onClick={onLogout}>退出登录</button>
        </div>
      </div>
    </aside>
  )
}

function formatCredits(value: number | undefined) {
  return Number.isFinite(value) ? Number(value).toLocaleString() : '0'
}
