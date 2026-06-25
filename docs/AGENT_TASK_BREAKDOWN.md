# Lyra Image Workbench SaaS 化多代理任务拆分总控

更新时间：2026-06-26  
目标分支：`dev`  
主控原则：主代理负责拆分、派发、验收、集成和风险收口；业务代码尽量交给子代理在明确边界内完成。

## 1. 总目标

把当前 `lyra-image-workbench` 从单机生图工作台推进到一个轻量单节点生图站点 MVP：

1. 用户可以注册、登录、配置个人资料、查看自己的作品和额度流水。
2. 管理员可以在前端看到管理入口，配置上游 Key、系统参数、邮箱、支付和邀请奖励。
3. 用户通过充值获得生图次数；管理员也可以手动给用户增加次数，必须填写原因，并进入流水。
4. 接入易支付，参考 `QuantumNous/new-api` 的订单创建、回调验签、成功入账思路。
5. 邀请系统只在被邀请用户首次成功充值后奖励邀请人次数。
6. 生成结果可以提交到广场，提交后永久保留；未提交的普通结果按 30 天保留策略提示和清理。
7. 广场支持投稿、点赞、每日 00:00-24:00 榜单、用户主页作品。
8. 移除视频/MiniMax 相关功能，但保留 GIF/FFmpeg 功能，并注意 FFmpeg 版本安全提示。
9. 合并任务队列和结果视图，降低重复导航。
10. 提示词助手独立成一级页面，并保留四栏：文生图、图片还原、灵感模式、历史。
11. 新增静态 API 文档页面，提供 curl、Python、Go、Java、JavaScript 示例和一键复制给 AI 的接入提示词。
12. 重构暗色模式，并提供多个主题：白蓝、黑紫、白紫、绿色、粉色、蓝色等。

## 2. 不做或暂缓

1. 暂不做复杂限流、套餐体系、优惠券、发票、分布式队列。
2. 暂不做多节点一致性、数据库迁移到 MySQL/PostgreSQL。
3. 暂不做真实邮件验证闭环，可以先保留 SMTP 配置和后续扩展点。
4. 暂不把未充分测试的新功能合并到 `master`，所有功能先进入 `dev`。
5. 暂不改动 `main/master` 分支历史。

## 3. 全局约束

1. 所有子代理必须先确认 `git status --short --branch`，确认在 `dev` 上。
2. 不允许回滚用户或其他代理的无关改动。
3. 同一文件只能有一个写入负责人。
4. 不允许删除 GIF/FFmpeg 相关模块：`internal/gifrender/*`、`internal/api/gif.go`、`web/src/api/gif.ts`、`web/src/components/GifWorkbenchPage.tsx`。
5. 删除视频功能时只删除 MiniMax/视频生成，不删除 GIF。
6. 支付相关不能把商户密钥、上游 Key、SMTP 密码返回给普通前端。
7. 易支付回调必须幂等，重复通知不能重复加次数。
8. 额度流水必须不可变追加；管理员加次数必须有必填原因。
9. 前端主题要保证可读性，暗色模式文本对比度优先。
10. 任务完成后必须跑最小验证，最终由主代理统一跑全量验证。

## 4. 代码现状摘要

后端入口：

- `cmd/local-server/main.go`
- `internal/api/router.go`
- `internal/api/users.go`
- `internal/api/admin_users.go`
- `internal/api/admin_config.go`
- `internal/api/auth.go`
- `internal/api/tasks.go`
- `internal/api/v1_image_tasks.go`
- `internal/api/prompt_square.go`

核心存储：

- 用户：`internal/users/store.go`
- 设置：`internal/settings/settings.go`
- 任务：`internal/jobs/*`
- 输出：`internal/output/store.go`
- 广场：`internal/promptsquare/store.go`
- API Key：`internal/apikeys/store.go`

前端入口：

- `web/src/main.tsx`
- `web/src/components/WorkbenchPage.tsx`
- `web/src/styles.css`
- `web/src/types.ts`

当前已存在但需要改造：

