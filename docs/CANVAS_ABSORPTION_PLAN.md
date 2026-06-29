# 创作画布吸收方案

更新时间：2026-06-29

本文用于把 `basketikun/infinite-canvas` 参考图里的优秀交互吸收到 Lyra Image Workbench。目标不是照搬上游代码，而是把“画布即创作主入口、节点关系可生成、结果可继续衍生”的产品逻辑落成 Lyra 自己的可持久化、可审计、可任务化方案。

## 1. 总判断

这组参考图值得吸收，核心价值有三点：

1. 用户不是在表单里填参数，而是在画布里组织素材、文字、生成配置和结果关系。
2. 每次生成不是孤立任务，而是从上游文本、参考图、配置节点流向结果图，链路可见、可复用。
3. 操作入口都贴近当前选中对象，常用动作不被塞到右侧复杂面板里。

Lyra 当前已经有图片节点、文字节点、连线、右侧预览、参考图和本地草稿，但还差三个关键层：

- 画布项目没有服务端持久化，刷新/跨设备/任务追溯不够稳。
- 画布中的关系还没有变成强结构化的生成上下文。
- 当前实现仍集中在 `NodeWorkflowPage.tsx`，页面、引擎、节点、边、工具条、任务绑定需要拆开。

建议把创作画布定为主入口，但不要一次性做成复杂专业节点编辑器。第一阶段先做“轻节点画布 + 生成配置节点 + 任务结果分支 + 后端保存”的闭环。

## 2. 只吸收思路，不复制代码

`basketikun/infinite-canvas` 是 AGPL-3.0 项目，Lyra 只能吸收产品逻辑、交互模式和信息架构，不复制其源码、组件结构、样式文件或具体实现。

还要注意两者定位不同：

| 维度 | infinite-canvas | Lyra |
| --- | --- | --- |
| 部署 | 本地/个人优先 | 自部署/站点化/多用户 |
| Key | 偏浏览器本地配置 | 后端任务链路，支持用户 Key 和系统 Key |
| 数据 | 浏览器本地项目为主 | 用户、空间、任务、素材、广场、额度、日志要服务端保存 |
| 任务 | 画布内直接生成体验强 | 必须走 Lyra 任务队列、额度和审计 |
| 风险 | 适合可信本机 | 要防越权引用、误扣费、公开素材泄漏 |

因此 Lyra 的落点应是：学习“画布工作流”，但保留自己的账号、额度、任务、广场和日志体系。

## 3. 要吸收的交互模块

### 3.1 无限网格画布

吸收点：

- 画布占据主区域，不再被右侧和底部面板挤成小窗口。
- 背景使用轻量网格，帮助用户感知空间位置。
- 支持平移、滚轮缩放、拖拽、粘贴图片、历史图拖入。
- 左下角显示缩放比例和缩放按钮，避免用户迷路。

Lyra 落地：

- 桌面端画布优先填满可用空间，底部输入和参数不遮挡画布。
- 1080p 屏幕下，一屏应同时看到画布、输入框、生成按钮和必要参数。
- 移动端不强塞完整无限画布，先做轻编辑和查看：添加参考、查看节点、提交生成。

### 3.2 浮动对象工具条

参考图中选中节点后，上方浮出一排动作：信息、删除、保存素材、编辑、编辑文字、生图、缩小、放大。

吸收点：

- 操作跟随对象出现，用户不用去右侧找按钮。
- 图片节点和文字节点显示不同动作。
- 删除、缩放、编辑是高频动作，必须近手。

Lyra 落地：

- 图片节点工具条：查看信息、作为参考图、编辑图片、保存素材、生成变体、缩小、放大、删除。
- 文字节点工具条：编辑文字、优化提示词、生成图片、缩小、放大、删除。
- 结果节点工具条：放大查看、下载、继续图生图、提交广场、保存素材、删除。
- 删除仍支持键盘 `Backspace/Delete`，工具条只是可见入口。

不要吸收：

- 不做过多图标堆叠，第一版最多 6 到 8 个动作，其他进右键菜单。
- 不用硬编码黑色工具条，颜色跟随系统主题。

### 3.3 文本节点

吸收点：

- 用户可以在画布上放多个文字块，分别表达主体、风格、限制、修改意见。
- 文本节点可以连接到生成配置节点或图片节点，表示它影响哪个生成。

Lyra 落地：

