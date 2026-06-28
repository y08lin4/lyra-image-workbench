import { createPromptSession, refinePromptSession } from '../../api/promptTools'
import { appendCanvasContextPrompt } from './connectionPrompt'
import type { CanvasConnection, CanvasItem } from './types'
import { IMAGE2_PROVIDER } from '../../lib/models'
import type { ModelProvider, PromptSession } from '../../types'

const PROMPT_OPTIMIZATION_MODEL = 'gpt-5.5'
const PROMPT_OPTIMIZATION_PROVIDER: ModelProvider = IMAGE2_PROVIDER
const PROMPT_OPTIMIZATION_MESSAGE = '请把这段画布文字优化成一段可直接用于图片生成的中文提示词。保留核心意图，补足主体、构图、光影、材质、风格和画面质量要求，输出自然完整的一段提示词。'
const CANVAS_PROMPT_MESSAGE = '请根据这份创作画布底稿，整理成一段可直接用于图片生成的中文提示词。必须理解 @参考图、文字块和连线关系；保留用户原意，明确主体、参考来源、构图关系、风格、质感和质量要求。只输出最终提示词，不要解释过程。'

export function buildCanvasPromptDraft(prompt: string, items: readonly CanvasItem[], connections: readonly CanvasConnection[], options: { ratio?: string } = {}) {
  const base = appendCanvasContextPrompt(prompt, items, connections).trim()
  const lines = [
    base,
    options.ratio ? `画面比例：${options.ratio}` : '',
    '请综合画布中的参考图、文字块和连线标注，生成一张完整、自然、可执行的图片。',
  ].filter(Boolean)
  return lines.join('\n')
}

export async function generateCanvasPromptFromCanvas(prompt: string, items: readonly CanvasItem[], connections: readonly CanvasConnection[], options: { ratio?: string } = {}) {
  const draft = buildCanvasPromptDraft(prompt, items, connections, options)
  if (!draft) throw new Error('画布为空，无法生成提示词。')

  const session = await createPromptSession({
    title: '画布内容生成提示词',
    initialPrompt: draft,
    ratio: options.ratio,
    target: '创作画布整体',
    provider: PROMPT_OPTIMIZATION_PROVIDER,
    model: PROMPT_OPTIMIZATION_MODEL,
  })
  const refinedSession = await refinePromptSession(session.id, {
    message: CANVAS_PROMPT_MESSAGE,
    currentVersionId: session.activeVersionId,
    provider: PROMPT_OPTIMIZATION_PROVIDER,
    model: PROMPT_OPTIMIZATION_MODEL,
  })
  return pickActivePrompt(refinedSession) || draft
}

export async function optimizeCanvasTextPrompt(text: string, options: { ratio?: string } = {}) {
  const initialPrompt = text.trim()
  if (!initialPrompt) throw new Error('文字块为空，无法优化提示词。')

  const session = await createPromptSession({
    title: '画布文字提示词优化',
    initialPrompt,
    ratio: options.ratio,
    target: '创作画布文字块',
    provider: PROMPT_OPTIMIZATION_PROVIDER,
    model: PROMPT_OPTIMIZATION_MODEL,
  })
  const refinedSession = await refinePromptSession(session.id, {
    message: PROMPT_OPTIMIZATION_MESSAGE,
    currentVersionId: session.activeVersionId,
    provider: PROMPT_OPTIMIZATION_PROVIDER,
    model: PROMPT_OPTIMIZATION_MODEL,
  })
  const optimizedPrompt = pickActivePrompt(refinedSession)
  if (!optimizedPrompt) throw new Error('提示词优化没有返回可用内容。')
  return optimizedPrompt
}

function pickActivePrompt(session: PromptSession) {
  const version = session.versions.find((item) => item.id === session.activeVersionId) || session.versions[session.versions.length - 1]
  return version?.prompt.trim() || ''
}
