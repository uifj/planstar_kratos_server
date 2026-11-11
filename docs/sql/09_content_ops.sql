-- ========================================================================
--  sql · 09_content_ops.sql（S2 全量规范重构版）
--  规范: 单数表名 / id BIGINT UNSIGNED 自增 / gmt_create,gmt_modified DATETIME
--        is_xxx TINYINT UNSIGNED / 金额 DECIMAL / 无物理外键 / 无冗余索引
--  表: banner, ad, home_floor, op_position, search_keyword, share_log, user_feedback, tracking_event
-- ========================================================================

USE `planstar_content`;

-- ----------------------------------------------------------------------
--  域 7 · 运营位 / 搜索 / 增长（ops / search / growth）
-- ----------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS `banner` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `banner_id` BIGINT UNSIGNED NOT NULL COMMENT 'ID',
  `type` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '类型',
  `city_id` BIGINT UNSIGNED COMMENT '城市ID',
  `platform` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '平台',
  `poster` VARCHAR(1024) COMMENT '图片',
  `bg_color` VARCHAR(32) COMMENT '背景色',
  `jump_schema` VARCHAR(1024) COMMENT '跳转 scheme',
  `url` VARCHAR(1024) COMMENT '链接',
  `sort` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '排序',
  `status` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '状态',
  `from_time` DATETIME COMMENT '生效开始',
  `to_time` DATETIME COMMENT '生效结束',
  `log_banner_id` BIGINT UNSIGNED COMMENT '埋点ID',
  `log_banner_type` VARCHAR(64) COMMENT '埋点类型',
  `gmt_create` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `gmt_modified` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_banner_banner_id` (`banner_id`),
  KEY `idx_banner_city_id_status` (`city_id`,`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='轮播图 / 广告位（原 banners）' ROW_FORMAT=DYNAMIC;

CREATE TABLE IF NOT EXISTS `ad` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '代理主键',
  `name` VARCHAR(256) COMMENT '名称',
  `icon_tips` VARCHAR(512) COMMENT '图标提示',
  `poster` VARCHAR(1024) COMMENT '图片',
  `jump_schema` VARCHAR(1024) COMMENT '跳转',
  `android_schema` VARCHAR(1024) COMMENT 'Android scheme',
  `ios_schema` VARCHAR(1024) COMMENT 'iOS scheme',
  `sub_title` VARCHAR(256) COMMENT '副标题',
  `title` VARCHAR(256) COMMENT '标题',
  `url` VARCHAR(1024) COMMENT '链接',
  `icon` VARCHAR(1024) COMMENT '图标',
  `click_num` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '点击数',
  `users` JSON COMMENT '用户列表',
  `log_op_position_id` BIGINT UNSIGNED COMMENT '埋点运营位ID',
  `log_banner_to_object_type` VARCHAR(64) COMMENT '埋点对象类型',
  `gmt_create` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `gmt_modified` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='运营图标 / 权益位 / 悬浮球（原 ads）' ROW_FORMAT=DYNAMIC;

CREATE TABLE IF NOT EXISTS `home_floor` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `floor_id` BIGINT UNSIGNED NOT NULL COMMENT '楼层ID',
  `type` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '楼层类型（见 wiki 27 §5.1 三套编号）',
  `title` VARCHAR(512) COMMENT '标题',
  `description` TEXT COMMENT '描述',
  `display_type` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '展示类型',
  `data` TEXT COMMENT '原始 JSON（HomeFloorBean.data 永久保留）',
  `search_hint` VARCHAR(512) COMMENT '搜索提示',
  `calendar_tip` JSON COMMENT '日历提示',
  `actors` JSON COMMENT '艺人楼层',
  `banners` JSON COMMENT '轮播',
  `center_op` JSON COMMENT '中央运营位',
  `choiceness` JSON COMMENT '精选',
  `discovery_wonder` JSON COMMENT '发现精彩',
  `guarantee` JSON COMMENT '保障',
  `home_show_list` JSON COMMENT '演出列表',
  `hot_actor_sites` JSON COMMENT '热门艺人站点',
  `hot_search_keywords` JSON COMMENT '热搜词',
  `hot_tour_list` JSON COMMENT '热门巡演',
  `op` JSON COMMENT '运营位列表',
  `recommend_word_list` JSON COMMENT '推荐词',
  `recommends` JSON COMMENT '瀑布流',
  `shortcuts` JSON COMMENT '快捷入口',
  `theme_recommend` JSON COMMENT '主题推荐',
  `venues` JSON COMMENT '场馆楼层',
  `waterfall` JSON COMMENT '瀑布流配置',
  `gmt_create` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `gmt_modified` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_home_floor_floor_id` (`floor_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='首页楼层 / 内容楼层（原 home_floors）' ROW_FORMAT=DYNAMIC;

