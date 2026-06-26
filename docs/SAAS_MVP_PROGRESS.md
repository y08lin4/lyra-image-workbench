# Lyra Image Workbench SaaS MVP 当前进度

> 历史进度快照：本文保留多代理调度过程和中间态记录，不代表当前 dev 最终功能面；当前第一阶段已收口为图片生成商业闭环，视频和动态合成工作流均不作为现役功能。

更新时间：2026-06-26  
目标分支：`dev`  
文档性质：当前调度进度记录，不是最终验收报告。本文只描述 SaaS 化实现进展、剩余任务和风险，不声称未完成或未验收的功能已经完成。

## 1. 用户最初需求清单

当前 SaaS 化目标是把单机生图工作台推进为轻量单节点生图站点 MVP，完整需求如下：

1. 用户可以注册、登录、配置个人资料、查看自己的作品和额度流水。
2. 管理员可以在前端看到管理入口，配置上游 Key、系统参数、邮箱、支付和邀请奖励。
3. 用户通过充值获得生图次数；管理员也可以手动给用户增加次数，必须填写原因，并进入流水。
4. 接入易支付，参考 `QuantumNous/new-api` 的订单创建、回调验签、成功入账思路。
5. 邀请系统只在被邀请用户首次成功充值后奖励邀请人次数。
6. 生成结果可以提交到广场，提交后永久保留；未提交的普通结果按 30 天保留策略提示和清理。
7. 广场支持投稿、点赞、每日 00:00-24:00 榜单、用户主页作品。
8. 移除视频/MiniMax 以及动态合成相关功能，当前阶段只保留图片生成闭环。
9. 合并任务队列和结果视图，降低重复导航。
10. 提示词助手独立成一级页面，并保留四栏：文生图、图片还原、灵感模式、历史。
11. 新增静态 API 文档页面，提供 curl、Python、Go、Java、JavaScript 示例和一键复制给 AI 的接入提示词。
12. 重构暗色模式，并提供多个主题：白蓝、黑紫、白紫、绿色、粉色、蓝色等。

暂缓范围仍然是：复杂限流、套餐体系、优惠券、发票、分布式队列、多节点一致性、数据库迁移、真实邮件验证闭环，以及把未充分测试的新功能合并到 `master`。

## 2. 已完成

### 2.1 调度与任务拆分

- 已完成 `docs/AGENT_TASK_BREAKDOWN.md`，包含总体目标、接口契约、Wave 1/2/3 子代理拆分、共享文件风险、最终验证清单。
- 当前文档基于该拆分文档和已落地代码面整理，不替代后续集成验收。

### 2.2 A1 用户、额度、邀请基础

已完成用户/额度/邀请基础能力的后端实现面：

- 用户模型扩展：`displayName`、`email`、`avatarUrl`、`isAdmin`、`creditsBalance`、`referralCode`、邀请关系、邀请奖励时间。
- 注册支持邮箱和邀请码；登录支持用户名或邮箱。
- 首个注册用户自动成为管理员。
- 用户资料读取/更新、邀请码生成、用户额度流水查询已接入用户 API。
- 管理员加次数、原因必填、用户流水查询、管理员角色设置已接入管理 API。
- 购买入账使用 `sourceId` 做幂等基础；邀请奖励只在首次成功充值后触发一次。
- 已有相关单元测试覆盖注册兼容、邮箱登录、管理员首用户、管理员加次数、重复订单、邀请奖励。

仍需注意：A1 已完成的是基础能力和测试面，仍需与 billing 回调、前端管理页、全量 `go test ./...` 做最终集成验收。

### 2.3 A2 billing 核心包

已完成 billing 核心包和易支付基础逻辑：

- 新增 `internal/billing/store.go`，支持充值订单持久化、订单号生成、按用户列出订单、订单成功/失败状态流转。
- 新增 `internal/billing/epay.go`，支持易支付参数排序签名、支付 URL 生成、回调签名和金额/订单状态校验。
- 新增 `internal/billing/*_test.go`，覆盖订单创建、成功幂等、签名、坏签名、金额不匹配等核心行为。
- 设置结构和管理配置侧已出现支付相关字段的实现面。

