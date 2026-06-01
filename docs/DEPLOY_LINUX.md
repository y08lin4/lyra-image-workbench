# Linux 服务器部署教程

本文档用于把本项目部署到 Linux 服务器上，以「后端 Go 托管前端 `web/dist`，前端只访问同源 `/api`，后端再请求 NewAPI / CLIProxyAPI」为生产形态。

推荐部署方式：

- 应用只监听 `127.0.0.1:8787`。
- 外部访问走 Nginx / Caddy 反向代理和 HTTPS。
- NewAPI / CLIProxyAPI 也尽量只监听内网或本机地址。
- 图片和空间数据放到 `/var/lib/image-workbench`，方便备份迁移。

---

## 1. 部署结构

推荐目录：

```text
/opt/image-workbench/                  # 项目代码和二进制
  image-workbench-local-server          # Go 构建产物
  web/dist/                             # 前端生产构建产物
  outputs -> /var/lib/image-workbench/outputs

/var/lib/image-workbench/
  data/                                 # 空间、配置、任务、Admin 密码
  outputs/                              # 生成结果图片
```

当前版本注意点：

- `LOCAL_IMAGE_DATA_DIR` 可以通过环境变量指定。
- `LOCAL_IMAGE_WEB_DIR` 可以通过环境变量指定。
- `outputs` 目前由后端以相对路径 `outputs/` 写入，所以生产环境建议在应用工作目录下建立软链接：

```bash
/opt/image-workbench/outputs -> /var/lib/image-workbench/outputs
```

这样 systemd 的 `WorkingDirectory=/opt/image-workbench` 不变，图片也能集中落到 `/var/lib/image-workbench/outputs`。

---

## 2. 服务器依赖

以 Ubuntu / Debian 为例。

### 2.1 基础工具

```bash
sudo apt update
sudo apt install -y git curl ca-certificates build-essential nginx
```

### 2.2 Go

项目要求 Go `1.22+`。

检查：

```bash
go version
```

如果系统源版本太旧，建议从 Go 官方安装包安装到 `/usr/local/go`，并确认：

```bash
export PATH=/usr/local/go/bin:$PATH
go version
```

### 2.3 Node.js / npm

前端构建建议使用 Node.js `20+` 或当前 LTS。

检查：

```bash
node -v
npm -v
```

---

## 3. 创建运行用户和目录

```bash
sudo useradd --system --home /var/lib/image-workbench --shell /usr/sbin/nologin imagewb || true

sudo mkdir -p /opt/image-workbench
sudo mkdir -p /var/lib/image-workbench/data
sudo mkdir -p /var/lib/image-workbench/outputs

sudo chown -R imagewb:imagewb /var/lib/image-workbench
```

---

## 4. 拉取代码

```bash
cd /opt
sudo git clone https://github.com/y08lin4/lyra-image-workbench.git image-workbench
cd /opt/image-workbench
```

如果目录已经存在：

```bash
cd /opt/image-workbench
sudo git pull
```

---

## 5. 构建前端和后端

### 5.1 构建前端

```bash
cd /opt/image-workbench/web
sudo npm ci
sudo npm run build
```

构建完成后应生成：

```text
/opt/image-workbench/web/dist/index.html
```

### 5.2 构建后端

```bash
cd /opt/image-workbench
sudo go test ./...
sudo go build -trimpath -ldflags="-s -w" -o /opt/image-workbench/image-workbench-local-server ./cmd/local-server
```

### 5.3 建立 outputs 软链接

```bash
cd /opt/image-workbench
sudo rm -rf outputs
sudo ln -s /var/lib/image-workbench/outputs outputs
```

---

## 6. 环境变量说明

后端支持这些关键环境变量：

| 变量 | 默认值 | 说明 |
|---|---:|---|
| `LOCAL_IMAGE_HOST` | `127.0.0.1` | 后端监听地址。生产建议保持 `127.0.0.1`，由 Nginx/Caddy 对外。 |
| `LOCAL_IMAGE_PORT` | `8787` | 后端监听端口。 |
| `LOCAL_IMAGE_DATA_DIR` | `data` | 空间、Admin、任务、Key 配置的存储目录。生产建议 `/var/lib/image-workbench/data`。 |
| `LOCAL_IMAGE_WEB_DIR` | `web/dist` | 前端静态文件目录。 |
| `NEWAPI_BASE_URL` | `http://127.0.0.1:3000/v1` | 初始 NewAPI / CLIProxyAPI 地址，也可后续在 `/admin` 页面改。 |
| `NEWAPI_TIMEOUT_SEC` | `600` | 上游生图请求超时，默认 600 秒。允许范围 60-3600。 |

说明：

