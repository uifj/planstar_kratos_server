# 企划星后端 · 服务器端代码更新与版本升级全流程

> 适用版本：planstar 生产部署（4C8G / 2M 带宽单机，Docker Compose 三文件合并）
> 最后核对：2026-11-10 · 维护：运维
> 衔接：[`first_depoly.md`](./first_depoly.md) 是**首次部署**教程；本文档负责**之后的每一次更新与升级**。
> 场景：常规 Linux 服务器 + Docker Compose v2（无需 K8s/Swarm/CI 平台等额外工具）。

---

## 0. 总览：六步法

一次更新/升级 = 下面六步（每一步都有对应命令，本文档逐节展开）：

```
0 备份   →  1 同步代码  →  2 更新依赖/配置  →  3 校验  →  4 构建部署  →  5 验证  →  6 收尾（含回滚预案）
```

按变更类型选择部署动作（下面各节对应）：

| 场景 | 变更内容 | 部署动作 | 章节 |
|---|---|---|---|
| **A 增量代码** | 只改了 Go 代码 / 前端静态资源 | 只重建有变化的服务 | §3 |
| **B 全量版本** | 新版本（含数据库结构变更） | 全量重建 + 数据库迁移 | §4 |
| **C 仅配置** | 改了 `.env.prod` / `config*.yaml` / compose 文件 | 重建或重启，**不重建镜像** | §5 |

> 先记下全文档统一使用的变量别名（服务器上每次新开会话都要重新定义）：

```bash
cd /opt/planstar_backend/planstar
export C="docker compose --env-file .env.prod \
  -f docker-compose.yml -f docker-compose.prod.yml -f docker-compose.server.yml"
```

---

## 1. 部署前检查项（Checklist）

**动手前逐项确认，任何一项不过就停下来：**

| # | 检查项 | 命令/方法 |
|---|---|---|
| 1 | 本地代码可编译 | `cd planstar_backend && go build ./... && go vet ./...`（或 `make check`） |
| 2 | 服务器磁盘空间足够（镜像构建） | `df -h`（建议剩余 ≥ 5G） |
| 3 | 服务器内存 | `free -h`（剩余 ≥ 2G） |
| 4 | `.env.prod` 存在且权限 600 | `ls -l planstar_backend/.env.prod`（应为 `-rw-------`） |
| 5 | `.env.prod` 与本地一致（改过配置时） | 本地 `diff` 或直接按 §5 重新上传 |
| 6 | `ops_token` 与 `PLANSTAR_OPS_TOKEN` 一致 | `cmp planstar_backend/deploy/prometheus/ops_token <(printf '%s' "$(grep '^PLANSTAR_OPS_TOKEN=' planstar_backend/.env.prod | cut -d= -f2-)")` |
| 7 | 发布前门禁 | `export ENV_FILE=.env.prod COMPOSE_DIR=$PWD PLANSTAR_CONFIG_FILE=config.lite.yaml && ./deploy/preflight.sh` |
| 8 | compose 合并可解析 | `$C config -q`（无输出 = 通过） |
| 9 | **备份就绪（必做）** | `./deploy/backup.sh pre-deploy`（产出 `mysql.sql.gz` + `env.backup`） |
| 10 | DNS / 证书状态 | `curl -I https://api.planstar.top`（返回 200 且非证书过期）；证书在 Traefik `/acme`，到期自动续 |
| 11 | 时机 | 避开业务高峰；2M 带宽下镜像构建可能 10~30 分钟，确认窗口足够 |

> 第 9 项**不要跳过**。`backup.sh` 会把 11 个库全量 mysqldump 到
> `/opt/planstar_backend/backups/pre-deploy/<时间戳>/`，保留 7 天自动轮转。

---

## 2. 同步代码（两种方式，任选其一）

### 方式 A：git 拉取（推荐，长期维护用这个）

首次在服务器上克隆一次（需要 GitHub 认证；推荐 SSH key）：

```bash
# SSH 方式（服务器 ~/.ssh 配好 GitHub key）
git clone git@github.com:uifj/planstar_kratos_server.git /opt/planstar
# 或 HTTPS 方式（首次会让你输 GitHub 账号密码/Token）
git clone https://github.com/uifj/planstar_kratos_server.git /opt/planstar
```

日常更新：

```bash
cd /opt/planstar && git pull --ff-only
```