- 登录注册：`web/src/components/SpaceLogin.tsx`
- 管理后台：`web/src/components/AdminPage.tsx`
- 设置页：`web/src/components/SettingsPanel.tsx`
- 广场：`web/src/components/PromptSquarePanel.tsx`
- 结果：`web/src/components/ResultCanvas.tsx`
- 队列：`web/src/components/TaskQueue.tsx`、`web/src/components/TaskSidebar.tsx`
- 提示词助手：`web/src/components/PromptAssistantModal.tsx`
- 主题切换：`web/src/components/ThemeToggle.tsx`

## 5. 第一阶段接口契约

### 5.1 用户模型

`PublicUser` 需要扩展字段：

- `username`
- `displayName`
- `email`
- `avatarUrl`
- `isAdmin`
- `creditsBalance`
- `referralCode`
- `referredByUsername`
- `createdAt`
- `lastLoginAt`
- `twoFactorEnabled`

注册请求：

```json
{
  "username": "alice",
  "email": "alice@example.com",
  "password": "password",
  "referralCode": "OPTIONAL"
}
```

登录请求：

```json
{
  "identifier": "alice or alice@example.com",
  "password": "password",
  "totpCode": "OPTIONAL"
}
```

兼容策略：

- 如果前端仍传 `username`，后端兼容。
- 老用户 `email` 可以为空。
- 第一个注册用户自动成为管理员。
- 密码继续使用现有不可逆哈希方案，不引入明文保存。

### 5.2 额度流水模型

`CreditLedgerEntry`：

```json
{
  "id": "ledger_xxx",
  "username": "alice",
  "delta": 10,
  "balanceAfter": 23,
  "type": "admin_add | purchase | referral_reward | task_charge | refund",
  "reason": "管理员补偿",
  "sourceId": "order_xxx",
  "adminActor": "admin",
  "relatedUsername": "bob",
  "createdAt": "2026-06-26T00:00:00Z"
}
```

必须满足：

- 增加和扣减都写流水。
- 管理员手动增加必须有 `reason`。
- 订单入账使用 `sourceId=tradeNo` 幂等。
- 邀请奖励只发一次。

### 5.3 管理员接口

新增或替换：

- `GET /api/admin/users`
- `POST /api/admin/users/credits/add`
- `GET /api/admin/users/{username}/ledger`
- `POST /api/admin/users/{username}/role`
- `GET /api/admin/config`
- `PUT /api/admin/config`

`POST /api/admin/users/credits/add`：

```json
{
  "username": "alice",
  "amount": 50,
  "reason": "线下付款补录"
}
```

### 5.4 用户主页接口

新增：

- `GET /api/users/profile`
- `PUT /api/users/profile`
- `GET /api/users/ledger`
- `GET /api/users/me/prompt-square-items`
- `POST /api/users/referral-code`

资料更新请求：

```json
{
  "displayName": "Alice",
  "email": "alice@example.com",
  "avatarUrl": "https://example.com/avatar.png"
}
```

### 5.5 易支付接口

参考 `QuantumNous/new-api`：

- 创建订单类似 `/api/user/pay`
- 通知回调类似 `/api/user/epay/notify`
- 参数按易支付规则排序，排除 `sign` 和 `sign_type`，拼接后追加商户 Key，MD5 小写。
- 回调成功返回纯文本 `success`，失败返回 `fail`。

本项目建议接口：

- `GET /api/billing/topup/options`
- `POST /api/billing/epay/orders`
- `GET /api/billing/epay/notify`
- `POST /api/billing/epay/notify`
- `GET /api/billing/topups`

订单创建请求：

```json
{
  "credits": 100,
  "method": "alipay"
}
```

订单创建响应：

```json
{
  "tradeNo": "LYRA202606260001",
  "payUrl": "https://pay.example.com/submit.php?...",
  "credits": 100,
  "amountCents": 1000,
  "status": "pending"
}
```

设置项：

- `epayEnabled`
- `epayApiUrl`
- `epayPid`
- `epayKey`
- `epayMethods`
- `creditPriceCents`
- `minTopUpCredits`
- `referralRewardCredits`

