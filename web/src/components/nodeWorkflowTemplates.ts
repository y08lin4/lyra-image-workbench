import type { InspirationIdea, ModelProvider, PromptLibraryItem } from '../types'
import { DEFAULT_IMAGE2_MODEL, IMAGE2_PROVIDER } from '../lib/models'

export type NodeWorkflowNodeKind =
  | 'input'
  | 'workbench'
  | 'canvas'
  | 'template'
  | 'reference'
  | 'gallery'
  | 'prompt'
  | 'model'
  | 'api'
  | 'review'
  | 'output'

export type NodeWorkflowValue = string | number | boolean | readonly string[]

export interface NodeWorkflowPosition {
  x: number
  y: number
}

export interface NodeWorkflowNodeData {
  label: string
  description: string
  prompt?: string
  provider?: ModelProvider
  model?: string
  ratio?: string
  fields?: Record<string, NodeWorkflowValue>
}

export interface NodeWorkflowNode {
  id: string
  kind: NodeWorkflowNodeKind
  title: string
  position: NodeWorkflowPosition
  data: NodeWorkflowNodeData
}

export interface NodeWorkflowConnection {
  id: string
  from: string
  to: string
  label?: string
}

export interface NodeWorkflowSeed {
  id: string
  title: string
  description: string
  nodes: NodeWorkflowNode[]
  connections: NodeWorkflowConnection[]
}

export interface NodeWorkflowTemplate extends NodeWorkflowSeed {
  category: 'generate' | 'rewrite' | 'reference' | 'canvas' | 'workbench' | 'api'
  tags: readonly string[]
}

export interface PromptWorkflowIdea {
  id?: string
  title: string
  category?: string
  prompt?: string
  summary?: string
  tags?: readonly string[]
  ratio?: string
  provider?: ModelProvider
  model?: string
}

export interface PromptIdeaWorkflowOptions {
  idPrefix?: string
  provider?: ModelProvider
  model?: string
  ratio?: string
}

