# 外部 Bearer API 设计

本文设计一套面向 SDK 和自动化程序调用的外部 API。目标是在不打乱现有网页工作台的前提下，增加独立的 Bearer API Key、任务创建接口和轮询式任务查询接口。

本文以单节点部署为第一版约束，不引入复杂限流、计费、多节点队列、对象存储或 webhook。

## 1. 目标

当前 Lyra Image Workbench 已经具备稳定的后台任务模型：

```text
浏览器
  -> 同源 /api
  -> Go 后端创建后台任务
  -> Go worker 请求 NewAPI / OpenAI 兼容网关
  -> 图片保存到 outputs/
  -> 前端通过查询或 SSE 观察任务状态
```

外部 API 需要在此基础上支持：

```text
用户网页登录
  -> 在设置中确认保存上游 API Key 到云端空间
  -> 生成 Lyra Bearer API Key
  -> SDK 使用 Authorization: Bearer lyra_sk_xxx
  -> POST /v1/image-tasks 创建任务
  -> 返回 task id
  -> SDK 轮询 GET /v1/image-tasks/{id}
  -> 任务成功后下载 /v1/image-tasks/{id}/images/{index}
```

核心原则：

- Bearer API Key 只代表调用 Lyra 的权限，不等于上游 NewAPI / OpenAI 兼容网关的 Key。
- 上游 Key 仍然只从用户空间的云端配置中读取。
- 生成 Bearer API Key 前，用户空间必须已经保存可用的云端上游 Key。
- 创建任务时仍要再次检查云端上游 Key 是否存在，避免用户生成 Bearer 后又清空上游 Key。
- 第一版只考虑单节点部署，继续复用现有 `jobs.Manager`、本地 JSON 任务状态和本机 `outputs/`。

## 2. 非目标

第一版不做：

- 计费和额度系统。
- 复杂限流和风控策略。
- 多节点队列、分布式 worker 或跨实例任务恢复。
- webhook 回调。
- 外部对象存储、CDN 或签名 URL。
- OAuth、团队空间或细粒度 scope。
- OpenAI API 完全兼容层。

可以保留现有 `count`、`concurrency`、上传大小和参考图数量限制，避免外部 API 绕过单节点保护。

## 3. 术语

| 术语 | 含义 |
| --- | --- |
| 上游 Key | 用户用于请求 NewAPI / OpenAI 兼容网关的真实 API Key，目前保存在用户空间配置里。 |
| Bearer API Key | Lyra 生成给 SDK 使用的调用凭证，格式类似 `lyra_sk_xxx`。 |
| 用户空间 | 现有账号绑定的 `storageToken`，用于隔离任务、上传图、配置和输出图。 |
| 外部 API | 新增 `/v1/*` 路由，只接受 Bearer 鉴权，面向程序调用。 |
| 网页 API | 现有 `/api/*` 路由，主要服务 React 工作台和浏览器登录态。 |

## 4. 总体架构

```text
SDK / Script
  |
  | Authorization: Bearer lyra_sk_xxx
  v
Go HTTP Server
  |
  | /v1 Bearer 鉴权
  |   -> apiKey hash lookup
  |   -> storageToken
  |   -> 检查用户空间云端上游 Key
  v
jobs.Manager
  |
  | 单节点内存队列 + 后台 worker
  v
NewAPI / OpenAI-compatible gateway
  |
  v
outputs/{storageToken}/YYYY-MM-DD/*.png
```

网页端新增一个开发者 API Key 管理区：

```text
React 设置页
  -> /api/developer/api-keys
  -> apikeys.Store
  -> data/api_keys.json
```

外部 API 与网页 API 分离：

- `/api/*` 继续使用 cookie/session 登录态。
- `/v1/*` 只使用 `Authorization: Bearer ...`。
- `/v1/*` 不接受运行时上游 Key 请求头。

## 5. 上游 Key 门槛

生成 Bearer API Key 前必须检查当前用户空间是否已经配置云端上游 Key。

判断规则：

