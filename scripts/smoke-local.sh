#!/bin/sh
# =====================================================================
# Planstar 本地烟雾测试（POSIX sh）
#
# 前置：
#   cd planstar && cp .env.example .env && docker compose up -d --build
# 用法：
#   sh scripts/smoke-local.sh
#   PS_API_BASE=http://127.0.0.1:8098 PS_ADMIN_BASE=http://127.0.0.1:9108 \
#     sh scripts/smoke-local.sh
#
# 检查项：
#   1. GET  <admin>/healthz                            -> 200
#   2. GET  <api>/v1/activities?pageIndex=1&pageSize=1 -> 200 且 JSON 可解析
#   3. POST <api>/v3/user/sendLoginCode                -> 不得返回 5xx
# 任一失败即以非 0 退出，并输出具体原因。
# 依赖 curl；无 curl 时用 wget 兜底（从 --server-response 解析状态码）。
# =====================================================================

set -u

API_BASE="${PS_API_BASE:-http://127.0.0.1:8098}"
ADMIN_BASE="${PS_ADMIN_BASE:-http://127.0.0.1:9108}"

failures=0
ERROR_MSG=""

TMP_DIR="$(mktemp -d 2>/dev/null || mktemp -d -t planstar-smoke)" || {
	echo "错误：无法创建临时目录。" >&2
	exit 2
}
trap 'rm -rf "$TMP_DIR"' EXIT INT TERM
BODY_FILE="$TMP_DIR/body"
HDR_FILE="$TMP_DIR/headers"
ERR_FILE="$TMP_DIR/stderr"

if command -v curl >/dev/null 2>&1; then
	HTTP_CLIENT="curl"
elif command -v wget >/dev/null 2>&1; then
	HTTP_CLIENT="wget"
	echo "提示：未找到 curl，使用 wget 兜底解析状态码。"
else
	echo "错误：需要 curl 或 wget 之一才能执行烟雾测试。" >&2
	exit 2
fi

pass() {
	echo "OK:   $1"
}

fail() {
	echo "FAIL: $1" >&2
	failures=$((failures + 1))
}

# http_call <GET|POST> <url> [json_body]
# 结果写入全局变量 STATUS，响应体写入 $BODY_FILE；连接失败返回 1。
http_call() {
	method="$1"
	url="$2"
	body="${3:-}"
	STATUS=""
	: > "$BODY_FILE"
	: > "$HDR_FILE"
	: > "$ERR_FILE"

	if [ "$HTTP_CLIENT" = "curl" ]; then
		if [ "$method" = "GET" ]; then
			STATUS="$(curl -sS -m 10 -o "$BODY_FILE" -D "$HDR_FILE" -w '%{http_code}' "$url" 2>"$ERR_FILE")"
		else
			STATUS="$(curl -sS -m 10 -o "$BODY_FILE" -D "$HDR_FILE" -w '%{http_code}' \
				-X POST -H 'Content-Type: application/json' -d "$body" "$url" 2>"$ERR_FILE")"
		fi
		rc=$?
		if [ "$rc" -ne 0 ]; then
			ERROR_MSG="$(cat "$ERR_FILE")"
			return 1
		fi
		return 0
	fi

	if [ "$method" = "GET" ]; then
		wget -q -S -O "$BODY_FILE" "$url" 2>"$HDR_FILE"
	else
		wget -q -S -O "$BODY_FILE" --header='Content-Type: application/json' \
			--post-data="$body" "$url" 2>"$HDR_FILE"
	fi
	rc=$?
	STATUS="$(awk '/^[[:space:]]*HTTP\//{code=$2} END{if (code!="") print code}' "$HDR_FILE")"
	if [ -z "$STATUS" ]; then
		ERROR_MSG="$(tail -n 3 "$HDR_FILE" 2>/dev/null)"
		[ "$rc" -ne 0 ] && return 1
	fi
	# wget 对 4xx/5xx 会返回非 0，但此时已拿到真实状态码，交由调用方判定。
	return 0
}

echo "==> 1/3 运维端口存活探针：$ADMIN_BASE/healthz"
if ! http_call GET "$ADMIN_BASE/healthz"; then
	fail "无法连接 $ADMIN_BASE/healthz（容器未启动或 9108 未映射）：$ERROR_MSG"
elif [ "$STATUS" != "200" ]; then
	fail "GET /healthz 期望 200，实际 $STATUS"
else
	pass "GET /healthz -> 200"
fi

echo "==> 2/3 演出目录：$API_BASE/v1/activities?pageIndex=1&pageSize=1"
if ! http_call GET "$API_BASE/v1/activities?pageIndex=1&pageSize=1"; then
	fail "无法连接 $API_BASE/v1/activities（容器未启动或 8098 未映射）：$ERROR_MSG"
elif [ "$STATUS" != "200" ]; then
	fail "GET /v1/activities 期望 200，实际 $STATUS"
elif [ ! -s "$BODY_FILE" ]; then
	fail "GET /v1/activities 返回空响应体"
elif ! grep -Eq '^[[:space:]]*[\[{]' "$BODY_FILE"; then
	fail "GET /v1/activities 响应不是可解析的 JSON（前 80 字节：$(head -c 80 "$BODY_FILE" 2>/dev/null)）"
else
	pass "GET /v1/activities -> 200 且 JSON 可解析"
fi

echo "==> 3/3 发码接口：POST $API_BASE/v3/user/sendLoginCode"
if ! http_call POST "$API_BASE/v3/user/sendLoginCode" '{"ua":"13800000001"}'; then
	fail "无法连接 $API_BASE/v3/user/sendLoginCode：$ERROR_MSG"
else
	case "$STATUS" in
		5*)
			fail "POST /v3/user/sendLoginCode 返回 5xx（$STATUS）：$(head -c 200 "$BODY_FILE" 2>/dev/null)"
			;;
		2* | 4*)
			pass "POST /v3/user/sendLoginCode -> $STATUS（非 5xx）"
			;;
		*)
			fail "POST /v3/user/sendLoginCode 返回异常状态码：$STATUS"
			;;
	esac
fi

if [ "$failures" -ne 0 ]; then
	echo "" >&2
	echo "烟雾测试失败：$failures 项未通过。" >&2
	exit 1
fi

echo ""
echo "烟雾测试通过：3/3。"