- `--ff-only`：远端有变但本地没同步时直接失败，避免出现"半套配置"。
- `.env.prod` / `ops_token` 被 gitignore，pull **不会**覆盖它们（安全）。

### 方式 B：本地打包上传（无 GitHub 认证时的备选）

本地（Windows）打包时**排除大目录和本地环境文件**，上传后覆盖式解压：

```powershell
# 本地打包
tar -czf planstar-update.tar.gz --exclude='.git' --exclude='node_modules' `
  --exclude='dist' --exclude='.umi' --exclude='.umi-production' --exclude='*.log' `
  --exclude='planstar_backend/.env' --exclude='planstar_backend/.env.local' `
  -C d:\workspaces\flutter_projects planstar_kratos_server
# 上传
scp planstar-update.tar.gz root@<服务器IP>:/opt/
```

服务器解压（**覆盖式更新**，不会动已有文件）：

```bash
tar -xzf /opt/planstar-update.tar.gz -C /opt/planstar --strip-components=1
rm -f /opt/planstar-update.tar.gz   # 包内不含密钥，但别留垃圾
```

⚠️ **方式 B 铁律**：
- 解压是"覆盖/新增"，不会删除服务器上已有的 `.env.prod`、`ops_token`、`backups/`。
- **绝不要先 `rm -rf /opt/planstar` 再解压**——会连带删掉 `.env.prod`、`ops_token` 和备份。确实要全新替换时，先把这三个先备份走（`cp planstar_backend/.env.prod ~/env.prod.bak` 等）。

---

## 3. 场景 A：增量代码更新（最常见）

只改了 Go 业务代码 / admin 控制台 / 部署脚本，数据库结构没变。

```bash
cd /opt/planstar_backend/planstar
export C="docker compose --env-file .env.prod \
  -f docker-compose.yml -f docker-compose.prod.yml -f docker-compose.server.yml"

# 1) 同步代码（§2 任选一种）
# 2) 校验（§1 的 7/8 项）
export ENV_FILE=.env.prod COMPOSE_DIR=$PWD PLANSTAR_CONFIG_FILE=config.lite.yaml
./deploy/preflight.sh && $C config -q

# 3) 只重建有变化的服务（--build 触发重新构建镜像）
#    Go 代码改动 → planstar-api；后台改动 → admin（+ admin-web，若控制台也改了）
$C up -d --build planstar-api
#    （若 admin 也改：$C up -d --build planstar-api admin admin-web）

# 4) 验证（§7）
```

**要点**：
- **不要全量 `up -d --build`**：mysql / redis / traefik / prometheus / grafana 的镜像没变，
  全量重建纯属浪费时间（尤其 2M 带宽 + admin-web 的 npm 构建很慢）。
- `up -d --build planstar-api` 只会重建并重启这一个服务，其余容器不受影响。
- 构建失败时**运行中的旧容器不受影响**（build 阶段不动容器）——这是 Docker 的优势，放心构建。

---

## 4. 场景 B：全量版本替换（新版本 / 含数据库变更）

新功能上线、或伴随数据库结构变更时，走完整流程。

### 4.1 版本号管理

`.env.prod` 里两处要一起改（`preflight` 强制语义化版本 `^v\d+.\d+.\d+`）：

```bash
# planstar_backend/.env.prod
PLANSTAR_TAG=v1.1.0    # 例：从 v1.0.0 升到 v1.1.0
VERSION=v1.1.0
```

改后 `chmod 600 .env.prod`。旧版本的镜像会留在本地（`docker images` 可见），这是回滚的资本。

### 4.2 流程

```bash
cd /opt/planstar_backend/planstar
export C="docker compose --env-file .env.prod \
  -f docker-compose.yml -f docker-compose.prod.yml -f docker-compose.server.yml"

# 0) 备份（必做）
./deploy/backup.sh pre-deploy

# 1) 同步代码（§2）

# 2) 【关键】数据库迁移 —— 必须先于服务重启执行！
#    原因：MySQL 初始化脚本只在"数据卷为空"时执行；存量库升级必须手工补 SQL。
#    a) 看本次版本带了哪些新迁移文件（planstar_backend/sql/）
ls -l sql/ | tail
#    b) 手工执行本次新增的迁移（例：23_xxx.sql）——用 root 密码
$C exec -T mysql mysql -uroot -p'<MYSQL_ROOT_PASSWORD>' < ./sql/23_xxx.sql
#    ⚠️ 生产环境【禁止】执行 14_mock_seed.sql（演示数据）！
#    ⚠️ 不推荐整包执行 `make sql-apply`：它会连演示种子一起灌，只适合本地。
#    c) 若只是幂等补丁，可重复执行；建表语句带 IF NOT EXISTS 的可安全重放。

