# image-Workbench-Localhost-Version

本项目是一个运行在本机的 AI 生图工作台：Go 后端同时托管前端页面和 `/api` 接口。前端代码只使用同源相对路径，不写 `localhost`，也不直接请求 NewAPI；真正的 NewAPI 请求由 Go 后端任务队列执行，并通过假流式 SSE 把状态推给页面。

## 技术栈

- 后端：Go，本机 HTTP 服务，后续承载任务队列、SSE、NewAPI 请求和本地图片落盘。
- 前端：React + TypeScript + Vite，优先做响应式网站，后续复用到 Wails / Capacitor。

## 开发命令

```bash
# 后端
go run ./cmd/local-server

# 前端开发服务器，仅开发期使用 Vite 代理 /api 到 Go 后端
cd web
npm install
npm run dev
```

生产/打包形态下由 Go 后端直接托管 `web/dist`，前端访问 `/api/...` 即可，不需要知道后端监听地址。

## 设计文档

- `docs/ROUTES.md`：参考项目路由与本地化调整。
- `docs/STACK.md`：Go 后端与 React/Vite 前端选型。
- `docs/PROJECT_REQUIREMENTS.md`：项目模块化和 10 分钟稳定性硬性要求。
- `docs/REFERENCE_PROJECT_ANALYSIS.md`：参考项目路由、后台任务和稳定性缺口分析。
- `docs/CLOSED_LOOP_DESIGN.md`：当前未闭环点、最终闭环设计和实现顺序。`n- `docs/SPACE_DESIGN.md`：个人空间、空间密码、固定模型和图生图第一版设计。

