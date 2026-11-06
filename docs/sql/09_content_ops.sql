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
  `banner_id` BIGINT UNSIGNED NOT NULL COMMENT 'ID | Banner.id / AdBean.id',
  `type` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '类型 | Banner.type',
  `city_id` BIGINT UNSIGNED COMMENT '城市ID | Banner.cityId',
  `platform` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '平台 | Banner.platform',
  `poster` VARCHAR(1024) COMMENT '图片 | Banner.poster',
  `bg_color` VARCHAR(32) COMMENT '背景色 | Banner.bgColor',
  `jump_schema` VARCHAR(1024) COMMENT '跳转 scheme | Banner.schema',
  `url` VARCHAR(1024) COMMENT '链接 | Banner.url',
  `sort` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '排序 | Banner.sort',
  `status` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '状态 | Banner.status',
  `from_time` DATETIME COMMENT '生效开始 | Banner.fromTime',
  `to_time` DATETIME COMMENT '生效结束 | Banner.toTime',
  `log_banner_id` BIGINT UNSIGNED COMMENT '埋点ID | Banner.logBannerId',
  `log_banner_type` VARCHAR(64) COMMENT '埋点类型 | Banner.logBannerType',
  `gmt_create` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `gmt_modified` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_banner_banner_id` (`banner_id`),
  KEY `idx_banner_city_id_status` (`city_id`,`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='轮播图 / 广告位 | Banner / AdBean / HeaderBackgroundBean（原 banners）' ROW_FORMAT=DYNAMIC;

CREATE TABLE IF NOT EXISTS `ad` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '代理主键',
  `name` VARCHAR(256) COMMENT '名称 | IconsBean.name',
  `icon_tips` VARCHAR(512) COMMENT '图标提示 | IconsBean.iconTips',
  `poster` VARCHAR(1024) COMMENT '图片 | IconsBean.poster',
  `jump_schema` VARCHAR(1024) COMMENT '跳转 | IconsBean.schema',
  `android_schema` VARCHAR(1024) COMMENT 'Android scheme | BenefitIconBean.androidSchema',
  `ios_schema` VARCHAR(1024) COMMENT 'iOS scheme | BenefitIconBean.iosSchema',
  `sub_title` VARCHAR(256) COMMENT '副标题 | BenefitIconBean.subTitle',
  `title` VARCHAR(256) COMMENT '标题 | BenefitIconBean.title',
  `url` VARCHAR(1024) COMMENT '链接 | BenefitIconBean.url',
  `icon` VARCHAR(1024) COMMENT '图标 | BenefitIconBean.icon',
  `click_num` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '点击数 | FeedAdBean.clickNum',
  `users` JSON COMMENT '用户列表（原 List<UserBean>） | FeedAdBean.users',
  `log_op_position_id` BIGINT UNSIGNED COMMENT '埋点运营位ID | HomeCenterOpBean.logOpPositionId',
  `log_banner_to_object_type` VARCHAR(64) COMMENT '埋点对象类型 | HomeCenterOpBean.logBannerToObjectType',
  `gmt_create` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `gmt_modified` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='运营图标 / 权益位 / 悬浮球 | IconsBean / BenefitIconBean / FeedAdBean / HomeCenterOpBean（原 ads）' ROW_FORMAT=DYNAMIC;

CREATE TABLE IF NOT EXISTS `home_floor` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `floor_id` BIGINT UNSIGNED NOT NULL COMMENT '楼层ID | HomeFloorShowBean.floorId',
  `type` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '楼层类型（见 wiki 27 §5.1 三套编号） | HomeFloorBean.type',
  `title` VARCHAR(512) COMMENT '标题 | HomeFloorShowBean.title',
  `description` TEXT COMMENT '描述 | HomeFloorShowBean.desc',
  `display_type` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '展示类型 | HomeFloorShowBean.displayType',
  `data` TEXT COMMENT '原始 JSON（HomeFloorBean.data 永久保留） | HomeFloorBean.data',
  `search_hint` VARCHAR(512) COMMENT '搜索提示 | HomeFloors.searchHint',
  `calendar_tip` JSON COMMENT '日历提示（原 CalendarTip） | HomeFloors.calendarTip',
  `actors` JSON COMMENT '艺人楼层（原 HomeActorsBean） | HomeFloorBean.actors',
  `banners` JSON COMMENT '轮播（原 List<? extends Banner>） | HomeFloorBean.banners',
  `center_op` JSON COMMENT '中央运营位（原 HomeCenterOpBean） | HomeFloorBean.centerOp',
  `choiceness` JSON COMMENT '精选（原 HomeChoicenessBean） | HomeFloorBean.choiceness',
  `discovery_wonder` JSON COMMENT '发现精彩 | HomeFloorBean.discoveryWonder',
  `guarantee` JSON COMMENT '保障（原 HomeGuaranteeBean） | HomeFloorBean.guarantee',
  `home_show_list` JSON COMMENT '演出列表（原 HomeShow） | HomeFloorBean.homeShowList',
  `hot_actor_sites` JSON COMMENT '热门艺人站点 | HomeFloorBean.hotActorSites',
  `hot_search_keywords` JSON COMMENT '热搜词 | HomeFloorBean.hotSearchKeywords',
  `hot_tour_list` JSON COMMENT '热门巡演 | HomeFloorBean.hotTourList',
  `op` JSON COMMENT '运营位列表 | HomeFloorBean.op',
  `recommend_word_list` JSON COMMENT '推荐词 | HomeFloorBean.recommendWordList',
  `recommends` JSON COMMENT '瀑布流（原 ArrayList<HomeWaterfallItem>） | HomeFloorBean.recommends',
  `shortcuts` JSON COMMENT '快捷入口 | HomeFloorBean.shortcuts',
  `theme_recommend` JSON COMMENT '主题推荐（原 HomeThemeRecommend） | HomeFloorBean.themeRecommend',
  `venues` JSON COMMENT '场馆楼层（原 HomeVenuesBean） | HomeFloorBean.venues',
  `waterfall` JSON COMMENT '瀑布流配置（原 HomeWaterfall） | HomeFloorBean.waterfall',
  `gmt_create` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `gmt_modified` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_home_floor_floor_id` (`floor_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='首页楼层 / 内容楼层 | HomeFloorBean / HomeFloors / HomeFloorShowBean / HomeShow（原 home_floors）' ROW_FORMAT=DYNAMIC;

