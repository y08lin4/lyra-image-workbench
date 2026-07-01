import type { ModelProvider, PromptRecord, PromptSession, PromptVersion } from '../../types'
import { DEFAULT_IMAGE2_MODEL, IMAGE2_PROVIDER, providerLabel } from '../../lib/models'
import { kindLabel, observationLabels, quickRefines } from './constants'

type PromptResultPanelProps = {
  record: PromptRecord | null
  session: PromptSession | null
  activeVersion: PromptVersion | null
  activeVersionId: string
  provider: ModelProvider
  refineText: string
  loading: boolean
  onVersionChange: (id: string) => void
  onProviderChange: (provider: ModelProvider) => void
  onRefineTextChange: (value: string) => void
  onQuickRefine: (value: string) => void
  onRefine: () => void
  onCopy: (prompt: string) => void
  onUse: (prompt: string, options: { provider: ModelProvider; model: string; ratio?: string }) => void
}

export function PromptResultPanel({
  record,
  session,
  activeVersion,
  activeVersionId,
  provider,
  refineText,
  loading,
  onVersionChange,
  onProviderChange,
  onRefineTextChange,
  onQuickRefine,
  onRefine,
  onCopy,
  onUse,
}: PromptResultPanelProps) {
  const prompt = activeVersion?.prompt || record?.flatPrompt || ''
  const negativePrompt = activeVersion?.negativePrompt || record?.negativePrompt || ''
  const promptRatio = activeVersion?.ratio || record?.ratio || session?.ratio || ''
  const mustKeep = activeVersion?.mustKeep?.length ? activeVersion.mustKeep : record?.mustKeep
  const title = session ? `${kindLabel(session.kind)} · ${session.title}` : record ? (record.mode === 'image-to-prompt' ? '图片还原提示词' : '提示词优化结果') : '结果预览'
  const elapsedMs = activeVersion?.elapsedMs || record?.elapsedMs || 0
  const promptModel = activeVersion?.model || record?.model || 'gpt-5.5'
  const model = DEFAULT_IMAGE2_MODEL
  const useOptions = { provider: IMAGE2_PROVIDER, model, ratio: promptRatio || undefined }
  const recentMessages = session?.messages.slice(-4) || []

  if (!prompt) {
    return (
      <aside className="prompt-result empty">
        <strong>结果预览</strong>
        <span>生成后会在这里显示，应用按钮会贴近提示词结果。</span>
      </aside>
    )
  }
  return (
    <aside className="prompt-result" aria-live="polite">
      <div className="prompt-result-title">
        <div className="prompt-result-title-main">
          <strong>{title}</strong>
          <span>{promptModel} · {elapsedMs ? `${(elapsedMs / 1000).toFixed(1)}s` : '会话'}</span>
        </div>
      </div>

      {session?.versions.length ? (
        <section className="prompt-version-list" aria-label="提示词版本">
          {session.versions.map((version) => (
            <button key={version.id} type="button" className={version.id === activeVersionId ? 'active' : ''} onClick={() => onVersionChange(version.id)}>
              V{version.index}
            </button>
          ))}
        </section>
      ) : null}

      {promptRatio ? (
        <div className="prompt-chips">
          <span>{promptRatio === 'auto' ? '自动比例' : `画面比例 ${promptRatio}`}</span>
        </div>
      ) : null}

      <section className="prompt-apply-model" aria-label="选择应用模型">
        <div className="section-title">
          <span>应用目标</span>
          <small>{providerLabel(IMAGE2_PROVIDER)} · {model}</small>
        </div>
        <div className="mode-tabs provider-tabs">
          <button type="button" className="active" onClick={() => onProviderChange(IMAGE2_PROVIDER)}>Image-2</button>
        </div>
      </section>

      <section className="prompt-output-card" aria-label="生成的正向提示词">
        <div className="prompt-output-head">
          <div className="prompt-output-label">
            <span>正向提示词</span>
            <small>确认后可直接填入生成页</small>
          </div>
          <div className="prompt-output-actions prompt-result-actions-main">
            <button type="button" onClick={() => onCopy(prompt)}>复制</button>
            <button type="button" className="primary" onClick={() => onUse(prompt, useOptions)}>应用到生成</button>
          </div>
        </div>
        <textarea value={prompt} readOnly rows={7} aria-label="正向提示词内容" />
      </section>
      {negativePrompt ? (
        <label>
          <span>负面提示词</span>
          <textarea value={negativePrompt} readOnly rows={3} />
        </label>
      ) : null}
      {mustKeep?.length ? (
        <div className="prompt-chips">
          {mustKeep.map((item) => <span key={item}>{item}</span>)}
        </div>
      ) : null}
      {activeVersion?.notes ? <p className="prompt-version-note">{activeVersion.notes}</p> : null}

      {session ? (
        <section className="prompt-refine-box">
          <div className="section-title">
            <span>继续对话修改</span>
            <small>会生成新版本，不覆盖旧版本</small>
          </div>
          <div className="prompt-quick-refines">
            {quickRefines.map((item) => <button key={item} type="button" onClick={() => onQuickRefine(item)}>{item}</button>)}
          </div>
          <textarea value={refineText} onChange={(event) => onRefineTextChange(event.target.value)} placeholder="例如：更写实一点，减少动漫感；主体换成猫；保留雨夜和霓虹灯" rows={3} />
          <button type="button" className="primary" disabled={loading} onClick={onRefine}>{loading ? '修改中...' : '生成新版本'}</button>
          {recentMessages.length ? (
            <div className="prompt-chat-thread" aria-label="最近对话">
              {recentMessages.map((item) => (
                <p key={item.id} className={item.role === 'user' ? 'from-user' : 'from-assistant'}>
                  <b>{item.role === 'user' ? '你' : '助手'}</b>
                  <span>{item.content}</span>
                </p>
              ))}
            </div>
          ) : null}
        </section>
      ) : null}

      <PromptInspector title="结构化观察" data={record?.jsonDescription} />
      <PromptInspector title="图片 Metadata / 原提示词" data={record?.metadata} />
    </aside>
  )
}