```text
provider=image-2:
  spaceconfig.Config.CloudAPIKeyEnabled == true
  且 APIKey 非空

provider=banana:
  spaceconfig.Config.CloudBananaAPIKeyEnabled == true
  且 BananaAPIKey 非空
```

第一版可以先采用更简单规则：

```text
当前用户空间至少有一个云端上游 Key
```

创建任务时按实际 `provider` 再做精确检查：

```text
POST /v1/image-tasks provider=image-2
  -> 必须有 image-2 云端 Key

POST /v1/image-tasks provider=banana
  -> 必须有 banana 云端 Key
```

如果没有上游 Key，返回：

```json
{
  "ok": false,
  "code": "UPSTREAM_KEY_REQUIRED",
  "message": "请先在设置中确认保存云端上游 Key，再使用外部 API 创建任务"
}
```

## 6. Bearer API Key 数据模型

新增模块：

```text
internal/apikeys/store.go
```

建议存储文件：

```text
data/api_keys.json
```

文件结构：

```json
{
  "keys": [
    {
      "id": "ak_20260625143000_abcd1234",
      "name": "local-sdk",
      "prefix": "lyra_sk_abcd1234",
      "hash": "sha256_hex",
      "username": "Alice",
      "storageToken": "64-byte-space-token",
      "createdAt": "2026-06-25T14:30:00+08:00",
      "lastUsedAt": "2026-06-25T15:20:00+08:00",
      "revokedAt": ""
    }
  ]
}
```

字段说明：

| 字段 | 说明 |
| --- | --- |
| `id` | API Key 记录 ID，用于删除和管理。 |
| `name` | 用户自定义备注名，例如 `local-sdk`。 |
| `prefix` | 明文 key 的短前缀，只用于列表展示和排查。 |
| `hash` | 明文 Bearer key 的 SHA-256 hash，服务端不保存明文。 |
| `username` | 创建者用户名，用于后台排查。 |
| `storageToken` | 绑定的用户空间，外部 API 只可访问该空间任务和输出。 |
| `createdAt` | 创建时间。 |
| `lastUsedAt` | 最近使用时间，可异步或在认证成功后更新。 |
| `revokedAt` | 删除或吊销时间。第一版删除也可以直接移除记录。 |

明文 key 格式：

```text
lyra_sk_<base64url_random_32_bytes>
```

安全要求：

- 明文只在创建成功响应中返回一次。
- 服务端只保存 hash。
- 列表接口只返回 `id`、`name`、`prefix`、`createdAt`、`lastUsedAt`。
- 删除后立即失效。

## 7. 网页管理接口

这些接口服务设置页，继续使用现有用户登录态。

```text
GET    /api/developer/api-keys
POST   /api/developer/api-keys
DELETE /api/developer/api-keys/{id}
```

### 7.1 列出 API Key

```http
GET /api/developer/api-keys
Cookie: lyra_user_session=...
```

响应：

```json
{
  "ok": true,
  "apiKeys": [
    {
      "id": "ak_20260625143000_abcd1234",
      "name": "local-sdk",
      "prefix": "lyra_sk_abcd1234",
      "createdAt": "2026-06-25T14:30:00+08:00",
      "lastUsedAt": "2026-06-25T15:20:00+08:00"
    }
  ]
}
```

### 7.2 创建 API Key

```http
POST /api/developer/api-keys
Content-Type: application/json
Cookie: lyra_user_session=...

{
  "name": "local-sdk"
}
```

成功响应：

```json
{
  "ok": true,
  "apiKey": {
    "id": "ak_20260625143000_abcd1234",
    "name": "local-sdk",
    "prefix": "lyra_sk_abcd1234",
    "createdAt": "2026-06-25T14:30:00+08:00"
  },
  "secret": "lyra_sk_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
}
```

失败响应：

```json
{
  "ok": false,
  "code": "UPSTREAM_KEY_REQUIRED",
  "message": "请先在设置中确认保存云端上游 Key，再生成 Bearer API Key"
}
```

### 7.3 删除 API Key

```http
DELETE /api/developer/api-keys/{id}
Cookie: lyra_user_session=...
```

响应：

```json
{
  "ok": true
}
```