# 3) 更新 .env.prod 版本号 + 校验（§1 7/8/9 项）
./deploy/preflight.sh && $C config -q

# 4) 全量重建并启动
$C up -d --build

# 5) 验证（§7）+ 收尾（§6.5 记录发布）
```

**顺序铁律**：**先迁移数据库 → 再重启应用**。新代码 + 旧表结构 = 接口 500；
先迁移再起新代码才是安全的。若迁移失败，**不要继续启动新版本**，直接进入 §8 回滚。

---

## 5. 场景 C：配置文件变更（不重建镜像）

区分"改了什么"决定用 `restart` 还是 `up -d`（重建容器）：

| 变更对象 | 生效方式 | 命令 |
|---|---|---|
| `configs/config.lite.yaml`（应用配置，容器内 bind-mount） | 改宿主机文件后**重启**即可 | `$C restart planstar-api` |
| `.env.prod`（环境变量，容器启动时注入） | 环境变量只在新容器生效 → **重建** | `$C up -d planstar-api` |
| compose 文件（`*.yml` 三件套） | 配置变更 → **重建** | `$C up -d` |
| `deploy/traefik/traefik.yml` / `dynamic.yml` | 静态配置 → 重启 traefik | `$C restart traefik` |
| `PLANSTAR_OPS_TOKEN` 变更 | **必须同步落盘** ops_token，否则 Prometheus 抓不到指标 | 见下 |

**ops_token 同步（改 token 时必做）**：

```bash
printf '%s' "$(grep '^PLANSTAR_OPS_TOKEN=' .env.prod | cut -d= -f2-)" \
  > deploy/prometheus/ops_token
chmod 600 deploy/prometheus/ops_token
```

**要点**：
- 应用配置用 `restart` 足够（文件是 bind-mount，进程重启即读新内容）；环境变量 / compose 结构变更必须 `up -d`（重建容器）。
- 改 `.env.prod` 记得 `chmod 600`；`.env.prod` 不在 git 里，改完两边（本地与服务器）保持一致，可用 scp 单向覆盖。

---

## 6. 部署中注意事项（防中断 / 防数据丢失）

1. **数据卷是命根子**：`docker compose down` 会停容器但**保留数据**；**严禁** `down -v` 或 `docker volume rm`（会删 MySQL/Redis 数据）。这条写进操作红线。
2. **先 build 后 up**：构建失败不影响运行中的容器；确认 build 成功再执行 `up`。
3. **中断窗口**：`up -d` 重建容器时有数秒停机（停旧起新）。单机无滚动，**避开业务高峰**执行。
4. **迁移先行**：凡涉及数据库结构的版本，先迁移再起服务（§4.2 顺序）。
5. **2M 带宽构建慢**：大镜像（golang:1.26.5、node:22）下载可能 10~30 分钟；保持终端会话不中断，必要时 `nohup ... &` 后台跑并 tail 日志。
6. **别在构建中途 Ctrl+C** 后直接再跑 `up -d --build`（会二次下载）；先确认 build 完。
7. **磁盘水位**：`df -h`；镜像 + 备份 + 日志（`/var/lib/docker`、`/opt/planstar_backend/backups`）是主要消耗，`backup.sh` 自动保留 7 天防爆盘。

---

## 7. 部署完成后的验证（每步都做）

```bash
# 1) 所有容器 Up 且无 Restarting
$C ps

# 2) 健康与就绪探针
curl -fsS http://127.0.0.1:9108/healthz          # {"status":"ok"}
curl -fsS http://127.0.0.1:9108/readyz           # 依赖就绪 → {"status":"ready"}

# 3) 业务接口冒烟
curl -fsS 'http://127.0.0.1:8098/v1/activities?pageIndex=1&pageSize=1'

# 4) 登录/发码回归（用测试手机号，确认 5xx 与否）
curl -s -X POST http://127.0.0.1:8098/v1/user/send-code \
  -H 'Content-Type: application/json' -d '{"mobile":"你的手机号"}'
# 预期：非 5xx（发码恒 200 + {success,...}）

# 5) 日志无异常
$C logs --tail=50 planstar-api | grep -iE 'error|panic' || echo "无异常日志"

