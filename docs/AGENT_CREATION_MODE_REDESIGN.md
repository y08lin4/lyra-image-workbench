# Agent 创作模式重设计稿

更新时间：2026-06-28

目标：在现有 Agent 创作页基础上重构内部能力，把它从“像预设提示词”的页面改成真正以大模型为核心、能多轮理解用户目标、能创建生图任务、能回流结果继续修改的独立创作模块。中文提示词只作为模型计划的辅助产物，不作为页面核心。

## 1. 当前判断

当前 Agent 页面已经有聊天外观，但核心能力仍然偏“提示词整理器”：

- 对话回复主要依赖固定快捷词、正则判断缺字段、本地拼接质量词。
- 大模型调用只复用 `prompt-tools` 的 session/refine，目标是“整理一段提示词”，不是“理解目标并编排任务”。
- `快捷生成` 只是把提示词填到快捷生成页，没有由 Agent 自己创建任务、展示任务 ID、跟踪状态。
- 没有结构化参考图引用，输入框里 `@图1` 和真实图片绑定没有区分。
- 没有 Agent 轮次、计划、确认扣费、任务卡、结果回流和继续修改的状态机。

结论：下一版不能继续在当前页面上加几个按钮，而要把 Agent 定义成“多轮创作导演”。

## 2. 产品定位

Agent 创作不是提示词助手，也不是创作画布。

它的职责是：

1. 用户用一句话或多张参考图描述想法。
2. Agent 通过大模型判断信息是否足够。
3. 信息不足时追问；信息足够时给出创作计划和生成参数。
4. 用户确认后，Agent 调用现有生图任务链路创建任务。
5. 页面显示任务 ID、状态、结果图和失败原因。
6. 用户可以基于某张结果继续要求修改，形成多轮创作记录。

一句话定义：Agent 是“会对话、会做计划、会调用生成任务、会记住上下文”的创作入口。

### 2.1 核心是模型，不是中文提示词

这次重构的重点不是把中文提示词写得更长，而是把“大模型决策”放到页面和链路中心。

页面主视图应该围绕这些内容：

- 模型理解到的用户目标。
- 模型认为还缺什么信息。
- 模型给出的创作计划。
- 模型选择的模式、比例、数量、参考图用途。
- 模型准备创建的任务和预计消耗。
- 任务状态、结果图和下一轮修改建议。

这里要区分两种“提示词”：

- 用户可见的最终中文提示词：只是模型方案里的一个字段，用来给用户检查、复制或手动微调；默认不占据右侧主面板，更不能继续做成大 textarea 的核心界面。
- 后端内置给 Agent 的系统提示词：这是核心能力的一部分，用来指导 Agent 补全场景描绘、镜头语言、光影、构图、材质、情绪和限制条件。

用户可见提示词可以放在“提示词”折叠区或 Tab 里，用户需要时再展开；发送到画布、快捷生成或 API 时可以携带，但页面叙事不围绕它展开。

验收口径：如果用户打开 Agent 页，第一眼看到的还是“整理后的中文提示词”和复制按钮，而不是模型计划、确认生成、任务状态，那就说明界面方向错了；如果后端没有内置 Agent 系统提示词帮助模型补全场景描绘，那说明能力方向也没做对。

## 3. 与现有模块边界

| 模块 | 职责 | Agent 是否替代 |
| --- | --- | --- |
| 创作画布 | 摆放素材、拖拽图片、连线、手动组织参考关系 | 不替代，仍是主入口 |
| 提示词助手 | 单次提示词优化、图片还原、灵感扩写 | 不替代，可复用底层 LLM 能力 |
| 快捷生成 | 快速输入提示词并创建任务 | 不替代，适合简单直达 |
| 结果页 | 全量任务列表、下载、重试、提交广场 | 不替代，Agent 只展示当前会话相关任务 |
| Agent 创作 | 多轮目标拆解、计划、确认、创建任务、结果回流 | 改造现有页面，形成独立闭环 |

Agent 和画布只通过显式动作互通：

- Agent 生成的创作方案可以“发送到创作画布”，中文提示词随方案作为辅助字段带过去。
- 画布或结果页的图片可以“作为 Agent 参考图”。
- Agent 不直接写画布内部坐标、连线、缩放、旋转状态。

## 4. 核心用户路径

### 4.1 文生图路径

