# Contributing

感谢你考虑参与本项目。

## 开发环境

- Go 1.22+
- Node.js 20+ / npm

```bash
go test ./...
cd web
npm ci
npm run build
```

## 提交建议

- 保持改动小而清晰，一个 PR 只解决一个主要问题。
- 不要提交 `data/`、`outputs/`、`.env`、日志、构建产物或任何 API Key。
- 涉及前端交互时，请说明手动验证步骤。
- 涉及后端任务、鉴权、上传或输出路径时，请补充 Go 测试。

## 安全和隐私

请不要在 issue、PR、截图或日志里公开 API Key、用户数据、任务数据、生成图片原始私有链接或服务器域名中的敏感信息。