- 右键画布新建文字，默认进入编辑状态。
- 文字节点不预填提示词，不自带废话。
- 文本节点支持 `Ctrl+Enter` 结束编辑，`Esc` 取消焦点。
- 文字节点可被选中、删除、缩放、移动。
- 文字内容进入生成上下文，但保留节点 ID，方便以后追溯是哪段文字影响了任务。

### 3.4 图片节点

吸收点：

- 图片是画布里的第一等对象，不只是上传列表里的缩略图。
- 历史结果、外部拖入、粘贴图片、上传素材都能变成画布节点。
- 原图比例和分辨率要保留，不能被拉伸变形。

Lyra 落地：

- 外部图片拖入画布即创建图片节点，并上传/缓存为真实素材。
- 粘贴图片时，如果焦点在画布，直接贴到画布中心或鼠标附近。
- 历史图片点击后更新右侧预览，同时可拖入画布作为参考。
- 图片节点保存 `assetId/uploadId/resultId`、原始宽高、缩略图、来源、prompt 快照。
- 图生图任务提交到广场时，参考图原图也应保存并可复用；否则用户无法理解“这张成品是怎么来的”。

参考图策略：

- 画布内复用使用服务端素材 ID，不建议把大图长期用 base64 存到画布 JSON。
- base64 只适合临时剪贴板或离线导入导出，不适合作为长期存储和 API payload。
- 广场公开详情可以展示参考图缩略图和必要原图，但需要明确用户提交确认。

### 3.5 生成配置节点

参考图里最值得吸收的是“生成配置节点”。它把模型、模式、比例、数量、参考图数量、提示词数量和开始生成集中成一个轻节点。

吸收点：

- 生成不再只靠底部表单，而是画布上有一个明确的“生成节点”。
- 文本节点和图片节点连到它，它再生成结果节点。
- 每个生成节点都有自己的参数和上游输入。

Lyra 落地：

- 新增 `generation` 节点类型。
- 节点内显示：
  - 模式：文生图 / 图生图 / GIF
  - 模型
  - 比例
  - 数量
  - 参考图数量
  - 提示词数量
  - 开始生成按钮
- 点击开始生成前弹出轻确认，不做大弹窗：
  - 将使用哪些文字
  - 将使用哪些参考图
  - 预计消耗
  - 使用用户 Key 还是系统 Key
- 生成后在配置节点右侧自动铺开结果节点。

不要吸收：

- 不把所有参数都塞进节点内部。高级参数可以折叠或由底部/右侧 inspector 编辑。
- 不做复杂脚本节点、插件节点、条件节点，第一阶段只服务生图闭环。

### 3.6 曲线连线和关系标签

吸收点：

- 连线表达“谁影响谁”，不是装饰线。
- 线上的文字说明关系，例如“主体参考”“眼睛细节”“海报文案”“风格来源”。
- 选中线后可以编辑或删除。

Lyra 落地：

- 连线必须可选中，`Backspace/Delete` 可删除。
- 连线标签支持点击编辑，输入的文字进入生成上下文。
- 拖动一个节点靠近另一个节点时可以吸附，自动创建默认关系线。
- 关系类型第一版只保留简化集合：
  - 参考
  - 主体
  - 风格
  - 细节
  - 文案
  - 自定义标签
- 不再强制用户选择过细的“服饰、地板材质、背景构图、光线”等标签，这些让用户自己用文字表达。

生成上下文规则：

- 文本 -> 生成配置：作为主要提示词片段。
- 图片 -> 生成配置：作为图生图参考。
- 图片 -> 图片：表示后者参考前者的主体、风格或细节。
- 文字 -> 图片：表示这段文字是对该图的修改意见。
- 生成配置 -> 结果：保存任务来源关系。

### 3.7 结果分支

吸收点：

- 结果图不只是右侧历史，它应该出现在画布上，成为下一轮创作素材。
- 一次生成多张图时，结果在配置节点右侧平铺成分支，用户能比较。

Lyra 落地：

- 任务完成后，按结果数量创建 `result` 节点。
- 每个结果节点保存 `taskId`、`resultIndex`、`imageUrl`、`promptSnapshot`、`referenceSnapshot`。
- 结果节点可以继续连接到新的生成配置节点，形成迭代链路。
- 点击结果节点，右侧预览或 inspector 显示完整图、提示词、参考图、任务状态和提交广场入口。

### 3.8 底部工具条

参考图底部有居中的快捷工具条，用于添加元素、选择工具、创建生成节点。

Lyra 落地：

- 底部工具条只放高频动作：
  - 选择
  - 文字
  - 图片
  - 生成配置
  - 连接
  - 撤销
  - 重做
