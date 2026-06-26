import { useEffect, useMemo, useState, type CSSProperties, type ReactNode } from 'react'
import type { CreateTaskRequest, Mode, ModelProvider, ReferenceUpload } from '../types'
import {
  BANANA_MODEL_OPTIONS,
  BANANA_PROVIDER,
  DEFAULT_BANANA_MODEL,
  DEFAULT_IMAGE2_MODEL,
  getBananaModelForRatio,
  getBananaModelOption,
  providerLabel,
} from '../lib/models'
import {
  OUTPUT_FORMATS,
  QUALITY_LEVELS,
  RATIOS,
  RESOLUTION_TIERS,
  getImageSize,
  getOutputFormatLabel,
  getQualityLabel,
  getResolutionLabel,
} from '../lib/ratios'

type FlowNodeId = 'input' | 'prompt' | 'model' | 'spec' | 'run' | 'result'
type FlowStatus = 'ready' | 'todo' | 'active'

type FlowNode = {
  id: FlowNodeId
  index: number
  eyebrow: string
  title: string
  purpose: string
  value: string
  meta: string[]
  status: FlowStatus
  x: number
  y: number
  width: number
  height: number
  tone: string
  input?: string
  output?: string
}

type FlowEdge = {
  id: string
  from: FlowNodeId
  to: FlowNodeId
  fromOffsetY?: number
  toOffsetY?: number
  label?: string
}

type WorkflowPreset = {
  id: string
  title: string
  description: string
  mode: Mode
  provider: ModelProvider
  ratio: string
  resolution: string
  quality: string
  outputFormat: string
  prompt: string
}

export type NodeWorkflowUsePromptOptions = {
  provider: ModelProvider
  model: string
  ratio?: string
}

export type NodeWorkflowPageProps = {
  provider: ModelProvider
  bananaModel: string
  prompt?: string
  initialPrompt?: string
  onUsePrompt: (prompt: string, options: NodeWorkflowUsePromptOptions) => void
  onCreateTask?: (payload: CreateTaskRequest) => void | Promise<void>
  referenceUploads?: ReferenceUpload[]
}

const DEFAULT_PROMPT = '一张电影感产品图：透明玻璃相机放在拉丝金属桌面上，柔和侧光，浅景深。'
const CANVAS_WIDTH = 1168
const CANVAS_HEIGHT = 536

const WORKFLOW_PRESETS: WorkflowPreset[] = [
  {
    id: 'fast-text-to-image',
    title: '最短文生图',
    description: '一句提示词直接生成，适合普通用户最快跑通。',
    mode: 'text-to-image',
    provider: 'image-2',
    ratio: '1:1',
    resolution: 'standard',
    quality: 'high',
    outputFormat: 'png',
    prompt: DEFAULT_PROMPT,
  },
  {
    id: 'product-poster',
    title: '产品海报',
    description: '保留商业海报结构，默认竖版、高质量输出。',
    mode: 'text-to-image',
    provider: 'image-2',
    ratio: '2:3',
    resolution: '2k',
    quality: 'high',
    outputFormat: 'png',
    prompt: '一张高级商业产品海报，主体居中，清晰标题区，干净背景，柔和棚拍光，画面有可读留白。',
  },
  {
    id: 'reference-remix',
    title: '参考图改写',
    description: '为图生图准备的流程，先写保留点和变化目标。',
    mode: 'image-to-image',
    provider: BANANA_PROVIDER,
    ratio: 'auto',
    resolution: 'auto',
    quality: 'auto',
    outputFormat: 'auto',
    prompt: '保留参考图的构图、主体姿态和光线方向，将主题改成未来感产品摄影，画面干净、真实、细节丰富。',
  },
]

const FLOW_EDGES: FlowEdge[] = [
  { id: 'input-prompt', from: 'input', to: 'prompt', label: '整理意图' },
  { id: 'prompt-model', from: 'prompt', to: 'model', label: '选择模型', fromOffsetY: -28, toOffsetY: -16 },
  { id: 'prompt-spec', from: 'prompt', to: 'spec', label: '约束规格', fromOffsetY: 30, toOffsetY: 16 },
  { id: 'model-run', from: 'model', to: 'run', label: '模型' },
  { id: 'spec-run', from: 'spec', to: 'run', label: '参数' },
  { id: 'run-result', from: 'run', to: 'result', label: '生成' },
]

