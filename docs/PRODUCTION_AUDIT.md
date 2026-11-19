# 商用生产级代码审计报告（PRODUCTION AUDIT）

- **审计日期**：2026-11-19
- **审计基线**：HEAD `2665b5a` + 工作区 21 项未提交改动（JWT + Casbin RBAC 实施）
- **审计方式**：三条独立只读审计线（安全 / 数据与业务 / 部署与运维）+ 主审复核关键证据
- **审计标准**：商用生产上线标准（fail-closed、最小权限、资金闭环、可回滚、可观测）
- **处置约定**：**本次审计发现的问题一律不修复**，仅登记；用户要求「没问题后再提交推送」，因此**本轮未提交、未推送**

## 0. 结论摘要

**G0/G1 已于 2026-11-19 修复并验证通过；仍有 8 项既有阻断未修复**（资金闭环 6 项、凭据/文档 2 项）。原报告登记 16 项阻断、17 项高/中风险，其中 8 项已标记 FIXED。

阻断项按来源分为三类，必须区分责任与修复顺序：

| 来源 | 数量 | 说明 |
|---|---|---|
| **本轮 JWT/RBAC 实施引入** | 3 | RBAC 自我提权 ×2、refresh 轮换非原子 |
| **目录改名遗留（本批次漏改）** | 4 | `release.yml` 构建上下文、`deploy.sh` / `release.yml` / `backup.sh` 的 `COMPOSE_DIR` |
| **既有功能/配置缺口** | 9 | 支付回调缺失、退款未真实退款、明文 Cookie、AK/SK 泄露等 |

> 主审必须明确：前两类共 7 项与本批次改动直接相关，其中 RBAC 提权属于**新引入的安全漏洞**，
> 目录改名遗留属于**本批次可避免的漏改**。其余为仓库既有的功能缺口，已有部分登记在
> `docs/FOLLOWUP_WORK.md`，本报告按生产标准重新分级。

---

## 1. 阻断上线问题（16 项）

### 1.1 安全线（4 项）

| ID | 等级 | 问题与证据 | 攻击/故障场景 | 建议（本轮不实施） |
|---|---|---|---|---|
| **S1** | ~~阻断~~ **FIXED**·**本轮引入** | `internal/server/admin_auth.go:214-225` 的 `addPolicy` 调用 `AddRule`，而 `internal/platform/authz/enforcer.go:207-215` 的 `validateRule` **只校验非空与 obj 格式，不限制 `obj=="*"` / `act=="*"`**；matcher (`authz/model.go:20`) 允许 `r.sub == p.sub` 字面命中 | 持 `rbac_admin` 的账号调用 `POST /admin/rbac/policies`，body `{"sub":"<自己用户名>","obj":"*","act":"*"}`，直接获得与 superadmin 相同的全放行规则，且规则持久化到 Redis 增量区 | 禁止运行时添加 `obj=="*"` 或 `act=="*"`；sub 限定为预定义角色码且不得为调用方自身；敏感变更要求 superadmin 二次审批 |
| **S2** | ~~阻断~~ **FIXED**·**本轮引入** | `internal/server/admin_auth.go:252-270` 的 `grantRole` 与 `authz/enforcer.go:161-179` 的 `AddGrouping` **均无角色白名单**，仅校验非空 | `POST /admin/rbac/users/{自己}/roles {"role":"superadmin"}` 生成 `g(self,superadmin)`，命中基线 `{superadmin,*,*}`，直接全端口写权限 | role 强制白名单（superadmin 不可通过 API 授予）；禁止对自己授权；引入双人审批 |
| **S3** | ~~阻断~~ **FIXED**·**本轮引入** | `admin_auth.go:123-149` refresh 轮换分三步：`Verify(refresh)` → `Revoke(oldJTI)` → 签发新令牌，非原子 | 攻击者截获 refresh 后并发 2 次请求，两者均在 `Revoke` 前通过校验，换出 2 套有效令牌，轮换失效 | Redis `GETDEL` / Lua 原子消费旧 jti，成功者才签发 |
| **S4** | ~~阻断~~ **FIXED**·既有 | `planstar_backend/configs/config.lite.yaml:51` `cookie_secure: false` + `:56` `allow_insecure_cookie: true`；`docker-compose.prod.yml:77/101` 把该档位作为生产配置挂载；`docker-compose.server.yml:29` `"8098:8098"` 公网直连 | 明文 HTTP 传输 `psSession`（30 天有效、仅 HttpOnly+Lax 无 Secure），公网抓包即可接管账号 | 备案完成则改回 `cookie_secure: true`、删除豁免与 8098 公网映射；若必须使用 HTTP，限制入口到内网/VPN 并大幅缩短会话 TTL |

