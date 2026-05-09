# 本项目闭环设计审查

本文记录当前项目里还没有完全闭环、没有明确或容易后续返工的点，并给出确定设计。后续实现以本文为准；如果实际代码和本文冲突，优先更新本文再改代码。

## 1. 已明确的核心目标

目标链路：

```text
浏览器 / WebView
  -> 同源页面与同源 /api
  -> Go 后端
  -> 内网 NewAPI Base URL
  -> Go 后端保存 outputs/
  -> 前端展示同源图片 URL
```

硬性要求：

- 前端不直接请求 NewAPI。
- 前端不保存 NewAPI Base URL。
- 前端不把 API Key 带去 NewAPI。
- Go 后端是唯一请求 NewAPI 的模块。
- NewAPI 地址内置在 Go 后端配置里，用户只填 Key。
- 创建生图任务必须快速返回，不能让前端 HTTP 请求等待 10 分钟。
- 前端刷新、断线、手机锁屏，只影响观察连接，不影响后端任务。
- 生成成功后先保存本机磁盘，再考虑任何可选上传或复制。

## 2. 当前未闭环点与结论

| 点 | 当前风险 | 结论设计 |
| --- | --- | --- |
| 前端访问边界 | 容易误解成前端硬编码 localhost 或请求 NewAPI | 前端只使用同源相对路径 `/api/...`；生产由 Go 托管前端静态资源 |
| NewAPI 地址 | “内置 URL”来源未明确 | 编译默认值 + 环境变量覆盖 + 本机配置文件覆盖；UI 默认不展示完整 URL，只显示“内置线路”状态 |
| API Key 保存 | 浏览器保存 Key 不符合目标 | Key 只通过 `POST /api/config` 交给 Go；Go 本机保存，`GET /api/config` 只返回 `apiKeySet` 和掩码 |
| 10 分钟任务 | 如果任务继承请求上下文，前端断开会取消 | job runner 使用独立后台 context；SSE/查询连接绝不能控制任务生命周期 |
| 上游超时 | Go 默认 HTTP 没有清晰超时策略 | NewAPI 单次请求默认 600 秒超时，可在 `/admin` 设置；服务端 `WriteTimeout=0` 支持 SSE |
| Cloudflare 524 | 本机假流式无法修复上游 524 | 本项目假设 Go 通过内网直连 NewAPI，不经过会 100 秒熔断的同步代理；若上游仍 524，需要 NewAPI 提供异步任务或非 CF 长任务入口 |
| 任务状态 | 状态字段未统一 | 使用 `queued/running/succeeded/partial_failed/failed/cancelled/interrupted`，另有 `stage` 表示细分阶段 |
| 假流式进度 | 没有明确算法 | SSE 推 `snapshot/progress/result/heartbeat/done/error`；等待上游期间按时间缓慢推进到 90%，真实结果覆盖进度 |
| 任务持久化 | JSON/SQLite 未定 | 首版用原子 JSON 存储快速落地；正式稳定版升级 SQLite。接口按 `store.Store` 抽象，避免后续大改 |
| 程序重启恢复 | running 任务无法知道上游是否完成 | 重启时 `queued` 可重新排队；`running` 标记为 `interrupted`，不自动重试，避免重复扣费 |
| 多图生成 | 一次请求 `n>1` 失败会全失败 | 后端拆成多次单图请求，按 `count` 创建子任务；每张独立落盘，允许部分成功 |
| 图生图参考图 | 参考项目依赖 PiXhost，不适合本机稳定 | 新增本机上传闭环：前端先 `POST /api/uploads/reference`，Go 保存到 `data/uploads/`，创建任务只传 `uploadId` |
| 图片落盘 | 路径安全和 URL 映射未明确 | 只允许输出到 `outputs/YYYY-MM-DD/{jobID}-{index}.{ext}`；读取时校验路径必须在 outputs 根目录内 |
| 结果历史 | 前端 IndexedDB 和后端任务历史可能重复 | 以后端任务库为主；前端只缓存 UI 偏好，不作为唯一历史来源 |
| 取消任务 | 参考项目没有取消 | 增加 `POST /api/background-tasks/:id/cancel`；排队任务可取消，运行中尽力取消但不能保证上游不计费 |
| 手机适配/APP | Web 响应式与自包含手机 App 是两件事 | 首版做响应式网站 + 桌面本机服务；桌面 App 用 Wails。手机浏览器可访问同源服务；自包含手机 App 需要后续 native 请求适配，不默认承诺 Go 后端嵌入手机 App |
| 局域网访问 | 手机访问 PC 服务需要监听 LAN，但会暴露 Key 能力 | 默认只监听 `127.0.0.1`；如开启 LAN 模式，必须增加本机访问口令或配对 token |

