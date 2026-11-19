# 架构决策记录（ADR）台账

> 本目录**只存放 ADR 格式的决策文档**。模板见 `ADR-NNNN.md`；新决策按 `ADR-XXXX-<短名>.md` 编号，并同步更新本索引。
> 目录约定、状态流转与「失真文档先 Deprecated 再迁 legacy」流程见 `docs/README.md`。

## 决策台账

| 编号 | 标题 | 状态 | 分类 | 优先级 | 日期 | 关联 |
|---|---|---|---|---|---|---|
| [ADR-0001](./ADR-0001-modular-monolith-governance.md) | 采用模块化单体架构治理，拒绝物理拆服务与全量契约生成 | ACCEPTED | ARC | P1 | 2026-11-19 | 上位决策；定义现状问题、目标、约束、5 个备选方案、最终决策与影响 |
| [ADR-0002](./ADR-0002-directory-layout.md) | 目录结构优化方案 —— 技术层为一级目录、层内按业务域分包 | ACCEPTED | ARC | P1 | 2026-11-19 | ADR-0001 的 D2/D3/D6 落地细化；含目标目录树、逐文件迁移映射、依赖矩阵、阶段守卫 |
| [ADR-0003](./ADR-0003-production-config-invariants.md) | 生产配置不变量与低配档位可启动性（显式 Cookie 豁免） | ACCEPTED | CFG | P0 | 2026-11-19 | 解决 `config.lite.yaml` 与 `validateProductionInvariants` 冲突；同步 Release 注释与 systemd 死引用 |
| [ADR-0004](./ADR-0004-mq-reliability.md) | MQ 消费幂等、DLQ 治理与 Outbox 取舍 | ACCEPTED | ARC | P1 | 2026-11-19 | 事件 ID TTL 去重 + 死信告警/重放工具；明确不引入事务性 Outbox |
| [ADR-0005](./ADR-0005-error-mapping-and-dict-enum.md) | 错误映射集中化与 `dict_enum` 运行时化决策 | ACCEPTED | DEF | P2 | 2026-11-19 | 新增 `data/shared` 错误映射层；字典表保留为数据基线，运行时用硬编码白名单 |
| [ADR-0006](./ADR-0006-p2-domain-packaging-review.md) | P2 域物理分包方案的再评估与收敛决策 | ACCEPTED | ARC | P1 | 2026-11-19 | 修订 `ADR-0002` P2：阶段性停止 7 域拆包，改用共享内核 + `platform/` 归位 + 命名分组 + 严格治理断言；列出重新拆包的触发条件 |

**状态说明**：模板默认 `PROPOSED`；本台账中 `ACCEPTED` 表示**设计决策已选定**。ADR-0001/0002 的实施按 P0–P3 分批推进，进度见下表。

> **目录重排已随本次提交完成**：`frontend/` → `planstar_frontend/`、`planstar/` → `planstar_backend/`，旧 `admin/` 重排一并纳入提交 `e514842`；CI paths、部署脚本与文档索引已同步（详见 `docs/local-deploy-verification.md` §1）。

## 实施进度（2026-11-19）

| 阶段 | 状态 | 主要内容 | 证据 |
|---|---|---|---|
| P0 基线治理 | ✅ 已完成 | 契约/规范口径修正、错误码单源、路由清单断言、`internal/governance` 治理断言包、CI `governance` job、`Makefile` 门禁、空目录与 11 库口径治理、legacy 死链清理 | `ADR-0001` §9.1 |
| P1 共享内核与装配 | ✅ 已完成 | `biz/shared` + `pkg/token` + 消费方 `SMSSender` 端口 + `conf.QuoteTTL()` 注入；`main.go`(26) / `wiring.go`(251) / `probes.go`(63) 拆分；`biz` 传递依赖 0 命中 `net/http` | `ADR-0001` §9.2 / §9.5 |
| P2 域分包与 platform 归位 | ✅ 已决策闭环 | `service/events` 子包、`data/project.go` 拆分、`internal/platform/` 归位（conf/observability/mq/ops/pkg）、`biz/shared` + `data/shared` 共享内核；`ADR-0006` 决策阶段性停止 7 域拆包并列出重拆触发条件 | `ADR-0006` / §9.5 |
| P3 接缝与门禁 | ✅ 门禁已固化 | 目录/文件阈值改为**默认严格阻断**（显式 `PLANSTAR_GOVERNANCE_STRICT=0` 才降级），CI 去掉 `continue-on-error`；依赖方向、传递依赖、错误码、ADR 台账、契约怪癖、路由清单全部必过 | `ADR-0001` §9.5 |
| 目录改名 | ✅ 已完成 | `frontend/` → `planstar_frontend/`、`planstar/` → `planstar_backend/`；CI/Release/部署脚本/文档路径全量同步；两个 Go module 名保持不变 | `docs/local-deploy-verification.md` §1 |
| 本地 Docker 验证 | ✅ PASS | `docker-compose.local.yml` 三容器启动、`/healthz` `/readyz`、业务接口、`/metrics` 401、烟雾脚本全通过 | `docs/local-deploy-verification.md` §4 |
| 功能 ADR 0003/0004/0005 | ✅ 已生成并实施 | 生产配置显式豁免、MQ 幂等与 DLQ 治理、错误映射与 dict_enum 决策 | `ADR-0003` / `0004` / `0005` |
| 功能缺口 C1-C11 | ✅ 全部闭环 | SMTP provider、OSS 接线、upload 常量清理、随机数失败关闭、可观测栈引用清理等 | `docs/OPEN_ITEMS.md` 闭环状态 |
| 测试与工程 D2-D7 | ✅ 全部闭环 | CI `quality` job、schema 集成测试、6 个 usecase 单测、压测阈值可选、Go 1.26.5 统一 + 工具链断言 | `docs/OPEN_ITEMS.md` 闭环状态 |
| 文档归档 E1-E6 | ✅ 全部闭环 | `docs/research/`、`docs/notes/`、`docs/legacy/` 删除、SQL 数字与执行顺序口径、Release/systemd 注释 | `docs/OPEN_ITEMS.md` 闭环状态 |

