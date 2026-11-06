-- ========================================================================
--  sql · 04_ticket_inventory.sql（S2 全量规范重构版）
--  规范: 单数表名 / id BIGINT UNSIGNED 自增 / gmt_create,gmt_modified DATETIME
--        is_xxx TINYINT UNSIGNED / 金额 DECIMAL / 无物理外键 / 无冗余索引
--  表: ticket_category, ticket_group, ticket_area, ticket_seat, seat_map_layout, discount_activity
-- ========================================================================

USE `planstar_ticket`;

-- ----------------------------------------------------------------------
--  域 3 · 演出商品（show / goods）
-- ----------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS `ticket_category` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `ticket_category_id` BIGINT UNSIGNED NOT NULL COMMENT '票档ID | TicketCategory.id',
  `activity_event_id` BIGINT UNSIGNED NOT NULL COMMENT '场次ID | TicketCategory.activityEventId',
  `activity_id` BIGINT UNSIGNED COMMENT '演出ID | 推断',
  `specification` VARCHAR(256) COMMENT '规格/票档名 | TicketCategory.specification',
  `description` TEXT COMMENT '描述 | TicketCategory.desc',
  `origin_price` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '原价 | TicketCategory.originPrice',
  `low_price` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '最低价 | TicketCategory.lowPrice',
  `discount_rate` DECIMAL(10,4) DEFAULT 0.0000 COMMENT '折扣率 | TicketCategory.discountRate',
  `is_discount` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '有折扣 | TicketCategory.discountFlag',
  `is_price_lowest` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '最低价标记 | TicketCategory.priceLowest',
  `is_has_ticket` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '有票 | TicketCategory.hasTicket',
  `tickets_num` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '票数 | TicketCategory.ticketsNum',
  `unit_ticket_num` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '单位票数 | TicketCategory.unitTicketNum',
  `is_e_ticket` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '电子票 | TicketCategory.isETicket',
  `is_need_speed_pack_buy` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '需加速包 | TicketCategory.needSpeedPackBuy',
  `is_show_ticket_list` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '显示票列表 | TicketCategory.showTicketList',
  `icon` VARCHAR(1024) COMMENT '图标 | TicketCategory.icon',
  `seat_color` VARCHAR(64) COMMENT '座位颜色 | TicketCategory.seatColor',
  `status` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '状态 | TicketCategory.status',
  `areas` JSON COMMENT '区域（原 List<TicketAreaBean>） | TicketCategory.areas',
  `available_numbers` JSON COMMENT '可选数量（原 List<Integer>） | TicketCategory.availableNumbers',
  `proxy_buy_info` JSON COMMENT '代抢信息（原 ProxyBuyInfo） | TicketCategory.proxyBuyInfo',
  `gmt_create` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `gmt_modified` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_ticket_category_ticket_category_id` (`ticket_category_id`),
  KEY `idx_ticket_category_activity_event_id` (`activity_event_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='票档 | TicketCategory（v4/ticketCategories · v5/ticketCategories）（原 ticket_categories）' ROW_FORMAT=DYNAMIC;

