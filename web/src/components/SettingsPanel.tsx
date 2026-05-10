import { type FormEvent, useEffect, useState } from 'react'
import { getUserConfig, saveUserConfig } from '../api/config'
import type { UserConfig } from '../types'

type NumericInputValue = number | ''

export function SettingsPanel({ onReady, onConfig }: { onReady?: (ready: boolean) => void; onConfig?: (config: UserConfig) => void }) {
  const [config, setConfig] = useState<UserConfig | null>(null)
  const [apiKey, setApiKey] = useState('')
  const [bananaApiKey, setBananaApiKey] = useState('')
  const [defaultConcurrency, setDefaultConcurrency] = useState<NumericInputValue>(1)
  const [autoUploadPixhost, setAutoUploadPixhost] = useState(false)
  const [message, setMessage] = useState('')
  useEffect(() => {
    void getUserConfig().then((cfg) => {
      setConfig(cfg)
      setDefaultConcurrency(cfg.defaultConcurrency || 1)
      setAutoUploadPixhost(Boolean(cfg.autoUploadPixhost))
      onReady?.(cfg.apiKeySet)
      onConfig?.(cfg)
    })
  }, [onReady, onConfig])
  async function submit(event: FormEvent) {
    event.preventDefault()
    const payload: { apiKey?: string; bananaApiKey?: string; defaultConcurrency: number; autoUploadPixhost: boolean } = {
      defaultConcurrency: numericOrDefault(defaultConcurrency, 1),
      autoUploadPixhost,
    }
    if (apiKey.trim()) payload.apiKey = apiKey
    if (bananaApiKey.trim()) payload.bananaApiKey = bananaApiKey
    const cfg = await saveUserConfig(payload)
    setConfig(cfg)
    setDefaultConcurrency(cfg.defaultConcurrency || 1)
    setAutoUploadPixhost(Boolean(cfg.autoUploadPixhost))
    setApiKey('')
    setBananaApiKey('')
    setMessage(apiKey.trim() || bananaApiKey.trim() ? 'API Key 和默认并发已保存' : '默认并发已保存')
    onReady?.(cfg.apiKeySet)
    onConfig?.(cfg)
  }
  return (
    <section className="settings-flow-panel">
      <form onSubmit={submit} className="settings-flow-form">
        <section className="settings-card key-card">
          <div className="section-title">
            <span>codex-key</span>
            <small>Image-2 / 提示词助手</small>
          </div>
          <p className="muted">保存到 Go 后端当前个人空间。前端不直接请求上游，提示词助手也复用这个 Key。</p>
          <div className={`status-line ${config?.apiKeySet ? 'ready' : 'missing'}`}>当前：{config?.apiKeySet ? `已设置 ${config.apiKeyPreview}` : '未设置'}</div>
          <input value={apiKey} onChange={(e) => setApiKey(e.target.value)} placeholder="填写 codex-key" />
        </section>

        <section className="settings-card banana-key-card">
          <div className="section-title">
            <span>Banana 分组 Key</span>
            <small>单独 apikey</small>
          </div>
          <p className="muted">请在 NewAPI / CLIProxyAPI 里新建一个“banana”分组的 apikey，然后填到这里；URL 仍使用 Admin 页面里的 NewAPI URL。</p>
          <div className={`status-line ${config?.bananaApiKeySet ? 'ready' : 'missing'}`}>当前：{config?.bananaApiKeySet ? `已设置 ${config.bananaApiKeyPreview}` : '未设置'}</div>
          <input value={bananaApiKey} onChange={(e) => setBananaApiKey(e.target.value)} placeholder="填写 banana 分组 API Key" />
        </section>

        <section className="settings-card defaults-card">
          <div className="section-title">
            <span>默认生成设置</span>
            <small>提交任务时可覆盖</small>
          </div>
          <label className="field">
            <span>默认并发</span>
            <input type="number" min={1} value={defaultConcurrency} onChange={(e) => setDefaultConcurrency(readNumberInput(e.target.value))} />
          </label>
        </section>

        <section className="settings-card pixhost-card">
          <div className="section-title">
            <span>PiXhost 图床</span>
            <small>可选</small>
          </div>
          <label className="check-row">
            <input type="checkbox" checked={autoUploadPixhost} onChange={(e) => setAutoUploadPixhost(e.target.checked)} />
            <span>生成成功后自动上传到 PiXhost 图床</span>
          </label>
          <small className="muted">自动上传可关闭；关闭后仍可在结果图悬浮时手动点击“上传图床”。PiXhost 单张最大 10MB。</small>
        </section>

        <div className="settings-submit-row">
          <button type="submit" className="primary">保存设置</button>
          {message ? <small className="ok">{message}</small> : null}
        </div>
      </form>
    </section>
  )
}

function readNumberInput(value: string): NumericInputValue {
  return value === '' ? '' : Number(value)
}

function numericOrDefault(value: NumericInputValue, fallback: number) {
  return value === '' ? fallback : value
}
