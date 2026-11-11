-- ========================================================================
--  sql · 04_ticket_inventory.sql（S2 全量规范重构版）
--  规范: 单数表名 / id BIGINT UNSIGNED 自增 / gmt_create,gmt_modified DATETIME
--        is_xxx TINYINT UNSIGNED / 金额 DECIMAL / 无物理外键 / 无冗余索引
--  表: ticket_category, ticket_group, ticket_area, ticket_seat, seat_map_layout, discount_activity
-- ========================================================================

USE `planstar_ticket`;

-- ----------------------------------------------------------------------
--  域 3 · 企划商品（show / goods）
-- ----------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS `ticket_category` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `ticket_category_id` BIGINT UNSIGNED NOT NULL COMMENT '票档ID',
  `activity_event_id` BIGINT UNSIGNED NOT NULL COMMENT '场次ID',
  `activity_id` BIGINT UNSIGNED COMMENT '演出ID | 推断',
  `specification` VARCHAR(256) COMMENT '规格/票档名',
  `description` TEXT COMMENT '描述',
  `origin_price` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '原价',
  `low_price` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '最低价',
  `discount_rate` DECIMAL(10,4) DEFAULT 0.0000 COMMENT '折扣率',
  `is_discount` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '有折扣',
  `is_price_lowest` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '最低价标记',
  `is_has_ticket` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '有票',
  `tickets_num` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '票数',
  `unit_ticket_num` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '单位票数',
  `is_e_ticket` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '电子票',
  `is_need_speed_pack_buy` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '需加速包',
  `is_show_ticket_list` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '显示票列表',
  `icon` VARCHAR(1024) COMMENT '图标',
  `seat_color` VARCHAR(64) COMMENT '座位颜色',
  `status` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '状态',
  `areas` JSON COMMENT '区域',
  `available_numbers` JSON COMMENT '可选数量',
  `proxy_buy_info` JSON COMMENT '代抢信息',
  `gmt_create` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `gmt_modified` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_ticket_category_ticket_category_id` (`ticket_category_id`),
  KEY `idx_ticket_category_activity_event_id` (`activity_event_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='票档（原 ticket_categories）' ROW_FORMAT=DYNAMIC;

CREATE TABLE IF NOT EXISTS `ticket_group` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `ticket_group_id` BIGINT UNSIGNED NOT NULL COMMENT '票组ID（冗余）',
  `ticket_category_id` BIGINT UNSIGNED NOT NULL COMMENT '票档ID',
  `ticket_area_id` BIGINT UNSIGNED COMMENT '区域ID',
  `ticket_row_id` BIGINT UNSIGNED COMMENT '排ID',
  `activity_event_id` BIGINT UNSIGNED NOT NULL COMMENT '场次ID | 推断',
  `seller_user_id` BIGINT UNSIGNED COMMENT '卖家ID',
  `saler_name` VARCHAR(128) COMMENT '卖家名',
  `sale_price` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '售价',
  `origin_price` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '原价',
  `specification` VARCHAR(256) COMMENT '规格',
  `ticket_category_specification` VARCHAR(256) COMMENT '票档规格',
  `area_name` VARCHAR(256) COMMENT '区域名',
  `description` TEXT COMMENT '描述',
  `ticket_category_desc` TEXT COMMENT '票档描述',
  `count` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '数量',
  `sold` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '已售',
  `selling_amount` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '在售数量',
  `is_saling` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '在售标记',
  `is_continuous_seat` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '连座',
  `is_e_ticket` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '电子票',
  `is_recommended` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '推荐',
  `promise_deliver_day` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '承诺发货天数',
  `promise_deliver_notice` TEXT COMMENT '发货承诺',
  `row_index` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '排索引',
  `seat_color` VARCHAR(64) COMMENT '座位颜色',
  `split_order_fee` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '拆单费',
  `succeed_num` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '已成团数',
  `pin_tuan_campaign_item_id` BIGINT UNSIGNED COMMENT '拼团项ID',
  `unit_ticket_num` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '单位票数',
  `full_view_image` VARCHAR(1024) COMMENT '全场图',
  `sale_tag` VARCHAR(256) COMMENT '销售标签',
  `num_max` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '最大购买数',
  `num_min` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '最小购买数',
  `sell_limit` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '限售',
  `is_speed_express` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '极速配送',
  `is_divide` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '是否可拆',
  `addition` JSON COMMENT '附加项',
  `available_numbers` JSON COMMENT '可选数量',
  `gmt_create` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `gmt_modified` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_ticket_group_ticket_group_id` (`ticket_group_id`),
  KEY `idx_ticket_group_ticket_category_id` (`ticket_category_id`),
  KEY `idx_ticket_group_seller_user_id` (`seller_user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='票组 / 挂单（原 ticket_groups）' ROW_FORMAT=DYNAMIC;

