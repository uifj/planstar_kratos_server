# ADR-0007: 运营域 JWT 鉴权与 Casbin RBAC

- **状态**: ACCEPTED（已实施）
- **分类**: SEC / ARC
- **优先级**: P1
- **影响维度**: 生产功能 / 安全合规 / 认知障碍
- **日期**: 2026-11-19
- **来源**: 用户要求「参考 go-wind 实现企业级 JWT + Casbin 用户鉴权与 RBAC 权限体系」；`docs/ADR/ADR-0001` 契约冻结约束；`docs/api/ADMIN_CONTRACT.md:219` 预留设计
- **修复 commit**: 本批次

## 背景（Context）

1. 现状 C 端（8098）用 `Cookie: psSession=<64 位 hex>` 不透明令牌，会话落 MySQL + Redis；契约（`API_CONTRACT.md:69-78`）明确不是 Bearer，且 `internal/governance` 有断言守护「响应头不下发 `psSession`」。9108 运维端口用单一 `PLANSTAR_OPS_TOKEN` Bearer + CIDR + fail-closed + 审计（`OpsAuth.Wrap`）。
2. `docs/api/ADMIN_CONTRACT.md:219` 预先写明：「后续接入运营账号体系时，在 `OpsAuth` **之后**增加 RBAC 中间件，不改动业务契约」。
3. 对 `.archive/go-wind/go-wind-admin` 一手代码核对（见 `docs/notes/` 探索记录）：JWT 用 `golang-jwt/jwt/v5 v5.3.1`；Casbin 经 `kratos-authz/engine/casbin` 间接依赖，**模型文件不在仓内不可审计**；`configs/auth.yaml:68` 默认 `authz.type: "noop"`（出厂全放行），且拼错引擎名会 `default → noop` **静默降级、无日志**；文档 `docs/tutorial/05-permission-model.md:35-43` 自己警告生产必须切换。仓内唯一可审计模型是 OPA Rego。
4. 硬约束（`ADR-0001`）：模块化单体、不改 57 条路由集合、裸 JSON 成功响应、错误码单源、C 端 Cookie 契约冻结、`biz` 纯 Go 无 infra 传递依赖。

## 决策（Decision）

### D1 双链路：C 端不动，运营域引入 JWT + RBAC

- **C 端 8098**：`psSession` 生成/校验/DB+Redis 会话**一行不改**。JWT 不作为 Cookie value（会破坏 64-hex 契约、`data.UserIDByToken` 等值查询、`AuthCache` Redis key 与 `session_test.go` 断言）。
- **9108 运维域**：新增 JWT 双令牌（access + refresh）与 Casbin RBAC。`OpsAuth` 的机器令牌保留，映射为**策略文件里显式声明的 `superadmin` 角色**（不是代码里的 if-bypass），因此现有运维/烟雾脚本零中断且决策可审计、可撤销。
- 认证与授权分两阶段：先解析 Bearer（ops token 或 JWT），再 `Enforce(method, path)`；401 与 403 分离。

### D2 JWT 规格（对标 go-wind，裁剪不必要复杂度）

- HS256，`jwt.WithValidMethods(["HS256"])` 白名单（**禁止 `none`**），密钥只从 `PLANSTAR_AUTHN_JWT_SECRET` 注入、不落 yaml，生产断言长度 ≥ 32 字节且非占位值。
- Claims 最小化：`sub`/`jti`/`iss`/`aud`/`iat`/`nbf`/`exp` + `typ`(access|refresh) + `roles` + `uid`。**不**搬 go-wind 的 `dss/dsu/ipa/ita` 权限快照进令牌（权限变更要等 TTL 才生效）。
- access 15 min、refresh 7 d，可经 env 调整；`WithLeeway` 默认 30s；`refresh` 令牌只能用于 `/admin/auth/refresh`。
- `jti` 撤销：Redis `planstar:authn:revoked:{jti}`（生产）/ 进程内 TTL 表（单实例降级）。Redis 校验失败 **fail-closed 返回 401**，不放行。
- 严格拒绝：`alg` 不符、`iss`/`aud` 不符、`exp` 缺失、过期、`nbf` 未到、`typ` 不符、已撤销。

### D3 Casbin 模型与策略（修正 go-wind 的「模型黑洞」）

