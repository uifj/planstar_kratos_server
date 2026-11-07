#!/bin/sh
# =====================================================================
# Planstar 运维端点压测（POSIX sh）
#
# 运行 TestOpsEndpointsLoad：100 goroutines × 200 请求，覆盖 /healthz、
# /readyz，并校验 /metrics 无令牌 401。输出 QPS/P50/P95/P99/错误率。
#
# 用法（任意目录）：
#   sh scripts/ops-loadtest.sh
# =====================================================================

set -eu

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
REPO_ROOT="$(CDPATH= cd -- "$SCRIPT_DIR/.." && pwd)"

cd "$REPO_ROOT/planstar"
exec go test ./internal/server -run TestOpsEndpointsLoad -count=1 -v