CREATE TABLE IF NOT EXISTS `op_position` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `position_id` BIGINT UNSIGNED NOT NULL COMMENT 'ID',
  `template_id` BIGINT UNSIGNED COMMENT '模板ID',
  `inner_title` VARCHAR(256) COMMENT '内标题',
  `main_title` VARCHAR(256) COMMENT '主标题',
  `sub_title` VARCHAR(256) COMMENT '副标题',
  `title` VARCHAR(256) COMMENT '标题',
  `poster` VARCHAR(1024) COMMENT '图片',
  `poster_url` VARCHAR(1024) COMMENT '图片 URL',
  `jump_schema` VARCHAR(1024) COMMENT '跳转',
  `style_type` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '样式类型',
  `sub_type` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '子类型',
  `activity_id` BIGINT UNSIGNED COMMENT '演出ID',
  `actor_name` VARCHAR(256) COMMENT '艺人名',
  `low_price` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '最低价',
  `name` VARCHAR(256) COMMENT '名称',
  `is_sold_out` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '售罄',
  `data` JSON COMMENT '数据体',
  `items` JSON COMMENT '条目',
  `icon` VARCHAR(1024) COMMENT '图标',
  `icon_schema` VARCHAR(1024) COMMENT '图标跳转',
  `position_key` VARCHAR(128) COMMENT '键',
  `type` VARCHAR(64) COMMENT '类型',
  `url` VARCHAR(1024) COMMENT '链接',
  `city_id` BIGINT UNSIGNED COMMENT '城市ID',
  `duration` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '时长',
  `video` VARCHAR(1024) COMMENT '视频',
  `open_screen_type` VARCHAR(64) COMMENT '开屏类型',
  `from_time` DATETIME COMMENT '生效开始',
  `to_time` DATETIME COMMENT '生效结束',
  `status` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '状态',
  `gmt_create` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `gmt_modified` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_op_position_position_id` (`position_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='运营位 / 弹窗 / 启动屏（原 op_positions）' ROW_FORMAT=DYNAMIC;

CREATE TABLE IF NOT EXISTS `search_keyword` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '代理主键',
  `keyword` VARCHAR(256) COMMENT '关键词',
  `recommend_word` VARCHAR(256) COMMENT '推荐词',
  `jump_schema` VARCHAR(1024) COMMENT '跳转',
  `log_id` BIGINT UNSIGNED COMMENT '埋点ID',
  `pn_local_rank` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '本地排名',
  `name` VARCHAR(256) COMMENT '名称',
  `type` VARCHAR(64) COMMENT '类型',
  `label` VARCHAR(256) COMMENT '展示文案 | Flutter HotKeyword.label',
  `word` VARCHAR(256) COMMENT '词 | Flutter HotKeyword.word',
  `hit_count` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '热度 | Flutter HotKeyword.hitCount',
  `is_hot` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '是否热 | Flutter HotKeyword.hot',
  `keyword_key` VARCHAR(128) COMMENT '搜索键',
  `seat_map` VARCHAR(64) COMMENT '选座',
  `sort` VARCHAR(64) COMMENT '排序',
  `start_time` DATETIME COMMENT '开始时间',
  `end_time` DATETIME COMMENT '结束时间',
  `gmt_create` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `gmt_modified` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  KEY `idx_search_keyword_keyword` (`keyword`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='热搜词 / 联想词 / 搜索条件 | HotKeyword（原 search_keywords）' ROW_FORMAT=DYNAMIC;

CREATE TABLE IF NOT EXISTS `share_log` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '代理主键',
  `user_id` BIGINT UNSIGNED COMMENT '用户ID | 推断',
  `subject_type` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '主体类型 | 推断',
  `subject_id` BIGINT UNSIGNED COMMENT '主体ID | 推断',
  `channel` VARCHAR(64) COMMENT '渠道（WECHAT / MOMENTS / WEIBO / QQ / QZONE） | dict_enum.ShareChannelType',
  `title` VARCHAR(512) COMMENT '标题',
  `description` TEXT COMMENT '描述',
  `image_url` VARCHAR(1024) COMMENT '图片',
  `share_image_url` VARCHAR(1024) COMMENT '分享图',
  `img_url` VARCHAR(1024) COMMENT '分享图',
  `url` VARCHAR(1024) COMMENT '链接',
  `mp_url` VARCHAR(1024) COMMENT '小程序链接',
  `mp_image_url` VARCHAR(1024) COMMENT '小程序图',
  `link` VARCHAR(1024) COMMENT '链接',
  `mp_link` VARCHAR(1024) COMMENT '小程序链接',
  `is_share_reward_confirm` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '分享奖励确认',
  `gmt_create` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `gmt_modified` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  KEY `idx_share_log_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='分享记录 / 分享参数 | ShareChannelType（原 share_logs）' ROW_FORMAT=DYNAMIC;

CREATE TABLE IF NOT EXISTS `user_feedback` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '代理主键',
  `user_id` BIGINT UNSIGNED COMMENT '用户ID | 推断',
  `content` TEXT COMMENT '反馈内容 | 推断',
  `contact` VARCHAR(256) COMMENT '联系方式 | 推断',
  `type` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '类型 | 推断',
  `status` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '状态 | 推断',
  `images` JSON COMMENT '图片列表 | 推断',
  `gmt_create` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `gmt_modified` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='意见反馈（原 feedback）' ROW_FORMAT=DYNAMIC;

CREATE TABLE IF NOT EXISTS `tracking_event` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '代理主键',
  `action` VARCHAR(64) COMMENT '动作（点击 / enter）',
  `event_key` VARCHAR(128) COMMENT '事件 key',
  `keyname` VARCHAR(256) COMMENT '事件名',
  `jump_schema` VARCHAR(256) COMMENT '页面 scheme',
  `schemaname` VARCHAR(256) COMMENT '页面名',
  `params` JSON COMMENT '参数',
  `city_id` BIGINT UNSIGNED COMMENT '城市ID | 埋点公共字段',
  `dvid` VARCHAR(128) COMMENT '设备ID | 埋点公共字段',
  `user_id` BIGINT UNSIGNED COMMENT '用户ID | 埋点公共字段',
  `channel` VARCHAR(128) COMMENT '渠道 | 埋点公共字段',
  `referer` VARCHAR(256) COMMENT '来源页 | 埋点公共字段',
  `gmt_create` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `gmt_modified` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  KEY `idx_tracking_event_user_id_gmt_create` (`user_id`,`gmt_create`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='埋点事件（原 tracking_events）' ROW_FORMAT=DYNAMIC;