安全要求：

- `epayKey` 不下发给普通前端。
- 回调不依赖用户登录。
- 回调必须校验签名。
- 回调必须校验金额、订单号、状态。
- 重复回调不重复加次数。

### 5.6 广场接口

保留：

- `GET /api/prompt-square/items`

新增：

- `POST /api/prompt-square/from-result`
- `POST /api/prompt-square/items/{id}/like`
- `GET /api/prompt-square/daily`
- `GET /api/prompt-square/mine`

提交结果请求：

```json
{
  "taskId": "task_xxx",
  "imageIndex": 0,
  "title": "可选标题",
  "tags": ["1:1", "高清", "gpt-image-2"]
}
```

提交时后端负责：

- 从任务结果中读取图片。
- 复制图片到广场永久目录，例如 `data/prompt_square/images`。
- 保存 prompt、模型、比例、质量、输出格式、作者、提交时间。
- 标记该广场作品永久保留。

点赞请求：

```json
{
  "liked": true
}
```

每日榜：

- 按服务器本地日期 00:00 到 24:00 过滤。
- 只统计当天投稿作品。
- 排序优先点赞数，点赞数相同按投稿时间早者靠前。

### 5.7 任务与结果

前端合并：

- 取消独立“任务队列”一级页。
- “结果”页左侧或上方展示任务列表，右侧展示当前任务结果。
- 每张结果图显示“提交到广场”按钮。
- 未提交结果提示：“未提交到广场的结果会在 30 天后清理；提交后将永久保留。”

后端暂定：

- 先不强制实现真实 30 天清理任务，可以补充提示和保留策略文档。
- 如果已有清理逻辑，则加入 `submittedToSquare` 或广场永久副本保护。

### 5.8 API 文档页面

新增前端静态页：

- curl 示例
- Python 示例
- Go 示例
- Java 示例
- JavaScript/Node 示例
- “复制给 AI 的提示词”按钮
- GitHub 文档仓库链接：`https://github.com/y08lin4/LyAi-Image-Generation-API-Documentation`

要求：

- 页面只写调用方法，不写营销废话。
- 示例统一使用 Bearer API Key。
- 明确先去 `https://ai-image.ailinyu.de/` 注册、配置上游、生成 Bearer Key。

### 5.9 主题系统

主题键建议：

- `white-blue`
- `black-purple`
- `white-purple`
- `green`
- `pink`
- `blue`

要求：

- `ThemeToggle` 从二选一改为主题选择。
- `main.tsx` 或主题入口保存 `localStorage`。
- CSS 使用 `[data-theme="..."]` 变量。
- 暗色主题必须修复输入、标签、弹窗、提示词助手、管理页、广场卡片的对比度。

### 5.10 视频删除边界

删除或移除引用：

- `internal/api/minimax_video.go`
- `internal/minimax/video.go`
- `web/src/api/minimaxVideos.ts`
- `web/src/components/MiniMaxVideoPanel.tsx`
- `docs/MINIMAX_VIDEO.md`

必须改：

- `cmd/local-server/main.go`
- `internal/api/router.go`
- `internal/api/auth.go`
- `internal/api/admin_users.go`
- `internal/settings/settings.go`
- `internal/users/store.go`
- `web/src/components/WorkbenchPage.tsx`
- `web/src/components/AdminPage.tsx`
- `web/src/api/admin.ts`
- `web/src/types.ts`
- `web/src/styles.css`
- `internal/api/router_test.go`
- `internal/settings/settings_test.go`
- `internal/users/store_test.go`

禁止误删：

- GIF 工作台
- GIF API
- FFmpeg GIF 渲染测试和服务

FFmpeg 安全备注：

- GIF 分支会用到 FFmpeg。
- 最终文档/部署说明要提醒生产环境使用已修复高危漏洞的 FFmpeg 版本。
- 不接受用户上传视频到 FFmpeg 解码链路作为本轮功能。

## 6. 子代理波次规划

### Wave 1：核心契约与高风险后端先行

并发上限：6 个代理。

