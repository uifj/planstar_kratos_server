# OpenTelemetry Collector（已决策：当前不编排）

**状态（2026-11-19，OPEN_ITEMS C10 已关闭）**：本仓库选择「删除引用并明确当前不具备链路采集能力」。

已完成的清理：
- `configs/config.prod.yaml` 的 `trace.endpoint` 由 `otel-collector:4317` 改为空字符串（tracing 关闭）；
- `deploy/prometheus/prometheus.yml` 的 `otel-collector` 抓取 job 已移除；
- `docker-compose.yml` 的 `OTEL_EXPORTER_OTLP_ENDPOINT` 默认值与相关注释已清理；
- `docs/operations.md` 顶部标注当前能力边界。

**影响**：`observability.InitTracer` 在 endpoint 为空时返回 no-op shutdown，不会阻断启动；
日志中的 `trace_id` / `span_id` 仍由 `NewTraceHandler` 注入，可用于日志关联，但**没有远端链路数据**。
低配档位 `config.lite.yaml` 本就把 endpoint 置空，不受影响。

**未来启用步骤**（需独立变更并验证镜像可拉取）：
1. 在 compose 中新增 `otel-collector` 服务并挂载本目录的 `otel-collector-config.yml`；
2. 恢复 `config.prod.yaml` 的 endpoint 与采样率、Prometheus 的 collector job；
3. 恢复 `docs/operations.md` 中 OTel/Jaeger 排障与巡检项。

选择前请勿把「配置开了 tracing」当成「有链路追踪能力」。
