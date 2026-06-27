# 当前回合任务台账与子代理总控

更新时间：2026-06-27
分支：dev
总控原则：主代理不直接推进大功能实现；主代理负责记录需求、拆分任务、派发子代理、回收结果、验收、再派新的子代理。除非页面已被语法错误阻断且用户明确要求先恢复，否则功能改动都交给子代理完成。

## 0. 本轮工作方式

1. 所有用户新增需求先进入本文档，不口头记忆。
2. 每个任务必须写清楚：优先级、改动范围、不可改范围、验收标准、负责子代理。
3. 主代理职责：维护台账、拆分任务、派发子代理、关闭完成代理、安排验收代理、统一跑最终验证。
4. 子代理职责：只做一个小任务，不越界，不回滚别人改动，完成后列出改动文件和验证结果。
5. 同一文件同一时间只允许一个写入负责人；冲突任务先改成只读审计。
6. 任何支付、额度、管理员、密钥、清理策略相关任务，必须经过二次验收代理复核。
7. 当前最多保持 6 个子代理运行；有代理结束就回收，再补新的高优先级任务。

## 1. 当前 P0 阻断

| 编号 | 任务 | 背景 | 负责人 | 状态 | 验收标准 |
| --- | --- | --- | --- | --- | --- |
| P0-1 | 修复 Vite 编译崩溃 | 页面报 `web/src/api/tasks.ts:75:34 Unterminated string literal`，工作台无法使用 | 待派子代理 | 待派发 | `npx tsc --noEmit --pretty false --incremental false` 通过；浏览器不再显示 Vite error overlay；确认不是掩盖子代理未合并冲突。 |
| P0-2 | 当前任务台账落地 | 用户要求先记录、拆分、派发、验收，不要主代理自己直接做功能 | 主代理 | 进行中 | 本文档存在且覆盖本轮所有明确需求；后续派发引用本文档编号。 |

## 2. 创作画布任务组

### C1. 画布输入与引用语义

| 编号 | 任务 | 需求原话口径 | 建议负责人 | 写入范围 | 不可改 | 验收标准 |
| --- | --- | --- | --- | --- | --- | --- |
| C1-1 | Ctrl+V 粘贴图片进画布 | 鼠标点一下画布后，可以 `Ctrl+V` 把剪贴板图片粘到画布 | Canvas 子代理 | `web/src/components/NodeWorkflowPage.tsx`、画布相关 CSS | API 文档、后台、主题全局大改 | 点击画布后粘贴图片，图片出现在点击附近；非图片粘贴不报错；仍支持拖拽上传。 |
| C1-2 | `@` 写入底部提示词输入框 | `@` 的意思是插入到底部输入框，不只是图片上打标签 | Canvas 子代理 | `NodeWorkflowPage.tsx` | 不改后端契约 | 右键或按钮选择 `@` 后，底部 textarea 出现 `@1 参考...` 文本或 chip；焦点回到底部输入框；提交任务时引用语义可见。 |
| C1-3 | 删除过细参考角色 | “服饰/地板材质/背景/构图/光线”太具体，不需要，用户自己写 | Canvas 子代理 | `NodeWorkflowPage.tsx` | 不改历史数据结构 | 角色只保留通用项，如“参考/主体/风格”；右侧不再出现过细分类按钮。 |
| C1-4 | 删除三枚场景预设按钮 | “产品主图/海报改写/参考图改写”意义不大，删掉 | Canvas 子代理 | `NodeWorkflowPage.tsx`、画布 CSS | 不删模型/比例/质量参数 | 页面不再显示这三个预设按钮。 |

### C2. 1080P 一屏工作台布局

| 编号 | 任务 | 需求 | 建议负责人 | 写入范围 | 验收标准 |
| --- | --- | --- | --- | --- | --- |
| C2-1 | 画布按 1080P 电脑设计 | 画布、预览、输入、参数一屏装下，不要往下翻 | Canvas/Layout 子代理 | `NodeWorkflowPage.tsx`、`web/src/styles.css` 画布区 | 1920x1080 下顶部导航可见，画布/右侧预览/底部输入与参数都在第一屏；不需要向下滚动才能生成。 |
| C2-2 | 高级参数并入同一行 | 高级参数不要另起一大片区域，跟模型/比例/清晰度放一排 | Canvas/Layout 子代理 | 画布 composer JSX/CSS | 质量、格式、数量、并发与模式/模型/比例在同一参数行或同一紧凑区。 |
| C2-3 | 输入区贴近操作区 | 既能看画布又能输入 | Canvas/Layout 子代理 | 画布 composer CSS | textarea 和生成按钮始终靠近画布底部，不被大空白隔开。 |
| C2-4 | 下一版左侧栏导航设计 | 用户建议把顶部 TAB 全部放左边做栏 | IA/UI 子代理 | 先做设计稿，不直接实现 | 输出左侧栏方案：左栏宽度、图标/文字、移动端折叠、对 1080P 画布的一屏影响。 |

### C3. 画布对象能力

| 编号 | 任务 | 需求 | 状态 | 验收标准 |
| --- | --- | --- | --- | --- |
| C3-1 | 外部图片拖入画布 | 从外部拖图吸附在画布上 | 已有部分，待验收 | 拖入图片生成可移动对象。 |
| C3-2 | 缩放/旋转/移动 | 图片可调整大小和方向 | 已有部分，待验收 | 鼠标拖动手柄可缩放、旋转、移动；图片不变形。 |
| C3-3 | 连线 | 几张图可连接 | 已有部分，待验收 | 可以选择图片 A 后连接图片 B；连线不挡住主要操作。 |
| C3-4 | 历史图拖入和 `@` 多图语义 | 历史生成图可拖到画布，`@1/@2/@3` 表达主体/服饰/地板等由用户自己写 | 待加强 | 多图引用顺序清楚；提交前显示将使用哪些图。 |

## 3. API 文档重构任务组