export function NodeWorkflowPage({
  provider,
  bananaModel,
  prompt,
  initialPrompt,
  onUsePrompt,
  onCreateTask,
  referenceUploads = [],
}: NodeWorkflowPageProps) {
  const [selectedNodeId, setSelectedNodeId] = useState<FlowNodeId>('prompt')
  const [selectedPresetId, setSelectedPresetId] = useState(WORKFLOW_PRESETS[0].id)
  const [mode, setMode] = useState<Mode>('text-to-image')
  const [draftPrompt, setDraftPrompt] = useState(prompt ?? initialPrompt ?? DEFAULT_PROMPT)
  const [localProvider, setLocalProvider] = useState<ModelProvider>(provider)
  const [localBananaModel, setLocalBananaModel] = useState(bananaModel || DEFAULT_BANANA_MODEL)
  const [ratio, setRatio] = useState('1:1')
  const [resolution, setResolution] = useState('standard')
  const [quality, setQuality] = useState('high')
  const [outputFormat, setOutputFormat] = useState('png')
  const [count, setCount] = useState(1)
  const [concurrency, setConcurrency] = useState(1)
  const [message, setMessage] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)

  useEffect(() => {
    setLocalProvider(provider)
  }, [provider])

  useEffect(() => {
    setLocalBananaModel(bananaModel || DEFAULT_BANANA_MODEL)
  }, [bananaModel])

  useEffect(() => {
    if (prompt !== undefined) setDraftPrompt(prompt)
  }, [prompt])

  const bananaOption = useMemo(() => getBananaModelOption(localBananaModel), [localBananaModel])
  const selectedModel = localProvider === BANANA_PROVIDER ? bananaOption.id : DEFAULT_IMAGE2_MODEL
  const effectiveRatio = localProvider === BANANA_PROVIDER ? bananaOption.ratio : ratio
  const effectiveResolution = localProvider === BANANA_PROVIDER ? bananaOption.resolution : resolution
  const effectiveQuality = localProvider === BANANA_PROVIDER ? 'auto' : quality
  const effectiveOutputFormat = localProvider === BANANA_PROVIDER ? 'auto' : outputFormat
  const imageSize = localProvider === BANANA_PROVIDER ? bananaOption.size : getImageSize(ratio, resolution)
  const trimmedPrompt = draftPrompt.trim()
  const canUsePrompt = trimmedPrompt.length > 0
  const selectedPreset = WORKFLOW_PRESETS.find((item) => item.id === selectedPresetId) ?? WORKFLOW_PRESETS[0]
  const referenceUploadIds = useMemo(() => referenceUploads.map((item) => item.id), [referenceUploads])
  const hasReferenceUploads = referenceUploadIds.length > 0

  const taskPayload = useMemo<CreateTaskRequest>(() => ({
    provider: localProvider,
    model: selectedModel,
    mode,
    prompt: trimmedPrompt,
    ratio: effectiveRatio,
    resolution: effectiveResolution,
    quality: effectiveQuality,
    outputFormat: effectiveOutputFormat,
    count,
    concurrency,
    uploadIds: mode === 'image-to-image' ? referenceUploadIds : [],
  }), [
    concurrency,
    count,
    effectiveOutputFormat,
    effectiveQuality,
    effectiveRatio,
    effectiveResolution,
    localProvider,
    mode,
    referenceUploadIds,
    selectedModel,
    trimmedPrompt,
  ])

  const flowNodes = useMemo<FlowNode[]>(() => [
    {
      id: 'input',
      index: 1,
      eyebrow: 'INPUT',
      title: mode === 'image-to-image' ? '参考图与目标' : '创作目标',
      purpose: mode === 'image-to-image' ? '准备参考图，并说明保留什么、改变什么。' : '先把用户真实想做的图说清楚。',
      value: mode === 'image-to-image' ? (hasReferenceUploads ? `已接入 ${referenceUploadIds.length} 张参考图` : '需要先在生成页上传参考图') : '纯文本生成，不需要素材',
      meta: [mode === 'image-to-image' ? (hasReferenceUploads ? `${referenceUploadIds.length} 张参考图` : '需要参考图') : '最快路径', '用户输入'],
      status: mode === 'image-to-image' ? (hasReferenceUploads ? 'ready' : 'todo') : 'ready',
      x: 34,
      y: 214,
      width: 176,
      height: 150,
      tone: '#64748b',
      output: 'intent',
    },
    {
      id: 'prompt',
      index: 2,
      eyebrow: 'PROMPT',
      title: '提示词整理',
      purpose: '把一句话变成可直接提交的生成描述。',
      value: trimmedPrompt || '还没有提示词',
      meta: [`${trimmedPrompt.length} 字符`, canUsePrompt ? '可提交' : '缺失'],
      status: canUsePrompt ? 'ready' : 'todo',
      x: 248,
      y: 150,
      width: 220,
      height: 226,
      tone: '#2563eb',
      input: 'intent',
      output: 'prompt',
    },
    {
      id: 'model',
      index: 3,
      eyebrow: 'MODEL',
      title: '模型路由',
      purpose: '决定走 Image-2 还是 Banana。',
      value: providerLabel(localProvider),
      meta: [compactModelId(selectedModel), localProvider],
      status: 'ready',
      x: 512,
      y: 70,
      width: 190,
      height: 160,
      tone: '#7c3aed',
      input: 'prompt',
      output: 'model',
    },
    {
      id: 'spec',
      index: 4,
      eyebrow: 'SPEC',
      title: '图片规格',
      purpose: '把比例、清晰度、质量和格式固定下来。',
      value: imageSize,
      meta: [effectiveRatio, effectiveResolution, effectiveQuality],
      status: 'ready',
      x: 512,
      y: 306,
      width: 190,
      height: 164,
      tone: '#0891b2',
      input: 'prompt',
      output: 'spec',
    },
    {
      id: 'run',
      index: 5,
      eyebrow: 'RUN',
      title: '创建任务',
      purpose: '把所有节点参数合并成一次后台任务。',
      value: `${count} 张图 / 并发 ${concurrency}`,
      meta: [onCreateTask ? '已接入任务队列' : '仅预览 payload', effectiveOutputFormat],
      status: canUsePrompt ? 'active' : 'todo',
      x: 744,
      y: 192,
      width: 196,
      height: 184,
      tone: '#ea580c',
      input: 'model + spec',
      output: 'task',
    },
    {
      id: 'result',
      index: 6,
      eyebrow: 'RESULT',
      title: '结果页查看',
      purpose: '任务创建后进入结果页，看状态、任务 ID 和生成图。',
      value: '提交后自动跳转结果页',
      meta: ['任务历史', '可复用'],
      status: 'todo',
      x: 982,
      y: 214,
      width: 154,
      height: 150,
      tone: '#16a34a',
      input: 'task',
    },
  ], [
    canUsePrompt,
    concurrency,
    count,
    effectiveOutputFormat,
    effectiveQuality,
    effectiveRatio,
    effectiveResolution,
    hasReferenceUploads,
    imageSize,
    localProvider,
    mode,
    referenceUploadIds,
    onCreateTask,
    referenceUploadIds.length,
    selectedModel,
    trimmedPrompt,
  ])

  const nodesById = useMemo(() => new Map(flowNodes.map((node) => [node.id, node])), [flowNodes])
  const selectedNode = nodesById.get(selectedNodeId) ?? flowNodes[1]
  const readyCount = flowNodes.filter((node) => node.status === 'ready' || node.status === 'active').length

  function applyPreset(preset: WorkflowPreset) {
    setSelectedPresetId(preset.id)
    setMode(preset.mode)
    setLocalProvider(preset.provider)
    if (preset.provider === BANANA_PROVIDER) {
      setLocalBananaModel(getBananaModelForRatio(preset.ratio, preset.resolution).id)
    } else {
      setRatio(preset.ratio)
      setResolution(preset.resolution)
      setQuality(preset.quality)
      setOutputFormat(preset.outputFormat)
    }
    setDraftPrompt(preset.prompt)
    setSelectedNodeId('prompt')
    setMessage(`已切换到：${preset.title}`)
  }

  function useWorkflowPrompt() {
    if (!canUsePrompt) {
      setMessage('请先填写提示词，再应用到生成页。')
      return
    }
    onUsePrompt(trimmedPrompt, {
      provider: localProvider,
      model: selectedModel,
      ratio: effectiveRatio || undefined,
    })
    setMessage(`已应用到生成页：${providerLabel(localProvider)}`)
  }

  async function createWorkflowTask() {
    if (!canUsePrompt) {
      setMessage('请先填写提示词，再创建任务。')
      setSelectedNodeId('prompt')
      return
    }
    if (mode === 'image-to-image' && !hasReferenceUploads) {
      setMessage('图生图节点会使用生成页已上传的参考图，请先到生成页上传参考图。')
      setSelectedNodeId('input')
      return
    }
    if (!onCreateTask) {
      setMessage('任务创建回调尚未接入；下方可查看 payload 摘要。')
      setSelectedNodeId('run')
      return
    }

    setIsSubmitting(true)
    setSelectedNodeId('run')
    setMessage('')
    try {
      await onCreateTask(taskPayload)
      setMessage('任务已创建，正在进入结果页。')
    } catch (error) {
      setMessage(error instanceof Error ? error.message : '节点工作流任务创建失败。')
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <main className="node-flow-page" aria-label="节点工作流">
      <header className="node-flow-header">
        <div className="node-flow-heading">
          <span>节点工作流</span>
          <h2>从想法到生成任务</h2>
          <p>{selectedPreset.title} / {providerLabel(localProvider)} / {imageSize}</p>
        </div>
        <div className="node-flow-header-actions">
          <span className={canUsePrompt ? 'node-flow-status ready' : 'node-flow-status missing'}>{readyCount}/6 就绪</span>
          <button type="button" onClick={useWorkflowPrompt} disabled={!canUsePrompt}>应用到生成页</button>
          <button type="button" className="primary" onClick={() => void createWorkflowTask()} disabled={!canUsePrompt || isSubmitting}>
            {isSubmitting ? '创建中...' : '创建任务'}
          </button>
        </div>
      </header>

      {message ? <div className="node-flow-message" role="status">{message}</div> : null}

      <section className="node-flow-shell">
        <aside className="node-flow-left" aria-label="工作流路径">
          <section className="node-flow-panel">
            <div className="node-flow-panel-title">
              <strong>最短路径</strong>
              <span>按这个顺序走</span>
            </div>
            <div className="node-flow-step-list">
              {flowNodes.map((node) => (
                <button
                  key={node.id}
                  type="button"
                  className={`node-flow-step ${selectedNodeId === node.id ? 'active' : ''} ${node.status}`}
                  onClick={() => setSelectedNodeId(node.id)}
                >
                  <b>{node.index}</b>
                  <span>
                    <strong>{node.title}</strong>
                    <small>{node.value}</small>
                  </span>
                </button>
              ))}
            </div>
          </section>

          <section className="node-flow-panel">
            <div className="node-flow-panel-title">
              <strong>场景模板</strong>
              <span>一键换参数</span>
            </div>
            <div className="node-flow-preset-list">
              {WORKFLOW_PRESETS.map((preset) => (
                <button
                  key={preset.id}
                  type="button"
                  className={selectedPresetId === preset.id ? 'active' : ''}
                  onClick={() => applyPreset(preset)}
                >
                  <strong>{preset.title}</strong>
                  <span>{preset.description}</span>
                </button>
              ))}
            </div>
          </section>
        </aside>

        <section className="node-flow-canvas-card" aria-label="节点画布">
          <div className="node-flow-canvas-toolbar">
            <div>
              <strong>生成请求流</strong>
              <span>点击节点后在下方编辑参数</span>
            </div>
            <div className="node-flow-mini-status">
              <span>{mode === 'image-to-image' ? '图生图' : '文生图'}</span>
              <span>{effectiveRatio}</span>
              <span>{effectiveResolution}</span>
            </div>
          </div>
          <div className="node-flow-canvas">
            <div className="node-flow-board" style={{ width: CANVAS_WIDTH, height: CANVAS_HEIGHT }}>
              <svg className="node-flow-wires" width={CANVAS_WIDTH} height={CANVAS_HEIGHT} viewBox={`0 0 ${CANVAS_WIDTH} ${CANVAS_HEIGHT}`} aria-hidden="true">
                <defs>
                  <marker id="node-flow-arrow" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="8" markerHeight="8" orient="auto-start-reverse">
                    <path d="M 0 0 L 10 5 L 0 10 z" />
                  </marker>
                </defs>
                {FLOW_EDGES.map((edge) => {
                  const from = nodesById.get(edge.from)
                  const to = nodesById.get(edge.to)
                  if (!from || !to) return null
                  const active = selectedNodeId === edge.from || selectedNodeId === edge.to
                  const path = buildEdgePath(from, to, edge)
                  const labelPoint = edgeLabelPoint(from, to, edge)
                  return (
                    <g key={edge.id} className={active ? 'active' : ''}>
                      <path d={path} />
                      {edge.label ? <text x={labelPoint.x} y={labelPoint.y}>{edge.label}</text> : null}
                    </g>
                  )
                })}
              </svg>

              {flowNodes.map((node) => (
                <FlowNodeCard
                  key={node.id}
                  node={node}
                  selected={selectedNodeId === node.id}
                  onSelect={() => setSelectedNodeId(node.id)}
                />
              ))}
            </div>
          </div>
        </section>

        <aside className="node-flow-inspector" aria-label="节点参数">
          <div className="node-flow-inspector-title">
            <span>{selectedNode.eyebrow}</span>
            <h3>{selectedNode.title}</h3>
            <p>{selectedNode.purpose}</p>
          </div>
          {renderInspector(selectedNode.id)}
        </aside>
      </section>
    </main>
  )

  function renderInspector(nodeId: FlowNodeId) {
    if (nodeId === 'input') {
      return (
        <div className="node-flow-form">
          <Field label="生成模式">
            <div className="node-flow-segmented" role="group" aria-label="生成模式">
              <button type="button" className={mode === 'text-to-image' ? 'active' : ''} onClick={() => setMode('text-to-image')}>文生图</button>
              <button type="button" className={mode === 'image-to-image' ? 'active' : ''} onClick={() => setMode('image-to-image')}>图生图</button>
            </div>
          </Field>
          <InfoRow label="参考图" value={mode === 'image-to-image' ? (hasReferenceUploads ? `${referenceUploadIds.length} 张已上传` : '请先在生成页上传') : '不需要'} />
          <p className="node-flow-note">图生图节点会直接使用生成页当前已上传的参考图；没有参考图时不会创建任务。</p>
        </div>
      )
    }

    if (nodeId === 'prompt') {
      return (
        <div className="node-flow-form">
          <Field label="主提示词">
            <textarea value={draftPrompt} onChange={(event) => setDraftPrompt(event.target.value)} rows={10} />
          </Field>
          <div className="node-flow-inline-actions">
            <span>{trimmedPrompt.length} 字符</span>
            <button type="button" onClick={useWorkflowPrompt} disabled={!canUsePrompt}>应用到生成页</button>
          </div>
        </div>
      )
    }

    if (nodeId === 'model') {
      return (
        <div className="node-flow-form">
          <Field label="模型供应商">
            <div className="node-flow-segmented" role="group" aria-label="模型供应商">
              <button type="button" className={localProvider === 'image-2' ? 'active' : ''} onClick={() => setLocalProvider('image-2')}>Image-2</button>
              <button type="button" className={localProvider === BANANA_PROVIDER ? 'active' : ''} onClick={() => setLocalProvider(BANANA_PROVIDER)}>Banana</button>
            </div>
          </Field>
          {localProvider === BANANA_PROVIDER ? (
            <Field label="Banana 模型">
              <select value={localBananaModel} onChange={(event) => setLocalBananaModel(event.target.value)}>
                {BANANA_MODEL_OPTIONS.map((option) => <option key={option.id} value={option.id}>{option.label} / {option.size}</option>)}
              </select>
            </Field>
          ) : null}
          <InfoRow label="模型 ID" value={selectedModel} />
        </div>
      )
    }

    if (nodeId === 'spec') {
      return (
        <div className="node-flow-form">
          {localProvider === BANANA_PROVIDER ? (
            <>
              <InfoRow label="比例" value={bananaOption.ratio} />
              <InfoRow label="清晰度" value={bananaOption.resolution} />
              <InfoRow label="尺寸" value={bananaOption.size} />
              <p className="node-flow-note">Banana 的比例和尺寸由模型规格决定。</p>
            </>
          ) : (
            <>
              <Field label="比例">
                <select value={ratio} onChange={(event) => setRatio(event.target.value)}>
                  {RATIOS.map((item) => <option key={item} value={item}>{ratioLabel(item)}</option>)}
                </select>
              </Field>
              <div className="node-flow-two-fields">
                <Field label="清晰度">
                  <select value={resolution} onChange={(event) => setResolution(event.target.value)}>
                    {RESOLUTION_TIERS.map((item) => <option key={item} value={item}>{getResolutionLabel(item)}</option>)}
                  </select>
                </Field>
                <Field label="质量">
                  <select value={quality} onChange={(event) => setQuality(event.target.value)}>
                    {QUALITY_LEVELS.map((item) => <option key={item} value={item}>{getQualityLabel(item)}</option>)}
                  </select>
                </Field>
              </div>
              <Field label="输出格式">
                <select value={outputFormat} onChange={(event) => setOutputFormat(event.target.value)}>
                  {OUTPUT_FORMATS.map((item) => <option key={item} value={item}>{getOutputFormatLabel(item)}</option>)}
                </select>
              </Field>
              <InfoRow label="计算尺寸" value={imageSize} />
            </>
          )}
        </div>
      )
    }

    if (nodeId === 'run') {
      return (
        <div className="node-flow-form">
          <div className="node-flow-two-fields">
            <Field label="数量">
              <input type="number" min={1} max={12} value={count} onChange={(event) => setCount(readBoundedInteger(event.target.value, count, 1, 12))} />
            </Field>
            <Field label="并发">
              <input type="number" min={1} max={12} value={concurrency} onChange={(event) => setConcurrency(readBoundedInteger(event.target.value, concurrency, 1, 12))} />
            </Field>
          </div>
          <button type="button" className="node-flow-run-button" onClick={() => void createWorkflowTask()} disabled={!canUsePrompt || isSubmitting}>
            {isSubmitting ? '创建中...' : '创建任务'}
          </button>
          <details className="node-flow-payload">
            <summary>查看请求摘要</summary>
            <pre>{JSON.stringify(taskPayload, null, 2)}</pre>
          </details>
        </div>
      )
    }

    return (
      <div className="node-flow-form">
        <div className="node-flow-result-preview">
          <span>任务创建后在结果页查看实时状态和图片</span>
        </div>
        <InfoRow label="状态" value="等待任务提交" />
        <InfoRow label="结果入口" value="结果页 / 任务历史" />
        <button type="button" onClick={useWorkflowPrompt} disabled={!canUsePrompt}>复用提示词</button>
      </div>
    )
  }
}

