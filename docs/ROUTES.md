# 路由参考与本地化设计

参考项目：`https://github.com/y08lin4/ai-image-generate-private`

## 参考路由

- `GET /api/health`
- `POST /api/generate-stream`
- `POST /api/upload-pixhost`
- `GET /api/image-proxy`
- `GET /api/stats`
- `POST /api/background-tasks`
- `GET /api/background-tasks`
- `GET /api/background-tasks/:id`
- `POST /api/background-tasks/:id/retry`
- `GET /api/background-tasks/:id/images/:index`

## 本地版保留与调整

本地版不再依赖 Cloudflare Worker、D1、Workflow，也不需要空间密码。浏览器只访问本机服务，API Key 保存在本机 `data/config.local.json` 中。

- 保留 `GET /api/health`：用于前端探活。
- 保留 `POST /api/background-tasks`：创建本机后台生图任务，立即返回任务 ID。
- 保留 `GET /api/background-tasks`：查看本机历史任务。
- 保留 `GET /api/background-tasks/:id`：查看单个任务状态。
- 新增 `GET /api/background-tasks/:id/events`：本机 SSE 假流式进度、心跳、结果推送。
- 保留 `POST /api/background-tasks/:id/retry`：失败任务重试。
- 保留 `GET /api/background-tasks/:id/images/:index`：读取本机落盘图片。
- 保留 `GET /api/stats`：本机统计。
- 保留兼容 `POST /api/generate-stream`：创建本机任务并在同一个本地连接里推送 SSE。
- 保留兼容 `GET /api/image-proxy?url=...`：由本机拉取远程图片，避免 CORS。
- 暂不实现 `POST /api/upload-pixhost`：本地版默认保存到本机 `outputs/`，不主动上传第三方图床。

## 稳定性原则

- 前端创建任务后立即返回，不直接等待上游出图。
- 上游请求只在本机 worker 中执行，浏览器刷新不会取消任务。
- SSE 只连接 localhost，定时发送 heartbeat，避免 UI 误判断流。
- 图片结果统一保存到本地 `outputs/YYYY-MM-DD/`，前端展示本地 URL。
- Key 不出现在日志和提交记录里。
