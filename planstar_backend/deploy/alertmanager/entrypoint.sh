#!/bin/sh
# =====================================================================
# Alertmanager 启动包装：把 alertmanager.yml 中的 ${AM_*} 占位渲染为环境变量。
#
# 背景：Alertmanager（0.27 及后续版本）本身不支持 ${VAR} 展开；
# --config.expand-env 是 Prometheus 的参数，不能用于 Alertmanager。
# 为避免把 SMTP 凭据写进配置文件，这里在启动前用 awk 渲染模板到 /tmp
# （umask 077），再启动 Alertmanager。
#
# 调试：ALERTMANAGER_RENDER_ONLY=1 时只输出渲染结果，便于 amtool 校验。
# =====================================================================
set -eu

# 未配置 SMTP 时使用占位值启动：Alertmanager 会因 "missing to address" 直接
# 退出并陷入重启循环，占位值可让进程保持可用（邮件发送会失败并留日志），
# 配置 AM_* 后重启即可真正送达。
placeholder=0
if [ -z "${AM_SMTP_SMARTHOST:-}" ]; then
  AM_SMTP_SMARTHOST="smtp.example.invalid:587"
  placeholder=1
fi
if [ -z "${AM_SMTP_FROM:-}" ]; then
  AM_SMTP_FROM="alertmanager@example.invalid"
  placeholder=1
fi
if [ -z "${AM_ALERT_EMAIL_TO:-}" ]; then
  AM_ALERT_EMAIL_TO="ops@example.invalid"
  placeholder=1
fi
if [ "$placeholder" = "1" ]; then
  echo "WARN: AM_SMTP_SMARTHOST/AM_SMTP_FROM/AM_ALERT_EMAIL_TO 未完整配置，使用占位地址启动；邮件通知不会真正送达。" >&2
fi
export AM_SMTP_SMARTHOST AM_SMTP_FROM AM_ALERT_EMAIL_TO

TEMPLATE="${ALERTMANAGER_TEMPLATE:-/etc/alertmanager/alertmanager.yml}"
RENDERED="${ALERTMANAGER_RENDERED:-/tmp/alertmanager.yml}"

umask 077
awk '
{
  line = $0
  out = ""
  while (match(line, /\$\{[A-Z0-9_]+\}/)) {
    name = substr(line, RSTART + 2, RLENGTH - 3)
    out = out substr(line, 1, RSTART - 1) ENVIRON[name]
    line = substr(line, RSTART + RLENGTH)
  }
  print out line
}' "${TEMPLATE}" > "${RENDERED}"

if [ "${ALERTMANAGER_RENDER_ONLY:-0}" = "1" ]; then
  cat "${RENDERED}"
  exit 0
fi

exec /bin/alertmanager --config.file="${RENDERED}" "$@"
