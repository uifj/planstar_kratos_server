# 归档项目参考集成方案（go-wind / zenith-admin）

- **日期**：2026-11-19
- **来源**：`.archive/go-wind`（后端）、`.archive/zenith-admin`（前端/全栈）
- **目标**：筛选可迁移到 `planstar_backend`（Go Kratos v3）与 `planstar_frontend`（React+Umi 管理后台 / Next.js 只读 web）的功能点与设计模式
- **性质**：**参考材料，不是决策依据**。每一项落地前须按 `planstar_backend/internal/governance` 门禁与 `docs/ADR` 流程评审；本文件不直接改动任何代码。
- **判定口径**：只保留「设计合理 + 可适配当前技术栈 + 无已知缺陷」的项；与当前技术栈冲突、已过时、存在已知缺陷的项在 §7 显式排除。

---

## 0. 结论速览

| 维度 | 结论 |
|---|---|
| 后端可复用 | go-wind 的「认证中间件链 + 鉴权后注入上下文」「repo 接口/实现分离 + 事务 defer 闭包」「错误码封装」「CRUD 模块 11 步约定」价值最高，但 Ent/proto 代码生成必须排除 |
| 前端可复用 | zenith-admin 的「请求封装（token 注入/401 并发刷新/错误统一处理）」「TanStack Query 契约调用层」「列表页/表格/表单弹窗抽象」「IAM 权限模型与数据权限 DataScope」价值最高，但 Semi Design 需替换为 Ant Design |
| 核心冲突 | Ent ORM、proto/Wire 代码生成、Kratos v2、多服务 BFF 拆分、Node/Hono/Drizzle 后端、Semi Design 均与 planstar 技术栈冲突，列为排除项 |
| 优先级 | P0 权限模型与数据权限、P0 请求封装与错误体系；P1 契约调用层与列表页抽象；P2 CRUD 约定与工程化补强 |

---

## 1. 源项目技术栈与架构速览

### 1.1 go-wind（后端）

`go-wind` 下有两个独立后端：

| 子项目 | 定位 | 关键证据 |
|---|---|---|
| `go-wind-cms/backend` | 多租户 Headless CMS，Kratos v2.9.2 + Ent v0.14.6，三服务架构 | `.archive/go-wind/go-wind-cms/backend/AGENTS.md`、`go.mod` |
| `go-wind-admin/backend` | 后台管理系统，Kratos v2 + Ent + `gow` 脚手架，REST/gRPC + CRUD 自动生成 | `.archive/go-wind/go-wind-admin/backend/README.md`、`go.mod` |

**go-wind-cms 三服务分层**（`backend/AGENTS.md:25-60`）：
- `admin-service` / `app-service`：瘦 BFF 网关，只做协议转换与转发。
- `core-service`：核心业务与数据落点。
- `api/` proto 契约层、`app/<admin|app|core>/service/` 应用实现、`pkg/` 跨服务公共库、`tools/register/` 模块装配。

**go-wind-admin 目录结构**（`docs/backend_project_struct.md`）：`api/protos` + `api/gen/go`、`app/admin/service/internal/{data,service,server}`、`pkg/`、`sql/`、`Makefile`。

### 1.2 zenith-admin（前端/全栈 monorepo）

| 包 | 技术栈 | 关键证据 |
|---|---|---|
| `packages/server` | Node.js + Hono v4 + `@hono/zod-openapi` + Drizzle ORM | `docs/guide/project-structure.md:24-44` |
| `packages/web` | React 19 + Vite + Semi Design + TanStack Query + react-router | `docs/guide/project-structure.md:56-85`、`packages/web/package.json` |
| `packages/shared` | 共享契约：`contracts/` `validation.ts` `constants.ts` `types.ts` | `docs/guide/project-structure.md:87-105` |
| `packages/electron` | Electron 桌面容器（承载 Web 构建产物） | `packages/electron/src/main.ts` |
| `packages/analytics-sdk` | 分析埋点 SDK | — |

