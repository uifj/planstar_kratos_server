# Promtail（已决策：当前不编排）

**状态（2026-11-19，OPEN_ITEMS C10 已关闭）**：与 `../loki/` 一致，选择「清理引用、明确当前不具备日志聚合能力」。

- 四份 `docker-compose*.yml` 均未定义 `promtail` 服务；
- Grafana Loki 数据源已移除；
- 当前日志通过 Docker `json-file` 驱动落盘，用 `docker compose logs` 查看。

**未来启用步骤**：先按 `../loki/README.md` 启用 Loki，再新增 `promtail` 服务、
`promtail-config.yml` scrape config，并确认 `service="planstar-api"` 标签生效。
恢复后需同步更新 `docs/operations.md` 的日志检索章节。