删除时必须校验该 key 属于当前登录用户空间。

## 8. 外部 API 鉴权

所有 `/v1/*` 路由要求：

```http
Authorization: Bearer lyra_sk_xxx
```

认证流程：

```text
读取 Authorization
  -> 校验 Bearer 格式
  -> SHA-256 hash
  -> apikeys.Store 查找未删除记录
  -> 得到 storageToken
  -> 将 storageToken 注入 request context
  -> 后续 handler 使用该 storageToken 调 jobs.Manager
```

认证失败：

```json
{
  "ok": false,
  "code": "UNAUTHORIZED",
  "message": "Bearer API Key 缺失或无效"
}
```

## 9. 外部 API 路由

第一版路由：

```text
POST /v1/image-tasks
GET  /v1/image-tasks/{id}
GET  /v1/image-tasks/{id}/images/{index}
POST /v1/image-tasks/{id}/cancel
```

暂不开放：

- 任务删除。
- 收藏。
- PiXhost 上传。
- SSE。
- webhook。

SDK 第一版只做轮询。

## 10. 创建任务

```http
POST /v1/image-tasks
Authorization: Bearer lyra_sk_xxx
Content-Type: application/json
```

请求体：

```json
{
  "provider": "image-2",
  "model": "gpt-image-2",
  "mode": "text-to-image",
  "prompt": "a cinematic portrait of a girl in neon rain",
  "ratio": "1:1",
  "resolution": "standard",
  "quality": "auto",
  "outputFormat": "png",
  "count": 1,
  "concurrency": 1
}
```

字段复用现有 `jobs.CreateRequest`，但 `/v1` 必须忽略或拒绝以下字段：

```text
apiKey
bananaApiKey
```

也不读取以下运行时上游 Key 请求头：

```text
X-Image-Workbench-API-Key
X-Image-Workbench-Banana-API-Key
```

创建流程：

```text
Bearer 鉴权成功
  -> 得到 storageToken
  -> 解析 JSON
  -> 清空 RuntimeSecrets
  -> 根据 provider 检查用户空间云端上游 Key
  -> 调 jobs.Manager.Create(storageToken, req)
  -> 返回任务公开视图
```

成功响应：

```json
{
  "ok": true,
  "task": {
    "id": "img_20260625143000_abcd1234abcd1234",
    "provider": "image-2",
    "model": "gpt-image-2",
    "mode": "text-to-image",
    "status": "queued",
    "stage": "queued",
    "progress": 0,
    "createdAt": "2026-06-25T14:30:00+08:00",
    "updatedAt": "2026-06-25T14:30:00+08:00"
  }
}
```

## 11. 查询任务

```http
GET /v1/image-tasks/{id}
Authorization: Bearer lyra_sk_xxx
```

成功响应：

```json
{
  "ok": true,
  "task": {
    "id": "img_20260625143000_abcd1234abcd1234",
    "provider": "image-2",
    "model": "gpt-image-2",
    "mode": "text-to-image",
    "prompt": "a cinematic portrait of a girl in neon rain",
    "ratio": "1:1",
    "resolution": "standard",
    "quality": "auto",
    "outputFormat": "png",
    "size": "1024x1024",
    "count": 1,
    "concurrency": 1,
    "status": "succeeded",
    "stage": "succeeded",
    "progress": 100,
    "error": "",
    "results": [
      {
        "index": 0,
        "ok": true,
        "status": "succeeded",
        "imageUrl": "/v1/image-tasks/img_20260625143000_abcd1234abcd1234/images/0",
        "mime": "image/png",
        "bytes": 123456,
        "revisedPrompt": "",
        "elapsedMs": 48231
      }
    ],
    "createdAt": "2026-06-25T14:30:00+08:00",
    "updatedAt": "2026-06-25T14:31:00+08:00",
    "startedAt": "2026-06-25T14:30:02+08:00",
    "finishedAt": "2026-06-25T14:30:50+08:00"
  }
}
```

权限规则：

- 只能查询 Bearer API Key 绑定空间内的任务。
- 如果任务不存在或不属于该空间，统一返回 `TASK_NOT_FOUND`。