### 1.2 数据与业务线（6 项，均为既有缺口）

| ID | 等级 | 问题与证据 | 场景 | 建议 |
|---|---|---|---|---|
| **D1** | 阻断 | `internal/data/payment.go:25-115` 只创建 `pending` 支付单；`internal/service/service.go:116-117` 无 webhook/callback 路由；`pay_order.status` 永不置 success | 用户实际付款后订单永远待支付、库存不确认 | 接入渠道异步通知 + 验签 + 幂等状态机，事务内同步更新支付单与订单 |
| **D2** | 阻断 | `internal/data/ops.go:117-129` 退款审核通过仅 `UPDATE order_refund SET status='success'` 与订单置 7，无渠道退款、无钱包回退、无流水 | 审核通过即标记「已退款」，用户未收到钱 | 渠道退款 API + 退款流水 + `refunding/success/failed` 状态机与失败补偿 |
| **D3** | 阻断 | `internal/data/project_write.go:255-264` 支持单写 `pledge_status='pending'`，全仓无推进到 `paid/delivered` 的代码；而 `project.go:224-237`、`project_write.go:88` 只认 `paid/confirmed/shipped/delivered` | 用户支持后永远不是支持者、永远无法评价 | 补齐支付 → 确认 → 发货状态机 |
| **D4** | 阻断 | `internal/data/project_helpers.go:349-357` 用 `MAX(id)+1` 生成主键；调用点 `project_write.go:114/246/347`、`auth.go:121/155`，无冲突重试 | 并发取到同一 ID，一个成功一个 1062→500；事务内 `MAX()` 还产生 gap lock | 改 AUTO_INCREMENT / 雪花 ID，冲突走幂等返回 |
| **D5** | 阻断 | `internal/data/project_write.go:96-103` 重复评价为「先 COUNT 后 INSERT」，`project_review` 无 `pledge_id` 唯一键 | 并发提交写入多条评价 | 加唯一索引并捕获 1062 返回既有记录 |
| **D6** | 阻断 | 跨库事务依赖「同一 MySQL 实例」假设：`data/order.go:241,260-264` 在 order 池事务内跨库 UPDATE `planstar_ticket`；`payment.go:42-46`、`ops.go:85-101`、`wallet.go:159-161`、`shop.go:99-136` 同类混用 | 一旦拆库或读写分离，锁与原子性同时失效 | 启动期断言同实例，或改 Saga/TCC |

### 1.3 部署与运维线（6 项，其中 4 项为本批次改名遗留）

| ID | 等级 | 问题与证据 | 场景 | 建议 |
|---|---|---|---|---|
| **O1** | ~~阻断~~ **FIXED**·**改名遗留** | `.github/workflows/release.yml:69` `context: ./planstar`，该目录在改名后已不存在（实际为 `planstar_backend/`） | release 构建必然失败 | 改为 `./planstar_backend`；用 `docker build` 实测 |
| **O2** | ~~阻断~~ **FIXED**·**改名遗留** | `planstar_backend/deploy.sh:73` 与 `.github/workflows/release.yml:148` 仍设 `COMPOSE_DIR=${APP_DIR}/planstar`；`deploy/preflight.sh:15` 会 `cd ${COMPOSE_DIR}` | 远程 preflight 因目录不存在中断，部署失败 | 改为 `${APP_DIR}/planstar_backend` |
| **O3** | ~~阻断~~ **FIXED**·**改名遗留** | `planstar_backend/deploy/backup.sh:9` `COMPOSE_DIR=/opt/planstar_backend/planstar`、`:11` `ENV_FILE=.env`（实际 `.env.prod`） | cron 调用备份脚本失败；无备份即无恢复 | 修正默认路径与 env 文件名；补 restore 脚本与演练 |
| **O4** | ~~阻断~~ **FIXED**·**改名遗留** | `.github/workflows/release.yml:89` 用 `vars.SERVER_HOST != ''` 作为 deploy job 条件，而实际 host 由 `secrets.SERVER_HOST` 提供 | 只配置 secrets 时部署被**静默跳过**，workflow 仍显示成功 | 条件改为 `secrets.SERVER_HOST != '' || vars.SERVER_HOST != ''`，并加「部署已跳过」显式输出 |
| **O5** | 阻断·既有 | `docs/PRODUCTION_DELIVERY.md:239` 记载主账号 AK/SK 曾出现在聊天记录，该项仍为未勾选 `[ ]` | 凭据已泄露，未轮换即可被滥用 | 立即轮换 AK/SK 并吊销旧密钥（这是运维动作，不是代码修复） |
| **O6** | 阻断·既有 | `docs/deployment.md:84-104` 指导生产 `cd /opt/planstar_backend/planstar`、`cp .env.example .env`（含 `root/root`、`admin/admin`）、`docker compose pull`（无 prod overlay） | 照文档操作将以弱口令和演示数据上线 | 标记该章节废弃或重写为 prod overlay 流程 |