CREATE TABLE IF NOT EXISTS `op_position` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `position_id` BIGINT UNSIGNED NOT NULL COMMENT 'ID | OperatingPositionBean.id / launchScreenBean.id',
  `template_id` BIGINT UNSIGNED COMMENT '模板ID | OpPosition.templateId',
  `inner_title` VARCHAR(256) COMMENT '内标题 | OperatingPositionBean.innerTitle',
  `main_title` VARCHAR(256) COMMENT '主标题 | HomeOpbean.mainTitle',
  `sub_title` VARCHAR(256) COMMENT '副标题 | OperatingPositionBean.subTitle',
  `title` VARCHAR(256) COMMENT '标题 | OperatingPositionBean.title',
  `poster` VARCHAR(1024) COMMENT '图片 | OperatingPositionBean.poster',
  `poster_url` VARCHAR(1024) COMMENT '图片 URL | HomeOpbean.posterUrl',
  `jump_schema` VARCHAR(1024) COMMENT '跳转 | OperatingPositionBean.schema',
  `style_type` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '样式类型 | HomeOpbean.styleType',
  `sub_type` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '子类型 | HomeOpbean.subType',
  `activity_id` BIGINT UNSIGNED COMMENT '演出ID | HomeOpbean.activityId',
  `actor_name` VARCHAR(256) COMMENT '艺人名 | HomeOpbean.actorName',
  `low_price` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '最低价 | HomeOpbean.lowPrice',
  `name` VARCHAR(256) COMMENT '名称 | HomeOpbean.name',
  `is_sold_out` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '售罄 | HomeOpbean.soldOut',
  `data` JSON COMMENT '数据体（原 JSONObject / DataBean） | OpPosition.data',
  `items` JSON COMMENT '条目（原 List<HomeOpItemData>） | HomeOpbean.items',
  `icon` VARCHAR(1024) COMMENT '图标 | PopBoxBean.icon',
  `icon_schema` VARCHAR(1024) COMMENT '图标跳转 | PopBoxBean.iconSchema',
  `position_key` VARCHAR(128) COMMENT '键 | PopBoxBean.key',
  `type` VARCHAR(64) COMMENT '类型 | PopBoxBean.type',
  `url` VARCHAR(1024) COMMENT '链接 | PopBoxBean.url',
  `city_id` BIGINT UNSIGNED COMMENT '城市ID | launchScreenBean.cityId',
  `duration` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '时长 | launchScreenBean.duration',
  `video` VARCHAR(1024) COMMENT '视频 | launchScreenBean.video',
  `open_screen_type` VARCHAR(64) COMMENT '开屏类型 | launchScreenBean.openScreenType',
  `from_time` DATETIME COMMENT '生效开始 | launchScreenBean.fromTime',
  `to_time` DATETIME COMMENT '生效结束 | launchScreenBean.toTime',
  `status` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '状态 | launchScreenBean.status',
  `gmt_create` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `gmt_modified` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_op_position_position_id` (`position_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='运营位 / 弹窗 / 启动屏 | OpPosition / OperatingPositionBean / PopBoxBean / launchScreenBean / HomeOpbean（原 op_positions）' ROW_FORMAT=DYNAMIC;

