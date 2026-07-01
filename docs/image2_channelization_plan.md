# image-2 / image-2-4k 与 OpenAI-compatible 图片渠道预置方案

更新时间：2026-07-01

状态：规划与实现记录。本文用于统一 image-2、image-2-4k 以及后续 OpenAI-compatible 图片模型渠道的命名、预置、请求路径和数据同步边界。

关联背景文档：`docs/gitee_image_model_square_plan.md`。该文档保留供应商模型广场调研细节；本文负责把实现命名收束为通用 OpenAI-compatible 渠道，不把具体供应商暴露成硬编码 provider、client、secret 或模块名。

## 1. 目标

1. 把现有 `image-2` 主线整理为一个 OpenAI-compatible 图片渠道预置，而不是散落在页面、任务和外部 API 的特殊分支。
2. 保留 `image-2-4k` 作为满血版渠道 ID，产品侧展示为 `image-2（满血版）`；复用同一类 OpenAI-compatible 客户端，但不强制 4K，尺寸由用户选择或自定义像素决定。
3. 为 OpenAI-compatible 类型图片渠道预留通用模型目录能力，支持从不同上游目录同步模型、价格、参数和请求路径。
4. 将 Gitee 当前图片模型按请求路径分组纳入目录规划，但代码命名保持 `openai-compatible`、`catalog`、`profile`、`channel` 等通用词。
5. 保持现有外部 API 兼容：`provider=image-2`、`provider=image2`、`model=gpt-image-2` 继续可用。

## 2. 命名规则

### 允许的实现命名

| 层级 | 建议命名 | 说明 |
| --- | --- | --- |
| 渠道类型 | `openai-compatible` | 表示协议和请求形态，不表示供应商 |
| 渠道预置 | `image-2`、`image-2-4k` | 产品侧稳定入口 |
| 客户端 | `OpenAICompatibleImageClient` | 统一处理 `/images/generations` 等路径 |
| 上游配置 | `upstreamProfile`、`catalogProfile` | 表示 base URL、Key、目录同步配置 |
| 模型目录 | `OpenAICompatibleCatalog` | 存储模型、路径、参数、价格快照 |
| 运行时密钥 | `openaiCompatibleImageKey` 或 profile 级 secret | 由 profile 决定实际上游 |

### 避免的实现命名

- 不新增以具体供应商命名的 provider 常量、client 类型、数据库表、secret 字段或任务类型。
- 不把 `image_generation` 当 provider，它只能作为上游目录分类、页面锚点或同步筛选条件。
- 不把供应商模型 ID 写进 `image-2` 或 `image-2-4k` 的逻辑分支。模型能力必须来自渠道预置或目录快照。
- UI 可以展示“来源：Gitee AI”等事实性来源信息，但这属于 `sourceDisplayName`，不是代码路径或配置键。

## 3. 渠道预置

| 渠道 ID | 渠道类型 | 默认请求路径 | 默认模型 | 默认策略 | 适用场景 |
| --- | --- | --- | --- | --- | --- |
| `image-2` | `openai-compatible` | `/images/generations` | 由 profile/catalog 决定，兼容旧 `gpt-image-2` | 不提交 `size`，保留 `quality/output_format` | 默认文生图、外部 API SDK 默认值 |
| `image-2-4k` | `openai-compatible` | `/images/generations` | 由 profile/catalog 决定，兼容旧 `gpt-image-2` | 产品展示为 `image-2（满血版）`，可选预设尺寸，也可自定义 `WIDTHxHEIGHT` | 高清海报、商品图、画布大图 |
| `openai-compatible-custom` | `openai-compatible` | 由 profile 声明 | 用户或管理员配置 | 手动填写 base URL、model、Key | 私有网关、兼容代理、临时测试 |
| `openai-compatible-catalog` | `openai-compatible` | 由模型目录声明 | 目录模型 ID | 从上游目录同步价格、参数、能力 | 模型广场、按模型启用、运营推荐 |

