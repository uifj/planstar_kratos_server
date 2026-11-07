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

# ---- 运维令牌落盘（Prometheus 抓取 9108/metrics 的 Bearer 令牌）----
# Prometheus 无法在配置中展开环境变量，故把 $PLANSTAR_OPS_TOKEN 写入
# deploy/prometheus/ops_token（bind mount 到容器，已加入 .gitignore）。
OPS_TOKEN_FILE="deploy/prometheus/ops_token"
mkdir -p deploy/prometheus
if [ -n "${PLANSTAR_OPS_TOKEN:-}" ]; then
  ( umask 077; printf '%s' "${PLANSTAR_OPS_TOKEN}" > "${OPS_TOKEN_FILE}" )
  chmod 600 "${OPS_TOKEN_FILE}" 2>/dev/null || true
  # 尽力归属当前用户；容器只读挂载由 root 读取，chown 失败不影响启动
  chown "$(id -u):$(id -g)" "${OPS_TOKEN_FILE}" 2>/dev/null || true
  echo "==> 已写入 ${OPS_TOKEN_FILE}（Prometheus Bearer 鉴权）"
else
  echo "WARN: PLANSTAR_OPS_TOKEN 为空，写入空 ${OPS_TOKEN_FILE}；Prometheus 抓取 /metrics 将返回 401。" >&2
  ( umask 077; : > "${OPS_TOKEN_FILE}" )
  chmod 600 "${OPS_TOKEN_FILE}" 2>/dev/null || true
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

# 令牌文件被 git 忽略，需单独同步到服务器（走 stdin，避免出现在进程列表）
echo "==> 同步运维令牌到服务器"
printf '%s' "${PLANSTAR_OPS_TOKEN:-}" | ssh -p "${SERVER_PORT}" "${SERVER_USER}@${SERVER_HOST}" \
  "cd ${APP_DIR} && mkdir -p planstar/deploy/prometheus && (umask 077; cat > planstar/deploy/prometheus/ops_token) && chmod 600 planstar/deploy/prometheus/ops_token"

echo "==> 远程拉取镜像并重启服务"
ssh -p "${SERVER_PORT}" "${SERVER_USER}@${SERVER_HOST}" "cd ${APP_DIR} && \
  export PLANSTAR_IMAGE=${REGISTRY} PLANSTAR_TAG=${TAG} && \
  docker compose -f planstar/docker-compose.yml pull && \
  docker compose -f planstar/docker-compose.yml up -d --remove-orphans"

echo "==> 部署完成: http://${SERVER_HOST}:8098"