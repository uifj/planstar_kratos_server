#!/usr/bin/env bash
# =====================================================================
# Planstar 一键推送部署脚本
#   本地构建镜像 -> 推送 GHCR -> SSH 到远程单服务器 -> compose 拉取并重启
#
# 用法:
#   cp .env.example .env      # 填写 SERVER_HOST / PLANSTAR_IMAGE 等
#   chmod +x deploy.sh
#   ./deploy.sh
#
# 服务器一次性准备（首次手动执行）:
#   1) 安装 Docker + Compose v2:
#        curl -fsSL https://get.docker.com | sh
#   2) 克隆仓库（保持 docs/sql 相对路径可用）:
#        git clone git@github.com:<owner>/planstar_kratos_server.git /opt/planstar
#   3) 登录镜像仓库（如需拉取私有 GHCR）:
#        echo <token> | docker login ghcr.io -u <user> --password-stdin
#   4) 配置免密 SSH（或改用密码，见下方 SSH 段）
# =====================================================================
set -euo pipefail
cd "$(dirname "$0")"

# 加载 .env（若存在）
if [ -f .env ]; then
  set -a; . ./.env; set +a
fi

REGISTRY="${PLANSTAR_IMAGE:-ghcr.io/uifj/planstar-api}"
TAG="${PLANSTAR_TAG:-latest}"
IMAGE="${REGISTRY}:${TAG}"

SERVER_HOST="${SERVER_HOST:?请在 .env 中配置 SERVER_HOST}"
SERVER_USER="${SERVER_USER:-root}"
SERVER_PORT="${SERVER_PORT:-22}"
APP_DIR="${APP_DIR:-/opt/planstar}"

echo "==> 构建镜像 ${IMAGE}"
docker build --build-arg VERSION="${VERSION:-dev}" -t "${IMAGE}" .

echo "==> 推送镜像 ${IMAGE}"
docker push "${IMAGE}"

echo "==> 同步代码到 ${SERVER_USER}@${SERVER_HOST}:${APP_DIR}"
ssh -p "${SERVER_PORT}" "${SERVER_USER}@${SERVER_HOST}" \
  "cd ${APP_DIR} && git pull --ff-only"

echo "==> 远程拉取镜像并重启服务"
ssh -p "${SERVER_PORT}" "${SERVER_USER}@${SERVER_HOST}" "cd ${APP_DIR} && \
  export PLANSTAR_IMAGE=${REGISTRY} PLANSTAR_TAG=${TAG} && \
  docker compose -f planstar/docker-compose.yml pull && \
  docker compose -f planstar/docker-compose.yml up -d --remove-orphans"

echo "==> 部署完成: http://${SERVER_HOST}:8098"