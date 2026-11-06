#!/usr/bin/env bash
# End-to-end smoke test for the folded Kratos v3 mall running under
# `docker compose up -d --build`.
#
# Prerequisites:
#   - the compose stack is up and healthy
#   - `make seed` has been run (demo catalogue, inventory, permissions, admin)
#
# Request/response JSON uses proto field names (snake_case): Kratos v3 uses
# encoding/json with the generated struct tags, and the form/query decoder
# accepts proto names as well as json names.
set -euo pipefail

BASE="${BASE:-http://127.0.0.1:8000}"
COMPOSE="${COMPOSE:-docker compose}"
PY="${PY:-python}"

SKU_ID="${SKU_ID:-019d0000-0000-7000-8000-000000000001}"
GOODS_ID="${GOODS_ID:-019c0000-0000-7000-8000-000000000001}"
SKU_HUGE="${SKU_HUGE:-019d0000-0000-7000-8000-000000000003}"

pass() { printf '  [OK]   %s\n' "$*"; }
fail() { printf '  [FAIL] %s\n' "$*" >&2; exit 1; }
step() { printf '\n=== %s ===\n' "$*"; }

# json_get KEY reads a JSON object on stdin and prints the value (or "").
json_get() {
  "$PY" -c 'import json,sys
try:
    d = json.load(sys.stdin)
except Exception:
    print(""); raise SystemExit
v = d.get(sys.argv[1], "")
print("" if v is None else v)' "$1"
}

# api METHOD PATH [TOKEN] [BODY]
api() {
  local method="$1" path="$2" token="${3:-}" body="${4:-}"
  local -a args=(-sS -X "$method" "$BASE$path" -H 'Content-Type: application/json')
  if [[ -n "$token" ]]; then args+=(-H "Authorization: Bearer $token"); fi
  if [[ -n "$body" ]]; then args+=(-d "$body"); fi
  curl "${args[@]}"
}

# expect_ok LABEL JSON ensures the reply is not a Kratos error envelope.
expect_ok() {
  local label="$1" body="$2"
  if [[ -z "$body" ]]; then fail "$label: empty response"; fi
  local reason
  reason="$(printf '%s' "$body" | json_get reason)"
  if [[ -n "$reason" ]]; then fail "$label: $body"; fi
}

wait_for() {
  local url="$1" label="$2" tries="${3:-40}"
  for _ in $(seq 1 "$tries"); do
    if curl -fsS "$url" >/dev/null 2>&1; then pass "$label"; return 0; fi
    sleep 1
  done
  fail "$label: not ready after ${tries}s"
}

# ---------------------------------------------------------------------------
step "0. Health"
wait_for "$BASE/healthz" "GET /healthz -> ok"
[[ "$(curl -fsS "$BASE/healthz")" == "ok" ]] || fail "unexpected /healthz body"

# ---------------------------------------------------------------------------
step "1. Captcha stored in Redis"
CAP="$(api GET /api/users/captcha)"
expect_ok "captcha" "$CAP"
CID="$(printf '%s' "$CAP" | json_get captcha_id)"
PIC="$(printf '%s' "$CAP" | json_get pic_path)"
[[ -n "$CID" && -n "$PIC" ]] || fail "captcha reply missing fields: $CAP"
CODE="$($COMPOSE exec -T redis redis-cli GET "mall:captcha:$CID" 2>/dev/null | tr -d '\r\n')"
[[ -n "$CODE" ]] || fail "captcha answer not found in Redis for id=$CID"
pass "captcha_id=$CID answer=$CODE (image is a data URL)"

# ---------------------------------------------------------------------------
step "2. Register (C-end)"
MOBILE="$($PY -c "import time; print('138%08d' % (int(time.time()) % 100000000))")"
REG="$(api POST /api/users/register "" "$(printf '{"mobile":"%s","username":"smoke","password":"pass123456"}' "$MOBILE")")"
expect_ok "register" "$REG"
TOKEN="$(printf '%s' "$REG" | json_get token)"
UID="$(printf '%s' "$REG" | json_get id)"
[[ -n "$TOKEN" && -n "$UID" ]] || fail "register reply missing token/id: $REG"
pass "mobile=$MOBILE id=$UID token acquired"

