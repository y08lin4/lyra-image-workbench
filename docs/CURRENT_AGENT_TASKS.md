# 当前任务总账与子代理调度记录

更新时间：2026-06-27  
适用分支：`dev`

本文用于防止上下文压缩、并行 agent 和未提交改动导致任务遗失。当前页只记录本轮有效状态；上一轮残留的旧子代理编号不再视为“运行中”，需要查历史时看 `docs/current-round-task-report.md`、`docs/CURRENT_ROUND_AGENT_CONTROL.md` 和 Git 工作区实际改动。

## 1. 当前总目标

把 Lyra Image Workbench 收口成一个逻辑完整、可自部署的生图站点：

- 用户注册登录后可以配置上游 Key、充值/领取额度、生成图片、查看历史和流水。
- 创作画布是主入口，支持参考图、文字、连线、关系说明和提示词生成。
- 提示词助手、提示词库、Agent、GIF、API 文档作为独立模块，不互相污染状态。
- 广场用于公开作品包，支持复用、点赞、每日榜、参考图公开保存和长期保留。
- 管理员可以初始化站点、设置免费额度、管理用户、配置上游 Key、充值套餐、邮箱和运营参数。
- 前后端必须契约一致，代码必须模块化，不能把所有逻辑塞进单个大文件。

本轮文档任务的目标：清点 `docs`、`README.md`、`PRODUCT.md`、`SECURITY.md` 等文档；新建/重写 `docs/PROJECT_STRUCTURE.md` 和 `docs/CURRENT_AGENT_TASKS.md`；不改前后端代码。

## 2. 本轮六路分工

| 路线 | 当前状态 | 范围 | 输出/验收 |
| --- | --- | --- | --- |
| 1. 前端工作台拆分 | 已回收，构建通过 | `WorkbenchPage.tsx`、`components/workbench/*`、侧栏、页面入口、Tab 分组、移动端更多入口 | 已拆出 `nav.ts`、`WorkbenchSidebar.tsx`、`WorkbenchMobileTabs.tsx`，工作台导航职责从主页面抽离 |
| 2. 广场 CSS 收敛 | 主线程接手收口，待最终浏览器验收 | `PromptSquarePanel.tsx`、`PromptSquarePanel.css` | 已清掉历史重复样式，保留单一 final layout；图片 `contain` 不拉伸，981-1040px 桌面宽度强制两列兜底 |
| 3. 后端模块审查 | 已回收，相关 Go 测试通过 | `internal/api`、`internal/jobs`、`internal/promptsquare`、`internal/retention`、`internal/billing` 等 | 新增 `docs/backend-module-review.md`，记录后端模块边界、耦合点、拆分顺序和风险 |
| 4. MD 文档整理 | 本次已写入 | `docs/PROJECT_STRUCTURE.md`、`docs/CURRENT_AGENT_TASKS.md` | 用简体中文说明项目功能、前后端模块、已完成/未完成、子代理分工、运行/测试命令、文档索引 |
| 5. 全项目结构清点 | 已回收，只读审计完成 | 根目录、`docs/`、`internal/`、`web/src/`、运行数据目录 | 已列出前后端 Top 10 模块化优先级，优先拆 `styles.css`、`jobs/manager.go`、`prompttools/service.go`、`NodeWorkflowPage.tsx` 等 |
| 6. 最终验证 | 已完成命令验证，浏览器细节仍需人工验收 | `go test ./...`、`npm run build`、工作区状态检查 | `npm run build` 通过，`go test ./...` 通过；Vite 仅有 chunk size 警告 |

旧状态处理：上一轮 `CURRENT_AGENT_TASKS.md` 中列出的旧子代理编号已从“当前运行中”表格移除。旧编号和旧结论只作为历史回收材料，不作为当前调度依据。'

## 2.1 本轮回收结论

