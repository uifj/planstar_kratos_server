# 企划星 Flutter 客户端 · 后端接口契约

> 交付对象：Go 后端同学
> 来源：**只从当前 `lib/` 代码提取**，不含推测；每个接口都标注了源文件与行号。
> 配套 SQL：[`docs/sql/00_all.sql`](../sql/00_all.sql)（建库）与 [`docs/sql/14_mock_seed.sql`](../sql/14_mock_seed.sql)（演示数据）。

## 1. 交付清单

| 文件 | 用途 |
| --- | --- |
| `docs/sql/00_all.sql` | 9 库 / 86 张业务表 / 0 物理外键，一次性建库 |
| `docs/sql/14_mock_seed.sql` | 40 张表 / 870 行 Flutter mock 演示数据 |
| `docs/sql/verify.sql` | 建库后的 `information_schema` 复核 |
| `docs/api/API_CONTRACT.md` | 本文件，客户端当前接口契约 |

## 2. 接入状态总览

| 域 | 客户端类 | 方法数 | 当前是否真的发请求 | 说明 |
| --- | --- | --- | --- | --- |
| 认证 | `PlanstarAuthApi` | 3 | **是** | 登录页直连；`RemoteAuthRepository` 已接线 |
| 演出库 | `PlanstarCatalogApi` | 4 | **否（开关）** | [catalog_providers.dart](../../lib/features/catalog/application/catalog_providers.dart#L16) `_demoMode = true`，改 `false` 即走网络 |
| 动态 | `PlanstarFeedApi` | 4 | **否** | UI 走 `demo_feed.dart` 内存数据 |
| 消息 | `PlanstarMessageApi` | 3 | **否** | UI 走 `demo_chat.dart` 内存实现 |
| 订单 | `PlanstarOrderApi` | 4 | **否** | 仓储是 `InMemoryTicketOrderRepository` |
| 用户/地址 | `PlanstarUserApi` | 5 | **否** | 「我的」资料取登录态；地址走 `demo_address.dart` |

**结论**：真正在跑的只有认证 3 个端点；其余 20 个已按真实后端形状写好客户端，但当前被 demo 数据替代。后端可按本契约先实现，客户端切换开关即可联调。

## 3. 通用约定

### 3.1 基础地址

| 环境 | Base URL | 说明 |
| --- | --- | --- |
| prod | `https://api.planstar.com/` | HTTPS |
| beta | `http://apibeta.planstar.com/` | 明文 HTTP，仅调试 |

- 路径全部**不带前导 `/`**（如 `v1/user/login`），客户端做 `base + path` 字符串拼接，所以 Base URL **必须以 `/` 结尾**。
- 连接 / 接收 / 发送超时均为 **20s**。
- 版本号存在历史错配：发码在 `v3`、提交登录在 `v1`，**客户端照抄，不要后端"顺手统一"**。

### 3.2 请求头

| 头 | 是否必带 | 说明 |
| --- | --- | --- |
| `User-Agent` | 是 | `PlanstarApi (com.planstar.app/9.9.39 Xiaomi developer-default Android/14)` |
| `dvid` | 有则带 | 设备标识；首次由服务端下发 |
| `Cookie: psSession=…` | 登录后必带 | 会话令牌，**不是 `Authorization: Bearer`** |
| `X-Longitude` / `X-Latitude` | 已定位才带 | 经纬度均 > 0 时才发送 |

来源：[ps_network_interceptor.dart](../../lib/app/network/ps_network_interceptor.dart#L40-L67)

### 3.3 会话与鉴权

- 登录成功后，**响应头** `psSession`（或 `Set-Cookie: psSession=…`）下发会话值；客户端两处都读。
- 之后每个请求带 `Cookie: psSession=<value>`。
- 任意接口返回 **HTTP 401** → 客户端触发全局登出。
- `dvid` 同理：响应头或 `Set-Cookie` 中读取并持久化。

### 3.4 响应格式（重要）

客户端使用 `PassthroughEnvelope`，**不解析 `{code, message, data}` 信封**：

- 成功：直接返回业务对象（裸 `List` 或裸 `Object`）。
- 失败：依赖 **HTTP 状态码**；登录失败时正文是 `text/plain`。
- 例外（历史行为，必须保留）：
  - `v3/user/sendLoginCode`：**HTTP 200 + `{"success":false,...}`**，不是错误码。
  - `v1/user/captcha`：返回 `image/png` 原始字节。

分页与包装也不统一，客户端已做兼容：

| 接口族 | 分页参数 | 响应包装 |
| --- | --- | --- |
| 演出库 | `page` / `pageSize`（默认 1 / 20） | 裸数组 |
| 动态 | `page` / `pageSize` / `nextPageKey` | 裸数组 **或** `{data, hasMore, nextPageKey}` |
| 消息/订单 | `pageIndex` / `pageSize` | 裸数组 **或** `{data/list, ...}` |

### 3.5 数据类型约定

| 类型 | 客户端期望 | 与 SQL 的差异 |
| --- | --- | --- |
| 金额 | **整数，单位：分**（`priceFrom/100` 转元；`walletBalance` 注释明确为分） | SQL 是 `DECIMAL(12,2)` 元，后端需 ×100 输出 |
| 时间 | ISO-8601 字符串（`DateTime.tryParse` 可解析） | SQL 是 `DATETIME`，按 `yyyy-MM-dd HH:mm:ss` 或 ISO 输出均可 |
| `birthday` | **毫秒时间戳 int**，`0` = 未设置 | SQL 是 `DATE`，需转换 |
| 布尔 | JSON `bool`，也兼容 `"true"` / `0/1` | SQL 是 `TINYINT UNSIGNED` |
| ID | 客户端统一 `toString()` 后按字符串处理 | SQL 是 `BIGINT UNSIGNED`，JSON 请输出数字或字符串（两者都兼容） |
| 枚举 | 数字 wire code（见 `dict_enum`） | 直接用 `dict_enum.code` |

## 4. 接口清单（23 个已实现端点）

### 4.1 认证（已接线）

| # | 方法 | 路径 | 参数 / Body | 响应 | 源 |
| --- | --- | --- | --- | --- | --- |
| 1 | POST | `v1/user/login` | body: `{ua, ct, pw?, code?, captcha?}` | 裸 `LoginBean`：`{needCaptcha, newUser, user}` | [planstar_auth_api.dart:43](../../lib/features/auth/data/planstar_auth_api.dart#L43) |
| 2 | POST | `v3/user/sendLoginCode` | body: `{ua, captcha?}` | `{success, needCaptcha, errorMsg}` | [planstar_auth_api.dart:46](../../lib/features/auth/data/planstar_auth_api.dart#L46) |
| 3 | GET | `v1/user/captcha` | — | `image/png` 二进制 | [planstar_auth_api.dart:49](../../lib/features/auth/data/planstar_auth_api.dart#L49) |

**登录请求字段**

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `ua` | string | 是 | 手机号 |
| `ct` | int | 是 | 客户端类型，Android 固定 `11` |
| `pw` | string | 二选一 | 密码登录 |
| `code` | string | 二选一 | 短信验证码登录 |
| `captcha` | string | 否 | 风控要求时的图形码 |

**`user` 字段**（`AuthUser`，来源 [auth_user.dart:75](../../lib/features/auth/domain/auth_user.dart#L75-L91)）

`userId(int)、userName(string)、avatar(string)、bio(string?)、gender(0/1/2)、birthday(ms int)、mobileNo(string?)、level(int)、followerNum(int)、followingNum(int)、wantWatchNum(int)、isDouliUser(bool)、badgeIcon(string?)`

> 注意：`needCaptcha=true` 时 `user` 为 `null`，客户端会重新弹图形码并重放原请求。

### 4.2 演出库（`_demoMode` 开关）

| # | 方法 | 路径 | 参数 | 响应 | 源 |
| --- | --- | --- | --- | --- | --- |
| 4 | GET | `v1/home/banners` | — | `[{id, title, image, url?, showId?}]` | [planstar_catalog_api.dart:26](../../lib/features/catalog/data/planstar_catalog_api.dart#L26) |
| 5 | GET | `v1/activities/hot` | `page`, `pageSize` | `ShowSummary[]` | [planstar_catalog_api.dart:44](../../lib/features/catalog/data/planstar_catalog_api.dart#L44) |
| 6 | GET | `v1/activities` | `page`, `pageSize`, `categoryId?`, `city?` | `ShowSummary[]` | [planstar_catalog_api.dart:71](../../lib/features/catalog/data/planstar_catalog_api.dart#L71) |
| 7 | GET | `v3/activities/{id}` | path `id` | `ShowSummary` | [planstar_catalog_api.dart:87](../../lib/features/catalog/data/planstar_catalog_api.dart#L87) |

**`ShowSummary` 响应字段**（来源 [show_summary.dart:75](../../lib/features/catalog/domain/show_summary.dart#L75)）

| JSON 字段 | 类型 | 说明 | SQL 来源 |
| --- | --- | --- | --- |
| `id` / `activityId` | string/int | 演出 ID | `activity.activity_id` |
| `title` / `name` | string | 标题 | `activity.name` |
| `artist` / `star` | string | 艺人 / 主办 | `activity_lineup` / `actor` |
| `categoryId` / `category` | int | 分类 ID | `activity.category_id` |
| `categoryName` | string | 分类名 | `activity_category.name` |
| `city` | string | 城市名 | `city.city_name` |
| `venue` / `venueName` | string | 场馆名 | `venue.name` |
| `venueAddress` | string | 场馆地址 | `venue.address` |
| `firstSessionAt` / `showTime` | datetime | 首场时间 | `activity_event.start_time` 最小值 |
| `priceFrom` / `minPrice` | int（分） | 起价 | `activity.low_price` ×100 |
| `listPriceFrom` / `originalPrice` | int（分）? | 划线价 | 优惠规则计算 |
| `discountPermille` / `discount` | int? | 千分比折扣，`980`=9.8 折 | `activity_campaign` 计算 |
| `tags` | string[] | 电子票 / 选座 / 无票赔付 | `activity` 标签字段聚合 |
| `wishCount` / `wannaSeeCount` | int | 想看数 | `activity.wish_count` |
| `soldOut` / `status` | bool | 售罄 | `activity_event.sale_status` |
| `proxyBuy` | bool | 代抢标识 | `proxy_buy_config` 是否存在 |
| `rating` / `ratingCount` | double / int | 评分 | `activity.rating` |
| `duration` | string | 时长文案 | `activity_detail` |
| `intro` / `description` | string | 演出介绍 | `activity_detail.description` |
| `sessions` | `{startsAt/showTime, tierSummary, soldOut}[]` | 场次 | `activity_event` |
| `tiers` | `{name/tierName, price(分), note, remaining}[]` | 票档 | `ticket_category` / `ticket_group` |
| `lineup` | string[] | 阵容 | `activity_lineup` |
| `purchaseNotice` | string[] | 购票须知 | `activity_detail.purchase_notice` |

### 4.3 动态

| # | 方法 | 路径 | 参数 | 响应 | 源 |
| --- | --- | --- | --- | --- | --- |
| 8 | GET | `v4/feed/hot` | `page`, `pageSize`, `nextPageKey?` | `FeedPost[]` 或 `{data, hasMore, nextPageKey}` | [planstar_feed_api.dart:31](../../lib/features/discover/data/planstar_feed_api.dart#L31) |
| 9 | GET | `v1/feed` | `page`, `pageSize`, `nextPageKey?` | 同上 | [planstar_feed_api.dart:49](../../lib/features/discover/data/planstar_feed_api.dart#L49) |
| 10 | GET | `v1/tweets/{id}` | path `id` | `FeedPost` | [planstar_feed_api.dart:63](../../lib/features/discover/data/planstar_feed_api.dart#L63) |
| 11 | GET | `v1/userReply` | `tweetId` | `FeedComment[]` | [planstar_feed_api.dart:75](../../lib/features/discover/data/planstar_feed_api.dart#L75) |

**`FeedPost`**（来源 [feed_post.dart:19](../../lib/features/discover/domain/feed_post.dart#L19-L35)）
`id / tweetId、title / content、author / userName、viewCount / readCount、publishedAt / createTime、tag、showId / activityId、likeCount / likeNum、hasVideo / video、body`

**`FeedComment`**：`id、author / userName、text、publishedAt / createTime、likeCount`

> 客户端**只实现了评论列表**；点赞、取消赞、发评论在文件头注释里列出，但当前没有方法，见第 6 节。

### 4.4 消息

| # | 方法 | 路径 | 参数 | 响应 | 源 |
| --- | --- | --- | --- | --- | --- |
| 12 | GET | `v1/messages` | `pageIndex`, `pageSize`, `type?` | `MessageThread[]` 或 `{data: [...]}` | [planstar_message_api.dart:33](../../lib/features/message/data/planstar_message_api.dart#L33) |
| 13 | GET | `v1/messages/unreadCount` | — | `int` 或 `{count/unreadCount}` | [planstar_message_api.dart:64](../../lib/features/message/data/planstar_message_api.dart#L64) |
| 14 | POST | `v1/messages/read` | body `{messageId}` | 空 | [planstar_message_api.dart:79](../../lib/features/message/data/planstar_message_api.dart#L79) |

`type`：`1` 系统通知 / `2` 回复我的 / `3` 赞和关注。
`MessageThread`：`id / messageId、title / subject、preview / content、updatedAt / updateTime / createTime、unreadCount / unread、avatar、isOfficial / official`

### 4.5 订单 / 支付

| # | 方法 | 路径 | 参数 | 响应 | 源 |
| --- | --- | --- | --- | --- | --- |
| 15 | GET | `v1/order/list` | `pageIndex`, `pageSize`, `status?`, `orderType?` | `TicketOrder[]` 或 `{data/list: [...]}` | [planstar_order_api.dart:39](../../lib/features/order/data/planstar_order_api.dart#L39) |
| 16 | GET | `v1/order/detail` | `orderNumber` | `TicketOrder` | [planstar_order_api.dart:71](../../lib/features/order/data/planstar_order_api.dart#L71) |
| 17 | POST | `v1/unifiedOrder/confirm` | body（客户端透传，见下） | `object` | [planstar_order_api.dart:86](../../lib/features/order/data/planstar_order_api.dart#L86) |
| 18 | POST | `v3/unifiedOrder/makeOrder` | body（客户端透传） | `object` | [planstar_order_api.dart:99](../../lib/features/order/data/planstar_order_api.dart#L99) |

`TicketOrder` 解析用到的字段（[planstar_order_api.dart:109](../../lib/features/order/data/planstar_order_api.dart#L109)）：

`orderId / id、orderNumber、showId、sessionId、createTime、orderType(默认 1)、buyType(默认 1)`

> `confirm` / `makeOrder` 的请求体当前由调用方原样透传，**客户端未固化字段**。后端需与订单域对齐（建议以 `order_confirm_snapshot` 的列结构为契约蓝本）。

### 4.6 用户 / 地址

| # | 方法 | 路径 | 参数 | 响应 | 源 |
| --- | --- | --- | --- | --- | --- |
| 19 | GET | `v1/user/` | — | `UserProfileDto` | [planstar_user_api.dart:27](../../lib/features/user/data/planstar_user_api.dart#L27) |
| 20 | GET | `v1/myaddress` | — | `ShippingAddress[]` | [planstar_user_api.dart:41](../../lib/features/user/data/planstar_user_api.dart#L41) |
| 21 | POST | `v1/myaddress` | body `{name, mobile, region, detail, isDefault}` | `ShippingAddress` | [planstar_user_api.dart:56](../../lib/features/user/data/planstar_user_api.dart#L56) |
| 22 | DELETE | `v1/myaddress/{id}` | path `id` | 空 | [planstar_user_api.dart:69](../../lib/features/user/data/planstar_user_api.dart#L69) |
| 23 | POST | `v1/myaddress/default/{id}` | path `id` | 空 | [planstar_user_api.dart:78](../../lib/features/user/data/planstar_user_api.dart#L78) |

**`UserProfileDto`**（[planstar_user_api.dart:112](../../lib/features/user/data/planstar_user_api.dart#L112)）
`userId / id、nickname / userName、avatar / headImg、mobile / phone、level / userLevel、levelLabel、wannaSeeCount / followActivityCount、followCount / followUserCount、couponCount、walletBalance / balance(分)`

**`ShippingAddress`** 解析兼容 `name/consignee`、`mobile/phone`、`region` 或 `province+city+district`、`detail/address`、`isDefault/default`。

> 注意：地址接口路径是 `v1/myaddress`，而 `address_providers.dart:16` 的注释写的是 `v1/address/*`（旧规划）。**以实现代码为准**。

## 5. 接口 → SQL 表映射

| 接口 | 主要表 | 关联表 |
| --- | --- | --- |
| 登录 / 发码 / 图形码 | `user_info`、`login_session` | `real_name_info` |
| Banner | `banner` | `home_floor` |
| 演出列表 / 热门 / 详情 | `activity`、`activity_event` | `city`、`venue`、`activity_category`、`activity_sub_tag`、`activity_detail`、`activity_lineup`、`activity_gallery`、`ticket_category`、`ticket_group`、`actor`、`proxy_buy_config` |
| 动态列表 / 详情 | `feed`、`feed_media` | `user_info`、`topic`、`user_follow`、`like_record` |
| 评论列表 | `review`、`review_reply` | `user_info`、`feed` |
| 消息列表 / 未读数 / 已读 | `user_message`、`chat_thread`、`chat_message` | `user_info` |
| 订单列表 / 详情 | `order_info`、`order_item` | `order_event`、`order_delivery`、`payment`、`refund`、`order_postage` |
| 确认订单 / 下单 | `order_confirm_snapshot`、`order_info`、`order_item` | `ticket_group`、`ticket_seat`、`coupon`、`gift_card`、`payment` |
| 用户资料 | `user_info`、`user_statistic` | `wallet_account`、`coupon` |
| 地址 | `user_address` | `city`、`country_area` |

## 6. 未实现 / 仅注释的端点（后端可先不做）

| 端点 | 出处 | 状态 |
| --- | --- | --- |
| `POST v1/userLike` / `DELETE v1/userLike` | [planstar_feed_api.dart:15-16](../../lib/features/discover/data/planstar_feed_api.dart#L15-L16) | 仅注释，无方法 |
| `POST v1/userReply`（发评论） | 同上第 18 行 | 仅注释，无方法 |
| `DELETE v1/messages/{id}` | [planstar_message_api.dart:14](../../lib/features/message/data/planstar_message_api.dart#L14) | 仅注释，无方法 |
| `GET v1/unifiedPay/payDetail`、`POST v1/unifiedPay/pay` | [planstar_order_api.dart:21-22](../../lib/features/order/data/planstar_order_api.dart#L21-L22) | 仅注释，支付页走本地模拟 |
| `GET v1/user/users`（批量用户） | [planstar_user_api.dart:13](../../lib/features/user/data/planstar_user_api.dart#L13) | 仅注释，无方法 |
| `PUT v1/user/baseInfo`（修改资料） | [planstar_user_api.dart:14](../../lib/features/user/data/planstar_user_api.dart#L14) | 仅注释，无方法 |
| `POST v1/feed/publish` | [feed_providers.dart:16](../../lib/features/discover/application/feed_providers.dart#L16) | 规划注释，发布仅本地插入 |
| `v1/address/list|add|delete|setDefault` | [address_providers.dart:16-17](../../lib/features/user/application/address_providers.dart#L16-L17) | 旧规划，已被 `v1/myaddress` 取代 |
| `v1/user/info`、`v1/coupon/count`、`v1/wallet/balance` | [profile_providers.dart:14-15](../../lib/features/user/application/profile_providers.dart#L14-L15) | 规划注释，当前用登录态 + 演示值 |

## 7. 后端落地建议

1. **先实现认证 3 个端点**：登录页是唯一真实链路，`psSession` 必须同时出现在响应头和 `Set-Cookie`。
2. **金额统一按分输出**：SQL 存元（`DECIMAL(12,2)`），接口输出 `long` 分。
3. **枚举直接用 `dict_enum.code`**：不要另造一套 code，客户端 `OrderType` / `OrderStatus` 都按 wire code 解析。
4. **保留两套错误风格**：登录失败 `5xx + text/plain`；发码 `200 + success:false`。切换成统一信封会直接打断客户端解析。
5. **分页参数别统一**：`page` 与 `pageIndex` 各接口不同，客户端已分别适配；后端如实实现即可，统一反而需要改客户端。
6. **无外键**：订单、支付、库存的一致性靠事务 + 幂等键（`order_number`、`payment` 支付单号）保证，SQL 层不做级联。
