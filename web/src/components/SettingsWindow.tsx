import type { UserConfig } from '../types'
import { SettingsPanel } from './SettingsPanel'

export function SettingsWindow({ onClose, onConfig }: { onClose: () => void; onConfig: (config: UserConfig) => void }) {
  return (
    <div className="settings-overlay" role="presentation" onMouseDown={onClose}>
      <section className="settings-window" role="dialog" aria-modal="true" aria-label="设置" onMouseDown={(event) => event.stopPropagation()}>
        <header className="window-header">
          <div>
            <p className="eyebrow">Settings</p>
            <h2>设置</h2>
            <p className="muted">codex-key 和 Banana 分组 API Key 保存在当前浏览器本地，个人默认配置随账号保存。</p>
          </div>
          <button type="button" onClick={onClose}>关闭</button>
        </header>
        <SettingsPanel onConfig={onConfig} />
      </section>
    </div>
  )
}
