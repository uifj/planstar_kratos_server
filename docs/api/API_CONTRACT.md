> 适用版本：planstar 单体 · 最后核对：2026-11-07 · 维护：后端组

# Planstar 客户端接口契约

## 1. 契约范围与事实源

- **唯一事实源**：`planstar/internal/service/service.go` 的 `Register` 路由表，以及同目录各 handler。
- 服务监听 `8098`；所有路径区分大小写，使用 `METHOD /path` 精确匹配。
- 当前注册 24 条路由，对应 **23 个唯一端点**：`GET /v1/user/` 是 `GET /v1/user` 的尾斜杠别名，复用同一 handler，本契约只记录一次。
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

不得在契约中混用元与分；`priceFrom`、`walletBalance` 等字段的客户端解析均按分处理。

## 4. 分页

| 端点族 | 参数 | 默认值 | 上限/异常处理 |
| --- | --- | --- | --- |
| 演出库、动态 | `page`、`pageSize` | `page=1`、`pageSize=20` | `page<=0` 归 1；`pageSize<=0` 或 `>100` 回落 20 |
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
- 需要登录的端点：订单 4 个、消息 3 个、用户资料/地址 6 个；其余目录与动态端点匿名可访问。
- 发码风控：同一手机号默认 60 秒冷却，同一 IP 默认每分钟 10 次（`auth.send_code_cooldown_seconds` / `auth.send_code_ip_limit_per_minute`），超限返回 `429 RATE_LIMITED`；IP 取 TCP 对端地址，不信任 `X-Forwarded-For`，经反向代理部署时可能被计为同一 IP。
- 登录失败风控：连续失败达到 `auth.login_fail_captcha_threshold`（默认 3）后要求图形码。发码/失败计数为**进程内**语义，多实例独立计数。

## 6. 版本错配说明

`v1` / `v3` / `v4` 是**兼容历史客户端的既有路径**，不是后端路由版本策略：

- 发码是 `POST /v3/user/sendLoginCode`，登录是 `POST /v1/user/login`。
- 演出详情是 `GET /v3/activities/{id}`，列表是 `GET /v1/activities`。
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

### 7.2 演出库

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
- 请求：路径参数 `id`（int64，演出 ID）
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
- 说明：价格来自 `planstar_ticket` 在售票档最低价（`ticket_group.sale_price` → `ticket_category.low_price` → `catalog.activity.low_price` 回退），**不接受客户端指定金额**。快照写入 Redis，未配置 Redis 时降级为进程内 TTL 缓存（仅单实例可用）
- 错误：`401 UNAUTHORIZED`、`400 BAD_REQUEST`、`409 SOLD_OUT`、`409 LIMIT_EXCEEDED`、`413 PAYLOAD_TOO_LARGE`、`500 INTERNAL`

POST /v3/unifiedOrder/makeOrder

- 鉴权：是
- 请求体（字段兼容多别名，依次取首个可解析值）：
  | 逻辑字段 | 允许的键 | 类型 | 必填 | 说明 |
  | --- | --- | --- | --- | --- |
  | 演出 ID | `activityId` / `activity_id` / `showId` | int64 | 是 | `>0` |
  | 场次 ID | `activityEventId` / `activity_event_id` / `sessionId` | int64 | 否 | 落 `activity_event_id` |
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
  | 票档 ID | `ticketCategoryId` / `ticket_category_id` / `ticketId` | int64 | 否 | 定价以服务端在售票档为准 |
- 请求头：`Idempotency-Key`（可选但推荐，同 key 24 小时内返回首次创建的订单）
- 响应：`TicketOrder`，字段见第 8.7 节
- 说明：服务端按报价快照或当前权威价格重算金额，**忽略客户端金额的定价作用**。创建在单个事务内完成：幂等查询 → 原子扣减 `planstar_ticket.ticket_category.stock`（影响 0 行即 `SOLD_OUT`）→ 写 `order_info` → 写 `order_idempotency`。订单 ID 由应用侧生成（毫秒时间戳 + 随机位），`order_number` 为 `PN + 毫秒 + 6 位随机` 且有唯一索引；不存在 `MAX(id)+1` 逻辑。演出数据通过 `CatalogReader` 端口从 catalog 库读取，不再跨库直查
- 错误：`401 UNAUTHORIZED`、`400 BAD_REQUEST`（缺 `activityId`/`quoteId`/`amount` 或 body 非法）、`409 PRICE_CHANGED`、`409 SOLD_OUT`、`409 LIMIT_EXCEEDED`、`409 QUOTE_EXPIRED`、`413 PAYLOAD_TOO_LARGE`、`500 INTERNAL`

### 7.6 用户与地址

GET /v1/user

- 鉴权：是
- 请求：无参数
- 响应：`UserProfile`，字段见第 8.8 节
- 别名：`GET /v1/user/`（尾斜杠）复用同一 handler，响应完全一致
- 错误：`401 UNAUTHORIZED`、`404 NOT_FOUND`、`500 INTERNAL`

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

## 8. 响应模型

字段名以 `internal/biz/models.go` 为准；同义字段为兼容客户端而重复输出。

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

## 9. 端点与数据表映射

| 端点域 | 主要表 |
| --- | --- |
| 登录 / 发码 / 图形码 | `planstar_user.user_info`、`login_session`、`login_code` |
| Banner | `planstar_content.banner` |
| 演出列表 / 热门 / 详情 | `planstar_catalog`（activity、activity_event、activity_category、city、venue、activity_lineup、activity_detail）+ `planstar_ticket`（ticket_category、ticket_group） |
| 动态 / 评论 | `planstar_community.feed`、`feed_media`、`review` |
| 消息 | `planstar_user.user_message` |
| 订单 | `planstar_order.order_info` |
| 用户资料 | `planstar_user.user_info`、`user_statistic` |
| 地址 | `planstar_user.user_address` |
