# 参考项目路由与闭环分析

参考仓库：`https://github.com/y08lin4/ai-image-generate-private`
本次检查对象：本地克隆目录 `_reference_ai_image_generate_private`。

## 1. Worker 后端路由提取

参考项目的后端入口在 `worker/index.ts`，路由集中写在 `fetch()` 分发逻辑里。

| 路由 | 方法 | 前端调用 | 作用 | 鉴权 |
| --- | --- | --- | --- | --- |
| `/api/health` | `GET` | 无直接主流程依赖 | Worker 探活，返回是否绑定 D1/Workflow | 无 |
| `/api/generate-stream` | `POST` | `generateImagesStream()` | Worker 流式代理生图，返回 SSE：`start`、`ping`、`result`、`done`、`error` | `X-Identity-Token` |
| `/api/upload-pixhost` | `POST` | `uploadImageToPixhost()` | 把前端传来的 data URL 上传到 PiXhost | `X-Identity-Token` |
| `/api/image-proxy?url=...` | `GET` | `getImageProxyUrl()` | 只代理 PiXhost 图片，解决 CORS/复制/下载 | 无 |
| `/api/stats` | `GET` | `getBackgroundStats()` | 查询当前空间的今日/累计生成统计 | `X-Identity-Token` |
| `/api/background-tasks` | `POST` | `createBackgroundTask()` | 创建 Cloudflare Workflow 后台任务，D1 插入任务记录 | `X-Identity-Token` |
| `/api/background-tasks` | `GET` | `listBackgroundTasks()` | 按空间密码派生的 owner hash 拉任务列表 | `X-Identity-Token` |
| `/api/background-tasks/:id` | `GET` | `getBackgroundTask()` | 查询单个后台任务 | `X-Identity-Token` |
| `/api/background-tasks/:id/retry` | `POST` | `retryBackgroundTask()` | 以原请求参数创建新的重试任务 | `X-Identity-Token` |
| `/api/background-tasks/:id/images/:index` | `GET` | `fetchBackgroundTaskImage()` | PiXhost 超 10MB 失败时，从 D1 分片回传原图 | `X-Identity-Token` |
| 静态资源 | `GET` | 浏览器 | `env.ASSETS.fetch(request)` 托管前端 | 无 |

## 2. 前端实际请求模式

参考项目前端有三种请求模式：

### 2.1 Worker 流式代理

```text
前端 POST /api/generate-stream
  -> Worker 立即返回 text/event-stream
  -> Worker 并发请求上游 /images/generations 或 /images/edits
  -> 每完成一张就推 result
```

特点：

- 生成任务仍然和这条 SSE 响应绑定。
- 每 10 秒发送一次 `ping`，主要是连接保活和提示用户。
- 多图时由 Worker 拆成多次单图请求，`count` 最大 12，`concurrency` 最大 6。

### 2.2 Worker 后台任务

```text
前端 POST /api/background-tasks
  -> Worker 写 D1
  -> 创建 Cloudflare Workflow
  -> Workflow 请求上游
  -> 结果上传 PiXhost
  -> 状态写 D1
  -> 前端每 5 秒 GET /api/background-tasks/:id 轮询
```

特点：

- 这是真正接近“前端断开也继续跑”的模式。
- 没有 SSE 事件流，前端靠轮询恢复。
- 图生图参考图会先上传 PiXhost，再把 URL 交给 Workflow。
- 生成结果默认走 PiXhost；超过 10MB 时才退回 D1 分片。

### 2.3 浏览器直连

```text
前端直接 fetch 上游 /images/generations 或 /images/edits
```

特点：

- API Key 在浏览器里。
- 请求完全受浏览器、WebView、CORS、页面生命周期影响。
- 不适合 10 分钟稳定生图目标。

## 3. 任务与数据闭环

### 3.1 身份空间

- 用户输入空间密码。
- 前端把密码派生为 64 位 hash 后保存。
- Worker 接受明文复杂密码或 64 位派生 token。
- D1 `tasks.owner_hash` 隔离不同用户空间。