function PromptInspector({ title, data, defaultOpen = false }: { title: string; data?: Record<string, unknown>; defaultOpen?: boolean }) {
  const entries = Object.entries(data || {}).filter(([, value]) => hasMetaValue(value))
  if (!entries.length) return null
  return (
    <details className="prompt-inspector" open={defaultOpen}>
      <summary>
        <span>{title}</span>
        <small>{entries.length} 项</small>
      </summary>
      <div className="prompt-inspector-grid">
        {entries.map(([key, value]) => (
          <div key={key} className="prompt-inspector-row">
            <span className="prompt-inspector-key">{observationLabel(key)}</span>
            <pre className="prompt-inspector-value">{prettyMetaValue(value)}</pre>
          </div>
        ))}
      </div>
    </details>
  )
}

function hasMetaValue(value: unknown): boolean {
  if (value == null) return false
  if (typeof value === 'string') return value.trim().length > 0
  if (Array.isArray(value)) return value.some(hasMetaValue)
  if (typeof value === 'object') return Object.values(value as Record<string, unknown>).some(hasMetaValue)
  return true
}

function observationLabel(key: string) {
  return observationLabels[key] || key
}

function prettyMetaValue(value: unknown): string {
  if (value == null) return ''
  if (typeof value === 'string') return value
  if (typeof value === 'number' || typeof value === 'boolean') return String(value)
  if (Array.isArray(value)) {
    if (value.every((item) => ['string', 'number', 'boolean'].includes(typeof item))) {
      return value.map(String).join(' / ')
    }
  }
  try {
    return JSON.stringify(value, null, 2)
  } catch {
    return String(value)
  }
}