SDK 轮询建议：

```text
queued/running:
  继续轮询

succeeded:
  下载全部 ok=true 的图片

partial_failed:
  下载 ok=true 的图片，并将失败 result 暴露给调用方

failed/cancelled/interrupted:
  停止轮询并抛出错误
```

默认轮询间隔：

```text
2 秒
```

默认最长等待：

```text
15 分钟
```

## 12. 下载图片

```http
GET /v1/image-tasks/{id}/images/{index}
Authorization: Bearer lyra_sk_xxx
```

行为：

- 校验 Bearer API Key。
- 校验任务属于该 key 绑定空间。
- 校验 `index` 存在且结果成功。
- 复用现有输出文件解析逻辑，返回图片二进制。

响应头：

```text
Content-Type: image/png
Cache-Control: private, max-age=86400
```

失败：

```json
{
  "ok": false,
  "code": "TASK_IMAGE_NOT_FOUND",
  "message": "任务图片不存在"
}
```

## 13. 取消任务

```http
POST /v1/image-tasks/{id}/cancel
Authorization: Bearer lyra_sk_xxx
```

行为：

- 校验 Bearer API Key。
- 校验任务属于该 key 绑定空间。
- 调用 `jobs.Manager.Cancel(storageToken, id)`。

响应：

```json
{
  "ok": true,
  "task": {
    "id": "img_...",
    "status": "cancelled",
    "stage": "cancelled",
    "progress": 100
  }
}
```

说明：

- 排队任务可以直接取消。
- 运行中任务为尽力取消，不能保证上游不计费。

## 14. 图生图设计

第一版建议分两步走。

### 14.1 第一阶段

先开放文生图：

```text
mode=text-to-image
```

原因：

- 外部 API 的参考图上传契约需要单独稳定下来。
- 当前网页端已有 `/api/uploads/reference`，但外部 API 需要 Bearer 鉴权版本。
- 先跑通 Bearer、任务创建、轮询和图片下载是最短闭环。

### 14.2 第二阶段

新增：

```text
POST /v1/reference-images
GET  /v1/reference-images
DELETE /v1/reference-images/{id}
```

上传接口使用 `multipart/form-data`：

```http
POST /v1/reference-images
Authorization: Bearer lyra_sk_xxx
Content-Type: multipart/form-data
```

响应：

```json
{
  "ok": true,
  "uploads": [
    {
      "id": "ref_abcdef123456abcdef123456",
      "originalName": "input.png",
      "mime": "image/png",
      "size": 12345,
      "createdAt": "2026-06-25T14:30:00+08:00"
    }
  ]
}
```

创建图生图任务：

```json
{
  "provider": "image-2",
  "mode": "image-to-image",
  "prompt": "turn this into a cinematic poster",
  "uploadIds": ["ref_abcdef123456abcdef123456"],
  "ratio": "1:1",
  "resolution": "standard",
  "count": 1,
  "concurrency": 1
}
```

上传限制沿用现有规则：

```text
最多 8 张参考图
单张最大 12MB
单次请求最大 50MB
支持 PNG / JPG / WEBP
```

## 15. 错误码

第一版建议稳定以下错误码：

| HTTP | Code | 说明 |
| --- | --- | --- |
| 400 | `BAD_JSON` | 请求体不是有效 JSON。 |
| 400 | `UPSTREAM_KEY_REQUIRED` | 用户空间没有对应 provider 的云端上游 Key。 |
| 400 | `TASK_CREATE_FAILED` | 创建任务失败，message 包含具体原因。 |
| 401 | `UNAUTHORIZED` | Bearer 缺失、格式错误或无效。 |
| 403 | `API_KEY_REVOKED` | Bearer 已删除或吊销。第一版可并入 `UNAUTHORIZED`。 |
| 404 | `TASK_NOT_FOUND` | 任务不存在或不属于当前 Bearer 绑定空间。 |
| 404 | `TASK_IMAGE_NOT_FOUND` | 图片不存在或结果未成功。 |
| 405 | `METHOD_NOT_ALLOWED` | 路由存在但方法不支持。 |
| 500 | `INTERNAL_ERROR` | 服务端内部错误。 |

