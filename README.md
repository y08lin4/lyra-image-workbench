# LyAI生图工作台

一个面向本机 / 私有服务器部署的 LyAI 生图工作台。项目由 Go 后端统一托管前端页面和同源 `/api`，前端不直连 NewAPI，也不保存上游地址；图片生成、任务队列、Key 使用、结果落盘、SSE 状态推送都由后端负责。

---

## 核心定位

本项目相较原始纯前端 / Worker 类生图页面，重点解决三个问题：

1. **开放服务可控**：普通用户只进入个人空间，Admin 独立管理上游 URL、超时、域名和 Debug。
2. **长任务稳定**：提交任务后后端后台执行，前端刷新、断开、手机锁屏不影响任务继续跑。
3. **结果可沉淀**：生成图保存到本机 `outputs/`，任务状态保存到 `data/`，可选再上传 PiXhost 图床。

---

## 技术栈

- 后端：Go
  - 托管 `web/dist`
  - 提供同源 `/api`
  - 管理个人空间、Admin、任务队列、SSE、NewAPI 请求、图片落盘
- 前端：React + TypeScript + Vite
  - 多标签工作台
  - 响应式桌面 / 手机 UI
  - 暗色模式

---

## 工作台功能

### 1. 多标签工作流

顶部横幅会提示 API 服务入口：`https://ai-cf.ailinyu.de`。

主界面固定为：

```text
生成 / 结果 / 队列 / 设置
```

工作流闭环：

```text
写提示词 → 选模型/规格 → 提交 → 看结果 → 下载/复制/复用/图生图 → 必要时用助手继续优化
```

关键交互：

- 提交任务成功后自动跳到“结果”。
- 点击队列任务自动跳到“结果”。
- 结果图点“作为参考图”后自动回到“生成”，并切到图生图。
- 复用参数会把历史任务的提示词、模型、规格、数量、并发带回生成页。

### 2. 生成页

生成页左侧是请求表单，右侧内嵌提示词助手。

请求表单分区：

```text
提示词
模型与模式
图生图参考图 / 合一方向
图片规格
数量与执行
```

当前默认值：

- 图片尺寸：`自动`
- 质量：`高`
- 输出格式：`PNG`
- 数量：读取当前空间默认数量，未设置时为 `1`
- 并发：读取当前空间默认并发，未设置时为 `1`

说明：

- `自动尺寸` 表示不向上游传具体 `size`，让模型或上游自行决定尺寸。
- Image-2 支持质量选择：自动 / 低 / 中 / 高，默认高。
- Banana 的比例、清晰度不是普通参数，而是通过模型 ID 路由。

### 3. 图生图 / 多图合一

图生图可以上传一张或多张参考图。

多参考图逻辑：

- 可以设置某张为主图。
- 提交时后端保证主图排在 `uploadIds[0]`。
- 前端会给提示词自动加合一方向前缀，大意是：

```text
以第一张参考图为主图，保留主图主体、构图、姿态、光影方向和比例，融合其他参考图的风格、元素、服装、材质或背景。
```

这样可以避免“到底往哪张图合”的歧义。

### 4. 结果页

结果页以当前任务为中心显示：

- 任务状态
- 原始提示词
- 模型 / 比例 / 清晰度 / 质量 / 格式 / 数量 / 并发
- 每张图片的实际尺寸、实际质量、输出格式、耗时、图床状态

每张结果图下方提供常驻操作按钮：

- 预览
- 下载
- 复制图片
- 复制链接
- 上传图床 / 重试图床
- 作为参考图

全屏预览只显示：

- 图片
- 真实像素尺寸
- 实际宽高比
- 文件大小
- 底部操作按钮

全屏预览不显示提示词和参数，避免遮挡图片。

### 5. 上游改写提示词逻辑

结果页的原始提示词下方可能出现“上游改写提示词”。这不是本项目主动改写，也不是提示词助手生成。

逻辑是：

1. 本项目把你的提示词发给 NewAPI / 上游模型。
2. 如果上游返回 JSON 中包含：

```json
{
  "revised_prompt": "..."
}
```

或：

```json
{
  "data": [
    { "revised_prompt": "..." }
  ]
}
```

3. 后端会把它保存为 `result.revisedPrompt`。
4. 前端只负责展示这个上游返回值。

它表示“上游返回的实际处理提示词 / 改写提示词”。它不会覆盖原始提示词，只作为结果信息保留。

### 6. 队列页

队列页是历史和异常处理中心：

- 任务统计：全部 / 进行中 / 成功 / 异常
- 筛选：全部 / 进行中 / 成功 / 失败 / 收藏
- 搜索：提示词、模型、错误码
- 缩略图预览
- 批量收藏、下载、删除
- 单任务复用、重试、详情、删除

失败任务会尽量显示：

```text
中文原因 / 错误码 / 英文标识
```

### 7. 提示词助手

提示词助手内嵌在生成页右侧，使用 `gpt-5.5`。

支持：

- 文字生成图片提示词
- 图片还原提示词
- 灵感模式
- 历史 / 会话
- 继续对话修改提示词
- 复制提示词
- 填入并使用指定模型

填入逻辑：

