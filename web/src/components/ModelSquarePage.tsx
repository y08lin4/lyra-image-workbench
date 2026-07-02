import './ModelSquarePage.css'

type ModelStatus = 'candidate' | 'needs-channel' | 'planned'

type ModelSquareItem = {
  id: string
  name: string
  channel: string
  tagline: string
  scenes: string[]
  strengths: string[]
  cost: string
  status: ModelStatus
}

const statusLabel: Record<ModelStatus, string> = {
  candidate: '候选接入',
  'needs-channel': '需后台启用',
  planned: '规划评估',
}

const modelItems: ModelSquareItem[] = [
  {
    id: 'image-2',
    name: 'image-2',
    channel: 'OpenAI-compatible 图片模型',
    tagline: '适合日常文生图、产品草图、海报和社媒配图的默认候选。',
    scenes: ['通用文生图', '产品图草案', '运营配图'],
    strengths: ['提示词理解稳定', '适合批量生成', '可作为工作站默认模型'],
    cost: '按后台渠道价格配置，前端仅显示占位',
    status: 'needs-channel',
  },
  {
    id: 'image-2-full',
    name: 'image-2（满血版）',
    channel: 'OpenAI-compatible 图片模型',
    tagline: '面向高质量出图、复杂语义和精修场景的增强候选。',
    scenes: ['高质量海报', '复杂主体', '精修成片'],
    strengths: ['质量档位更高', '适合最终稿探索', '可与普通版分层计费'],
    cost: '建议使用更高消耗占位，实际以渠道规则为准',
    status: 'candidate',
  },
  {
    id: 'z-image-turbo',
    name: 'z-image-turbo',
    channel: '自定义渠道',
    tagline: '偏向快速预览和多方案探索的轻量候选。',
    scenes: ['快速打样', '多版本试错', '低成本预览'],
    strengths: ['响应速度优先', '适合灵感草图', '可作为低消耗档位'],
    cost: '低消耗占位，需在后台绑定真实渠道',
    status: 'planned',
  },
  {
    id: 'qwen-image',
    name: 'Qwen-Image',
    channel: 'OpenAI-compatible 图片模型 / 自定义渠道',
    tagline: '适合中文语义、文字元素和本土化视觉表达的候选。',
    scenes: ['中文海报', '字体元素', '本土品牌视觉'],
    strengths: ['中文提示词友好', '适合含字画面探索', '便于接入自定义渠道'],
    cost: '按渠道单价折算为工作站消耗',
    status: 'needs-channel',
  },
  {
    id: 'kolors',
    name: 'Kolors',
    channel: '自定义渠道',
    tagline: '适合人物、摄影感和风格化构图的扩展候选。',
    scenes: ['人像风格', '摄影构图', '电商视觉'],
    strengths: ['视觉风格丰富', '适合频道化配置', '可作为特色模型入口'],
    cost: '消耗占位待后台配置',
    status: 'candidate',
  },
]

const setupSteps = [
  '在后台模型渠道中创建 OpenAI-compatible 图片模型或自定义渠道。',
  '为候选模型配置真实模型名、Key、Base URL、消耗倍率和可见范围。',
  '启用后再在创作画布、快捷生成或 API 文档中暴露给用户选择。',
]

export function ModelSquarePage() {
  return (
    <section className="model-square-page">
      <header className="model-square-hero">
        <div>
          <p className="eyebrow">Model Square</p>
          <h2>模型广场</h2>
          <p>整理可加入工作站的图片模型能力，用于规划渠道接入、定价占位和创作入口。这里展示的是候选能力，不代表当前已经真实启用。</p>
        </div>
        <div className="model-square-hero-note" role="note">
          <strong>启用说明</strong>
          <span>模型需要先在后台模型渠道启用。前端仅展示候选、适用场景和消耗占位，不暴露具体渠道品牌。</span>
        </div>
      </header>

      <div className="model-square-layout">
        <div className="model-square-grid">
          {modelItems.map((model) => (
            <article key={model.id} className="model-square-card">
              <header>
                <span className="model-square-status">{statusLabel[model.status]}</span>
                <h3>{model.name}</h3>
                <p>{model.channel}</p>
              </header>
              <p className="model-square-tagline">{model.tagline}</p>

              <div className="model-square-section">
                <span>适用场景</span>
                <div className="model-square-tags">
                  {model.scenes.map((scene) => <em key={scene}>{scene}</em>)}
                </div>
              </div>

              <div className="model-square-section">
                <span>能力说明</span>
                <ul>
                  {model.strengths.map((strength) => <li key={strength}>{strength}</li>)}
                </ul>
              </div>

              <footer>
                <span>价格 / 消耗</span>
                <strong>{model.cost}</strong>
              </footer>
            </article>
          ))}
        </div>

        <aside className="model-square-sidebar">
          <section>
            <p className="eyebrow">Activation</p>
            <h3>接入前检查</h3>
            <ol>
              {setupSteps.map((step) => <li key={step}>{step}</li>)}
            </ol>
          </section>
          <section>
            <p className="eyebrow">Visibility</p>
            <h3>状态口径</h3>
            <p>所有卡片都使用候选或待启用状态。只有后台渠道配置完成后，才应在生成表单中声明可用。</p>
          </section>
        </aside>
      </div>
    </section>
  )
}
