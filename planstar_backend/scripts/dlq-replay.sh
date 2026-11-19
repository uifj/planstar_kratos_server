#!/usr/bin/env sh
# DLQ 重放工具（ADR-0004）。
#
# 用途：人工确认死信根因并修复后，把 planstar.api.events.dlq 中的消息重新发布到
#       主 exchange。默认 dry-run，不会移动任何消息。
#
# 依赖：curl、python3、RabbitMQ management 插件（15672）。
# 用法：
#   RABBITMQ_USER=xxx RABBITMQ_PASS=xxx ./dlq-replay.sh --dry-run
#   RABBITMQ_USER=xxx RABBITMQ_PASS=xxx ./dlq-replay.sh --limit 20 --execute
#
# 环境变量：RABBITMQ_HOST(默认 http://127.0.0.1:15672)、RABBITMQ_VHOST(默认 %2F)、
#           PLANSTAR_QUEUE、PLANSTAR_EXCHANGE、REPLAY_LIMIT。
set -eu

HOST="${RABBITMQ_HOST:-http://127.0.0.1:15672}"
RUSER="${RABBITMQ_USER:-guest}"
RPASS="${RABBITMQ_PASS:-guest}"
VHOST="${RABBITMQ_VHOST:-%2F}"
QUEUE="${PLANSTAR_QUEUE:-planstar.api.events}"
EXCHANGE="${PLANSTAR_EXCHANGE:-planstar.exchange}"
DLQ="${QUEUE}.dlq"
LIMIT="${REPLAY_LIMIT:-10}"
MODE="dry-run"

while [ "$#" -gt 0 ]; do
  case "$1" in
    --dry-run) MODE="dry-run"; shift ;;
    --execute) MODE="execute"; shift ;;
    --limit) LIMIT="$2"; shift 2 ;;
    *) echo "未知参数: $1" >&2; exit 2 ;;
  esac
done

api() { curl -fsS -u "$RUSER:$RPASS" "$1"; }
api_post() { curl -fsS -u "$RUSER:$RPASS" -H 'Content-Type: application/json' -X POST -d "$2" "$1"; }

echo "==> 查询 DLQ: $DLQ (host=$HOST, mode=$MODE, limit=$LIMIT)"
INFO=$(api "$HOST/api/queues/$VHOST/$DLQ")
COUNT=$(printf '%s' "$INFO" | python3 -c 'import json,sys; print(json.load(sys.stdin).get("messages", 0))')
echo "    待重放消息数: $COUNT"

if [ "$MODE" = "dry-run" ]; then
  echo "    dry-run：未移动任何消息。确认根因已修复后加 --execute 重放。"
  exit 0
fi

if [ "$COUNT" = "0" ]; then
  echo "    DLQ 为空，无需重放。"
  exit 0
fi

echo "==> 从 DLQ 取消息并重新发布到 $EXCHANGE"
GET_URL="$HOST/api/queues/$VHOST/$DLQ/get"
PUB_URL="$HOST/api/exchanges/$VHOST/$EXCHANGE/publish"
python3 - "$GET_URL" "$PUB_URL" "$RUSER" "$RPASS" "$LIMIT" <<'PY'
import base64, json, sys, urllib.request

get_url, pub_url, user, pwd, limit = sys.argv[1], sys.argv[2], sys.argv[3], sys.argv[4], int(sys.argv[5])
auth = base64.b64encode(f"{user}:{pwd}".encode()).decode()
headers = {"Authorization": "Basic " + auth, "Content-Type": "application/json"}

def post(url, payload):
    req = urllib.request.Request(url, data=json.dumps(payload).encode(), headers=headers, method="POST")
    with urllib.request.urlopen(req, timeout=10) as resp:
        return json.loads(resp.read() or b"null")

msgs = post(get_url, {"count": limit, "ackmode": "ack_requeue_false", "encoding": "base64"})
if not msgs:
    print("    未取到消息（可能已被其他消费者取走）")
    raise SystemExit(0)
for m in msgs:
    props = m.get("properties", {}) or {}
    post(pub_url, {
        "routing_key": m.get("routing_key", ""),
        "payload": m.get("payload", ""),
        "payload_encoding": "base64",
        "properties": props,
    })
print(f"    已重新发布 {len(msgs)} 条消息；请观察 planstar_mq_duplicate_total 与消费日志。")
PY

echo "==> 重放完成。再次查询：$HOST/api/queues/$VHOST/$DLQ"
