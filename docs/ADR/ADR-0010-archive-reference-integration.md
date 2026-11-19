# ADR-0010: 归档项目参考集成方案立项与分期执行

- **状态**: ACCEPTED（立项：采纳 `docs/research/archive-reference-integration.md` 筛选出的可复用项，按 P0→P2 分期落地）
- **分类**: ARC（关联 SEC / DEF / FE）
- **优先级**: 分项见下（P0 数据权限与审计 / P1 契约与后台抽象 / P2 工程固化）
- **影响维度**: 安全合规 / 前端体验 / 工程一致性（**不改变既有对外契约与分层治理**）
- **日期**: 2026-11-19
- **来源**: `docs/research/archive-reference-integration.md`（对 `.archive/go-wind`、`.archive/zenith-admin` 的系统分析）
- **修复 commit**: 本批次（P0①-③ / P1④-⑥ / P2⑦-⑨ 已落地；待提交后回填 hash）

## 背景（Context）

1. 对 `.archive/go-wind`（Kratos v2 + Ent，多服务/后台两套后端）与 `.archive/zenith-admin`（Hono/Drizzle 服务端 + React/Vite/Semi 前端 monorepo）做了系统拆解，产出参考集成方案（`docs/research/archive-reference-integration.md`）。
2. 该方案筛选出 14 个可复用项，并**显式排除了与 planstar 技术栈冲突或已知缺陷的项**：Ent ORM、proto/Wire 代码生成、Kratos v2 三服务 BFF 拆分、Node/Hono/Drizzle 后端代码、Semi Design、zenith 的 `pageOffset` 分页语义。
3. 本 ADR 将「筛选结果」提升为「立项决策」，明确采纳哪些项、执行顺序、以及不改变既有契约的硬约束。

## 决策（Decision）

1. **采纳范围**（仅借鉴「设计」为主、少部分「代码」为前端组件）：
   - **P0**：① 鉴权决策审计日志（补 ADR-0007 审计闭环）；② 运营后台数据权限 `DataScope` + IDOR 防护；③ 前端请求封装（token 注入 + 401 单飞刷新 + 错误统一处理）。
   - **P1**：④ 统一 API 契约文档补强（响应/分页/文件下载）；⑤ 后台列表页/表格/表单弹窗抽象；⑥ 契约调用层统一（useRequest/TanStack Query 风格）。
   - **P2**：⑦ CRUD 模块约定文档化；⑧ wiring 分层装配清晰化 + repo 接口/事务 helper 统一；⑨ 前端页面错误边界 + chunk 失败兜底。
2. **硬约束（不变量）**：
   - 不引入 Ent / proto / Wire / 多进程拆分（`ADR-0001`、`ADR-0006`）。
   - 不改变 planstar 契约怪癖（裸 JSON 成功、`{code,message,traceId}` 错误、金额元/分、分页 `page`/`pageIndex`/`pageSize<=0→20`/`nextPageKey` 为页码、硬编码枚举、`/v1` `/v3` `/v4` 混用）。
   - 所有后端改动过 `make check`；前端改动过对应 lint/build；新增决策走 ADR。
3. **执行顺序**：P0 → P1 → P2；每项落地在 `docs/FOLLOWUP_WORK.md` 登记，关闭后同步本 ADR 与 `docs/ADR/README.md`。
4. **排除项不再实施**：§背景 3 所列冲突项按「不采纳」处理，不做兼容性修补。

## 后果（Consequences）

- **正面**：鉴权可审计、运营数据权限有行级隔离与防越权、前端请求/列表样板代码减少；可复用项有正式决策记录与优先级。
- **负面**：P0 数据权限改动触及 `data` 层查询与 `server` 鉴权链路，需谨慎防回归（治理门禁 + 集成测试兜底）。
- **中性**：多数 P1/P2 项为文档/一致性补强，不改变对外契约，风险可控。

## 验证（Verification）

- 每项实施后分别验证：
  - P0①：鉴权拒绝/通过均产出审计记录，`go test ./... -count=1` 通过。
  - P0②：不同 `DataScope` 查询结果正确，越权访问返回 403/404，`go test` 通过。
  - P0③：401 时仅一次刷新请求（单飞），错误统一出口生效。
  - 全量：`go build ./...`、`go vet ./...`、`go test ./... -count=1`、`make governance`（含 `TestADRIndexComplete`）通过。
- `docs/ADR/README.md` 登记本 ADR（已同步）。

**实施记录（2026-11-19，全部落地）**：

| 项 | 落地物 | 验证 |
|---|---|---|
| P0① 审计 | `server/ops_auth.go` 补 `rbac allowed` 审计 + `TestAdminRBACDecisionAudit` | `go test ./internal/server/... -count=1` 通过 |
| P0② 数据权限 | `biz/shared/datascope.go` + `data/shared/datascope.go` + 单测（机制落地；接线见 F27） | `go test ./internal/biz/shared/... ./internal/data/shared/... -count=1` 通过 |
| P0③ 请求封装 | `console/requestErrorConfig.ts` 补 401 跳转单飞（cookie 注入/错误处理本已有） | `biome lint` 通过 |
| P1④ 契约 | `API_CONTRACT.md` §10 文件下载约定（响应/分页本已覆盖） | 文档 |
| P1⑤⑥ 前端抽象 | 已具备（`DataTable` + 生成式客户端 + `useRequest`），无需新增 | 复核 |
| P2⑦ CRUD | `docs/CRUD_MODULE_GUIDE.md` | 文档 |
| P2⑧ 事务 | `data/shared/tx.go` `WithTx` + `user.go` 迁移示范（余 11 处见 F28） | `go build/vet/test ./...` 通过 |
| P2⑨ 前端容错 | `console/src/components/PageErrorBoundary` + `app.tsx` 接入 | `biome lint` 通过 |

全量后端门禁：`go build ./...`、`go vet ./...`、`go test ./... -count=1`（含 `internal/governance`）全部通过。

## 关联（Links）

- 参考方案：`docs/research/archive-reference-integration.md`
- 上位决策：`ADR-0001`（模块化单体）、`ADR-0007`（JWT + Casbin RBAC）、`ADR-0005`（错误映射）
- 后续台账：`docs/FOLLOWUP_WORK.md`
- 源材料：`.archive/go-wind/`、`.archive/zenith-admin/`（只读快照，不进 CI/deploy）