function FlowNodeCard({ node, selected, onSelect }: { node: FlowNode; selected: boolean; onSelect: () => void }) {
  const style = {
    '--node-x': `${node.x}px`,
    '--node-y': `${node.y}px`,
    '--node-width': `${node.width}px`,
    '--node-height': `${node.height}px`,
    '--node-tone': node.tone,
  } as CSSProperties

  return (
    <button type="button" className={`node-flow-node ${selected ? 'selected' : ''} ${node.status}`} style={style} onClick={onSelect} aria-pressed={selected}>
      {node.input ? <span className="node-flow-port input"><i />{node.input}</span> : null}
      <span className="node-flow-node-head">
        <small>{node.eyebrow}</small>
        <strong>{node.title}</strong>
        <em>{node.index}</em>
      </span>
      <span className="node-flow-node-body">
        <span>{node.purpose}</span>
        <b>{node.value}</b>
      </span>
      <span className="node-flow-node-meta">
        {node.meta.map((item) => <i key={item}>{item}</i>)}
      </span>
      {node.output ? <span className="node-flow-port output">{node.output}<i /></span> : null}
    </button>
  )
}

function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    <label className="node-flow-field">
      <span>{label}</span>
      {children}
    </label>
  )
}

function InfoRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="node-flow-info-row">
      <span>{label}</span>
      <strong title={value}>{value}</strong>
    </div>
  )
}