# 6) HTTPS 入口（ICP 备案完成后）
curl -I https://api.planstar.top
```

任一验证不过 → 看日志定位；判断为版本问题 → 直接进 §8 回滚。

---

## 8. 快速回滚操作指引

按影响从小到大排列，**先回滚配置/镜像，数据回滚是最后手段**：

### 8.1 代码回滚（最常见）

```bash
# git 方式：切回上一个 tag
cd /opt/planstar && git checkout <上一版本tag> && cd planstar_backend
export C="docker compose --env-file .env.prod \
  -f docker-compose.yml -f docker-compose.prod.yml -f docker-compose.server.yml"
$C up -d --build planstar-api
```

### 8.2 镜像回滚（代码没改、镜像 tag 不对时）

```bash
# 查看本地有哪些版本镜像
docker images | grep planstar-api
# 把 .env.prod 的 PLANSTAR_TAG 改回旧 tag，然后重建（用旧镜像，很快）
$C up -d planstar-api
```

### 8.3 配置回滚

```bash
# .env.prod 用备份里的 env.backup 覆盖（backup.sh pre-deploy 产物）
cp /opt/planstar_backend/backups/pre-deploy/<时间戳>/env.backup planstar_backend/.env.prod
chmod 600 planstar_backend/.env.prod
# config 文件同理（若有备份）→ 恢复后
$C up -d planstar-api
```

### 8.4 数据回滚（最后手段，谨慎）

只有部署导致**数据写坏**且必须恢复时才用。恢复会**覆盖当前数据**，先确认根因：

```bash
# 用发布前快照恢复 11 个库
zcat /opt/planstar_backend/backups/pre-deploy/<时间戳>/mysql.sql.gz \
  | $C exec -T mysql mysql -uroot -p'<MYSQL_ROOT_PASSWORD>'
# 恢复后重启应用并完整验证（§7）
$C restart planstar-api admin
```

### 8.5 回滚后必做

- 重新执行 §7 全套验证；
- 把失败的版本号与原因记录到发布日志（§9）；
- 修复问题后再走正常的 §4 流程发布，不要直接在回滚状态上叠新版本。

---

## 9. 生产部署最佳实践清单

1. **语义化版本 + 每版一个 tag**：`v1.0.0 → v1.1.0`；`PLANSTAR_TAG` 与 `VERSION` 同步改。
2. **每次发布前 `./deploy/backup.sh pre-deploy`**（自动 7 天轮转）。
3. **配置每日备份**：服务器 cron 加一条 `0 3 * * * cd /opt/planstar_backend/planstar && COMPOSE_DIR=/opt/planstar_backend/planstar ENV_FILE=.env.prod ./deploy/backup.sh daily`。
4. **小步快跑**：频繁小版本优于一次大版本；每版变更集中在 §4.1 的 tag 上，回滚点清晰。
5. **发布留痕**：每次发布记 `版本号 + 时间 + 变更摘要`（文件或运维笔记）。
6. **密钥纪律**：`.env.prod` 权限 600、不进 git、随备份归档；改密钥只改一处后全链路核对（§1 第 6 项）。
7. **变更前看清单**：`git log --stat -1` 或 `git diff HEAD~1 --stat`，确认本次动了哪些文件，判断是 §3/§4/§5 哪种场景。
8. **失败不硬扛**：验证不过或迁移失败，第一时间回滚（§8），排查清楚再发。

---

## 附录：常用命令速查

```bash
# 会话别名（每次新开会话执行）
cd /opt/planstar_backend/planstar
export C="docker compose --env-file .env.prod \
  -f docker-compose.yml -f docker-compose.prod.yml -f docker-compose.server.yml"

# 发布门禁
export ENV_FILE=.env.prod COMPOSE_DIR=$PWD PLANSTAR_CONFIG_FILE=config.lite.yaml
./deploy/preflight.sh

# 备份（发布前）
./deploy/backup.sh pre-deploy

# 一键部署（含 ops_token 落盘 + server.yml 自动叠加，等价于手动三文件流程）
./up-prod.sh

# 常用操作
$C ps                     # 状态
$C logs -f planstar-api   # 实时日志
$C restart planstar-api   # 重启（配置变更用）
$C up -d --build planstar-api   # 重建（代码变更用）
$C down                   # 停容器（保留数据；严禁 -v）
docker images | grep planstar-api   # 本地镜像（回滚依据）
```