未完成边界：`internal/api/billing.go` 当前尚未出现，`/api/billing/*` 路由接线仍需单独派发；A2 不能视为“支付全链路已完成”。

### 2.4 A5 前端账号/API 文档骨架

已完成前端账号和 API 文档骨架：

- `web/src/types.ts` 已扩展用户、额度流水、billing 类型。
- `web/src/api/users.ts` 已支持注册邮箱/邀请码、资料、流水、邀请码、我的广场作品接口包装。
- `web/src/api/admin.ts` 已加入管理员用户、加次数、流水、角色设置等 wrapper。
- `web/src/components/SpaceLogin.tsx` 已支持注册邮箱和邀请码。
- `web/src/components/ProfilePage.tsx` 已创建，包含资料、余额、邀请码、额度流水、我的投稿骨架。
- `web/src/components/ApiDocsPage.tsx` 已创建，包含 curl、Python、Go、Java、JavaScript 示例和复制给 AI 的提示词。
- `web/src/components/WorkbenchPage.tsx` 已接入“我的”和“API 文档”一级入口。

仍需注意：支付管理页、管理员用户流水 UI 还没有完成；`web/src/api/billing.ts` 已存在 wrapper，但后端路由还未接线。

### 2.5 A6 多主题、暗色模式、提示词助手一级页

已完成多主题与提示词助手入口的主要实现面：

- 新增 `web/src/lib/themes.ts`，支持 `white-blue`、`black-purple`、`white-purple`、`green`、`pink`、`blue`。
- `web/src/main.tsx` 使用 `localStorage` 保存主题，并通过 `data-theme` 应用。
- `web/src/components/ThemeToggle.tsx` 从二选一切换改为主题选择。
- `web/src/components/PromptAssistantModal.tsx` 支持嵌入式一级页面展示，并为 tab 补充 `aria-selected`。
- `web/src/components/WorkbenchPage.tsx` 已接入“提示词助手”一级入口。
- `web/src/styles.css` 已新增多主题变量和多处主题覆盖。

仍需注意：`styles.css` 是共享高冲突文件，后续 B1/B2/B3/B4/B5/C2 继续改动前必须统一样式所有权。

### 2.6 A4 MiniMax 删除计划

已完成 MiniMax/视频删除计划和边界定义：

- 后端删除目标和残留扫描命令已在 `docs/AGENT_TASK_BREAKDOWN.md` 写清；本文为历史调度口径，最终功能面以后续集成结果为准。
- 当前工作区已有后端 MiniMax 文件删除和 README/测试/设置相关改动痕迹，说明 A4-impl 已进入实现阶段。

历史状态说明：该段记录的是中途状态，最终状态见第 8 节；当前前端视频入口和相关动态合成功能已从现役功能面移除。

### 2.7 I2 支付接入契约

已完成支付接入契约层：

- 易支付配置项、订单创建、通知回调、订单列表、签名规则、回调返回值、金额和状态校验、重复回调幂等要求已明确。
- 前端 `web/src/api/billing.ts` 已按 `/api/billing/topup/options`、`/api/billing/epay/orders`、`/api/billing/topups` 准备 wrapper。
- billing 核心包已覆盖订单和签名逻辑。

未完成边界：后端 billing HTTP handler、router 接线、用户入账联调、支付管理页和浏览器验收仍未完成。

## 3. 正在进行

1. A3 广场后端：`internal/promptsquare/store.go` 已出现作者、点赞、每日榜、永久保留、`SubmitFromResult` 等扩展；但 `internal/api/prompt_square.go` 仍主要是旧上传接口，`from-result`、like、daily、mine API 接线需要继续完成并验收。
2. A4-impl 后端视频删除：后端 MiniMax 文件和文档已有删除痕迹；动态合成工作流进入最终收口清理，还需要完成残留扫描、测试和前端协同。
3. FE-nav 导航接线：`WorkbenchPage.tsx` 已接入“提示词助手 / 我的 / API 文档”，但仍保留“视频”和独立“队列”入口；后续要与结果队列合并、前端视频删除一起收口。

## 4. 未开始或待派

