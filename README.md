# Planstar 后端服务

企划星（Planstar）Flutter 客户端的后端 API 服务：单一 Kratos v3 服务、单一二进制，
数据落在 9 个 `planstar_*` MySQL 库。客户端契约以
[docs/api/API_CONTRACT.md](docs/api/API_CONTRACT.md) 为准。

> 仓库根目录的历史 kratos-layout 模板已归档清理（根 `openapi.yaml`、根
> `docker-compose.yml`、buf 工具链等不再保留），真实服务代码全部在
> [`planstar/`](planstar) 独立模块中，后端 CI 只构建 `planstar/`。
>
> 根目录 `admin/` 是运营后台（Go API + 控制台），作为生产编排的**构建上下文**
> 参与部署；`docs/architecture.md` 为历史架构文档仅作参考。排障与开发请以
> `planstar/` 与 `docs/` 为准。

## 技术栈

- **Kratos v3**（HTTP 传输，`http.ServeMux` + 原生 `net/http` handler）
- **MySQL 8.x**：9 个库（user/catalog/ticket/order/payment/marketing/community/content/meta）
- **Redis**：会话缓存、验证码、发码限流、登录失败计数
- **RabbitMQ**：领域事件发布/消费（重试 + DLX + W3C trace 透传）
- **Elasticsearch**：演出读模型与热榜查询（失败回退 MySQL）
- **Consul** 服务注册、**Jaeger** 链路追踪、**Prometheus** 指标
- 依赖在 [`planstar/cmd/planstar/main.go`](planstar/cmd/planstar/main.go) 手工组装，无 Wire

## 目录结构

```text
planstar/                    独立 Go 模块（module planstar）
  cmd/planstar/              入口 + 依赖装配
  internal/conf/             配置结构与环境变量覆盖
  internal/biz/              用例、领域对象（DO）、仓储端口、领域事件
  internal/data/             仓储实现（MySQL / Redis / ES）、*Data 客户端持有者
  internal/service/          HTTP handler（DTO ↔ DO）
  internal/server/           业务端口 + 管理端口（/metrics /healthz /readyz）
  internal/mq/               RabbitMQ 发布器 / 消费者（DLX + trace）
  internal/observability/    Prometheus 指标 + Jaeger tracing + trace 日志
  internal/pkg/              captcha / secret / session / httpx
  configs/                   config.yaml / config.local.yaml / config.lite.yaml / config.prod.yaml
  deploy/                    Prometheus / Alertmanager / Grafana / Traefik / preflight.sh / backup.sh
  sql/                       认证支撑表与生产化补丁
  docker-compose.yml         生产编排 base（+ docker-compose.prod.yml 生产叠加）
                             （+ docker-compose.server.yml 部署侧覆盖；docker-compose.local.yml 本地联调）
docs/api/                    客户端 API 契约
docs/sql/                    9 个库建表 + 演示数据
```

## 本地开发

```bash
cd planstar
go build ./...
go test ./...
go run ./cmd/planstar -conf ./configs/config.yaml
```

端口：业务 `8098`；运维 `9108`（`/metrics`、`/healthz`、`/readyz`）。

## 本地一键启动（docker compose）

本地联调/Flutter 测试推荐使用**独立编排** `docker-compose.local.yml`（仅 3 个容器，
带演示数据与演示账号测试口令）：

```bash
cd planstar
./up-local.sh
# 或手动执行：
# docker compose --env-file .env.local -f docker-compose.local.yml up -d --build
```

本地版包含 MySQL / Redis / planstar-api，端口默认绑 `127.0.0.1`；真机联调设
`PLANSTAR_LOCAL_BIND=0.0.0.0`。停止：`docker compose --env-file .env.local -f docker-compose.local.yml down`。

> 说明：仓库另有一套 `docker-compose.yml` + `docker-compose.prod.yml`（生产）
> 与 `docker-compose.server.yml`（部署侧覆盖），仅用于服务器，禁止本地联调直连。

首次启动时 MySQL 按文件名字典序执行初始化脚本：

| 文件 | 用途 |
| --- | --- |
| `docs/sql/00_all.sql` | 建库建表 |
| `docs/sql/14_mock_seed.sql` | 演示数据（生产由 `14_noop.sql` 覆盖，零演示数据） |
| `planstar/sql/15_auth_support.sql` | 演示账号与认证支撑表 |
| `planstar/sql/16_production_hardening.sql` | 订单幂等/库存/唯一索引生产化补丁 |
| `planstar/sql/21_disable_test_accounts.sql`、`22_create_app_user.sh` | 仅生产：禁用演示账号 + 创建最小权限 DB 账号 |

> 初始化脚本仅在数据卷为空时执行；已有存量库用 `make sql-apply`（在 `planstar/` 下执行，
> 口令取 `MYSQL_ROOT_PASSWORD`，默认 `root`）补齐。

## 验证命令

```bash
curl -fsS http://127.0.0.1:9108/healthz    # {"status":"ok"}
curl -fsS http://127.0.0.1:9108/readyz     # 依赖就绪时 {"status":"ready"}
curl -fsS 'http://127.0.0.1:8098/v1/activities?pageIndex=1&pageSize=1'

# 一键烟雾测试：/healthz=200、目录 JSON 可解析、发码接口非 5xx
sh scripts/smoke-local.sh                                             # Git Bash / Linux
powershell -ExecutionPolicy Bypass -File scripts\smoke-local.ps1      # Windows
```

面板入口（低配档位默认只启 Grafana / Prometheus）：
Grafana `http://127.0.0.1:3000`、Prometheus `http://127.0.0.1:9090`；
全量档位（`COMPOSE_PROFILES=full`）另有 Jaeger `http://127.0.0.1:16686`、
Consul `http://127.0.0.1:8500`、RabbitMQ 管理台 `http://127.0.0.1:15672`。