1. 用户进入 Agent 创作。
2. 输入：“帮我做一张奶茶新品海报，适合小红书发布。”
3. Agent 判断信息不足，追问 1 到 2 个关键问题，例如产品卖点、风格倾向。
4. 用户回答或点击“跳过，直接生成方案”。
5. Agent 返回创作计划：
   - 画面目标
   - 主体与场景
   - 风格方向
   - 推荐比例
   - 生成参数
   - 预计创建任务数量和消耗
6. 用户点击“确认生成”。
7. 后端创建背景任务，返回 task id。
8. Agent 页面展示任务卡和生成状态。
9. 结果完成后，用户可以选择“继续修改这张”“发送到画布”“查看结果详情”。

### 4.2 图生图路径

1. 用户上传或选择历史图片作为参考图。
2. Agent 输入框显示真实引用 token，例如 `@参考图1`，并在提交前列出已绑定图片。
3. 用户输入：“保持主体不变，改成赛博朋克夜景。”
4. Agent 读取参考图快照，生成图生图计划。
5. 用户确认后，任务 payload 使用已有 `uploadIds` 创建 `image-to-image` 任务。
6. 结果完成后，本轮记录保存“输入参考图 + 提示词 + 输出图”的链路。

### 4.3 多轮修改路径

1. 用户在某个结果图上点“继续改”。
2. Agent 自动把该结果注册为下一轮参考。
3. 用户输入：“把背景改成雪山，人物表情更开心。”
4. Agent 根据上一轮 prompt、参考图、用户修改要求生成下一轮计划。
5. 确认后创建新任务。

## 5. 页面设计稿

### 5.1 桌面端布局

适配 1366x768 到 1920x1080。目标是一屏内看清对话、计划、任务状态，不让右侧大面积空着。

```text
--------------------------------------------------------------------------------+
| Agent 创作 / 当前会话标题                    模型  Key 状态  余额/预计消耗      |
+----------------------+--------------------------------+------------------------+
| 会话列表             | 多轮对话时间线                 | 本轮执行面板           |
| + 新会话             |                                |                        |
| 最近会话 A           | 用户消息                       | 创作计划               |
| 最近会话 B           | Agent 追问/计划                | - 目标                 |
| 最近会话 C           | 参数确认块                     | - 参数                 |
|                      | 任务卡 task_xxx                | - 参考图               |
|                      | 结果缩略图                     | - 预计消耗             |
|                      |                                |                        |
|                      |                                | [确认生成] [改计划]    |
+----------------------+--------------------------------+------------------------+
| 参考图 token / 输入框 / 上传 / 选择历史图 / 发送                               |
+--------------------------------------------------------------------------------+
```

布局要点：

- 左侧会话栏窄而实用，宽度约 220 到 260px。
- 中间是最大区域，只放当前会话的轮次，不堆多余说明。
- 右侧不是 textarea，而是“计划、参数、参考图、任务状态、结果预览”的执行面板。
- 底部输入框固定在 Agent 页面内部，但不要遮挡内容。
- 当前没有计划时，右侧显示轻量空状态：“发送想法后会在这里生成计划”。

### 5.2 移动端布局

移动端不做三栏，改成顶部会话选择 + 内容 Tab：

```text
Agent 创作          新会话
[对话] [计划] [结果]

对话时间线

输入框 / 参考图 / 发送
```

规则：

- 默认显示“对话”。
- Agent 生成计划后自动切到“计划”或显示底部提示。
- 任务生成中显示底部状态条：`task_xxx 生成中 40%`。
- 结果完成后给“查看结果”和“继续修改”。

### 5.3 主要区块

#### 会话列表

显示：

- 会话标题
- 最近更新时间
- 状态：草稿 / 待确认 / 生成中 / 已完成 / 失败
- 当前会话产生的任务数

操作：

- 新建会话
- 重命名
- 归档
- 删除

#### 对话时间线

消息类型不再只有纯文本：

- `user_message`：用户输入。
- `agent_question`：Agent 追问。
- `agent_plan`：创作计划。
- `agent_parameters`：生成参数确认块。
- `task_card`：任务 ID、状态、进度、失败原因。
- `result_grid`：本轮结果图。
- `reference_notice`：引用图失效、已删除、需要重新上传。

#### 本轮执行面板

右侧面板始终服务当前轮次：

