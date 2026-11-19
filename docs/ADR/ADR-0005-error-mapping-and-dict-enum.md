# ADR-0005: 错误映射集中化与 dict_enum 运行时化决策

- **状态**: ACCEPTED（已实施）
- **分类**: DEF / DEBT
- **优先级**: P2
- **影响维度**: 认知障碍 / 生产功能
- **日期**: 2026-11-19
- **来源**: `docs/OPEN_ITEMS.md` B3、`agents.md` 契约口径修正
- **修复 commit**: 治理批次（本次）

## 背景（Context）

1. 全仓 `.go` 搜 `dict_enum` **0 命中**：`fundingStatus`、`sort`、`payChannels`、`refundReasonCodes`、创意投票理由码等全是 `biz/project.go`、`biz/payment.go`、`biz/project_idea.go` 中的硬编码白名单。而 `planstar_meta.dict_enum` 有建表与 143 条种子，`agents.md` 旧版却写「枚举取自 dict_enum」，构成口径漂移。
2. `internal/data` 无统一 driver→biz 错误映射：`sql.ErrNoRows`、MySQL 1062、`redis.Nil`、ES 错误分散在十余个 repo 中手写；唯一公共 helper 是 `data/order.go:401 isDuplicateKey`。
3. 错误码字符串已由 `internal/pkg/errcode` 单点定义（ADR-0001 P0），但**驱动错误 → 领域错误**这一层仍未收敛。

## 决策（Decision）

1. **`dict_enum` 保留为数据基线与逆向证据，不做运行时读取**。理由：枚举是 API 契约的一部分，必须与代码同版本发布；运行时字典可被运营改库后静默改变校验行为，且与 Flutter 客户端的联合类型不同步。白名单继续放在 biz 域代码中，由 `internal/governance` 契约断言守护。
2. **新增 `data/shared` 错误映射层**：提供 `MapQueryError`（`sql.ErrNoRows` → 可配置 fallback）、`IsDuplicateKey`、`MapRedisError`、`MapContextError` 等统一 helper；driver 错误语义只在此处定义一次。
3. **分批收敛**：优先在认证、订单、支付、退款、钱包链路使用统一 helper；其余读面 repo 保留原有显式分支（行为等价），并标注「P2 域分包时统一替换」。不追求一次性重写全部 SQL 分支，避免把治理变成大规模回归。
4. **1062 语义保持**：重复键在多数场景是「幂等命中」而非错误（重试单号、回读首单、签到、投票），helper 只负责识别，不擅自转成 `ErrConflict`。
5. `agents.md` 已改为「枚举为硬编码白名单」，本 ADR 为该口径提供决策依据。

## 后果（Consequences）

- **正面**：枚举行为可预测、可版本化；驱动错误语义有单一解释点；新增 repo 不必重复记忆 `sql.ErrNoRows` 的十种写法。
- **负面/风险**：`dict_enum` 表与代码白名单可能不同步（需要离线巡检，见 `docs/operations.md`）；未收敛的读面 repo 仍存在重复代码。
- **中性**：`data/shared` 与 `biz/shared` 的命名对应，为 P2 分包提供落点。

## 验证（Verification）

- `go test ./internal/data ./internal/data/shared -count=1`：helper 表驱动单测覆盖 no rows / duplicate key / redis nil / context 超时 / 透传未知错误。
- `grep -rn 'dict_enum' planstar_backend/internal/` **0 命中**（运行时不读字典）。
- `grep -rn 'sql.ErrNoRows' planstar_backend/internal/data/` 的命中数在后续批次中单调下降。
- `internal/governance` 断言「`AGENTS.md` 不含旧的 dict_enum 口径」通过。

## 关联（Links）

- 代码: `planstar_backend/internal/data/shared/errors.go`、`internal/data/{order.go,payment.go,auth.go,wallet.go}`、`internal/pkg/errcode/errcode.go`
- 文档: `agents.md`、`docs/api/API_CONTRACT.md`、`docs/sql/10_dict_enums.sql`
- 关联 ADR: `ADR-0001`（错误码单源）、`ADR-0002`（data/shared 落点）、`ADR-0004`