- 点击“应用此提示词”后，会填回生成页主输入框。
- 如果助手里选择了 Image-2 / Banana，会同步主表单的模型选择。

### 8. 设置页

普通设置页属于当前个人空间，主要保存：

- `codex-key`：Image-2 与提示词助手使用
- Banana 分组 Key：Banana Nano 使用
- 默认数量
- 默认并发
- PiXhost 自动上传开关

Key 只在后端保存，前端只显示掩码。

### 9. Admin 页面

Admin 独立路径：

```text
/admin
```

普通工作台页面不再展示 Admin 入口；管理员需要自己在域名后拼接 `/admin` 进入，避免普通用户误触管理配置。

首次访问需要设置 Admin 密码，后续登录后可以配置：

- NewAPI Base URL
- 对外访问域名
- 请求超时时间，默认 `600s`
- Debug 日志开关

Debug 开启后，只对新创建任务生效。前端结果页会显示脱敏后的请求、响应、保存等关键日志，不显示 API Key 明文。

---

## 模型与 Key 逻辑

### Image-2

- 模型 ID：`gpt-image-2`
- Key 名称：`codex-key`
- 支持文生图和图生图
- 支持自动尺寸、固定比例、清晰度、质量、输出格式

### Banana Nano

- 使用单独 Banana 分组 Key
- URL 仍走后端统一配置的 NewAPI Base URL
- 规格通过模型 ID 决定，而不是通过普通参数传递

内置 Banana 模型 ID 包括：

```text
gemini-3.1-flash-image-preview
gemini-3.1-flash-image-preview-16x9-4k
gemini-3.1-flash-image-preview-9x16-4k
gemini-3.1-flash-image-preview-16x9-2k
gemini-3.1-flash-image-preview-9x16-2k
gemini-3.1-flash-image-preview-2k
gemini-3.1-flash-image-preview-4k
gemini-3.1-flash-image-preview-4x3-4k
gemini-3.1-flash-image-preview-4x3-2k
gemini-3.1-flash-image-preview-1x1-4k
gemini-3.1-flash-image-preview-3x4-2k
gemini-3.1-flash-image-preview-3x4-4k
gemini-3.1-flash-image-preview-1x1-2k
```

---

## 后端任务逻辑

任务创建后：

1. 前端提交 `/api/background-tasks`。
2. 后端创建任务并立刻返回任务对象。
3. 后台 worker 继续执行上游请求。
4. 每张图保存到 `outputs/`。
5. 任务状态写入 `data/`。
6. 前端通过 SSE 或轮询刷新状态。

多图任务逻辑：

- 前端可以填数量 `n`。
- 后端按单图请求拆分执行。
- 单张失败不会强制整单失败。
- 部分成功时任务状态为 `partial_failed`。

程序重启逻辑：

- 已完成任务保留。
- 排队任务可恢复。
- 运行中任务会标记为 interrupted，避免重复请求导致重复扣费。

---

## 目录说明

```text
cmd/local-server        Go 服务入口
internal/api            HTTP 路由
internal/jobs           任务队列、状态机、执行逻辑
internal/newapi         NewAPI 图片接口客户端
internal/output         图片落盘与读取
internal/uploads        图生图参考图保存
internal/events         SSE 事件中心
internal/settings       Admin 全局配置
internal/spaceconfig    个人空间配置
internal/adminauth      Admin 初始密码和登录
web/                    React + Vite 前端
data/                   本机配置、空间和任务状态
outputs/                本机生成图片
```

---

## 开发命令

```bash
# 后端开发
go run ./cmd/local-server

# 前端开发服务器，仅开发期使用 Vite 代理 /api 到 Go 后端
cd web
npm install
npm run dev

# 前端生产构建
cd web
npm run build

# 后端测试
go test ./...
```

生产形态下由 Go 后端直接托管 `web/dist`，访问 `/api/...` 即可。

---

## Linux / 宝塔部署更新命令

服务器已部署后，一键更新：

```bash
cd /www/wwwroot/image-workbench && git pull && cd web && npm run build && cd .. && go build -o image-workbench ./cmd/local-server && systemctl restart image-workbench && systemctl status image-workbench --no-pager
```

看到：

```text
Active: active (running)
```

说明服务正常。

---

## 设计文档

- `docs/DEPLOY_LINUX.md`：Linux 服务器部署、systemd、Nginx/Caddy、升级和备份教程。
- `docs/DEPLOY_BAOTA.md`：宝塔面板 Go 项目部署教程，包含字段填写、Nginx 反代和常见问题。
- `docs/CHANGES_FROM_AI_IMAGE_GENERATE.md`：相较 `AI-Image-generate` 的架构、功能和部署更新说明。
- `docs/ROUTES.md`：参考项目路由与本地化调整。
- `docs/STACK.md`：Go 后端与 React/Vite 前端选型。
- `docs/PROJECT_REQUIREMENTS.md`：项目模块化和稳定性要求。
- `docs/REFERENCE_PROJECT_ANALYSIS.md`：参考项目路由、后台任务和稳定性缺口分析。
- `docs/CLOSED_LOOP_DESIGN.md`：闭环设计和实现顺序。
- `docs/SPACE_DESIGN.md`：个人空间、空间密码、固定模型和图生图设计。
