# 运营后台整合、上线与回滚

本文覆盖 `planstar/admin` 与 `planstar/` 业务服务的部署整合：风险登记、上线检查单、回滚步骤，以及短信/邮件/OSS 外部服务的启用方式。架构背景见 [architecture.md](architecture.md)，接口契约见 [api/ADMIN_CONTRACT.md](api/ADMIN_CONTRACT.md)。

## 1. 风险登记表

与整合 spec 的 17 项风险一致；「状态」标注当前实现与仍需人工确认的部分。

| # | 风险 | 影响 | 触发条件 | 解决方案 | 当前状态 |
| --- | --- | --- | --- | --- | --- |
| 1 | 旧 admin 与新模板依赖/端口/表冲突 | 编译失败、误连旧库 | 替换旧 admin 代码或依赖时 | 全量覆盖并删除旧 `go.sum` 依赖；全局检索 `9099`/`shop_user`/`AuthorityId`；CI 编译校验 | 已由 CI `admin-go` 覆盖；发布前仍需检索旧端口/表名 |
| 2 | Go `internal` 跨模块不可复用 | 业务逻辑重复或无法复用 | 需要复用 planstar 内部业务逻辑时 | 管理端自建 `biz/data`，只读企划星库；审核/举报动作走 ops API | 已实现；禁止在 admin 内复制业务规则 |
| 3 | `psSession` 与 `psAdminToken` 混淆 | 用户会话被覆盖或越权 | 同一浏览器同时访问业务端与后台时 | 不同 Cookie 名与 Path；管理端仅绑定 `admin.planstar.top`；互不读取 | `psAdminToken` 默认 host-only；需确认 `ADMIN_AUTH_COOKIE_DOMAIN` 留空 |
| 4 | Cookie 鉴权的 CSRF/XSS | 后台被跨站操作或脚本窃取 | 跨站请求、恶意脚本注入或 CSP 配置缺失时 | `HttpOnly` + `SameSite=Strict` + 生产 `Secure`；Origin 白名单；CSP；Ant Design Pro 默认转义；禁止内联脚本 | nginx 已加 CSP/安全头；发布前检查控制台无内联脚本 |
| 5 | 默认口令与暴力破解 | 后台被接管 | 使用初始口令或短时间内大量失败登录时 | 首次登录强制改密；bcrypt cost≥12；登录限流、失败锁定、审计告警；上线后禁用/改密种子账号 | 限流/审计/告警已配；`19_admin_console.sql` 的种子账号 `must_change_password=1`，上线必须验证并立即改密 |
| 6 | 角色越权 | 运营越权改数据 | 角色配置错误或直接调用写接口时 | 服务端逐方法校验 `viewer/operator/admin` 矩阵；仅 `admin` 可管理管理员账号；单测覆盖 | 已实现并有单测；新增端点必须同步矩阵 |
| 7 | 直接读库造成 schema 耦合 | 企划星改表导致后台 500 | 业务库改表、字段下线或迁移未同步时 | 查询字段白名单、优先视图；禁止跨库 JOIN；迁移评审与契约测试 | 已按只读边界实现；改表需同步评审 admin 查询 |
| 8 | ent `auto_migrate` 生产误用 | 隐式改表/锁表 | 生产启动或发布迁移时 | 生产 `auto_migrate=false`；`planstar/sql/19_admin_console.sql` 评审执行；启动时校验表存在 | compose 未开启 auto_migrate；首次部署由 MySQL 初始化挂载执行 19 |
| 9 | 数据库权限过大 | 误删/越权写业务库 | 复用高权限账号或 DSN 配置错误时 | 专用账号：业务库仅 `SELECT` + `planstar_user.user_info` 的 `UPDATE`；`planstar_admin` 全权限；连接池上限 | 本地默认 root 仅为开发；生产必须在 `.env` 设置最小权限 DSN |
| 10 | 审核规则双写漂移 | 同一企划两套结论 | 审核规则在 admin 与 planstar 各实现一次时 | 审核/举报只调 ops API；超时/失败返回 503，不做本地规则兜底 | 已实现 503 语义；禁止在 admin 内加黑名单 |
| 11 | 指标/链路标签冲突 | 监控误判 | 指标/服务名与 planstar 同名混用时 | 独立 service 名 `admin.api`、独立 job；9110 独立 `/metrics`；trace service name 区分 | 已增加 Prometheus job/告警/Grafana 面板 |
| 12 | 管理端口公网暴露 | 攻击面扩大 | 将 `9100/9101/9110` 直接映射到公网时 | compose 不发布 `9100/9101`，`9110` 仅回环；Traefik 强制 HTTPS + 可选 IP 白名单 | 已实现；IP 白名单中间件以注释形式预留 |
| 13 | 控制台构建漂移 | 页面与 API 不匹配 | 前端与 API 契约变更但未同步生成客户端时 | `make api` 统一生成 TS 客户端；CI 执行 `npm run build`；SPA fallback 与 `index.html no-store` | CI 已加 `admin-console`；nginx 已配 no-store |
| 14 | 外部服务空实现被误判为已接入 | 短信/邮件/OSS 静默失败 | `enabled=true` 但供应商参数/密钥缺失时 | `enabled=false` 时启动日志明确标记 Noop；接口幂等；配置校验；密钥仅走环境变量 | 默认关闭；启用步骤见第 4 节 |
| 15 | 部署顺序与回滚 | 迁移失败导致不可用 | 迁移失败或新版本启动异常需回滚时 | 先迁移后启动；compose `depends_on` 健康检查；镜像固定 tag；回滚上一 tag；迁移保持向后兼容 | `admin` 依赖 mysql/redis healthy；回滚见第 3 节 |
| 16 | Kratos v3 与 contrib 版本漂移 | 编译/运行时不兼容 | 升级 Kratos/contrib 或 Go 版本时 | 统一 v3 版本；`go mod tidy`；CI 锁定 Go 版本 | CI 固定 Go 1.26 + `GOPROXY` |
| 17 | 时区不一致（UTC vs +08:00） | 审核时间/报表偏差 | 容器、DSN 或数据库时区配置不一致时 | DSN `parseTime=true&loc=Asia%2FShanghai`；库表统一 `DATETIME`；展示层统一时区 | 容器 `TZ=Asia/Shanghai`；DSN 默认含 `loc=Asia%2FShanghai` |

