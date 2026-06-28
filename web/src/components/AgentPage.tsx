import { FormEvent, useMemo, useRef, useState } from 'react'
import type { ModelProvider, PromptSession } from '../types'
import { createPromptSession, refinePromptSession } from '../api/promptTools'
import { DEFAULT_IMAGE2_MODEL, IMAGE2_PROVIDER } from '../lib/models'
import './AgentPage.css'

type AgentRole = 'assistant' | 'user'
type AgentMessageKind = 'question' | 'draft' | 'prompt' | 'note'

export type AgentChatMessage = {
  id: string
  role: AgentRole
  content: string
  kind?: AgentMessageKind
  createdAt: string
}

export type AgentPromptPayload = {
  prompt: string
  source: 'local' | 'session'
  sessionId?: string
  provider: ModelProvider
  model: string
}

export type AgentPromptService = {
  createSession?: typeof createPromptSession
  refineSession?: typeof refinePromptSession
}

export type AgentPageProps = {
  initialPrompt?: string
  provider?: ModelProvider
  model?: string
  promptService?: AgentPromptService
  onCopyPrompt?: (payload: AgentPromptPayload) => void
  onSendToCanvas?: (payload: AgentPromptPayload) => void
  onQuickGenerate?: (payload: AgentPromptPayload) => void
}

const openingMessages: AgentChatMessage[] = [
  {
    id: 'agent-opening',
    role: 'assistant',
    kind: 'question',
    createdAt: new Date(0).toISOString(),
    content: '把你想做的图像用一句话发给我。我会追问主体、画面、风格和约束，然后整理成可直接生成的提示词。',
  },
]

const quickReplies = ['更写实', '电影感光影', '商品海报', '竖图 9:16', '不要文字', '保留主体但换场景']

export function AgentPage({
  initialPrompt = '',
  provider = IMAGE2_PROVIDER,
  model = DEFAULT_IMAGE2_MODEL,
  promptService,
  onCopyPrompt,
  onSendToCanvas,
  onQuickGenerate,
}: AgentPageProps) {
  const services = promptService || defaultPromptService
  const [messages, setMessages] = useState<AgentChatMessage[]>(() => {
    if (!initialPrompt.trim()) return openingMessages
    return [
      ...openingMessages,
      makeMessage('user', initialPrompt.trim()),
      makeMessage('assistant', buildAssistantReply([initialPrompt.trim()], ''), 'draft'),
    ]
  })
  const [input, setInput] = useState('')
  const [session, setSession] = useState<PromptSession | null>(null)
  const [loading, setLoading] = useState(false)
  const [status, setStatus] = useState('')
  const [error, setError] = useState('')
  const composerRef = useRef<HTMLTextAreaElement | null>(null)

  const userInputs = useMemo(() => messages.filter((item) => item.role === 'user').map((item) => item.content), [messages])
  const promptPayload = useMemo<AgentPromptPayload>(() => ({
    prompt: pickSessionPrompt(session) || buildLocalPrompt(userInputs),
    source: session ? 'session' : 'local',
    sessionId: session?.id,
    provider,
    model,
  }), [model, provider, session, userInputs])
  const canUsePrompt = promptPayload.prompt.trim().length > 0

  async function submitMessage(event?: FormEvent) {
    event?.preventDefault()
    const text = input.trim()
    if (!text || loading) return
    setInput('')
    setStatus('')
    setError('')
    const nextInputs = [...userInputs, text]
    const userMessage = makeMessage('user', text)
    setMessages((current) => [...current, userMessage, makeMessage('assistant', buildAssistantReply(nextInputs, pickSessionPrompt(session)), 'draft')])
  }

  async function generatePrompt() {
    if (!userInputs.length) {
      setError('先描述你想生成的图片。')
      composerRef.current?.focus()
      return
    }
    setLoading(true)
    setStatus('')
    setError('')
    try {
      if (services.createSession && services.refineSession) {
        const seed = buildLocalPrompt(userInputs)
        const nextSession = session || await services.createSession({
          title: titleFromInputs(userInputs),
          initialPrompt: seed,
          target: 'Agent 创作模块',
          provider,
          model,
        })
        const refined = await services.refineSession(nextSession.id, {
          message: buildRefineMessage(userInputs, pickSessionPrompt(nextSession)),
          currentVersionId: nextSession.activeVersionId,
          provider,
          model,
        })
        setSession(refined)
        const prompt = pickSessionPrompt(refined)
        setMessages((current) => [...current, makeMessage('assistant', prompt || buildLocalPrompt(userInputs), 'prompt')])
        setStatus('已生成整理后的提示词。')
        return
      }
      const prompt = buildLocalPrompt(userInputs)
      setMessages((current) => [...current, makeMessage('assistant', prompt, 'prompt')])
      setStatus('已用本地草稿整理提示词。')
    } catch (err) {
      setError(err instanceof Error ? err.message : '生成提示词失败')
    } finally {
      setLoading(false)
    }
  }

  async function copyPrompt() {
    if (!canUsePrompt) return
    await navigator.clipboard.writeText(promptPayload.prompt)
    onCopyPrompt?.(promptPayload)
    setStatus('提示词已复制。')
  }

  function appendQuickReply(text: string) {
    setInput((current) => current ? `${current}，${text}` : text)
    composerRef.current?.focus()
  }

  function resetChat() {
    setMessages(openingMessages)
    setInput('')
    setSession(null)
    setStatus('')
    setError('')
  }

  return (
    <section className="agent-page" aria-label="Agent 创作模块">
      <header className="agent-header">
        <div>
          <p className="eyebrow">Agent Studio</p>
          <h1>Agent 创作</h1>
          <p>用多轮对话把模糊想法整理成可生成、可复制、可送到画布的图片提示词。</p>
        </div>
        <button type="button" onClick={resetChat}>新对话</button>
      </header>

      <div className="agent-layout">
        <main className="agent-thread" aria-live="polite">
          {messages.map((message) => (
            <article key={message.id} className={`agent-message from-${message.role} ${message.kind ? `is-${message.kind}` : ''}`}>
              <div className="agent-avatar">{message.role === 'assistant' ? 'Ly' : '你'}</div>
              <div className="agent-bubble">
                <span>{message.role === 'assistant' ? messageKindLabel(message.kind) : '你的补充'}</span>
                <p>{message.content}</p>
              </div>
            </article>
          ))}
        </main>

        <aside className="agent-result" aria-label="整理后的提示词">
          <div className="agent-result-head">
            <div>
              <span>Prompt</span>
              <strong>{session ? '会话版本' : '本地草稿'}</strong>
            </div>
            <button type="button" onClick={generatePrompt} disabled={loading}>{loading ? '生成中...' : '整理提示词'}</button>
          </div>
          <textarea value={promptPayload.prompt} readOnly rows={12} aria-label="整理后的提示词内容" />
          <div className="agent-action-row">
            <button type="button" onClick={copyPrompt} disabled={!canUsePrompt}>复制提示词</button>
            <button type="button" onClick={() => onSendToCanvas?.(promptPayload)} disabled={!canUsePrompt || !onSendToCanvas}>发送到创作画布</button>
            <button type="button" className="primary" onClick={() => onQuickGenerate?.(promptPayload)} disabled={!canUsePrompt || !onQuickGenerate}>快捷生成</button>
          </div>
        </aside>
      </div>

      <form className="agent-composer" onSubmit={(event) => void submitMessage(event)}>
        <div className="agent-quick-row" aria-label="快捷补充">
          {quickReplies.map((item) => <button key={item} type="button" onClick={() => appendQuickReply(item)}>{item}</button>)}
        </div>
        <label>
          <span>继续对话</span>
          <textarea
            ref={composerRef}
            value={input}
            onChange={(event) => setInput(event.target.value)}
            placeholder="例如：主体是银色跑车，雨夜街头，低机位，霓虹反射，不要出现文字"
            rows={3}
          />
        </label>
        <div className="agent-composer-actions">
          <span>{status || error || 'Enter 换行，点击发送继续补充需求。'}</span>
          <button type="submit" className="primary" disabled={loading || !input.trim()}>发送</button>
        </div>
      </form>
    </section>
  )
}

