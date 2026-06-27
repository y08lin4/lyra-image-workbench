# 多架构构建和一键安装

项目通过 GitHub Actions 在推送版本 tag 时自动构建 Release 包和 Docker 镜像。

## 触发发版

tag 必须符合 `v*.*.*`：

```bash
git tag v0.1.0
git push origin v0.1.0
```

触发后会运行：

- `.github/workflows/release.yml`：构建 GitHub Release 二进制包。
- `.github/workflows/docker.yml`：构建并推送 GHCR Docker 镜像。

## Release 二进制架构

Release 会包含这些包：

| 系统 | 架构 | 文件 |
| --- | --- | --- |
| Linux | amd64 | `lyra-image-workbench-vX.Y.Z-linux-amd64.tar.gz` |
| Linux | arm64 | `lyra-image-workbench-vX.Y.Z-linux-arm64.tar.gz` |
| Linux | armv7 | `lyra-image-workbench-vX.Y.Z-linux-armv7.tar.gz` |
| macOS | amd64 | `lyra-image-workbench-vX.Y.Z-darwin-amd64.tar.gz` |
| macOS | arm64 | `lyra-image-workbench-vX.Y.Z-darwin-arm64.tar.gz` |
| Windows | amd64 | `lyra-image-workbench-vX.Y.Z-windows-amd64.zip` |
| Windows | arm64 | `lyra-image-workbench-vX.Y.Z-windows-arm64.zip` |

每个包都包含：

```text
lyra-image-workbench     后端可执行文件，Windows 为 .exe
web/dist/                前端静态文件
docs/                    部署文档
README.md
LICENSE
.env.example
```

Release 还会生成 `checksums.txt`，用于校验下载文件。

## Linux 一键安装

安装脚本会自动识别 Linux 架构，目前支持：

- `amd64`
- `arm64`
- `armv7`

默认安装最新 Release：

```bash
curl -fsSL https://raw.githubusercontent.com/y08lin4/lyra-image-workbench/master/scripts/install.sh | sudo bash
```

安装完成后脚本会输出“安装令牌”，首次打开 `/admin` 初始化站点时必须填写；请立即保存该令牌。

指定版本：

```bash
curl -fsSL https://raw.githubusercontent.com/y08lin4/lyra-image-workbench/master/scripts/install.sh | sudo env VERSION=v0.1.0 bash
```

指定 NewAPI 地址、监听端口：

```bash
curl -fsSL https://raw.githubusercontent.com/y08lin4/lyra-image-workbench/master/scripts/install.sh | sudo env \
  VERSION=v0.1.0 \
  NEWAPI_BASE_URL=http://127.0.0.1:3000/v1 \
  HOST=127.0.0.1 \
  PORT=8787 \
  bash
```

默认安装位置：

```text
/opt/lyra-image-workbench/current           当前版本软链接
/opt/lyra-image-workbench/releases/         历史版本目录
/var/lib/lyra-image-workbench/data          用户、Admin、任务、配置、参考图
/var/lib/lyra-image-workbench/outputs       生成结果图片
/etc/systemd/system/lyra-image-workbench.service
```

安装脚本常用环境变量：

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `VERSION` | `latest` | 安装最新 Release 或指定版本，例如 `v0.1.0`。 |
| `INSTALL_DIR` | `/opt/lyra-image-workbench` | 程序和历史版本目录。 |
| `DATA_ROOT` | `/var/lib/lyra-image-workbench` | 运行时数据和输出图片目录。 |
| `SERVICE_NAME` | `lyra-image-workbench` | systemd 服务名。 |
| `RUN_USER` | `lyra-image-workbench` | systemd 运行用户。 |
| `HOST` | `127.0.0.1` | 服务监听地址；反代部署建议保持默认。 |
| `PORT` | `8787` | 服务监听端口。 |
| `LOCAL_IMAGE_ADMIN_SETUP_TOKEN` | 自动生成 | 首次 `/admin` 初始化站点必填；也可以安装前自行指定。 |
| `NEWAPI_BASE_URL` | `http://127.0.0.1:3000/v1` | OpenAI 兼容图片网关地址。 |

安装后常用命令：

```bash
systemctl status lyra-image-workbench --no-pager
journalctl -u lyra-image-workbench -f
systemctl restart lyra-image-workbench
```

再次执行同一条安装命令即可更新。脚本会下载对应版本 Release 包，切换 `/opt/lyra-image-workbench/current`，并重启 systemd 服务。

## Docker 多架构镜像

Docker workflow 会构建：

- `linux/amd64`
- `linux/arm64`

镜像地址：

```text
ghcr.io/y08lin4/lyra-image-workbench:latest
```

Docker 部署教程见 [`DOCKER.md`](DOCKER.md)。
