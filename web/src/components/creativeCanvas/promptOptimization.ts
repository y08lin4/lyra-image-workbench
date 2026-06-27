import { createPromptSession, refinePromptSession } from '../../api/promptTools'
import { IMAGE2_PROVIDER } from '../../lib/models'
import type { ModelProvider, PromptSession } from '../../types'

const PROMPT_OPTIMIZATION_MODEL = 'gpt-5.5'
const PROMPT_OPTIMIZATION_PROVIDER: ModelProvider = IMAGE2_PROVIDER
const PROMPT_OPTIMIZATION_MESSAGE = '请把这段画布文字优化成一段可直接用于图片生成的中文提示词。保留核心意图，补足主体、构图、光影、材质、风格和画面质量要求，输出自然完整的一段提示词。'

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