- 生成输入框可以在底部，但不要和工具条混成一堆。
- 如果底部空间紧张，工具条居中悬浮，输入框作为可展开 composer。

### 3.9 顶部和侧边入口

参考图里顶部保留画布名、设置、主题、GitHub、助手等轻入口。Lyra 已经有左侧导航，这部分要按 Lyra 自己的信息架构落。

Lyra 落地：

- 左上：站点名和当前空间/画布名。
- 左侧栏：画布、快捷生成、Agent、提示词库、广场、GIF、API 文档、我的、充值、管理后台。
- 左下：余额、当前用户、主题、GitHub、退出登录。
- 画布内部右上只保留视图相关：小地图、适配视图、设置。

## 4. 推荐的第一阶段目标

第一阶段目标：逻辑完整闭环，UI 简洁，创作画布真正成为主入口。

成功标准：

1. 用户能把图片、文字、历史结果放到画布。
2. 用户能用连线表达参考关系。
3. 用户能创建生成配置节点并发起真实任务。
4. 任务结果能回到画布，成为结果节点。
5. 刷新页面后画布状态不丢。
6. 任务详情能看到当时的文字、参考图、关系和参数。
7. 提交广场时，成品图、提示词、参考图和任务来源一并保存。

## 5. 分期规划

### P0：画布生成闭环

必须做：

- 服务端画布项目保存。
- 节点类型：`text`、`image`、`generation`、`result`。
- 连线类型和连线标签。
- 选中、删除、移动、缩放、旋转稳定。
- 外部拖入/粘贴图片创建节点。
- 历史结果拖入画布。
- 生成配置节点创建真实后台任务。
- 任务完成后结果节点回流。
- 任务保存 reference snapshot。
- 画布刷新后恢复。
- 主题适配，不出现绿色主题下黑色按钮看不清的问题。

不做：

- 复杂节点类型。
- 自动排版全部场景。
- MCP/本地 Agent 操作画布。
- 多人协作。

### P1：画布可用性增强

- 浮动工具条。
- 自动吸附和自动连线。
- 框选、多选、复制、粘贴。
- 撤销/重做。
- 小地图和适配视图。
- 结果分支自动布局。
- 存素材、复用素材。
- 画布生成提示词：根据节点和连线生成可编辑 prompt。
- 右键菜单补充：新建文字、作为参考图、生成变体、优化文字。

### P2：高级创作能力

- 画布内 Agent：只读画布快照，给修改建议，用户确认后写入。
- JSON 导入导出。
- 分组/图层。
- 局部编辑/区域 mask。
- 模板化工作流。
- 多画布项目管理。
- 协作和评论。

## 6. 数据模型建议

### 6.1 CanvasProject

```ts
type CanvasProject = {
  id: string
  ownerUserId: string
  spaceToken: string
  title: string
  viewport: CanvasViewport
  nodes: CanvasNode[]
  edges: CanvasEdge[]
  assets: CanvasAssetRef[]
  createdAt: string
  updatedAt: string
}
```

### 6.2 CanvasNode

```ts
type CanvasNode =
  | CanvasTextNode
  | CanvasImageNode
  | CanvasGenerationNode
  | CanvasResultNode
  | CanvasGroupNode

type CanvasNodeBase = {
  id: string
  type: string
  x: number
  y: number
  width: number
  height: number
  rotation: number
  selected?: boolean
  zIndex: number
  createdAt: string
  updatedAt: string
}
```

### 6.3 图片、生成、结果节点

```ts
type CanvasImageNode = CanvasNodeBase & {
  type: 'image'
  assetId: string
  source: 'upload' | 'history' | 'result' | 'clipboard'
  naturalWidth: number
  naturalHeight: number
  thumbnailUrl: string
  originalUrl?: string
}

type CanvasGenerationNode = CanvasNodeBase & {
  type: 'generation'
  mode: 'text-to-image' | 'image-to-image' | 'gif'
  provider: string
  model: string
  ratio: string
  quality: string
  outputFormat: string
  count: number
  status: 'idle' | 'ready' | 'creating' | 'running' | 'completed' | 'failed'
  taskIds: string[]
}

type CanvasResultNode = CanvasNodeBase & {
  type: 'result'
  taskId: string
  resultIndex: number
  imageUrl: string
  promptSnapshot: string
  referenceSnapshotIds: string[]
}
```

### 6.4 CanvasEdge