| 编号 | 任务 | 背景 | 建议负责人 | 写入范围 | 验收标准 |
| --- | --- | --- | --- | --- | --- |
| A1 | API 文档整体重构 | 当前像代码堆叠，页面压抑，信息层级不清 | API Docs 子代理 | `web/src/components/ApiDocsPage.tsx`、API docs CSS | 页面像“SDK 接入控制台”：左侧最短步骤/Key/端点，右侧语言示例/AI 提示词；少废话；不出现大面积灰暗遮罩。 |
| A2 | 代码块跟随主题 | 当前代码区像蒙灰/暗色遮罩，绿色主题也压暗 | API Docs 子代理 + Theme 验收 | API docs CSS、主题 token | 代码块背景使用 `surface-soft` 或主题 code token；白/绿主题不发暗；黑色主题才用深色代码区。 |
| A3 | 给 AI 的提示词完整化 | 不要让 AI 去查仓库或外部文档 | API Docs 子代理 | `ApiDocsPage.tsx` 文案 | 提示词包含 Base URL、认证、创建任务、轮询、下载、错误码、参数、环境变量；复制后 AI 可直接写 SDK。 |
| A4 | Key 遮罩体验 | Key 黑色遮罩、悬浮查看、点击复制完整 | API Docs 子代理 | `ApiDocsPage.tsx` | 有 Key 时默认遮罩，hover/focus 可查看，点击复制完整 Key；无 Key 时复制占位符。 |
| A5 | 语言示例两栏/紧凑布局 | curl/TS/Python/Go/Java 不要把页面拖得很长 | API Docs 子代理 | `ApiDocsPage.tsx` | 语言 tab 清晰，代码可滚动，不撑破页面；复制按钮固定可见。 |

## 4. 主题系统审查任务组

| 编号 | 任务 | 问题 | 建议负责人 | 写入范围 | 验收标准 |
| --- | --- | --- | --- | --- | --- |
| T1 | 绿色主题黑色按钮清理 | 选绿色后很多按钮/active 仍是黑色，文字被盖住 | Theme 子代理 | `web/src/styles.css`、`web/src/lib/themes.ts` | 绿色主题下主按钮、选中态、充值档位、API docs tab、画布按钮均不出现大块默认黑色；文字可读。 |
| T2 | 全主题 hardcode 扫描 | 旧 CSS 中大量 `#111827/#0f172a/#020617` | Theme 子代理 | CSS 后段主题覆盖 | 保留必要深色主题用色；浅色/绿色/粉色/蓝色主题不被硬编码黑覆盖。 |
| T3 | 主题增删建议 | 用户允许自己增加或删除主题 | Theme 子代理 | 先输出建议，再改 | 给出保留主题清单：白蓝、黑紫、白紫、绿色、粉色、蓝色、黑色/白色是否保留。 |
| T4 | API 文档代码区主题 | 与 A2 联动 | Theme 子代理 | CSS | API 文档不再像有暗色遮罩。 |
| T5 | 全站对比度验收 | 夜间模式和多主题看不清 | Theme 验收代理 | 只读/截图 | 重点验收：按钮、输入框、tab、充值、后台、API 文档、画布、提示词库。 |

## 5. 管理后台任务组

| 编号 | 任务 | 问题 | 建议负责人 | 写入范围 | 验收标准 |
| --- | --- | --- | --- | --- | --- |
| M1 | 删除后台重复导航 | 同一页面顶部 tab 和底部按钮重复 | Admin 子代理 | `web/src/components/AdminPage.tsx`、`web/src/components/admin/**`、`AdminPage.css` | 只保留顶部 tab；底部“系统配置/额度支付/邮件设置/用户管理/用户流水”重复按钮消失。 |
| M2 | 管理后台更立体 | 管理后台与设置页职责区分；单独用户列表/用户页 | Admin 子代理 | AdminPage 拆分组件 | 后台内部有总览、系统配置、额度支付、邮件、用户管理、用户流水；普通设置页不混后台管理。 |
| M3 | 管理员初始化提醒 | 安装令牌用途和设置步骤要提醒 | Backend/Admin 子代理 | 初始化页与文档 | 初始化页说明 `LOCAL_IMAGE_ADMIN_SETUP_TOKEN` 来源和未设置后果。 |
| M4 | 免费额度配置 | 管理员可设置新用户初始额度、每日免费额度 | Admin/Backend 子代理 | settings/admin | 前后端字段对应；保存后总览可见。 |

## 6. 提示词库缓存任务组

| 编号 | 任务 | 问题 | 建议负责人 | 写入范围 | 验收标准 |
| --- | --- | --- | --- | --- | --- |
| L1 | 启动预热提示词库 | 点击提示词库时还在加载，本机应该秒开 | Prompt Library 子代理 | `internal/promptlibrary/**`、相关 API、前端提示词库组件 | 服务启动后后台预热本地缓存；页面优先展示本地缓存；远端同步后台刷新。 |
| L2 | GitHub 图片本地化 | 提示词库图片不应长期指向 GitHub raw | Prompt Library 子代理 | image cache | 图片保存到本机，用本机链接展示；仓库有新增再同步。 |
| L3 | 加载状态优化 | 不要空白等很久 | Prompt Library 子代理 | 前端提示词库组件/CSS | 有缓存时秒出内容；无缓存才显示轻量 skeleton；不阻塞搜索。 |

## 7. GIF 动图模式任务组

| 编号 | 任务 | 需求 | 建议负责人 | 状态 | 验收标准 |
| --- | --- | --- | --- | --- | --- |
| G1 | GIF 与视频边界 | 视频删除，GIF 单独模块 | GIF 设计子代理 | 待派 | 文档明确 GIF 不是视频入口，不恢复 MiniMax/video。 |
| G2 | 上传图片 + 动效描述 | 上传一张图片，说“头发动起来”等生成 GIF | GIF 设计/后端子代理 | 待设计 | 输出模式设计：输入图片、动效提示词、模板、任务状态、结果 GIF。 |
| G3 | 预设动效模板 | 让图片动起来，支持一些模板 | GIF 子代理 | 待设计 | 模板如轻微摇摆、头发飘动、镜头推进、眨眼、光影流动；不依赖不安全 FFmpeg 解码用户视频。 |
| G4 | FFmpeg 安全 | gif 分支会用 FFmpeg，要考虑高危漏洞 | Security 子代理 | 待设计 | 文档要求生产使用修复版本 FFmpeg；限制输入类型和处理范围；不接收用户上传视频解码链路。 |

## 8. 模块化和前后端契约任务组

