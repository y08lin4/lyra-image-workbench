import type { CanvasConnection, ReferenceRole } from './types'

export type ConnectionPromptItem = {
  id: string
  isReference: boolean
  type?: 'image' | 'text'
  name?: string
  role?: ReferenceRole
  text?: string
  naturalWidth?: number
  naturalHeight?: number
  width?: number
  height?: number
}

export const DEFAULT_CONNECTION_LABEL = '关系'

const ROLE_LABELS: Record<ReferenceRole, string> = {
  reference: '参考',
  subject: '主体',
  style: '风格',
}

export function normalizeConnectionLabel(label?: string | null) {
  const trimmed = label?.trim()
  return trimmed || DEFAULT_CONNECTION_LABEL
}

export function connectionLabel(connection: Pick<CanvasConnection, 'label' | 'text'>) {
  return normalizeConnectionLabel(connection.label || connection.text)
}

export function buildCanvasContextPromptLines(items: readonly ConnectionPromptItem[], connections: readonly CanvasConnection[]) {
  const lines: string[] = []
  const referenceLines = buildReferenceImagePromptLines(items)
  const textLines = buildTextItemPromptLines(items)
  const connectionLines = buildConnectionRelationPromptLines(connections, items)

  if (referenceLines.length) lines.push('画布参考图：', ...referenceLines)
  if (textLines.length) lines.push('画布文字：', ...textLines)
  if (connectionLines.length) lines.push('画布连线：', ...connectionLines)
  return lines
}

export function appendCanvasContextPrompt(prompt: string, items: readonly ConnectionPromptItem[], connections: readonly CanvasConnection[]) {
  return appendPromptLines(prompt, buildCanvasContextPromptLines(items, connections))
}

export function buildConnectionRelationPromptLines(connections: readonly CanvasConnection[], items: readonly ConnectionPromptItem[]) {
  const references = referencePromptItems(items)
  const textItems = textPromptItems(items)
  const referenceIndexById = new Map(references.map((item, index) => [item.id, index + 1]))
  const textIndexById = new Map(textItems.map((item, index) => [item.id, index + 1]))

  return connections.map((connection) => {
    const from = items.find((item) => item.id === connection.fromId)
    const to = items.find((item) => item.id === connection.toId)
    if (!from || !to) return ''

    const label = connectionLabel(connection)
    const fromLabel = describePromptItem(from, referenceIndexById, textIndexById)
    const toLabel = describePromptItem(to, referenceIndexById, textIndexById)
    return `- ${fromLabel} -> ${toLabel}：${label}。请按箭头理解这条画布关系；连接文字“${label}”是两端素材或文字之间的语义标注。`
  }).filter((line): line is string => Boolean(line))
}

export function appendConnectionRelationPrompt(prompt: string, connections: readonly CanvasConnection[], items: readonly ConnectionPromptItem[]) {
  return appendPromptLines(prompt, buildConnectionRelationPromptLines(connections, items))
}

export function appendPromptLines(prompt: string, lines: readonly string[]) {
  return lines.reduce((current, line) => appendPromptLine(current, line), prompt)
}

export function appendPromptLine(prompt: string, line: string) {
  const current = prompt.trimEnd()
  return current ? `${current}\n${line}` : line
}

function buildReferenceImagePromptLines(items: readonly ConnectionPromptItem[]) {
  return referencePromptItems(items).map((item, index) => {
    const role = item.role ? ROLE_LABELS[item.role] : '参考'
    const name = item.name ? `「${item.name}」` : ''
    const naturalSize = imageSizeText('原始尺寸', item.naturalWidth, item.naturalHeight)
    const displaySize = imageSizeText('画布显示', item.width, item.height)
    return `- @${index + 1}：${role}${name}${naturalSize}${displaySize}`
  })
}

function buildTextItemPromptLines(items: readonly ConnectionPromptItem[]) {
  return textPromptItems(items)
    .map((item, index) => {
      const text = compactText(item.text)
      if (!text) return ''
      const name = item.name ? `（${item.name}）` : ''
      return `- 文字块${index + 1}${name}：${text}`
    })
    .filter((line): line is string => Boolean(line))
}

function referencePromptItems(items: readonly ConnectionPromptItem[]) {
  return items.filter((item) => item.type !== 'text' && item.isReference)
}

function textPromptItems(items: readonly ConnectionPromptItem[]) {
  return items.filter((item) => item.type === 'text')
}

function describePromptItem(item: ConnectionPromptItem, referenceIndexById: Map<string, number>, textIndexById: Map<string, number>) {
  if (item.type === 'text') {
    const index = textIndexById.get(item.id)
    const text = compactText(item.text)
    return `文字块${index || ''}${text ? `「${text}」` : ''}`
  }

  const referenceIndex = referenceIndexById.get(item.id)
  const role = item.role ? ROLE_LABELS[item.role] : '参考'
  const name = item.name ? `「${item.name}」` : ''
  return referenceIndex ? `@${referenceIndex}（${role}${name}）` : `图片${name || ` ${item.id}`}`
}

function imageSizeText(label: string, width?: number, height?: number) {
  const roundedWidth = Number.isFinite(width) ? Math.round(Number(width)) : 0
  const roundedHeight = Number.isFinite(height) ? Math.round(Number(height)) : 0
  return roundedWidth > 0 && roundedHeight > 0 ? `，${label} ${roundedWidth}x${roundedHeight}` : ''
}

function compactText(value?: string) {
  return value?.trim().replace(/\s+/g, ' ').slice(0, 240) || ''
}
