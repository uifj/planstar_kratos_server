#!/usr/bin/env bash
# 生产上线版本一键启动（4 核 8G / 2M 带宽，禁用测试口令）
#   cp .env.prod.example .env.prod && chmod 600 .env.prod   # 逐项填写
#   ./up-prod.sh
set -euo pipefail
cd "$(dirname "$0")"

[ -f .env.prod ] || { echo "缺少 .env.prod，请先从 .env.prod.example 复制并填写" >&2; exit 1; }
chmod 600 .env.prod

echo "==> 发布前门禁 preflight"
ENV_FILE=.env.prod COMPOSE_DIR="${PWD}" PLANSTAR_CONFIG_FILE=config.lite.yaml ./deploy/preflight.sh

# 读取 .env.prod 中的密钥，供运维令牌落盘使用
set -a
# shellcheck disable=SC1090
. ./.env.prod
set +a

# 运维令牌落盘（Prometheus 抓取 9108/metrics 的 Bearer；文件缺失会导致挂载失败）
: "${PLANSTAR_OPS_TOKEN:?生产必须配置 PLANSTAR_OPS_TOKEN}"
mkdir -p deploy/prometheus
( umask 077; printf '%s' "${PLANSTAR_OPS_TOKEN}" > deploy/prometheus/ops_token )
chmod 600 deploy/prometheus/ops_token 2>/dev/null || true

COMPOSE_FILES="-f docker-compose.yml -f docker-compose.prod.yml"
# 部署侧可选覆盖（例如实际对象存储是火山引擎 TOS 时用来关闭阿里云 OSS）。
# 存在即自动叠加，保证脚本与线上实际使用的三文件合并方式一致。
if [ -f docker-compose.server.yml ]; then
  echo "==> 检测到部署侧覆盖 docker-compose.server.yml，自动叠加"
  COMPOSE_FILES="${COMPOSE_FILES} -f docker-compose.server.yml"
fi
COMPOSE="docker compose --env-file .env.prod ${COMPOSE_FILES}"
echo "==> 启动生产版本（config.lite.yaml + prod overlay）"
${COMPOSE} up -d

echo "==> 等待 /readyz"
ready=0
for _ in $(seq 1 45); do
  if curl -fsS http://127.0.0.1:9108/readyz >/dev/null 2>&1; then ready=1; break; fi
  sleep 2
done
[ "${ready}" -eq 1 ] || { echo "readyz 未就绪，查看日志：${COMPOSE} logs planstar-api" >&2; exit 1; }

echo "==> 生产版本就绪，请按 docs/PRODUCTION_DELIVERY.md 执行上线检查项"
${COMPOSE} ps
