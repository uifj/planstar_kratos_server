> 适用版本：planstar 单体 · 最后核对：2026-11-08 · 维护：后端组

# 阿里云短信（Dypnsapi）与 OSS 接入

`planstar` 的短信与对象存储默认关闭（`enabled=false` 时为 Noop）。启用真实阿里云服务时需要控制台开通、RAM 授权、签名/模板审核与密钥注入四步。人工填写项与变量清单见 `planstar_manual_setup/.trae/manual-setup-guide.md`。

## 1. 代码调用路径

| 项目 | 实现 |
| --- | --- |
| 产品 / API 版本 | 号码认证服务 Dypnsapi，`2017-05-25` |
| Go SDK | `github.com/alibabacloud-go/dypnsapi-20170525/v2` + `darabonba-openapi/v2` |
| 入口 | `notify.NewSMS`：`provider=aliyun` 或 `aliyun-dypns` → `newDypnsClient` |
| 接口 | `SendSmsVerifyCodeWithOptions`（`internal/pkg/notify/aliyun.go`） |
| Endpoint | 默认 `dypnsapi.aliyuncs.com`；可用 `PLANSTAR_NOTIFY_SMS_ENDPOINT` 覆盖 |
| 模板参数 | `TemplateParam={"code":"<6位数字>"}`，JSON key 必须是 `code` |
| 验证码策略 | `CodeLength=6`、`ValidTime=300`（5 分钟） |
| 超时 | `ConnectTimeout=5000ms`、`ReadTimeout=5000ms` |
| 成功判定 | 响应 `Body.Code == "OK"`，其余按失败返回 |
| 限制 | `Send()` 明确拒绝自由文本，Dypnsapi 只发模板验证码 |

配置键：`notify.sms.enabled/provider/endpoint/access_key/access_secret/sign_name/template_code`，对应 `PLANSTAR_NOTIFY_SMS_*` 环境变量。

## 2. 开通号码认证服务

1. 阿里云控制台搜索「号码认证服务」→ 开通服务；短信认证依赖「短信服务」，需同时开通并充值。
2. 完成企业实名认证；个人认证账号的短信签名与业务范围受限。
3. 在「号码认证服务 → 短信认证」确认可创建方案；签名与模板实际在短信服务控制台审核（见第 4 节）。
4. 记录账号的 AccessKey 归属，后续按第 3 节收敛权限，不要使用主账号 AK。

## 3. RAM 最小权限

控制台：RAM 访问控制 → 用户 → 创建用户 → 勾选「OpenAPI 调用访问」→ 保存 AccessKey ID/Secret（Secret 只显示一次）。

不要授予 `AliyunDypnsFullAccess`、`AliyunOSSFullAccess`。创建自定义策略并只挂载到该用户：

```json
{
  "Version": "1",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": "dypns:SendSmsVerifyCode",
      "Resource": "*"
    },
    {
      "Effect": "Allow",
      "Action": ["oss:PutObject", "oss:GetObject", "oss:DeleteObject"],
      "Resource": ["acs:oss:*:*:planstar-prod/*"]
    }
  ]
}
```

| 权限 | Action | Resource | 说明 |
| --- | --- | --- | --- |
| 发送短信验证码 | `dypns:SendSmsVerifyCode` | `*` | 该 API 不支持资源级授权 |
| OSS 上传/读取/删除 | `oss:PutObject` / `oss:GetObject` / `oss:DeleteObject` | `acs:oss:*:*:<bucket>/*` | 仅指定 Bucket 的对象前缀 |

- Bucket 名按实际替换；不要把 `Resource` 写成 `*`。
- AK/SK 只写入部署机 `.env` 或密钥管理，禁止入库、入日志、提交仓库；建议 90 天轮换。
- 用 RAM 策略模拟器验证，或按第 7 节发起一次真实发码/上传；不足权限时返回 `Forbidden.RAM`。

## 4. 短信签名与模板

控制台：短信服务控制台 `https://dysms.console.aliyun.com` → 国内消息 → 签名管理 / 模板管理。

| 项目 | 要求 |
| --- | --- |
| 签名 | 与企业主体、App/网站名称一致，如「企划星」；需营业执照等资质，审核约 2 小时 |
| 模板类型 | 验证码 |
| 模板内容 | `您的验证码为${code}，5分钟内有效，请勿泄露。` |
| 变量 | 只能使用 `code`，变量属性选「验证码」；内容必须包含 `${code}` |
| 模板编号 | 审核通过后形如 `SMS_123456789`，填入 `PLANSTAR_NOTIFY_SMS_TEMPLATE_CODE` |
| 签名名称 | 审核通过后填入 `PLANSTAR_NOTIFY_SMS_SIGN_NAME`，必须与模板绑定签名一致 |

服务端固定发送 6 位验证码、5 分钟有效；模板文案不要写「10 分钟」等不一致时效，否则用户按错误时效输入会失败。

## 5. 常见错误码

