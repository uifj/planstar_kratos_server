-- ========================================================================
--  sql · 07_marketing.sql（S2 全量规范重构版）
--  规范: 单数表名 / id BIGINT UNSIGNED 自增 / gmt_create,gmt_modified DATETIME
--        is_xxx TINYINT UNSIGNED / 金额 DECIMAL / 无物理外键 / 无冗余索引
--  表: coupon, campaign, coupon_package, pin_tuan_campaign, pin_tuan_campaign_item, pin_tuan_order, group_buy_member, proxy_buy_config, proxy_buy_order, transfer, to_watch, favorite
-- ========================================================================

USE `planstar_marketing`;

-- ----------------------------------------------------------------------
--  域 5 · 营销资产（marketing）
-- ----------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS `coupon` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `coupon_id` BIGINT UNSIGNED NOT NULL COMMENT '券ID',
  `user_id` BIGINT UNSIGNED NOT NULL COMMENT '持有用户',
  `order_id` BIGINT UNSIGNED COMMENT '使用订单',
  `activity_id` BIGINT UNSIGNED COMMENT '适用演出',
  `title` VARCHAR(512) COMMENT '券名',
  `description` TEXT COMMENT '说明',
  `discount_type` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '抵扣类型（1 固定金额 / 2 折扣率） | Flutter CouponDiscountType',
  `discount_amount` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '抵扣金额',
  `discount_rate` DECIMAL(10,4) DEFAULT 0.0000 COMMENT '折扣率（万分比口径）',
  `require_amount` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '使用门槛',
  `effective_time_from` DATETIME COMMENT '生效时间',
  `effective_time_to` DATETIME COMMENT '失效时间',
  `cost_bear_type` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '成本承担方',
  `is_revocable` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '可撤销',
  `is_disabled` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '已禁用',
  `status` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '状态（可用/已用/过期） | Flutter CouponStatus',
  `is_success` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '领取结果',
  `source_id` BIGINT UNSIGNED COMMENT '来源ID',
  `source_type` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '来源类型',
  `type` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '券类型',
  `activity_event_id` BIGINT UNSIGNED COMMENT '适用场次',
  `extra_condition` JSON COMMENT '附加条件',
  `data` JSON COMMENT '自嵌套券',
  `activity_category_ids` JSON COMMENT '适用分类',
  `activity_ids` JSON COMMENT '适用演出',
  `client_types` JSON COMMENT '适用客户端',
  `shop_ids` JSON COMMENT '适用店铺',
  `gmt_create` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `gmt_modified` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_coupon_coupon_id` (`coupon_id`),
  KEY `idx_coupon_user_id_status` (`user_id`,`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='优惠券 | Flutter Coupon（原 coupons）' ROW_FORMAT=DYNAMIC;

CREATE TABLE IF NOT EXISTS `campaign` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `campaign_id` BIGINT UNSIGNED NOT NULL COMMENT '活动ID',
  `title` VARCHAR(512) COMMENT '标题',
  `description` TEXT COMMENT '描述',
  `discount_amount` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '优惠金额',
  `background_img` VARCHAR(1024) COMMENT '背景图',
  `content` TEXT COMMENT '内容',
  `url` VARCHAR(1024) COMMENT '链接',
  `is_can_buy` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '可购买',
  `is_can_use_coupon` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '可叠加券',
  `error_hint` VARCHAR(1024) COMMENT '错误提示',
  `coupons` JSON COMMENT '券列表',
  `disable_coupons` JSON COMMENT '不可用券',
  `gmt_create` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `gmt_modified` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_campaign_campaign_id` (`campaign_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='营销活动 / 折扣（原 campaigns）' ROW_FORMAT=DYNAMIC;

CREATE TABLE IF NOT EXISTS `coupon_package` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '代理主键',
  `is_has_coupon` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '是否有券',
  `message` VARCHAR(1024) COMMENT '文案',
  `title` VARCHAR(512) COMMENT '标题',
  `total` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '总数',
  `url` VARCHAR(1024) COMMENT '链接',
  `gmt_create` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `gmt_modified` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='券包（原 coupon_packages）' ROW_FORMAT=DYNAMIC;