- `/admin` 页面保存的 URL 和超时时间会写入 `data/config.local.json`。
- 普通用户的 `codex-key` 会写入个人空间配置，只向前端返回掩码。
- Banana Nano 使用单独的 Banana API Key。请在 NewAPI / CLIProxyAPI 中新建一个 `banana` 分组的 apikey，再在工作台设置窗口填写；URL 仍复用 `/admin` 配置的 NewAPI URL。
- 模型当前固定：
  - 生图：`gpt-image-2`
  - Banana Nano：通过规格选择路由到 `gemini-3.1-flash-image-preview...` 系列模型 ID
  - 提示词助手：`gpt-5.5`

---

## 7. systemd 服务

创建服务文件：

```bash
sudo tee /etc/systemd/system/image-workbench.service >/dev/null <<'EOF'
[Unit]
Description=Lyra Image Workbench
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=imagewb
Group=imagewb
WorkingDirectory=/opt/image-workbench

Environment=LOCAL_IMAGE_HOST=127.0.0.1
Environment=LOCAL_IMAGE_PORT=8787
Environment=LOCAL_IMAGE_DATA_DIR=/var/lib/image-workbench/data
Environment=LOCAL_IMAGE_WEB_DIR=/opt/image-workbench/web/dist
Environment=NEWAPI_BASE_URL=http://127.0.0.1:3000/v1
Environment=NEWAPI_TIMEOUT_SEC=600

ExecStart=/opt/image-workbench/image-workbench-local-server
Restart=always
RestartSec=3

NoNewPrivileges=true
PrivateTmp=true
ProtectHome=true
ProtectSystem=full
ReadWritePaths=/var/lib/image-workbench

[Install]
WantedBy=multi-user.target
EOF
```

授权运行用户读取项目目录：

```bash
sudo chown -R root:root /opt/image-workbench
sudo chmod -R a+rX /opt/image-workbench
sudo chown -R imagewb:imagewb /var/lib/image-workbench
```

启动：

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now image-workbench
sudo systemctl status image-workbench --no-pager
```

查看日志：

```bash
sudo journalctl -u image-workbench -f
```

本机健康检查：

```bash
curl -i http://127.0.0.1:8787/api/health
```

预期返回 `HTTP/1.1 200 OK`。

---

## 8. Nginx 反向代理

生产环境推荐由 Nginx 对外监听 80/443，应用仍只监听 `127.0.0.1:8787`。

创建配置：

```bash
sudo tee /etc/nginx/sites-available/image-workbench >/dev/null <<'EOF'
server {
    listen 80;
    server_name your-domain.com;

    client_max_body_size 64m;

    location / {
        proxy_pass http://127.0.0.1:8787;
        proxy_http_version 1.1;

        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header Connection "";

        # SSE / 长任务状态推送：不要缓冲，不要过早断开。
        proxy_buffering off;
        proxy_cache off;
        proxy_read_timeout 700s;
        proxy_send_timeout 700s;
        proxy_connect_timeout 60s;
    }
}
EOF
```

启用：

```bash
sudo ln -sfn /etc/nginx/sites-available/image-workbench /etc/nginx/sites-enabled/image-workbench
sudo nginx -t
sudo systemctl reload nginx
```

如果使用 HTTPS，建议再用 certbot 或已有证书接入 TLS。上游生图任务本身由后端后台执行，Nginx 主要影响的是页面访问和 SSE 状态连接；即使前端断开，后端任务仍应继续跑。

---

## 9. Caddy 反向代理可选

如果用 Caddy，可以使用类似配置：

```text
your-domain.com {
    request_body {
        max_size 64MB
    }

    reverse_proxy 127.0.0.1:8787 {
        flush_interval -1
        transport http {
            read_timeout 700s
            write_timeout 700s
        }
    }
}
```

---

## 10. 防火墙

如果只通过 Nginx/Caddy 对外，通常只开放 80/443：

```bash
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw enable
sudo ufw status
```

不推荐直接把应用端口 `8787` 暴露到公网。若确实需要直接访问：

```bash
sudo ufw allow 8787/tcp
```

同时必须确认 `/admin` 已设置强密码，并且所有个人空间密码足够强。

---

## 11. 首次访问配置

浏览器打开：

```text
http://your-domain.com/
```

首次配置顺序：

1. 进入 `/admin`。
2. 首次访问设置 Admin 管理密码。
3. 设置 NewAPI / CLIProxyAPI URL，例如：

```text
http://127.0.0.1:3000/v1
```

4. 设置超时时间，建议保持：

```text
600
```

5. 回到 `/`，输入空间密码创建个人空间。
6. 在设置窗口填写 `codex-key`。
7. 提交文生图 / 图生图任务测试。

提示词助手会复用当前空间里的 `codex-key`，并通过同一个 NewAPI / CLIProxyAPI 地址调用 `gpt-5.5`。

---

## 12. CLIProxyAPI / NewAPI 部署关系

如果服务器上同时跑 CLIProxyAPI，建议：

```text
外部浏览器
  -> Nginx/Caddy
  -> image-workbench:127.0.0.1:8787
  -> CLIProxyAPI/NewAPI:127.0.0.1:3000/v1
  -> 上游模型
