import type { CanvasProject } from './types'
import { buildCanvasSnapshotRequest, buildCanvasTaskDraftFromProject, canvasEdgeToTaskEdge, canvasNodeToTaskNode } from './taskAdapter'

function assert(condition: boolean, message: string): asserts condition {
  if (!condition) throw new Error(message)
}

function assertJsonEqual(actual: unknown, expected: unknown, message: string) {
  const actualJson = JSON.stringify(actual)
  const expectedJson = JSON.stringify(expected)
  assert(actualJson === expectedJson, `${message}: expected ${expectedJson}, received ${actualJson}`)
}

const project: CanvasProject = {
  id: 'cvp_demo',
  title: 'Demo canvas',
  revision: 7,
  viewport: { x: 12, y: 24, zoom: 0.8 },
  nodes: [
    {
      id: 'subject',
      type: 'image',
      name: 'Subject',
      x: 10,
      y: 20,
      width: 240,
      height: 160,
      rotation: 5,
      zIndex: 1,
      text: 'glass violin maker',
      role: 'subject',
      assetId: 'asset_subject',
      uploadId: 'upload_subject',
      resultIndex: 0,
      source: 'upload',
    },
    {
      id: 'generate',
      type: 'generation',
      name: 'Generate',
      x: 380,
      y: 30,
      width: 280,
      height: 180,
      rotation: 0,
      zIndex: 2,
      text: 'cinematic portrait',
      mode: 'image-to-image',
      provider: 'image-2',
      model: 'gpt-image-2',
      ratio: '3:4',
      count: 2,
      concurrency: 1,
    },
  ],
  edges: [
    {
      id: 'subject-generate',
      fromNodeId: 'subject',
      toNodeId: 'generate',
      role: 'subject',
      label: 'primary subject',
    },
  ],
  createdAt: '2026-06-29T00:00:00Z',
  updatedAt: '2026-06-29T00:00:00Z',
}

export function runCanvasTaskAdapterExamples() {
  assert(project.nodes[0].type === 'image', 'canvas nodes use backend type field')
  assert(project.nodes[0].x === 10 && project.nodes[0].width === 240 && project.nodes[0].rotation === 5, 'node geometry is flat backend geometry')
  assertJsonEqual(canvasNodeToTaskNode(project.nodes[0]).promptParts, ['glass violin maker'], 'adapter reads flat text prompt parts')
  assertJsonEqual(canvasNodeToTaskNode(project.nodes[0]).referenceAssetIds, ['asset_subject'], 'adapter reads flat asset ids')
  assertJsonEqual(canvasEdgeToTaskEdge(project.edges[0]), {
    id: 'subject-generate',
    fromNodeId: 'subject',
    toNodeId: 'generate',
    label: 'primary subject',
  }, 'adapter preserves backend edge endpoints')

  const draft = buildCanvasTaskDraftFromProject(project, 'generate')
  assertJsonEqual(draft.contextNodeIds, ['subject', 'generate'], 'draft follows fromNodeId/toNodeId graph')
  assert(draft.prompt === 'glass violin maker\ncinematic portrait', 'draft prompt includes source and generation text')

  const request = buildCanvasSnapshotRequest(project, 'generate')
  assert(request.revision === 7, 'snapshot request carries project revision')
  assert(request.parameters.mode === 'image-to-image', 'snapshot request carries generation mode')
  assertJsonEqual(request.uploadIds, ['upload_subject'], 'snapshot request carries reference upload ids')
  assert(request.references?.[0]?.resultIndex === 0, 'snapshot request preserves zero result index')
}

export const canvasSnapshotRequestExample = buildCanvasSnapshotRequest(project, 'generate')
