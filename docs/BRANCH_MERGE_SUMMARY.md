# Branch Merge Summary

> 历史说明：本文记录 2026-06-25 的分支收敛过程，不代表当前功能面。当前 dev 已移除视频/MiniMax、GIF 工作台和 FFmpeg 渲染链路，现役范围以 README、docs/ROUTES.md 和实际路由为准。

## 目标

本文记录 2026-06-25 这轮分支收敛的实际处理过程。

本轮目标是把远程分支压缩为只保留 `master` 和 `dev`：

- `dev` 作为后续开发集成分支。
- `master` 作为稳定发布/默认分支。
- 已经合入的功能分支在确认 `master` 与 `dev` 都包含对应改动后删除远程分支。

## 合并结果

最终集成提交：

- `09fa3e8 Merge origin/video into dev`

本轮所有功能分支都采用保留功能的合并策略，没有丢弃任何已确认要保留的功能族。

## 合并顺序

| 顺序 | 源分支 | 目标分支 | 合并提交 | 处理结论 |
| --- | --- | --- | --- | --- |
| 1 | `origin/dev` | `dev` | `566e4a0` | 合入 Prompt Square / Prompt Library 相关开发内容 |
| 2 | `origin/gif` | `dev` | `71e1bc4` | 合入单图生成 GIF 工作流 |
| 3 | `origin/prompts` | `dev` | `2ca638c` | 与已有提示词库历史基本重叠，保留集成后的 `dev` 版本 |
| 4 | `origin/feature/external-bearer-api` | `dev` | `4aa4ac5` | 合入外部 Bearer API 和开发者 Key 管理 |
| 5 | `origin/video` | `dev` | `09fa3e8` | 合入 MiniMax 视频生成与后台额度/Key 配置 |

合并完成后，`main` 曾被快进到同一集成点；按最终分支策略，远程只保留 `master` 和 `dev`。

## 冲突处理原则

冲突解决采用加法合并：

- 保留 `dev` 上已经集成的 Prompt Library、Prompt Square、GIF、Bearer API 能力。
- 合入源分支新增的路由、服务、类型、前端页面和文档。
- 对重复历史或重复功能，以当前集成后的 `dev` 实现为准。
- 不在冲突解决中混入无关重构。
- 涉及 FFmpeg/GIF 的代码按安全优先处理，低版本或无法识别版本只禁用 GIF 合并，不影响普通图像生成。

## 各分支处理

### `origin/dev`

- 处理结论：已合入。
- 关键内容：Prompt Square、提示词工具增强、前端面板、后端路由和存储。
- 冲突处理：保留 Prompt Library 和 Prompt Square 两条能力线，避免互相覆盖。
- 代表文件：`docs/PROMPT_SQUARE.md`、`internal/api/prompt_square.go`、`internal/promptsquare/store.go`、`web/src/components/PromptSquarePanel.tsx`。

### `origin/gif`

- 处理结论：已合入。
- 关键内容：`/gif` 单图到 GIF 工作流、GIF 计划生成、帧图生成、最终 GIF 合成、部署文档。
- 冲突处理：在保留提示词功能和现有工作台行为的基础上，新增 GIF 路由、服务和页面。
- 安全处理：
  - 最终 GIF 合成依赖系统 FFmpeg，不把 FFmpeg 编译进 Go 二进制。
  - 服务端要求 FFmpeg `8.1.2+`，低于该版本或无法解析版本时禁用 GIF 合并。
  - 渲染输入限定为本项目任务产生的 PNG 帧，FFmpeg 使用 `-f image2 -c:v png -i frame_%04d.png`。
  - 当前业务边界不是接收任意 AVI/MKV/MOV 视频文件给 FFmpeg 解码，而是把已生成、已校验的 PNG 帧序列交给 FFmpeg 合成 GIF。
  - 该处理参考 `CVE-2026-8461` 对 FFmpeg `libavcodec` MagicYUV 解码器的风险说明；NVD 记录显示该漏洞影响 FFmpeg `8.1.2` 之前版本，CNA/JFrog 评分为 CVSS 3.1 `8.8 HIGH`，NVD 发布日期为 2026-06-18，最后修改日期为 2026-06-22。
  - 参考来源：<https://nvd.nist.gov/vuln/detail/CVE-2026-8461>。
  - 注意：如果发行版把修复 backport 到低于 `8.1.2` 的版本号，当前实现仍会因为版本字符串低于最低门槛而禁用 GIF 合并；部署侧应以 `ffmpeg -version` 验证运行时版本。
- 代表文件：`internal/gifrender/ffmpeg.go`、`internal/gifrender/ffmpeg_test.go`、`internal/api/gif.go`、`web/src/components/GifWorkbenchPage.tsx`。

### `origin/prompts`

- 处理结论：已合入，但主要作为重复提示词库历史确认。
- 关键内容：同步的 GPT 图像提示词库。
- 冲突处理：因为同类内容已经在集成后的 `dev` 中存在，冲突处保留当前 `dev` 的综合版本。

### `origin/feature/external-bearer-api`

- 处理结论：已合入。
- 关键内容：外部 Bearer API、开发者 API Key 管理、`/v1/image-tasks` 任务创建和查询、相关文档。
- 冲突处理：保持原有登录态接口，同时新增 Bearer Key 鉴权通道。
- 代表文件：`docs/EXTERNAL_API_DESIGN.md`、`internal/api/v1_image_tasks.go`、`internal/api/developer_api_keys.go`、`internal/apikeys/store.go`。

### `origin/video`

- 处理结论：已合入。
- 关键内容：MiniMax 视频生成预览、后台 MiniMax Key 配置、用户视频额度。
- 冲突处理：保留已经集成的 GIF、Bearer API、Prompt Square 能力，同时加入视频路由、服务、后台配置和前端面板。
- 代表文件：`docs/MINIMAX_VIDEO.md`、`internal/api/minimax_video.go`、`internal/minimax/video.go`、`web/src/components/MiniMaxVideoPanel.tsx`。

## 验证

已完成的项目验证：

| 验证项 | 命令 | 结果 |
| --- | --- | --- |
| Go 测试 | `go test ./...` | 通过 |
| 前端构建 | `npm run build`（在 `web` 目录） | 通过 |
| 合并历史核对 | `git log --oneline --first-parent dev --max-count=20` | 合并顺序符合本文记录 |
| 远程分支核对 | `git branch -r -vv` / `git ls-remote --heads origin` | 清理后应只剩 `master` 和 `dev` |

## 最终远程分支策略

远程最终只保留：

- `origin/master`
- `origin/dev`

以下远程分支在确认已被合入后删除：

| 远程分支 | 处理 | 原因 |
| --- | --- | --- |
| `origin/main` | 删除 | 内容已同步到 `master` 和 `dev`，最终策略不保留 `main` |
| `origin/prompts` | 删除 | 提示词库内容已合入 |
| `origin/gif` | 删除 | GIF 工作流已合入 |
| `origin/video` | 删除 | MiniMax 视频能力已合入 |
| `origin/feature/external-bearer-api` | 删除 | 外部 Bearer API 已合入 |

清理后校验标准：

```powershell
git ls-remote --heads origin
```

只应返回：

```text
<same-sha> refs/heads/dev
<same-sha> refs/heads/master
```

其中 `dev` 和 `master` 的 SHA 应一致。
