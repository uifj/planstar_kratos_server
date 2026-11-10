# 企划星后端 · 生产上线整改交付说明

| 项 | 内容 |
|---|---|
| 交付日期 | 2026-11-08 |
| 交付范围 | `planstar_kratos_server/planstar/`（Go 服务、配置、SQL、compose、CI、部署脚本）；依据工作区 `.trae/specs/*` 与上一轮 `planstar-code-review-2026-11-08.md` 的 P0/P1 未完成任务 |
| 资源约束 | 单台 4 核 8G / 2M 带宽阿里云 ECS；短信已开通 **Dypnsapi 与 Dysmsapi 两种**；对象存储实际为**火山引擎 TOS（S3 协议）**，与代码内置阿里云 OSS SDK 不兼容，运行时通过部署侧 overlay 关闭 |
| 规范基线 | 《阿里巴巴 Java 开发手册（黄山版）》【强制】条款（Go 映射）+ GitHub 真实企业级项目实践，见 §9 |
| 交付物 | **本地版**（允许测试口令）与**生产上线版**（禁用测试口令）两套可运行 Docker 编排 |
| 验证状态 | `go build/vet/test` 全绿；两版 compose 与三文件合并 `config -q` 通过；`verify-hardening.sh` **PASS=31 / FAIL=0**；客户端 `flutter analyze` 无问题、`flutter test` 253 项全绿；详见 §7 |

---

## 1. 一句话结论

上一轮审查的 11 个 P0 中，本轮已闭环 **9 个**（后端侧全部），剩余 2 个属 Flutter 客户端侧（见 §8.1）。新增**默认拒绝的环境判定 + 生产启动 fail-fast 断言 + 测试口令双重禁用 + 最小权限 DB 账号**，并交付两套隔离的 Docker 版本。**后端代码本身已达可上线状态；上线仍须完成 §8 的人工项。**

---

## 2. 按《阿里巴巴 Java 开发手册》完成的整改映射

