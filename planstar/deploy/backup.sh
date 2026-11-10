#!/usr/bin/env bash
# 生产备份：全部 11 个 planstar_* 库 mysqldump + .env，按标签归档并保留 7 天。
# 库名以 docs/sql/00_all.sql 与各 sql/*.sql 的 CREATE DATABASE 为唯一事实源。
# 用法:
#   ./backup.sh daily        # 每日全量（由 cron 调用）
#   ./backup.sh pre-deploy   # 发布前快照
set -euo pipefail

COMPOSE_DIR="${COMPOSE_DIR:-/opt/planstar/planstar}"
BACKUP_ROOT="${BACKUP_DIR:-/opt/planstar/backups}"
ENV_FILE="${ENV_FILE:-.env}"
LABEL="${1:-daily}"
STAMP="$(date +%F-%H%M)"
DIR="${BACKUP_ROOT}/${LABEL}/${STAMP}"

cd "${COMPOSE_DIR}"
[ -f "${ENV_FILE}" ] || { echo "missing ${ENV_FILE}" >&2; exit 1; }
mkdir -p "${DIR}"

MYSQL_PASSWORD="$(grep -E '^MYSQL_ROOT_PASSWORD=' "${ENV_FILE}" | tail -1 | cut -d= -f2-)"
[ -n "${MYSQL_PASSWORD}" ] || { echo "MYSQL_ROOT_PASSWORD 未配置" >&2; exit 1; }

docker compose -f docker-compose.yml exec -T mysql \
  mysqldump -uroot -p"${MYSQL_PASSWORD}" \
  --single-transaction --routines --triggers --databases \
  planstar_user planstar_catalog planstar_ticket planstar_order \
  planstar_payment planstar_marketing planstar_community planstar_content \
  planstar_meta planstar_shop planstar_admin \
  | gzip > "${DIR}/mysql.sql.gz"

# 空文件即视为失败（mysqldump 失败时 pipefail 已生效，此处再兜一层）
[ -s "${DIR}/mysql.sql.gz" ] || { echo "backup produced empty archive" >&2; exit 1; }

cp "${ENV_FILE}" "${DIR}/env.backup"
chmod 600 "${DIR}/mysql.sql.gz" "${DIR}/env.backup"

# 仅保留最近 7 天（pre-deploy 同样适用，避免磁盘打满）
find "${BACKUP_ROOT}/${LABEL}" -mindepth 1 -maxdepth 1 -type d -mtime +7 -exec rm -rf {} +

echo "backup ok: ${DIR}"