- 当前目标
- 参考图列表
- 模型生成的创作方案
- 中文提示词摘要，默认折叠
- 模式：文生图 / 图生图
- 模型、比例、质量、数量
- 预计消耗
- 确认生成按钮
- 改计划按钮

只有用户确认后才创建任务和扣费。

#### 输入区

输入区必须支持：

- 普通文本。
- 上传图片。
- 从历史结果选择图片。
- 结构化引用 token。
- “跳过追问，直接给方案”。
- “查看/复制中文提示词”作为次要动作。

不建议保留一排固定的“更写实、电影感、竖图 9:16”作为主要体验。它们最多作为 Agent 追问后的快捷答案，不应成为 Agent 的核心。

## 6. 大模型接入设计

### 6.1 调用原则

- 大模型必须是 Agent 的核心决策层，由后端调用，前端不暴露上游 Key。
- 复用现有 `internal/llm` 客户端，但 Agent 要有自己的 service 和 system prompt。
- `prompttools` 可以复用部分提示词模板和图片理解能力，但不能作为 Agent 的唯一后端。
- Agent 输出必须是结构化结果，不只是一段 Markdown，也不能只是一段中文提示词。

### 6.1.1 内置 Agent 提示词

Agent 后端必须内置一套系统提示词，作为每次规划调用的固定上下文。它不是展示给用户看的文本，而是让模型更稳定地补全画面方案。

内置提示词要指导模型完成这些事情：

- 从用户的一句话里识别主体、用途、目标受众和发布场景。
- 自动补全场景描绘，例如环境、空间关系、前景/中景/背景。
- 自动补全镜头语言，例如景别、视角、焦段感、构图方式。
- 自动补全光影和色彩，例如自然光、棚拍、电影感、低饱和、商业海报色调。
- 自动补全材质和细节，例如皮肤、布料、金属、玻璃、包装、产品质感。
- 自动补全情绪和氛围，例如高级、清爽、可爱、科技感、节日感。
- 根据图生图参考图判断哪些内容必须保留，哪些可以变化。
- 在信息不足时只追问关键问题，避免问太多打断用户。
- 输出结构化计划，包含 `sceneBrief`、`visualPlan`、`generationPrompt`、`negativePrompt`、`parameters`、`referenceUsages`。

内置提示词的职责是帮助 Agent “会想画面”，不是让页面变成提示词编辑器。用户可见的中文提示词应由这个内置提示词驱动生成，并作为方案的辅助字段保存。

### 6.2 Key 和额度规则

Agent 创建图片任务时，必须复用现有任务扣费规则：

- 用户使用自己的上游 Key 创建生图任务：不扣平台额度。
- 用户使用系统上游 Key 创建生图任务：扣平台额度。
- 仅生成计划或追问是否扣费，后续可配置；MVP 建议暂不单独扣用户次数，只记录 LLM 调用日志。
- 无可用上游 Key 时，Agent 可以聊天和整理计划，但不能创建图片任务，按钮显示“去设置 Key”。

### 6.3 Agent 工具能力

MVP 不需要直接做复杂 function calling，可以让模型返回 JSON 计划，由后端校验后执行。

建议的模型输出结构：

```json
{
  "action": "ask_question | propose_plan",
  "question": "还需要用户补充的问题",
  "plan": {
    "title": "会话标题",
    "mode": "text-to-image",
    "prompt": "最终生成提示词",
    "negativePrompt": "负面约束",
    "ratio": "9:16",
    "quality": "high",
    "count": 1,
    "referenceIds": [],
    "notes": ["为什么这样设计"]
  }
}
```

后端职责：

1. 调用 LLM。
2. 解析 JSON。
3. 校验模式、比例、数量、参考图权限。
4. 保存 round 和 plan。
5. 等用户确认后创建任务。

## 7. API 草案

新增前端模块：`web/src/api/agents.ts`

新增后端模块：

- `internal/agents`
- `internal/api/agents.go`

### 7.1 会话接口

```http
GET /api/agents/sessions?limit=30
POST /api/agents/sessions
GET /api/agents/sessions/{sessionId}
PATCH /api/agents/sessions/{sessionId}
DELETE /api/agents/sessions/{sessionId}
```

### 7.2 多轮消息接口

```http
POST /api/agents/sessions/{sessionId}/messages
```

