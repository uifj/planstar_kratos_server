-- ========================================================================
--  planstar · 17_pay_refund_chat.sql
--  支付单 / 退款单 / 客服会话增量 DDL + 联调示例数据（幂等，可重复执行）
--    1) planstar_payment.pay_order      支付单（订单号 + 渠道唯一，支持过期）
--    2) planstar_payment.order_refund   退款单（订单唯一，保证重复提交幂等）
--    3) planstar_user.chat_thread       会话线程（基线已存在，IF NOT EXISTS 保证兼容）
--    4) planstar_user.chat_message      会话消息（基线已存在，IF NOT EXISTS 保证兼容）
--  说明：docs/sql 为基线结构；本文件只做增量补丁，不修改基线。
-- ========================================================================

SET NAMES utf8mb4;

-- ---------------------------------------------------------------------
-- 1. 支付单
--    同一订单同一渠道只允许一条支付单（uk_pay_order_order_channel），
--    重复 pay 请求返回已有支付单；expire_at 之后仍为 pending 视为 expired。
-- ---------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `planstar_payment`.`pay_order` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `pay_no` VARCHAR(64) NOT NULL COMMENT '支付单号 | POST v1/unifiedPay/pay',
  `order_number` VARCHAR(64) NOT NULL COMMENT '订单号 | order_info.order_number',
  `order_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '订单ID（逻辑外键 order_info.order_id）',
  `user_id` BIGINT UNSIGNED NOT NULL COMMENT '用户ID',
  `channel` VARCHAR(32) NOT NULL COMMENT '支付渠道：wechat/alipay/unionpay/balance',
  `amount` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '支付金额（元）',
  `status` VARCHAR(16) NOT NULL DEFAULT 'pending' COMMENT '状态：pending/success/failed/expired',
  `third_trade_no` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '第三方交易号',
  `pay_params` JSON COMMENT '支付参数（占位结构，字段留给后续 SDK）',
  `expire_at` DATETIME NOT NULL COMMENT '支付单过期时间',
  `paid_at` DATETIME DEFAULT NULL COMMENT '支付完成时间',
  `gmt_create` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `gmt_modified` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_pay_order_pay_no` (`pay_no`),
  UNIQUE KEY `uk_pay_order_order_channel` (`order_number`, `channel`),
  KEY `idx_pay_order_user_id` (`user_id`),
  KEY `idx_pay_order_status_expire` (`status`, `expire_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='支付单（订单+渠道幂等）' ROW_FORMAT=DYNAMIC;

-- ---------------------------------------------------------------------
-- 2. 退款单
--    order_id 唯一：同一订单重复申请退款返回同一退款单。
-- ---------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `planstar_payment`.`order_refund` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `refund_no` VARCHAR(64) NOT NULL COMMENT '退款单号 | POST v1/order/{id}/refund',
  `order_id` BIGINT UNSIGNED NOT NULL COMMENT '订单ID（逻辑外键 order_info.order_id）',
  `order_number` VARCHAR(64) NOT NULL COMMENT '订单号 | order_info.order_number',
  `user_id` BIGINT UNSIGNED NOT NULL COMMENT '用户ID',
  `reason_code` VARCHAR(64) NOT NULL DEFAULT 'other' COMMENT '退款原因码：notNeeded/boughtWrong/tooExpensive/foundBetter/other',
  `reason_text` VARCHAR(1024) NOT NULL DEFAULT '' COMMENT '退款原因补充',
  `refundable_amount` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '可退金额（元）',
  `status` VARCHAR(16) NOT NULL DEFAULT 'applied' COMMENT '状态：applied/processing/success/rejected',
  `expected_arrival_days` INT UNSIGNED NOT NULL DEFAULT 3 COMMENT '预计到账天数',
  `gmt_create` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `gmt_modified` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_order_refund_refund_no` (`refund_no`),
  UNIQUE KEY `uk_order_refund_order_id` (`order_id`),
  KEY `idx_order_refund_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='退款单（订单唯一幂等）' ROW_FORMAT=DYNAMIC;

-- ---------------------------------------------------------------------
-- 3. 会话线程（与 docs/sql/02_user.sql 基线一致）
-- ---------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `planstar_user`.`chat_thread` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `thread_id` BIGINT UNSIGNED NOT NULL COMMENT '会话ID | Flutter MessageThread.id',
  `user_id` BIGINT UNSIGNED NOT NULL COMMENT '用户ID',
  `title` VARCHAR(256) COMMENT '会话标题 | MessageThread.title',
  `preview` VARCHAR(1024) COMMENT '最后一条预览 | MessageThread.preview',
  `unread_count` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '未读数 | MessageThread.unreadCount',
  `avatar_url` VARCHAR(512) COMMENT '头像（开源图床 URL） | MessageThread.avatarFallbackUrl',
  `is_official` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '是否官方客服 | MessageThread.isOfficial',
  `visitor_token` VARCHAR(256) COMMENT '环信 visitor token | v1/chat/visitor',
  `gmt_create` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `gmt_modified` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_chat_thread_thread_id` (`thread_id`),
  KEY `idx_chat_thread_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='客服 / 会话线程 | ChatService v1/chat/user · v1/chat/visitor + Flutter MessageThread（原 chat_threads）' ROW_FORMAT=DYNAMIC;

-- ---------------------------------------------------------------------
-- 4. 会话消息（与 docs/sql/02_user.sql 基线一致）
-- ---------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `planstar_user`.`chat_message` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `message_id` BIGINT UNSIGNED NOT NULL COMMENT '消息ID | ChatMessage.id',
  `thread_id` BIGINT UNSIGNED NOT NULL COMMENT '会话ID',
  `text` TEXT COMMENT '消息文本 | ChatMessage.text',
  `is_from_me` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '是否自己发出 | ChatMessage.fromMe',
  `sender_name` VARCHAR(128) COMMENT '发送者名 | ChatMessage.senderName',
  `gmt_create` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `gmt_modified` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_chat_message_message_id` (`message_id`),
  KEY `idx_chat_message_thread_id` (`thread_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='会话消息 | Flutter ChatMessage（原 chat_messages）' ROW_FORMAT=DYNAMIC;