# ---------------------------------------------------------------------------
step "3. Login with captcha (C-end)"
LOGIN="$(api POST /api/users/login "" "$(printf '{"mobile":"%s","password":"pass123456","captcha":"%s","captcha_id":"%s"}' "$MOBILE" "$CODE" "$CID")")"
expect_ok "login" "$LOGIN"
[[ -n "$(printf '%s' "$LOGIN" | json_get token)" ]] || fail "login reply missing token: $LOGIN"
pass "login ok (captcha consumed from Redis)"

# ---------------------------------------------------------------------------
step "4. Detail + goods list/detail"
DETAIL="$(api GET /api/users/detail "$TOKEN")"
expect_ok "detail" "$DETAIL"
[[ "$(printf '%s' "$DETAIL" | json_get id)" == "$UID" ]] || fail "detail id mismatch: $DETAIL"

GL="$(api GET '/api/goods/list?page=1&page_size=10')"
expect_ok "goods/list" "$GL"
COUNT="$($PY -c 'import json,sys; print(len(json.load(sys.stdin).get("list", [])))' <<<"$GL")"
[[ "$COUNT" -ge 1 ]] || fail "goods/list returned no rows: $GL"
pass "goods/list returned $COUNT on-sale goods"

GD="$(api GET "/api/goods/detail?id=$GOODS_ID")"
expect_ok "goods/detail" "$GD"
[[ "$(printf '%s' "$GD" | json_get id)" == "$GOODS_ID" ]] || fail "goods/detail mismatch: $GD"
pass "goods/detail ok"

# ---------------------------------------------------------------------------
step "5. Cart add + address create"
CART="$(api POST /api/cart/add "$TOKEN" "$(printf '{"goods_id":"%s","sku_id":"%s","goods_num":1,"is_select":true}' "$GOODS_ID" "$SKU_ID")")"
expect_ok "cart/add" "$CART"
CART_ID="$(printf '%s' "$CART" | json_get id)"
[[ -n "$CART_ID" ]] || fail "cart/add missing id: $CART"
pass "cart/add id=$CART_ID (price re-read from SKU)"

ADDR="$(api POST /api/address/create "$TOKEN" '{"name":"张三","mobile":"13800138000","province":"广东省","city":"深圳市","districts":"南山区","address":"科技园 1 号","post_code":"518000","is_default":1}')"
expect_ok "address/create" "$ADDR"
ADDR_ID="$(printf '%s' "$ADDR" | json_get id)"
[[ -n "$ADDR_ID" ]] || fail "address/create missing id: $ADDR"
pass "address/create id=$ADDR_ID"

# ---------------------------------------------------------------------------
step "6. Order create -> Outbox -> inventory lock (Saga)"
ORDER="$(api POST /api/order/create "$TOKEN" "$(printf '{"address":"%s","cart_item":[{"cart_id":"%s","sku_id":"%s","sku_num":1}]}' "$ADDR_ID" "$CART_ID" "$SKU_ID")")"
expect_ok "order/create" "$ORDER"
ORDER_SN="$(printf '%s' "$ORDER" | json_get order_sn)"
TOTAL="$(printf '%s' "$ORDER" | json_get total)"
[[ -n "$ORDER_SN" && -n "$TOTAL" ]] || fail "order/create missing order_sn/total: $ORDER"
pass "order/create order_sn=$ORDER_SN total=$TOTAL status=$(printf '%s' "$ORDER" | json_get status)"

STATUS=""
for _ in $(seq 1 20); do
  OD="$(api GET "/api/order/detail?order_sn=$ORDER_SN" "$TOKEN")"
  STATUS="$(printf '%s' "$OD" | json_get status)"
  [[ "$STATUS" == "2" ]] && break
  sleep 1
done
[[ "$STATUS" == "2" ]] || fail "order did not reach LOCKED(2) within 20s (last=$STATUS)"
pass "order reached LOCKED(2): order.created outbox published + inventory locked"

# ---------------------------------------------------------------------------
step "7. Payment create + mock-pay -> paid (Saga)"
PAY="$(api POST /api/payment/create "$TOKEN" "$(printf '{"order_sn":"%s","amount":%s,"channel":"mock"}' "$ORDER_SN" "$TOTAL")")"
expect_ok "payment/create" "$PAY"
PAY_NO="$(printf '%s' "$PAY" | json_get payment_no)"
[[ -n "$PAY_NO" ]] || fail "payment/create missing payment_no: $PAY"
pass "payment/create payment_no=$PAY_NO"