请求：

```json
{
  "content": "我想做一张奶茶新品海报",
  "referenceIds": ["ref_123"],
  "skipQuestions": false,
  "provider": "image-2",
  "model": "gpt-image-2"
}
```

返回：

```json
{
  "ok": true,
  "session": {},
  "round": {},
  "blocks": [
    { "type": "agent_question", "content": "产品卖点是什么？" }
  ]
}
```

### 7.3 确认生成接口

```http
POST /api/agents/sessions/{sessionId}/rounds/{roundId}/confirm
```

返回：

```json
{
  "ok": true,
  "taskIds": ["task_xxx"],
  "tasks": []
}
```

### 7.4 参考图接口

Agent 不单独保存图片文件，复用已有上传和结果图片：

```http
POST /api/agents/sessions/{sessionId}/references
DELETE /api/agents/sessions/{sessionId}/references/{referenceId}
```

参考图在数据库中保存快照：

- 来源：upload / task-result / agent-result
- upload id 或 task id
- result index
- 原始文件名
- 缩略图 URL
- prompt 快照
- 是否已删除

### 7.5 任务状态

MVP 可以复用现有任务轮询和 SSE：

- 创建任务后仍然使用 `/api/background-tasks/{id}`。
- Agent session 只保存 task id 和 round id 的关联。
- 前端 Agent 页根据 task id 获取状态。

## 8. 数据结构草案

### 8.1 前端类型

```ts
export type AgentSession = {
  id: string
  title: string
  status: 'draft' | 'awaiting_confirmation' | 'generating' | 'completed' | 'failed'
  rounds: AgentRound[]
  references: AgentReference[]
  taskIds: string[]
  createdAt: string
  updatedAt: string
}

export type AgentRound = {
  id: string
  index: number
  userMessage: AgentMessage
  assistantBlocks: AgentBlock[]
  plan?: AgentPlan
  referenceIds: string[]
  taskIds: string[]
  status: 'collecting' | 'planning' | 'awaiting_confirmation' | 'generating' | 'completed' | 'failed'
  error?: string
}

export type AgentBlock =
  | { type: 'text'; content: string }
  | { type: 'question'; content: string }
  | { type: 'plan'; plan: AgentPlan }
  | { type: 'task'; taskId: string }
  | { type: 'result'; taskId: string; imageUrls: string[] }
  | { type: 'error'; message: string }
```

### 8.2 后端类型

```go
type AgentSession struct {
    ID        string
    SpaceToken string
    UserID    string
    Title     string
    Status    string
    Rounds    []AgentRound
    References []AgentReference
    TaskIDs   []string
    CreatedAt time.Time
    UpdatedAt time.Time
}

type AgentRound struct {
    ID          string
    Index       int
    UserMessage AgentMessage
    Blocks      []AgentBlock
    Plan        *AgentPlan
    ReferenceIDs []string
    TaskIDs     []string
    Status      string
    Error       string
    CreatedAt   time.Time
    FinishedAt  *time.Time
}
```

## 9. 状态机

```text
empty
  -> collecting              用户输入初始目标
  -> planning                后端调用 LLM
  -> asking                  Agent 追问
  -> awaiting_confirmation   Agent 给出计划，等待用户确认
  -> creating_tasks          用户确认，后端创建任务
  -> generating              任务生成中
  -> reviewing               结果完成，可继续修改
  -> completed               会话暂时结束
  -> failed                  LLM、Key、额度、任务任一环节失败
```

失败必须可恢复：

- LLM 失败：允许重试计划。
- Key 缺失：跳转设置。
- 额度不足：跳转充值。
- 任务失败：允许重试任务或修改计划。
- 参考图失效：允许移除引用或重新上传。

## 10. 模块拆分

### 10.1 前端

建议拆成：

```text
web/src/api/agents.ts
web/src/types/agent.ts
web/src/components/agent/AgentPage.tsx
web/src/components/agent/AgentShell.tsx
web/src/components/agent/AgentSessionList.tsx
web/src/components/agent/AgentTimeline.tsx
web/src/components/agent/AgentMessageBlock.tsx
web/src/components/agent/AgentPlanPanel.tsx
web/src/components/agent/AgentTaskCard.tsx
web/src/components/agent/AgentReferencePicker.tsx
web/src/components/agent/AgentComposer.tsx
web/src/components/agent/AgentMobileTabs.tsx
web/src/components/agent/agentReducer.ts
web/src/components/agent/agentReferences.ts
web/src/components/agent/AgentPage.css
```

