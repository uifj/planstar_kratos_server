> 适用版本：planstar 单体 · 最后核对：2026-11-07 · 维护：后端组

# 部署手册（planstar 单体）

## 1. 本地一键启动

前置：Docker Desktop / Docker Engine + Compose v2，仓库完整克隆（compose 通过 `../docs/sql` 挂载建库脚本，不能只拷贝 `planstar/` 目录）。

```bash
cd planstar
cp .env.example .env
docker compose up -d --build
```

PowerShell 等价：

```powershell
cd planstar
Copy-Item .env.example .env
docker compose up -d --build
```

启动后：

| 地址 | 用途 |
| --- | --- |
| `http://127.0.0.1:8098` | 业务 API |
| `http://127.0.0.1:9108/healthz` | 存活探针 |
| `http://127.0.0.1:9108/readyz` | 就绪探针（MySQL/Redis/ES） |
| `http://127.0.0.1:9108/metrics` | Prometheus 指标 |
| `http://127.0.0.1:9090` | Prometheus |
| `http://127.0.0.1:3000` | Grafana（默认 `admin/admin`，见 `.env`） |
| `http://127.0.0.1:8500` | Consul UI |
| `http://127.0.0.1:15672` | RabbitMQ 管理台（`root/root`） |
| `http://127.0.0.1:16686` | Jaeger UI |
| `http://127.0.0.1:8080` | Traefik Dashboard |

### 验证命令

```bash
curl -fsS http://127.0.0.1:9108/healthz
# {"status":"ok"}

curl -fsS http://127.0.0.1:9108/readyz
# {"status":"ready"}；依赖不可用时 HTTP 503 + {"status":"unavailable"}

curl -fsS http://127.0.0.1:9108/metrics | head
# planstar_http_requests_total / planstar_http_request_duration_seconds / go_* / process_*

curl -D captcha.headers -o captcha.png http://127.0.0.1:8098/v1/user/captcha
# 200 image/png，并下发 psCaptcha Cookie
```

演示账号（来自 `planstar/sql/15_auth_support.sql`）：`13800000001` / `13800000002`，密码 `123456`。

```bash
curl -i -X POST http://127.0.0.1:8098/v1/user/login \
  -H 'Content-Type: application/json' \
  -d '{"ua":"13800000001","pw":"123456","ct":11}'
# 成功返回 LoginBean，并下发 Set-Cookie: psSession=...
```

## 2. 生产部署

### 2.1 单机 Compose（当前支持的生产形态）

服务器一次性准备：

1. 安装 Docker + Compose v2。
2. 将仓库克隆到固定目录（默认 `/opt/planstar`），保持 `docs/sql` 与 `planstar/` 的相对路径。
3. 如拉取私有 GHCR 镜像，执行 `docker login ghcr.io`。
4. 配置 SSH 免密（供 `deploy.sh` / CI 使用）。

服务器侧：

```bash
cd /opt/planstar/planstar
cp .env.example .env
# 编辑 .env：数据库/Redis/RabbitMQ 口令、PLANSTAR_IMAGE、PLANSTAR_TAG、GRAFANA 口令、ACME_EMAIL
docker compose pull
docker compose up -d --remove-orphans
```

`planstar/docker-compose.yml` 使用 `configs/config.prod.yaml`，并挂载为容器内 `/data/conf/config.yaml`；敏感口令通过 `PLANSTAR_MYSQL_PASSWORD`、`PLANSTAR_REDIS_PASSWORD`、`PLANSTAR_MQ_ADDR` 注入，不写入配置文件。

### 2.2 一键推送部署

`planstar/deploy.sh`（也可由 CI 执行）流程：

1. 本地 `docker build` 生成 `${PLANSTAR_IMAGE}:${PLANSTAR_TAG}`。
2. `docker push` 推送到镜像仓库。
3. SSH 到服务器执行 `git pull --ff-only`。
4. 远程 `docker compose -f planstar/docker-compose.yml pull` + `up -d --remove-orphans`。

使用前在 `planstar/.env` 填写：`SERVER_HOST`、`SERVER_USER`、`SERVER_PORT`、`APP_DIR`、`PLANSTAR_IMAGE`、`PLANSTAR_TAG`。

### 2.3 镜像发布（GitHub Actions）

- 推送 tag `v*.*.*` 触发 `Release` 工作流：构建 `ghcr.io/<owner>/<repo>/planstar-api`，生成 semver 标签与 `latest`，创建 GitHub Release；配置 `SERVER_HOST` 等 secrets 后自动 SSH 部署。
- 手动构建：

```bash
cd planstar
docker build --build-arg VERSION=v0.1.0 -t ghcr.io/uifj/planstar-api:v0.1.0 .
docker push ghcr.io/uifj/planstar-api:v0.1.0
```

