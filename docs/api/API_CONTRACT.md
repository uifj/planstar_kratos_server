> 适用版本：planstar 单体 · 最后核对：2026-11-11 · 维护：后端组

# Planstar 客户端接口契约

## 1. 契约范围与事实源

- **唯一事实源**：`planstar/internal/service/service.go` 的 `Register` 路由表，以及同目录各 handler。
- 服务监听 `8098`；所有路径区分大小写，使用 `METHOD /path` 精确匹配。
- 当前注册 56 条路由，对应 **55 个唯一端点**：`GET /v1/user/` 是 `GET /v1/user` 的尾斜杠别名，复用同一 handler，本契约只记录一次。
- 成功响应为裸 JSON（无 `{code,message,data}` 信封），`Content-Type: application/json; charset=utf-8`；图片端点返回 `image/png`。
- 请求体字段大小写敏感，允许的字段别名已在各端点列出；未列出的字段当前被忽略。

## 2. 统一错误体（本次改造目标）

失败响应统一为：

```json
{
  "code": "BAD_REQUEST",
  "message": "参数错误",
  "traceId": "4bf92f3577b34da6a3ce929d0e0e4736"
}
```

| HTTP 状态 | `code` | 语义 |
| --- | --- | --- |
| 400 | `BAD_REQUEST` | 参数缺失、类型错误、路径 ID 非法 |
| 401 | `UNAUTHORIZED` | 未登录、`psSession` 缺失/过期/已撤销 |
| 403 | `FORBIDDEN` | 已登录但无权访问该资源 |
| 404 | `NOT_FOUND` | 资源不存在或不属于当前用户 |
| 409 | `CONFLICT` | 业务状态冲突 |
| 429 | `RATE_LIMITED` | 触发限流（如发码过于频繁） |
| 413 | `PAYLOAD_TOO_LARGE` | 请求体超过限制 |
| 500 | `INTERNAL` | 服务内部错误，不回显 SQL/堆栈 |
| 504 | `TIMEOUT` | 请求处理超时 |

订单业务码（HTTP 409，`code` 取以下值）：`PRICE_CHANGED`（金额/版本不一致）、`SOLD_OUT`（售罄或库存不足）、`LIMIT_EXCEEDED`（超过限购）、`QUOTE_EXPIRED`（报价过期或不存在）。四个错误值定义在 `internal/biz/errors.go`，由 `service.fail` 映射为 409，错误体与其它失败一致。

> **实现状态**：`service.fail` / `httpx.Error` 已输出统一 JSON 错误体，并在响应头回写 `X-Trace-Id`（与 body 的 `traceId` 相同）。请求体默认上限 1 MiB，超限由中间件返回 `413 PAYLOAD_TOO_LARGE`；单请求超过 15 秒返回 `504 TIMEOUT`。`httpx.Plain` 仅为历史兼容保留，当前业务错误不再使用。

## 3. 金额单位

| 位置 | 单位 | 说明 |
| --- | --- | --- |
| HTTP 请求（`payAmount` / `totalAmount`） | **分** | 下单入参，整数 |
| HTTP 响应（`payAmount`、`priceFrom`、`minPrice`、`listPriceFrom`、`originalPrice`、`price`、`walletBalance`、`balance`） | **分** | 整数 |
| MySQL `order_info.pay_amount` / `total_amount` | **元** | `DECIMAL(12,2)`；biz 层 `Fen` 转分输出，下单时 `fenToYuan` 落库 |
| 星币账户 `walletBalance` / `balance` / 钱包流水 `amount` / `balanceAfter` | **分** | 1 星币 = 100 分；客户端 `Money.fromMinorUnits` 渲染为「星币」 |
| 商品规格 `priceCoins` | **星币个数** | 1000 表示 1000 星币；客户端展示时乘以 100 转最小单位，不参与下单金额计算 |

星币是本项目唯一的前台货币口径：展示层统一为「星币」，不存在人民币符号；接口内部仍沿用整数最小单位以避免浮点误差。

不得在契约中混用元与分；`priceFrom`、`walletBalance` 等字段的客户端解析均按分处理。

## 4. 分页

| 端点族 | 参数 | 默认值 | 上限/异常处理 |
| --- | --- | --- | --- |
| 企划库、动态 | `page`、`pageSize` | `page=1`、`pageSize=20` | `page<=0` 归 1；`pageSize<=0` 或 `>100` 回落 20 |
| 消息、订单 | `pageIndex`、`pageSize` | `pageIndex=1`、`pageSize=20` | `pageIndex<=0` 归 1；`pageSize<=0` 或 `>100` 回落 20 |
| 动态翻页 | `nextPageKey` | 无 | `>0` 时覆盖 `page`；返回值是**下一页页码**，不是游标 |

注意：`pageSize=101` 会回落到 20，而不是截断为 100。

## 5. 认证与 Cookie

- 登录成功同时下发：
  - `Set-Cookie: psSession=<64 位 hex>; Path=/; HttpOnly; SameSite=Lax; Max-Age=2592000`；生产环境（`server.env=prod` 或 `auth.cookie_secure=true`）追加 `Secure`
  - 响应头 `psSession: <token>`（客户端两处都读）
  - `X-Trace-Id: <traceId>`
- 后续请求使用 `Cookie: psSession=<token>`；**不是** `Authorization: Bearer`。也兼容读取 `psSession` 请求头（`internal/pkg/session`）。
- 服务端会话有效期由 `auth.session_expire_days` 控制（默认 30 天），写入 `login_session.expire_time` 与 Redis TTL；Cookie 的 `Max-Age` 固定为 30 天（`internal/pkg/session/session.go`）。
- `dvid`：登录成功时若无设备号，服务端生成并同时写 `Set-Cookie: dvid=...` 与响应头 `dvid`。
- 图形码：`GET /v1/user/captcha` 写 `psCaptcha` Cookie（HttpOnly，5 分钟），明文码存 Redis 并一次性消费。
- 需要登录的端点：订单 8 个、消息 3 个、客服会话 4 个、用户资料/地址 6 个；其余目录与动态端点匿名可访问。
- 发码风控：同一手机号默认 60 秒冷却，同一 IP 默认每分钟 10 次（`auth.send_code_cooldown_seconds` / `auth.send_code_ip_limit_per_minute`），超限返回 `429 RATE_LIMITED`；IP 取 TCP 对端地址，不信任 `X-Forwarded-For`，经反向代理部署时可能被计为同一 IP。
- 登录失败风控：连续失败达到 `auth.login_fail_captcha_threshold`（默认 3）后要求图形码。发码/失败计数为**进程内**语义，多实例独立计数。

## 6. 版本错配说明

`v1` / `v3` / `v4` 是**兼容历史客户端的既有路径**，不是后端路由版本策略：