CREATE TABLE IF NOT EXISTS `ticket_area` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `ticket_area_id` BIGINT UNSIGNED NOT NULL COMMENT '区域ID',
  `area_name` VARCHAR(256) COMMENT '区域名',
  `height` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '高',
  `width` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '宽',
  `is_select` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '可选',
  `is_sold_out` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '售罄',
  `color` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '颜色（Flutter SeatArea.color） | Flutter',
  `label` VARCHAR(128) COMMENT '标签（Flutter） | Flutter',
  `stage_label` VARCHAR(128) COMMENT '舞台标签（Flutter SeatMapData） | Flutter',
  `rows` JSON COMMENT '排列表（Flutter List<SeatRow>）',
  `gmt_create` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `gmt_modified` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_ticket_area_ticket_area_id` (`ticket_area_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='选座区域 | Flutter SeatArea（原 ticket_areas）' ROW_FORMAT=DYNAMIC;

CREATE TABLE IF NOT EXISTS `ticket_seat` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `ticket_seat_id` BIGINT UNSIGNED NOT NULL COMMENT '座位ID',
  `area_id` BIGINT UNSIGNED COMMENT '区域ID | Flutter Seat.areaName 关联',
  `area_name` VARCHAR(256) COMMENT '区域名 | Flutter Seat.areaName',
  `seat_row` VARCHAR(64) COMMENT '排',
  `seat_col` VARCHAR(64) COMMENT '列',
  `x` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT 'X 坐标',
  `y` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT 'Y 坐标',
  `z` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT 'Z 坐标',
  `status` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '状态（0 可选 / 1 已售 / 2 锁定 / 3 通道 / 4 不可选，见 SeatStatus）',
  `description` VARCHAR(512) COMMENT '描述',
  `price` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '价格（Flutter） | Flutter Seat.price',
  `category_id` BIGINT UNSIGNED COMMENT '票档ID（Flutter） | Flutter Seat.categoryId',
  `ticket_group_id` BIGINT UNSIGNED COMMENT '票组ID | 推断',
  `ticket` JSON COMMENT '票快照',
  `gmt_create` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `gmt_modified` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_ticket_seat_ticket_seat_id` (`ticket_seat_id`),
  KEY `idx_ticket_seat_area_id` (`area_id`),
  KEY `idx_ticket_seat_ticket_group_id` (`ticket_group_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='座位 | Flutter Seat（原 ticket_seats）' ROW_FORMAT=DYNAMIC;

CREATE TABLE IF NOT EXISTS `seat_map_layout` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `layout_id` BIGINT UNSIGNED NOT NULL COMMENT '布局ID',
  `activity_event_id` BIGINT UNSIGNED COMMENT '场次ID | 推断',
  `venue_name` VARCHAR(256) COMMENT '场馆名（Flutter） | Flutter SeatMapData.venueName',
  `stage_label` VARCHAR(128) COMMENT '舞台标签（Flutter） | Flutter',
  `type` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '类型（见 dict_enum.TicketType）',
  `status` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '状态',
  `is_has_full_view_image` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '是否有全场图',
  `icon_checked` VARCHAR(1024) COMMENT '选中图标',
  `icon_unable` VARCHAR(1024) COMMENT '不可选图标',
  `icon_un_checked` VARCHAR(1024) COMMENT '未选图标',
  `icon_ticket_category` VARCHAR(1024) COMMENT '票档图标',
  `icon_unable_template` JSON COMMENT '模板',
  `package_discount` JSON COMMENT '套餐折扣',
  `ticket_categories_icon` JSON COMMENT '票档图标映射',
  `area_tickets` JSON COMMENT '区域票',
  `ticket_categories` JSON COMMENT '票档',
  `areas` JSON COMMENT '区域（Flutter List<SeatArea>） | Flutter',
  `gmt_create` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `gmt_modified` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_seat_map_layout_layout_id` (`layout_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='选座图布局 | Flutter SeatMapData（原 seat_map_layouts）' ROW_FORMAT=DYNAMIC;

CREATE TABLE IF NOT EXISTS `discount_activity` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '代理主键',
  `activity_event_id` BIGINT UNSIGNED COMMENT '场次ID',
  `location_type` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '位置类型',
  `origin_price` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '原价',
  `sale_price` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '售价',
  `ticket_category_id` BIGINT UNSIGNED COMMENT '票档ID',
  `ticket_group_id` BIGINT UNSIGNED COMMENT '票组ID',
  `gmt_create` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `gmt_modified` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='折扣活动（原 discount_activities）' ROW_FORMAT=DYNAMIC;
