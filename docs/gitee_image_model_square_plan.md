# Gitee AI 图片模型广场与接入规划草案

更新时间：2026-07-01

状态：规划草案，仅用于产品与工程拆解，不代表最终实现承诺。

## 1. 产品目标

在 Lyra Image Workbench 新增一个「Gitee AI 图片模型广场」，让用户可以在站内发现、比较、启用并调用 Gitee AI 的图片生成模型。它不是营销落地页，而是一个可操作的模型管理入口。

核心目标：

1. 看懂模型：展示模型定位、能力、适合场景、限制和示例参数。
2. 看懂价格：展示 Gitee AI 官方参考价格、站内预计消耗、更新时间和数据来源。
3. 加入工作站：用户可以把模型加入「我的模型」，用于画布、快捷生成和外部 API。
4. 统一调用：在现有任务队列、历史记录、结果保存和外部 API 基础上接入 `gitee-ai` provider。
5. 控制风险：价格、参数、模型可用性都以官方接口运行时刷新为准，不把临时抓取结果写死。

首阶段只做「模型广场 + 加入工作站 + 文生图调用」闭环。图生图、编辑、分层、超分、抠图等工具型能力保留入口设计，但按官方 operations 参数逐个开放。

## 2. 官方数据来源

- 模型广场来源 URL：https://ai.gitee.com/serverless-api#image_generation
- 官方 services API（模型/服务清单）：`GET https://ai.gitee.com/api/pay/services?type=serverless&status=1&size=1000`
- 官方 operations API（模型操作、参数与价格明细）：`GET https://ai.gitee.com/api/pay/service/operations?service_ident=<modelId>`
- 官方计费单位 API：`GET https://ai.gitee.com/api/base/tags?category_slugs=billing-unit`
- Gitee AI 文生图调用文档：https://ai.gitee.com/docs/products/apis/images-vision/text2image
- Gitee AI Serverless API 文档：https://ai.gitee.com/docs/products/serverless-api/intro

价格必须运行时刷新并展示更新时间，不要长期硬编码。本文中的价格只是 2026-07-01 从官方 services API 与 operations API 读取的参考值，线上展示必须以运行时刷新结果为准。

## 3. 推荐模型

以下推荐按「先覆盖文生图主路径，再扩展编辑/工具能力」排序。价格列是参考展示格式，不作为静态配置。

### P0：首批文生图模型

| 模型 ID | 参考价，需运行时刷新 | 定位 | 推荐原因 | 首阶段能力 |
| --- | ---: | --- | --- | --- |
| `z-image-turbo` | `0.05 - 0.12 / 张` | 默认推荐/高速质量平衡 | 支持 1024 和 2K 价档，适合做 Gitee 默认候选 | 文生图 |
| `Qwen-Image` | `0.05 - 0.12 / 张` | 中文与文字渲染 | 中文语义、海报文字、商品图场景优先 | 文生图 |
| `qwen-image-2.0` | `0.20 / 张` | 高质量多模态生成 | 价格较高，适合放入「高质量」档 | 文生图 |
| `Qwen-Image-2512` | `0.05 - 0.12 / 次` | 新版 Qwen 图像生成 | 支持 1024/2048 操作，作为 Qwen 新版候选 | 文生图 |
| `FLUX.1-schnell` | `0.01 / 张` | 低价快速试稿 | 适合草图、灵感探索、批量试方向 | 文生图 |
| `FLUX.1-dev` | `0.05 - 0.10 / 次` | 通用高质量创作 | 覆盖 1024/1536、局部重绘和图生图操作，首阶段先开文生图 | 文生图 |
| `FLUX.2-dev` | `0.10 / 次` | 专业创意设计 | 强文本理解和细节表现，适合高质量创作 | 文生图 |
| `FLUX.2-klein-4B` | `0 - 0.06 / 次` | 低成本快速试稿 | 价格低，适合新用户试用和批量探索 | 文生图 |
| `FLUX.2-klein-9B` | `0.01 - 0.10 / 次` | 快速高质量折中 | 比 4B 更强，仍适合高频试稿 | 文生图 |
| `CogView4_6B` | `0.01 / 次` | 低价中文文生图 | 中文场景性价比高，适合做低价推荐 | 文生图 |
| `Kolors` | `0.01 - 0.05 / 次` | 中文理解/风格能力 | Gitee 文生图文档示例模型，用户认知成本低 | 文生图，后续扩展编辑 |
| `stable-diffusion-3.5-large-turbo` | `0.03 / 次` | SD 系通用模型 | 保留用户熟悉的 Stable Diffusion 入口 | 文生图 |
| `HiDream-I1-Full` | `0.03 / 次` | 高质量基础生图 | 适合写实、插画和通用创作 | 文生图 |
| `GLM-Image` | `0.05 / 次` | 国产基础生图 | 作为国产模型候选，适合后续横向对比 | 文生图 |
| `LongCat-Image` | `0.05 / 次` | 通用生图 | 可作为备选模型，观察稳定性后排序 | 文生图 |