**P2 收敛说明（ADR-0006）**：`biz` / `data` / `service` 三个单包共 100+ 文件共享大量包内私有符号（`s.fail`、`s.requireUser`、`r.d`、`fen()`、各 DO 等）。一次性拆成 7 个域包会使这些引用跨包失效，必须逐域抽取端口、下沉 DO、重写 import 并分批跑全量测试；把它塞进一次变更会让「目录改名 + 部署验证 + 提交」失去可回滚的稳定点。已完成的 `service/events` 与 `data/project.go` 拆分验证了迁移路径可行，2026-11-19 治理批次已完成 `internal/platform/` 归位与严格门禁，并由 `ADR-0006` 决策阶段性停止 7 域拆包；重新拆包的触发条件（单包 > 60 文件 / 多团队并行 / 拆多进程）见该 ADR。

**门禁状态**（2026-11-19 治理批次后）：`gofmt -l .` 空、`go build ./...` 0（两个 Go module）、`go vet ./...` 0、`go test ./... -count=1` 与 `-tags integration`（无 DSN 自动 skip）全绿、`golangci-lint run ./...` `0 issues.`、四份 compose `config -q` 通过、`promtool check rules` 20 rules SUCCESS、CI YAML 解析出 8 个 job、本地 Docker local deploy 两次 PASS。`go test -race` 本机因缺 `gcc`/CGO 无法执行，由 CI ubuntu runner 覆盖；目录/文件阈值门禁默认严格阻断。

> **台账分两份**：[`docs/OPEN_ITEMS.md`](../OPEN_ITEMS.md) 是 33 项**已闭环**治理记录（闭环含实施、决策关闭、环境限制与预留扩展点）；
> [`docs/FOLLOWUP_WORK.md`](../FOLLOWUP_WORK.md) 是其中**仍需后续动作**的 26 项（TODO / DEBT / VERIFY / ENV / TRIGGER），
> 含 7 域拆包触发条件、ICP 备案期 Cookie 豁免、MQ 多实例去重、真实环境验证缺口等。
> ADR-0003/0004/0005/0006 已在本台账登记，不再是「待立项」。

## 模板

| 文件 | 用途 |
|---|---|
| `ADR-NNNN.md` | 新 ADR 的骨架：状态 / 分类 / 优先级 / 影响维度 / 日期 / 来源 / 修复 commit + 背景 / 决策 / 后果 / 验证 / 关联。要求背景给出 `文件:行号`、grep 或构建输出等客观证据 |

## 参考材料（**不是 ADR，不得作为决策依据**）

| 文件 | 性质 | 状态 |
|---|---|---|
| [`docs/research/go-wind-cms-split-proposal.md`](../research/go-wind-cms-split-proposal.md) | 43 KB 的 go-wind-cms 拆分**设计推演**文章（2026-06-18，带博客 frontmatter）。经 `ADR-0001` §1.1 逐条核对：三层骨架与 BFF 零 DB 属实，但「新增商城业务」「五层架构已落地」「Wire / Consul / Kafka」「收益数字」均与 `.archive/go-wind/` 代码不符或无证据 | ✅ 已迁入 `docs/research/`，文首已标注「设计推演，非实战复盘；收益数字未经测量」；仅作思路参考，不得作决策依据 |

`docs/notes/` 已收纳 `PLANSTAR_DEEP_STUDY.md`、`MIDDLEWARE.md`、`CODEGEN_VERIFICATION.md`、`KRATOS_STUDY_NOTES.md`、`KRATOS_SHOP_STUDY_NOTES.md`，由 `docs/README.md` 标注「不得作为实现或排障依据」。

## 已立项（2026-11-19 已生成 ADR-0003/0004/0005 并实施）

| 建议编号 | 主题 | 触发原因 |
|---|---|---|
| ADR-0003 | 生产配置不变量与 `config.lite.yaml` 可启动性修复 | `env:prod` 下 `cookie_secure:false` / `mysql.user:root` / 空 ops 令牌必被 `conf.Validate()` 拒绝，而 `release.yml` 硬编码该档位 |
| ADR-0004 | MQ 消费幂等、DLQ 治理与事务性 Outbox | 消费端无 `ev.ID` 去重、DLQ 无消费者/告警/重放、发布失败只 Warn、`catalog.activity.*` 无发布点 |
| ADR-0005 | 错误映射集中化与 `dict_enum` 运行时化/废弃决策 | 驱动错误映射散落手写、无统一 adapter；契约称「枚举取自 dict_enum」但代码为硬编码白名单 |

---

**维护规则**

1. 新增/变更 ADR 必须同步本表，并保证「状态」「日期」「关联」三列真实。
2. 每个 ADR 的「背景」必须给出可复核证据（路径 + 行号 / 命令输出），不接受主观评价。
3. 涉及对外契约（路径、Cookie、错误码、金额单位、分页参数）的决策，须显式标注「是否需要 Flutter 客户端协同」。
4. 研究材料、推演方案、学习笔记不得放进本目录；归档材料见 `docs/notes/` 与 `docs/research/`，`docs/legacy/` 空目录已删除。
