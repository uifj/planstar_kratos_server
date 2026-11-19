# Promtail（当前未编排）

本目录为空，是 Loki 日志采集 agent 的占位，**Promtail 未运行**。

- 依赖 `../loki/`：只有 Loki 服务存在时 Promtail 才有意义；
- 四份 `docker-compose*.yml` 均未定义 `promtail` 服务；
- 当前日志通过 Docker `json-file` 驱动落盘，由 `docker compose logs` 或 `docker logs` 查看。

启用前请同步：compose 服务定义、Promtail scrape config、Loki 地址与标签
（`service="planstar-api"`）。启用条件与低配档位取舍见 `../loki/README.md`。

详见 `docs/ADR/ADR-0001` Q7。
