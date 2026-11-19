# ADR-0006: P2 域物理分包方案的再评估与收敛决策

- **状态**: ACCEPTED（对 ADR-0002 P2 的修订）
- **分类**: ARC（关联 DEBT）
- **优先级**: P1
- **影响维度**: 认知障碍 / 上下文膨胀
- **日期**: 2026-11-19
- **来源**: `docs/OPEN_ITEMS.md` A1/A3/A4/A5、`ADR-0002` §1/§6 实施反馈
- **修复 commit**: 治理批次（本次）

## 背景（Context）

`ADR-0002` 的 P2 目标是把 `biz` / `data` / `service` 三个单包按 7 个业务域（`identity`/`catalog`/`project`/`trade`/`wallet`/`community`/`ops`）物理拆成独立 Go 包，并在 P3 删除兼容层。治理批次已完成的前置工作：

- `biz/shared`（错误/事件/单位/时间/脱敏）与 `data/shared`（驱动错误映射）共享内核；
- `service/events` 独立子包（端口声明在消费方）；
- `data/project.go` 34 KB 拆为 3 个文件；
- `internal/platform/` 归位（`conf`/`observability`/`mq`/`ops`/`pkg`）；
- 严格治理门禁：依赖方向、传递依赖、文件/包阈值、错误码单源、路由清单、契约怪癖全部 CI 必过。

在此基础上对「剩余 6 域全量分包」做了成本评估，客观事实：

1. 三个包共 100+ 文件共享包内私有符号：`s.fail`、`s.requireUser`、`s.limit`、`r.d.log`、`r.d.DB(...)`、`fen()` / `FmtDT()` 以及 `models.go` 中的全部 DO。
2. 拆包必须同时：下沉 DO 到 `biz/shared/do.go`、为每个域重写 import、迁移并改写 30 个同包白盒测试、为跨域引用额外声明端口、改造 `*Data` 的字段可见性、重写装配层与治理测试的路径断言。
3. 当前最大单包 `biz` 为 29 个 `.go`（含 11 个测试），`data` 21 个，`service` 17 个；均远低于治理阈值 40。最大单文件已降到 14 KB。
4. 服务是**单进程单镜像单 MySQL 实例**（ADR-0001），按域拆 CI 路径、指标 label 或 k8s 部署单元在当前形态下没有对应收益。

## 决策（Decision）

1. **阶段性地停止把 7 个域拆成独立 Go 包**（修订 ADR-0002 P2）。域边界改由以下机制物理表达，而不是 Go package 边界：
   - 共享内核 `biz/shared` / `data/shared`；
   - 独立子包 `service/events`（已落地）与横切层 `internal/platform/`（已落地）；
   - 域实现保留在单包内，按域命名前缀分组（`project*.go`、`quote*.go`/`order.go`/`payment.go`、`feed.go`/`message.go`/`chat.go` 等）；
   - `internal/governance` 强制依赖方向、传递依赖与阈值，防止单包重新膨胀。
2. **A3（`service/{api,admin}` 拆分）暂不执行**：`s.fail` / `s.requireUser` / `s.authCfg` 等私有成员被全部 handler 共享，拆分需先抽公共 handler 基类；已通过 `service/events` 验证了「按端拆包」的可行路径，留作未来触发条件满足时的第一批。
3. **A4（删除 `biz` 根包兼容层）取消**：该层是 P1 有意设计的稳定门面（共享 DO、哨兵错误、事件类型的单一导出点），删除会迫使 `data`/`service`/`cmd` 直接依赖多个域包，反而增加耦合。保留并明确其定位。
4. **A5（按域 CI 路径 / 指标 label / 候选服务清单）改为文档化**：在 `docs/OPEN_ITEMS.md` 与 `ADR-0002` §3 维护域清单与迁移映射；CI 继续按 module 触发（单镜像），指标继续按 route 归一化（避免高基数）。不再产出「候选服务清单」。
5. **重新拆包的触发条件（写入治理断言注释）**：任一单包 `.go` 文件数 > 60，或同一包出现两个以上团队并行高频修改，或确实要拆成多进程时，按 `ADR-0002` §3 映射表**逐域**（每域一次提交）迁移。

## 后果（Consequences）

- **正面**：避免了为「目录好看」而进行的高风险大规模 import 重写；系统保持可运行、可回滚、可验证；域边界仍可通过命名与治理断言被机器检查。
- **负面/风险**：域之间在编译期没有包级隔离，误跨域引用只能靠治理断言与 code review 拦截；若未来团队规模扩大，P2 仍需重新执行。
- **中性**：`ADR-0002` 的目标目录树降级为「长期演进参考」而非本阶段验收标准；`docs/OPEN_ITEMS.md` A1/A3/A4/A5 以本 ADR 的决策作为关闭依据。

## 验证（Verification）

- `go build ./...` / `go vet ./...` / `go test ./... -count=1` / `golangci-lint run ./...` 全部通过（治理批次实测）。
- `go test ./internal/governance -count=1`：依赖方向、传递依赖、阈值（默认严格）、错误码单源、ADR 台账、契约怪癖、路由清单、工具链版本全部通过。
- `grep -rn 's.fail\\|s.requireUser' planstar_backend/internal/service/` 仍能命中（说明 A3 未做且兼容层保留，符合本决策）。
- `docs/OPEN_ITEMS.md` 中 A1/A3/A4/A5 状态标记为「已决策关闭（ADR-0006）」。

## 关联（Links）

- 代码: `planstar_backend/internal/{biz,data,service,platform,governance}/`
- 文档: `ADR-0001`（模块化单体）、`ADR-0002`（原目录方案）、`docs/OPEN_ITEMS.md`
- 关联 ADR: `ADR-0005`（data/shared 落点）
