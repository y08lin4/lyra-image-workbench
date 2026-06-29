# GIF 动图功能实现说明

更新时间：2026-06-29  
适用分支：`dev`  
现役入口：工作台左侧栏 `GIF 动图`

本文说明当前 GIF 功能到底怎么实现、前后端怎么对应、哪些能力已经上线、哪些不是当前实现。

## 1. 一句话结论

当前 GIF 是一个独立的本地动图模块：用户上传或从历史结果选择一张图片，选择动效预设并输入动效描述，前端创建 `mode=gif` 的后台任务，Go 后端在本机把这张图做成循环 GIF，保存到结果历史。

当前实现不调用视频、不调用 MiniMax、不调用 FFmpeg、不调用上游图片生成接口。

## 2. 当前能力边界

已实现：

- 独立 GIF 页面：`web/src/components/GifPage.tsx`
- 独立前端任务封装：`web/src/api/gifTasks.ts`
- 通用任务 API 创建 GIF：`POST /api/background-tasks`
- 后端任务模式：`jobs.ModeGIF = "gif"`
- 本地 GIF 渲染器：`internal/gifrender/render.go`
- GIF 结果进入任务历史、结果页和任务详情
- 从历史结果复制为 GIF 参考图
- 输出保存为 `image/gif`
- GIF 任务不扣用户生图额度
- 旧 GIF 占位错误会映射为“历史 GIF 任务，请重新创建 GIF 动图任务”

当前没有实现：

- 真正的视频生成
- 任意视频上传/解码
- MiniMax 视频链路
- FFmpeg 合成链路
- `/api/gif/status`、`/api/gif/plans`、`/api/gif-renders` 等旧实验路由
- 基于大模型/视频模型的语义级真实动画
- 多张参考图混合生成 GIF

## 3. 用户流程

```text
用户进入 GIF 动图页
  -> 上传一张图片，或从历史结果复制一张图片作为参考图
  -> 选择动效预设
  -> 输入自然语言描述，例如“头发动起来，背景保持不变”
  -> 选择动效幅度和循环节奏
  -> 提交 GIF 任务
  -> 前端跳到结果历史
  -> 后端本地生成 .gif
  -> 结果历史显示 GIF 产物
```

前端页面会把用户输入整理成一份普通后台任务请求，核心字段是：

```json
{
  "provider": "image-2",
  "model": "gpt-image-2",
  "mode": "gif",
  "prompt": "[GIF 动图] 头发飘动\n头发动起来，背景保持不变\npreset: animate hair with gentle wind, keep the face identity and background stable\nmotion strength: 标准\nloop rhythm: 平滑循环\npreserve identity, composition, and non-moving regions",
  "framePrompts": [
    "animate hair with gentle wind, keep the face identity and background stable",
    "motion strength: standard",
    "loop rhythm: smooth",
    "user intent: 头发动起来，背景保持不变"
  ],
  "ratio": "auto",
  "resolution": "auto",
  "quality": "auto",
  "outputFormat": "gif",
  "count": 1,
  "concurrency": 1,
  "uploadIds": ["reference_upload_id"]
}
```

注意：这里保留 `provider/model` 是为了复用现有任务类型和前端契约。后端遇到 `mode=gif` 时不会调用这个 provider，也不会检查上游 Key。

## 4. 前端实现

主要文件：

| 文件 | 职责 |
| --- | --- |
| `web/src/components/GifPage.tsx` | GIF 页面，负责参考图、预设、描述、幅度、节奏和提交 |
| `web/src/components/GifPage.css` | GIF 页面布局和样式 |
| `web/src/api/gifTasks.ts` | GIF 预设、提示词拼装、`CreateTaskRequest` 构造 |
| `web/src/api/tasks.ts` | 通用后台任务 API wrapper |
| `web/src/components/WorkbenchPage.tsx` | 工作台集成，负责提交 GIF 任务、跳转结果页、从历史图转参考图 |
| `web/src/components/workbench/nav.ts` | 左侧导航中注册 `gif` tab |
| `web/src/components/ResultCanvas.tsx` | 结果历史展示 GIF 任务和 GIF 输出 |
| `web/src/components/TaskDetailModal.tsx` | 任务详情中展示 GIF 模式 |

### 4.1 GIF 预设

`web/src/api/gifTasks.ts` 内置 6 个动效预设：