- 发码是 `POST /v3/user/sendLoginCode`，登录是 `POST /v1/user/login`。
- 企划详情（legacy activity 端点）是 `GET /v3/activities/{id}`，列表是 `GET /v1/activities`。
- 热门动态是 `GET /v4/feed/hot`，普通动态是 `GET /v1/feed`。
- 下单是 `POST /v3/unifiedOrder/makeOrder`，确认是 `POST /v1/unifiedOrder/confirm`。

客户端按原路径调用，后端不得“顺手统一”版本号；新增端点不沿用这套错配规则。

## 7. 端点明细

### 7.1 认证

GET /v1/user/captcha

- 鉴权：否
- 请求：无参数
- 响应：`image/png` 原始字节；同时 `Set-Cookie: psCaptcha=<captchaId>`
- 说明：4 位数字图形码，Redis TTL 5 分钟，校验一次即删除
- 错误：`500 INTERNAL`（改造前为 `text/plain`）

POST /v3/user/sendLoginCode

- 鉴权：否
- 请求体：
  | 字段 | 类型 | 必填 | 说明 |
  | --- | --- | --- | --- |
  | `ua` | string | 是 | 手机号，`^1\d{10}$` |
  | `captcha` | string | 否 | 图形码；提供时校验 `psCaptcha` Cookie |
- 响应：
  - 正常/参数/图形码/发送失败：HTTP 200，`{success: bool, needCaptcha: bool, errorMsg?: string}`
  - 触发手机号冷却或 IP 限频：HTTP 429，统一错误体（`RATE_LIMITED`）
- 说明：同一手机号默认 60 秒内只允许一次，同一 IP 默认每分钟 10 次（进程内计数）；验证码 6 位，落 `login_code`，TTL 5 分钟；当前不发真实短信，验证码写入服务端日志
- 错误：`429 RATE_LIMITED`、`413 PAYLOAD_TOO_LARGE`；业务失败仍以 `success=false` 表达

POST /v1/user/login

- 鉴权：否
- 请求体：
  | 字段 | 类型 | 必填 | 说明 |
  | --- | --- | --- | --- |
  | `ua` | string | 是 | 手机号 |
  | `ct` | int | 否 | 客户端类型，当前未参与校验 |
  | `pw` | string | 二选一 | 密码登录 |
  | `code` | string | 二选一 | 短信验证码登录；提供时优先走验证码 |
  | `captcha` | string | 否 | 图形码；提供时校验 Cookie 中的 `psCaptcha` |
- 响应：裸 `LoginBean`
  | 字段 | 类型 | 说明 |
  | --- | --- | --- |
  | `needCaptcha` | bool | 为 `true` 时 `user` 为 `null`，客户端需重新获取图形码 |
  | `newUser` | bool | 验证码登录是否新建用户 |
  | `user` | object/null | `AuthUser`，字段见第 8.2 节 |
- 会话：成功下发 `psSession` Cookie + 响应头，必要时下发 `dvid`
- 风控：同一手机号 10 分钟内失败 3 次后要求图形码
- 错误：`400 BAD_REQUEST`（手机号/请求格式非法）、`401 UNAUTHORIZED`（密码或验证码错误）、`413 PAYLOAD_TOO_LARGE`、`500 INTERNAL`

### 7.2 企划库（legacy activity 端点 · 仅线下回报可选）

> 产品主叙事已从「演出/购票」切换为「企划/支持」。本节端点保留，仅服务线下活动型企划的
> 回报场次与座位能力；新客户端请优先使用 7.7 企划域接口。字段语义对应关系：
> 演出 → 企划、场次 → 线下回报场次、票档 → 回报档位、购票 → 支持。

GET /v1/home/banners

- 鉴权：否
- 请求：无参数
- 响应：`Banner[]`，字段见第 8.3 节；数据源 `planstar_content.banner`（`status=1`，按 `sort`、`banner_id` 升序）
- 错误：`500 INTERNAL`

GET /v1/activities/hot

- 鉴权：否
- 请求：`page`、`pageSize`（分页规则见第 4 节）
- 响应：`ShowSummary[]`，字段见第 8.4 节
- 说明：无 `categoryId`/`city` 过滤时优先走 Elasticsearch 热门排序（`want_watch_num`、`rank_weight`、`activity_id` 降序），ES 失败回退 MySQL；列表项 `sessions`/`tiers`/`purchaseNotice` 为空数组
- 错误：`500 INTERNAL`

GET /v1/activities

- 鉴权：否
- 请求：
  | 参数 | 类型 | 必填 | 说明 |
  | --- | --- | --- | --- |
  | `page` | int | 否 | 默认 1 |
  | `pageSize` | int | 否 | 默认 20，上限 100 |
  | `categoryId` | int64 | 否 | 分类过滤 |
  | `city` | string | 否 | 城市名或城市 ID；兼容 `cityId` |
- 响应：`ShowSummary[]`，按 `rank_weight`、`activity_id` 降序
- 错误：`500 INTERNAL`

GET /v3/activities/{id}

- 鉴权：否
- 请求：路径参数 `id`（int64，企划 ID）
- 响应：单个 `ShowSummary`，额外填充：
  - `sessions[]`：`startsAt`、`showTime`、`tierSummary`、`soldOut`
  - `tiers[]`：`name`、`tierName`、`price`（分）、`note`、`remaining`
  - `lineup[]`、`purchaseNotice[]`
- 错误：`400 BAD_REQUEST`（id 非法）、`404 NOT_FOUND`、`500 INTERNAL`

### 7.3 动态

GET /v4/feed/hot

- 鉴权：否
- 请求：`page`、`pageSize`、`nextPageKey`
- 响应：`{data: FeedPost[], hasMore: bool, nextPageKey: int64}`（字段见第 8.5 节）
- 说明：按 `likes`、`views`、`feed_id` 降序；`nextPageKey` 按下一页页码解释
- 错误：`500 INTERNAL`

GET /v1/feed

- 鉴权：否
- 请求：`page`、`pageSize`、`nextPageKey`
- 响应：同上，按 `gmt_create`、`feed_id` 降序
- 错误：`500 INTERNAL`

GET /v1/tweets/{id}

- 鉴权：否
- 请求：路径参数 `id`（int64，= `feed_id` / `tweetId`）
- 响应：单个 `FeedPost`
- 错误：`400 BAD_REQUEST`、`404 NOT_FOUND`、`500 INTERNAL`

GET /v1/userReply

- 鉴权：否
- 请求：query `tweetId`（int64，必填且 `>0`）
- 响应：`FeedComment[]`，最多 100 条，按 `gmt_create` 降序
- 说明：先按该 feed 关联的 `activity_id` 查 `review`；无结果时退回把 `tweetId` 当 `activity_id` 查询
- 错误：`400 BAD_REQUEST`、`500 INTERNAL`

