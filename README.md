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

# 前端生产构建
cd web
npm run build
```

生产/打包形态下由 Go 后端直接托管 `web/dist`，前端访问 `/api/...` 即可，不需要知道后端监听地址。

## 使用闭环

1. 运行 `go run ./cmd/local-server` 启动本机后端，默认监听 `127.0.0.1:8787`。
2. 打开工作台，输入空间密码创建/进入个人空间。
3. 在设置窗口填写用于 `gpt-image-2` 的 codex-key；如果要使用 Banana Nano，请在 NewAPI / CLIProxyAPI 里新建一个 `banana` 分组的 apikey，并在设置窗口单独填写 Banana API Key。两类 Key 后端只向前端返回掩码。
4. 首次打开 `/admin` 需要先设置 Admin 管理密码；后续访问 Admin 页面需要重新输入管理密码。
5. 在 `/admin` 配置 NewAPI URL 和请求超时时间，默认超时为 `600s`，模型固定为 `gpt-image-2`。
6. 回到 `/` 提交文生图或图生图任务。前端只请求同源 `/api`，后端后台任务继续执行，刷新页面后可从本机 JSON 状态和 `outputs/` 结果恢复。

图片规格里比例支持 `自动`。选择自动比例时，后端不会把它强行映射成方图尺寸，而是让上游按自动尺寸处理。Banana Nano 的比例和清晰度不作为参数发送，而是通过规格选择路由到对应模型 ID。

结果区支持下载、复制图片到剪贴板、复制 URL、全屏预览和“作为图生图参考图”。全屏预览会读取图片真实像素尺寸、实际宽高比和文件大小。

设置窗口可开启“生成成功后自动上传到 PiXhost 图床”。关闭自动上传时，单张结果图仍可在悬浮操作里手动点击“上传图床”；PiXhost 单张最大 10MB，失败不会影响本机原图保存。

## 设计文档

- `docs/DEPLOY_LINUX.md`：Linux 服务器部署、systemd、Nginx/Caddy、升级和备份教程。
- `docs/ROUTES.md`：参考项目路由与本地化调整。
- `docs/STACK.md`：Go 后端与 React/Vite 前端选型。
- `docs/PROJECT_REQUIREMENTS.md`：项目模块化和 10 分钟稳定性硬性要求。
- `docs/REFERENCE_PROJECT_ANALYSIS.md`：参考项目路由、后台任务和稳定性缺口分析。
- `docs/CLOSED_LOOP_DESIGN.md`：当前未闭环点、最终闭环设计和实现顺序。
- `docs/SPACE_DESIGN.md`：个人空间、空间密码、固定模型和图生图第一版设计。