#### Agent A1：用户、额度、邀请后端

角色：`backend-engineer`  
模式：写入  
目标：实现用户模型扩展、额度余额、流水、管理员加次数、邀请绑定与首次充值奖励基础方法。

可编辑：

- `internal/users/store.go`
- `internal/users/store_test.go`
- `internal/api/users.go`
- `internal/api/admin_users.go`

可创建：

- `internal/users/*_test.go`

不可编辑：

- `internal/settings/settings.go`
- `internal/api/router.go`
- `internal/api/prompt_square.go`
- `web/**`

具体任务：

1. 扩展用户记录字段：邮箱、头像、管理员、余额、邀请码、邀请关系、邀请奖励时间。
2. 注册接口支持邮箱和邀请码。
3. 登录支持用户名或邮箱。
4. 第一个注册用户自动管理员。
5. 新增用户资料读取和更新方法。
6. 新增额度流水结构和追加方法。
7. 新增管理员加次数方法，原因必填。
8. 新增购买入账方法，按 `sourceId` 幂等。
9. 新增邀请奖励方法，只在被邀请用户首次成功充值后触发一次。
10. 更新用户列表给管理员展示余额和邮箱。
11. 增加单元测试覆盖注册兼容、管理员首用户、加次数、原因必填、重复订单不重复入账、邀请奖励只发一次。

验收：

- `go test ./internal/users ./internal/api -run "Users|AdminUsers|Credit|Referral"` 可通过或给出不能通过原因。
- 输出改动文件和新增方法清单。

#### Agent A2：易支付和计费后端

角色：`backend-engineer`  
模式：写入  
目标：新增易支付订单、签名、回调、订单存储和入账衔接。

可编辑：

- `internal/settings/settings.go`
- `internal/settings/settings_test.go`
- `internal/api/admin_config.go`
- `internal/api/router.go`

可创建：

- `internal/billing/store.go`
- `internal/billing/epay.go`
- `internal/billing/store_test.go`
- `internal/api/billing.go`

不可编辑：

- `internal/users/store.go`
- `internal/api/users.go`
- `internal/api/admin_users.go`
- `web/**`

具体任务：

1. 新增计费配置字段：易支付开关、网关地址、商户 PID、商户 Key、支付方式、次数单价、最小充值次数、邀请奖励次数。
2. 管理配置接口支持保存和清空易支付 Key。
3. 新增 `billing.Store`，JSON 持久化订单。
4. 订单字段包括：订单号、用户名、次数、金额、支付方式、状态、三方交易号、创建时间、支付时间。
5. 实现易支付签名函数：排序参数，排除 `sign/sign_type`，拼接 `key` 后 MD5。
6. 实现创建订单接口，返回 `payUrl`。
7. 实现 GET/POST 回调，校验签名、金额、订单状态。
8. 回调成功后调用用户入账接口；若 A1 未完成，先定义清晰接口 TODO 并保持编译。
9. 回调重复通知必须返回成功但不重复入账。
10. 参考 `QuantumNous/new-api` 的 `controller/topup.go` 和 `model/topup.go` 组织流程，但不强行引入外部依赖。

验收：

- `go test ./internal/billing ./internal/settings` 可通过。
- 给出与 A1 的接口对接点。

#### Agent A3：广场、投稿、点赞、每日榜后端

角色：`backend-engineer`  
模式：写入  
目标：实现从生成结果投稿到广场、永久保存副本、点赞、每日榜和我的作品接口。

可编辑：

- `internal/promptsquare/store.go`
- `internal/api/prompt_square.go`

可创建：

- `internal/promptsquare/store_test.go`
- `internal/api/prompt_square_test.go`

不可编辑：

- `internal/users/store.go`
- `internal/jobs/**`
- `internal/output/**`
- `web/**`

具体任务：

