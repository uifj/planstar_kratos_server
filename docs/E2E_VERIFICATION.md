# 本地 Docker 端到端接口验证报告（E2E VERIFICATION）

- **日期**：2026-11-19
- **基线**：`main` @ `7f52a22`，本地镜像 `planstar-api:local`（`g0g1`）、重建的 `planstar-admin:latest`、`planstar-admin-web:latest`
- **方法**：以 `docker-compose.local.yml` 三容器（mysql/redis/api）为底座，另起 admin 后端与 admin-web 容器接入同一 `planstar-local_planstar-local` 网络；用 `curl.exe` 逐条请求并统计状态码
- **判定**：**0 个 5xx**，所有异常均为合理 400/401/403/404（参数不足、未登录、数据不存在）

## 1. 环境与启动

| 组件 | 端口 | 状态 |
|---|---|---|
| planstar-api | 8098 / 9108（127.0.0.1） | Up (healthy) |
| mysql / redis | 3306 / 6379 | Up (healthy) |
| admin 后端 | 9100 / 9110（127.0.0.1） | Up |
| admin-web (nginx) | 8080→80（127.0.0.1） | Up (healthy) |

admin 后端用 `ADMIN_DATA_DATABASE_DRIVER=mysql` 等 env 启动，连接本地 `planstar_admin` 库（seed 账号 `admin` / `admin@planstar.top`）。

## 2. 8098 业务接口（57 条）

逐条请求 `service.routes()` 的全部 57 条路由（路径参数替换为 `1`，POST/DELETE 带 `{}`，携带有效 `psSession` Cookie）：

| 状态 | 条数 | 说明 |
|---|---|---|
| 200 | 34 | 正常 |
| 400 | 15 | 空 body / 缺必填参数，符合边界校验 |
| 404 | 8 | 路径参数 `id=1` 对应的数据不存在，符合读面语义 |
| **5xx** | **0** | — |

需登录接口在带有效 Cookie 时正常返回（例如 `GET /v1/user` 200），无 Cookie 时 401。

> 客户端注意：PowerShell 5.1 的 `Invoke-WebRequest -Headers @{Cookie=...}` 不会正确发送 Cookie，
> 导致误判为 401；用 `curl.exe -H "Cookie: psSession=..."` 复核为 200。**这是本地测试客户端问题，不是服务端缺陷。**

## 3. 9108 运维与 RBAC 接口

以 `PLANSTAR_OPS_TOKEN` 作为 break-glass 凭据逐条验证 `/metrics`、探针、`/admin/*` 运营写端点、`/admin/auth/*` 与 `/admin/rbac/*`：

| 状态 | 条数 | 说明 |
|---|---|---|
| 200 | 11 | 探针、metrics、console、举报列表、JWT 登录/登出、RBAC 查询 |
| 400 | 5 | 写端点 body 为 `{}`、缺 status/decision 或目标不存在 |
| 404 | 1 | `GET /admin/projects/1`（数据不存在） |
| **5xx** | **0** | — |
| 无令牌 `/metrics` | 401 | fail-closed 符合预期 |

## 4. Admin 后端（9100）

先以 seed 口令登录，触发**首次登录强制改密**门禁（改密前业务接口均 403，改密后恢复 200）。改密后逐条验证 proto 定义的 28 条路由：

| 状态 | 条数 | 说明 |
|---|---|---|
| 200 | 26 | 管理端 CRUD、用户/项目/订单/举报/支付/退款/动态等 |
| 400 | 0 | — |
| 404 | 1 | `GET /v1/planstar/orders/1`（订单不存在） |
| **5xx** | **0** | — |

**验证环境注意事项**：最初有 14 条路由返回 404，原因是本地 `planstar-admin:latest` 镜像构建于 2026-11-07，早于这些路由的实现；用当前源码重建镜像后全部恢复。源码中 `GetDashboard`/`ListIdeas`/`ListGoods` 等 12 个方法均已实现且在 `internal/server/http.go` 注册。

## 5. Admin 控制台前端（admin-web / 8080）

| 检查 | 结果 |
|---|---|
| `/` HTML | 200（Kratos Admin Umi SPA，标题 Kratos Admin） |
| 构建产物 CSS/JS（5 个资源） | 全部 200 |
| `/healthz` | 200 |
| SPA fallback（`/admin`、`/login`、`/users`） | 全部 200 |
| 安全响应头 | nginx 配置含 CSP / X-Frame-Options DENY / X-Content-Type-Options / Referrer-Policy（与审计报告中 planstar-api 缺失响应头不同，此处已具备） |

## 6. planstar_web 前端

**运行结果：未能启动，属环境限制而非项目缺陷。**

- `apps/web/node_modules/next` 存在但入口不完整；`pnpm install --frozen-lockfile` 在本环境触发 safe-delete 批量删除保护（10021 文件 > 阈值 500），被工具链阻止，无法完成依赖重装。
- `.next/` 仅有 `static/`，缺 `BUILD_ID` 与 `server/app/page.html`，无法 `next start`。
- **源码级核验**：`packages/api-client/src/client.ts` 共 10 个后端调用路径（`v1/home/banners`、`v1/categories`、`v1/projects/hot`、`v1/projects`、`v1/projects/{id}`、`.../reward-tiers`、`.../updates`、`.../faqs`、`.../reviews`、`v1/rankings`），全部包含在第 2 节已实测为 200 的 57 条路由内；默认 `PLANSTAR_API_BASE=http://127.0.0.1:8098/`，与本地服务一致。Cookie 转发逻辑为从 Next 的 `cookies()` 取 `psSession` 注入请求头，与后端契约一致。
- 待具备完整依赖的环境后，应补跑 `pnpm build` 与 `pnpm test:e2e`，本报告不宣称 web 端已通过运行验证。

## 7. 结论

**按本次可执行的验证范围，所有已启动的后端与 admin 前端接口均无 5xx，未发现功能性故障。** 需要如实说明的 3 点：

1. 400/401/403/404 均为合理业务响应，不是服务端错误。
2. admin 首次登录强制改密是设计行为；后续联调需使用改密后的口令。
3. planstar_web 仅完成源码级对接核验，运行验证受本地依赖安装保护机制阻断，未宣称通过。
