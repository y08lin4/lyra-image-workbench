import type { FormEvent } from 'react'
import type { AdminConfig } from '../../types'
import { readNumberInput, type NumericInputValue } from './adminHelpers'

type SystemTabProps = {
  config: AdminConfig | null
  siteName: string
  url: string
  publicBaseUrl: string
  systemApiKey: string
  systemBananaApiKey: string
  clearSystemApiKey: boolean
  clearSystemBananaApiKey: boolean
  timeout: NumericInputValue
  debugEnabled: boolean
  onSiteNameChange: (value: string) => void
  onUrlChange: (value: string) => void
  onPublicBaseUrlChange: (value: string) => void
  onSystemApiKeyChange: (value: string) => void
  onSystemBananaApiKeyChange: (value: string) => void
  onClearSystemApiKeyChange: (value: boolean) => void
  onClearSystemBananaApiKeyChange: (value: boolean) => void
  onTimeoutChange: (value: NumericInputValue) => void
  onDebugEnabledChange: (value: boolean) => void
  onSubmit: (event: FormEvent) => void
}

export function SystemTab({
  config,
  siteName,
  url,
  publicBaseUrl,
  systemApiKey,
  systemBananaApiKey,
  clearSystemApiKey,
  clearSystemBananaApiKey,
  timeout,
  debugEnabled,
  onSiteNameChange,
  onUrlChange,
  onPublicBaseUrlChange,
  onSystemApiKeyChange,
  onSystemBananaApiKeyChange,
  onClearSystemApiKeyChange,
  onClearSystemBananaApiKeyChange,
  onTimeoutChange,
  onDebugEnabledChange,
  onSubmit,
}: SystemTabProps) {
  return (
    <form className="admin-tab-panel admin-system-form" id="admin-panel-system" role="tabpanel" aria-labelledby="admin-tab-system" onSubmit={onSubmit}>
      <div className="admin-section-heading">
        <div>
          <h2>系统配置</h2>
          <p className="muted">站点展示、NewAPI 上游、公开域名和调试日志。</p>
        </div>
      </div>
      <div className="admin-form-grid">
        <label>站点名称<input value={siteName} onChange={(e) => onSiteNameChange(e.target.value)} placeholder="Lyra Image Workbench" /></label>
        <label>NewAPI 请求 URL<input value={url} onChange={(e) => onUrlChange(e.target.value)} placeholder="http://127.0.0.1:3000/v1" /></label>
        <label>对外访问域名<input value={publicBaseUrl} onChange={(e) => onPublicBaseUrlChange(e.target.value)} placeholder="https://image.example.com，可留空" /></label>
        <label>系统 Image-2 / codex-key<input type="password" value={systemApiKey} onChange={(e) => onSystemApiKeyChange(e.target.value)} placeholder={config?.systemApiKeySet ? `已设置 ${config.systemApiKeyPreview || ''}，留空不修改` : '填写系统 codex-key'} autoComplete="off" spellCheck={false} /></label>
        <label>系统 Banana Key<input type="password" value={systemBananaApiKey} onChange={(e) => onSystemBananaApiKeyChange(e.target.value)} placeholder={config?.systemBananaApiKeySet ? `已设置 ${config.systemBananaApiKeyPreview || ''}，留空不修改` : '填写系统 Banana API Key'} autoComplete="off" spellCheck={false} /></label>
        <label>超时时间（秒）<input type="number" min={config?.limits.minTimeoutSec || 60} max={config?.limits.maxTimeoutSec || 3600} value={timeout} onChange={(e) => onTimeoutChange(readNumberInput(e.target.value))} /></label>
      </div>
      <div className="admin-inline-notes">
        <label className="check-row">
          <input type="checkbox" checked={clearSystemApiKey} disabled={!config?.systemApiKeySet} onChange={(e) => onClearSystemApiKeyChange(e.target.checked)} />
          <span>清除系统 Image-2 / codex-key</span>
        </label>
        <label className="check-row">
          <input type="checkbox" checked={clearSystemBananaApiKey} disabled={!config?.systemBananaApiKeySet} onChange={(e) => onClearSystemBananaApiKeyChange(e.target.checked)} />
          <span>清除系统 Banana Key</span>
        </label>
      </div>
      <label className="check-row admin-debug-toggle admin-debug-row">
        <input type="checkbox" checked={debugEnabled} onChange={(e) => onDebugEnabledChange(e.target.checked)} />
        <span>开启 Debug 日志：新任务会在前端结果页显示脱敏后的请求 URL、参数、上游状态和错误详情</span>
      </label>
      <div className="admin-inline-notes">
        <div className="status-line">当前对外域名：{config?.publicBaseUrl || '未设置'}。用于记录部署域名，反代仍在宝塔/Nginx 里配置。</div>
        <div className="status-line">系统 Image-2 Key：{config?.systemApiKeySet ? `已设置 ${config.systemApiKeyPreview || ''}` : '未设置'}；系统 Banana Key：{config?.systemBananaApiKeySet ? `已设置 ${config.systemBananaApiKeyPreview || ''}` : '未设置'}。</div>
        <div className="status-line">默认 Image-2 模型：{config?.model || 'gpt-image-2'}；Banana Nano 在工作台按规格路由到独立模型 ID。</div>
      </div>
      <div className="admin-panel-actions">
        <button className="primary" type="submit">保存管理配置</button>
      </div>
    </form>
  )
}