| 编号 | 任务 | 目标 | 当前代理 | 状态 | 验收标准 |
| --- | --- | --- | --- | --- | --- |
| R1 | 前端 API contracts 拆分 | 前后端字段对应，避免堆在 types.ts | Euclid | 运行中 | `web/src/api/contracts/**` 清晰；`tsc` 通过。 |
| R2 | AdminPage 拆分 | 后台不要堆一个大文件 | Aquinas | 运行中 | 拆成 admin 子组件；删除重复底部导航；`tsc` 通过。 |
| R3 | PromptAssistant 拆分 | 提示词助手模块化 | Harvey | 已完成 | 常量/灵感逻辑/结果面板已拆；已通过 `tsc`。 |
| R4 | WorkbenchPage hooks 拆分 | 主工作台减少 localStorage/提交逻辑堆积 | Averroes | 已完成 | `useSubmittedSquareKeys` 已拆；已通过 `tsc`。 |
| R5 | 后端 settings 拆分 | settings update/校验解耦 | Faraday | 运行中 | 后端设置相关测试通过。 |
| R6 | 提示词库缓存拆分 | 缓存/同步/前端读取解耦 | Tesla | 运行中 | 本地缓存优先、后台同步、相关测试或构建通过。 |

## 9. 当前子代理池

| 子代理 | 状态 | 当前任务 | 下一步 |
| --- | --- | --- | --- |
| Euclid | 运行中 | 前端 API contracts 拆分 | 等完成后回收并派验收代理。 |
| Aquinas | 运行中 | AdminPage 拆分；追加删除后台重复导航 | 等完成后回收，检查 M1。 |
| Harvey | 已完成并关闭 | PromptAssistant 拆分 | 后续可派新的提示词助手 UI 验收。 |
| Averroes | 已完成并关闭 | WorkbenchPage hooks 拆分 | 后续可派 Workbench 集成验收。 |
| Faraday | 运行中 | 后端 settings 拆分 | 等完成后跑相关 Go 测试。 |
| Tesla | 运行中 | 提示词库缓存预热 | 等完成后验收 L1/L2/L3。 |

当前空位：2 个。
建议马上补位：
1. Crash Fix 子代理：处理 P0-1。
2. API Docs Redesign 子代理：处理 A1-A5。

## 10. 验收流程

### 10.1 单任务验收

每个子代理完成后必须提供：

1. 改动文件列表。
2. 自己运行的命令和结果。
3. 是否越界改动。
4. 已知风险。
5. 下一步建议。

主代理验收：

1. 检查 `git status --short`。
2. 检查子代理改动是否在 ownership 内。
3. 按任务验收标准运行最小命令。
4. 如失败，不直接大改，派修复子代理或让原代理补丁。
5. 验收通过后更新本文档状态。

### 10.2 当前 P0 验收命令

```powershell
cmd /d /c "pushd Z:\github\lyra-image-workbench\web && npx tsc --noEmit --pretty false --incremental false"
```

浏览器验收：

1. 页面不再显示 Vite overlay。
2. 后台、创作画布、API 文档能进入。
3. 若仍报错，记录新文件和行号，继续派 P0 修复子代理。

### 10.3 最终集成验收

```powershell
go test ./...
cmd /d /c "pushd Z:\github\lyra-image-workbench\web && npx tsc --noEmit --pretty false --incremental false"
cmd /d /c "pushd Z:\github\lyra-image-workbench\web && npm run build -- --mode production"
git -c safe.directory=//fnos1/16T/github/lyra-image-workbench diff --check
```

页面人工验收重点：

1. 1920x1080 创作画布一屏工作台。
2. Ctrl+V 粘贴图片。
3. `@` 写入底部提示词。
4. 绿色主题无黑色 active 覆盖。
5. API 文档不再灰暗、不是代码堆。
6. 后台页无重复导航。
7. 提示词库二次打开接近秒开。

## 11. 暂不执行但已记录的下一版方向

1. 左侧栏替代顶部 tabs：需要先出设计稿，再实施。
2. 创作画布作为主入口：当前方向已确认，但页面结构还需按 1080P 一屏重排。
3. 提示词助手 ChatGPT/Claude 式灵感模式：已做部分拆分，后续继续设计与验收。
4. 用户系统、充值、邀请、每日榜、清理器：已有闭环实现记录，但还需要端到端人工验收。
5. 外部 API 文档仓库同步：站内文档重构后，需要同步 `y08lin4/LyAi-Image-Generation-API-Documentation`。

## 12. 子代理回收记录（2026-06-27 当前批次）

| 子代理 | 原任务 | 回收状态 | 改动文件 | 自验结果 | 主控后续 |
| --- | --- | --- | --- | --- | --- |
| Faraday | 后端 settings 拆分 | 已完成并关闭 | `internal/settings/update.go`、`internal/settings/update_test.go`、`internal/api/admin_auth.go` | `go test -C '\\fnos1\16T\github\lyra-image-workbench' ./internal/settings ./internal/api` 通过 | 后续若继续拆 `settings.go`，需单独派任务，避免和现有 dirty change 冲突。 |
| Euclid | 前端 API contracts 拆分 | 已完成并关闭 | `web/src/api/contracts/**`、`web/src/api/*.ts`、`web/src/types.ts` | `npx tsc --noEmit --pretty false --incremental false` 通过；targeted `git diff --check` 通过 | 需要 P0 崩溃验收代理复跑全 web 类型检查，确认 `tasks.ts` 当前无语法错误。 |
| Aquinas | AdminPage 拆分/删除重复导航 | 已完成并关闭 | `web/src/components/AdminPage.tsx`、`web/src/components/admin/**` | `npx tsc --noEmit --pretty false --incremental false` 通过；底部重复跳转文本无残留 | 需要页面人工验收后台顶部 tab 是否够清楚，底部重复按钮是否消失。 |
| Harvey | PromptAssistant 拆分 | 已完成并关闭 | `web/src/components/promptAssistant/**`、`PromptAssistantModal.tsx` | `npx tsc --noEmit --pretty false --incremental false` 通过 | 后续再派提示词助手 UI 验收，不在本轮 P0 前抢占。 |
| Averroes | Workbench hooks 拆分 | 已完成并关闭 | `web/src/hooks/useSubmittedSquareKeys.ts`、`WorkbenchPage.tsx` | `npx tsc --noEmit --pretty false --incremental false` 通过 | 后续由集成验收代理检查提交广场状态是否正常。 |

当前仍运行：Tesla（提示词库缓存预热）。

## 13. 新派发优先级队列

