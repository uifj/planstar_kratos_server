# ADR-0009: 本地演示数据缺口登记（订单 / 退款 / 商城）

- **状态**: ACCEPTED（登记为本地种子缺口，不伪造业务数据）
- **分类**: DEBT（关联 DEF / OPS）
- **优先级**: P2
- **影响维度**: 本地联调 / 接口冒烟 / 测试可覆盖性（不影响生产运行）
- **日期**: 2026-11-19
- **来源**: `planstar_api_check_report.md`（工作区根目录；本地 57 条业务 + 23 条运维路由全量检查，2026-11-19）
- **修复 commit**: —（本 ADR 只登记缺口；不新增订单/退款/商品业务数据）

## 背景（Context）

2026-11-19 在 Windows + Docker Desktop 本地栈（`docker-compose.local.yml`：planstar-api / MySQL / Redis）完成全量接口检查，发现以下三处**本地种子数据**与运行时代码不匹配。接口本身可正常响应，但只能走 4xx 分支，无法在本地验证成功链路。

| ID | 对象 | 现状（证据） | 受影响接口 | 实际表现 |
|---|---|---|---|---|
| L1 | `planstar_order.order_info` | `SELECT COUNT(*)` = **0** | `GET /v1/order/list`、`GET /v1/order/detail`、`POST /v1/unifiedPay/pay`、`GET /v1/unifiedPay/payDetail`、`POST/GET /v1/order/{id}/refund` | 列表为空；详情/支付/退款按契约返回 **404**；下单/确认报价因缺少活动票档返回业务 **400/404** |
| L2 | `planstar_payment.order_refund` | `SELECT COUNT(*)` = **0**；同库 `refund` 表虽有 11 行但状态均为 `success` | `POST /admin/refunds/{id}/review` | 审核查询按待审核单过滤，返回 **404** |
| L3 | `planstar_shop.goods` | 仅 1 行，`goods_id=1` 且 `status=0`（下架） | `GET /v1/shop/goods/1`、`POST /v1/shop/goods/1/skus/1/redeem` | 详情与兑换按 `status=1` 过滤，返回 **404** |

补充事实：

1. 上述响应均为**可解释的业务 4xx**，检查期间 **0 个 5xx、0 次超时**；报告第五节已逐条记录。
2. `order_info` 为空使支付、退款、支持单状态推进等链路在本地没有可用的数据锚点；强行补单会牵涉 `order → payment → order_refund → refund` 多表一致性与金额口径。
3. `goods.status=0` 是商品对外不可见的正常业务开关，不是接口缺陷。
4. 同一批次已单独处理与本 ADR 相邻的两项：`planstar_catalog.activity` 最小兼容种子（见 `docs/sql/14_mock_seed.sql` 末尾）与 `.env.local.example` 的 `PLANSTAR_OPS_TOKEN` 示例值；两者不在本 ADR 的三个缺口内。

## 决策（Decision）

1. **只登记，不伪造数据**：不在本批次写入订单、支付、退款、兑换记录，避免为“让接口返回 200”而制造跨表不一致的假数据。
2. **保持契约行为**：订单/支付/退款/商城在资源不存在或不可售时继续返回 404/400；本地冒烟脚本把这三类响应标记为“预期业务 4xx”，不得计为服务异常。
3. **本地可测性以显式缺口呈现**：CI 或人工检查若需要成功链路，必须先补齐符合不变量的种子数据，而不是放宽断言。
4. **不改变生产配置**：本 ADR 不涉及生产库、生产编排或发布流程；生产数据由真实业务产生。
5. **关闭条件**：任一项在本地补齐并通过成功链路验证后，必须在本 ADR 把对应 ID 标为 `FIXED` 并附 SQL/commit 证据。

## 后果（Consequences）

**负面**

- 本地无法端到端验证：订单详情/支付/退款、退款审核、商品详情/兑换的成功路径。
- 接口检查报告中这些路径只能覆盖鉴权、参数校验与 404 映射。

**正面**

- 不会产生支付金额、退款状态、库存与支持单之间互相矛盾的演示数据。
- 缺口被显式记录，后续补种子时有明确的表、状态与不变量约束。

**中性**

- 生产功能不受影响；缺口仅存在于 `docs/sql` 演示种子层。

## 补齐时必须满足的不变量

1. `order_info` 的 `user_id`、`activity_id`、金额、状态必须与对应 `payment`、`refund` 记录一致；金额遵循“库内元、接口分”。
2. `order_refund` 至少保留 1 条待审核单，`status` 与审核状态机一致，才能覆盖 `POST /admin/refunds/{id}/review` 的成功路径。
3. 商品种子需至少 1 条 `goods.status=1` 且 SKU `status=1`、`stock>0`，并保证钱包余额足够覆盖 `price_coins`，否则兑换仍会因余额不足返回业务 4xx。
4. 新增种子必须幂等（`INSERT ... ON DUPLICATE KEY UPDATE` 或 `INSERT IGNORE`），避免重复执行放大数据。

## 验证（Verification）

```sql
SELECT COUNT(*) FROM planstar_order.order_info;                 -- 0
SELECT COUNT(*) FROM planstar_payment.order_refund;             -- 0
SELECT goods_id, status FROM planstar_shop.goods;               -- 1, 0
```

```text
GET  /v1/order/detail?orderNumber=NO_SUCH_ORDER  -> 404
POST /v1/unifiedPay/pay                          -> 404
POST /admin/refunds/14015/review                 -> 404
GET  /v1/shop/goods/1                            -> 404
```

完整请求/响应与耗时见工作区根目录 `planstar_api_check_report.md` §5.2 / §5.3。

## 关联

- `docs/sql/14_mock_seed.sql`：企划众筹演示数据；文件末尾已补充 `activity` 最小兼容种子（解决活动详情 404 与 Banner 为空）。
- `planstar_backend/sql/15_auth_support.sql`：`banner` / `review` 依赖 `activity`，活动种子补齐后其 `ON DUPLICATE KEY` 语句可正常落 Banner。
- `planstar_backend/.env.local.example`：已增加 `PLANSTAR_OPS_TOKEN` 本地示例，避免 9108 运维接口默认全部 401。
- `docs/ADR/ADR-0008-audit-debt-deferral.md`：D1/D2/D3 已记录支付与退款链路的既有阻断；本 ADR 仅补充“本地种子不可测”这一事实。
