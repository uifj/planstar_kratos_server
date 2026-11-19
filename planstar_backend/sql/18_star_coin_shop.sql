-- ========================================================================
--  planstar · 18_star_coin_shop.sql
--  星币经济增量 DDL + 商品种子数据（幂等，可重复执行）
--    1) planstar_marketing.daily_sign_in   每日签到（每人每自然日唯一）
--    2) planstar_shop                       星币商城库（goods / goods_sku / goods_order）
--    3) 种子商品：企划星UP主的祝福视频（1 个商品 + 2 个规格）
--  说明：本文件只做增量补丁，不修改 docs/sql 基线；种子数据重复执行不重置
--        库存（stock）与销量（sold）。
-- ========================================================================

SET NAMES utf8mb4;

-- ---------------------------------------------------------------------
-- 1. 每日签到
--    uk_daily_sign_in_user_date 保证同一用户同一自然日只有一条签到记录，
--    重复签到直接触发唯一键冲突（服务端据此返回幂等结果）。
-- ---------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `planstar_marketing`.`daily_sign_in` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `user_id` BIGINT UNSIGNED NOT NULL COMMENT '用户ID',
  `sign_date` DATE NOT NULL COMMENT '签到自然日（服务端本地时区）',
  `reward_coins` INT UNSIGNED NOT NULL DEFAULT 100 COMMENT '本次奖励星币',
  `gmt_create` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_daily_sign_in_user_date` (`user_id`,`sign_date`),
  KEY `idx_daily_sign_in_user_created` (`user_id`,`gmt_create`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='每日签到（每人每自然日唯一）' ROW_FORMAT=DYNAMIC;

-- ---------------------------------------------------------------------
-- 2. 星币商城库
--    商品、规格与星币订单独立成库，避免与交易库（planstar_order）耦合。
-- ---------------------------------------------------------------------
CREATE DATABASE IF NOT EXISTS `planstar_shop` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- 2.1 商品
CREATE TABLE IF NOT EXISTS `planstar_shop`.`goods` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `goods_id` BIGINT UNSIGNED NOT NULL COMMENT '商品ID（业务主键）',
  `goods_sn` VARCHAR(64) NOT NULL COMMENT '商品编码',
  `name` VARCHAR(255) NOT NULL COMMENT '商品名称',
  `brief` VARCHAR(1024) DEFAULT '' COMMENT '商品简介',
  `description` TEXT COMMENT '商品详情',
  `cover` VARCHAR(512) DEFAULT '' COMMENT '封面图 URL',
  `category` VARCHAR(64) DEFAULT 'video' COMMENT '商品分类',
  `status` TINYINT UNSIGNED NOT NULL DEFAULT 1 COMMENT '状态：1 上架 / 0 下架',
  `sort` INT NOT NULL DEFAULT 0 COMMENT '排序（越小越前）',
  `gmt_create` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `gmt_modified` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_goods_id` (`goods_id`),
  UNIQUE KEY `uk_goods_sn` (`goods_sn`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='星币商城商品' ROW_FORMAT=DYNAMIC;

-- 2.2 商品规格
CREATE TABLE IF NOT EXISTS `planstar_shop`.`goods_sku` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `sku_id` BIGINT UNSIGNED NOT NULL COMMENT '规格ID（业务主键）',
  `goods_id` BIGINT UNSIGNED NOT NULL COMMENT '商品ID（逻辑外键 goods.goods_id）',
  `sku_name` VARCHAR(128) NOT NULL COMMENT '规格名称',
  `price_coins` INT UNSIGNED NOT NULL COMMENT '价格（星币）',
  `stock` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '库存',
  `sold` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '已售数量',
  `status` TINYINT UNSIGNED NOT NULL DEFAULT 1 COMMENT '状态：1 可售 / 0 停售',
  `gmt_create` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `gmt_modified` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_sku_id` (`sku_id`),
  KEY `idx_goods_sku_goods_id` (`goods_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='星币商城商品规格' ROW_FORMAT=DYNAMIC;

-- 2.3 星币订单
CREATE TABLE IF NOT EXISTS `planstar_shop`.`goods_order` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `order_no` VARCHAR(64) NOT NULL COMMENT '订单号',
  `user_id` BIGINT UNSIGNED NOT NULL COMMENT '用户ID',
  `goods_id` BIGINT UNSIGNED NOT NULL COMMENT '商品ID（逻辑外键 goods.goods_id）',
  `sku_id` BIGINT UNSIGNED NOT NULL COMMENT '规格ID（逻辑外键 goods_sku.sku_id）',
  `sku_name` VARCHAR(128) NOT NULL COMMENT '规格名称（下单快照）',
  `price_coins` INT UNSIGNED NOT NULL COMMENT '支付星币（下单快照）',
  `status` VARCHAR(32) NOT NULL DEFAULT 'paid' COMMENT '状态：paid/completed/cancelled',
  `gmt_create` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `gmt_modified` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_goods_order_no` (`order_no`),
  KEY `idx_goods_order_user` (`user_id`,`gmt_create`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='星币商城订单' ROW_FORMAT=DYNAMIC;

-- ---------------------------------------------------------------------
-- 3. 种子数据
--    ON DUPLICATE KEY UPDATE 只刷新名称/价格/说明/状态，
--    绝不写 stock / sold，避免重跑脚本把已售库存重置。
-- ---------------------------------------------------------------------
INSERT INTO `planstar_shop`.`goods`
  (`goods_id`, `goods_sn`, `name`, `brief`, `description`, `category`, `status`, `sort`)
VALUES
  (1, 'PS-UP-VIDEO-001', '企划星UP主的祝福视频',
   '内容形式为UP主根据用户需求发布对应祝福内容，支持生日祝福、毕业祝福等场景',
   '内容形式为UP主根据用户需求发布对应祝福内容，支持生日祝福、毕业祝福等场景',
   'video', 1, 1)
ON DUPLICATE KEY UPDATE
  `name` = VALUES(`name`),
  `brief` = VALUES(`brief`),
  `description` = VALUES(`description`),
  `status` = VALUES(`status`);

INSERT INTO `planstar_shop`.`goods_sku`
  (`sku_id`, `goods_id`, `sku_name`, `price_coins`, `stock`, `sold`, `status`)
VALUES
  (1, 1, '30秒祝福视频', 1000, 500, 0, 1),
  (2, 1, '1分钟祝福视频', 1500, 500, 0, 1)
ON DUPLICATE KEY UPDATE
  `sku_name` = VALUES(`sku_name`),
  `price_coins` = VALUES(`price_coins`),
  `status` = VALUES(`status`);
