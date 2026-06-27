import { inspirationSteps, type InspirationStepId } from './constants'

export type InspirationAnswers = Record<InspirationStepId, string>

export type InspirationChatMessage = {
  id: string
  role: 'assistant' | 'user'
  label?: string
  content: string
}

export function buildInspirationBrief(answers: InspirationAnswers, extraSeed: string) {
  const lines = inspirationSteps
    .map((stepItem) => {
      const value = answers[stepItem.id]?.trim()
      return value ? `${stepItem.label}：${value}` : ''
    })
    .filter(Boolean)
  if (extraSeed.trim()) lines.push(`补充方向：${extraSeed.trim()}`)
  return lines.join('\n')
}

export function buildInspirationMessages(answers: InspirationAnswers, stepIndex: number): InspirationChatMessage[] {
  const messages: InspirationChatMessage[] = []
  const maxStepIndex = Math.max(0, Math.min(stepIndex, inspirationSteps.length - 1))
  messages.push({
    id: `assistant-${inspirationSteps[0].id}`,
    role: 'assistant',
    content: inspirationSteps[0].question,
  })
  for (let index = 0; index < inspirationSteps.length; index += 1) {
    const stepItem = inspirationSteps[index]
    const answer = answers[stepItem.id]?.trim()
    if (answer) {
      messages.push({ id: `user-${stepItem.id}`, role: 'user', label: stepItem.label, content: answer })
    }
    const nextStep = inspirationSteps[index + 1]
    if (!nextStep) continue
    const shouldShowNextQuestion =
      Boolean(answer) &&
      (index < maxStepIndex || Boolean(answers[nextStep.id]?.trim()))
    if (shouldShowNextQuestion) {
      messages.push({ id: `assistant-${nextStep.id}`, role: 'assistant', content: nextStep.question })
    }
    if (index >= maxStepIndex && !answer) break
  }
  if (inspirationSteps.every((stepItem) => answers[stepItem.id]?.trim())) {
    messages.push({
      id: 'assistant-ready',
      role: 'assistant',
      content: '信息够了。我可以把这些回答整理成可直接用于生图的完整提示词。',
    })
  }
  return messages
}

export function normalizeRandom(value: string) {
  return value === '随机' ? '' : value
}
