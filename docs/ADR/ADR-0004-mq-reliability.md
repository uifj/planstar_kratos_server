# ADR-0004: MQ 消费幂等、DLQ 治理与 Outbox 取舍

- **状态**: ACCEPTED（已实施）
- **分类**: ARC（关联 DEP / TST）
- **优先级**: P1
- **影响维度**: 生产功能
- **日期**: 2026-11-19
- **来源**: `docs/OPEN_ITEMS.md` B2、`docs/notes/PLANSTAR_DEEP_STUDY.md` §13
- **修复 commit**: 治理批次（本次）

## 背景（Context）

`internal/mq` 采用 topic exchange + broker 级 DLX，最多 5 次指数退避重试。整改前存在：

- 消费端不校验 `DomainEvent.ID`，重试/重连会重复投递，导致重复站内通知；
- DLQ `<queue>.dlq` 只声明与绑定，**无消费者、无告警、无重放工具**；
- 发布失败在 `AuthUsecase.Login` / `OrderUsecase.MakeOrder` 中只 `Warn`，无事务性 Outbox；
- `catalog.activity.changed/deleted` 只有常量与订阅、无发布点；目录读模型实际靠 300s 全量重建兜底；
- 重试是消费者内 `time.After` 阻塞，串行 `Run` 下单条退避会拖住整队列。

## 决策（Decision）

1. **消费端幂等优先**：在 `Consumer` 内基于 `ev.ID` 做带 TTL 的已处理集合（默认保留 24h，容量上限 + 惰性清理），命中即 `Ack` 并跳过；同时提供 `planstar_mq_duplicate_total` 指标。理由：MQ 只承载站内通知与目录增量，不承载资金/订单一致性，消费端幂等足以消除重复投递的主要后果。
2. **以「幂等 + 可观测告警 + 重放工具」替代事务性 Outbox**：订单/支付一致性由 MySQL 单事务 + 唯一键保证（见 `data/order.go`），事件发布失败只影响通知，不引入跨存储事务。明确**不引入 Outbox/Saga**，避免为低价值消息付出分布式事务复杂度。
3. **DLQ 治理**：新增 `planstar_mq_consume_failed_total` / `planstar_mq_dlq_total` 指标与 Prometheus 告警规则（持续增长即告警），并提供 `scripts/dlq-replay.sh` 用 RabbitMQ management API 把 DLQ 消息移回主队列（需人工确认根因后再执行）。
4. **`catalog.*` 事件定位为预留扩展点**：当前目录同步由 `RunSync` 全量重建保证最终一致，不伪造发布点；`routingKeys` 与消费分支保留，待真正出现目录写入口时补齐发布者，并在 `mq/server.go` 与 `service/events/handler.go` 注释中明确标注。
5. **重试阻塞改为后续批次**：先用告警与重放工具覆盖运维需求；将重试队列化（TTL 延迟队列）需要额外 exchange/queue 拓扑与灰度，另立变更，不在本 ADR 内。

## 后果（Consequences）

- **正面**：重复投递不再产生重复通知；DLQ 消息可被发现、告警与人工重放；MQ 可靠性边界有明确文档，不再被误认为「已具备 exactly-once」。
- **负面/风险**：内存去重是**单实例语义**，多实例部署时同一事件可能在不同实例各处理一次；彻底消除需 Redis/MySQL 去重表（后续批次）。发布失败仍可能丢通知（已被接受，见决策 2）。
- **中性**：`catalog.*` 保持空链路但被明确标注为预留，不再算「未发现的缺口」。

## 验证（Verification）

- `go test ./internal/mq ./internal/service/events -count=1`：新增去重单测（同 ID 重复投递只处理一次、不同 ID 正常处理、TTL 过期后可再次处理、容量上限不无限增长）。
- `promtool check rules deploy/prometheus/rules/alerts.yml` 通过。
- `bash scripts/dlq-replay.sh --dry-run` 能列出 DLQ 消息而不移动。
- `grep -rn 'planstar_mq_' planstar_backend/` 能找到指标定义、埋点与告警规则三处。

## 关联（Links）

- 代码: `planstar_backend/internal/mq/{consumer.go,publisher.go,server.go,dedup.go,metrics.go}`、`internal/service/events/handler.go`、`internal/biz/shared/event.go`
- 运维: `planstar_backend/deploy/prometheus/rules/alerts.yml`、`planstar_backend/scripts/dlq-replay.sh`、`docs/operations.md`
- 关联 ADR: `ADR-0001`（不拆进程，MQ 非一致性边界）、`ADR-0005`