| 顺位 | 任务 | 原因 | 是否可并行 | 写入冲突策略 |
| --- | --- | --- | --- | --- |
| 1 | P0-1 Vite 崩溃修复/验证 | 当前页面可用性阻断 | 可并行 | 只允许改 `web/src/api/tasks.ts` 和 `web/src/api/contracts/tasks.ts`。 |
| 2 | API 文档重构 | 用户明确不满意，且视觉问题严重 | 可并行 | 使用 `ApiDocsPage.tsx` 和新建组件级 `ApiDocsPage.css`，避免直接争抢全局 `styles.css`。 |
| 3 | 画布 1080P/@/Ctrl+V 任务设计与最小补丁 | 用户连续多次指出画布不符合使用逻辑 | 可并行 | 使用 `NodeWorkflowPage.tsx` 和新建组件级 `NodeWorkflowPage.css`；如必须改全局 CSS，先报告不改。 |
| 4 | 主题黑色残留只读审计 | 主题问题是全站性质，直接并行写 `styles.css` 易冲突 | 可并行 | 只读审计，输出精确选择器和建议，等 UI 写入任务结束再派写入修复。 |
| 5 | 左侧栏导航设计稿 | 用户说下一版可以把 TAB 放左侧栏 | 可并行 | 只写设计文档，不改代码。 |
| 6 | 集成验收代理 | 检查完成代理是否引入冲突 | 等 P0 修复后派 | 只读/测试。 |

## 14. API 文档删减策略追加

用户明确补充：“文档该删的就删了”。API 文档重构不再以保留旧页面全部内容为目标，而是以可用、清爽、足够接入为目标。

保留：
1. 注册/配置上游 Key/生成 Bearer Key 的最短前置步骤。
2. Base URL、认证 Header、核心端点。
3. curl、TypeScript、Python、Go、Java 调用示例。
4. 任务状态、轮询规则、下载规则、错误码。
5. 一键复制给 AI 的完整提示词。
6. GitHub 文档仓库链接。

可删除或折叠：
1. 外部文档同步块的大段 Markdown。
2. 重复解释同一端点的长段落。
3. 过长的参数散文说明，改为紧凑表格/列表。
4. 让页面变成代码堆的超长展开区。
5. 与当前调用无关的营销或历史说明。

## 15. 新一轮子代理派发记录

| 子代理 | 任务编号 | 任务 | 权限 | 状态 |
| --- | --- | --- | --- | --- |
| Tesla | L1/L2/L3 | 提示词库缓存预热、本地图片缓存、加载体验 | write-owned | 运行中 |
| Volta | P0-1 | 修复/验证 Vite 编译崩溃 `tasks.ts` | write-owned narrow | 运行中 |
| Godel | A1-A5 | API 文档重构和删减 | write-owned | 运行中 |
| Locke | C1/C2 | 画布 Ctrl+V、@ 写入输入框、删预设、1080P 一屏布局 | write-owned | 运行中 |
| Archimedes | T1-T5 | 多主题黑色残留和遮罩问题只读审计 | read-only | 运行中 |
| Mencius | C2-4 | 左侧栏导航/信息架构设计稿 | read-only | 运行中 |

当前策略：等待 P0 修复优先回收；API 文档和画布分别限定在组件文件/组件级 CSS；主题先只读审计，避免并发争抢 `web/src/styles.css`。

## 16. P0 崩溃验收记录

| 时间 | 任务 | 结果 | 证据 | 后续 |
| --- | --- | --- | --- | --- |
| 2026-06-27 | P0-1 Vite 编译崩溃 | 编译层已恢复 | Volta 运行 `cmd /d /c "pushd Z:\github\lyra-image-workbench\web && npx tsc --noEmit --pretty false --incremental false"`，退出码 0；`web/src/api/tasks.ts:75` 当前为 `block.split('\\n')` | 待浏览器刷新确认 overlay 是否消失；若仍出现，优先清 Vite HMR/刷新页面，不作为源码语法错误处理。 |

## 17. 左侧栏设计稿回收记录

Mencius 已完成只读设计稿，未改代码。

采用建议：
1. 桌面左侧 `64-72px` 紧凑图标栏，悬停 tooltip；第一阶段不做可拖拽/可 pin 展开栏。
2. 主栏顺序：创作画布、快捷生成、提示词助手、提示词库、广场、结果。
3. 底部次级：我的、API 文档、设置；后台仅管理员可见，单独 admin 分组。
4. 1080P 目标：去掉顶部全局 tab 高度，创作页内部工具条控制在 `40-48px`，页面本身不滚动，只有局部区域滚动。
5. 手机端：底部 5 项为创作、快捷、助手、结果、更多；更多内放提示词库、广场、我的、API 文档、设置、后台。

状态：已出设计，待用户确认后再实施；本轮不直接改代码。

## 18. 提示词库缓存任务回收记录

Tesla 已完成 L1/L2/L3，已关闭。

完成内容：
1. 后端新增 prompt library 内存缓存和默认本地缓存启动预热。
2. `List` 优先返回缓存，陈旧内容和图片后台刷新。
3. 图片本地化从请求路径移到后台刷新路径。
4. 前端新增 memory/localStorage 缓存，提示词库打开时可先渲染缓存再网络刷新。
5. 新增 stale cache 立即返回、后台继续刷新回归测试。

改动路径：
- `internal/promptlibrary/service.go`
- `internal/promptlibrary/image_cache.go`
- `internal/promptlibrary/types.go`
- `internal/promptlibrary/service_cache_test.go`
- `internal/api/prompt_library.go`
- `web/src/api/promptLibrary.ts`
- `web/src/components/PromptLibraryPage.tsx`

自验：`go test ./internal/promptlibrary`、`go test ./internal/api`、`npm run build`、`git diff --check` 已通过。

风险：完全冷启动且没有任何后端/浏览器缓存时，第一次仍需要 GitHub 拉取。

状态：待主控或验收代理做只读复核。

## 19. 集成看守回收记录

Fermat 已完成只读集成看守，已关闭。

关键发现：
1. 当前无 staged changes，但 dirty tree 很大，包含历史和本轮多代理改动。
2. `web/src/styles.css` 是最大冲突面，已有约 4096 行 diff，并同时承载 API 文档、画布、主题、设置、任务详情、提示词助手样式。
3. API Docs 和 Canvas 都存在写入全局 `styles.css` 的风险，不符合“优先组件级 CSS 隔离”的策略。
4. Tesla 提示词库缓存未见越界到画布/API 文档。
5. 画布旧预设按钮和过细角色当前仍可被搜索到：`产品主图/海报改写/参考图改写`、`服饰/地板材质/背景/构图/光线`，C1-3/C1-4 未满足。

主控动作：
1. 提醒 Godel/Locke 收敛 CSS，不再扩大全局 `styles.css`。
2. Locke 若未删除旧预设和过细角色，不通过验收。
3. Archimedes 作为主题审计只接受只读报告，不允许写入主题文件。
4. 后续需单独派 CSS cascade 验收代理。

## 20. 主题审计回收记录

