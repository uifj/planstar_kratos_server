# 架构决策记录（ADR）台账

> 本目录**只存放 ADR 格式的决策文档**。模板见 `ADR-NNNN.md`；新决策按 `ADR-XXXX-<短名>.md` 编号，并同步更新本索引。
> 目录约定、状态流转与「失真文档先 Deprecated 再迁 legacy」流程见 `docs/README.md`。

## 决策台账

| 编号 | 标题 | 状态 | 分类 | 优先级 | 日期 | 关联 |
|---|---|---|---|---|---|---|
| [ADR-0001](./ADR-0001-modular-monolith-governance.md) | 采用模块化单体架构治理，拒绝物理拆服务与全量契约生成 | ACCEPTED | ARC | P1 | 2026-11-19 | 上位决策；定义现状问题、目标、约束、5 个备选方案、最终决策与影响 |
| [ADR-0002](./ADR-0002-directory-layout.md) | 目录结构优化方案 —— 技术层为一级目录、层内按业务域分包 | ACCEPTED | ARC | P1 | 2026-11-19 | ADR-0001 的 D2/D3/D6 落地细化；含目标目录树、逐文件迁移映射、依赖矩阵、阶段守卫 |

**状态说明**：模板默认 `PROPOSED`；本台账中 `ACCEPTED` 表示**设计决策已选定**。ADR-0001/0002 的实施按 P0–P3 分批推进，进度见下表。

> **目录重排已随本次提交完成**：`frontend/` → `planstar_frontend/`、`planstar/` → `planstar_backend/`，旧 `admin/` 重排一并纳入提交；CI paths、部署脚本与文档索引已同步（详见 `docs/local-deploy-verification.md` §1）。

## 实施进度（2026-11-19）

| 阶段 | 状态 | 主要内容 | 证据 |
|---|---|---|---|
| P0 基线治理 | ✅ 已完成 | 契约/规范口径修正、错误码单源、路由清单断言、`internal/governance` 治理断言包、CI `governance` job、`Makefile` 门禁、空目录与 11 库口径治理、legacy 死链清理 | `ADR-0001` §9.1 |
| P1 共享内核与装配 | ✅ 已完成 | `biz/shared` + `pkg/token` + 消费方 `SMSSender` 端口 + `conf.QuoteTTL()` 注入；`main.go`(26) / `wiring.go`(251) / `probes.go`(63) 拆分；`biz` 传递依赖 0 命中 `net/http` | `ADR-0001` §9.2 / §9.5 |
| P2 层内按域分包 | 🟡 首批落地 | `service/events` 子包（端口声明在消费方）、`data/project.go` 34 KB 拆为 3 个文件；**7 个业务域的物理分包按批次推进**（见下方说明） | `ADR-0002` §6 |
| P3 接缝与门禁 | ✅ 门禁已固化 | 目录/文件阈值改为**默认严格阻断**（显式 `PLANSTAR_GOVERNANCE_STRICT=0` 才降级），CI 去掉 `continue-on-error`；依赖方向、传递依赖、错误码、ADR 台账、契约怪癖、路由清单全部必过 | `ADR-0001` §9.5 |
| 目录改名 | ✅ 已完成 | `frontend/` → `planstar_frontend/`、`planstar/` → `planstar_backend/`；CI/Release/部署脚本/文档路径全量同步；两个 Go module 名保持不变 | `docs/local-deploy-verification.md` §1 |
| 本地 Docker 验证 | ✅ PASS | `docker-compose.local.yml` 三容器启动、`/healthz` `/readyz`、业务接口、`/metrics` 401、烟雾脚本全通过 | `docs/local-deploy-verification.md` §4 |

**P2 为何分批**：`biz` / `data` / `service` 三个单包共 100+ 文件共享大量包内私有符号（`s.fail`、`s.requireUser`、`r.d`、`fen()`、各 DO 等）。一次性拆成 7 个域包会使这些引用跨包失效，必须逐域抽取端口、下沉 DO、重写 import 并分批跑全量测试；把它塞进一次变更会让「目录改名 + 部署验证 + 提交」失去可回滚的稳定点。已完成的 `service/events` 与 `data/project.go` 拆分验证了迁移路径可行，剩余 6 域按 `ADR-0002` §3 映射表逐批推进。

**门禁状态**：`gofmt -l .` 空、`go build ./...` 0（两个 Go module）、`go vet ./...` 0、`go test ./... -count=1` 全绿、四份 compose `config -q` 通过、CI YAML 解析出 7 个 job、本地 Docker local deploy PASS。`go test -race` 本机因缺 `gcc` 无法执行，由 CI ubuntu runner 覆盖；`golangci-lint` 仍为报告式，待补 `.golangci.yml` 后转必过。

## 模板

| 文件 | 用途 |
|---|---|
| `ADR-NNNN.md` | 新 ADR 的骨架：状态 / 分类 / 优先级 / 影响维度 / 日期 / 来源 / 修复 commit + 背景 / 决策 / 后果 / 验证 / 关联。要求背景给出 `文件:行号`、grep 或构建输出等客观证据 |

## 参考材料（**不是 ADR，不得作为决策依据**）

| 文件 | 性质 | 处理建议 |
|---|---|---|
| `planstar_arch_optimize.md` | 43 KB 的 go-wind-cms 拆分**设计推演**文章（2026-06-18，带博客 frontmatter）。经 `ADR-0001` §1.1 逐条核对：三层骨架与 BFF 零 DB 属实，但「新增商城业务」「五层架构已落地」「Wire / Consul / Kafka」「收益数字」均与 `.archive/go-wind/` 代码不符或无证据 | 建议迁入 `docs/research/`，文首标注「设计推演，非实战复盘；收益数字未经测量」；在此之前，本目录内引用它时必须显式注明性质 |

根目录 `MIDDLEWARE.md`、`CODEGEN_VERIFICATION.md`、`KRATOS_STUDY_NOTES.md`、`KRATOS_SHOP_STUDY_NOTES.md` 与 `PLANSTAR_DEEP_STUDY.md` 同样属于归档/研究材料，建议归 `docs/notes/`，由 `docs/README.md` 标注「不得作为实现或排障依据」。

## 待立项（尚未形成 ADR）

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
4. 研究材料、推演方案、学习笔记不得放进本目录；`docs/legacy/` 与死链按 `docs/README.md` 流程处置。