错误响应统一格式：

```json
{
  "ok": false,
  "code": "TASK_NOT_FOUND",
  "message": "任务不存在"
}
```

## 16. SDK 行为

第一版 SDK 可以只做轮询，不需要 SSE。

伪代码：

```text
client = LyraClient(baseURL, bearerKey)

task = client.createImageTask({
  prompt,
  ratio,
  resolution,
  count
})

finalTask = client.waitForTask(task.id, {
  intervalMs: 2000,
  timeoutMs: 15 * 60 * 1000
})

for result in finalTask.results:
  if result.ok:
    client.downloadImage(finalTask.id, result.index, outputPath)
```

SDK 的终止状态：

```text
succeeded:
  返回任务和图片信息

partial_failed:
  返回任务，同时标记 partial

failed/cancelled/interrupted:
  抛出任务失败错误

timeout:
  抛出 SDK 等待超时，但不取消服务端任务
```

## 17. 安全边界

必须做：

- Bearer API Key 明文只展示一次。
- 服务端只存 hash。
- `/v1` 不接受用户传入的上游 Key。
- 创建任务时强制从用户空间云端配置取上游 Key。
- 生成 Bearer 前检查用户已经保存云端上游 Key。
- 创建任务时再次检查对应 provider 的云端上游 Key。
- 图片下载必须校验任务属于当前 Bearer 绑定空间。
- 删除 Bearer 后立即失效。

可以暂缓：

- 每 key scope。
- IP allowlist。
- 请求频率限制。
- 额度和计费。
- 审计日志详情页。
- webhook 签名。

建议保留：

- 现有 `count` 最大 12 的限制。
- 现有 `concurrency` clamp。
- 现有参考图数量和大小限制。
- 上游请求超时配置。

## 18. 代码落点

新增：

```text
internal/apikeys/store.go
internal/api/developer_api_keys.go
internal/api/v1_auth.go
internal/api/v1_image_tasks.go
```

修改：

```text
cmd/local-server/main.go
internal/api/router.go
web/src/api/developerKeys.ts
web/src/components/SettingsPanel.tsx
```

可选新增：

```text
docs/EXTERNAL_API_DESIGN.md
docs/EXTERNAL_API_USAGE.md
```

## 19. 实现顺序

1. 新增 `internal/apikeys.Store`，支持创建、列表、删除、Bearer hash 查找。
2. 在 `cmd/local-server/main.go` 装配 `apikeys.Store`。
3. 在 `api.Dependencies` 加入 `APIKeys`。
4. 新增 `/api/developer/api-keys` 管理接口。
5. 设置页增加开发者 API Key 管理区。
6. 新增 `/v1` Bearer 鉴权 middleware。
7. 新增 `/v1/image-tasks` 创建、查询、取消和图片下载。
8. 文生图闭环跑通后，再考虑 `/v1/reference-images` 和图生图。
9. 补充 SDK 示例和使用文档。

## 20. 测试清单

Go 单元测试建议覆盖：

- 未登录不能调用 `/api/developer/api-keys`。
- 未保存云端上游 Key 时，不能生成 Bearer API Key。
- 保存云端上游 Key 后，可以生成 Bearer API Key。
- 创建 Bearer 后，列表接口不返回明文 secret。
- Bearer 明文只在创建响应出现一次。
- 删除 Bearer 后，`/v1` 认证失败。
- 无 Bearer 调 `/v1/image-tasks` 返回 `UNAUTHORIZED`。
- 错误 Bearer 调 `/v1/image-tasks` 返回 `UNAUTHORIZED`。
- 有效 Bearer 可以创建任务。
- `/v1/image-tasks` 请求体中的 `apiKey`、`bananaApiKey` 不被使用。
- 清空云端上游 Key 后，旧 Bearer 创建任务返回 `UPSTREAM_KEY_REQUIRED`。
- Bearer 只能查询自己空间里的任务。
- Bearer 不能下载其他空间任务的图片。
- 图片不存在时返回 `TASK_IMAGE_NOT_FOUND`。