| 返回码 | 含义 | 处理 |
| --- | --- | --- |
| `OK` | 发送成功 | 检查手机号是否收到；运营商延迟通常 < 10s |
| `isv.SMS_SIGNATURE_ILLEGAL` | 签名不合法/未审核/与账号不匹配 | 核对签名全称、审核状态、是否属于当前账号 |
| `isv.SMS_TEMPLATE_ILLEGAL` | 模板不存在/未审核/变量不匹配 | 核对 `SMS_xxx`、模板变量必须为 `code` 且内容含 `${code}` |
| `isv.BUSINESS_LIMIT_CONTROL` | 触发流控（号码/分钟、小时、日） | 保持发码冷却 60s，排查压测或攻击；必要时提交工单提额 |
| `isv.AMOUNT_NOT_ENOUGH` | 账户余额不足 | 充值短信服务；配置余额告警 |
| `isv.MOBILE_NUMBER_ILLEGAL` | 手机号格式非法 | 校验 `^1\d{10}$`，排查测试号码 |
| `isv.OUT_OF_SERVICE` | 业务停机/欠费 | 检查账号状态与套餐 |
| `InvalidAccessKeyId` | AK 不存在、已禁用或 Endpoint 错误 | 核对 AK 状态、`dypnsapi.aliyuncs.com` |
| `SignatureDoesNotMatch` | SK 错误或系统时间偏差 | 重置 AK/SK，校准服务器时间 |
| `Forbidden.RAM` | RAM 权限不足 | 检查第 3 节策略是否挂载、Bucket 资源是否匹配 |

## 6. OSS 配置

1. OSS 控制台 → Bucket 列表 → 创建 Bucket：名称如 `planstar-prod`（全局唯一），地域如华东1（杭州），读写权限选**私有**，按需开启版本控制/同城冗余。
2. Endpoint 与 Region 必须成对：

| 场景 | Endpoint | Region |
| --- | --- | --- |
| 公网/本地调试 | `https://oss-cn-hangzhou.aliyuncs.com` | `oss-cn-hangzhou` |
| ECS 同地域内网 | `https://oss-cn-hangzhou-internal.aliyuncs.com` | `oss-cn-hangzhou` |

3. CDN：CDN 控制台 → 添加域名 `cdn.planstar.top` → 源站类型 OSS 域名 → 回源 `planstar-prod.oss-cn-hangzhou.aliyuncs.com`；按提示添加 CNAME，开启 HTTPS 并上传证书。静态图片缓存 7 天，`index.html` 不缓存。
4. 预签名：`PLANSTAR_STORAGE_OSS_PRESIGN_TTL_SECONDS=900`。建议 300~900；配了 `CDN_HOST` 时 `URL()` 直接返回 CDN 地址，不再签名。
5. 跨域（仅前端直传时需要）：Bucket → 数据安全 → 跨域设置，来源 `https://app.planstar.top`，方法 `PUT/GET/HEAD`，允许 Headers `*`，暴露 `ETag`，缓存 600s。当前服务端 `URL()` 只生成 GET 预签名；若要让前端直传，需新增 PUT 预签名接口，不能复用现有上传路径。
6. 对象 key 建议 `uploads/<yyyy>/<MM>/<uuid>.<ext>`，私有 Bucket + CDN 鉴权或短 TTL 预签名，避免公开读。

必需变量：`PLANSTAR_STORAGE_OSS_ENABLED=true`、`PROVIDER=aliyun`、`ENDPOINT`、`REGION`、`BUCKET`、`ACCESS_KEY`、`ACCESS_SECRET`，可选 `CDN_HOST`、`PRESIGN_TTL_SECONDS`。

## 7. 验证命令

```bash
# 1) 让配置生效
cd /opt/planstar/planstar
docker compose up -d --force-recreate planstar-api
docker compose logs --since 2m planstar-api | grep -E 'aliyun (sms|oss) provider enabled'

# 2) 发码（成功返回 200 + success=true；生产日志不得出现明文验证码）
curl -sS -X POST http://127.0.0.1:8098/v3/user/sendLoginCode \
  -H 'Content-Type: application/json' -d '{"ua":"13800000001"}'

# 3) 日志脱敏检查（预期只匹配到启动行，无 AK/SK、无 "code":"123456"）
docker compose logs --since 10m planstar-api | grep -Ei 'LTAI|AccessKeySecret|"code"|验证码'

# 4) OSS 冒烟（ossutil 需本机安装；v1/v2 命令略有差异）
ossutil64 cp ./smoke.txt oss://planstar-prod/smoke/smoke.txt \
  -e oss-cn-hangzhou.aliyuncs.com -i "$OSS_AK" -k "$OSS_SK"
ossutil64 sign oss://planstar-prod/smoke/smoke.txt --timeout 900
curl -fsSI '<上一步输出的签名 URL>' | head -1
```

失败时先看 `docker compose logs planstar-api` 中的 `code=`/`message=`；短信错误码按第 5 节处理，OSS 先确认 Bucket、Endpoint 与 RAM 资源 ARN 一致。
