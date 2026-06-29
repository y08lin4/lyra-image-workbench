import { buildCanvasTaskDraft, type CanvasTaskEdge, type CanvasTaskNode } from './context'

function assert(condition: boolean, message: string): asserts condition {
  if (!condition) throw new Error(message)
}

function assertJsonEqual(actual: unknown, expected: unknown, message: string) {
  const actualJson = JSON.stringify(actual)
  const expectedJson = JSON.stringify(expected)
  assert(actualJson === expectedJson, `${message}: expected ${expectedJson}, received ${actualJson}`)
}

export function runBuildCanvasTaskDraftExamples() {
  const nodes: CanvasTaskNode[] = [
    {
      id: 'style',
      data: {
        promptParts: ['cinematic rim light'],
        referenceAssetIds: ['asset-style'],
        parameters: {
          provider: 'local',
          image: { ratio: '1:1', quality: 'draft' },
        },
      },
    },
    {
      id: 'subject',
      promptParts: [{ id: 'subject-main', text: 'portrait of a glass violin maker' }],
      referenceAssetIds: ['asset-subject', 'asset-style'],
    },
    {
      id: 'unrelated',
      promptParts: ['should not be included'],
      referenceAssetIds: ['asset-unrelated'],
    },
    {
      id: 'generate',
      promptParts: ['final high detail composition'],
      parameters: {
        model: 'gpt-image-2',
        image: { quality: 'high' },
      },
    },
  ]
  const edges: CanvasTaskEdge[] = [
    {
      id: 'style-to-subject',
      source: 'style',
      target: 'subject',
      label: 'applies lighting style',
      data: { parameters: { image: { ratio: '3:4' } } },
    },
    {
      id: 'subject-to-generate',
      sourceNodeId: 'subject',
      targetNodeId: 'generate',
      text: 'use as primary subject',
    },
  ]

  const draft = buildCanvasTaskDraft({ nodes, edges, generationNodeId: 'generate' })

  assertJsonEqual(draft.contextNodeIds, ['style', 'subject', 'generate'], 'context nodes follow ancestor order')
  assertJsonEqual(
    draft.promptParts.map((part) => part.text),
    ['cinematic rim light', 'portrait of a glass violin maker', 'final high detail composition'],
    'prompt parts are collected from generation ancestors',
  )
  assert(draft.prompt === 'cinematic rim light\nportrait of a glass violin maker\nfinal high detail composition', 'prompt joins parts with newlines')
  assertJsonEqual(draft.referenceAssetIds, ['asset-style', 'asset-subject'], 'reference assets are deduped in context order')
  assertJsonEqual(
    draft.edgeLabels.map((edgeLabel) => edgeLabel.label),
    ['applies lighting style', 'use as primary subject'],
    'edge labels are collected from context edges',
  )
  assertJsonEqual(
    draft.parameters,
    {
      provider: 'local',
      image: { ratio: '3:4', quality: 'high' },
      model: 'gpt-image-2',
    },
    'parameters are merged with generation node values last',
  )
  assertJsonEqual(draft.diagnostics, [], 'valid graph has no diagnostics')
}

export const canvasTaskDraftExample = buildCanvasTaskDraft({
  generationNodeId: 'generate',
  nodes: [
    { id: 'note', promptParts: ['ink wash skyline'], referenceAssetIds: ['asset-note'] },
    { id: 'generate', parameters: { count: 1 } },
  ],
  edges: [{ id: 'note-generate', fromId: 'note', toId: 'generate', label: 'sets scene' }],
})