export const defaultNodeWorkflowTemplates = [
  {
    id: 'text-to-image-basic',
    title: '文生图基础流',
    description: 'ComfyUI 式基础节点流：从一句话想法到提示词、模型参数和图片输出。',
    category: 'generate',
    tags: ['文生图', 'ComfyUI', '提示词扩写', '快速生成'],
    nodes: [
      node('basic-idea', 'input', '一句话想法', 40, 120, {
        label: '输入灵感',
        description: '记录主题、情绪、主体和用途。',
        prompt: '雨夜东京街头的赛博朋克少女',
        fields: {
          source: 'manual',
          language: 'zh-CN',
        },
      }),
      node('basic-prompt', 'prompt', '提示词扩写', 300, 120, {
        label: '结构化提示词',
        description: '补全主体、构图、光线、材质、风格和负面约束。',
        fields: {
          style: '自动判断',
          keepEditable: true,
        },
      }),
      node('basic-model', 'model', '模型参数', 560, 120, {
        label: 'Image-2 默认生成',
        description: '使用当前模型参数，无需额外配置。',
        provider: IMAGE2_PROVIDER,
        model: DEFAULT_IMAGE2_MODEL,
        ratio: 'auto',
        fields: {
          count: 1,
          quality: 'auto',
          outputFormat: 'png',
        },
      }),
      node('basic-output', 'output', '结果审阅', 820, 120, {
        label: '图片结果',
        description: '查看生成图，复制修订提示词，或继续进入二次改写。',
        fields: {
          nextActions: ['收藏', '复制提示词', '再次生成'],
        },
      }),
    ],
    connections: [
      connection('basic-idea-to-prompt', 'basic-idea', 'basic-prompt', '扩写'),
      connection('basic-prompt-to-model', 'basic-prompt', 'basic-model', '应用'),
      connection('basic-model-to-output', 'basic-model', 'basic-output', '生成'),
    ],
  },
  {
    id: 'poster-template-rewrite',
    title: '海报模板改写流',
    description: '把海报模板想法拆成文案层级、视觉约束和最终生成参数。',
    category: 'rewrite',
    tags: ['海报', '模板改写', '商业设计'],
    nodes: [
      node('poster-template', 'template', '海报模板', 40, 80, {
        label: '模板骨架',
        description: '承接来自提示词库或人工输入的海报构想。',
        prompt: '新品发布会主视觉海报，强标题，科技感，竖版构图',
        fields: {
          layout: '2:3 poster',
          editableSections: ['主标题', '副标题', '视觉主体', '品牌区'],
        },
      }),
      node('poster-copy', 'prompt', '文案改写', 300, 40, {
        label: '标题与信息层级',
        description: '把模板中的文字诉求改写成清晰的海报层级。',
        fields: {
          headline: '主标题优先',
          subtitle: '保留活动信息',
        },
      }),
      node('poster-visual', 'prompt', '视觉改写', 300, 180, {
        label: '画面语言',
        description: '约束主体、色彩、材质、光影和留白。',
        fields: {
          style: '高级商业海报',
          avoid: ['低清晰度文字', '杂乱排版', '多余图标'],
        },
      }),
      node('poster-model', 'model', '竖版生成', 560, 110, {
        label: '海报输出参数',
        description: '默认使用竖版比例，方便直接进入海报场景。',
        provider: IMAGE2_PROVIDER,
        model: DEFAULT_IMAGE2_MODEL,
        ratio: '2:3',
        fields: {
          quality: 'high',
          outputFormat: 'png',
        },
      }),
      node('poster-review', 'review', '版式检查', 820, 110, {
        label: '排版审阅',
        description: '重点检查文字可读性、主体占比和品牌区域。',
        fields: {
          checklist: ['文字清晰', '主体突出', '品牌区完整'],
        },
      }),
    ],
    connections: [
      connection('poster-template-to-copy', 'poster-template', 'poster-copy', '拆文案'),
      connection('poster-template-to-visual', 'poster-template', 'poster-visual', '拆视觉'),
      connection('poster-copy-to-model', 'poster-copy', 'poster-model', '合并'),
      connection('poster-visual-to-model', 'poster-visual', 'poster-model', '合并'),
      connection('poster-model-to-review', 'poster-model', 'poster-review', '生成'),
    ],
  },
  {
    id: 'image-to-image-reference',
    title: '图生图参考流',
    description: '从参考图提取视觉信息，再生成保留构图或风格的改写提示词。',
    category: 'reference',
    tags: ['图生图', '参考图', '图片还原'],
    nodes: [
      node('reference-image', 'reference', '参考图片', 40, 120, {
        label: '图片输入',
        description: '选择上传图或历史结果图作为视觉参考。',
        fields: {
          sourceTypes: ['upload', 'result'],
          required: true,
        },
      }),
      node('reference-analysis', 'prompt', '图片分析', 300, 120, {
        label: '结构化观察',
        description: '提取主体、构图、镜头、色彩、材质和氛围。',
        fields: {
          observations: ['subject', 'composition', 'lighting', 'style', 'ratio'],
        },
      }),
      node('reference-rewrite', 'prompt', '改写目标', 560, 60, {
        label: '保留与变化',
        description: '声明保留项和希望变化的方向。',
        prompt: '保留构图和光影，主体换成未来感产品摄影',
        fields: {
          mustKeep: ['构图', '光线方向', '景深'],
          editable: true,
        },
      }),
      node('reference-model', 'model', '参考生成', 560, 200, {
        label: '图生图参数',
        description: '保留当前生成设置，用节点表达上传引用。',
        provider: IMAGE2_PROVIDER,
        model: DEFAULT_IMAGE2_MODEL,
        ratio: 'auto',
        fields: {
          mode: 'image-to-image',
          referenceStrength: 'medium',
        },
      }),
      node('reference-output', 'output', '对照结果', 820, 130, {
        label: '结果对比',
        description: '对照参考图与生成结果，决定继续加强相似度或扩大变化。',
        fields: {
          nextActions: ['增强相似度', '扩大变化', '复制提示词'],
        },
      }),
    ],
    connections: [
      connection('reference-image-to-analysis', 'reference-image', 'reference-analysis', '分析'),
      connection('reference-analysis-to-rewrite', 'reference-analysis', 'reference-rewrite', '提取'),
      connection('reference-image-to-model', 'reference-image', 'reference-model', '引用'),
      connection('reference-rewrite-to-model', 'reference-rewrite', 'reference-model', '应用'),
      connection('reference-model-to-output', 'reference-model', 'reference-output', '生成'),
    ],
  },
  {
    id: 'invokeai-canvas-gallery-return',
    title: '画布创作流 / 图库回流',
    description: 'InvokeAI 风格的创作闭环：在画布上迭代局部修改，把满意结果回流到图库并再次作为参考。',
    category: 'canvas',
    tags: ['InvokeAI', '画布', '图库回流', '局部重绘'],
    nodes: [
      node('invoke-gallery-source', 'gallery', '图库素材', 40, 90, {
        label: '图库选择',
        description: '从历史结果或上传素材中选择一张图作为创作起点。',
        fields: {
          sourceTypes: ['result', 'upload', 'favorite'],
          reusable: true,
        },
      }),
      node('invoke-canvas', 'canvas', '画布创作', 300, 90, {
        label: 'Canvas',
        description: '面向普通创作者的画布节点，表达扩图、遮罩、局部重绘和版本迭代。',
        prompt: '保留主体姿态，扩展背景空间，并重绘右侧光源',
        fields: {
          tools: ['outpaint', 'inpaint', 'mask', 'brush'],
          preserveLayers: true,
        },
      }),
      node('invoke-prompt', 'prompt', '局部提示词', 560, 40, {
        label: '区域描述',
        description: '为当前遮罩或扩展区域补充更具体的视觉描述。',
        fields: {
          targetArea: 'selected region',
          editable: true,
        },
      }),
      node('invoke-model', 'model', '画布生成', 560, 180, {
        label: '参考生成参数',
        description: '把画布操作转为当前生成参数。',
        provider: IMAGE2_PROVIDER,
        model: DEFAULT_IMAGE2_MODEL,
        ratio: 'auto',
        fields: {
          mode: 'image-to-image',
          referenceStrength: 'high',
        },
      }),
      node('invoke-gallery-return', 'gallery', '回流图库', 820, 120, {
        label: '保存版本',
        description: '将结果保存为图库版本，可收藏、继续编辑，或作为新的参考图。',
        fields: {
          nextActions: ['保存到图库', '设为新参考', '继续画布编辑'],
        },
      }),
    ],
    connections: [
      connection('invoke-gallery-to-canvas', 'invoke-gallery-source', 'invoke-canvas', '打开'),
      connection('invoke-canvas-to-prompt', 'invoke-canvas', 'invoke-prompt', '选区'),
      connection('invoke-prompt-to-model', 'invoke-prompt', 'invoke-model', '描述'),
      connection('invoke-canvas-to-model', 'invoke-canvas', 'invoke-model', '遮罩/参考'),
      connection('invoke-model-to-return', 'invoke-model', 'invoke-gallery-return', '保存'),
      connection('invoke-return-to-canvas', 'invoke-gallery-return', 'invoke-canvas', '继续迭代'),
    ],
  },
  {
    id: 'workbench-user-flow',
    title: '普通用户工作台流',
    description: '面向非节点用户的工作台流程：输入想法、选择模型、上传参考、生成结果并复用到提示词库或图库。',
    category: 'workbench',
    tags: ['工作台', '普通用户', '生成页', '结果复用'],
    nodes: [
      node('workbench-compose', 'workbench', '生成页输入', 40, 100, {
        label: '输入与设置',
        description: '用户在现有生成页完成提示词、比例、数量、格式和模型选择。',
        prompt: '一张清爽的产品主图，白色背景，柔和自然光',
        provider: IMAGE2_PROVIDER,
        model: DEFAULT_IMAGE2_MODEL,
        ratio: '1:1',
        fields: {
          count: 1,
          outputFormat: 'png',
        },
      }),
      node('workbench-reference', 'reference', '可选参考图', 300, 40, {
        label: '上传参考',
        description: '普通用户可跳过，也可上传参考图进入图生图或提示词还原。',
        fields: {
          optional: true,
          sourceTypes: ['upload', 'result'],
        },
      }),
      node('workbench-task', 'model', '提交任务', 300, 180, {
        label: '进度跟踪',
        description: '展示创建、进行中和完成状态。',
        provider: IMAGE2_PROVIDER,
        model: DEFAULT_IMAGE2_MODEL,
        ratio: '1:1',
        fields: {
          statusFlow: ['queued', 'running', 'succeeded'],
          diagnosticsOptional: true,
        },
      }),
      node('workbench-results', 'gallery', '结果图库', 560, 110, {
        label: '结果管理',
        description: '查看、收藏、复制图片链接，或把满意结果沉淀到图库。',
        fields: {
          actions: ['预览', '收藏', '复制', '提交广场'],
        },
      }),
      node('workbench-reuse', 'template', '再次复用', 820, 110, {
        label: '回到创作',
        description: '把结果作为参考图、把修订提示词填回生成页，或保存为模板。',
        fields: {
          reuseAs: ['参考图', '提示词模板', '广场作品'],
        },
      }),
    ],
    connections: [
      connection('workbench-compose-to-task', 'workbench-compose', 'workbench-task', '提交'),
      connection('workbench-reference-to-task', 'workbench-reference', 'workbench-task', '引用'),
      connection('workbench-task-to-results', 'workbench-task', 'workbench-results', '生成'),
      connection('workbench-results-to-reuse', 'workbench-results', 'workbench-reuse', '沉淀'),
      connection('workbench-reuse-to-compose', 'workbench-reuse', 'workbench-compose', '再创作'),
    ],
  },
  {
    id: 'api-task-flow',
    title: 'API 调用流',
    description: '用节点描述一次 API 调用：鉴权、请求体、进度跟踪和结果读取。',
    category: 'api',
    tags: ['API', '进度跟踪', '自动化'],
    nodes: [
      node('api-auth', 'api', '鉴权配置', 40, 80, {
        label: 'API Key',
        description: '引用已保存的访问密钥配置，不在节点中存储明文。',
        fields: {
          header: 'Authorization',
          secretStorage: 'settings',
        },
      }),
      node('api-payload', 'api', '请求体', 300, 80, {
        label: 'Create Task',
        description: '映射前端生成参数到任务创建请求。',
        prompt: 'A clean product hero image on a glass table',
        provider: IMAGE2_PROVIDER,
        model: DEFAULT_IMAGE2_MODEL,
        ratio: '1:1',
        fields: {
          endpoint: 'POST /api/tasks',
          count: 1,
          concurrency: 1,
        },
      }),
      node('api-queue', 'api', '进度跟踪', 560, 80, {
        label: 'Task Status',
        description: '跟踪 queued、running、succeeded、failed 等状态。',
        fields: {
          polling: 'GET /api/tasks/:id',
          stream: 'SSE optional',
        },
      }),
      node('api-result', 'output', '结果消费', 820, 80, {
        label: 'Image URLs',
        description: '读取结果图、错误信息、耗时和修订提示词。',
        fields: {
          outputFields: ['imageUrl', 'remoteUrl', 'statusText', 'elapsedMs'],
        },
      }),
    ],
    connections: [
      connection('api-auth-to-payload', 'api-auth', 'api-payload', '签名'),
      connection('api-payload-to-queue', 'api-payload', 'api-queue', '提交'),
      connection('api-queue-to-result', 'api-queue', 'api-result', '完成'),
    ],
  },
] as const satisfies readonly NodeWorkflowTemplate[]