---

## 2. go-wind 后端拆解

### 2.1 核心业务模块

| 模块 | 职责 | 关键路径 |
|---|---|---|
| 认证 | Bearer token 提取与校验、payload 注入 context | `pkg/middleware/auth/auth.go:25-67` |
| 鉴权/多租户 | 认证后注入 Ent viewer、operator metadata，执行租户/组织/数据权限 | `pkg/middleware/auth/auth.go:82-128` |
| 用户/内容等业务域 | repo 接口 + service 实现分离 | `AGENTS.md:121-140` 的 user 模块示例 |
| CRUD 模块 | proto → service → data → server → wiring 11 步 | `docs/crud_module_guide/README.md` |

### 2.2 代码组织规范

- **接口/实现分离**：每个业务域声明 repo 接口，`NewXxx` 构造器注入；service 只依赖接口不依赖 data 实现。
- **事务模式**：`defer` 闭包 + ent 事务客户端。
- **分层装配**：`wiring_ent.go:21-40` 按「基础设施 → 仓储层 → 认证与鉴权 → 服务层 → 传输层」单向分层手动装配。
- **模块装配工具**：`tools/register/` 集中注册服务模块，避免 main 入口膨胀。

### 2.3 通用工具组件

| 组件 | 实现要点 | 路径 |
|---|---|---|
| 认证中间件 | 从 Kratos transport metadata 取 Bearer，校验 access token，失败返回统一错误 | `pkg/middleware/auth/auth.go:24-64` |
| 鉴权引擎 | authz（casbin/opa/noop 可插拔）在 HTTP 中间件层执行，策略热重建，写 PolicyEvaluationLog 审计 | `docs/tutorial/05-permission-model.md:20-53` |
| 错误码封装 | 统一业务错误码 + 错误映射 | `AGENTS.md:121-140` 示例 |
| CRUD 生成工具 | `go-crud`、`go-utils` 通用 CRUD/工具组件 | `go-wind-admin/backend/go.mod` |

### 2.4 权限管理方案

- 接口级授权由 authz 引擎在 HTTP 中间件层执行（不在 handler 内散落判断）。
- 策略热重建，变更后即时生效。
- 每次鉴权决策写入 `PolicyEvaluationLog` 用于审计回溯。
- 认证后把用户身份 + 组织/租户/数据权限统一注入 context，后续 data 层据此做数据过滤。

### 2.5 接口设计模式

- 路由注册集中在 service 层；BFF 层「薄转发 + core 真实业务」。
- 请求/响应用 proto 定义（`api/protos`），统一代码生成。
- 错误统一为结构体返回。

### 2.6 数据交互逻辑

- Ent ORM：schema 定义实体 + 关联，`make ent` 生成查询 builder。
- 事务用 `defer` 闭包保证提交/回滚。
- 迁移机制由 Ent schema + sql 目录配合。

### 2.7 工程化配置

- `Makefile` 驱动 `make api` / `make ent` / `make register` / `make build`。
- Docker、CI workflow（`.github/workflows`）、Swagger/OpenAPI 输出。
- 环境变量 + 配置文件分离（多环境档位）。

---

## 3. zenith-admin 前端拆解

### 3.1 核心业务模块（web）

| 模块 | 职责 | 关键路径 |
|---|---|---|
| 入口与路由 | BrowserRouter/HashRouter、RouteSuspense、懒加载 AdminLayout | `packages/web/src/App.tsx:2-23` |
| 登录/守卫 | 登录重定向、已登录访问 `/login` 跳 403/404 | `App.tsx:91-155` |
| 列表页 | 状态开关、搜索工具栏、筛选、删除、批量状态、分页、行选择 | `components/list-page/index.ts` |
| 表格接入 | `listTableProps` 把 TanStack Query 列表接入 Semi Table | `components/list-page/listTableProps.ts:5-49` |
| 可配置表格 | 列设置、刷新、加载态 | `components/ConfigurableTable.tsx` |
| 表单弹窗 | `EditFormModal`：AppModal > Spin > Form，页面只写字段 | `components/EditFormModal.tsx:24-55` |
| 错误边界 | 页面级错误边界 + chunk 加载失败处理 | `components/PageErrorBoundary.tsx` |
| 空状态 | 懒加载 light/dark 插画 | `components/EmptyIllustration.tsx` |