1. 扩展广场条目字段：作者、作者显示名、模型、比例、质量、输出格式、标签、点赞数、是否我已点赞、永久保留标记、原任务 ID。
2. 增加 `SubmitFromResult` 相关 API。
3. 读取任务结果图片并复制到广场永久目录。
4. 复制失败时返回明确错误，不污染广场记录。
5. 增加点赞/取消点赞，按用户名去重。
6. 增加每日榜查询。
7. 增加我的作品查询。
8. 保留旧上传接口只作为兼容或标记废弃；前端不再展示上传入口。
9. 测试点赞幂等、每日榜排序、我的作品过滤。

验收：

- `go test ./internal/promptsquare ./internal/api -run "PromptSquare"` 可通过或说明依赖缺口。
- 明确告诉前端需要的请求/响应字段。

#### Agent A4：视频删除和构建修复

角色：`backend-engineer`  
模式：写入  
目标：彻底移除 MiniMax/视频功能，同时保护 GIF/FFmpeg。

可编辑：

- `cmd/local-server/main.go`
- `internal/api/router.go`
- `internal/api/auth.go`
- `internal/api/admin_users.go`
- `internal/settings/settings.go`
- `internal/settings/settings_test.go`
- `internal/api/router_test.go`
- `internal/users/store.go`
- `internal/users/store_test.go`
- `README.md`

可删除：

- `internal/api/minimax_video.go`
- `internal/minimax/video.go`
- `docs/MINIMAX_VIDEO.md`

不可编辑：

- `internal/gifrender/**`
- `internal/api/gif.go`
- `web/**`

具体任务：

1. 移除 MiniMax 依赖注入。
2. 移除 `/api/minimax/*` 路由。
3. 移除视频额度后台路由。
4. 移除设置里的 MiniMax Key 字段。
5. 移除用户存储里的 `VideoQuota` 和相关方法，或与 A1 协调改成 credits 后不重复删除。
6. 删除 MiniMax 后端文件和视频文档。
7. 更新测试，确保不再引用 MiniMax。
8. 扫描残留关键字：`minimax|videoQuota|MiniMaxVideo|/api/minimax`。
9. 保留所有 GIF/FFmpeg 文件。
10. README 中删除视频说明，但保留 GIF/FFmpeg 安全部署提醒。

验收：

- `rg -n -i "minimax|videoQuota|MiniMaxVideo|/api/minimax" internal cmd docs README.md` 无业务残留，若只剩历史说明需标明。
- `go test ./...` 至少不因 MiniMax 删除而失败。

#### Agent A5：前端类型、账号、管理入口、API 文档

角色：`frontend-engineer`  
模式：写入  
目标：完成用户系统前端骨架、管理员入口、资料页、API 文档静态页。

可编辑：

- `web/src/types.ts`
- `web/src/api/users.ts`
- `web/src/api/admin.ts`
- `web/src/components/SpaceLogin.tsx`
- `web/src/components/WorkbenchPage.tsx`
- `web/src/components/SettingsPanel.tsx`

可创建：

- `web/src/components/ProfilePage.tsx`
- `web/src/components/ApiDocsPage.tsx`
- `web/src/api/billing.ts`

不可编辑：

- `web/src/components/PromptSquarePanel.tsx`
- `web/src/components/ResultCanvas.tsx`
- `web/src/components/ThemeToggle.tsx`
- `web/src/styles.css`
- `internal/**`

具体任务：

1. 扩展前端用户类型：邮箱、头像、管理员、余额、邀请码。
2. 注册表单增加邮箱和邀请码；登录输入改为用户名或邮箱。
3. 新增资料页：头像、用户名、显示名、邮箱、余额、流水列表、邀请码复制、被邀请作品入口。
4. 设置页只在 `session.user.isAdmin` 为真时展示管理入口。
5. 工作台增加一级导航：我的、API 文档。
6. 新增 API 文档页面，包含 curl、Python、Go、Java、JavaScript 示例。
7. API 文档页面增加“复制给 AI 的接入提示词”按钮。
8. 文档页面链接 GitHub 文档仓库。
9. 不碰广场和结果组件，避免和 A6/A7 冲突。

验收：

- `cd web && npm run build` 若失败，列出是否等待后端类型或其他代理。
- 输出新增页面和 API wrapper 清单。