### 1.1 已知实现差异（发布前必须确认）

| 项目 | 现象 | 影响 | 建议 |
| --- | --- | --- | --- |
| admin 镜像 CMD | `admin/Dockerfile` 的 `make build` 产出 `/app/server`，但镜像 CMD 为 `./kratos-admin` | 直接 `docker run` / 未覆盖 command 时容器启动失败 | compose 已用 `command: ["/app/server", "-conf", "/data/conf"]` 覆盖；其他部署方式需同样覆盖，或由镜像维护方修正 CMD |
| Redis 密码 | `admin` 配置仅有 `data.redis.addr`，没有密码字段；现有 `redis` 服务启用 `requirepass` | admin 的 `/readyz` 会因 `NOAUTH` 失败，JWT 撤销/登录限流不可用 | 当前 compose 仍指向 `redis:6379`；上线前需为 admin 增加 Redis 密码配置（改源码）或提供独立无密码内网 Redis（不推荐）；在此之前勿将 `/readyz` 作为唯一健康判据 |
| 健康检查依赖 | 运行层 `debian:stable-slim` 未内置 `curl`/`wget` | 仅用 curl/wget 的 healthcheck 会一直 unhealthy | compose 依次尝试 wget/curl，最后用 bash `/dev/tcp` 仅检查 9110 端口连通；若镜像改为 alpine 可直接用 wget 校验 `/healthz` |
| `/readyz` 指标 | admin 已导出 `admin_readyz_ready`（1=ready，0=not ready） | readyz 持续失败可直接由 Prometheus 告警 | 已由 `AdminReadinessFailing` 覆盖（`admin_readyz_ready == 0` 持续 2m）；容器 healthcheck 也改为请求 `/readyz` |
| CORS 白名单 | `pkg/auth.CORS` 会拒绝任何携带 `Origin` 但不在白名单的请求 | 浏览器同源 POST（登录、写操作）会被 403，控制台不可用 | CORS 白名单直接写在 `admin/configs/config.docker.yaml`，已设置 `https://admin.planstar.top`；域名变更时同步修改 |
| service.name | `admin/Dockerfile` 只注入 `main.Version`，未注入 `main.Name` | 日志/链路中 `service.name` 可能为空，Loki 查询不到 `admin.api` | Prometheus 已通过 job label 标记 `service=admin.api`；日志检索暂用 `job="admin"` 或容器名，建议镜像侧补 `-X main.Name=admin.api` |