CREATE TABLE IF NOT EXISTS `pin_tuan_campaign` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `pin_tuan_campaign_id` BIGINT UNSIGNED NOT NULL COMMENT '拼团活动ID',
  `activity_id` BIGINT UNSIGNED NOT NULL COMMENT '演出ID',
  `shop_id` BIGINT UNSIGNED NOT NULL COMMENT '店铺ID',
  `low_price` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '拼团低价',
  `original_price` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '原价（Flutter） | Flutter GroupBuyInfo.originalPrice',
  `pin_tuan_tag` VARCHAR(256) COMMENT '拼团标签',
  `slogan` VARCHAR(1024) COMMENT '标语',
  `stock` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '库存',
  `succeed_num` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '成团人数',
  `current_members` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '当前人数（Flutter） | Flutter GroupBuyInfo.currentMembers',
  `is_open` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '是否开启',
  `status` VARCHAR(64) NOT NULL COMMENT '状态（Flutter GroupBuyStatus） | Flutter',
  `show_title` VARCHAR(512) COMMENT '演出名（Flutter） | Flutter GroupBuyInfo.showTitle',
  `venue` VARCHAR(256) COMMENT '场馆（Flutter） | Flutter',
  `event_time` DATETIME COMMENT '场次时间（Flutter） | Flutter',
  `effective_from` DATETIME COMMENT '生效时间',
  `effective_to` DATETIME COMMENT '失效时间',
  `gmt_create` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `gmt_modified` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_pin_tuan_campaign_pin_tuan_campaign_id` (`pin_tuan_campaign_id`),
  UNIQUE KEY `uk_pin_tuan_campaign_activity_id_shop_id` (`activity_id`,`shop_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='拼团活动（GET v1/pinTuanCampaign/{id}）+ Flutter GroupBuyInfo（原 pin_tuan_campaigns）' ROW_FORMAT=DYNAMIC;

