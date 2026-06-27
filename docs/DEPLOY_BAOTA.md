# 宝塔面板部署教程

本文档按宝塔面板「网站 / Go 项目 / 添加 Go 项目」的方式部署本项目。核心思路：

- Go 后端作为唯一服务进程。
- Go 后端托管 `web/dist` 前端。
- 前端只请求同源 `/api`。
- 后端再通过 NewAPI / CLIProxyAPI 请求 `gpt-image-2`、Banana Nano 和 `gpt-5.5`。
- 如果绑定域名访问，应用推荐只监听 `127.0.0.1:8787`，由宝塔/Nginx 反代。

---

## 1. 推荐目录

建议放在：

```text
/www/wwwroot/image-workbench/
  image-workbench-local-server     # Linux 后端可执行文件
  web/dist/                        # 前端构建产物
  data/                            # 空间、Admin、任务、Key 配置
  outputs/                         # 本机生成图片
```

注意：

- `data/` 和 `outputs/` 需要给宝塔运行用户 `www` 写入权限。
- 不要把 Windows 上的 `.exe` 直接传到 Linux 跑，必须在 Linux 服务器上重新 `go build`。

---

## 2. 宝塔安装依赖

在宝塔面板里安装：

- Nginx
- Go 项目管理器 / Go 环境，要求 Go `1.22+`
- Node.js，建议 Node `20+`
- Git

也可以在服务器终端检查：

```bash
go version
node -v
npm -v
git --version
```

---

## 3. 拉取和构建项目

### 3.1 私有仓库怎么拉

如果这个 GitHub 仓库是私有项目，服务器不能直接 `git clone https://github.com/...`，需要先给服务器授权。

推荐用 **SSH Deploy Key**，不要把你的 GitHub 主账号密码或长期 Token 写进命令里。

在服务器终端执行：

```bash
ssh-keygen -t ed25519 -C "image-workbench-baota" -f ~/.ssh/image_workbench_deploy
cat ~/.ssh/image_workbench_deploy.pub
```

复制输出的公钥，然后到 GitHub：

```text
仓库页面 -> Settings -> Deploy keys -> Add deploy key
```

填写：

```text
Title: baota-image-workbench
Key: 粘贴刚才 cat 出来的公钥
Allow write access: 不勾选
```

然后在服务器配置 SSH：

```bash
cat >> ~/.ssh/config <<'EOF'
Host github-image-workbench
  HostName github.com
  User git
  IdentityFile ~/.ssh/image_workbench_deploy
  IdentitiesOnly yes
EOF

chmod 600 ~/.ssh/config
ssh -T github-image-workbench
```

看到类似 `successfully authenticated` 就可以拉私有仓库。

私有仓库 clone 地址用这个：

```bash
git clone git@github-image-workbench:y08lin4/lyra-image-workbench.git /www/wwwroot/image-workbench
```

> 不推荐把 GitHub Token 直接写到 `https://token@github.com/...` 里，因为容易留在 shell 历史、宝塔日志或 `.git/config`。

### 3.2 一键脚本方式

如果已经配置好 Deploy Key，可以先 clone，再运行项目内置脚本：

```bash
cd /www/wwwroot
git clone git@github-image-workbench:y08lin4/lyra-image-workbench.git image-workbench
cd /www/wwwroot/image-workbench
bash scripts/deploy-baota.sh
```

如果仓库是公开仓库，也可以直接：

```bash
cd /www/wwwroot
git clone https://github.com/y08lin4/lyra-image-workbench.git image-workbench
cd /www/wwwroot/image-workbench
bash scripts/deploy-baota.sh
```

脚本会自动：

- 拉取/更新代码。
- 构建 `web/dist`。
- 构建 Linux 后端二进制。
- 创建 `data/`、`outputs/`。
- 授权给 `www` 用户。
- 生成首次 `/admin` 初始化站点必填的安装令牌，并写入 `baota.env.example`。
- 输出宝塔「添加 Go 项目」应该填写的字段和安装令牌；请立即保存该令牌。

如果你想让应用直接监听公网端口，可以这样跑脚本：

```bash
LOCAL_IMAGE_HOST=0.0.0.0 bash scripts/deploy-baota.sh
```

如果要跳过测试：

```bash
SKIP_TEST=1 bash scripts/deploy-baota.sh
```

### 3.3 手动方式

进入宝塔终端或 SSH：

```bash
cd /www/wwwroot
git clone git@github-image-workbench:y08lin4/lyra-image-workbench.git image-workbench
cd /www/wwwroot/image-workbench
```

