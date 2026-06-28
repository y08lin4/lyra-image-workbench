# 项目结构与文档索引

更新时间：2026-06-27
适用分支：`dev`

本文用于给新接手的人或子代理快速建立项目地图。它整理当前仓库功能、前后端模块、运行/测试命令、文档入口和本轮协作边界，不替代具体需求文档或最终验收报告。

## 1. 项目功能

Lyra Image Workbench（LyAI 生图工作台）是一个 Go 单进程后端 + React/Vite 前端的 AI 生图工作台，可本机运行，也可部署到私有服务器。它把生图请求、账号、任务队列、结果落盘、状态推送和管理后台收拢到同源服务里，浏览器只访问本项目的 `/api`。

当前主要功能面：

- 生图工作台：文生图、图生图、多参考图、模型/比例/清晰度/质量/格式配置。
- 创作画布：主入口，支持图片/文字素材、连线、关系说明、提示词生成和图生图交接。
- 任务系统：后台执行、SSE 事件、轮询兜底、取消、重试、失败退款、结果历史。
- 结果管理：本机落盘、预览、下载、复制、作为参考图、PiXhost 上传、提交到广场。
- 用户系统：注册登录、用户资料、2FA、额度余额、流水、邀请链接。
- 管理后台：初始化、站点/上游/支付/邮箱配置、用户管理、加次数和流水查看。
- 计费闭环：易支付订单、回调验签、重复回调幂等、邀请首充奖励。
- 提示词能力：提示词助手、灵感模式、图片还原、提示词库缓存与创作模式。
- 提示词广场：从结果投稿、作品资产保存、参考图保存、点赞、每日榜、我的投稿。
- API 能力：外部 Bearer API、任务创建/查询/下载、站内 `/api-docs` 文档页。
- GIF 动图：当前已有独立 GIF 页面和 `mode=gif` 本地动图任务；基于单张参考图生成循环 GIF，不调用视频或 FFmpeg。

明确边界：

- 视频/MiniMax 生成功能不属于当前功能面。
- 普通图片生成和 GIF 动图要保持独立，不复用视频命名、路由或 FFmpeg 流程。
- `data/`、`outputs/`、日志、`.env`、用户数据、生成结果和密钥都不应提交。

## 2. 高层架构

```text
Browser / React UI
  │
  │ 同源 /api、/outputs；开发期由 Vite 代理到 127.0.0.1:8787
  ▼
Go local-server
  ├─ HTTP 路由、鉴权、用户 Session、Admin Session
  ├─ 用户、额度、邀请、计费、开发者 Bearer Key
  ├─ 任务队列、SSE、重试、取消、失败退款
  ├─ 上传参考图、任务引用快照、输出图片落盘
  ├─ 提示词助手、提示词库、提示词广场
  ├─ 日清理任务和广场永久资产保护
  └─ NewAPI / OpenAI-compatible image API
```

生产形态下 Go 后端同时托管 `web/dist` 静态前端和全部 API；开发形态下前端由 Vite 提供热更新，`/api`、`/outputs` 代理到 Go 服务。

## 3. 根目录结构

| 路径 | 作用 | 备注 |
| --- | --- | --- |
| `cmd/local-server/` | Go 服务入口 | 装配 config、store、job manager、router、静态服务 |
| `internal/` | 后端业务模块 | 标准库 HTTP，多数状态用文件存储 |
| `web/` | React + TypeScript + Vite 前端 | `web/src/main.tsx` 是浏览器入口 |
| `docs/` | 产品、架构、部署、调度和 API 文档 | 本文第 10 节有索引 |
| `scripts/` | 安装、部署、本地重启脚本 | Linux、宝塔和 Windows 辅助脚本 |
| `data/` | 运行时数据 | 不提交；用户、配置、任务、缓存、广场资产等 |
| `outputs/` | 生成图片输出 | 不提交；旧输出 URL 仍需鉴权 |
| `logs/`、`tmp/`、`bin/` | 本地运行、临时和构建产物 | 不作为源码入口 |
| `.env.example` | 环境变量示例 | 真实 `.env` 不提交 |
| `Dockerfile` | 生产镜像构建 | 配合 GHCR/Docker 文档 |
| `README.md` | 公开项目首页 | 功能、快速开始、部署和安全口径 |
| `PRODUCT.md` | 产品语气和设计原则 | 面向产品/设计协作 |
| `SECURITY.md` | 安全报告和保护对象 | 面向漏洞报告和隐私边界 |
| `CONTRIBUTING.md` | 贡献说明 | 开发、提交、安全注意事项 |

## 4. 后端模块