- 模型以**仓内字符串常量**提交（`internal/platform/authz/model.go`），含 `[request_definition]` / `[policy_definition]` / `[role_definition] g=_,_` / `[policy_effect]` / `[matchers]`，可审计、可单测；不依赖外部引擎库隐藏模型。
- 判据 `method + path`，`keyMatch2` 支持 `/admin/reports/:id`；matcher 允许 `r.sub == p.sub || g(r.sub,p.sub)`，既支持角色直判也支持用户→角色映射。
- **默认拒绝**：Casbin 无命中 allow 即 deny；内置策略含 `p, superadmin, *, *`（显式、可删）。绝不提供 `noop` 引擎，配置非法**启动失败**而非静默全放行。
- 策略分两层：编译期内置基线 + Redis 增量（`planstar:authz:policies`）。运行时 `AddPolicy`/`RemovePolicy` 持久化并 `Enforcer` 热更新，兼容 `read_only: true` 容器（不写文件）。

### D4 端点与兼容

- 新增 9108 端点：`POST /admin/auth/{login,refresh,logout}`、`GET /admin/rbac/me`、`GET|POST|DELETE /admin/rbac/policies`、`GET|POST /admin/rbac/roles`。
- `login` / `refresh` 不经 `OpsAuth`（否则没令牌无法登录），但保留 CIDR 校验、密码 bcrypt 校验、失败审计与限流；`logout` 及其余端点仍在 `OpsAuth.Wrap` 之后。
- **不改 `service.routes()`**：9108 端点不在 `routes_test.go` 的 57/56 常量口径内，C 端契约与路由清单断言零改动。
- 不新增错误码：401 复用 `UNAUTHORIZED`、403 复用 `FORBIDDEN`，细分原因放 `message`，避免 `errcode.All()` 与契约码表双向断言扩散。

### D5 依赖方向

JWT / Casbin / bcrypt / Redis 撤销与策略存储全部放 `internal/platform/{authn,authz}`；`biz` 不引用任何新包，保持 `TestBizHasNoTransitiveInfraDependency` 0 命中。`server` 只依赖两个最小接口（`TokenVerifier` / `Authorizer`），由 `cmd/planstar/wiring.go` 注入具体实现。

## 后果（Consequences）

- **正面**：运营域具备可审计的账号→角色→权限体系、即时吊销、401/403 分离、策略热更新；C 端零改动；go-wind 的 fail-open 缺陷被明确规避。
- **负面/风险**：新增 3 个第三方依赖（`golang-jwt/jwt/v5`、`casbin/casbin/v2`、`golang.org/x/crypto`）；内存撤销/策略降级在重启后丢失（access TTL 短，风险可控，多实例必须启用 Redis）；静态用户目录需通过配置/env 维护，尚未接 LDAP/OIDC。
- **中性**：C 端用户仍无 RBAC。未来若需要，可复用 `authz.Enforcer` 与 `authn.Identity` context，在 `requireUser` 之后挂同一中间件（影子模式 → 强制），见 `docs/FOLLOWUP_WORK.md`。

## 验证（Verification）

- `go build ./...` / `go vet ./...` / `go test ./... -count=1` / `golangci-lint run ./...` 全绿。
- `internal/platform/authn` 单测：签发/验签、`alg=none` 拒绝、过期、`nbf`、iss/aud、`typ` 混淆、jti 撤销、refresh 只能刷新、bcrypt 错误口令。
- `internal/platform/authz` 单测：默认拒绝、superadmin 显式放行、`keyMatch2` 路径、角色继承 `g`、策略增删与重建、并发 Enforce。
- `internal/server/ops_auth_test.go` 既有 8 组用例保持通过；新增 RBAC 集成用例覆盖 401/403/放行与审计字段。
- 本地 Docker local deploy 复测：原有 `/admin/*` 用 ops token 仍可用，JWT 登录 → 受限角色越权 403 → 提权后 200。

## 关联（Links）

- 代码: `planstar_backend/internal/platform/{authn,authz}/`、`internal/server/{ops_auth.go,admin.go,admin_auth.go}`、`cmd/planstar/wiring.go`
- 配置: `internal/platform/conf/conf.go`、`configs/*.yaml`
- 契约: `docs/api/ADMIN_CONTRACT.md`（9108）；`docs/api/API_CONTRACT.md` **不变**
- 关联 ADR: `ADR-0001`（契约冻结/模块化单体）、`ADR-0003`（生产 fail-fast 断言）、`ADR-0006`（platform 归位）