项目是 Go 服务，手册条款按语义映射到 Go（`BigDecimal`→`int64 分`、`SLF4J`→`log/slog`、`#{}`→`?` 占位、`finally 关闭`→`defer Close`）。约束级别以 [alibaba/p3c](https://github.com/alibaba/p3c) 仓库 `p3c-gitbook/` 为准。

| 手册条款 | 级别 | 原缺陷 | 本次修复 |
|---|---|---|---|
| [常量定义 1](https://github.com/alibaba/p3c/blob/master/p3c-gitbook/%E7%BC%96%E7%A8%8B%E8%A7%84%E7%BA%A6/%E5%B8%B8%E9%87%8F%E5%AE%9A%E4%B9%89.md) 禁魔法值 | 【强制】 | `"123456"`、`root/root` 硬编码 | `conf.LocalTestCredentialPassword` 单一常量；口令全部外置到 env |
| [异常处理 2/4/6](https://github.com/alibaba/p3c/blob/master/p3c-gitbook/%E5%BC%82%E5%B8%B8%E6%97%A5%E5%BF%97/%E5%BC%82%E5%B8%B8%E5%A4%84%E7%90%86.md) 不吞异常、不控流、关资源 | 【强制】 | `_, _ = ExecContext`、`_ = Publish` 后 Ack、`crypto/rand` 失败回退固定码 | `NewToken/loginCode` 返回 error；MQ 逐条检查 Publish/Ack/Nack；补偿逻辑删除 |
| [日志规约 1/6/7](https://github.com/alibaba/p3c/blob/master/p3c-gitbook/%E5%BC%82%E5%B8%B8%E6%97%A5%E5%BF%97/%E6%97%A5%E5%BF%97%E8%A7%84%E7%BA%A6.md) 门面、现场+堆栈、生产禁 debug | 1/6【强制】 | `TextHandler`、panic 无堆栈、验证码明文入日志 | `slog.JSONHandler`；`recoverer` 记 `debug.Stack()`；验证码只记 SHA-256 指纹 |
| [安全规约 1/2/4/7](https://github.com/alibaba/p3c/blob/master/p3c-gitbook/%E5%AE%89%E5%85%A8%E8%A7%84%E7%BA%A6.md) 越权、脱敏、校验、防重放 | 全部【强制】 | 运维 fail-open、令牌进响应头、幂等键可缺、限流单机 | fail-closed、移除响应头令牌、服务端派生幂等键、Redis 全局限流 |
| [索引规约 1](https://github.com/alibaba/p3c/blob/master/p3c-gitbook/MySQL%E6%95%B0%E6%8D%AE%E5%BA%93/%E7%B4%A2%E5%BC%95%E8%A7%84%E7%BA%A6.md) 唯一索引承接业务唯一性 | 【强制】 | 同 `quoteId` 可下 N 单 | `order_idempotency` 的 `uk_order_idempotency_user_key` 承接 `quote:<id>` |
| [建表规约 6/9](https://github.com/alibaba/p3c/blob/master/p3c-gitbook/MySQL%E6%95%B0%E6%8D%AE%E5%BA%93/%E5%BB%BA%E8%A1%A8%E8%A7%84%E7%BA%A6.md) decimal 禁 float、id 自增 | 【强制】 | `MAX(id)+1`、float 金额往返 | 支持单事务化；金额用 `int64 分`（本次未全面替换 float，列入 §8.2） |
| [单元测试 1/2/4](https://github.com/alibaba/p3c/blob/master/p3c-gitbook/%E5%8D%95%E5%85%83%E6%B5%8B%E8%AF%95.md) AIR、assert、环境无关 | 【强制】 | 旧测试固化了「生产静默丢弃短信」错误行为 | 新增/改写 6 个测试文件，全部表驱动、无外部依赖 |
| 黄山版 安全规约 10 配置密码加密（[二手佐证](https://blog.csdn.net/qq_41460383/article/details/128317067)，编号待 PDF 复核） | 【强制】 | compose `:-root` / `:-admin` | 生产 overlay 全部 `${VAR:?}`，`preflight.sh` 拒 `<REPLACE_ME>` |

---

## 3. 已修复的阻塞项（对应上一轮审查编号）

| 编号 | 问题 | 修复落点 |
|---|---|---|
| P0-01 | 短信注册用户固定口令 `123456` | `internal/data/auth.go`：默认 `password_hash=""`；仅 `allow_test_credentials=true` 时下发；`cmd/planstar/main.go` 生产启动断言 `AssertNoTestCredentials`；`sql/21_disable_test_accounts.sql` 清空演示账号哈希并停用 |
| P0-02 | 运维鉴权 fail-open | `internal/server/ops_auth.go`：`Wrap` 无令牌直接 401；`Validate` 改用 `conf.IsProductionLikeEnv`；`internal/conf/conf.go` 默认拒绝环境判定 |
| P0-03 | 会话令牌进可读响应头 | `internal/pkg/session/session.go` 删除 `Header().Set(CookieKey, token)`；`internal/server/http.go` 从 `Expose-Headers` 移除 `psSession`；`dvid` Cookie 补 HttpOnly/Secure/SameSite 并缩短为 7 天 |
| P0-04 | 下单幂等键可缺失 | `internal/service/order.go` 新增 `deriveIdempotencyKey()`，缺头时按「用户+报价+订单要素」派生稳定 `auto:` 键 |
| P0-05 | 支持单跨库非事务 + 补偿失效 | `internal/data/project.go` 改为单 `*sql.Tx` 完成扣库存/写单/更新 `raised_amount`、`supporter_count`；删除 `restoreStock` |
| P0-06 | MQ 重发失败仍 Ack | `internal/mq/consumer.go` 重写：broker 级 DLX、投递失败 `Nack(requeue)`、`Qos(32)`、panic recover、指数退避 |
| P0-07 | 发布无 confirm | `internal/mq/publisher.go` 启用 `ch.Confirm(false)` + `NotifyPublish`，等待 ack/nack/超时 |
| P0-10 | 监控面公网无鉴权 | `deploy/traefik/dynamic.yml` 删除 3 条 `web` 面板路由并加 `security-headers`；`traefik.yml` 关闭 insecure dashboard；prod overlay 去掉 `--web.enable-lifecycle` |
| P0-11 | 部署命令注入 | `.github/workflows/release.yml` 白名单正则校验 tag + `envs` 传参，远程脚本零 `${{ }}` 插值 |
| P1-08 | Redis 风控死代码 / 单机限流 | `internal/service/auth.go` 接入 `AllowSendCode` / `NeedsCaptcha` / `BumpLoginFail` / `ResetLoginFail`；`biz` 补 nil-cache 保护 |
| P1-11/12/13 | 验证码重放、明文入库、随机数回退 | `internal/data/auth.go` 单条原子 `DELETE ... AND expire_time > NOW()` 校验 `RowsAffected()==1`；HMAC 摘要入库；`crypto/rand.Int` 逐位生成且失败返回 error |
| P1-14 | 消费者无 recover/Qos | 见 P0-06 |
| P1-16/17/19 | 弱口令、备份库名错、演示数据入生产 | prod overlay 强制 `${VAR:?}`；`deploy/backup.sh` 改为真实 11 个库；`sql/14_noop.sql` 覆盖演示种子挂载 |
| P1-20/22 | 日志泄露验证码、文档契约漂移 | `notify.go` 只打指纹；本文件即更新后的交付事实源 |
| P2-13/14/19/20 | 会话 Cookie 属性、时序侧信道、fail-fast、告警占位 | 分别由 session.go、biz/auth.go 的 dummy hash、`conf.validateProductionInvariants`、`preflight.sh` 的 `AM_*` 必填校验覆盖 |

---

## 4. 两套 Docker 可运行版本

### 4.1 本地版（允许测试口令）

```bash
cd planstar_kratos_server/planstar
cp .env.local.example .env.local          # 可选，compose 内已有等价默认值
./up-local.sh
# 或：docker compose --env-file .env.local -f docker-compose.local.yml up -d --build
```

- 独立文件 `docker-compose.local.yml`，**不依赖** `docker-compose.yml`，`name: planstar-local`，独立 volume/network。
- 仅 3 个容器：`mysql` / `redis` / `planstar-api`；资源上限 1G/256M/768M。
- 端口默认绑 `127.0.0.1`；真机联调设 `PLANSTAR_LOCAL_BIND=0.0.0.0`。
- Flutter 联调：`flutter run --dart-define=PS_API_BASE=http://10.0.2.2:8098/`。
- 测试口令：演示账号 `13800000001` / `13800000002`，口令取自 `conf.LocalTestCredentialPassword`；演示业务数据来自 `docs/sql/14_mock_seed.sql`。

### 4.2 生产上线版（禁用测试口令）

```bash
cd planstar_kratos_server/planstar
cp .env.prod.example .env.prod && chmod 600 .env.prod
# 逐项替换全部 <REPLACE_ME>（见 planstar_manual_setup/.trae/manual-setup-guide.md）
./up-prod.sh
# 或：docker compose --env-file .env.prod \
#       -f docker-compose.yml -f docker-compose.prod.yml \
#       -f docker-compose.server.yml up -d     # 第三份为部署侧覆盖，存在才加
```

- `docker-compose.yml`（base）+ `docker-compose.prod.yml`（overlay）+ 可选 `docker-compose.server.yml`（部署侧覆盖，运行时关闭与代码不兼容的对象存储），遵循 [Docker Compose Merge 规范](https://docs.docker.com/compose/how-tos/multiple-compose-files/merge/)：`environment` 按键覆盖、`volumes` 按容器路径覆盖、`command`/`mem_limit`/`cpus` 单值替换。
- `up-prod.sh` / `deploy.sh` / `preflight.sh` / `release.yml` 会**自动检测并叠加** `docker-compose.server.yml`，脚本与线上三文件合并方式一致。
- 4C8G/2M 档位：`PLANSTAR_CONFIG_FILE=config.lite.yaml`，ES/MQ/Consul/Jaeger/OTel/Loki 由 `profiles: ["full"]` 隔离，不指定 profile 即不启动；搜索降级 MySQL。
- 公网只保留 Traefik 80/443，其余组件端口一律 `127.0.0.1`。

### 4.3 「禁用测试口令」的三道闸门

1. **SQL 层**：prod overlay 用 `sql/14_noop.sql` 覆盖 `14_mock_seed.sql` 挂载点（零演示数据）；追加 `sql/21_disable_test_accounts.sql` 清空演示账号 `password_hash`、置 `status=0`、清会话。
2. **配置层**：`PLANSTAR_AUTH_ALLOW_TEST_CREDENTIALS="false"`；`conf.Validate()` → `validateProductionInvariants()` 在该开关为 true 时**拒绝启动**；`preflight.sh` 额外检查 env 与 YAML。
3. **运行层**：`cmd/planstar/main.go` 的 `AssertNoTestCredentials` 扫描所有 `status=1` 账号的口令哈希，命中测试/弱口令即终止启动；`NewAuthRepoForEnvironment` 在生产给新注册用户写入空 `password_hash`，密码登录分支必然失败。

---

## 5. 两版配置差异

| 维度 | 本地版 | 生产上线版 |
|---|---|---|
| 编排文件 | `docker-compose.local.yml`（standalone） | `docker-compose.yml` + `docker-compose.prod.yml` + `docker-compose.server.yml`（部署侧可选） |
| 环境文件 | `.env.local`（测试口令可明文） | `.env.prod`（`chmod 600`，`:?` 强制，禁 `<REPLACE_ME>`） |
| 服务配置 | `configs/config.local.yaml` | `configs/config.lite.yaml`（4C8G 档位） |
| `server.env` | `local` | `prod` |
| `auth.allow_test_credentials` | **true** | **false**（`conf.Validate` 强制） |
| `auth.cookie_secure` | false（本地 HTTP） | true |
| 演示业务数据 `14_mock_seed.sql` | ✅ 挂载 | ❌ 由 `14_noop.sql` 覆盖 |
| 演示账号 `15_auth_support.sql` | ✅ 挂载（含测试口令） | ✅ 挂载（保 schema）+ `21_disable_test_accounts.sql` 禁用 |
| 最小权限账号 `22_create_app_user.sh` | ❌ 用 root | ✅ 创建 `planstar_app` |
| DB 连接账号 | root / 本地测试口令 | `planstar_app` / 强口令；`conf.Validate` 拒绝 root |
| 短信通道 | logSMS（仅验证码指纹，不发明文） | **阿里云 Dypnsapi（强制启用）**，`NewSMS` 缺配置即启动失败 |
| OSS | 关闭 | 基础编排启用阿里云 OSS；**实际部署由 `docker-compose.server.yml` 覆盖为关闭**（实际存储为火山引擎 TOS / S3，SDK 不兼容） |
| 运维端点 | 无令牌 → `/metrics` 返回 401（fail-closed） | 令牌 + CIDR 白名单，缺一启动失败 |
| 公网监控面板 | 不适用 | 已从 :80 撤下，仅 SSH 隧道访问 |
| 验证码入库 | HMAC 摘要 | HMAC 摘要（独立 `PLANSTAR_AUTH_CODE_HMAC_KEY`） |
| 日志格式 | JSON | JSON（结构化、脱敏、带 traceId） |
| 资源上限 | mysql 1G / redis 256M / api 768M | mysql 2G / redis 512M / api 1G / admin 512M / prometheus 768M |
| MySQL buffer pool | 256M | 1G（不照搬官方「专用服务器 80%」） |
| Redis maxmemory | 128mb / allkeys-lru | 384mb / allkeys-lru |
| 镜像 | 本地构建 `planstar-api:local` | GHCR 固定语义化 tag（禁 latest） |
| 初始化挂载数 | 8 个 SQL | 8 个 SQL + `14_noop` 覆盖 + `21` + `22` |

---

## 6. 改动文件清单

**修改（30 个）**

```
.github/workflows/release.yml
planstar/.gitignore                          # 忽略 .env.local / .env.prod / 校验临时文件
planstar/cmd/planstar/main.go
planstar/configs/config.docker.yaml          # 显式 env: local，避免默认推导成 prod
planstar/configs/config.lite.yaml            # 显式禁测试口令
planstar/configs/config.prod.yaml            # 显式禁测试口令
planstar/deploy.sh                           # 只走 prod overlay + preflight + 标签校验
planstar/deploy/backup.sh                    # 修正为真实 11 个库 + 空档校验
planstar/deploy/preflight.sh                 # 发布前门禁：必填/占位/口令/备份/compose
planstar/deploy/traefik/dynamic.yml          # 撤下公网面板路由 + 安全响应头
planstar/deploy/traefik/traefik.yml          # 关闭 insecure dashboard
planstar/internal/biz/auth.go                # 随机码失败返回 error、时序侧信道、nil-cache 保护
planstar/internal/conf/conf.go               # 环境判定/测试口令开关/生产 fail-fast/验证码 HMAC 密钥
planstar/internal/conf/conf_test.go          # 适配默认拒绝
planstar/internal/data/auth.go               # 删除固定口令、原子消费验证码、HMAC 入库、弱口令断言
planstar/internal/data/order.go              # 报价快照一次性消费
planstar/internal/data/project.go            # 支持单单事务化 + 聚合指标
planstar/internal/mq/consumer.go             # DLX/confirm/Nack/Qos/recover/退避（重写）
planstar/internal/mq/publisher.go            # publisher confirm（重写）
planstar/internal/mq/server.go               # 注入结构化 logger
planstar/internal/pkg/notify/notify.go       # 生产 fail-closed、日志脱敏、环境统一
planstar/internal/pkg/notify/notify_test.go
planstar/internal/pkg/notify/aliyun_test.go
planstar/internal/pkg/session/session.go     # 移除令牌响应头、Cookie 加固（重写）
planstar/internal/pkg/session/session_test.go
planstar/internal/server/http.go             # CORS/头部超时/panic 堆栈/不信任转发头
planstar/internal/server/ops_auth.go         # fail-closed + 默认拒绝
planstar/internal/server/ops_auth_test.go
planstar/internal/service/auth.go            # Redis 全局风控接线、Cookie Secure
planstar/internal/service/order.go           # 服务端派生幂等键
```

**新增（14 个，含本文档）**

```
docs/PRODUCTION_DELIVERY.md                  ← 本文件
planstar/.env.local.example
planstar/.env.prod.example
planstar/configs/config.local.yaml
planstar/docker-compose.local.yml
planstar/docker-compose.prod.yml
planstar/internal/conf/production_test.go
planstar/internal/service/order_idempotency_test.go
planstar/sql/14_noop.sql
planstar/sql/21_disable_test_accounts.sql
planstar/sql/22_create_app_user.sh
planstar/up-local.sh
planstar/up-prod.sh
planstar/verify-hardening.sh
```

---

## 7. 验证要点（本次实测）

| # | 验证命令 | 结果 |
|---|---|---|
| 1 | `cd planstar && go build ./...` | ✅ 通过 |
| 2 | `go vet ./...` | ✅ 通过 |
| 3 | `go test ./...` | ✅ 全绿（`conf`/`data`/`server`/`service`/`notify`/`session` 等 12 包） |
| 4 | `gofmt -l internal cmd` | ✅ 无未格式化文件 |
| 5 | `docker compose --env-file .env.local.example -f docker-compose.local.yml config -q` | ✅ 通过 |
| 6 | `docker compose --env-file <占位副本> -f docker-compose.yml -f docker-compose.prod.yml -f docker-compose.server.yml config -q` | ✅ 三文件合并通过（与线上一致） |
| 7 | `docker compose ... config \| grep 14_noop / 21_disable / 22_create` | ✅ 生产版挂载 `14_noop.sql`、`21_disable_test_accounts.sql`、`22_create_app_user.sh`；本地版只挂 `14_mock_seed.sql` + `15_auth_support.sql` |
| 8 | `bash -n` 全部 7 个 shell 脚本 | ✅ 语法通过 |
| 9 | 新增单测：`TestProductionInvariants*`、`TestOpsAuthFailsClosedWithoutToken`、`TestDeriveIdempotencyKey*`、`TestSetAndReadCookie`、`TestNewSMSProductionFailsClosed` | ✅ 通过 |
| 10 | `CGO_ENABLED=1 go test -race ./...` | ⚠️ 本机无 gcc（Windows），`-race` 无法执行；CI 的 Linux runner 必须执行此项 |
| 11 | `./verify-hardening.sh` | ✅ **PASS=31 / FAIL=0，退出码 0**（含新增的三文件 compose 检查） |
| 12 | `ENV_FILE=<占位副本> COMPOSE_DIR=$PWD PLANSTAR_CONFIG_FILE=config.lite.yaml ./deploy/preflight.sh` | ✅ 通过（必填项、占位符、固定 tag、禁测试口令、备份库名、三文件 compose 解析） |
| 13 | `flutter analyze lib test` | ✅ No issues found（2026-11-09 客户端修复后） |
| 14 | `flutter test`（全量） | ✅ **253 项全绿**（含新增选票参数、提交并发、聚合回滚回归） |

**手工验收入口**

```bash
# 本地版：启动 → 健康检查 → 演示账号登录 → 目录接口
./up-local.sh
curl -fsS http://127.0.0.1:9108/healthz
curl -fsS 'http://127.0.0.1:8098/v1/projects?page=1&pageSize=1'
curl -i -X POST http://127.0.0.1:8098/v1/user/login \
  -H 'Content-Type: application/json' \
  -d '{"ua":"13800000001","pw":"123456"}'      # 本地应成功；生产必须 401
# 关键回归：登录响应不得出现 psSession 响应头
```

```bash
# 生产版：先门禁后启动
ENV_FILE=.env.prod COMPOSE_DIR="$PWD" PLANSTAR_CONFIG_FILE=config.lite.yaml ./deploy/preflight.sh
./up-prod.sh
curl -fsS http://127.0.0.1:9108/readyz
curl -i -X POST https://api.planstar.top/v1/user/login \
  -H 'Content-Type: application/json' -d '{"ua":"13800000001","pw":"123456"}'
# 期望 401；若返回 200 说明测试口令未禁用，立即回滚
```

---

## 8. 上线检查项

### 8.1 阻断项（未完成不得上线）

- [ ] **`.env.prod` 全部 `<REPLACE_ME>` 已替换**，`chmod 600`，且已加入服务器备份白名单（不含入 git）。
- [ ] **`./deploy/preflight.sh` 全绿**；`docker compose --env-file .env.prod -f docker-compose.yml -f docker-compose.prod.yml -f docker-compose.server.yml config -q` 通过（三文件合并，与线上一致）。
- [x] **短信产品已确认**：用户已开通 **Dypnsapi（号码认证）与 Dysmsapi（传统短信）两种**。当前代码按 `.trae` 规格走 Dypnsapi `SendSmsVerifyCode`（`internal/pkg/notify/aliyun.go`），生产配置为签名「恒创联众」+ 模板 `10001`（登录注册）。若日后要切到 Dysmsapi `SendSms`，需新增 provider 分支（SDK `dysmsapi-20170525`），模板参数与返回码校验不同，见根目录《生产上线人工配置与服务器部署说明》。
- [ ] **真实短信联调**：正确端点是 `POST /v3/user/sendLoginCode`（**不是** `/v1/user/send-code`）。用测试手机号走发码 → 查收 → 登录；确认 Dypnsapi 返回的验证码与本地生成并入库的码一致（该 API 由服务端下发 `TemplateParam {"code":...}`），日志只有脱敏手机号与 `code_fingerprint`，无验证码/密钥明文。
- [ ] **🚨 AK/SK 轮换（最高优先级）**：配置阶段的主账号 AK/SK 曾出现在聊天记录中。上线后立即在阿里云 RAM 创建仅授权 `dypnsapi:SendSmsVerifyCode`（及所用 Dysmsapi 权限）的子账号 AK/SK，替换 `.env.prod` 并重启，然后在 RAM 禁用/删除旧 AK。
- [ ] **对象存储**：实际为火山引擎 TOS（S3 协议），与代码内置阿里云 OSS SDK 不兼容，已由 `docker-compose.server.yml` 运行时关闭。注意 `docker-compose.prod.yml` 对 OSS 变量用了 `${VAR:?}`，因此 `.env.prod` 中 OSS 字段**即使关闭也必须保留非空占位**，否则 compose 解析阶段即失败。后续实现 S3 provider 后再启用。
- [ ] **测试口令回归**：用 `13800000001/13800000002` + `123456` 请求生产登录，必须 401；查 `user_info` 中 `password_hash='' AND status=0`。
- [ ] **会话令牌回归**：登录响应 `psSession` 只出现在 `Set-Cookie`（含 `HttpOnly; Secure; SameSite=Lax`），**不得**出现在普通响应头；`curl -I` 的 `Access-Control-Expose-Headers` 不含 `psSession`。
- [ ] **运维面回归**：无 Bearer 访问 `127.0.0.1:9108/metrics` 返回 401；公网 `http://<host>/` 带 `Host: grafana.example.com` 不再到达 Grafana；`/-/reload` 返回 404。
- [ ] **备份恢复演练**：`./deploy/backup.sh pre-deploy` 产出非空 `mysql.sql.gz`，并在临时实例 `mysql < mysql.sql` 能导入 11 个库。
- [x] **Flutter 客户端 4 项 P0/P1 已修复**（2026-11-09，`flutter_piaoniu_app/`）：
  - P0-08 选票参数：`router.dart` 解析并透传 `tier/qty/session`；新增 `OrderConfirmArgs` 作为 controller/seed 的 family 参数；票数按限购收敛、越界回退，金额始终由服务端票档重算。
  - P0-09 下单一致性：`TicketOrder.reopenForRetry()` 在外部分写入失败时回滚聚合并丢弃幽灵 `OrderPlaced`；`_apply()` 在提交中/已提交时禁止编辑且用 `copyWith` 保留 `submitting`；`_refreshServerQuote()` 用请求代数丢弃陈旧响应；建单体改用本次校验过的报价。
  - P1-01 金额口径：`order_list_page.dart` / `order_detail_page.dart` 三处跳转改取 `order.quote?.paymentAmount ?? goodsTotal`。
  - P1-02/03 Android：`build.gradle.kts` release 改用 `key.properties` 正式签名（缺配置即 fail-fast，禁止回退 debug key），启用 R8 + `proguard-rules.pro`；`usesCleartextTraffic=false` + `networkSecurityConfig`（release 禁明文，debug 源集放开本地联调）。
  - 验证：`flutter analyze lib test` 无问题；`flutter test` **253 项全绿**（含新增的选票参数、提交并发与回滚回归测试）。
- [ ] **客户端发版前**：按 `android/key.properties.example` 生成正式 keystore 并配置签名，做一次 release 真机回归（推送/地图/微信/登录/支付），确认 R8 混淆未破坏第三方 SDK。

### 8.2 建议跟进（不阻断本次上线）

- 金额全链路仍有 `float64 ↔ int64 分` 往返（`biz/models.go:9-13`、`data/order.go:395`、`data/project.go`），建议按 [shopspring/decimal](https://github.com/shopspring/decimal) 或全链路 `int64 分` 统一。
- `nextID` 仍是 `MAX()+1`（`data/project.go`），并发下依赖唯一键冲突；建议改 `AUTO_INCREMENT` / 雪花 ID。
- `appleboy/ssh-action@v1.2.0` 等第三方 action 仍用可变 tag，建议 pin 到完整 commit SHA（需从上游仓库获取真实 SHA，本次不编造）。
- 无版本化 migration；`14_noop`/`21`/`22` 仅在数据卷为空时执行，存量库需手工执行等价 SQL。
- 黄山版 PDF 全文未在本机提取，「新增 11 条」编号来自二手转载，正式合规文档请以 [PDF 原文](https://github.com/alibaba/p3c/blob/master/Java%E5%BC%80%E5%8F%91%E6%89%8B%E5%86%8C(%E9%BB%84%E5%B1%B1%E7%89%88).pdf) 复核。
- `internal/mq/` 仍无单测；建议在 CI（Linux + gcc）补 `go test -race ./...` 与 MQ 重试/DLQ 表驱动测试。

### 8.3 4C8G / 2M 资源预算（已写入 prod overlay）

| 组件 | mem_limit | cpus | 说明 |
|---|---|---|---|
| mysql | 2g | 1.5 | `innodb-buffer-pool-size=1G`、`max_connections=150` |
| redis | 512m | 0.5 | `maxmemory 384mb`、`allkeys-lru` |
| planstar-api | 1g | 1.5 | 非 root、`read_only`、`no-new-privileges` |
| admin | 512m | 0.5 | |
| prometheus | 768m | 0.5 | TSDB 保留 15d，去掉 lifecycle API |
| grafana / traefik / promtail | 384m / 256m / 192m | 0.25 / 0.5 / 0.25 | |
| **合计上限** | **≈ 5.6g** | **≈ 5.5** | 留 ~2.4G 给 OS、page cache 与突发；cpus 为份额非硬限 |

2M 带宽：Traefik `compress` 已启用；镜像用 [`docker save | gzip` / `load`](https://docs.docker.com/reference/cli/docker/image/save/) 离线预热，避免公网拉镜像打满带宽。依据 [MySQL Buffer Pool 官方](https://dev.mysql.com/doc/refman/8.0/en/innodb-buffer-pool.html)（80% 仅适用专用服务器）、[Redis Eviction 官方](https://redis.io/docs/latest/develop/reference/eviction/)（默认 maxmemory=0 无限制）、[Traefik Compress](https://doc.traefik.io/traefik/middlewares/http/compress/)。

---

## 9. 参考来源（GitHub / 官方文档，均经实际访问核验）

**规范**
1. [alibaba/p3c](https://github.com/alibaba/p3c) — 官方仓库，README 标注最新为黄山版（2022.2.3）；PMD 49 条规则。
2. [Java开发手册(黄山版).pdf](https://github.com/alibaba/p3c/blob/master/Java%E5%BC%80%E5%8F%91%E6%89%8B%E5%86%8C(%E9%BB%84%E5%B1%B1%E7%89%88).pdf) — 仓库根目录 PDF。
3. [p3c-gitbook/SUMMARY.md](https://github.com/alibaba/p3c/blob/master/p3c-gitbook/SUMMARY.md) — 七大章节与【强制】/【推荐】/【参考】分级。
4. [安全规约](https://github.com/alibaba/p3c/blob/master/p3c-gitbook/%E5%AE%89%E5%85%A8%E8%A7%84%E7%BA%A6.md) / [异常处理](https://github.com/alibaba/p3c/blob/master/p3c-gitbook/%E5%BC%82%E5%B8%B8%E6%97%A5%E5%BF%97/%E5%BC%82%E5%B8%B8%E5%A4%84%E7%90%86.md) / [日志规约](https://github.com/alibaba/p3c/blob/master/p3c-gitbook/%E5%BC%82%E5%B8%B8%E6%97%A5%E5%BF%97/%E6%97%A5%E5%BF%97%E8%A7%84%E7%BA%A6.md) / [建表规约](https://github.com/alibaba/p3c/blob/master/p3c-gitbook/MySQL%E6%95%B0%E6%8D%AE%E5%BA%93/%E5%BB%BA%E8%A1%A8%E8%A7%84%E7%BA%A6.md) / [索引规约](https://github.com/alibaba/p3c/blob/master/p3c-gitbook/MySQL%E6%95%B0%E6%8D%AE%E5%BA%93/%E7%B4%A2%E5%BC%95%E8%A7%84%E7%BA%A6.md) / [单元测试](https://github.com/alibaba/p3c/blob/master/p3c-gitbook/%E5%8D%95%E5%85%83%E6%B5%8B%E8%AF%95.md)。

**Go / 后端工程实践**

5. [go-kratos/kratos app.go](https://raw.githubusercontent.com/go-kratos/kratos/main/app.go) — `/v3`、`errgroup` + signal、生命周期钩子。
6. [go-kratos/kratos/config](https://github.com/go-kratos/kratos/tree/main/config) — `env`/`file` 多源 merge。
7. [uber-go/guide style.md](https://github.com/uber-go/guide/blob/master/style.md) — Don't Panic / Exit Once / Goroutine 生命周期。
8. [RabbitMQ 官方 Confirms](https://www.rabbitmq.com/docs/confirms) — `confirm.select`、manual ack、prefetch、幂等；直接支撑 P0-06/07 修复。
9. [pkg.go.dev/log/slog](https://pkg.go.dev/log/slog) — `JSONHandler`、`*Context`、`LevelVar`。
10. [pkg.go.dev/crypto/rand](https://pkg.go.dev/crypto/rand) — `Read` 不返回错误、`Int` 返回 err；固定回退值语义错误。
11. [shopspring/decimal](https://github.com/shopspring/decimal) — float64 精度问题。
12. [open-telemetry/opentelemetry-go](https://github.com/open-telemetry/opentelemetry-go) — W3C `traceparent` 传播。
13. [alibaba/sentinel-golang](https://github.com/alibaba/sentinel-golang) — 限流/熔断（后续演进参考）。

**阿里云资源**

14. [aliyun/aliyun-oss-go-sdk](https://github.com/aliyun/aliyun-oss-go-sdk) — V1 v3.0.2；官方已出 [V2 SDK](https://github.com/aliyun/alibabacloud-oss-go-sdk-v2)。
15. [阿里云 OSS Go 配置访问凭证](https://help.aliyun.com/zh/oss/developer-reference/go-configure-access-credentials) — 推荐最小权限 RAM 用户 AK、9 种凭据方式。
16. [aliyun/credentials-go](https://github.com/aliyun/credentials-go) — 默认 provider chain、ECS RAM 角色免 AK。
17. [alibabacloud-go/dypnsapi-20170525](https://github.com/alibabacloud-go/dypnsapi-20170525) — 当前 `.trae` 规格所用 SDK。
18. [alibabacloud-go-sdk Usage-CN.md](https://raw.githubusercontent.com/aliyun/alibabacloud-go-sdk/master/docs/Usage-CN.md) — 「默认不开启超时/重试」，必须显式配置。

**部署与隔离**

19. [Docker Compose Merge 官方](https://docs.docker.com/compose/how-tos/multiple-compose-files/merge/) — overlay 合并语义（本次两版隔离的实现依据）。
20. [Docker Compose 环境变量官方](https://docs.docker.com/compose/how-tos/environment-variables/set-environment-variables/) — `env_file` 优先级、敏感信息建议用 secrets。
21. [Spring Boot Profiles 官方](https://docs.spring.io/spring-boot/reference/features/profiles.html) — 环境分支装配范式。
22. [docker/awesome-compose](https://github.com/docker/awesome-compose) — 官方声明示例「must not be deployed in production」。
23. [Docker Compose services 参考](https://docs.docker.com/reference/compose-file/services/) 与 [Dockerfile USER](https://docs.docker.com/reference/dockerfile/#user)。
24. [Traefik BasicAuth](https://doc.traefik.io/traefik/middlewares/http/basicauth/) / [Compress](https://doc.traefik.io/traefik/middlewares/http/compress/)。
25. [MySQL InnoDB Buffer Pool](https://dev.mysql.com/doc/refman/8.0/en/innodb-buffer-pool.html) / [Redis Eviction](https://redis.io/docs/latest/develop/reference/eviction/) / [docker image save](https://docs.docker.com/reference/cli/docker/image/save/)。

> 说明：「生产禁用测试口令」在权威手册中无单一对应条款，本次按 [Spring Profiles](https://docs.spring.io/spring-boot/reference/features/profiles.html) + [Compose Merge](https://docs.docker.com/compose/how-tos/multiple-compose-files/merge/) + 独立 env/SQL 集 + 启动期断言组合实现，定位为工程方案而非引用现成规范。