Archimedes 已完成只读主题审计，未改文件，已关闭。

核心根因：不是绿色主题 token 缺失，而是 `.gallery-shell` 在子树里重新声明了黑色系 token（`--primary: #111827`、`--text: #0f172a` 等），导致 Workbench、Profile、API docs、NodeWorkflow 中的 `var(--primary)` 被解析成黑色。

P1 修复点：
1. `.gallery-shell` 局部 token shadowing：删除局部黑色 token，或限定到 `:root:not([data-theme]) .gallery-shell` fallback。
2. `.topup-option-list button.active`：不要使用 primary 黑底，改为 selected tokens 或 choice active tokens。
3. `--code-bg`：浅色/绿色主题不要混入 `#020617 26%`，改为更浅的 `surface-soft + primary 6%`；暗色主题保留深色代码区。
4. `:root[data-theme] .api-docs-page pre` border 改为 `var(--code-border)`。

P2 修复点：
1. `.queue-source-badge` 不应 `background: var(--text)`。
2. `.gallery-shell .brand-mark` 和生成按钮硬编码 `#111827` 改为 token。
3. debug/developer/2FA/node-flow 代码块逐步统一吃 `--code-bg/--code-text/--code-border`。

主题保留建议：默认保留 `white-blue`、`green`、`black`、`black-purple`、`white`；`blue/pink/white-purple` 可后续考虑隐藏或减少维护面。

状态：待 Theme Fixer 小范围写入修复。

## 21. 窄范围主题修复派发

新增子代理 Planck：Narrow Theme Token Fixer。

权限边界：只可编辑 `web/src/styles.css`，且只限以下点：
1. `.gallery-shell` token shadowing。
2. `--code-bg/--code-border/--code-text`。
3. `.topup-option-list button.active`。
4. `.queue-source-badge`。
5. `.gallery-shell .brand-mark` / generate submit 硬编码黑色。

禁止：不改 API 文档组件、不改画布组件、不改 `themes.ts`、不做大范围主题重构。

## 22. API 文档重构回收记录

Godel 已完成 A1-A5，已关闭。

完成内容：
1. API 文档重构为“SDK 接入控制台”。
2. 左侧：3 步接入、Bearer Key、端点/错误码。
3. 右侧：语言示例、完整 AI 提示词。
4. 删除外部文档同步长文，只保留 GitHub 文档仓库链接。
5. 保留 curl / TypeScript / Python / Go / Java 示例与复制动作。
6. 新增组件级 `ApiDocsPage.css`，未继续扩大 `web/src/styles.css`。

改动路径：
- `web/src/components/ApiDocsPage.tsx`
- `web/src/components/ApiDocsPage.css`

自验：`cmd /d /c "pushd Z:\github\lyra-image-workbench\web && npx tsc --noEmit --pretty false --incremental false"` 通过。

状态：待 API 文档验收代理/人工视觉验收。

## 23. 提示词库缓存验收回收记录

Lagrange 已完成只读验收，已关闭。

结论：
1. 缓存目标：通过，带 caveat。
2. 具备已有本地缓存时秒开、过期缓存后台刷新、GitHub 图片本地化不阻塞请求。
3. 冷启动且无后端/浏览器缓存时，第一次仍需远端拉取；若要求冷启动也主动后台 fetch，需要后续补逻辑和测试。
4. 范围风险：验收代理看到当前 dirty tree 中有画布/API 文档/主题改动，但这些路径不在 Tesla 自报改动范围内；暂记为“工作区混杂风险”，不直接认定 Tesla 越界。
5. 缺口：前端 localStorage 缓存缺单元测试，目前靠 TypeScript 与人工页面验收。

已跑命令：
- `go test ./internal/promptlibrary` 通过。
- `go test ./internal/api` 通过。
- `tsc --noEmit --pretty false -p web/tsconfig.json` 通过。

状态：提示词库缓存功能可进入集成验收；后续人工确认二次打开速度。

## 24. 创作画布交互/布局回收记录

Locke 已完成 C1/C2，已关闭。

完成内容：
1. 画布可聚焦，点击记录粘贴点，`Ctrl+V` 可把剪贴板图片上传并放到点击附近。
2. `@ 作为参考图` 会把 `@1 参考「图片名」：` 追加到底部提示词输入框并聚焦，同时标记为图生图参考。
3. 删除旧三枚预设按钮：产品主图、海报改写、参考图改写。
4. 删除过细角色：服饰、地板/材质、背景、构图、光线；只保留 `reference / subject / style`。
5. 高级参数并入同一行紧凑参数区。
6. 新增组件级 `NodeWorkflowPage.css` 压缩 1080P 首屏布局。

改动路径：
- `web/src/components/NodeWorkflowPage.tsx`
- `web/src/components/NodeWorkflowPage.css`

自验：
- `cmd /d /c "pushd Z:\github\lyra-image-workbench\web && npx tsc --noEmit --pretty false --incremental false"` 通过。
- owned 文件内搜索 `产品主图|海报改写|参考图改写|outfit|material|background|composition|lighting|creative-composer-presets|creative-canvas-advanced|gallery-shell` 无匹配。

风险：全局 `styles.css` 仍有旧 `.creative-composer-presets` / `.creative-canvas-advanced` 死样式，组件已不引用；后续由 CSS owner 清理。

状态：待画布验收代理/人工验收 1080P 一屏、Ctrl+V、@ 输入框。

## 25. API 文档验收记录

Peirce 已完成只读验收，已关闭。

结论：源码验收通过，待浏览器视觉确认。

证据：
1. `ApiDocsPage.tsx` 已导入 `./ApiDocsPage.css`。
2. 页面结构已改为“SDK 接入控制台”：3 步接入、Bearer Key、端点/错误码、语言示例、AI 提示词。
3. 外部同步长文已删除，只保留必要注册链接和 GitHub 文档仓库链接。
4. 代码块使用组件级明亮样式 `.api-console-code-block`，不再像暗色遮罩。
5. curl / TypeScript / Python / Go / Java 示例仍可达且可复制。
6. 给 AI 的提示词包含 Base URL、Authorization Bearer、创建任务、轮询、下载、错误码、`LYRA_API_KEY`。

风险：未做浏览器截图或 live API 调用。

## 26. 主题窄修回收记录

Planck 已完成窄范围主题修复，已关闭。