| 预设 ID | 页面名称 | 适用场景 |
| --- | --- | --- |
| `hair-sway` | 头发飘动 | 人像、二次元头像，轻微发丝摆动 |
| `camera-push` | 镜头推近 | 轻微放大主体，制造短循环镜头感 |
| `blink` | 眨眼微笑 | 人像轻眨眼，表情微变化 |
| `poster-loop` | 海报轻动效 | 光影、烟雾、背景元素小幅循环 |
| `cloth-breeze` | 衣摆微风 | 服饰、裙摆、披风边缘摆动 |
| `product-turn` | 产品呼吸感 | 商品图高光和小幅视差 |

### 4.2 幅度和节奏

幅度：

- `subtle`：轻微
- `standard`：标准
- `bold`：明显

节奏：

- `smooth`：平滑循环
- `breathing`：呼吸节奏
- `snappy`：短促活泼

前端只是把这些参数写入 `prompt` 和 `framePrompts`。真正的幅度、帧延迟和效果识别在后端 `gifrender` 里完成。

### 4.3 参考图来源

GIF 页面支持两种参考图来源：

1. 直接上传图片，走 `/api/uploads/reference`
2. 从历史结果读取图片，再重新上传为参考图

从历史结果加入 GIF 参考图的逻辑在 `WorkbenchPage.tsx`：

- `uploadResultImageAsReference(src, index)` 先 fetch 历史结果图片
- 将 blob 转成 `File`
- 调用 `uploadReferenceImages`
- 返回新的 `ReferenceUpload`

这样做的结果是：GIF 任务依赖的是本项目受控的参考图上传记录，而不是直接读任意外部 URL。

## 5. 后端 API 和任务链路

GIF 没有单独的 `/api/gif/*` 路由。它复用后台任务 API：

| 方法 | 路径 | 作用 |
| --- | --- | --- |
| `POST` | `/api/background-tasks` | 创建 GIF 任务 |
| `GET` | `/api/background-tasks` | 查看任务历史 |
| `GET` | `/api/background-tasks/{id}` | 查看 GIF 任务状态 |
| `GET` | `/api/background-tasks/{id}/events` | SSE 任务进度 |
| `POST` | `/api/background-tasks/{id}/cancel` | 取消任务 |
| `POST` | `/api/background-tasks/{id}/retry` | 重试任务 |
| `GET` | `/api/background-tasks/{id}/images/{index}` | 读取 GIF 结果 |

路由注册在 `internal/api/router.go`。

### 5.1 创建任务

入口：`internal/api/tasks.go`

`TaskHandler.Create` 做这些事：

1. 解析 JSON 到 `jobs.CreateRequest`
2. 写入 runtime secrets 和 source
3. 调用 `billableTaskCredits`
4. 检查用户额度
5. 设置 `BeforeEnqueue` 扣费回调
6. 调用 `jobs.Manager.Create`
7. 返回 `taskId`、`job`、`consumedCredits`

GIF 的特殊点：

- `jobs.CreditCostForRequest(req)` 对 `ModeGIF` 返回 `0`
- `requestUsesPersonalUpstreamKey` 对 `ModeGIF` 返回 `true`
- 因此 GIF 任务不会扣生图额度

### 5.2 Job Manager 创建规则

入口：`internal/jobs/manager.go`

`Manager.Create` 对 `ModeGIF` 做了这些特殊处理：

- 必须有参考图
- 只能有 1 张参考图
- 跳过上游 API Key 检查
- `ratio = "auto"`
- `resolution = "auto"`
- `quality = "auto"`
- `outputFormat = "gif"`
- `size = "自动"`
- `count = 1`
- `consumedCredits = 0`
- 将参考图复制成任务引用快照，避免原上传被删除后任务丢图

### 5.3 后台执行

后台执行入口：`internal/jobs/manager.go`

`generateOne` 会先进入通用任务状态机，然后按模式分支：

```go
if job.Mode == ModeGIF {
    return m.generateGIF(ctx, spaceToken, job, index, prompt, started)
}
```

也就是说 GIF 不会走普通图片生成的 `newapi.Client`，不会向上游接口发请求。

`internal/jobs/manager_test.go` 里有测试 `TestManagerGeneratesGIFWithoutUpstream`，会启动一个会直接失败的上游测试服务，并断言 GIF 任务没有任何上游请求。

## 6. 本地 GIF 渲染器

核心文件：`internal/gifrender/render.go`

渲染入口：

```go
gifrender.RenderFile(ctx, path, gifrender.Options{
    Prompt: prompt,
    FramePrompts: job.FramePrompts,
})
```

### 6.1 输入

渲染器当前能解码：

- PNG
- JPG/JPEG
- GIF

但要注意：前端参考图上传入口目前白名单是 PNG/JPG/WEBP，不含 GIF 文件。因此从正常页面直接上传 GIF 作为参考图目前不算完整入口能力；这是实现边界不一致，需要后续统一。