```ts
type CanvasEdge = {
  id: string
  fromNodeId: string
  toNodeId: string
  role: 'reference' | 'subject' | 'style' | 'detail' | 'copy' | 'custom'
  label?: string
  createdAt: string
  updatedAt: string
}
```

### 6.5 CanvasGenerationContext

生成前由后端或前端 service 聚合：

```ts
type CanvasGenerationContext = {
  projectId: string
  generationNodeId: string
  promptParts: Array<{
    nodeId: string
    text: string
    edgeLabel?: string
  }>
  references: Array<{
    nodeId: string
    assetId: string
    role: string
    edgeLabel?: string
    snapshotId: string
  }>
  parameters: {
    mode: string
    provider: string
    model: string
    ratio: string
    quality: string
    count: number
  }
}
```

## 7. 后端落地策略

新增模块建议：

```text
internal/canvas/types.go
internal/canvas/store.go
internal/canvas/service.go
internal/canvas/context.go
internal/canvas/snapshots.go
internal/api/canvas.go
internal/api/canvas_test.go
```

职责：

- `types.go`：项目、节点、边、视口、快照类型。
- `store.go`：按 user/space 保存画布项目。
- `service.go`：权限、保存、读取、更新、删除。
- `context.go`：把节点和边聚合成生成上下文。
- `snapshots.go`：任务引用快照和广场提交快照。
- `api/canvas.go`：REST API。

建议 API：

```http
GET /api/canvas/projects
POST /api/canvas/projects
GET /api/canvas/projects/{projectId}
PUT /api/canvas/projects/{projectId}
PATCH /api/canvas/projects/{projectId}/viewport
POST /api/canvas/projects/{projectId}/nodes
PATCH /api/canvas/projects/{projectId}/nodes/{nodeId}
DELETE /api/canvas/projects/{projectId}/nodes/{nodeId}
POST /api/canvas/projects/{projectId}/edges
PATCH /api/canvas/projects/{projectId}/edges/{edgeId}
DELETE /api/canvas/projects/{projectId}/edges/{edgeId}
POST /api/canvas/projects/{projectId}/generation-nodes/{nodeId}/tasks
```

任务创建规则：

- 前端提交 generation node id。
- 后端读取画布项目，校验节点、边、素材权限。
- 后端生成 reference snapshot。
- 后端复用现有 jobs manager 创建任务。
- 任务元数据写入：
  - `source = "canvas"`
  - `canvasProjectId`
  - `canvasGenerationNodeId`
  - `referenceSnapshotIds`
- 任务完成后，前端根据 task id 创建 result node，或由后端返回建议布局。

## 8. 前端模块拆分

当前 `NodeWorkflowPage.tsx` 体积太大，后续不要继续堆。建议拆成：

```text
web/src/components/canvas/CanvasPage.tsx
web/src/components/canvas/CanvasShell.tsx
web/src/components/canvas/CanvasStage.tsx
web/src/components/canvas/CanvasGrid.tsx
web/src/components/canvas/CanvasNodeLayer.tsx
web/src/components/canvas/CanvasEdgeLayer.tsx
web/src/components/canvas/CanvasFloatingToolbar.tsx
web/src/components/canvas/CanvasBottomToolbar.tsx
web/src/components/canvas/CanvasZoomControls.tsx
web/src/components/canvas/CanvasInspector.tsx
web/src/components/canvas/GenerationNode.tsx
web/src/components/canvas/ImageNode.tsx
web/src/components/canvas/TextNode.tsx
web/src/components/canvas/ResultNode.tsx
web/src/components/canvas/CanvasContextMenu.tsx
web/src/components/canvas/hooks/useCanvasStore.ts
web/src/components/canvas/hooks/useCanvasInteractions.ts
web/src/components/canvas/hooks/useCanvasKeyboard.ts
web/src/components/canvas/hooks/useCanvasPersistence.ts
web/src/components/canvas/hooks/useCanvasGeneration.ts
web/src/components/canvas/model/types.ts
web/src/components/canvas/model/context.ts
web/src/components/canvas/model/layout.ts
web/src/api/canvas.ts
web/src/api/contracts/canvas.ts
```

拆分原则：

- 页面容器只负责拼布局和路由状态。
- CanvasStage 只负责画布坐标系、拖拽、缩放、平移。
- Node 组件只渲染自己，不直接创建任务。
- Edge 组件只渲染线和标签，不懂任务。
- Generation service 负责把画布状态转成任务请求。
- API 层和类型层独立，前后端契约清楚。