镜像基于 `golang:1.26.5` 多阶段构建，运行层为 `debian:stable-slim`，非 root 用户，暴露 `8098/9108`，内置 `curl` 健康检查。

## 3. 回滚

### 3.1 应用回滚（推荐）

```bash
cd /opt/planstar
export PLANSTAR_IMAGE=ghcr.io/uifj/planstar-api
export PLANSTAR_TAG=v0.1.0          # 上一个已验证版本
docker compose -f planstar/docker-compose.yml pull planstar-api
docker compose -f planstar/docker-compose.yml up -d --remove-orphans planstar-api
docker compose -f planstar/docker-compose.yml ps
```

回滚后立即验证 `/healthz`、`/readyz` 与一条只读业务接口（如 `GET /v1/home/banners`）。

### 3.2 数据回滚

- 应用回滚不自动回滚数据库；如新版本执行过 DDL/DML，先用备份恢复（见第 4 节）。
- `docs/sql/90_teardown.sql` 会**逆序删除 9 个 `planstar_*` 库**，仅用于测试环境，生产禁止直接执行。
- DDL 隐式提交，不能用事务回滚。

## 4. 备份与恢复

### 4.1 MySQL 逻辑备份

仓库根目录的 `scripts/backup.sh` / `scripts/restore.sh` 是 kratos-shop 模板遗留，操作 **PostgreSQL `shop_*`** 库，**不可用于当前服务**。请使用以下命令：

```bash
cd /opt/planstar/planstar
set -a; . ./.env; set +a

# 备份 9 个逻辑库
docker compose exec -T mysql \
  mysqldump -uroot -p"$MYSQL_ROOT_PASSWORD" \
  --single-transaction --routines --triggers --default-character-set=utf8mb4 \
  --databases planstar_user planstar_catalog planstar_ticket planstar_order \
              planstar_payment planstar_marketing planstar_community \
              planstar_content planstar_meta \
  > planstar-$(date +%Y%m%d-%H%M%S).sql
```

建议每日凌晨执行，保留 7~30 天，并定期在独立实例做恢复演练。

### 4.2 恢复

```bash
cd /opt/planstar/planstar
set -a; . ./.env; set +a

# 停业务写入
docker compose stop planstar-api

# 恢复到现有 MySQL（dump 内含 CREATE DATABASE IF NOT EXISTS；如存在同名表需先清理目标库）
docker compose exec -T mysql \
  mysql -uroot -p"$MYSQL_ROOT_PASSWORD" --default-character-set=utf8mb4 \
  < planstar-20261107-020000.sql

docker compose start planstar-api
curl -fsS http://127.0.0.1:9108/readyz
```

### 4.3 其他数据

| 数据 | 是否需备份 | 说明 |
| --- | --- | --- |
| MySQL 业务库 | **必须** | 唯一业务事实源 |
| Redis | 否 | 会话/验证码/限流短期状态，丢失后用户重新登录即可 |
| Elasticsearch | 否 | 读模型，服务重启后会重新同步 `planstar-activities` |
| RabbitMQ | 视情况 | 业务事件短期队列；DLQ 中可能有未处理消息，排障时保留 |
| Docker volumes | 可选 | `mysql_data` 是核心；其余可重建 |

## 5. 配置项（对应 `planstar/configs/config.prod.yaml`）

| 配置键 | 生产值 | 说明 |
| --- | --- | --- |
| `server.env` | `prod` | 生产环境；未显式配置 `auth.cookie_secure` 时自动启用 Secure Cookie |
| `server.http.addr` | `0.0.0.0:8098` | 业务监听 |
| `server.http.read_timeout_seconds` | `10` | 连接读超时 |
| `server.http.write_timeout_seconds` | `30` | 连接写超时 |
| `server.http.idle_timeout_seconds` | `60` | 空闲连接超时 |
| `server.http.request_timeout_seconds` | `15` | 单请求超时，超时返回 504 |
| `server.http.max_body_bytes` | `1048576` | 请求体上限 1 MiB；`/v1/upload` 内部放宽到 20 MiB |
| `server.http.cors_allow_origins` | `[]` | 跨域白名单；空数组仅允许同源 |
| `server.admin.addr` | `0.0.0.0:9108` | 运维监听，勿暴露公网 |
| `data.mysql.addr` | `mysql:3306` | Compose 服务名 |
| `data.mysql.user` | `root` | 生产建议改为最小权限账号 |
| `data.mysql.password` | 空 | 由 `PLANSTAR_MYSQL_PASSWORD` 覆盖 |
| `data.mysql.charset` | `utf8mb4` | 默认值 |
| `data.redis.addr` | `redis:6379` | 未配置则退化为进程内缓存 |
| `data.redis.password` | 空 | 由 `PLANSTAR_REDIS_PASSWORD` 覆盖 |
| `data.redis.db` | `0` | |
| `data.elastic.addr` | `http://elasticsearch:9200` | 未配置则热门榜直接走 MySQL |
| `data.elastic.index_prefix` | `planstar` | 索引名 `<prefix>-activities` |
| `data.elastic.sync_interval_seconds` | `300` | 全量同步周期 |
| `data.elastic.request_timeout_seconds` | `3` | ES 查询超时 |
| `auth.session_expire_days` | `30` | 服务端会话有效期 |
| `auth.cookie_secure` | `true` | `psSession` 的 Secure 属性 |
| `auth.send_code_cooldown_seconds` | `60` | 同一手机号发码冷却 |
| `auth.send_code_ip_limit_per_minute` | `10` | 同一 IP 每分钟发码上限（取 TCP 对端 IP） |
| `auth.login_fail_captcha_threshold` | `3` | 连续登录失败后要求图形码 |
| `trace.endpoint` | `http://jaeger:14268/api/traces` | 留空关闭 tracing |
| `trace.env` | `prod` | 环境标签 |
| `trace.sample_ratio` | `0.1` | 10% 采样 |
| `mq.addr` | `amqp://rabbitmq:5672/` | 由 `PLANSTAR_MQ_ADDR` 覆盖为带账号密码的完整 URL |
| `mq.exchange` | `planstar.exchange` | topic exchange |
| `mq.queue` | `planstar.api.events` | 消费队列 |
| `registry.consul.address` | `consul:8500` | 留空则不注册 |
| `registry.consul.scheme` | `http` | |
| `registry.consul.service_host` | `planstar-api` | 注册到 Consul 的可达主机名 |

