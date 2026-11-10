#!/usr/bin/env bash
# =====================================================================
# 发布前配置预检（生产上线强制门禁）
#   COMPOSE_DIR=/opt/planstar/planstar ENV_FILE=.env.prod ./preflight.sh
#
# 检查：必需变量、<REPLACE_ME> 占位、固定版本标签、禁止测试口令、
#       环境文件权限、生产 compose 可解析、备份脚本库名与建库脚本一致。
# =====================================================================
set -euo pipefail

COMPOSE_DIR="${COMPOSE_DIR:-/opt/planstar/planstar}"
ENV_FILE="${ENV_FILE:-.env.prod}"
CONFIG_FILE="${PLANSTAR_CONFIG_FILE:-config.lite.yaml}"

cd "${COMPOSE_DIR}"

die() {
  echo "preflight failed: $*" >&2
  exit 1
}

[ -f "${ENV_FILE}" ] || die "缺少 ${ENV_FILE}（应先从 .env.prod.example 复制并填写）"
perm=$(stat -c '%a' "${ENV_FILE}")
[ "${perm}" = "600" ] || echo "WARN: ${ENV_FILE} 权限为 ${perm}，建议 chmod 600" >&2
[ -f "configs/${CONFIG_FILE}" ] || die "配置文件不存在: configs/${CONFIG_FILE}"

get() {
  grep -E "^${1}=" "${ENV_FILE}" | tail -1 | cut -d= -f2- || true
}

missing=0
REQUIRED_KEYS=(
  PLANSTAR_IMAGE PLANSTAR_TAG PLANSTAR_OPS_TOKEN PLANSTAR_OPS_ALLOWED_CIDRS
  PLANSTAR_AUTH_CODE_HMAC_KEY ADMIN_AUTH_JWT_SECRET ADMIN_DATA_DATABASE_SOURCE
  ADMIN_PLANSTAR_OPS_TOKEN MYSQL_ROOT_PASSWORD REDIS_PASSWORD
  PLANSTAR_APP_DB_PASSWORD GF_SECURITY_ADMIN_PASSWORD ACME_EMAIL
  PLANSTAR_NOTIFY_SMS_ACCESS_KEY PLANSTAR_NOTIFY_SMS_ACCESS_SECRET
  PLANSTAR_NOTIFY_SMS_SIGN_NAME PLANSTAR_NOTIFY_SMS_TEMPLATE_CODE
  PLANSTAR_STORAGE_OSS_ENDPOINT PLANSTAR_STORAGE_OSS_REGION
  PLANSTAR_STORAGE_OSS_BUCKET PLANSTAR_STORAGE_OSS_ACCESS_KEY
  PLANSTAR_STORAGE_OSS_ACCESS_SECRET
  AM_SMTP_SMARTHOST AM_SMTP_FROM AM_ALERT_EMAIL_TO
)
for key in "${REQUIRED_KEYS[@]}"; do
  val=$(get "${key}")
  if [ -z "${val}" ]; then
    echo "MISSING: ${key}" >&2
    missing=1
  elif printf '%s' "${val}" | grep -q '<REPLACE_ME>'; then
    echo "PLACEHOLDER: ${key}" >&2
    missing=1
  fi
done

TAG="$(get PLANSTAR_TAG)"
if [[ ! "${TAG}" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[A-Za-z0-9.]+)?$ ]]; then
  echo "INVALID: PLANSTAR_TAG 必须是语义化版本（如 v1.0.0），当前=${TAG}" >&2
  missing=1
fi

# 测试口令绝不能出现在生产环境文件 / 配置中
if [ "$(get PLANSTAR_AUTH_ALLOW_TEST_CREDENTIALS)" = "true" ]; then
  echo "FORBIDDEN: 生产环境禁止 PLANSTAR_AUTH_ALLOW_TEST_CREDENTIALS=true" >&2
  missing=1
fi
if grep -RqsE '^[[:space:]]*allow_test_credentials:[[:space:]]*true' configs/config.lite.yaml configs/config.prod.yaml; then
  echo "FORBIDDEN: 生产配置中 allow_test_credentials 为 true" >&2
  missing=1
fi
if [ "$(get PLANSTAR_NOTIFY_SMS_ENABLED)" != "true" ] || [ "$(get PLANSTAR_NOTIFY_SMS_PROVIDER)" != "aliyun" ]; then
  echo "INVALID: 生产必须启用 PLANSTAR_NOTIFY_SMS_ENABLED=true / PROVIDER=aliyun" >&2
  missing=1
fi
if grep -qE 'password:[[:space:]]*(root|admin|123456)[[:space:]]*$' configs/config.lite.yaml; then
  echo "INVALID: 生产配置含弱口令字面量" >&2
  missing=1
fi

# 备份脚本库名必须与建库脚本一致（历史缺陷：脚本里 6 个库名不存在）
for db in planstar_user planstar_catalog planstar_ticket planstar_order \
          planstar_payment planstar_marketing planstar_community \
          planstar_content planstar_meta planstar_shop planstar_admin; do
  grep -q "${db}" deploy/backup.sh || { echo "BACKUP: 缺少 ${db}" >&2; missing=1; }
done

# 生产 compose 必须可解析（所有 ${VAR:?} 在此处被强制求值）。
# 部署侧若存在 docker-compose.server.yml（例如运行时关闭不兼容的 OSS），
# 必须与线上一样叠加后再校验，避免「门禁过了、实际启动的配置没过」。
COMPOSE_ARGS="-f docker-compose.yml -f docker-compose.prod.yml"
if [ -f docker-compose.server.yml ]; then
  COMPOSE_ARGS="${COMPOSE_ARGS} -f docker-compose.server.yml"
fi
docker compose --env-file "${ENV_FILE}" ${COMPOSE_ARGS} config -q

[ "${missing}" -eq 0 ] || die "存在缺失/占位/不安全配置，请先补全 ${ENV_FILE}"
echo "preflight ok (${CONFIG_FILE}, ${ENV_FILE})"