说明：

- `image-2-4k` 不是新 provider，只是同一个 OpenAI-compatible 客户端上的满血版预置；命名保留是为了兼容历史任务、外部 API 和现有渠道配置。
- 外部 API 的 `provider` 字段可继续承担兼容入口，但服务端内部应尽量归一成 `channel_id`。
- `model` 仍表示上游模型 ID。渠道预置可提供默认模型，但不应禁止管理员在 OpenAI-compatible profile 中配置其他模型。

## 4. 请求归一化

内部任务创建建议先归一为：

```json
{
  "channel_id": "image-2",
  "provider_type": "openai-compatible",
  "request_path": "/images/generations",
  "model": "image-2",
  "prompt": "一张干净的产品海报",
  "n": 1,
  "quality": "auto",
  "output_format": "png",
  "extra_body": {}
}
```

兼容映射：

| 输入 | 内部归一 |
| --- | --- |
| `provider=image-2` | `channel_id=image-2` |
| `provider=image2` | `channel_id=image-2` |
| `provider=gpt-image-2` | 兼容旧调用：`channel_id=image-2`，模型按 profile/catalog 解析 |
| `provider=image-2-4k` | `channel_id=image-2-4k`，产品侧展示为 `image-2（满血版）`，按请求中的 `size` 或 `ratio + resolution` 决定尺寸 |
| `provider=openai-compatible` | 必须提供或解析出 `channel_id`、`model`、`upstreamProfile` |
| 仅传 `model=gpt-image-2` | 兼容旧调用：默认 `channel_id=image-2` |

`size`、`quality`、`output_format`、`response_format`、`extra_body` 必须按渠道和模型声明校验。`extra_body` 只允许目录或 profile 中声明的字段透传，不能作为任意 JSON 后门。

## 5. image-2 与 image-2-4k 策略

### `image-2`

- 用作默认文生图渠道。
- 默认请求：`POST {base_url}/images/generations`。
- 默认模型由管理员 profile/catalog 决定；旧 `gpt-image-2` 输入继续兼容。
- 默认 size：基础版不提交 `size`；仅提交 `quality` 和 `output_format`，由上游自动决定画幅。
- 用于现有快捷生成、画布生成、外部 API `/v1/images/generations` 的默认路径。
- 价格与额度沿用现有 image-2 配置，不在模型目录里重复静态写死。

### `image-2-4k` / `image-2（满血版）`

- 用作满血版预置渠道，方便 UI、外部 API 和计费提示区分；内部 ID 仍为 `image-2-4k`。
- 默认请求仍是 `POST {base_url}/images/generations`。
- 默认模型由管理员 profile/catalog 决定；旧 `gpt-image-2` 输入继续兼容。
- 不强制 4K。尺寸策略支持三种路径：`auto` 不传 `size`；预设 `ratio + resolution` 映射为标准/2K/4K 尺寸；自定义像素按 OpenAI 风格 `WIDTHxHEIGHT` 透传。
- UI 必须提示满血版和大尺寸生成的耗时、失败重试和额度消耗可能更高。
- 历史记录需要保存最终 `channel_id=image-2-4k`，便于用户复用同样规格。

## 6. OpenAI-compatible 目录数据模型

建议目录快照使用通用字段：

```ts
type OpenAICompatibleImageModel = {
  catalogProfileId: string
  sourceDisplayName: string
  sourceUrl: string
  modelId: string
  displayName: string
  requestPath: '/images/generations' | '/images/edits' | string
  mode: 'text-to-image' | 'image-edit' | 'layer' | 'segmentation' | 'unknown'
  capabilities: string[]
  defaultParams: Record<string, unknown>
  allowedExtraBody: Record<string, unknown>
  price: {
    min: number | null
    max: number | null
    unit: 'image' | 'call' | 'layer' | 'second' | 'token' | 'unknown'
    updatedAt: string
    stale: boolean
  }
  enabledByDefault: boolean
}
```

