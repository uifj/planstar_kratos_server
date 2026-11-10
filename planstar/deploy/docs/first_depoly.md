部署链路（生产版本 = `docker-compose.yml` + `docker-compose.prod.yml` + `.env.prod`，4C8G 低配档位关闭 ES/MQ/Consul/tracing，Traefik 走 HTTPS），并完成了以下准备：

1. **`.env.prod`** — 已生成并填入：你的阿里云 AK/SK、短信签名「恒创联众」、模板 CODE `10001`（登录注册），以及随机生成的强密钥（MySQL/Redis/DB 最小权限账号/运维令牌/JWT 等）
2. **`docker-compose.server.yml`** — 部署侧覆盖文件，运行时**关闭对象存储**。原因：你的存储是**火山引擎 TOS（S3 协议）**，而代码内置的是**阿里云 OSS SDK**（[storage.go](file:///d:/workspaces/flutter_projects/planstar_kratos_server/planstar/internal/pkg/storage/storage.go#L28-L44)），两者不兼容，先禁用不影响上线
3. **`ops_token`** — 与 `.env.prod` 中运维令牌同步
4. **本地验证**：三份 compose 合并解析通过（`COMPOSE_CONFIG_OK`）；preflight 门禁的配置检查项全部通过（只有权限和 WSL 无 docker 两个环境性提示，服务器上有 docker 不受影响）

**关于 5 个短信模板**：代码当前只支持单一模板（`biz/auth.go` 的 `SendLoginCode`），后端只实现了登录/注册发码流程，所以配了 `10001`。100002~100005（改绑/重置/绑新/验证新手机）对应后端未实现的功能，等以后加功能时再扩展。

**注意**：你的 AK/SK 已出现在聊天中，建议上线后到阿里云 RAM 创建一个只授权 dypnsapi 发短信权限的子账号替换。

---

## 部署包

已打包为 [planstar-deploy.tar.gz](computer://d:\workspaces\flutter_projects\planstar-deploy.tar.gz)（66MB，已排除 node_modules/dist/.git 和本地 .env）。

## 操作命令（你执行）

### 第 1 步：上传部署包到 ECS

在你**本机的 Windows 终端**执行（把 `<ECS_USER>`、`<ECS_IP>` 换成你的 SSH 账号和 ECS 公网 IP）：

```powershell
scp d:\workspaces\flutter_projects\planstar-deploy.tar.gz <ECS_USER>@<ECS_IP>:/opt/
```

### 第 2 步：在 ECS 上解包

在你**已连接的 ECS 终端**执行：

```bash
mkdir -p /opt/planstar
tar -xzf /opt/planstar-deploy.tar.gz -C /opt/planstar --strip-components=1
rm -f /opt/planstar-deploy.tar.gz   # 含真实密钥，解包后删除
cd /opt/planstar/planstar
chmod 600 .env.prod
```

### 第 3 步（强烈建议）：配置 Docker 镜像加速

国内 ECS 拉 Docker Hub 镜像（golang/mysql/nginx 等）很慢，建议先配阿里云加速器：

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

> 加速器地址在阿里云控制台「容器镜像服务 → 镜像加速器」获取；不想配也可以，就是构建会慢。

### 第 4 步：域名解析 + 安全组（部署前完成）

- **DNS 添加两条 A 记录**：`api.planstar.top` → ECS 公网 IP；`admin.planstar.top` → ECS 公网 IP（Traefik 路由已写死这两个域名，见 [dynamic.yml](file:///d:/workspaces/flutter_projects/planstar_kratos_server/planstar/deploy/traefik/dynamic.yml)）
- **ECS 安全组放行端口**：`80`、`443`（仅这两口对外开放，其余全绑 127.0.0.1）

### 第 5 步：部署启动（在 ECS 终端）

```bash
cd /opt/planstar/planstar
export ENV_FILE=.env.prod COMPOSE_DIR=/opt/planstar/planstar PLANSTAR_CONFIG_FILE=config.lite.yaml
./deploy/preflight.sh

docker compose --env-file .env.prod \
  -f docker-compose.yml -f docker-compose.prod.yml -f docker-compose.server.yml \
  up -d --build
```

首次启动 MySQL 会自动执行建库建表 + 生产化补丁 + 禁用测试账号 + 创建最小权限账号（10 个 SQL/SH 按字典序跑完）。

### 第 6 步：验证

```bash
docker compose --env-file .env.prod -f docker-compose.yml -f docker-compose.prod.yml -f docker-compose.server.yml ps
curl -fsS http://127.0.0.1:9108/healthz          # {"status":"ok"}
curl -fsS http://127.0.0.1:9108/readyz           # 依赖就绪后 {"status":"ready"}
curl -fsS 'http://127.0.0.1:8098/v1/activities?pageIndex=1&pageSize=1'

# HTTPS（DNS 生效 + 证书签发后，Let's Encrypt 首次约 1-2 分钟）
curl -fsS 'https://api.planstar.top/v1/activities?pageIndex=1&pageSize=1'

# 发码接口非 5xx（会真实发短信，用测试手机号）
curl -s -X POST https://api.planstar.top/v1/user/send-code -H 'Content-Type: application/json' -d '{"mobile":"你的手机号"}'
```

看日志：`docker compose ... logs -f planstar-api`

---

## 两个待你后续替换的占位值（不影响本次上线）

| 项 | 现状 | 建议 |
|---|---|---|
| `ACME_EMAIL=ops@example.com` | Let's Encrypt 可用 | 换成真实邮箱收证书到期通知 |
| `AM_SMTP_*` | 占位，告警邮件发不出去 | 有 SMTP 后填入，Alertmanager 才真正报警 |

有问题随时把日志贴给我。