本项目本机版不需要这一层；用户只需要填 API Key，本机后端保存配置即可。

### 3.2 请求参数归一化

参考项目会归一化：

- `mode`: `text-to-image` / `image-to-image`
- `ratio`: `auto` / 固定比例
- `resolution`: `auto` / `standard` / `2k` / `4k`
- `size`: 根据比例和分辨率映射，自动时传“自动”并不发送给上游
- `timeoutSec`: 10 到 900 秒
- `count`: 1 到 12
- `concurrency`: 1 到 6

这些映射可以迁移，但本项目目标是 10 分钟稳定，后端默认超时不应低于 720 秒，最好 900 秒起。

### 3.3 图片返回处理

参考项目支持三种上游返回：

- 响应本身是 `image/*`。
- JSON `data[].b64_json`。
- JSON `data[].url`，Worker/前端再去下载 URL。

这部分适合迁移到 Go 后端。

### 3.4 结果存储

参考项目：

- 普通流式代理：图片 data URL 在前端内存和 IndexedDB 历史里。
- 后台任务：优先上传 PiXhost，D1 保存远程 URL。
- PiXhost 超 10MB：D1 `task_image_chunks` 保存 base64 分片。

本项目本机版应该改为：

```text
Go 后端先把原图落盘到 outputs/YYYY-MM-DD/job-index.png
  -> 任务表记录本地 URL
  -> 前端只展示 /outputs/... 或 /api/background-tasks/:id/images/:index
  -> 图床上传只能作为可选后处理，不能影响生图成功状态
```

## 4. 缺陷与不闭环点

### 4.1 `/api/generate-stream` 不满足 10 分钟稳定目标

问题：

- 生成仍绑定一条 Worker SSE 响应。
- 前端断开、App 切后台、Worker 流关闭后，任务没有 durable job id 可恢复。
- 只靠 `ping` 保活，解决的是“连接看起来不断”，不是“任务一定跑完”。
- 结果没有先落到服务端持久化；如果连接中途失败，前端拿不到已生成图片。

结论：本项目不能把 `/api/generate-stream` 作为主链路，只能作为兼容路由或调试路由。

### 4.2 后台任务比流式稳定，但依然不完全闭环

问题：

- 依赖 Cloudflare Workflow、D1、PiXhost 三个外部组件。
- Workflow 里 `generateOne()` 仍对单图请求设置 `timeoutSec`，默认前端是 420 秒，不是 10 分钟。
- 即使 Workflow step timeout 放宽，上游 NewAPI 如果经过 Cloudflare 同步接口，仍可能在约 100 秒返回 524。
- 没有后台任务 SSE，只能轮询；用户体验上无法看到平滑进度。

结论：后台任务思路正确，但在本机版应由 Go 进程和本地存储替代云端 Workflow/D1/PiXhost。

### 4.3 图床上传被放进成功关键路径

问题：

- `generateOneAndUpload()` 生成成功后立刻上传 PiXhost。
- 如果 PiXhost 失败且不是 10MB 限制，结果会变成 `ok:false`。
- 这会导致“图片已经生成成功，但因为图床失败，任务被判失败”。

结论：本项目必须把“生成成功”和“外部上传成功”拆开。稳定出图目标下，第一落点必须是本机磁盘。

### 4.4 图生图参考图上传前置，任务可能还没创建就失败

问题：

- 后台图生图会先调用 `uploadReferenceImages()` 上传 PiXhost。
- PiXhost 限制、网络失败、图片类型限制都会导致后台任务无法创建。

结论：本项目应由 Go 后端接收参考图并保存到本地临时目录，任务创建和执行不依赖第三方图床。

### 4.5 API Key 与 URL 模型不符合本项目目标

问题：

- 参考项目让前端填写 API URL 和 API Key。
- 默认设置可能把 Key 保存到 localStorage/sessionStorage。
- 后台任务重试还要求浏览器重新提供 Key。

结论：本项目应内置 NewAPI Base URL，前端只填 Key；Key 保存到 Go 后端本机配置，前端不持有 Key，不参与重试请求体。

