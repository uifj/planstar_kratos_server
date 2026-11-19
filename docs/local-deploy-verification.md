# 本地 Docker 部署验证记录

> 日期：2026-11-19 · 环境：Windows 11 / Docker Desktop · 结论：**PASS**
> 关联决策：`docs/ADR/ADR-0001`（模块化单体）、`docs/ADR/ADR-0002`（目录结构）

## 1. 本次变更范围

| 项 | 变更前 | 变更后 |
|---|---|---|
| 前端目录 | `frontend/` | `planstar_frontend/` |
| 后端目录 | `planstar/` | `planstar_backend/` |
| Go module 名 | `planstar`、`planstar/admin` | **不变**（module path 与物理目录解耦，因此 100+ Go 文件的 import 无需改写） |
| CI / Release / 部署脚本路径 | `planstar/**`、`frontend/admin/**`、`working-directory: planstar` | `planstar_backend/**`、`planstar_frontend/admin/**`、`working-directory: planstar_backend` |
| 装配入口 | `cmd/planstar/main.go`（354 行） | `main.go`(26) + `wiring.go`(251) + `probes.go`(63) |
| 事件消费端 | `internal/service/event.go` | `internal/service/events/handler.go`（P2 首个域包子包） |
| 超大文件 | `internal/data/project.go`（34 KB / 1121 行） | `project.go`(8.3 KB) + `project_write.go`(14 KB) + `project_helpers.go`(11 KB) |

**compose 相对路径无需改动**：`../docs/sql/*.sql`、`./sql/*.sql`、`./configs/*.yaml`、`build.context: .` 均相对 compose 文件自身解析，目录改名后仍有效。已在 `docker-compose.yml` 与 `docker-compose.local.yml` 头部补注目录布局说明。

**刻意保护、未替换的标识符**：`@planstar/*`（pnpm workspace 包名）、`/v1/planstar/*`（admin HTTP 路径）、`module planstar/admin`、`planstar_*` 数据库名、`planstar-api` 服务名、`PLANSTAR_*` 环境变量、`planstar.exchange`、Prometheus `Namespace: "planstar"`。第一次批量替换曾误伤 `@planstar/` 与 `/v1/planstar/`，已全部还原并做残留扫描（0 命中）。

## 2. 环境与前置条件

```
Docker Server       : 29.6.2
Docker Compose      : v5.3.1
本地已有基础镜像     : golang:1.26.5 / debian:stable-slim / mysql:8.4 / redis:7-alpine
启动配置            : planstar_backend/.env.local（由 .env.local.example 复制）
compose             : planstar_backend/docker-compose.local.yml
```

## 3. 遇到的问题与处置

| 问题 | 原因 | 处置 |
|---|---|---|
| `docker compose up -d --build` 失败：`failed to resolve reference "docker.io/moby/buildkit:buildx-stable-1"` | Compose v5 的 bake 流程要拉取 buildx 镜像，本机到 Docker Hub 不通（Docker Desktop 无 HTTPS proxy） | 改用内置 builder 单独构建：`docker build --pull=false --build-arg VERSION=local -t planstar-api:local .`（全部基础镜像命中本地缓存，构建成功）；随后 `docker compose ... up -d --no-build` 启动 |
| API 容器反复重启：`Error 1045 Access denied for user 'root'@'172.22.0.4'` | 旧数据卷 `planstar-local_mysql_local_data` 是 7 天前用另一套口令初始化的，MySQL 不会重跑 initdb | `docker compose ... down -v` 删除本地演示卷后重新 `up -d --no-build`，用当前 `.env.local` 的 `planstar_local_only` 重新初始化 8 个 SQL 脚本 |
| 烟雾脚本提示「服务端鉴权关闭」 | 旧文案与 `OpsAuth` fail-closed 实现不符 | 修正 `scripts/smoke-local.{sh,ps1}` 文案为「无令牌时 `/metrics` 应返回 401」 |

## 4. 验证结果

### 4.1 静态门禁（提交前）

| 命令 | 结果 |
|---|---|
| `gofmt -l .`（`planstar_backend/`） | 空 |
| `go build ./...`（`planstar_backend/`） | 退出 0 |
| `go vet ./...` | 退出 0 |
| `go test ./... -count=1` | 全部包 `ok`（含 `internal/governance`、`internal/service/events`） |
| `go build ./...`（`planstar_frontend/admin/`，第二个 Go module） | 退出 0 |
| `docker compose -f docker-compose.yml config -q` | 退出 0 |
| `docker compose -f docker-compose.local.yml config -q` | 退出 0 |
| `.github/workflows/ci.yml` YAML 解析 | 7 个 job：`go / admin-go / admin-console / contract / governance / compose / observability-config` |
| `PLANSTAR_GOVERNANCE_STRICT` 默认开启后的 `internal/governance` | `ok`（main.go 26 行、最大文件 14 KB、单包 ≤40 文件全部达标） |
| `go test -race ./...` | 本机缺 `gcc`/CGO 无法执行，由 CI ubuntu runner 覆盖 |

### 4.2 运行态（local deploy）

启动命令（Windows PowerShell 等价形式）：

```powershell
cd planstar_backend
Copy-Item .env.local.example .env.local
docker build --pull=false --build-arg VERSION=local -t planstar-api:local .
docker compose --env-file .env.local -f docker-compose.local.yml up -d --no-build
```

| 检查 | 结果 |
|---|---|
| 容器状态 | `mysql` Up(healthy) / `redis` Up(healthy) / `planstar-api` Up |
| `GET 127.0.0.1:9108/healthz` | `200 {"status":"ok","version":"local",...}` |
| `GET 127.0.0.1:9108/readyz` | `200 {"ready":true,"status":"ok","checks":[mysql ok, redis ok, elasticsearch skipped, rabbitmq skipped, consul skipped]}` |
| `GET /v1/categories` | `200`，3116 字节 |
| `GET /v1/activities?pageIndex=1&pageSize=1` | `200`，JSON 可解析 |
| `POST /v3/user/sendLoginCode` | `200 {"success":true,"needCaptcha":false}`（脚本第二次调用命中限流返回 `429`，符合契约） |
| `GET /metrics` 无令牌 | `401 UNAUTHORIZED`（fail-closed 生效） |
| `powershell -File scripts/smoke-local.ps1` | `烟雾测试通过：4/6 项通过，2 项跳过`（退出码 0；跳过的两项是未配置 Ops 令牌时的 Bearer 抓取） |

`readyz` 中 ES / RabbitMQ / Consul 显示 `skipped` 是低配 local 档位的预期行为（`PLANSTAR_ES_ADDR`、`PLANSTAR_MQ_ADDR`、`PLANSTAR_CONSUL_ADDRESS` 置空），不是故障。

## 5. 结论

目录改名后，两个 Go module、四份 compose 配置、CI/Release 路径与本地 Docker 部署均验证通过，业务接口与运维探针行为未发生回归。**满足提交条件**。

## 6. 清理与复现提示

- 本地验证产生 `.env.local`（已被 `.gitignore` 排除）与 `planstar-local_*` 数据卷；如需从零复现，执行
  `docker compose --env-file .env.local -f docker-compose.local.yml down -v` 后再 `up -d --no-build`。
- 因 Docker Hub 不可达，后续 CI 的镜像构建仍走 GitHub Actions runner；本地复现请使用本文的「内置 builder + `--no-build`」两段式命令。
- 停止本地栈：`docker compose --env-file .env.local -f docker-compose.local.yml down`。
