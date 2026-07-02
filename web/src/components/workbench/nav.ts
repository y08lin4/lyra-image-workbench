export type WorkbenchTab = 'generate' | 'gif' | 'assistant' | 'agent' | 'nodes' | 'library' | 'square' | 'modelSquare' | 'result' | 'profile' | 'topup' | 'apiDocs' | 'settings' | 'admin'
export type WorkbenchNavId = WorkbenchTab
export type WorkbenchMobileNavId = WorkbenchNavId | 'more'
export type WorkbenchTabItem = { id: WorkbenchNavId; label: string; hint: string; badge?: string; tone?: 'normal' | 'danger' | 'active' | 'admin' }
export type WorkbenchMobileTabItem = Omit<WorkbenchTabItem, 'id'> & { id: WorkbenchMobileNavId }
export type WorkbenchNavGroup = { title: string; items: WorkbenchTabItem[] }

type WorkbenchNavGroupDefinition = { title: string; ids: WorkbenchNavId[] }
type ActiveWorkbenchTaskSummary = { statusText: string; progress: number; isFinal: boolean }

export type BuildWorkbenchTabItemsOptions = {
  currentKeyReady: boolean
  activeTask?: ActiveWorkbenchTaskSummary | null
  activeCount: number
  creditsBalance?: number | null
  isAdmin?: boolean
  missingKeyCount: number
}

const workflowTabs: WorkbenchTabItem[] = [
  { id: 'nodes', label: '创作画布', hint: 'Canvas' },
  { id: 'gif', label: 'GIF 动图', hint: '动效' },
  { id: 'agent', label: 'Agent 创作', hint: '多轮' },
  { id: 'generate', label: '快捷生成', hint: '快速' },
  { id: 'assistant', label: '提示词助手', hint: '润色' },
  { id: 'library', label: '提示词库', hint: '灵感' },
  { id: 'square', label: '广场', hint: '社区' },
  { id: 'modelSquare', label: '模型广场', hint: '候选' },
  { id: 'result', label: '结果', hint: '历史' },
  { id: 'profile', label: '我的', hint: '账号' },
  { id: 'topup', label: '充值', hint: '次数' },
  { id: 'apiDocs', label: 'API 文档', hint: '接入' },
  { id: 'settings', label: '设置', hint: '偏好' },
]

const desktopNavGroupDefinitions: WorkbenchNavGroupDefinition[] = [
  { title: '创作', ids: ['nodes', 'agent', 'assistant', 'generate', 'gif'] },
  { title: '素材', ids: ['result', 'library', 'square', 'modelSquare'] },
  { title: '管理', ids: ['profile', 'topup', 'apiDocs', 'settings', 'admin'] },
]

export const workbenchNavIconById: Record<WorkbenchNavId, string> = {
  nodes: '✦',
  agent: '⌘',
  assistant: '✧',
  generate: '↯',
  gif: '▣',
  result: '▤',
  library: '▥',
  square: '☆',
  modelSquare: '◇',
  profile: '◎',
  topup: '¥',
  apiDocs: '{}',
  settings: '⚙',
  admin: '⚑',
}

const mobilePrimaryTabIds: WorkbenchTab[] = ['nodes', 'result', 'square', 'profile']
const mobileMoreTabIds: WorkbenchNavId[] = ['gif', 'agent', 'generate', 'assistant', 'library', 'modelSquare', 'topup', 'apiDocs', 'settings', 'admin']

