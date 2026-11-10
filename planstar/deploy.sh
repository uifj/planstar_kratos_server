#!/usr/bin/env bash
# =====================================================================
# Planstar 生产一键推送部署（4 核 8G / 2M 带宽单机）
#
# 用法:
#   cp .env.prod.example .env.prod    # 逐项填写，禁止 <REPLACE_ME>
#   chmod 600 .env.prod
#   ./deploy/preflight.sh             # 发布前强制校验（本机预检）
#   ./deploy.sh
#
# 与本地版本的区别：本脚本只会拉起 docker-compose.prod.yml 叠加的生产档位，
# 强制使用 .env.prod、config.lite.yaml、最小权限 DB 账号与阿里云资源。
# 本地联调请使用：docker compose -f docker-compose.local.yml up -d --build
#
# 服务器一次性准备：安装 Docker + Compose v2、克隆仓库到 /opt/planstar、
# 登录 GHCR、配置免密 SSH；详见 docs/PRODUCTION_DELIVERY.md。
# =====================================================================
set -euo pipefail
cd "$(dirname "$0")"

ENV_FILE="${ENV_FILE:-.env.prod}"
[ -f "${ENV_FILE}" ] || { echo "缺少 ${ENV_FILE}，请先从 .env.prod.example 复制并填写" >&2; exit 1; }
set -a
# shellcheck disable=SC1090
. "./${ENV_FILE}"
set +a

# 标签必须固定语义化版本，禁止 latest
TAG="${PLANSTAR_TAG:-}"
if [[ ! "${TAG}" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[A-Za-z0-9.]+)?$ ]]; then
  echo "PLANSTAR_TAG 必须是语义化版本（如 v1.0.0），当前=${TAG:-<空>}" >&2
  exit 1
fi
REGISTRY="${PLANSTAR_IMAGE:-ghcr.io/uifj/planstar-api}"
IMAGE="${REGISTRY}:${TAG}"

SERVER_HOST="${SERVER_HOST:?请在 ${ENV_FILE} 中配置 SERVER_HOST}"
SERVER_USER="${SERVER_USER:-planstar}"
SERVER_PORT="${SERVER_PORT:-22}"
APP_DIR="${APP_DIR:-/opt/planstar}"

# ---- 运维令牌落盘（Prometheus 抓取 9108/metrics 的 Bearer 令牌）----
OPS_TOKEN_FILE="deploy/prometheus/ops_token"
mkdir -p deploy/prometheus
: "${PLANSTAR_OPS_TOKEN:?生产必须配置 PLANSTAR_OPS_TOKEN}"
( umask 077; printf '%s' "${PLANSTAR_OPS_TOKEN}" > "${OPS_TOKEN_FILE}" )
chmod 600 "${OPS_TOKEN_FILE}" 2>/dev/null || true

COMPOSE_FILES="-f planstar/docker-compose.yml -f planstar/docker-compose.prod.yml"
# 部署侧覆盖（如火山引擎 TOS 场景关闭阿里云 OSS）：存在即叠加，
# 与线上实际使用的三文件合并方式保持一致。
if [ -f docker-compose.server.yml ]; then
  COMPOSE_FILES="${COMPOSE_FILES} -f planstar/docker-compose.server.yml"
fi
COMPOSE_ENV="--env-file planstar/.env.prod"

echo "==> 构建并推送镜像 ${IMAGE}"
docker build --build-arg VERSION="${VERSION:-${TAG}}" -t "${IMAGE}" .
docker push "${IMAGE}"

echo "==> 同步代码到 ${SERVER_USER}@${SERVER_HOST}:${APP_DIR}"
ssh -p "${SERVER_PORT}" "${SERVER_USER}@${SERVER_HOST}" \
  "cd ${APP_DIR} && git pull --ff-only"

echo "==> 同步环境文件与运维令牌（走 stdin，不出现在进程列表）"
ssh -p "${SERVER_PORT}" "${SERVER_USER}@${SERVER_HOST}" \
  "cd ${APP_DIR} && mkdir -p planstar/deploy/prometheus && (umask 077; cat > planstar/.env.prod) && chmod 600 planstar/.env.prod" < "${ENV_FILE}"
printf '%s' "${PLANSTAR_OPS_TOKEN}" | ssh -p "${SERVER_PORT}" "${SERVER_USER}@${SERVER_HOST}" \
  "cd ${APP_DIR} && (umask 077; cat > planstar/deploy/prometheus/ops_token) && chmod 600 planstar/deploy/prometheus/ops_token"

echo "==> 远程 preflight + 生产档位上线"
ssh -p "${SERVER_PORT}" "${SERVER_USER}@${SERVER_HOST}" "cd ${APP_DIR} && \
  export ENV_FILE=.env.prod COMPOSE_DIR=${APP_DIR}/planstar PLANSTAR_CONFIG_FILE=config.lite.yaml && \
  ./planstar/deploy/preflight.sh && \
  docker compose ${COMPOSE_ENV} ${COMPOSE_FILES} pull planstar-api admin admin-web && \
  docker compose ${COMPOSE_ENV} ${COMPOSE_FILES} up -d --remove-orphans && \
  docker compose ${COMPOSE_ENV} ${COMPOSE_FILES} ps"

echo "==> 部署完成，请执行上线检查项（docs/PRODUCTION_DELIVERY.md）"
