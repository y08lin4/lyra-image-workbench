#!/usr/bin/env bash
set -euo pipefail

REPO="${REPO:-y08lin4/lyra-image-workbench}"
VERSION="${VERSION:-latest}"
APP_NAME="${APP_NAME:-lyra-image-workbench}"
SERVICE_NAME="${SERVICE_NAME:-lyra-image-workbench}"
RUN_USER="${RUN_USER:-lyra-image-workbench}"
RUN_GROUP="${RUN_GROUP:-}"
INSTALL_DIR="${INSTALL_DIR:-/opt/lyra-image-workbench}"
DATA_ROOT="${DATA_ROOT:-/var/lib/lyra-image-workbench}"
HOST="${HOST:-127.0.0.1}"
PORT="${PORT:-8787}"
NEWAPI_BASE_URL="${NEWAPI_BASE_URL:-http://127.0.0.1:3000/v1}"
NEWAPI_TIMEOUT_SEC="${NEWAPI_TIMEOUT_SEC:-600}"

log() {
  printf '\033[1;34m==>\033[0m %s\n' "$*"
}

die() {
  printf '\033[1;31mERROR:\033[0m %s\n' "$*" >&2
  exit 1
}

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || die "缺少命令：$1"
}

curl_json() {
  if [ -n "${GITHUB_TOKEN:-}" ]; then
    curl -fsSL -H "Authorization: Bearer ${GITHUB_TOKEN}" "$1"
  else
    curl -fsSL "$1"
  fi
}

detect_sudo() {
  if [ "${EUID}" -eq 0 ]; then
    SUDO=""
  elif command -v sudo >/dev/null 2>&1; then
    SUDO="sudo"
  else
    die "请用 root 运行，或安装 sudo 后重试"
  fi
}

detect_arch() {
  case "$(uname -s)" in
    Linux) ;;
    *) die "一键安装脚本目前只支持 Linux；其他系统请从 GitHub Releases 手动下载对应包" ;;
  esac

  case "$(uname -m)" in
    x86_64 | amd64) ARCH="amd64" ;;
    aarch64 | arm64) ARCH="arm64" ;;
    armv7l | armv7) ARCH="armv7" ;;
    *) die "暂不支持当前架构：$(uname -m)" ;;
  esac
}

resolve_version() {
  if [ "${VERSION}" = "latest" ]; then
    log "查询最新版本"
    VERSION="$(curl_json "https://api.github.com/repos/${REPO}/releases/latest" | sed -nE 's/.*"tag_name"[[:space:]]*:[[:space:]]*"([^"]+)".*/\1/p' | head -n1)"
    [ -n "${VERSION}" ] || die "无法获取最新版本；如果仓库还没有 Release，请先推送 v*.*.* tag 触发发版"
  fi
}

install_user_and_dirs() {
  if ! id -u "${RUN_USER}" >/dev/null 2>&1; then
    log "创建运行用户 ${RUN_USER}"
    local nologin
    nologin="/usr/sbin/nologin"
    [ -x /sbin/nologin ] && nologin="/sbin/nologin"
    ${SUDO} useradd --system --user-group --home-dir "${DATA_ROOT}" --shell "${nologin}" "${RUN_USER}"
  fi
  if [ -z "${RUN_GROUP}" ]; then
    RUN_GROUP="$(id -gn "${RUN_USER}")"
  fi

  ${SUDO} mkdir -p "${INSTALL_DIR}/releases" "${DATA_ROOT}/data" "${DATA_ROOT}/outputs"
  ${SUDO} chown -R "${RUN_USER}:${RUN_GROUP}" "${DATA_ROOT}"
  ${SUDO} chmod 755 "${INSTALL_DIR}" "${INSTALL_DIR}/releases"
}

download_and_install() {
  local asset url tmp archive unpack release_dir
  asset="${APP_NAME}-${VERSION}-linux-${ARCH}.tar.gz"
  url="https://github.com/${REPO}/releases/download/${VERSION}/${asset}"
  tmp="$(mktemp -d)"
  archive="${tmp}/${asset}"
  unpack="${tmp}/package"
  release_dir="${INSTALL_DIR}/releases/${VERSION}-linux-${ARCH}"

  trap "rm -rf '${tmp}'" EXIT

  log "下载 ${url}"
  curl -fL --retry 3 --retry-delay 2 -o "${archive}" "${url}"

  mkdir -p "${unpack}"
  tar -xzf "${archive}" -C "${unpack}"

  log "安装到 ${release_dir}"
  ${SUDO} rm -rf "${release_dir}.tmp"
  ${SUDO} mkdir -p "${release_dir}.tmp"
  ${SUDO} cp -a "${unpack}/." "${release_dir}.tmp/"
  ${SUDO} chmod +x "${release_dir}.tmp/${APP_NAME}"
  ${SUDO} ln -sfn "${DATA_ROOT}/outputs" "${release_dir}.tmp/outputs"
  ${SUDO} chown -R root:root "${release_dir}.tmp"
  ${SUDO} rm -rf "${release_dir}"
  ${SUDO} mv "${release_dir}.tmp" "${release_dir}"
  ${SUDO} ln -sfn "${release_dir}" "${INSTALL_DIR}/current"
}

write_systemd_service() {
  need_cmd systemctl

  log "写入 systemd 服务 ${SERVICE_NAME}.service"
  cat <<EOF | ${SUDO} tee "/etc/systemd/system/${SERVICE_NAME}.service" >/dev/null
[Unit]
Description=Lyra Image Workbench
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=${RUN_USER}
Group=${RUN_GROUP}
WorkingDirectory=${INSTALL_DIR}/current
Environment=LOCAL_IMAGE_HOST=${HOST}
Environment=LOCAL_IMAGE_PORT=${PORT}
Environment=LOCAL_IMAGE_DATA_DIR=${DATA_ROOT}/data
Environment=LOCAL_IMAGE_WEB_DIR=${INSTALL_DIR}/current/web/dist
Environment=NEWAPI_BASE_URL=${NEWAPI_BASE_URL}
Environment=NEWAPI_TIMEOUT_SEC=${NEWAPI_TIMEOUT_SEC}
ExecStart=${INSTALL_DIR}/current/${APP_NAME}
Restart=on-failure
RestartSec=3
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=full
ReadWritePaths=${DATA_ROOT}

[Install]
WantedBy=multi-user.target
EOF

  ${SUDO} systemctl daemon-reload
  ${SUDO} systemctl enable --now "${SERVICE_NAME}"
  ${SUDO} systemctl restart "${SERVICE_NAME}"
}

main() {
  need_cmd curl
  need_cmd tar
  need_cmd sed
  need_cmd uname
  detect_sudo
  detect_arch
  resolve_version
  install_user_and_dirs
  download_and_install
  write_systemd_service

  log "安装完成：${APP_NAME} ${VERSION} linux/${ARCH}"
  printf '\n访问地址：\n'
  printf '  http://127.0.0.1:%s\n' "${PORT}"
  printf '\n常用命令：\n'
  printf '  systemctl status %s --no-pager\n' "${SERVICE_NAME}"
  printf '  journalctl -u %s -f\n' "${SERVICE_NAME}"
  printf '\n如需反代公开访问，建议让服务保持监听 127.0.0.1:%s，再由 Nginx/Caddy/宝塔提供 HTTPS。\n' "${PORT}"
}

main "$@"
