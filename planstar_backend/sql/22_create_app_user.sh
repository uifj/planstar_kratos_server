#!/usr/bin/env bash
# ---------------------------------------------------------------------------
# 生产最小权限数据库账号（MySQL 官方镜像 initdb.d 会按文件名字典序执行 .sh）。
# 只在首次初始化（数据卷为空）时运行；存量库请用 make sql-apply 或手工执行等价 SQL。
#
# 依据：阿里云 OSS Go SDK 文档「推荐最小化授权的 RAM 用户 AK」同理，
# 数据库也禁止应用以 root 连接；conf.Validate() 会在生产校验 user != root。
# ---------------------------------------------------------------------------
set -euo pipefail

: "${MYSQL_ROOT_PASSWORD:?MYSQL_ROOT_PASSWORD 必须设置}"
APP_USER="${PLANSTAR_APP_DB_USER:-planstar_app}"
APP_PASSWORD="${PLANSTAR_APP_DB_PASSWORD:?PLANSTAR_APP_DB_PASSWORD 必须设置}"

# 仅授予业务库 DML，不给 DDL / GRANT / DROP，避免注入或漏洞扩大为全库失守。
DATABASES=(
  planstar_user planstar_catalog planstar_ticket planstar_order
  planstar_payment planstar_marketing planstar_community
  planstar_content planstar_meta planstar_shop planstar_admin
)

{
  echo "CREATE USER IF NOT EXISTS '${APP_USER}'@'%' IDENTIFIED BY '${APP_PASSWORD}';"
  echo "ALTER USER '${APP_USER}'@'%' IDENTIFIED BY '${APP_PASSWORD}';"
  for db in "${DATABASES[@]}"; do
    echo "GRANT SELECT, INSERT, UPDATE, DELETE, EXECUTE ON ${db}.* TO '${APP_USER}'@'%';"
  done
  echo "FLUSH PRIVILEGES;"
} | mysql -uroot -p"${MYSQL_ROOT_PASSWORD}" --batch

echo "planstar initdb: least-privilege user ${APP_USER} ready"