| 路径 | 职责 |
| --- | --- |
| `internal/api/` | HTTP 路由、请求/响应、鉴权包装；`router.go` 注册全部 API。 |
| `internal/adminauth/` | Admin 初始化、登录、Session 存储。 |
| `internal/apikeys/` | 外部 Bearer API Key 管理。 |
| `internal/billing/` | 易支付订单、签名、状态流转、幂等处理。 |
| `internal/config/` | 环境变量、默认模型、监听地址、数据目录、超时配置。 |
| `internal/events/` | SSE 事件中心。 |
| `internal/jobs/` | 任务模型、状态机、后台执行、引用快照、扣费/退款、GIF 本地动图生成。 |
| `internal/llm/` | 提示词类 LLM 调用封装。 |
| `internal/newapi/` | NewAPI/OpenAI 兼容图片接口客户端。 |
| `internal/output/` | 生成结果保存、MIME/扩展名判断、鉴权读取。 |
| `internal/passwordhash/` | 密码哈希。 |
| `internal/pixhost/` | PiXhost 图片上传。 |
| `internal/promptlibrary/` | 提示词库解析、图片缓存、本地刷新。 |
| `internal/promptsquare/` | 广场作品、永久图片、参考图复制、点赞、每日榜。 |
| `internal/prompttools/` | 提示词优化、图片还原、灵感模式、历史和会话。 |
| `internal/retention/` | 30 天清理、路径守卫、广场资产保护。 |
| `internal/server/` | HTTP server 超时和 handler 装配。 |
| `internal/settings/` | 全局运行配置、Admin 配置、支付/邮箱/上游设置。 |
| `internal/spaceconfig/` | 用户空间 Key 状态和个人配置。 |
| `internal/spaces/` | 用户空间文件存储和 storage token。 |
| `internal/statusmeta/` | 状态元数据。 |
| `internal/uploads/` | 图生图参考图上传和读取。 |
| `internal/users/` | 用户、角色、Session、2FA、额度、流水、邀请。 |
| `internal/minimax/` | 历史遗留空目录；当前没有生产代码入口。 |

主要 API 分组：

- `/api/admin/*`：Admin 初始化、登录、配置、用户管理。
- `/api/users/*`：注册登录、当前用户、资料、流水、每日额度、2FA、邀请。
- `/api/billing/*`：充值选项、易支付下单/查询/回调、订单列表。
- `/api/config`：用户空间配置和 Key 状态。
- `/api/developer/api-keys`：外部 Bearer Key。
- `/api/uploads/reference`：参考图上传、列表、读取、删除。
- `/api/background-tasks/*`：图片/GIF 任务创建、列表、详情、事件、重试、取消、收藏、删除、图片读取。
- `/v1/images/generations`、`/v1/image-tasks/*`：外部兼容 API。
- `/api/prompt-tools/*`：提示词助手、会话、灵感、历史。
- `/api/prompt-library/*`：提示词库缓存和图片读取。
- `/api/prompt-square/*`：广场列表、投稿、从结果投稿、点赞、每日榜、我的投稿、图片读取。
- `/outputs/{space}/{date}/{file}`：旧结果图路径，仍需登录鉴权。

## 5. 前端模块

| 路径 | 职责 |
| --- | --- |
| `web/src/main.tsx` | 入口；主题初始化；`/api-docs` 公共页；`/admin` 前端路径回到工作台；站点未初始化时展示初始化流程。 |
| `web/src/components/WorkbenchPage.tsx` | 工作台外壳、导航、任务提交、页面切换、状态整合。 |
| `web/src/components/GenerationPanel.tsx` | 兼容/快捷生成表单。 |
| `web/src/components/NodeWorkflowPage.tsx` | 创作画布主入口。 |
| `web/src/components/GifPage.tsx` | GIF 独立页面和单图动效任务参数。 |
| `web/src/components/PromptAssistantModal.tsx` | 提示词助手一级页面/嵌入模式。 |
| `web/src/components/PromptLibraryPage.tsx` | 提示词库、缓存、创作模式、例图复用。 |
| `web/src/components/PromptSquarePanel.tsx` | 提示词广场、筛选、榜单、投稿展示。 |
| `web/src/components/ResultCanvas.tsx` | 结果区、任务历史、投稿广场、图片操作。 |
| `web/src/components/TaskSidebar.tsx`、`TaskDetailModal.tsx` | 任务列表和任务详情。 |
| `web/src/components/AdminPage.tsx` | 管理后台，支持初始化和工作台内嵌模式。 |
| `web/src/components/ProfilePage.tsx`、`TopUpPage.tsx` | 用户主页、流水、邀请、充值入口。 |
| `web/src/components/ApiDocsPage.tsx` | 站内/公开 API 文档页面。 |
| `web/src/components/SettingsPanel.tsx` | 当前用户配置、Key、2FA 等设置。 |
| `web/src/api/` | 前端 API wrapper。 |
| `web/src/api/contracts/` | 前端请求/响应类型契约。 |
| `web/src/components/creativeCanvas/` | 画布数据、几何、交互、持久化、提示词生成等辅助模块。 |
| `web/src/components/promptAssistant/` | 提示词助手常量、模板、结果展示、灵感数据。 |
| `web/src/lib/` | 主题、比例、模型、格式、错误标签、本地 Key、原生桥等工具。 |
| `web/src/styles.css` | 全局主题 token 和共享样式。 |
| `web/src/components/*.css` | 页面/组件级样式。 |

