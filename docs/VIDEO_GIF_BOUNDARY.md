# 视频删除 / GIF 独立模块边界扫描

扫描日期：2026-06-27  
范围：当前生产代码、部署配置、依赖声明、现役产品文档和历史文档中的视频/GIF 口径。

## 产品决策

1. 视频生成已从当前产品范围删除。当前功能面不再提供 MiniMax/video 入口、路由、组件、API 或配置字段。
2. GIF/动图以后作为独立模块设计，不和视频生成混在一起，不复用 MiniMax/video 命名、额度、路由或配置。
3. 当前 GIF 只保留为图片 MIME/资产格式支持；没有独立 GIF 工作台、GIF API、FFmpeg 合成服务或动态渲染链路。
4. 如果未来恢复 GIF/动图，应作为独立图片衍生/动画模块上线，并单独定义任务模型、权限、部署依赖和安全边界。

## 扫描命令

生产代码和配置扫描：

```powershell
rg -n -S --hidden -g '!node_modules' -g '!dist' -g '!build' -g '!logs' -g '!outputs' -g '!tmp' "MiniMax|minimax|MINIMAX" "web/src" "internal" "cmd" ".env.example" "Dockerfile" "web/package.json" "go.mod"
rg -n -S --hidden -g '!node_modules' -g '!dist' -g '!build' -g '!logs' -g '!outputs' -g '!tmp' "\bvideo\b|\bVideo\b|\bVIDEO\b|mp4|webm" "web/src" "internal" "cmd" ".env.example" "Dockerfile" "web/package.json" "go.mod"
rg -n -S --hidden -g '!node_modules' -g '!dist' -g '!build' -g '!logs' -g '!outputs' -g '!tmp' "\bgif\b|\bGIF\b|ffmpeg|FFmpeg|image/gif|animate|animation|frames?" "web/src" "internal" "cmd" ".env.example" "Dockerfile" "web/package.json" "go.mod"
rg -n -S "/api/(video|videos|gif|gif-renders)|gif-renders|gif/status|gif/plans|MiniMax|minimax|video" "internal/api" "web/src/api" "web/src/components"
rg -n -S "@ffmpeg|ffmpeg|fluent-ffmpeg|gifencoder|gif.js|gifshot|sharp|imagemagick|magick|canvas|wasm|webm|mp4" "web/package.json" "go.mod" "Dockerfile" "scripts"
```

全仓残留和文件名扫描：

```powershell
rg -n -S "gifrender|GifWorkbench|Gif|GIF|/gif|gif-renders|api/gif|ffmpeg|FFmpeg" . -g '!web/node_modules/**' -g '!web/dist/**' -g '!data/**' -g '!logs/**' -g '!outputs/**' -g '!tmp/**' -g '!.git/**'
rg -n -S "minimax|MiniMax|MINIMAX|videoQuota|VideoQuota|MiniMaxVideo|/api/minimax|video" . -g '!web/node_modules/**' -g '!web/dist/**' -g '!data/**' -g '!logs/**' -g '!outputs/**' -g '!tmp/**' -g '!.git/**'
Get-ChildItem -Recurse -Force -LiteralPath "internal" -Directory | Where-Object { $_.Name -match 'gif|video|minimax|ffmpeg' } | Select-Object FullName
Get-ChildItem -Recurse -Force -LiteralPath "web/src" -File | Where-Object { $_.Name -match 'gif|video|minimax|ffmpeg' } | Select-Object FullName
```

关键文件人工复核：

```powershell
Get-Content internal/api/router.go
Get-Content internal/api/router_test.go
Get-Content internal/api/v1_image_tasks.go
Get-Content internal/jobs/types.go
Get-Content internal/jobs/manager.go
Get-Content internal/newapi/client.go
Get-Content internal/config/config.go
Get-Content internal/settings/settings.go
Get-Content web/src/lib/ratios.ts
Get-Content web/src/components/OutputFormatPicker.tsx
Get-Content web/src/components/GenerationPanel.tsx
Get-Content web/src/api/tasks.ts
Get-Content web/package.json
Get-Content go.mod
Get-Content Dockerfile
```