不要把所有状态和 UI 堆在一个 `AgentPage.tsx`。

### 10.2 后端

建议拆成：

```text
internal/agents/types.go
internal/agents/store.go
internal/agents/service.go
internal/agents/planner.go
internal/agents/references.go
internal/agents/prompts.go
internal/api/agents.go
internal/api/agents_test.go
```

`service.go` 负责业务编排，`planner.go` 负责 LLM 请求和 JSON 解析，`references.go` 负责引用校验和快照。

## 11. MVP 范围

第一版只做真实闭环，不做花活：

必须做：

1. 改造现有 Agent 页面，不新增重复入口。
2. 会话列表和会话详情。
3. 后端真实调用 LLM 生成追问或计划。
4. 用户确认计划后创建现有 background task。
5. Agent 页面显示 task id、状态、结果。
6. 支持文本输入和参考图选择。
7. 支持把结果作为下一轮参考。
8. 支持无 Key、额度不足、任务失败的清晰提示。

暂不做：

- 分支编辑。
- 多 Agent 并行作图。
- web search。
- 自动复杂排版。
- 画布坐标同步。
- GIF Agent。
- 复杂富文本编辑器。

## 12. 后续阶段

### P1

- 历史轮次编辑后生成新分支。
- 支持 `@第1轮图2` 的结构化引用选择。
- 支持流式输出计划。
- 支持一轮生成多张不同方案并比较。
- 从结果页一键进入 Agent 继续创作。

### P2

- Agent 模板市场。
- 品牌风格记忆。
- 多图关系推理。
- 批量生成方案。
- 自动提交广场草稿。
- 与创作画布共享引用 token 编码，但仍不共享画布状态。

## 13. UI 风格要求

- 不做大段说明文字。
- 不做深色代码块式聊天区，跟随全局主题。
- 不用硬编码黑色、蓝色，要走主题 token。
- 不做多层卡片套卡片。
- 主要区域平铺，右侧空间必须有效利用。
- 按钮高度、文字居中、移动端触控面积要统一。
- 空状态要短，只告诉用户下一步。
- 任务状态必须明确，不显示“假成功”。

## 14. 验收标准

功能验收：

- 新建会话后刷新页面仍存在。
- 用户发送消息后，Network 中能看到 `/api/agents/.../messages`，后端会真实调用 LLM。
- 如果模型返回追问，页面显示追问，而不是固定本地模板。
- 如果模型返回计划，页面显示计划和确认按钮。
- 用户未确认前，不创建图片任务，不扣生图额度。
- 用户确认后，返回真实 task id，并能在结果页找到同一任务。
- 任务状态能在 Agent 页面更新。
- 任务完成后，结果图能作为下一轮参考。
- 参考图删除后，Agent 显示“引用已失效”，不会崩溃。
- 用户使用自己的上游 Key 生图不扣平台额度；使用系统 Key 生图才扣额度。

代码验收：

- 前端 Agent 代码不再集中堆在单文件。
- 新增 `web/src/api/agents.ts`，不复用 `promptTools.ts` 假装 Agent。
- 后端有独立 `internal/agents` 模块。
- Agent 不修改创作画布内部状态文件。
- 创建任务必须复用现有 background task service，不新增一套任务队列。
- TypeScript build 通过。
- Go test 至少覆盖 Agent store、message、confirm 创建任务。

体验验收：

- 1366x768 一屏能看见对话、计划、输入框。
- 1920x1080 右侧不空。
- 390x844 移动端无重叠，输入和发送可用。
- 绿色、黑色、白色主题下文字对比正常。
- 错误信息只说用户能行动的内容，不显示多余技术废话。

## 15. 推荐实施拆分

可以拆给子代理的任务：

1. 后端 Agent store/types：只做会话、轮次、引用、持久化。
2. 后端 planner：复用 `internal/llm`，实现 JSON 计划解析和校验。
3. 后端 confirm：把 Agent plan 转换为现有 background task payload。
4. 前端 API/types：新增 `agents.ts` 和类型定义。
5. 前端页面骨架：会话栏、时间线、右侧计划面板。
6. 前端 composer/reference：输入框、上传、历史图引用 token。
7. 前端 task card：task id、状态、结果图、继续修改。
8. 移动端和主题验收：专门检查 390x844、绿色主题、暗色主题。
9. 回归验收：确认画布、快捷生成、提示词助手没有被 Agent 改坏。