### 3.2 状态管理与数据获取

- TanStack Query 单例，统一响应解包。
- 契约调用层：`api/useApiQuery` / `useApiMutation` / `createResourceQueries`。
- 服务端状态（列表/详情）走 Query 缓存，本地 UI 状态走组件内部 state。

### 3.3 请求封装

- `HttpClient`：token 注入、401 刷新、**并发刷新去重**、错误统一处理、流式/SSE 支持。
- 后台/会员/审批等不同域使用独立请求实例（`docs/frontend/auth-request.md:58-93`）。

### 3.4 权限控制

- 路由级：根据菜单/权限码生成动态路由与守卫。
- 按钮级：权限指令/组件/hook 控制显隐。
- 数据权限：`DataScope` 枚举 + 角色/用户/组权限合并（最宽松原则）+ 部门递归 + 本人字段过滤（IDOR 防护），见 `docs/iam/index.md:72-92`。

### 3.5 工程化

- Vite + `tsc -b` + 自定义 `scripts/build.mjs`；`vitest` 测试；`eslint/stylelint`。
- monorepo（pnpm workspace / turbo）脚本统一编排。

---

## 4. zenith-admin 服务端与共享层拆解（可借鉴的设计）

### 4.1 API 约定（`docs/backend/api-conventions.md`）

- 统一成功/失败响应格式。
- 分页统一返回格式。
- 文件下载响应约定。
- 认证：Access/Refresh Token + JWT + `authMiddleware` + `currentUser()`。
- 参数校验：Zod 校验 + 契约路由。
- 权限/审计从契约的 `access` / `audit` 元数据装配。

### 4.2 数据库操作（`docs/backend/database-operations.md`）

- 分页统一 `withPagination` / `pageOffset`。
- 关联查询优先 RQB（Relation Query Builder）。
- Drizzle relations + SQL builder + 查询客户端分页规范。

### 4.3 IAM/RBAC（`docs/iam/index.md`）

- 表模型：用户、角色、菜单、权限码、用户组、数据权限。
- 权限解析入口 + Redis 缓存 + 超管判定 + 权限清理。
- `DataScope` 枚举 + 角色/用户/组权限合并 + 最宽松原则 + 部门递归 + IDOR 防护。

### 4.4 shared 共享层

- 前后端共享 `contracts/`、`validation.ts`、`constants.ts`、`types.ts`，保证类型与校验一致。

---

## 5. 可复用项筛选总表

> 标注：**适用场景 / 实现逻辑 / 优势**；「代码/设计」区分是直接迁移代码还是仅借鉴设计。

