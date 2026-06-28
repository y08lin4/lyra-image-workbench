import type { WorkbenchMobileTabItem, WorkbenchNavId, WorkbenchTab } from './nav'

export function WorkbenchMobileTabs({ tabs, activeTab, moreActive, moreOpen, onChange, onMore }: { tabs: WorkbenchMobileTabItem[]; activeTab: WorkbenchTab; moreActive: boolean; moreOpen: boolean; onChange: (tab: WorkbenchNavId) => void; onMore: () => void }) {
  return (
    <nav className="workflow-tabs mobile-tabs" aria-label="移动端工作流导航">
      {tabs.map((tab) => {
        const isMore = tab.id === 'more'
        const active = isMore ? moreActive : activeTab === tab.id
        const className = `${active ? 'active' : ''} ${tab.tone ? `tone-${tab.tone}` : ''}`.trim()
        return (
          <button
            key={tab.id}
            type="button"
            aria-current={active ? 'page' : undefined}
            aria-expanded={isMore ? moreOpen : undefined}
            aria-haspopup={isMore ? 'dialog' : undefined}
            aria-controls={isMore ? 'mobile-more-sheet' : undefined}
            className={className}
            onClick={() => { if (tab.id === 'more') onMore(); else onChange(tab.id) }}
          >
            <strong>{tab.label}</strong>
          </button>
        )
      })}
    </nav>
  )
}