## 9. UI 结构方案

### 9.1 桌面端

```text
┌────────────────────────────────────────────────────────────────────────────┐
│ 左侧导航 │ 画布标题 / 保存状态                         视图/主题/设置       │
├─────────┼──────────────────────────────────────────────────────────────────┤
│         │                                                                  │
│         │                    无限网格画布                                  │
│         │        text -> generation -> result/result/result                 │
│         │                                                                  │
│         │                                                                  │
│         │  缩放控件                                         右侧轻 inspector │
├─────────┼──────────────────────────────────────────────────────────────────┤
│ 左下用户 │          底部工具条             输入框 / 生成按钮 / 参数摘要      │
└─────────┴──────────────────────────────────────────────────────────────────┘
```

核心要求：

- 画布是最大区域。
- 右侧 inspector 可折叠，不能长期占掉主要空间。
- 底部输入框是辅助，不再主导整个页面。
- 生成配置节点在画布里可见，用户知道任务从哪里来。

### 9.2 移动端

移动端不做完整桌面画布复刻：

- 默认显示“当前画布预览 + 节点列表 + 生成按钮”。
- 支持上传/粘贴/选择历史图。
- 支持编辑文本节点和生成配置。
- 复杂拖拽、连线和大范围布局可弱化为列表编辑。

移动端验收：

- 390x844 不重叠。
- 输入框、上传、生成按钮不被底部栏挡住。
- 选中图片能看到完整预览和删除按钮。

## 10. 与 Agent/GIF/广场的边界

### Agent

- Agent 独立模块，不嵌入画布。
- Agent 可以把方案发送到画布，创建文字节点和生成配置节点。
- 画布可以把当前结构摘要发送给 Agent，让 Agent 生成提示词或修改建议。
- Agent 不直接移动、删除、缩放画布元素，必须用户确认。

### GIF

- GIF 是生成配置节点的一种模式，不属于视频模块。
- GIF 节点需要支持：上传一张图 + 动作描述 + 模板。
- 第一版 GIF 可以从画布图片节点进入，创建 mode = `gif` 的 generation node。
- 仍然要走独立 GIF 后端任务，不调用已删除的视频流程。

### 广场

提交广场时应保存：

- 成品图。
- 提示词。
- 模型、比例、质量、数量。
- 参考图原图或可复用素材引用。
- 画布关系摘要。
- 任务 ID。

如果不保存参考图，用户无法复用图生图作品。建议提交确认里明确显示“将公开成品图、提示词和参考图”。

## 11. 关键风险

### 11.1 范围膨胀

不能一次性做完整 Figma/ComfyUI。第一阶段只做生图闭环，其他都后置。

### 11.2 性能

风险：

- 大图太多导致内存高。
- 连线每次拖动重绘卡顿。
- base64 存储导致画布 JSON 膨胀。
- 结果节点太多导致 DOM 过重。

策略：

- 节点显示缩略图，原图由素材服务管理。
- 画布保存 ID 和元数据，不保存大图 base64。
- 连线层用 SVG 或 canvas 单独渲染。
- 大项目后续做虚拟化或分组折叠。

### 11.3 数据一致性

风险：

- 用户删了参考图，历史任务无法解释。
- 用户刷新后本地草稿和服务端画布冲突。
- 任务完成后 result node 没写回。

策略：

- 任务创建时保存 reference snapshot。
- 服务端项目为准，本地只做临时草稿。
- result node 回流要幂等，重复轮询不能重复创建节点。

### 11.4 主题与 UI

此前绿色主题、API 文档、按钮错位反复出问题。画布重构必须走主题 token：

- 不写死 `#000`、`#fff` 作为按钮底色。
- 选中态、连线、工具条、浮层都走 `--primary`、`--surface`、`--text`、`--border`。
- 每个主题都验收：白蓝、白紫、黑紫、绿色、粉色、蓝色。

## 12. 验收清单

P0 必须人工验收：

- 外部图片拖入画布后创建节点，预览不变形。
- 画布空白处 `Ctrl+V` 可粘贴图片。
- 右键可新建文字，文字不自带提示词。
- 选中节点后浮动工具条位置正确，不错位。
- 点击节点，滚轮缩放节点大小。
- `Backspace/Delete` 删除选中节点或线。
- 拖动节点靠近另一个节点，可吸附并创建连线。
- 线条可选中，可编辑标签，可删除。
- 生成配置节点能读取上游文本和图片。
- 开始生成返回真实 task id。
- 生成结果回到画布成为 result node。
- 切到其他页面再回来，画布状态不丢。
- 刷新页面后，画布项目从后端恢复。
- 任务详情能展示参考图、提示词、模型、参数。
- 提交广场时能看到参考图和提示词确认。
- 绿色主题、暗色主题下按钮和文字不糊。
- 1366x768、1920x1080、390x844 都不重叠。