1. billing 路由接线：创建 `internal/api/billing.go`，在 `internal/api/router.go` 接入 `/api/billing/topup/options`、`/api/billing/epay/orders`、GET/POST `/api/billing/epay/notify`、`/api/billing/topups`。
2. 前端广场重构：移除旧上传入口，改为广场浏览、每日榜、我的投稿、点赞、排名、元信息展示。
3. 结果队列合并：取消独立队列一级页，在结果页内展示任务列表和当前任务详情，并提供“提交到广场”。
4. 支付管理页：资料页充值入口、订单列表、支付方式选择、管理页易支付配置、邀请奖励配置。
5. 管理员用户流水：管理用户列表展示邮箱/余额/管理员状态/注册时间，加次数原因必填，用户流水详情，设置/取消管理员确认。
6. 前端视频删除：历史待办已完成，当前现役功能面不包含视频入口、相关管理配置或动态合成 API。
7. 全量测试：`go test ./...`、前端 `npm run build`、残留扫描、安全扫描。
8. 启动验收：本地启动后完成注册、充值/回调、广场投稿、点赞榜单、资料页、API 文档、管理页、主题切换。
9. 提交 push：在 `dev` 上整理提交，确认未完成和未测试项后再 push。

## 5. 当前风险

1. 共享文件冲突风险：`internal/api/router.go`、`internal/settings/settings.go`、`web/src/components/WorkbenchPage.tsx`、`web/src/styles.css`、`web/src/types.ts`、`web/src/api/admin.ts` 都是多代理共享面。后续同一文件只能保留一个当前写入负责人，集成修复必须人工合并。
2. PromptSquare 编译和契约风险：store 层已扩展到投稿/点赞/每日榜，但 API handler 和前端仍保留旧上传模型，容易出现类型、路由、字段和编译不一致。
3. 支付幂等风险：billing store 有订单成功幂等基础，A1 也有 `sourceId` 幂等入账基础；真正风险点在 HTTP 回调接线后，必须验证重复通知不会重复加次数，且订单金额、状态、订单号都被校验。
4. 密钥泄露风险：`epayKey`、上游 Key、SMTP 密码、MiniMax 残留 Key、API Key 预览字段都必须审查，普通前端只能拿到 masked/preview 或布尔状态，不能拿明文。
5. 视频/动态合成残留风险：当前第一阶段不保留该工作流，部署侧不再引入额外媒体处理依赖。
6. 导航和信息架构风险：提示词助手、我的、API 文档已新增，视频和队列尚未移除/合并；如果 FE-nav、B2、B5 同时改 `WorkbenchPage.tsx`，很容易互相覆盖。
7. 当前工作区未全量验收：已有大量未提交改动和新增文件，D1 本轮未运行测试或启动服务，所有完成项都应在 C1/C2/C4 阶段复验。

## 6. 下一步派发建议

建议按以下顺序派发，尽量减少共享文件冲突：

1. 派 `integration-manager` 只读锁定共享文件当前状态，输出 `router/settings/WorkbenchPage/styles.css/types/admin.ts` 的冲突地图。
2. 派 billing 路由接线后端代理，写入范围限定为 `internal/api/billing.go`、`internal/api/router.go`，只消费 A1/A2 已有接口。
3. 派 A3 收口代理，优先补 `internal/api/prompt_square.go` 的 `from-result`、like、daily、mine 路由和测试；完成后再派前端广场重构。
4. 派 A4/B5 协同收口：先完成后端 MiniMax 残留扫描，再派唯一前端写入者删除视频 tab、MiniMax UI/API/类型。
5. 派 B2 结果队列合并，独占 `WorkbenchPage.tsx`、`ResultCanvas`、`TaskSidebar`、`TaskQueue` 当前窗口；FE-nav 不要同时改这些文件。
6. 派 B3/B4 支付和管理员 UI，等 billing 路由可编译后再接真实接口。
7. 最后派 C1/C2/C3/C4 做后端集成、前端构建、安全审查和浏览器验收，再由 C6 整理提交。

## 7. 建议验收命令

分支和工作区：