如果图片无法解码，会返回：

```text
GIF 动图目前支持 PNG/JPG/GIF 参考图，当前图片无法解码
```

### 6.2 尺寸限制

渲染前会调用 `fitLongEdge(source, 1024)`：

- 如果长边小于等于 1024，保持原尺寸
- 如果长边超过 1024，按比例缩小到长边 1024

这是为了避免超大图导致内存和 CPU 压力过大。

### 6.3 帧数和循环

当前固定生成：

- `defaultFrames = 12`
- `LoopCount = 0`，即无限循环
- 每帧写入 `Delay`
- 最终用 `image/gif.EncodeAll` 编码

### 6.4 效果识别

后端会把 `prompt` 和 `framePrompts` 拼起来，用关键词识别动效：

| 关键词 | 效果 |
| --- | --- |
| `camera push`、`push-in`、`镜头` | `camera-push` |
| `product`、`highlight`、`商品`、`产品` | `product-turn` |
| `blink`、`眨眼` | `blink` |
| `poster`、`smoke`、`light`、`海报`、`光影` | `poster-loop` |
| `cloth`、`clothing`、`衣` | `cloth-breeze` |
| `hair`、`头发`、`发丝` | `hair-sway` |
| 未命中 | `subtle-loop` |

### 6.5 幅度和节奏识别

幅度：

| 关键词 | strength |
| --- | --- |
| `bold`、`明显` | `0.06` |
| `subtle`、`轻微` | `0.022` |
| 默认 | `0.038` |

节奏：

| 关键词 | delay |
| --- | --- |
| `snappy`、`短促` | `7` |
| `breathing`、`呼吸` | `12` |
| 默认 | `9` |

GIF 的 delay 单位是 1/100 秒，所以默认大约是每帧 90ms。

### 6.6 当前真实动效方式

当前不是语义级动画，而是对整张图做轻量像素变换：

- 缩放
- 横向/纵向平移
- 明暗变化
- 对 `hair-sway`、`cloth-breeze` 做行级波动
- 使用最近邻采样
- 使用 `palette.Plan9`
- 用 Floyd-Steinberg 抖动转换到 GIF 调色板

这意味着它能做“轻微循环动效”，但还不能真正理解“只让头发局部动、脸完全不动”这种语义级编辑。前端文案说的“头发动起来”目前会映射到近似的行波动和轻微平移。

## 7. 输出保存和结果读取

GIF 任务保存路径和普通图片任务一样走 `internal/output.Store`：

```go
saved, err := m.output.Save(spaceToken, jobID, index, rendered.Bytes, "image/gif")
```

保存后结果字段：

| 字段 | 值 |
| --- | --- |
| `ImageURL` | `/api/background-tasks/{jobID}/images/{index}` |
| `Mime` | `image/gif` |
| `OutputFormat` | `gif` |
| `RevisedPrompt` | GIF prompt |
| `ActualSize` | 实际 GIF 尺寸 |
| `ActualQuality` | `{frames} frames` |

任务列表对外返回时，`PublicResult` 会隐藏本机文件名和日期，只保留可鉴权读取的 API URL。

如果用户空间开启了 `AutoUploadPixhost`，GIF 保存后还会尝试上传到 PiXhost，并写入：

- `RemoteURL`
- `RemoteThumbURL`
- `UploadError`

## 8. 额度和 Key 策略

当前 GIF 的策略是：

- 不消耗用户生图次数
- 不要求上游 Key
- 不调用上游图片生成
- 不使用管理员设置的系统上游 Key
- 不使用用户本地 Key

原因是当前 GIF 渲染完全在本机完成，没有调用外部模型，也没有产生上游生图成本。

相关代码：

- `jobs.CreditCostForRequest`：`ModeGIF` 返回 `0`
- `Manager.Create`：`ModeGIF` 设置 `ConsumedCredits = 0`
- `TaskHandler.requestUsesPersonalUpstreamKey`：`ModeGIF` 返回 `true`
- `Manager.Create`：`ModeGIF` 跳过上游 Key 检查

后续如果接入真实 GIF Provider 或视频/动画模型，必须重新定义：

- 单次 GIF 消耗多少次数
- 失败是否退款
- 是否允许用户自己的 Key
- 是否允许管理员系统 Key
- 任务日志和错误码怎么展示

## 9. 错误和遗留占位

之前页面出现过：

```text
GIF backend unavailable: GIF 动图生成后端尚未接入
```

这属于旧占位任务或旧实现遗留，不应再出现在新建 GIF 任务里。

