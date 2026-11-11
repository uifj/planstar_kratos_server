-- ========================================================================
--  sql · 06_payment.sql（S2 全量规范重构版）
--  规范: 单数表名 / id BIGINT UNSIGNED 自增 / gmt_create,gmt_modified DATETIME
--        is_xxx TINYINT UNSIGNED / 金额 DECIMAL / 无物理外键 / 无冗余索引
--  表: payment, payment_channel, refund, wallet_account, wallet_sequence, gift_card, gift_card_usage
-- ========================================================================

USE `planstar_payment`;

-- ----------------------------------------------------------------------
--  域 4 · 交易与支付（order / payment / refund）
-- ----------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS `payment` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `pay_id` BIGINT UNSIGNED NOT NULL COMMENT '支付单ID | 推断',
  `order_id` BIGINT UNSIGNED NOT NULL COMMENT '订单ID',
  `left_time` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '剩余支付秒数（倒计时基线）',
  `pay_amount` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '应付金额',
  `balance` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '可用余额',
  `title` VARCHAR(512) COMMENT '标题',
  `value` VARCHAR(512) COMMENT '值',
  `content` TEXT COMMENT '内容',
  `jump_schema` VARCHAR(1024) COMMENT '跳转',
  `friendly_tips` TEXT COMMENT '友好提示',
  `is_continuous_seat` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '连座',
  `count` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '数量',
  `product_id` BIGINT UNSIGNED COMMENT '商品ID',
  `product_name` VARCHAR(512) COMMENT '商品名',
  `type` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '类型',
  `client_type` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '客户端类型',
  `fail_page` VARCHAR(1024) COMMENT '失败跳转',
  `success_page` VARCHAR(1024) COMMENT '成功跳转',
  `description` TEXT COMMENT '描述',
  `detail_info` JSON COMMENT '费用明细',
  `pay_unified_order_info` JSON COMMENT '统一订单信息',
  `tip` JSON COMMENT '提示',
  `pay_return_page` JSON COMMENT '返回页',
  `channels` JSON COMMENT '渠道列表',
  `credit_pay` JSON COMMENT '信用付',
  `huabei_installment` JSON COMMENT '花呗分期',
  `leka_installment` JSON COMMENT '乐卡分期',
  `pay_after_use_info` JSON COMMENT '先用后付',
  `payment_tips` JSON COMMENT '支付提示',
  `payment_types` JSON COMMENT '支付方式',
  `gmt_create` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `gmt_modified` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_payment_pay_id` (`pay_id`),
  KEY `idx_payment_order_id` (`order_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='支付单 / 支付明细（GET v1/unifiedPay/payDetail）（原 payments）' ROW_FORMAT=DYNAMIC;