## 2. 上线检查单

### 2.1 网络、域名与 TLS

- [ ] `admin.planstar.top` A/AAAA 记录指向 Traefik 入口，且与 `planstar.top` 使用同一 ACME 账户/resolver。
- [ ] 80/443 可从公网访问；`9100`、`9101` 未出现在宿主机监听端口（`ss -lntp` / `docker compose ps`）。
- [ ] `9110` 仅监听 `127.0.0.1`；如使用 Prometheus 容器抓取，确认 `admin:9110` 在 `planstar` 网络内可达。
- [ ] Traefik 动态配置已加载：`admin-api`（`/v1`，priority 100）、`admin-console`（其余，priority 10）、`admin-http`（301）。
- [ ] 浏览器访问 `http://admin.planstar.top` 自动跳转 HTTPS，证书链有效。
- [ ] 可选：如启用 `admin-ip-allowlist`，确认办公出口 CIDR 已加入且不会误伤 Prometheus/人工排障。

### 2.2 配置与密钥

- [ ] `planstar/.env` 由 `.env.example` 复制，未提交真实密钥。
- [ ] `ADMIN_AUTH_JWT_SECRET` 长度 ≥ 32 字节且与业务 JWT 密钥不同；轮换前评估所有管理员重新登录。
- [ ] `ADMIN_AUTH_COOKIE_NAME` 保持 `psAdminToken`，不得复用 `psSession`；`ADMIN_AUTH_COOKIE_DOMAIN` 留空（host-only）或严格限定 `.planstar.top`。
- [ ] `ADMIN_PLANSTAR_OPS_TOKEN` 与 `PLANSTAR_OPS_TOKEN` 值一致；`planstar/deploy/prometheus/ops_token` 同步更新。
- [ ] `ADMIN_DATA_DATABASE_SOURCE` 指向 `planstar_admin`，账号仅具备约定权限；密码含 `@`/`:`/`/` 时使用正确转义或专用账号。
- [ ] `ADMIN_DATA_REDIS_ADDR` 与实际 Redis 鉴权方式匹配（见 1.1 已知差异）。
- [ ] `ADMIN_AUTH_COOKIE_SECURE=true`、`ADMIN_SERVER_ENV=prod`。
- [ ] CORS 白名单直接写在 `admin/configs/config.docker.yaml`，与实际域名一致（否则浏览器的同源 POST 会被 CORS filter 403）；域名变更时同步修改。
- [ ] 外部服务开关默认 `false`；未采购前不要打开。

### 2.3 数据库与迁移

- [ ] 新环境首次启动时确认 `19_admin_console.sql` 在 MySQL 初始化日志中执行成功；已有环境手工执行并校验 `admins`、`admin_audit_log`。
- [ ] `auto_migrate` 保持关闭；表结构变更走评审 SQL。
- [ ] 业务库账号只读测试：`SELECT` 通过，`INSERT/UPDATE/DELETE` 被拒绝（`user_info` 更新除外）。
- [ ] 种子管理员已创建且 `must_change_password=1`；首次登录已改密，初始口令不再可用。

### 2.4 构建与启动

- [ ] `docker compose -f planstar/docker-compose.yml config -q` 通过。
- [ ] `docker compose build admin admin-web` 成功；`admin` 镜像内 `/app/server` 存在。
- [ ] `docker compose up -d admin admin-web traefik` 后，`admin` 与 `admin-web` 均为 healthy。
- [ ] `curl -fsS http://127.0.0.1:9110/healthz` 返回 200；`curl -fsS http://127.0.0.1:9110/readyz` 返回 200（Redis 鉴权问题需先解决）。
- [ ] `curl -I https://admin.planstar.top/` 返回 200 且包含安全响应头；`/v1/admins/current` 未登录返回 401。

