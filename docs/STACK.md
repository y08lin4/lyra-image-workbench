# 技术栈选型

## 结论

- 后端：Go。
- 前端：React + TypeScript + Vite。
- Web 形态：Go 后端同时托管前端静态资源和 `/api`，前端只使用同源相对路径。
- 后续桌面 App：优先 Wails，因为 Wails 原生就是 Go 后端 + Web 前端。
- 后续手机 App：优先 Capacitor，把同一套 React/Vite 前端打包成 Android/iOS WebView App；如果仍要求手机端也由本机后端代理请求，再评估 gomobile 或改成 App 内原生请求层。

## 为什么前端选 React + Vite

1. 参考项目 `ai-image-generate-private` 已经是 React + Vite，页面、组件、状态模型和路由习惯可以直接参考。
2. Vite 开发快，构建产物就是静态文件，Go 可以很容易 `embed` 或直接托管 `web/dist`。
3. React 生态对响应式工作台、图片预览、任务队列、移动端底栏等 UI 很成熟。
4. 后续封装路线清晰：
   - 网站/PWA：直接复用。
   - 桌面 App：Wails 复用 Go 后端和 React 前端。
   - 手机 App：Capacitor 复用 React 前端。

## 请求边界

前端不直接请求 NewAPI，也不硬编码本机监听地址。前端只请求同源相对接口：

```text
POST /api/background-tasks
GET  /api/background-tasks/:id
GET  /api/background-tasks/:id/events
```

Go 后端负责：

```text
读取当前空间里的 Image-2 Key
  -> 使用内置 NewAPI Base URL
  -> 创建独立后台任务
  -> 请求 NewAPI
  -> 下载/解码图片
  -> 保存到 outputs/
  -> 通过本机 API/SSE 暴露任务状态和本地图片 URL
```

## 关键架构约束

前端连接不能绑定真实生图任务生命周期。

正确模型：

```text
前端 POST /api/background-tasks
  -> Go 后端创建 job 并立即返回 job_id
  -> Go worker 使用独立后台 context 请求 NewAPI
  -> 前端通过 EventSource 连接 /api/background-tasks/:id/events 观察进度
  -> 前端断开只丢观察连接，不取消 job
  -> 前端刷新后用 job_id 重新查询/重连
```

错误模型：

```text
前端直接请求 NewAPI
前端硬编码 http://localhost:xxxx 请求本机后端
前端 POST /api/generate 并一直等上游返回
```

## 本项目前端页面策略

首版不引入重型 UI 框架，先用 React + CSS Grid/Flex 做响应式：

- 桌面端：左侧参数栏 + 中间结果区 + 右侧历史/任务栏。
- 手机端：顶部简化配置 + 结果流 + 底部任务/历史抽屉。
- SSE 断线自动重连，重连后先 `GET /api/background-tasks/:id` 拉快照。
- 图片只展示 Go 后端提供的同源 URL，例如 `/outputs/2026-05-09/xxx.png`。

## 后端实现原则

- Go 后端是唯一请求 NewAPI 的进程。
- 内置 NewAPI Base URL，用户只填 Image-2 Key；后续 Gemini Banana 等模型再增加独立 Key 槽位。
- 任务进入本地队列，worker 独立执行。
- SSE 是假流式状态通道，不承载真实上游长请求。
- 图片落盘到 `outputs/`，任务状态保存到 `data/`，后续可升级 SQLite。