- 前端工作台已完成小步模块化：导航定义、桌面侧栏、移动端 Tab 已从 `WorkbenchPage.tsx` 拆到 `web/src/components/workbench/`。
- 组件样式已开始从全局 `styles.css` 迁出：`WorkbenchSidebar.css`、`ResultCanvas.css`、`PromptSquarePanel.css` 由 `main.tsx` 在全局样式之后导入。
- 广场样式已收为单一 final layout，不再保留 B1b/Task A/Desktop polish 三段历史覆盖；窄桌面两列兜底已加。
- 后端本轮不做冒险拆源码，先产出 `docs/backend-module-review.md`，明确后端拆分边界和顺序。
- `docs/PROJECT_STRUCTURE.md` 已新增，作为新 agent 接手项目的结构地图和文档索引。
- 命令验证：前端 `npm run build` 通过；后端 `go test ./...` 通过；剩余为浏览器视觉和真实上游/支付链路验收。
'

## 3. 已确认产品决策

### 3.1 参考图编码与存储

参考图不建议长期保存为 base64。base64 只作为 API/SDK 的可选输入格式或上游模型调用时的临时转换格式。后端收到 base64 后应立即解码、校验、计算 hash、落盘或存对象存储，并给出 `reference_id` / URL。

数据库和广场作品包保存引用、元数据和用途备注，不保存大段 base64 文本。这样更利于缓存、去重、清理、权限控制和复用。

### 3.2 广场公开作品包

提交到广场时，结果图、提示词、模型、比例、质量、输出格式、原始参考图和参考图用途备注都要公开保存。未提交到广场的普通生成记录继续按私有临时记录处理，走 30 天清理策略。

广场复用入口应区分：

- 复用提示词：带提示词和参数。
- 完整复刻：带提示词、参数、原始参考图。
- 套用模板：保留结构，用户替换自己的参考图。

### 3.3 视频与 GIF 边界

- 视频/MiniMax 生成功能删除，不属于当前产品范围。
- GIF/动图作为独立模块存在，不混入视频模块，也不复用 MiniMax/video 命名。
- 当前工作区已有 GIF 页面和 `mode=gif` 本地动图任务；它会创建任务记录并保存 `.gif` 结果。
- 当前 GIF 任务不调用视频、不调用 FFmpeg、不调用上游图片生成。
- 后续如接外部 GIF provider，必须单独定义任务模型、额度、部署依赖和输入安全边界。

### 3.4 创作画布定位

创作画布是主入口。生成页可以保留为兼容/快捷入口，但不能和画布抢主流程。画布应支持用户摆放素材、写文字、连线表达关系、生成提示词/图生图。

### 3.5 Agent 定位

Agent 单独一个模块，不影响画布。Agent 负责多轮目标拆解、计划、调用图片生成任务、把历史结果作为下一轮上下文。

### 3.6 Admin 路由

取消单独 `/admin` 前端页面和独立管理入口。管理员能力全部内嵌到工作台里，通过左侧“管理”组的“管理员后台”tab 进入；非管理员不显示入口。后端 `/api/admin/*` 路由必须保留。

## 4. 当前完成快照

已基本完成或已有实现面：

- Go 后端单进程托管 API 和静态前端。
- 用户、Admin、配置、2FA、额度、流水、邀请基础。
- 易支付核心、后端路由、回调验签、重复回调幂等。
- 任务队列、SSE、结果落盘、失败退款、重试/取消。
- Prompt Square 从结果投稿、永久副本、参考图复制、点赞、每日榜、我的投稿。
- 提示词助手、提示词库缓存与创作模式、API 文档页、多主题。
- 创作画布作为主入口的主体能力。
- GIF 独立页面和本地 GIF 生成任务，不调用视频或 FFmpeg。
- Admin 前端独立路径已取消，管理后台以内嵌工作台形态存在。
- `docs/PROJECT_STRUCTURE.md` 已建立项目结构、模块地图、命令和文档索引。
- `docs/CURRENT_AGENT_TASKS.md` 已按本轮六路重写，旧运行中编号不再展示为当前状态。

仍需继续验收或补齐：

