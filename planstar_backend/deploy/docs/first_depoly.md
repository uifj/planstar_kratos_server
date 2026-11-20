# 企划星后端 · 首次部署指南（4C8G 单机 / Docker Compose）

> 适用：planstar 生产首次上线（4C8G / 2M 带宽单机，Docker Compose 三文件合并）
> 最后核对：2026-11-20 · 维护：运维
> 后续每次更新/升级见 [`update-upgrade-guide.md`](./update-upgrade-guide.md)。

---

## 0. 上线前提（当前状态）

| 项 | 状态 |
|---|---|
| 域名 / ICP | `planstar.top` 已完成 **ICP 备案**；`api.`、`admin.`、`www.`（官网）三个子域走 HTTPS（不启用裸域） |
| 证书 | Traefik + Let's Encrypt（火山引擎 DNS Challenge）自动签发、自动续期 |
| 生产档位 | `config.lite.yaml`：4C8G，关闭 ES / MQ / Consul / tracing，搜索降级 MySQL |
| 端口暴露 | 业务端口 8098 与运维端口 9108 仅绑 `127.0.0.1`；公网只开放 Traefik 的 `80`/`443` |
| 会话安全 | `auth.cookie_secure=true` + `allow_insecure_cookie=false`，psSession 仅经 HTTPS 传输，**不再有公网 IP 明文直连** |
| 对象存储 | **火山引擎 TOS**，provider=`tos`（官方 SDK `ve-tos-golang-sdk/v2`），运行时启用 |
| 短信 | 阿里云 dypnsapi，签名「恒创联众」，模板 `10001`（登录/注册） |
| 数据库 | MySQL 首启自动建库建表 + 生产化补丁 + 禁用测试账号 + 创建最小权限账号（共 11 个 SQL/SH 按字典序执行） |

---

## 1. 上线前必须补全 `.env.prod`

`.env.prod` 已配置完成：**口令为固定强口令（2026-11-20 起不再每次随机生成）**，与服务器上运行中的数据库/Redis 一致；
TOS 的 AK/SK、短信、TLS 均已填好，无需再改。重新打包会原样带上本文件。

> 如需轮换口令，必须同步修改存量数据库/Redis 内的账号口令，否则 api 会 `Access denied`。

### 1.1 对象存储 TOS 配置

| 变量 | 值 / 示例 | 说明 |
|---|---|---|
| `PLANSTAR_STORAGE_OSS_ENABLED` | `true` | 启用对象存储 |
| `PLANSTAR_STORAGE_OSS_PROVIDER` | `tos` | 走火山引擎 TOS |
| `PLANSTAR_STORAGE_OSS_ENDPOINT` | `tos-cn-beijing.volces.com` | 地域端点，**不含 `https://` 前缀** |
| `PLANSTAR_STORAGE_OSS_REGION` | `cn-beijing` | 地域；provider=tos 时**必填** |
| `PLANSTAR_STORAGE_OSS_BUCKET` | `planstaross` | 桶名 |
| `PLANSTAR_STORAGE_OSS_ACCESS_KEY` / `_ACCESS_SECRET` | 火山引擎访问密钥 AK/SK | **必填**，见下方注意 |
| `PLANSTAR_STORAGE_OSS_CDN_HOST` | （空） | 可选；填了则返回 CDN 域名直链，不填则返回预签名 URL |
| `PLANSTAR_STORAGE_OSS_PRESIGN_TTL_SECONDS` | `900` | 预签名 URL 有效期（秒） |

> 现有 `.env.prod` 的 endpoint / region / bucket 已按 TOS 华北2（北京）填好；
> 只需把 `PLANSTAR_STORAGE_OSS_ACCESS_KEY` / `_ACCESS_SECRET` 的 `<REPLACE_ME>` 换成真实值。
> **必须使用火山引擎控制台「访问密钥(AccessKey)」的 AK/SK，不能复用阿里云的 `LTAI...` 密钥**（两套密钥体系不互通）。
> 桶不在北京时，同步改 endpoint / region / bucket 三项。
> 改完执行 `chmod 600 .env.prod`；preflight 会拦下残留的 `<REPLACE_ME>`。

### 1.2 短信模板