## 3. 最终后端模块边界

```text
cmd/local-server
  main.go
    只加载配置、装配依赖、启动 HTTP server。

internal/config
  读取端口、监听模式、内置 NewAPI Base URL、默认模型、超时、并发限制。

internal/api
  router.go
  config_handler.go
  task_handler.go
  upload_handler.go
  sse_handler.go
  output_handler.go
    只负责 HTTP 入参、状态码、JSON/SSE 输出。

internal/app
  service.go
    编排 config、jobs、store、events、output、newapi，避免 handler 直接互相依赖。

internal/jobs
  types.go
  queue.go
  runner.go
  progress.go
    管理任务状态机、队列、并发、重试、取消和假进度。

internal/newapi
  client.go
  request.go
  response.go
    只负责调用内网 NewAPI，解析 image/b64/url 返回。

internal/store
  store.go
  json_store.go
  sqlite_store.go
    抽象任务、配置、统计持久化。

internal/events
  hub.go
    管理 SSE 订阅、广播、事件序号、心跳。

internal/output
  files.go
    图片落盘、MIME/扩展名判断、安全路径读取。

internal/uploads
  reference.go
    图生图参考图保存、清理、读取。
```

## 4. 最终前端模块边界

```text
web/src/App.tsx
  只做布局编排，不直接写 fetch 细节。

web/src/api/
  client.ts       # fetch 包装、错误处理
  config.ts       # /api/config
  tasks.ts        # /api/background-tasks
  uploads.ts      # /api/uploads/reference

web/src/hooks/
  useTaskEvents.ts     # SSE 自动重连，断线后先拉快照
  useTaskList.ts       # 任务列表同步
  useConfig.ts         # Key 设置状态

web/src/components/
  SettingsPanel.tsx
  PromptPanel.tsx
  UploadPanel.tsx
  TaskQueue.tsx
  ResultGrid.tsx
  HistoryPanel.tsx
  ImagePreview.tsx

web/src/lib/
  ratios.ts
  format.ts
  files.ts
```

## 5. API 路由闭环设计

### 配置

```text
GET  /api/config
POST /api/config
```

`GET /api/config` 返回：

```json
{
  "ok": true,
  "config": {
    "apiKeySet": true,
    "apiKeyPreview": "sk-...abcd",
    "baseUrlMode": "builtin",
    "model": "gpt-image-2",
    "defaultTimeoutSec": 600,
    "maxCount": 12,
    "maxConcurrency": 4
  }
}
```

`POST /api/config` 只接收用户需要填的内容：

```json
{
  "apiKey": "...",
  第一版模型固定为 `gpt-image-2`，暂不允许请求体覆盖模型
}
```

### 参考图上传

```text
POST   /api/uploads/reference
GET    /api/uploads/reference/:id
DELETE /api/uploads/reference/:id
```

- 使用 `multipart/form-data` 上传。
- 后端保存到 `data/uploads/`。
- 创建图生图任务时只传 `uploadIds`。
- 不依赖第三方图床。

### 生图任务

```text
POST /api/background-tasks
GET  /api/background-tasks
GET  /api/background-tasks/:id
GET  /api/background-tasks/:id/events
POST /api/background-tasks/:id/retry
POST /api/background-tasks/:id/cancel
GET  /api/background-tasks/:id/images/:index
```

创建任务请求：

```json
{
  "mode": "text-to-image",
  "prompt": "...",
  "ratio": "1:1",
  "resolution": "standard",
  "count": 1,
  "concurrency": 1,
  "uploadIds": []
}
```

创建任务响应必须快速返回：

```json
{
  "ok": true,
  "job": {
    "id": "img_xxx",
    "status": "queued",
    "stage": "queued",
    "progress": 0
  }
}
```

### 输出图片

```text
GET /outputs/YYYY-MM-DD/file.png
```

或：

```text
GET /api/background-tasks/:id/images/:index
```

后者会校验任务和结果存在；前者只做安全静态文件读取。

### 兼容路由

```text
POST /api/generate-stream
```

只作为兼容旧前端/调试：内部创建后台任务，然后桥接 `/events`。不允许把真实 NewAPI 请求绑定在这个 HTTP 响应上。

## 6. 任务状态机

任务级状态：

```text
queued
  -> running
  -> succeeded
  -> partial_failed
  -> failed
  -> cancelled
  -> interrupted
```

阶段 `stage`：

```text
queued
preparing
submitting
waiting_upstream
downloading
saving
succeeded
partial_failed
failed
cancelled
interrupted
```

子结果状态：

```text
pending
running
succeeded
failed
cancelled
```