export function buildWorkbenchTabItems({
  currentKeyReady,
  activeTask,
  activeCount,
  creditsBalance,
  isAdmin,
  missingKeyCount,
}: BuildWorkbenchTabItemsOptions) {
  const items = workflowTabs.map<WorkbenchTabItem>((tab) => {
    if (tab.id === 'generate') return { ...tab, hint: currentKeyReady ? '可提交' : '缺 Key', tone: currentKeyReady ? 'normal' : 'danger' }
    if (tab.id === 'gif') return { ...tab, hint: '单图动效' }
    if (tab.id === 'agent') return { ...tab, hint: '多轮' }
    if (tab.id === 'assistant') return { ...tab, hint: '润色' }
    if (tab.id === 'library') return { ...tab, hint: '收藏', tone: 'normal' }
    if (tab.id === 'square') return { ...tab, hint: '社区' }
    if (tab.id === 'modelSquare') return { ...tab, hint: '待启用' }
    if (tab.id === 'result') {
      return {
        ...tab,
        hint: activeTask ? activeTask.statusText : activeCount ? `${activeCount} 进行中` : '队列',
        badge: activeTask ? `${activeTask.progress}%` : activeCount ? String(activeCount) : undefined,
        tone: activeTask && !activeTask.isFinal ? 'active' : activeCount ? 'active' : 'normal',
      }
    }
    if (tab.id === 'profile') return { ...tab, hint: creditsBalance != null ? `${creditsBalance} 次` : '账号' }
    if (tab.id === 'apiDocs') return { ...tab, hint: '接入' }
    if (tab.id === 'settings') return { ...tab, hint: missingKeyCount ? `${missingKeyCount} 个待设` : '已配置', badge: missingKeyCount ? '!' : undefined, tone: missingKeyCount ? 'danger' : 'normal' }
    return tab
  })
  if (isAdmin) {
    items.push({ id: 'admin', label: '站点管理', hint: '运营', tone: 'admin' })
  }
  return items
}

export function buildWorkbenchNavGroups(tabItems: WorkbenchTabItem[]): WorkbenchNavGroup[] {
  const tabItemById = new Map<WorkbenchNavId, WorkbenchTabItem>(tabItems.map((tab) => [tab.id, tab]))
  return desktopNavGroupDefinitions.map((group) => {
    const items = group.ids.map((id) => tabItemById.get(id)).filter((tab): tab is WorkbenchTabItem => Boolean(tab))
    return { title: group.title, items }
  }).filter((group) => group.items.length > 0)
}

export function buildWorkbenchMobilePrimaryTabs(tabItems: WorkbenchTabItem[]) {
  return selectWorkbenchTabs(tabItems, mobilePrimaryTabIds)
}

export function buildWorkbenchMobileMoreTabs(tabItems: WorkbenchTabItem[]) {
  return selectWorkbenchTabs(tabItems, mobileMoreTabIds)
}

export function buildWorkbenchMobileMoreSummary(mobileMoreTabs: WorkbenchTabItem[]): WorkbenchMobileTabItem {
  const hiddenDanger = mobileMoreTabs.find((tab) => tab.tone === 'danger' || tab.badge === '!')
  if (hiddenDanger) return { id: 'more', label: '更多', hint: hiddenDanger.label, badge: hiddenDanger.badge || '!', tone: 'danger' }
  const hiddenActive = mobileMoreTabs.find((tab) => tab.tone === 'active' || tab.badge)
  if (hiddenActive) return { id: 'more', label: '更多', hint: hiddenActive.label, badge: hiddenActive.badge, tone: hiddenActive.tone }
  const adminTab = mobileMoreTabs.find((tab) => tab.id === 'admin')
  if (adminTab) return { id: 'more', label: '更多', hint: '含管理', badge: adminTab.badge, tone: adminTab.tone }
  return { id: 'more', label: '更多', hint: '菜单' }
}

export function buildWorkbenchMobileTabs(mobilePrimaryTabs: WorkbenchTabItem[], mobileMoreSummary: WorkbenchMobileTabItem): WorkbenchMobileTabItem[] {
  return [
    ...mobilePrimaryTabs,
    mobileMoreSummary,
  ]
}

function selectWorkbenchTabs(tabItems: WorkbenchTabItem[], ids: WorkbenchNavId[]) {
  const tabItemById = new Map<WorkbenchNavId, WorkbenchTabItem>(tabItems.map((tab) => [tab.id, tab]))
  return ids.map((id) => tabItemById.get(id)).filter((tab): tab is WorkbenchTabItem => Boolean(tab))
}