CREATE TABLE IF NOT EXISTS `pin_tuan_campaign_item` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '代理主键',
  `pin_tuan_id` BIGINT UNSIGNED COMMENT '拼团活动ID | 推断',
  `name` VARCHAR(256) COMMENT '名称',
  `ticket_group` JSON COMMENT '票组',
  `ticket_category_id` BIGINT UNSIGNED COMMENT '票档ID | 推断',
  `gmt_create` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `gmt_modified` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  KEY `idx_pin_tuan_campaign_item_pin_tuan_id` (`pin_tuan_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='拼团票档项（原 pin_tuan_campaign_items）' ROW_FORMAT=DYNAMIC;

CREATE TABLE IF NOT EXISTS `pin_tuan_order` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `pin_tuan_order_id` BIGINT UNSIGNED NOT NULL COMMENT '拼团订单ID | 推断',
  `user_id` BIGINT UNSIGNED NOT NULL COMMENT '用户ID | 推断',
  `order_id` BIGINT UNSIGNED NOT NULL COMMENT '统一订单ID | 推断',
  `pin_tuan_id` BIGINT UNSIGNED COMMENT '拼团活动ID',
  `pin_tuan_campaign_item_id` BIGINT UNSIGNED COMMENT '拼团项ID',
  `count` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '数量',
  `succeed_num` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '已成团人数 | 推断',
  `status` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '状态 | 推断',
  `gmt_create` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `gmt_modified` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_pin_tuan_order_pin_tuan_order_id` (`pin_tuan_order_id`),
  KEY `idx_pin_tuan_order_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='拼团订单 | v1/pinTuanOrder/make（原 pin_tuan_orders）' ROW_FORMAT=DYNAMIC;

CREATE TABLE IF NOT EXISTS `group_buy_member` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '代理主键',
  `group_id` BIGINT UNSIGNED COMMENT '拼团ID | 推断',
  `user_id` BIGINT UNSIGNED COMMENT '用户ID | Flutter GroupBuyMember.userId',
  `nickname` VARCHAR(128) COMMENT '昵称 | Flutter',
  `avatar_url` VARCHAR(1024) COMMENT '头像 | Flutter',
  `is_leader` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '是否团长 | Flutter',
  `gmt_create` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `gmt_modified` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  KEY `idx_group_buy_member_group_id` (`group_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='拼团成员 | Flutter GroupBuyMember（原 group_buy_members）' ROW_FORMAT=DYNAMIC;

CREATE TABLE IF NOT EXISTS `proxy_buy_config` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '代理主键',
  `activity_id` BIGINT UNSIGNED COMMENT '演出ID | 推断',
  `ticket_category_id` BIGINT UNSIGNED COMMENT '票档ID | 推断',
  `proxy_buy_fee` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '代抢费',
  `proxy_buy_rule_desc` TEXT COMMENT '代抢规则',
  `is_compensation_open` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '无票赔付开关',
  `end_value` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '截止时间档位值',
  `end_value_desc` VARCHAR(256) COMMENT '截止时间文案',
  `time_str` VARCHAR(128) COMMENT '轮次时间',
  `turn_title` VARCHAR(256) COMMENT '轮次标题',
  `proxy_buy_time_info` JSON COMMENT '时间档位列表',
  `proxy_buy_turn_info` JSON COMMENT '轮次',
  `gmt_create` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `gmt_modified` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  KEY `idx_proxy_buy_config_activity_id` (`activity_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='代抢配置（原 proxy_buy_configs）' ROW_FORMAT=DYNAMIC;

CREATE TABLE IF NOT EXISTS `proxy_buy_order` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `proxy_buy_order_id` BIGINT UNSIGNED NOT NULL COMMENT '代抢单ID | Flutter ProxyBuyOrder.id',
  `order_number` VARCHAR(64) NOT NULL COMMENT '订单号 | Flutter ProxyBuySubmitState.orderNumber',
  `user_id` BIGINT UNSIGNED COMMENT '用户ID | 推断',
  `show_title` VARCHAR(512) COMMENT '演出名 | Flutter',
  `show_poster` VARCHAR(1024) COMMENT '海报 | Flutter',
  `venue` VARCHAR(256) COMMENT '场馆 | Flutter',
  `event_time` DATETIME COMMENT '场次时间 | Flutter',
  `turn` JSON COMMENT '轮次 | Flutter',
  `deadline` JSON COMMENT '截止档 | Flutter',
  `ticket_price` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '票款 | Flutter',
  `quantity` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '数量 | Flutter',
  `proxy_buy_fee` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '代抢费 | Flutter',
  `total_amount` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '总额（= 票款×数量 + 代抢费） | Flutter ProxyBuyHelper.calculateTotalPrice',
  `status` VARCHAR(64) NOT NULL COMMENT '状态（pending / inProgress / success / failed / cancelled） | Flutter ProxyBuyStatus',
  `is_compensation_enabled` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '是否赔付 | Flutter',
  `gmt_create` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `gmt_modified` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_proxy_buy_order_proxy_buy_order_id` (`proxy_buy_order_id`),
  KEY `idx_proxy_buy_order_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='代抢订单 | Flutter ProxyBuyOrder / ProxyBuySubmitState（原 proxy_buy_orders）' ROW_FORMAT=DYNAMIC;

CREATE TABLE IF NOT EXISTS `transfer` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `transfer_id` BIGINT UNSIGNED NOT NULL COMMENT '转票ID',
  `user_id` BIGINT UNSIGNED NOT NULL COMMENT '发布用户',
  `activity_id` BIGINT UNSIGNED NOT NULL COMMENT '演出ID',
  `activity_event_id` BIGINT UNSIGNED COMMENT '场次ID',
  `ticket_category_id` BIGINT UNSIGNED COMMENT '票档ID',
  `activity_name` VARCHAR(512) COMMENT '演出名',
  `address` VARCHAR(1024) COMMENT '交易地点',
  `category_desc` VARCHAR(256) COMMENT '票档描述',
  `event_specification` VARCHAR(256) COMMENT '场次规格',
  `count` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '数量',
  `price` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '价格',
  `content` TEXT COMMENT '说明',
  `is_check` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '是否已核验',
  `poster` VARCHAR(1024) COMMENT '海报',
  `is_liked` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '已赞',
  `likes` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '赞数',
  `reply_count` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '回复数',
  `status` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '状态（见 dict_enum.TransferStatus）',
  `venue_city_id` BIGINT UNSIGNED COMMENT '场馆城市',
  `activity` JSON COMMENT '演出快照',
  `activity_event` JSON COMMENT '场次快照',
  `ticket_category` JSON COMMENT '票档快照',
  `user` JSON COMMENT '用户快照',
  `gmt_create` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `gmt_modified` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_transfer_transfer_id` (`transfer_id`),
  KEY `idx_transfer_activity_id` (`activity_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='转票 / 求票（原 transfers）' ROW_FORMAT=DYNAMIC;

CREATE TABLE IF NOT EXISTS `to_watch` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '代理主键',
  `user_id` BIGINT UNSIGNED NOT NULL COMMENT '用户ID | 推断',
  `activity_id` BIGINT UNSIGNED COMMENT '演出ID',
  `activity_event_id` BIGINT UNSIGNED NOT NULL COMMENT '场次ID',
  `activity_name` VARCHAR(512) COMMENT '演出名',
  `short_name` VARCHAR(256) COMMENT '简称',
  `activity_event_date` DATE COMMENT '场次日期',
  `order_id` BIGINT UNSIGNED COMMENT '订单ID',
  `unified_order_id` BIGINT UNSIGNED COMMENT '统一订单ID',
  `venue` JSON COMMENT '场馆快照',
  `gmt_create` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `gmt_modified` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_to_watch_user_id_activity_event_id` (`user_id`,`activity_event_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='想看 / 观演日历' ROW_FORMAT=DYNAMIC;

CREATE TABLE IF NOT EXISTS `favorite` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `favorite_id` BIGINT UNSIGNED NOT NULL COMMENT '收藏ID',
  `user_id` BIGINT UNSIGNED NOT NULL COMMENT '用户ID',
  `activity_id` BIGINT UNSIGNED NOT NULL COMMENT '演出ID',
  `type` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '类型',
  `is_liked` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '已赞',
  `likes` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '赞数',
  `reply_count` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '回复数',
  `shares` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '分享数',
  `notify_status` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '开售提醒状态',
  `status` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '状态',
  `activity` JSON COMMENT '演出快照',
  `user` JSON COMMENT '用户快照',
  `gmt_create` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `gmt_modified` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_favorite_favorite_id` (`favorite_id`),
  UNIQUE KEY `uk_favorite_user_id_activity_id` (`user_id`,`activity_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='收藏 / 想看动态（原 favorites）' ROW_FORMAT=DYNAMIC;