const defaultPromptService: AgentPromptService = {
  createSession: createPromptSession,
  refineSession: refinePromptSession,
}

function makeMessage(role: AgentRole, content: string, kind: AgentMessageKind = 'note'): AgentChatMessage {
  return {
    id: `${role}-${Date.now()}-${Math.random().toString(36).slice(2)}`,
    role,
    content,
    kind,
    createdAt: new Date().toISOString(),
  }
}

function messageKindLabel(kind?: AgentMessageKind) {
  if (kind === 'prompt') return '整理后的提示词'
  if (kind === 'draft') return '需求草稿'
  if (kind === 'question') return 'Agent'
  return 'Agent'
}

function buildAssistantReply(inputs: string[], currentPrompt: string) {
  const latest = inputs[inputs.length - 1] || ''
  const missing = suggestMissingFields(inputs.join(' '))
  if (missing.length) {
    return `已记录：${latest}\n还可以补充 ${missing.join('、')}。需要时点击「整理提示词」，我会先按现有信息生成完整版本。`
  }
  if (currentPrompt) return '信息已经比较完整。可以继续微调，也可以把当前版本发送到创作画布或快捷生成。'
  return '信息已经比较完整。点击「整理提示词」生成一版结构化提示词，也可以继续补充限制条件。'
}

function suggestMissingFields(text: string) {
  const checks = [
    { label: '主体', pattern: /主体|人物|角色|产品|动物|建筑|车辆|风景/ },
    { label: '场景', pattern: /场景|室内|室外|街头|森林|城市|海边|房间|背景/ },
    { label: '风格', pattern: /风格|写实|插画|动漫|电影|摄影|海报|赛博|国风/ },
    { label: '构图/比例', pattern: /构图|近景|远景|特写|俯拍|低机位|比例|9:16|16:9|1:1/ },
  ]
  return checks.filter((item) => !item.pattern.test(text)).map((item) => item.label)
}

function buildLocalPrompt(inputs: string[]) {
  const brief = inputs.map((item) => item.trim()).filter(Boolean).join('；')
  if (!brief) return ''
  return [
    brief,
    '画面主体清晰，构图稳定，层次分明，光影自然，材质细节丰富，色彩协调，适合高质量图片生成。',
    '避免低清晰度、畸形结构、错乱文字、水印、过度噪点和无关元素。',
  ].join('\n')
}

function buildRefineMessage(inputs: string[], currentPrompt: string) {
  return [
    '请基于以下多轮需求，整理成一段可直接用于图片生成的中文正向提示词。',
    '要求：保留用户意图，补足主体、场景、构图、光影、材质、风格、质量要求；不要输出解释。',
    currentPrompt ? `当前版本：${currentPrompt}` : '',
    `用户需求：${inputs.join('；')}`,
  ].filter(Boolean).join('\n')
}

function pickSessionPrompt(session: PromptSession | null) {
  if (!session?.versions.length) return ''
  const active = session.versions.find((item) => item.id === session.activeVersionId) || session.versions[session.versions.length - 1]
  return active?.prompt.trim() || ''
}

function titleFromInputs(inputs: string[]) {
  return (inputs[0] || 'Agent 创作提示词').slice(0, 28)
}
