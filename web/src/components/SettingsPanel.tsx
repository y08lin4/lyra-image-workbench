import { type FormEvent, useEffect, useState } from 'react'
import { QRCodeSVG } from 'qrcode.react'
import { formatError } from '../api/client'
import { getUserConfig, saveUserConfig, type SaveUserConfigPayload } from '../api/config'
import { createDeveloperApiKey, deleteDeveloperApiKey, listDeveloperApiKeys } from '../api/developerKeys'
import { disableTwoFactor, enableTwoFactor, getCurrentUser, setupTwoFactor, type TwoFactorSetup } from '../api/users'
import type { DeveloperApiKey, UserConfig } from '../types'
import { clearLocalApiKeys } from '../lib/localApiKeys'

type NumericInputValue = number | ''

export function SettingsPanel({ onReady, onConfig }: { onReady?: (ready: boolean) => void; onConfig?: (config: UserConfig) => void }) {
  const [config, setConfig] = useState<UserConfig | null>(null)
  const [apiKey, setApiKey] = useState('')
  const [bananaApiKey, setBananaApiKey] = useState('')
  const [developerKeys, setDeveloperKeys] = useState<DeveloperApiKey[]>([])
  const [developerKeyName, setDeveloperKeyName] = useState('local-sdk')
  const [developerSecret, setDeveloperSecret] = useState('')
  const [saveApiKeyToCloud, setSaveApiKeyToCloud] = useState(false)
  const [saveBananaKeyToCloud, setSaveBananaKeyToCloud] = useState(false)
  const [defaultCount, setDefaultCount] = useState<NumericInputValue>(1)
  const [defaultConcurrency, setDefaultConcurrency] = useState<NumericInputValue>(1)
  const [autoUploadPixhost, setAutoUploadPixhost] = useState(false)
  const [twoFactorEnabled, setTwoFactorEnabled] = useState(false)
  const [twoFactorSetup, setTwoFactorSetup] = useState<TwoFactorSetup | null>(null)
  const [twoFactorCode, setTwoFactorCode] = useState('')
  const [showCloudWarning, setShowCloudWarning] = useState(false)
  const [savingCloudConfirmed, setSavingCloudConfirmed] = useState(false)
  const [message, setMessage] = useState('')
  const [error, setError] = useState('')

  useEffect(() => {
    void refreshAll()
  }, [onReady, onConfig])

  async function refreshAll() {
    const [cfg, session, apiKeys] = await Promise.all([getUserConfig(), getCurrentUser(), listDeveloperApiKeys()])
    applyConfig(cfg)
    setTwoFactorEnabled(Boolean(session.user.twoFactorEnabled))
    setDeveloperKeys(apiKeys)
  }

  function applyConfig(cfg: UserConfig) {
    setConfig(cfg)
    setDefaultCount(cfg.defaultCount || 1)
    setDefaultConcurrency(cfg.defaultConcurrency || 1)
    setAutoUploadPixhost(Boolean(cfg.autoUploadPixhost))
    onReady?.(cfg.apiKeySet)
    onConfig?.(cfg)
  }

  async function submit(event: FormEvent) {
    event.preventDefault()
    try {
      await saveSettings(false)
    } catch (err) {
      handleActionError(err, '保存设置失败')
    }
  }

  async function saveSettings(confirmedCloudRisk: boolean, forceLocalOnly = false) {
    setError('')
    const wantsCloud = !forceLocalOnly && ((apiKey.trim() && saveApiKeyToCloud) || (bananaApiKey.trim() && saveBananaKeyToCloud))
    if (wantsCloud && !confirmedCloudRisk) {
      setShowCloudWarning(true)
      return
    }
    setShowCloudWarning(false)
    setSavingCloudConfirmed(false)
    const payload: SaveUserConfigPayload = {
      defaultCount: numericOrDefault(defaultCount, 1),
      defaultConcurrency: numericOrDefault(defaultConcurrency, 1),
      autoUploadPixhost,
    }
    if (apiKey.trim()) {
      payload.apiKey = apiKey
      payload.saveApiKeyToCloud = !forceLocalOnly && Boolean(saveApiKeyToCloud)
    }
    if (bananaApiKey.trim()) {
      payload.bananaApiKey = bananaApiKey
      payload.saveBananaKeyToCloud = !forceLocalOnly && Boolean(saveBananaKeyToCloud)
    }
    const cfg = await saveUserConfig(payload)
    applyConfig(cfg)
    setApiKey('')
    setBananaApiKey('')
    setSaveApiKeyToCloud(false)
    setSaveBananaKeyToCloud(false)
    setMessage(apiKey.trim() || bananaApiKey.trim() ? 'API Key 和默认生成设置已保存' : '默认生成设置已保存')
  }

  async function clearLocalKey(kind: 'apiKey' | 'bananaApiKey') {
    setError('')
    clearLocalApiKeys(kind === 'apiKey' ? { apiKey: true } : { bananaApiKey: true })
    const cfg = await getUserConfig()
    applyConfig(cfg)
    setApiKey('')
    setBananaApiKey('')
    setMessage(kind === 'apiKey' ? 'codex-key 已从当前浏览器清除' : 'Banana Key 已从当前浏览器清除')
  }

  async function clearCloudKey(kind: 'apiKey' | 'bananaApiKey') {
    setError('')
    const cfg = await saveUserConfig(kind === 'apiKey' ? { clearCloudApiKey: true } : { clearCloudBananaApiKey: true })
    applyConfig(cfg)
    setMessage(kind === 'apiKey' ? 'codex-key 已从云端清除' : 'Banana Key 已从云端清除')
  }

  async function createDeveloperKey() {
    try {
      setError('')
      setDeveloperSecret('')
      const result = await createDeveloperApiKey(developerKeyName)
      setDeveloperKeys(await listDeveloperApiKeys())
      setDeveloperSecret(result.secret)
      setMessage('开发者 API Key 已生成，请保存 Secret')
    } catch (err) {
      handleActionError(err, '生成开发者 API Key 失败')
    }
  }

  async function removeDeveloperKey(id: string) {
    try {
      setError('')
      await deleteDeveloperApiKey(id)
      setDeveloperKeys((items) => items.filter((item) => item.id !== id))
      setMessage('开发者 API Key 已删除')
    } catch (err) {
      handleActionError(err, '删除开发者 API Key 失败')
    }
  }

  async function startTwoFactorSetup() {
    setError('')
    setTwoFactorSetup(await setupTwoFactor())
    setTwoFactorCode('')
    setMessage('请用验证器 App 录入 2FA 密钥，并输入 6 位验证码完成开启')
  }

  async function confirmTwoFactor() {
    try {
      setError('')
      const session = await enableTwoFactor(twoFactorCode)
      setTwoFactorEnabled(Boolean(session.user.twoFactorEnabled))
      setTwoFactorSetup(null)
      setTwoFactorCode('')
      setMessage('2FA 已开启')
    } catch (err) {
      handleActionError(err, '开启 2FA 失败')
    }
  }

  async function turnOffTwoFactor() {
    try {
      setError('')
      const session = await disableTwoFactor(twoFactorCode)
      setTwoFactorEnabled(Boolean(session.user.twoFactorEnabled))
      setTwoFactorSetup(null)
      setTwoFactorCode('')
      setMessage('2FA 已关闭')
    } catch (err) {
      handleActionError(err, '关闭 2FA 失败')
    }
  }

  function handleActionError(err: unknown, fallback: string) {
    setSavingCloudConfirmed(false)
    setMessage('')
    setError(formatError(err, fallback))
  }

  const cloudKeyReady = Boolean(config?.cloudApiKeySet || config?.cloudBananaApiKeySet)
  const hasPendingUpstreamKey = Boolean(apiKey.trim() || bananaApiKey.trim())
  const pendingCloudKey = Boolean((apiKey.trim() && saveApiKeyToCloud) || (bananaApiKey.trim() && saveBananaKeyToCloud))
  const saveCue = hasPendingUpstreamKey
    ? pendingCloudKey
      ? '会先确认云端风险；确认后保存 Key 路径和默认生成设置。'
      : '会保存到当前浏览器；开发者 Bearer Key 仍需云端上游 Key。'
    : '会保存默认数量、并发和图床偏好。'

  return (
    <section className="settings-flow-panel">
      <form onSubmit={submit} className="settings-flow-form">
        <section className="settings-card security-card">
          <div className="section-title">
            <span>账号安全</span>
            <small>2FA</small>
          </div>
          <p className="muted">如果选择把 Key 上传到云端，强烈建议先开启 2FA。即使账号密码泄露，攻击者还需要一次性验证码才能登录。</p>
          <div className="settings-key-actions">
            <div className={`status-line ${twoFactorEnabled ? 'ready' : 'missing'}`}>2FA：{twoFactorEnabled ? '已开启' : '未开启'}</div>
            {twoFactorEnabled ? (
              <button type="button" onClick={() => setTwoFactorSetup(null)}>管理 2FA</button>
            ) : (
              <button type="button" onClick={() => void startTwoFactorSetup()}>开启 2FA</button>
            )}
          </div>
          {twoFactorSetup ? (
            <div className="two-factor-setup">
              <div className="two-factor-qr">
                <QRCodeSVG value={twoFactorSetup.otpauthUrl} size={168} marginSize={2} level="M" title="2FA 二维码" />
                <span>扫码添加到验证器</span>
              </div>
              <div className="two-factor-manual">
                <span>手动录入密钥</span>
                <code>{twoFactorSetup.secret}</code>
                <a href={twoFactorSetup.otpauthUrl}>打开验证器链接</a>
                <input inputMode="numeric" value={twoFactorCode} onChange={(e) => setTwoFactorCode(e.target.value)} placeholder="输入 6 位验证码" />
                <button type="button" className="primary" onClick={() => void confirmTwoFactor()}>验证并开启</button>
              </div>
            </div>
          ) : twoFactorEnabled ? (
            <div className="two-factor-setup compact">
              <input inputMode="numeric" value={twoFactorCode} onChange={(e) => setTwoFactorCode(e.target.value)} placeholder="输入 6 位验证码后关闭 2FA" />
              <button type="button" onClick={() => void turnOffTwoFactor()}>关闭 2FA</button>
            </div>
          ) : null}
        </section>

        <section className="settings-card key-card">
          <div className="section-title">
            <span>codex-key</span>
            <small>Image-2 / 提示词助手</small>
          </div>
          <p className="muted">默认只保存在当前浏览器本地。勾选上传到云端后，其他设备登录同一账号也能使用，但账号泄露时 Key 有被使用或窃取的风险。</p>
          <div className="settings-key-actions">
            <div className={`status-line ${config?.apiKeySet ? 'ready' : 'missing'}`}>
              当前：{config?.apiKeySet ? `已设置 ${config.apiKeyPreview}（${sourceLabel(config.apiKeySource)}）` : '未设置'}
            </div>
            <button type="button" disabled={!config?.localApiKeySet} onClick={() => void clearLocalKey('apiKey')}>清除本地 Key</button>
            <button type="button" disabled={!config?.cloudApiKeySet} onClick={() => void clearCloudKey('apiKey')}>清除云端 Key</button>
          </div>
          <input type="password" value={apiKey} onChange={(e) => setApiKey(e.target.value)} placeholder="填写 codex-key" spellCheck={false} autoComplete="off" />
          <label className="check-row cloud-key-check">
            <input type="checkbox" checked={saveApiKeyToCloud} onChange={(e) => setSaveApiKeyToCloud(e.target.checked)} />
            <span>同时上传到云端，用于多设备和开发者 Key</span>
          </label>
          <div className={`settings-path-note ${saveApiKeyToCloud ? 'ready' : ''}`}>
            <strong>保存路径：{saveApiKeyToCloud ? '当前浏览器 + 云端账号' : '仅当前浏览器'}</strong>
            <span>{saveApiKeyToCloud ? '点击底部“保存设置”后才会上传；云端 Key 可供 Bearer/SDK 请求使用。' : '点击底部“保存设置”后只写入本机浏览器；换设备或生成开发者 Key 需要云端 Key。'}</span>
          </div>
        </section>

        <section className="settings-card banana-key-card">
          <div className="section-title">
            <span>Banana 分组 Key</span>
            <small>单独 apikey</small>
          </div>
          <p className="muted">默认只保存在当前浏览器本地。勾选上传到云端后，其他设备登录同一账号也能使用；账号泄露时同样存在 Key 被使用或窃取的风险。</p>
          <div className="settings-key-actions">
            <div className={`status-line ${config?.bananaApiKeySet ? 'ready' : 'missing'}`}>
              当前：{config?.bananaApiKeySet ? `已设置 ${config.bananaApiKeyPreview}（${sourceLabel(config.bananaApiKeySource)}）` : '未设置'}
            </div>
            <button type="button" disabled={!config?.localBananaApiKeySet} onClick={() => void clearLocalKey('bananaApiKey')}>清除本地 Key</button>
            <button type="button" disabled={!config?.cloudBananaApiKeySet} onClick={() => void clearCloudKey('bananaApiKey')}>清除云端 Key</button>
          </div>
          <input type="password" value={bananaApiKey} onChange={(e) => setBananaApiKey(e.target.value)} placeholder="填写 banana 分组 API Key" spellCheck={false} autoComplete="off" />
          <label className="check-row cloud-key-check">
            <input type="checkbox" checked={saveBananaKeyToCloud} onChange={(e) => setSaveBananaKeyToCloud(e.target.checked)} />
            <span>同时上传到云端，用于多设备和开发者 Key</span>
          </label>
          <div className={`settings-path-note ${saveBananaKeyToCloud ? 'ready' : ''}`}>
            <strong>保存路径：{saveBananaKeyToCloud ? '当前浏览器 + 云端账号' : '仅当前浏览器'}</strong>
            <span>{saveBananaKeyToCloud ? '点击底部“保存设置”后才会上传；云端 Key 可供 Bearer/SDK 请求使用。' : '点击底部“保存设置”后只写入本机浏览器；开发者 Key 无法读取本地浏览器 Key。'}</span>
          </div>
        </section>

        <section className="settings-card developer-key-card">
          <div className="section-title">
            <span>开发者 API Key</span>
            <small>Bearer / SDK</small>
          </div>
          <p className="muted">Bearer/SDK 请求由云端 worker 执行，需先保存至少一个云端上游 Key；浏览器本地 Key 云端拿不到。</p>
          <div className={`settings-path-note ${cloudKeyReady ? 'ready' : 'missing'}`}>
            <strong>{cloudKeyReady ? '云端上游 Key 已就绪' : '生成前先保存云端上游 Key'}</strong>
            <span>{cloudKeyReady ? '新 Secret 只显示一次，生成后请立即保存。' : '在上方任一上游 Key 填写内容，勾选“同时上传到云端”，点底部“保存设置”后再生成。'}</span>
          </div>
          <div className="developer-key-create">
            <input value={developerKeyName} onChange={(e) => setDeveloperKeyName(e.target.value)} placeholder="API Key 名称" />
            <button type="button" className="primary" disabled={!cloudKeyReady} onClick={() => void createDeveloperKey()}>生成 Bearer Key</button>
          </div>
          {developerSecret ? (
            <div className="developer-secret-box">
              <span>Secret</span>
              <code>{developerSecret}</code>
            </div>
          ) : null}
          <div className="developer-key-list">
            {developerKeys.length ? developerKeys.map((item) => (
              <div className="developer-key-item" key={item.id}>
                <div>
                  <strong>{item.name}</strong>
                  <small>{item.prefix} · 创建于 {formatDateTime(item.createdAt)}{item.lastUsedAt ? ` · 最近使用 ${formatDateTime(item.lastUsedAt)}` : ''}</small>
                </div>
                <button type="button" onClick={() => void removeDeveloperKey(item.id)}>删除</button>
              </div>
            )) : <small className="muted">暂无开发者 API Key</small>}
          </div>
        </section>

        <section className="settings-card defaults-card">
          <div className="section-title">
            <span>默认生成设置</span>
            <small>提交任务时可覆盖</small>
          </div>
          <label className="field">
            <span>默认数量</span>
            <input type="number" min={1} max={12} value={defaultCount} onChange={(e) => setDefaultCount(readNumberInput(e.target.value))} />
          </label>
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
          <small className="muted">自动上传可关闭；关闭后仍可在结果页手动点击上传图床。PiXhost 单张最大 10MB。</small>
        </section>

        <div className="settings-submit-row">
          <button type="submit" className="primary">保存设置</button>
          <div className="settings-submit-copy" aria-live="polite">
            <strong>保存设置后生效</strong>
            <small className="settings-save-cue">{saveCue}</small>
            {message ? <small className="ok">{message}</small> : null}
            {error ? <small className="error">{error}</small> : null}
          </div>
        </div>
      </form>

      {showCloudWarning ? (
        <div className="cloud-warning-mask" role="dialog" aria-modal="true">
          <div className="cloud-warning-dialog">
            <h3>确认上传 Key 到云端？</h3>
            <p>上传后 Key 会保存在服务器账号空间中，方便多设备使用。但如果账号密码泄露，攻击者可能使用或窃取这些 Key。</p>
            <p>建议先开启 2FA，再上传云端 Key。</p>
            <div className="cloud-warning-actions">
              <button type="button" onClick={() => { setShowCloudWarning(false); if (!twoFactorEnabled) void startTwoFactorSetup() }}>先开启 2FA</button>
              <button type="button" onClick={() => { setSaveApiKeyToCloud(false); setSaveBananaKeyToCloud(false); void saveSettings(true, true).catch((err) => handleActionError(err, '保存设置失败')) }}>仅本地保存</button>
              <button type="button" className="primary" disabled={savingCloudConfirmed} onClick={() => { setSavingCloudConfirmed(true); void saveSettings(true).catch((err) => handleActionError(err, '保存设置失败')) }}>我了解风险，上传云端</button>
            </div>
          </div>
        </div>
      ) : null}
    </section>
  )
}

function sourceLabel(source?: 'local' | 'cloud' | 'none') {
  if (source === 'cloud') return '云端'
  if (source === 'local') return '本地'
  return '未设置'
}

function readNumberInput(value: string): NumericInputValue {
  return value === '' ? '' : Number(value)
}

function numericOrDefault(value: NumericInputValue, fallback: number) {
  return value === '' ? fallback : value
}


function formatDateTime(value?: string) {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString()
}