#### Agent A6：主题、暗色模式、提示词助手一级页

角色：`frontend-engineer`  
模式：写入  
目标：重构主题系统，修复暗色模式，提示词助手独立一级页。

可编辑：

- `web/src/main.tsx`
- `web/src/components/ThemeToggle.tsx`
- `web/src/components/PromptAssistantModal.tsx`
- `web/src/components/WorkbenchPage.tsx`
- `web/src/styles.css`

可创建：

- `web/src/lib/themes.ts`

不可编辑：

- `web/src/types.ts`
- `web/src/api/**`
- `web/src/components/ProfilePage.tsx`
- `web/src/components/ApiDocsPage.tsx`
- `internal/**`

具体任务：

1. 主题从 `light/dark` 改为多主题选择。
2. 支持主题：白蓝、黑紫、白紫、绿色、粉色、蓝色。
3. 用 CSS 变量定义背景、文字、边框、按钮、卡片、输入、危险/成功状态。
4. 修复暗色模式提示词助手、输入框、管理区、弹窗、标签、按钮的对比度。
5. 将提示词助手从生成页内小组件提升为一级页。
6. 提示词助手内部四栏保持：文生图、图片还原、灵感模式、历史。
7. 给 tab 增加合理的 `aria-selected`。
8. 不新增营销型说明文案。
9. 保证移动端文字不溢出按钮。
10. 不使用装饰性渐变球、嵌套卡片。

验收：

- 至少人工说明白蓝、黑紫、白紫、绿色、粉色、蓝色的变量覆盖点。
- `npm run build` 结果或阻塞原因。

### Wave 2：前端业务体验和支付联调

在 Wave 1 返回后，主代理先关闭完成代理，再派发：

#### Agent B1：广场页面重构

可编辑：

- `web/src/components/PromptSquarePanel.tsx`
- `web/src/api/promptSquare.ts`
- `web/src/styles.css`

任务：

1. 移除上传提示词入口。
2. 改为浏览广场、每日榜、我的投稿。
3. 图片容器使用 `object-fit: contain` 或按比例容器，禁止拉伸变形。
4. 加点赞按钮、点赞数、每日排名徽章。
5. 卡片展示比例、质量、模型、作者、发布时间。
6. 空状态和加载状态清晰。

#### Agent B2：结果页和任务队列合并

可编辑：

- `web/src/components/ResultCanvas.tsx`
- `web/src/components/TaskSidebar.tsx`
- `web/src/components/TaskQueue.tsx`
- `web/src/components/WorkbenchPage.tsx`
- `web/src/api/promptSquare.ts`
- `web/src/styles.css`

任务：

1. 取消独立队列一级页。
2. 结果页内显示任务列表和当前任务详情。
3. 每张结果图增加“提交到广场”。
4. 点击后弹确认，展示 prompt、模型、比例、质量、图片。
5. 调用 `POST /api/prompt-square/from-result`。
6. 提示未投稿结果 30 天后删除，投稿后永久保留。

#### Agent B3：支付和充值前端

可编辑：

- `web/src/api/billing.ts`
- `web/src/components/ProfilePage.tsx`
- `web/src/components/AdminPage.tsx`
- `web/src/styles.css`

任务：

1. 资料页显示充值入口。
2. 选择充值次数和支付方式。
3. 创建订单后打开或展示支付链接。
4. 显示订单列表和状态。
5. 管理页显示易支付配置。
6. 管理页显示邀请奖励次数配置。

#### Agent B4：管理员用户和流水界面

可编辑：

- `web/src/components/AdminPage.tsx`
- `web/src/api/admin.ts`
- `web/src/styles.css`

任务：

1. 管理用户列表展示邮箱、余额、管理员状态、注册时间。
2. 管理员加次数表单必须填写原因。
3. 用户详情展示流水。
4. 可设置/取消管理员。
5. 所有敏感操作显示确认。

#### Agent B5：前端移除视频

可编辑：

- `web/src/components/WorkbenchPage.tsx`
- `web/src/components/AdminPage.tsx`
- `web/src/api/admin.ts`
- `web/src/types.ts`
- `web/src/styles.css`