### 4.6 没有真正的任务取消/恢复执行闭环

参考项目有查询、列表、重试，但没有取消接口。

本项目建议补齐：

- `POST /api/background-tasks/:id/cancel`
- 程序启动时扫描 `queued/running`：
  - 未提交上游的任务可以重新排队。
  - 已提交同步上游但进程退出的任务不能盲目重试，避免重复扣费；应标记 `interrupted`，让用户手动重试。

### 4.7 进度模型不完整

参考项目的后台任务只有状态：`queued/running/uploading/completed/failed/partial_failed`，没有百分比。

本项目为了 10 分钟体验应增加：

- `progress`: 0-100
- `stage`: `queued/submitting/waiting/downloading/saving/succeeded/failed`
- `lastHeartbeatAt`
- `startedAt/finishedAt`
- `attempt`
- SSE 事件：`snapshot/status/progress/result/heartbeat/done/error`

### 4.8 上游 524 不能靠本机假流式解决

关键点：

- 本机假流式只能保证“前端到 Go 后端不断体验”。
- 如果 NewAPI 自身的同步生图接口在 Cloudflare 后面，并且超过 Cloudflare 限制，Go 后端也会收到 524。

要做到“10 分钟生成过程也不会失败”，需要至少满足一个条件：

1. NewAPI 上游提供异步任务接口：创建任务后轮询结果。
2. NewAPI 生图接口不走会 100 秒熔断的 Cloudflare 代理。
3. NewAPI 服务端也改造成后台任务模式，短请求返回 task id。

本项目 Go 后端能保证的是：前端刷新、断线、锁屏不导致任务失败；但不能神奇绕过上游自身的同步超时/524。

## 5. 本项目建议保留的路由

主链路建议只保留“后台任务”语义：

| 路由 | 方法 | 用途 |
| --- | --- | --- |
| `/api/health` | `GET` | 探活 |
| `/api/config` | `GET` | 返回模型、内置 URL 摘要、是否已设置 Key；不返回明文 Key |
| `/api/config` | `POST` | 保存 API Key、本机偏好设置 |
| `/api/background-tasks` | `POST` | 创建本机生图任务，立即返回 job |
| `/api/background-tasks` | `GET` | 查询本机任务列表 |
| `/api/background-tasks/:id` | `GET` | 查询任务快照 |
| `/api/background-tasks/:id/events` | `GET` | SSE 假流式状态；断开不取消任务 |
| `/api/background-tasks/:id/retry` | `POST` | 重试失败/中断任务 |
| `/api/background-tasks/:id/cancel` | `POST` | 取消排队中或可取消的任务 |
| `/api/background-tasks/:id/images/:index` | `GET` | 返回本地落盘图片 |
| `/outputs/...` | `GET` | 静态输出图片 |
| `/api/stats` | `GET` | 本机统计 |
| `/api/generate-stream` | `POST` | 兼容旧前端：内部创建 job 并桥接 SSE，不作为真实长请求 |

## 6. 10 分钟不失败的本机设计底线

为了达到“生成过程 10 分钟也不会失败”，本项目应按下面底线实现：

1. Go 后端创建任务后立即返回，不让前端等待上游。
2. 任务 goroutine/worker 使用独立 context，不能继承 `http.Request.Context()`。
3. 默认上游超时至少 15 分钟，显式支持 10 分钟以上。
4. SSE 只负责观察：断开、重连、刷新都不影响任务。
5. 任务状态和阶段性结果持久化到本地：首版 JSON 可用，正式版建议 SQLite。
6. 生成成功后先保存本机磁盘，再做任何可选上传。
7. 多图任务逐张落盘，部分成功不能因为某一张失败而丢失已成功结果。
8. 程序重启后能恢复任务列表，并明确区分 `queued`、`running/interrupted`、`succeeded`、`failed`。
9. NewAPI Base URL 内置，前端不持有 URL；API Key 只交给 Go 后端保存和使用。
10. 如果上游同步接口会 524，需要换成上游异步接口或非 Cloudflare 长任务入口；本机假流式不能修复上游 524。