function buildEdgePath(from: FlowNode, to: FlowNode, edge: FlowEdge) {
  const start = { x: from.x + from.width, y: from.y + from.height / 2 + (edge.fromOffsetY ?? 0) }
  const end = { x: to.x, y: to.y + to.height / 2 + (edge.toOffsetY ?? 0) }
  const distance = Math.max(110, Math.abs(end.x - start.x) * 0.52)
  return `M ${start.x} ${start.y} C ${start.x + distance} ${start.y}, ${end.x - distance} ${end.y}, ${end.x} ${end.y}`
}

function edgeLabelPoint(from: FlowNode, to: FlowNode, edge: FlowEdge) {
  return {
    x: (from.x + from.width + to.x) / 2 - 18,
    y: (from.y + from.height / 2 + (edge.fromOffsetY ?? 0) + to.y + to.height / 2 + (edge.toOffsetY ?? 0)) / 2 - 8,
  }
}

function readBoundedInteger(value: string, fallback: number, min: number, max: number) {
  const parsed = Number(value)
  if (!Number.isFinite(parsed)) return fallback
  return Math.min(max, Math.max(min, Math.round(parsed)))
}

function compactModelId(id: string) {
  if (id.length <= 32) return id
  return `${id.slice(0, 16)}...${id.slice(-12)}`
}

function ratioLabel(value: string) {
  return value === 'auto' ? '自动' : value
}