如果已经拉过：

```bash
cd /www/wwwroot/image-workbench
git pull
```

构建前端：

```bash
cd /www/wwwroot/image-workbench/web
npm ci
npm run build
```

构建后端：

```bash
cd /www/wwwroot/image-workbench
go test ./...
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o image-workbench-local-server ./cmd/local-server
chmod +x image-workbench-local-server
```

创建运行目录并授权：

```bash
mkdir -p /www/wwwroot/image-workbench/data
mkdir -p /www/wwwroot/image-workbench/outputs
chown -R www:www /www/wwwroot/image-workbench
```

---

## 4. 添加 Go 项目：按截图字段填写

宝塔路径一般是：

```text
网站 -> Go 项目 -> 添加 Go 项目
```

按窗口字段填写：

| 宝塔字段 | 推荐填写 |
|---|---|
| 项目执行文件 | `/www/wwwroot/image-workbench/image-workbench-local-server` |
| 项目名称 | `image-workbench` |
| 项目端口 | `8787` |
| 执行命令 | `./image-workbench-local-server` |
| 环境变量 | 选择「指定变量」 |
| 运行用户 | `www` |
| 开机启动 | 勾选 |
| 项目备注 | `Lyra Image Workbench` |
| 绑定域名 | 你的域名，例如 `img.example.com` |

### 4.1 环境变量

选择「指定变量」，填入：

```text
LOCAL_IMAGE_HOST=127.0.0.1
LOCAL_IMAGE_PORT=8787
LOCAL_IMAGE_DATA_DIR=/www/wwwroot/image-workbench/data
LOCAL_IMAGE_WEB_DIR=/www/wwwroot/image-workbench/web/dist
NEWAPI_BASE_URL=http://127.0.0.1:3000/v1
NEWAPI_TIMEOUT_SEC=600
```

说明：

- 如果你使用宝塔绑定域名/Nginx 反代，推荐 `LOCAL_IMAGE_HOST=127.0.0.1`。
- 如果你不绑定域名，想直接用 `服务器IP:8787` 访问，则改成：

```text
LOCAL_IMAGE_HOST=0.0.0.0
```

并且勾选「放行端口」或手动放行 `8787`。

### 4.2 放行端口怎么选

推荐：

- 绑定域名访问：**不要勾选放行端口**，只开放 80/443。
- 直接 IP:8787 访问：勾选「放行端口」。

生产环境更推荐绑定域名 + HTTPS，不建议直接暴露 `8787`。

---

## 5. 宝塔 Nginx 配置建议

如果宝塔自动生成了站点/反代配置，建议在对应站点的 Nginx 配置里确认这些参数。

重点是：

- 支持 SSE。
- 上传参考图不被 Nginx 拦截。
- 10 分钟任务期间前端连接不被过早切断。

可在 `server { ... }` 内加入：

```nginx
client_max_body_size 64m;
```

在代理到 `127.0.0.1:8787` 的 `location / { ... }` 内加入或确认：

```nginx
proxy_pass http://127.0.0.1:8787;
proxy_http_version 1.1;

proxy_set_header Host $host;
proxy_set_header X-Real-IP $remote_addr;
proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
proxy_set_header X-Forwarded-Proto $scheme;
proxy_set_header Connection "";

proxy_buffering off;
proxy_cache off;
proxy_read_timeout 700s;
proxy_send_timeout 700s;
proxy_connect_timeout 60s;
```

保存后在宝塔里重载 Nginx。

---

## 6. 首次访问配置

访问：

```text
https://你的域名/
```

首次配置顺序：

1. 打开 `/admin`。
2. 首次访问设置 Admin 管理密码。
3. 设置 NewAPI / CLIProxyAPI URL，例如：

```text
http://127.0.0.1:3000/v1
```

4. 设置超时时间：

```text
600
```

5. 回到 `/` 创建个人空间。
6. 打开设置窗口：
   - 填写 `codex-key`，用于 Image-2 和提示词助手。
   - 如果要用 Banana Nano，请在 NewAPI / CLIProxyAPI 里新建一个 `banana` 分组的 apikey，然后填写 Banana API Key。
7. 提交文生图 / 图生图任务测试。

---

## 7. Banana Nano 特别说明

Banana Nano 的 URL 不变，仍使用 `/admin` 里的 NewAPI URL。

但是 Key 单独保存：

```text
Image-2 / 提示词助手 -> codex-key
Banana Nano -> Banana API Key
```

