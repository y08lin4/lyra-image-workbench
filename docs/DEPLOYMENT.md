# 部署教程索引

这份文档用于快速判断“我应该看哪一个部署教程”。如果只是本地体验，看 README 的快速开始即可；如果要对外长期运行，优先按 Linux 或宝塔教程部署。

## 选择路径

| 目标 | 推荐入口 | 说明 |
| --- | --- | --- |
| 本机体验 / 二次开发 | [`README.md#快速开始`](../README.md#快速开始) | 最短路径跑起前后端，适合先看功能或本地开发。 |
| Release 一键安装 | [`RELEASE.md`](RELEASE.md) | Linux 服务器自动识别 amd64/arm64/armv7，下载 Release 包并安装 systemd 服务。 |
| Docker 镜像部署 | [`DOCKER.md`](DOCKER.md) | 使用 GHCR 镜像、Docker Compose、容器数据卷和反代部署。 |
| Linux 服务器正式部署 | [`DEPLOY_LINUX.md`](DEPLOY_LINUX.md) | 推荐生产方案：systemd 托管 Go 服务，Nginx/Caddy 做 HTTPS 反代。 |
| 宝塔面板部署 | [`DEPLOY_BAOTA.md`](DEPLOY_BAOTA.md) | 适合使用宝塔 Go 项目管理器和宝塔 Nginx 的服务器。 |
| 已部署后更新 | [`README.md#已部署服务器一键更新`](../README.md#已部署服务器一键更新) | 服务器已跑起来，只需要拉新代码、重新构建并重启服务。 |
| 开源发布前自查 | [`OPEN_SOURCE_CHECKLIST.md`](OPEN_SOURCE_CHECKLIST.md) | 检查仓库、文档、示例配置和敏感文件是否适合公开。 |

## 部署前准备

- 一台 Linux 服务器，或已安装宝塔面板的服务器。
- 一个域名，建议开启 HTTPS。
- Go `1.22+`、Node.js `20+`、npm、Git。
  - 如果直接使用 Docker 镜像，不需要在宿主机安装 Go 和 Node.js。
- 可用的 OpenAI 兼容图片接口 / NewAPI Base URL。
  - 不想自建网关时，可查看 API 服务入口：[https://ai-cf.ailinyu.de/](https://ai-cf.ailinyu.de/)。
- 明确运行时数据目录，至少包含：
  - `data/`：用户、Admin、任务、配置、参考图。
  - `outputs/`：生成结果图片。

> 不要把真实 `.env`、API Key、`data/`、`outputs/`、日志或用户数据提交到仓库。

## 标准上线流程

```text
拉取代码
  → 构建 web/dist
  → 构建 Go 后端二进制
  → 配置 data/outputs 写入权限
  → 用 systemd / 宝塔 Go 项目托管后端
  → 用 Nginx / Caddy / 宝塔反代到 127.0.0.1:8787
  → 访问 /admin 配置 NewAPI Base URL
  → 注册普通账号并设置本地 Key 或云端 Key
```

Docker 部署可以跳过本地构建步骤，直接拉取 `ghcr.io/y08lin4/lyra-image-workbench:latest`，然后挂载 `/app/data` 和 `/app/outputs`。

Release 一键安装也可以跳过本地构建步骤，直接运行：

```bash
curl -fsSL https://raw.githubusercontent.com/y08lin4/lyra-image-workbench/master/scripts/install.sh | sudo bash
```

## 推荐生产结构

```text
Browser
  → HTTPS 域名
  → Nginx / Caddy / 宝塔反代
  → 127.0.0.1:8787
  → Lyra Image Workbench Go 服务
  → NewAPI / OpenAI-compatible image API
```

生产环境建议让应用只监听 `127.0.0.1`，不要直接把 Go 服务暴露到公网端口。

## 更新前检查

更新服务前建议先确认：

1. `git status` 干净，避免服务器上有未提交的手改文件。
2. 已备份 `data/` 和 `outputs/`。
3. 当前服务有可回滚的旧二进制或旧提交号。
4. 更新后能访问 `/admin` 和普通工作台页面。

默认宝塔/systemd 路径的一键更新命令见 [`README.md#已部署服务器一键更新`](../README.md#已部署服务器一键更新)。如果你的目录或 systemd 服务名不同，请先替换命令里的路径和服务名。

## GIF / FFmpeg note

The `/gif` page can generate animation frames without FFmpeg, but final GIF merging requires the server to run patched `ffmpeg` version `8.1.2` or newer. Older or unparsable versions are treated as unavailable. For bare binary, systemd, or panel deployments, install FFmpeg on the host and verify:

```bash
ffmpeg -version
```

Set `FFMPEG_BIN=/path/to/ffmpeg` if the executable is not on `PATH`, or `GIF_ENABLED=false` to disable final GIF rendering while keeping normal image generation available.