CREATE TABLE IF NOT EXISTS `payment_channel` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '代理主键',
  `pay_id` BIGINT UNSIGNED COMMENT '支付单ID | 推断',
  `type` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '渠道码',
  `title` VARCHAR(256) COMMENT '标题',
  `value` VARCHAR(256) COMMENT '值',
  `tip` VARCHAR(1024) COMMENT '提示',
  `description` TEXT COMMENT '描述',
  `is_available` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '是否可用 | Flutter PayChannelOption.available',
  `client_type` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '客户端类型 | 推断',
  `installment_num` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '分期数',
  `installment_rate` DECIMAL(10,4) DEFAULT 0.0000 COMMENT '分期费率',
  `is_select` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '是否选中',
  `gmt_create` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `gmt_modified` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  KEY `idx_payment_channel_pay_id` (`pay_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='支付渠道 | PayChannelOption（原 payment_channels）' ROW_FORMAT=DYNAMIC;

CREATE TABLE IF NOT EXISTS `refund` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `refund_id` BIGINT UNSIGNED NOT NULL COMMENT '退款单ID | 推断',
  `order_id` BIGINT UNSIGNED NOT NULL COMMENT '订单ID | PUT v1/order/{orderId}/refund',
  `user_id` BIGINT UNSIGNED COMMENT '用户ID | 推断',
  `refund_amount` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '退款金额 | Flutter RefundState.refundAmount',
  `reason_code` VARCHAR(64) COMMENT '退款原因码（notNeeded / boughtWrong / tooExpensive / foundBetter / other） | Flutter RefundReason',
  `custom_reason` TEXT COMMENT '自定义原因 | Flutter RefundState.customReason',
  `status` VARCHAR(64) NOT NULL COMMENT '退款状态 | Flutter RefundStatus',
  `error_message` VARCHAR(1024) COMMENT '错误信息 | Flutter RefundState.errorMessage',
  `refund_mode` VARCHAR(64) COMMENT '退款手续费模式（见 dict_enum.RefundRuleMode）',
  `gmt_create` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `gmt_modified` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_refund_refund_id` (`refund_id`),
  KEY `idx_refund_order_id` (`order_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='退款申请 | v1/order/{orderId}/refund + Flutter RefundState / RefundReason（原 refunds）' ROW_FORMAT=DYNAMIC;

-- ----------------------------------------------------------------------
--  域 5 · 营销资产（marketing）
-- ----------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS `wallet_account` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `account_id` BIGINT UNSIGNED NOT NULL COMMENT '账户ID | 推断',
  `user_id` BIGINT UNSIGNED NOT NULL COMMENT '用户ID | 推断',
  `balance` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '余额 | v1/moneyaccount / Flutter WalletAccount.balance',
  `status` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '状态 | 推断',
  `gmt_create` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `gmt_modified` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_wallet_account_account_id` (`account_id`),
  UNIQUE KEY `uk_wallet_account_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='钱包账户 | v1/moneyaccount + Flutter WalletAccount（原 wallet_accounts）' ROW_FORMAT=DYNAMIC;

CREATE TABLE IF NOT EXISTS `wallet_sequence` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `sequence_id` BIGINT UNSIGNED NOT NULL COMMENT '流水ID | Flutter WalletTransaction.id',
  `user_id` BIGINT UNSIGNED NOT NULL COMMENT '用户ID | 推断',
  `account_id` BIGINT UNSIGNED NOT NULL COMMENT '账户ID | 推断',
  `title` VARCHAR(512) COMMENT '标题 | Flutter WalletTransaction.title',
  `amount` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '发生额（有符号） | Flutter WalletTransaction.amount',
  `balance_after` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '交易后余额 | Flutter WalletTransaction.balanceAfter',
  `type` VARCHAR(64) COMMENT '收支类型（income / expense） | Flutter WalletTransactionType',
  `status` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '状态 | 推断',
  `gmt_create` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `gmt_modified` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_wallet_sequence_sequence_id` (`sequence_id`),
  KEY `idx_wallet_sequence_user_id_gmt_create` (`user_id`,`gmt_create`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='钱包流水 | v1/moneyaccount/sequences + Flutter WalletTransaction（原 wallet_sequences）' ROW_FORMAT=DYNAMIC;

CREATE TABLE IF NOT EXISTS `gift_card` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `gift_card_id` BIGINT UNSIGNED NOT NULL COMMENT '卡ID',
  `user_id` BIGINT UNSIGNED NOT NULL COMMENT '持有用户',
  `template_id` BIGINT UNSIGNED COMMENT '卡模板ID',
  `title` VARCHAR(512) COMMENT '卡名',
  `description` TEXT COMMENT '说明',
  `denomination` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '面额',
  `balance` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '余额',
  `use_amount` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '本次抵扣额',
  `effective_time_from` DATETIME COMMENT '生效时间',
  `effective_time_to` DATETIME COMMENT '失效时间',
  `status` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '状态 | Flutter GiftCardStatus',
  `type` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '类型',
  `is_revocable` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '可撤销',
  `is_disabled` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '已禁用',
  `serial_number` VARCHAR(128) NOT NULL COMMENT '卡号（Flutter 补充） | Flutter GiftCard.serialNumber',
  `gmt_create` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `gmt_modified` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_gift_card_gift_card_id` (`gift_card_id`),
  UNIQUE KEY `uk_gift_card_serial_number` (`serial_number`),
  KEY `idx_gift_card_user_id_status` (`user_id`,`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='礼品卡 | Flutter GiftCard（原 gift_cards）' ROW_FORMAT=DYNAMIC;

CREATE TABLE IF NOT EXISTS `gift_card_usage` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '代理主键',
  `order_id` BIGINT UNSIGNED NOT NULL COMMENT '订单ID | 推断',
  `gift_card_id` BIGINT UNSIGNED NOT NULL COMMENT '卡ID',
  `use_amount` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '抵扣额',
  `gmt_create` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `gmt_modified` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_gift_card_usage_gift_card_id_order_id` (`gift_card_id`,`order_id`),
  KEY `idx_gift_card_usage_order_id` (`order_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='订单礼品卡抵扣明细（原 gift_card_usages）' ROW_FORMAT=DYNAMIC;
