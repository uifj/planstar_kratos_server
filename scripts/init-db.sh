#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MYSQL_CONTAINER="${MYSQL_CONTAINER:-ks-mysql}"

if command -v mysql >/dev/null 2>&1; then
  run_sql() {
    mysql -uroot -proot < "$1"
  }
else
  run_sql() {
    docker exec -i "$MYSQL_CONTAINER" mysql -uroot -proot < "$1"
  }
fi

echo "==> 创建数据库"
run_sql "$ROOT_DIR/sql/01_init_databases.sql"

echo "==> 初始化 user 库"
run_sql "$ROOT_DIR/sql/02_user.sql"

echo "==> 初始化权限表"
run_sql "$ROOT_DIR/sql/09_permissions.sql"

echo "==> 初始化 goods 库"
run_sql "$ROOT_DIR/sql/03_goods.sql"

echo "==> 初始化 cart 库"
run_sql "$ROOT_DIR/sql/04_cart.sql"

echo "==> 初始化 order 库"
run_sql "$ROOT_DIR/sql/05_order.sql"

echo "==> 初始化 inventory 库"
run_sql "$ROOT_DIR/sql/06_inventory.sql"

echo "==> 初始化 payment 库"
run_sql "$ROOT_DIR/sql/07_payment.sql"

echo "==> 完成"