### 7.4 消息

GET /v1/messages

- 鉴权：是（`psSession`）
- 请求：
  | 参数 | 类型 | 必填 | 说明 |
  | --- | --- | --- | --- |
  | `pageIndex` | int | 否 | 默认 1 |
  | `pageSize` | int | 否 | 默认 20，上限 100 |
  | `type` | int | 否 | `1` 系统通知 / `2` 回复我的 / `3` 赞和关注；`0` 不过滤 |
- 响应：`MessageThread[]`，字段见第 8.6 节；按 `unread_count`、`gmt_create` 降序
- 错误：`401 UNAUTHORIZED`、`500 INTERNAL`

GET /v1/messages/unreadCount

- 鉴权：是
- 请求：无参数
- 响应：裸整数（`int`），为当前用户所有 `status=1` 消息的 `unread_count` 之和
- 错误：`401 UNAUTHORIZED`、`500 INTERNAL`

POST /v1/messages/read

- 鉴权：是
- 请求体：`{"messageId": 123}`；兼容别名 `{"id": 123}`
- 响应：`{}`
- 说明：只更新当前用户的消息；`messageId<=0` 返回 400
- 错误：`401 UNAUTHORIZED`、`400 BAD_REQUEST`、`500 INTERNAL`

GET /v1/chat/threads

- 鉴权：是（`psSession`）
- 请求：无参数
- 响应：`ChatThread[]`，字段见第 8.11 节；按 `gmt_modified` 降序，仅当前用户
- 错误：`401 UNAUTHORIZED`、`500 INTERNAL`

GET /v1/chat/threads/{id}/messages

- 鉴权：是
- 请求：
  | 参数 | 类型 | 必填 | 说明 |
  | --- | --- | --- | --- |
  | `limit` | int | 否 | 默认 20，上限 100；`<=0` 时回退读取 `pageSize` |
  | `pageSize` | int | 否 | `limit` 缺省时的别名，默认 20 |
  | `before` | int64 | 否 | 上一页最小 `id`（游标），只返回 `id < before` 的消息；`0` 表示最新一页 |
- 响应：`ChatMessage[]`，字段见第 8.11 节；单页内按发送时间升序，便于客户端直接追加
- 说明：会话必须属于当前用户，否则 404
- 错误：`401 UNAUTHORIZED`、`400 BAD_REQUEST`、`404 NOT_FOUND`、`500 INTERNAL`

POST /v1/chat/threads/{id}/messages

- 鉴权：是
- 请求体：`{"text": "你好"}`；兼容别名 `content` / `message`，去首尾空白后 1~2000 字符
- 响应：`ChatMessage`（落库后的消息，`fromMe=true`、`senderName` 取当前用户昵称）
- 说明：写入 `chat_message` 并同步更新会话 `preview` 与 `gmt_modified`
- 错误：`401 UNAUTHORIZED`、`400 BAD_REQUEST`、`404 NOT_FOUND`、`500 INTERNAL`

POST /v1/chat/threads/{id}/read

- 鉴权：是
- 请求体：无
- 响应：`{"unreadCount": 0}`
- 说明：只清零当前用户该会话的 `unread_count`
- 错误：`401 UNAUTHORIZED`、`400 BAD_REQUEST`、`404 NOT_FOUND`、`500 INTERNAL`

### 7.5 订单

GET /v1/order/list

- 鉴权：是
- 请求：
  | 参数 | 类型 | 必填 | 说明 |
  | --- | --- | --- | --- |
  | `pageIndex` | int | 否 | 默认 1 |
  | `pageSize` | int | 否 | 默认 20，上限 100 |
  | `status` | int | 否 | `>0` 时按订单状态过滤 |
  | `orderType` | int | 否 | `>0` 时按订单类型过滤 |
- 响应：`TicketOrder[]`，字段见第 8.7 节；按 `order_id` 降序，仅当前用户
- 错误：`401 UNAUTHORIZED`、`500 INTERNAL`

GET /v1/order/detail

- 鉴权：是
- 请求：query `orderNumber`（string，必填）
- 响应：单个 `TicketOrder`；仅允许查询当前用户的订单
- 错误：`401 UNAUTHORIZED`、`400 BAD_REQUEST`（缺少订单号）、`404 NOT_FOUND`、`500 INTERNAL`

POST /v1/unifiedOrder/confirm

- 鉴权：是
- 请求体：`activityId`（必填，兼容 `activity_id` / `showId`）、`activityEventId`（可选，兼容 `sessionId`）、`ticketCategoryId`（可选，兼容 `ticketId`）、`quantity`（可选，兼容 `count` / `buyCount`，缺省 1，上限 6）
- 响应：服务端报价快照 `QuoteSnapshot`（`quoteId`、`userId`、`activityId`、`activityEventId`、`ticketCategoryId`、`title`、`poster`、`unitPrice`、`quantity`、`totalAmount`、`version`、`expiresAt`），金额均为**分**，默认 15 分钟过期
- 说明：价格来自 `planstar_ticket` 在售回报档位最低价（`ticket_group.sale_price` → `ticket_category.low_price` → `catalog.activity.low_price` 回退），**不接受客户端指定金额**。快照写入 Redis，未配置 Redis 时降级为进程内 TTL 缓存（仅单实例可用）
- 错误：`401 UNAUTHORIZED`、`400 BAD_REQUEST`、`409 SOLD_OUT`、`409 LIMIT_EXCEEDED`、`413 PAYLOAD_TOO_LARGE`、`500 INTERNAL`

POST /v3/unifiedOrder/makeOrder

- 鉴权：是
- 请求体（字段兼容多别名，依次取首个可解析值）：
  | 逻辑字段 | 允许的键 | 类型 | 必填 | 说明 |
  | --- | --- | --- | --- | --- |
  | 企划 ID | `activityId` / `activity_id` / `showId` | int64 | 是 | `>0` |
  | 线下回报场次 ID | `activityEventId` / `activity_event_id` / `sessionId` | int64 | 否 | 落 `activity_event_id` |
  | 订单类型 | `orderType` / `order_type` | int | 否 | 缺省/0 时取 1 |
  | 支付金额 | `payAmount` / `pay_amount` | int64（分） | 否 | 兼容字段，落库值以服务端报价为准 |
  | 订单总额 | `totalAmount` / `total_amount` | int64（分） | 否 | 兼容字段，落库值以服务端报价为准 |
  | 收货人 | `name` / `consignee` | string | 否 | |
  | 手机号 | `phone` / `mobile` | string | 否 | |
  | 地址 ID | `userAddressId` / `user_address_id` / `addressId` | int64 | 否 | |
  | 报价 ID | `quoteId` / `quote_id` | string | 推荐 | 指向 confirm 下发的报价快照 |
  | 报价版本 | `quoteVersion` / `quote_version` / `version` | int | 否 | 传了则必须与快照 `version` 一致 |
  | 客户端金额 | `amount` / `payAmount` / `pay_amount` | int64（分） | 条件必填 | 无 `quoteId` 时必填，仅用于与服务端重算结果比对 |
  | 购买数量 | `quantity` / `count` / `buyCount` | int | 否 | 单笔上限 6（`biz.MaxQuantityPerOrder`） |
  | 回报档位 ID | `ticketCategoryId` / `ticket_category_id` / `ticketId` | int64 | 否 | 定价以服务端在售回报档位为准 |