---

## 2. 高风险与中风险（17 项）

### 2.1 安全

| ID | 等级 | 问题 | 证据 |
|---|---|---|---|
| S5 | 高 | `docker-compose.prod.yml:150` 只读挂载 `/var/run/docker.sock` + `deploy/traefik/traefik.yml:15` `api.insecure: true`，等价于宿主机 root | 审计线 C |
| S6 | 中 | 登录/刷新入口的 CIDR 拒绝与 429 均无审计日志；`clientKey()=RemoteAddr` 为进程内计数器，多实例阈值放大 N 倍 | `admin_auth.go:51-68,298-351` |
| S7 | 中 | RBAC 管理接口把 `err.Error()` 回显到 HTTP 响应，可能含 Redis/持久化内部细节 | `admin_auth.go:217,221,235,266` |
| S8 | 中 | 图形验证码可整体跳过：`service/auth.go:69` `if body.Captcha != "" && !verify(...)`，不带字段即不校验 | `service/auth.go:69,158-168` |
| S9 | 中 | C 端会话无服务端注销：`ResolveSession` (`biz/auth.go:284-307`) 无撤销态检查，`login_session` 无 logout 更新 | 全仓仅运营域有 logout |
| S10 | 中 | PBKDF2 仅 10000 次迭代，远低于 OWASP 600000 建议 | `internal/platform/pkg/secret/secret.go:14` |
| S11 | 中 | 全站无 CSP / X-Frame-Options / X-Content-Type-Options / HSTS 响应头（Traefik 仅配 HSTS） | 全局搜索 0 命中；`dynamic.yml:74-82` |
| S12 | 中 | dev 基线配置含 `root/root`、`amqp://root:root`；`conf.go:309` 允许 env 覆盖 `server.env`，误设 `dev` 可跳过全部生产断言 | `configs/config.yaml:25-30` |
| S13 | 低 | `.gitignore` 未覆盖 `.env.development` / `.env.staging` 等变体 | 根与后端 `.gitignore` |

### 2.2 数据与业务