关键规则：

- `succeeded`：所有图片成功落盘。
- `partial_failed`：至少一张成功落盘，至少一张失败。
- `failed`：没有任何图片成功。
- `interrupted`：程序停止时任务还在 running，无法确认上游结果。
- `cancelled`：用户取消，排队任务直接取消，运行中任务尽力取消。

## 7. SSE 假流式设计

路由：

```text
GET /api/background-tasks/:id/events
```

事件：

```text
snapshot   # 连接建立时发送完整任务快照
progress   # 状态、阶段、百分比、当前文案
result     # 某一张图片成功或失败
heartbeat  # 固定心跳，证明观察连接还活着
done       # 任务进入最终状态
error      # 仅表示 SSE 层或任务失败状态，不表示连接失败
```

等待上游时的假进度：

```text
0-5%     queued/preparing
5-15%    submitting
15-90%   waiting_upstream，按预计 10-15 分钟缓慢推进
90-97%   downloading/saving
100%     succeeded/partial_failed/failed
```

前端断开后：

- 后端不取消任务。
- 前端重连先收 `snapshot`。
- 前端也可以 `GET /api/background-tasks/:id` 拉快照。

## 8. NewAPI 请求闭环

首版只实现 OpenAI 兼容图片接口：

```text
POST {baseUrl}/images/generations
POST {baseUrl}/images/edits
```

请求策略：

- 后端按 `count` 拆成多次单图请求，`n=1`。
- 每个单图请求有独立超时，默认 600 秒。
- 全局并发和单任务并发都有限制。
- 上游返回支持三种解析：
  - `Content-Type: image/*`
  - JSON `data[].b64_json`
  - JSON `data[].url` 后端再下载
- 下载 URL 也必须在后端完成，然后落盘。

不做的事情：

- 不让前端拿远程图片 URL 直接展示。
- 不把图床上传作为任务成功条件。
- 不在日志输出 API Key、完整 Authorization、完整请求体。

## 9. 持久化闭环

首版 JSON 文件：

```text
data/config.local.json
data/jobs.json
data/uploads/
outputs/YYYY-MM-DD/
```

写入规则：

- 任务状态更新后原子写 `jobs.json`。
- 图片文件先写临时文件，再 rename 成正式文件。
- 配置保存不返回明文 Key。

启动恢复规则：

```text
queued       -> 重新入队
running      -> interrupted
preparing    -> queued 或 interrupted，取决于是否已经提交上游
succeeded    -> 保持
partial_failed -> 保持
failed       -> 保持
cancelled    -> 保持
```

为了避免重复扣费：只要任务可能已经提交给同步上游，就不自动重试。

## 10. 手机和 App 闭环

当前确定范围：

- 首版是响应式网站，由 Go 后端托管页面。
- 电脑浏览器访问同源页面最稳定。
- 桌面 App 后续优先 Wails：Go 后端和 React 前端可以直接复用。

需要后续单独决策的范围：

- 如果手机只是访问电脑上的 Go 服务：需要 LAN 模式和访问口令。
- 如果要自包含手机 App：Capacitor 不能直接嵌入 Go 后端，需要：
  1. 用 App 原生插件实现 NewAPI 请求和本地存储；或
  2. 手机 App 仍连接一个外部/局域网 Go 服务；或
  3. 评估 gomobile，但工程复杂度明显更高。

所以“适配手机页面”和“手机端自包含后端”不能混为一个需求。

## 11. 建议实现顺序

1. 配置模块：内置 NewAPI URL、Key 保存、`GET/POST /api/config`。
2. Store 抽象和 JSON Store：任务、配置、统计持久化。
3. Output 模块：图片落盘与安全读取。
4. Events Hub：SSE 订阅、心跳、snapshot。
5. Jobs 队列：创建、状态机、假进度、取消、重试。
6. NewAPI Client：文生图、图生图、结果解析。
7. API 路由：任务列表、单任务、events、images。
8. 前端 API client/hooks：只走同源 `/api`。
9. 响应式 UI：桌面三栏、手机单列/抽屉。
10. 打包：Go 托管 `web/dist`，再考虑 Wails。

## 12. 仍需用户最终确认的开放点

这几个点不影响先写主链路，但最好后续确认：

1. 内置 NewAPI Base URL 最终值是什么，是否只允许环境变量覆盖。
2. 是否需要 LAN 模式给手机浏览器访问电脑服务。
3. 手机 App 是“访问本机/局域网 Go 服务”，还是“自包含 App”。
4. 是否需要多模型选择，还是固定一个默认模型。
5. 图生图第一版就做；参考图上传到本机空间目录，不走第三方图床。