- 请求头：`Idempotency-Key`（可选但推荐，同 key 24 小时内返回首次创建的订单）
- 响应：`TicketOrder`，字段见第 8.7 节
- 说明：服务端按报价快照或当前权威价格重算金额，**忽略客户端金额的定价作用**。创建在单个事务内完成：幂等查询 → 原子扣减 `planstar_ticket.ticket_category.stock`（影响 0 行即 `SOLD_OUT`）→ 写 `order_info` → 写 `order_idempotency`。订单 ID 由应用侧生成（毫秒时间戳 + 随机位），`order_number` 为 `PN + 毫秒 + 6 位随机` 且有唯一索引；不存在 `MAX(id)+1` 逻辑。企划数据通过 `CatalogReader` 端口从 catalog 库读取，不再跨库直查
- 错误：`401 UNAUTHORIZED`、`400 BAD_REQUEST`（缺 `activityId`/`quoteId`/`amount` 或 body 非法）、`409 PRICE_CHANGED`、`409 SOLD_OUT`、`409 LIMIT_EXCEEDED`、`409 QUOTE_EXPIRED`、`413 PAYLOAD_TOO_LARGE`、`500 INTERNAL`

POST /v1/unifiedPay/pay

- 鉴权：是
- 请求体：
  | 字段 | 类型 | 必填 | 说明 |
  | --- | --- | --- | --- |
  | `orderNumber` | string | 是 | 订单号（兼容 `order_number`） |
  | `channel` | string | 是 | `wechat` / `alipay` / `unionpay` / `balance`（兼容 `payChannel` / `pay_channel`） |
- 响应：`PaymentOrder`，字段见第 8.12 节；金额单位为**分**，`expireAt` 为毫秒时间戳
- 说明：按 `(order_number, channel)` 唯一幂等，同一订单同一渠道重复调用返回同一 `payNo`；已过期（`pending` 或读取时已落为 `expired`）的支付单再次调用会重置为 `pending` 并续期 30 分钟。`payParams` 当前为占位对象 `{}`，字段留给后续支付 SDK
- 错误：`401 UNAUTHORIZED`、`400 BAD_REQUEST`（缺订单号/渠道非法）、`404 NOT_FOUND`（订单不存在或不属于当前用户）、`409 CONFLICT`（订单已关闭/已退款）、`500 INTERNAL`

GET /v1/unifiedPay/payDetail

- 鉴权：是
- 请求：query `payNo` 或 `orderNumber`，二者至少一个；只传 `orderNumber` 时返回该订单最新一笔支付单
- 响应：`PaymentOrder`，字段见第 8.12 节
- 说明：读取时若 `status=pending` 且已过 `expireAt`，返回 `status=expired`，并同步把库内状态落为 `expired`
- 错误：`401 UNAUTHORIZED`、`400 BAD_REQUEST`（两者都未传）、`404 NOT_FOUND`、`500 INTERNAL`

POST /v1/order/{id}/refund

- 鉴权：是
- 请求体：`reasonCode`（string，可选，缺省 `other`；白名单 `notNeeded` / `boughtWrong` / `tooExpensive` / `foundBetter` / `other`）、`reasonText`（string，可选，≤512 字，兼容 `reason`）
- 响应：`Refund`，字段见第 8.12 节；`refundableAmount` 单位为**分**
- 说明：按 `order_id` 唯一幂等，同一订单重复提交返回同一 `refundNo`；仅 `order_info.status` 为 1~5 的订单可退，0/6/7 返回 409
- 错误：`401 UNAUTHORIZED`、`400 BAD_REQUEST`（`id` 非法/原因码不在白名单/原因过长）、`404 NOT_FOUND`、`409 REFUND_NOT_ALLOWED`、`500 INTERNAL`

GET /v1/order/{id}/refund

- 鉴权：是
- 请求：路径参数 `id`（订单 ID）
- 响应：`Refund`，字段见第 8.12 节
- 错误：`401 UNAUTHORIZED`、`400 BAD_REQUEST`、`404 NOT_FOUND`（订单不存在或尚无退款单）、`500 INTERNAL`

### 7.6 用户与地址

GET /v1/user

- 鉴权：是
- 请求：无参数
- 响应：`UserProfile`，字段见第 8.8 节
- 别名：`GET /v1/user/`（尾斜杠）复用同一 handler，响应完全一致
- 错误：`401 UNAUTHORIZED`、`404 NOT_FOUND`、`500 INTERNAL`

GET /v1/user/signIn

- 鉴权：是
- 请求：无参数
- 响应：`SignInStatus`，字段见第 8.13 节；`signed=true` 表示当天已签到，`balance` 为当前星币余额（分）
- 说明：只读，不产生签到记录
- 错误：`401 UNAUTHORIZED`、`500 INTERNAL`

POST /v1/user/signIn

- 鉴权：是
- 请求：无参数
- 响应：`SignInStatus`；当天首签返回 `alreadySigned=false`，重复签到返回 `alreadySigned=true`
- 说明：每个自然日最多入账一次；首签在单事务内写入 `planstar_marketing.daily_sign_in`、`wallet_account.balance += 100` 并写一条「每日签到」流水；重复签到不改变余额
- 错误：`401 UNAUTHORIZED`、`500 INTERNAL`

GET /v1/moneyaccount

- 鉴权：是
- 请求：无参数
- 响应：`WalletAccount`，字段见第 8.13 节；账户不存在时按 `user_statistic.balance` 初始化后返回
- 错误：`401 UNAUTHORIZED`、`500 INTERNAL`

GET /v1/moneyaccount/sequences

- 鉴权：是
- 请求：无参数（固定返回最近 50 条）
- 响应：`{"items":[WalletTransaction],"total":int}`；按时间倒序
- 错误：`401 UNAUTHORIZED`、`500 INTERNAL`

GET /v1/myaddress

- 鉴权：是
- 请求：无参数
- 响应：`ShippingAddress[]`，仅 `status=1`，按 `is_default`、`address_id` 降序
- 错误：`401 UNAUTHORIZED`、`500 INTERNAL`

