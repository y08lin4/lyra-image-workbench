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

本地版不再依赖 Cloudflare Worker、D1、Workflow，但保留“空间密码/个人空间”体验。Go 后端同时托管前端页面和 API，前端只使用同源相对路径；管理配置保存在本机 `data/config.local.json` 中。

- 保留 `GET /api/health`：用于前端探活。`n- 新增 `POST /api/spaces/session` / `GET /api/spaces/session` / `DELETE /api/spaces/session`：创建、恢复、退出个人空间。`n- 新增 `POST /api/uploads/reference`：第一版图生图参考图上传到本机空间目录。
- 保留 `POST /api/background-tasks`：创建本机后台生图任务，立即返回任务 ID。
- 保留 `GET /api/background-tasks`：查看本机历史任务。
- 保留 `GET /api/background-tasks/:id`：查看单个任务状态。
- 新增 `GET /api/background-tasks/:id/events`：本机 SSE 假流式进度、心跳、结果推送。
- 保留 `POST /api/background-tasks/:id/retry`：失败任务重试。
- 保留 `GET /api/background-tasks/:id/images/:index`：读取本机落盘图片。
- 保留 `GET /api/stats`：本机统计。
- 保留兼容 `POST /api/generate-stream`：创建本机任务并在同一个同源连接里推送 SSE。
- 保留兼容 `GET /api/image-proxy?url=...`：由 Go 后端拉取远程图片，避免 CORS。
- 暂不实现 `POST /api/upload-pixhost`：本地版默认保存到本机 `outputs/`，不主动上传第三方图床。

## 稳定性原则

- 前端创建任务后立即返回，不直接等待上游出图。
- 上游请求只在 Go worker 中执行，浏览器刷新不会取消任务。
- SSE 只是同源状态观察通道，定时发送 heartbeat，避免 UI 误判断流。
- 图片结果统一保存到本地 `outputs/YYYY-MM-DD/`，前端展示同源图片 URL。
- Key 不出现在日志和提交记录里。

