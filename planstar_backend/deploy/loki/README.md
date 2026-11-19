# Loki（已决策：当前不编排）

**状态（2026-11-19，OPEN_ITEMS C10 已关闭）**：本仓库选择「删除引用并明确当前不具备日志聚合能力」，而不是补齐未验证的容器编排。

已完成的清理：
- `deploy/grafana/provisioning/datasources/prometheus.yml` 中的 Loki 数据源已移除；
- `docker-compose.yml` 头部注释不再宣称提供 Loki/Promtail；
- `docs/operations.md` 顶部标注当前能力边界，Loki 查询示例仅作为「未来启用后的目标态」保留；
- 四份 compose 仍然不定义 `loki` / `promtail` / `otel-collector` 服务。

**当前日志查看方式**：容器 `json-file` 驱动（`20m × 5` 轮转），
`docker compose logs --tail=200 planstar-api` 或 `docker logs <container>`。

**未来启用步骤**（需独立变更并验证镜像可拉取）：
1. 在 compose 中新增 `loki` 服务并挂载本目录的 `loki-config.yml`；
2. 同时启用 `../promtail/` 采集容器日志；
3. 恢复 Grafana Loki 数据源与 `docs/operations.md` 中的查询示例。

低配档位（`config.lite.yaml` / 4C8G）默认不启用，避免额外内存开销。