POST /v1/myaddress

- 鉴权：是
- 请求体：
  | 字段 | 类型 | 必填 | 说明 |
  | --- | --- | --- | --- |
  | `name` / `consignee` | string | 是 | 收货人 |
  | `mobile` / `phone` | string | 是 | 手机号 |
  | `region` | string | 否 | 单个“省 市 区”字符串，按空格拆分 |
  | `province` / `city` / `district` | string | 否 | 显式省市区，优先于 `region` 拆分结果 |
  | `detail` / `address` | string | 否 | 详细地址 |
  | `isDefault` / `default` | bool | 否 | 设为默认时先清除同用户其他默认地址 |
- 响应：新建的 `ShippingAddress`
- 错误：`401 UNAUTHORIZED`、`400 BAD_REQUEST`（缺少 `name` 或 `mobile`）、`500 INTERNAL`

DELETE /v1/myaddress/{id}

- 鉴权：是
- 请求：路径参数 `id`（int64，地址 ID）
- 响应：`{}`
- 说明：软删除（`status=0`），仅作用于当前用户
- 错误：`401 UNAUTHORIZED`、`400 BAD_REQUEST`、`500 INTERNAL`

POST /v1/myaddress/default/{id}

- 鉴权：是
- 请求：路径参数 `id`（int64，地址 ID）
- 响应：`{}`
- 说明：事务内先清除当前用户所有默认标记，再把目标地址置为默认；不存在或非本人地址不报 404，仍返回 `{}`
- 错误：`401 UNAUTHORIZED`、`400 BAD_REQUEST`、`500 INTERNAL`

## 7.7 企划域

GET /v1/projects/hot

- 鉴权：否
- 请求：`page`、`pageSize`
- 响应：`ProjectSummary[]`，字段见第 8.10 节
- 说明：按 `is_featured`、`rank_weight`、`supporter_count` 降序；仅返回 `publish_status=1 AND status=1`
- 错误：`500 INTERNAL`

GET /v1/projects

- 鉴权：否
- 请求：
  | 参数 | 类型 | 必填 | 说明 |
  | --- | --- | --- | --- |
  | `page` | int | 否 | 默认 1 |
  | `pageSize` | int | 否 | 默认 20，上限 100 |
  | `categoryId` | int64 | 否 | 企划一级品类 |
  | `fundingStatus` | string | 否 | `preheating` / `ongoing` / `achieved` / `delivered` / `failed` |
  | `sort` | string | 否 | `hot`（默认）/ `new` / `ending` / `achievement` |
- 响应：`ProjectSummary[]`
- 错误：`400 BAD_REQUEST`（枚举非法）、`500 INTERNAL`

GET /v1/projects/{id}

- 鉴权：否
- 请求：路径参数 `id`（int64，企划 ID）
- 响应：`ProjectDetail`，在 `ProjectSummary` 基础上填充：
  - `story`、`gallery[]`、`riskNote`、`deliveryPlan`、`fundUsage`、`faq[]`
  - `refundRule`、`refundDays`（未达成退款规则可视化）
  - `ratingAvg`（十分制，保留一位小数）、`reviewCount`、`showCount`
  - `creator`：`creatorId`、`displayName`、`avatar`、`isVerified`、`projectNum`、`successNum`、`fulfillmentRate`
  - `rewardTiers[]`：字段见第 8.10 节
  - `updates[]`（最近 5 条）、`stretchGoals[]`
- 错误：`400 BAD_REQUEST`、`404 NOT_FOUND`、`500 INTERNAL`

GET /v1/projects/{id}/reward-tiers

- 鉴权：否
- 响应：`RewardTier[]`，按 `sort` 升序
- 错误：`404 NOT_FOUND`、`500 INTERNAL`

GET /v1/projects/{id}/updates

- 鉴权：否（`is_supporter_only=1` 的条目需登录且已支持）
- 请求：`page`、`pageSize`
- 响应：`ProjectUpdate[]`，按 `publish_time` 降序
- 错误：`401 UNAUTHORIZED`（未登录且存在仅支持者可见内容）、`403 FORBIDDEN`（已登录但未支持）、`500 INTERNAL`

POST /v1/projects/{id}/pledges

- 鉴权：是（`psSession`）
- 请求体：
  | 字段 | 类型 | 必填 | 说明 |
  | --- | --- | --- | --- |
  | `rewardTierId` | int64 | 是 | 回报档位 ID |
  | `quantity` | int | 否 | 默认 1，单笔上限 99 |
  | `isAnonymous` | bool | 否 | 默认 false |
  | `message` | string | 否 | 支持留言，≤255 字 |
  | `addressId` | int64 | 否 | 周边类回报必填 |
- 响应：`Pledge`，字段见第 8.10 节
- 说明：校验企划 `funding_status in (preheating, ongoing)`、档位库存与地址完整性；限量档位原子扣减库存，创建后状态为 `pending`，等待支付
- 错误：`400 BAD_REQUEST`、`401 UNAUTHORIZED`、`404 NOT_FOUND`（企划不存在）、`409 CONFLICT`（库存不足/已结束）、`500 INTERNAL`

GET /v1/pledges

- 鉴权：是（`psSession`）
- 请求：`page`、`pageSize`、`status`（可选）
- 响应：`Pledge[]`，按 `gmt_create` 降序
- 错误：`401 UNAUTHORIZED`、`500 INTERNAL`

GET /v1/pledges/{id}

- 鉴权：是（仅本人）
- 响应：单个 `Pledge`
- 错误：`401 UNAUTHORIZED`、`404 NOT_FOUND`、`500 INTERNAL`

GET /v1/creators/{id}

- 鉴权：否
- 响应：`CreatorProfile` 字段平铺 + `projects: ProjectSummary[]`（其已发布企划，按 `rank_weight` 降序）
- 错误：`404 NOT_FOUND`、`500 INTERNAL`

GET /v1/projects/{id}/faqs

- 鉴权：否
- 响应：`ProjectFaq[]`，按 `sort` 升序；与 `ProjectDetail.faq` 同源
- 错误：`404 NOT_FOUND`、`500 INTERNAL`

GET /v1/projects/{id}/reviews

- 鉴权：否
- 请求：`page`、`pageSize`、`sort`（`latest` 默认 / `hot`）、`onlyShow`（bool，仅看晒单）
- 响应：`{items: ProjectReview[], ratingAvg: number, reviewCount: int, showCount: int}`
- 说明：仅返回 `status=1`；匿名评价隐藏 `userName`/`userAvatar`；`ratingAvg` 为十分制一位小数
- 错误：`404 NOT_FOUND`、`500 INTERNAL`

POST /v1/projects/{id}/reviews