可删除：

- `web/src/components/MiniMaxVideoPanel.tsx`
- `web/src/api/minimaxVideos.ts`

任务：

1. 移除视频一级 tab。
2. 移除 MiniMax 设置表单。
3. 移除视频额度 UI。
4. 移除视频相关类型和 API。
5. 扫描残留关键字。

#### Agent B6：前端 QA 和响应式审查

模式：只读或测试

任务：

1. 检查 375、768、1440 三个宽度可能溢出的区域。
2. 检查主题对比度风险。
3. 检查提交广场、充值、资料页的空/错/加载状态。
4. 不修改业务代码，输出问题清单。

### Wave 3：集成、测试、启动、提交

#### Agent C1：后端集成修复

任务：

1. 解决 A1/A2/A3/A4 的接口冲突。
2. 跑 `go test ./...`。
3. 修复编译错误。

#### Agent C2：前端集成修复

任务：

1. 解决 A5/A6/B1/B2/B3/B4/B5 的类型和样式冲突。
2. 跑 `cd web && npm run build`。
3. 修复构建错误。

#### Agent C3：安全审查

任务：

1. 审查支付签名、回调幂等、密钥下发、管理员鉴权、额度重复入账。
2. 审查广场投稿是否越权读取其他用户结果。
3. 审查 API Key 和上游 Key 是否泄露。

#### Agent C4：浏览器验收

任务：

1. 启动本地服务。
2. 打开 `http://127.0.0.1:5173/`。
3. 截图检查首页、结果页、广场、资料页、API 文档、管理页、主题切换。
4. 记录无法验收的后端依赖。

#### Agent C5：文档和变更说明

任务：

1. 更新 README 或 docs。
2. 写简体中文变更说明。
3. 写支付配置说明。
4. 写 FFmpeg 安全提醒。

#### Agent C6：发布负责人

任务：

1. 检查 `git status`。
2. 整理提交信息。
3. 确认只在 `dev` 提交和 push。
4. 列出未完成和未测试项。

## 7. 主代理调度规则

1. 同时最多保持 6 个运行中子代理。
2. 每次有子代理完成，主代理立即读取结果。
3. 完成且不再需要上下文的子代理立即 `close_agent` 回收。
4. 如果结果可合并，主代理登记到本文档或最终集成清单。
5. 如果结果阻塞，主代理新派一个更小的问题定位代理。
6. 如果两个子代理改了同一文件，优先保留契约更完整的一方，另一方作为参考，不直接盲合。
7. Wave 1 未完成前，不派 Wave 2 中依赖后端接口的写入任务。
8. Wave 2 完成后才进入 Wave 3 的全量构建和启动验收。

## 8. 监控看板