前端当前关键状态：

- 侧栏按“创作 / 素材 / 管理”分组，Admin 只对管理员可见。
- `/api-docs` 可公开访问并复用主题。
- `/admin` 不再作为独立前端页面使用，会回到工作台或初始化流程。
- GIF 页面不会污染快捷生成状态；会创建独立 `mode=gif` 任务并进入结果历史。
- 画布草稿、提示词库缓存、已投稿标记等部分状态会使用浏览器本地存储。

## 6. 运行时数据

默认目录由 `internal/config/config.go` 控制：

| 路径 | 内容 |
| --- | --- |
| `data/config.local.json` | Admin 配置和运行时配置。 |
| `data/admin.auth.json` | Admin 鉴权信息。 |
| `data/users.json` | 用户、角色、额度、流水、邀请。 |
| `data/api_keys.json` | 外部 Bearer API Key。 |
| `data/topups.json` | 充值订单。 |
| `data/cache/prompt-library/` | 提示词库缓存。 |
| `data/prompt_square/` | 广场作品、永久图片和参考图副本。 |
| `data/spaces` 相关文件 | 用户空间配置、上传参考图、任务记录等。 |
| `outputs/` | 普通生成结果。 |

清理规则：未提交广场的普通生成结果按 30 天清理；已提交广场的结果和参考图需要永久保留；删除运行时文件必须经过路径守卫，避免越界删除。

## 7. 环境变量

常用变量见 `.env.example`：

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `LOCAL_IMAGE_HOST` | `0.0.0.0` | 服务监听地址；反代部署建议 `127.0.0.1`。 |
| `LOCAL_IMAGE_PORT` | `8787` | 服务监听端口。 |
| `LOCAL_IMAGE_DATA_DIR` | `data` | 运行时数据目录。 |
| `LOCAL_IMAGE_WEB_DIR` | `web/dist` | 前端生产构建目录。 |
| `LOCAL_IMAGE_ADMIN_SETUP_TOKEN` | 空 | 首次初始化 Admin 必须设置。 |
| `NEWAPI_BASE_URL` | `http://127.0.0.1:3000/v1` | OpenAI 兼容图片网关地址。 |
| `NEWAPI_TIMEOUT_SEC` | `600` | 单张图片请求超时，范围 60-3600 秒。 |

## 8. 运行和测试命令

本机生产形态：

```powershell
cd Z:\github\lyra-image-workbench\web
npm ci
npm run build

cd Z:\github\lyra-image-workbench
go run ./cmd/local-server
```

访问：`http://127.0.0.1:8787`

前后端开发形态：

```powershell
cd Z:\github\lyra-image-workbench
go run ./cmd/local-server
```

```powershell
cd Z:\github\lyra-image-workbench\web
npm ci
npm run dev
```

访问：`http://127.0.0.1:5173`

测试和构建：

```powershell
cd Z:\github\lyra-image-workbench
go test ./...
go build -trimpath -ldflags="-s -w" -o .\bin\lyra-image-workbench.exe .\cmd\local-server
```

```powershell
cd Z:\github\lyra-image-workbench\web
npm run build
npx tsc --noEmit --pretty false --incremental false
```

常用扫描：

```powershell
rg -n -i "minimax|videoQuota|MiniMaxVideo|/api/minimax" web/src internal cmd .env.example Dockerfile go.mod web/package.json
rg -n -i "epayKey|smtpPassword|upstreamKey|newApiKey|minimaxApiKey" web/src internal/api
rg -n "/api/billing|billing" internal/api internal/billing web/src/api
```

## 9. 当前完成与未完成快照

已基本完成或已有实现面：

- Go 后端单进程托管 API 和静态前端。
- 用户、Admin、配置、2FA、额度、流水、邀请基础。
- 易支付核心、后端路由、回调验签、重复回调幂等。
- 任务队列、SSE、结果落盘、失败退款、重试/取消。
- Prompt Square 从结果投稿、永久副本、参考图复制、点赞、每日榜、我的投稿。
- 提示词助手、提示词库缓存与创作模式、API 文档页、多主题。
- 创作画布作为主入口的主体能力。
- GIF 独立页面和本地 GIF 生成任务，不调用视频或 FFmpeg。

仍需继续验收或补齐：