### P1：第二阶段工具型模型

| 模型 ID | 参考价，需运行时刷新 | 定位 | 建议 |
| --- | ---: | --- | --- |
| `Qwen-Image-Edit` | `0.05 / 次` | 图像编辑 | 第二阶段加入「改图/编辑」模式 |
| `Qwen-Image-Edit-2511` | `0.10 / 次` | 新版图像编辑 | 适合画布参考图编辑，需确认参数和结果格式 |
| `FLUX.1-Kontext-dev` | `0.05 - 0.12 / 次` | 上下文感知改图 | 适合图生图、局部调整和参考图编辑 |
| `Qwen-Image-Layered` | `0.05 / 图层` | 图像分层 | 适合画布后续图层编辑，不进入首批生图入口 |
| `SAM 3` / `sam3` | `0.02 / 次` | 图像分割 | 可作为结果页工具，不作为生图模型 |

暂不加入：

- 视频相关模型，例如 Wan/HunyuanVideo，继续放在独立视频或 GIF/动图规划中。
- OCR、检测、分类、文档处理类模型，不放进图片生成模型首屏。
- 官方价格为空、operations 为空、服务不可见或 `status` 非可用的模型。
- 参数形态不清楚的编辑/工具模型，先进入「待确认」池。

## 4. 价格展示

官方服务清单的 `operation_summary` 适合作为模型卡片价格摘要；进入详情页或真正发起生成前，再用 `service_ident` 拉取 operations 明细，避免模型操作、分辨率、单位或促销价变化后前端仍展示旧价格。

常见字段解释：

| 字段 | 含义 |
| --- | --- |
| `min_price` / `max_price` | 官方最低/最高计费值 |
| `operation_count` | 模型暴露的操作数量 |
| `free_operation_count` | 免费操作数量，可能出现部分免费 |
| `min_output_million_tokens_price` / `max_output_million_tokens_price` | Token 类模型输出价格摘要，图片模型通常为 0 |
| `unit_tag=1227` | 张 |
| `unit_tag=1270` | 图层 |
| `unit_tag=1264` | 秒，通常用于视频模型，不进入图片首屏 |
| `unit_tag=0` | 官方前端按默认调用单位展示，建议页面写成“次”，并保留“以官方页面为准” |

卡片展示建议：

- 官方参考价：`0.05 - 0.12 / 张`、`0.10 / 次`、`官方暂未返回价格`。
- 站内预计消耗：`预计 1 点/张` 或 `按管理员倍率计算`，不要直接等同官方人民币价格。
- 价格状态：`已刷新于 2026-07-01 20:xx`、`刷新失败，显示上次缓存`、`官方暂未返回价格`。
- 来源链接：详情里保留「以 Gitee AI 官方价格为准」，链接到模型广场来源 URL 或模型详情页。
- 后端保存：仅保存官方价格缓存和更新时间；用户加入模型时不复制长期静态价格。

