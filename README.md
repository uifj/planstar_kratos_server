# Planstar 后端服务

企划星（Planstar）Flutter 客户端的后端 API 服务：单一 Kratos v3 服务、单一二进制，
数据落在 9 个 `planstar_*` MySQL 库。客户端契约以
[docs/api/API_CONTRACT.md](docs/api/API_CONTRACT.md) 为准。

> 仓库根目录保留的是历史 kratos-layout 模板（已归档、无 Go 包），真实服务代码全部在
> [`planstar/`](planstar) 独立模块中。CI 只构建 `planstar/`。

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
  configs/                   config.yaml / config.docker.yaml / config.prod.yaml
  deploy/                    Prometheus / Alertmanager / Loki / Promtail / Traefik / Grafana
  sql/                       认证支撑表
  docker-compose.yml         单机全容器编排
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

## 容器化运行

```bash
cd planstar
cp .env.example .env          # 按需修改口令
docker compose up -d --build
```

全栈包含 MySQL / Redis / RabbitMQ / Elasticsearch / Consul / Jaeger /
Prometheus / Alertmanager / Grafana / Loki / Promtail / Traefik。

## CI/CD（GitHub Actions）

| 工作流 | 触发 | 产物 |
|---|---|---|
| [`planstar-deploy.yml`](.github/workflows/planstar-deploy.yml) | push / PR 到 main（`planstar/**`） | PR：仅 build/vet/test；push：测试 → 镜像 `main`/`latest` → GHCR → SSH 部署 → **Deployments** |
| [`planstar-release.yml`](.github/workflows/planstar-release.yml) | push tag `v*.*.*` | 镜像 `0.1.0`/`0.1`/`v0.1.0`/`latest`（**Packages**）+ **Release** + 部署 |

发布新版本：

```bash
git tag v0.1.1
git push origin v0.1.1
```

### 仓库一次性配置

1. **Settings → Actions → General → Workflow permissions** 选择 **Read and write**
   （否则无法推送 GHCR / 创建 Release）。
2. **Settings → Environments** 新建 `production`（可加保护规则或人工审批）。
3. **Settings → Secrets and variables → Actions** 配置：
   - `SERVER_HOST`、`SERVER_USER`、`SERVER_SSH_KEY`、`SERVER_PORT`、`APP_DIR`
   - `GHCR_TOKEN`（可选，拉取私有镜像用）

未配置 `SERVER_HOST` 时，部署 job 自动跳过（Packages / Releases 仍正常生成）。

## 部署到单台服务器

由 CI 自动完成；也可在服务器上手动执行：

```bash
cd /opt/planstar
PLANSTAR_IMAGE=ghcr.io/uifj/planstar-api PLANSTAR_TAG=v0.1.0 \
  docker compose -f planstar/docker-compose.yml pull
PLANSTAR_IMAGE=ghcr.io/uifj/planstar-api PLANSTAR_TAG=v0.1.0 \
  docker compose -f planstar/docker-compose.yml up -d --remove-orphans
```

## 说明

- 本服务为手写 HTTP handler，不使用 protobuf / Wire 代码生成。
- 金额：数据库存元（`DECIMAL`），API 输出分（×100）。
- 鉴权：`Cookie: psSession=…`；登录失败 `5xx + text/plain`；发码恒 `200 + {success,needCaptcha,errorMsg}`。