CREATE TABLE IF NOT EXISTS `ticket_group` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `ticket_group_id` BIGINT UNSIGNED NOT NULL COMMENT '票组ID（冗余） | TicketGroup.ticketGroupId',
  `ticket_category_id` BIGINT UNSIGNED NOT NULL COMMENT '票档ID | TicketGroup.ticketCategoryId',
  `ticket_area_id` BIGINT UNSIGNED COMMENT '区域ID | TicketGroup.ticketAreaId',
  `ticket_row_id` BIGINT UNSIGNED COMMENT '排ID | TicketGroup.ticketRowId',
  `activity_event_id` BIGINT UNSIGNED NOT NULL COMMENT '场次ID | 推断',
  `seller_user_id` BIGINT UNSIGNED COMMENT '卖家ID | TicketGroup.saler',
  `saler_name` VARCHAR(128) COMMENT '卖家名 | TicketGroup.salerName',
  `sale_price` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '售价 | TicketGroup.salePrice',
  `origin_price` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '原价 | TicketGroup.originPrice',
  `specification` VARCHAR(256) COMMENT '规格 | TicketGroup.specification',
  `ticket_category_specification` VARCHAR(256) COMMENT '票档规格 | TicketGroup.ticketCategorySpecification',
  `area_name` VARCHAR(256) COMMENT '区域名 | TicketGroup.areaName',
  `description` TEXT COMMENT '描述 | TicketGroup.desc',
  `ticket_category_desc` TEXT COMMENT '票档描述 | TicketGroup.ticketCategoryDesc',
  `count` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '数量 | TicketGroup.count',
  `sold` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '已售 | TicketGroup.sold',
  `selling_amount` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '在售数量 | TicketGroup.sellingAmount',
  `is_saling` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '在售标记 | TicketGroup.saling',
  `is_continuous_seat` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '连座 | TicketGroup.isContinuousSeat',
  `is_e_ticket` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '电子票 | TicketGroup.isETicket',
  `is_recommended` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '推荐 | TicketGroup.isRecommended',
  `promise_deliver_day` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '承诺发货天数 | TicketGroup.promiseDeliverDay',
  `promise_deliver_notice` TEXT COMMENT '发货承诺 | TicketGroup.promiseDeliverNotice',
  `row_index` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '排索引 | TicketGroup.rowIndex',
  `seat_color` VARCHAR(64) COMMENT '座位颜色 | TicketGroup.seatColor',
  `split_order_fee` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '拆单费 | TicketGroup.splitOrderFee',
  `succeed_num` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '已成团数 | TicketGroup.succeedNum',
  `pin_tuan_campaign_item_id` BIGINT UNSIGNED COMMENT '拼团项ID | TicketGroup.pinTuanCampaignItemId',
  `unit_ticket_num` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '单位票数 | TicketGroup.unitTicketNum',
  `full_view_image` VARCHAR(1024) COMMENT '全场图 | TicketGroup.fullViewImage',
  `sale_tag` VARCHAR(256) COMMENT '销售标签 | TicketGroup.saleTag',
  `num_max` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '最大购买数 | TicketGroup.numMax',
  `num_min` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '最小购买数 | TicketGroup.numMin',
  `sell_limit` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '限售 | TicketGroup.sellLimit',
  `is_speed_express` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '极速配送 | TicketGroup.speedExpress',
  `is_divide` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '是否可拆 | TicketGroup.isIsDivide',
  `addition` JSON COMMENT '附加项（原 AdditionBean） | TicketGroup.addition',
  `available_numbers` JSON COMMENT '可选数量（原 List<Integer>） | TicketGroup.availableNumbers',
  `gmt_create` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `gmt_modified` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_ticket_group_ticket_group_id` (`ticket_group_id`),
  KEY `idx_ticket_group_ticket_category_id` (`ticket_category_id`),
  KEY `idx_ticket_group_seller_user_id` (`seller_user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='票组 / 挂单 | TicketGroup（v1/ticketGroup · v4/seats）（原 ticket_groups）' ROW_FORMAT=DYNAMIC;

CREATE TABLE IF NOT EXISTS `ticket_area` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `ticket_area_id` BIGINT UNSIGNED NOT NULL COMMENT '区域ID | TicketAreaBean.id',
  `area_name` VARCHAR(256) COMMENT '区域名 | TicketAreaBean.areaName',
  `height` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '高 | TicketAreaBean.height',
  `width` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '宽 | TicketAreaBean.width',
  `is_select` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '可选 | TicketAreaBean.isSelect',
  `is_sold_out` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '售罄 | TicketAreaBean.soldOut',
  `color` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '颜色（Flutter SeatArea.color） | Flutter',
  `label` VARCHAR(128) COMMENT '标签（Flutter） | Flutter',
  `stage_label` VARCHAR(128) COMMENT '舞台标签（Flutter SeatMapData） | Flutter',
  `rows` JSON COMMENT '排列表（原 List<TicketSeatBean> / Flutter List<SeatRow>） | TicketAreaBean.ticketSeats',
  `gmt_create` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `gmt_modified` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_ticket_area_ticket_area_id` (`ticket_area_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='选座区域 | TicketAreaBean + Flutter SeatArea（原 ticket_areas）' ROW_FORMAT=DYNAMIC;