代码当前只实现单一模板（`biz/auth.go` 的 `SendLoginCode`），因此只配 `PLANSTAR_NOTIFY_SMS_TEMPLATE_CODE=10001`。
100002~100005（改绑 / 重置 / 绑新 / 验证新手机）对应后端尚未实现的流程，后续扩展时再加。

### 1.3 密钥安全

阿里云短信与 TOS 的 AK/SK 都已在聊天中出现过，建议上线后分别到阿里云 RAM、火山引擎控制台创建**仅授权对应产品的最小权限子账号**替换。

### 1.4 运营域 JWT 密钥（缺失会导致 api 启动失败）

`planstar-api` 在 `server.env=prod` 下启动时会校验 `security.jwt.secret` 长度，缺失或短于 32 字节会直接退出并报：

```
生产配置校验失败（server.env=prod）: security.jwt.secret 至少 32 字节（PLANSTAR_SECURITY_JWT_SECRET）
```

| 变量 | 说明 |
|---|---|
| `PLANSTAR_SECURITY_JWT_SECRET` | api 运营域 JWT 签名密钥，**≥32 字节**；由 `openssl rand -base64 36` 生成 |
| `ADMIN_AUTH_JWT_SECRET` | admin 后台独立密钥，两者不可共用 |

compose 已把该变量注入 `planstar-api`，preflight 也会校验其非空；`.env.prod` 中已生成，无需手工填写。

---

## 2. 打包与上传

### 2.1 本机（Windows）打包

```powershell
tar -czf planstar-deploy.tar.gz `
  --exclude='.git' --exclude='.archive' --exclude='node_modules' `
  --exclude='.next' --exclude='.turbo' --exclude='bin' `
  --exclude='dist' --exclude='.umi' --exclude='.umi-production' --exclude='*.log' `
  --exclude='.env' --exclude='.env.local' `
  -C d:\workspaces\flutter_projects\planstar_app planstar_kratos_server
```

> 说明：`planstar_web`（官网 Next.js）**参与** compose 构建（服务 `web`），源码必须打包；
> 只排除 `.next`/`.turbo`/`node_modules` 等本地产物与 `.archive` 历史归档，包约 13MB（含 `.env.prod`；
> 大头是官网 `public/fonts` 的两个 woff2 字体，共约 9MB，无法再压）。

> 包内**包含** `.env.prod`（含真实密钥），上传解包后请立即删除压缩包。

### 2.2 上传到服务器

本机执行（`<ECS_USER>` / `<ECS_IP>` 换成 SSH 账号与公网 IP）：

```powershell
scp planstar-deploy.tar.gz <ECS_USER>@<ECS_IP>:/opt/
```

### 2.3 服务器解包

```bash
mkdir -p /opt/planstar

# 若服务器上已有 .env.prod（尤其已按存量库口令调整过），先备份，解包后还原，
# 否则包内的 .env.prod 会覆盖它，导致 api 连不上旧数据库。
[ -f /opt/planstar/planstar_backend/.env.prod ] && \
  cp /opt/planstar/planstar_backend/.env.prod /root/env.prod.keep

tar -xzf /opt/planstar-deploy.tar.gz -C /opt/planstar --strip-components=1
rm -f /opt/planstar-deploy.tar.gz        # 含真实密钥，解包后立即删除

[ -f /root/env.prod.keep ] && \
  cp /root/env.prod.keep /opt/planstar/planstar_backend/.env.prod

# compose 中 admin/admin-web 的构建上下文是 ../admin，需指到 planstar_frontend/admin
[ -e /opt/planstar/admin ] || ln -s planstar_frontend/admin /opt/planstar/admin

