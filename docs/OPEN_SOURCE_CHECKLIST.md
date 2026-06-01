# Open Source Checklist

这是本仓库作为开源项目发布前的维护清单。

## 已整理

- [x] 运行时数据目录 `data/` 已忽略。
- [x] 生成结果目录 `outputs/` 已忽略。
- [x] 构建产物 `bin/`、`web/dist/`、`*.tsbuildinfo` 已忽略。
- [x] 本地环境文件 `.env`、`.env.*` 已忽略，保留 `.env.example`。
- [x] 本地参考仓库 `_reference_*/`、`_project_remote/` 已忽略。
- [x] 添加 MIT `LICENSE`。
- [x] 添加 `CONTRIBUTING.md` 和 `SECURITY.md`。
- [x] 添加 GitHub Actions CI：Go 测试 + 前端构建。
- [x] 添加 `.editorconfig` 和 `.gitattributes`，统一文本格式。
- [x] README 使用通用部署说明，不硬编码个人服务域名。

## 发布前再次确认

- [ ] `git status --short` 干净或只包含预期改动。
- [ ] `git ls-files` 不包含真实 `data/`、`outputs/`、`.env`、日志、API Key 或用户数据。
- [ ] `go test ./...` 通过。
- [ ] `cd web && npm ci && npm run build` 通过。
- [ ] README 中的部署路径、服务名和二进制名与实际部署一致。
- [ ] 如果仓库从私有转公开，先检查历史提交中是否出现过密钥；如有，需要先轮换密钥并清理历史。

## 不应提交的内容

- API Key、Token、Cookie、账号密码和 2FA 密钥。
- `data/users.json`、`data/spaces/**`、`data/admin.auth.json`。
- `outputs/**` 和用户上传参考图。
- 本地日志、调试抓包、私有部署域名截图。
- 供应商私有接口文档或非公开上游配置。
