# Lyra Image Workbench

[![CI](https://github.com/y08lin4/lyra-image-workbench/actions/workflows/ci.yml/badge.svg)](https://github.com/y08lin4/lyra-image-workbench/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

Lyra Image Workbench（LyAI 生图工作台）是一个可本机运行、也可私有服务器部署的 AI 生图工作台。它把前端页面、用户账号、任务队列、上游图片接口请求、结果落盘和状态推送都收拢到一个 Go 后端里，前端只访问同源 `/api`。

- 开源地址：[`y08lin4/lyra-image-workbench`](https://github.com/y08lin4/lyra-image-workbench)
- 在线演示：[https://ai-image.ailinyu.de/](https://ai-image.ailinyu.de/)
- API 服务入口广告：[https://ai-cf.ailinyu.de/](https://ai-cf.ailinyu.de/) —— 不想自建 OpenAI 兼容网关时，可以从这里了解可用服务。

> 在线演示是公开环境，不建议上传敏感图片或长期保存高权限 API Key。生产使用建议自部署。

---

## 目录

- [这个项目解决什么问题](#这个项目解决什么问题)
- [功能总览](#功能总览)
- [核心工作流](#核心工作流)
- [架构和数据流](#架构和数据流)
- [部署教程索引](#部署教程索引)
- [快速开始](#快速开始)
- [生产部署](#生产部署)
- [环境变量](#环境变量)
- [安全和 Key 策略](#安全和-key-策略)
- [项目结构](#项目结构)
- [开发和测试](#开发和测试)
- [更多文档](#更多文档)
- [开源协作](#开源协作)

---

## 这个项目解决什么问题

很多生图页面是纯前端或 Worker 形态：浏览器直接处理 Key、请求容易被刷新/锁屏打断、历史和结果依赖本地浏览器状态。Lyra Image Workbench 的目标是把它变成一个长期运行的私有工作台：

1. **上游请求由后端接管**：浏览器只请求同源 `/api`，NewAPI / OpenAI 兼容网关地址由 Admin 管理。
2. **长任务不怕前端断开**：提交后后端后台执行，前端刷新、断网、手机锁屏都不影响已创建任务继续跑。
3. **结果和历史可沉淀**：任务、配置、参考图和生成结果落到服务器目录，账号多设备登录后可继续查看。
4. **开放服务更可控**：普通用户账号与 Admin 管理分离，支持 2FA、云端 Key 风险确认、Debug 脱敏日志。

---

## 功能总览

| 模块 | 功能 |
| --- | --- |
| 生成 | 文生图、图生图、多参考图、数量/并发、比例/清晰度/质量/格式配置 |
| 模型 | Image-2（`gpt-image-2`）、Banana Nano 分组模型、模型 ID 路由规格 |
| 图生图 | 支持 1-8 张参考图；提示词原样提交；提交后自动清空当前参考图上传区；任务内部保留参考图快照 |
| 任务队列 | 后台执行、SSE 状态推送、轮询兜底、取消、重试、部分成功、重启恢复/中断标记 |
| 结果管理 | 本机落盘、预览、下载、复制图片/链接、作为参考图、PiXhost 自动/手动上传 |
| 用户系统 | 用户名密码注册登录、大小写用户名展示、2FA、每用户独立空间、旧空间导入 |
| Key 管理 | API Key 默认只保存在当前浏览器；可选风险确认后保存到账号空间；接口不返回明文 Key |
| 提示词助手 | 文字扩写、图片还原、灵感模式、多版本会话、继续对话修改、应用到生成表单 |
| Admin | NewAPI Base URL、公开访问域名、超时时间、Debug 日志开关 |
| 开源部署 | Go 单进程托管 API + 静态前端；支持本机、Linux systemd、宝塔/Nginx 反代 |

---

## 核心工作流

```text
注册/登录账号
  ↓
设置本地 Key 或确认后保存云端 Key
  ↓
写提示词 / 使用提示词助手
  ↓
选择文生图或图生图，设置模型和规格
  ↓
提交任务，后端立即返回任务并后台执行
  ↓
结果页通过 SSE/轮询更新进度
  ↓
预览、下载、复制、上传图床、作为下一次图生图参考
```

### 图生图行为

- 多参考图按上传列表顺序传给后端。
- 系统不会自动追加“主图”“融合方向”等内置提示词。
- 点击生成成功后，生成页参考图上传区会自动清空。
- 后端会先复制参考图快照到任务目录，所以清空上传区不会影响后台生成或后续重试。

### 模型返回提示词

结果页可能展示“模型返回提示词”。这不是本项目主动改写，也不是提示词助手生成，而是上游响应里的 `revised_prompt`。它只作为结果信息保留，不会覆盖原始提示词。

---

## 架构和数据流

```text
Browser / React UI
  │  same-origin /api
  ▼
Go local-server
  ├─ 用户账号 / Session / 2FA
  ├─ Admin 配置 / NewAPI Base URL / Debug
  ├─ 任务队列 / SSE / 重试 / 取消
  ├─ 上传参考图 / 图生图任务快照
  ├─ outputs 本机结果落盘
  └─ NewAPI / OpenAI-compatible image API
```

默认目录：

```text
data/                         用户、空间、任务、配置、参考图
outputs/                      生成图片
web/dist/                     前端生产构建产物
cmd/local-server              Go 服务入口
```

生产形态下 Go 后端同时负责：

- 托管 `web/dist` 静态前端；
- 提供 `/api/*`；
- 调用上游 NewAPI / OpenAI 兼容网关；
- 保存任务状态和图片结果；
- 对结果图片做登录鉴权。

---

## 部署教程索引

如果你不知道该从哪里开始，先看这一段。更完整的一页式导航见 [`docs/DEPLOYMENT.md`](docs/DEPLOYMENT.md)。

| 场景 | 入口 | 适合谁 |
| --- | --- | --- |
| 部署总索引 | [`docs/DEPLOYMENT.md`](docs/DEPLOYMENT.md) | 想先判断本机、Linux、宝塔、更新分别该看哪份文档 |
| 本机体验 / 开发 | [快速开始](#快速开始) | 想先跑起来看功能，或本地开发调试 |
| Linux 服务器正式部署 | [`docs/DEPLOY_LINUX.md`](docs/DEPLOY_LINUX.md) | 熟悉 SSH、systemd、Nginx/Caddy 的用户 |
| 宝塔面板部署 | [`docs/DEPLOY_BAOTA.md`](docs/DEPLOY_BAOTA.md) | 使用宝塔 Go 项目管理器和 Nginx 反代的用户 |
| 已部署后更新 | [已部署服务器一键更新](#已部署服务器一键更新) | 服务器已跑起来，只需要拉新代码、重构建、重启 |
| 开源发布检查 | [`docs/OPEN_SOURCE_CHECKLIST.md`](docs/OPEN_SOURCE_CHECKLIST.md) | 准备 fork、二次发布或公开部署文档前自查 |

推荐生产路径：

```text
准备域名和服务器
  → 按 Linux / 宝塔教程部署 Go 服务
  → 配置 Nginx / Caddy / 宝塔反代到 127.0.0.1:8787
  → 访问 /admin 配置 NewAPI Base URL
  → 注册普通账号并设置本地 Key 或云端 Key
  → 开始生成
```

---

## 快速开始

### 依赖

- Go 1.22+
- Node.js 20+
- npm

### 本机生产形态运行

```bash
git clone https://github.com/y08lin4/lyra-image-workbench.git
cd lyra-image-workbench

cd web
npm ci
npm run build
cd ..

go run ./cmd/local-server
```

打开：

```text
http://127.0.0.1:8787
```

首次进入：

1. 注册普通用户账号。
2. 访问 `/admin` 设置 Admin 密码。
3. 在 Admin 中配置 `NewAPI Base URL`。
4. 回到工作台设置页保存本地 `codex-key` / Banana Key。

### 前后端开发模式

终端 A：

```bash
go run ./cmd/local-server
```

终端 B：

```bash
cd web
npm ci
npm run dev
```

访问：

```text
http://127.0.0.1:5173
```

Vite 会把 `/api` 和 `/outputs` 代理到 `127.0.0.1:8787`。

---

## 生产部署

如果只是想快速跑起来，请先看 [快速开始](#快速开始)。如果要放到服务器长期运行，建议先看 [部署教程索引](#部署教程索引) 选择对应教程。

### Linux 构建

```bash
cd /opt/lyra-image-workbench
cd web && npm ci && npm run build && cd ..
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o image-workbench-local-server ./cmd/local-server
```

### 已部署服务器一键更新

适用于默认宝塔/systemd 路径 `/www/wwwroot/image-workbench`：

```bash
git config --global --add safe.directory /www/wwwroot/image-workbench && cd /www/wwwroot/image-workbench && git pull --ff-only origin master && cd web && npm ci && npm run build && cd .. && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o image-workbench-local-server.new ./cmd/local-server && chmod +x image-workbench-local-server.new && mv -f image-workbench-local-server.new image-workbench-local-server && chown -R www:www /www/wwwroot/image-workbench && systemctl restart image-workbench && systemctl status image-workbench --no-pager
```

看到下面状态说明服务正常：

```text
Active: active (running)
```

如果使用宝塔 Go 项目托管而不是 systemd，构建完成后在宝塔面板里重启 Go 项目即可。

更完整的部署说明：

- [`docs/DEPLOYMENT.md`](docs/DEPLOYMENT.md)：部署教程总索引，按本机、Linux、宝塔、更新场景选择入口。
- [`docs/DEPLOY_LINUX.md`](docs/DEPLOY_LINUX.md)：Linux、systemd、Nginx/Caddy、备份恢复。
- [`docs/DEPLOY_BAOTA.md`](docs/DEPLOY_BAOTA.md)：宝塔 Go 项目、Nginx 反代、常见问题。

---

## 环境变量

复制 `.env.example` 后按部署环境调整；不要提交真实 `.env`。

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `LOCAL_IMAGE_HOST` | `0.0.0.0` | 服务监听地址；反代部署建议 `127.0.0.1`。 |
| `LOCAL_IMAGE_PORT` | `8787` | 服务监听端口。 |
| `LOCAL_IMAGE_DATA_DIR` | `data` | 用户、任务、配置等运行时数据目录。 |
| `LOCAL_IMAGE_WEB_DIR` | `web/dist` | 前端生产构建目录。 |
| `NEWAPI_BASE_URL` | `http://127.0.0.1:3000/v1` | OpenAI 兼容图片网关地址。 |
| `NEWAPI_TIMEOUT_SEC` | `600` | 单张图片请求超时，允许范围 60-3600 秒。 |

运行时数据、上传参考图、生成结果、日志和 `.env` 均已在 `.gitignore` 中排除。

---

## 安全和 Key 策略

### 用户和空间

- 普通用户通过用户名 + 密码登录。
- 用户名支持大小写展示；登录和重复判断大小写不敏感。
- 每个用户绑定内部 `storageToken`，不通过登录接口暴露给前端。
- 支持 2FA：开启后登录需要验证器 App 的一次性验证码。
- 新账号可导入旧空间历史，避免迁移时丢失任务和结果。

### API Key

默认策略是**浏览器本地保存 Key**：

- `codex-key` 和 Banana Key 默认保存在当前浏览器 `localStorage`。
- 后端创建任务时通过临时请求头拿到本次 Key，并只在内存里为该任务暂存。
- `GET /api/config` 只返回是否已设置和脱敏预览，不返回明文。

可选策略是**确认风险后保存到云端账号空间**：

- 适合多设备登录后继续生成。
- 如果账号密码泄露，云端 Key 可能被使用或窃取。
- 保存云端 Key 前建议先开启 2FA。

### 输出图片

- 结果优先通过 `/api/background-tasks/{taskId}/images/{index}` 读取，并校验当前登录用户是否拥有该任务。
- 旧 `/outputs/{storageToken}/...` 地址也要求登录并校验空间归属。

### Debug 日志

Admin 开启 Debug 后，只对新任务生效。日志会显示请求阶段、上游状态、保存路径等信息，但会对 API Key 和长提示词做脱敏处理。

---

## 项目结构

```text
cmd/local-server        Go 服务入口
internal/api            HTTP 路由和鉴权
internal/jobs           任务队列、状态机、执行、重试、取消
internal/newapi         NewAPI / OpenAI 兼容图片接口客户端
internal/output         图片落盘与读取
internal/uploads        图生图参考图保存
internal/events         SSE 事件中心
internal/settings       Admin 全局配置
internal/spaceconfig    用户空间配置和 Key 状态
internal/users          用户注册、登录、2FA、会话
internal/adminauth      Admin 初始密码和登录
internal/prompttools    提示词助手服务和历史
web/                    React + TypeScript + Vite 前端
docs/                   部署、设计和迁移说明
scripts/                部署和本地重启脚本
```

---

## 开发和测试

```bash
# Go 测试
go test ./...

# 前端依赖和构建
cd web
npm ci
npm run build
```

CI 会在 push 和 PR 时运行：

- `go test ./...`
- `cd web && npm ci && npm run build`

---

## 更多文档

- [`docs/DEPLOYMENT.md`](docs/DEPLOYMENT.md)：部署教程总索引，按本机、Linux、宝塔、更新场景选择入口。
- [`docs/DEPLOY_LINUX.md`](docs/DEPLOY_LINUX.md)：Linux 服务器部署、systemd、Nginx/Caddy、升级和备份。
- [`docs/DEPLOY_BAOTA.md`](docs/DEPLOY_BAOTA.md)：宝塔面板 Go 项目部署教程。
- [`docs/CHANGES_FROM_AI_IMAGE_GENERATE.md`](docs/CHANGES_FROM_AI_IMAGE_GENERATE.md)：相较参考项目的架构和功能更新。
- [`docs/STACK.md`](docs/STACK.md)：Go 后端与 React/Vite 前端选型。
- [`docs/PROJECT_REQUIREMENTS.md`](docs/PROJECT_REQUIREMENTS.md)：项目模块化和稳定性要求。
- [`docs/OPEN_SOURCE_CHECKLIST.md`](docs/OPEN_SOURCE_CHECKLIST.md)：开源发布前检查清单。

---

## 开源协作

- 许可证：MIT，见 [`LICENSE`](LICENSE)。
- 贡献说明：见 [`CONTRIBUTING.md`](CONTRIBUTING.md)。
- 安全问题报告：见 [`SECURITY.md`](SECURITY.md)。

欢迎提交 issue / PR。提交前请确认没有包含 `data/`、`outputs/`、`.env`、日志、API Key、用户数据或生成图片隐私内容。