完成内容：
1. 只修改 `web/src/styles.css`。
2. `.gallery-shell` 在 `:root[data-theme]` 下重新继承 root theme tokens，避免把绿色主题覆盖回黑色。
3. 非暗色主题代码块改用浅色/tinted code tokens；暗色主题保留深色代码块。
4. 充值档位 active 不再使用 primary/black token，改为 selected-token 语义。
5. 修复 `.queue-source-badge` 和若干 `.gallery-shell` 下硬编码黑色的按钮/brand mark。

自验：`cmd /d /c "pushd Z:\github\lyra-image-workbench\web && npx tsc --noEmit --pretty false --incremental false"` 通过。

状态：待浏览器人工验收绿色主题、API 文档代码块、充值档位 active。

## 27. 画布验收与当前验证记录

Banach 已完成画布只读验收，已关闭。

结论：源码验收通过，待浏览器人工验收。

证据：
1. `ReferenceRole` 只剩 `reference | subject | style`，渲染只显示“参考 / 主体 / 风格”。
2. 目标文件搜索不到 `产品主图 / 海报改写 / 参考图改写`，也搜不到 `服饰 / 地板材质 / 背景 / 构图 / 光线`。
3. stage 有 `tabIndex={0}`、点击记录 paste point、clipboard image files 通过 `addFilesToCanvas`。
4. `@` action 会构建 prompt line 并 `setDraftPrompt`，不是只改图片 badge。
5. `质量/格式/数量/并发` 已在 `creative-composer-tools` 内联渲染，不再有目标文件 `creative-canvas-advanced`。
6. `NodeWorkflowPage.css` 有 100svh/clamp/dense grid，具备 1080P 首屏布局意图。

人工验收仍需确认：
1. 1920x1080 下真实数据是否没有页面级长滚动。
2. 点击空画布后 Ctrl+V 是否符合直觉。
3. 粘贴是 stage focus 行为，不是全局粘贴。

当前主控已跑验证：
1. `cmd /d /c "pushd Z:\github\lyra-image-workbench\web && npx tsc --noEmit --pretty false --incremental false"` 通过。
2. `go test ./internal/promptlibrary ./internal/api ./internal/settings` 通过。
3. `git diff --check` 失败，仅剩两个 whitespace 问题：`web/src/components/AdminPage.tsx` 末尾空行，`web/src/components/PromptAssistantModal.tsx` 末尾空行；其余为 CRLF/LF warning。

## 28. Whitespace 修复回收记录

Wegener 已完成并关闭。

修复文件：
- `web/src/components/AdminPage.tsx`
- `web/src/components/PromptAssistantModal.tsx`

修复内容：仅移除文件末尾多余空行。

验证：`git -c safe.directory=//fnos1/16T/github/lyra-image-workbench diff --check` 退出码 0；仍有 CRLF/LF warning，但无 whitespace error。

## 29. 本轮最终验证结果

最终验证已完成：

| 命令 | 结果 |
| --- | --- |
| `cmd /d /c "pushd Z:\github\lyra-image-workbench\web && npx tsc --noEmit --pretty false --incremental false"` | 通过 |
| `cmd /d /c "pushd Z:\github\lyra-image-workbench\web && npm run build -- --mode production"` | 通过，Vite transformed 92 modules |
| `go test ./internal/promptlibrary ./internal/api ./internal/settings` | 通过 |
| `git -c safe.directory=//fnos1/16T/github/lyra-image-workbench diff --check` | 通过；仅 CRLF/LF warning，无 whitespace error |

本轮完成状态：
1. P0 Vite 崩溃：编译层已恢复。
2. API 文档：已重构并通过源码验收。
3. 创作画布：已完成 Ctrl+V、@ 写入输入框、删预设/过细角色、高级参数同排、组件级 CSS，并通过源码验收。
4. 绿色主题黑色残留：已修复 `.gallery-shell` token shadowing、代码块 token、充值 active 等核心问题，待浏览器视觉确认。
5. 提示词库缓存：已完成缓存/预热/stale-while-refresh/图片后台本地化，相关测试通过。
6. 后台重复导航：AdminPage 拆分并删除底部重复跳转按钮。
7. 模块拆分：API contracts、AdminPage、PromptAssistant、Workbench hook、settings update 已完成一轮拆分。
8. 左侧栏：已出只读设计稿，未实施，待用户确认。

仍需人工浏览器验收：
1. 刷新页面确认 Vite overlay 消失。
2. 绿色主题下 Profile 充值档位、API 文档、画布 active 不再黑。
3. API 文档视觉是否符合“清爽调用页”。
4. 1920x1080 创作画布是否一屏可用。
5. 点击画布后 Ctrl+V 粘贴图片是否符合预期。
6. 右键/按钮 `@` 是否准确写入底部输入框。
7. 提示词库二次打开速度是否接近秒开。

## 30. 漏项复核与用户最新反馈（2026-06-27）

本节按用户要求重新对照历史需求、当前源码和专项子代理结果整理。结论口径从严：设计稿、台账记录、源码自查通过，不等于已经交付；没有浏览器视觉验收或端到端跑通的，只能算“部分完成/待验收”。

### 30.1 明确没做完或没有落地