自动化建议：

```powershell
go test ./internal/canvas ./internal/api ./internal/jobs
cd web
npm run build
```

浏览器检查：

- 桌面 1366x768。
- 桌面 1920x1080。
- 手机 390x844。
- 主题切换后截图对比。

## 13. 推荐实施顺序

1. 冻结数据模型和 API 契约。
2. 新建 `internal/canvas` 和 `web/src/api/canvas.ts`。
3. 把当前 localStorage 草稿升级为服务端项目保存。
4. 拆分 `NodeWorkflowPage.tsx`，先保留现有功能不丢。
5. 新增 generation node。
6. 将画布上下文接入现有任务创建。
7. 任务完成后创建 result node。
8. 加浮动工具条、缩放控件、底部工具条。
9. 补主题、移动端、性能验收。
10. 再做 Agent/GIF/广场的深度互通。

## 14. 子代理拆分计划

这一轮建议继续按小任务并行：

| 子任务 | 角色 | 权限 | 交付 |
| --- | --- | --- | --- |
| 当前画布模块地图 | codebase-cartographer | 只读 | 文件入口、状态流、拆分建议 |
| 画布数据模型/API | api-contract-engineer | 只读或独占写 `internal/canvas` | 类型和接口 |
| 前端画布壳拆分 | frontend-engineer | 独占写 `web/src/components/canvas/*` | CanvasShell/Stage/Toolbar |
| 节点组件拆分 | frontend-engineer | 独占写 `web/src/components/canvas/nodes/*` | Text/Image/Generation/Result |
| 连线与快捷键 | frontend-engineer | 独占写 interactions/hooks | 选线、删线、吸附、键盘 |
| 任务接入 | fullstack-integrator | 独占写 canvas generation service/API glue | 创建任务和 result 回流 |
| 主题和移动端 | ui-ux-reviewer/qa | 只读或样式独占 | 主题问题和响应式修复 |
| 广场提交快照 | backend-engineer | 独占写 promptsquare/canvas snapshot | 参考图和 prompt 公开策略 |

并行规则：

- 同一个文件只能一个写代理。
- `NodeWorkflowPage.tsx` 拆分阶段由一个集成代理负责，其他代理只写新目录。
- 后端 `internal/canvas` 由一个后端代理独占，避免数据模型冲突。
- 主题验收代理只提问题，不直接改全局 token，除非单独授权。

## 15. 决策建议

建议吸收，但按“轻画布工作流”吸收，不按“完整节点编辑器”吸收。

第一版最值得做的是：

1. 生成配置节点。
2. 结果自动分支。
3. 浮动对象工具条。
4. 服务端画布项目保存。
5. 任务 reference snapshot。

这五项做完，Lyra 的画布就会从“把参考图摆一摆”升级为“能解释、能生成、能复用、能发布”的主入口。后续再加 Agent 操作画布、模板化、GIF 动图节点和高级自动布局，会更稳。

## 16. 子代理回收结论

本轮已并行回收 6 个只读子代理结论，统一判断如下：

- 当前方向是对的：Lyra 应吸收“提示词/参考图 -> 生成配置 -> 批量结果 -> 继续引用迭代”的无限画布生成工作流。
- 当前最大问题不是少几个按钮，而是缺少服务端画布项目、生成配置节点、任务快照、结果分支和模块化边界。
- 不建议引入重型流程图库重写全部逻辑。第一阶段先自研轻量 camera/viewport、节点、边和配置节点，保留现有任务链路。
- `NodeWorkflowPage.tsx` 不能继续堆功能，下一轮应先“提取边界，不改行为”，再新增能力。
- 不要把整份画布塞进 `jobs.Job`。正确做法是新增 `canvas/projects` 聚合模型，任务只保存 `canvasBinding/contextSnapshotId`。
- 参考图复用应走服务端素材 ID 和快照，长期存储不使用 base64。
- 生成上下文应从“全画布所有参考图”改成“连入生成配置节点的文字和图片”，这才符合节点工作流。
- 无限画布前要先引入 world/viewport 坐标模型，否则拖拽、缩放、连线命中和粘贴落点会继续出错。