- 真实上游 Key 生图、真实支付异步回调、真实广场投稿的浏览器验收。
- 创作画布细节交互：删除、旋转、连线文字、吸附、切页恢复等。
- GIF 外部 provider、计费策略和更丰富动效模板。
- SMTP 真实发信服务和找回/通知闭环。
- API Key 遮罩交互、移动端 375/390/768/1440 视觉检查。
- 多代理工作区仍有大量未提交代码改动，需要集成负责人统一复验。

## 5. P0 收口任务

### 5.1 前端工作台拆分

待验收：

- 左侧导航按创作、素材、管理分组。
- 站点名称、GitHub、主题、余额、当前登录、退出登录位置稳定。
- Admin 只对管理员显示；普通用户不应看到入口。
- `/admin` 不出现另一套独立页面。
- GIF、提示词助手、广场、结果、设置互不污染状态。
- 移动端更多入口不挤压、不丢功能。

### 5.2 广场 CSS 收敛

待验收：

- 一屏可看到标题、筛选和至少一行作品。
- 右上刷新/投稿入口明确。
- 卡片图片保持比例，不拉伸。
- 参考图数量、作者、点赞数、标签信息不挤压。
- 投稿弹窗展示公开保存提醒、参考图缩略图和用途备注。
- 多主题下广场不要出现硬编码黑块或低对比文字。

### 5.3 后端模块审查

待验收：

- `internal/api/router.go` 中路由和前端 wrapper 一致。
- `/api/prompt-square/from-result` 以任务引用快照为可信来源，广场资产复制到永久目录。
- retention 不误删广场结果图和参考图。
- `mode=gif` 创建本地 GIF 任务，不扣图片生成次数，不调用上游/视频/FFmpeg。
- 易支付回调验签、金额校验、PID/支付方式校验、重复回调幂等。
- 普通前端不能拿到明文上游 Key、易支付 Key、SMTP 密码。

### 5.4 MD 文档整理

本次已完成：

- 清点根目录文档、`docs/` 文档、后端模块、前端组件和运行命令。
- 新建 `docs/PROJECT_STRUCTURE.md`。
- 重写 `docs/CURRENT_AGENT_TASKS.md`。
- 旧子代理编号从当前运行中表格删除，改为历史材料说明。

未做：

- 未修改 README、PRODUCT、SECURITY 或前后端代码。
- 未运行全量测试；文档改动只做内容一致性检查。

### 5.5 全项目结构清点

本次文档已覆盖：

- 根目录结构和运行时目录。
- `internal/` 后端模块职责。
- `web/src/` 前端入口、页面、API wrapper 和组件目录。
- 运行/测试/构建命令。
- 文档索引和当前用途。

仍需最终复核：

- 如果其他 agent 继续改代码，`PROJECT_STRUCTURE.md` 中模块状态可能需要同步。
- `VIDEO_GIF_BOUNDARY.md` 是历史扫描快照，当前 GIF 页面已经接入本地单图动效生成，读者需结合实际代码判断。

### 5.6 最终验证

等待集成后执行：

- `go test ./...`
- `npm run build`
- 定向后端测试：users、billing、jobs、promptsquare、retention、api。
- 浏览器验收：画布、GIF、提示词助手、提示词库、广场、API 文档、后台、支付、主题、移动端。
- 残留扫描：MiniMax/video、敏感字段、billing/API 路由。

## 6. P1/P2 产品闭环状态