export type DefaultNodeWorkflowTemplateId = typeof defaultNodeWorkflowTemplates[number]['id']

export function getDefaultNodeWorkflowTemplate(id: DefaultNodeWorkflowTemplateId | string) {
  return defaultNodeWorkflowTemplates.find((template) => template.id === id) || defaultNodeWorkflowTemplates[0]
}

export function templateToNodeWorkflowSeed(template: NodeWorkflowTemplate): NodeWorkflowSeed {
  return cloneWorkflowSeed(template)
}

export function createWorkflowSeedFromPromptLibraryItem(
  item: Pick<PromptLibraryItem, 'id' | 'title' | 'category' | 'prompt'>,
  options: PromptIdeaWorkflowOptions = {},
) {
  return createWorkflowSeedFromPromptIdea({
    id: item.id,
    title: item.title,
    category: item.category,
    prompt: item.prompt,
  }, options)
}

export function createWorkflowSeedFromInspirationIdea(idea: InspirationIdea, options: PromptIdeaWorkflowOptions = {}) {
  return createWorkflowSeedFromPromptIdea({
    id: idea.id,
    title: idea.title,
    category: idea.category,
    summary: idea.summary,
    tags: idea.tags,
  }, options)
}

export function createWorkflowSeedFromPromptIdea(idea: PromptWorkflowIdea, options: PromptIdeaWorkflowOptions = {}): NodeWorkflowSeed {
  const provider = IMAGE2_PROVIDER
  const model = DEFAULT_IMAGE2_MODEL
  const ratio = options.ratio || idea.ratio || 'auto'
  const seedId = `${options.idPrefix || 'prompt-idea'}-${slugifyWorkflowId(idea.id || idea.title)}`
  const prompt = idea.prompt || idea.summary || ''
  const tagSummary = idea.tags?.length ? idea.tags.join(' / ') : idea.category || '未分类'

  return {
    id: seedId,
    title: idea.title,
    description: `从提示词库或灵感条目生成的节点工作流种子：${tagSummary}`,
    nodes: [
      node(`${seedId}-source`, 'template', '模板来源', 40, 120, {
        label: idea.category || 'Prompt Idea',
        description: '保留原始条目的标题、分类和摘要，方便回溯。',
        prompt,
        fields: {
          sourceId: idea.id || '',
          tags: idea.tags || [],
        },
      }),
      node(`${seedId}-prompt`, 'prompt', '提示词整理', 320, 120, {
        label: '可编辑提示词',
        description: '把原始条目整理成当前任务可直接使用的提示词。',
        prompt,
        fields: {
          editable: true,
          sourceCategory: idea.category || '',
        },
      }),
      node(`${seedId}-model`, 'model', '模型参数', 600, 120, {
        label: 'Image-2 生成',
        description: '默认参数可由 NodeWorkflowPage 映射到现有前端任务创建逻辑。',
        provider,
        model,
        ratio,
        fields: {
          count: 1,
          quality: 'auto',
        },
      }),
      node(`${seedId}-output`, 'output', '生成结果', 880, 120, {
        label: '结果预览',
        description: '输出图片和修订提示词，后续可继续作为参考图或模板。',
        fields: {
          nextActions: ['应用到生成页', '保存为模板', '继续改写'],
        },
      }),
    ],
    connections: [
      connection(`${seedId}-source-to-prompt`, `${seedId}-source`, `${seedId}-prompt`, '整理'),
      connection(`${seedId}-prompt-to-model`, `${seedId}-prompt`, `${seedId}-model`, '应用'),
      connection(`${seedId}-model-to-output`, `${seedId}-model`, `${seedId}-output`, '生成'),
    ],
  }
}