| 波次 | 代理 | 状态 | 负责人角色 | 任务摘要 | 写入范围 |
| --- | --- | --- | --- | --- | --- |
| Wave 1 | A1 | 待派发 | backend-engineer | 用户、额度、邀请后端 | `internal/users/*`, `internal/api/users.go`, `internal/api/admin_users.go` |
| Wave 1 | A2 | 待派发 | backend-engineer | 易支付和计费后端 | `internal/billing/*`, `internal/settings/*`, `internal/api/billing.go`, `internal/api/router.go`, `internal/api/admin_config.go` |
| Wave 1 | A3 | 待派发 | backend-engineer | 广场投稿、点赞、每日榜后端 | `internal/promptsquare/*`, `internal/api/prompt_square.go` |
| Wave 1 | A4 | 待派发 | backend-engineer | 后端视频删除 | `cmd/*`, `internal/api/*`, `internal/settings/*`, `internal/users/*`, `README.md` |
| Wave 1 | A5 | 待派发 | frontend-engineer | 前端账号、资料、API 文档 | `web/src/types.ts`, `web/src/api/users.ts`, `web/src/api/admin.ts`, `web/src/components/SpaceLogin.tsx`, `web/src/components/SettingsPanel.tsx`, `ProfilePage`, `ApiDocsPage` |
| Wave 1 | A6 | 待派发 | frontend-engineer | 多主题、暗色模式、提示词助手一级页 | `web/src/main.tsx`, `ThemeToggle`, `PromptAssistantModal`, `WorkbenchPage`, `styles.css`, `themes.ts` |
| Wave 2 | B1 | 未开始 | frontend-engineer | 广场页面重构 | `PromptSquarePanel`, `promptSquare.ts`, `styles.css` |
| Wave 2 | B2 | 未开始 | frontend-engineer | 结果和队列合并 | `ResultCanvas`, `TaskSidebar`, `TaskQueue`, `WorkbenchPage`, `styles.css` |
| Wave 2 | B3 | 未开始 | frontend-engineer | 充值支付前端 | `billing.ts`, `ProfilePage`, `AdminPage`, `styles.css` |
| Wave 2 | B4 | 未开始 | frontend-engineer | 管理用户和流水界面 | `AdminPage`, `admin.ts`, `styles.css` |
| Wave 2 | B5 | 未开始 | frontend-engineer | 前端视频删除 | `WorkbenchPage`, `AdminPage`, `admin.ts`, `types.ts`, `styles.css` |
| Wave 2 | B6 | 未开始 | ui-ux-reviewer | 响应式和主题审查 | 只读 |
| Wave 3 | C1 | 未开始 | backend-engineer | 后端集成修复 | 待定 |
| Wave 3 | C2 | 未开始 | frontend-engineer | 前端集成修复 | 待定 |
| Wave 3 | C3 | 未开始 | security-reviewer | 安全审查 | 只读 |
| Wave 3 | C4 | 未开始 | qa-engineer | 浏览器验收 | 只读/测试 |
| Wave 3 | C5 | 未开始 | technical-writer | 文档和变更说明 | `docs/*`, `README.md` |
| Wave 3 | C6 | 未开始 | release-manager | 提交和发布检查 | 只读 |

## 9. 最终验证清单

后端：

```powershell
go test ./...
go build -trimpath -ldflags="-s -w" -o .\bin\lyra-image-workbench.exe .\cmd\local-server
```

前端：

```powershell
cd web
npm run build
```

残留扫描：

```powershell
rg -n -i "minimax|videoQuota|MiniMaxVideo|/api/minimax" .
rg -n -i "epayKey|smtpPassword|upstreamKey" web/src internal/api
```

浏览器验收：

1. 注册首个用户，确认自动管理员。
2. 管理员配置上游 Key、易支付、SMTP、邀请奖励。
3. 用户生成图片，结果页显示 30 天保留提示。
4. 用户提交结果到广场，确认图片不变形。
5. 广场点赞，确认每日榜变化。
6. 用户主页显示投稿作品、余额和流水。
7. 管理员给用户加次数，原因必填且流水可见。
8. 创建易支付订单，模拟或真实回调只入账一次。
9. 切换白蓝、黑紫、白紫、绿色、粉色、蓝色主题。
10. API 文档页面复制提示词和代码示例。

## 10. 回滚和风险点

最高风险：

1. A1 和 A4 都会碰 `internal/users/store.go`，主代理需要让 A4 如果发现 A1 已改 credits，就只删除视频残留，不重写用户模型。
2. A2 和 A4 都会碰 `internal/settings/settings.go`，A2 负责新增支付设置，A4 只删除 MiniMax 设置。
3. A5 和 A6 都会碰 `WorkbenchPage.tsx`，A5 负责导航和新页面接入，A6 负责主题和提示词助手入口；谁先完成谁先落地，后完成者必须适配。
4. 多个前端任务会碰 `styles.css`，需要后续 C2 统一整理。
5. 易支付回调如果没有幂等会造成重复加次数，必须优先测。
6. 广场永久保存不能直接引用临时任务输出，否则 30 天清理会破图。

主代理处理方式：

- 对共享文件，先接收更小、更明确的 patch。
- 对冲突区，主代理做人工合并，不让子代理互相覆盖。
- 任何支付、额度、权限相关变更必须通过安全审查后再 push。