CREATE TABLE IF NOT EXISTS `search_keyword` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '代理主键',
  `keyword` VARCHAR(256) COMMENT '关键词 | SchemaKeywordBean.keyword',
  `recommend_word` VARCHAR(256) COMMENT '推荐词 | SchemaKeywordBean.recommendWord',
  `jump_schema` VARCHAR(1024) COMMENT '跳转 | SchemaKeywordBean.schema',
  `log_id` BIGINT UNSIGNED COMMENT '埋点ID | SchemaKeywordRankBean.logId',
  `pn_local_rank` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '本地排名 | SchemaKeywordRankBean.pnLocalRank',
  `name` VARCHAR(256) COMMENT '名称 | SearchSuggestBean.name',
  `type` VARCHAR(64) COMMENT '类型 | SearchSuggestBean.type',
  `label` VARCHAR(256) COMMENT '展示文案 | Flutter HotKeyword.label',
  `word` VARCHAR(256) COMMENT '词 | Flutter HotKeyword.word',
  `hit_count` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '热度 | Flutter HotKeyword.hitCount',
  `is_hot` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '是否热 | Flutter HotKeyword.hot',
  `keyword_key` VARCHAR(128) COMMENT '搜索键 | SearchOption.key',
  `seat_map` VARCHAR(64) COMMENT '选座 | SearchOption.seatMap',
  `sort` VARCHAR(64) COMMENT '排序 | SearchOption.sort',
  `start_time` DATETIME COMMENT '开始时间 | SearchOption.startTime',
  `end_time` DATETIME COMMENT '结束时间 | SearchOption.endTime',
  `gmt_create` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `gmt_modified` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  KEY `idx_search_keyword_keyword` (`keyword`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='热搜词 / 联想词 / 搜索条件 | SchemaKeywordBean / SchemaKeywordRankBean / SearchSuggestBean / SearchOption / HotKeyword（原 search_keywords）' ROW_FORMAT=DYNAMIC;

CREATE TABLE IF NOT EXISTS `share_log` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '代理主键',
  `user_id` BIGINT UNSIGNED COMMENT '用户ID | 推断',
  `subject_type` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '主体类型 | 推断',
  `subject_id` BIGINT UNSIGNED COMMENT '主体ID | 推断',
  `channel` VARCHAR(64) COMMENT '渠道（WECHAT / MOMENTS / WEIBO / QQ / QZONE） | dict_enum.ShareChannelType',
  `title` VARCHAR(512) COMMENT '标题 | ShareModel.title / ShareParams.title',
  `description` TEXT COMMENT '描述 | ShareModel.desc / ShareParams.desc',
  `image_url` VARCHAR(1024) COMMENT '图片 | ShareModel.imageURL',
  `share_image_url` VARCHAR(1024) COMMENT '分享图 | ShareModel.shareImageUrl',
  `img_url` VARCHAR(1024) COMMENT '分享图 | ShareParams.imgUrl',
  `url` VARCHAR(1024) COMMENT '链接 | ShareModel.url',
  `mp_url` VARCHAR(1024) COMMENT '小程序链接 | ShareModel.mpUrl',
  `mp_image_url` VARCHAR(1024) COMMENT '小程序图 | ShareModel.mpImageUrl',
  `link` VARCHAR(1024) COMMENT '链接 | ShareParams.link',
  `mp_link` VARCHAR(1024) COMMENT '小程序链接 | ShareParams.mpLink',
  `is_share_reward_confirm` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '分享奖励确认 | v1/shareReward/confirm',
  `gmt_create` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `gmt_modified` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  KEY `idx_share_log_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='分享记录 / 分享参数 | ShareModel / ShareParams / ShareChannelType / v1/share/log（原 share_logs）' ROW_FORMAT=DYNAMIC;

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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='意见反馈 | v1/feedback（原 feedback）' ROW_FORMAT=DYNAMIC;

CREATE TABLE IF NOT EXISTS `tracking_event` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '代理主键',
  `action` VARCHAR(64) COMMENT '动作（点击 / enter） | PNLogAction',
  `event_key` VARCHAR(128) COMMENT '事件 key | PNLogger',
  `keyname` VARCHAR(256) COMMENT '事件名 | PNLogger',
  `jump_schema` VARCHAR(256) COMMENT '页面 scheme | PNSchemeMap',
  `schemaname` VARCHAR(256) COMMENT '页面名 | PNSchemeMap',
  `params` JSON COMMENT '参数 | PNLogger.sendLog',
  `city_id` BIGINT UNSIGNED COMMENT '城市ID | 埋点公共字段',
  `dvid` VARCHAR(128) COMMENT '设备ID | 埋点公共字段',
  `user_id` BIGINT UNSIGNED COMMENT '用户ID | 埋点公共字段',
  `channel` VARCHAR(128) COMMENT '渠道 | 埋点公共字段',
  `referer` VARCHAR(256) COMMENT '来源页 | 埋点公共字段',
  `gmt_create` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `gmt_modified` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  KEY `idx_tracking_event_user_id_gmt_create` (`user_id`,`gmt_create`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='埋点事件 | PNLogger / PNLogUploader（https://track 域/log）（原 tracking_events）' ROW_FORMAT=DYNAMIC;