文档扫描：

```powershell
rg -n -S "MiniMax|minimax|视频|video|Video|GIF|gif|动图|FFmpeg|ffmpeg|gif-renders|/api/gif" "docs" "README.md" "PRODUCT.md"
```

## 视频 / MiniMax 扫描结论

- `web/src/**`、`internal/**`、`cmd/**`、`.env.example`、`Dockerfile`、`web/package.json`、`go.mod` 中没有 MiniMax 生产代码引用。
- `internal/api/router.go` 未注册 `/api/minimax/*`、`/api/video*` 或类似视频路由；当前路由集中在图片任务、用户、管理、广场、billing、提示词工具和静态资源。
- 前端任务 API `web/src/api/tasks.ts` 只调用 `/api/background-tasks` 和任务图片/事件/图床接口；没有视频 API wrapper。
- 前端组件文件名扫描未发现 `MiniMaxVideoPanel`、video 页面或 video API 文件。
- 配置面 `internal/config/config.go`、`internal/settings/settings.go`、`web/src/api/config.ts`、`.env.example` 未发现 MiniMax/video Key、额度、超时或模型字段。
- `internal/api/router_test.go` 保留了一个反向断言：Admin config 响应不应包含已移除的视频配置。
- 文件系统里有一个空的 `internal/minimax` 目录，但其中没有源文件；它不是生产入口、路由、组件、API 或配置字段。

结论：当前生产代码中没有 MiniMax/video 生成功能入口。全仓命中主要来自历史文档和移除测试，不代表现役功能。

## GIF / 动图扫描结论

当前仍存在的 GIF 支持是图片资产层支持：

- `internal/output/store.go` 可根据 `image/gif` 保存 `.gif`，并能从 `.gif` 文件名还原 MIME。
- `internal/newapi/client.go` 可在上游返回 `image/gif` 时识别输出格式，但请求侧 `normalizeOutputFormat` 只会主动请求 `png`、`jpeg`、`webp`。
- `web/src/lib/ratios.ts` 的输出格式常量只有 `png`、`jpeg`、`webp`；`OutputFormatPicker` 和创作画布高级参数都使用这组常量。
- `web/src/lib/nativeBridge.ts`、`ResultCanvas.tsx`、`WorkbenchPage.tsx` 有 GIF MIME/扩展名识别，用于保存、下载、展示或历史结果处理。
- `internal/promptsquare`、`internal/promptlibrary`、`internal/pixhost` 支持 GIF 作为图片资产格式。
- `internal/uploads/store.go` 的图生图参考图上传只允许 PNG/JPG/WEBP，不允许 GIF 参考图。

已移除或不存在的 GIF 动态模块：

- 未发现 `internal/gifrender/**`、`internal/api/gif.go`、`web/src/api/gif.ts`、`web/src/components/GifWorkbenchPage.tsx`。
- `internal/api/router.go` 未注册 `/api/gif/status`、`/api/gif/plans`、`/api/gif-renders` 等旧路由。
- `internal/api/router_test.go` 的 `TestGIFAPIsAreRemoved` 明确要求旧 GIF API 返回 404 或 405。
- `web/package.json`、`go.mod`、`Dockerfile`、部署脚本未声明或安装 FFmpeg、gif encoder、canvas/wasm GIF 渲染依赖。

结论：当前 GIF 不是独立工作台，也不是动态渲染链路；它只是部分图片资产路径可识别/保存/展示的 MIME 格式。

## 文档矛盾清单

这些矛盾主要存在于历史调度/合并档案；不建议大面积改历史文档，只建议读者以本文、README、docs/ROUTES.md 和实际路由为准。