| # | 可复用项 | 来源 | 代码/设计 | 适用场景 | 优势 |
|---|---|---|---|---|---|
| 1 | 认证中间件链 + 鉴权后注入上下文 | go-wind `pkg/middleware/auth` | 设计 | planstar 9108 JWT 中间件、C 端 psSession 中间件 | 身份/权限集中注入，handler 不散落判断 |
| 2 | repo 接口/实现分离 + 事务 defer 闭包 | go-wind user 模块 | 设计 | planstar biz/data 分层（已有雏形） | 依赖方向清晰、事务边界可控 |
| 3 | authz 引擎可插拔 + 策略热重建 + 审计日志 | go-wind `docs/tutorial/05` | 设计 | planstar Casbin RBAC（ADR-0007）补审计 | 鉴权决策可审计、策略热生效 |
| 4 | 错误码封装与统一错误映射 | go-wind | 设计 | planstar `data/shared` 错误映射（ADR-0005） | 错误单源、可映射 |
| 5 | CRUD 模块 11 步约定 | go-wind `docs/crud_module_guide` | 设计 | planstar 新增业务域的标准流程 | 新模块开发顺序清晰、防漏 |
| 6 | 分层单向装配（wiring） | go-wind `wiring_ent.go` | 设计 | planstar `cmd/planstar/wiring.go`（已有） | 装配顺序固定、易审查 |
| 7 | 统一请求封装（token/401 并发刷新/错误处理） | zenith `docs/frontend/auth-request.md` | 设计+代码 | planstar 管理后台 axios/useRequest 封装 | 鉴权失效体验稳定、防重复刷新 |
| 8 | TanStack Query 契约调用层 | zenith `docs/frontend/data-fetching.md` | 设计 | planstar 后台 useRequest/ahooks 适配 | 服务端状态与缓存统一 |
| 9 | 列表页/表格/表单弹窗抽象 | zenith `components/list-page` 等 | 设计+代码 | planstar 后台 ProTable/ProForm 适配 | 减少列表页样板代码 |
| 10 | 页面错误边界 + chunk 失败处理 | zenith `PageErrorBoundary.tsx` | 代码 | planstar 后台/Next.js | 前端容错、动态导入失败兜底 |
| 11 | IAM 数据权限 DataScope 模型 | zenith `docs/iam/index.md` | 设计 | planstar 运营后台数据权限 | 行级数据隔离 + IDOR 防护 |
| 12 | 统一 API 契约与分页/文件下载约定 | zenith `docs/backend/api-conventions.md` | 设计 | planstar API 契约文档补强 | 前后端对齐、减少歧义 |
| 13 | 分页统一 `withPagination` 封装 | zenith `docs/backend/database-operations.md` | 设计 | planstar data 层分页 helper | 分页口径统一（注意 planstar 已有怪癖，见 §7） |
| 14 | shared 契约包（类型/校验/常量） | zenith `packages/shared` | 设计 | planstar 前后端共享类型 | 契约漂移早发现 |

---

## 6. planstar_backend 后端参考集成清单

### 6.1 中间件实现

| 参考项 | 参考路径 | 集成建议 | 适配要点 |
|---|---|---|---|
| 认证中间件「取 token → 校验 → 注入 context」单链 | go-wind `pkg/middleware/auth/auth.go` | 提取为 `internal/platform/authn` 统一入口 | 复用现有 JWT/Casbin（ADR-0007），仅补「鉴权结果注入 context + 审计」 |
| 鉴权中间件统一在 HTTP 层拦截 | go-wind authz 引擎 | 在 `internal/server` 的 OpsAuth 之后挂 Casbin | 保持 fail-closed（empty token → 401） |
| 审计日志（PolicyEvaluationLog） | go-wind `docs/tutorial/05-permission-model.md` | 新增鉴权决策审计 | 鉴权日志走可观测栈，避免影响主链路 |

### 6.2 数据库操作封装

| 参考项 | 参考路径 | 集成建议 | 适配要点 |
|---|---|---|---|
| repo 接口 + `NewXxx` 构造器 + 事务 defer | go-wind user 模块 | 强化 `biz` repo 端口 + `data` 实现 | planstar 已有雏形，统一事务 helper |
| 统一分页 helper | zenith `withPagination` | 新增 `data/shared` 分页封装 | **必须保留** planstar 分页怪癖（`page` vs `pageIndex`、`pageSize<=0→20`、`nextPageKey` 是页码非 cursor），只抽公共计算不统一语义 |
| 错误映射单源 | go-wind 错误码 | 复用 `data/shared` 错误映射（ADR-0005） | 驱动错误 → biz 错误集中映射 |

### 6.3 接口规范