```powershell
git -c safe.directory=//fnos1/16T/github/lyra-image-workbench status --short --branch
```

后端定向测试：

```powershell
go test ./internal/users ./internal/billing ./internal/settings
go test ./internal/promptsquare ./internal/api -run "PromptSquare|Billing|Users|AdminUsers|Credit|Referral"
```

后端全量测试和构建：

```powershell
go test ./...
go build -trimpath -ldflags="-s -w" -o .\bin\lyra-image-workbench.exe .\cmd\local-server
```

前端构建：

```powershell
Push-Location web
npm run build
Pop-Location
```

残留和敏感字段扫描：

```powershell
rg -n -i "minimax|videoQuota|MiniMaxVideo|/api/minimax" .
rg -n -i "epayKey|smtpPassword|upstreamKey|newApiKey|minimaxApiKey" web/src internal/api
rg -n "/api/billing|billing" internal/api internal/billing web/src/api
```

启动验收：

```powershell
go run ./cmd/local-server
```

```powershell
Push-Location web
npm run dev -- --host 127.0.0.1
Pop-Location
```

浏览器手工验收路径：

1. 注册首个用户，确认自动管理员。
2. 管理员配置上游 Key、易支付、SMTP、邀请奖励。
3. 用户生成图片，结果页显示 30 天保留提示。
4. 用户提交结果到广场，确认图片不变形且永久副本可访问。
5. 广场点赞，确认每日榜变化。
6. 用户主页显示投稿作品、余额和流水。
7. 管理员给用户加次数，原因必填且流水可见。
8. 创建易支付订单，模拟或真实回调只入账一次。
9. 切换白蓝、黑紫、白紫、绿色、粉色、蓝色主题。
10. API 文档页面复制提示词和代码示例。
11. 确认 MiniMax/视频入口已移除，动态合成工作流不再作为当前功能面提供。


## 8. 最终调度更新

更新时间：2026-06-26 最终集成后

本节覆盖前文中途状态。最终实现和验证结果如下：

1. 用户系统、邮箱、头像、管理员角色、额度余额、额度流水、管理员加次数原因、购买入账幂等、邀请首充奖励已经落地。
2. 易支付核心包、后端 `/api/billing/*` 路由、回调验签、金额/PID/支付方式校验、重复回调幂等、管理员配置脱敏已经落地。
3. Prompt Square 已支持从结果投稿、永久副本、点赞/取消点赞、每日榜、我的投稿；前端已移除上传入口并重构为浏览/榜单/我的投稿页面。
4. 结果页和任务队列已合并；生成结果图可提交到广场，并展示“未提交 30 天后清理，提交后永久保留”的提示。
5. 前端视频/MiniMax、后端 MiniMax 与动态合成工作流均已从当前功能面移除；README 和部署文档不再要求 FFmpeg。
6. 提示词助手已成为一级页面，并保留文生图、图片还原、灵感模式、历史四栏。
7. API 文档页已加入 curl、Python、Go、Java、JavaScript 示例，以及复制给 AI 的接入提示词。
8. 多主题已落地：白蓝、黑紫、白紫、绿色、粉色、蓝色；暗色模式对比度和移动端导航兜底已做集成修复。
9. 管理员页已包含易支付配置、用户列表、余额、加次数、流水查看、管理员角色切换。
10. 用户资料页已包含资料编辑、余额、流水、邀请码、我的投稿、充值入口和订单列表。

最终验证：

- `go test ./...` 通过。
- `go build -trimpath -ldflags="-s -w" -o .\bin\lyra-image-workbench.exe .\cmd\local-server` 通过。
- `npm run build` 通过。
- 生产代码 MiniMax/video 残留扫描无命中；仅历史/调度文档保留相关说明。
- 敏感字段扫描命中均为管理员表单字段、后端内部使用或防泄露测试，未发现普通前端明文下发易支付 Key。

仍需人工浏览器验收：

- 真实上游 Key 生图、真实结果投稿、真实易支付异步回调、移动端 375/768/1440 视觉检查。
- 真实支付依赖商户配置和可公网访问的 `PublicBaseURL`；本地可先用签名回调或单元测试验证幂等。