CREATE TABLE IF NOT EXISTS `ticket_seat` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `ticket_seat_id` BIGINT UNSIGNED NOT NULL COMMENT '座位ID | TicketSeatBean.id',
  `area_id` BIGINT UNSIGNED COMMENT '区域ID | Flutter Seat.areaName 关联',
  `area_name` VARCHAR(256) COMMENT '区域名 | Flutter Seat.areaName',
  `seat_row` VARCHAR(64) COMMENT '排 | TicketSeatBean.row',
  `seat_col` VARCHAR(64) COMMENT '列 | TicketSeatBean.col',
  `x` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT 'X 坐标 | TicketSeatBean.x',
  `y` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT 'Y 坐标 | TicketSeatBean.y',
  `z` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT 'Z 坐标 | TicketSeatBean.z',
  `status` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '状态（0 可选 / 1 已售 / 2 锁定 / 3 通道 / 4 不可选，见 SeatStatus） | TicketSeatBean.status',
  `description` VARCHAR(512) COMMENT '描述 | TicketSeatBean.desc',
  `price` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '价格（Flutter） | Flutter Seat.price',
  `category_id` BIGINT UNSIGNED COMMENT '票档ID（Flutter） | Flutter Seat.categoryId',
  `ticket_group_id` BIGINT UNSIGNED COMMENT '票组ID | 推断',
  `ticket` JSON COMMENT '票快照（原 TicketGroup） | TicketSeatBean.ticket',
  `gmt_create` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `gmt_modified` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_ticket_seat_ticket_seat_id` (`ticket_seat_id`),
  KEY `idx_ticket_seat_area_id` (`area_id`),
  KEY `idx_ticket_seat_ticket_group_id` (`ticket_group_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='座位 | TicketSeatBean + Flutter Seat（原 ticket_seats）' ROW_FORMAT=DYNAMIC;

CREATE TABLE IF NOT EXISTS `seat_map_layout` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `layout_id` BIGINT UNSIGNED NOT NULL COMMENT '布局ID | TicketTableBean.id',
  `activity_event_id` BIGINT UNSIGNED COMMENT '场次ID | 推断',
  `venue_name` VARCHAR(256) COMMENT '场馆名（Flutter） | Flutter SeatMapData.venueName',
  `stage_label` VARCHAR(128) COMMENT '舞台标签（Flutter） | Flutter',
  `type` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '类型（见 dict_enum.TicketType） | TicketTableBean.type',
  `status` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '状态 | TicketTableBean.status',
  `is_has_full_view_image` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '是否有全场图 | TicketTableBean.hasFullViewImage',
  `icon_checked` VARCHAR(1024) COMMENT '选中图标 | TicketTableBean.iconChecked',
  `icon_unable` VARCHAR(1024) COMMENT '不可选图标 | TicketTableBean.iconUnable',
  `icon_un_checked` VARCHAR(1024) COMMENT '未选图标 | TicketTableBean.iconUnChecked',
  `icon_ticket_category` VARCHAR(1024) COMMENT '票档图标 | TicketTableBean.iconTicketCategory',
  `icon_unable_template` JSON COMMENT '模板（原 IconTemplateBean） | TicketTableBean.iconUnableTemplate',
  `package_discount` JSON COMMENT '套餐折扣（原 Map<String,Map<String,Double>>） | TicketTableBean.packageDiscount',
  `ticket_categories_icon` JSON COMMENT '票档图标映射 | TicketTableBean.ticketCategoriesIcon',
  `area_tickets` JSON COMMENT '区域票（原 List<TicketAreaBean>） | TicketTableBean.areaTickets',
  `ticket_categories` JSON COMMENT '票档（原 List<TicketCategory>） | TicketTableBean.ticketCategories',
  `areas` JSON COMMENT '区域（Flutter List<SeatArea>） | Flutter',
  `gmt_create` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `gmt_modified` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_seat_map_layout_layout_id` (`layout_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='选座图布局 | TicketTableBean + Flutter SeatMapData（原 seat_map_layouts）' ROW_FORMAT=DYNAMIC;

CREATE TABLE IF NOT EXISTS `discount_activity` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '代理主键',
  `activity_event_id` BIGINT UNSIGNED COMMENT '场次ID | DiscountActivityBean.activityEventId',
  `location_type` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '位置类型 | DiscountActivityBean.locationType',
  `origin_price` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '原价 | DiscountActivityBean.originPrice',
  `sale_price` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '售价 | DiscountActivityBean.salePrice',
  `ticket_category_id` BIGINT UNSIGNED COMMENT '票档ID | DiscountActivityBean.ticketCategoryId',
  `ticket_group_id` BIGINT UNSIGNED COMMENT '票组ID | DiscountActivityBean.ticketGroupId',
  `gmt_create` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `gmt_modified` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='折扣活动 | DiscountActivityBean（首页 discounts 楼层）（原 discount_activities）' ROW_FORMAT=DYNAMIC;