- `docs/AGENT_TASK_BREAKDOWN.md` 开头和总目标已声明当前移除视频/MiniMax、GIF 工作台和 FFmpeg 渲染链路；但同文件早期全局约束仍写着“不允许删除 GIF/FFmpeg 相关模块”和“删除视频功能时只删除 MiniMax/视频生成，不删除 GIF”，后文还多处要求“保留所有 GIF/FFmpeg 文件”。
- `docs/BRANCH_MERGE_SUMMARY.md` 开头已声明这是 2026-06-25 的历史合并记录，不代表当前功能面；但正文仍记录 `origin/gif` 和 `origin/video` 已合入，并列出 GIF 工作流、MiniMax 视频、FFmpeg `8.1.2+` 门槛和代表文件。
- `docs/SAAS_MVP_PROGRESS.md` 开头和后续收口段落说明当前不保留视频/动态合成；但中间历史进度段仍记录“动态合成工作流进入最终收口清理”“视频入口尚未移除/合并”等中间态。
- `README.md` 的“媒体能力边界”和 `docs/ROUTES.md` 的“已移除的实验路由”与当前代码扫描一致：视频和动态合成不属于当前第一阶段闭环。

## 后续 GIF 独立模块边界建议

如果未来实现 GIF/动图模块，建议按独立产品线处理：

1. 独立入口：使用独立页面和导航，例如 `/gif` 或“动图工作台”，不要复用视频入口或 MiniMax 文案。
2. 独立 API：使用 `/api/gif/*` 或 `/api/animations/*`，不要塞进 `/api/video*`；也不要让 `/v1/images/*` 图片生成接口直接承担合成语义。
3. 独立任务模型：区分图片生成任务和 GIF 渲染任务，记录帧来源、帧时长、循环次数、尺寸、调色板/压缩参数、产物 MIME 和错误状态。
4. 输入边界：优先只接受本项目已生成或已校验的图片帧；不要把任意用户上传视频交给 FFmpeg 解码。
5. 额度与审计：GIF 合成成本和失败退款应独立计量，避免和视频额度或图片张数混淆。
6. 部署依赖：FFmpeg 作为可选运行时依赖，缺失或版本不合规时只禁用 GIF 合成，不影响普通图片生成。

FFmpeg 安全提醒：

- 版本门槛不要写死后长期遗忘；上线前必须重新核对 FFmpeg 官方发布、发行版安全公告和 NVD/CVE 状态。
- 截至本次扫描，NVD 对 `CVE-2026-8461` 的记录显示 FFmpeg `libavcodec` MagicYUV 解码器在 `8.1.2` 之前受影响，CNA 分数为 CVSS 3.1 `8.8 HIGH`，页面最后修改时间为 2026-06-22：<https://nvd.nist.gov/vuln/detail/CVE-2026-8461>。
- 如果发行版把修复 backport 到低版本号，不能只看字符串版本；需要记录发行版安全公告或包 changelog 作为放行依据。
- 调用 FFmpeg 时必须使用参数数组/固定白名单，不拼 shell 字符串；禁止用户控制 filtergraph、协议、输入路径或输出路径。
- 限制帧数、单帧尺寸、总字节数、运行时间、并发数和临时目录大小；渲染失败要清理临时文件。
- 只允许受控目录下的图片帧输入，禁用任意远程 URL、任意视频容器和未校验路径穿越。

## GIF 动图模式补充决策

GIF 后续应作为独立模式存在，不归入视频生成。核心路径是：用户上传一张图片，描述一个动效想法，也可以选择预设动效模板，然后生成轻量循环动图。模板示例包括呼吸缩放、镜头推近、左右摇镜、光影扫过、粒子闪烁、海报元素入场等。典型用例：上传一张动漫人物图，输入“头发动起来”，系统生成头发轻微飘动的循环 GIF，并尽量保持脸部、身体和背景稳定。

边界要求：

1. 不接收任意视频文件进入解码链路。
2. 输入优先是本站生成/上传的静态图片或受控帧序列。
3. 模板参数白名单化，避免把用户输入直接拼接成 FFmpeg 命令。
4. 如果服务端使用 FFmpeg，只允许安全版本和受控参数；保留 CVE-2026-8461 相关版本提醒。
5. GIF 模式应有自己的路由、API、前端页面和额度规则，不复用已删除的视频功能命名。