- 鉴权：是（`psSession`）
- 请求体：
  | 字段 | 类型 | 必填 | 说明 |
  | --- | --- | --- | --- |
  | `pledgeId` | int64 | 是 | 本人的支持记录，且 `pledge_status=delivered` |
  | `rating` | int | 是 | 1–5 |
  | `content` | string | 否 | ≤1024 字 |
  | `images` | string[] | 否 | 晒单图片 |
  | `isShow` | bool | 否 | 默认 false |
  | `isAnonymous` | bool | 否 | 默认 false |
- 响应：`ProjectReview`
- 说明：同一 `pledgeId` 仅可评价一次；评价进入 `status=1`，命中风控词则 `status=2`
- 错误：`400 BAD_REQUEST`、`401 UNAUTHORIZED`、`403 FORBIDDEN`（非本人/未交付）、`409 CONFLICT`（重复评价）、`500 INTERNAL`

POST /v1/projects/{id}/reports

- 鉴权：是（`psSession`）
- 请求体：
  | 字段 | 类型 | 必填 | 说明 |
  | --- | --- | --- | --- |
  | `targetType` | string | 是 | `project` / `review` / `feed` / `pledge` |
  | `targetId` | int64 | 是 | 举报对象 ID |
  | `reasonCode` | string | 是 | `fraud` / `infringe` / `porn` / `violence` / `false_claim` / `other` |
  | `description` | string | 否 | ≤1024 字 |
  | `evidence` | string[] | 否 | 证据图片 |
- 响应：`ProjectReport`（`status=pending`）
- 说明：同一用户对同一对象 24 小时内重复举报返回 `409`
- 错误：`400 BAD_REQUEST`、`401 UNAUTHORIZED`、`409 CONFLICT`、`500 INTERNAL`

GET /v1/rankings

- 鉴权：否
- 请求：
  | 参数 | 类型 | 必填 | 说明 |
  | --- | --- | --- | --- |
  | `type` | string | 否 | `hot`（默认）/ `achievement` / `ending` |
  | `categoryId` | int64 | 否 | 品类筛选 |
  | `limit` | int | 否 | 默认 10，上限 50 |
- 响应：`ProjectRanking`，含 `type`、`generatedAt(毫秒时间戳)`、`items[]`（`rank`、`project`、`metricValue`）
- 说明：`hot` 按支持人数与 `rank_weight`，`achievement` 按达成率，`ending` 按剩余天数升序；仅统计 `funding_status in (ongoing, achieved)`
- 错误：`400 BAD_REQUEST`、`500 INTERNAL`

## 7.8 星币商城

GET /v1/shop/goods

- 鉴权：否
- 请求：无参数
- 响应：`{"items":[Goods],"total":int}`，`Goods` / `GoodsSKU` 字段见第 8.13 节；仅返回 `status=1` 的商品与规格，按 `sort`、`goods_id` 升序
- 错误：`500 INTERNAL`

GET /v1/shop/goods/{id}

- 鉴权：否
- 请求：路径参数 `id`（int64，商品 ID）
- 响应：单个 `Goods`（含全部 `status=1` 的规格）
- 错误：`400 BAD_REQUEST`、`404 NOT_FOUND`、`500 INTERNAL`

POST /v1/shop/goods/{id}/skus/{skuId}/redeem

- 鉴权：是
- 请求：路径参数 `id`（商品 ID）、`skuId`（规格 ID）；无请求体
- 响应：`RedeemResult`，字段见第 8.13 节
- 说明：单事务内校验库存与星币余额，扣星币、写「兑换「商品·规格」」支出流水、`stock-1` / `sold+1`、生成 `goods_order`
- 错误：`401 UNAUTHORIZED`、`400 INSUFFICIENT_COINS`（星币余额不足）、`409 SOLD_OUT`（库存不足）、`404 NOT_FOUND`、`500 INTERNAL`

## 7.9 企划创意

企划创意是「立项前的态度征集」，只做支持/不支持统计，**不扣减星币、不产生钱包流水与订单**。

GET /v1/projectIdeas/feed

- 鉴权：否（携带 `psSession` 时回填 `myVote`）
- 请求：
  | 参数 | 类型 | 必填 | 说明 |
  | --- | --- | --- | --- |
  | `cursor` | string | 否 | 游标，格式 `hotScore_ideaId`；首页留空，非法游标回退首页 |
  | `size` | int | 否 | 默认 10，上限 20 |
- 响应：`{"items": ProjectIdea[], "nextCursor": string, "hasMore": bool}`
- 说明：仅返回 `publish_status=1 AND status=1`，按 `hot_score DESC, idea_id DESC` 排序；`description` 不在列表返回
- 错误：`500 INTERNAL`

GET /v1/projectIdeas/{ideaId}

- 鉴权：否（携带 `psSession` 时返回 `myVote`）
- 请求：路径参数 `ideaId`（int64，创意 ID）
- 响应：单个 `ProjectIdea`，含 `description`、`tierDistribution`、`reasonCounts`、`myVote`（游客无 `myVote`）
- 说明：`tierDistribution` 由 `project_idea_vote` 实时聚合，键为 `"5"` / `"10"` / `"20"`；`reasonCounts` 为不支持理由的标准 `reasonCode` 计数聚合（不含理由原文）；`supportRate` 保留三位小数
- 错误：`404 IDEA_NOT_FOUND`、`500 INTERNAL`

POST /v1/projectIdeas/{ideaId}/vote

- 鉴权：是（`psSession`）
- 请求体：
  | 字段 | 类型 | 必填 | 说明 |
  | --- | --- | --- | --- |
  | `voteType` | int | 是 | `1` 支持 / `2` 不支持 |
  | `coinTier` | int | 支持时必填 | `5` / `10` / `20`（仅意愿，不扣费） |
  | `reasonCode` | string | 不支持时必填 | `not_interested` / `too_many_similar` / `not_enough_info` / `not_executable` / `unreasonable_pricing` / `other` |
  | `reasonText` | string | `other` 时必填 | 支持 ≤60 字；不支持 ≤100 字；不计入公开返回 |
- 响应：`IdeaVoteResult`，字段见第 8.14 节；成功响应为裸 JSON（与其余端点一致，错误体见第 2 节）
  ```json
  {
    "ideaId": 1001,
    "supportCount": 128,
    "opposeCount": 12,
    "supportRate": 0.914,
    "coinIntentSum": 1360,
    "tierDistribution": {"5": 60, "10": 48, "20": 20},
    "myVote": {"voteType": 1, "coinTier": 10},
    "alreadyVoted": false
  }
  ```
