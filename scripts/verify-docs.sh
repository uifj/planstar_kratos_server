#!/bin/sh
# =====================================================================
# 文档事实源完整性校验（POSIX sh）
# 任意必需文档缺失即以非 0 退出，并列出缺失项。
# 用法：sh scripts/verify-docs.sh
# =====================================================================

set -u

ROOT_DIR="$(cd "$(dirname -- "$0")/.." && pwd)"

REQUIRED_FILES="
docs/README.md
docs/architecture.md
docs/api/API_CONTRACT.md
docs/deployment.md
docs/operations.md
"

missing=0
total=0
for rel in $REQUIRED_FILES; do
	total=$((total + 1))
	if [ ! -f "$ROOT_DIR/$rel" ]; then
		echo "MISSING: $rel" >&2
		missing=$((missing + 1))
	fi
done

if [ "$missing" -ne 0 ]; then
	echo "" >&2
	echo "文档校验失败：缺少 $missing/$total 个必需文件（基准目录：$ROOT_DIR）。" >&2
	exit 1
fi

echo "文档校验通过：$total 个必需文档均存在（基准目录：$ROOT_DIR）。"
