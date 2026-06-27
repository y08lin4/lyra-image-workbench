# Agent 多轮创作模块设计

更新时间：2026-06-27
参考项目：[CookSleep/gpt_image_playground](https://github.com/CookSleep/gpt_image_playground)
当前结论：Agent 必须作为独立模块接入 Lyra，不嵌入创作画布，不复用画布内部状态，不把提示词助手改造成 Agent。

## 1. 参考项目的核心功能要点

### 1.1 数据模型

参考项目不是简单聊天框，而是“会话 + 轮次树 + 消息 + 图片任务”的结构。

- `AgentConversation` 保存会话标题、消息列表、轮次列表、当前 active leaf。
- `AgentRound` 是一轮用户请求，包含 parent round、用户消息、助手消息、输入图、输出任务 ID。
- `AgentMessage` 保存用户或助手文本。
- 图片结果仍然是普通 `TaskRecord`，但任务上反向记录 agent conversation / round / message / tool call。

可参考：
- [AgentRound / AgentConversation 类型](https://github.com/CookSleep/gpt_image_playground/blob/main/src/types.ts#L161-L235)
- [TaskRecord Agent 反链字段](https://github.com/CookSleep/gpt_image_playground/blob/main/src/types.ts#L249-L288)

### 1.2 多轮与分支

参考项目的聊天区只渲染当前 active 分支路径，而不是把整棵分支树全部铺出来。

关键能力：

1. 用户每次提交形成一个 round。
2. round 可以有 parent，天然形成分支树。
3. 编辑历史轮次时，不直接覆盖正常历史，而是创建 sibling 分支。
4. 重新生成时，正常轮次也会创建 sibling 分支。
5. 删除轮次时，需要重算路径和图片引用编号。

可参考：
- [当前 active path 渲染](https://github.com/CookSleep/gpt_image_playground/blob/main/src/components/AgentWorkspace.tsx#L485-L502)
- [active rounds 计算](https://github.com/CookSleep/gpt_image_playground/blob/main/src/store.ts#L2673-L2694)
- [多轮主循环与 function_call 处理](https://github.com/CookSleep/gpt_image_playground/blob/7ce8f2f4f66f442a0a0f51ef249ef9ccc9cc0b94/src/store.ts#L3937-L4177)

### 1.3 Agent 回复不是纯 Markdown

参考项目会把助手回复拆成块：

- 普通文本 Markdown。
- web search 状态。
- 批量参数状态。
- 图片任务卡。
- 已删除图片占位。

这点值得保留。Lyra 不应该把所有内容塞进一段 assistant text，否则图片任务状态、失败、重试、删除都会很难维护。

可参考：
- [Assistant block 组装](https://github.com/CookSleep/gpt_image_playground/blob/main/src/components/AgentWorkspace.tsx#L103-L230)
- [output item 到任务块映射](https://github.com/CookSleep/gpt_image_playground/blob/7ce8f2f4f66f442a0a0f51ef249ef9ccc9cc0b94/src/components/AgentWorkspace.tsx#L124-L137)

### 1.4 工具调用链

参考项目真正的多轮循环主要在 `store.ts`，`agentApi.ts` 负责 Responses API 封装、工具 schema 和流式事件解析。

调用链：

1. 用户提交消息。
2. 创建 conversation round 和 user message。
3. 构造 Agent input：历史文本、历史图、当前输入图、可用 refs。
4. 调用 Responses API。
5. 模型返回文本或 function call。
6. 如果是 `generate_image`，执行单张图片任务。
7. 如果是 `generate_image_batch`，预创建多张任务卡并并发执行。
8. 如果是 `continue_generation`，把新 refs 注入下一轮，继续循环。
9. 没有新的 function call 后，本轮结束。

可参考：
- [Agent instructions 与工具策略](https://github.com/CookSleep/gpt_image_playground/blob/7ce8f2f4f66f442a0a0f51ef249ef9ccc9cc0b94/src/lib/agentApi.ts#L26-L80)
- [Agent tools schema](https://github.com/CookSleep/gpt_image_playground/blob/7ce8f2f4f66f442a0a0f51ef249ef9ccc9cc0b94/src/lib/agentApi.ts#L103-L229)
- [Responses API 请求体](https://github.com/CookSleep/gpt_image_playground/blob/7ce8f2f4f66f442a0a0f51ef249ef9ccc9cc0b94/src/lib/agentApi.ts#L690-L750)

### 1.5 图片引用体系

参考项目支持：

- 当前轮输入图：`round-N-reference-M`。
- 历史生成图：`round-N-image-M`。
- 发送给模型时转成 `<ref id="round-1-image-1" />`。
- 已删除或缺失图转成 `<removed_ref id="..." />`。
- 用户可以输入 `@第1轮图2` 指向历史生成图。

Lyra 不建议完整照搬不可见字符方案。更稳的做法是拆成三层：

1. `mentionCodec`：负责输入框里 `@图1` / `@第1轮图2` 的显示和编辑。
2. `agentReferenceRegistry`：保存 `refId -> imageId/status/source/promptSnapshot`。
3. `agentReferenceApiAdapter`：把结构化引用转换成模型可读的 `<ref />` 或 `<removed_ref />`。

可参考：
- [引用解析 helper](https://github.com/CookSleep/gpt_image_playground/blob/7ce8f2f4f66f442a0a0f51ef249ef9ccc9cc0b94/src/lib/agentImageReferences.ts#L4-L83)
- [prompt mention 处理](https://github.com/CookSleep/gpt_image_playground/blob/main/src/lib/promptImageMentions.ts)

## 2. Lyra 的接入原则

### 2.1 独立模块

Agent 新增为独立一级入口，例如 `agent` tab。

必须遵守：

- 不改 `NodeWorkflowPage` 的画布交互。
- 不把 Agent 消息写进画布状态。
- 不把画布对象、缩放、旋转、连线作为 Agent P0 依赖。
- 不复制结果页完整队列、下载、提交广场、重试删除能力。
- 不复制提示词助手四栏页面。

Agent 和画布最多通过显式动作互通：

- “把这张图发送到画布”。
- “把这个任务结果作为 Agent 参考图”。
- “从结果页继续发起 Agent 对话”。

这些都不进入 P0。

### 2.2 产品职责边界

| 模块 | 职责 |
| --- | --- |
| 创作画布 | 主入口，组织参考图、拖拽、粘贴、调整位置、写提示词、发起普通生成。 |
| 提示词助手 | 从一句想法追问、扩写、优化提示词。 |
| Agent 创作 | 多轮目标拆解、计划、调用图片生成任务、把历史结果作为下一轮上下文。 |
| 结果页 | 任务状态、图片资产、下载、重试、提交广场。 |
| API 文档 | 外部调用说明，不承载站内创作状态。 |

## 3. 推荐架构

### 3.1 后端模块

新增：

- `internal/agents`：Agent store/service。
- `internal/api/agents.go`：Agent API 路由。
- per-space 文件：`agent_sessions.json` 或 `agent_runs.json`。

不新增图片队列。图片生成继续复用现有 background task / jobs manager。

最小数据模型：

```go
type AgentSession struct {
    ID        string
    Title     string
    Status    string
    Objective string
    Rounds    []AgentRound
    TaskIDs   []string
    CreatedAt time.Time
    UpdatedAt time.Time
}

type AgentRound struct {
    ID                 string
    ParentRoundID      string
    Index              int
    UserMessageID      string
    AssistantMessageID string
    InputReferenceIDs  []string
    OutputTaskIDs      []string
    Status             string
    Error              string
    CreatedAt          time.Time
    FinishedAt         *time.Time
}

type AgentReference struct {
    RefID          string
    Kind           string
    RoundID        string
    SlotIndex      int
    ImageID        string
    TaskID         string
    Status         string
    PromptSnapshot string
}
```

最小 API：

- `GET /api/agents`：列出会话。
- `POST /api/agents`：创建会话。
- `GET /api/agents/{id}`：查看会话详情。
- `PATCH /api/agents/{id}`：改标题/归档。
- `DELETE /api/agents/{id}`：删除会话元数据。
- `POST /api/agents/{id}/messages`：提交一轮用户消息。
- `POST /api/agents/{id}/image-tasks`：由 Agent 编排创建图片任务，并把 task id 写回 round。

安全要求：

- `/api/agents` 必须加入用户认证保护。
- Agent 创建图片任务必须复用现有额度预检和扣费逻辑。
- 单轮必须限制最大工具调用数、最大图片任务数、最大 continue 次数。

### 3.2 前端模块

新增：

- `web/src/api/agents.ts`
- `web/src/components/agent/AgentPage.tsx`
- `web/src/components/agent/AgentPage.css`
- `web/src/components/agent/AgentConversationList.tsx`
- `web/src/components/agent/AgentMessageList.tsx`
- `web/src/components/agent/AgentComposer.tsx`
- `web/src/components/agent/AgentAssistantBlocks.tsx`
- `web/src/components/agent/agentReferences.ts`

工作台接入：

- `WorkbenchPage.tsx` 增加 `agent` tab。
- 桌面左侧栏显示“Agent 创作”。
- 移动端先放“更多”，不挤占底部主导航。
- Agent 创建任务后可设置结果页 active task，但不要更新画布 latest task。

## 4. P0 MVP

P0 先做“可用闭环”，不做完整参考项目所有能力。

范围：

1. 独立 Agent 页面。
2. 会话列表和单会话详情。
3. 用户输入目标。
4. Agent 生成简短执行计划或直接生成任务参数。
5. 用户确认后调用现有图片任务。
6. 页面显示 task id、状态、跳转结果页。
7. 支持把本轮生成结果作为下一轮参考，但必须人工确认。
8. 限制单轮最多 3 个图片任务、最多 2 次 continue。

P0 不做：

- web search。
- 自动标题。
- 分支编辑。
- 删除轮次重映射。
- mask。
- partial image 持久化。
- 复杂富文本 mention editor。
- 画布对象联动。
- GIF/视频。

## 5. P1/P2

P1：

- 轮次树和 sibling 分支。
- 编辑历史轮次生成新分支。
- 重新生成助手回复。
- 结构化图片引用注册表。
- `@第1轮图2` 选择和解析。
- Agent task source / session id 过滤。
- 自动标题。

P2：

- streaming text delta。
- partial image preview。
- web search。
- 多 Agent 子任务并行。
- Agent 到画布显式发送。
- Agent 模板市场。

## 6. 验收标准

P0 验收：

1. 工作台有独立 Agent 入口，画布仍是默认主入口。
2. `rg -n "agent" web/src/components/NodeWorkflowPage.tsx web/src/components/NodeWorkflowPage.css` 不应因为 Agent 接入新增命中。
3. 新建 Agent 会话后，刷新页面会话仍在。
4. 提交目标后，必须先展示计划或任务参数，用户确认后才扣费创建任务。
5. 创建任务后能在结果页看到同一 task id。
6. Agent 页不复刻完整结果队列，只展示当前会话相关任务块和跳转按钮。
7. 无 Key、额度不足、任务失败时都有清楚状态，不崩溃。
8. 绿色主题下 Agent 页面 active/按钮不出现硬编码黑色覆盖文字。
9. 1920x1080 和 390x844 下无控件重叠。
10. `npx tsc --noEmit --pretty false --incremental false` 通过。
11. `npm run build -- --mode production` 通过。
12. 后端 Agent store/API 测试通过。

## 7. 主要风险

1. 模型循环风险：`continue_generation` 必须有硬限制。
2. 批量扣费风险：批量任务必须二次确认，失败是否退费要跟现有策略一致。
3. 引用错位风险：删除任务或删除轮次后，必须保留 tombstone 或重映射引用。
4. 权限风险：`/api/agents` 不能漏掉用户认证。
5. 重复功能风险：Agent 不做结果页、不做提示词助手、不做画布。
6. 大文件风险：不要照搬参考项目单文件实现，必须拆组件和 service。

## 8. 建议实施顺序

1. 先实现后端 `internal/agents` store 和 CRUD。
2. 加 `/api/agents` 认证路由和测试。
3. 前端加独立 Agent tab 和只读会话 UI。
4. 接入 `POST /api/agents/{id}/image-tasks` 包装现有任务创建。
5. 加最小 Agent 编排：用户目标 -> 计划 -> 确认 -> create task。
6. 加引用注册表，但第一版只支持手动选择历史任务结果作为参考。
7. 再做分支、编辑、重试、`@第N轮图M`。

## 9. 引用 Token 与任务快照补充

参考项目里 `@图N` 不是普通文本。它用不可见 marker 或富文本 span 把“用户选中的图片引用”和“用户手打的 @图1 文本”区分开。这个点很关键，因为多轮 Agent 里最容易出错的就是：界面看起来引用了某张图，实际提交时没有绑定真实图片。

Lyra 的 Agent 模块建议第一阶段就在 Agent 内部实现真实引用 token，不要求创作画布同步改造。

Agent 内部引用 token 建议字段：

```ts
type AgentImageMentionToken = {
  id: string
  label: string
  kind: 'upload' | 'task-result' | 'agent-round-output'
  uploadId?: string
  taskId?: string
  resultIndex?: number
  roundId?: string
  imageIndex?: number
  role?: string
  order: number
  removed?: boolean
}
```

后端保存引用快照，避免历史图片、上传图片、任务结果图片三类 ID 混用后无法复现：

```go
type AgentReferenceSnapshot struct {
    RefID          string
    SourceType     string // upload | task-result | agent-round-output
    UploadID       string
    TaskID         string
    ResultIndex    int
    RoundID        string
    ImageIndex     int
    Role           string
    Order          int
    Removed        bool
    PromptSnapshot string
}
```

P0 要求：

1. Agent 输入框里只有通过选择器插入的 token 才算绑定引用。
2. 用户手打 `@图1` 只当普通文本，不直接绑定图片。
3. 提交前显示“将提交 N 张参考图”，并列出来源。
4. 创建任务时把引用快照保存到 Agent round，不绕过 `/api/uploads/reference` 和 `/api/background-tasks`。
5. 如果引用图片对应的任务结果被删，Agent 不崩溃，而是显示“已移除”并在发给模型时降级为 `<removed_ref />`。

注意：创作画布当前的 `@1` 是轻量文本提示，本轮只用于画布用户体验；Agent 的多轮引用要单独做结构化 token。后续如果需要统一体验，可以再把公共 `mentionCodec` 抽给画布使用，但这不是 Agent P0 的前置条件。
