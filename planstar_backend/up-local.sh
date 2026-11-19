#!/usr/bin/env bash
# 本地开发/联调版本一键启动（允许测试口令）
#   ./up-local.sh
#   Flutter: flutter run --dart-define=PS_API_BASE=http://10.0.2.2:8098/
set -euo pipefail
cd "$(dirname "$0")"

[ -f .env.local ] || cp .env.local.example .env.local
echo "==> 启动本地版本（config.local.yaml + 演示种子 + 测试口令）"
docker compose --env-file .env.local -f docker-compose.local.yml up -d --build

echo "==> 等待 /healthz"
ready=0
for _ in $(seq 1 30); do
  if curl -fsS http://127.0.0.1:9108/healthz >/dev/null 2>&1; then ready=1; break; fi
  sleep 2
done
[ "${ready}" -eq 1 ] || { echo "healthz 未就绪，查看日志：docker compose -f docker-compose.local.yml logs planstar-api" >&2; exit 1; }

curl -fsS http://127.0.0.1:9108/healthz; echo
echo "==> 本地版本就绪"
echo "    业务 API : http://127.0.0.1:8098/"
echo "    运维探针 : http://127.0.0.1:9108/healthz （/metrics 需 Bearer，本地未配置故返回 401）"
echo "    演示账号 : 13800000001 / 13800000002，测试口令见 conf.LocalTestCredentialPassword"
echo "    停止     : docker compose --env-file .env.local -f docker-compose.local.yml down"