## 5. 产品设计

### 导航

- 左侧导航新增「模型」或「模型广场」。
- 桌面位置：创作组，靠近「创作画布」「快捷生成」。
- 手机端：先放入「更多」，避免底部导航继续拥挤。

### 页面结构

1. 顶部标题：`模型广场`
2. 一句说明：`选择 Gitee AI 图片模型，加入工作站后可在画布和快捷生成中使用。`
3. 状态条：价格更新时间、我的模型数量、Gitee AI Key 状态、当前余额/次数。
4. 工具条：搜索模型、推荐/全部/我的模型分段、文生图/编辑/低价/高清/中文友好筛选。
5. 模型卡片网格：模型名、模型 ID、一句话简介、官方价、站内预计消耗、优势标签、加入/使用按钮。
6. 详情抽屉：完整简介、官方价和来源、能力限制、默认参数、示例提示词、使用入口。

### 卡片字段

```ts
type ModelCatalogItem = {
  provider: 'gitee-ai'
  modelId: string
  name: string
  category: 'text-to-image' | 'image-edit' | 'upscale' | 'matting' | 'layer'
  summary: string
  strengths: string[]
  useCases: string[]
  capabilities: string[]
  officialPrice: {
    min: number | null
    max: number | null
    unit: 'image' | 'call' | 'layer' | 'second' | 'token' | 'unknown'
    sourceUrl: string
    updatedAt: string
    stale: boolean
  }
  lyraPricing: {
    creditCostPerImage: number | null
    note: string
  }
  defaultParams: Record<string, unknown>
  limitations: string[]
  enabledByDefault: boolean
}
```

## 6. 接入架构

### Channel/Profile 划分

不新增具体供应商 provider。Gitee 只作为 OpenAI-compatible catalog/profile 的一个来源，代码里使用 `openai-compatible`、`channel`、`profile`、`catalog` 等通用命名。`image_generation` 只是上游模型分类或页面锚点。

| channel/profile | base URL | key 分组 | 说明 |
| --- | --- | --- | --- |
| `image-2` | 现有 OpenAI-compatible base URL | codex/codex-pro 分组 Key | 继续保持基础版主线 |
| `image-2-4k` | 生图分组 OpenAI-compatible base URL | 生图分组 Key | 产品展示为 `image-2（满血版）`，可选尺寸/自定义像素 |
| `openai-compatible-catalog` | 管理员配置 | 管理员配置 | 模型广场使用，来源信息只存 `sourceDisplayName` |
| `gif` | 本地模式 | 无上游 Key | 继续独立模块 |

Banana 已拆到 `banana` 分支，不要在本阶段把 Banana 的模型 ID 或规格模式带回 dev。

### 调用链路

Gitee AI 文生图使用 OpenAI 兼容端点：

```text
Lyra 前端/外部 API
  -> Lyra 后端任务队列
  -> channel resolver 解析 channel/profile 与 model=<目录模型 ID>
  -> POST {profile.baseURL}/images/generations
     Authorization: Bearer <profile key>
     X-Failover-Enabled: true
  -> 保存 b64_json 或 URL 结果到现有 outputs/任务结果体系
```

官方文档示例使用 `base_url=https://ai.gitee.com/v1`、`model=Kolors`、`size=1024x1024`，并通过 `extra_body` 传 `num_inference_steps`、`guidance_scale` 等模型参数。Lyra 侧需要把参数校验放在后端 registry，不能只依赖前端表单。

### Key 策略

- 网页用户：优先使用用户云端保存的 Gitee AI Key；允许本地临时 Key 作为开发/试用入口。
- 管理员：可配置系统托管 Gitee AI Key，是否开放给普通用户由后台开关控制。
- 外部 API：调用方只使用 Lyra Bearer Key，不直接提交 Gitee AI Key。
- 日志：不得记录 Gitee AI Key、Authorization 头、完整敏感请求体。

