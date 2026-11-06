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
  `order_id` BIGINT UNSIGNED NOT NULL COMMENT '订单ID | PayInfo.orderId',
  `left_time` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '剩余支付秒数（倒计时基线） | PayInfo.leftTime',
  `pay_amount` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '应付金额 | PayInfo.payAmount',
  `balance` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '可用余额 | PayInfo.balance',
  `title` VARCHAR(512) COMMENT '标题 | PayInfo.title',
  `value` VARCHAR(512) COMMENT '值 | PayInfo.value',
  `content` TEXT COMMENT '内容 | PayInfo.content',
  `jump_schema` VARCHAR(1024) COMMENT '跳转 | PayInfo.schema',
  `friendly_tips` TEXT COMMENT '友好提示 | PayInfo.friendlyTips',
  `is_continuous_seat` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '连座 | PayInfo.continuousSeat',
  `count` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '数量 | PayInfo.count',
  `product_id` BIGINT UNSIGNED COMMENT '商品ID | PayInfo.productId',
  `product_name` VARCHAR(512) COMMENT '商品名 | PayInfo.productName',
  `type` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '类型 | PayInfo.type',
  `client_type` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '客户端类型 | PayInfo.clientType',
  `fail_page` VARCHAR(1024) COMMENT '失败跳转 | PayInfo.failPage',
  `success_page` VARCHAR(1024) COMMENT '成功跳转 | PayInfo.successPage',
  `description` TEXT COMMENT '描述 | PayInfo.desc',
  `detail_info` JSON COMMENT '费用明细（原 List<DetailInfo>） | PayInfo.detailInfo',
  `pay_unified_order_info` JSON COMMENT '统一订单信息 | PayInfo.payUnifiedOrderInfo',
  `tip` JSON COMMENT '提示（原 Tip） | PayInfo.tip',
  `pay_return_page` JSON COMMENT '返回页（原 PayReturnPage） | PayInfo.payReturnPage',
  `channels` JSON COMMENT '渠道列表（原 List<Channel>） | PayInfo.channels',
  `credit_pay` JSON COMMENT '信用付 | PayInfo.creditPay',
  `huabei_installment` JSON COMMENT '花呗分期（原 List<CommonInstallment>） | PayInfo.huabeiInstallment',
  `leka_installment` JSON COMMENT '乐卡分期 | PayInfo.lekaInstallment',
  `pay_after_use_info` JSON COMMENT '先用后付 | PayInfo.payAfterUseInfo',
  `payment_tips` JSON COMMENT '支付提示（原 Map<String,String>） | OrderPaymentInfo.paymentTips',
  `payment_types` JSON COMMENT '支付方式（原 List<Integer>） | OrderPaymentInfo.paymentTypes',
  `gmt_create` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `gmt_modified` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_payment_pay_id` (`pay_id`),
  KEY `idx_payment_order_id` (`order_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='支付单 / 支付明细 | PayInfo（GET v1/unifiedPay/payDetail）· OrderPaymentInfo（原 payments）' ROW_FORMAT=DYNAMIC;

CREATE TABLE IF NOT EXISTS `payment_channel` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '代理主键',
  `pay_id` BIGINT UNSIGNED COMMENT '支付单ID | 推断',
  `type` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '渠道码 | Channel.type',
  `title` VARCHAR(256) COMMENT '标题 | Channel.title',
  `value` VARCHAR(256) COMMENT '值 | Channel.value',
  `tip` VARCHAR(1024) COMMENT '提示 | Channel.tip',
  `description` TEXT COMMENT '描述 | Channel.desc',
  `is_available` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '是否可用 | Flutter PayChannelOption.available',
  `client_type` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '客户端类型 | 推断',
  `installment_num` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '分期数 | CommonInstallment.installmentNum',
  `installment_rate` DECIMAL(10,4) DEFAULT 0.0000 COMMENT '分期费率 | CommonInstallment.installmentRate',
  `is_select` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '是否选中 | CommonInstallment.isSelect',
  `gmt_create` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `gmt_modified` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  KEY `idx_payment_channel_pay_id` (`pay_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='支付渠道 | Channel / PayChannelOption / CommonInstallment（原 payment_channels）' ROW_FORMAT=DYNAMIC;

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
  `refund_mode` VARCHAR(64) COMMENT '退款手续费模式（见 dict_enum.RefundRuleMode） | ActivityRefundRuleDetailBean.refundMode',
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
  `gift_card_id` BIGINT UNSIGNED NOT NULL COMMENT '卡ID | GiftCard.id',
  `user_id` BIGINT UNSIGNED NOT NULL COMMENT '持有用户 | GiftCard.userId',
  `template_id` BIGINT UNSIGNED COMMENT '卡模板ID | GiftCard.templateId',
  `title` VARCHAR(512) COMMENT '卡名 | GiftCard.title',
  `description` TEXT COMMENT '说明 | GiftCard.desc',
  `denomination` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '面额 | GiftCard.denomination',
  `balance` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '余额 | GiftCard.balance',
  `use_amount` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '本次抵扣额 | GiftCard.useAmount',
  `effective_time_from` DATETIME COMMENT '生效时间 | GiftCard.effectiveTimeFrom',
  `effective_time_to` DATETIME COMMENT '失效时间 | GiftCard.effectiveTimeTo',
  `status` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '状态 | GiftCard.status / Flutter GiftCardStatus',
  `type` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '类型 | GiftCard.type',
  `is_revocable` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '可撤销 | GiftCard.isCanRevoke',
  `is_disabled` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '已禁用 | GiftCard.isDisable',
  `serial_number` VARCHAR(128) NOT NULL COMMENT '卡号（Flutter 补充） | Flutter GiftCard.serialNumber',
  `gmt_create` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `gmt_modified` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_gift_card_gift_card_id` (`gift_card_id`),
  UNIQUE KEY `uk_gift_card_serial_number` (`serial_number`),
  KEY `idx_gift_card_user_id_status` (`user_id`,`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='礼品卡 | GiftCard（v1/giftCards · v2/giftCards · v3/order/allGiftCard）+ Flutter GiftCard（原 gift_cards）' ROW_FORMAT=DYNAMIC;

CREATE TABLE IF NOT EXISTS `gift_card_usage` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '代理主键',
  `order_id` BIGINT UNSIGNED NOT NULL COMMENT '订单ID | 推断',
  `gift_card_id` BIGINT UNSIGNED NOT NULL COMMENT '卡ID | GiftCard.id',
  `use_amount` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '抵扣额 | GiftCard.useAmount',
  `gmt_create` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `gmt_modified` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_gift_card_usage_gift_card_id_order_id` (`gift_card_id`,`order_id`),
  KEY `idx_gift_card_usage_order_id` (`order_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='订单礼品卡抵扣明细 | MakeOrderRequestBody.giftCards（原 gift_card_usages）' ROW_FORMAT=DYNAMIC;
