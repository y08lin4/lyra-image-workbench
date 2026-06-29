import { FormEvent, useMemo, useRef, useState } from 'react'
import { confirmAgentRound, createAgentSession, sendAgentMessage } from '../api/agents'
import type { AgentPlan, AgentRound, AgentSession, ConfirmAgentRoundResponse } from '../api/contracts/agents'
import type { ModelProvider, Task } from '../types'
import { DEFAULT_IMAGE2_MODEL, IMAGE2_PROVIDER } from '../lib/models'
import './AgentPage.css'

type AgentRole = 'assistant' | 'user'
type AgentMessageKind = 'question' | 'draft' | 'prompt' | 'note' | 'task'

type AgentChatMessage = {
  id: string
  role: AgentRole
  content: string
  kind?: AgentMessageKind
  createdAt: string
  plan?: AgentPlan
}

export type AgentPromptPayload = {
  prompt: string
  source: 'local' | 'session'
  sessionId?: string
  provider: ModelProvider
  model: string
}

type AgentTaskBackflowPayload = ConfirmAgentRoundResponse | Task | Task[]

export type AgentPageProps = {
  initialPrompt?: string
  provider?: ModelProvider
  model?: string
  onCopyPrompt?: (payload: AgentPromptPayload) => void
  onTaskConfirmed?: (payload: AgentTaskBackflowPayload) => void
  onTaskCreated?: (payload: AgentTaskBackflowPayload) => void
  onConfirmTask?: (payload: AgentTaskBackflowPayload) => void
  onConfirmedTasks?: (payload: AgentTaskBackflowPayload) => void
}

const openingMessages: AgentChatMessage[] = [
  {
    id: 'agent-opening',
    role: 'assistant',
    kind: 'question',
    createdAt: new Date(0).toISOString(),
    content: '发一句想法，我会先规划画面，再创建生成任务。',
  },
]

const quickReplies = ['更写实', '电影感光影', '商品海报', '竖图 9:16', '不要文字', '保留主体但换场景']

