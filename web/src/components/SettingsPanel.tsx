import { type FormEvent, useEffect, useState } from 'react'
import { getUserConfig, saveApiKey } from '../api/config'
import type { UserConfig } from '../types'

export function SettingsPanel({ onReady, onConfig }: { onReady?: (ready: boolean) => void; onConfig?: (config: UserConfig) => void }) {
  const [config, setConfig] = useState<UserConfig | null>(null)
  const [apiKey, setApiKey] = useState('')
  const [message, setMessage] = useState('')
  useEffect(() => {
    void getUserConfig().then((cfg) => {
      setConfig(cfg)
      onReady?.(cfg.apiKeySet)
      onConfig?.(cfg)
    })
  }, [onReady, onConfig])
  async function submit(event: FormEvent) {
    event.preventDefault()
    const cfg = await saveApiKey(apiKey)
    setConfig(cfg)
    setApiKey('')
    setMessage('Image-2 Key 已保存到当前个人空间')
    onReady?.(cfg.apiKeySet)
    onConfig?.(cfg)
  }
  return (
    <section className="form-section key-section">
      <div className="section-title">
        <span>Image-2 Key</span>
        <small>后端保存</small>
      </div>
      <p className="muted">Image-2 Key 只保存到 Go 后端当前个人空间，前端不请求上游；后续会为 Gemini Banana 等模型预留独立 Key。</p>
      <div className="status-line">当前：{config?.apiKeySet ? `已设置 ${config.apiKeyPreview}` : '未设置'}</div>
      <form onSubmit={submit} className="inline-form">
        <input value={apiKey} onChange={(e) => setApiKey(e.target.value)} placeholder="填写 Image-2 Key" />
        <button type="submit">保存 Image-2 Key</button>
      </form>
      {message ? <small className="ok">{message}</small> : null}
    </section>
  )
}
