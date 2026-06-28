# Backend Module Review

审查日期：2026-06-27
范围：`internal/api`、`internal/jobs`、`internal/promptsquare`、`internal/retention`
目标：只读优先检查后端模块边界、职责混杂、重复 DTO、接口/存储耦合；不触碰前端，不做大行为改动。

## 1. 本轮结论

本轮未修改后端源码，仅新增本文档作为审查结果。

原因是当前工作树中 `internal/api/prompt_square.go`、`internal/api/tasks.go`、`internal/jobs/manager.go`、`internal/jobs/types.go`、`internal/promptsquare/store.go`、`internal/retention/retention.go` 等目标文件已经存在未提交改动，并且 `docs/CURRENT_AGENT_TASKS.md` 记录仍有并行 agent 在广场、GIF、提示词库、提示词助手等区域工作。为避免覆盖或扩大冲突，本轮把安全的小拆分候选记录为后续建议，不直接改动业务代码。

整体边界可接受：HTTP、任务、广场、清理器四个包已经分开，清理器也已有路径守卫和广场保护索引。但仍有几个明显的轻量整理点，尤其是 `jobs.Manager` 过重、`jobs.Job` 同时承担持久化模型和 API 响应 DTO、以及 `api` 层为不同路由重复做结果投影和图片服务。

## 2. 当前模块边界

| 模块 | 当前职责 | 边界评价 |
| --- | --- | --- |
| `internal/api` | HTTP 路由、请求解析、认证上下文、错误响应、SSE、公开响应投影，向 `jobs.Manager`、各 store/service 分发调用。 | 作为传输层基本合理，但已有较多业务适配逻辑，例如任务扣费、`/v1` 兼容请求转换、广场 from-result 的参考图翻译、结果图公开 URL 改写。 |
| `internal/jobs` | 任务领域模型、任务 JSON 存储、队列 worker、状态机、上游 NewAPI 调用、结果保存、参考图快照、Pixhost 上传、事件发布、debug 日志。 | 包边界清楚，但 `manager.go` 职责过宽；`Manager` 直接持有多个 store/client，并通过 `m.store.spaces.SpaceDir` 触达存储细节。 |
| `internal/promptsquare` | 广场作品模型、点赞/每日榜/我的作品、投稿保存、结果图与参考图永久副本、`items.json` 持久化、广场图片解析。 | 已从 API 层独立出来，但 `store.go` 同时承载领域规则、JSON 持久化、图片复制和路径解析，文件偏大。 |
| `internal/retention` | 30 天清理调度、输出文件清理、上传参考图清理、任务参考快照清理、路径守卫、广场投稿保护。 | 独立后台清理器定位合理；当前直接依赖 `*jobs.Store` 与 `*promptsquare.Store`，适合后续改成小接口以降低耦合。 |
| `cmd/local-server` | 组合根：创建 config/settings/spaces/uploads/output/jobs/promptsquare/retention/api router。 | 依赖组装集中在入口处是合理的，未发现需要拆的高风险点。 |

## 3. 主要发现

### 3.1 职责混杂

1. `internal/jobs/manager.go` 是最大混合点。
   它同时处理任务创建校验、队列运行、上游请求、输出保存、Pixhost、参考图快照、取消/恢复、事件和 debug 日志。短期行为可保留，但后续维护成本会继续上升。

2. `internal/promptsquare/store.go` 的 store 边界偏宽。
   它既是领域 store，又负责 multipart 图片保存、从任务结果复制图片、参考图永久复制、图片 URL 解析和文件删除回滚。作为单包实现可以接受，但建议按文件拆分，保持同一个 package 与公开 API 不变。

3. `internal/api` 有少量业务逻辑上浮。
   `TaskHandler` 和 `V1ImageTaskHandler` 共用扣费 helper 是好事，但 `api` 仍需要理解 `jobs.Job` 的内部字段来做 public URL、result image serving、SSE payload 投影。后续可以先把这些投影函数集中到同包单文件。

### 3.2 重复 DTO 和响应投影

1. `jobs.Job` / `jobs.Result` 目前同时用于持久化、内部事件、Web API 响应、`/v1` API 响应。
   `jobs.PublicJob` 和 `jobs.PublicResult` 会清理敏感字段，但这仍让 API 契约紧贴内部持久化字段。

2. `/api/background-tasks` 与 `/v1/image-tasks` 有重复结果投影。
   `publicResult`、`publicV1Job`、`serveResultImage`、`serveV1ResultImage`、`resultByIndex` 与 `promptSquareResultByIndex` 都围绕同一组 `jobs.Result` 做轻微不同的 URL 和错误处理。