本地联调（不经 Traefik）时额外确认：

- [ ] 后端：`cd admin && go build -o bin/server.exe ./cmd/server && ./bin/server.exe -conf configs`，`http://127.0.0.1:9100/v1/admins/current` 返回 401。
- [ ] 前端：`cd admin/console && API_TARGET=http://127.0.0.1:9100 npm run dev`，登录后出现「Change initial password」弹窗。
- [ ] `ADMIN_*` 环境变量由 `admin/internal/conf/env.go` 做类型转换：bool/int 直接写裸值（`true`/`0`），`ADMIN_SERVER_HTTP_CORS_ALLOW_ORIGINS` 支持逗号分隔或 JSON 数组，其余按字符串处理。
- [ ] 本机 9110 被容器占用时，用 `ADMIN_SERVER_ADMIN_ADDR=127.0.0.1:19110` 错开运维端口。

### 2.5 鉴权、权限与数据边界

- [ ] 登录成功后 Cookie 为 `psAdminToken`，具备 `HttpOnly`/`Secure`/`SameSite=Strict`，未与 `psSession` 混用。
- [ ] 三种角色分别验证：viewer 写操作 403；operator 可审核/处置；admin 可管理账号。
- [ ] 越权请求返回 403 且无写库；`Origin` 不在白名单的写请求返回 403。
- [ ] 连续失败登录触发 429；审计写入 `admin_audit_log`，Loki 可检索 `service.name=admin.api`。
- [ ] 审核/举报在 ops API 不可用时返回 503，未发生本地兜底写入。

### 2.6 可观测与告警

- [ ] Prometheus `Status → Targets` 中 `job=admin`、`service=admin.api` 为 UP。
- [ ] Grafana 出现 `Planstar Admin Ops` 仪表盘，存活/请求速率/P95/登录失败/5xx 面板有数据。
- [ ] 规则组 `planstar-admin` 已加载：`AdminServiceDown`、`AdminReadinessFailing`、`AdminLoginFailureSpike`、`AdminHighErrorRate5xx`、`AdminCriticalErrorRate5xx`。
- [ ] 告警路由可送达值班渠道；触发测试后恢复。
- [ ] 指标名核对：`requests_total{method,path,code}`、`duration_seconds_bucket`、`admin_readyz_ready`（历史名 `http_server_requests_*` 仅用于旧版本面板）。

### 2.7 CI 与发布物

- [ ] CI `admin-go`（build/vet/test）与 `admin-console`（npm ci/build）通过。
- [ ] `compose` 校验作业通过。
- [ ] 镜像使用不可变 tag（commit SHA 或版本号），记录当前与上一版本 tag。
- [ ] 发布记录包含配置 diff（尤其 `.env` 新增变量）。

## 3. 回滚步骤

1. **停止扩散**：将 `admin-api`/`admin-console` 路由指向维护页，或在 Traefik 启用 `admin-ip-allowlist` 仅保留运维出口。
2. **回滚镜像**：在 `planstar/.env` 将 `ADMIN_IMAGE/ADMIN_TAG`、`ADMIN_WEB_IMAGE/ADMIN_WEB_TAG` 改回上一版本，执行 `docker compose up -d --no-deps admin admin-web`。
3. **回滚配置**：将 `.env` 恢复为上一版本；若新增变量导致启动失败，先删除新增 `ADMIN_*` 再启动。`docker compose config` 确认解析结果。
4. **数据库**：迁移遵循向后兼容；优先回滚应用而不是删表。若必须回退 `19_admin_console.sql`，先备份 `planstar_admin`，再在维护窗口执行评审过的回退 SQL（不自动删库）。
5. **验证**：`curl -fsS http://127.0.0.1:9110/healthz`、`/readyz`；管理员登录、列表、审核/举报只读验证；Grafana 5xx/延迟恢复；Prometheus `job=admin` UP。
6. **复盘**：记录失败版本 tag、触发时间、回滚耗时、数据影响与后续修复项；保留 `admin_audit_log` 与日志证据。