Banana 的比例和清晰度不是请求参数，而是路由到对应模型 ID，例如：

```text
gemini-3.1-flash-image-preview
gemini-3.1-flash-image-preview-16x9-4k
gemini-3.1-flash-image-preview-9x16-4k
gemini-3.1-flash-image-preview-1x1-2k
```

所以在工作台里选 Banana 规格即可，不需要在 Admin 改 URL。

---

## 8. 升级流程

宝塔面板里先停止 Go 项目。

如果你用一键脚本，执行：

```bash
cd /www/wwwroot/image-workbench
bash scripts/deploy-baota.sh
```

如果你手动升级，执行：

```bash
cd /www/wwwroot/image-workbench
git pull

cd /www/wwwroot/image-workbench/web
npm ci
npm run build

cd /www/wwwroot/image-workbench
go test ./...
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o image-workbench-local-server ./cmd/local-server
chmod +x image-workbench-local-server
chown -R www:www /www/wwwroot/image-workbench
```

然后回宝塔面板启动 Go 项目。

如果页面还是旧 UI：

1. 确认 `web/dist` 已重新构建。
2. 确认 Go 项目已重启。
3. 浏览器强刷。
4. 如果开了 CDN，清 CDN 缓存。

---

## 9. 验收命令

服务器本机检查：

```bash
curl -i http://127.0.0.1:8787/api/health
```

预期：

```text
HTTP/1.1 200 OK
```

提示词助手路由检查：

```bash
curl -i -X POST http://127.0.0.1:8787/api/prompt-tools/text-to-prompt \
  -H 'Content-Type: application/json' \
  --data '{"input":"route-check","style":"auto","ratio":"auto","language":"zh","target":"image-2"}'
```

说明：

- 没带空间 token 时返回 `400` 是正常的。
- 如果返回 `405`，说明宝塔还在跑旧二进制或代理到了旧服务。

---

## 10. 常见问题

### 10.1 502 Bad Gateway

排查：

```bash
ps aux | grep image-workbench
ss -lntp | grep 8787
curl -i http://127.0.0.1:8787/api/health
```

常见原因：

- Go 项目没启动。
- 项目端口填错。
- `LOCAL_IMAGE_HOST=127.0.0.1`，但宝塔配置代理到了其他地址。
- 二进制不是 Linux 版本。

### 10.2 权限错误，无法保存配置或图片

执行：

```bash
chown -R www:www /www/wwwroot/image-workbench/data
chown -R www:www /www/wwwroot/image-workbench/outputs
```

### 10.3 上传参考图失败

确认 Nginx 有：

```nginx
client_max_body_size 64m;
```

应用自身限制：

- 单张参考图最大约 `12MB`。
- 单次上传总大小约 `50MB`。
- 支持 `PNG / JPG / WEBP`。

### 10.4 任务 10 分钟内前端断开

后端任务不会因为前端断开而失败；刷新页面后可恢复任务状态。

但 Nginx 建议配置：

```nginx
proxy_buffering off;
proxy_read_timeout 700s;
proxy_send_timeout 700s;
```

### 10.5 HTTP 405

通常是旧后端还在跑。

处理：

```bash
cd /www/wwwroot/image-workbench
git pull
go build -trimpath -ldflags="-s -w" -o image-workbench-local-server ./cmd/local-server
chmod +x image-workbench-local-server
chown -R www:www /www/wwwroot/image-workbench
```

然后在宝塔里重启 Go 项目。

### 10.6 Admin 密码忘记

停止 Go 项目后删除：

```bash
rm /www/wwwroot/image-workbench/data/admin.auth.json
```

再启动项目，重新进入 `/admin` 设置密码。

---

## 11. 最小验收清单

部署完成后按这个顺序测：

1. 域名能打开首页。
2. `/admin` 能设置/登录管理密码。
3. `/admin` 能保存 NewAPI URL 和 `600` 秒超时。
4. `/` 能创建个人空间。
5. 设置窗口能保存 `codex-key`。
6. 设置窗口能保存 Banana API Key。
7. Image-2 文生图能提交任务。
8. Banana 能选择规格并提交任务。
9. 图生图能上传参考图并提交任务。
10. 刷新页面后历史任务和结果仍在。
11. 提示词助手能生成提示词，并能选择应用到 Image-2 或 Banana。



> 必须设置 LOCAL_IMAGE_ADMIN_SETUP_TOKEN，首次打开 /admin 时填写该令牌后再设置管理密码。