export function AgentPage({
  initialPrompt = '',
  provider = IMAGE2_PROVIDER,
  model = DEFAULT_IMAGE2_MODEL,
  onCopyPrompt,
  onTaskConfirmed,
  onTaskCreated,
  onConfirmTask,
  onConfirmedTasks,
}: AgentPageProps) {
  const [messages, setMessages] = useState<AgentChatMessage[]>(() => {
    if (!initialPrompt.trim()) return openingMessages
    return [...openingMessages, makeMessage('user', initialPrompt.trim())]
  })
  const [input, setInput] = useState('')
  const [session, setSession] = useState<AgentSession | null>(null)
  const [activeRound, setActiveRound] = useState<AgentRound | null>(null)
  const [loading, setLoading] = useState(false)
  const [confirming, setConfirming] = useState(false)
  const [status, setStatus] = useState('')
  const [error, setError] = useState('')
  const composerRef = useRef<HTMLTextAreaElement | null>(null)

  const userInputs = useMemo(() => messages.filter((item) => item.role === 'user').map((item) => item.content), [messages])
  const activePlan = activeRound?.plan || latestPlan(session)
  const promptPayload = useMemo<AgentPromptPayload>(() => ({
    prompt: activePlan?.generationPrompt?.trim() || buildLocalPrompt(userInputs),
    source: activePlan ? 'session' : 'local',
    sessionId: session?.id,
    provider,
    model,
  }), [activePlan, model, provider, session?.id, userInputs])
  const canUsePrompt = promptPayload.prompt.trim().length > 0
  const canConfirm = Boolean(session?.id && activeRound?.id && activeRound.plan && !confirming)

  async function submitMessage(event?: FormEvent) {
    event?.preventDefault()
    const text = input.trim()
    if (!text || loading || confirming) return
    setInput('')
    setStatus('')
    setError('')
    setMessages((current) => [...current, makeMessage('user', text)])
    setLoading(true)
    try {
      const nextSession = session || await createAgentSession({ title: titleFromText(text) })
      if (!session) setSession(nextSession)
      const response = await sendAgentMessage(nextSession.id, {
        content: text,
        provider,
        model,
        ratio: 'auto',
        skipQuestions: true,
      })
      setSession(response.session)
      if (response.round) {
        setActiveRound(response.round)
        setMessages((current) => [...current, roundToMessage(response.round)])
        setStatus(response.round.plan ? '计划已生成，确认后创建任务。' : 'Agent 已回复。')
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Agent 规划失败')
    } finally {
      setLoading(false)
    }
  }

  async function confirmTask() {
    if (!session?.id || !activeRound?.id || !activeRound.plan || confirming) return
    setConfirming(true)
    setStatus('')
    setError('')
    try {
      const plan = activeRound.plan
      const params = plan.parameters
      const response = await confirmAgentRound(session.id, activeRound.id, {
        provider: params.provider || provider,
        model: params.model || model,
        ratio: params.ratio || 'auto',
        resolution: params.resolution || 'standard',
        quality: params.quality || 'auto',
        outputFormat: params.outputFormat || 'png',
        count: params.count || 1,
        concurrency: params.concurrency || 1,
        uploadIds: uploadIDsFromPlan(plan),
      })
      if (response.session) setSession(response.session)
      if (response.round) setActiveRound(response.round)
      const callbackPayload = response as AgentTaskBackflowPayload
      onTaskConfirmed?.(callbackPayload)
      onTaskCreated?.(callbackPayload)
      onConfirmTask?.(callbackPayload)
      onConfirmedTasks?.(callbackPayload)
      const taskId = response.taskIds?.[0] || response.task?.id || response.job?.id
      setMessages((current) => [...current, makeMessage('assistant', taskId ? `任务已创建：${taskId}` : '任务已创建。', 'task')])
      setStatus('任务已创建，已进入结果历史。')
    } catch (err) {
      setError(err instanceof Error ? err.message : '创建任务失败')
    } finally {
      setConfirming(false)
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
    setActiveRound(null)
    setStatus('')
    setError('')
  }

  return (
    <section className="agent-page" aria-label="Agent 创作模块">
      <header className="agent-header">
        <div>
          <p className="eyebrow">Agent Studio</p>
          <h1>Agent 创作</h1>
        </div>
        <button type="button" onClick={resetChat}>新对话</button>
      </header>

      <div className="agent-layout">
        <main className="agent-thread" aria-live="polite">
          {messages.map((message) => (
            <article key={message.id} className={`agent-message from-${message.role} ${message.kind ? `is-${message.kind}` : ''}`}>
              <div className="agent-avatar">{message.role === 'assistant' ? 'Ly' : '你'}</div>
              <div className="agent-bubble">
                <span>{message.role === 'assistant' ? messageKindLabel(message.kind) : '你的需求'}</span>
                {message.plan ? (
                  <details className="agent-thinking">
                    <summary>思考过程</summary>
                    <AgentPlanSummary plan={message.plan} />
                    {message.plan.sceneBrief ? <p className="agent-thinking-note">{message.plan.sceneBrief}</p> : null}
                  </details>
                ) : null}
                <p className={message.kind === 'prompt' ? 'agent-final-prompt' : undefined}>{message.content}</p>
              </div>
            </article>
          ))}
          {loading ? (
            <article className="agent-message from-assistant is-thinking">
              <div className="agent-avatar">Ly</div>
              <div className="agent-bubble">
                <span>思考中</span>
                <div className="agent-thinking-visual" aria-hidden="true"><i /><i /><i /></div>
                <p>正在拆解主体、场景、风格和光影，稍等一下。</p>
              </div>
            </article>
          ) : null}
        </main>

        <aside className="agent-result" aria-label="Agent 思考过程">
          <div className="agent-result-head">
            <div>
              <span>思考过程</span>
              <strong>{activePlan?.title || '等待规划'}</strong>
            </div>
            <button type="button" onClick={confirmTask} disabled={!canConfirm}>{confirming ? '创建中...' : '确认生成'}</button>
          </div>
          <div className={`agent-prompt-card ${canUsePrompt ? '' : 'is-empty'}`} aria-label="Agent 生成计划">
            {loading || activePlan ? (
              <AgentThinkingPanel plan={activePlan || null} loading={loading} />
            ) : (
              <p>等待描述。</p>
            )}
          </div>
          <div className="agent-action-row">
            <button type="button" onClick={copyPrompt} disabled={!canUsePrompt}>复制提示词</button>
            <button type="button" onClick={() => void submitMessage()} disabled={loading || confirming || !input.trim()}>{loading ? '规划中...' : '发送规划'}</button>
            <button type="button" className="primary" onClick={confirmTask} disabled={!canConfirm}>{confirming ? '创建中...' : '确认生成'}</button>
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
          <span>{error || status || '就绪'}</span>
          <button type="submit" className="primary" disabled={loading || confirming || !input.trim()}>{loading ? '规划中...' : '发送'}</button>
        </div>
      </form>
    </section>
  )
}

function AgentPlanSummary({ plan }: { plan: AgentPlan }) {
  const items = [
    ['主体', plan.visualPlan.subject],
    ['场景', plan.visualPlan.environment],
    ['风格', plan.visualPlan.style],
    ['光影', plan.visualPlan.lighting],
    ['比例', plan.parameters.ratio],
    ['数量', String(plan.parameters.count || 1)],
  ].filter(([, value]) => value)
  return (
    <div className="agent-plan-summary">
      {items.map(([label, value]) => (
        <span key={label}><b>{label}</b>{value}</span>
      ))}
    </div>
  )
}

function AgentThinkingPanel({ plan, loading }: { plan: AgentPlan | null; loading: boolean }) {
  if (loading) {
    return (
      <div className="agent-side-thinking is-loading">
        <div className="agent-thinking-orbit" aria-hidden="true"><i /><i /><i /></div>
        <strong>正在组织画面结构</strong>
        <p>Agent 会先判断主体和参考关系，再把结果整理成可直接生成的新提示词。</p>
      </div>
    )
  }
  if (!plan) return <p>等待描述。</p>
  return (
    <details className="agent-side-thinking" open>
      <summary>思考过程</summary>
      <AgentPlanSummary plan={plan} />
      <ol>
        <li>提取用户意图，确认主体、场景和最终用途。</li>
        <li>合并风格、光影、比例和质量参数。</li>
        <li>把可执行的新提示词发送到左侧对话。</li>
      </ol>
    </details>
  )
}

function makeMessage(role: AgentRole, content: string, kind: AgentMessageKind = 'note', plan?: AgentPlan): AgentChatMessage {
  return {
    id: `${role}-${Date.now()}-${Math.random().toString(36).slice(2)}`,
    role,
    content,
    kind,
    createdAt: new Date().toISOString(),
    plan,
  }
}

function messageKindLabel(kind?: AgentMessageKind) {
  if (kind === 'prompt') return '新提示词'
  if (kind === 'draft') return '需求草稿'
  if (kind === 'task') return '任务'
  if (kind === 'question') return 'Agent'
  return 'Agent'
}

function roundToMessage(round: AgentRound): AgentChatMessage {
  if (round.question) return makeMessage('assistant', round.question, 'question')
  if (round.plan) {
    const prompt = round.plan.generationPrompt?.trim() || round.plan.sceneBrief?.trim() || '计划已生成。'
    return makeMessage('assistant', `这是我整理好的新提示词：

${prompt}`, 'prompt', round.plan)
  }
  return makeMessage('assistant', round.error || 'Agent 已处理。')
}

function latestPlan(session: AgentSession | null) {
  if (!session?.rounds.length) return null
  for (let i = session.rounds.length - 1; i >= 0; i -= 1) {
    const plan = session.rounds[i]?.plan
    if (plan) return plan
  }
  return null
}

function buildLocalPrompt(inputs: string[]) {
  const brief = inputs.map((item) => item.trim()).filter(Boolean).join('；')
  if (!brief) return ''
  return `${brief}\n画面主体清晰，构图稳定，光影自然，细节完整。`
}

function uploadIDsFromPlan(plan: AgentPlan) {
  const seen = new Set<string>()
  const ids: string[] = []
  for (const usage of plan.referenceUsages || []) {
    const id = usage.uploadId?.trim()
    if (!id || seen.has(id) || usage.usage === 'ignore') continue
    seen.add(id)
    ids.push(id)
  }
  return ids
}

function titleFromText(text: string) {
  return text.trim().replace(/\s+/g, ' ').slice(0, 28) || 'Agent 创作会话'
}