## 16. 最终取舍

建议先把 Agent 做成“可靠的多轮生图助手”，而不是一开始追求复杂画布或自动代理系统。

第一阶段只要跑通这条线就算成功：

```text
一句话想法 -> 大模型追问/计划 -> 用户确认 -> 创建任务 -> 展示 task id 和结果 -> 继续修改
```

这条线跑通后，Agent 才真正从“预设提示词页面”变成“可用的创作模式”。

## 17. 子代理回收结论

本轮已回收的只读审查结论一致：

- 当前 Agent 的本质问题不是 UI 少，而是缺少专属 Agent 数据模型和任务闭环。
- `AgentPage.tsx` 现在使用本地快捷词、正则补字段和 `prompt-tools` 整理提示词，所以用户会感觉像“预设提示词”。
- 真 Agent 必须新增独立的 `AgentSession / AgentRound / AgentReference / AgentBlock`，不能只复用 `PromptSession`。
- 桌面端应采用“对话 + 可操作产物面板”的结构：左侧表达意图，右侧承载计划、参数、预览、任务。
- 生成前必须有确认卡，确认前不扣费、不创建图片任务。
- 创建任务后必须显示真实 task id，并能在结果页看到同一任务。
- 参考图必须是结构化 token 和后端快照，用户手打 `@图1` 不能直接当作真实引用。
- Agent 不应复刻结果页、提示词助手、创作画布；只展示当前会话相关任务和显式跳转动作。
- 绿色主题、暗色主题、移动端布局必须作为验收项，不能再出现硬编码黑色和控件错位。

## 18. 本次设计决策

最终设计采用以下方向：

1. Agent 是独立一级模块，不嵌入画布。
2. Agent 主路径是“对话 -> 计划 -> 确认 -> 任务 -> 结果 -> 继续修改”。
3. 右侧执行面板替代当前只读 prompt textarea，主显示模型计划、参数、状态和结果。
4. LLM 调用放在后端 Agent service，不在前端直连上游。
5. 图片生成仍复用现有 background task，不新增任务队列。
6. MVP 先不做分支树、web search、GIF Agent、复杂富文本编辑器。
7. 模块拆分优先，前端和后端都不允许继续堆单文件。

## 19. 技术接入补充

技术审查结论：现有图片生成闭环已经完整，Agent 不应该新建图片队列。

落地规则：

- Agent 新增 `internal/agents` 作为会话、轮次、计划、引用、任务反链层。
- 图片生成仍调用现有 background task 链路，最终复用 `jobs.Manager.Create`。
- Agent 创建的任务需要能在任务历史中识别来源，建议扩展 `jobs.Job` 元数据：
  - `source: "agent"`
  - `agentSessionId`
  - `agentRoundId`
- 当前后端如果只识别 `web/api` source，后续实现时要新增 `JobSourceAgent`，否则结果页无法按 Agent 过滤或回链。
- LLM 规划阶段 MVP 不扣生图额度；确认创建图片任务时才进入现有额度预检和扣费。
- 单轮确认必须限制总图片数，建议 MVP 限制 `count` 总和不超过 3，避免批量任务误扣。
- 历史结果作为参考图，P0 可复用现有前端 fetch 后上传；P0.5 建议新增服务端 `task result -> reference upload` 直拷接口，避免浏览器跨源和大图下载问题。

## 20. 硬性完成红线

后续实现时，以下任一项缺失，都不能判定 Agent 创作模式完成：

1. 没有真实 `/api/agents` session / round / message / confirm 接口。
2. 没有真实 task id 回流到 Agent 页面。
3. 结果页找不到 Agent 创建的同一个 task id。
4. 用户确认计划前已经创建任务或扣除生图额度。
5. 用户自带 Key 生图仍扣平台额度。
6. Agent 页面仍然主要依赖 `promptTools` 输出一个 textarea，而不是由模型驱动独立 session / round / plan。
7. 参考图没有结构化 token 和后端快照。
8. 移动端、绿色主题、暗色主题没有经过实际验收。