- 说明：
  - 一人一卡一票（唯一键 `uk_idea_user`）；重复提交返回 `200` 且 `alreadyVoted=true`，计数不重复增加
  - 作者自投返回 `403`；支持写 `vote_type=1, coin_tier=tier`、`support_count+1`、`coin_intent_sum+tier`；不支持写 `vote_type=2, reason_code=code`、`oppose_count+1`
  - `hot_score` 由定时任务刷新，投票事务内不更新
- 错误：`400 BAD_REQUEST`（`voteType` 非法或文本超长）、`400 INVALID_COIN_TIER`（档位不在 5/10/20）、`400 REASON_REQUIRED`（缺少 `reasonCode`，或 `other` 未填 `reasonText`）、`400 INVALID_REASON_CODE`（理由不在字典）、`401 UNAUTHORIZED`、`403 IDEA_SELF_VOTE_FORBIDDEN`、`404 IDEA_NOT_FOUND`、`429 RATE_LIMITED`（频控：单用户 60 次/小时、单 IP 300 次/小时）、`500 INTERNAL`

## 8. 响应模型

字段名以 `internal/biz` 包内的模型文件为准；同义字段为兼容客户端而重复输出。

### 8.1 SendCodeResp

`success(bool)`、`needCaptcha(bool)`、`errorMsg(string, omitempty)`

### 8.2 LoginBean 与 AuthUser

`LoginBean`：`needCaptcha(bool)`、`newUser(bool)`、`user(AuthUser|null)`

`AuthUser`：`userId(int64)`、`userName(string)`、`avatar(string)`、`bio(string,omitempty)`、`gender(0/1/2)`、`birthday(毫秒时间戳, 0=未设置)`、`mobileNo(string,omitempty)`、`level(int)`、`followerNum(int)`、`followingNum(int)`、`wantWatchNum(int)`、`isDouliUser(bool)`、`badgeIcon(string,omitempty)`

### 8.3 Banner

`id(int64)`、`title(string)`、`image(string)`、`poster(string,omitempty)`、`url(string,omitempty)`、`showId(int64,omitempty)`、`sort(int,omitempty)`

### 8.4 ShowSummary

`id`、`activityId`、`title`、`name`、`artist`、`star`、`categoryId`、`category`、`categoryName`、`city`、`venue`、`venueName`、`venueAddress`、`firstSessionAt`、`showTime`、`priceFrom`、`minPrice`、`listPriceFrom`、`originalPrice`（金额均为分）、`discountPermille`、`discount`（当前固定 1000）、`tags(string[])`、`wishCount`、`wannaSeeCount`、`soldOut`、`status`、`proxyBuy`、`rating(float64)`、`ratingCount`、`duration`、`intro`、`description`、`poster`、`sessions(SessionSummary[])`、`tiers(TierSummary[])`、`lineup(string[])`、`purchaseNotice(string[])`

`SessionSummary`：`startsAt`、`showTime`、`tierSummary`、`soldOut`

`TierSummary`：`name`、`tierName`、`price`（分）、`note(omitempty)`、`remaining`

### 8.5 FeedPage / FeedPost / FeedComment

`FeedPage`：`data(FeedPost[])`、`hasMore(bool)`、`nextPageKey(int64)`

`FeedPost`：`id`、`tweetId`、`title`、`content`、`author`、`userName`、`viewCount`、`readCount`、`publishedAt`、`createTime`、`tag`、`showId`、`activityId`、`likeCount`、`likeNum`、`hasVideo`、`video`、`body(omitempty)`、`images(string[],omitempty)`

`FeedComment`：`id`、`author`、`userName`、`text`、`publishedAt`、`createTime`、`likeCount`

### 8.6 MessageThread

`id`、`messageId`、`title`、`subject`、`preview`、`content`、`updatedAt`、`updateTime`、`createTime`、`unreadCount`、`unread`、`avatar(omitempty)`、`isOfficial`、`official`

### 8.7 TicketOrder

`orderId`、`id`、`orderNumber`、`showId`、`sessionId`、`createTime`（`yyyy-MM-dd HH:mm:ss`）、`orderType`、`buyType`（当前固定 1）、`status`、`title`、`poster`、`payAmount`（分）

### 8.8 UserProfile

`userId`、`id`、`nickname`、`userName`、`avatar`、`headImg`、`mobile`、`phone`、`level`、`userLevel`、`levelLabel(omitempty)`、`wannaSeeCount`、`followActivityCount`、`followCount`、`followUserCount`、`couponCount`、`walletBalance`（分）、`balance`（分）

### 8.9 ShippingAddress

`id`、`addressId`、`name`、`consignee`、`mobile`、`phone`、`province`、`city`、`district`、`region`、`detail`、`address`、`isDefault`、`default`

### 8.10 企划域

> 以下模型对应第 7.7 节，已在 `internal/biz/project.go` 落地，属于第 1 节所述
> 55 个唯一端点。金额字段与第 3 节一致，均为**分**。

`ProjectSummary`：
`projectId(int64)`、`projectNo(string)`、`title(string)`、`subtitle(string,omitempty)`、`cover(string)`、`categoryId(int64)`、`categoryName(string)`、`projectType(string)`、`creatorId(int64)`、`creatorName(string)`、`goalAmount(int64)`、`raisedAmount(int64)`、`supporterCount(int)`、`deadline(毫秒时间戳)`、`remainingDays(int)`、`fundingStatus(string)`、`achievementRate(int, 千分比)`、`minSupportAmount(int64)`、`tags(string[])`、`isFeatured(bool)`

`RewardTier`：
`rewardTierId(int64)`、`title(string)`、`price(int64)`、`rewardType(string)`、`items(string[])`、`limitNum(int)`、`soldNum(int)`、`remainingNum(int, 不限量档位省略该字段)`、`estimatedDeliveryTime(string, omitempty)`、`isEarlyBird(bool)`、`isLimited(bool)`

`ProjectUpdate`：
`updateId(int64)`、`title(string)`、`content(string)`、`images(string[])`、`isSupporterOnly(bool)`、`publishTime(毫秒时间戳)`

`StretchGoal`：
`goalId(int64)`、`title(string)`、`description(string,omitempty)`、`targetAmount(int64)`、`currentAmount(int64)`、`isAchieved(bool)`、`sort(int)`

`Pledge`：
`pledgeId(int64)`、`pledgeNo(string)`、`projectId(int64)`、`rewardTierId(int64)`、`supportAmount(int64)`、`quantity(int)`、`isAnonymous(bool)`、`supportMessage(string,omitempty)`、`deliveryType(string)`、`pledgeStatus(string)`、`paidTime(毫秒时间戳, 0=未支付)`、`deliveredTime(毫秒时间戳, 0=未发货)`、`trackingNo(string,omitempty)`、`contentUrl(string,omitempty)`、`redeemCode(string,omitempty)`、`redeemedTime(毫秒时间戳, 0=未核销)`