## 17. P0 实施顺序

第一阶段不要同时铺太多功能，按下面顺序执行：

### 17.1 提取边界，不改行为

先把当前大页面拆出来，但保持现有体验不变：

- `NodeWorkflowPage.tsx` 变成页面壳。
- 新增 `useCreativeCanvasController` 统一管理状态和命令。
- 抽出 `CanvasStage`、`CanvasItemNode`、`CanvasConnectionLayer`。
- 抽出 `CanvasInspector`、`ReferenceStrip`、`HistoryPanel`、`RenderPreview`、`CanvasComposer`。
- 抽出 `useCanvasPersistence`、`usePreviewUrls`、`useCanvasKeyboard`、`useCanvasInteractions`。

这一阶段的验收标准是：页面看起来基本不变，但文件边界变清楚，后续不会继续在一个文件里硬塞。

### 17.2 无限画布基础视口

新增 `world -> viewport` camera：

- 画布坐标使用 world 坐标。
- 视口使用 `translate3d + scale`。
- 支持滚轮缩放、空格/中键平移、重置视图。
- 缩放范围建议 20% 到 200%。
- 连线、标签、选择框和命中区随缩放正确对齐。

非目标：

- 不做复杂小地图。
- 不做多人协作。
- 不做完整白板能力。

### 17.3 生成配置节点

新增 `generation` 节点：

- 默认显示：模式、模型、比例、数量、参考图数量、提示词数量、开始生成。
- 高级参数折叠，避免节点变成噪音控制台。
- 文字节点和图片节点连入生成配置节点，才进入该任务上下文。
- 无提示词、图生图缺参考图、缺 Key、额度不足时，节点内直接显示可行动状态。

第一版可以只支持一个活跃配置节点，后续再支持多配置节点并行。

### 17.4 多输入引用链路

提交任务时只读取连入配置节点的输入：

- 文本节点 -> 生成配置：作为 prompt parts。
- 图片节点 -> 生成配置：作为 upload/reference ids。
- 连线标签 -> 进入 reference usage note。
- 断开连接后，统计和 payload 立即更新。

这能支持用户说的场景：一张美女图、一张眼睛特写图、线标注“眼睛”，AI 就知道主体参考美女，眼睛参考细节图。

### 17.5 结果自动落画布

任务完成后自动创建结果节点：

- `count = N` 时创建 N 个 `result` 节点。
- 节点自动排在生成配置节点右侧。
- 配置节点连到每个结果节点。
- 结果节点保存 `taskId`、`resultIndex`、prompt、参数摘要、reference snapshot。
- 点击结果可作为下一轮参考图。
- 失败结果显示占位和错误状态，不破坏已有画布。

## 18. API 契约补充

推荐新增画布 API，不破坏现有 `/api/background-tasks` 和 `/v1/image-tasks`。

```http
GET /api/canvas/projects?limit=50
POST /api/canvas/projects
GET /api/canvas/projects/{projectId}
PATCH /api/canvas/projects/{projectId}
POST /api/canvas/projects/{projectId}/snapshots
POST /api/canvas/projects/{projectId}/tasks
POST /api/canvas/projects/{projectId}/nodes/from-task-result
POST /api/canvas/projects/{projectId}/imports/prompt-square
POST /api/uploads/reference/from-result
POST /api/uploads/reference/from-prompt-square
```

关键规则：

- `PATCH /api/canvas/projects/{projectId}` 使用 `revision` 乐观锁，避免多标签页互相覆盖。
- `POST /snapshots` 生成不可变上下文快照，返回 `snapshotId`、`contextHash`、`resolvedPrompt`、`uploadIds`。
- `POST /tasks` 从 snapshot 创建任务，内部复用现有 jobs manager。
- 任务只保存轻量绑定：

```ts
type CanvasTaskBinding = {
  projectId: string
  snapshotId: string
  sourceNodeIds: string[]
  targetNodeId?: string
  createdNodeId?: string
  contextHash: string
}
```

- 广场投稿从后端复制成品图和参考图，不让前端下载后再上传。
- `PublicJob` 不暴露内部文件路径、空间 token、API key。

## 19. 前端模块化补充

下一轮前端建议按这个目录拆：