### 官方数据同步

建议后端新增只读同步能力：

- 启动或定时拉取官方 services API，缓存服务清单、`operation_summary`、`props.resolution_price_factors`。
- 详情页或生成前按模型调用 official operations API，刷新操作列表、参数、单价和单位。
- 拉取 billing-unit tags，把 `unit_tag` 映射为 `张`、`图层`、`秒`、`百万 Token`、`次`。
- 缓存失败时继续使用最近一次成功快照，但前端必须显示「价格刷新失败/上次更新时间」。
- 官方返回价格为空、操作为空、模型不可见或 `status` 非可用时，不允许加入工作站。

## 7. 后端改造

建议新增：

```text
internal/imagemodels/
  registry.go        // provider/model/capability 注册表
  gitee.go           // Gitee AI 模型清单和参数策略
  pricing.go         // 官方价、站内消耗、单位映射
  params.go          // 参数 allowlist / extra_body 映射
```

需要改造的后端边界：

- `internal/config`：新增 provider 常量和默认 Gitee base URL。
- `internal/settings`：管理员系统 Gitee AI Key 和是否允许系统托管调用。
- `internal/spaceconfig`：用户云端 Gitee AI Key。
- `internal/api/runtime_secrets.go`：浏览器本地 Gitee AI Key header。
- `internal/newapi/client.go` 或新 provider client：支持 per-provider base URL、`extra_body`、不同 response 字段。
- `internal/jobs/manager.go`：provider/model 走 registry，不继续手写分支。
- `internal/api/v1_image_tasks.go`：外部 API 支持 `provider=gitee-ai` 和 `model=<Gitee 模型 ID>`。
- `data/` 或配置存储：增加模型启用偏好、价格缓存快照和更新时间。

### 参数策略

首阶段通用字段：

- `prompt`
- `model`
- `size`
- `n`
- `extra_body`

Gitee extra body 只允许 registry 中声明的参数透传。未声明参数进入高级 JSON 也必须后端校验，默认拒绝或记录为待支持。不要把 Image-2 的 `quality`、`output_format`、`response_format` 直接套给 Gitee，除非该模型文档明确支持。

示例配置：

```ts
{
  modelId: 'Kolors',
  provider: 'gitee-ai',
  endpoint: '/images/generations',
  supports: {
    textToImage: true,
    imageToImage: false,
    extraBody: true
  },
  params: {
    num_inference_steps: { type: 'number', min: 1, max: 50, default: 20 },
    guidance_scale: { type: 'number', min: 0, max: 20, default: 7.5 }
  }
}
```

## 8. 前端改造

建议新增：

```text
web/src/components/ModelSquarePage.tsx
web/src/components/modelSquare/
  ModelSquareToolbar.tsx
  ModelCard.tsx
  ModelDetailDrawer.tsx
  ModelPrice.tsx
  ModelFilters.tsx
web/src/lib/modelCatalog.ts
web/src/lib/modelPricing.ts
```

改造点：

- 左侧导航增加「模型」。
- 设置页增加 Gitee AI Key：本地、云端、系统托管三层状态。
- 快捷生成模型选择器改成 registry 驱动。
- 创作画布生成节点支持选择已加入模型。
- 提示词助手输出可选择应用到已加入模型。
- 结果/任务详情展示 provider/model 的人类可读名称。
- 价格组件支持加载中、刷新失败、价格为空、部分免费、价格区间、不同计费单位。
- 「加入工作站」只保存模型 ID、provider、用户偏好参数和启用状态；官方价格、能力和参数每次从缓存/接口读取，不复制成长期静态配置。

## 9. API 设计

内部任务创建：

```json
{
  "provider": "gitee-ai",
  "model": "z-image-turbo",
  "mode": "text-to-image",
  "prompt": "一张奶茶海报，夏日清爽风格",
  "ratio": "1:1",
  "resolution": "standard",
  "count": 1,
  "extraBody": {
    "num_inference_steps": 20,
    "guidance_scale": 7.5
  }
}
```

