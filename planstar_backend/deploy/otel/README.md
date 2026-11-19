# OpenTelemetry Collector（当前未编排）

本目录为空，是 OTLP Collector 的占位，**otel-collector 服务未运行**。

## 现状与影响

- `configs/config.prod.yaml` 的 `trace.endpoint: otel-collector:4317`、
  `deploy/prometheus/prometheus.yml` 的 `otel-collector:8889` job、
  base compose 的 `OTEL_EXPORTER_OTLP_ENDPOINT` 都引用了该服务；
- 但四份 `docker-compose*.yml` **均未定义 `otel-collector` 服务**；
- 因此生产全量档位启用 tracing 时，OTLP/gRPC exporter 会连不上端点
  （`observability.InitTracer` 失败只记日志、不阻断启动，链路数据会静默缺失）；
- 低配档位 `config.lite.yaml` 已把 `trace.endpoint` 置空，不受影响。

## 处置选项

1. 补 collector 服务定义 + 本目录的 `otel-collector-config.yaml`，再启用 `config.prod.yaml` 的 tracing；
2. 或在 `config.prod.yaml` 中把 `trace.endpoint` 留空，并删除 Prometheus 的 otel-collector job，
   明确「当前不具备链路采集能力」。

选择前请勿把「配置开了 tracing」当成「有链路追踪能力」。详见 `docs/ADR/ADR-0001` Q7。