## 21. 本轮执行与验收台账（2026-06-28）

本节用于记录本轮 Agent 创作模式上线任务的执行口径、子代理分工和最终验收清单，方便上下文压缩后继续接手。状态口径从严：本文档更新只代表设计和任务台账完成，不代表 Agent 模块已经上线。

### 21.1 当前任务口径

当前主目标：

1. 把现有 Agent 创作页从“提示词整理器”升级为独立多轮创作模块。
2. 新增真实 `/api/agents` 会话、轮次、消息、引用、确认生成接口。
3. 后端真实调用 LLM 生成追问或结构化计划。
4. 用户确认计划后复用现有 background task service 创建图片任务。
5. Agent 页面展示真实 task id、任务状态、结果图，并允许把结果作为下一轮参考。
6. 保持创作画布为主入口，但 Agent 不直接写画布内部状态。

明确非目标：

- 不把 Agent 做成画布侧边栏。
- 不做完整分支树、web search、GIF Agent、视频 Agent、局部重绘或复杂富文本编辑器。
- 不复制 `basketikun/infinite-canvas` 的本地 MCP/插件架构作为 P0。
- 不绕过现有 Key、额度、任务、结果、空间隔离和日志链路。

### 21.2 子代理分工

| role_id | 微任务 | 权限边界 | 交付物 | 验收重点 |
| --- | --- | --- | --- | --- |
| documentation-engineer | 整理本轮 Agent 上线台账和 infinite-canvas 参考分析 | 只写 `docs/AGENT_CREATION_MODE_REDESIGN.md`、`docs/INFINITE_CANVAS_REFERENCE_ANALYSIS.md` | 本节和参考分析文档 | 简体中文、范围准确、不宣称未完成项已完成 |
| backend-agent-store | 新增 Agent 会话、轮次、引用、计划、任务反链数据模型 | `internal/agents/types.go`、`store.go`、`references.go` 及对应测试 | 可持久化 store、引用快照校验 | 刷新不丢会话；跨用户/space 不越权；引用失效可表达 |
| backend-agent-planner | 接入 `internal/llm`，实现 Agent system prompt、JSON 计划解析和校验 | `internal/agents/planner.go`、`prompts.go` 及测试 | 追问/计划结构化输出 | 不只返回 Markdown；模型输出异常可恢复；不把 prompttools 当 Agent |
| backend-agent-confirm | 把 Agent plan 转成现有 background task payload | `internal/agents/service.go`、`internal/api/agents.go`、`internal/api/agents_test.go` | confirm 接口和任务创建链路 | 确认前不建任务不扣费；确认后返回真实 task id；用户 Key/系统 Key 额度规则正确 |
| frontend-agent-contract | 新增前端 API wrapper 和类型定义 | `web/src/api/agents.ts`、`web/src/types/agent.ts` | 前后端 DTO、错误码和状态枚举 | 字段与后端一致；不复用 `promptTools.ts` 假装 Agent |
| frontend-agent-ui | 拆分 Agent 页面骨架、时间线、计划面板、任务卡 | `web/src/components/agent/*`、`AgentPage.css` | 桌面三栏和移动端 Tabs | 1366/1920 桌面可用；390 移动端不遮挡；右侧不是大 textarea |
| frontend-agent-references | 实现上传/历史结果/Agent 结果的引用 token 和选择器 | `AgentReferencePicker.tsx`、`AgentComposer.tsx`、`agentReferences.ts` | 结构化 `@参考图N` chip 和引用列表 | 手打 `@图1` 不算真实绑定；提交前显示来源和顺序 |
| qa-integration-reviewer | 只读验收构建、测试、契约、浏览器和回归 | read-only / tests-only | 命令输出、截图或失败证据、风险清单 | `go test ./...`、`npm run build`、`git diff --check`、主题和移动端验收 |

并行规则：

- 同一文件只能有一个写入者。
- 前端 Agent UI 写入者不得同时改创作画布内部组件。
- 后端 Agent 写入者不得新建第二套任务队列。
- 只读验收代理不能直接修代码，必须把风险交回主代理或对应写入者。

### 21.3 实施顺序