# Windows 打的 tar 不带可执行位：脚本与 mysql initdb 的 .sh 必须是 755，否则会被跳过
chmod +x /opt/planstar/planstar_backend/*.sh \
         /opt/planstar/planstar_backend/deploy/*.sh \
         /opt/planstar/planstar_backend/sql/*.sh

cd /opt/planstar/planstar_backend
chmod 600 .env.prod
ls -l /opt/planstar/admin/Dockerfile     # 应为存在的文件，否则 admin 构建会失败
```

---

## 3. 部署前准备

### 3.1（强烈建议）配置 Docker 镜像加速

国内 ECS 拉 Docker Hub 镜像（golang / mysql / node 等）很慢：

```bash
sudo mkdir -p /etc/docker
sudo tee /etc/docker/daemon.json <<'EOF'
{
  "registry-mirrors": [
    "https://<你的阿里云加速器地址>.mirror.aliyuncs.com"
  ]
}
EOF
sudo systemctl restart docker
```

> 加速器地址在阿里云控制台「容器镜像服务 → 镜像加速器」获取；不配也能部署，只是构建慢。
> 需要从 Docker Hub 拉取的只有基础/中间件镜像：`golang:1.26.5`、`debian:stable-slim`、
> `node:22-alpine`、`nginx:alpine`、`mysql:8.4`、`redis:7-alpine`、`traefik:v3.6`、
> `prom/*`、`grafana/grafana`、`oliver006/redis_exporter`；业务四个镜像（`ghcr.io/uifj/*`，
> 含官网 `planstar-web`）由 `--build` 本地构建，无需拉取 ghcr。
>
> 官网镜像构建时 `pnpm install` 需访问 npm registry，首次构建耗时较长（约 3~8 分钟）。

### 3.2 域名解析 + 安全组

- **DNS 三条 A 记录**：`api.planstar.top`、`admin.planstar.top`、`www.planstar.top` → ECS 公网 IP（Traefik 路由写死这些域名，见 [dynamic.yml](file:///d:/workspaces/flutter_projects/planstar_app/planstar_kratos_server/planstar_backend/deploy/traefik/dynamic.yml)）
- **安全组**：放行 `80`、`443`；其余端口不对外开放（内部服务均绑 127.0.0.1）

---

## 4. 启动部署

```bash
cd /opt/planstar/planstar_backend
export ENV_FILE=.env.prod COMPOSE_DIR=/opt/planstar/planstar_backend PLANSTAR_CONFIG_FILE=config.lite.yaml

# 1) 发布前门禁（缺变量 / 占位符 / 弱口令 / compose 解析）
./deploy/preflight.sh

# 2) 三文件合并构建并启动
docker compose --env-file .env.prod \
  -f docker-compose.yml -f docker-compose.prod.yml -f docker-compose.server.yml \
  up -d --build
```

> 也可用 `./up-prod.sh`：等价于 preflight + ops_token 落盘 + 三文件叠加启动，但**不含 `--build`**，
> 首次上线仍建议用上面的显式命令（需要构建镜像）。

---

## 5. 验证

```bash
C="docker compose --env-file .env.prod -f docker-compose.yml -f docker-compose.prod.yml -f docker-compose.server.yml"

# 1) 容器全部 Up，无 Restarting
$C ps

# 2) 探针
curl -fsS http://127.0.0.1:9108/healthz          # {"status":"ok"}
curl -fsS http://127.0.0.1:9108/readyz           # {"status":"ready"}

# 3) 对象存储已启用（应看到 bucket / endpoint）
$C logs planstar-api | grep -i "tos object storage enabled"

# 4) HTTPS 业务入口（证书首次签发约 1~2 分钟）
curl -fsS 'https://api.planstar.top/v1/activities?pageIndex=1&pageSize=1'

# 5) 官网（服务 web）与后台控制台
curl -fsS https://www.planstar.top -o /dev/null -w 'www %{http_code}\n'   # 期望 200
curl -fsS https://admin.planstar.top -o /dev/null -w 'admin %{http_code}\n'

# 6) 发码接口非 5xx（会真实发短信，用测试手机号）
curl -s -X POST https://api.planstar.top/v1/user/send-code \
  -H 'Content-Type: application/json' -d '{"mobile":"你的手机号"}'
```

官网排查：`$C logs web --tail 50`（Next 启动失败会打印 `Error: Could not find a production build`，说明镜像没走 standalone 产物）。

看日志：`$C logs -f planstar-api`

---

## 6. 仍为占位的两项（不影响上线）

| 项 | 现状 | 建议 |
|---|---|---|
| `ACME_EMAIL=ops@example.com` | Let's Encrypt 可正常签发 | 换成真实邮箱以接收证书到期通知 |
| `AM_SMTP_*` | 占位，告警邮件发不出去 | 配好 SMTP 后填入，Alertmanager 才能真正报警 |

有问题把日志贴出来即可定位。