| 参考项 | 参考路径 | 集成建议 | 适配要点 |
|---|---|---|---|
| 统一响应格式 + 分页 + 文件下载约定 | zenith `docs/backend/api-conventions.md` | 补强 `docs/api/API_CONTRACT.md` | **不改变** planstar 现有「裸 JSON 成功、`{code,message,traceId}` 错误」契约 |
| 契约元数据装配权限/审计 | zenith `access`/`audit` | 在路由清单断言（governance）中补审计元数据 | 与 `service.go routes()` 路由清单对齐 |

### 6.4 错误处理体系

- 参考 go-wind「统一业务错误码 + 错误映射」与 zenith「Zod 校验 + 统一失败响应」。
- 集成建议：`data/shared` 已收敛错误映射；补充「参数校验失败」统一到同一错误码族，与 planstar 现有 `biz` typed error 对齐。

### 6.5 通用工具类

| 工具 | 来源 | 集成建议 |
|---|---|---|
| CRUD 模块生成约定 | go-wind `go-crud`/`go-utils` | **仅借鉴 11 步约定**写成文档，不引入生成器（planstar 无 proto/Wire） |
| 分层装配 wiring | go-wind `wiring_ent.go` | 复用 `cmd/planstar/wiring.go` 单向装配（已有），补模块注册清晰化 |

### 6.6 权限管理（运营后台数据权限）

- 参考 zenith `docs/iam/index.md` 的 `DataScope`（本人/本部门/本部门及下级/全部/自定义）。
- 集成建议：在 Casbin RBAC 之上增加**数据权限查询条件生成器**（`data` 层拼 `WHERE` 条件），并做 IDOR 防护（校验资源归属）。
- 优先级：P0（运营后台面向多角色时必需）。

---

## 7. 集成注意事项与排除项

### 7.1 与当前技术栈冲突 / 排除

| 项 | 来源 | 排除原因 |
|---|---|---|
| Ent ORM | go-wind | planstar 用手写 `database/sql` + PO/DO，无 Ent；引入会破坏分层与治理门禁 |
| proto / Wire 代码生成 | go-wind | planstar_backend 明确无 proto/Wire（ADR-0001），手写 handler |
| Kratos v2 三服务 BFF 拆分 | go-wind-cms | planstar 是模块化单体（ADR-0001），不拆进程 |
| Asynq / DTM / 多租户 CMS | go-wind-cms | planstar 用 RabbitMQ + 自身状态机，无分布式事务/多租户需求 |
| Node/Hono/Drizzle 后端 | zenith server | planstar 后端是 Go，Node 栈仅借鉴设计不可迁移代码 |
| Semi Design 组件库 | zenith web | planstar 后台是 Ant Design Pro，Semi 组件需逐组件替换为 AntD 等价物 |
| 电子表格/虚拟列表/网格布局组件 | zenith web | 依赖 Semi + 特定场景，当前后台无对应需求 |

### 7.2 已知缺陷 / 需规避

| 项 | 说明 |
|---|---|
| go-wind「五层架构」「Wire/Consul/Kafka」等表述 | 已由 ADR-0001 §1.1 核对与代码不符，仅作思路参考，不得作为事实依据 |
| zenith 分页 `pageOffset` 与 planstar 分页怪癖冲突 | planstar `pageSize<=0→20`（非 clamp 100）、`nextPageKey` 是页码，**不可**直接套 zenith 的分页语义 |
| zenith 后端为 Node 运行时 | 仅借鉴 API/权限/数据权限的「设计」，任何 TS 代码不可直接用于 Go |

### 7.3 集成通用注意事项

1. 所有后端改动须通过 `make check`（gofmt/vet/governance/race）+ `go build ./...`。
2. 涉及对外契约（响应格式、分页、错误码、金额单位）的改动，须显式标注「是否需要 Flutter 客户端协同」，并走 ADR。
3. 前端改动须区分「管理后台（AntD/Umi）」与「Next.js 只读 web」两套技术栈，参考项按目标分别适配。
4. 复用「设计」类项时先写 ADR/文档说明，复用「代码」类项时保持与现有 lint 规则一致。