| ID | 等级 | 问题 | 证据 |
|---|---|---|---|
| D7 | 高 | 支付单过期后直接 `UPDATE status='pending'` 复用，未核对渠道真实支付结果，可导致二次支付 | `data/payment.go:63-78` |
| D8 | 高 | 重复举报同为「先查后写」且无唯一键，24h 限制可并发绕过 | `data/project_write.go:331-341` |
| D9 | 中 | 单号依赖「毫秒+6 位随机」，重试仅 3 次；shop 用秒级+4 位冲突概率更高 | `data/order.go:410-412`、`payment.go:319-325`、`shop.go:188-190` |
| D10 | 中 | 退款金额校验不足：仅凭订单状态 1-5 与 `pay_amount>0` 建退款单，不校验已支付成功；多渠道支付时退款额可能不等于实付 | `data/payment.go:190-196`、`sql/17_pay_refund_chat.sql:35,59` |
| D11 | 中 | 下单兜底幂等键包含客户端金额，同一订单金额字段变化产生两个幂等键 | `service/order.go:119-126` |
| D12 | 中 | 签到星币/分单位错配：`SignInRewardCoins=100` 星币，入账 `reward*100=10000 分=100 元`，但 `TodayReward` 按「分」返回 100，前端展示差 100 倍 | `biz/wallet.go:8-9,99`、`data/wallet.go:179,184,188` |
| D13 | 中 | float64 中转 DECIMAL 金额，`fenToYuan` 用 `float64/100` 写库 | `data/order.go:66,87,377,414` 等 |
| D14 | 中 | `data/shared` helper 未覆盖关键路径：`order.go:96-100` 等仍手写 `sql.ErrNoRows`；`MapContextError` 全仓无调用 | `data/shared/errors.go:17-53` |
| D15 | 中 | 1062 回读仍在已回滚事务内，REPEATABLE READ 快照读可能看不到新提交记录 | `payment.go:97-101`、`project_write.go:221-224` |
| D16 | 中 | 状态机可越级：退款审核不校验订单状态即置 7；`SetPublishStatus` 无前置状态校验 | `data/ops.go:95-108`、`data/project.go:461-471` |
| D17 | 中 | 软删唯一键冲突（唯一键不含状态）、时区用 `loc=Local`、N+1 与深分页、JSON 解析静默吞错 | `data/data.go:136`、`project.go:161-184`、`project_helpers.go:58-75` 等 |

### 2.3 部署与运维

| ID | 等级 | 问题 | 证据 |
|---|---|---|---|
| O7 | 高 | release 无回滚 job/step；无版本化 migration，DB 回滚不可逆 | `release.yml:170` 仅注释；`PRODUCTION_DELIVERY.md:258` |
| O8 | 高 | 镜像名可能不一致：`release.yml:26` 推送 `${{ github.repository }}/planstar-api`，`.env.prod.example:16` 期望 `ghcr.io/uifj/planstar-api` | 两处配置 |
| O9 | 中 | base compose 无资源 limits（仅 prod overlay 有），单独用 base 启动无上限 | `compose.yml:366-480` |
| O10 | 中 | 告警错配：`PlanstarInfraDown` 含 consul 但 lite 档位不启动；3 条 MQ 告警在低配档位为死规则 | `alerts.yml:184-186,288-313` |
| O11 | 中 | tracing 实际关闭但文档仍称启用 Jaeger/otel-collector job | `docs/operations.md:266`、`docs/architecture.md:140` |
| O12 | 中 | `preflight.sh:36` 强制 `ACME_EMAIL`，但 `prod.yml:157 command: []` 清空命令行，实际使用硬编码邮箱，形成无效门禁 | 两处配置 |

---

## 3. 明确达标项（避免误判为遗漏）

审计同时确认以下生产关键能力**已正确实现**，不应被上述清单掩盖：

- **JWT**：算法白名单真正禁 `none`/HS384 混淆；iss/aud/exp/nbf/iat 全校验；typ 不可混淆；jti 撤销 fail-closed；密钥 32 字节下限与弱占位值双重断言；令牌不进日志/响应头。
- **OpsAuth**：共享令牌 fail-closed；CIDR 白名单；审计用 SHA256 前 8 字节作 actor 且不泄漏明文；break-glass 可在审计与 `/admin/rbac/me` 中区分。
- **容器**：Dockerfile 非 root；read_only / tmpfs / no-new-privileges；健康检查；`.dockerignore` 排除 `.env.*`、`configs/`、`deploy/`，构建阶段不泄漏密钥。
- **探针**：healthz 只表示存活（不因依赖抖动误杀容器）；readyz 有 2s TTL 缓存 + singleflight + 失败退避；关键/非关键依赖划分明确；未配置依赖标记 skipped。
- **可观测**：route 标签做数字 ID/token 归一化控制基数；`/metrics` 并发抓取限流；访问日志不含 query/body；SQL 全参数化，`fmt.Sprintf` 拼接点均为硬编码常量；无 `exec.Command`、无用户可控文件路径；panic 恢复仅日志记堆栈。
- **幂等正向证据**：`project_idea.go:143-158` 唯一键冲突回滚后回读并标记已投票；签到 `ON DUPLICATE KEY UPDATE id=id` 配唯一键有效。

