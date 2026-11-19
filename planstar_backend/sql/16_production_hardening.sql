-- ========================================================================
--  planstar · 16_production_hardening.sql
--  订单链路与 ES 读模型生产化改造配套迁移（幂等，可重复执行）
--    1) order_info.order_number 唯一索引
--    2) order_idempotency 幂等键表（唯一约束 user_id + idem_key）
--    3) ticket_category.stock 可售库存字段 + 首次添加时按票组/票数回填
--  说明：docs/sql/05_order.sql 为基线结构，本文件只做增量补丁，不修改基线。
-- ========================================================================

-- ---------------------------------------------------------------------
-- 1. order_info.order_number 唯一索引（并发下订单号必须唯一）
-- ---------------------------------------------------------------------
SET @has_order_number_uk := (
  SELECT COUNT(*) FROM information_schema.statistics
   WHERE table_schema = 'planstar_order' AND table_name = 'order_info'
     AND index_name = 'uk_order_info_order_number'
);
SET @ddl := IF(@has_order_number_uk = 0,
  'ALTER TABLE `planstar_order`.`order_info` ADD UNIQUE KEY `uk_order_info_order_number` (`order_number`)',
  'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

-- ---------------------------------------------------------------------
-- 2. order_idempotency 幂等键表
--    同一用户同一 Idempotency-Key 只允许一条记录，重复请求返回已有订单。
-- ---------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `planstar_order`.`order_idempotency` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `idem_key` VARCHAR(128) NOT NULL COMMENT '客户端幂等键 | 请求头 Idempotency-Key',
  `user_id` BIGINT UNSIGNED NOT NULL COMMENT '用户ID',
  `order_id` BIGINT UNSIGNED NOT NULL COMMENT '已创建订单ID | order_info.order_id',
  `order_number` VARCHAR(64) NOT NULL COMMENT '已创建订单号 | order_info.order_number',
  `gmt_create` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间（created_at）',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_order_idempotency_user_key` (`user_id`, `idem_key`),
  KEY `idx_order_idempotency_order_id` (`order_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='订单创建幂等键（生产化加固）' ROW_FORMAT=DYNAMIC;

-- ---------------------------------------------------------------------
-- 3. ticket_category.stock 可售库存
--    下单事务执行 UPDATE ... SET stock = stock - ? WHERE stock >= ? 扣减。
-- ---------------------------------------------------------------------
SET @has_stock := (
  SELECT COUNT(*) FROM information_schema.columns
   WHERE table_schema = 'planstar_ticket' AND table_name = 'ticket_category' AND column_name = 'stock'
);
SET @added_stock := (@has_stock = 0);
SET @ddl := IF(@added_stock,
  'ALTER TABLE `planstar_ticket`.`ticket_category` ADD COLUMN `stock` INT NOT NULL DEFAULT 0 COMMENT ''可售库存（下单事务原子扣减）'' AFTER `tickets_num`',
  'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

-- 仅在本次新增字段时回填，避免覆盖运营已维护的库存。
-- 优先票组在售余量（count - sold），否则回退回报档位 tickets_num。
UPDATE `planstar_ticket`.`ticket_category` tc
   SET tc.`stock` = GREATEST(
         COALESCE((
           SELECT SUM(GREATEST(tg.`count` - tg.`sold`, 0))
             FROM `planstar_ticket`.`ticket_group` tg
            WHERE tg.`ticket_category_id` = tc.`ticket_category_id`
              AND tg.`is_saling` = 1
         ), 0),
         COALESCE(tc.`tickets_num`, 0),
         0)
 WHERE @added_stock = 1;
