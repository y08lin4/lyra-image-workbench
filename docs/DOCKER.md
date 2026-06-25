# Docker 镜像部署

本项目提供多阶段 `Dockerfile`，GitHub Actions 会把镜像发布到 GitHub Container Registry：

```text
ghcr.io/y08lin4/lyra-image-workbench
```

如果首次发布后外部机器无法拉取，请到 GitHub 仓库的 Packages 页面确认该 GHCR package 已设置为公开可见。

## 镜像标签

工作流触发规则：

- push 到 `master`：构建并推送 `latest`、`master`、`sha-xxxxxxx`。
- push tag，例如 `v1.0.0`：构建并推送 `1.0.0`、`1.0`、`sha-xxxxxxx`。
- PR：只构建验证，不推送镜像。
- 支持手动 `workflow_dispatch`。

## 方式一：docker run

```bash
docker run -d \
  --name lyra-image-workbench \
  --restart unless-stopped \
  -p 8787:8787 \
  -v lyra-image-workbench-data:/app/data \
  -v lyra-image-workbench-outputs:/app/outputs \
  ghcr.io/y08lin4/lyra-image-workbench:latest
```

打开：

```text
http://127.0.0.1:8787
```

首次进入后：

1. 注册普通账号。
2. 访问 `/admin` 设置 Admin 密码。
3. 在 Admin 中配置 `NewAPI Base URL`。
4. 回到工作台设置页保存本地 Key 或确认后保存云端 Key。

### 连接宿主机上的 NewAPI

如果 NewAPI 跑在宿主机 `127.0.0.1:3000`，容器内不能直接用 `127.0.0.1` 访问宿主机。Linux Docker 可这样运行：

```bash
docker run -d \
  --name lyra-image-workbench \
  --restart unless-stopped \
  --add-host=host.docker.internal:host-gateway \
  -p 8787:8787 \
  -e NEWAPI_BASE_URL=http://host.docker.internal:3000/v1 \
  -v lyra-image-workbench-data:/app/data \
  -v lyra-image-workbench-outputs:/app/outputs \
  ghcr.io/y08lin4/lyra-image-workbench:latest
```

更新 `docker run` 部署：

```bash
docker pull ghcr.io/y08lin4/lyra-image-workbench:latest
docker rm -f lyra-image-workbench
docker run -d \
  --name lyra-image-workbench \
  --restart unless-stopped \
  --add-host=host.docker.internal:host-gateway \
  -p 8787:8787 \
  -e NEWAPI_BASE_URL=http://host.docker.internal:3000/v1 \
  -v lyra-image-workbench-data:/app/data \
  -v lyra-image-workbench-outputs:/app/outputs \
  ghcr.io/y08lin4/lyra-image-workbench:latest
```

命名卷 `lyra-image-workbench-data` 和 `lyra-image-workbench-outputs` 不会因为删除容器而丢失。

## 方式二：Docker Compose

新建目录和 `compose.yml`：

```bash
mkdir -p /opt/lyra-image-workbench-docker
cd /opt/lyra-image-workbench-docker
cat > compose.yml <<'EOF'
services:
  lyra-image-workbench:
    image: ghcr.io/y08lin4/lyra-image-workbench:latest
    container_name: lyra-image-workbench
    restart: unless-stopped
    ports:
      - "127.0.0.1:8787:8787"
    environment:
      LOCAL_IMAGE_HOST: 0.0.0.0
      LOCAL_IMAGE_PORT: 8787
      NEWAPI_BASE_URL: http://host.docker.internal:3000/v1
      NEWAPI_TIMEOUT_SEC: 600
      GIF_ENABLED: "true"
      FFMPEG_BIN: ffmpeg
      GIF_WORK_DIR: /app/data/gif_work
    extra_hosts:
      - "host.docker.internal:host-gateway"
    volumes:
      - ./data:/app/data
      - ./outputs:/app/outputs
EOF
```

或者手动保存为：

```yaml
services:
  lyra-image-workbench:
    image: ghcr.io/y08lin4/lyra-image-workbench:latest
    container_name: lyra-image-workbench
    restart: unless-stopped
    ports:
      - "127.0.0.1:8787:8787"
    environment:
      LOCAL_IMAGE_HOST: 0.0.0.0
      LOCAL_IMAGE_PORT: 8787
      NEWAPI_BASE_URL: http://host.docker.internal:3000/v1
      NEWAPI_TIMEOUT_SEC: 600
      GIF_ENABLED: "true"
      FFMPEG_BIN: ffmpeg
      GIF_WORK_DIR: /app/data/gif_work
    extra_hosts:
      - "host.docker.internal:host-gateway"
    volumes:
      - ./data:/app/data
      - ./outputs:/app/outputs
```

启动：

```bash
docker compose up -d
```

如果前面还有 Nginx / Caddy / 宝塔反代，建议保持 `127.0.0.1:8787:8787`，只让反代服务对公网暴露 HTTPS。

更新 Compose 部署：

```bash
cd /opt/lyra-image-workbench-docker
docker compose pull
docker compose up -d
docker image prune -f
```

## 数据目录和权限

容器内默认目录：

```text
/app/data       用户、Admin、任务、配置、参考图
/app/outputs    生成结果图片
/app/web/dist   前端静态文件
```

镜像默认使用非 root 用户运行，UID/GID 为 `10001`。如果使用宿主机 bind mount，必要时先授权：

```bash
mkdir -p data outputs
sudo chown -R 10001:10001 data outputs
```

## 本地构建镜像

```bash
docker build -t lyra-image-workbench:local .
```

运行本地镜像：

```bash
docker run --rm -p 8787:8787 \
  -v "$(pwd)/data:/app/data" \
  -v "$(pwd)/outputs:/app/outputs" \
  lyra-image-workbench:local
```

## GIF / FFmpeg

The Docker image installs `ffmpeg` in the Alpine runtime stage, and the backend verifies it before allowing final GIF merging. FFmpeg is still treated as an external command dependency: it is not bundled into the Go binary and cgo is not used. The runtime FFmpeg must report version `8.1.2` or newer; older or unparsable versions disable only GIF merging.

If you build a custom runtime image, include:

```dockerfile
RUN apk add --no-cache ffmpeg
```

Useful environment variables:

```yaml
environment:
  GIF_ENABLED: "true"
  FFMPEG_BIN: ffmpeg
  GIF_WORK_DIR: /app/data/gif_work
  GIF_MAX_FRAMES: 24
  GIF_MAX_FPS: 15
  GIF_MAX_SIZE: 1024
  GIF_RENDER_TIMEOUT_SEC: 60
```

If `ffmpeg -version` is not available inside the container, or if the reported version is below `8.1.2`, `/gif` can still generate frames but the merge button is disabled.