`CreatorProfile`：
`creatorId(int64)`、`displayName(string)`、`avatar(string)`、`isVerified(bool)`、`projectNum(int)`、`successNum(int)`、`fulfillmentRate(int, 千分比)`、`bio(string,omitempty)`

`ProjectFaq`：
`faqId(int64)`、`question(string)`、`answer(string)`、`sort(int)`

`ProjectReview`：
`reviewId(int64)`、`projectId(int64)`、`userName(string)`、`userAvatar(string,omitempty)`、`rating(int)`、`content(string,omitempty)`、`images(string[])`、`isShow(bool)`、`isAnonymous(bool)`、`likeCount(int)`、`gmtCreate(毫秒时间戳)`

`ProjectReport`：
`reportId(int64)`、`targetType(string)`、`targetId(int64)`、`reasonCode(string)`、`description(string,omitempty)`、`evidence(string[])`、`status(string)`、`handleNote(string,omitempty)`、`gmtCreate(毫秒时间戳)`

`ProjectRanking`：
`type(string)`、`generatedAt(毫秒时间戳)`、`items[]`：`rank(int)`、`project(ProjectSummary)`、`metricValue(int64)`

### 8.11 客服会话（ChatThread / ChatMessage）

`ChatThread`：`threadId(int64)`、`title(string)`、`preview(string)`、`updatedAt(string, yyyy-MM-dd HH:mm:ss)`、`unreadCount(int)`、`avatarUrl(string)`、`isOfficial(bool)`

`ChatMessage`：`id(int64)`、`text(string)`、`sentAt(string, yyyy-MM-dd HH:mm:ss)`、`fromMe(bool)`、`senderName(string)`

### 8.12 支付单与退款单（PaymentOrder / Refund）

`PaymentOrder`：`payNo(string)`、`orderNumber(string)`、`amount(int64, 分)`、`channel(string)`、`status(string, pending/success/failed/expired)`、`payParams(object, 当前为 {})`、`expireAt(毫秒时间戳)`、`paidAt(毫秒时间戳, 0=未支付)`、`thirdTradeNo(string, 空字符串=未支付)`

`Refund`：`refundNo(string)`、`orderId(int64)`、`refundableAmount(int64, 分)`、`status(string, applied/processing/success/rejected)`、`expectedArrivalDays(int)`

### 8.13 星币钱包与商城（WalletAccount / WalletTransaction / SignInStatus / Goods / RedeemResult）

`WalletAccount`：`accountId(int64)`、`balance(int64, 分)`、`currency(string, 固定 star_coin)`、`status(int32)`

`WalletTransaction`：`id(int64)`、`title(string)`、`amount(int64, 分，收入为正、支出为负)`、`balanceAfter(int64, 分)`、`type(string, income/expense)`、`happenTime(string, RFC3339)`

`SignInStatus`：`signed(bool)`、`alreadySigned(bool)`、`todayReward(int64, 星币个数)`、`balance(int64, 分)`、`signedAt(string, RFC3339, 未签到省略)`

`Goods`：`goodsId(int64)`、`name(string)`、`brief(string)`、`description(string)`、`cover(string)`、`skus(GoodsSKU[])`

`GoodsSKU`：`skuId(int64)`、`name(string)`、`priceCoins(int64, 星币个数)`、`stock(int64)`、`sold(int64)`

`RedeemResult`：`orderNo(string)`、`balance(int64, 分)`、`stock(int64)`、`sold(int64)`

### 8.14 企划创意（ProjectIdea / IdeaVote / IdeaVoteResult / IdeaFeedPage）

`ProjectIdea`：
`ideaId(int64)`、`userId(int64)`、`userName(string)`、`userAvatar(string,omitempty)`、`title(string)`、`summary(string)`、`cover(string,omitempty)`、`categoryId(int64)`、`categoryName(string)`、`stage(int)`、`description(string,omitempty, 仅详情)`、`supportCount(int)`、`opposeCount(int)`、`coinIntentSum(int, 意向档位求和，不扣费)`、`supportRate(float64, 保留三位小数)`、`hotScore(float64)`、`tierDistribution(object, 仅详情/投票结果，键 "5"/"10"/"20")`、`reasonCounts(object, 仅详情，不支持理由标准 code 计数聚合，不含原文)`、`myVote(IdeaVote, 有会话且已表态时返回)`、`gmtCreate(毫秒时间戳)`

`IdeaVote`：`voteType(int, 1 支持 / 2 不支持)`、`coinTier(int, 0/5/10/20)`、`reasonCode(string, omitempty)`

`IdeaVoteResult`：`ideaId(int64)`、`supportCount(int)`、`opposeCount(int)`、`supportRate(float64)`、`coinIntentSum(int)`、`tierDistribution(object)`、`myVote(IdeaVote)`、`alreadyVoted(bool)`

`IdeaFeedPage`：`items(ProjectIdea[])`、`nextCursor(string, 首页/无下一页为空)`、`hasMore(bool)`

## 9. 端点与数据表映射

| 端点域 | 主要表 |
| --- | --- |
| 登录 / 发码 / 图形码 | `planstar_user.user_info`、`login_session`、`login_code` |
| Banner | `planstar_content.banner` |
| 企划列表 / 详情 / 档位 / 进度 | `planstar_catalog`（project、reward_tier、project_update、stretch_goal、project_faq、project_category、project_sub_tag、creator_profile） |
| 评价晒单 / 举报风控 | `planstar_catalog.project_review`、`project_report` |
| 榜单 | `planstar_catalog.project`（`funding_status` + `rank_weight` 索引） |
| 企划创意 / 表态 | `planstar_catalog.project_idea`、`project_idea_vote` |
| 支持记录 / 支持下单 | `planstar_order.pledge` |
| 组队支持 | `planstar_catalog.support_team`、`support_team_member` |
| 企划列表 / 热门 / 详情（legacy activity 表） | `planstar_catalog`（activity、activity_event、activity_category、city、venue、activity_lineup、activity_detail）+ `planstar_ticket`（ticket_category、ticket_group） |
| 动态 / 评论 | `planstar_community.feed`、`feed_media`、`review` |
| 消息 | `planstar_user.user_message` |
| 客服会话 / 消息 | `planstar_user.chat_thread`、`chat_message` |
| 支付单 / 退款单 | `planstar_payment.pay_order`、`order_refund` |
| 星币钱包 / 钱包流水 | `planstar_payment.wallet_account`、`wallet_sequence` |
| 每日签到 | `planstar_marketing.daily_sign_in` |
| 星币商城 / 兑换单 | `planstar_shop.goods`、`goods_sku`、`goods_order` |
| 订单 | `planstar_order.order_info` |
| 用户资料 | `planstar_user.user_info`、`user_statistic` |
| 地址 | `planstar_user.user_address` |