手动验收建议：

```text
1. 网页登录。
2. 设置中保存云端上游 Key。
3. 创建 Bearer API Key，复制明文。
4. 用 curl 调 POST /v1/image-tasks。
5. 轮询 GET /v1/image-tasks/{id}。
6. 成功后下载 GET /v1/image-tasks/{id}/images/0。
7. 删除 Bearer API Key。
8. 再次调用 /v1，确认返回 UNAUTHORIZED。
```

## 21. curl 示例

创建任务：

```bash
curl -X POST "http://127.0.0.1:8787/v1/image-tasks" \
  -H "Authorization: Bearer lyra_sk_xxx" \
  -H "Content-Type: application/json" \
  -d '{
    "provider": "image-2",
    "mode": "text-to-image",
    "prompt": "a cinematic portrait of a girl in neon rain",
    "ratio": "1:1",
    "resolution": "standard",
    "quality": "auto",
    "outputFormat": "png",
    "count": 1,
    "concurrency": 1
  }'
```

查询任务：

```bash
curl "http://127.0.0.1:8787/v1/image-tasks/img_xxx" \
  -H "Authorization: Bearer lyra_sk_xxx"
```

下载图片：

```bash
curl "http://127.0.0.1:8787/v1/image-tasks/img_xxx/images/0" \
  -H "Authorization: Bearer lyra_sk_xxx" \
  -o result.png
```

## 22. 后续可扩展项

当第一版稳定后，可以逐步增加：

- `/v1/reference-images` 图生图上传。
- webhook 回调。
- 简单按 key 的并发上限。
- 每日任务统计。
- OpenAPI 文档。
- TypeScript SDK。
- Python SDK。
- 对象存储结果落盘。
- 多节点队列和 worker。

这些能力不应阻塞第一版 Bearer API 闭环。

## 23. 站内/外部文档同步口径

外部 GitHub 文档仓库不在当前工作区，无法由本次补丁直接提交。以下 Markdown 与站内 `ApiDocsPage` 的 AI 提示词/同步块保持同一口径，可直接复制到外部文档仓库。

# LyAi Image Generation API 快速接入口径

> 本段可同步到外部文档仓库，和站内 API 文档保持同一口径。

## 基础信息

- Base URL: `https://ai-image.ailinyu.de`
- 注册和配置站点: `https://ai-image.ailinyu.de/`
- 认证: 所有 `/v1/*` 请求都带 `Authorization: Bearer <API_KEY>`。
- Bearer Key 格式: `lyra_sk_...`，只代表调用 Lyra 的权限，不是上游 NewAPI/OpenAI 兼容网关 Key。
- 用户需要先在网页端注册登录，在设置页保存对应 provider 的云端上游 Key，然后生成 Bearer API Key。
- 不要把 Bearer Key 写死在前端代码里；SDK/脚本优先读取环境变量 `LYRA_API_KEY`。

## 创建任务

推荐 SDK 默认使用 OpenAI 兼容创建端点：

```http
POST /v1/images/generations
Authorization: Bearer lyra_sk_xxx
Content-Type: application/json
```

示例请求体：

```json
{
  "model": "gpt-image-2",
  "prompt": "A clean product photo of a translucent smart speaker on a stone pedestal",
  "size": "1024x1024",
  "quality": "auto",
  "output_format": "png",
  "n": 1
}
```

也可以使用 Lyra 原生端点：

```http
POST /v1/image-tasks
Authorization: Bearer lyra_sk_xxx
Content-Type: application/json
```

原生端点请求体示例：

```json
{
  "provider": "image-2",
  "model": "gpt-image-2",
  "mode": "text-to-image",
  "prompt": "A clean product photo of a translucent smart speaker on a stone pedestal",
  "ratio": "1:1",
  "resolution": "standard",
  "quality": "auto",
  "outputFormat": "png",
  "count": 1,
  "concurrency": 1
}
```

创建成功响应：