- 真实上游 Key 生图、真实支付异步回调、真实广场投稿的浏览器验收。
- 创作画布细节交互：删除、旋转、连线文字、吸附、切页恢复等。
- GIF 外部 provider、计费策略和更丰富动效模板。
- SMTP 真实发信服务和找回/通知闭环。
- API Key 遮罩交互、移动端 375/390/768/1440 视觉检查。
- 多代理工作区仍有大量未提交代码改动，需要集成负责人统一复验。

## 10. 文档索引

根目录文档：

| 文档 | 用途 |
| --- | --- |
| `README.md` | 公开项目说明、功能总览、快速开始、部署、安全策略和更多文档入口。 |
| `PRODUCT.md` | 产品目标、品牌语气、设计原则和反参考。 |
| `SECURITY.md` | 安全支持范围、报告方式、重点保护信息。 |
| `CONTRIBUTING.md` | 开发环境、提交建议、安全与隐私注意事项。 |
| `LICENSE` | MIT 许可证。 |

`docs/` 文档：

| 文档 | 用途 |
| --- | --- |
| `PROJECT_STRUCTURE.md` | 当前项目结构、模块地图、运行命令和文档索引。 |
| `CURRENT_AGENT_TASKS.md` | 当前任务总账、子代理分工、完成/未完成状态和验收命令。 |
| `CURRENT_ROUND_AGENT_CONTROL.md` | 当前回合更详细的多代理总控记录。 |
| `current-round-task-report.md` | 本轮任务过程报告和回收记录。 |
| `REQUIREMENTS_LEDGER.md` | 产品需求总账和状态表。 |
| `SAAS_MVP_PROGRESS.md` | SaaS MVP 阶段进度快照和历史调度状态。 |
| `AGENT_TASK_BREAKDOWN.md` | SaaS 化多代理任务拆分总控。 |
| `AGENT_MULTITURN_MODULE_DESIGN.md` | Agent 多轮创作模块设计。 |
| `CLOSED_LOOP_DESIGN.md` | 任务、API、持久化和移动端闭环设计。 |
| `PHASE1_OPTIMIZATION_PLAN.md` | 第一阶段优化计划、验收路径和技术验证命令。 |
| `FIRST_PHASE_UI_DESIGN_BRIEF.md` | 第一阶段 UI 设计稿和实施边界。 |
| `STACK.md` | 技术栈选型、请求边界和前后端原则。 |
| `PROJECT_REQUIREMENTS.md` | 项目开发要求、模块化和稳定性要求。 |
| `ROUTES.md` | 路由参考、本地化设计和已移除实验路由。 |
| `EXTERNAL_API_DESIGN.md` | 外部 Bearer API 设计。 |
| `API_DOCUMENTATION_SYNC.md` | 站内/外 API 文档同步草稿和 SDK 示例。 |
| `PROMPT_SQUARE.md` | 提示词广场试验版设计。 |
| `SPACE_DESIGN.md` | 个人空间和第一版用户设计。 |
| `VIDEO_GIF_BOUNDARY.md` | 视频删除和 GIF 独立模块边界扫描；部分内容是快照，应结合当前代码再判断。 |
| `REFERENCE_PROJECT_ANALYSIS.md` | 参考项目路由和闭环分析。 |
| `CHANGES_FROM_AI_IMAGE_GENERATE.md` | 相较参考项目的功能和架构更新。 |
| `BRANCH_MERGE_SUMMARY.md` | 历史分支合并摘要。 |
| `DEPLOYMENT.md` | 部署教程总索引。 |
| `RELEASE.md` | 多架构 Release 包和一键安装。 |
| `DOCKER.md` | Docker/GHCR/Compose 部署。 |
| `DEPLOY_LINUX.md` | Linux、systemd、反代、备份恢复。 |
| `DEPLOY_BAOTA.md` | 宝塔面板部署。 |
| `payment-invite-email-verification.md` | 支付、邀请和邮件配置最小验证。 |
| `OPEN_SOURCE_CHECKLIST.md` | 开源发布前检查清单。 |

## 11. 本轮协作注意事项

本轮按六路并行收口：前端工作台拆分、广场 CSS 收敛、后端模块审查、MD 文档整理、全项目结构清点、最终验证。文档整理只写 `docs/PROJECT_STRUCTURE.md` 和 `docs/CURRENT_AGENT_TASKS.md`，不改前后端代码。

协作规则：

- 当前 `dev` 工作区有多个 agent 的未提交改动；不要回滚他人代码。
- 同一文件同一时间只应有一个写入负责人；其他代理以只读审查或测试为主。
- 共享高冲突文件包括 `internal/api/router.go`、`internal/jobs/*`、`internal/promptsquare/*`、`web/src/components/WorkbenchPage.tsx`、`web/src/styles.css`、`web/src/types.ts`、`web/src/api/*`。
- 改动后必须报告改动文件、验证命令和未验证风险。
