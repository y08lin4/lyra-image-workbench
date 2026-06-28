import { emptyInspirationSkipped, inspirationSteps, type InspirationSkipState, type InspirationStepId } from './constants'

export type InspirationAnswers = Record<InspirationStepId, string>

export type InspirationChatMessage = {
  id: string
  role: 'assistant' | 'user'
  label?: string
  content: string
}

const skippedAnswerText = '已跳过，交给模型自由判断。'
const skippedBriefText = '已跳过（不指定，交给模型自由判断）'

export function isInspirationStepAnswered(answers: InspirationAnswers, id: InspirationStepId) {
  return Boolean(answers[id]?.trim())
}

export function isInspirationStepSkipped(skipped: InspirationSkipState, id: InspirationStepId) {
  return Boolean(skipped[id])
}

export function isInspirationStepComplete(answers: InspirationAnswers, skipped: InspirationSkipState, id: InspirationStepId) {
  return isInspirationStepAnswered(answers, id) || isInspirationStepSkipped(skipped, id)
}

export function countInspirationProgress(answers: InspirationAnswers, skipped: InspirationSkipState) {
  return inspirationSteps.filter((stepItem) => isInspirationStepComplete(answers, skipped, stepItem.id)).length
}

export function buildInspirationBrief(answers: InspirationAnswers, extraSeed: string, skipped: InspirationSkipState = emptyInspirationSkipped) {
  const lines = inspirationSteps
    .map((stepItem) => {
      const value = answers[stepItem.id]?.trim()
      if (value) return `${stepItem.label}：${value}`
      if (skipped[stepItem.id]) return `${stepItem.label}：${skippedBriefText}`
      return ''
    })
    .filter(Boolean)
  if (extraSeed.trim()) lines.push(`补充方向：${extraSeed.trim()}`)
  return lines.join('\n')
}

export function buildInspirationMessages(answers: InspirationAnswers, stepIndex: number, skipped: InspirationSkipState = emptyInspirationSkipped): InspirationChatMessage[] {
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
    const skippedStep = skipped[stepItem.id]
    const completed = Boolean(answer) || Boolean(skippedStep)
    if (answer || skippedStep) {
      messages.push({ id: `user-${stepItem.id}`, role: 'user', label: stepItem.label, content: answer || skippedAnswerText })
    }
    const nextStep = inspirationSteps[index + 1]
    if (!nextStep) continue
    const shouldShowNextQuestion =
      completed &&
      (index < maxStepIndex || isInspirationStepComplete(answers, skipped, nextStep.id))
    if (shouldShowNextQuestion) {
      messages.push({ id: `assistant-${nextStep.id}`, role: 'assistant', content: nextStep.question })
    }
    if (index >= maxStepIndex && !completed) break
  }
  if (inspirationSteps.every((stepItem) => isInspirationStepComplete(answers, skipped, stepItem.id))) {
    messages.push({
      id: 'assistant-ready',
      role: 'assistant',
      content: '信息够了。我会把已回答和已跳过的项一起整理成可直接用于生图的完整提示词。',
    })
  }
  return messages
}

export function normalizeRandom(value: string) {
  return value === '随机' ? '' : value
}