```json
{
  "ok": true,
  "taskId": "img_...",
  "consumedCredits": 1,
  "task": {
    "id": "img_...",
    "status": "queued",
    "progress": 0,
    "results": []
  }
}
```

## 参数说明

- `prompt`: 必填，生成提示词。
- `model`: 可选。`image-2` 会使用服务端默认 `gpt-image-2`；`banana` 可传服务端支持的 banana 模型 ID，不传时使用默认 banana 模型。
- `provider`: 可选，支持 `image-2`/`gpt-image-2`/`image2` 或 `banana`/`banana-nano`/`nano-banana`。
- `size`: OpenAI 兼容字段，可传 `auto`、`1024x1024`、`1024x1536`、`1536x1024`、`768x1024`、`1024x768`、`1008x1792`、`1792x1008`，以及 2K/4K 对应尺寸；服务端会映射为 `ratio` + `resolution`。
- `ratio`: 可选，`auto`、`1:1`、`2:3`、`3:2`、`3:4`、`4:3`、`9:16`、`16:9`。
- `resolution`: 可选，`auto`、`standard`、`2k`、`4k`。
- `quality`: 可选，`auto`、`low`、`medium`、`high`；未知值归一为 `auto`。
- `output_format`/`outputFormat`: 可选，`png`、`jpeg`/`jpg`、`webp`、`auto`；未知值归一为 `png`。
- `n`/`count`: 可选，生成张数，服务端归一到 1-24。
- `concurrency`: 可选，最小 1。
- 外部 API 第一版仅支持 `text-to-image`；`image-to-image` 参考图上传仍走网页工作台，外部 Bearer 版本后续再扩展。
- `/v1/*` 不读取 `apiKey`、`bananaApiKey`、`X-Image-Workbench-API-Key` 或 `X-Image-Workbench-Banana-API-Key` 作为上游 Key。

## 轮询任务

```http
GET /v1/image-tasks/{taskId}
Authorization: Bearer lyra_sk_xxx
```

SDK 轮询逻辑：

- 每 2-5 秒轮询一次；默认最长等待建议 15 分钟。
- `queued`、`running`: 继续轮询。
- `succeeded`: 停止轮询，下载所有 `task.results` 中 `ok=true` 的结果。
- `partial_failed`: 停止轮询，下载所有 `ok=true` 的结果，同时把失败 result 暴露给调用方。
- `failed`、`cancelled`、`interrupted`: 停止轮询并抛出任务失败错误。

任务结果中的 `imageUrl` 是相对路径，推荐仍用下载接口并附带 Bearer 认证。

## 下载结果

```http
GET /v1/image-tasks/{taskId}/images/{index}
Authorization: Bearer lyra_sk_xxx
```

`index` 来自 `task.results[].index`，从 0 开始。只下载 `ok=true` 的结果；响应 body 是图片二进制，按 `png`/`jpeg`/`webp` 写入文件。

## 取消任务

```http
POST /v1/image-tasks/{taskId}/cancel
Authorization: Bearer lyra_sk_xxx
```

排队任务可取消；运行中任务为尽力取消，不能保证上游不计费。

## 错误码

- `400 BAD_JSON`: 请求体不是有效 JSON。
- `400 UPSTREAM_KEY_REQUIRED`: 用户空间没有对应 provider 的云端上游 Key。
- `400 TASK_CREATE_FAILED`: 创建任务失败，message 包含参数或上游失败原因。
- `401 UNAUTHORIZED`: Bearer 缺失、格式错误或无效。
- `402 USER_CREDITS_NOT_ENOUGH`: 账号生成次数不足。
- `404 TASK_NOT_FOUND`: 任务不存在或不属于当前 Bearer Key。
- `404 TASK_IMAGE_NOT_FOUND`: 图片不存在、结果未成功或不属于当前 Bearer Key。
- `429 AUTH_RATE_LIMITED`: 无效 Bearer 尝试过多，请稍后重试。
- `500 INTERNAL_ERROR`: 服务端内部错误。

错误响应统一为：

```json
{
  "ok": false,
  "code": "TASK_NOT_FOUND",
  "message": "任务不存在"
}
```