```

也就是说：

- 浏览器不直接请求 CLIProxyAPI / NewAPI。
- 前端只请求当前站点同源 `/api`。
- Go 后端用空间里的 `codex-key` 请求 CLIProxyAPI / NewAPI。
- CLIProxyAPI / NewAPI 能不暴露公网就不要暴露公网。

---

## 13. 升级流程

```bash
cd /opt/image-workbench

sudo git pull

cd /opt/image-workbench/web
sudo npm ci
sudo npm run build

cd /opt/image-workbench
sudo go test ./...
sudo go build -trimpath -ldflags="-s -w" -o /opt/image-workbench/image-workbench-local-server ./cmd/local-server

sudo systemctl restart image-workbench
sudo systemctl status image-workbench --no-pager
```

升级后检查：

```bash
curl -i http://127.0.0.1:8787/api/health
sudo journalctl -u image-workbench -n 100 --no-pager
```

---

## 14. 备份和迁移

需要备份的核心数据：

```text
/var/lib/image-workbench/data
/var/lib/image-workbench/outputs
```

备份：

```bash
sudo tar -czf /root/image-workbench-backup-$(date +%F).tar.gz /var/lib/image-workbench
```

恢复：

```bash
sudo systemctl stop image-workbench
sudo tar -xzf /root/image-workbench-backup-YYYY-MM-DD.tar.gz -C /
sudo chown -R imagewb:imagewb /var/lib/image-workbench
sudo systemctl start image-workbench
```

---

## 15. 常见问题

### 15.1 页面能打开，但提示词助手返回 HTTP 405

通常是旧后端还在跑，或者 Nginx 代理到了旧服务。

检查当前服务：

```bash
sudo systemctl status image-workbench --no-pager
sudo ss -lntp | grep 8787
```

重新构建并重启：

```bash
cd /opt/image-workbench
sudo git pull
sudo go build -trimpath -ldflags="-s -w" -o /opt/image-workbench/image-workbench-local-server ./cmd/local-server
sudo systemctl restart image-workbench
```

路由探测：

```bash
curl -i -X POST http://127.0.0.1:8787/api/prompt-tools/text-to-prompt \
  -H 'Content-Type: application/json' \
  --data '{"input":"route-check","style":"auto","ratio":"auto","language":"zh","target":"image-2"}'
```

只要不是 `405`，就说明最新 POST 路由已经生效。没有空间 token 时返回 `400` 属于正常探测结果。

### 15.2 生成 10 分钟任务时前端断开

这是允许的。任务在 Go 后端后台运行，不继承前端 HTTP request context。前端刷新后会重新拉任务快照和结果。

但反向代理仍建议设置：

```text
proxy_buffering off
proxy_read_timeout 700s
proxy_send_timeout 700s
```

这样 SSE 状态连接更稳定。

### 15.3 上传参考图失败

检查 Nginx：

```text
client_max_body_size 64m;
```

当前应用限制：

- 单张参考图最大约 `12MB`。
- 单次上传总大小约 `50MB`。
- 支持 `PNG / JPG / WEBP`。

### 15.4 生成成功但看不到图片

检查 `outputs` 软链接和权限：

```bash
ls -la /opt/image-workbench/outputs
ls -la /var/lib/image-workbench/outputs
sudo chown -R imagewb:imagewb /var/lib/image-workbench
sudo systemctl restart image-workbench
```

### 15.5 Admin 密码忘了

Admin 鉴权文件在：

```text
/var/lib/image-workbench/data/admin.auth.json
```

如果确认要重置：

```bash
sudo systemctl stop image-workbench
sudo rm /var/lib/image-workbench/data/admin.auth.json
sudo systemctl start image-workbench
```

然后重新访问 `/admin` 设置新密码。

### 15.6 个人空间密码忘了

个人空间密码用于派生/进入空间。忘记后无法直接恢复原空间密码。建议通过备份恢复数据，或创建新空间重新填写 `codex-key`。

---

## 16. 最小验收清单

部署后按顺序验收：

```bash
curl -i http://127.0.0.1:8787/api/health
sudo journalctl -u image-workbench -n 100 --no-pager
```

浏览器验收：

1. `/admin` 可设置/登录管理密码。
2. `/admin` 可保存 NewAPI URL 和 `600s` 超时。
3. `/` 可创建个人空间。
4. 设置窗口可保存 `codex-key`，刷新后只显示掩码。
5. 文生图任务提交后立即进入队列。
6. 图生图参考图可上传并创建任务。
7. 任务执行中刷新页面，任务和进度仍可恢复。
8. 生成结果图片可下载、复制、预览、作为图生图参考图。
9. 提示词助手可调用 `gpt-5.5`，且错误显示中文、错误码、英文。