```text
web/src/components/canvas/
  CanvasPage.tsx
  CanvasShell.tsx
  CanvasStage.tsx
  CanvasNodeLayer.tsx
  CanvasEdgeLayer.tsx
  CanvasFloatingToolbar.tsx
  CanvasBottomToolbar.tsx
  CanvasZoomControls.tsx
  CanvasInspector.tsx
  CanvasComposer.tsx
  nodes/
    TextNode.tsx
    ImageNode.tsx
    GenerationNode.tsx
    ResultNode.tsx
  hooks/
    useCanvasController.ts
    useCanvasInteractions.ts
    useCanvasKeyboard.ts
    useCanvasPersistence.ts
    usePreviewAssetCache.ts
    useCanvasGeneration.ts
  model/
    types.ts
    reducer.ts
    selectors.ts
    context.ts
    layout.ts
web/src/api/canvas.ts
web/src/api/contracts/canvas.ts
```

状态拆分：

- `CanvasDocument`：只放可持久化内容，包含 nodes、edges、prompt、mode、generationConfig、viewport。
- `CanvasRuntimeState`：只放运行态，包含 selection、interaction、contextMenu、editingTextId、asyncFlags、previewUrls。
- `WorkbenchBridge`：只负责外部能力，包含 uploads、recentResults、latestTask、onCreateTask、onUploadReferences、onUsePrompt。

## 20. 性能与资源策略

无限画布不能靠“把 stage 放大”解决，要先做资源和渲染预算。

### 20.1 前端策略

- 交互中用 ref + `requestAnimationFrame` 更新拖拽和平移，pointerup/idle 后再提交 React state。
- 节点数据使用 `Map<id, item>`，边使用邻接表；移动单节点时只重算相关边。
- 画布默认显示缩略图，原图只用于提交、放大查看和下载。
- 新增 `PreviewAssetCache`：按 uploadId/hash 做引用计数、LRU、并发限制、AbortController、删除即 revoke。
- 图片解码优先 `createImageBitmap` 或 async decode，桌面并发 4，移动端并发 2。
- 节点和边数量变多后做视口虚拟化，只挂载可见、近邻和选中的节点。

### 20.2 后端策略

- 上传保存时生成 thumbnail/webp preview，记录原始宽高、hash、mime、size。
- 画布 JSON 只保存素材 ID 和元数据，不保存 blob/base64。
- 任务 reference snapshot 优先内容寻址、hardlink/reflink；必须复制时异步化并暴露进度。
- 增加空间配额、素材清理和快照清理策略。

### 20.3 性能指标

桌面 MVP：

- 100 节点、200 边、8 张提交参考图。
- pan/drag/zoom p95 frame <= 16ms。
- 首屏可见缩略图 p95 <= 800ms。
- 已解码图片总量 <= 300MB。

移动 MVP：

- 40 节点、80 边。
- pan/drag/zoom p95 frame <= 33ms。
- 峰值 JS heap 增量 <= 80MB。
- 已解码图片总量 <= 120MB。

后端：

- 8 张 12MB 参考图创建任务 p95 <= 2s。
- 磁盘放大目标 <= 1.2x。
- 删除素材/节点后 5 秒内 object URL 和 decoded cache 可回收。

## 21. UI 取舍补充

可吸收：

- 轻网格背景。
- 文本、图片、生成配置、结果四类节点。
- 蓝色或主题主色选中态。
- 曲线连线表达来源。
- 选中对象浮动工具条。
- 底部工具条。
- 左下缩放控件。
- 结果自动分支。

不要照搬：

- 过小的配置卡。
- 过密的右上角图标。
- 文字很长的浮动工具条。
- 过多参数塞进节点内部。
- 只适配深色主题。
- 结果散到屏外，用户无法比较。

Lyra 的视觉原则：

- 1080p 首屏可用。
- 主流程节点更大、更清晰。
- 未选中节点保持安静。
- 高频动作近手，低频动作折叠。
- 每个主题都走 token，不写死黑色按钮。

## 22. 最终验收补充

新增验收用例：

- 1 张 8K 图拖入后页面不崩。
- 8 张 12MB 图拖入后 100ms 内有占位反馈。
- 100 节点 / 200 边桌面图谱可拖拽 30 秒不卡死。
- 40 节点 / 80 边移动图谱可缩放和平移。
- 删除节点后关联边同步删除。
- 删除图片后 object URL 和解码缓存能回收。
- 生成配置节点只使用连入它的参考图，不误用全画布所有图片。
- 任务完成后 result node 不重复插入。
- 广场投稿能公开成品图、提示词和参考图，并提示用户确认。
- 绿色主题、黑紫主题、白紫主题下工具条、连线、标签、按钮对比正常。
