# image-Workbench-Localhost-Version

本项目是一个运行在本机的 AI 生图工作台：浏览器只连接 `127.0.0.1`，本机后端负责请求内置 NewAPI 地址，并通过本地任务队列和假流式事件保证前端不断流。

详细实现会参考 `ai-image-generate-private` 的路由命名，并改成本机稳定任务模式。
