# Loki（当前未编排）

本目录为空，是**日志聚合能力的占位**，不代表 Loki 正在运行。

## 现状

- `planstar_backend/deploy/grafana/provisioning/datasources/prometheus.yml` 配置了
  `http://loki:3100` 数据源，`docs/operations.md` 的告警处置也提到「检索 Loki 日志」；
- 但四份 `docker-compose*.yml` **均未定义 `loki` 服务**，容器内日志目前只走
  `json-file` 驱动（`20m × 5` 轮转），`{service="planstar-api"} |= "ops audit"` 查询不会命中。

## 启用条件

1. 在 compose 中新增 `loki` 服务并挂载本目录的配置；
2. 同时启用 `../promtail/`（采集容器日志）与 Grafana Loki 数据源；
3. 低配档位（`config.lite.yaml` / 4C8G）默认**不启用**，避免额外内存开销。

详见 `docs/ADR/ADR-0001` Q7 与 `docs/deployment.md`。若长期不启用，应删除 Grafana
数据源配置与文档中的 Loki 引用，避免「配置已写、能力不存在」的治理幻觉。
