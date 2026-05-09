# image-Workbench-Localhost-Version

本项目是一个运行在本机的 AI 生图工作台：浏览器只连接 `127.0.0.1`，本机 Go 后端负责请求内置 NewAPI 地址，并通过本地任务队列和假流式 SSE 保证前端不断流。

## 技术栈

- 后端：Go，本机 HTTP 服务，后续承载任务队列、SSE、NewAPI 请求和本地图片落盘。
- 前端：React + TypeScript + Vite，优先做响应式网站，后续复用到 Wails / Capacitor。

## 开发命令

```bash
# 后端
go run ./cmd/local-server

# 前端
cd web
npm install
npm run dev
```

## 设计文档

- `docs/ROUTES.md`：参考项目路由与本地化调整。
- `docs/STACK.md`：Go 后端与 React/Vite 前端选型。