function node(
  id: string,
  kind: NodeWorkflowNodeKind,
  title: string,
  x: number,
  y: number,
  data: NodeWorkflowNodeData,
): NodeWorkflowNode {
  return {
    id,
    kind,
    title,
    position: { x, y },
    data,
  }
}

function connection(id: string, from: string, to: string, label?: string): NodeWorkflowConnection {
  return { id, from, to, label }
}

function cloneWorkflowSeed(seed: NodeWorkflowSeed): NodeWorkflowSeed {
  return {
    id: seed.id,
    title: seed.title,
    description: seed.description,
    nodes: seed.nodes.map((item) => ({
      ...item,
      position: { ...item.position },
      data: {
        ...item.data,
        fields: cloneFields(item.data.fields),
      },
    })),
    connections: seed.connections.map((item) => ({ ...item })),
  }
}

function cloneFields(fields?: Record<string, NodeWorkflowValue>) {
  if (!fields) return undefined
  return Object.fromEntries(Object.entries(fields).map(([key, value]) => [key, Array.isArray(value) ? [...value] : value]))
}

function defaultModelForProvider(provider: ModelProvider) {
  return DEFAULT_IMAGE2_MODEL
}

function slugifyWorkflowId(value: string) {
  const slug = value
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '')
  return slug || 'workflow'
}