MP="$(api POST /api/payment/mock-pay "$TOKEN" "$(printf '{"payment_no":"%s"}' "$PAY_NO")")"
expect_ok "payment/mock-pay" "$MP"

STATUS=""
for _ in $(seq 1 20); do
  OD="$(api GET "/api/order/detail?order_sn=$ORDER_SN" "$TOKEN")"
  STATUS="$(printf '%s' "$OD" | json_get status)"
  [[ "$STATUS" == "3" ]] && break
  sleep 1
done
[[ "$STATUS" == "3" ]] || fail "order did not reach PAID(3) within 20s (last=$STATUS)"
pass "order reached PAID(3): payment.success -> order.paid -> stock confirmed"

# ---------------------------------------------------------------------------
step "8. Admin login + dashboard / RBAC / brands"
ADMIN_LOGIN="$(api POST /admin/api/users/login "" '{"username":"admin","password":"admin123456"}')"
expect_ok "admin login" "$ADMIN_LOGIN"
ADMIN_TOKEN="$(printf '%s' "$ADMIN_LOGIN" | json_get token)"
[[ -n "$ADMIN_TOKEN" ]] || fail "admin login missing token: $ADMIN_LOGIN"
pass "admin login ok (role=2)"

for p in \
  "/admin/api/dashboard/stats" \
  "/admin/api/system/permissions" \
  "/admin/api/goods/brands?page=1&page_size=10" \
  "/admin/api/goods/categories" \
  "/admin/api/user/list?page=1&page_size=10"; do
  R="$(api GET "$p" "$ADMIN_TOKEN")"
  expect_ok "GET $p" "$R"
done
pass "admin dashboard / permissions / brands / categories / users all ok"

# ---------------------------------------------------------------------------
step "9. Negative path: inventory shortage cancels the order"
BAD="$(api POST /api/order/create "$TOKEN" "$(printf '{"address":"%s","cart_item":[{"sku_id":"%s","sku_num":999999}]}' "$ADDR_ID" "$SKU_HUGE")")"
expect_ok "order/create(negative)" "$BAD"
BAD_SN="$(printf '%s' "$BAD" | json_get order_sn)"
[[ -n "$BAD_SN" ]] || fail "negative order/create missing order_sn: $BAD"
STATUS=""
for _ in $(seq 1 20); do
  OD="$(api GET "/api/order/detail?order_sn=$BAD_SN" "$TOKEN")"
  STATUS="$(printf '%s' "$OD" | json_get status)"
  [[ "$STATUS" == "6" ]] && break
  sleep 1
done
[[ "$STATUS" == "6" ]] || fail "shortage order did not reach CANCELLED(6) (last=$STATUS)"
pass "shortage order reached CANCELLED(6) via inventory.lock.failed"

# ---------------------------------------------------------------------------
step "10. Async infrastructure sanity"
QUEUES="$($COMPOSE exec -T rabbitmq rabbitmqctl list_queues name messages 2>/dev/null || true)"
echo "$QUEUES" | grep -q 'q.order.created' || fail "q.order.created missing"
echo "$QUEUES" | grep -q 'q.mall.dlx' || fail "DLQ q.mall.dlx missing"
pass "RabbitMQ queues present (including DLQ)"
printf '%s\n' "$QUEUES"

OUTBOX="$($COMPOSE exec -T mysql mysql -uroot -proot -N -B -e \
  "SELECT COUNT(*) FROM mall.order_event_outboxes WHERE status=1" 2>/dev/null | tr -d '\r\n')"
pass "published outbox rows: ${OUTBOX:-unknown}"

CONSUMED="$($COMPOSE exec -T mysql mysql -uroot -proot -N -B -e \
  "SELECT COUNT(*) FROM mall.consumed_events" 2>/dev/null | tr -d '\r\n')"
pass "consumed_event idempotency rows: ${CONSUMED:-unknown}"

printf '\n=== ALL SMOKE CHECKS PASSED ===\n'
