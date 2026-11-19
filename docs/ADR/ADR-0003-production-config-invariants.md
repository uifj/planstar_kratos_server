# ADR-0003: 生产配置不变量与低配档位可启动性

- **状态**: ACCEPTED（已实施）
- **分类**: CFG（关联 SEC / DEP）
- **优先级**: P0
- **影响维度**: 生产功能 / 安全合规
- **日期**: 2026-11-19
- **来源**: `docs/OPEN_ITEMS.md` B1 / E5 / E6
- **修复 commit**: `e514842` 之后的治理批次

## 背景（Context）

`configs/config.lite.yaml` 是 Release deploy job 硬编码的 4C8G 档位（`release.yml:91,146`），但它同时是 `server.env: prod`，而 `conf.validateProductionInvariants()` 强制要求 `cookie_secure=true`、`ops.token`/`allowed_cidrs` 非空、MySQL 账号非 root、短信启用等。整改前配置中：

- `cookie_secure: false`（ICP 备案期明文 HTTP 直连所需）**且没有任何环境变量可以覆盖它**；
- `data.mysql.user: root`；
- `ops.token: ""`、`allowed_cidrs: []`、`notify.sms.enabled: false`（后三项可由 env 注入）。

结论：以 `env:prod` 启动该档位必然被 `conf.Validate()` 拒绝，而线上恰好用它 → 生产启动失败。此外 `release.yml:9` 与 `deploy/planstar.service` 引用了不存在的 `docker-compose.lite.yml`，`WorkingDirectory` 仍是改名前的路径。

## 决策（Decision）

1. **保留生产硬约束，但引入显式豁免开关**：新增 `auth.allow_insecure_cookie`（`PLANSTAR_AUTH_ALLOW_INSECURE_COOKIE` 可覆盖）。`Validate` 仅在 `cookie_secure=false` **且**未显式豁免时拒绝；ops 令牌 / CIDR / DB 最小权限账号 / 短信通道 / 测试口令等其余断言全部保持，不因豁免而放宽。
2. **`config.lite.yaml` 显式声明豁免**：`allow_insecure_cookie: true` 与 `cookie_secure: false` 成对出现，并写明「备案完成切回 HTTPS 后必须删除开关并改回 true」。
3. **DB 账号改为 `planstar_app`**（与 `sql/22_create_app_user.sh` 创建的最小权限账号一致），保留 `PLANSTAR_MYSQL_USER` 覆盖能力。
4. **修正部署文档与单元**：`release.yml` 注释改为现行三文件 compose；`planstar.service` 的 `WorkingDirectory`、`ExecStart`、`ExecStop` 改为 `/opt/planstar_backend` + `docker-compose.yml + prod + server` 三文件合并并带 `--env-file .env.prod`。
5. **不选方案**：不降低 `server.env` 到 `local`（会连带跳过全部生产断言），不删除 `CookieSecure` 断言（备案期需求与安全规约需要可审计的平衡点）。

## 后果（Consequences）

- **正面**：生产档位可启动；备案期 HTTP 直连得以保留；豁免是显式、可审计、可关闭的，不构成静默降级。
- **负面/风险**：豁免期内会话 Cookie 不带 Secure，明文 HTTP 下有被截获风险 → 必须由 `PLANSTAR_OPS_ALLOWED_CIDRS` 与 Traefik 访问限制配合，并在备案完成后立即关闭；这是**有时间边界的技术债**。
- **中性**：`config.lite.yaml` 其它占位值仍依赖部署侧 env 注入，`.env.prod.example` 与 `preflight.sh` 继续承担校验。

## 验证（Verification）

- `go test ./internal/conf -count=1 -v`：`TestLiteConfigStartsWithProductionEnv`、`TestProductionInvariantsRequireExplicitInsecureCookieAck`、`TestProductionInvariantsRejectUnsafeConfig`（11 项）全部通过。
- `deploy/preflight.sh` 校验 `.env.prod` 的必需键与 tag。
- `docker compose --env-file .env.prod -f docker-compose.yml -f docker-compose.prod.yml -f docker-compose.server.yml config -q` 退出 0。
- 备案完成后：`grep -rn 'allow_insecure_cookie' planstar_backend/` 应只在兼容代码与历史文档中命中，配置文件中应为 false/删除。

## 关联（Links）

- 代码: `planstar_backend/internal/conf/conf.go`（Auth 结构、Load 覆盖、Validate）、`configs/config.lite.yaml`、`internal/conf/production_test.go`
- 部署: `.github/workflows/release.yml`、`planstar_backend/deploy/planstar.service`、`up-prod.sh`
- 关联 ADR: `ADR-0001`（模块化单体与治理门禁）、`ADR-0002`（目录布局）
