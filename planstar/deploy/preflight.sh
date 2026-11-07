#!/usr/bin/env bash
# 发布前配置预检：必需变量、占位符、环境文件权限、compose 合并校验。
# 用法:
#   COMPOSE_DIR=/opt/planstar/planstar PLANSTAR_CONFIG_FILE=config.lite.yaml ./preflight.sh
set -euo pipefail

COMPOSE_DIR="${COMPOSE_DIR:-/opt/planstar/planstar}"
ENV_FILE="${ENV_FILE:-.env}"
CONFIG_FILE="${PLANSTAR_CONFIG_FILE:-config.lite.yaml}"

cd "${COMPOSE_DIR}"

die() {
  echo "preflight failed: $*" >&2
  exit 1
}

[ -f "${ENV_FILE}" ] || die "缺少 ${ENV_FILE}（应先从 production.env.example 复制并填写）"
perm=$(stat -c '%a' "${ENV_FILE}")
[ "${perm}" = "600" ] || echo "WARN: ${ENV_FILE} 权限为 ${perm}，建议 chmod 600" >&2
[ -f "${CONFIG_FILE}" ] || die "配置文件不存在: ${CONFIG_FILE}"

get() {
  grep -E "^${1}=" "${ENV_FILE}" | tail -1 | cut -d= -f2- || true
}

missing=0
for key in \
  PLANSTAR_IMAGE PLANSTAR_TAG PLANSTAR_OPS_TOKEN ADMIN_AUTH_JWT_SECRET \
  ADMIN_DATA_DATABASE_SOURCE MYSQL_ROOT_PASSWORD REDIS_PASSWORD ACME_EMAIL; do
  val=$(get "${key}")
  if [ -z "${val}" ]; then
    echo "MISSING: ${key}" >&2
    missing=1
  elif printf '%s' "${val}" | grep -q '<REPLACE_ME>'; then
    echo "PLACEHOLDER: ${key}" >&2
    missing=1
  fi
done

if [ "$(get PLANSTAR_TAG)" = "latest" ]; then
  echo "INVALID: PLANSTAR_TAG 不能是 latest（生产必须固定语义化版本）" >&2
  missing=1
fi

if [ "${CONFIG_FILE}" = "config.lite.yaml" ]; then
  PLANSTAR_CONFIG_FILE="${CONFIG_FILE}" \
    docker compose -f docker-compose.yml -f docker-compose.lite.yml config -q
else
  COMPOSE_PROFILES=full PLANSTAR_CONFIG_FILE="${CONFIG_FILE}" \
    docker compose -f docker-compose.yml config -q
fi

[ "${missing}" -eq 0 ] || die "存在缺失或占位配置，请先补全 .env"
echo "preflight ok (${CONFIG_FILE})"