外部 API：

```json
{
  "provider": "gitee-ai",
  "model": "Kolors",
  "prompt": "商品海报，干净背景，中文标题清晰",
  "size": "1024x1024",
  "extra_body": {
    "num_inference_steps": 20,
    "guidance_scale": 7.5
  }
}
```

注意：`extra_body` 只透传到 Gitee/OpenAI-compatible 请求，不应写入日志明文敏感内容。

## 10. 子任务拆分

### 任务 A：官方数据同步

- 拉取官方 services API。
- 拉取 operations API 和 billing-unit tags。
- 过滤图片生成相关模型。
- 映射价格单位和能力标签。
- 生成 `ModelCatalogItem` 缓存快照。

### 任务 B：后端 registry

- 新增 `internal/imagemodels`。
- 保持 Image-2 行为不变。
- 新增 `gitee-ai` provider。
- 支持 `extra_body` 和 provider/model 校验。
- 增加 provider/model 单元测试。

### 任务 C：Key 与配置

- 用户设置增加 Gitee AI Key。
- 管理员设置增加系统 Gitee AI Key。
- 本地浏览器 key header 增加 Gitee AI Key。
- 外部 API 只用 Lyra Bearer Key，不直接接收上游 Key。

### 任务 D：模型广场前端

- 新建 ModelSquarePage。
- 实现卡片、筛选、详情抽屉。
- 实现加入工作站和我的模型。
- 跳转画布/快捷生成并带入模型。

### 任务 E：生成链路

- 快捷生成支持 Gitee 模型。
- 创作画布生成节点支持 Gitee 模型。
- 任务状态、历史、结果展示 provider/model 名称。

### 任务 F：文档与验收

- 更新 API 文档。
- 更新设置说明。
- 写模型广场使用说明。
- 验证 Image-2、GIF、Gitee 文生图互不影响。

## 11. 验收清单

- 模型广场能显示 Gitee AI 官方模型、优势、价格、更新时间和价格来源。
- 价格从运行时缓存或官方接口刷新，不依赖硬编码表。
- `z-image-turbo`、`Qwen-Image`、`Kolors` 至少 3 个模型能加入工作站。
- 未配置 Gitee AI Key 时，页面提示清楚，不创建任务。
- 配置 Gitee AI Key 后，可以创建 Gitee 文生图任务并显示任务 ID、状态、历史和结果。
- 后端请求 Gitee AI 时带 `Authorization: Bearer <profile key>`，并支持 `X-Failover-Enabled: true`。
- 外部 API 支持 `provider=gitee-ai` 和 `model=<模型 ID>`。
- `extra_body` 只允许 registry 声明参数透传，未声明参数被拒绝或明确报错。
- Image-2 原有流程不变。
- GIF 模式不受影响。
- dev 分支不因 Gitee 接入复活 Banana 入口。
- 移动端一列卡片可读，按钮触控高度不小于 44px。

## 12. TODO 与未知项

- 需要确认最终过滤条件：按官方 tag、operation 名称、模型 ID 白名单，还是三者组合。
- 需要确认 Gitee AI 每个图片模型的真实响应格式是否都返回 `b64_json`，还是部分模型返回 URL/异步任务。
- 需要确认用户余额/资源包信息是否需要接入 Gitee 账号态接口；首阶段可以只展示 Key 状态和官方价格。
- 需要确认站内积分倍率、免费额度和管理员兜底 Key 的商业规则。
- 需要逐个模型补充默认参数、最大尺寸、是否支持参考图、是否支持异步。

## 13. 本阶段不做

- 不做视频模型接入。
- 不做所有 Serverless 模型全量开放。
- 不做复杂动态计费系统，只预留站内倍率。
- 不做 Gitee 图生图/编辑全量能力，除非官方 operations 参数和响应格式明确。
- 不把 Gitee Key 暴露给外部 API 调用方。
