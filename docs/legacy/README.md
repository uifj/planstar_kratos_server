# Legacy 文档区（当前为空）

本目录用于存放**已废弃、仅作历史对照**的架构文档。按 `docs/README.md` 的约定，
失真文档应先标记 `Deprecated`，再迁移到本目录；**本目录中的任何内容都不得作为
实现、部署或排障依据**。

## 当前状态（2026-11-19 治理核查）

- 目录为空。`docs/README.md` 与 `docs/architecture.md` 中曾列出的
  `architecture-microservices.md`、`configuration.md`、`database.md`、
  `development.md`、`evolution-plan.md`、`feasibility-p1.md` 等文件**未随本仓库检出**，
  因此所有指向它们的链接都是死链，请勿据此推断历史架构。
- 历史架构（gRPC 微服务 / `shop_*` / PostgreSQL / Consul）已被
  `docs/architecture.md` 与 `docs/ADR/ADR-0001` 明确否决，当前形态是**模块化单体**。
- 需要查阅历史决策时，以 `docs/ADR/` 台账与 `.archive/go-wind/` 只读快照为准。

治理动作见 `docs/ADR/ADR-0001` D7：要么补入真实废弃文档，要么删除本目录并清理死链。