## 4. 外部服务（短信 / 邮件 / OSS）启用

三类外部服务默认关闭（`enabled=false`），使用 Noop/Log 实现，不会发起真实网络调用。契约在 `admin` 与 `planstar` 两侧同构，可独立启用。

### 4.1 配置项

| 服务 | 环境变量 | 说明 |
| --- | --- | --- |
| SMS | `ADMIN_NOTIFY_SMS_ENABLED` | `true` 才启用真实供应商 |
| SMS | `ADMIN_NOTIFY_SMS_PROVIDER` | 供应商标识，如 `aliyun`/`tencent` |
| SMS | `ADMIN_NOTIFY_SMS_ENDPOINT` | 供应商 API 地址（内网出口/代理） |
| SMS | `ADMIN_NOTIFY_SMS_ACCESS_KEY` / `ADMIN_NOTIFY_SMS_ACCESS_SECRET` | 仅环境变量注入，禁止入库/入日志 |
| SMS | `ADMIN_NOTIFY_SMS_SIGN_NAME` / `ADMIN_NOTIFY_SMS_TEMPLATE_CODE` | 签名与模板编号 |
| Email | `ADMIN_NOTIFY_EMAIL_ENABLED` | `true` 才启用 SMTP |
| Email | `ADMIN_NOTIFY_EMAIL_PROVIDER` | 如 `smtp` |
| Email | `ADMIN_NOTIFY_EMAIL_SMTP_HOST` / `ADMIN_NOTIFY_EMAIL_SMTP_PORT` | SMTP 地址与端口 |
| Email | `ADMIN_NOTIFY_EMAIL_USERNAME` / `ADMIN_NOTIFY_EMAIL_PASSWORD` | 认证信息 |
| Email | `ADMIN_NOTIFY_EMAIL_FROM` | 发件人地址 |
| OSS | `ADMIN_STORAGE_OSS_ENABLED` | `true` 才启用对象存储 |
| OSS | `ADMIN_STORAGE_OSS_PROVIDER` | 如 `aliyun`/`s3` |
| OSS | `ADMIN_STORAGE_OSS_ENDPOINT` / `ADMIN_STORAGE_OSS_REGION` / `ADMIN_STORAGE_OSS_BUCKET` | 端点、区域、桶 |
| OSS | `ADMIN_STORAGE_OSS_ACCESS_KEY` / `ADMIN_STORAGE_OSS_ACCESS_SECRET` | 仅环境变量注入 |
| OSS | `ADMIN_STORAGE_OSS_CDN_HOST` | 可选 CDN 域名 |
| OSS | `ADMIN_STORAGE_OSS_PRESIGN_TTL_SECONDS` | 预签名有效期，`0` 表示默认 |

### 4.2 启用步骤

1. 在供应商控制台创建最小权限子账号/密钥，限制来源 IP、短信签名与模板、OSS 桶前缀。
2. 将密钥写入部署机 `planstar/.env`（不要提交仓库），同步更新密钥管理系统与轮换计划。
3. 先以 `enabled=false` → `true` 单项启用，执行 `docker compose up -d --no-deps admin`（及需要启用的 `planstar-api`）。
4. 检查启动日志：Noop 标记消失，出现供应商初始化成功/失败信息；确认日志中无密钥明文。
5. 用小流量测试账号验证：短信模板、邮件到达率/退信、OSS 上传/预签名下载；确认幂等（重复请求不重复扣费）。
6. 观察 `requests_total` 5xx、延迟与外部服务日志；异常时立即将对应 `enabled` 改回 `false` 并重启。
7. 记录启用时间、供应商合同/配额、密钥到期日与回滚联系人。

### 4.3 约束

- 外部调用必须设置超时与重试上限；失败不得阻塞核心审核/举报事务。
- 密钥只通过 `ADMIN_*` / `PLANSTAR_*` 环境变量注入，不写入 YAML、数据库、审计日志或前端。
- 供应商变更时保持接口签名不变，仅替换实现；先灰度一个渠道再全量。
- `enabled=true` 但关键参数为空时应在启动校验阶段失败，避免静默降级。