当前错误映射：

- `gif backend unavailable`
- `GIF 动图生成后端尚未接入`
- `历史 GIF 任务`

都会映射为：

```text
历史 GIF 任务，请重新创建 GIF 动图任务
```

测试 `internal/api/tasks_test.go` 里有 `assertNoGIFPlaceholder`，保证新建 GIF 任务响应不包含 `E_GIF_BACKEND_UNAVAILABLE` 或旧占位中文。

## 10. 测试覆盖

当前已有测试：

| 测试 | 覆盖点 |
| --- | --- |
| `internal/gifrender/render_test.go::TestRenderFileCreatesAnimatedGIF` | 能生成 GIF 字节、能被 DecodeAll 解码、多帧、尺寸和帧数正确 |
| `internal/gifrender/render_test.go::TestRenderFileRejectsUndecodableReference` | 无法解码图片会失败 |
| `internal/jobs/manager_test.go::TestManagerGeneratesGIFWithoutUpstream` | GIF 不调用上游、强制 count=1、ConsumedCredits=0、引用快照保存、输出为 GIF |
| `internal/api/tasks_test.go::TestGIFBackgroundTaskCreatesLocalGIF` | 从 HTTP API 创建 GIF，最终任务成功，结果 MIME 是 `image/gif`，图片接口返回 GIF 字节 |
| `internal/api/router_test.go::TestGIFAPIsAreRemoved` | 旧 `/api/gif/*` 和 `/api/gif-renders` 路由不存在 |
| `internal/jobs/errors_test.go::TestErrorMetaMapsGIFBackendUnavailable` | 旧占位错误映射为历史任务提示 |

建议修改 GIF 后至少运行：

```powershell
go test -count=1 ./internal/gifrender ./internal/jobs ./internal/api
```

如果改了前端页面，再运行：

```powershell
npm --prefix web run build
```

## 11. 安全边界

当前安全边界比较简单：

- 不接收视频文件
- 不调用 FFmpeg
- 不拼接 shell 命令
- 不处理远程 URL 输入
- 不允许任意路径输入
- 通过上传模块或任务引用快照读取受控目录里的图片
- 长边缩到 1024，降低本地渲染压力

这意味着用户之前提到的 FFmpeg 高危漏洞对当前 GIF 实现没有直接影响，因为当前实现没有 FFmpeg 运行时依赖。

如果未来改成 FFmpeg 或外部 GIF 动画服务，必须单独设计安全边界：

- 只允许受控目录中的图片帧输入
- 禁止任意视频容器进入解码链路
- 禁止远程 URL
- 命令参数使用数组和白名单，不拼 shell 字符串
- 限制帧数、尺寸、运行时间、并发数和临时目录大小
- FFmpeg 版本和发行版 backport 安全状态要上线前重新核对
- 失败后清理临时文件

## 12. 当前限制和下一步建议

当前限制：

- 动效是轻量像素变换，不是真正的 AI 语义动画
- “眨眼”“头发动起来”等效果只是关键词映射后的近似效果
- 只能用 1 张参考图
- 帧数固定 12 帧
- 长边超过 1024 会缩小
- GIF 调色板会带来颜色损失
- Web 上传入口目前不接受 GIF 文件作为参考图
- GIF 仍会校验 provider/model，传非法 provider/model 会创建失败
- 无单独 GIF 计费策略
- 无外部 GIF provider
- 无用户可调帧率、循环次数、输出尺寸

建议下一阶段：

1. 保留当前本地 GIF 作为免费轻量动效。
2. 新增真实 GIF Provider 抽象，例如 `internal/gifprovider`，不要塞进 `newapi` 图片生成客户端。
3. 后端新增 GIF 配置项：启用状态、provider、单次消耗次数、最大尺寸、最大时长。
4. 前端把“本地轻量动效”和“AI 动图生成”区分开。
5. 增加历史图片直接作为 GIF 参考图的更顺手入口。
6. 给 GIF 结果详情展示：参考图、动效描述、预设、幅度、节奏、生成方式。
7. 如果要上传到广场，明确同时保存参考图和最终 GIF，方便复用和查看生成链路。

## 13. 和历史文档的关系

`docs/VIDEO_GIF_BOUNDARY.md` 是 2026-06-27 的历史扫描快照。当时结论是“没有独立 GIF 动态模块”。这已经不是当前代码状态。

当前现役口径以本文为准：

- GIF 独立页面已经存在
- `mode=gif` 后台任务已经存在
- 本地 GIF 渲染器已经存在
- FFmpeg/video/MiniMax 仍然不属于当前 GIF 实现

