#!/usr/bin/env bash
set -euo pipefail

APP_DIR="${APP_DIR:-/www/wwwroot/image-workbench}"
REPO_URL="${REPO_URL:-git@github.com:y08lin4/lyra-image-workbench.git}"
BRANCH="${BRANCH:-master}"
RUN_USER="${RUN_USER:-www}"
LOCAL_IMAGE_HOST="${LOCAL_IMAGE_HOST:-127.0.0.1}"
LOCAL_IMAGE_PORT="${LOCAL_IMAGE_PORT:-8787}"
NEWAPI_BASE_URL="${NEWAPI_BASE_URL:-http://127.0.0.1:3000/v1}"
NEWAPI_TIMEOUT_SEC="${NEWAPI_TIMEOUT_SEC:-600}"
SKIP_TEST="${SKIP_TEST:-0}"

log() {
  printf '\033[1;34m[image-workbench]\033[0m %s\n' "$*"
}

need_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "缺少命令：$1，请先在宝塔安装对应环境。" >&2
    exit 1
  fi
}

need_cmd git
need_cmd go
need_cmd npm

log "部署目录：${APP_DIR}"
mkdir -p "$(dirname "$APP_DIR")"

if [ -d "${APP_DIR}/.git" ]; then
  log "检测到已有仓库，拉取最新代码"
  cd "$APP_DIR"
  git fetch origin "$BRANCH"
  git checkout "$BRANCH"
  git pull --ff-only origin "$BRANCH"
else
  if [ -e "$APP_DIR" ] && [ "$(ls -A "$APP_DIR" 2>/dev/null || true)" ]; then
    echo "目录 ${APP_DIR} 已存在且不是空目录，也不是 Git 仓库。请先备份或清空该目录。" >&2
    exit 1
  fi
  log "克隆仓库：${REPO_URL}"
  git clone --branch "$BRANCH" "$REPO_URL" "$APP_DIR"
  cd "$APP_DIR"
fi

log "构建前端 web/dist"
cd "${APP_DIR}/web"
if [ -f package-lock.json ]; then
  npm ci
else
  npm install
fi
npm run build

cd "$APP_DIR"
if [ "$SKIP_TEST" != "1" ]; then
  log "运行 Go 测试"
  go test ./...
else
  log "已跳过 Go 测试：SKIP_TEST=1"
fi

log "构建 Linux Go 后端"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o image-workbench-local-server ./cmd/local-server
chmod +x image-workbench-local-server

log "创建数据目录"
mkdir -p "${APP_DIR}/data" "${APP_DIR}/outputs"

if id "$RUN_USER" >/dev/null 2>&1; then
  log "授权给运行用户：${RUN_USER}"
  chown -R "${RUN_USER}:${RUN_USER}" "$APP_DIR"
else
  log "未找到运行用户 ${RUN_USER}，跳过 chown。请在宝塔里确认运行用户。"
fi

cat > "${APP_DIR}/baota.env.example" <<EOF
LOCAL_IMAGE_HOST=${LOCAL_IMAGE_HOST}
LOCAL_IMAGE_PORT=${LOCAL_IMAGE_PORT}
LOCAL_IMAGE_DATA_DIR=${APP_DIR}/data
LOCAL_IMAGE_WEB_DIR=${APP_DIR}/web/dist
NEWAPI_BASE_URL=${NEWAPI_BASE_URL}
NEWAPI_TIMEOUT_SEC=${NEWAPI_TIMEOUT_SEC}
EOF

log "部署准备完成"
cat <<EOF

请到宝塔「网站 -> Go 项目 -> 添加 Go 项目」填写：

项目执行文件：
${APP_DIR}/image-workbench-local-server

项目名称：
image-workbench

项目端口：
${LOCAL_IMAGE_PORT}

执行命令：
./image-workbench-local-server

运行用户：
${RUN_USER}

环境变量选择「指定变量」，填入：
$(cat "${APP_DIR}/baota.env.example")

如果绑定域名/走 Nginx 反代，LOCAL_IMAGE_HOST 保持 127.0.0.1。
如果直接用 服务器IP:${LOCAL_IMAGE_PORT} 访问，把 LOCAL_IMAGE_HOST 改成 0.0.0.0 并放行端口。

EOF
