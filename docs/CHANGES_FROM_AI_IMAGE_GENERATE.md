# 相较于 AI-Image-generate 的更新说明

本文对比对象：

- 原项目：[`y08lin4/AI-Image-generate`](https://github.com/y08lin4/AI-Image-generate)，检查提交 `207d10c`。
- 当前项目：[`y08lin4/lyra-image-workbench`](https://github.com/y08lin4/lyra-image-workbench)，检查提交 `4cf3afa`。

当前项目不是对原项目的简单换皮，而是把原来偏 **Cloudflare Worker / 浏览器侧配置** 的生图工具，重构成偏 **本机或私有服务器稳定运行** 的生图工作台。核心变化是：前端只负责提交任务和观察状态，所有上游请求、Key 保存、任务恢复、图片落盘都交给 Go 后端处理。

---

## 1. 总体定位变化

| 维度 | AI-Image-generate | lyra-image-workbench |
| --- | --- | --- |
| 部署目标 | Cloudflare Worker + 前端静态站点 | 本机 / VPS / 宝塔 / Linux systemd 私有部署 |
| 后端形态 | Cloudflare Worker，配合 D1 / Workflows | Go 后端，托管前端和同源 `/api` |
| 前端职责 | 保存 API URL / API Key，可直连或走 Worker | 不保存上游 URL，不直连 NewAPI，只请求同源后端 |
| 稳定性目标 | 通过 Worker 流式代理、Workflow 后台任务缓解断流 | 后端任务队列独立执行，前端断开不影响任务 |
| 数据位置 | 浏览器 IndexedDB、Cloudflare D1、PiXhost | 本机 `data/`、`outputs/`，可选 PiXhost |
| 适用场景 | 轻量云端生图页面、Cloudflare 生态 | 长时间生成、私有 Key、内网 NewAPI、服务器持续运行 |

---

## 2. 后端架构更新

### 2.1 从 Worker 迁移为 Go 后端

原项目的主要后端逻辑集中在 `worker/index.ts`：

```text
浏览器 -> Cloudflare Worker -> 上游图片接口
```

当前项目改成：

```text
浏览器 -> Go 同源 /api -> Go 后台任务 -> NewAPI / CLIProxyAPI -> outputs 本机落盘
```

更新点：

- 新增 Go 服务入口 `cmd/local-server`。
- 后端模块拆分到 `internal/`：
  - `api`：HTTP 路由和 JSON/SSE 响应。
  - `jobs`：任务队列、状态机、重试、取消、恢复。
  - `newapi`：OpenAI 兼容图片接口客户端。
  - `output`：生成图片落盘和安全读取。
  - `uploads`：图生图参考图本机保存。
  - `events`：SSE 事件中心。
  - `settings` / `spaceconfig` / `adminauth`：配置、空间、Admin 鉴权。

### 2.2 不再依赖 Cloudflare D1 / Workflows

原项目后台任务依赖：

```text
Cloudflare Workflows + D1 + PiXhost
```

当前项目改为：

```text
Go worker + 本机 JSON 状态 + 本机 outputs 图片文件
```

结果：

- 不需要 Wrangler、D1 迁移、Workflow 绑定。
- 私有服务器上可直接 systemd 常驻运行。
- 任务和图片可以随服务器磁盘备份迁移。

---

## 3. 10 分钟稳定性闭环更新

原项目需要面对 Cloudflare Worker / 浏览器生命周期限制，README 中也明确提到 `524` 和 CORS 场景。

当前项目围绕“生成过程 10 分钟也不失败”重做了闭环：

- 创建任务接口快速返回，不等待上游出图。
- 后端任务不继承前端 HTTP request context。
- SSE 只是观察通道，断开、刷新、手机锁屏不会取消任务。
- 默认上游请求超时为 `600s`，可在 `/admin` 调整。
- 多图任务按单图请求拆分，逐张保存，允许 `partial_failed`。
- 程序重启时：
  - `queued` 可恢复排队。
  - `running` 标记为 `interrupted`，避免重复扣费。
- 生成图片先落盘到 `outputs/`，PiXhost 只是可选上传。

---

## 4. API 与路由更新

当前项目保留了原项目的任务概念，但路由从 Worker 代理模式改成本机后端模式。

### 4.1 新增 / 强化的本机路由

```text
GET    /api/health
GET    /api/admin/auth
POST   /api/admin/auth/setup
POST   /api/admin/auth/session
DELETE /api/admin/auth/session
GET    /api/admin/config
POST   /api/admin/config

POST   /api/spaces/session
GET    /api/spaces/session
DELETE /api/spaces/session

GET    /api/config
POST   /api/config

POST   /api/uploads/reference
GET    /api/uploads/reference
DELETE /api/uploads/reference/:id

POST   /api/background-tasks
GET    /api/background-tasks
GET    /api/background-tasks/:id
GET    /api/background-tasks/:id/events
POST   /api/background-tasks/:id/retry
POST   /api/background-tasks/:id/cancel
GET    /api/background-tasks/:id/images/:index
POST   /api/background-tasks/:id/images/:index/pixhost

GET    /api/stats
GET    /outputs/{spaceToken}/{date}/{file}
```

### 4.2 移除或替代的原项目模式

原项目支持：

- Worker 流式代理：`POST /api/generate-stream`
- 浏览器直连上游
- Worker 后台任务
- Worker 图片代理

当前项目改为：

- 不再让浏览器直连上游。
- 不再让前端保存 NewAPI URL。
- 不再用 Cloudflare Worker 做主链路。
- 统一通过 Go 后端后台任务执行。

---

## 5. Key、空间和 Admin 安全更新

### 5.1 Key 保存位置变化

原项目：

- API URL / API Key 主要在浏览器本地设置中保存。
- Worker 不长期保存 API Key。

当前项目：

- codex-key / Banana 分组 Key 保存到当前个人空间的后端配置。
- 前端只显示 Key 是否已设置和掩码。
- 前端不把 Key 直接发给 NewAPI。
- 提示词助手复用后端保存的 codex-key。

### 5.2 增加 Admin 管理区

当前项目新增独立 `/admin`：

- 初次访问设置 Admin 管理密码。
- 后续访问需要登录。
- 可配置：
  - NewAPI 请求 URL。
  - 请求超时时间。
  - 对外访问域名。
- 模型 `gpt-image-2` 在 v1 固定展示，不让普通用户误改。

### 5.3 服务器部署默认更适合公网服务

当前项目现在默认监听：

```text
0.0.0.0:8787
```

并支持 systemd 常驻、宝塔/Nginx 反代和域名配置。

---

## 6. 模型能力更新

### 6.1 Image-2 主链路

继续支持：

- 文生图：`/images/generations`
- 图生图：`/images/edits`
- 模型固定为 `gpt-image-2`
- 比例、清晰度、质量、输出格式
- `image/*`、`b64_json`、`url` 三类返回解析

### 6.2 新增 Banana Nano 适配

当前项目新增 Banana 分组 Key 和 Banana 模型路由：

- URL 仍使用 Admin 里的 NewAPI URL。
- Key 使用单独的 Banana 分组 API Key。
- 比例和清晰度不是普通参数，而是路由到具体模型 ID。
- 内置模型包括：
  - `gemini-3.1-flash-image-preview`
  - `gemini-3.1-flash-image-preview-16x9-4k`
  - `gemini-3.1-flash-image-preview-9x16-4k`
  - `gemini-3.1-flash-image-preview-16x9-2k`
  - `gemini-3.1-flash-image-preview-9x16-2k`
  - `gemini-3.1-flash-image-preview-2k`
  - `gemini-3.1-flash-image-preview-4k`
  - `gemini-3.1-flash-image-preview-4x3-4k`
  - `gemini-3.1-flash-image-preview-4x3-2k`
  - `gemini-3.1-flash-image-preview-1x1-4k`
  - `gemini-3.1-flash-image-preview-3x4-2k`
  - `gemini-3.1-flash-image-preview-3x4-4k`
  - `gemini-3.1-flash-image-preview-1x1-2k`

---

## 7. 提示词助手更新

原项目主要聚焦生图工作流，没有内置完整的语言模型提示词助手。

当前项目新增基于 `gpt-5.5` 的提示词助手：

- 文字生成图片提示词。
- 图片还原提示词。
- 灵感模式。
- 历史会话。
- 继续对话修改提示词。
- 多版本提示词管理。
- 生成后选择应用到 Image-2 或 Banana。
- 助手已和生成页合并，生成/改提示词后可直接填入主输入框。

---

## 8. 图生图参考图更新

原项目支持多参考图上传，但主要围绕 Worker / PiXhost 链路。

当前项目改成本机参考图闭环：

- 参考图上传到当前空间的本机目录。
- 创建任务只传 `uploadIds`。
- 提示词按输入框内容原样提交，不再自动追加内置方向说明。
- 多张参考图按上传列表顺序提交。

结果图也可以一键“作为参考图”，自动切回图生图。

---

## 9. 结果、历史和图床更新

当前项目保留并强化了原项目的结果操作：

- 下载图片。
- 复制图片到剪贴板。
- 复制 URL。
- 作为图生图参考图。
- 全屏预览。
- 显示真实像素尺寸、实际宽高比、文件大小。
- 自动上传或手动上传到 PiXhost。

主要变化：

- 原图优先保存在服务器 `outputs/`。
- PiXhost 上传失败不影响本机结果保存。
- 历史记录以后端任务列表为主，不再依赖浏览器 IndexedDB 作为唯一历史来源。
- 队列页支持搜索、筛选、收藏、批量下载、批量删除。

---

## 10. UI / UX 更新

当前项目相较原项目做了更明显的工作台化改造：

- 从单页堆叠改为工作流标签：

```text
生成 / 结果 / 队列 / 设置
```

- 生成页采用分步流程：

```text
提示词 -> 模型与模式 -> 参考图 -> 图片规格 -> 数量与执行
```

- 提示词助手合并进生成页。
- 结果页以当前任务为中心，可复用参数、查看队列、重试失败任务。
- 队列页作为历史和异常处理中心。
- 设置页按 codex-key、Banana Key、默认生成设置、PiXhost 分组。
- 新增黑夜模式，可在登录页、工作台、Admin 页切换，并保存到浏览器。
- UI 更偏白色/紧凑/少阴影，同时适配手机单列布局。

---

## 11. 错误码和状态展示更新

原项目对常见 HTTP 错误有友好提示。

当前项目进一步统一状态和错误展示：

- 任务状态：

```text
queued / running / succeeded / partial_failed / failed / cancelled / interrupted
```

- SSE 事件：

```text
snapshot / progress / result / heartbeat / done / error
```

- 前端展示格式：

```text
中文原因 / 错误码 / 英文标识
```

例如：

```text
上游请求超时 / E_UPSTREAM_TIMEOUT / upstream_timeout
上游不支持当前参数 / E_PROVIDER_UNSUPPORTED_PARAM / provider_unsupported_parameter
当前后端不支持这个请求方法 / HTTP_405 / method_not_allowed
```

---

## 12. 部署与运维更新

原项目部署重点是 Cloudflare：

- Worker
- D1
- Workflows
- Wrangler

当前项目新增完整 Linux / 宝塔部署链路：

- Go 构建单二进制。
- Go 后端托管 `web/dist`。
- systemd 常驻运行。
- 默认监听 `0.0.0.0:8787`。
- Nginx / 宝塔反代到 `127.0.0.1:8787` 或直接开放端口。
- 私有 GitHub 仓库可通过 Deploy Key 拉取。
- 数据目录：

```text
data/       # 空间、Admin、任务、Key 配置
outputs/    # 生成结果图片
```

---

## 13. 功能取舍

当前项目不是“功能全部更多”，而是做了方向取舍。

### 当前项目新增或强化

- Go 后端主控上游请求。
- 10 分钟后台任务稳定性。
- 本机/服务器图片落盘。
- Admin 管理配置。
- Banana Nano 模型路由。
- gpt-5.5 提示词助手。
- 图生图多参考图上传。
- 黑夜模式。
- Linux / 宝塔 / systemd 私有部署。

### 当前项目不再主推

- 浏览器直连上游。
- Cloudflare Worker 作为主链路。
- D1 / Workflows 作为任务持久化。
- 只靠浏览器 IndexedDB 保存历史。
- 把 API URL / Key 保存在浏览器侧。

---

## 14. 一句话总结

`AI-Image-generate` 更像一个 Cloudflare Worker 生态下的轻量生图页面；`lyra-image-workbench` 则升级成一个可在本机或私有服务器长期运行的生图工作台：后端接管上游请求、任务队列、Key 管理、图片落盘和恢复能力，目标是让 10 分钟级别的文生图 / 图生图任务稳定完成。