| 编号 | 用户要求 | 当前确认状态 | 证据/说明 | 下一步 |
| --- | --- | --- | --- | --- |
| GAP-1 | 桌面端把“创作画布/快捷生成/提示词助手/提示词库/广场/结果/我的/API 文档/设置/后台”从顶部 TAB 改成左侧栏 | 未落地 | `WorkbenchPage.tsx` 仍渲染 `workflow-tabs desktop-tabs`；子代理确认没有 Workbench 级左侧栏 | 派唯一前端写入者实现 WorkbenchSidebar，移动端保留底部导航。 |
| GAP-2 | 创作画布底部按钮、参数区不要变形错位 | 未达标 | 用户截图显示“参考图生成/模型”等控件重叠；CSS 仍用 `auto-fit minmax(92px, 1fr)` 搭配 `creative-segmented min-width:168px`，容易溢出 | 先修 composer 网格，改稳定列/分行布局，再做 1920x1080 截图验收。 |
| GAP-3 | 旋转按钮、缩放手柄位置正确 | 未达标 | 手柄放在被 `rotate(...)` 的 `.creative-canvas-item` 内部，跟图片一起旋转；resize 也未做旋转坐标换算 | 拆位移层/旋转内容层/独立控制浮层；只在选中时显示手柄。 |
| GAP-4 | 创作画布 1080P 一屏可用，不用下翻 | 未完成验收 | 有紧凑 CSS，但用户仍反馈页面需要翻、按钮挤压；没有浏览器截图验收 | 以 1920x1080 为硬验收，画布、预览、输入、参数、生成按钮必须同屏。 |
| GAP-5 | GIF 独立动图模式：上传图片 + 描述动效 + 模板生成 GIF | 未实现 | 子代理确认没有 `/gif` 页面、GIF API、动效模板、任务模型、FFmpeg 合成链路；现有 GIF 只是图片 MIME 支持 | 新增独立 GIF/动图模块设计与实现，不复用视频命名和路由。 |
| GAP-6 | 提示词库启动时预热、点击秒开、GitHub 图片本地化 | 部分完成但体验未达标 | 后端有 JSON 硬盘缓存和图片缓存，前端有 localStorage；但 `StartWarmCache` 只加载已有硬盘缓存，没有缓存时不会启动主动拉远端，冷启动仍慢 | 增加启动后台主动同步；页面已有缓存时秒出；无缓存时后台预拉并显示明确状态。 |
| GAP-7 | API 文档页面彻底重构到满意 | 部分完成但未通过用户验收 | 已重构过，但用户明确说“不满意”；需要重新按两栏、完整 AI 提示词、Key 遮罩复制、无暗色遮罩验收 | 重新设计 API docs，而不是继续补丁式堆代码块。 |
| GAP-8 | 多主题特别是绿色主题不出现黑色 active/按钮覆盖文字 | 部分完成但未通过用户验收 | 修过 `.gallery-shell` token shadowing，但用户截图仍看到绿色主题黑色按钮和文字被盖住 | 做全站 token 扫描 + 浏览器主题截图验收。 |
| GAP-9 | 管理员初始化设置页：站点名、管理员账号、基础设置、安装令牌提醒 | 部分完成/待验收 | 有初始化路径和令牌提醒，但存在“先注册普通用户再初始化后台”卡死风险 | 修首个管理员/初始化兜底流程，重新验收首屏安装向导。 |
| GAP-10 | 用户系统完整闭环：邮箱、头像、用户名/显示名、密码哈希、用户主页、作品、流水图表/热力图 | 半闭环/待验收 | 注册、登录、资料、头像、2FA、流水字段有实现；图表/热力图和全流程未验收 | 以注册、登录、主页、流水、作品、头像修改为验收脚本。 |
| GAP-11 | 额度/充值/易支付/邀请奖励闭环 | 半闭环/待验收 | 易支付、邀请奖励基本接上；普通任务失败/取消/部分失败未自动退回额度，支付后前端缺自动轮询 | 修退费和支付状态刷新，跑订单创建、回调模拟、邀请奖励、失败态。 |
| GAP-12 | 管理后台更立体：用户列表、用户详情、加次数必填理由、管理员操作记录、额度流水 | 部分完成/待验收 | 后台已有 tab 和加次数，但角色变更没有单独操作审计，也未阻止取消最后一个用户管理员 | 后台重新按运营工作台验收，补操作审计和最后管理员保护。 |
| GAP-13 | 每日投稿、点赞、日榜、奖励预留 | 部分完成/待验收 | 台账有每日榜/点赞记录，但没有页面和 API 端到端验收 | 单独跑当天投稿、点赞、日榜刷新、重复点赞限制。 |
| GAP-14 | 广场提交云端副本、比例/质量/图片保存、未提交 30 天清理、提交后保护 | 部分完成/待验收 | 清理器和广场副本有记录，但用户要求的是完整真实链路 | 验证 submit square、副本文件、清理保护、图片不变形。 |
| GAP-15 | 提示词助手改成 ChatGPT/Claude 式灵感对话体验 | 部分完成 | 已拆页和拆模块，但还不像最终对话式体验 | 重新设计“从一句想法追问完善”的主体验，保留历史/图片还原。 |
| GAP-16 | 同页面重复功能清理 | 部分完成/需复扫 | 清过部分旧组件和后台重复按钮，但用户仍截图指出重复功能 | 再扫页面级重复入口，先列决策表，再删。 |
| GAP-17 | 移动端 UI 全链路 | 部分完成/待验收 | 做过移动端样式，但没有完整视口截图验收 | 覆盖 360/390/430/768/980 视口。 |
| GAP-18 | 邮件发件能力 | 配置闭环，发信未实现 | SMTP 只保存配置和脱敏回传；没有测试邮件、邮箱验证或通知触发 | 补真实发信服务、测试邮件、邮箱验证/通知触发。 |

### 30.2 做了但不能再直接算完成的事项

| 事项 | 当前更准确口径 |
| --- | --- |
| Bearer API / 任务轮询 / SDK 文档 | 后端和文档有实现，但站内 API 文档仍需用户验收；前端“走 API 请求并同步显示任务 ID、状态、历史”的体验也需单独验收。 |
| 视频删除 | 生产入口基本删除；当前 GIF 只是图片格式支持，不是动图模式。历史文档仍有记录，不应混淆为现役功能。 |
| 任务队列/结果合并 | 旧组件删过一批，但仍要页面复扫确认没有重复操作和重复入口。 |
| 主题系统 | token 方向已做过一轮，但真实主题截图不通过，所以必须返工验收。 |
| 创作画布高级能力 | 代码里已有拖拽、Ctrl+V、缩放、旋转、连线、右键 @ 的雏形，但手柄/布局/语义绑定/提交快照还没完整验收。 |
| 用户/后台/支付/邀请 | 主链路大体接上，但首个普通注册不是管理员、失败任务不自动退费、支付后不自动轮询、SMTP 不发信，不能算闭环。 |

### 30.3 最新 P0/P1 排队

1. P0：修复画布底部按钮/参数区错位，修复旋转/缩放手柄坐标结构。
2. P0：实现桌面左侧栏导航，替代顶部横向 TAB。
3. P0：提示词库冷启动也要后台主动同步，页面已有缓存时必须秒开。
4. P1：修复用户初始化/首个管理员卡死风险。
5. P1：修复任务失败/取消/部分失败自动退回额度。
6. P1：重新审 API 文档页，不再以“代码已经重构过”为完成口径。
7. P1：绿色主题和所有主题的黑色残留全站视觉验收。
8. P1：GIF 独立模块进入新任务，不再误称已有。
9. P1：用户/后台/充值/邀请/广场/清理策略做端到端验收表。

## 31. P0 执行回收记录（2026-06-27）

本节记录用户要求“执行吧”后的本轮实际落地内容。口径：代码已改并通过命令验证；浏览器视觉因当前环境无法连接 in-app browser，仍需人工页面验收。

### 31.1 已落地