---

## 8. 优先级与实施路径

### 8.1 优先级分级

| 优先级 | 项 | 目标 |
|---|---|---|
| P0 | 数据权限 DataScope 模型 + IDOR 防护（§6.6） | 运营后台多角色数据隔离 |
| P0 | 鉴权决策审计日志（§6.1） | 补 ADR-0007 的审计闭环 |
| P0 | 前端请求封装 + 401 并发刷新 + 错误统一处理（§6.3 前端） | 后台鉴权体验稳定 |
| P1 | 统一 API 契约文档补强（响应/分页/文件下载） | 前后端对齐 |
| P1 | 列表页/表格/表单弹窗抽象（§3.1） | 减少后台样板代码 |
| P1 | TanStack Query/useRequest 契约调用层统一 | 服务端状态管理统一 |
| P2 | CRUD 模块 11 步约定文档化 | 新业务域开发流程固化 |
| P2 | 分层装配 wiring 清晰化、repo 接口/事务 helper 统一 | 工程一致性 |
| P2 | 页面错误边界 + chunk 失败兜底 | 前端容错 |

### 8.2 实施路径（建议顺序）

1. **审计补强（P0 后端）**：在 Casbin 鉴权后新增决策审计日志，走可观测栈；确认 fail-closed 不变。
2. **数据权限（P0 后端）**：定义 `DataScope` 枚举 → `data` 层查询条件生成器 → IDOR 校验；先做「本人/全部」两档最小闭环。
3. **前端请求层（P0 前端）**：抽取统一 `HttpClient`（token 注入 + 401 单飞刷新 + 错误统一处理），后台各域实例化。
4. **契约文档（P1）**：补强 `docs/api/API_CONTRACT.md` 与 `ADMIN_CONTRACT.md` 的响应/分页/文件下载约定，标注 Flutter 协同点。
5. **后台抽象组件（P1）**：落地列表页/表单弹窗/可配置表格抽象（AntD 版）。
6. **工程固化（P2）**：CRUD 约定 + wiring 清晰化 + 前端错误边界。

### 8.3 交付与归档

- 每一项落地时在 `docs/FOLLOWUP_WORK.md` 登记，关闭后同步 `docs/ADR/README.md`（若产生决策）。
- 本文件作为参考材料，纳入 `docs/research/`；不直接作为 ADR 或实现依据。

---

## 附：关键文件索引

**go-wind**
- `.archive/go-wind/go-wind-cms/backend/AGENTS.md`（分层/三服务/repo 示例）
- `.archive/go-wind/go-wind-cms/backend/pkg/middleware/auth/auth.go`（认证中间件）
- `.archive/go-wind/go-wind-admin/backend/README.md`、`go.mod`（技术栈）
- `.archive/go-wind/go-wind-admin/docs/backend_project_struct.md`、`docs/crud_module_guide/README.md`、`docs/tutorial/05-permission-model.md`
- `.archive/go-wind/go-wind-admin/backend/app/admin/service/cmd/server/wiring_ent.go`（分层装配）

**zenith-admin**
- `.archive/zenith-admin/docs/guide/project-structure.md`
- `.archive/zenith-admin/docs/backend/api-conventions.md`、`docs/backend/database-operations.md`
- `.archive/zenith-admin/docs/frontend/auth-request.md`、`docs/frontend/data-fetching.md`
- `.archive/zenith-admin/docs/iam/index.md`
- `.archive/zenith-admin/packages/web/src/App.tsx`
- `.archive/zenith-admin/packages/web/src/components/list-page/`、`ConfigurableTable.tsx`、`EditFormModal.tsx`、`PageErrorBoundary.tsx`、`EmptyIllustration.tsx`