| 模块 | 当前状态 | 还需要做 |
| --- | --- | --- |
| 用户系统 | 注册、登录、资料、头像、邮箱、2FA、首个用户管理员、用户主页基础已有 | 真实浏览器验收，头像/资料/多设备体验复查 |
| 额度流水 | 余额、流水、管理员加次数、失败退款、每日额度已有实现面 | 生成扣费/失败退款真实路径复验，图表/热力图后续再做 |
| 支付 | 易支付下单、回调验签、幂等入账、订单列表已有实现面 | 真实商户异步回调、公网 `PublicBaseURL` 验收 |
| 邀请 | 邀请链接、首充奖励已有实现面 | 只注册不奖励、首次充值奖励一次的端到端验收 |
| 管理后台 | 内嵌工作台，配置、用户、额度、支付、邮箱已有实现面 | 布局、权限变化、普通用户不可见、初始化流程复验 |
| API | Bearer Key、外部任务 API、站内/外文档草稿已有 | Key 遮罩交互、示例复制、无上游 Key 时禁止创建 Bearer Key 的验收 |
| 广场 | 投稿、参考图、点赞、每日榜、我的投稿已有实现面 | 复用模式、永久资产、移动端和多主题验收 |
| Agent | 独立模块方向明确，页面存在 | 多轮会话、任务块、引用历史结果、扣费前计划仍待做 |
| 移动端/主题 | 多主题和部分响应式已做 | 375/390/768/1440 视觉检查，绿色主题和暗色可读性继续复查 |
| SMTP | 配置和遮罩基础已有 | 真实发信服务、注册/找回/通知闭环未完成 |

## 7. 运行和测试命令

前端构建：

```powershell
cd Z:\github\lyra-image-workbench\web
npm run build
```

前端开发服务：

```powershell
cd Z:\github\lyra-image-workbench\web
npm ci
npm run dev
```

后端运行：

```powershell
cd Z:\github\lyra-image-workbench
go run ./cmd/local-server
```

后端全量测试：

```powershell
cd Z:\github\lyra-image-workbench
go test ./...
```

后端定向测试建议：

```powershell
go test ./internal/users ./internal/billing ./internal/settings
go test ./internal/jobs ./internal/promptsquare ./internal/retention
go test ./internal/api -run "PromptSquare|Billing|Users|AdminUsers|Credit|Referral|GIF|ImageTasks"
```

生产构建：

```powershell
cd Z:\github\lyra-image-workbench\web
npm ci
npm run build

cd Z:\github\lyra-image-workbench
go build -trimpath -ldflags="-s -w" -o .\bin\lyra-image-workbench.exe .\cmd\local-server
```

工作区状态：

```powershell
git -c safe.directory=//fnos1/16T/github/lyra-image-workbench status --short --branch
```

残留和敏感字段扫描：

```powershell
rg -n -i "minimax|videoQuota|MiniMaxVideo|/api/minimax" web/src internal cmd .env.example Dockerfile go.mod web/package.json
rg -n -i "epayKey|smtpPassword|upstreamKey|newApiKey|minimaxApiKey" web/src internal/api
rg -n "/api/billing|billing" internal/api internal/billing web/src/api
```

## 8. 浏览器手工验收清单

- 创作画布：拖图、粘贴、文字、连线、旋转、删除、滚轮缩放、切页恢复、生成提示词。
- GIF：上传图、历史图、模板、创建任务，不污染快捷生成；任务进入历史并保存 `.gif` 结果。
- 提示词助手：提示词优化、灵感跳过、图片还原粘贴/删除/完整预览、继续修改、应用到生成。
- 提示词库：秒开、进入创作模式、例图参考、CORS 失败提示。
- 广场：提交结果、公开参考图提醒、复用策略、模型标签去重、点赞、每日榜、我的投稿。
- API 文档：语言切换、复制按钮、Key 遮罩、主题一致、示例包含创建/轮询/下载/错误处理。
- 后台：初始化、用户列表、用户流水、充值套餐、免费额度、邮箱配置、权限变化。
- 支付：创建易支付订单，模拟或真实回调只入账一次。
- 主题：白蓝、黑紫、白紫、绿色、粉色、蓝色主页面文字可读。
- 移动端：375px、390px、768px 下主要页面和弹窗不溢出、不遮挡。

## 9. 文档索引

优先读：