3. `/v1/image-tasks` 仍直接解码 `jobs.CreateRequest`。
   目前会清空 `RuntimeSecrets` 并限制 `text-to-image`，风险可控；但外部 API 长期最好有独立 request DTO，避免未来给 `jobs.CreateRequest` 增加内部字段时被外部路由自动接受。

### 3.3 接口和存储耦合

1. `jobs.Manager` 通过 `m.store.spaces.SpaceDir` 获取空间目录。
   这让 Manager 依赖 `Store` 的内部字段形状，而不只是依赖任务存取接口。后续可把参考图快照存储抽为 helper 或向 `Store` 增加窄方法，先减少字段穿透。

2. `retention.Config` 直接接收 `*jobs.Store` 和 `*promptsquare.Store`。
   清理器真实需要的是 `AllSpacesJobs()` 与 `SourceTaskIDs()` 这两个能力。改成小接口后，单测和未来存储迁移会更轻。

3. `promptsquare.Store.resolveReferenceSourcePath` 允许绝对路径。
   当前调用链中 API 层以 `job.References` 生成 `data/spaces/<token>/job_refs/...` 相对路径，风险主要受 API 层约束。但如果未来把 `SubmitFromResultRequest` 暴露给更多入口，store 层应收紧为 data root 路径守卫，避免绝对路径成为隐患。

## 4. 已做的小修

- 新增本文档：`docs/backend-module-review.md`。
- 未做后端源码小拆分，避免与当前并行未提交改动冲突。
- 未触碰 `web/` 前端文件。

## 5. 建议的轻量整理顺序

1. `internal/api` 先做同包无行为拆分。
   新增 `task_projection.go` 或类似文件，集中 `publicResult`、`publicV1Job`、`resultByIndex`、结果图片解析/服务 helper。保留现有响应字段和状态码差异。

2. `internal/api/prompt_square.go` 再拆 from-result adapter。
   可把 `promptSquareFromResultRequest`、`promptSquareReferencesForSubmit`、`fallbackPromptSquareReferences`、`cleanPromptSquareJobReferencePath` 移到 `prompt_square_from_result.go`。这是文件拆分，不改变包和行为。

3. `internal/jobs/manager.go` 拆成同 package 文件。
   建议顺序：`manager_reference.go` 放参考图快照和路径清理，`manager_upstream.go` 放上游请求/debug payload，`manager_lifecycle.go` 放 worker/run/fakeProgress，`manager_pixhost.go` 放 Pixhost 上传。先移动函数，不改逻辑。

4. `internal/promptsquare/store.go` 拆资产处理文件。
   把图片 MIME/复制/ResolveImage/reference path 相关函数移到 `assets.go`，把 item/list/like/ranking 保留在 `store.go`。同时为参考图 source path 增加 data root path guard 测试。

5. `internal/retention` 引入窄接口。
   将 `Config.Jobs` 与 `Config.PromptSquare` 的消费面收窄到 `AllSpacesJobs() (map[string][]jobs.Job, error)` 和 `SourceTaskIDs() (map[string]struct{}, error)`。这是低风险解耦，行为不变。

6. 外部 API 稳定前建立响应 DTO。
   优先从 `/v1` 开始，新增小 DTO 而不是直接返回 `jobs.Job`。Web 内部 API 可暂时继续复用 `jobs.PublicJob`。

## 6. 风险和注意事项

1. 当前工作树已有大量未提交改动，任何代码拆分都应由单一后端写入者在目标文件稳定后进行。
2. `jobs.Job` 作为 API 响应时需要持续防泄露测试，尤其是 `SpaceToken`、输出真实文件名、运行时 key、参考图内部路径。
3. `retention` 属于删除逻辑，后续重构必须保留路径守卫测试、广场投稿保护测试、越界拒删测试。
4. 广场永久保存依赖 `promptsquare.Store` 对结果图和参考图复制成功后再写 `items.json`。拆分时不能破坏失败回滚。
5. `/v1` 外部接口一旦公开，内部 DTO 字段新增会变成兼容性风险，应尽快独立 request/response contract。

## 7. 验证记录

已运行：

```powershell
go test ./internal/api ./internal/jobs ./internal/promptsquare ./internal/retention ./cmd/local-server
```

结果：首次运行时 `internal/api`、`internal/promptsquare`、`internal/retention`、`cmd/local-server` 通过，`internal/jobs` 在 `TestManagerCancelDoesNotWaitForUpstreamCompletion` 的 `TempDir RemoveAll cleanup` 阶段出现 Windows 临时目录未清空错误。该失败不是断言失败，随后单独重跑该用例通过。

最终干净重跑：

```powershell
go test -count=1 ./internal/api ./internal/jobs ./internal/promptsquare ./internal/retention ./cmd/local-server
```

结果：通过。
