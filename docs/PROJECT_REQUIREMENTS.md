# 项目开发要求

## 总目标

本项目要做成本机运行的稳定生图工作台：Go 后端是唯一请求 NewAPI 的执行方，前端只通过同源 `/api` 提交任务和观察状态。核心目标是：即使生成过程持续 10 分钟，前端刷新、断线、手机锁屏也不应导致后端任务失败。

## 强制代码规范

1. 代码必须模块化，禁止把所有逻辑堆在一个文件里。
2. `cmd/` 只做进程入口、参数加载和启动服务，不写业务逻辑。
3. `internal/` 承载 Go 后端模块，每个模块只负责一类事情。
4. 前端 `web/src` 也必须拆分：页面、组件、API 客户端、hooks、类型、工具函数分开。
5. 路由层只做请求解析、响应输出和调用服务，不直接写 NewAPI 请求细节。
6. NewAPI 客户端、任务队列、任务存储、SSE、图片落盘必须是独立模块。
7. 任何新增功能都要配套更新相关文档，并用简体中文提交 git 记录。

## Go 后端建议模块结构

```text
cmd/local-server/
  main.go                 # 只启动程序

internal/config/
  config.go               # 环境变量、本机端口、内置 NewAPI 配置

internal/api/
  router.go               # 路由注册
  json.go                 # JSON 响应工具
  health.go               # 健康检查
  config_handler.go       # Key/配置接口
  task_handler.go         # 任务接口
  sse_handler.go          # SSE 事件接口

internal/newapi/
  client.go               # NewAPI HTTP 客户端
  image_response.go       # b64/url/image 响应解析

internal/jobs/
  queue.go                # 后台队列
  runner.go               # 任务执行
  progress.go             # 假流式进度模型
  types.go                # Job/Result 类型

internal/store/
  store.go                # 存储接口
  json_store.go           # 首版 JSON 存储
  sqlite_store.go         # 后续 SQLite 存储

internal/events/
  hub.go                  # SSE 订阅/广播

internal/output/
  files.go                # 图片落盘、outputs 路径安全
```

## 前端建议模块结构

```text
web/src/
  main.tsx
  App.tsx                 # 只做页面编排
  types.ts

  api/
    client.ts             # 同源 /api 请求封装
    tasks.ts              # 任务接口
    config.ts             # 配置接口

  components/
    SettingsPanel.tsx
    PromptPanel.tsx
    TaskQueue.tsx
    ResultGrid.tsx
    HistoryPanel.tsx
    ImagePreview.tsx

  hooks/
    useTaskEvents.ts      # SSE 自动重连
    useTaskList.ts        # 任务列表刷新
    useResponsiveLayout.ts

  lib/
    ratios.ts
    format.ts
    files.ts
```

## 10 分钟稳定性要求

后续实现必须遵守：

- 创建任务接口必须快速返回，不能等待 NewAPI 出图。
- 后台任务不能继承前端 HTTP 请求的取消上下文。
- SSE 只是观察通道，断开不能取消任务。
- 任务状态必须持久化。
- 图片生成成功后先保存本机磁盘，再做任何可选上传。
- 多图任务要逐张落盘，允许部分成功。
- 默认上游请求超时要覆盖 10 分钟目标，建议 15 分钟以上。