## 前端联调 base URL

Flutter 客户端通过编译期开关 `PS_API_BASE` 指向本地后端（**结尾带 `/`**，未写时客户端自动补）：

```bash
cd flutter_piaoniu_app

# Android 模拟器：10.0.2.2 指向宿主机
flutter run --dart-define=PS_API_BASE=http://10.0.2.2:8098/

# 真机 / iOS 模拟器：换成开发机局域网 IP
flutter run --dart-define=PS_API_BASE=http://192.168.1.10:8098/

# 需要本地假数据时显式打开（生产默认关闭）
flutter run --dart-define=PS_API_BASE=http://10.0.2.2:8098/ --dart-define=PS_USE_DEMO=true
```

## CI/CD（GitHub Actions）

| 工作流 | 触发 | 产物 |
|---|---|---|
| [`CI`](.github/workflows/ci.yml) | push / PR 到 main（`planstar/**`、`docs/api/**`、`scripts/**`） | Go `gofmt`/`build`/`vet`/`test -race`、接口契约测试、`docker compose config`（base + local 两套） |
| Flutter `CI`（客户端仓库 `.github/workflows/ci.yml`） | push / PR 到 main（`lib/**`、`test/**`、`packages/**`） | `flutter analyze lib test packages/infras/core/lib`、`flutter test` |
| [`Release`](.github/workflows/release.yml) | push tag `v*.*.*` | 镜像 `0.1.0`/`0.1`/`v0.1.0`/`latest`（**Packages**）+ **Release** + 部署（**Deployments**） |
| [`GitHub Pages`](.github/workflows/pages.yml) | push 到 main（`site/**`） | 发布 `site/` 到 Pages |
| `Dependency Graph` | 仓库已启用（GitHub 内置） | 依赖关系可视化 |
| `Dependabot Updates` | 每周（[dependabot.yml](.github/dependabot.yml)） | 自动升级 Go / Actions / Docker 依赖 PR |

发布新版本：

```bash
git tag v0.1.1
git push origin v0.1.1
```

### 仓库设置（已完成）

1. **Settings → Actions → General → Workflow permissions**：**Read and write**（推送 GHCR / 创建 Release）。
2. **Settings → Pages → Source**：**GitHub Actions**（发布 `site/`）。
3. **Settings → Code security → Dependency graph**：**Enable**。
4. **Settings → Environments**：`production`（Release 部署记录，可加保护规则）。
5. **Settings → Secrets and variables → Actions**（部署用）：
   - `SERVER_HOST`、`SERVER_USER`、`SERVER_SSH_KEY`、`SERVER_PORT`、`APP_DIR`
   - `GHCR_TOKEN`（可选，拉取私有镜像用）

未配置 `SERVER_HOST` 时，部署 job 自动跳过（Packages / Releases 仍正常生成）。

## 部署到单台服务器

生产编排为三文件合并：`docker-compose.yml`（base）+ `docker-compose.prod.yml`
（生产叠加）+ `docker-compose.server.yml`（部署侧覆盖：关闭不兼容的对象存储、
临时公网暴露 8098 等），发布前强制走 `preflight.sh` 门禁。

```bash
cd /opt/planstar/planstar
cp .env.prod.example .env.prod && chmod 600 .env.prod   # 逐项填写 <REPLACE_ME>
# 关键项：各口令、阿里云号码认证(dypnsapi)短信 AK/SK/签名/模板CODE、
# VOLC_ACCESSKEY / VOLC_SECRETKEY（火山引擎 DNS Challenge 申请证书）、
# PLANSTAR_OPS_TOKEN 运维令牌、ACME_EMAIL 等

./up-prod.sh
# 或手动执行：
# export C="docker compose --env-file .env.prod -f docker-compose.yml -f docker-compose.prod.yml -f docker-compose.server.yml"
# ./deploy/preflight.sh && $C up -d --build
```

验证：`curl -fsS http://127.0.0.1:9108/readyz`；业务接口
`curl -fsS 'http://127.0.0.1:8098/v1/activities?pageIndex=1&pageSize=1'`。
HTTPS 由 Traefik 按 `api.planstar.top` / `admin.planstar.top` 自动签发证书
（Let's Encrypt DNS Challenge），需先把两条 A 记录解析到服务器公网 IP，
并在 ECS 安全组放行 80/443。

## 说明

- 本服务为手写 HTTP handler，不使用 protobuf / Wire 代码生成。
- 金额：数据库存元（`DECIMAL`），API 输出分（×100）。
- 鉴权：`Cookie: psSession=…`；登录失败 `5xx + text/plain`；发码恒 `200 + {success,needCaptcha,errorMsg}`。

## 许可证

本项目采用 **PolyForm Noncommercial License 1.0.0**（SPDX：`PolyForm-Noncommercial-1.0.0`），
完整条款见 [LICENSE](LICENSE)。核心约定：

- **允许**：任何非商业目的，包括个人学习、研究、实验、爱好项目，以及教育机构、
  慈善组织、公共研究机构、政府机构等非商业主体的使用与修改
- **禁止**：任何商业用途；商业使用需另行取得版权所有者授权（版权所有者 uifj 本人不受限制）
- **义务**：分发时须附带本许可证或其链接，并保留 `Required Notice` 版权声明

如需商业授权，请通过 GitHub（@uifj）联系。

根目录 `admin/` 作为运营后台随生产编排构建部署，其代码保留原始许可证，
不受本项目许可证影响。

Copyright (C) 2026 uifj