| 文档 | 用途 |
| --- | --- |
| `README.md` | 公开项目说明、快速开始、部署、安全和更多文档入口。 |
| `docs/PROJECT_STRUCTURE.md` | 项目结构、前后端模块、命令、文档索引。 |
| `docs/CURRENT_AGENT_TASKS.md` | 当前任务、子代理分工、完成/未完成状态。 |
| `docs/REQUIREMENTS_LEDGER.md` | 产品需求总账和状态表。 |
| `docs/current-round-task-report.md` | 本轮任务报告和过程记录。 |
| `docs/CURRENT_ROUND_AGENT_CONTROL.md` | 更详细的当前回合多代理调度记录。 |

产品和设计：

| 文档 | 用途 |
| --- | --- |
| `PRODUCT.md` | 产品目的、品牌语气、设计原则。 |
| `docs/PHASE1_OPTIMIZATION_PLAN.md` | 第一阶段优化计划和验收路径。 |
| `docs/FIRST_PHASE_UI_DESIGN_BRIEF.md` | UI 设计稿和实施边界。 |
| `docs/CLOSED_LOOP_DESIGN.md` | 业务/API/状态机/持久化闭环设计。 |
| `docs/SPACE_DESIGN.md` | 个人空间、用户和 Key 策略。 |
| `docs/PROMPT_SQUARE.md` | 提示词广场试验版口径。 |
| `docs/AGENT_MULTITURN_MODULE_DESIGN.md` | Agent 多轮模块设计。 |

技术和 API：

| 文档 | 用途 |
| --- | --- |
| `docs/STACK.md` | 技术栈和架构约束。 |
| `docs/PROJECT_REQUIREMENTS.md` | 模块化、稳定性和代码要求。 |
| `docs/ROUTES.md` | 路由参考和已移除实验路由。 |
| `docs/EXTERNAL_API_DESIGN.md` | 外部 Bearer API 设计。 |
| `docs/API_DOCUMENTATION_SYNC.md` | API 文档同步稿和 SDK 示例。 |
| `docs/REFERENCE_PROJECT_ANALYSIS.md` | 参考项目分析。 |
| `docs/CHANGES_FROM_AI_IMAGE_GENERATE.md` | 相较参考项目的变化。 |
| `docs/VIDEO_GIF_BOUNDARY.md` | 视频/GIF 边界扫描快照；需结合当前代码判断最新状态。 |

部署和发布：

| 文档 | 用途 |
| --- | --- |
| `docs/DEPLOYMENT.md` | 部署教程索引。 |
| `docs/RELEASE.md` | 多架构 Release 和一键安装。 |
| `docs/DOCKER.md` | Docker/GHCR/Compose。 |
| `docs/DEPLOY_LINUX.md` | Linux、systemd、反代、备份恢复。 |
| `docs/DEPLOY_BAOTA.md` | 宝塔面板部署。 |
| `docs/OPEN_SOURCE_CHECKLIST.md` | 开源发布前检查。 |
| `SECURITY.md` | 安全报告和保护对象。 |
| `CONTRIBUTING.md` | 贡献说明。 |

历史/调度档案：

| 文档 | 用途 |
| --- | --- |
| `docs/AGENT_TASK_BREAKDOWN.md` | SaaS 化多代理任务拆分总控。 |
| `docs/SAAS_MVP_PROGRESS.md` | SaaS MVP 历史进度快照。 |
| `docs/BRANCH_MERGE_SUMMARY.md` | 历史分支合并记录。 |
| `docs/payment-invite-email-verification.md` | 支付、邀请、邮件最小验证说明。 |

## 10. 后续主线程动作

1. 等待本轮六路代理返回，并逐个回收结论。
2. 检查所有改动文件是否越界，尤其是共享高冲突文件。
3. 对前后端契约做一次集中复查：工作台导航、广场、GIF、支付、用户、Admin。
4. 运行 `go test ./...` 和 `npm run build`。
5. 启动本地服务，完成浏览器手工验收清单。
6. 更新 `docs/current-round-task-report.md` 和必要的需求状态。
7. 整理简体中文提交说明，再由主线程决定提交和推送。