要点：

- `catalogProfileId` 是通用 profile ID，不使用供应商名作为业务分支。
- `sourceDisplayName` 只用于 UI 和审计，例如模型详情页展示数据来源。
- `requestPath` 是模型能被调用的路径，模型广场、参数表单和任务队列都应以它为准。
- 价格缓存必须带 `updatedAt` 和 `stale`，不得把一次抓取结果当永久配置。

## 7. Gitee 当前模型按请求路径分组

本节把现有 Gitee 模型调研转换为 OpenAI-compatible 目录分组。实现侧不要新增 `gitee-*` provider 或 client；可通过一个 catalog profile 记录 base URL、目录 API、鉴权 secret 和展示名称。

### 目录同步路径

| 目录用途 | 请求路径 | 内容 |
| --- | --- | --- |
| 服务清单 | `GET /api/pay/services?type=serverless&status=1&size=1000` | 模型 ID、服务状态、价格摘要、分类 |
| 操作明细 | `GET /api/pay/service/operations?service_ident=<modelId>` | 操作名、参数、价格单位、分辨率档位 |
| 计费单位 | `GET /api/base/tags?category_slugs=billing-unit` | 张、次、图层、秒、Token 等单位映射 |

这些路径只属于 catalog sync，不进入生成任务的 `requestPath`。

### `POST /images/generations`

首阶段可进入 OpenAI-compatible 文生图目录的模型：

| 模型 ID | 建议定位 | 首阶段状态 |
| --- | --- | --- |
| `z-image-turbo` | 默认推荐/高速质量平衡 | 可候选启用 |
| `Qwen-Image` | 中文与文字渲染 | 可候选启用 |
| `qwen-image-2.0` | 高质量多模态生成 | 可候选启用 |
| `Qwen-Image-2512` | 新版 Qwen 图像生成 | 可候选启用 |
| `FLUX.1-schnell` | 低价快速试稿 | 可候选启用 |
| `FLUX.1-dev` | 通用高质量创作 | 可候选启用，编辑能力后续再开 |
| `FLUX.2-dev` | 专业创意设计 | 可候选启用 |
| `FLUX.2-klein-4B` | 低成本快速试稿 | 可候选启用 |
| `FLUX.2-klein-9B` | 快速高质量折中 | 可候选启用 |
| `CogView4_6B` | 低价中文文生图 | 可候选启用 |
| `Kolors` | 中文理解/风格能力 | 可候选启用 |
| `stable-diffusion-3.5-large-turbo` | Stable Diffusion 系通用模型 | 可候选启用 |
| `HiDream-I1-Full` | 高质量基础生图 | 可候选启用 |
| `GLM-Image` | 国产基础生图 | 可候选启用 |
| `LongCat-Image` | 通用生图 | 可候选启用 |

调用形态：

```text
POST {base_url}/images/generations
Authorization: Bearer <profile secret>
X-Failover-Enabled: true
```

请求体保持 OpenAI-compatible 字段：`model`、`prompt`、`size`、`n`，模型特有参数放入经过 allowlist 校验的 `extra_body`。

### `POST /images/edits` 或目录声明的编辑路径

以下模型只在 operations 明确返回编辑类路径、参数和响应格式后启用：

| 模型 ID | 建议定位 | 首阶段状态 |
| --- | --- | --- |
| `Qwen-Image-Edit` | 图像编辑 | 待确认路径和参数 |
| `Qwen-Image-Edit-2511` | 新版图像编辑 | 待确认路径和参数 |
| `FLUX.1-Kontext-dev` | 上下文感知改图 | 待确认路径和参数 |
| `Kolors` | 后续编辑扩展 | 文生图先启用，编辑待确认 |

如果上游没有 OpenAI-compatible 标准编辑路径，不要在代码里硬造固定路径。目录快照应保存实际 `requestPath`，前端按 `mode=image-edit` 和能力开关展示。