---

## 4. 上线前闸门（Gate）

按以下顺序闭环后，才具备重新评估上线的条件：

1. **G0 安全**（必须先做）：S1 / S2 / S3 / S4 全部修复并补回归测试；RBAC 提权需专门攻击性测试。
2. **G1 发布链路**：O1 / O2 / O3 / O4 修复后，在真实 CI 与预发环境跑通 build → deploy → preflight → healthz/readyz → 回滚演练。
3. **G2 资金闭环**：D1 / D2 / D3 补齐支付回调、真实退款与状态机，并对账验证。
4. **G3 凭据**：O5 完成 AK/SK 轮换；`.env.prod` 全量替换；preflight 全绿。
5. **G4 数据完整性**：D4 / D5 / D6 修复并做并发压力测试。
6. **G5 文档**：O6 废弃/重写错误生产指引；O11 统一 tracing 状态表述。

---

## 5. 本次处置与维护约定

- 本报告**只登记不修复**；所有问题保留原状，代码与配置未做任何变更。
- 本轮工作区 21 项 JWT/RBAC 改动**未提交、未推送**（遵循用户「没问题后再提交推送」的要求）。
- 修复时应逐项在本表把状态改为 `FIXED` 并填 commit hash；新增问题必须给出可复核的 `路径:行号`。
- 每次修复后必须重跑：`go build` / `go vet` / `go test ./... -count=1` / `golangci-lint` / `governance` / 四份 compose `config -q` / 本地 Docker local deploy。
- 涉及 S1/S2/S3 的修复必须补**攻击性测试**（越权、并发重放），不能只补 happy path。

---

## 6. G0/G1 修复记录（2026-11-19，本批次）

按用户「先修 G0/G1」的决策，以下 8 项已修复并重新验证；**其余 8 项既有阻断（D1–D6、O5、O6）保持原状，未修复**。

| ID | 修复内容 | 关键实现 |
|---|---|---|
| S1 | RBAC 策略注入提权 | `authz/validateRule` 禁止运行时 `obj=*`/`act=*` 与 `sub=superadmin`；`admin_auth.go:addPolicy` 禁止给自己授权 |
| S2 | grantRole 提权 | `authz/GrantableRoles` 白名单（operator/auditor/rbac_admin），`AddGrouping` 拒绝 superadmin；`grantRole` 禁止自我授权 |
| S3 | refresh 并发重放 | `Revocation` 新增 `RevokeOnce`（Redis `SETNX` / 内存互斥），refresh 轮换改为原子消费，重复消费返回 401 |
| S4 | 生产明文 Cookie | `config.lite.yaml` 恢复 `cookie_secure: true`、关闭豁免；`docker-compose.server.yml` 的 8098 改为仅回环发布；生产档位测试断言同步 |
| O1 | release 构建失败 | `release.yml` 的 build context 改为 `./planstar_backend` |
| O2 | 远程部署路径中断 | `release.yml` 与 `deploy.sh` 的 `COMPOSE_DIR` 改为 `${APP_DIR}/planstar_backend` |
| O3 | 备份脚本失效 | `backup.sh` 默认 `COMPOSE_DIR` 与 `ENV_FILE=.env.prod` 修正 |
| O4 | 部署静默跳过 | 移除 job 级 `if` 条件，新增 `Assert deploy target configured` step，配置缺失时 workflow 显式失败 |

**攻击性回归测试**：`authz` 新增 `TestRuntimeRuleCannotGrantWildcardPrivilege` / `TestGroupingCannotGrantSuperAdmin`；`authn` 新增 `TestRevokeOnceIsAtomic`。既有 `ops_auth` / `admin_rbac` 测试全部保持通过。

**修复后验证**：`gofmt -l` 0、`go build ./...` 0、`go vet ./...` 0、`go test ./... -count=1` 0 失败、`golangci-lint` 0 issues、四份 compose `config -q` 全通过（含 prod+server 三文件合并）、`ci.yml` 8 jobs / `release.yml` 2 jobs YAML 解析通过、旧路径残留 0 命中。

> 修复代码与本审计记录同批提交；提交哈希以 `git log` 为准。剩余 8 项既有阻断仍在第 1/2 节登记，未修复。