`.env` 中除上述覆盖项外，还包含 `MYSQL_ROOT_PASSWORD`、`REDIS_PASSWORD`、`RABBITMQ_USER`、`RABBITMQ_PASS`、`PLANSTAR_IMAGE`、`PLANSTAR_TAG`、`VERSION`、Grafana 账号、`ACME_EMAIL`、`SERVER_*`、`APP_DIR`。报价快照有效期可用 `PLANSTAR_QUOTE_TTL_SECONDS` 覆盖（默认 900 秒，见 `internal/biz/quote.go`）。

## 6. 常见启动失败排查

| 现象 | 原因 | 处理 |
| --- | --- | --- |
| `bind: address already in use` | 本机已有 MySQL/Redis/RabbitMQ/ES/Consul 或 8098/9108 占用 | `docker ps` 找到占用容器/进程并停止，或改 compose 端口映射 |
| `no such file or directory: ../docs/sql/00_all.sql` | 只在 `planstar/` 目录单独部署，缺少仓库 `docs/` | 从仓库根完整部署；保持 `planstar/` 与 `docs/` 同级 |
| MySQL 初始化脚本未执行 | 脚本只在**数据卷首次创建**时执行 | 确认 `mysql_data` 为空；如需重建：`docker compose down -v`（**删除全部数据**）后重新 `up` |
| 改了 `MYSQL_ROOT_PASSWORD` 后 `Access denied` | 已有数据卷仍使用旧密码 | 改回旧密码启动，或进入容器改密码；不要直接依赖 `.env` 覆盖已初始化实例 |
| `planstar-api` 反复重启，日志 `init elasticsearch ...` | ES 未就绪或地址不可达；`NewData` 启动即检查 MySQL/Redis/ES | `docker compose logs elasticsearch`；等待健康检查通过；确认 `config.prod.yaml` 地址 |
| Elasticsearch 容器退出码 78 / `max virtual memory areas` | Linux 宿主机 `vm.max_map_count` 过小 | `sysctl -w vm.max_map_count=262144` 并写入 `/etc/sysctl.conf` |
| Elasticsearch OOM | 默认堆 512m，数据量大或内存不足 | 调大 `ES_JAVA_OPTS`，并保证宿主机内存充足 |
| `/readyz` 返回 503 | MySQL、Redis 或 ES 健康检查失败 | 分别 `docker compose logs mysql/redis/elasticsearch`；`/healthz` 只代表进程存活 |
| Consul 注册失败导致启动退出 | `registry.consul.address` 指向不可达 Consul，kratos registrar 注册失败 | 恢复 Consul；或清空 `registry.consul.address` 关闭注册 |
| RabbitMQ 健康检查长时间未通过 | 首次启动初始化较慢 | 等待 1~2 分钟；查看 `docker compose logs rabbitmq` |
| `planstar-api` 日志 `open /data/conf/config.yaml: no such file` | compose 挂载路径错误/未在 `planstar/` 目录执行 | 确认在 `planstar/` 下执行，且 `configs/config.prod.yaml` 存在 |
| Traefik 启动失败（80/443 被占用） | 宿主机已有 Nginx/Apache | 停止占用服务，或移除 Traefik 端口映射 |
| 修改 `.env` 后配置未生效 | Compose 环境变量在容器重建时注入 | `docker compose up -d --force-recreate planstar-api` |