### 非标准工具路径

以下模型不进入首阶段 OpenAI-compatible 生图预置：

| 模型 ID | 类型 | 处理方式 |
| --- | --- | --- |
| `Qwen-Image-Layered` | 图像分层 | 等目录返回稳定路径后进入工具类能力 |
| `SAM 3` / `sam3` | 图像分割 | 等目录返回稳定路径后进入结果页工具 |
| 视频模型 | 视频/动图 | 放入视频或 GIF 规划，不混入图片渠道 |
| OCR、检测、分类、文档处理模型 | 非生图 | 不进入图片生成首屏 |

## 8. 配置与密钥边界

配置建议分三层：

1. 默认 profile：承载现有 `image-2` / `image-2-4k` 的 base URL 和 Key。
2. 管理员 profile：配置额外 OpenAI-compatible 上游、模型目录、系统托管 Key 和开放范围。
3. 用户 profile：用户自己的 OpenAI-compatible Key 或临时本地 Key。

密钥日志规则：

- 不记录 Authorization 头。
- 不记录完整上游请求体中的敏感字段。
- 任务历史只保存 `channel_id`、`model`、`requestPath`、规格和结果摘要。
- 外部 API 调用方只使用 Lyra Bearer Key，不直接提交或读取上游 Key。

## 9. 实施顺序

1. 建立渠道归一层：把现有 `provider`、`model`、`ratio`、`resolution` 归一成 `channel_id`、`provider_type`、`requestPath`、`size`。
2. 添加 `image-2-4k` / `image-2（满血版）` 预置：只改渠道配置、UI/计费提示和尺寸选择，不复制一套客户端。
3. 抽出 OpenAI-compatible 图片客户端：统一处理 base URL、请求路径、鉴权、响应解析和错误映射。
4. 增加目录快照结构：支持模型按请求路径、mode、参数和价格分组。
5. 接入 catalog profile 同步：先只读同步和展示，确认后再允许加入工作站。
6. 放开目录模型生成：只对 `POST /images/generations` 且参数/响应已确认的模型启用。
7. 外部 API 文档同步：说明 `image-2`、`image-2-4k` 和 `openai-compatible` 的兼容关系。

## 10. 验收清单

- `provider=image-2`、`provider=image2`、`model=gpt-image-2` 的旧请求仍能落到 `channel_id=image-2`。
- `provider=image-2-4k` 能创建满血版任务，支持预设尺寸、自定义 `WIDTHxHEIGHT` 和 `auto`，并在历史记录里保留 `channel_id=image-2-4k`。
- `image-2` 和 `image-2-4k` 复用同一个 OpenAI-compatible 客户端。
- 目录模型按 `requestPath` 分组，前端不需要靠供应商名判断能力。
- Gitee 当前文生图模型归入 `/images/generations`，编辑/分层/分割模型保持待确认。
- 代码命名不出现硬编码供应商 provider/client/secret/table。
- UI 允许展示上游来源名称和价格来源链接，但任务队列、API、配置键保持通用命名。
- `extra_body` 只允许按模型声明透传，未知参数被拒绝或进入待支持错误。
- 价格、单位、参数和模型可用性都有更新时间，缓存失效时前端明确提示。

## 11. 风险与注意事项

- 部分 OpenAI-compatible 上游只兼容生成接口，不一定兼容编辑、图层、分割等工具路径。
- 同名模型在不同 profile 下能力和价格可能不同，模型唯一键应包含 `catalogProfileId + modelId + requestPath`。
- `image-2（满血版）` 以及大尺寸输出的成本和失败率可能明显高于默认渠道，需要单独的 UI 提示和额度策略。
- 上游目录价格可能随时间变化，不能把 2026-07-01 的调研价格写成长期配置。
- 供应商目录和生成接口可能使用不同 base URL，profile 需要分别保存 `catalogBaseUrl` 与 `apiBaseUrl`。