| 编号 | 任务 | 状态 | 改动文件 | 说明 |
| --- | --- | --- | --- | --- |
| FIX-1 | 桌面左侧栏导航替代顶部 TAB | 已落地，待视觉验收 | `web/src/components/WorkbenchPage.tsx`、`web/src/components/WorkbenchSidebar.css` | 桌面端新增左侧 sidebar，入口包含创作画布、快捷生成、提示词助手、提示词库、广场、结果、我的、API 文档、设置、后台；移动端保留底部导航和更多面板。侧栏 active 已改用主题 token，避免绿色主题继续黑底。 |
| FIX-2 | 创作画布旋转/缩放手柄错位 | 已落地，待视觉验收 | `web/src/components/NodeWorkflowPage.tsx`、`web/src/components/NodeWorkflowPage.css` | 画布对象改为外层位移、内层旋转、控制浮层；手柄只在选中时显示；控制浮层跟随图片角度，按钮自身反向旋转保持正向；resize 使用旋转反向矩阵把屏幕位移换算成本地宽高。 |
| FIX-3 | 创作画布底部 composer 控件重叠 | 已落地，待视觉验收 | `web/src/components/NodeWorkflowPage.css` | 组件级 CSS 显式重置旧全局 sticky：`position: static; bottom: auto;`；参数区改为稳定 grid，避免 `参考图生成` segmented 溢出覆盖“模型/比例”等控件。 |
| FIX-4 | 提示词库冷启动预热 | 已落地并通过后端测试 | `internal/promptlibrary/service.go`、`internal/promptlibrary/service_cache_test.go` | `StartWarmCache` 在没有硬盘 JSON 缓存时也会后台预拉默认语言库；`List` 可复用同一个 in-flight 同步，避免重复打 GitHub；拉取成功后继续保存本地 JSON 并触发图片后台缓存。 |

### 31.2 验证结果

| 命令 | 结果 |
| --- | --- |
| `cmd /d /c "pushd Z:\github\lyra-image-workbench\web && npx tsc --noEmit --pretty false --incremental false"` | 通过 |
| `cmd /d /c "pushd Z:\github\lyra-image-workbench\web && npm run build -- --mode production"` | 通过，Vite transformed 93 modules |
| `go test ./internal/promptlibrary` | 通过 |
| `git -c safe.directory=//fnos1/16T/github/lyra-image-workbench diff --check` | 通过；仅剩既有 CRLF/LF warning，无 whitespace error |

### 31.3 未完成/待人工验收

1. 浏览器视觉验收未完成：本轮尝试连接 in-app browser 被 Windows 映射盘/会话环境阻断。
2. 需要在页面刷新后确认：左侧栏是否显示并替代顶部 TAB；移动端底部导航是否仍正常；绿色主题侧栏 active 是否不再黑底。
3. 需要在创作画布确认：底部参数区不再重叠；旋转到 30-45 度后手柄位置正确；缩放方向符合旋转后的视觉方向。
4. API 文档重构、全站主题黑色残留、用户/后台/支付/邀请闭环、GIF 独立模式仍在 GAP/P1 队列，尚未在本轮实现。

## 32. Agent 多轮创作模块源码分析与接入决策（2026-06-27）

用户最新决策：Agent 单独一个模块，不影响创作画布。

本轮已派 6 个只读子代理并回收：
1. 参考项目 Agent UI 分析：确认其核心是会话、轮次树、当前分支路径渲染、助手回复块化。
2. 参考项目 Agent API 分析：确认多轮循环主要在 store，`agentApi.ts` 负责 Responses API/tools/SSE 封装。
3. 图片引用体系分析：建议 Lyra 拆成 mentionCodec、agentReferenceRegistry、agentReferenceApiAdapter，不硬搬不可见字符方案。
4. Lyra 前端边界分析：Agent 应新增独立 tab/独立组件，不改 `NodeWorkflowPage`。
5. Lyra 后端边界分析：新增 `internal/agents` 和 `/api/agents`，图片生成继续复用现有 jobs manager。
6. 产品验收分析：P0 只做目标 -> 计划 -> 确认 -> 创建现有任务 -> 去结果页验收，不复刻画布/提示词助手/结果页。

已新增设计文档：`docs/AGENT_MULTITURN_MODULE_DESIGN.md`。

关键决策：
1. Agent 是独立一级创作入口。
2. 创作画布仍是主入口，Agent 不写画布状态。
3. Agent 第一阶段做编排闭环，不做完整分支树、web search、partial image、mask。
4. Agent 后端必须做 per-space store，并纳入用户认证。
5. 图片任务复用现有 background task/额度/轮询链路。
6. P0 必须限制单轮最大图片数和 continue 次数，避免循环扣费。

同步小修：
1. 创作画布参考图素材条新增删除按钮，并接入父组件已有 `handleDeleteUpload`。
2. 创作画布输入框不再自带示例提示词。
3. 右键 `@ 作为参考图` 只写入 `@1`，不再自动追加“参考某某”的提示词文案。
4. 图生图提交只使用画布中明确标记的引用图，不再在引用为空时自动回退到全部已上传素材。

验证：
1. `cmd /d /c "pushd Z:\github\lyra-image-workbench\web && npx tsc --noEmit --pretty false --incremental false"` 通过。
2. `git -c safe.directory=//fnos1/16T/github/lyra-image-workbench diff --check -- web/src/components/NodeWorkflowPage.tsx web/src/components/NodeWorkflowPage.css web/src/components/WorkbenchPage.tsx` 通过，仅 CRLF/LF warning。

待验收：
1. 浏览器手测删除参考图按钮是否不被裁切、点击是否删除后端素材和画布对象。
2. 图生图没有明确引用图时是否阻止提交。
3. Agent 模块尚未实现；当前只完成源码分析和接入设计。

### 32.1 Agent 引用 token 决策补充

最后一个只读子代理建议为 Agent Workspace 增加真实 mention token / 引用快照模型。主控采纳为 Agent 模块内部 P0 设计，但不把创作画布重构成 Agent 的前置条件。

更新到 `docs/AGENT_MULTITURN_MODULE_DESIGN.md` 第 9 节：
1. Agent 内部 `@图N` 必须是结构化 token，手打文本不算绑定。
2. Agent round 需要保存引用快照，记录 upload/task-result/agent-round-output 来源。
3. 被删除的图片要降级为 removed ref，不允许崩溃或错指。
4. Agent 必须继续复用现有上传、任务、额度、空间隔离链路，不直接调上游。

画布当前 `@1` 仍作为轻量文本提示处理；是否和 Agent 共用 mentionCodec，放到后续统一体验阶段决策。