1. 冻结前后端契约：先确定 session、round、message、reference、plan、confirm、task link 的字段和状态。
2. 落后端 store/types/references：先让会话、轮次、引用快照可持久化。
3. 落 planner：接 `internal/llm`，输出追问或结构化计划，并处理 JSON 解析失败。
4. 落 confirm：把 plan 转为现有 background task，写入 `source=agent`、`agentSessionId`、`agentRoundId`。
5. 落前端 API/types：所有组件只走 `web/src/api/agents.ts`。
6. 落前端 UI：先完成桌面主路径，再补移动端 tabs 和主题。
7. 落引用回流：历史结果、上传图、Agent 输出图都能成为下一轮参考。
8. 集成验收：命令、浏览器、扣费、结果页回链和回归一起收口。

### 21.4 功能验收清单

| 验收项 | 必须通过的标准 |
| --- | --- |
| 会话持久化 | 新建 Agent 会话后刷新页面仍存在，标题、轮次、引用和 task id 不丢。 |
| 消息接口 | 用户发送消息后，Network 中出现 `/api/agents/.../messages`，后端真实调用 LLM。 |
| 追问 | 信息不足时显示模型追问，不是固定本地模板或正则缺字段文案。 |
| 计划 | 信息足够时显示结构化计划、参数、参考图用途、预计消耗和确认按钮。 |
| 确认前行为 | 用户未确认前不创建图片任务、不扣生图额度、不写假 task id。 |
| 确认后行为 | confirm 返回真实 task id，Agent 页面出现任务卡，结果页能找到同一 task id。 |
| 状态更新 | generating、completed、failed 状态能同步到 Agent 页面，失败原因可读。 |
| 继续修改 | 任务完成后，某张结果可作为下一轮结构化参考图。 |
| 引用失效 | 参考图删除或清理后显示“引用已失效”，允许移除或重新上传，不崩溃。 |
| 额度规则 | 用户自带上游 Key 生图不扣平台额度；系统 Key 生图才进入平台额度预检和扣费。 |

### 21.5 代码验收清单

| 验收项 | 必须通过的标准 |
| --- | --- |
| 后端模块 | 存在独立 `internal/agents`，包含 types、store、service、planner、references、prompts。 |
| API 接线 | 存在 `internal/api/agents.go`，路由包含 sessions、messages、references、confirm。 |
| 前端契约 | 存在 `web/src/api/agents.ts` 和 `web/src/types/agent.ts`，字段与后端一致。 |
| 任务复用 | Agent 创建图片任务必须复用现有 background task / jobs manager，不新增队列。 |
| 任务反链 | Agent 任务元数据能记录 `source=agent`、session id、round id，方便结果页回链。 |
| 模块化 | Agent 前端不能继续堆在单个 `AgentPage.tsx`；后端不能把所有逻辑塞进 handler。 |
| 回归边界 | Agent 改动不得破坏创作画布、快捷生成、提示词助手、结果页和广场提交。 |
| 测试 | 至少覆盖 Agent store、message、confirm 创建任务、引用失效、额度分支。 |

### 21.6 命令和浏览器验收

集成后必须重新执行：

```powershell
go test ./...
cd web
npm run build
cd ..
git diff --check
```

浏览器验收至少覆盖：

- 1366x768：对话、计划、输入框、任务卡同屏可用。
- 1920x1080：右侧执行面板不空、不变成 prompt textarea。
- 390x844：移动端 `对话 / 计划 / 结果` tabs 可用，输入和发送不被遮挡。
- 绿色主题、暗色主题、白色主题：文字、按钮、任务状态、引用 chip 对比正常。
- Agent 创建的任务在结果页可见，同一 task id 能重试、查看、下载或复用。

当前已知环境风险：此前浏览器自动化多次受网络盘/会话环境影响启动失败。如果自动截图仍不可用，必须保留人工验收页面列表和截图缺口说明，不能把“构建通过”写成“视觉验收通过”。

### 21.7 当前风险与冲突

- 当前文档完成不等于 Agent 创作模式已经实现；代码仍需按上方分工落地和验收。
- 工作区有多个并行代理改动，主代理合并时必须检查所有权和同文件冲突。
- `prompttools` 可复用图片理解和提示词模板，但不能继续作为 Agent 的唯一后端。
- infinite-canvas 只能作为参考项目分析；其 AGPL 代码、本地 Key 模式、本地 MCP 插件不能无审查并入 Lyra。
- 移动端、绿色主题、暗色主题必须真实验收，否则不能判定上线完成。
