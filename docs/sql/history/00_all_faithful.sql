-- ========================================================================
--  sql_rebuildnew · 00_all_faithful.sql —— 忠实原样版（0 ERROR / 119 WARNING 1681）
--  与 docs/sql_new 的 SQL 正文逐语句一致，仅替换过时头部；不做 R4 的 TINYINT 归一化。
--  用于对照：证明「119 条告警全部来自 TINYINT(1) 显示宽度」，除此之外无差异。
-- ========================================================================

SET NAMES utf8mb4;



-- ------------------------------------------------------------------------
--  ↓ 以下为 01_init_databases.sql
-- ------------------------------------------------------------------------

-- ========================================================================
--  sql_rebuildnew · 01_init_databases.sql（忠实原样版，不做 R4）
--  SQL 正文与 docs/sql_new 逐语句一致，仅替换过时生成头部。
--  在 MySQL 8.4.10 上会产出 119 条 Warning 1681（TINYINT(1) 显示宽度弃用），0 ERROR。
-- ========================================================================

-- ========================================================================
--  01_init_databases.sql · 初始化企划星各服务库
-- ========================================================================
--  目录结构参考: github.com/arixbit/kratos-shop/sql/01_init_databases.sql
--  差异: 参考仓库用 PostgreSQL 的 \gexec 幂等建库；本文件保持 MySQL 8.0，
--        使用 CREATE DATABASE IF NOT EXISTS，语义等价且可直接执行。
--  参考仓库 6 个库；本票务域拆为 9 个库（见下）。
-- ========================================================================

-- planstar_user · 用户与账号
CREATE DATABASE IF NOT EXISTS `planstar_user` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- planstar_catalog · 基础字典与演出商品
CREATE DATABASE IF NOT EXISTS `planstar_catalog` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- planstar_ticket · 票档 / 座位 / 库存
CREATE DATABASE IF NOT EXISTS `planstar_ticket` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- planstar_order · 订单
CREATE DATABASE IF NOT EXISTS `planstar_order` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- planstar_payment · 支付 / 退款 / 钱包 / 礼品卡
CREATE DATABASE IF NOT EXISTS `planstar_payment` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- planstar_marketing · 营销资产
CREATE DATABASE IF NOT EXISTS `planstar_marketing` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- planstar_community · 内容与社区 UGC
CREATE DATABASE IF NOT EXISTS `planstar_community` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- planstar_content · 运营位 / 搜索 / 增长
CREATE DATABASE IF NOT EXISTS `planstar_content` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- planstar_meta · 枚举字典 + 逆向证据目录
CREATE DATABASE IF NOT EXISTS `planstar_meta` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- ========================================================================
--  后续文件按 02 → 12 顺序执行；每个 NN_<域>.sql 开头 USE 对应服务库。
-- ========================================================================


-- ------------------------------------------------------------------------
--  ↓ 以下为 02_user.sql
-- ------------------------------------------------------------------------

-- ========================================================================
--  sql_rebuildnew · 02_user.sql（忠实原样版，不做 R4）
--  SQL 正文与 docs/sql_new 逐语句一致，仅替换过时生成头部。
--  在 MySQL 8.4.10 上会产出 119 条 Warning 1681（TINYINT(1) 显示宽度弃用），0 ERROR。
-- ========================================================================

-- ========================================================================
--  02 · 用户与账号
--  服务库: planstar_user
--  执行顺序: 01_init_databases.sql → 02 … 09 → 10 / 11 → 12_constraints.sql
--  本文件 9 张表: users, login_sessions, real_name_infos, user_addresses, user_follows, user_statistics, user_messages, chat_threads, chat_messages
-- ========================================================================
--  目录结构参考: github.com/arixbit/kratos-shop/tree/main/sql
--    NN_<域>.sql 按服务分库 + migrations/<库>/000001_init.up.sql / .down.sql
--  方言: MySQL 8.0 / InnoDB / utf8mb4（按用户选择保持，未切换 PostgreSQL）
--  第三方 SDK 表（小米推送 / 极光 JPush / 百度定位·mshield / Bugly /
--    字节 AppLog·bdtracker）共 31 张已按要求全部移除。
-- ========================================================================

USE `planstar_user`;

-- ----------------------------------------------------------------------
--  域 1 · 账号与身份（account / identity）
-- ----------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS `users` (
  `user_id` BIGINT NOT NULL COMMENT '用户ID | UserBean.userId（原 Java int，逆向统一放宽）',
  `user_name` VARCHAR(128) COMMENT '用户名 | UserBean.userName',
  `avatar` VARCHAR(512) COMMENT '头像 URL | UserBean.avatar',
  `badge_icon` VARCHAR(512) COMMENT '徽章图标 | UserBean.badgeIcon',
  `bio` VARCHAR(1024) COMMENT '个人简介 | UserBean.bio',
  `birthday` BIGINT COMMENT '生日（epoch 毫秒）| UserBean.birthday',
  `gender` INT COMMENT '性别 | UserBean.gender',
  `mobile_no` VARCHAR(32) COMMENT '手机号 | UserBean.mobileNo',
  `phone` VARCHAR(32) COMMENT '手机号（UserProfile 冗余字段）| UserProfile.phone',
  `level` INT COMMENT '用户等级 | UserBean.level',
  `level_desc` VARCHAR(128) COMMENT '等级描述 | UserProfile.levelDesc',
  `score` DOUBLE COMMENT '评分 / 信用分 | UserBean.score',
  `status` INT COMMENT '账号状态 | UserBean.status',
  `shop_type` INT COMMENT '店铺类型（卖家标识）| UserBean.shopType',
  `is_douli_user` TINYINT(1) COMMENT '是否斗牛用户 | UserBean.isDouliUser',
  `blocked` TINYINT(1) COMMENT '是否被拉黑 | UserBean.blocked',
  `follow_him` INT COMMENT '我关注他（原字段名 followed）| UserBean.followed',
  `followed_by_him` INT COMMENT '他关注我 | UserBean.followedByHim',
  `follower_num` INT COMMENT '粉丝数 | UserBean.followerNum',
  `following_num` INT COMMENT '关注数（仅用户）| UserBean.followingNum',
  `force_follow` TINYINT(1) COMMENT '强制关注 | UserBean.forceFollow',
  `review_liked_num` INT COMMENT '获赞数 | UserBean.reviewLikedNum',
  `want_watch_num` INT COMMENT '想看数 | UserBean.wantWatchNum',
  `watched_num` INT COMMENT '看过数 | UserBean.watchedNum',
  `add_time` BIGINT COMMENT '创建时间（epoch 毫秒）| UserBean.addTime',
  `update_time` BIGINT COMMENT '更新时间（epoch 毫秒）| UserBean.updateTime',
  `ext` JSON COMMENT '扩展字段 | 预留',
  PRIMARY KEY (`user_id`),
  KEY `idx_users_mobile` (`mobile_no`),
  KEY `idx_users_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户主档 | Android UserBean / UserProfile / LoginBean + Flutter AuthUser';

CREATE TABLE IF NOT EXISTS `login_sessions` (
  `session_id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '代理主键',
  `user_id` BIGINT COMMENT '用户ID | LoginBean.user.userId',
  `login_type` INT COMMENT '登录方式（密码 / 短信 / 三方）| v1/user/login · tpLogin',
  `access_token` VARCHAR(512) COMMENT '访问令牌 | TPLoginBean.accessToken',
  `union_id` VARCHAR(128) COMMENT '三方 UnionID | TPLoginBean.unionId',
  `is_bound` TINYINT(1) COMMENT '是否已绑定 | TPLoginBean.isBound',
  `new_user` TINYINT(1) COMMENT '是否新用户 | LoginBean.newUser',
  `need_captcha` TINYINT(1) COMMENT '是否需要图形验证码 | LoginBean.needCaptcha',
  `captcha_cooldown` INT COMMENT '验证码冷却秒数（客户端 60 秒倒计时依据）| BaseLoginActivity',
  `error_msg` VARCHAR(512) COMMENT '错误提示 | SendCodeResponseDto.errorMsg',
  `success` TINYINT(1) COMMENT '请求是否成功 | SendCodeResponseDto.success',
  `device_id` VARCHAR(128) COMMENT '设备标识 | dvid',
  `add_time` BIGINT COMMENT '登录时间 | 推断',
  `expire_time` BIGINT COMMENT '过期时间 | 推断',
  PRIMARY KEY (`session_id`),
  KEY `idx_login_user` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='登录会话 / 三方登录 / 验证码 | LoginBean / TPLoginBean / SendCodeResponseDto';

CREATE TABLE IF NOT EXISTS `real_name_infos` (
  `id` BIGINT NOT NULL COMMENT '实名信息ID | IdInfo.id',
  `user_id` BIGINT COMMENT '所属用户 | v1/user/realNameInfo/save',
  `name` VARCHAR(128) COMMENT '姓名 | IdInfo.name',
  `id_card_no` VARCHAR(64) COMMENT '证件号 | IdInfo.idCardNo / Visitor.idCard.value',
  `id_card_type` INT COMMENT '证件类型（1 身份证 / 2 护照 / 3 港澳通行证 / 4 台湾通行证 / 5 港澳台居住证）| IdCardType',
  `country_area_id` INT COMMENT '国家/地区ID | IdInfo.countryAreaId',
  `country_area_name` VARCHAR(128) COMMENT '国家/地区名 | IdInfo.countryAreaName',
  `status` INT COMMENT '状态 | 推断',
  `add_time` BIGINT COMMENT '创建时间 | 推断',
  `update_time` BIGINT COMMENT '更新时间 | 推断',
  PRIMARY KEY (`id`),
  KEY `idx_realname_user` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='实名信息 / 观演人 | IdInfo / IdCard / Visitor / realNameInfo';

CREATE TABLE IF NOT EXISTS `user_addresses` (
  `address_id` BIGINT NOT NULL COMMENT '地址ID | AddressBean.id',
  `user_id` BIGINT COMMENT '用户ID | AddressBean.userId',
  `name` VARCHAR(128) COMMENT '收件人 | AddressBean.name',
  `phone` VARCHAR(32) COMMENT '联系电话 | AddressBean.phone',
  `province` VARCHAR(64) COMMENT '省 | Flutter ShippingAddress.region',
  `city` VARCHAR(64) COMMENT '市 | AddressBean.district 解析',
  `district` VARCHAR(64) COMMENT '区 | AddressBean.district',
  `district_id` INT COMMENT '区县ID | AddressBean.districtId',
  `detail_address` TEXT COMMENT '详细地址 | AddressBean.address',
  `address_type` INT COMMENT '地址类型 | AddressBean.addressType',
  `is_default` TINYINT(1) COMMENT '是否默认 | AddressBean.isDefault',
  `is_recent` TINYINT(1) COMMENT '是否最近使用 | AddressBean.isRecent',
  `latest_id` BIGINT COMMENT '最近地址ID（MyAddressBean）| MyAddressBean.latestId',
  `receiver_name` VARCHAR(128) COMMENT '最近收件人 | MyAddressBean.receiverName',
  `receiver_phone` VARCHAR(32) COMMENT '最近收件电话 | MyAddressBean.receiverPhone',
  `status` INT COMMENT '状态 | AddressBean.status',
  `add_time` BIGINT COMMENT '创建时间 | AddressBean.addTime',
  `update_time` BIGINT COMMENT '更新时间 | AddressBean.updateTime',
  PRIMARY KEY (`address_id`),
  KEY `idx_addr_user` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='收货地址 | AddressBean / MyAddressBean + Flutter ShippingAddress';

CREATE TABLE IF NOT EXISTS `user_follows` (
  `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '代理主键',
  `user_id` BIGINT COMMENT '发起人 | v1/user/follow',
  `target_type` INT COMMENT '目标类型（1 用户 / 2 艺人 / 3 群组）| 推断',
  `target_id` BIGINT COMMENT '目标ID | actorId / userId',
  `status` INT COMMENT '状态 | 推断',
  `add_time` BIGINT COMMENT '关注时间 | 推断',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_follow` (`user_id`,`target_type`,`target_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='关注关系（用户 / 艺人）| v1/user/follow · v1/actorFollow · following/actors · following/mine';

CREATE TABLE IF NOT EXISTS `user_statistics` (
  `user_id` BIGINT NOT NULL COMMENT '用户ID',
  `actor_following` INT COMMENT '关注艺人数（与 following_num 相加为总关注数）| StatisticBean.actorFollowing',
  `balance` DECIMAL(12,2) COMMENT '钱包余额（元）| StatisticBean.balance',
  `coupon_num` INT COMMENT '优惠券张数 | StatisticBean.couponNum',
  `follower_num` INT COMMENT '粉丝数 | StatisticBean.followerNum',
  `following_num` INT COMMENT '关注用户数 | StatisticBean.followingNum',
  `getting_tickets_num` INT COMMENT '待收货 | StatisticBean.gettingTicketsNum',
  `gift_card_num` INT COMMENT '礼品卡张数 | StatisticBean.giftCardNum',
  `msg_num` INT COMMENT '未读消息数 | StatisticBean.msgNum',
  `paying_num` INT COMMENT '待付款 | StatisticBean.payingNum',
  `review_num` INT COMMENT '评论数 | StatisticBean.reviewNum',
  `review_todo_num` INT COMMENT '待评论 | StatisticBean.reviewTodoNum',
  `waiting_for_tickets_num` INT COMMENT '待发货 | StatisticBean.waitingForTicketsNum',
  `want_watch_num` INT COMMENT '想看数 | StatisticBean.wantWatchNum',
  `update_time` BIGINT COMMENT '统计时间 | 推断',
  PRIMARY KEY (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户中心角标统计 | StatisticBean（GET v1/user/stat）';

CREATE TABLE IF NOT EXISTS `user_messages` (
  `message_id` BIGINT NOT NULL COMMENT '消息ID | UserMessage.id',
  `user_id` BIGINT COMMENT '接收用户 | UserMessage.userId',
  `from_user_id` BIGINT COMMENT '发送用户 | UserMessage.fromUserId',
  `from_user` JSON COMMENT '发送人快照（原 UserBean）| UserMessage.fromUser',
  `title` VARCHAR(512) COMMENT '标题 | UserMessage.title',
  `content` TEXT COMMENT '正文 | UserMessage.content',
  `msg_type` INT COMMENT '消息类型（见 dict_enums.UserMsgType）| UserMessage.type',
  `status` INT COMMENT '状态 | UserMessage.status',
  `url` VARCHAR(1024) COMMENT '跳转链接 | UserMessage.url',
  `unread_count` INT COMMENT '未读数 | UserMessageSimple.unReadCount',
  `add_time` BIGINT COMMENT '创建时间 | UserMessage.addTime',
  `update_time` BIGINT COMMENT '更新时间 | UserMessage.updateTime',
  PRIMARY KEY (`message_id`),
  KEY `idx_msg_user_type` (`user_id`,`msg_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='消息中心 | UserMessage / UserMessageSimple / UserMsgCount / SystemMessageEntry';

CREATE TABLE IF NOT EXISTS `chat_threads` (
  `thread_id` VARCHAR(64) NOT NULL COMMENT '会话ID | Flutter MessageThread.id',
  `user_id` BIGINT COMMENT '用户ID',
  `title` VARCHAR(256) COMMENT '会话标题 | MessageThread.title',
  `preview` VARCHAR(1024) COMMENT '最后一条预览 | MessageThread.preview',
  `updated_at` BIGINT COMMENT '最后更新时间 | MessageThread.updatedAt',
  `unread_count` INT COMMENT '未读数 | MessageThread.unreadCount',
  `avatar_url` VARCHAR(512) COMMENT '头像（开源图床 URL）| MessageThread.avatarFallbackUrl',
  `is_official` TINYINT(1) COMMENT '是否官方客服 | MessageThread.isOfficial',
  `visitor_token` VARCHAR(256) COMMENT '环信 visitor token | v1/chat/visitor',
  PRIMARY KEY (`thread_id`),
  KEY `idx_thread_user` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='客服 / 会话线程 | ChatService v1/chat/user · v1/chat/visitor + Flutter MessageThread';

CREATE TABLE IF NOT EXISTS `chat_messages` (
  `id` VARCHAR(64) NOT NULL COMMENT '消息ID | ChatMessage.id',
  `thread_id` VARCHAR(64) COMMENT '会话ID',
  `text` TEXT COMMENT '消息文本 | ChatMessage.text',
  `sent_at` BIGINT COMMENT '发送时间 | ChatMessage.sentAt',
  `from_me` TINYINT(1) COMMENT '是否自己发出 | ChatMessage.fromMe',
  `sender_name` VARCHAR(128) COMMENT '发送者名 | ChatMessage.senderName',
  PRIMARY KEY (`id`),
  KEY `idx_chatmsg_thread` (`thread_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='会话消息 | Flutter ChatMessage';


-- ------------------------------------------------------------------------
--  ↓ 以下为 03_catalog.sql
-- ------------------------------------------------------------------------

-- ========================================================================
--  sql_rebuildnew · 03_catalog.sql（忠实原样版，不做 R4）
--  SQL 正文与 docs/sql_new 逐语句一致，仅替换过时生成头部。
--  在 MySQL 8.4.10 上会产出 119 条 Warning 1681（TINYINT(1) 显示宽度弃用），0 ERROR。
-- ========================================================================

-- ========================================================================
--  03 · 基础字典与演出商品
--  服务库: planstar_catalog
--  执行顺序: 01_init_databases.sql → 02 … 09 → 10 / 11 → 12_constraints.sql
--  本文件 23 张表: cities, country_areas, venues, activity_categories, activity_sub_tags, app_configs, app_versions, activities, activity_events, activity_details, activity_lineups, activity_galleries, activity_refund_policies, activity_refund_rules, activity_refund_rule_details, activity_refund_relations, activity_campaigns, activity_subjects, actors, actor_qr_codes, tours, shops, organizers
-- ========================================================================
--  目录结构参考: github.com/arixbit/kratos-shop/tree/main/sql
--    NN_<域>.sql 按服务分库 + migrations/<库>/000001_init.up.sql / .down.sql
--  方言: MySQL 8.0 / InnoDB / utf8mb4（按用户选择保持，未切换 PostgreSQL）
--  第三方 SDK 表（小米推送 / 极光 JPush / 百度定位·mshield / Bugly /
--    字节 AppLog·bdtracker）共 31 张已按要求全部移除。
-- ========================================================================

USE `planstar_catalog`;

-- ----------------------------------------------------------------------
--  域 2 · 基础字典与配置（dictionary / config）
-- ----------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS `cities` (
  `city_id` INT NOT NULL COMMENT '城市ID | CityBean.cityId',
  `city_name` VARCHAR(128) COMMENT '城市名 | CityBean.cityName',
  `city_en_name` VARCHAR(128) COMMENT '英文名 | CityBean.cityEnName',
  `city_abbr_code` VARCHAR(32) COMMENT '缩写码 | CityBean.cityAbbrCode',
  `first_letter` VARCHAR(8) COMMENT '首字母 | CityBean.firstLetter',
  `hot_city` TINYINT(1) COMMENT '是否热门城市 | CityBean.hotCity',
  `province_id` INT COMMENT '省份ID | CityBean.provinceId',
  `region_city_id` INT COMMENT '区域城市ID | CityBean.regionCityId',
  `status` INT COMMENT '状态 | CityBean.status',
  `g_lat` DOUBLE COMMENT '纬度 | CityBean.GLat',
  `g_lng` DOUBLE COMMENT '经度 | CityBean.GLng',
  PRIMARY KEY (`city_id`),
  KEY `idx_city_hot` (`hot_city`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='城市 | CityBean（v1/cities · v2/cities/active · v1/cities/hot）';

CREATE TABLE IF NOT EXISTS `country_areas` (
  `country_area_id` INT NOT NULL COMMENT 'ID | CountryArea.countryAreaId',
  `country_area_name` VARCHAR(128) COMMENT '名称 | CountryArea.countryAreaName',
  `country_area_en_name` VARCHAR(128) COMMENT '英文名 | CountryArea.countryAreaEnName',
  `country_area_code` VARCHAR(32) COMMENT '编码 | CountryArea.countryAreaCode',
  `country_area_pin_yin` VARCHAR(128) COMMENT '拼音 | CountryArea.countryAreaPinYin',
  `country_level` INT COMMENT '层级 | CountryArea.countryLevel',
  `is_common` INT COMMENT '是否常用 | CountryArea.isCommon',
  PRIMARY KEY (`country_area_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='国家/地区 | CountryArea（v1/countryArea/list）';

CREATE TABLE IF NOT EXISTS `venues` (
  `venue_id` BIGINT NOT NULL COMMENT '场馆ID | Venue.id',
  `name` VARCHAR(256) COMMENT '场馆名 | Venue.name',
  `address` VARCHAR(1024) COMMENT '地址 | Venue.address',
  `avatar` VARCHAR(512) COMMENT '头像图 | Venue.avatar',
  `city_id` INT COMMENT '城市ID | Venue.cityId',
  `region_id` INT COMMENT '区域ID | Venue.regionId',
  `latitude` DOUBLE COMMENT '纬度 | Venue.latitude',
  `longitude` DOUBLE COMMENT '经度 | Venue.longitude',
  `parking` TEXT COMMENT '停车信息 | Venue.parking',
  `phones` VARCHAR(512) COMMENT '联系电话 | Venue.phones',
  `transportation` TEXT COMMENT '交通信息 | Venue.transportation',
  `show_count` INT COMMENT '演出数 | Flutter VenueSummary.showCount',
  `show_ids` JSON COMMENT '关联演出ID列表 | Flutter VenueSummary.showIds',
  `add_time` BIGINT COMMENT '创建时间 | Venue.addTime',
  `update_time` BIGINT COMMENT '更新时间 | Venue.updateTime',
  PRIMARY KEY (`venue_id`),
  KEY `idx_venue_city` (`city_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='场馆 | Venue + Flutter VenueSummary（v1/venues/{id}）';

CREATE TABLE IF NOT EXISTS `activity_categories` (
  `id` INT NOT NULL COMMENT '分类ID | ShowCategory.id',
  `name` VARCHAR(128) COMMENT '分类名 | ShowCategory.name',
  `en_name` VARCHAR(128) COMMENT '英文名 | ShowCategory.enName',
  `pinyin` VARCHAR(128) COMMENT '拼音 | ShowCategory.pinyin',
  `status` INT COMMENT '状态（1 上架）| ShowCategory.status',
  `is_select` TINYINT(1) COMMENT '是否选中 | PnConfig.OptionsBean.isSelect',
  `type` VARCHAR(64) COMMENT '类型 | PnConfig.type',
  `value` VARCHAR(256) COMMENT '值 | PnConfig.value',
  `dacu` VARCHAR(256) COMMENT '大促标记 | PnConfig.dacu',
  `add_time` BIGINT COMMENT '创建时间 | PnConfig.addTime',
  `update_time` BIGINT COMMENT '更新时间 | PnConfig.updateTime',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='演出一级分类 | ShowCategory / PnConfig.CategoriesBean（v2/config/categories）';

CREATE TABLE IF NOT EXISTS `activity_sub_tags` (
  `id` INT NOT NULL COMMENT '标签ID | SubCategoryTag.id',
  `category_id` INT COMMENT '一级分类ID | SubCategoryTag.categoryId',
  `name` VARCHAR(128) COMMENT '标签名 | SubCategoryTag.name',
  `sub_category_id` VARCHAR(64) COMMENT '子分类ID | PnConfig.subCategoryId',
  `status` INT COMMENT '状态 | PnConfig.status',
  `add_time` BIGINT COMMENT '创建时间 | PnConfig.addTime',
  `update_time` BIGINT COMMENT '更新时间 | PnConfig.updateTime',
  PRIMARY KEY (`id`),
  KEY `idx_subtag_cat` (`category_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='演出二级标签 | SubCategoryTag / SubCategoryTagsBean';

CREATE TABLE IF NOT EXISTS `app_configs` (
  `id` BIGINT NOT NULL COMMENT '配置ID | PnConfig.id',
  `balance_pay` TINYINT(1) COMMENT '是否支持余额支付 | PnConfig.balancePay',
  `contact_phone` VARCHAR(64) COMMENT '客服电话 | PnConfig.contactPhone',
  `express_name` VARCHAR(128) COMMENT '快递名称 | PnConfig.expressName',
  `express_phone` VARCHAR(64) COMMENT '快递电话 | PnConfig.expressPhone',
  `express_address` VARCHAR(512) COMMENT '快递地址 | PnConfig.expressAddress',
  `self_get_address` VARCHAR(512) COMMENT '自取地址 | PnConfig.selfGetAddress',
  `self_get_time` VARCHAR(128) COMMENT '自取时间 | PnConfig.selfGetTime',
  `img_domain` VARCHAR(256) COMMENT '图片域名 | PnConfig.imgDomain',
  `img_ssl_domain` VARCHAR(256) COMMENT '图片 HTTPS 域名 | PnConfig.imgSslDomain',
  `is_time_to_dacu` TINYINT(1) COMMENT '是否大促期 | PnConfig.isTimeToDacu',
  `sms_captcha_on` TINYINT(1) COMMENT '是否开启短信验证码 | PnConfig.smsCaptchaOn',
  `ticket_enough_count` INT COMMENT '票量充足阈值 | PnConfig.ticketEnoughCount',
  `launch_screen` VARCHAR(1024) COMMENT '启动屏 | PnConfig.launchScreen',
  `payment_types` JSON COMMENT '支付方式列表 | PnConfig.paymentTypes',
  `receive_types` JSON COMMENT '收货方式列表 | PnConfig.receiveTypes',
  `payment_tips` JSON COMMENT '支付提示 | PnConfig.paymentTips',
  `activity_icons` JSON COMMENT '活动图标 | PnConfig.activityIcons',
  `search_navigation_v2` JSON COMMENT '搜索导航配置 | PnConfig.searchNavigationV2',
  `categories` JSON COMMENT '分类配置 | PnConfig.categories',
  `sub_category_tags` JSON COMMENT '二级标签 | PnConfig.subCategoryTags',
  `options` JSON COMMENT '选项列表 | PnConfig.options',
  `add_time` BIGINT COMMENT '创建时间 | PnConfig.addTime',
  `update_time` BIGINT COMMENT '更新时间 | PnConfig.updateTime',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='全局配置 | PnConfig（v2/config · v3/config）';

CREATE TABLE IF NOT EXISTS `app_versions` (
  `version_code` INT NOT NULL COMMENT '版本号 | UpdateInfo.versionCode',
  `version_name` VARCHAR(64) COMMENT '版本名 | UpdateInfo.versionName',
  `changelog` TEXT COMMENT '更新日志 | UpdateInfo.changelog',
  `install_url` VARCHAR(1024) COMMENT '安装地址 | UpdateInfo.installUrl',
  `is_force_update` TINYINT(1) COMMENT '是否强制更新 | UpdateInfo.isForceUpdate',
  `is_ignorable` TINYINT(1) COMMENT '是否可忽略 | UpdateInfo.isIgnorable',
  `tips_interval_days` INT COMMENT '提示间隔天数 | UpdateInfo.tipsIntervalDays',
  `type` VARCHAR(64) COMMENT '类型 | UpdateInfo.type',
  PRIMARY KEY (`version_code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='版本更新 | UpdateInfo（GET apps/latest/{id} · v2/apkTool/latest）';

-- ----------------------------------------------------------------------
--  域 3 · 演出商品（show / goods）
-- ----------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS `activities` (
  `id` BIGINT NOT NULL COMMENT '演出ID | ActivityBean.id（注意：activityId 是另一个独立字段，服务端有时只填一个）',
  `activity_id` BIGINT COMMENT '演出ID（冗余/别名）| ActivityBean.activityId',
  `name` VARCHAR(512) COMMENT '演出名 | ActivityBean.name',
  `shortname` VARCHAR(256) COMMENT '简称 | ActivityBean.shortname',
  `activity_shortname` VARCHAR(256) COMMENT '简称（冗余）| ActivityBean.activityShortname',
  `sub_category_name` VARCHAR(128) COMMENT '二级分类名 | ActivityBean.subCategoryName',
  `category_id` INT COMMENT '分类ID | ActivityBean.categoryId',
  `category_name` VARCHAR(128) COMMENT '分类名 | ActivityBean.categoryName',
  `category_hot_rank` VARCHAR(128) COMMENT '分类热度排名 | ActivityBean.categoryHotRank',
  `activity_type` INT COMMENT '演出类型 | ActivityBean.activityType',
  `show_type` INT COMMENT '演出形态 | ActivityBean.showType',
  `link_type` VARCHAR(64) COMMENT '链接类型 | ActivityBean.linkType',
  `type` INT COMMENT '类型 | ActivityBean.type',
  `type_desc` VARCHAR(256) COMMENT '类型描述 | ActivityBean.typeDesc',
  `actor_id` BIGINT COMMENT '艺人ID | ActivityBean.actorId',
  `venue_id` BIGINT COMMENT '场馆ID | ActivityBean.venueId',
  `venue_name` VARCHAR(256) COMMENT '场馆名 | ActivityBean.venueName',
  `venue_city_id` INT COMMENT '场馆城市ID | ActivityBean.venueCityId',
  `city_id` INT COMMENT '城市ID | ActivityBean.cityId',
  `city_name` VARCHAR(128) COMMENT '城市名 | ActivityBean.cityName',
  `shop_id` BIGINT COMMENT '店铺ID | ActivityBean.shopId',
  `desc` TEXT COMMENT '描述 | ActivityBean.desc',
  `poster` VARCHAR(1024) COMMENT '海报 | ActivityBean.poster',
  `img` VARCHAR(1024) COMMENT '图片 | ActivityBean.img',
  `promo_img` VARCHAR(1024) COMMENT '推广图 | ActivityBean.promoImg',
  `tag_image` VARCHAR(1024) COMMENT '角标图 | ActivityBean.tagImage',
  `area_image` VARCHAR(1024) COMMENT '区域图 | ActivityBean.areaImage',
  `icon` VARCHAR(1024) COMMENT '图标 | ActivityBean.icon',
  `video` VARCHAR(1024) COMMENT '视频 | ActivityBean.video',
  `start` BIGINT COMMENT '开始时间（epoch 毫秒）| ActivityBean.start',
  `end` BIGINT COMMENT '结束时间（epoch 毫秒）| ActivityBean.end',
  `open_time` BIGINT COMMENT '开售时间 | ActivityBean.openTime',
  `start_sell_time` BIGINT COMMENT '开卖时间 | ActivityBean.startSellTime',
  `time_range` VARCHAR(256) COMMENT '时间文案 | ActivityBean.timeRange',
  `duration` VARCHAR(128) COMMENT '时长 | ActivityBean.duration',
  `year` VARCHAR(16) COMMENT '年份 | ActivityBean.year',
  `distance` VARCHAR(64) COMMENT '距离 | ActivityBean.distance',
  `low_price` DECIMAL(12,2) COMMENT '最低价 | ActivityBean.lowPrice',
  `high_price` DECIMAL(12,2) COMMENT '最高价 | ActivityBean.highPrice',
  `discount_amount` DECIMAL(12,2) COMMENT '优惠金额 | ActivityBean.discountAmount',
  `discount_rate` DECIMAL(10,4) COMMENT '折扣率 | ActivityBean.discountRate',
  `privilege_discount_amount` DECIMAL(12,2) COMMENT '会员优惠金额 | ActivityBean.privilegeDiscountAmount',
  `rank` DOUBLE COMMENT '排序权重 | ActivityBean.rank',
  `star` INT COMMENT '星级 | ActivityBean.star',
  `review_num` INT COMMENT '评论数 | ActivityBean.reviewNum',
  `want_watch_num` INT COMMENT '想看数 | ActivityBean.wantWatchNum',
  `total_order_count` INT COMMENT '总订单数 | ActivityBean.totalOrderCount',
  `activity_event_num` INT COMMENT '场次数 | ActivityBean.activityEventNum',
  `activity_num` INT COMMENT '活动数 | ActivityBean.activityNum',
  `favor_type` INT COMMENT '收藏类型 | ActivityBean.favorType',
  `status` INT COMMENT '状态 | ActivityBean.status',
  `publish_status` INT COMMENT '发布状态 | ActivityBean.publishStatus',
  `searchable` INT COMMENT '是否可搜索 | ActivityBean.searchable',
  `sold_out` TINYINT(1) COMMENT '是否售罄 | ActivityBean.soldOut',
  `normal_sold_out` TINYINT(1) COMMENT '普通票售罄 | ActivityBean.normalSoldOut',
  `has_seat` TINYINT(1) COMMENT '是否有座 | ActivityBean.hasSeat',
  `area_ticket_type` INT COMMENT '区域票类型 | ActivityBean.areaTicketType',
  `activity_seat_map_opened` TINYINT(1) COMMENT '选座图是否开放 | ActivityBean.activitySeatMapOpened',
  `b2c_opened` TINYINT(1) COMMENT 'B2C 是否开放 | ActivityBean.b2cOpened',
  `credit_pay` TINYINT(1) COMMENT '是否支持信用付 | ActivityBean.creditPay',
  `can_buy_ecard` TINYINT(1) COMMENT '是否可买电子卡 | ActivityBean.canBuyEcard',
  `can_review` TINYINT(1) COMMENT '是否可评论 | ActivityBean.canReview',
  `direct_buy_official` TINYINT(1) COMMENT '官方直购 | ActivityBean.directBuyOfficial',
  `seat_buy_official` TINYINT(1) COMMENT '官方选座 | ActivityBean.seatBuyOfficial',
  `general_agency` TINYINT(1) COMMENT '总代 | ActivityBean.generalAgency',
  `proxy_buy` TINYINT(1) COMMENT '支持代抢 | ActivityBean.proxyBuy',
  `only_proxy_buy` TINYINT(1) COMMENT '仅代抢 | ActivityBean.onlyProxyBuy',
  `support_direct_buy` TINYINT(1) COMMENT '支持直购 | ActivityBean.supportDirectBuy',
  `support_proxy_buy` TINYINT(1) COMMENT '支持代抢 | ActivityBean.supportProxyBuy',
  `group_buy` TINYINT(1) COMMENT '支持拼团 | ActivityBean.groupBuy',
  `ultra_low_group` TINYINT(1) COMMENT '超低价拼团 | ActivityBean.ultraLowGroup',
  `is_best_discount` TINYINT(1) COMMENT '最优折扣 | ActivityBean.isBestDiscount',
  `need_speed_pack_buy` TINYINT(1) COMMENT '需加速包 | ActivityBean.needSpeedPackBuy',
  `no_ticket_compensate_switch` TINYINT(1) COMMENT '无票赔付开关 | ActivityBean.noTicketCompensateSwitch',
  `non_refundable_exchangeable` TINYINT(1) COMMENT '不可退可换 | ActivityBean.nonRefundableExchangeable',
  `discontinued_ticket_notice` TINYINT(1) COMMENT '停售通知 | ActivityBean.discontinuedTicketNotice',
  `pay_after_use_switch` TINYINT(1) COMMENT '先用后付开关 | ActivityBean.payAfterUseSwitch',
  `service_charge_open` TINYINT(1) COMMENT '服务费开关 | ActivityBean.serviceChargeOpen',
  `show_start_remind` TINYINT(1) COMMENT '开演提醒 | ActivityBean.showStartRemind',
  `speed_express` TINYINT(1) COMMENT '极速配送 | ActivityBean.speedExpress',
  `wuyou_transfer_supported` TINYINT(1) COMMENT '无忧转票 | ActivityBean.wuyouTransferSupported',
  `price_guarantee_type` INT COMMENT '保价类型 | ActivityBean.priceGuaranteeType',
  `postage_tip` VARCHAR(512) COMMENT '运费提示 | ActivityBean.postageTip',
  `pre_sale_tip` VARCHAR(512) COMMENT '预售提示 | ActivityBean.preSaleTip',
  `schema` VARCHAR(1024) COMMENT '跳转 scheme | ActivityBean.schema',
  `num_max` INT COMMENT '单笔最大购买数 | ActivityBean.numMax',
  `num_min` INT COMMENT '单笔最小购买数 | ActivityBean.numMin',
  `order_limit` INT COMMENT '限购 | ActivityBean.orderLimit',
  `tour_name` VARCHAR(256) COMMENT '巡演名 | ActivityBean.tourName',
  `venue` JSON COMMENT '场馆快照（原 Venue）| ActivityBean.venue',
  `actors` JSON COMMENT '艺人列表（原 List<ActorBean>）| ActivityBean.actors',
  `actor_info` JSON COMMENT '艺人二维码（原 ActorQrBean）| ActivityBean.actorInfo',
  `activity_details` JSON COMMENT '详情列表（原 List<ActivityDetail>）| ActivityBean.activityDetails',
  `activity_lineup` JSON COMMENT '阵容（原 List<ActivityLineupBean>）| ActivityBean.activityLineup',
  `activity_linked_groups` JSON COMMENT '关联群组（原 List<ActivityGroupBean>）| ActivityBean.activityLinkedGroups',
  `activity_purchase_info` JSON COMMENT '购买信息（内部类）| ActivityBean.activityPurchaseInfo',
  `activity_refund_policy` JSON COMMENT '退票政策（原 ActivityRefundPolicyBean）| ActivityBean.activityRefundPolicy',
  `campaigns` JSON COMMENT '活动列表（原 List<ActivityCampaign>）| ActivityBean.campaigns',
  `choose_ticket_calendar_data` JSON COMMENT '选票日历（原 ChooseTicketCalendarData）| ActivityBean.chooseTicketCalendarData',
  `display_tags` JSON COMMENT '展示标签（内部类 ActivityDisplayTag）| ActivityBean.displayTags',
  `enter_tip` JSON COMMENT '入场提示（原 ShowTip）| ActivityBean.enterTip',
  `events` JSON COMMENT '场次列表（原 List<ActivityEventBean>）| ActivityBean.events',
  `has_ticket_events` JSON COMMENT '有票场次 | ActivityBean.hasTicketEvents',
  `order_num_limit` JSON COMMENT '限购（内部类 OrderNumLimitBean）| ActivityBean.orderNumLimit',
  `pin_tuan_info` JSON COMMENT '拼团信息（原 PinTuanInfo）| ActivityBean.pinTuanInfo',
  `proxy_buy_info` JSON COMMENT '代抢信息（原 ProxyBuyInfo）| ActivityBean.proxyBuyInfo',
  `ticket_categories` JSON COMMENT '票档（原 List<TicketCategory>）| ActivityBean.ticketCategories',
  `ticket_tip` JSON COMMENT '购票提示（原 ShowTip）| ActivityBean.ticketTip',
  `tour` JSON COMMENT '巡演（内部类 TourBean）| ActivityBean.tour',
  `activities` JSON COMMENT '关联演出（原 List<ActivityBean>）| ActivityBean.activities',
  `add_time` BIGINT COMMENT '创建时间 | ActivityBean.addTime',
  `update_time` BIGINT COMMENT '更新时间 | ActivityBean.updateTime',
  PRIMARY KEY (`id`),
  KEY `idx_act_city` (`city_id`),
  KEY `idx_act_cat` (`category_id`),
  KEY `idx_act_venue` (`venue_id`),
  KEY `idx_act_status` (`status`,`publish_status`),
  KEY `idx_act_start` (`start`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='演出主档 | ActivityBean（100+ 顶层字段，含 4 个内部类；DTO 字段全量落列）';

CREATE TABLE IF NOT EXISTS `activity_events` (
  `id` BIGINT NOT NULL COMMENT '场次ID | ActivityEventBean.id',
  `activity_id` BIGINT COMMENT '演出ID | ActivityEventBean.activityId',
  `start` BIGINT COMMENT '开演时间 | ActivityEventBean.start',
  `end` BIGINT COMMENT '结束时间 | ActivityEventBean.end',
  `time_range` VARCHAR(256) COMMENT '时间文案 | Test.timeRange',
  `specification` VARCHAR(256) COMMENT '规格 | ActivityEventBean.specification',
  `desc` TEXT COMMENT '描述 | ActivityEventBean.desc',
  `low_price` DECIMAL(12,2) COMMENT '最低价 | ActivityEventBean.lowPrice',
  `discount_amount` DECIMAL(12,2) COMMENT '优惠金额 | ActivityEventBean.discountAmount',
  `discount_rate` DECIMAL(10,4) COMMENT '折扣率 | ActivityEventBean.discountRate',
  `discount_flag` TINYINT(1) COMMENT '是否有折扣 | ActivityEventBean.discountFlag',
  `price_lowest` TINYINT(1) COMMENT '是否最低价 | ActivityEventBean.priceLowest',
  `has_ticket` TINYINT(1) COMMENT '是否有票 | ActivityEventBean.hasTicket',
  `tickets_number` INT COMMENT '票数量 | ActivityEventBean.ticketsNumber',
  `tickets_flag` INT COMMENT '票标记 | ActivityEventBean.ticketsFlag',
  `seat_map_flag` TINYINT(1) COMMENT '是否选座图 | ActivityEventBean.seatMapFlag',
  `seat_tickets_size` INT COMMENT '座位票数 | ActivityEventBean.seatTicketsSize',
  `ticket_group_id` BIGINT COMMENT '票组ID | ActivityEventBean.ticketGroupId',
  `status` INT COMMENT '状态（2=停售，见 dict_enums.SaleStatus）| ActivityEventBean.status',
  `venue_id` BIGINT COMMENT '场馆ID（Test 补充）| Test.venueId',
  `venue_name` VARCHAR(256) COMMENT '场馆名（Test 补充）| Test.venueName',
  `available_numbers` JSON COMMENT '可选数量（原 List<Integer>）| ActivityEventBean.availableNumbers',
  `ticket_categories` JSON COMMENT '票档列表（原 List<? extends TicketCategory>）| ActivityEventBean.ticketCategories',
  `add_time` BIGINT COMMENT '创建时间 | ActivityEventBean.addTime',
  `update_time` BIGINT COMMENT '更新时间 | ActivityEventBean.updateTime',
  PRIMARY KEY (`id`),
  KEY `idx_event_act` (`activity_id`),
  KEY `idx_event_start` (`start`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='演出场次 | ActivityEventBean（v5/activities/getAllEvents · v1/activities/{id}/events）';

CREATE TABLE IF NOT EXISTS `activity_details` (
  `id` BIGINT NOT NULL COMMENT 'ID | ActivityDetail.id',
  `activity_id` BIGINT COMMENT '演出ID | ActivityDetail.activityId',
  `detail_desc` TEXT COMMENT '详情内容 | ActivityDetail.detailDesc',
  `type` INT COMMENT '详情类型 | ActivityDetail.type',
  `add_time` BIGINT COMMENT '创建时间 | ActivityDetail.addTime',
  `update_time` BIGINT COMMENT '更新时间 | ActivityDetail.updateTime',
  PRIMARY KEY (`id`),
  KEY `idx_adetail_act` (`activity_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='演出详情分段 | ActivityDetail（type 见 dict_enums.ActivityDetailType）';

CREATE TABLE IF NOT EXISTS `activity_lineups` (
  `id` BIGINT NOT NULL COMMENT 'ID | ActivityLineupBean.id',
  `activity_id` BIGINT COMMENT '演出ID | 推断',
  `name` VARCHAR(256) COMMENT '姓名 | ActivityLineupBean.name',
  `role` VARCHAR(128) COMMENT '角色 | ActivityLineupBean.role',
  `avatar` VARCHAR(1024) COMMENT '头像 | ActivityLineupBean.avatar',
  `actor_detail_schema` VARCHAR(1024) COMMENT '艺人详情 scheme | ActivityLineupBean.actorDetailSchema',
  `follow_num` INT COMMENT '关注数 | ActivityLineupBean.followNum',
  `followed` TINYINT(1) COMMENT '是否已关注 | ActivityLineupBean.followed',
  `fans` JSON COMMENT '粉丝列表（原 List<UserBean>）| ActivityLineupBean.fans',
  PRIMARY KEY (`id`),
  KEY `idx_lineup_act` (`activity_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='演出阵容 | ActivityLineupBean / ActorBean';

CREATE TABLE IF NOT EXISTS `activity_galleries` (
  `id` BIGINT NOT NULL COMMENT 'ID | ActivityGalleryBean.id',
  `activity_id` BIGINT COMMENT '演出ID | ActivityGalleryBean.activityId',
  `content` TEXT COMMENT '内容 | ActivityGalleryBean.content',
  `poster` VARCHAR(1024) COMMENT '图片 | ActivityGalleryBean.poster',
  `url` VARCHAR(1024) COMMENT '链接 | ActivityGalleryBean.url',
  `type` INT COMMENT '类型 | ActivityGalleryBean.type',
  `feed_type` VARCHAR(64) COMMENT '流类型 | ActivityGalleryBean.feedType',
  `subject_id` BIGINT COMMENT '主体ID | ActivityGalleryBean.subjectId',
  `subject_type` INT COMMENT '主体类型 | ActivityGalleryBean.subjectType',
  `review_id` BIGINT COMMENT '评论ID | ActivityGalleryBean.reviewId',
  `liked` TINYINT(1) COMMENT '已赞 | ActivityGalleryBean.liked',
  `likes` INT COMMENT '赞数 | ActivityGalleryBean.likes',
  `reply_count` INT COMMENT '回复数 | ActivityGalleryBean.replyCount',
  `status` INT COMMENT '状态 | ActivityGalleryBean.status',
  `user` JSON COMMENT '用户快照（原 UserBean）| ActivityGalleryBean.user',
  `add_time` BIGINT COMMENT '创建时间 | ActivityGalleryBean.addTime',
  `update_time` BIGINT COMMENT '更新时间 | ActivityGalleryBean.updateTime',
  PRIMARY KEY (`id`),
  KEY `idx_gallery_act` (`activity_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='演出图库 / 买家秀 | ActivityGalleryBean（v1/activityGallery/{activityId}）';

CREATE TABLE IF NOT EXISTS `activity_refund_policies` (
  `id` BIGINT NOT NULL COMMENT 'ID | ActivityRefundPolicyBean.id',
  `activity_id` BIGINT COMMENT '演出ID | ActivityRefundPolicyBean.activityId',
  `content` TEXT COMMENT '政策正文 | ActivityRefundPolicyBean.content',
  `tips` TEXT COMMENT '提示 | ActivityRefundPolicyBean.tips',
  `refund_rule_list` JSON COMMENT '规则列表（原 List<ActivityRefundRuleBean>）| ActivityRefundPolicyBean.refundRuleList',
  PRIMARY KEY (`id`),
  KEY `idx_refpolicy_act` (`activity_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='退票政策 | ActivityRefundPolicyBean / ShowTips';

CREATE TABLE IF NOT EXISTS `activity_refund_rules` (
  `id` BIGINT NOT NULL COMMENT '规则ID | ActivityRefundRuleBean.id',
  `activity_id` BIGINT COMMENT '演出ID | 推断',
  `relations` JSON COMMENT '关联对象（原 List<ActivityRefundRelationBean>）| ActivityRefundRuleBean.relations',
  `rule_details` JSON COMMENT '规则明细（原 List<ActivityRefundRuleDetailBean>）| ActivityRefundRuleBean.ruleDetails',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='退票规则 | ActivityRefundRuleBean';

CREATE TABLE IF NOT EXISTS `activity_refund_rule_details` (
  `id` BIGINT NOT NULL COMMENT 'ID | ActivityRefundRuleDetailBean.id',
  `rule_id` BIGINT COMMENT '规则ID | 推断',
  `start_time` BIGINT COMMENT '起始时间 | ActivityRefundRuleDetailBean.startTime',
  `end_time` BIGINT COMMENT '结束时间 | ActivityRefundRuleDetailBean.endTime',
  `refund_amount` VARCHAR(128) COMMENT '退款金额/比例 | ActivityRefundRuleDetailBean.refundAmount',
  `refund_mode` VARCHAR(64) COMMENT '退款模式 | ActivityRefundRuleDetailBean.refundMode',
  PRIMARY KEY (`id`),
  KEY `idx_refdetail_rule` (`rule_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='退票规则明细 | ActivityRefundRuleDetailBean（refundMode 见 dict_enums.RefundRuleMode）';

CREATE TABLE IF NOT EXISTS `activity_refund_relations` (
  `id` BIGINT NOT NULL COMMENT 'ID | ActivityRefundRelationBean.id',
  `biz_id` BIGINT COMMENT '业务ID | ActivityRefundRelationBean.bizId',
  `biz_type` VARCHAR(64) COMMENT '业务类型 | ActivityRefundRelationBean.bizType',
  `name` VARCHAR(256) COMMENT '名称 | ActivityRefundRelationBean.name',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='退票规则关联对象 | ActivityRefundRelationBean（bizType 见 RefundRelationBizType）';

CREATE TABLE IF NOT EXISTS `activity_campaigns` (
  `id` BIGINT NOT NULL COMMENT 'ID | ActivityCampaign.id',
  `activity_id` BIGINT COMMENT '演出ID | 推断',
  `name` VARCHAR(256) COMMENT '活动名 | ActivityCampaign.name',
  `desc` TEXT COMMENT '描述 | ActivityCampaign.desc',
  `campaign_rule` TEXT COMMENT '规则 | ActivityCampaign.campaignRule',
  `head_img` VARCHAR(1024) COMMENT '头图 | ActivityCampaign.headImg',
  `share_title` VARCHAR(256) COMMENT '分享标题 | ActivityCampaign.shareTitle',
  `share_desc` VARCHAR(1024) COMMENT '分享描述 | ActivityCampaign.shareDesc',
  `share_img` VARCHAR(1024) COMMENT '分享图 | ActivityCampaign.shareImg',
  `purchase_type` INT COMMENT '购买类型 | ActivityCampaign.purchaseType',
  `status` INT COMMENT '状态 | ActivityCampaign.status',
  `type` INT COMMENT '类型 | ActivityCampaign.type',
  `url` VARCHAR(1024) COMMENT '链接 | ActivityCampaign.url',
  `user_group_id` INT COMMENT '用户群ID | ActivityCampaign.userGroupId',
  `user_group_type` INT COMMENT '用户群类型 | ActivityCampaign.userGroupType',
  `add_time` BIGINT COMMENT '创建时间 | ActivityCampaign.addTime',
  `update_time` BIGINT COMMENT '更新时间 | ActivityCampaign.updateTime',
  PRIMARY KEY (`id`),
  KEY `idx_actcamp_act` (`activity_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='演出营销活动 | ActivityCampaign';

CREATE TABLE IF NOT EXISTS `activity_subjects` (
  `id` BIGINT NOT NULL COMMENT 'ID | ActivitySubjectBean.id',
  `title` VARCHAR(512) COMMENT '标题 | ActivitySubjectBean.title',
  `desc` TEXT COMMENT '描述 | ActivitySubjectBean.desc',
  `head_img` VARCHAR(1024) COMMENT '头图 | ActivitySubjectBean.headImg',
  `img` VARCHAR(1024) COMMENT '图片 | ActivitySubjectBean.img',
  `share_title` VARCHAR(256) COMMENT '分享标题 | ActivitySubjectBean.shareTitle',
  `share_desc` VARCHAR(1024) COMMENT '分享描述 | ActivitySubjectBean.shareDesc',
  `share_img` VARCHAR(1024) COMMENT '分享图 | ActivitySubjectBean.shareImg',
  `low_price` DECIMAL(12,2) COMMENT '最低价 | ActivitySubjectBean.lowPrice',
  `sold_out` TINYINT(1) COMMENT '售罄 | ActivitySubjectBean.soldOut',
  `sort` INT COMMENT '排序 | ActivitySubjectBean.sort',
  `type` INT COMMENT '类型 | ActivitySubjectBean.type',
  `activity` JSON COMMENT '演出快照（原 ActivityBean）| ActivitySubjectBean.activity',
  `venue` JSON COMMENT '场馆快照（原 Venue）| ActivitySubjectBean.venue',
  `reviews` JSON COMMENT '评论列表（原 List<Review>）| ActivitySubjectBean.reviews',
  `floors` JSON COMMENT '楼层（原 List<FloorsBean>）| ActivitySubjectBean.floors',
  `contents` JSON COMMENT '内容（原 List<ContentsBean>）| ActivitySubjectBean.contents',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='演出专题 | ActivitySubjectBean';

CREATE TABLE IF NOT EXISTS `actors` (
  `id` BIGINT NOT NULL COMMENT '艺人ID | ActorBean.id / StationBean.id',
  `user_id` BIGINT COMMENT '关联用户ID | ActorBean.userId',
  `name` VARCHAR(256) COMMENT '姓名 | ActorBean.name',
  `avatar` VARCHAR(1024) COMMENT '头像 | ActorBean.avatar',
  `background_image` VARCHAR(1024) COMMENT '背景图 | ActorBean.backgroundImage',
  `show_desc` TEXT COMMENT '简介 | ActorBean.showDesc',
  `actor_detail_schema` VARCHAR(1024) COMMENT '详情 scheme | ActorBean.actorDetailSchema',
  `ip_type` INT COMMENT 'IP 类型 | StationBean.ipType',
  `follow_num` INT COMMENT '关注数 | ActorBean.followNum',
  `followings` INT COMMENT '关注数（StationBean）| StationBean.followings',
  `followed` TINYINT(1) COMMENT '是否已关注 | StationBean.followed',
  `activity_count` INT COMMENT '演出数 | ActorBean.activityCount',
  `goods_count` INT COMMENT '商品数 | StationBean.goodsCount',
  `year` VARCHAR(16) COMMENT '年份 | StationBean.year',
  `status` VARCHAR(64) COMMENT '状态 | StationBean.status',
  `fans` JSON COMMENT '粉丝（原 List<UserBean>）| ActorBean.fans',
  `top_activities` JSON COMMENT '热门演出（原 List<ActivityBean>）| ActorBean.topActivities',
  `activities` JSON COMMENT '演出列表（原 List<Activity>）| StationBean.activities',
  `tours` JSON COMMENT '巡演（原 List<Tour>）| StationBean.tours',
  `posters` JSON COMMENT '海报（原 List<Poster>）| StationBean.posters',
  `head_posters` JSON COMMENT '头图（原 List<String>）| StationBean.headPosters',
  `related_activity_id_list` JSON COMMENT '关联演出（原 List<Long>）| StationBean.relatedActivityIdList',
  `activity_marketing_image_url` VARCHAR(1024) COMMENT '营销图 | StationBean.activityMarketingImageUrl',
  `activity_poster_icon` VARCHAR(1024) COMMENT '海报角标 | StationBean.activityPosterIcon',
  `add_time` BIGINT COMMENT '创建时间 | StationBean.addTime',
  `update_time` BIGINT COMMENT '更新时间 | StationBean.updateTime',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='艺人 / IP 站点 | ActorBean / StationBean / HotActorsBean / HomeActorsBean';

CREATE TABLE IF NOT EXISTS `actor_qr_codes` (
  `actor_id` VARCHAR(64) NOT NULL COMMENT '艺人ID | ActorQrBean.actorId',
  `pre_sales_qr` VARCHAR(1024) COMMENT '售前二维码 | ActorQrBean.preSalesQr',
  `after_sales_qr` VARCHAR(1024) COMMENT '售后二维码 | ActorQrBean.afterSalesQr',
  PRIMARY KEY (`actor_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='艺人客服二维码 | ActorQrBean';

CREATE TABLE IF NOT EXISTS `tours` (
  `tour_id` BIGINT NOT NULL COMMENT '巡演ID | HotTourBean.tourId',
  `actor_id` BIGINT COMMENT '艺人ID | HotTourBean.actorId',
  `tour_intellectual_property_id` BIGINT COMMENT 'IP ID | HotTourBean.tourIntellectualPropertyId',
  `tour_title` VARCHAR(512) COMMENT '巡演标题 | HotTourBean.tourTitle',
  `tour_poster` VARCHAR(1024) COMMENT '巡演海报 | HotTourBean.tourPoster',
  `activity_id` BIGINT COMMENT '演出ID | StationBean.activityId',
  `activity_count` INT COMMENT '演出数 | HotTourBean.activityCount',
  `activity_in_tour_tag_type` VARCHAR(64) COMMENT '标签类型 | StationBean.activityInTourTagType',
  `city_id` INT COMMENT '城市ID | StationBean.cityId',
  `city_name` VARCHAR(128) COMMENT '城市名 | StationBean.cityName',
  `start_time` BIGINT COMMENT '开始时间 | StationBean.startTime',
  `end_time` BIGINT COMMENT '结束时间 | StationBean.endTime',
  `time_range` VARCHAR(256) COMMENT '时间文案 | StationBean.timeRange',
  `low_price` DECIMAL(12,2) COMMENT '最低价 | StationBean.lowPrice',
  `group_buy` TINYINT(1) COMMENT '拼团 | StationBean.groupBuy',
  `marketing_tag` VARCHAR(256) COMMENT '营销标签 | StationBean.marketingTag',
  `marketing_tag_type` INT COMMENT '标签类型 | StationBean.marketingTagType',
  `show_time_pending` TINYINT(1) COMMENT '时间待定 | StationBean.showTimePending',
  `show_time_pending_text` VARCHAR(256) COMMENT '待定文案 | StationBean.showTimePendingText',
  `type` VARCHAR(64) COMMENT '类型 | StationBean.type',
  `url` VARCHAR(1024) COMMENT '链接 | StationBean.url',
  PRIMARY KEY (`tour_id`),
  KEY `idx_tour_actor` (`actor_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='巡演 / 站点演出 | TourBean / HotTourBean / StationBean 的 tour 部分';

CREATE TABLE IF NOT EXISTS `shops` (
  `shop_id` BIGINT NOT NULL COMMENT '店铺ID | Shop.id',
  `name` VARCHAR(256) COMMENT '店铺名 | Shop.name',
  `img` VARCHAR(1024) COMMENT '店铺图 | Shop.img',
  `mobile` VARCHAR(64) COMMENT '联系电话 | Shop.mobile',
  `owner` BIGINT COMMENT '店主 | Shop.owner',
  `city_id` INT COMMENT '城市ID | Shop.cityId',
  `auto_approve` INT COMMENT '自动审核 | Shop.autoApprove',
  `status` INT COMMENT '状态 | Shop.status',
  `type` INT COMMENT '类型 | Shop.type',
  `total_score` INT COMMENT '总分 | Shop.totalScore',
  `shop_name` VARCHAR(256) COMMENT '店铺名（ShopInfo 冗余）| ShopInfo.shopName',
  `shop_lisence_url` VARCHAR(1024) COMMENT '营业执照 | ShopInfo.shopLisenceUrl',
  `verified` TINYINT(1) COMMENT '是否认证 | ShopInfo.verified',
  `activity_event_num` INT COMMENT '场次数 | SearchResultShopBean.activityEventNum',
  `activity_num` INT COMMENT '演出数 | SearchResultShopBean.activityNum',
  `activity_type` INT COMMENT '演出类型 | SearchResultShopBean.activityType',
  `promo_img` VARCHAR(1024) COMMENT '推广图 | SearchResultShopBean.promoImg',
  `add_time` BIGINT COMMENT '创建时间 | Shop.addTime',
  `update_time` BIGINT COMMENT '更新时间 | Shop.updateTime',
  PRIMARY KEY (`shop_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='店铺 / 卖家 | Shop / ShopInfo / SearchResultShopBean';

CREATE TABLE IF NOT EXISTS `organizers` (
  `shop_id` BIGINT NOT NULL COMMENT '机构ID | OrganizerBean.shopId',
  `name` VARCHAR(256) COMMENT '名称 | OrganizerBean.name',
  `head_img` VARCHAR(1024) COMMENT '头图 | OrganizerBean.headImg',
  `logo` VARCHAR(1024) COMMENT 'Logo | OrganizerBean.logo',
  `slogan` VARCHAR(512) COMMENT '标语 | OrganizerBean.slogan',
  `summary` TEXT COMMENT '简介 | OrganizerBean.summary',
  `fans_num` INT COMMENT '粉丝数 | OrganizerBean.fansNum',
  `activity_num` INT COMMENT '演出数 | OrganizerBean.activityNum',
  `activity_event_num` INT COMMENT '场次数 | OrganizerBean.activityEventNum',
  `category_ids` JSON COMMENT '分类（原 List<Integer>）| OrganizerBean.categoryIds',
  `activities` JSON COMMENT '演出（原 List<ActivityBean>）| OrganizerBean.activities',
  PRIMARY KEY (`shop_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='主办方 / 机构 | OrganizerBean（v3/home/organizer）';


-- ------------------------------------------------------------------------
--  ↓ 以下为 04_ticket_inventory.sql
-- ------------------------------------------------------------------------

-- ========================================================================
--  sql_rebuildnew · 04_ticket_inventory.sql（忠实原样版，不做 R4）
--  SQL 正文与 docs/sql_new 逐语句一致，仅替换过时生成头部。
--  在 MySQL 8.4.10 上会产出 119 条 Warning 1681（TINYINT(1) 显示宽度弃用），0 ERROR。
-- ========================================================================

-- ========================================================================
--  04 · 票档 / 座位 / 库存
--  服务库: planstar_ticket
--  执行顺序: 01_init_databases.sql → 02 … 09 → 10 / 11 → 12_constraints.sql
--  本文件 6 张表: ticket_categories, ticket_groups, ticket_areas, ticket_seats, seat_map_layouts, discount_activities
-- ========================================================================
--  目录结构参考: github.com/arixbit/kratos-shop/tree/main/sql
--    NN_<域>.sql 按服务分库 + migrations/<库>/000001_init.up.sql / .down.sql
--  方言: MySQL 8.0 / InnoDB / utf8mb4（按用户选择保持，未切换 PostgreSQL）
--  第三方 SDK 表（小米推送 / 极光 JPush / 百度定位·mshield / Bugly /
--    字节 AppLog·bdtracker）共 31 张已按要求全部移除。
-- ========================================================================

USE `planstar_ticket`;

-- ----------------------------------------------------------------------
--  域 3 · 演出商品（show / goods）
-- ----------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS `ticket_categories` (
  `id` BIGINT NOT NULL COMMENT '票档ID | TicketCategory.id',
  `activity_event_id` BIGINT COMMENT '场次ID | TicketCategory.activityEventId',
  `activity_id` BIGINT COMMENT '演出ID | 推断',
  `specification` VARCHAR(256) COMMENT '规格/票档名 | TicketCategory.specification',
  `desc` TEXT COMMENT '描述 | TicketCategory.desc',
  `origin_price` DECIMAL(12,2) COMMENT '原价 | TicketCategory.originPrice',
  `low_price` DECIMAL(12,2) COMMENT '最低价 | TicketCategory.lowPrice',
  `discount_rate` DECIMAL(10,4) COMMENT '折扣率 | TicketCategory.discountRate',
  `discount_flag` TINYINT(1) COMMENT '有折扣 | TicketCategory.discountFlag',
  `price_lowest` TINYINT(1) COMMENT '最低价标记 | TicketCategory.priceLowest',
  `has_ticket` TINYINT(1) COMMENT '有票 | TicketCategory.hasTicket',
  `tickets_num` INT COMMENT '票数 | TicketCategory.ticketsNum',
  `unit_ticket_num` INT COMMENT '单位票数 | TicketCategory.unitTicketNum',
  `is_e_ticket` TINYINT(1) COMMENT '电子票 | TicketCategory.isETicket',
  `need_speed_pack_buy` TINYINT(1) COMMENT '需加速包 | TicketCategory.needSpeedPackBuy',
  `show_ticket_list` TINYINT(1) COMMENT '显示票列表 | TicketCategory.showTicketList',
  `icon` VARCHAR(1024) COMMENT '图标 | TicketCategory.icon',
  `seat_color` VARCHAR(64) COMMENT '座位颜色 | TicketCategory.seatColor',
  `status` INT COMMENT '状态 | TicketCategory.status',
  `areas` JSON COMMENT '区域（原 List<TicketAreaBean>）| TicketCategory.areas',
  `available_numbers` JSON COMMENT '可选数量（原 List<Integer>）| TicketCategory.availableNumbers',
  `proxy_buy_info` JSON COMMENT '代抢信息（原 ProxyBuyInfo）| TicketCategory.proxyBuyInfo',
  `add_time` BIGINT COMMENT '创建时间 | TicketCategory.addTime',
  `update_time` BIGINT COMMENT '更新时间 | TicketCategory.updateTime',
  PRIMARY KEY (`id`),
  KEY `idx_tc_event` (`activity_event_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='票档 | TicketCategory（v4/ticketCategories · v5/ticketCategories）';

CREATE TABLE IF NOT EXISTS `ticket_groups` (
  `id` BIGINT NOT NULL COMMENT '票组ID | TicketGroup.id',
  `ticket_group_id` BIGINT COMMENT '票组ID（冗余）| TicketGroup.ticketGroupId',
  `ticket_category_id` BIGINT COMMENT '票档ID | TicketGroup.ticketCategoryId',
  `ticket_area_id` BIGINT COMMENT '区域ID | TicketGroup.ticketAreaId',
  `ticket_row_id` BIGINT COMMENT '排ID | TicketGroup.ticketRowId',
  `activity_event_id` BIGINT COMMENT '场次ID | 推断',
  `saler` BIGINT COMMENT '卖家ID | TicketGroup.saler',
  `saler_name` VARCHAR(128) COMMENT '卖家名 | TicketGroup.salerName',
  `sale_price` DECIMAL(12,2) COMMENT '售价 | TicketGroup.salePrice',
  `origin_price` DECIMAL(12,2) COMMENT '原价 | TicketGroup.originPrice',
  `specification` VARCHAR(256) COMMENT '规格 | TicketGroup.specification',
  `ticket_category_specification` VARCHAR(256) COMMENT '票档规格 | TicketGroup.ticketCategorySpecification',
  `area_name` VARCHAR(256) COMMENT '区域名 | TicketGroup.areaName',
  `desc` TEXT COMMENT '描述 | TicketGroup.desc',
  `ticket_category_desc` TEXT COMMENT '票档描述 | TicketGroup.ticketCategoryDesc',
  `count` INT COMMENT '数量 | TicketGroup.count',
  `sold` INT COMMENT '已售 | TicketGroup.sold',
  `selling_amount` INT COMMENT '在售数量 | TicketGroup.sellingAmount',
  `saling` INT COMMENT '在售标记 | TicketGroup.saling',
  `is_continuous_seat` TINYINT(1) COMMENT '连座 | TicketGroup.isContinuousSeat',
  `is_e_ticket` TINYINT(1) COMMENT '电子票 | TicketGroup.isETicket',
  `is_recommended` TINYINT(1) COMMENT '推荐 | TicketGroup.isRecommended',
  `promise_deliver_day` INT COMMENT '承诺发货天数 | TicketGroup.promiseDeliverDay',
  `promise_deliver_notice` TEXT COMMENT '发货承诺 | TicketGroup.promiseDeliverNotice',
  `row_index` INT COMMENT '排索引 | TicketGroup.rowIndex',
  `seat_color` VARCHAR(64) COMMENT '座位颜色 | TicketGroup.seatColor',
  `split_order_fee` DECIMAL(12,2) COMMENT '拆单费 | TicketGroup.splitOrderFee',
  `succeed_num` INT COMMENT '已成团数 | TicketGroup.succeedNum',
  `pin_tuan_campaign_item_id` BIGINT COMMENT '拼团项ID | TicketGroup.pinTuanCampaignItemId',
  `unit_ticket_num` INT COMMENT '单位票数 | TicketGroup.unitTicketNum',
  `full_view_image` VARCHAR(1024) COMMENT '全场图 | TicketGroup.fullViewImage',
  `sale_tag` VARCHAR(256) COMMENT '销售标签 | TicketGroup.saleTag',
  `num_max` INT COMMENT '最大购买数 | TicketGroup.numMax',
  `num_min` INT COMMENT '最小购买数 | TicketGroup.numMin',
  `sell_limit` INT COMMENT '限售 | TicketGroup.sellLimit',
  `speed_express` TINYINT(1) COMMENT '极速配送 | TicketGroup.speedExpress',
  `is_is_divide` TINYINT(1) COMMENT '是否可拆 | TicketGroup.isIsDivide',
  `addition` JSON COMMENT '附加项（原 AdditionBean）| TicketGroup.addition',
  `available_numbers` JSON COMMENT '可选数量（原 List<Integer>）| TicketGroup.availableNumbers',
  PRIMARY KEY (`id`),
  KEY `idx_tg_cat` (`ticket_category_id`),
  KEY `idx_tg_saler` (`saler`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='票组 / 挂单 | TicketGroup（v1/ticketGroup · v4/seats）';

CREATE TABLE IF NOT EXISTS `ticket_areas` (
  `id` BIGINT NOT NULL COMMENT '区域ID | TicketAreaBean.id',
  `area_name` VARCHAR(256) COMMENT '区域名 | TicketAreaBean.areaName',
  `height` INT COMMENT '高 | TicketAreaBean.height',
  `width` INT COMMENT '宽 | TicketAreaBean.width',
  `is_select` TINYINT(1) COMMENT '可选 | TicketAreaBean.isSelect',
  `sold_out` TINYINT(1) COMMENT '售罄 | TicketAreaBean.soldOut',
  `color` INT COMMENT '颜色（Flutter SeatArea.color）| Flutter',
  `label` VARCHAR(128) COMMENT '标签（Flutter）| Flutter',
  `stage_label` VARCHAR(128) COMMENT '舞台标签（Flutter SeatMapData）| Flutter',
  `rows` JSON COMMENT '排列表（原 List<TicketSeatBean> / Flutter List<SeatRow>）| TicketAreaBean.ticketSeats',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='选座区域 | TicketAreaBean + Flutter SeatArea';

CREATE TABLE IF NOT EXISTS `ticket_seats` (
  `id` BIGINT NOT NULL COMMENT '座位ID | TicketSeatBean.id',
  `area_id` BIGINT COMMENT '区域ID | Flutter Seat.areaName 关联',
  `area_name` VARCHAR(256) COMMENT '区域名 | Flutter Seat.areaName',
  `seat_row` VARCHAR(64) COMMENT '排 | TicketSeatBean.row',
  `seat_col` VARCHAR(64) COMMENT '列 | TicketSeatBean.col',
  `x` INT COMMENT 'X 坐标 | TicketSeatBean.x',
  `y` INT COMMENT 'Y 坐标 | TicketSeatBean.y',
  `z` INT COMMENT 'Z 坐标 | TicketSeatBean.z',
  `status` INT COMMENT '状态（0 可选 / 1 已售 / 2 锁定 / 3 通道 / 4 不可选，见 SeatStatus）| TicketSeatBean.status',
  `desc` VARCHAR(512) COMMENT '描述 | TicketSeatBean.desc',
  `price` DECIMAL(12,2) COMMENT '价格（Flutter）| Flutter Seat.price',
  `category_id` BIGINT COMMENT '票档ID（Flutter）| Flutter Seat.categoryId',
  `ticket_group_id` BIGINT COMMENT '票组ID | 推断',
  `ticket` JSON COMMENT '票快照（原 TicketGroup）| TicketSeatBean.ticket',
  PRIMARY KEY (`id`),
  KEY `idx_seat_area` (`area_id`),
  KEY `idx_seat_group` (`ticket_group_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='座位 | TicketSeatBean + Flutter Seat';

CREATE TABLE IF NOT EXISTS `seat_map_layouts` (
  `id` BIGINT NOT NULL COMMENT '布局ID | TicketTableBean.id',
  `activity_event_id` BIGINT COMMENT '场次ID | 推断',
  `venue_name` VARCHAR(256) COMMENT '场馆名（Flutter）| Flutter SeatMapData.venueName',
  `stage_label` VARCHAR(128) COMMENT '舞台标签（Flutter）| Flutter',
  `type` INT COMMENT '类型（见 dict_enums.TicketType）| TicketTableBean.type',
  `status` INT COMMENT '状态 | TicketTableBean.status',
  `has_full_view_image` TINYINT(1) COMMENT '是否有全场图 | TicketTableBean.hasFullViewImage',
  `icon_checked` VARCHAR(1024) COMMENT '选中图标 | TicketTableBean.iconChecked',
  `icon_unable` VARCHAR(1024) COMMENT '不可选图标 | TicketTableBean.iconUnable',
  `icon_un_checked` VARCHAR(1024) COMMENT '未选图标 | TicketTableBean.iconUnChecked',
  `icon_ticket_category` VARCHAR(1024) COMMENT '票档图标 | TicketTableBean.iconTicketCategory',
  `icon_unable_template` JSON COMMENT '模板（原 IconTemplateBean）| TicketTableBean.iconUnableTemplate',
  `package_discount` JSON COMMENT '套餐折扣（原 Map<String,Map<String,Double>>）| TicketTableBean.packageDiscount',
  `ticket_categories_icon` JSON COMMENT '票档图标映射 | TicketTableBean.ticketCategoriesIcon',
  `area_tickets` JSON COMMENT '区域票（原 List<TicketAreaBean>）| TicketTableBean.areaTickets',
  `ticket_categories` JSON COMMENT '票档（原 List<TicketCategory>）| TicketTableBean.ticketCategories',
  `areas` JSON COMMENT '区域（Flutter List<SeatArea>）| Flutter',
  `add_time` BIGINT COMMENT '创建时间 | TicketTableBean.addTime',
  `update_time` BIGINT COMMENT '更新时间 | TicketTableBean.updateTime',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='选座图布局 | TicketTableBean + Flutter SeatMapData';

CREATE TABLE IF NOT EXISTS `discount_activities` (
  `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '代理主键',
  `activity_event_id` BIGINT COMMENT '场次ID | DiscountActivityBean.activityEventId',
  `location_type` INT COMMENT '位置类型 | DiscountActivityBean.locationType',
  `origin_price` DECIMAL(12,2) COMMENT '原价 | DiscountActivityBean.originPrice',
  `sale_price` DECIMAL(12,2) COMMENT '售价 | DiscountActivityBean.salePrice',
  `ticket_category_id` BIGINT COMMENT '票档ID | DiscountActivityBean.ticketCategoryId',
  `ticket_group_id` BIGINT COMMENT '票组ID | DiscountActivityBean.ticketGroupId',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='折扣活动 | DiscountActivityBean（首页 discounts 楼层）';


-- ------------------------------------------------------------------------
--  ↓ 以下为 05_order.sql
-- ------------------------------------------------------------------------

-- ========================================================================
--  sql_rebuildnew · 05_order.sql（忠实原样版，不做 R4）
--  SQL 正文与 docs/sql_new 逐语句一致，仅替换过时生成头部。
--  在 MySQL 8.4.10 上会产出 119 条 Warning 1681（TINYINT(1) 显示宽度弃用），0 ERROR。
-- ========================================================================

-- ========================================================================
--  05 · 订单
--  服务库: planstar_order
--  执行顺序: 01_init_databases.sql → 02 … 09 → 10 / 11 → 12_constraints.sql
--  本文件 10 张表: orders, order_events, order_items, order_goods, order_goods_packages, order_deliveries, order_confirm_snapshots, order_postage, order_comments, speed_packs
-- ========================================================================
--  目录结构参考: github.com/arixbit/kratos-shop/tree/main/sql
--    NN_<域>.sql 按服务分库 + migrations/<库>/000001_init.up.sql / .down.sql
--  方言: MySQL 8.0 / InnoDB / utf8mb4（按用户选择保持，未切换 PostgreSQL）
--  第三方 SDK 表（小米推送 / 极光 JPush / 百度定位·mshield / Bugly /
--    字节 AppLog·bdtracker）共 31 张已按要求全部移除。
-- ========================================================================

USE `planstar_order`;

-- ----------------------------------------------------------------------
--  域 4 · 交易与支付（order / payment / refund）
-- ----------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS `orders` (
  `order_id` BIGINT NOT NULL COMMENT '订单ID | unifiedOrder.orderId',
  `order_number` VARCHAR(64) COMMENT '订单号 | Flutter OrderNumber.value',
  `user_id` BIGINT COMMENT '用户ID | 推断',
  `activity_id` BIGINT COMMENT '演出ID | 推断',
  `activity_event_id` BIGINT COMMENT '场次ID | 推断',
  `shop_id` BIGINT COMMENT '店铺ID | 推断',
  `business_order_id` BIGINT COMMENT '业务订单ID | OrderItem.businessOrderId',
  `business_order_type` INT COMMENT '业务订单类型 | OrderItem.businessOrderType',
  `order_type` INT COMMENT '订单类型（1 普通票 / 2 拼团 / 5 抢票 / 60 Plus会员 / 70 加速包，见 dict_enums.OrderType）| MakeOrderRequestBody.orderType',
  `subject_type` INT COMMENT '支付主体类型（9 普通 / 6 会员充值 / 2 拼团 / 60 Plus，见 dict_enums.SubjectType）| gotoPayActivity 硬编码',
  `status` INT COMMENT '订单状态（见 dict_enums.OrderStatus）| OrderEvent.currentStatus',
  `order_status_desc` VARCHAR(256) COMMENT '状态文案 | OrderItem.orderStatusDesc',
  `order_type_desc` VARCHAR(256) COMMENT '类型文案 | OrderItem.orderTypeDesc',
  `order_type_desc_rgb` VARCHAR(64) COMMENT '类型文案色值 | OrderItem.orderTypeDescRGB',
  `title` VARCHAR(512) COMMENT '标题 | OrderItem.title',
  `poster` VARCHAR(1024) COMMENT '海报 | OrderItem.poster',
  `order_amount_desc` VARCHAR(256) COMMENT '金额文案 | OrderItem.orderAmountDesc',
  `descs` JSON COMMENT '描述行（原 List<String>）| OrderItem.descs',
  `spec` JSON COMMENT '规格（原 List<Spec>）| OrderItem.spec',
  `buttons` JSON COMMENT '操作按钮（原 List<OrderButtonBean>）| OrderItem.buttons',
  `pay_amount` DECIMAL(12,2) COMMENT '实付金额 | MakeOrderRequestBody.payAmount',
  `total_amount` DECIMAL(12,2) COMMENT '订单总额 | MakeOrderRequestBody.totalAmount',
  `good_total_amount` DECIMAL(12,2) COMMENT '商品总额 | OrderConfirmInfo.goodTotalAmount',
  `postage` DECIMAL(12,2) COMMENT '运费 | MakeOrderRequestBody.postage',
  `discount_amount` DECIMAL(12,2) COMMENT '优惠金额 | 推断',
  `split_order_fee` DECIMAL(12,2) COMMENT '拆单费 | BizParam.splitOrderContext.splitOrderFee',
  `consume_amount` DECIMAL(12,2) COMMENT '消费金额 | MakeOrderRequestBody.consumeAmount',
  `coupon_ids` JSON COMMENT '使用的券ID（与 campaignIds 互斥）| MakeOrderRequestBody.couponIds',
  `campaign_ids` JSON COMMENT '使用的活动ID（与 couponIds 互斥）| MakeOrderRequestBody.campaignIds',
  `gift_cards` JSON COMMENT '使用的礼品卡（原 List<GiftCard>）| MakeOrderRequestBody.giftCards',
  `goods` JSON COMMENT '商品（原 List<Goods>）| MakeOrderRequestBody.goods',
  `deliver` JSON COMMENT '配送（原 Deliver）| MakeOrderRequestBody.deliver',
  `additions` JSON COMMENT '附加项（原 List<? extends JSONObject>）| MakeOrderRequestBody.additions',
  `real_name_info_ids` JSON COMMENT '实名信息ID（原 List<Integer>）| MakeOrderRequestBody.realNameInfoIds',
  `biz_param` JSON COMMENT '业务参数（原 BizParam）| MakeOrderRequestBody.bizParam',
  `confirm_snapshot` JSON COMMENT '确认页快照（原 ConfirmSnapshot）| MakeOrderRequestBody.confirmSnapshot',
  `channel_attribute_info` TEXT COMMENT '渠道归因（自嵌套 JSON 字符串）| MakeOrderRequestBody.channelAttributeInfo',
  `id_card` VARCHAR(128) COMMENT '证件号 | MakeOrderRequestBody.idCard',
  `id_card_type` INT COMMENT '证件类型 | MakeOrderRequestBody.idCardType',
  `name` VARCHAR(128) COMMENT '收件人 | MakeOrderRequestBody.name',
  `phone` VARCHAR(64) COMMENT '联系电话 | MakeOrderRequestBody.phone',
  `user_address_id` BIGINT COMMENT '地址ID | MakeOrderRequestBody.userAddressId',
  `is_continuous_seat` TINYINT(1) COMMENT '是否连座 | MakeOrderRequestBody.isContinuousSeat',
  `price_guarantee_type` INT COMMENT '保价类型 | MakeOrderRequestBody.priceGuaranteeType',
  `pin_tuan_id` BIGINT COMMENT '拼团ID | MakeOrderRequestBody.pinTuanId',
  `balance_speed_pack` INT COMMENT '加速包余额部分 | MakeOrderRequestBody.balanceSpeedPack',
  `cash_speed_pack` INT COMMENT '加速包现金部分 | MakeOrderRequestBody.cashSpeedPack',
  `channel` VARCHAR(128) COMMENT '渠道 | BizParam.channel',
  `pay_after_use` TINYINT(1) COMMENT '先用后付 | PayAfterUseInfo.check',
  `credit_pay` TINYINT(1) COMMENT '信用付 | CreditPay',
  `success_page` VARCHAR(1024) COMMENT '支付成功跳转 | PayInfo.successPage',
  `fail_page` VARCHAR(1024) COMMENT '支付失败跳转 | PayInfo.failPage',
  `refund_amount` DECIMAL(12,2) COMMENT '已退金额 | 推断',
  `add_time` BIGINT COMMENT '下单时间 | 推断',
  `pay_time` BIGINT COMMENT '支付时间 | 推断',
  `update_time` BIGINT COMMENT '更新时间 | 推断',
  PRIMARY KEY (`order_id`),
  UNIQUE KEY `uk_order_number` (`order_number`),
  KEY `idx_order_user` (`user_id`,`status`),
  KEY `idx_order_act` (`activity_id`),
  KEY `idx_order_addtime` (`add_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='统一订单 | unifiedOrder · OrderItem · MakeOrderRequestBody · OrderEvent';

CREATE TABLE IF NOT EXISTS `order_events` (
  `id` BIGINT NOT NULL COMMENT '事件ID | OrderEvent.id',
  `order_id` BIGINT COMMENT '订单ID | OrderEvent.orderId',
  `origin_status` INT COMMENT '原状态 | OrderEvent.originStatus',
  `current_status` INT COMMENT '新状态 | OrderEvent.currentStatus',
  `operator_id` BIGINT COMMENT '操作人 | OrderEvent.operatorId',
  `operator_type` INT COMMENT '操作人类型 | OrderEvent.operatorType',
  `supplement` TEXT COMMENT '补充说明 | OrderEvent.supplement',
  `add_time` BIGINT COMMENT '发生时间 | OrderEvent.addTime',
  `update_time` BIGINT COMMENT '更新时间 | OrderEvent.updateTime',
  PRIMARY KEY (`id`),
  KEY `idx_orderev_order` (`order_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='订单状态流转事件 | OrderEvent（v1/unifiedOrder/{orderId}/status）';

CREATE TABLE IF NOT EXISTS `order_items` (
  `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '代理主键',
  `order_id` BIGINT COMMENT '订单ID | OrderItem.orderId',
  `business_order_id` BIGINT COMMENT '业务订单ID | OrderItem.businessOrderId',
  `business_order_type` INT COMMENT '业务订单类型 | OrderItem.businessOrderType',
  `title` VARCHAR(512) COMMENT '标题 | OrderItem.title',
  `poster` VARCHAR(1024) COMMENT '海报 | OrderItem.poster',
  `order_status_desc` VARCHAR(256) COMMENT '状态文案 | OrderItem.orderStatusDesc',
  `order_type_desc` VARCHAR(256) COMMENT '类型文案 | OrderItem.orderTypeDesc',
  `order_type_desc_rgb` VARCHAR(64) COMMENT '色值 | OrderItem.orderTypeDescRGB',
  `order_amount_desc` VARCHAR(256) COMMENT '金额文案 | OrderItem.orderAmountDesc',
  `descs` JSON COMMENT '描述行 | OrderItem.descs',
  `spec` JSON COMMENT '规格（原 List<Spec>）| OrderItem.spec',
  `buttons` JSON COMMENT '按钮（原 List<OrderButtonBean>）| OrderItem.buttons',
  PRIMARY KEY (`id`),
  KEY `idx_orderitem_order` (`order_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='订单列表条目 | OrderItem（v1/unifiedOrder/myOrders）';

CREATE TABLE IF NOT EXISTS `order_goods` (
  `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '代理主键',
  `order_id` BIGINT COMMENT '订单ID | 推断',
  `good_id` BIGINT COMMENT '商品/票组ID | Goods.goodId',
  `count` INT COMMENT '数量 | Goods.count',
  PRIMARY KEY (`id`),
  KEY `idx_ogoods_order` (`order_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='订单商品行 | Goods / GoodsBean';

CREATE TABLE IF NOT EXISTS `order_goods_packages` (
  `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '代理主键',
  `order_id` BIGINT COMMENT '订单ID | 推断',
  `activity_id` BIGINT COMMENT '演出ID | GoodsPackage.activityId',
  `activity_name` VARCHAR(512) COMMENT '演出名 | GoodsPackage.activityName',
  `activity_event_name` VARCHAR(512) COMMENT '场次名 | GoodsPackage.activityEventName',
  `venue_name` VARCHAR(256) COMMENT '场馆名 | GoodsPackage.venueName',
  `poster` VARCHAR(1024) COMMENT '海报 | GoodsPackage.poster',
  `detail_type` INT COMMENT '明细类型 | GoodsPackage.detailType',
  `non_refundable_exchangeable` TINYINT(1) COMMENT '不可退可换 | GoodsPackage.nonRefundableExchangeable',
  `shop_info` JSON COMMENT '店铺（原 ShopInfo）| GoodsPackage.shopInfo',
  `speed_express` TINYINT(1) COMMENT '极速配送 | GoodsPackage.speedExpress',
  `deliver_types` JSON COMMENT '配送方式（原 List<Integer>）| GoodsPackage.deliverTypes',
  `goods` JSON COMMENT '商品（原 List<Goods>）| GoodsPackage.goods',
  `details` JSON COMMENT '明细（原 List<Detail>）| GoodsPackage.details',
  `deliver_notice` TEXT COMMENT '配送须知 | GoodsPackage.deliverNotice',
  `shop_notice` TEXT COMMENT '卖家须知 | GoodsPackage.shopNotice',
  `count` INT COMMENT '数量 | GoodsPackage.count',
  `has_seat` TINYINT(1) COMMENT '有座 | GoodsPackage.hasSeat',
  `sale_price` DECIMAL(12,2) COMMENT '售价 | GoodsPackage.salePrice',
  `ticket_category_specification` VARCHAR(256) COMMENT '票档规格 | GoodsPackage.ticketCategorySpecification',
  `seat_name` VARCHAR(256) COMMENT '座位名 | GoodsPackage.seatName',
  `area_show` TINYINT(1) COMMENT '显示区域 | GoodsPackage.areaShow',
  `continue_seat` TINYINT(1) COMMENT '连座 | GoodsPackage.continueSeat',
  `tips` JSON COMMENT '提示集合（原 Tips）| GoodsPackage.tips',
  `no_ticket_compensate_tip` JSON COMMENT '无票赔付提示 | GoodsPackage.noTicketCompensateTip',
  `no_ticket_onsite_compensate_tip` JSON COMMENT '现场无票赔付提示 | GoodsPackage.noTicketOnsiteCompensateTip',
  `refund_tip` JSON COMMENT '退款提示 | GoodsPackage.refundTip',
  `snap_up_fail_tip` JSON COMMENT '抢票失败提示 | GoodsPackage.snapUpFailTip',
  `snap_up_un_finish_tip` JSON COMMENT '抢票未完成提示 | GoodsPackage.snapUpUnFinishTip',
  PRIMARY KEY (`id`),
  KEY `idx_ogpkg_order` (`order_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='商品包 | GoodsPackage（确认页 goodPackages）';

CREATE TABLE IF NOT EXISTS `order_deliveries` (
  `id` BIGINT NOT NULL COMMENT '配送ID | Deliver.id',
  `order_id` BIGINT COMMENT '订单ID | 推断',
  `type` INT COMMENT '配送类型（见 dict_enums.DeliverType）| Deliver.type',
  `tip` VARCHAR(1024) COMMENT '提示 | Deliver.tip',
  `desc` TEXT COMMENT '描述 | Deliver.desc',
  `amount` DECIMAL(12,2) COMMENT '运费金额 | Deliver.amount / Postage.amount',
  `postage` JSON COMMENT '运费（原 Postage）| Deliver.postage',
  `postage_origin` DECIMAL(12,2) COMMENT '原运费 | PostageBean.postageOrigin',
  `postage_desc` VARCHAR(512) COMMENT '运费说明 | PostageBean.postageDesc',
  `address` JSON COMMENT '地址快照（原 Address）| Deliver.address',
  `contact` JSON COMMENT '联系人（原 Contact）| Deliver.contact',
  `additions` JSON COMMENT '附加项（原 List<Addition>）| Deliver.additions',
  `name` VARCHAR(128) COMMENT '收件人 | Deliver.name',
  `phone` VARCHAR(64) COMMENT '电话 | Deliver.phone',
  `district` VARCHAR(128) COMMENT '区县 | Deliver.district',
  `value` VARCHAR(512) COMMENT '值 | Deliver.value',
  `contact_phone` VARCHAR(64) COMMENT '联系电话 | DeliverBean.contactPhone',
  `deliver_tip` TEXT COMMENT '配送提示 | DeliverBean.deliverTip',
  `need_id_card` VARCHAR(64) COMMENT '是否需证件 | DeliverBean.needIdCard',
  `self_get_address` VARCHAR(1024) COMMENT '自取地址 | DeliverBean.selfGetAddress',
  `self_get_time` VARCHAR(256) COMMENT '自取时间 | DeliverBean.selfGetTime',
  `multi_deliver_group` TINYINT(1) COMMENT '多配送组 | OrderConfirmInfo.multiDeliverGroup',
  `check` TINYINT(1) COMMENT '是否勾选 | OrderConfirmInfo.check',
  `remark` TEXT COMMENT '备注 | OrderConfirmInfo.remark',
  PRIMARY KEY (`id`),
  KEY `idx_odel_order` (`order_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='配送 / 自取 / 电子票 | Deliver / DeliverBean / DeliverInfo / Postage / Contact / Addition';

CREATE TABLE IF NOT EXISTS `order_confirm_snapshots` (
  `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '代理主键',
  `order_id` BIGINT COMMENT '订单ID | 推断',
  `user_id` BIGINT COMMENT '用户ID | 推断',
  `ad_tips` VARCHAR(1024) COMMENT '广告提示 | OrderConfirmInfo.adTips',
  `good_total_amount` DECIMAL(12,2) COMMENT '商品总额 | OrderConfirmInfo.goodTotalAmount',
  `group_buy_rule_desc_for_c` TEXT COMMENT '拼团规则 | OrderConfirmInfo.groupBuyRuleDescForC',
  `price_guarantee_type` INT COMMENT '保价类型 | OrderConfirmInfo.priceGuaranteeType',
  `need_plus_member` TINYINT(1) COMMENT '需 Plus 会员 | OrderConfirmInfo.needPlusMember',
  `can_buy` TINYINT(1) COMMENT '可购买 | OrderConfirmInfo.canBuy',
  `reason` VARCHAR(1024) COMMENT '不可购买原因 | OrderConfirmInfo.reason',
  `count` INT COMMENT '数量 | OrderConfirmInfo.count',
  `need_id_card` TINYINT(1) COMMENT '需实名 | OrderConfirmInfo.needIdCard',
  `need_id_card_v2` TINYINT(1) COMMENT '需实名 V2 | OrderConfirmInfoBean.needIdCardV2',
  `id_card_count` INT COMMENT '观演人数量 | OrderConfirmInfoBean.idCardCount',
  `can_use` TINYINT(1) COMMENT '券/卡可用 | OrderConfirmInfo.canUse',
  `error` VARCHAR(1024) COMMENT '错误 | OrderConfirmInfo.error',
  `content` TEXT COMMENT '内容 | OrderConfirmInfo.content',
  `schema` VARCHAR(1024) COMMENT '跳转 | OrderConfirmInfo.schema',
  `desc` TEXT COMMENT '描述 | OrderConfirmInfo.desc',
  `title` VARCHAR(512) COMMENT '标题 | OrderConfirmInfo.title',
  `remark` TEXT COMMENT '备注 | OrderConfirmInfo.remark',
  `notice` TEXT COMMENT '须知 | OrderConfirmInfoBean.notice',
  `notice_schema` VARCHAR(1024) COMMENT '须知跳转 | OrderConfirmInfoBean.noticeSchema',
  `refund_tip` TEXT COMMENT '退款提示 | OrderConfirmInfoBean.refundTip',
  `service_charge_title` VARCHAR(256) COMMENT '服务费标题 | OrderConfirmInfoBean.serviceChargeTitle',
  `service_charge_desc` TEXT COMMENT '服务费说明 | OrderConfirmInfoBean.serviceChargeDesc',
  `service_charge_open` TINYINT(1) COMMENT '服务费开关 | OrderConfirmInfoBean.serviceChargeOpen',
  `promise_deliver_notice` TEXT COMMENT '发货承诺 | OrderConfirmInfoBean.promiseDeliverNotice',
  `no_ticket_compensate_switch` TINYINT(1) COMMENT '无票赔付开关 | OrderConfirmInfoBean.noTicketCompensateSwitch',
  `only_plus_member_can_buy` TINYINT(1) COMMENT '仅会员可买 | OrderConfirmInfoBean.onlyPlusMemberCanBuy',
  `can_refund` TINYINT(1) COMMENT '可退 | OrderConfirmInfoBean.canRefund',
  `area_name` VARCHAR(256) COMMENT '区域名 | OrderConfirmInfoBean.areaName',
  `biz_param` JSON COMMENT '业务参数 | OrderConfirmInfo.bizParam',
  `campaign` JSON COMMENT '活动 | OrderConfirmInfo.campaign',
  `can_buy_info` JSON COMMENT '限购信息 | OrderConfirmInfo.canBuyInfo',
  `credit_pay` JSON COMMENT '信用付 | OrderConfirmInfo.creditPay',
  `pay_after_use_info` JSON COMMENT '先用后付 | OrderConfirmInfo.payAfterUseInfo',
  `plus_member` JSON COMMENT 'Plus 会员 | OrderConfirmInfo.plusMember',
  `id_card` JSON COMMENT '实名 | OrderConfirmInfo.idCard',
  `id_card_types` JSON COMMENT '证件类型（原 ArrayList<IDCardType>）| OrderConfirmInfo.idCardTypes',
  `coupon` JSON COMMENT '券 | OrderConfirmInfo.coupon',
  `gift_card` JSON COMMENT '礼品卡 | OrderConfirmInfo.giftCard',
  `addition_components` JSON COMMENT '附加组件（原 List<OrderConfirmExtraInfoBean>）| OrderConfirmInfo.additionComponents',
  `good_packages` JSON COMMENT '商品包（原 List<GoodsPackage>）| OrderConfirmInfo.goodPackages',
  `tip` JSON COMMENT '提示（原 Tip）| OrderConfirmInfo.tip',
  `deliver` JSON COMMENT '配送（原 DeliverInfo）| OrderConfirmInfo.deliver',
  `details` JSON COMMENT '配送明细（原 List<Deliver>）| OrderConfirmInfo.details',
  `campaigns` JSON COMMENT '活动列表（原 List<Campaign>）| OrderConfirmInfo.campaigns',
  `enable_coupons` JSON COMMENT '可用券（原 List<Coupon>）| OrderConfirmInfo.enableCoupons',
  `disable_coupons` JSON COMMENT '不可用券 | OrderConfirmInfo.disableCoupons',
  `notice_obj` JSON COMMENT '通知对象 | OrderConfirmInfo.notice',
  `trade` JSON COMMENT '交易说明 | OrderConfirmInfo.trade',
  `discount` JSON COMMENT '折扣 | OrderConfirmInfo.discount',
  `activity` JSON COMMENT '演出快照 | OrderConfirmInfoBean.activity',
  `activity_event` JSON COMMENT '场次快照 | OrderConfirmInfoBean.activityEvent',
  `address` JSON COMMENT '地址快照 | OrderConfirmInfoBean.address',
  `postage` JSON COMMENT '运费 | OrderConfirmInfoBean.postage',
  `speed_packs` JSON COMMENT '加速包（原 List<JSONObject>）| OrderConfirmInfoBean.speedPacks',
  `speed_packs_balance` JSON COMMENT '加速包余额 | OrderConfirmInfoBean.speedPacksBalance',
  `order_group_products` JSON COMMENT '订单组商品 | OrderConfirmInfoBean.orderGroupProducts',
  `ticket_category` JSON COMMENT '票档 | OrderConfirmInfoBean.ticketCategory',
  `venue` JSON COMMENT '场馆 | OrderConfirmInfoBean.venue',
  `goods` JSON COMMENT '商品（原 List<Goods>）| OrderConfirmInfoRequestBody.goods',
  `order_type` INT COMMENT '订单类型 | OrderConfirmInfoRequestBody.orderType',
  PRIMARY KEY (`id`),
  KEY `idx_ocs_order` (`order_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='确认订单页快照 | OrderConfirmInfo / OrderConfirmInfoBean / OrderConfirmInfoRequestBody';

CREATE TABLE IF NOT EXISTS `order_postage` (
  `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '代理主键',
  `order_id` BIGINT COMMENT '订单ID | 推断',
  `postage` DECIMAL(12,2) COMMENT '运费 | PostageBean.postage',
  `postage_desc` VARCHAR(512) COMMENT '运费说明 | PostageBean.postageDesc',
  `postage_origin` DECIMAL(12,2) COMMENT '原运费 | PostageBean.postageOrigin',
  `deliver_type` INT COMMENT '配送类型 | 推断',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='运费试算 | PostageBean（v1/order/postage）';

CREATE TABLE IF NOT EXISTS `order_comments` (
  `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '代理主键',
  `order_id` BIGINT COMMENT '订单ID | 推断',
  `comment` JSON COMMENT '评价内容 | OrderCommentBean.comment',
  `reviews` JSON COMMENT '评价列表（原 List<Review>）| OrderCommentBean.reviews',
  `schema` JSON COMMENT '评价项（原 List<CommentSchemeBean>）| OrderCommentBean.schema',
  `rank` INT COMMENT '星级 | Review.rank',
  `content` TEXT COMMENT '内容 | Review.content',
  `add_time` BIGINT COMMENT '创建时间 | 推断',
  PRIMARY KEY (`id`),
  KEY `idx_ocomment_order` (`order_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='订单评价 | OrderCommentBean / unifiedOrderComment';

CREATE TABLE IF NOT EXISTS `speed_packs` (
  `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '代理主键',
  `user_id` BIGINT COMMENT '用户ID | 推断',
  `order_id` BIGINT COMMENT '订单ID | 推断',
  `amount` INT COMMENT '加速包金额 | SpeedPackChoice.amount',
  `desc` VARCHAR(512) COMMENT '描述 | SpeedPackChoice.desc',
  `recommend` TINYINT(1) COMMENT '是否推荐 | SpeedPackChoice.recommend',
  `success_rate` DECIMAL(10,4) COMMENT '成功率 | SpeedPackChoice.successRate',
  `speed_pack_balance` INT COMMENT '加速包余额 | BizParam.speedPackBalance',
  `balance_speed_pack` INT COMMENT '余额抵扣部分 | MakeOrderRequestBody.balanceSpeedPack',
  `cash_speed_pack` INT COMMENT '现金部分 | MakeOrderRequestBody.cashSpeedPack',
  `speed_pack_choices` JSON COMMENT '选项列表（原 List<SpeedPackChoice>）| BizParam.speedPackChoices',
  `split_order_desc` VARCHAR(1024) COMMENT '拆单说明 | SplitOrderContext.splitOrderDesc',
  `split_order_fee` DECIMAL(12,2) COMMENT '拆单费 | SplitOrderContext.splitOrderFee',
  `channel` VARCHAR(128) COMMENT '渠道 | BizParam.channel',
  `expire_time` BIGINT COMMENT '过期时间 | BizParam.expireTime',
  `pin_tuan_id` BIGINT COMMENT '拼团ID | BizParam.pinTuanId',
  PRIMARY KEY (`id`),
  KEY `idx_speed_user` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='加速包 / 拆单参数 | SpeedPackChoice / BizParam / SplitOrderContext';


-- ------------------------------------------------------------------------
--  ↓ 以下为 06_payment.sql
-- ------------------------------------------------------------------------

-- ========================================================================
--  sql_rebuildnew · 06_payment.sql（忠实原样版，不做 R4）
--  SQL 正文与 docs/sql_new 逐语句一致，仅替换过时生成头部。
--  在 MySQL 8.4.10 上会产出 119 条 Warning 1681（TINYINT(1) 显示宽度弃用），0 ERROR。
-- ========================================================================

-- ========================================================================
--  06 · 支付 / 退款 / 钱包 / 礼品卡
--  服务库: planstar_payment
--  执行顺序: 01_init_databases.sql → 02 … 09 → 10 / 11 → 12_constraints.sql
--  本文件 7 张表: payments, payment_channels, refunds, wallet_accounts, wallet_sequences, gift_cards, gift_card_usages
-- ========================================================================
--  目录结构参考: github.com/arixbit/kratos-shop/tree/main/sql
--    NN_<域>.sql 按服务分库 + migrations/<库>/000001_init.up.sql / .down.sql
--  方言: MySQL 8.0 / InnoDB / utf8mb4（按用户选择保持，未切换 PostgreSQL）
--  第三方 SDK 表（小米推送 / 极光 JPush / 百度定位·mshield / Bugly /
--    字节 AppLog·bdtracker）共 31 张已按要求全部移除。
-- ========================================================================

USE `planstar_payment`;

-- ----------------------------------------------------------------------
--  域 4 · 交易与支付（order / payment / refund）
-- ----------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS `payments` (
  `pay_id` BIGINT NOT NULL COMMENT '支付单ID | 推断',
  `order_id` BIGINT COMMENT '订单ID | PayInfo.orderId',
  `left_time` INT COMMENT '剩余支付秒数（倒计时基线）| PayInfo.leftTime',
  `pay_amount` DECIMAL(12,2) COMMENT '应付金额 | PayInfo.payAmount',
  `balance` DECIMAL(12,2) COMMENT '可用余额 | PayInfo.balance',
  `title` VARCHAR(512) COMMENT '标题 | PayInfo.title',
  `value` VARCHAR(512) COMMENT '值 | PayInfo.value',
  `content` TEXT COMMENT '内容 | PayInfo.content',
  `schema` VARCHAR(1024) COMMENT '跳转 | PayInfo.schema',
  `friendly_tips` TEXT COMMENT '友好提示 | PayInfo.friendlyTips',
  `continuous_seat` TINYINT(1) COMMENT '连座 | PayInfo.continuousSeat',
  `count` INT COMMENT '数量 | PayInfo.count',
  `product_id` VARCHAR(128) COMMENT '商品ID | PayInfo.productId',
  `product_name` VARCHAR(512) COMMENT '商品名 | PayInfo.productName',
  `type` INT COMMENT '类型 | PayInfo.type',
  `client_type` INT COMMENT '客户端类型 | PayInfo.clientType',
  `fail_page` VARCHAR(1024) COMMENT '失败跳转 | PayInfo.failPage',
  `success_page` VARCHAR(1024) COMMENT '成功跳转 | PayInfo.successPage',
  `desc` TEXT COMMENT '描述 | PayInfo.desc',
  `detail_info` JSON COMMENT '费用明细（原 List<DetailInfo>）| PayInfo.detailInfo',
  `pay_unified_order_info` JSON COMMENT '统一订单信息 | PayInfo.payUnifiedOrderInfo',
  `tip` JSON COMMENT '提示（原 Tip）| PayInfo.tip',
  `pay_return_page` JSON COMMENT '返回页（原 PayReturnPage）| PayInfo.payReturnPage',
  `channels` JSON COMMENT '渠道列表（原 List<Channel>）| PayInfo.channels',
  `credit_pay` JSON COMMENT '信用付 | PayInfo.creditPay',
  `huabei_installment` JSON COMMENT '花呗分期（原 List<CommonInstallment>）| PayInfo.huabeiInstallment',
  `leka_installment` JSON COMMENT '乐卡分期 | PayInfo.lekaInstallment',
  `pay_after_use_info` JSON COMMENT '先用后付 | PayInfo.payAfterUseInfo',
  `payment_tips` JSON COMMENT '支付提示（原 Map<String,String>）| OrderPaymentInfo.paymentTips',
  `payment_types` JSON COMMENT '支付方式（原 List<Integer>）| OrderPaymentInfo.paymentTypes',
  `add_time` BIGINT COMMENT '创建时间 | 推断',
  PRIMARY KEY (`pay_id`),
  KEY `idx_pay_order` (`order_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='支付单 / 支付明细 | PayInfo（GET v1/unifiedPay/payDetail）· OrderPaymentInfo';

CREATE TABLE IF NOT EXISTS `payment_channels` (
  `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '代理主键',
  `pay_id` BIGINT COMMENT '支付单ID | 推断',
  `type` INT COMMENT '渠道码 | Channel.type',
  `title` VARCHAR(256) COMMENT '标题 | Channel.title',
  `value` VARCHAR(256) COMMENT '值 | Channel.value',
  `tip` VARCHAR(1024) COMMENT '提示 | Channel.tip',
  `desc` TEXT COMMENT '描述 | Channel.desc',
  `available` TINYINT(1) COMMENT '是否可用 | Flutter PayChannelOption.available',
  `client_type` INT COMMENT '客户端类型 | 推断',
  `installment_num` INT COMMENT '分期数 | CommonInstallment.installmentNum',
  `installment_rate` DECIMAL(10,4) COMMENT '分期费率 | CommonInstallment.installmentRate',
  `is_select` TINYINT(1) COMMENT '是否选中 | CommonInstallment.isSelect',
  PRIMARY KEY (`id`),
  KEY `idx_paych_pay` (`pay_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='支付渠道 | Channel / PayChannelOption / CommonInstallment';

CREATE TABLE IF NOT EXISTS `refunds` (
  `refund_id` BIGINT NOT NULL COMMENT '退款单ID | 推断',
  `order_id` BIGINT COMMENT '订单ID | PUT v1/order/{orderId}/refund',
  `user_id` BIGINT COMMENT '用户ID | 推断',
  `refund_amount` DECIMAL(12,2) COMMENT '退款金额 | Flutter RefundState.refundAmount',
  `reason_code` VARCHAR(64) COMMENT '退款原因码（notNeeded / boughtWrong / tooExpensive / foundBetter / other）| Flutter RefundReason',
  `custom_reason` TEXT COMMENT '自定义原因 | Flutter RefundState.customReason',
  `status` VARCHAR(64) COMMENT '退款状态 | Flutter RefundStatus',
  `error_message` VARCHAR(1024) COMMENT '错误信息 | Flutter RefundState.errorMessage',
  `refund_mode` VARCHAR(64) COMMENT '退款手续费模式（见 dict_enums.RefundRuleMode）| ActivityRefundRuleDetailBean.refundMode',
  `apply_time` BIGINT COMMENT '申请时间 | 推断',
  `update_time` BIGINT COMMENT '更新时间 | 推断',
  PRIMARY KEY (`refund_id`),
  KEY `idx_refund_order` (`order_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='退款申请 | v1/order/{orderId}/refund + Flutter RefundState / RefundReason';

-- ----------------------------------------------------------------------
--  域 5 · 营销资产（marketing）
-- ----------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS `wallet_accounts` (
  `account_id` BIGINT NOT NULL COMMENT '账户ID | 推断',
  `user_id` BIGINT COMMENT '用户ID | 推断',
  `balance` DECIMAL(12,2) COMMENT '余额 | v1/moneyaccount / Flutter WalletAccount.balance',
  `status` INT COMMENT '状态 | 推断',
  `add_time` BIGINT COMMENT '创建时间 | 推断',
  `update_time` BIGINT COMMENT '更新时间 | 推断',
  PRIMARY KEY (`account_id`),
  UNIQUE KEY `uk_wallet_user` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='钱包账户 | v1/moneyaccount + Flutter WalletAccount';

CREATE TABLE IF NOT EXISTS `wallet_sequences` (
  `id` BIGINT NOT NULL COMMENT '流水ID | Flutter WalletTransaction.id',
  `user_id` BIGINT COMMENT '用户ID | 推断',
  `account_id` BIGINT COMMENT '账户ID | 推断',
  `title` VARCHAR(512) COMMENT '标题 | Flutter WalletTransaction.title',
  `amount` DECIMAL(12,2) COMMENT '发生额（有符号）| Flutter WalletTransaction.amount',
  `balance_after` DECIMAL(12,2) COMMENT '交易后余额 | Flutter WalletTransaction.balanceAfter',
  `type` VARCHAR(64) COMMENT '收支类型（income / expense）| Flutter WalletTransactionType',
  `time` BIGINT COMMENT '发生时间 | Flutter WalletTransaction.time',
  `status` INT COMMENT '状态 | 推断',
  PRIMARY KEY (`id`),
  KEY `idx_walletseq_user` (`user_id`,`time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='钱包流水 | v1/moneyaccount/sequences + Flutter WalletTransaction';

CREATE TABLE IF NOT EXISTS `gift_cards` (
  `id` BIGINT NOT NULL COMMENT '卡ID | GiftCard.id',
  `user_id` BIGINT COMMENT '持有用户 | GiftCard.userId',
  `template_id` INT COMMENT '卡模板ID | GiftCard.templateId',
  `title` VARCHAR(512) COMMENT '卡名 | GiftCard.title',
  `desc` TEXT COMMENT '说明 | GiftCard.desc',
  `denomination` DECIMAL(12,2) COMMENT '面额 | GiftCard.denomination',
  `balance` DECIMAL(12,2) COMMENT '余额 | GiftCard.balance',
  `use_amount` DECIMAL(12,2) COMMENT '本次抵扣额 | GiftCard.useAmount',
  `effective_time_from` BIGINT COMMENT '生效时间 | GiftCard.effectiveTimeFrom',
  `effective_time_to` BIGINT COMMENT '失效时间 | GiftCard.effectiveTimeTo',
  `status` INT COMMENT '状态 | GiftCard.status / Flutter GiftCardStatus',
  `type` INT COMMENT '类型 | GiftCard.type',
  `is_can_revoke` TINYINT(1) COMMENT '可撤销 | GiftCard.isCanRevoke',
  `is_disable` TINYINT(1) COMMENT '已禁用 | GiftCard.isDisable',
  `serial_number` VARCHAR(128) COMMENT '卡号（Flutter 补充）| Flutter GiftCard.serialNumber',
  `add_time` BIGINT COMMENT '创建时间 | GiftCard.addTime',
  `update_time` BIGINT COMMENT '更新时间 | GiftCard.updateTime',
  PRIMARY KEY (`id`),
  KEY `idx_gift_user` (`user_id`,`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='礼品卡 | GiftCard（v1/giftCards · v2/giftCards · v3/order/allGiftCard）+ Flutter GiftCard';

CREATE TABLE IF NOT EXISTS `gift_card_usages` (
  `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '代理主键',
  `order_id` BIGINT COMMENT '订单ID | 推断',
  `gift_card_id` BIGINT COMMENT '卡ID | GiftCard.id',
  `use_amount` DECIMAL(12,2) COMMENT '抵扣额 | GiftCard.useAmount',
  PRIMARY KEY (`id`),
  KEY `idx_gcu_order` (`order_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='订单礼品卡抵扣明细 | MakeOrderRequestBody.giftCards';


-- ------------------------------------------------------------------------
--  ↓ 以下为 07_marketing.sql
-- ------------------------------------------------------------------------

-- ========================================================================
--  sql_rebuildnew · 07_marketing.sql（忠实原样版，不做 R4）
--  SQL 正文与 docs/sql_new 逐语句一致，仅替换过时生成头部。
--  在 MySQL 8.4.10 上会产出 119 条 Warning 1681（TINYINT(1) 显示宽度弃用），0 ERROR。
-- ========================================================================

-- ========================================================================
--  07 · 营销资产
--  服务库: planstar_marketing
--  执行顺序: 01_init_databases.sql → 02 … 09 → 10 / 11 → 12_constraints.sql
--  本文件 12 张表: coupons, campaigns, coupon_packages, pin_tuan_campaigns, pin_tuan_campaign_items, pin_tuan_orders, group_buy_members, proxy_buy_configs, proxy_buy_orders, transfers, to_watch, favorites
-- ========================================================================
--  目录结构参考: github.com/arixbit/kratos-shop/tree/main/sql
--    NN_<域>.sql 按服务分库 + migrations/<库>/000001_init.up.sql / .down.sql
--  方言: MySQL 8.0 / InnoDB / utf8mb4（按用户选择保持，未切换 PostgreSQL）
--  第三方 SDK 表（小米推送 / 极光 JPush / 百度定位·mshield / Bugly /
--    字节 AppLog·bdtracker）共 31 张已按要求全部移除。
-- ========================================================================

USE `planstar_marketing`;

-- ----------------------------------------------------------------------
--  域 5 · 营销资产（marketing）
-- ----------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS `coupons` (
  `id` BIGINT NOT NULL COMMENT '券ID | Coupon.id',
  `user_id` BIGINT COMMENT '持有用户 | Coupon.userId',
  `order_id` BIGINT COMMENT '使用订单 | Coupon.orderId',
  `activity_id` BIGINT COMMENT '适用演出 | Coupon.activityId',
  `title` VARCHAR(512) COMMENT '券名 | Coupon.title',
  `desc` TEXT COMMENT '说明 | Coupon.desc',
  `discount_type` INT COMMENT '抵扣类型（1 固定金额 / 2 折扣率）| Coupon.discountType / Flutter CouponDiscountType',
  `discount_amount` DECIMAL(12,2) COMMENT '抵扣金额 | Coupon.discountAmount',
  `discount_rate` DECIMAL(10,4) COMMENT '折扣率（万分比口径）| Coupon.discountRate',
  `require_amount` DECIMAL(12,2) COMMENT '使用门槛 | Coupon.requireAmount',
  `effective_time_from` BIGINT COMMENT '生效时间 | Coupon.effectiveTimeFrom',
  `effective_time_to` BIGINT COMMENT '失效时间 | Coupon.effectiveTimeTo',
  `cost_bear_type` INT COMMENT '成本承担方 | Coupon.costBearType',
  `is_can_revoke` TINYINT(1) COMMENT '可撤销 | Coupon.isCanRevoke',
  `is_disable` TINYINT(1) COMMENT '已禁用 | Coupon.isDisable',
  `status` INT COMMENT '状态（可用/已用/过期）| Coupon.status / Flutter CouponStatus',
  `success` TINYINT(1) COMMENT '领取结果 | Coupon.success',
  `source_id` BIGINT COMMENT '来源ID | Coupon.sourceId',
  `source_type` INT COMMENT '来源类型 | Coupon.sourceType',
  `type` INT COMMENT '券类型 | Coupon.type',
  `activity_event_id` BIGINT COMMENT '适用场次 | Coupon.activityEventId',
  `extra_condition` JSON COMMENT '附加条件（原 ExtraConditionBean）| Coupon.extraCondition',
  `data` JSON COMMENT '自嵌套券（原 Coupon data）| Coupon.data',
  `activity_category_ids` JSON COMMENT '适用分类（原 List<Integer>）| Coupon.activityCategoryIds',
  `activity_ids` JSON COMMENT '适用演出（原 List<Integer>）| Coupon.activityIds',
  `client_types` JSON COMMENT '适用客户端（原 List<Integer>）| Coupon.clientTypes',
  `shop_ids` JSON COMMENT '适用店铺（原 List<Integer>）| Coupon.shopIds',
  `add_time` BIGINT COMMENT '创建时间 | Coupon.addTime',
  `update_time` BIGINT COMMENT '更新时间 | Coupon.updateTime',
  PRIMARY KEY (`id`),
  KEY `idx_coupon_user` (`user_id`,`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='优惠券 | Coupon（v1/coupons · POST v1/coupons）+ Flutter Coupon';

CREATE TABLE IF NOT EXISTS `campaigns` (
  `id` BIGINT NOT NULL COMMENT '活动ID | Campaign.id / OrderDiscount.id',
  `title` VARCHAR(512) COMMENT '标题 | Campaign.title',
  `desc` TEXT COMMENT '描述 | Campaign.desc',
  `discount_amount` DECIMAL(12,2) COMMENT '优惠金额 | Campaign.discountAmount / OrderDiscount.discountAmount',
  `background_img` VARCHAR(1024) COMMENT '背景图 | CampaignBean.backgroundImg',
  `content` TEXT COMMENT '内容 | CampaignBean.content',
  `url` VARCHAR(1024) COMMENT '链接 | CampaignBean.url',
  `can_buy` TINYINT(1) COMMENT '可购买 | OrderDiscount.canBuy',
  `can_use_coupon` TINYINT(1) COMMENT '可叠加券 | OrderDiscount.canUseCoupon',
  `error_hint` VARCHAR(1024) COMMENT '错误提示 | OrderDiscount.errorHint',
  `coupons` JSON COMMENT '券列表（原 List<Coupon>）| OrderDiscount.coupons',
  `disable_coupons` JSON COMMENT '不可用券 | OrderDiscount.disableCoupons',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='营销活动 / 折扣 | Campaign / CampaignBean / OrderDiscount';

CREATE TABLE IF NOT EXISTS `coupon_packages` (
  `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '代理主键',
  `has_coupon` TINYINT(1) COMMENT '是否有券 | CouponPackageBean.hasCoupon',
  `message` VARCHAR(1024) COMMENT '文案 | CouponPackageBean.message',
  `title` VARCHAR(512) COMMENT '标题 | CouponPackageBean.title',
  `total` INT COMMENT '总数 | CouponPackageBean.total',
  `url` VARCHAR(1024) COMMENT '链接 | CouponPackageBean.url',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='券包 | CouponPackageBean';

CREATE TABLE IF NOT EXISTS `pin_tuan_campaigns` (
  `id` BIGINT NOT NULL COMMENT '拼团活动ID | PinTuanInfo.id',
  `activity_id` BIGINT COMMENT '演出ID | PinTuanInfo.activityId',
  `shop_id` BIGINT COMMENT '店铺ID | PinTuanInfo.shopId',
  `low_price` DECIMAL(12,2) COMMENT '拼团低价 | PinTuanInfo.lowPrice',
  `original_price` DECIMAL(12,2) COMMENT '原价（Flutter）| Flutter GroupBuyInfo.originalPrice',
  `pin_tuan_tag` VARCHAR(256) COMMENT '拼团标签 | PinTuanInfo.pinTuanTag',
  `slogan` VARCHAR(1024) COMMENT '标语 | PinTuanInfo.slogan',
  `stock` INT COMMENT '库存 | PinTuanInfo.stock',
  `succeed_num` INT COMMENT '成团人数 | PinTuanInfo.succeedNum',
  `current_members` INT COMMENT '当前人数（Flutter）| Flutter GroupBuyInfo.currentMembers',
  `is_open` INT COMMENT '是否开启 | PinTuanInfo.isOpen',
  `status` VARCHAR(64) COMMENT '状态（Flutter GroupBuyStatus）| Flutter',
  `show_title` VARCHAR(512) COMMENT '演出名（Flutter）| Flutter GroupBuyInfo.showTitle',
  `venue` VARCHAR(256) COMMENT '场馆（Flutter）| Flutter',
  `event_time` VARCHAR(128) COMMENT '场次时间（Flutter）| Flutter',
  `effective_from` BIGINT COMMENT '生效时间 | PinTuanInfo.effectiveFrom',
  `effective_to` BIGINT COMMENT '失效时间 | PinTuanInfo.effectiveTo',
  `add_time` BIGINT COMMENT '创建时间 | PinTuanInfo.addTime',
  `update_time` BIGINT COMMENT '更新时间 | PinTuanInfo.updateTime',
  PRIMARY KEY (`id`),
  KEY `idx_pt_act` (`activity_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='拼团活动 | PinTuanInfo（GET v1/pinTuanCampaign/{id}）+ Flutter GroupBuyInfo';

CREATE TABLE IF NOT EXISTS `pin_tuan_campaign_items` (
  `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '代理主键',
  `pin_tuan_id` BIGINT COMMENT '拼团活动ID | 推断',
  `name` VARCHAR(256) COMMENT '名称 | PinTuanCampaignItem.name',
  `ticket_group` JSON COMMENT '票组（原 TicketGroup）| PinTuanCampaignItem.pinTuanCampaignItem',
  `ticket_category_id` BIGINT COMMENT '票档ID | 推断',
  PRIMARY KEY (`id`),
  KEY `idx_ptitem_pt` (`pin_tuan_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='拼团票档项 | PinTuanCampaignItem / PinTuanCampaignMap';

CREATE TABLE IF NOT EXISTS `pin_tuan_orders` (
  `id` BIGINT NOT NULL COMMENT '拼团订单ID | 推断',
  `user_id` BIGINT COMMENT '用户ID | 推断',
  `order_id` BIGINT COMMENT '统一订单ID | 推断',
  `pin_tuan_id` BIGINT COMMENT '拼团活动ID | PinTuanOrderRequestBody.pinTuanId',
  `pin_tuan_campaign_item_id` BIGINT COMMENT '拼团项ID | PinTuanOrderRequestBody.pinTuanCampaignItemId',
  `count` INT COMMENT '数量 | PinTuanOrderRequestBody.count',
  `succeed_num` INT COMMENT '已成团人数 | 推断',
  `status` INT COMMENT '状态 | 推断',
  `add_time` BIGINT COMMENT '创建时间 | 推断',
  PRIMARY KEY (`id`),
  KEY `idx_ptorder_user` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='拼团订单 | v1/pinTuanOrder/make · confirmWithLogin · myOrders';

CREATE TABLE IF NOT EXISTS `group_buy_members` (
  `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '代理主键',
  `group_id` BIGINT COMMENT '拼团ID | 推断',
  `user_id` BIGINT COMMENT '用户ID | Flutter GroupBuyMember.userId',
  `nickname` VARCHAR(128) COMMENT '昵称 | Flutter',
  `avatar_url` VARCHAR(1024) COMMENT '头像 | Flutter',
  `is_leader` TINYINT(1) COMMENT '是否团长 | Flutter',
  `joined_at` BIGINT COMMENT '加入时间 | Flutter GroupBuyMember.joinedAt',
  PRIMARY KEY (`id`),
  KEY `idx_gbm_group` (`group_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='拼团成员 | Flutter GroupBuyMember';

CREATE TABLE IF NOT EXISTS `proxy_buy_configs` (
  `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '代理主键',
  `activity_id` BIGINT COMMENT '演出ID | 推断',
  `ticket_category_id` BIGINT COMMENT '票档ID | 推断',
  `proxy_buy_fee` DECIMAL(12,2) COMMENT '代抢费 | ProxyBuyInfo.proxyBuyFee',
  `proxy_buy_rule_desc` TEXT COMMENT '代抢规则 | ProxyBuyInfo.proxyBuyRuleDesc',
  `compensation_open` TINYINT(1) COMMENT '无票赔付开关 | ProxyBuyCompensationInfo.open',
  `end_value` INT COMMENT '截止时间档位值 | ProxyBuyTimeInfo.endValue',
  `end_value_desc` VARCHAR(256) COMMENT '截止时间文案 | ProxyBuyTimeInfo.endValueDesc',
  `time_str` VARCHAR(128) COMMENT '轮次时间 | ProxyBuyTurnInfo.timeStr',
  `turn_title` VARCHAR(256) COMMENT '轮次标题 | ProxyBuyTurnInfo.title',
  `proxy_buy_time_info` JSON COMMENT '时间档位列表（原 List<ProxyBuyTimeInfo>）| ProxyBuyInfo.proxyBuyTimeInfo',
  `proxy_buy_turn_info` JSON COMMENT '轮次（原 ProxyBuyTurnInfo）| ProxyBuyInfo.proxyBuyTurnInfo',
  PRIMARY KEY (`id`),
  KEY `idx_proxycfg_act` (`activity_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='代抢配置 | ProxyBuyInfo / ProxyBuyCompensationInfo / ProxyBuyTimeInfo / ProxyBuyTurnInfo';

CREATE TABLE IF NOT EXISTS `proxy_buy_orders` (
  `id` BIGINT NOT NULL COMMENT '代抢单ID | Flutter ProxyBuyOrder.id',
  `order_number` VARCHAR(64) COMMENT '订单号 | Flutter ProxyBuySubmitState.orderNumber',
  `user_id` BIGINT COMMENT '用户ID | 推断',
  `show_title` VARCHAR(512) COMMENT '演出名 | Flutter',
  `show_poster` VARCHAR(1024) COMMENT '海报 | Flutter',
  `venue` VARCHAR(256) COMMENT '场馆 | Flutter',
  `event_time` VARCHAR(128) COMMENT '场次时间 | Flutter',
  `turn` JSON COMMENT '轮次（原 ProxyBuyTurn）| Flutter',
  `deadline` JSON COMMENT '截止档（原 ProxyBuyDeadline）| Flutter',
  `ticket_price` DECIMAL(12,2) COMMENT '票款 | Flutter',
  `quantity` INT COMMENT '数量 | Flutter',
  `proxy_buy_fee` DECIMAL(12,2) COMMENT '代抢费 | Flutter',
  `total_amount` DECIMAL(12,2) COMMENT '总额（= 票款×数量 + 代抢费）| Flutter ProxyBuyHelper.calculateTotalPrice',
  `status` VARCHAR(64) COMMENT '状态（pending / inProgress / success / failed / cancelled）| Flutter ProxyBuyStatus',
  `compensation_enabled` TINYINT(1) COMMENT '是否赔付 | Flutter',
  `created_at` BIGINT COMMENT '创建时间 | Flutter',
  PRIMARY KEY (`id`),
  KEY `idx_proxyorder_user` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='代抢订单 | Flutter ProxyBuyOrder / ProxyBuySubmitState';

CREATE TABLE IF NOT EXISTS `transfers` (
  `id` BIGINT NOT NULL COMMENT '转票ID | TransformBean.transformId / TransformFeedBean.id',
  `user_id` BIGINT COMMENT '发布用户 | TransformFeedBean.userId',
  `activity_id` BIGINT COMMENT '演出ID | TransferTicket.activityId',
  `activity_event_id` BIGINT COMMENT '场次ID | TransferTicket.activityEventId',
  `ticket_category_id` BIGINT COMMENT '票档ID | TransferTicket.ticketCategoryId',
  `activity_name` VARCHAR(512) COMMENT '演出名 | TransformBean.activityName',
  `address` VARCHAR(1024) COMMENT '交易地点 | TransformBean.address',
  `category_desc` VARCHAR(256) COMMENT '票档描述 | TransformBean.categoryDesc',
  `event_specification` VARCHAR(256) COMMENT '场次规格 | TransformBean.eventSpecification',
  `count` INT COMMENT '数量 | TransferTicket.amount / TransformBean.count',
  `price` DECIMAL(12,2) COMMENT '价格 | TransferTicket.price / TransformBean.price',
  `content` TEXT COMMENT '说明 | TransferTicket.content / TransformFeedBean.content',
  `is_check` TINYINT(1) COMMENT '是否已核验 | TransformBean.isCheck',
  `poster` VARCHAR(1024) COMMENT '海报 | TransformBean.poster / TransformFeedBean.poster',
  `liked` TINYINT(1) COMMENT '已赞 | TransformFeedBean.liked',
  `likes` INT COMMENT '赞数 | TransformFeedBean.likes',
  `reply_count` INT COMMENT '回复数 | TransformFeedBean.replyCount',
  `status` INT COMMENT '状态（见 dict_enums.TransferStatus）| TransformFeedBean.status',
  `venue_city_id` INT COMMENT '场馆城市 | TransformFeedBean.venueCityId',
  `activity` JSON COMMENT '演出快照 | TransformFeedBean.activity',
  `activity_event` JSON COMMENT '场次快照 | TransformFeedBean.activityEvent',
  `ticket_category` JSON COMMENT '票档快照 | TransformFeedBean.ticketCategory',
  `user` JSON COMMENT '用户快照（原 UserBean）| TransformFeedBean.user',
  `add_time` BIGINT COMMENT '创建时间 | TransformFeedBean.addTime',
  `update_time` BIGINT COMMENT '更新时间 | TransformFeedBean.updateTime',
  PRIMARY KEY (`id`),
  KEY `idx_transfer_act` (`activity_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='转票 / 求票 | TransferTicket / TransformBean / TransformFeedBean / TransferPagination';

CREATE TABLE IF NOT EXISTS `to_watch` (
  `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '代理主键',
  `user_id` BIGINT COMMENT '用户ID | 推断',
  `activity_id` BIGINT COMMENT '演出ID | ToWatchBean.activityId',
  `activity_event_id` BIGINT COMMENT '场次ID | ToWatchBean.activityEventId',
  `activity_name` VARCHAR(512) COMMENT '演出名 | ToWatchBean.activityName',
  `short_name` VARCHAR(256) COMMENT '简称 | ToWatchBean.shortName',
  `activity_event_date` BIGINT COMMENT '场次日期 | ToWatchBean.activityEventDate',
  `order_id` BIGINT COMMENT '订单ID | ToWatchBean.orderId',
  `unified_order_id` BIGINT COMMENT '统一订单ID | ToWatchBean.unifiedOrderId',
  `venue` JSON COMMENT '场馆快照（原 Venue）| ToWatchBean.venue',
  `add_time` BIGINT COMMENT '想看时间 | 推断',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_towatch` (`user_id`,`activity_event_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='想看 / 观演日历 | ToWatchBean（v1/user/toWatch）';

CREATE TABLE IF NOT EXISTS `favorites` (
  `id` BIGINT NOT NULL COMMENT '收藏ID | FavoriteFeedBean.id',
  `user_id` BIGINT COMMENT '用户ID | FavoriteFeedBean.userId',
  `activity_id` BIGINT COMMENT '演出ID | FavoriteFeedBean.activityId',
  `type` INT COMMENT '类型 | FavoriteFeedBean.type',
  `liked` TINYINT(1) COMMENT '已赞 | FavoriteFeedBean.liked',
  `likes` INT COMMENT '赞数 | FavoriteFeedBean.likes',
  `reply_count` INT COMMENT '回复数 | FavoriteFeedBean.replyCount',
  `shares` INT COMMENT '分享数 | FavoriteFeedBean.shares',
  `notify_status` INT COMMENT '开售提醒状态 | FavoriteFeedBean.notifyStatus',
  `status` INT COMMENT '状态 | FavoriteFeedBean.status',
  `activity` JSON COMMENT '演出快照 | FavoriteFeedBean.activity',
  `user` JSON COMMENT '用户快照 | FavoriteFeedBean.user',
  `add_time` BIGINT COMMENT '创建时间 | FavoriteFeedBean.addTime',
  `update_time` BIGINT COMMENT '更新时间 | FavoriteFeedBean.updateTime',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_fav` (`user_id`,`activity_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='收藏 / 想看动态 | FavoriteFeedBean（v1/activity/favorite/my）';


-- ------------------------------------------------------------------------
--  ↓ 以下为 08_community.sql
-- ------------------------------------------------------------------------

-- ========================================================================
--  sql_rebuildnew · 08_community.sql（忠实原样版，不做 R4）
--  SQL 正文与 docs/sql_new 逐语句一致，仅替换过时生成头部。
--  在 MySQL 8.4.10 上会产出 119 条 Warning 1681（TINYINT(1) 显示宽度弃用），0 ERROR。
-- ========================================================================

-- ========================================================================
--  08 · 内容与社区 UGC
--  服务库: planstar_community
--  执行顺序: 01_init_databases.sql → 02 … 09 → 10 / 11 → 12_constraints.sql
--  本文件 10 张表: reviews, review_replies, review_todos, feeds, feed_media, topics, communities, user_articles, likes, video_channels
-- ========================================================================
--  目录结构参考: github.com/arixbit/kratos-shop/tree/main/sql
--    NN_<域>.sql 按服务分库 + migrations/<库>/000001_init.up.sql / .down.sql
--  方言: MySQL 8.0 / InnoDB / utf8mb4（按用户选择保持，未切换 PostgreSQL）
--  第三方 SDK 表（小米推送 / 极光 JPush / 百度定位·mshield / Bugly /
--    字节 AppLog·bdtracker）共 31 张已按要求全部移除。
-- ========================================================================

USE `planstar_community`;

-- ----------------------------------------------------------------------
--  域 6 · 内容与社区（UGC / feed / content）
-- ----------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS `reviews` (
  `id` BIGINT NOT NULL COMMENT '评论ID | Review.id',
  `user_id` BIGINT COMMENT '用户ID | Review.userId',
  `activity_id` BIGINT COMMENT '演出ID | Review.activityId',
  `content` TEXT COMMENT '内容 | Review.content',
  `rank` INT COMMENT '星级 | Review.rank / CreateReviewRequestBody.rank',
  `images` JSON COMMENT '图片（原 List<String>）| Review.images',
  `videos` JSON COMMENT '视频（原 List<JSONObject> / JSONArray）| Review.videos',
  `liked` TINYINT(1) COMMENT '已赞 | Review.liked',
  `likes` INT COMMENT '赞数 | Review.likes',
  `reply_count` INT COMMENT '回复数 | Review.replyCount',
  `status` INT COMMENT '状态 | Review.status',
  `num_plus_related` INT COMMENT '相关数 | Review.numPlusRelated',
  `page_index` INT COMMENT '页码（DTO 复用）| Review.pageIndex',
  `page_size` INT COMMENT '页大小（DTO 复用）| Review.pageSize',
  `total_num` INT COMMENT '总数（DTO 复用）| Review.totalNum',
  `data` JSON COMMENT '自嵌套列表（原 List<Review>）| Review.data',
  `activity` JSON COMMENT '演出快照 | Review.activity',
  `user` JSON COMMENT '用户快照 | Review.user',
  `add_time` BIGINT COMMENT '创建时间 | Review.addTime',
  `update_time` BIGINT COMMENT '更新时间 | Review.updateTime',
  PRIMARY KEY (`id`),
  KEY `idx_review_act` (`activity_id`),
  KEY `idx_review_user` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='评论 / 评价 | Review / CreateReviewRequestBody（v1/reviews · v3/reviews）';

CREATE TABLE IF NOT EXISTS `review_replies` (
  `id` BIGINT NOT NULL COMMENT '回复ID | Reply.id',
  `review_id` BIGINT COMMENT '评论ID | Reply.reviewId',
  `user_id` BIGINT COMMENT '用户ID | Reply.userId',
  `reply_to_user_id` BIGINT COMMENT '被回复用户 | Reply.replyToUserId',
  `reply_to_user_name` VARCHAR(128) COMMENT '被回复用户名 | Reply.replyToUserName',
  `content` TEXT COMMENT '内容 | Reply.content',
  `is_top` TINYINT(1) COMMENT '是否置顶 | Reply.isTop',
  `page_index` INT COMMENT '页码 | Reply.pageIndex',
  `page_size` INT COMMENT '页大小 | Reply.pageSize',
  `total_num` INT COMMENT '总数 | Reply.totalNum',
  `data` JSON COMMENT '自嵌套列表（原 List<Reply>）| Reply.data',
  `user` JSON COMMENT '用户快照 | Reply.user',
  `add_time` BIGINT COMMENT '创建时间 | Reply.addTime',
  PRIMARY KEY (`id`),
  KEY `idx_reply_review` (`review_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='评论回复 | Reply / ReplySendRequestBody（v1/reviews/{id}/replies）';

CREATE TABLE IF NOT EXISTS `review_todos` (
  `id` BIGINT NOT NULL COMMENT 'ID | ReviewToDo.id',
  `user_id` BIGINT COMMENT '用户ID | ReviewToDo.userId',
  `activity_id` BIGINT COMMENT '演出ID | ReviewToDo.activityId',
  `activity_event_id` BIGINT COMMENT '场次ID | ReviewToDo.activityEventId',
  `event_time` BIGINT COMMENT '场次时间 | ReviewToDo.eventTime',
  `status` INT COMMENT '状态 | ReviewToDo.status',
  `activity` JSON COMMENT '演出快照 | ReviewToDo.activity',
  `add_time` BIGINT COMMENT '创建时间 | ReviewToDo.addTime',
  `update_time` BIGINT COMMENT '更新时间 | ReviewToDo.updateTime',
  PRIMARY KEY (`id`),
  KEY `idx_todo_user` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='待评价 | ReviewToDo（v1/reviews/todo）';

CREATE TABLE IF NOT EXISTS `feeds` (
  `id` BIGINT NOT NULL COMMENT '动态ID | FeedBean.id / Tweet.id',
  `type` INT COMMENT '动态类型（见 dict_enums.FeedType）| FeedBean.type / FeedContentBean.type',
  `user_id` BIGINT COMMENT '作者 | FeedContentBean.userId / Tweet.userId',
  `user` JSON COMMENT '作者快照（原 UserBean）| FeedContentBean.user',
  `content` TEXT COMMENT '正文 | FeedContentBean.content / Tweet.content',
  `title` VARCHAR(512) COMMENT '标题 | FeedContentBean.title',
  `name` VARCHAR(256) COMMENT '名称 | FeedContentBean.name',
  `desc` TEXT COMMENT '描述 | FeedContentBean.desc',
  `poster` VARCHAR(1024) COMMENT '海报 | FeedContentBean.poster',
  `head_img` VARCHAR(1024) COMMENT '头图 | FeedContentBean.headImg',
  `background_img` VARCHAR(1024) COMMENT '背景图 | FeedContentBean.backgroundImg',
  `schema` VARCHAR(1024) COMMENT '跳转 scheme | FeedContentBean.schema',
  `price` DECIMAL(12,2) COMMENT '价格 | FeedContentBean.price',
  `amount` INT COMMENT '数量 | FeedContentBean.amount',
  `liked` TINYINT(1) COMMENT '已赞 | FeedContentBean.liked / Tweet.liked',
  `likes` INT COMMENT '赞数 | FeedContentBean.likes / Tweet.likes',
  `reply_count` INT COMMENT '回复数 | FeedContentBean.replyCount / Tweet.replyCount',
  `shares` INT COMMENT '分享数 | FeedContentBean.shares',
  `views` INT COMMENT '浏览数 | FeedContentBean.views / Tweet.views',
  `click_num` INT COMMENT '点击数 | FeedContentBean.clickNum / Tweet.clickNum',
  `rank` INT COMMENT '权重 | FeedContentBean.rank',
  `quality` TINYINT(1) COMMENT '精华 | FeedContentBean.quality',
  `status` INT COMMENT '状态 | FeedContentBean.status / Tweet.status',
  `feed_num` INT COMMENT '动态数 | FeedContentBean.feedNum',
  `want_watch_num` INT COMMENT '想看数 | FeedContentBean.wantWatchNum',
  `relative_activity_count` INT COMMENT '关联演出数 | FeedContentBean.relativeActivityCount',
  `top_feed` VARCHAR(64) COMMENT '置顶标记 | FeedContentBean.topFeed',
  `top_feed_id` BIGINT COMMENT '置顶ID | FeedContentBean.topFeedId',
  `top_feed_type` INT COMMENT '置顶类型 | FeedContentBean.topFeedType',
  `next_page_key` VARCHAR(256) COMMENT '下一页游标 | FeedPagination.nextPageKey',
  `has_more` TINYINT(1) COMMENT '是否有更多 | FeedPagination.hasMore',
  `content_obj` JSON COMMENT '内容体（原 FeedContentBean）| FeedBean.content',
  `addition` JSON COMMENT '自嵌套附加内容（原 FeedContentBean）| FeedContentBean.addition',
  `images` JSON COMMENT '图片（原 List<ImageInfo> / List<String>）| FeedContentBean.images',
  `videos` JSON COMMENT '视频（原 List<VideoItemBean> / List<JSONObject>）| FeedContentBean.videos',
  `tags` JSON COMMENT '标签（原 List<TagBean>）| FeedContentBean.tags',
  `topic_list` JSON COMMENT '话题（原 List<TopicBean>）| FeedContentBean.topicList',
  `users` JSON COMMENT '用户列表（原 List<UserBean>）| FeedContentBean.users',
  `group` JSON COMMENT '群组（原 CommunityBean）| FeedContentBean.group',
  `activity` JSON COMMENT '演出快照（原 ActivityBean）| FeedContentBean.activity',
  `activity_event` JSON COMMENT '场次快照（原 ActivityEventBean）| FeedContentBean.activityEvent',
  `ticket_category` JSON COMMENT '票档快照 | FeedContentBean.ticketCategory',
  `relative_activities` JSON COMMENT '关联演出（原 List<RelativeActivityBean>）| FeedContentBean.relativeActivities',
  `recommend_tweets` JSON COMMENT '推荐（原 List<FeedContentBean>）| FeedContentBean.recommendTweets',
  `follow_groups` JSON COMMENT '推荐群组（原 List<CommunityBean>）| FeedContentBean.followGroups',
  `follow_recommends` JSON COMMENT '推荐用户（原 List<RecommendUser>）| FeedContentBean.followRecommends',
  `add_time` BIGINT COMMENT '创建时间 | FeedContentBean.addTime / Tweet.addTime',
  `update_time` BIGINT COMMENT '更新时间 | FeedContentBean.updateTime / Tweet.updateTime',
  PRIMARY KEY (`id`),
  KEY `idx_feed_user` (`user_id`),
  KEY `idx_feed_type` (`type`),
  KEY `idx_feed_addtime` (`add_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='动态 / 推文 | FeedBean / FeedContentBean / Tweet / FeedPagination';

CREATE TABLE IF NOT EXISTS `feed_media` (
  `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '代理主键',
  `feed_id` BIGINT COMMENT '动态ID | 推断',
  `media_type` VARCHAR(32) COMMENT 'image / video | 推断',
  `format` VARCHAR(32) COMMENT '图片格式 | ImageInfo.format',
  `thumbnail` VARCHAR(1024) COMMENT '缩略图 | ImageInfo.thumbnail',
  `url` VARCHAR(1024) COMMENT '资源地址 | ImageInfo.url / VideoItemBean.url',
  `title` VARCHAR(512) COMMENT '视频标题 | VideoItemBean.title',
  `poster` VARCHAR(1024) COMMENT '视频封面 | VideoItemBean.poster / TweetVideo.poster',
  `duration` VARCHAR(64) COMMENT '时长 | VideoItemBean.duration / TweetVideo.duration',
  `height` INT COMMENT '高 | VideoItemBean.height',
  `width` INT COMMENT '宽 | VideoItemBean.width',
  PRIMARY KEY (`id`),
  KEY `idx_feedmedia_feed` (`feed_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='动态媒体 | ImageInfo / VideoItemBean / TweetVideo';

CREATE TABLE IF NOT EXISTS `topics` (
  `id` BIGINT NOT NULL COMMENT '话题ID | TopicBean.id',
  `name` VARCHAR(256) COMMENT '话题名 | TopicBean.name',
  `desc` TEXT COMMENT '描述 | TopicBean.desc',
  `background_img` VARCHAR(1024) COMMENT '背景图 | TopicBean.backgroundImg',
  `activity_id` BIGINT COMMENT '关联演出 | TopicBean.activityId',
  `feed_num` INT COMMENT '动态数 | TopicBean.feedNum',
  `status` INT COMMENT '状态 | TopicBean.status',
  `top_feed_id` BIGINT COMMENT '置顶动态ID | TopicBean.topFeedId',
  `top_feed_type` INT COMMENT '置顶类型 | TopicBean.topFeedType',
  `type` INT COMMENT '类型 | TopicBean.type',
  `background_color` VARCHAR(32) COMMENT '背景色 | TopicWithColorBean.backgroundColor',
  `text_color` VARCHAR(32) COMMENT '文字色 | TopicWithColorBean.textColor',
  `text` VARCHAR(256) COMMENT '文案 | TopicWithColorBean.name',
  `add_time` BIGINT COMMENT '创建时间 | TopicBean.addTime',
  `update_time` BIGINT COMMENT '更新时间 | TopicBean.updateTime',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='话题 | TopicBean / TopicWithColorBean / FeedListTopicBean';

CREATE TABLE IF NOT EXISTS `communities` (
  `id` BIGINT NOT NULL COMMENT '群组ID | CommunityBean.id',
  `name` VARCHAR(256) COMMENT '名称 | CommunityBean.name / ActivityGroupBean.groupName',
  `icon` VARCHAR(1024) COMMENT '图标 | CommunityBean.icon / ActivityGroupBean.icon',
  `desc` TEXT COMMENT '简介 | CommunityBean.desc / ActivityGroupBean.intro',
  `background_img` VARCHAR(1024) COMMENT '背景图 | CommunityBean.backgroundImg',
  `feed_num` INT COMMENT '动态数 | CommunityBean.feedNum',
  `participant_num` INT COMMENT '参与人数 | CommunityBean.participantNum',
  `update_num` INT COMMENT '更新数 | CommunityBean.updateNum',
  `subscribed` TINYINT(1) COMMENT '已订阅 | CommunityBean.subscribed',
  `tid` VARCHAR(128) COMMENT '环信群ID | ActivityGroupBean.tid',
  `is_my_team` TINYINT(1) COMMENT '我的队伍 | ActivityGroupBean.isMyTeam',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='社群 / 圈子 | CommunityBean / CommunityCategoryBean / ActivityGroupBean';

CREATE TABLE IF NOT EXISTS `user_articles` (
  `id` BIGINT NOT NULL COMMENT '文章ID | ArticleBean.id / Article.id',
  `user_id` BIGINT COMMENT '作者 | ArticleBean.userId',
  `title` VARCHAR(512) COMMENT '标题 | ArticleBean.title / Article.title',
  `desc` TEXT COMMENT '摘要 | ArticleBean.desc / Article.desc',
  `head_img` VARCHAR(1024) COMMENT '头图 | ArticleBean.headImg / Article.headImg',
  `content` JSON COMMENT '内容块（原 List<ContentBean>）| ArticleBean.content',
  `text` TEXT COMMENT '正文文本 | ArticleBean.text',
  `poster` VARCHAR(1024) COMMENT '海报 | ArticleBean.poster',
  `image` VARCHAR(1024) COMMENT '图片 | ArticleBean.image',
  `author` VARCHAR(256) COMMENT '作者名 | Article.author',
  `friendly_time` VARCHAR(128) COMMENT '时间文案 | Article.friendlyTime',
  `time_range` VARCHAR(128) COMMENT '时间范围 | ArticleBean.timeRange',
  `type` INT COMMENT '类型 | ArticleBean.type',
  `status` INT COMMENT '状态 | ArticleBean.status',
  `liked` TINYINT(1) COMMENT '已赞 | ArticleBean.liked',
  `likes` INT COMMENT '赞数 | ArticleBean.likes',
  `reply_count` INT COMMENT '回复数 | ArticleBean.replyCount',
  `venue_city_id` INT COMMENT '场馆城市 | ArticleBean.venueCityId',
  `name` VARCHAR(256) COMMENT '名称 | ArticleBean.name',
  `attributes` JSON COMMENT '属性（原 JSONObject）| ArticleBean.attributes',
  `tags` JSON COMMENT '标签（原 List<TagBean>）| ArticleBean.tags',
  `group` JSON COMMENT '群组（原 CommunityBean）| ArticleBean.group',
  `user` JSON COMMENT '用户快照 | ArticleBean.user',
  `activities` JSON COMMENT '关联演出（原 List<ActivityBean>）| Article.activities',
  `related_activies` JSON COMMENT '相关演出（原 List<ActivityBean>）| ArticleDetailBean.relatedActivies',
  `related_user_article` JSON COMMENT '相关文章（原 List<FeedBean>）| ArticleDetailBean.relatedUserArticle',
  `add_time` BIGINT COMMENT '创建时间 | ArticleBean.addTime',
  `update_time` BIGINT COMMENT '更新时间 | ArticleBean.updateTime',
  PRIMARY KEY (`id`),
  KEY `idx_article_user` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='长文章 / 用户文章 | ArticleBean / Article / ArticleDetailBean';

CREATE TABLE IF NOT EXISTS `likes` (
  `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '代理主键',
  `user_id` BIGINT COMMENT '用户ID | 推断',
  `subject_id` BIGINT COMMENT '主体ID | ReplySendRequestBody.subjectId',
  `subject_type` INT COMMENT '主体类型 | 推断',
  `type` INT COMMENT '类型 | ReplySendRequestBody.type',
  `content` TEXT COMMENT '回复内容 | ReplySendRequestBody.content',
  `reply_to_user_id` BIGINT COMMENT '被回复用户 | ReplySendRequestBody.replyToUserId',
  `status` INT COMMENT '状态 | 推断',
  `add_time` BIGINT COMMENT '创建时间 | 推断',
  PRIMARY KEY (`id`),
  KEY `idx_like_user` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='点赞 / 回复关系 | v1/userLike · v1/userReply · userLike 删除';

CREATE TABLE IF NOT EXISTS `video_channels` (
  `id` BIGINT NOT NULL COMMENT '视频ID | VideoDetailBean.id',
  `featured_video_id` INT COMMENT '精选视频ID | FeaturedVideoBean.featuredVideoId',
  `featured_video_name` VARCHAR(256) COMMENT '精选名 | FeaturedVideoBean.featuredVideoName',
  `user_id` BIGINT COMMENT '作者 | VideoDetailBean.user.userId',
  `content` TEXT COMMENT '内容 | VideoDetailBean.content',
  `group` JSON COMMENT '群组（原 CommunityBean）| VideoDetailBean.group',
  `video` JSON COMMENT '视频体（原 VideoItemBean）| VideoDetailBean.video',
  `tweet_vo` JSON COMMENT '推文（原 Tweet）| FeaturedVideoBean.tweetVO',
  `tag` JSON COMMENT '标签（原 TagBean）| VideoChannelBean.tag',
  `tags` JSON COMMENT '标签列表 | VideoDetailBean.tags',
  `related_activities` JSON COMMENT '关联演出 | VideoDetailBean.relatedActivities',
  `is_liked` TINYINT(1) COMMENT '已赞 | VideoDetailBean.isLiked',
  `likes` INT COMMENT '赞数 | VideoDetailBean.likes',
  `rank` INT COMMENT '权重 | VideoDetailBean.rank',
  `reply_count` INT COMMENT '回复数 | VideoDetailBean.replyCount',
  `shares` INT COMMENT '分享数 | VideoDetailBean.shares',
  `views` INT COMMENT '浏览数 | VideoDetailBean.views',
  `type` INT COMMENT '类型 | VideoDetailBean.type',
  `add_time` BIGINT COMMENT '创建时间 | VideoDetailBean.addTime',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='视频频道 / 视频详情 | VideoChannelBean / VideoDetailBean / FeaturedVideoBean';


-- ------------------------------------------------------------------------
--  ↓ 以下为 09_content_ops.sql
-- ------------------------------------------------------------------------

-- ========================================================================
--  sql_rebuildnew · 09_content_ops.sql（忠实原样版，不做 R4）
--  SQL 正文与 docs/sql_new 逐语句一致，仅替换过时生成头部。
--  在 MySQL 8.4.10 上会产出 119 条 Warning 1681（TINYINT(1) 显示宽度弃用），0 ERROR。
-- ========================================================================

-- ========================================================================
--  09 · 运营位 / 搜索 / 增长
--  服务库: planstar_content
--  执行顺序: 01_init_databases.sql → 02 … 09 → 10 / 11 → 12_constraints.sql
--  本文件 8 张表: banners, ads, home_floors, op_positions, search_keywords, share_logs, feedback, tracking_events
-- ========================================================================
--  目录结构参考: github.com/arixbit/kratos-shop/tree/main/sql
--    NN_<域>.sql 按服务分库 + migrations/<库>/000001_init.up.sql / .down.sql
--  方言: MySQL 8.0 / InnoDB / utf8mb4（按用户选择保持，未切换 PostgreSQL）
--  第三方 SDK 表（小米推送 / 极光 JPush / 百度定位·mshield / Bugly /
--    字节 AppLog·bdtracker）共 31 张已按要求全部移除。
-- ========================================================================

USE `planstar_content`;

-- ----------------------------------------------------------------------
--  域 7 · 运营位 / 搜索 / 增长（ops / search / growth）
-- ----------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS `banners` (
  `id` BIGINT NOT NULL COMMENT 'ID | Banner.id / AdBean.id',
  `type` INT COMMENT '类型 | Banner.type',
  `city_id` INT COMMENT '城市ID | Banner.cityId',
  `platform` INT COMMENT '平台 | Banner.platform',
  `poster` VARCHAR(1024) COMMENT '图片 | Banner.poster',
  `bg_color` VARCHAR(32) COMMENT '背景色 | Banner.bgColor',
  `schema` VARCHAR(1024) COMMENT '跳转 scheme | Banner.schema',
  `url` VARCHAR(1024) COMMENT '链接 | Banner.url',
  `sort` INT COMMENT '排序 | Banner.sort',
  `status` INT COMMENT '状态 | Banner.status',
  `from_time` BIGINT COMMENT '生效开始 | Banner.fromTime',
  `to_time` BIGINT COMMENT '生效结束 | Banner.toTime',
  `log_banner_id` INT COMMENT '埋点ID | Banner.logBannerId',
  `log_banner_type` VARCHAR(64) COMMENT '埋点类型 | Banner.logBannerType',
  `add_time` BIGINT COMMENT '创建时间 | Banner.addTime',
  `update_time` BIGINT COMMENT '更新时间 | Banner.updateTime',
  PRIMARY KEY (`id`),
  KEY `idx_banner_city` (`city_id`,`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='轮播图 / 广告位 | Banner / AdBean / HeaderBackgroundBean';

CREATE TABLE IF NOT EXISTS `ads` (
  `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '代理主键',
  `name` VARCHAR(256) COMMENT '名称 | IconsBean.name',
  `icon_tips` VARCHAR(512) COMMENT '图标提示 | IconsBean.iconTips',
  `poster` VARCHAR(1024) COMMENT '图片 | IconsBean.poster',
  `schema` VARCHAR(1024) COMMENT '跳转 | IconsBean.schema',
  `android_schema` VARCHAR(1024) COMMENT 'Android scheme | BenefitIconBean.androidSchema',
  `ios_schema` VARCHAR(1024) COMMENT 'iOS scheme | BenefitIconBean.iosSchema',
  `sub_title` VARCHAR(256) COMMENT '副标题 | BenefitIconBean.subTitle',
  `title` VARCHAR(256) COMMENT '标题 | BenefitIconBean.title',
  `url` VARCHAR(1024) COMMENT '链接 | BenefitIconBean.url',
  `icon` VARCHAR(1024) COMMENT '图标 | BenefitIconBean.icon',
  `click_num` INT COMMENT '点击数 | FeedAdBean.clickNum',
  `users` JSON COMMENT '用户列表（原 List<UserBean>）| FeedAdBean.users',
  `log_op_position_id` INT COMMENT '埋点运营位ID | HomeCenterOpBean.logOpPositionId',
  `log_banner_to_object_type` VARCHAR(64) COMMENT '埋点对象类型 | HomeCenterOpBean.logBannerToObjectType',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='运营图标 / 权益位 / 悬浮球 | IconsBean / BenefitIconBean / FeedAdBean / HomeCenterOpBean';

CREATE TABLE IF NOT EXISTS `home_floors` (
  `floor_id` VARCHAR(64) NOT NULL COMMENT '楼层ID | HomeFloorShowBean.floorId',
  `type` INT COMMENT '楼层类型（见 wiki 27 §5.1 三套编号）| HomeFloorBean.type',
  `title` VARCHAR(512) COMMENT '标题 | HomeFloorShowBean.title',
  `desc` TEXT COMMENT '描述 | HomeFloorShowBean.desc',
  `display_type` INT COMMENT '展示类型 | HomeFloorShowBean.displayType',
  `data` TEXT COMMENT '原始 JSON（HomeFloorBean.data 永久保留）| HomeFloorBean.data',
  `search_hint` VARCHAR(512) COMMENT '搜索提示 | HomeFloors.searchHint',
  `calendar_tip` JSON COMMENT '日历提示（原 CalendarTip）| HomeFloors.calendarTip',
  `actors` JSON COMMENT '艺人楼层（原 HomeActorsBean）| HomeFloorBean.actors',
  `banners` JSON COMMENT '轮播（原 List<? extends Banner>）| HomeFloorBean.banners',
  `center_op` JSON COMMENT '中央运营位（原 HomeCenterOpBean）| HomeFloorBean.centerOp',
  `choiceness` JSON COMMENT '精选（原 HomeChoicenessBean）| HomeFloorBean.choiceness',
  `discovery_wonder` JSON COMMENT '发现精彩 | HomeFloorBean.discoveryWonder',
  `guarantee` JSON COMMENT '保障（原 HomeGuaranteeBean）| HomeFloorBean.guarantee',
  `home_show_list` JSON COMMENT '演出列表（原 HomeShow）| HomeFloorBean.homeShowList',
  `hot_actor_sites` JSON COMMENT '热门艺人站点 | HomeFloorBean.hotActorSites',
  `hot_search_keywords` JSON COMMENT '热搜词 | HomeFloorBean.hotSearchKeywords',
  `hot_tour_list` JSON COMMENT '热门巡演 | HomeFloorBean.hotTourList',
  `op` JSON COMMENT '运营位列表 | HomeFloorBean.op',
  `recommend_word_list` JSON COMMENT '推荐词 | HomeFloorBean.recommendWordList',
  `recommends` JSON COMMENT '瀑布流（原 ArrayList<HomeWaterfallItem>）| HomeFloorBean.recommends',
  `shortcuts` JSON COMMENT '快捷入口 | HomeFloorBean.shortcuts',
  `theme_recommend` JSON COMMENT '主题推荐（原 HomeThemeRecommend）| HomeFloorBean.themeRecommend',
  `venues` JSON COMMENT '场馆楼层（原 HomeVenuesBean）| HomeFloorBean.venues',
  `waterfall` JSON COMMENT '瀑布流配置（原 HomeWaterfall）| HomeFloorBean.waterfall',
  PRIMARY KEY (`floor_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='首页楼层 / 内容楼层 | HomeFloorBean / HomeFloors / HomeFloorShowBean / HomeShow';

CREATE TABLE IF NOT EXISTS `op_positions` (
  `id` BIGINT NOT NULL COMMENT 'ID | OperatingPositionBean.id / launchScreenBean.id',
  `template_id` INT COMMENT '模板ID | OpPosition.templateId',
  `inner_title` VARCHAR(256) COMMENT '内标题 | OperatingPositionBean.innerTitle',
  `main_title` VARCHAR(256) COMMENT '主标题 | HomeOpbean.mainTitle',
  `sub_title` VARCHAR(256) COMMENT '副标题 | OperatingPositionBean.subTitle',
  `title` VARCHAR(256) COMMENT '标题 | OperatingPositionBean.title',
  `poster` VARCHAR(1024) COMMENT '图片 | OperatingPositionBean.poster',
  `poster_url` VARCHAR(1024) COMMENT '图片 URL | HomeOpbean.posterUrl',
  `schema` VARCHAR(1024) COMMENT '跳转 | OperatingPositionBean.schema',
  `style_type` INT COMMENT '样式类型 | HomeOpbean.styleType',
  `sub_type` INT COMMENT '子类型 | HomeOpbean.subType',
  `activity_id` BIGINT COMMENT '演出ID | HomeOpbean.activityId',
  `actor_name` VARCHAR(256) COMMENT '艺人名 | HomeOpbean.actorName',
  `low_price` DECIMAL(12,2) COMMENT '最低价 | HomeOpbean.lowPrice',
  `name` VARCHAR(256) COMMENT '名称 | HomeOpbean.name',
  `sold_out` TINYINT(1) COMMENT '售罄 | HomeOpbean.soldOut',
  `data` JSON COMMENT '数据体（原 JSONObject / DataBean）| OpPosition.data',
  `items` JSON COMMENT '条目（原 List<HomeOpItemData>）| HomeOpbean.items',
  `icon` VARCHAR(1024) COMMENT '图标 | PopBoxBean.icon',
  `icon_schema` VARCHAR(1024) COMMENT '图标跳转 | PopBoxBean.iconSchema',
  `key` VARCHAR(128) COMMENT '键 | PopBoxBean.key',
  `type` VARCHAR(64) COMMENT '类型 | PopBoxBean.type',
  `url` VARCHAR(1024) COMMENT '链接 | PopBoxBean.url',
  `city_id` INT COMMENT '城市ID | launchScreenBean.cityId',
  `duration` INT COMMENT '时长 | launchScreenBean.duration',
  `video` VARCHAR(1024) COMMENT '视频 | launchScreenBean.video',
  `open_screen_type` VARCHAR(64) COMMENT '开屏类型 | launchScreenBean.openScreenType',
  `from_time` BIGINT COMMENT '生效开始 | launchScreenBean.fromTime',
  `to_time` BIGINT COMMENT '生效结束 | launchScreenBean.toTime',
  `status` INT COMMENT '状态 | launchScreenBean.status',
  `add_time` BIGINT COMMENT '创建时间 | launchScreenBean.addTime',
  `update_time` BIGINT COMMENT '更新时间 | launchScreenBean.updateTime',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='运营位 / 弹窗 / 启动屏 | OpPosition / OperatingPositionBean / PopBoxBean / launchScreenBean / HomeOpbean';

CREATE TABLE IF NOT EXISTS `search_keywords` (
  `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '代理主键',
  `keyword` VARCHAR(256) COMMENT '关键词 | SchemaKeywordBean.keyword',
  `recommend_word` VARCHAR(256) COMMENT '推荐词 | SchemaKeywordBean.recommendWord',
  `schema` VARCHAR(1024) COMMENT '跳转 | SchemaKeywordBean.schema',
  `log_id` INT COMMENT '埋点ID | SchemaKeywordRankBean.logId',
  `pn_local_rank` INT COMMENT '本地排名 | SchemaKeywordRankBean.pnLocalRank',
  `name` VARCHAR(256) COMMENT '名称 | SearchSuggestBean.name',
  `type` VARCHAR(64) COMMENT '类型 | SearchSuggestBean.type',
  `label` VARCHAR(256) COMMENT '展示文案 | Flutter HotKeyword.label',
  `word` VARCHAR(256) COMMENT '词 | Flutter HotKeyword.word',
  `hit_count` INT COMMENT '热度 | Flutter HotKeyword.hitCount',
  `hot` TINYINT(1) COMMENT '是否热 | Flutter HotKeyword.hot',
  `key` VARCHAR(128) COMMENT '搜索键 | SearchOption.key',
  `seat_map` VARCHAR(64) COMMENT '选座 | SearchOption.seatMap',
  `sort` VARCHAR(64) COMMENT '排序 | SearchOption.sort',
  `start_time` BIGINT COMMENT '开始时间 | SearchOption.startTime',
  `end_time` BIGINT COMMENT '结束时间 | SearchOption.endTime',
  `time` VARCHAR(128) COMMENT '时间 | SearchOption.time',
  PRIMARY KEY (`id`),
  KEY `idx_searchkw_word` (`keyword`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='热搜词 / 联想词 / 搜索条件 | SchemaKeywordBean / SchemaKeywordRankBean / SearchSuggestBean / SearchOption / HotKeyword';

CREATE TABLE IF NOT EXISTS `share_logs` (
  `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '代理主键',
  `user_id` BIGINT COMMENT '用户ID | 推断',
  `subject_type` INT COMMENT '主体类型 | 推断',
  `subject_id` BIGINT COMMENT '主体ID | 推断',
  `channel` VARCHAR(64) COMMENT '渠道（WECHAT / MOMENTS / WEIBO / QQ / QZONE）| dict_enums.ShareChannelType',
  `title` VARCHAR(512) COMMENT '标题 | ShareModel.title / ShareParams.title',
  `desc` TEXT COMMENT '描述 | ShareModel.desc / ShareParams.desc',
  `image_url` VARCHAR(1024) COMMENT '图片 | ShareModel.imageURL',
  `share_image_url` VARCHAR(1024) COMMENT '分享图 | ShareModel.shareImageUrl',
  `img_url` VARCHAR(1024) COMMENT '分享图 | ShareParams.imgUrl',
  `url` VARCHAR(1024) COMMENT '链接 | ShareModel.url',
  `mp_url` VARCHAR(1024) COMMENT '小程序链接 | ShareModel.mpUrl',
  `mp_image_url` VARCHAR(1024) COMMENT '小程序图 | ShareModel.mpImageUrl',
  `link` VARCHAR(1024) COMMENT '链接 | ShareParams.link',
  `mp_link` VARCHAR(1024) COMMENT '小程序链接 | ShareParams.mpLink',
  `share_reward_confirm` TINYINT(1) COMMENT '分享奖励确认 | v1/shareReward/confirm',
  `add_time` BIGINT COMMENT '分享时间 | 推断',
  PRIMARY KEY (`id`),
  KEY `idx_share_user` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='分享记录 / 分享参数 | ShareModel / ShareParams / ShareChannelType / v1/share/log';

CREATE TABLE IF NOT EXISTS `feedback` (
  `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '代理主键',
  `user_id` BIGINT COMMENT '用户ID | 推断',
  `content` TEXT COMMENT '反馈内容 | 推断',
  `contact` VARCHAR(256) COMMENT '联系方式 | 推断',
  `type` INT COMMENT '类型 | 推断',
  `status` INT COMMENT '状态 | 推断',
  `images` JSON COMMENT '图片列表 | 推断',
  `add_time` BIGINT COMMENT '创建时间 | 推断',
  `update_time` BIGINT COMMENT '更新时间 | 推断',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='意见反馈 | v1/feedback';

CREATE TABLE IF NOT EXISTS `tracking_events` (
  `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '代理主键',
  `action` VARCHAR(64) COMMENT '动作（点击 / enter）| PNLogAction',
  `key` VARCHAR(128) COMMENT '事件 key | PNLogger',
  `keyname` VARCHAR(256) COMMENT '事件名 | PNLogger',
  `schema` VARCHAR(256) COMMENT '页面 scheme | PNSchemeMap',
  `schemaname` VARCHAR(256) COMMENT '页面名 | PNSchemeMap',
  `params` JSON COMMENT '参数 | PNLogger.sendLog',
  `city_id` INT COMMENT '城市ID | 埋点公共字段',
  `dvid` VARCHAR(128) COMMENT '设备ID | 埋点公共字段',
  `user_id` BIGINT COMMENT '用户ID | 埋点公共字段',
  `channel` VARCHAR(128) COMMENT '渠道 | 埋点公共字段',
  `referer` VARCHAR(256) COMMENT '来源页 | 埋点公共字段',
  `add_time` BIGINT COMMENT '事件时间 | 推断',
  PRIMARY KEY (`id`),
  KEY `idx_track_user` (`user_id`,`add_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='埋点事件 | PNLogger / PNLogUploader（https://track 域/log）';


-- ------------------------------------------------------------------------
--  ↓ 以下为 10_dict_enums.sql
-- ------------------------------------------------------------------------

-- ========================================================================
--  sql_rebuildnew · 10_dict_enums.sql（忠实原样版，不做 R4）
--  SQL 正文与 docs/sql_new 逐语句一致，仅替换过时生成头部。
--  在 MySQL 8.4.10 上会产出 119 条 Warning 1681（TINYINT(1) 显示宽度弃用），0 ERROR。
-- ========================================================================

-- ========================================================================
--  10 · 枚举 / 字典表
--  服务库: planstar_meta
--  执行顺序: 01_init_databases.sql → 02 … 09 → 10 / 11 → 12_constraints.sql
--  本文件 1 张表: dict_enums
-- ========================================================================
--  目录结构参考: github.com/arixbit/kratos-shop/tree/main/sql
--    NN_<域>.sql 按服务分库 + migrations/<库>/000001_init.up.sql / .down.sql
--  方言: MySQL 8.0 / InnoDB / utf8mb4（按用户选择保持，未切换 PostgreSQL）
--  第三方 SDK 表（小米推送 / 极光 JPush / 百度定位·mshield / Bugly /
--    字节 AppLog·bdtracker）共 31 张已按要求全部移除。
--  143 条种子为普通 INSERT；uk_dict (enum_name, constant_name) 会阻止重复插入，
--  重复执行本文件前请先执行 migrations/planstar_meta/000001_init.down.sql。
-- ========================================================================

USE `planstar_meta`;

CREATE TABLE IF NOT EXISTS `dict_enums` (
  `id` INT NOT NULL AUTO_INCREMENT COMMENT '字典自增代理主键（非业务 ID）',
  `enum_name` VARCHAR(128) NOT NULL COMMENT '枚举 / 常量类名',
  `constant_name` VARCHAR(128) NOT NULL COMMENT '枚举常量名',
  `code` INT NOT NULL COMMENT '服务端 wire code（字符串枚举按序号）',
  `description` VARCHAR(512) NOT NULL COMMENT '中文含义 / 备注',
  `source` VARCHAR(512) NOT NULL COMMENT '证据来源（文件:行号）',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_dict` (`enum_name`,`constant_name`),
  KEY `idx_dict_code` (`enum_name`,`code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='全量枚举字典：Android 18 枚举 + Flutter 枚举 + 编译期常量';

INSERT INTO `dict_enums` (`enum_name`, `constant_name`, `code`, `description`, `source`) VALUES
('ActivityDetailType','SUMMARY','1','简介','Android lib/enums/ActivityDetailType.java:5'),
('ActivityDetailType','DETAIL','2','详情','Android lib/enums/ActivityDetailType.java:6'),
('ActivityDetailType','LIGHT','3','亮点','Android lib/enums/ActivityDetailType.java:7'),
('ActivityDetailType','ONE_SENTENCE_SUMMARY','4','一句话简介','Android lib/enums/ActivityDetailType.java:8'),
('ActivityDetailType','SPECIAL_INFO','5','购买须知','Android lib/enums/ActivityDetailType.java:9'),
('DeliverType','EXPRESS','1','快递','Android lib/enums/DeliverType.java:12'),
('DeliverType','SELF','2','自取','Android lib/enums/DeliverType.java:13'),
('DeliverType','ONSITE','3','现场取票','Android lib/enums/DeliverType.java:14'),
('DeliverType','ETICKET','4','电子票','Android lib/enums/DeliverType.java:15'),
('DeliverType','EPASS','5','电子票','Android lib/enums/DeliverType.java:16'),
('DeliverType','SPEED_EXPRESS','7','极速配送','Android lib/enums/DeliverType.java:17'),
('DeliverType','UNKOWN','-1','未知配送（原枚举拼写错误，勿改成 UNKNOWN）','Android lib/enums/DeliverType.java:18'),
('DiscoveryOpType','RECOMMAND_CAROUSEL','1','推荐轮播图','Android lib/enums/DiscoveryOpType.java:13'),
('DiscoveryOpType','RECOMMAND_CAMPAIGN','2','推荐活动','Android lib/enums/DiscoveryOpType.java:14'),
('DiscoveryOpType','RECOMMEND_FEATURED_VIDEOS','3','推荐视频专题','Android lib/enums/DiscoveryOpType.java:15'),
('FeedOpType','AD_COMMON_BANNER','1','通用推荐栏','Android lib/enums/FeedOpType.java:5'),
('FeedOpType','AD_USER_ARTICLE','2','文章推荐位','Android lib/enums/FeedOpType.java:6'),
('FeedOpType','AD_TOPIC','3','话题推荐位','Android lib/enums/FeedOpType.java:7'),
('FeedOpType','AD_VIDEO','4','视频','Android lib/enums/FeedOpType.java:8'),
('FeedOpType','AD_VIDEO_TOPIC','5','视频话题','Android lib/enums/FeedOpType.java:9'),
('FeedOpType','AD_FEATURED_VIDEOS','6','视频专题','Android lib/enums/FeedOpType.java:10'),
('FeedOpType','AD_TOPIC_LIST','7','话题列表页','Android lib/enums/FeedOpType.java:11'),
('FeedType','REVIEW','1','review 评论','Android lib/enums/FeedType.java:8'),
('FeedType','FAVORITE','2','favorite 收藏','Android lib/enums/FeedType.java:9'),
('FeedType','TWEET','3','tweet 动态','Android lib/enums/FeedType.java:10'),
('FeedType','TRANSFORM','4','转票','Android lib/enums/FeedType.java:11'),
('FeedType','ARTICLE','5','文章','Android lib/enums/FeedType.java:12'),
('FeedType','BANNER','11','Feed 运营位','Android lib/enums/FeedType.java:13'),
('FeedType','RECOMMEND_FOLLOW','12','推荐关注','Android lib/enums/FeedType.java:14'),
('FeedType','UNKNOW','99','未知（原枚举拼写）','Android lib/enums/FeedType.java:15'),
('OrderStatus','INIT','1','订单创建成功','Android lib/enums/OrderStatus.java:5（零引用，描述从 INIT 复制到下一条）'),
('OrderStatus','PAYING','2','订单待支付','Android lib/enums/OrderStatus.java:6'),
('OrderStatus','WAITING_FOR_TICKETS','3','订单创建成功（描述错误，实为待发货）','Android lib/enums/OrderStatus.java:7'),
('OrderStatus','GETTING_TICKETS','4','订单待确认收票','Android lib/enums/OrderStatus.java:8'),
('OrderStatus','FINISHED','5','订单已完成','Android lib/enums/OrderStatus.java:9'),
('OrderStatus','CANCELED','6','订单已关闭','Android lib/enums/OrderStatus.java:10'),
('OrderStatus','REFUNDED','7','订单已退款','Android lib/enums/OrderStatus.java:11'),
('OrderUtils','INIT','0','与 OrderStatus.INIT 差 1（旧常量类，零引用）','Android lib/enums/OrderUtils.java:8'),
('OrderUtils','PAYING','1','与 OrderStatus.PAYING 差 1','Android lib/enums/OrderUtils.java:9'),
('OrderUtils','WAITING_FOR_TICKETS','2','差 1','Android lib/enums/OrderUtils.java:11'),
('OrderUtils','GETTING_TICKETS','3','差 1','Android lib/enums/OrderUtils.java:7'),
('OrderUtils','FINISHED','5','一致','Android lib/enums/OrderUtils.java:6'),
('OrderUtils','CANCELED','6','一致','Android lib/enums/OrderUtils.java:5'),
('OrderUtils','REFUNDED','7','一致','Android lib/enums/OrderUtils.java:10'),
('RefundRelationBizType','ACTIVITY_EVENT','1','演出场次','Android lib/enums/RefundRelationBizType.java:5'),
('RefundRuleMode','TICKET_AMOUNT_PERCENT','1','按票款金额百分比收取退款手续费','Android lib/enums/RefundRuleMode.java:5'),
('RefundRuleMode','FREE','2','免手续费','Android lib/enums/RefundRuleMode.java:6'),
('RefundRuleMode','FORBIDDEN','3','禁止退','Android lib/enums/RefundRuleMode.java:7'),
('SaleStatus','OnSale','1','售票中','Android lib/enums/SaleStatus.java:5'),
('SaleStatus','StopSale','2','已停售','Android lib/enums/SaleStatus.java:6'),
('SaleStatus','Expired','3','已过期','Android lib/enums/SaleStatus.java:7'),
('SaleStatus','OnReservation','4','预售中','Android lib/enums/SaleStatus.java:8'),
('SaleStatus','Refunding','5','退票中','Android lib/enums/SaleStatus.java:9'),
('SaleStatus','NotOnSale','6','未开售','Android lib/enums/SaleStatus.java:10'),
('SearchType','ACTIVITY','1','演出','Android lib/enums/SearchType.java:5（零引用）'),
('SearchType','REVIEW','2','评论','Android lib/enums/SearchType.java:6'),
('ShareChannelType','WECHAT','0','微信','Android lib/enums/ShareChannelType.java:12（Kotlin enum，无 code，按序号）'),
('ShareChannelType','MOMENTS','1','朋友圈','Android lib/enums/ShareChannelType.java:13'),
('ShareChannelType','WEIBO','2','微博','Android lib/enums/ShareChannelType.java:14'),
('ShareChannelType','QQ','3','QQ','Android lib/enums/ShareChannelType.java:15'),
('ShareChannelType','QZONE','4','QQ 空间','Android lib/enums/ShareChannelType.java:16'),
('TicketType','ALL','0','所有票','Android lib/enums/TicketType.java:5'),
('TicketType','SEAT_MAP','1','选排购票','Android lib/enums/TicketType.java:6'),
('TicketType','NORMAL','2','普通购票','Android lib/enums/TicketType.java:7'),
('TicketType','SEAT_TABLE','3','选座购票','Android lib/enums/TicketType.java:8'),
('TipsBizType','REFUND','0','条件退','Android lib/enums/TipsBizType.java:11'),
('TipsBizType','NO_TICKET_COMPENSATE','1','无票赔付','Android lib/enums/TipsBizType.java:12'),
('TipsDisplayType','CORRECT','0','普通标签','Android lib/enums/TipsDisplayType.java:11'),
('TipsDisplayType','WRAN','1','警告标签（原枚举拼写错误，勿改成 WARN）','Android lib/enums/TipsDisplayType.java:12'),
('TipsScene','ACTIVITY_DETAIL','0','商品详情页','Android lib/enums/TipsScene.java:11'),
('TipsScene','ORDER_CONFIRM','1','订单确认页','Android lib/enums/TipsScene.java:12'),
('TipsScene','ORDER_DETAIL','2','订单详情页','Android lib/enums/TipsScene.java:13'),
('TipsScene','ORDER_CONFIRM_DECLARE','3','订单确认页提示信息','Android lib/enums/TipsScene.java:14'),
('TipsScene','ORDER_CONFIRM_PROXY_BUY_WARNING','4','确认订单页代抢提示','Android lib/enums/TipsScene.java:15'),
('TransferStatus','Canceled','0','已下架','Android lib/enums/TransferStatus.java:5（零引用）'),
('TransferStatus','Auditing','1','待审核','Android lib/enums/TransferStatus.java:6'),
('TransferStatus','Passed','2','已通过在售中','Android lib/enums/TransferStatus.java:7'),
('TransferStatus','StopSell','3','已停售','Android lib/enums/TransferStatus.java:8'),
('TransferStatus','Special','4','特殊的','Android lib/enums/TransferStatus.java:9'),
('TransferStatus','Rejected','5','已驳回','Android lib/enums/TransferStatus.java:10'),
('TransferStatus','Sold','10','待发货','Android lib/enums/TransferStatus.java:11'),
('TransferStatus','Shipped','11','已发货','Android lib/enums/TransferStatus.java:12'),
('TransferStatus','Compelete','12','已完成（原枚举拼写错误）','Android lib/enums/TransferStatus.java:13'),
('UserMsgType','SYSTEM_NOTICE','1','系统通知','Android lib/enums/UserMsgType.java:5'),
('UserMsgType','MEMBER_NOTICE','2','会员消息','Android lib/enums/UserMsgType.java:6'),
('UserMsgType','CAMPAIGN_NOTICE','3','活动通知','Android lib/enums/UserMsgType.java:7'),
('UserMsgType','LIKE_NOTICE','4','赞','Android lib/enums/UserMsgType.java:8'),
('UserMsgType','FOLLOW_NOTICE','5','关注','Android lib/enums/UserMsgType.java:9'),
('UserMsgType','REPLY_NOTICE','6','回复','Android lib/enums/UserMsgType.java:10'),
('UserMsgType','FEED_AT','7','FeedAT','Android lib/enums/UserMsgType.java:11'),
('UserMsgType','TWEET_NOTICE','8','tweet 通知','Android lib/enums/UserMsgType.java:12'),
('OrderType','ticket','1','普通票','Android OrderType @interface（wiki_zh/12 §5.4，编译期常量）'),
('OrderType','pinTuan','2','拼团','Android OrderType @interface'),
('OrderType','grabTicket','5','抢票','Android OrderType @interface'),
('OrderType','plusMember','60','Plus 会员','Android OrderType @interface'),
('OrderType','speedPack','70','加速包','Android OrderType @interface'),
('OrderType','unknown','-1','未知','Android OrderType @interface'),
('SubjectType','normal','9','普通订单','gotoPayActivity 硬编码（wiki_zh/12 §5.4）'),
('SubjectType','recharge','6','会员充值','gotoPayActivity 硬编码'),
('SubjectType','pinTuan','2','拼团','gotoPayActivity 硬编码'),
('SubjectType','plusMember','60','Plus 会员','gotoPayActivity 硬编码'),
('IdCardType','idCard','1','身份证','Flutter order/domain/value_objects/visitor.dart'),
('IdCardType','passport','2','护照','Flutter visitor.dart'),
('IdCardType','hongKongPass','3','港澳通行证','Flutter visitor.dart'),
('IdCardType','taiwanPass','4','台湾通行证','Flutter visitor.dart'),
('IdCardType','hmtResidencePermit','5','港澳台居住证','Flutter visitor.dart'),
('PaymentChannel','wechat','101','微信支付（wireCode wxpay）','Flutter payment_channel.dart'),
('PaymentChannel','alipay','102','支付宝（wireCode alipay）','Flutter payment_channel.dart'),
('PaymentChannel','unionPay','103','银行卡（wireCode unionpay）','Flutter payment_channel.dart'),
('PaymentChannel','creditPay','104','信用付（wireCode creditpay）','Flutter payment_channel.dart'),
('PaymentChannel','balance','105','余额支付（wireCode balance）','Flutter payment_channel.dart'),
('SeatStatus','available','0','可选','Flutter goods/seatmap/seat_map.dart'),
('SeatStatus','sold','1','已售','Flutter seat_map.dart'),
('SeatStatus','locked','2','已锁定','Flutter seat_map.dart'),
('SeatStatus','aisle','3','通道/间隔','Flutter seat_map.dart'),
('SeatStatus','blocked','4','不可选','Flutter seat_map.dart'),
('CouponStatus','usable','0','可使用','Flutter marketing/coupon.dart'),
('CouponStatus','used','1','已使用','Flutter coupon.dart'),
('CouponStatus','expired','2','已过期','Flutter coupon.dart'),
('CouponDiscountType','amount','1','固定金额（现金券）','Flutter coupon.dart / Android Coupon.discountType'),
('CouponDiscountType','rate','2','折扣率','Flutter coupon.dart'),
('GiftCardStatus','active','0','可用','Flutter marketing/giftcard/gift_card.dart'),
('GiftCardStatus','depleted','1','已用完','Flutter gift_card.dart'),
('GiftCardStatus','disabled','2','已禁用','Flutter gift_card.dart'),
('WalletTransactionType','income','0','收入','Flutter marketing/wallet.dart'),
('WalletTransactionType','expense','1','支出','Flutter wallet.dart'),
('GroupBuyStatus','grouping','1','拼团中','Flutter group_buy/group_buy.dart'),
('GroupBuyStatus','success','2','拼团成功','Flutter group_buy.dart'),
('GroupBuyStatus','failed','3','拼团失败','Flutter group_buy.dart'),
('GroupBuyStatus','cancelled','4','已取消','Flutter group_buy.dart'),
('ProxyBuyStatus','pending','0','待抢票','Flutter proxy_buy/proxy_buy.dart'),
('ProxyBuyStatus','inProgress','1','抢票中','Flutter proxy_buy.dart'),
('ProxyBuyStatus','success','2','抢票成功','Flutter proxy_buy.dart'),
('ProxyBuyStatus','failed','3','抢票失败','Flutter proxy_buy.dart'),
('ProxyBuyStatus','cancelled','4','已取消','Flutter proxy_buy.dart'),
('RefundReason','notNeeded','0','不需要了','Flutter order/refund/refund_state.dart'),
('RefundReason','boughtWrong','1','买错了','Flutter refund_state.dart'),
('RefundReason','tooExpensive','2','票价太贵','Flutter refund_state.dart'),
('RefundReason','foundBetter','3','找到更好的票源','Flutter refund_state.dart'),
('RefundReason','other','4','其他原因','Flutter refund_state.dart'),
('DeliveryMethod','electronic','1','电子票','Flutter order/domain/value_objects/delivery.dart'),
('DeliveryMethod','express','2','快递配送','Flutter delivery.dart'),
('DeliveryMethod','selfPickup','3','现场自取','Flutter delivery.dart');


-- ------------------------------------------------------------------------
--  ↓ 以下为 11_rev_evidence.sql
-- ------------------------------------------------------------------------

-- ========================================================================
--  sql_rebuildnew · 11_rev_evidence.sql（忠实原样版，不做 R4）
--  SQL 正文与 docs/sql_new 逐语句一致，仅替换过时生成头部。
--  在 MySQL 8.4.10 上会产出 119 条 Warning 1681（TINYINT(1) 显示宽度弃用），0 ERROR。
-- ========================================================================

-- ========================================================================
--  11 · 逆向证据目录
--  服务库: planstar_meta
--  执行顺序: 01_init_databases.sql → 02 … 09 → 10 / 11 → 12_constraints.sql
--  本文件 3 张表: _rev_dto_field_catalog, _rev_endpoint_catalog, _rev_dto_table_mapping
-- ========================================================================
--  目录结构参考: github.com/arixbit/kratos-shop/tree/main/sql
--    NN_<域>.sql 按服务分库 + migrations/<库>/000001_init.up.sql / .down.sql
--  方言: MySQL 8.0 / InnoDB / utf8mb4（按用户选择保持，未切换 PostgreSQL）
--  第三方 SDK 表（小米推送 / 极光 JPush / 百度定位·mshield / Bugly /
--    字节 AppLog·bdtracker）共 31 张已按要求全部移除。
--  本目录是逆向证据，勿用于生产业务查询；仅用于核对字段零遗漏。
--  原 wiki_zh/27 §2 的 1,420 私有字段只统计 lib/model/*.java（195 个 DTO），
--  本目录含 lib/enums、models、lib/domain，共 217 类 / 1,736 字段，二者不矛盾。
-- ========================================================================

USE `planstar_meta`;

CREATE TABLE IF NOT EXISTS `_rev_dto_field_catalog` (
  `id` INT NOT NULL AUTO_INCREMENT COMMENT 'DTO 字段证据自增代理主键',
  `dto_class` VARCHAR(128) NOT NULL COMMENT 'DTO 类名（反编译 lib/model、lib/enums、models、lib/domain）',
  `java_type` VARCHAR(255) NOT NULL COMMENT '反编译得到的字段声明类型',
  `field_name` VARCHAR(128) NOT NULL COMMENT '字段名（原始 Java 命名）',
  PRIMARY KEY (`id`),
  KEY `idx_rev_dto` (`dto_class`),
  KEY `idx_rev_field` (`field_name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='逆向证据：全部 DTO 字段';

CREATE TABLE IF NOT EXISTS `_rev_endpoint_catalog` (
  `id` INT NOT NULL AUTO_INCREMENT COMMENT '端点证据自增代理主键',
  `http_method` VARCHAR(16) NOT NULL COMMENT 'HTTP 方法',
  `path` VARCHAR(512) NOT NULL COMMENT 'Retrofit @GET/@POST 注解里的路径',
  `service_file` VARCHAR(128) NOT NULL COMMENT '来源 Service 文件名',
  PRIMARY KEY (`id`),
  KEY `idx_rev_ep_path` (`path`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='逆向证据：全部 REST 端点';

CREATE TABLE IF NOT EXISTS `_rev_dto_table_mapping` (
  `id` INT NOT NULL AUTO_INCREMENT COMMENT 'DTO 映射证据自增代理主键',
  `dto_class` VARCHAR(128) NOT NULL COMMENT 'DTO 类名',
  `representation` VARCHAR(255) NOT NULL COMMENT '落库位置 / 是否为响应封装',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_rev_mapping` (`dto_class`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='逆向证据：DTO 到表的映射';

INSERT INTO `_rev_dto_field_catalog` (`dto_class`, `java_type`, `field_name`) VALUES
('ActivityBean','List<ActivityDetail>','activityDetails'),
('ActivityBean','int','activityEventNum'),
('ActivityBean','int','activityId'),
('ActivityBean','List<ActivityLineupBean>','activityLineup'),
('ActivityBean','List<ActivityGroupBean>','activityLinkedGroups'),
('ActivityBean','int','activityNum'),
('ActivityBean','ActivityPurchaseInfo','activityPurchaseInfo'),
('ActivityBean','ActivityRefundPolicyBean','activityRefundPolicy'),
('ActivityBean','boolean','activitySeatMapOpened'),
('ActivityBean','String','activityShortname'),
('ActivityBean','int','activityType'),
('ActivityBean','int','actorId'),
('ActivityBean','ActorQrBean','actorInfo'),
('ActivityBean','List<ActorBean>','actors'),
('ActivityBean','long','addTime'),
('ActivityBean','String','areaImage'),
('ActivityBean','int','areaTicketType'),
('ActivityBean','boolean','b2cOpened'),
('ActivityBean','List<ActivityCampaign>','campaigns'),
('ActivityBean','boolean','canBuyEcard'),
('ActivityBean','boolean','canReview'),
('ActivityBean','String','categoryHotRank'),
('ActivityBean','int','categoryId'),
('ActivityBean','String','categoryName'),
('ActivityBean','ChooseTicketCalendarData','chooseTicketCalendarData'),
('ActivityBean','int','cityId'),
('ActivityBean','String','cityName'),
('ActivityBean','boolean','creditPay'),
('ActivityBean','String','desc'),
('ActivityBean','boolean','directBuyOfficial'),
('ActivityBean','boolean','discontinuedTicketNotice'),
('ActivityBean','double','discountAmount'),
('ActivityBean','double','discountRate'),
('ActivityBean','List<ActivityDisplayTag>','displayTags'),
('ActivityBean','String','distance'),
('ActivityBean','String','duration'),
('ActivityBean','long','end'),
('ActivityBean','ShowTip','enterTip'),
('ActivityBean','List<ActivityEventBean>','events'),
('ActivityBean','int','favorType'),
('ActivityBean','boolean','generalAgency'),
('ActivityBean','Boolean','groupBuy'),
('ActivityBean','boolean','hasSeat'),
('ActivityBean','transient List<ActivityEventBean>','hasTicketEvents'),
('ActivityBean','double','highPrice'),
('ActivityBean','String','icon'),
('ActivityBean','int','id'),
('ActivityBean','String','img'),
('ActivityBean','boolean','isBestDiscount'),
('ActivityBean','String','linkType'),
('ActivityBean','double','lowPrice'),
('ActivityBean','String','name'),
('ActivityBean','boolean','needSpeedPackBuy'),
('ActivityBean','boolean','noTicketCompensateSwitch'),
('ActivityBean','boolean','nonRefundableExchangeable'),
('ActivityBean','boolean','normalSoldOut'),
('ActivityBean','Boolean','onlyProxyBuy'),
('ActivityBean','long','openTime'),
('ActivityBean','OrderNumLimitBean','orderNumLimit'),
('ActivityBean','boolean','payAfterUseSwitch'),
('ActivityBean','PinTuanInfo','pinTuanInfo'),
('ActivityBean','String','postageTip'),
('ActivityBean','String','poster'),
('ActivityBean','String','preSaleTip'),
('ActivityBean','Integer','priceGuaranteeType'),
('ActivityBean','double','privilegeDiscountAmount'),
('ActivityBean','String','promoImg'),
('ActivityBean','boolean','proxyBuy'),
('ActivityBean','ProxyBuyInfo','proxyBuyInfo'),
('ActivityBean','int','publishStatus'),
('ActivityBean','double','rank'),
('ActivityBean','int','reviewNum'),
('ActivityBean','String','schema'),
('ActivityBean','int','searchable'),
('ActivityBean','boolean','seatBuyOfficial'),
('ActivityBean','boolean','serviceChargeOpen'),
('ActivityBean','Integer','shopId'),
('ActivityBean','String','shortname'),
('ActivityBean','boolean','showStartRemind'),
('ActivityBean','int','showType'),
('ActivityBean','boolean','soldOut'),
('ActivityBean','boolean','speedExpress'),
('ActivityBean','int','star'),
('ActivityBean','long','start'),
('ActivityBean','long','startSellTime'),
('ActivityBean','int','status'),
('ActivityBean','String','subCategoryName'),
('ActivityBean','String','tagImage'),
('ActivityBean','List<TicketCategory>','ticketCategories'),
('ActivityBean','ShowTip','ticketTip'),
('ActivityBean','String','timeRange'),
('ActivityBean','int','totalOrderCount'),
('ActivityBean','TourBean','tour'),
('ActivityBean','boolean','ultraLowGroup'),
('ActivityBean','long','updateTime'),
('ActivityBean','Venue','venue'),
('ActivityBean','int','venueCityId'),
('ActivityBean','int','venueId'),
('ActivityBean','String','venueName'),
('ActivityBean','String','video'),
('ActivityBean','int','wantWatchNum'),
('ActivityBean','Boolean','wuyouTransferSupported'),
('ActivityBean','String','year'),
('ActivityBean','List<ActivityBean>','activities'),
('ActivityBean','String','tourName'),
('ActivityBean','int','numMax'),
('ActivityBean','int','numMin'),
('ActivityBean','int','orderLimit'),
('ActivityBean','boolean','supportDirectBuy'),
('ActivityBean','boolean','supportProxyBuy'),
('ActivityBean','int','type'),
('ActivityBean','String','typeDesc'),
('ActivityCalendarBean','List<ActivityBean>','data'),
('ActivityCalendarBean','String','name'),
('ActivityCampaign','long','addTime'),
('ActivityCampaign','String','campaignRule'),
('ActivityCampaign','String','desc'),
('ActivityCampaign','String','headImg'),
('ActivityCampaign','int','id'),
('ActivityCampaign','String','name'),
('ActivityCampaign','int','purchaseType'),
('ActivityCampaign','String','shareDesc'),
('ActivityCampaign','String','shareImg'),
('ActivityCampaign','String','shareTitle'),
('ActivityCampaign','int','status'),
('ActivityCampaign','int','type'),
('ActivityCampaign','long','updateTime'),
('ActivityCampaign','String','url'),
('ActivityCampaign','int','userGroupId'),
('ActivityCampaign','int','userGroupType'),
('ActivityDetail','int','activityId'),
('ActivityDetail','long','addTime'),
('ActivityDetail','String','detailDesc'),
('ActivityDetail','int','id'),
('ActivityDetail','int','type'),
('ActivityDetail','long','updateTime'),
('ActivityEventBean','int','activityId'),
('ActivityEventBean','long','addTime'),
('ActivityEventBean','double','discountAmount'),
('ActivityEventBean','boolean','discountFlag'),
('ActivityEventBean','double','discountRate'),
('ActivityEventBean','long','end'),
('ActivityEventBean','boolean','hasTicket'),
('ActivityEventBean','int','id'),
('ActivityEventBean','double','lowPrice'),
('ActivityEventBean','boolean','priceLowest'),
('ActivityEventBean','boolean','seatMapFlag'),
('ActivityEventBean','int','seatTicketsSize'),
('ActivityEventBean','long','start'),
('ActivityEventBean','int','status');

INSERT INTO `_rev_dto_field_catalog` (`dto_class`, `java_type`, `field_name`) VALUES
('ActivityEventBean','int','ticketGroupId'),
('ActivityEventBean','int','ticketsFlag'),
('ActivityEventBean','int','ticketsNumber'),
('ActivityEventBean','long','updateTime'),
('ActivityEventBean','String','specification'),
('ActivityEventBean','String','desc'),
('ActivityEventBean','List<? extends TicketCategory>','ticketCategories'),
('ActivityEventBean','List<Integer>','availableNumbers'),
('ActivityGalleryBean','int','activityId'),
('ActivityGalleryBean','long','addTime'),
('ActivityGalleryBean','String','content'),
('ActivityGalleryBean','String','feedType'),
('ActivityGalleryBean','int','id'),
('ActivityGalleryBean','boolean','liked'),
('ActivityGalleryBean','int','likes'),
('ActivityGalleryBean','String','poster'),
('ActivityGalleryBean','int','replyCount'),
('ActivityGalleryBean','int','reviewId'),
('ActivityGalleryBean','int','status'),
('ActivityGalleryBean','int','subjectId'),
('ActivityGalleryBean','int','subjectType'),
('ActivityGalleryBean','int','type'),
('ActivityGalleryBean','long','updateTime'),
('ActivityGalleryBean','String','url'),
('ActivityGalleryBean','UserBean','user'),
('ActivityGroupBean','String','groupName'),
('ActivityGroupBean','String','icon'),
('ActivityGroupBean','String','intro'),
('ActivityGroupBean','boolean','isMyTeam'),
('ActivityGroupBean','String','tid'),
('ActivityLineupBean','String','actorDetailSchema'),
('ActivityLineupBean','String','avatar'),
('ActivityLineupBean','List<UserBean>','fans'),
('ActivityLineupBean','int','followNum'),
('ActivityLineupBean','boolean','followed'),
('ActivityLineupBean','int','id'),
('ActivityLineupBean','String','name'),
('ActivityLineupBean','String','role'),
('ActivityRefundPolicyBean','int','activityId'),
('ActivityRefundPolicyBean','String','content'),
('ActivityRefundPolicyBean','int','id'),
('ActivityRefundPolicyBean','List<ActivityRefundRuleBean>','refundRuleList'),
('ActivityRefundPolicyBean','String','tips'),
('ActivityRefundRelationBean','int','bizId'),
('ActivityRefundRelationBean','String','bizType'),
('ActivityRefundRelationBean','int','id'),
('ActivityRefundRelationBean','String','name'),
('ActivityRefundRuleBean','int','id'),
('ActivityRefundRuleBean','List<ActivityRefundRelationBean>','relations'),
('ActivityRefundRuleBean','List<ActivityRefundRuleDetailBean>','ruleDetails'),
('ActivityRefundRuleDetailBean','Long','endTime'),
('ActivityRefundRuleDetailBean','int','id'),
('ActivityRefundRuleDetailBean','String','refundAmount'),
('ActivityRefundRuleDetailBean','String','refundMode'),
('ActivityRefundRuleDetailBean','Long','startTime'),
('ActivitySubjectBean','String','desc'),
('ActivitySubjectBean','List<FloorsBean>','floors'),
('ActivitySubjectBean','String','headImg'),
('ActivitySubjectBean','int','id'),
('ActivitySubjectBean','String','shareDesc'),
('ActivitySubjectBean','String','shareImg'),
('ActivitySubjectBean','String','shareTitle'),
('ActivitySubjectBean','String','title'),
('ActivitySubjectBean','List<ContentsBean>','contents'),
('ActivitySubjectBean','ActivityBean','activity'),
('ActivitySubjectBean','String','img'),
('ActivitySubjectBean','double','lowPrice'),
('ActivitySubjectBean','List<Review>','reviews'),
('ActivitySubjectBean','boolean','soldOut'),
('ActivitySubjectBean','int','sort'),
('ActivitySubjectBean','int','type'),
('ActivitySubjectBean','Venue','venue'),
('ActorBean','String','actorDetailSchema'),
('ActorBean','String','avatar'),
('ActorBean','String','backgroundImage'),
('ActorBean','List<UserBean>','fans'),
('ActorBean','String','name'),
('ActorBean','String','showDesc'),
('ActorBean','List<ActivityBean>','topActivities'),
('ActorBean','int','followNum'),
('ActorBean','int','id'),
('ActorBean','int','userId'),
('ActorBean','boolean','isFollowed'),
('ActorBean','int','activityCount'),
('ActorQrBean','String','actorId'),
('ActorQrBean','String','afterSalesQr'),
('ActorQrBean','String','preSalesQr'),
('AdBean','long','addTime'),
('AdBean','String','bgColor'),
('AdBean','int','cityId'),
('AdBean','int','id'),
('AdBean','int','platform'),
('AdBean','String','poster'),
('AdBean','String','schema'),
('AdBean','int','sort'),
('AdBean','int','status'),
('AdBean','int','type'),
('AdBean','long','updateTime'),
('AdBean','String','url'),
('AddressBean','long','addTime'),
('AddressBean','String','address'),
('AddressBean','int','addressType'),
('AddressBean','String','district'),
('AddressBean','int','districtId'),
('AddressBean','int','id'),
('AddressBean','int','isDefault'),
('AddressBean','int','isRecent'),
('AddressBean','String','name'),
('AddressBean','String','phone'),
('AddressBean','int','status'),
('AddressBean','long','updateTime'),
('AddressBean','int','userId'),
('Article','List<ActivityBean>','activities'),
('Article','long','addTime'),
('Article','String','author'),
('Article','String','content'),
('Article','String','desc'),
('Article','String','friendlyTime'),
('Article','String','headImg'),
('Article','int','id'),
('Article','String','title'),
('ArticleBean','long','addTime'),
('ArticleBean','List<ContentBean>','content'),
('ArticleBean','String','desc'),
('ArticleBean','CommunityBean','group'),
('ArticleBean','String','headImg'),
('ArticleBean','int','id'),
('ArticleBean','boolean','liked'),
('ArticleBean','int','likes'),
('ArticleBean','int','replyCount'),
('ArticleBean','int','status'),
('ArticleBean','List<TagBean>','tags'),
('ArticleBean','String','title'),
('ArticleBean','long','updateTime'),
('ArticleBean','UserBean','user'),
('ArticleBean','int','userId'),
('ArticleBean','JSONObject','attributes'),
('ArticleBean','String','image'),
('ArticleBean','String','name'),
('ArticleBean','String','poster'),
('ArticleBean','String','text'),
('ArticleBean','String','timeRange'),
('ArticleBean','int','type'),
('ArticleBean','int','venueCityId'),
('ArticleDetailBean','List<ActivityBean>','relatedActivies'),
('ArticleDetailBean','List<FeedBean>','relatedUserArticle'),
('ArticleDetailBean','ArticleBean','userArticle'),
('Banner','long','addTime'),
('Banner','String','bgColor'),
('Banner','int','cityId');

INSERT INTO `_rev_dto_field_catalog` (`dto_class`, `java_type`, `field_name`) VALUES
('Banner','long','fromTime'),
('Banner','int','id'),
('Banner','int','logBannerId'),
('Banner','String','logBannerType'),
('Banner','int','platform'),
('Banner','String','poster'),
('Banner','String','schema'),
('Banner','int','sort'),
('Banner','int','status'),
('Banner','long','toTime'),
('Banner','int','type'),
('Banner','long','updateTime'),
('Banner','String','url'),
('BenefitIconBean','String','androidSchema'),
('BenefitIconBean','String','icon'),
('BenefitIconBean','String','iosSchema'),
('BenefitIconBean','String','subTitle'),
('BenefitIconBean','String','title'),
('BenefitIconBean','String','url'),
('BizParam','String','channel'),
('BizParam','Long','expireTime'),
('BizParam','Integer','pinTuanId'),
('BizParam','int','speedPackBalance'),
('BizParam','List<SpeedPackChoice>','speedPackChoices'),
('BizParam','SplitOrderContext','splitOrderContext'),
('CalendarTip','String','date'),
('CalendarTip','String','desc'),
('CalendarTip','int','tipId'),
('Campaign','double','discountAmount'),
('Campaign','Integer','id'),
('Campaign','String','title'),
('Campaign','String','desc'),
('CampaignBean','String','backgroundImg'),
('CampaignBean','String','content'),
('CampaignBean','String','title'),
('CampaignBean','String','url'),
('ChooseTicketBean','int','numMax'),
('ChooseTicketBean','int','numMin'),
('ChooseTicketBean','int','selectNumMax'),
('ChooseTicketBean','int','selectNumMin'),
('ChooseTicketBean','List<TicketGroupInfo>','ticketGroupInfoList'),
('ChooseTicketBean','String','type'),
('ChooseTicketBean','int','selectCount'),
('ChooseTicketBean','TicketGroupInfoDetail','ticketGroupInfoDetail'),
('ChooseTicketBean','String','tip'),
('ChooseTicketBean','TicketGroup','ticketGroup'),
('ChooseTicketCalendarData','long','defaultSelectedDayInMillis'),
('ChooseTicketCalendarData','Map<String, List<ActivityEventBean>>','eventMap'),
('ChooseTicketCalendarData','List<Calendar>','months'),
('ChooseTicketTipBean','String','desc'),
('ChooseTicketTipBean','String','icon'),
('ChooseTicketTipBean','String','title'),
('CityBean','double','GLat'),
('CityBean','double','GLng'),
('CityBean','String','cityAbbrCode'),
('CityBean','String','cityEnName'),
('CityBean','int','cityId'),
('CityBean','String','cityName'),
('CityBean','String','firstLetter'),
('CityBean','boolean','hotCity'),
('CityBean','int','provinceId'),
('CityBean','int','regionCityId'),
('CityBean','int','status'),
('CommentSchemeBean','String','desc'),
('CommentSchemeBean','String','name'),
('CommonInstallment','int','installmentNum'),
('CommonInstallment','double','installmentRate'),
('CommonInstallment','boolean','isSelect'),
('CommonShareInfoBean','String','headImage'),
('CommunityBean','String','backgroundImg'),
('CommunityBean','String','desc'),
('CommunityBean','int','feedNum'),
('CommunityBean','String','icon'),
('CommunityBean','int','id'),
('CommunityBean','String','name'),
('CommunityBean','int','participantNum'),
('CommunityBean','boolean','subscribed'),
('CommunityBean','int','updateNum'),
('CommunityCategoryBean','int','id'),
('CommunityCategoryBean','String','name'),
('Contact','String','phone'),
('Contact','String','name'),
('CountryArea','String','countryAreaCode'),
('CountryArea','String','countryAreaEnName'),
('CountryArea','int','countryAreaId'),
('CountryArea','String','countryAreaName'),
('CountryArea','String','countryAreaPinYin'),
('CountryArea','int','countryLevel'),
('CountryArea','int','isCommon'),
('Coupon','int','activityId'),
('Coupon','long','addTime'),
('Coupon','int','costBearType'),
('Coupon','Coupon','data'),
('Coupon','double','discountAmount'),
('Coupon','double','discountRate'),
('Coupon','int','discountType'),
('Coupon','long','effectiveTimeFrom'),
('Coupon','long','effectiveTimeTo'),
('Coupon','ExtraConditionBean','extraCondition'),
('Coupon','int','id'),
('Coupon','boolean','isCanRevoke'),
('Coupon','boolean','isDisable'),
('Coupon','int','orderId'),
('Coupon','double','requireAmount'),
('Coupon','int','sourceId'),
('Coupon','int','sourceType'),
('Coupon','int','status'),
('Coupon','boolean','success'),
('Coupon','int','type'),
('Coupon','long','updateTime'),
('Coupon','int','userId'),
('Coupon','String','title'),
('Coupon','String','desc'),
('Coupon','Integer','activityEventId'),
('Coupon','List<Integer>','activityCategoryIds'),
('Coupon','List<Integer>','activityIds'),
('Coupon','List<Integer>','clientTypes'),
('Coupon','List<Integer>','shopIds'),
('CouponPackageBean','boolean','hasCoupon'),
('CouponPackageBean','String','message'),
('CouponPackageBean','String','title'),
('CouponPackageBean','int','total'),
('CouponPackageBean','String','url'),
('CreateReviewRequestBody','int','activityId'),
('CreateReviewRequestBody','String','content'),
('CreateReviewRequestBody','List<String>','images'),
('CreateReviewRequestBody','int','rank'),
('CreateReviewRequestBody','JSONArray','videos'),
('Deliver','int','type'),
('Deliver','String','tip'),
('Deliver','Postage','postage'),
('Deliver','Address','address'),
('Deliver','Contact','contact'),
('Deliver','List<Addition>','additions'),
('Deliver','double','amount'),
('Deliver','String','desc'),
('Deliver','int','id'),
('Deliver','String','name'),
('Deliver','String','phone'),
('Deliver','String','district'),
('Deliver','String','value'),
('DeliverBean','AdditionBean','addition'),
('DeliverBean','int','type'),
('DeliverBean','String','contactPhone'),
('DeliverBean','String','deliverTip'),
('DeliverBean','String','needIdCard'),
('DeliverBean','String','selfGetAddress'),
('DeliverBean','String','selfGetTime'),
('DiscountActivityBean','int','activityEventId'),
('DiscountActivityBean','int','locationType');

INSERT INTO `_rev_dto_field_catalog` (`dto_class`, `java_type`, `field_name`) VALUES
('DiscountActivityBean','double','originPrice'),
('DiscountActivityBean','double','salePrice'),
('DiscountActivityBean','int','ticketCategoryId'),
('DiscountActivityBean','int','ticketGroupId'),
('DiscoveryOpBean','List<Banner>','banners'),
('DiscoveryOpBean','List<CampaignBean>','campaigns'),
('DiscoveryOpBean','List<FeaturedVideoBean>','featuredVideos'),
('DiscoveryOpBean','int','type'),
('FavoriteFeedBean','ActivityBean','activity'),
('FavoriteFeedBean','int','activityId'),
('FavoriteFeedBean','long','addTime'),
('FavoriteFeedBean','int','id'),
('FavoriteFeedBean','boolean','liked'),
('FavoriteFeedBean','int','likes'),
('FavoriteFeedBean','int','notifyStatus'),
('FavoriteFeedBean','int','replyCount'),
('FavoriteFeedBean','int','shares'),
('FavoriteFeedBean','int','status'),
('FavoriteFeedBean','int','type'),
('FavoriteFeedBean','long','updateTime'),
('FavoriteFeedBean','UserBean','user'),
('FavoriteFeedBean','int','userId'),
('FeaturedVideoBean','int','featuredVideoId'),
('FeaturedVideoBean','String','featuredVideoName'),
('FeaturedVideoBean','Tweet','tweetVO'),
('FeedAdBean','int','clickNum'),
('FeedAdBean','List<UserBean>','users'),
('FeedBean','FeedContentBean','content'),
('FeedBean','int','id'),
('FeedBean','int','type'),
('FeedContentBean','ActivityBean','activity'),
('FeedContentBean','ActivityEventBean','activityEvent'),
('FeedContentBean','int','activityId'),
('FeedContentBean','long','addTime'),
('FeedContentBean','FeedContentBean','addition'),
('FeedContentBean','int','amount'),
('FeedContentBean','String','backgroundImg'),
('FeedContentBean','int','clickNum'),
('FeedContentBean','String','content'),
('FeedContentBean','String','desc'),
('FeedContentBean','int','feedNum'),
('FeedContentBean','List<CommunityBean>','followGroups'),
('FeedContentBean','List<RecommendUser>','followRecommends'),
('FeedContentBean','CommunityBean','group'),
('FeedContentBean','String','headImg'),
('FeedContentBean','int','id'),
('FeedContentBean','List<ImageInfo>','images'),
('FeedContentBean','boolean','liked'),
('FeedContentBean','int','likes'),
('FeedContentBean','String','name'),
('FeedContentBean','String','poster'),
('FeedContentBean','int','price'),
('FeedContentBean','boolean','quality'),
('FeedContentBean','int','rank'),
('FeedContentBean','List<FeedContentBean>','recommendTweets'),
('FeedContentBean','List<RelativeActivityBean>','relativeActivities'),
('FeedContentBean','int','relativeActivityCount'),
('FeedContentBean','int','replyCount'),
('FeedContentBean','String','schema'),
('FeedContentBean','int','shares'),
('FeedContentBean','int','status'),
('FeedContentBean','List<TagBean>','tags'),
('FeedContentBean','TicketCategory','ticketCategory'),
('FeedContentBean','String','title'),
('FeedContentBean','String','topFeed'),
('FeedContentBean','int','topFeedId'),
('FeedContentBean','int','topFeedType'),
('FeedContentBean','List<TopicBean>','topicList'),
('FeedContentBean','int','type'),
('FeedContentBean','long','updateTime'),
('FeedContentBean','UserBean','user'),
('FeedContentBean','int','userId'),
('FeedContentBean','List<UserBean>','users'),
('FeedContentBean','List<VideoItemBean>','videos'),
('FeedContentBean','int','views'),
('FeedContentBean','int','wantWatchNum'),
('FeedListTopicBean','List<FeedBean>','feedList'),
('FeedListTopicBean','TopicBean','topic'),
('FeedPagination','List<FeedBean>','data'),
('FeedPagination','boolean','hasMore'),
('FeedPagination','String','nextPageKey'),
('FeedPagination','int','numPlusRelated'),
('GiftCard','long','addTime'),
('GiftCard','double','balance'),
('GiftCard','double','denomination'),
('GiftCard','String','desc'),
('GiftCard','long','effectiveTimeFrom'),
('GiftCard','long','effectiveTimeTo'),
('GiftCard','int','id'),
('GiftCard','boolean','isCanRevoke'),
('GiftCard','boolean','isDisable'),
('GiftCard','int','status'),
('GiftCard','int','templateId'),
('GiftCard','String','title'),
('GiftCard','int','type'),
('GiftCard','long','updateTime'),
('GiftCard','double','useAmount'),
('GiftCard','int','userId'),
('GiftCardInfo','boolean','canUse'),
('GiftCardInfo','String','error'),
('GiftCardInfo','List<GiftCard>','enableGiftCards'),
('GiftCardInfo','List<GiftCard>','disableGiftCards'),
('Goods','int','count'),
('Goods','int','goodId'),
('GoodsPackage','int','detailType'),
('GoodsPackage','boolean','nonRefundableExchangeable'),
('GoodsPackage','ShopInfo','shopInfo'),
('GoodsPackage','boolean','speedExpress'),
('GoodsPackage','List<Integer>','deliverTypes'),
('GoodsPackage','List<Goods>','goods'),
('GoodsPackage','List<Detail>','details'),
('GoodsPackage','Integer','activityId'),
('GoodsPackage','String','activityName'),
('GoodsPackage','String','venueName'),
('GoodsPackage','String','poster'),
('GoodsPackage','String','activityEventName'),
('GoodsPackage','String','deliverNotice'),
('GoodsPackage','String','shopNotice'),
('GoodsPackage','Tips','tips'),
('GoodsPackage','boolean','areaShow'),
('GoodsPackage','boolean','continueSeat'),
('GoodsPackage','int','count'),
('GoodsPackage','boolean','hasSeat'),
('GoodsPackage','double','salePrice'),
('GoodsPackage','String','ticketCategorySpecification'),
('GoodsPackage','String','seatName'),
('GoodsPackage','Tip','noTicketCompensateTip'),
('GoodsPackage','Tip','noTicketOnsiteCompensateTip'),
('GoodsPackage','Tip','refundTip'),
('GoodsPackage','Tip','snapUpFailTip'),
('GoodsPackage','Tip','snapUpUnFinishTip'),
('HeaderBackgroundBean','long','addTime'),
('HeaderBackgroundBean','String','bgColor'),
('HeaderBackgroundBean','int','cityId'),
('HeaderBackgroundBean','long','fromTime'),
('HeaderBackgroundBean','int','id'),
('HeaderBackgroundBean','int','platform'),
('HeaderBackgroundBean','String','poster'),
('HeaderBackgroundBean','String','schema'),
('HeaderBackgroundBean','int','sort'),
('HeaderBackgroundBean','int','status'),
('HeaderBackgroundBean','long','toTime'),
('HeaderBackgroundBean','int','type'),
('HeaderBackgroundBean','long','updateTime'),
('HeaderBackgroundBean','String','url'),
('HomeActorsBean','List<ItemsBean>','items'),
('HomeActorsBean','List<TabsBean>','tabs'),
('HomeActorsBean','String','title'),
('HomeActorsBean','int','id'),
('HomeActorsBean','String','name');

INSERT INTO `_rev_dto_field_catalog` (`dto_class`, `java_type`, `field_name`) VALUES
('HomeActorsBean','int','activityCount'),
('HomeActorsBean','String','actorDetailSchema'),
('HomeActorsBean','String','avatar'),
('HomeActorsBean','String','schema'),
('HomeBean','List<ActivityCalendarBean>','activityCalendar'),
('HomeBean','ActivitySubjectBean','activitySubject'),
('HomeBean','List<AdBean>','adBanner'),
('HomeBean','List<Banner>','banners'),
('HomeBean','List<DiscountActivityBean>','discounts'),
('HomeBean','HeaderBackgroundBean','headerBackground'),
('HomeBean','List<HotActorsBean>','hotActors'),
('HomeBean','List<ActivityBean>','hots'),
('HomeBean','List<IconsBean>','icons'),
('HomeBean','List<OpPosition>','opPositions'),
('HomeBean','List<OrganizerBean>','organizer'),
('HomeBean','PopBoxBean','popBox'),
('HomeBean','List<ArticleBean>','userArticles'),
('HomeBean','VideoChannelBean','videoChannel'),
('HomeCenterOpBean','String','logBannerToObjectType'),
('HomeCenterOpBean','int','logOpPositionId'),
('HomeCenterOpBean','String','poster'),
('HomeCenterOpBean','String','schema'),
('HomeChoicenessBean','List<ItemsBean>','items'),
('HomeChoicenessBean','List<TabsBean>','tabs'),
('HomeChoicenessBean','String','title'),
('HomeChoicenessBean','int','id'),
('HomeChoicenessBean','String','name'),
('HomeChoicenessBean','String','categoryName'),
('HomeChoicenessBean','double','discountRate'),
('HomeChoicenessBean','double','lowPrice'),
('HomeChoicenessBean','String','poster'),
('HomeChoicenessBean','String','tagImage'),
('HomeFloorBean','HomeActorsBean','actors'),
('HomeFloorBean','List<? extends Banner>','banners'),
('HomeFloorBean','HomeCenterOpBean','centerOp'),
('HomeFloorBean','HomeChoicenessBean','choiceness'),
('HomeFloorBean','String','data'),
('HomeFloorBean','List<? extends Banner>','discoveryWonder'),
('HomeFloorBean','HomeGuaranteeBean','guarantee'),
('HomeFloorBean','HomeShow','homeShowList'),
('HomeFloorBean','List<HomeHotActorSite>','hotActorSites'),
('HomeFloorBean','List<? extends SchemaKeywordBean>','hotSearchKeywords'),
('HomeFloorBean','List<? extends HotTourBean>','hotTourList'),
('HomeFloorBean','List<? extends HomeOpbean>','op'),
('HomeFloorBean','List<? extends SchemaKeywordBean>','recommendWordList'),
('HomeFloorBean','ArrayList<HomeWaterfallItem>','recommends'),
('HomeFloorBean','List<? extends IconsBean>','shortcuts'),
('HomeFloorBean','HomeThemeRecommend','themeRecommend'),
('HomeFloorBean','int','type'),
('HomeFloorBean','HomeVenuesBean','venues'),
('HomeFloorBean','HomeWaterfall','waterfall'),
('HomeFloorShowBean','List<ActivityBean>','data'),
('HomeFloorShowBean','String','desc'),
('HomeFloorShowBean','int','displayType'),
('HomeFloorShowBean','String','floorId'),
('HomeFloorShowBean','String','title'),
('HomeFloors','CalendarTip','calendarTip'),
('HomeFloors','List<HomeFloorBean>','floors'),
('HomeFloors','String','searchHint'),
('HomeGuaranteeBean','List<String>','items'),
('HomeGuaranteeBean','String','schema'),
('HomeGuaranteeBean','String','title'),
('HomeMultiItem','ActivityBean','activity'),
('HomeMultiItem','List<ActivityCalendarBean>','activityCalendar'),
('HomeMultiItem','ActivitySubjectBean','activitySubject'),
('HomeMultiItem','AdBean','ad'),
('HomeMultiItem','List<Banner>','banners'),
('HomeMultiItem','List<DiscountActivityBean>','discounts'),
('HomeMultiItem','FeedBean','feedBean'),
('HomeMultiItem','List<HotActorsBean>','hotActorsList'),
('HomeMultiItem','List<ActivityBean>','hots'),
('HomeMultiItem','List<IconsBean>','icons'),
('HomeMultiItem','List<OpPosition>','opPositions'),
('HomeMultiItem','List<OrganizerBean>','organizer'),
('HomeMultiItem','int','type'),
('HomeMultiItem','List<ArticleBean>','userArticles'),
('HomeMultiItem','VideoChannelBean','videoChannel'),
('HomeOpbean','List<HomeOpItemData>','items'),
('HomeOpbean','String','mainTitle'),
('HomeOpbean','String','posterUrl'),
('HomeOpbean','String','schema'),
('HomeOpbean','int','styleType'),
('HomeOpbean','String','subTitle'),
('HomeOpbean','Integer','subType'),
('HomeOpbean','Integer','activityId'),
('HomeOpbean','String','actorName'),
('HomeOpbean','Double','lowPrice'),
('HomeOpbean','String','name'),
('HomeOpbean','String','poster'),
('HomeOpbean','boolean','soldOut'),
('HomeShow','List<HomeShowItem>','data'),
('HomeShow','String','desc'),
('HomeShow','int','displayType'),
('HomeShow','String','floorId'),
('HomeShow','String','title'),
('HomeShowItem','String','categoryName'),
('HomeShowItem','Double','discountRate'),
('HomeShowItem','Boolean','groupBuy'),
('HomeShowItem','int','id'),
('HomeShowItem','Double','lowPrice'),
('HomeShowItem','String','name'),
('HomeShowItem','String','poster'),
('HomeShowItem','boolean','proxyBuySwitch'),
('HomeShowItem','boolean','soldOut'),
('HomeShowItem','String','tagImage'),
('HomeShowItem','String','venueCityName'),
('HomeThemeRecommend','List<Item>','items'),
('HomeThemeRecommend','String','title'),
('HomeThemeRecommend','String','color'),
('HomeThemeRecommend','String','desc'),
('HomeThemeRecommend','boolean','isSelect'),
('HomeThemeRecommend','String','logSpecialtopicType'),
('HomeThemeRecommend','String','poster'),
('HomeThemeRecommend','String','schema'),
('HomeThemeRecommend','String','tag'),
('HomeThemeRecommend','int','themeId'),
('HomeVenuesBean','List<HomeVenuesItem>','items'),
('HomeVenuesBean','String','title'),
('HomeVenuesBean','List<? extends ActivityBean>','activities'),
('HomeVenuesBean','int','activityCount'),
('HomeVenuesBean','int','id'),
('HomeVenuesBean','String','name'),
('HomeWaterfall','Pagination<HomeWaterfallItem>','items'),
('HomeWaterfall','List<Tab>','tabs'),
('HomeWaterfall','String','title'),
('HomeWaterfall','int','id'),
('HomeWaterfall','String','name'),
('HomeWaterfall','int','type'),
('HomeWaterfallActorBean','String','avatar'),
('HomeWaterfallActorBean','String','desc'),
('HomeWaterfallActorBean','List<IpCampaign>','ipCampaigns'),
('HomeWaterfallActorBean','int','ipId'),
('HomeWaterfallActorBean','String','name'),
('HomeWaterfallActorBean','String','stars'),
('HomeWaterfallActorBean','String','url'),
('HomeWaterfallActorBean','String','content'),
('HomeWaterfallActorBean','String','iconUrl'),
('HomeWaterfallCollectionBean','String','desc'),
('HomeWaterfallCollectionBean','String','imageUrl'),
('HomeWaterfallCollectionBean','int','logId'),
('HomeWaterfallCollectionBean','String','title'),
('HomeWaterfallCollectionBean','String','url'),
('HomeWaterfallItem','JSONObject','data'),
('HomeWaterfallItem','int','type'),
('HomeWaterfallListFliperBean','String','avatar'),
('HomeWaterfallListFliperBean','String','desc'),
('HomeWaterfallListFliperBean','String','nickName'),
('HomeWaterfallListShowBean','String','discountTag'),
('HomeWaterfallListShowBean','int','id'),
('HomeWaterfallListShowBean','String','name');

INSERT INTO `_rev_dto_field_catalog` (`dto_class`, `java_type`, `field_name`) VALUES
('HomeWaterfallListShowBean','String','poster'),
('HomeWaterfallListShowBean','boolean','proxyBuySwitch'),
('HomeWaterfallListShowBean','int','rank'),
('HomeWaterfallListShowBean','Double','salePrice'),
('HomeWaterfallListShowBean','String','timeRange'),
('HomeWaterfallListShowBean','String','venueName'),
('HotActorsBean','int','activityCount'),
('HotActorsBean','ActorBean','actorBean'),
('HotTourBean','int','activityCount'),
('HotTourBean','int','actorId'),
('HotTourBean','int','tourId'),
('HotTourBean','int','tourIntellectualPropertyId'),
('HotTourBean','String','tourPoster'),
('HotTourBean','String','tourTitle'),
('HttpResult','int','code'),
('HttpResult','T','data'),
('HttpResult','int','errorCode'),
('HttpResult','String','errorHint'),
('HttpResult','boolean','success'),
('IDCardType','String','desc'),
('IDCardType','int','value'),
('IconsBean','String','iconTips'),
('IconsBean','String','name'),
('IconsBean','String','poster'),
('IconsBean','String','schema'),
('IdInfo','Integer','countryAreaId'),
('IdInfo','String','countryAreaName'),
('IdInfo','int','id'),
('IdInfo','String','idCardNo'),
('IdInfo','String','idCardType'),
('IdInfo','String','name'),
('ImageInfo','String','format'),
('ImageInfo','String','thumbnail'),
('ImageInfo','String','url'),
('LoginBean','boolean','needCaptcha'),
('LoginBean','boolean','newUser'),
('LoginBean','UserProfile','user'),
('MakeOrderRequestBody','List<? extends JSONObject>','additions'),
('MakeOrderRequestBody','List<Integer>','campaignIds'),
('MakeOrderRequestBody','String','channelAttributeInfo'),
('MakeOrderRequestBody','List<Integer>','couponIds'),
('MakeOrderRequestBody','Deliver','deliver'),
('MakeOrderRequestBody','List<GiftCard>','giftCards'),
('MakeOrderRequestBody','List<IdCard>','idCards'),
('MakeOrderRequestBody','double','payAmount'),
('MakeOrderRequestBody','List<Integer>','realNameInfoIds'),
('MakeOrderRequestBody','double','totalAmount'),
('MakeOrderRequestBody','BizParam','bizParam'),
('MakeOrderRequestBody','List<Goods>','goods'),
('MakeOrderRequestBody','Integer','orderType'),
('MakeOrderRequestBody','ConfirmSnapshot','confirmSnapshot'),
('MakeOrderRequestBody','int','isContinuousSeat'),
('MakeOrderRequestBody','int','priceGuaranteeType'),
('MakeOrderRequestBody','Integer','balanceSpeedPack'),
('MakeOrderRequestBody','Integer','cashSpeedPack'),
('MakeOrderRequestBody','Integer','pinTuanId'),
('MakeOrderRequestBody','Double','splitOrderFee'),
('MakeOrderRequestBody','String','idCard'),
('MakeOrderRequestBody','Integer','idCardType'),
('MakeOrderRequestBody','String','name'),
('MakeOrderRequestBody','double','consumeAmount'),
('MakeOrderRequestBody','int','id'),
('MakeOrderRequestBody','String','phone'),
('MakeOrderRequestBody','double','postage'),
('MakeOrderRequestBody','int','type'),
('MakeOrderRequestBody','int','userAddressId'),
('MyAddressBean','List<AddressBean>','addresses'),
('MyAddressBean','int','latestId'),
('MyAddressBean','String','receiverName'),
('MyAddressBean','String','receiverPhone'),
('OpPosition','JSONObject','data'),
('OpPosition','int','templateId'),
('OperatingPositionBean','DataBean','data'),
('OperatingPositionBean','int','templateId'),
('OperatingPositionBean','String','innerTitle'),
('OperatingPositionBean','String','poster'),
('OperatingPositionBean','String','schema'),
('OperatingPositionBean','String','subTitle'),
('OperatingPositionBean','String','title'),
('OrderButtonBean','String','name'),
('OrderButtonBean','ShareParams','shareParams'),
('OrderButtonBean','Integer','type'),
('OrderCommentBean','JSONObject','comment'),
('OrderCommentBean','List<? extends Review>','reviews'),
('OrderCommentBean','List<? extends CommentSchemeBean>','schema'),
('OrderConfirmExtraInfoBean','long','addTime'),
('OrderConfirmExtraInfoBean','boolean','canEdit'),
('OrderConfirmExtraInfoBean','String','hint'),
('OrderConfirmExtraInfoBean','int','id'),
('OrderConfirmExtraInfoBean','String','label'),
('OrderConfirmExtraInfoBean','List<String>','options'),
('OrderConfirmExtraInfoBean','boolean','required'),
('OrderConfirmExtraInfoBean','int','templateId'),
('OrderConfirmExtraInfoBean','int','type'),
('OrderConfirmExtraInfoBean','long','updateTime'),
('OrderConfirmInfo','String','adTips'),
('OrderConfirmInfo','BizParam','bizParam'),
('OrderConfirmInfo','CampaignInfo','campaign'),
('OrderConfirmInfo','CanBuyInfo','canBuyInfo'),
('OrderConfirmInfo','CreditPay','creditPay'),
('OrderConfirmInfo','double','goodTotalAmount'),
('OrderConfirmInfo','String','groupBuyRuleDescForC'),
('OrderConfirmInfo','ArrayList<IDCardType>','idCardTypes'),
('OrderConfirmInfo','PayAfterUseInfo','payAfterUseInfo'),
('OrderConfirmInfo','int','priceGuaranteeType'),
('OrderConfirmInfo','PlusMember','plusMember'),
('OrderConfirmInfo','IdCard','idCard'),
('OrderConfirmInfo','CouponInfo','coupon'),
('OrderConfirmInfo','GiftCardInfo','giftCard'),
('OrderConfirmInfo','List<? extends OrderConfirmExtraInfoBean>','additionComponents'),
('OrderConfirmInfo','List<GoodsPackage>','goodPackages'),
('OrderConfirmInfo','Tip','tip'),
('OrderConfirmInfo','DeliverInfo','deliver'),
('OrderConfirmInfo','boolean','needPlusMember'),
('OrderConfirmInfo','Boolean','canBuy'),
('OrderConfirmInfo','String','reason'),
('OrderConfirmInfo','int','count'),
('OrderConfirmInfo','boolean','needIdCard'),
('OrderConfirmInfo','List<Campaign>','campaigns'),
('OrderConfirmInfo','boolean','canUse'),
('OrderConfirmInfo','String','error'),
('OrderConfirmInfo','List<Coupon>','enableCoupons'),
('OrderConfirmInfo','List<Coupon>','disableCoupons'),
('OrderConfirmInfo','String','content'),
('OrderConfirmInfo','String','schema'),
('OrderConfirmInfo','Notice','notice'),
('OrderConfirmInfo','Notice','trade'),
('OrderConfirmInfo','Notice','discount'),
('OrderConfirmInfo','List<Deliver>','details'),
('OrderConfirmInfo','boolean','multiDeliverGroup'),
('OrderConfirmInfo','boolean','check'),
('OrderConfirmInfo','String','desc'),
('OrderConfirmInfo','String','title'),
('OrderConfirmInfo','String','remark'),
('OrderConfirmInfoBean','ActivityBean','activity'),
('OrderConfirmInfoBean','ActivityEventBean','activityEvent'),
('OrderConfirmInfoBean','List<OrderConfirmExtraInfoBean>','additionComponents'),
('OrderConfirmInfoBean','MyAddressBean','address'),
('OrderConfirmInfoBean','String','areaName'),
('OrderConfirmInfoBean','boolean','canBuy'),
('OrderConfirmInfoBean','boolean','canRefund'),
('OrderConfirmInfoBean','List<DeliverBean>','deliver'),
('OrderConfirmInfoBean','OrderDiscount','discount'),
('OrderConfirmInfoBean','GiftCardInfo','giftCard'),
('OrderConfirmInfoBean','int','idCardCount'),
('OrderConfirmInfoBean','boolean','needIdCardV2'),
('OrderConfirmInfoBean','boolean','noTicketCompensateSwitch'),
('OrderConfirmInfoBean','String','notice'),
('OrderConfirmInfoBean','String','noticeSchema'),
('OrderConfirmInfoBean','boolean','onlyPlusMemberCanBuy');

INSERT INTO `_rev_dto_field_catalog` (`dto_class`, `java_type`, `field_name`) VALUES
('OrderConfirmInfoBean','List<OrderGroupProduct>','orderGroupProducts'),
('OrderConfirmInfoBean','JSONObject','plusMember'),
('OrderConfirmInfoBean','PostageBean','postage'),
('OrderConfirmInfoBean','String','promiseDeliverNotice'),
('OrderConfirmInfoBean','String','refundTip'),
('OrderConfirmInfoBean','String','serviceChargeDesc'),
('OrderConfirmInfoBean','boolean','serviceChargeOpen'),
('OrderConfirmInfoBean','String','serviceChargeTitle'),
('OrderConfirmInfoBean','List<JSONObject>','speedPacks'),
('OrderConfirmInfoBean','JSONObject','speedPacksBalance'),
('OrderConfirmInfoBean','TicketCategory','ticketCategory'),
('OrderConfirmInfoBean','Venue','venue'),
('OrderConfirmInfoRequestBody','BizParam','bizParam'),
('OrderConfirmInfoRequestBody','List<Goods>','goods'),
('OrderConfirmInfoRequestBody','int','orderType'),
('OrderDesc','String','desc'),
('OrderDesc','String','title'),
('OrderDiscount','Campaign','campaign'),
('OrderDiscount','boolean','canBuy'),
('OrderDiscount','boolean','canUseCoupon'),
('OrderDiscount','List<Coupon>','coupons'),
('OrderDiscount','List<Coupon>','disableCoupons'),
('OrderDiscount','String','errorHint'),
('OrderDiscount','String','desc'),
('OrderDiscount','double','discountAmount'),
('OrderDiscount','long','id'),
('OrderDiscount','String','title'),
('OrderEvent','long','addTime'),
('OrderEvent','int','currentStatus'),
('OrderEvent','int','id'),
('OrderEvent','int','operatorId'),
('OrderEvent','int','operatorType'),
('OrderEvent','int','orderId'),
('OrderEvent','int','originStatus'),
('OrderEvent','String','supplement'),
('OrderEvent','long','updateTime'),
('OrderGroupProduct','ActivityBean','activity'),
('OrderGroupProduct','ActivityEventBean','activityEvent'),
('OrderGroupProduct','int','areaTicketType'),
('OrderGroupProduct','String','deliverNotice'),
('OrderGroupProduct','boolean','hasSeat'),
('OrderGroupProduct','int','noTicketCompensateAmount'),
('OrderGroupProduct','int','noTicketOnsiteCompensateAmount'),
('OrderGroupProduct','List<JSONObject>','pinTuanCampaignItems'),
('OrderGroupProduct','String','shopNotice'),
('OrderGroupProduct','List<TicketGroup>','ticketGroups'),
('OrderGroupProduct','Venue','venue'),
('OrderGroupProduct','String','yoho'),
('OrderItem','int','businessOrderId'),
('OrderItem','int','businessOrderType'),
('OrderItem','List<OrderButtonBean>','buttons'),
('OrderItem','List<String>','descs'),
('OrderItem','String','orderAmountDesc'),
('OrderItem','int','orderId'),
('OrderItem','String','orderStatusDesc'),
('OrderItem','String','orderTypeDesc'),
('OrderItem','String','orderTypeDescRGB'),
('OrderItem','String','poster'),
('OrderItem','List<Spec>','spec'),
('OrderItem','String','title'),
('OrderItem','String','name'),
('OrderItem','String','value'),
('OrderPaymentInfo','List<CommonInstallment>','huabeiInstallment'),
('OrderPaymentInfo','Map<String, String>','paymentTips'),
('OrderPaymentInfo','List<Integer>','paymentTypes'),
('OrderTypeBean','String','name'),
('OrderTypeBean','int','type'),
('OrganizerBean','List<ActivityBean>','activities'),
('OrganizerBean','int','activityEventNum'),
('OrganizerBean','int','activityNum'),
('OrganizerBean','List<Integer>','categoryIds'),
('OrganizerBean','int','fansNum'),
('OrganizerBean','String','headImg'),
('OrganizerBean','String','logo'),
('OrganizerBean','String','name'),
('OrganizerBean','int','shopId'),
('OrganizerBean','String','slogan'),
('OrganizerBean','String','summary'),
('Pagination','List<T>','data'),
('Pagination','EmptyData','emptyData'),
('Pagination','boolean','hasMore'),
('Pagination','boolean','hasTopData'),
('Pagination','String','nextPageKey'),
('Pagination','int','numPlusRelated'),
('Pagination','int','pageIndex'),
('Pagination','int','pageSize'),
('Pagination','int','total'),
('Pagination','int','totalNum'),
('Pagination','int','totalPage'),
('PayInfo','int','leftTime'),
('PayInfo','double','payAmount'),
('PayInfo','List<DetailInfo>','detailInfo'),
('PayInfo','PayUnifiedOrderInfo','payUnifiedOrderInfo'),
('PayInfo','Tip','tip'),
('PayInfo','PayReturnPage','payReturnPage'),
('PayInfo','List<Channel>','channels'),
('PayInfo','String','friendlyTips'),
('PayInfo','String','title'),
('PayInfo','String','value'),
('PayInfo','String','content'),
('PayInfo','String','schema'),
('PayInfo','boolean','continuousSeat'),
('PayInfo','int','count'),
('PayInfo','long','orderId'),
('PayInfo','String','productId'),
('PayInfo','String','productName'),
('PayInfo','Double','balance'),
('PayInfo','CreditPay','creditPay'),
('PayInfo','List<CommonInstallment>','huabeiInstallment'),
('PayInfo','List<CommonInstallment>','lekaInstallment'),
('PayInfo','PayAfterUseInfo','payAfterUseInfo'),
('PayInfo','int','type'),
('PayInfo','int','clientType'),
('PayInfo','String','failPage'),
('PayInfo','String','successPage'),
('PayInfo','String','desc'),
('PaySuccessDetail','Integer','activityId'),
('PaySuccessDetail','String','activityName'),
('PaySuccessDetail','PopBoxBean','popBox'),
('PinTuanCampaignItem','String','name'),
('PinTuanCampaignItem','TicketGroup','pinTuanCampaignItem'),
('PinTuanCampaignMap','HashMap<Integer, List<PinTuanCampaignItem>>','pinTuanCampaignItems'),
('PinTuanInfo','int','activityId'),
('PinTuanInfo','long','addTime'),
('PinTuanInfo','long','effectiveFrom'),
('PinTuanInfo','long','effectiveTo'),
('PinTuanInfo','int','id'),
('PinTuanInfo','int','isOpen'),
('PinTuanInfo','double','lowPrice'),
('PinTuanInfo','String','pinTuanTag'),
('PinTuanInfo','int','shopId'),
('PinTuanInfo','String','slogan'),
('PinTuanInfo','int','stock'),
('PinTuanInfo','int','succeedNum'),
('PinTuanInfo','long','updateTime'),
('PinTuanOrderRequestBody','int','count'),
('PinTuanOrderRequestBody','int','pinTuanCampaignItemId'),
('PinTuanOrderRequestBody','Integer','pinTuanId'),
('PnConfig','ActivityIconsBean','activityIcons'),
('PnConfig','boolean','balancePay'),
('PnConfig','List<CategoriesBean>','categories'),
('PnConfig','String','contactPhone'),
('PnConfig','String','expressAddress'),
('PnConfig','String','expressName'),
('PnConfig','String','expressPhone'),
('PnConfig','String','imgDomain'),
('PnConfig','String','imgSslDomain'),
('PnConfig','boolean','isTimeToDacu'),
('PnConfig','String','launchScreen'),
('PnConfig','PaymentTipsBean','paymentTips');

INSERT INTO `_rev_dto_field_catalog` (`dto_class`, `java_type`, `field_name`) VALUES
('PnConfig','List<Integer>','paymentTypes'),
('PnConfig','List<Integer>','receiveTypes'),
('PnConfig','List<SearchNavigationBean>','searchNavigationV2'),
('PnConfig','String','selfGetAddress'),
('PnConfig','String','selfGetTime'),
('PnConfig','boolean','smsCaptchaOn'),
('PnConfig','int','ticketEnoughCount'),
('PnConfig','String','dacu'),
('PnConfig','String','value7'),
('PnConfig','String','name'),
('PnConfig','List<OptionsBean>','options'),
('PnConfig','String','type'),
('PnConfig','String','value'),
('PnConfig','long','addTime'),
('PnConfig','String','id'),
('PnConfig','boolean','isSelect'),
('PnConfig','int','status'),
('PnConfig','List<SubCategoryTagsBean>','subCategoryTags'),
('PnConfig','long','updateTime'),
('PnConfig','String','subCategoryId'),
('PopBoxBean','String','icon'),
('PopBoxBean','String','iconSchema'),
('PopBoxBean','Integer','id'),
('PopBoxBean','String','key'),
('PopBoxBean','String','type'),
('PopBoxBean','String','url'),
('PostageBean','double','postage'),
('PostageBean','String','postageDesc'),
('PostageBean','double','postageOrigin'),
('ProductBean','String','desc'),
('ProductBean','int','id'),
('ProductBean','Double','originPrice'),
('ProductBean','String','poster'),
('ProductBean','String','productTip'),
('ProductBean','Double','salePrice'),
('ProductBean','String','scheme'),
('ProductBean','String','title'),
('ProductBean','List<TopicWithColorBean>','topicList'),
('ProductBottomItemWithStateWrapperBean','List<Integer>','indexList'),
('ProductBottomItemWithStateWrapperBean','boolean','showIndexNumState'),
('ProductBottomItemWithStateWrapperBean','int','totalNum'),
('ProductGroupBean','List<ProductBean>','productList'),
('ProductGroupBean','String','productTypeName'),
('ProductWithDisplayWrapperBean','ProductBean','productBean'),
('ProductWithDisplayWrapperBean','boolean','productDisplay'),
('ProductWithMultiTypeBean','ProductBottomItemWithStateWrapperBean','productBottomItemWithStateWrapperBean'),
('ProductWithMultiTypeBean','ProductWithDisplayWrapperBean','productWithDisplayWrapperBean'),
('ProductWithMultiTypeBean','int','type'),
('Provider','long','addTime'),
('Provider','String','avatar'),
('Provider','int','level'),
('Provider','String','mobileNo'),
('Provider','double','score'),
('Provider','int','status'),
('Provider','long','updateTime'),
('Provider','int','userId'),
('Provider','String','userName'),
('ProxyBuyCompensationInfo','boolean','open'),
('ProxyBuyInfo','ProxyBuyCompensationInfo','proxyBuyCompensationInfo'),
('ProxyBuyInfo','double','proxyBuyFee'),
('ProxyBuyInfo','String','proxyBuyRuleDesc'),
('ProxyBuyInfo','List<ProxyBuyTimeInfo>','proxyBuyTimeInfo'),
('ProxyBuyInfo','ProxyBuyTurnInfo','proxyBuyTurnInfo'),
('ProxyBuyTimeInfo','int','endValue'),
('ProxyBuyTimeInfo','String','endValueDesc'),
('ProxyBuyTurnInfo','String','timeStr'),
('ProxyBuyTurnInfo','String','title'),
('RecommendBean','Object','recommendContent'),
('RecommendBean','int','recommendType'),
('RecommendUser','int','followerNum'),
('RecommendUser','String','recommendReason'),
('RecommendUser','UserBean','user'),
('Reply','long','addTime'),
('Reply','String','content'),
('Reply','List<Reply>','data'),
('Reply','int','id'),
('Reply','Boolean','isTop'),
('Reply','int','pageIndex'),
('Reply','int','pageSize'),
('Reply','int','replyToUserId'),
('Reply','String','replyToUserName'),
('Reply','int','reviewId'),
('Reply','int','totalNum'),
('Reply','UserBean','user'),
('Reply','int','userId'),
('ReplySendRequestBody','String','content'),
('ReplySendRequestBody','int','replyToUserId'),
('ReplySendRequestBody','int','subjectId'),
('ReplySendRequestBody','int','type'),
('Review','ActivityBean','activity'),
('Review','int','activityId'),
('Review','long','addTime'),
('Review','String','content'),
('Review','List<Review>','data'),
('Review','int','id'),
('Review','List<String>','images'),
('Review','boolean','liked'),
('Review','int','likes'),
('Review','int','numPlusRelated'),
('Review','int','pageIndex'),
('Review','int','pageSize'),
('Review','int','rank'),
('Review','int','replyCount'),
('Review','int','status'),
('Review','int','totalNum'),
('Review','long','updateTime'),
('Review','UserBean','user'),
('Review','int','userId'),
('Review','List<JSONObject>','videos'),
('ReviewDetailBean','List<FeedBean>','relatedReviews'),
('ReviewDetailBean','FeedBean','review'),
('ReviewToDo','ActivityBean','activity'),
('ReviewToDo','int','activityEventId'),
('ReviewToDo','int','activityId'),
('ReviewToDo','long','addTime'),
('ReviewToDo','List<ReviewToDo>','data'),
('ReviewToDo','long','eventTime'),
('ReviewToDo','int','id'),
('ReviewToDo','int','pageIndex'),
('ReviewToDo','int','pageSize'),
('ReviewToDo','int','status'),
('ReviewToDo','int','totalNum'),
('ReviewToDo','long','updateTime'),
('ReviewToDo','int','userId'),
('SchemaKeywordBean','String','keyword'),
('SchemaKeywordBean','String','recommendWord'),
('SchemaKeywordBean','String','schema'),
('SchemaKeywordRankBean','String','keyword'),
('SchemaKeywordRankBean','int','logId'),
('SchemaKeywordRankBean','int','pnLocalRank'),
('SchemaKeywordRankBean','String','schema'),
('SearchResultBannerBean','int','activityType'),
('SearchResultBannerBean','String','schema'),
('SearchResultBannerBean','int','id'),
('SearchResultBannerBean','String','poster'),
('SearchResultShopBean','int','activityEventNum'),
('SearchResultShopBean','int','activityNum'),
('SearchResultShopBean','int','activityType'),
('SearchResultShopBean','int','id'),
('SearchResultShopBean','String','img'),
('SearchResultShopBean','String','name'),
('SearchResultShopBean','String','promoImg'),
('SearchSuggestBean','int','id'),
('SearchSuggestBean','String','name'),
('SearchSuggestBean','String','type'),
('SelectionBean','boolean','isSelected'),
('SendTweetRequestBody','String','content'),
('SendTweetRequestBody','int','groupId'),
('SendTweetRequestBody','List<String>','images'),
('SendTweetRequestBody','List<Integer>','relativeActivityIds');

INSERT INTO `_rev_dto_field_catalog` (`dto_class`, `java_type`, `field_name`) VALUES
('SendTweetRequestBody','List<String>','tags'),
('SendTweetRequestBody','JSONArray','videos'),
('ShareModel','String','title'),
('ShareModel','String','desc'),
('ShareModel','String','imageURL'),
('ShareModel','String','shareImageUrl'),
('ShareModel','String','url'),
('ShareModel','String','mpUrl'),
('ShareModel','String','mpImageUrl'),
('ShareParams','String','desc'),
('ShareParams','String','imgUrl'),
('ShareParams','String','link'),
('ShareParams','String','mpLink'),
('ShareParams','String','title'),
('Shop','long','addTime'),
('Shop','int','autoApprove'),
('Shop','int','cityId'),
('Shop','int','id'),
('Shop','String','img'),
('Shop','String','mobile'),
('Shop','String','name'),
('Shop','int','owner'),
('Shop','int','status'),
('Shop','int','totalScore'),
('Shop','int','type'),
('Shop','long','updateTime'),
('ShopInfo','Integer','shopId'),
('ShopInfo','String','shopLisenceUrl'),
('ShopInfo','String','shopName'),
('ShopInfo','boolean','verified'),
('ShowDetailBean','ActivityBean','activity'),
('ShowDetailBean','List<ActivityLineupBean>','activityLineup'),
('ShowDetailBean','String','activityPosterIcon'),
('ShowDetailBean','String','adTips'),
('ShowDetailBean','Pagination<FeedBean>','articles'),
('ShowDetailBean','Pagination<FeedBean>','comments'),
('ShowDetailBean','String','creditPayDesc'),
('ShowDetailBean','String','distance'),
('ShowDetailBean','boolean','favorite'),
('ShowDetailBean','List<ActivityGalleryBean>','galleries'),
('ShowDetailBean','boolean','hasTicketNotice'),
('ShowDetailBean','int','numGalleries'),
('ShowDetailBean','int','numPlusRelatedArticle'),
('ShowDetailBean','int','numPlusRelatedComment'),
('ShowDetailBean','boolean','openSaleNotice'),
('ShowDetailBean','boolean','openSelectionSeatV2'),
('ShowDetailBean','PinTuanInfo','pinTuanInfo'),
('ShowDetailBean','String','redirectUrl'),
('ShowDetailBean','List<ActivityBean>','relativeActivities'),
('ShowDetailBean','int','reviewId'),
('ShowDetailBean','boolean','reviewed'),
('ShowDetailBean','String','seekTicketUrl'),
('ShowDetailBean','boolean','travelTag'),
('ShowFeedPagination','ActivityBean','activity'),
('ShowFeedPagination','List<FeedBean>','data'),
('ShowFeedPagination','boolean','hasMore'),
('ShowFeedPagination','String','nextPageKey'),
('ShowFeedPagination','int','numPlusRelated'),
('ShowFeedPagination','int','pageIndex'),
('ShowFeedPagination','int','pageSize'),
('ShowFeedPagination','int','totalNum'),
('ShowTip','List<TipDetailBean>','detail'),
('ShowTip','String','desc'),
('ShowTip','String','title'),
('ShowTip','String','type'),
('ShowTips','TipsBizData','bizData'),
('ShowTips','String','content'),
('ShowTips','String','desc'),
('ShowTips','String','displayType'),
('ShowTips','String','schema'),
('ShowTips','String','type'),
('ShowTips','String','refundContent'),
('ShowTips','String','refundPolicyTips'),
('ShowTips','List<? extends ActivityRefundRuleBean>','refundRuleList'),
('ShowTips','String','noTicketCompensateContent'),
('ShowTips','String','noTicketCompensateTips'),
('ShowTips','List<NoTicketCompensateStandardVO>','noTicketCompensateStandardVOList'),
('ShowTips','String','title'),
('ShowTips','String','startTime'),
('ShowTips','String','titleTimeDesc'),
('ShowTips','String','proxyBuyDeclare'),
('ShowTips','List<String>','info'),
('ShowTips','String','noTicketCompensateMatchStr'),
('ShowTips','String','noTicketCompensateModeStr'),
('SkuDetailBean','String','announcementUrl'),
('SkuDetailBean','List<ActivityEventBean>','events'),
('SkuDetailBean','ProductBean','product'),
('SpeedPackChoice','int','amount'),
('SpeedPackChoice','String','desc'),
('SpeedPackChoice','boolean','recommend'),
('SpeedPackChoice','double','successRate'),
('SplitOrderContext','String','splitOrderDesc'),
('SplitOrderContext','double','splitOrderFee'),
('StationBean','String','activityMarketingImageUrl'),
('StationBean','String','activityPosterIcon'),
('StationBean','ActorQrBean','actorInfo'),
('StationBean','String','avatar'),
('StationBean','Boolean','followed'),
('StationBean','int','followings'),
('StationBean','int','goodsCount'),
('StationBean','int','id'),
('StationBean','int','ipType'),
('StationBean','String','name'),
('StationBean','List<Poster>','posters'),
('StationBean','List<Long>','relatedActivityIdList'),
('StationBean','List<Tour>','tours'),
('StationBean','List<Activity>','activities'),
('StationBean','long','addTime'),
('StationBean','List<String>','headPosters'),
('StationBean','String','poster'),
('StationBean','String','status'),
('StationBean','long','updateTime'),
('StationBean','int','year'),
('StationBean','long','activityId'),
('StationBean','String','activityInTourTagType'),
('StationBean','int','cityId'),
('StationBean','String','cityName'),
('StationBean','long','endTime'),
('StationBean','Boolean','groupBuy'),
('StationBean','double','lowPrice'),
('StationBean','String','marketingTag'),
('StationBean','Integer','marketingTagType'),
('StationBean','Boolean','showTimePending'),
('StationBean','String','showTimePendingText'),
('StationBean','long','startTime'),
('StationBean','String','timeRange'),
('StationBean','String','type'),
('StationBean','String','url'),
('StatisticBean','int','actorFollowing'),
('StatisticBean','double','balance'),
('StatisticBean','int','couponNum'),
('StatisticBean','int','followerNum'),
('StatisticBean','int','followingNum'),
('StatisticBean','int','gettingTicketsNum'),
('StatisticBean','int','giftCardNum'),
('StatisticBean','int','msgNum'),
('StatisticBean','int','payingNum'),
('StatisticBean','int','reviewNum'),
('StatisticBean','int','reviewTodoNum'),
('StatisticBean','int','waitingForTicketsNum'),
('StatisticBean','int','wantWatchNum'),
('TPLoginBean','DataBean','data'),
('TPLoginBean','boolean','success'),
('TPLoginBean','String','accessToken'),
('TPLoginBean','boolean','isBound'),
('TPLoginBean','boolean','newUser'),
('TPLoginBean','String','unionId'),
('TPLoginBean','UserProfile','user'),
('TagBean','int','customItemType'),
('TagBean','int','id');

INSERT INTO `_rev_dto_field_catalog` (`dto_class`, `java_type`, `field_name`) VALUES
('TagBean','String','name'),
('TeamFeedBean','List<DescBean>','desc'),
('TeamFeedBean','long','happenTime'),
('TeamFeedBean','String','poster'),
('TeamFeedBean','String','schema'),
('TeamFeedBean','int','subjectId'),
('TeamFeedBean','String','title'),
('TeamFeedBean','int','type'),
('TeamFeedBean','UserBean','user'),
('TeamFeedBean','String','avatar'),
('TeamFeedBean','String','bio'),
('TeamFeedBean','double','gender'),
('TeamFeedBean','double','level'),
('TeamFeedBean','double','score'),
('TeamFeedBean','double','status'),
('TeamFeedBean','int','userId'),
('TeamFeedBean','String','userName'),
('TeamFeedBean','String','key'),
('TeamFeedBean','String','value'),
('Test','List<DataBean>','data'),
('Test','int','pageIndex'),
('Test','int','pageSize'),
('Test','int','totalNum'),
('Test','int','activityEventId'),
('Test','int','actorId'),
('Test','long','addTime'),
('Test','int','categoryId'),
('Test','int','cityId'),
('Test','int','discountRate'),
('Test','String','duration'),
('Test','List<EventsBean>','events'),
('Test','int','highPrice'),
('Test','int','id'),
('Test','int','lowPrice'),
('Test','String','name'),
('Test','long','openTime'),
('Test','int','originPrice'),
('Test','String','poster'),
('Test','int','publishStatus'),
('Test','int','salePrice'),
('Test','int','status'),
('Test','List<TicketCategoriesBean>','ticketCategories'),
('Test','int','ticketCategoryId'),
('Test','int','ticketGroupId'),
('Test','String','timeRange'),
('Test','long','updateTime'),
('Test','int','venueId'),
('Test','String','venueName'),
('Test','int','activityId'),
('Test','long','end'),
('Test','String','specification'),
('Test','long','start'),
('Test','String','desc'),
('TicketAreaBean','String','areaName'),
('TicketAreaBean','int','height'),
('TicketAreaBean','Integer','id'),
('TicketAreaBean','boolean','isSelect'),
('TicketAreaBean','boolean','soldOut'),
('TicketAreaBean','List<TicketSeatBean>','ticketSeats'),
('TicketAreaBean','int','width'),
('TicketCategory','int','activityEventId'),
('TicketCategory','long','addTime'),
('TicketCategory','List<TicketAreaBean>','areas'),
('TicketCategory','List<Integer>','availableNumbers'),
('TicketCategory','String','desc'),
('TicketCategory','boolean','discountFlag'),
('TicketCategory','double','discountRate'),
('TicketCategory','boolean','hasTicket'),
('TicketCategory','String','icon'),
('TicketCategory','int','id'),
('TicketCategory','boolean','isETicket'),
('TicketCategory','double','lowPrice'),
('TicketCategory','boolean','needSpeedPackBuy'),
('TicketCategory','double','originPrice'),
('TicketCategory','boolean','priceLowest'),
('TicketCategory','ProxyBuyInfo','proxyBuyInfo'),
('TicketCategory','String','seatColor'),
('TicketCategory','boolean','showTicketList'),
('TicketCategory','String','specification'),
('TicketCategory','int','status'),
('TicketCategory','int','ticketsNum'),
('TicketCategory','int','unitTicketNum'),
('TicketCategory','long','updateTime'),
('TicketGroup','AdditionBean','addition'),
('TicketGroup','int','count'),
('TicketGroup','String','fullViewImage'),
('TicketGroup','int','id'),
('TicketGroup','boolean','isContinuousSeat'),
('TicketGroup','boolean','isETicket'),
('TicketGroup','boolean','isRecommended'),
('TicketGroup','double','originPrice'),
('TicketGroup','Integer','pinTuanCampaignItemId'),
('TicketGroup','int','promiseDeliverDay'),
('TicketGroup','String','promiseDeliverNotice'),
('TicketGroup','int','rowIndex'),
('TicketGroup','double','salePrice'),
('TicketGroup','int','saler'),
('TicketGroup','int','saling'),
('TicketGroup','String','seatColor'),
('TicketGroup','int','sellingAmount'),
('TicketGroup','int','sold'),
('TicketGroup','String','specification'),
('TicketGroup','double','splitOrderFee'),
('TicketGroup','Integer','succeedNum'),
('TicketGroup','int','ticketAreaId'),
('TicketGroup','int','ticketCategoryId'),
('TicketGroup','String','ticketCategorySpecification'),
('TicketGroup','int','ticketGroupId'),
('TicketGroup','int','ticketRowId'),
('TicketGroup','int','unitTicketNum'),
('TicketGroup','String','salerName'),
('TicketGroup','String','desc'),
('TicketGroup','String','ticketCategoryDesc'),
('TicketGroup','String','areaName'),
('TicketGroup','List<Integer>','availableNumbers'),
('TicketGroup','Boolean','speedExpress'),
('TicketGroup','String','saleTag'),
('TicketGroup','boolean','isIsContinuousSeat'),
('TicketGroup','boolean','isIsDivide'),
('TicketGroup','int','numMax'),
('TicketGroup','int','numMin'),
('TicketGroup','int','sellLimit'),
('TicketPackage','List<TicketGroup>','randomTicket'),
('TicketPackage','TicketGroup','ticket'),
('TicketPackage','int','type'),
('TicketPackagePagination','int','activityEventId'),
('TicketPackagePagination','long','addTime'),
('TicketPackagePagination','int','creditPay'),
('TicketPackagePagination','String','desc'),
('TicketPackagePagination','int','id'),
('TicketPackagePagination','double','originPrice'),
('TicketPackagePagination','String','specification'),
('TicketPackagePagination','int','status'),
('TicketPackagePagination','long','updateTime'),
('TicketPagination','int','activityEventId'),
('TicketPagination','long','addTime'),
('TicketPagination','int','creditPay'),
('TicketPagination','String','desc'),
('TicketPagination','int','id'),
('TicketPagination','double','originPrice'),
('TicketPagination','String','specification'),
('TicketPagination','int','status'),
('TicketPagination','long','updateTime'),
('TicketSeatBean','String','col'),
('TicketSeatBean','String','desc'),
('TicketSeatBean','int','id'),
('TicketSeatBean','String','row'),
('TicketSeatBean','int','status'),
('TicketSeatBean','TicketGroup','ticket'),
('TicketSeatBean','int','x');

INSERT INTO `_rev_dto_field_catalog` (`dto_class`, `java_type`, `field_name`) VALUES
('TicketSeatBean','int','y'),
('TicketSeatBean','int','z'),
('TicketTableBean','List<TicketAreaBean>','areaTickets'),
('TicketTableBean','boolean','hasFullViewImage'),
('TicketTableBean','String','iconChecked'),
('TicketTableBean','String','iconUnable'),
('TicketTableBean','IconTemplateBean','iconUnableTemplate'),
('TicketTableBean','Map<String, Map<String, Double>>','packageDiscount'),
('TicketTableBean','List<TicketCategory>','ticketCategories'),
('TicketTableBean','Map<String, IconTemplateBean>','ticketCategoriesIcon'),
('TicketTableBean','long','addTime'),
('TicketTableBean','String','iconTicketCategory'),
('TicketTableBean','String','iconUnChecked'),
('TicketTableBean','int','id'),
('TicketTableBean','int','status'),
('TicketTableBean','int','type'),
('TicketTableBean','long','updateTime'),
('TimeLine','ActivityBean','activity'),
('TimeLine','long','addTime'),
('TimeLine','CharSequence','friendlyTime'),
('TimeLine','Review','review'),
('TimeLine','int','wantWatchNum'),
('TimeLinePagination','int','reviewNum'),
('TimeLinePagination','int','unWatchNum'),
('Tip','String','content'),
('Tip','String','schema'),
('ToWatchBean','long','activityEventDate'),
('ToWatchBean','int','activityEventId'),
('ToWatchBean','int','activityId'),
('ToWatchBean','String','activityName'),
('ToWatchBean','int','orderId'),
('ToWatchBean','String','shortName'),
('ToWatchBean','int','unifiedOrderId'),
('ToWatchBean','Venue','venue'),
('TopicBean','int','activityId'),
('TopicBean','long','addTime'),
('TopicBean','String','backgroundImg'),
('TopicBean','String','desc'),
('TopicBean','int','feedNum'),
('TopicBean','int','id'),
('TopicBean','String','name'),
('TopicBean','int','status'),
('TopicBean','int','topFeedId'),
('TopicBean','int','topFeedType'),
('TopicBean','int','type'),
('TopicBean','long','updateTime'),
('TopicWithColorBean','String','backgroundColor'),
('TopicWithColorBean','String','name'),
('TopicWithColorBean','String','textColor'),
('TransferPagination','List<TransformFeedBean>','data'),
('TransferPagination','boolean','hasMore'),
('TransferPagination','String','nextPageKey'),
('TransferTicket','int','activityEventId'),
('TransferTicket','int','activityId'),
('TransferTicket','int','amount'),
('TransferTicket','String','content'),
('TransferTicket','double','price'),
('TransferTicket','int','ticketCategoryId'),
('TransformBean','String','activityName'),
('TransformBean','String','address'),
('TransformBean','String','categoryDesc'),
('TransformBean','double','count'),
('TransformBean','String','eventSpecification'),
('TransformBean','boolean','isCheck'),
('TransformBean','String','poster'),
('TransformBean','double','price'),
('TransformBean','int','transformId'),
('TransformFeedBean','ActivityBean','activity'),
('TransformFeedBean','ActivityEventBean','activityEvent'),
('TransformFeedBean','int','activityEventId'),
('TransformFeedBean','int','activityId'),
('TransformFeedBean','long','addTime'),
('TransformFeedBean','int','amount'),
('TransformFeedBean','String','content'),
('TransformFeedBean','int','id'),
('TransformFeedBean','boolean','liked'),
('TransformFeedBean','int','likes'),
('TransformFeedBean','double','price'),
('TransformFeedBean','int','replyCount'),
('TransformFeedBean','int','status'),
('TransformFeedBean','TicketCategory','ticketCategory'),
('TransformFeedBean','int','ticketCategoryId'),
('TransformFeedBean','long','updateTime'),
('TransformFeedBean','UserBean','user'),
('TransformFeedBean','int','userId'),
('TransformFeedBean','int','venueCityId'),
('Tweet','long','addTime'),
('Tweet','int','clickNum'),
('Tweet','String','content'),
('Tweet','int','id'),
('Tweet','List<String>','images'),
('Tweet','boolean','liked'),
('Tweet','int','likes'),
('Tweet','int','replyCount'),
('Tweet','int','status'),
('Tweet','long','updateTime'),
('Tweet','UserBean','user'),
('Tweet','int','userId'),
('Tweet','List<JSONObject>','videos'),
('Tweet','int','views'),
('TweetVideo','String','duration'),
('TweetVideo','String','poster'),
('TweetVideo','String','url'),
('UpdateInfo','String','changelog'),
('UpdateInfo','String','installUrl'),
('UpdateInfo','Boolean','isForceUpdate'),
('UpdateInfo','Boolean','isIgnorable'),
('UpdateInfo','int','tipsIntervalDays'),
('UpdateInfo','String','type'),
('UpdateInfo','int','versionCode'),
('UpdateInfo','String','versionName'),
('Urge','String','desc'),
('Urge','String','title'),
('UserMessage','long','addTime'),
('UserMessage','String','content'),
('UserMessage','UserBean','fromUser'),
('UserMessage','int','fromUserId'),
('UserMessage','int','id'),
('UserMessage','int','status'),
('UserMessage','String','title'),
('UserMessage','int','type'),
('UserMessage','long','updateTime'),
('UserMessage','String','url'),
('UserMessage','int','userId'),
('UserMessageSimple','long','addTime'),
('UserMessageSimple','String','content'),
('UserMessageSimple','int','type'),
('UserMessageSimple','int','unReadCount'),
('UserMsgCount','int','campaignMsgUnreadCount'),
('UserMsgCount','int','interactMsgUnreadCount'),
('UserMsgCount','int','kefuMsgUnreadCount'),
('UserMsgCount','int','likeAndFollowUnreadCount'),
('UserMsgCount','int','replyAndAtUnreadCount'),
('UserMsgCount','int','systemMsgUnreadCount'),
('Venue','long','addTime'),
('Venue','String','address'),
('Venue','String','avatar'),
('Venue','int','cityId'),
('Venue','int','id'),
('Venue','double','latitude'),
('Venue','double','longitude'),
('Venue','String','name'),
('Venue','String','parking'),
('Venue','String','phones'),
('Venue','int','regionId'),
('Venue','String','transportation'),
('Venue','long','updateTime'),
('VideoChannelBean','TagBean','tag'),
('VideoChannelBean','List<Tweet>','videos'),
('VideoDetailBean','long','addTime');

INSERT INTO `_rev_dto_field_catalog` (`dto_class`, `java_type`, `field_name`) VALUES
('VideoDetailBean','String','content'),
('VideoDetailBean','CommunityBean','group'),
('VideoDetailBean','int','id'),
('VideoDetailBean','boolean','isLiked'),
('VideoDetailBean','int','likes'),
('VideoDetailBean','int','rank'),
('VideoDetailBean','List<? extends RelativeActivityBean>','relatedActivities'),
('VideoDetailBean','int','replyCount'),
('VideoDetailBean','int','shares'),
('VideoDetailBean','List<? extends TagBean>','tags'),
('VideoDetailBean','int','type'),
('VideoDetailBean','UserBean','user'),
('VideoDetailBean','VideoItemBean','video'),
('VideoDetailBean','int','views'),
('VideoItemBean','String','duration'),
('VideoItemBean','int','height'),
('VideoItemBean','String','poster'),
('VideoItemBean','String','title'),
('VideoItemBean','String','url'),
('VideoItemBean','int','width'),
('launchScreenBean','Integer','id'),
('launchScreenBean','Long','updateTime'),
('launchScreenBean','Long','addTime'),
('launchScreenBean','Integer','cityId'),
('launchScreenBean','String','poster'),
('launchScreenBean','String','video'),
('launchScreenBean','String','schema'),
('launchScreenBean','int','duration'),
('launchScreenBean','Long','fromTime'),
('launchScreenBean','Long','toTime'),
('launchScreenBean','Integer','status'),
('launchScreenBean','String','openScreenType'),
('ActivityDetailType','int','code'),
('ActivityDetailType','String','description'),
('FeedOpType','String','desc'),
('FeedOpType','int','value'),
('FeedType','String','key'),
('FeedType','int','value'),
('OrderStatus','int','code'),
('OrderStatus','String','description'),
('SaleStatus','int','code'),
('SaleStatus','String','description'),
('SearchType','int','code'),
('SearchType','String','description'),
('TicketType','int','code'),
('TicketType','String','description'),
('TransferStatus','int','code'),
('TransferStatus','String','desc'),
('UserMsgType','int','code'),
('UserMsgType','String','desc'),
('AppLinkBean','String','schema'),
('AppLinkBean','String','uri'),
('CustomTagBean','boolean','customSelected'),
('CustomTagBean','float','scaleMagnification'),
('SearchOption','Long','endTime'),
('SearchOption','String','key'),
('SearchOption','String','keyword'),
('SearchOption','String','seatMap'),
('SearchOption','String','sort'),
('SearchOption','Long','startTime'),
('SearchOption','String','time'),
('UserBean','long','addTime'),
('UserBean','String','avatar'),
('UserBean','String','badgeIcon'),
('UserBean','String','bio'),
('UserBean','long','birthday'),
('UserBean','boolean','blocked'),
('UserBean','boolean','followHim'),
('UserBean','int','followed'),
('UserBean','boolean','followedByHim'),
('UserBean','int','followerNum'),
('UserBean','int','followingNum'),
('UserBean','boolean','forceFollow'),
('UserBean','int','gender'),
('UserBean','boolean','isDouliUser'),
('UserBean','int','level'),
('UserBean','String','mobileNo'),
('UserBean','int','reviewLikedNum'),
('UserBean','double','score'),
('UserBean','int','shopType'),
('UserBean','int','status'),
('UserBean','long','updateTime'),
('UserBean','int','userId'),
('UserBean','String','userName'),
('UserBean','int','wantWatchNum'),
('UserBean','int','watchedNum');

INSERT INTO `_rev_endpoint_catalog` (`http_method`, `path`, `service_file`) VALUES
('GET','/v1/activities/{activityId}/events','ActivityService.java'),
('GET','/v1/activity/favorite/detail/{id}','FeedService.java'),
('POST','/v1/actorAggregation/detail','StationService.java'),
('POST','/v1/actorAggregation/ipGoods/list','GoodsService.java'),
('POST','/v1/ad/findClickDataByTerminalId','ADEventService.java'),
('GET','/v1/ad/findOaidCert','ADEventService.java'),
('GET','/v1/file/direct/uploadParams','UploadService.java'),
('GET','/v1/hotTours/home','HomeService.java'),
('POST','/v1/myaddress','AddressService.java'),
('GET','/v1/myaddress/district','AddressService.java'),
('GET','/v1/myaddress/district/current','AddressService.java'),
('GET','/v1/order/hasWaitTicketOrder','OrderService.java'),
('POST','/v1/track/advertisement/event','ADEventService.java'),
('DELETE','/v1/user/toWatch/{activityEventId}/cancel','UserService.java'),
('GET','/v2/actorAggregation/query','StationService.java'),
('GET','/v2/hotTours/home','HomeService.java'),
('GET','/v2/transform/{id}','FeedService.java'),
('GET','/v3/share/featuredVideoList','ShareService.java'),
('GET','/v3/share/feed','ShareService.java'),
('GET','/v3/share/topic','ShareService.java'),
('GET','/v3/share/userHomepage','ShareService.java'),
('GET','apps/latest/{id}','UpdateHelper.java'),
('GET','uri-schema-association.json','AppLinkHelper.java'),
('GET','v1/activities/byActor','ActivityService.java'),
('GET','v1/activities/discount','ActivityService.java'),
('GET','v1/activities/hot','ActivityService.java'),
('GET','v1/activities/hottest','SearchService.java'),
('GET','v1/activities/{activityId}/couponForActivity','ActivityService.java'),
('GET','v1/activities/{activityId}/relative','ActivityService.java'),
('GET','v1/activities/{id}/reviewed','ActivityService.java'),
('GET','v1/activities/{id}/stockTips','ActivityService.java'),
('GET','v1/activity/favorite/my','ActivityService.java'),
('DELETE','v1/activity/favorite/{id}','ActivityService.java'),
('POST','v1/activity/favorite/{id}','ActivityService.java'),
('GET','v1/activity/favorite/{id}?type=1','ActivityService.java'),
('GET','v1/activity/favorite/{id}?type=2','ActivityService.java'),
('GET','v1/activity/favorite/{id}?type=3','ActivityService.java'),
('GET','v1/activityGallery/{activityId}','ActivityService.java'),
('GET','v1/actor','LineUpService.java'),
('GET','v1/actor/{actorId}','LineUpService.java'),
('DELETE','v1/actorFollow/{actorId}','LineUpService.java'),
('GET','v1/actorFollow/{actorId}','UserService.java'),
('POST','v1/actorFollow/{actorId}','LineUpService.java'),
('GET','v1/articles','ArticleService.java'),
('GET','v1/articles/banners','ArticleService.java'),
('GET','v1/articles/{id}','ArticleService.java'),
('GET','v1/articlesSearch','ArticleService.java'),
('GET','v1/benefits/benefitIcons','UserService.java'),
('GET','v1/calendarActivities','ActivityService.java'),
('GET','v1/chat/user','ChatService.java'),
('GET','v1/chat/visitor','ChatService.java'),
('GET','v1/chatGroup/activityLinkedGroups','ActivityService.java'),
('GET','v1/chatGroup/feed','ChatService.java'),
('GET','v1/chatGroup/relatedGroups','ChatService.java'),
('GET','v1/cities','CityService.java'),
('GET','v1/cities/current','CityService.java'),
('GET','v1/cities/currentByGeo','CityService.java'),
('GET','v1/cities/hot','CityService.java'),
('GET','v1/countryArea/list','UserIdService.java'),
('GET','v1/coupons','CouponService.java'),
('POST','v1/coupons','CouponService.java'),
('GET','v1/creditPay/creditPayAmount','ChooseTicketService.java'),
('GET','v1/featuredVideos','FeedService.java'),
('GET','v1/featuredVideos/info','FeedService.java'),
('GET','v1/feed','FeedService.java'),
('GET','v1/feed/actor','LineUpService.java'),
('GET','v1/feed/banner','FeedService.java'),
('GET','v1/feed/getFeedRelativeActivity','FeedService.java'),
('GET','v1/feed/hasNewFeeds','FeedService.java'),
('GET','v1/feed/noRepeatHot','FeedService.java'),
('GET','v1/feed/payRecommendFeed','FeedService.java'),
('GET','v1/feed/recommendVideo','FeedService.java'),
('GET','v1/feed/search','FeedService.java'),
('GET','v1/feed/searchWithTypes','SearchService.java'),
('GET','v1/feed/topBanner','FeedService.java'),
('GET','v1/feed/{userId}','FeedService.java'),
('POST','v1/feedback','FeedbackService.java'),
('POST','v1/giftCards','GiftCardService.java'),
('GET','v1/group','FeedService.java'),
('GET','v1/group/follow','FeedService.java'),
('POST','v1/group/subscribe/{groupId}','FeedService.java'),
('POST','v1/group/unsubscribe/{groupId}','FeedService.java'),
('GET','v1/group/{id}','FeedService.java'),
('GET','v1/group/{id}/feed','FeedService.java'),
('GET','v1/groupCategory','FeedService.java'),
('GET','v1/home/floor/{floorId}','HomeService.java'),
('POST','v1/image','UploadService.java'),
('POST','v1/imei/record','UserService.java'),
('GET','v1/launchScreen','ConfigService.java'),
('GET','v1/messages','MessagesService.java'),
('GET','v1/messages/info','MessagesService.java'),
('POST','v1/messages/types/{types}/read','MessagesService.java'),
('POST','v1/messages/types/{type}/read','MessagesService.java'),
('POST','v1/messages/{id}/read','MessagesService.java'),
('GET','v1/moneyaccount','UserService.java'),
('GET','v1/moneyaccount/sequences','UserService.java'),
('GET','v1/myaddress','AddressService.java'),
('DELETE','v1/myaddress/{id}','AddressService.java'),
('GET','v1/order/discounts/allCampaign','OrderService.java'),
('GET','v1/order/discounts/display','OrderService.java'),
('GET','v1/order/postage','OrderService.java'),
('PUT','v1/order/{orderId}/refund','OrderService.java'),
('GET','v1/order/{orderId}/status','OrderService.java'),
('POST','v1/pay','OrderService.java'),
('GET','v1/pay/paymentTypes?ct=11','OrderService.java'),
('GET','v1/pinTuanCampaign/{pinTuanCampaignId}','ActivityService.java'),
('GET','v1/pinTuanCampaign/{pinTuanCampaignId}/ticketCategories','ChooseTicketService.java'),
('GET','v1/pinTuanCampaign/{pinTuanCampaignId}/tickets','ChooseTicketService.java'),
('POST','v1/pinTuanOrder/confirmWithLogin','OrderService.java'),
('POST','v1/pinTuanOrder/make','OrderService.java'),
('GET','v1/pinTuanOrder/myOrders','OrderService.java'),
('GET','v1/plusMember','GrabTicketOrderService.java'),
('POST','v1/plusMember/charge/makeOrder','OrderService.java'),
('GET','v1/popBox/home','PopboxService.java'),
('GET','v1/popBox/orderSuccess','PopboxService.java'),
('GET','v1/product/{productId}/chooseSku','ProductService.java'),
('GET','v1/reviews','ReviewService.java'),
('POST','v1/reviews','ReviewService.java'),
('GET','v1/reviews/my','ReviewService.java'),
('GET','v1/reviews/myReviews','ReviewService.java'),
('GET','v1/reviews/todo','ReviewService.java'),
('DELETE','v1/reviews/{id}','FeedService.java'),
('GET','v1/reviews/{id}/replies','ReviewService.java'),
('POST','v1/reviews/{id}/replies','ReviewService.java'),
('PUT','v1/reviews/{reviewId}','ReviewService.java'),
('POST','v1/reviews/{reviewId}/cancelLike','ReviewService.java'),
('POST','v1/reviews/{reviewId}/like','ReviewService.java'),
('POST','v1/share/log','FeedService.java'),
('POST','v1/shareReward/confirm','CampaignService.java'),
('PUT','v1/shortURL','FeedService.java'),
('GET','v1/snapUpOrder/chooseSpeedPacks','GrabTicketOrderService.java'),
('POST','v1/snapUpOrder/makeOrder','OrderService.java'),
('GET','v1/speedPack/balance','GrabTicketOrderService.java'),
('GET','v1/tag/getRecommendChoiceTag','FeedService.java'),
('GET','v1/tag/{id}','FeedService.java'),
('GET','v1/ticketGroup','SeatMapService.java'),
('GET','v1/topic','TopicService.java'),
('GET','v1/topic/hotTopics','TopicService.java'),
('GET','v1/topic/myTopics','TopicService.java'),
('GET','v1/topic/recommendTopics','FeedService.java'),
('GET','v1/topic/search','TopicService.java'),
('GET','v1/topic/topicInfo','TopicService.java'),
('GET','v1/transform/lowPrice','TransferService.java'),
('GET','v1/transform/priceTip','TransferService.java'),
('POST','v1/tweets','FeedService.java'),
('DELETE','v1/tweets/{id}','FeedService.java'),
('GET','v1/tweets/{id}','FeedService.java'),
('POST','v1/unifiedOrder/cancel/{orderId}','OrderService.java'),
('GET','v1/unifiedOrder/checkGoodStock','SeatMapService.java'),
('POST','v1/unifiedOrder/confirm','UnifiedOrderService.java');

INSERT INTO `_rev_endpoint_catalog` (`http_method`, `path`, `service_file`) VALUES
('POST','v1/unifiedOrder/confirmCheck','UnifiedOrderService.java'),
('POST','v1/unifiedOrder/makeOrder','UnifiedOrderService.java'),
('GET','v1/unifiedOrder/myOrders','OrderService.java'),
('GET','v1/unifiedOrder/types','OrderService.java'),
('POST','v1/unifiedOrder/urge/{orderId}','OrderService.java'),
('DELETE','v1/unifiedOrder/{orderId}','OrderService.java'),
('GET','v1/unifiedOrder/{orderId}/paySuccessDetail','OrderService.java'),
('POST','v1/unifiedOrder/{orderId}/userReceived','OrderService.java'),
('GET','v1/unifiedOrderComment','OrderService.java'),
('POST','v1/unifiedOrderComment/insertOrUpdate','OrderService.java'),
('POST','v1/unifiedPay/pay','OrderService.java'),
('GET','v1/unifiedPay/payDetail','OrderService.java'),
('POST','v1/unifiedPay/sendCode','OrderService.java'),
('GET','v1/user/','UserService.java'),
('POST','v1/user/avatar','UserService.java'),
('PUT','v1/user/baseInfo','UserService.java'),
('POST','v1/user/block','UserService.java'),
('POST','v1/user/cancelAccount','LoginService.java'),
('GET','v1/user/captcha','LoginService.java'),
('PUT','v1/user/findPassword/change','LoginService.java'),
('GET','v1/user/findPassword/check','LoginService.java'),
('POST','v1/user/follow','UserService.java'),
('GET','v1/user/following/actors','UserService.java'),
('GET','v1/user/following/mine','UserService.java'),
('GET','v1/user/fqlCredits','UserService.java'),
('POST','v1/user/login','LoginService.java'),
('PUT','v1/user/password','LoginService.java'),
('POST','v1/user/pushSwitch','UserService.java'),
('POST','v1/user/realNameInfo/save','UserIdService.java'),
('POST','v1/user/realNameInfo/search','UserIdService.java'),
('DELETE','v1/user/realNameInfo/{id}','UserIdService.java'),
('POST','v1/user/reg','LoginService.java'),
('GET','v1/user/search','SearchService.java'),
('GET','v1/user/stat','UserService.java'),
('GET','v1/user/toWatch','UserService.java'),
('POST','v1/user/tpLogin','LoginService.java'),
('POST','v1/user/tpReg','LoginService.java'),
('POST','v1/user/unblock','UserService.java'),
('POST','v1/user/unfollow','UserService.java'),
('GET','v1/user/userName','UserService.java'),
('GET','v1/user/users','UserService.java'),
('GET','v1/user/{userId}','UserService.java'),
('GET','v1/user/{userId}/followers','UserService.java'),
('GET','v1/user/{userId}/following','UserService.java'),
('POST','v1/userArticles','ArticleService.java'),
('DELETE','v1/userArticles/{id}','ArticleService.java'),
('GET','v1/userArticles/{id}','ArticleService.java'),
('PUT','v1/userArticles/{id}','ArticleService.java'),
('DELETE','v1/userLike','FeedService.java'),
('GET','v1/userLike','FeedService.java'),
('POST','v1/userLike','FeedService.java'),
('GET','v1/userReply','FeedService.java'),
('POST','v1/userReply','FeedService.java'),
('DELETE','v1/userReply/{id}','FeedService.java'),
('GET','v1/venues/${venueId}','ActivityService.java'),
('GET','v1/venues/{id}','VenueService.java'),
('GET','v1/venues/{venueId}/activities','ActivityService.java'),
('GET','v1/video','FeedService.java'),
('POST','v1/videoPlay/log','FeedService.java'),
('GET','v2/activities','ActivityService.java'),
('GET','v2/apkTool/latest','UpdateHelper.java'),
('GET','v2/cities/active','CityService.java'),
('GET','v2/config','ConfigManager.java'),
('GET','v2/config/categories','ActivityService.java'),
('GET','v2/giftCards','GiftCardService.java'),
('GET','v2/order/balance','OrderService.java'),
('GET','v2/order/{orderId}/paymentTypes?ct=11','OrderService.java'),
('GET','v2/reviews/{id}','ReviewService.java'),
('GET','v2/searchRecommend','SearchService.java'),
('GET','v2/spu/{spuId}/productList','ProductService.java'),
('POST','v2/transform','TransferService.java'),
('GET','v2/transform/activity','FeedService.java'),
('GET','v2/transform/list','TransferService.java'),
('GET','v2/transform/search','FeedService.java'),
('DELETE','v2/transform/{id}','FeedService.java'),
('GET','v2/user/{userId}/timeline','UserService.java'),
('GET','v2/userArticles/{id}','FeedService.java'),
('GET','v3/activities','ActivityService.java'),
('GET','v3/activities/{activityId}','ActivityService.java'),
('GET','v3/activities/{activityId}?supportSpeedPackBuy=true','ActivityService.java'),
('GET','v3/config','ConfigService.java'),
('GET','v3/home/dailySelected','HomeService.java'),
('GET','v3/home/organizer','HomeService.java'),
('POST','v3/order/allGiftCard','GiftCardService.java'),
('POST','v3/order/sendCode','OrderService.java'),
('GET','v3/reviews','ReviewService.java'),
('GET','v3/reviews/relatedReview/{activityId}','ReviewService.java'),
('GET','v3/share/activity','ShareService.java'),
('GET','v3/share/group','ShareService.java'),
('GET','v3/tickets?supportSpeedPackBuy=true','ChooseTicketService.java'),
('POST','v3/unified/tips','ActivityService.java'),
('POST','v3/unifiedOrder/makeOrder','UnifiedOrderService.java'),
('GET','v3/unifiedOrder/paySuccess','OrderService.java'),
('POST','v3/user/sendFindPasswordCode','LoginService.java'),
('POST','v3/user/sendLoginCode','LoginService.java'),
('POST','v3/user/sendRegCode','LoginService.java'),
('POST','v3/user/sendTpCode','LoginService.java'),
('POST','v4/activities/suggest','ActivityService.java'),
('GET','v4/feed/hot','FeedService.java'),
('GET','v4/home/collection','HomeService.java'),
('GET','v4/home/list','HomeService.java'),
('GET','v4/ip/ipRecommendList','HomeService.java'),
('POST','v4/order','OrderService.java'),
('POST','v4/order/confirmInfo','GrabTicketOrderService.java'),
('GET','v4/seatMap','SeatMapService.java'),
('GET','v4/seats','ChooseTicketService.java'),
('GET','v4/ticketCategories?supportSpeedPackBuy=true','ChooseTicketService.java'),
('GET','v4/unifiedOrder/getRecentOrderInfos','HomeService.java'),
('GET','v5/activities/getAllEvents','ActivityService.java'),
('GET','v5/activityPageDetail?supportSpeedPackBuy=true','ActivityService.java'),
('GET','v5/ticketCategories','SeatMapService.java'),
('POST','v6/activities','ActivityService.java'),
('GET','v6/tickets/notice','ChooseTicketService.java'),
('GET','v6/tickets?supportSpeedPackBuy=true','ChooseTicketService.java'),
('GET','v7/home','HomeService.java'),
('GET','v7/home/recommend','HomeService.java'),
('POST','v8/tickets?supportSpeedPackBuy=true','ChooseTicketService.java');

INSERT INTO `_rev_dto_table_mapping` (`dto_class`, `representation`) VALUES
('ActivityBean','activities'),
('ActivityCalendarBean','响应聚合 / 视图模型（不单独建表，字段见 _rev_dto_field_catalog）'),
('ActivityCampaign','activity_campaigns'),
('ActivityDetail','activity_details'),
('ActivityEventBean','activity_events'),
('ActivityGalleryBean','activity_galleries'),
('ActivityGroupBean','响应聚合 / 视图模型（不单独建表，字段见 _rev_dto_field_catalog）'),
('ActivityLineupBean','activity_lineups'),
('ActivityRefundPolicyBean','activity_refund_policies'),
('ActivityRefundRelationBean','activity_refund_relations'),
('ActivityRefundRuleBean','activity_refund_rules'),
('ActivityRefundRuleDetailBean','activity_refund_rule_details'),
('ActivitySubjectBean','activity_subjects'),
('ActorBean','actors'),
('ActorQrBean','actor_qr_codes'),
('AdBean','banners'),
('AddressBean','user_addresses'),
('Article','user_articles'),
('ArticleBean','user_articles'),
('ArticleDetailBean','响应聚合 / 视图模型（不单独建表，字段见 _rev_dto_field_catalog）'),
('Banner','banners'),
('BenefitIconBean','ads'),
('BizParam','值对象 / 枚举 / 内嵌对象（字段展平进父表 JSON 列，见 _rev_dto_field_catalog）'),
('CalendarTip','值对象 / 枚举 / 内嵌对象（字段展平进父表 JSON 列，见 _rev_dto_field_catalog）'),
('Campaign','campaigns'),
('CampaignBean','campaigns'),
('ChooseTicketBean','响应聚合 / 视图模型（不单独建表，字段见 _rev_dto_field_catalog）'),
('ChooseTicketCalendarData','值对象 / 枚举 / 内嵌对象（字段展平进父表 JSON 列，见 _rev_dto_field_catalog）'),
('ChooseTicketTipBean','响应聚合 / 视图模型（不单独建表，字段见 _rev_dto_field_catalog）'),
('CityBean','cities'),
('CommentSchemeBean','响应聚合 / 视图模型（不单独建表，字段见 _rev_dto_field_catalog）'),
('CommonInstallment','值对象 / 枚举 / 内嵌对象（字段展平进父表 JSON 列，见 _rev_dto_field_catalog）'),
('CommonShareInfoBean','响应聚合 / 视图模型（不单独建表，字段见 _rev_dto_field_catalog）'),
('CommunityBean','communities'),
('CommunityCategoryBean','响应聚合 / 视图模型（不单独建表，字段见 _rev_dto_field_catalog）'),
('Contact','值对象 / 枚举 / 内嵌对象（字段展平进父表 JSON 列，见 _rev_dto_field_catalog）'),
('CountryArea','country_areas'),
('Coupon','coupons'),
('CouponPackageBean','coupon_packages'),
('CreateReviewRequestBody','reviews'),
('Deliver','order_deliveries'),
('DeliverBean','order_deliveries'),
('DiscountActivityBean','discount_activities'),
('DiscoveryOpBean','home_floors'),
('FavoriteFeedBean','favorites'),
('FeaturedVideoBean','video_channels'),
('FeedAdBean','响应聚合 / 视图模型（不单独建表，字段见 _rev_dto_field_catalog）'),
('FeedBean','feeds'),
('FeedContentBean','feeds'),
('FeedListTopicBean','响应聚合 / 视图模型（不单独建表，字段见 _rev_dto_field_catalog）'),
('FeedPagination','feeds（分页封装）'),
('GiftCard','gift_cards'),
('GiftCardInfo','gift_cards（响应封装，enable/disable 列表）'),
('Goods','order_goods'),
('GoodsPackage','order_goods_packages'),
('HeaderBackgroundBean','响应聚合 / 视图模型（不单独建表，字段见 _rev_dto_field_catalog）'),
('HomeActorsBean','响应聚合 / 视图模型（不单独建表，字段见 _rev_dto_field_catalog）'),
('HomeBean','home_floors（首页聚合响应）'),
('HomeCenterOpBean','ads / op_positions'),
('HomeChoicenessBean','响应聚合 / 视图模型（不单独建表，字段见 _rev_dto_field_catalog）'),
('HomeFloorBean','home_floors'),
('HomeFloorShowBean','响应聚合 / 视图模型（不单独建表，字段见 _rev_dto_field_catalog）'),
('HomeFloors','home_floors'),
('HomeGuaranteeBean','响应聚合 / 视图模型（不单独建表，字段见 _rev_dto_field_catalog）'),
('HomeMultiItem','响应聚合 / 视图模型（不单独建表，字段见 _rev_dto_field_catalog）'),
('HomeOpbean','op_positions'),
('HomeShow','值对象 / 枚举 / 内嵌对象（字段展平进父表 JSON 列，见 _rev_dto_field_catalog）'),
('HomeShowItem','activities（首页卡片投影）'),
('HomeThemeRecommend','值对象 / 枚举 / 内嵌对象（字段展平进父表 JSON 列，见 _rev_dto_field_catalog）'),
('HomeVenuesBean','响应聚合 / 视图模型（不单独建表，字段见 _rev_dto_field_catalog）'),
('HomeWaterfall','值对象 / 枚举 / 内嵌对象（字段展平进父表 JSON 列，见 _rev_dto_field_catalog）'),
('HomeWaterfallActorBean','响应聚合 / 视图模型（不单独建表，字段见 _rev_dto_field_catalog）'),
('HomeWaterfallCollectionBean','响应聚合 / 视图模型（不单独建表，字段见 _rev_dto_field_catalog）'),
('HomeWaterfallItem','home_floors（瀑布流投影）'),
('HomeWaterfallListFliperBean','响应聚合 / 视图模型（不单独建表，字段见 _rev_dto_field_catalog）'),
('HomeWaterfallListShowBean','响应聚合 / 视图模型（不单独建表，字段见 _rev_dto_field_catalog）'),
('HotActorsBean','响应聚合 / 视图模型（不单独建表，字段见 _rev_dto_field_catalog）'),
('HotTourBean','响应聚合 / 视图模型（不单独建表，字段见 _rev_dto_field_catalog）'),
('HttpResult','全局响应封装（不建表）'),
('IDCardType','值对象 / 枚举 / 内嵌对象（字段展平进父表 JSON 列，见 _rev_dto_field_catalog）'),
('IconsBean','ads'),
('IdInfo','real_name_infos'),
('ImageInfo','feed_media'),
('LoginBean','login_sessions'),
('MakeOrderRequestBody','orders'),
('MyAddressBean','user_addresses'),
('OpPosition','op_positions'),
('OperatingPositionBean','op_positions'),
('OrderButtonBean','orders.buttons（JSON）'),
('OrderCommentBean','order_comments'),
('OrderConfirmExtraInfoBean','order_confirm_snapshots.addition_components（JSON）'),
('OrderConfirmInfo','order_confirm_snapshots'),
('OrderConfirmInfoBean','order_confirm_snapshots'),
('OrderConfirmInfoRequestBody','order_confirm_snapshots'),
('OrderDesc','值对象 / 枚举 / 内嵌对象（字段展平进父表 JSON 列，见 _rev_dto_field_catalog）'),
('OrderDiscount','campaigns'),
('OrderEvent','order_events'),
('OrderGroupProduct','值对象 / 枚举 / 内嵌对象（字段展平进父表 JSON 列，见 _rev_dto_field_catalog）'),
('OrderItem','order_items'),
('OrderPaymentInfo','payments'),
('OrderTypeBean','响应聚合 / 视图模型（不单独建表，字段见 _rev_dto_field_catalog）'),
('OrganizerBean','organizers'),
('Pagination','通用分页封装（不建表）'),
('PayInfo','payments'),
('PaySuccessDetail','值对象 / 枚举 / 内嵌对象（字段展平进父表 JSON 列，见 _rev_dto_field_catalog）'),
('PinTuanCampaignItem','pin_tuan_campaign_items'),
('PinTuanCampaignMap','值对象 / 枚举 / 内嵌对象（字段展平进父表 JSON 列，见 _rev_dto_field_catalog）'),
('PinTuanInfo','pin_tuan_campaigns'),
('PinTuanOrderRequestBody','pin_tuan_orders'),
('PnConfig','app_configs'),
('PopBoxBean','op_positions'),
('PostageBean','order_postage（并写入 order_deliveries）'),
('ProductBean','activities（商品化演出，未单独建表）'),
('ProductBottomItemWithStateWrapperBean','响应聚合 / 视图模型（不单独建表，字段见 _rev_dto_field_catalog）'),
('ProductGroupBean','响应聚合 / 视图模型（不单独建表，字段见 _rev_dto_field_catalog）'),
('ProductWithDisplayWrapperBean','响应聚合 / 视图模型（不单独建表，字段见 _rev_dto_field_catalog）'),
('ProductWithMultiTypeBean','响应聚合 / 视图模型（不单独建表，字段见 _rev_dto_field_catalog）'),
('Provider','值对象 / 枚举 / 内嵌对象（字段展平进父表 JSON 列，见 _rev_dto_field_catalog）'),
('ProxyBuyCompensationInfo','proxy_buy_configs'),
('ProxyBuyInfo','proxy_buy_configs'),
('ProxyBuyTimeInfo','proxy_buy_configs'),
('ProxyBuyTurnInfo','proxy_buy_configs'),
('RecommendBean','响应聚合 / 视图模型（不单独建表，字段见 _rev_dto_field_catalog）'),
('RecommendUser','值对象 / 枚举 / 内嵌对象（字段展平进父表 JSON 列，见 _rev_dto_field_catalog）'),
('Reply','review_replies'),
('ReplySendRequestBody','review_replies'),
('Review','reviews'),
('ReviewDetailBean','响应聚合 / 视图模型（不单独建表，字段见 _rev_dto_field_catalog）'),
('ReviewToDo','review_todos'),
('SchemaKeywordBean','search_keywords'),
('SchemaKeywordRankBean','search_keywords'),
('SearchResultBannerBean','响应聚合 / 视图模型（不单独建表，字段见 _rev_dto_field_catalog）'),
('SearchResultShopBean','shops'),
('SearchSuggestBean','search_keywords'),
('SelectionBean','响应聚合 / 视图模型（不单独建表，字段见 _rev_dto_field_catalog）'),
('SendTweetRequestBody','值对象 / 枚举 / 内嵌对象（字段展平进父表 JSON 列，见 _rev_dto_field_catalog）'),
('ShareModel','share_logs'),
('ShareParams','share_logs'),
('Shop','shops'),
('ShopInfo','shops'),
('ShowDetailBean','activities（详情聚合响应，未单独建表）'),
('ShowFeedPagination','feeds（分页封装）'),
('ShowTip','值对象 / 枚举 / 内嵌对象（字段展平进父表 JSON 列，见 _rev_dto_field_catalog）'),
('ShowTips','值对象 / 枚举 / 内嵌对象（字段展平进父表 JSON 列，见 _rev_dto_field_catalog）'),
('SkuDetailBean','响应聚合 / 视图模型（不单独建表，字段见 _rev_dto_field_catalog）'),
('SpeedPackChoice','speed_packs'),
('SplitOrderContext','speed_packs'),
('StationBean','响应聚合 / 视图模型（不单独建表，字段见 _rev_dto_field_catalog）'),
('StatisticBean','user_statistics'),
('TPLoginBean','响应聚合 / 视图模型（不单独建表，字段见 _rev_dto_field_catalog）');

INSERT INTO `_rev_dto_table_mapping` (`dto_class`, `representation`) VALUES
('TagBean','响应聚合 / 视图模型（不单独建表，字段见 _rev_dto_field_catalog）'),
('TeamFeedBean','响应聚合 / 视图模型（不单独建表，字段见 _rev_dto_field_catalog）'),
('Test','响应聚合 / 视图模型（不单独建表，字段见 _rev_dto_field_catalog）'),
('TicketAreaBean','ticket_areas'),
('TicketCategory','ticket_categories'),
('TicketGroup','ticket_groups'),
('TicketPackage','值对象 / 枚举 / 内嵌对象（字段展平进父表 JSON 列，见 _rev_dto_field_catalog）'),
('TicketPackagePagination','响应聚合 / 视图模型（不单独建表，字段见 _rev_dto_field_catalog）'),
('TicketPagination','响应聚合 / 视图模型（不单独建表，字段见 _rev_dto_field_catalog）'),
('TicketSeatBean','ticket_seats'),
('TicketTableBean','seat_map_layouts'),
('TimeLine','值对象 / 枚举 / 内嵌对象（字段展平进父表 JSON 列，见 _rev_dto_field_catalog）'),
('TimeLinePagination','响应聚合 / 视图模型（不单独建表，字段见 _rev_dto_field_catalog）'),
('Tip','值对象 / 枚举 / 内嵌对象（字段展平进父表 JSON 列，见 _rev_dto_field_catalog）'),
('ToWatchBean','to_watch'),
('TopicBean','topics'),
('TopicWithColorBean','响应聚合 / 视图模型（不单独建表，字段见 _rev_dto_field_catalog）'),
('TransferPagination','响应聚合 / 视图模型（不单独建表，字段见 _rev_dto_field_catalog）'),
('TransferTicket','transfers'),
('TransformBean','transfers'),
('TransformFeedBean','transfers'),
('Tweet','feeds'),
('TweetVideo','feed_media'),
('UpdateInfo','app_versions'),
('Urge','值对象 / 枚举 / 内嵌对象（字段展平进父表 JSON 列，见 _rev_dto_field_catalog）'),
('UserMessage','user_messages'),
('UserMessageSimple','user_messages'),
('UserMsgCount','user_messages / user_statistics（未读数）'),
('Venue','venues'),
('VideoChannelBean','video_channels'),
('VideoDetailBean','video_channels'),
('VideoItemBean','feed_media'),
('launchScreenBean','op_positions'),
('ActivityDetailType','值对象 / 枚举 / 内嵌对象（字段展平进父表 JSON 列，见 _rev_dto_field_catalog）'),
('FeedOpType','值对象 / 枚举 / 内嵌对象（字段展平进父表 JSON 列，见 _rev_dto_field_catalog）'),
('FeedType','值对象 / 枚举 / 内嵌对象（字段展平进父表 JSON 列，见 _rev_dto_field_catalog）'),
('OrderStatus','值对象 / 枚举 / 内嵌对象（字段展平进父表 JSON 列，见 _rev_dto_field_catalog）'),
('SaleStatus','值对象 / 枚举 / 内嵌对象（字段展平进父表 JSON 列，见 _rev_dto_field_catalog）'),
('SearchType','值对象 / 枚举 / 内嵌对象（字段展平进父表 JSON 列，见 _rev_dto_field_catalog）'),
('TicketType','值对象 / 枚举 / 内嵌对象（字段展平进父表 JSON 列，见 _rev_dto_field_catalog）'),
('TransferStatus','值对象 / 枚举 / 内嵌对象（字段展平进父表 JSON 列，见 _rev_dto_field_catalog）'),
('UserMsgType','值对象 / 枚举 / 内嵌对象（字段展平进父表 JSON 列，见 _rev_dto_field_catalog）'),
('AppLinkBean','响应聚合 / 视图模型（不单独建表，字段见 _rev_dto_field_catalog）'),
('CustomTagBean','响应聚合 / 视图模型（不单独建表，字段见 _rev_dto_field_catalog）'),
('SearchOption','search_keywords'),
('UserBean','users'),
('ChooseTicketPagination','响应聚合 / 视图模型（不单独建表，字段见 _rev_dto_field_catalog）'),
('CreditPay','值对象 / 枚举 / 内嵌对象（字段展平进父表 JSON 列，见 _rev_dto_field_catalog）'),
('EmptyData','值对象 / 枚举 / 内嵌对象（字段展平进父表 JSON 列，见 _rev_dto_field_catalog）'),
('GoodsBean','响应聚合 / 视图模型（不单独建表，字段见 _rev_dto_field_catalog）'),
('HomeHotActorSite','值对象 / 枚举 / 内嵌对象（字段展平进父表 JSON 列，见 _rev_dto_field_catalog）'),
('IFeedContent','值对象 / 枚举 / 内嵌对象（字段展平进父表 JSON 列，见 _rev_dto_field_catalog）'),
('ItemTypeWrapper','值对象 / 枚举 / 内嵌对象（字段展平进父表 JSON 列，见 _rev_dto_field_catalog）'),
('PopIconBean','响应聚合 / 视图模型（不单独建表，字段见 _rev_dto_field_catalog）'),
('RelativeActivityBean','响应聚合 / 视图模型（不单独建表，字段见 _rev_dto_field_catalog）'),
('ShowStatus','值对象 / 枚举 / 内嵌对象（字段展平进父表 JSON 列，见 _rev_dto_field_catalog）'),
('TicketStockBean','响应聚合 / 视图模型（不单独建表，字段见 _rev_dto_field_catalog）'),
('TipDetailTypes','值对象 / 枚举 / 内嵌对象（字段展平进父表 JSON 列，见 _rev_dto_field_catalog）'),
('DeliverType','值对象 / 枚举 / 内嵌对象（字段展平进父表 JSON 列，见 _rev_dto_field_catalog）'),
('DiscoveryOpType','值对象 / 枚举 / 内嵌对象（字段展平进父表 JSON 列，见 _rev_dto_field_catalog）'),
('OrderUtils','值对象 / 枚举 / 内嵌对象（字段展平进父表 JSON 列，见 _rev_dto_field_catalog）'),
('RefundRelationBizType','值对象 / 枚举 / 内嵌对象（字段展平进父表 JSON 列，见 _rev_dto_field_catalog）'),
('RefundRuleMode','值对象 / 枚举 / 内嵌对象（字段展平进父表 JSON 列，见 _rev_dto_field_catalog）'),
('ShareChannelType','值对象 / 枚举 / 内嵌对象（字段展平进父表 JSON 列，见 _rev_dto_field_catalog）'),
('TipsBizType','值对象 / 枚举 / 内嵌对象（字段展平进父表 JSON 列，见 _rev_dto_field_catalog）'),
('TipsDisplayType','值对象 / 枚举 / 内嵌对象（字段展平进父表 JSON 列，见 _rev_dto_field_catalog）'),
('TipsScene','值对象 / 枚举 / 内嵌对象（字段展平进父表 JSON 列，见 _rev_dto_field_catalog）');


-- ------------------------------------------------------------------------
--  ↓ 以下为 12_constraints.sql
-- ------------------------------------------------------------------------

-- ========================================================================
--  sql_rebuildnew · 12_constraints.sql（忠实原样版，不做 R4）
--  SQL 正文与 docs/sql_new 逐语句一致，仅替换过时生成头部。
--  在 MySQL 8.4.10 上会产出 119 条 Warning 1681（TINYINT(1) 显示宽度弃用），0 ERROR。
-- ========================================================================

-- ========================================================================
--  12_constraints.sql · 推断外键 / 校验查询 / 落地建议
-- ========================================================================
--  ⚠ 以下 61 条外键全部是【按字段名与业务语义推断】的，不是从真实库导出的。
--  ⚠ 本重构按服务拆库后，绝大多数外键是【跨库】的：
--      MySQL InnoDB 支持 `库`.`表` 形式的跨库外键，但要求所有库在同一实例；
--      真实微服务落地通常【不建物理外键】、改由应用层保证一致性。
--      若只想要建表、不想要约束，跳过本文件即可（不影响 01–11 执行）。
--  所有表都已用所属服务库完全限定，因此本文件不依赖 USE 的当前库。
-- ========================================================================

ALTER TABLE `planstar_user`.`login_sessions` ADD CONSTRAINT `fk_login_sessions_user_id` FOREIGN KEY (`user_id`) REFERENCES `planstar_user`.`users` (`user_id`);
ALTER TABLE `planstar_user`.`real_name_infos` ADD CONSTRAINT `fk_real_name_infos_user_id` FOREIGN KEY (`user_id`) REFERENCES `planstar_user`.`users` (`user_id`);
ALTER TABLE `planstar_user`.`user_addresses` ADD CONSTRAINT `fk_user_addresses_user_id` FOREIGN KEY (`user_id`) REFERENCES `planstar_user`.`users` (`user_id`);
ALTER TABLE `planstar_user`.`user_follows` ADD CONSTRAINT `fk_user_follows_user_id` FOREIGN KEY (`user_id`) REFERENCES `planstar_user`.`users` (`user_id`);
ALTER TABLE `planstar_user`.`user_statistics` ADD CONSTRAINT `fk_user_statistics_user_id` FOREIGN KEY (`user_id`) REFERENCES `planstar_user`.`users` (`user_id`);
ALTER TABLE `planstar_user`.`user_messages` ADD CONSTRAINT `fk_user_messages_user_id` FOREIGN KEY (`user_id`) REFERENCES `planstar_user`.`users` (`user_id`);
ALTER TABLE `planstar_user`.`chat_threads` ADD CONSTRAINT `fk_chat_threads_user_id` FOREIGN KEY (`user_id`) REFERENCES `planstar_user`.`users` (`user_id`);
ALTER TABLE `planstar_user`.`chat_messages` ADD CONSTRAINT `fk_chat_messages_thread_id` FOREIGN KEY (`thread_id`) REFERENCES `planstar_user`.`chat_threads` (`thread_id`);
ALTER TABLE `planstar_catalog`.`venues` ADD CONSTRAINT `fk_venues_city_id` FOREIGN KEY (`city_id`) REFERENCES `planstar_catalog`.`cities` (`city_id`);
ALTER TABLE `planstar_catalog`.`activities` ADD CONSTRAINT `fk_activities_city_id` FOREIGN KEY (`city_id`) REFERENCES `planstar_catalog`.`cities` (`city_id`);
ALTER TABLE `planstar_catalog`.`activities` ADD CONSTRAINT `fk_activities_venue_id` FOREIGN KEY (`venue_id`) REFERENCES `planstar_catalog`.`venues` (`venue_id`);
ALTER TABLE `planstar_catalog`.`activity_events` ADD CONSTRAINT `fk_activity_events_activity_id` FOREIGN KEY (`activity_id`) REFERENCES `planstar_catalog`.`activities` (`id`);
ALTER TABLE `planstar_catalog`.`activity_details` ADD CONSTRAINT `fk_activity_details_activity_id` FOREIGN KEY (`activity_id`) REFERENCES `planstar_catalog`.`activities` (`id`);
ALTER TABLE `planstar_catalog`.`activity_lineups` ADD CONSTRAINT `fk_activity_lineups_activity_id` FOREIGN KEY (`activity_id`) REFERENCES `planstar_catalog`.`activities` (`id`);
ALTER TABLE `planstar_catalog`.`activity_galleries` ADD CONSTRAINT `fk_activity_galleries_activity_id` FOREIGN KEY (`activity_id`) REFERENCES `planstar_catalog`.`activities` (`id`);
ALTER TABLE `planstar_catalog`.`activity_refund_policies` ADD CONSTRAINT `fk_activity_refund_policies_activity_id` FOREIGN KEY (`activity_id`) REFERENCES `planstar_catalog`.`activities` (`id`);
ALTER TABLE `planstar_catalog`.`activity_refund_rule_details` ADD CONSTRAINT `fk_activity_refund_rule_details_rule_id` FOREIGN KEY (`rule_id`) REFERENCES `planstar_catalog`.`activity_refund_rules` (`id`);
ALTER TABLE `planstar_catalog`.`activity_campaigns` ADD CONSTRAINT `fk_activity_campaigns_activity_id` FOREIGN KEY (`activity_id`) REFERENCES `planstar_catalog`.`activities` (`id`);
ALTER TABLE `planstar_catalog`.`tours` ADD CONSTRAINT `fk_tours_actor_id` FOREIGN KEY (`actor_id`) REFERENCES `planstar_catalog`.`actors` (`id`);
ALTER TABLE `planstar_ticket`.`ticket_categories` ADD CONSTRAINT `fk_ticket_categories_activity_event_id` FOREIGN KEY (`activity_event_id`) REFERENCES `planstar_catalog`.`activity_events` (`id`);
ALTER TABLE `planstar_ticket`.`ticket_groups` ADD CONSTRAINT `fk_ticket_groups_ticket_category_id` FOREIGN KEY (`ticket_category_id`) REFERENCES `planstar_ticket`.`ticket_categories` (`id`);
ALTER TABLE `planstar_ticket`.`ticket_seats` ADD CONSTRAINT `fk_ticket_seats_area_id` FOREIGN KEY (`area_id`) REFERENCES `planstar_ticket`.`ticket_areas` (`id`);
ALTER TABLE `planstar_order`.`orders` ADD CONSTRAINT `fk_orders_user_id` FOREIGN KEY (`user_id`) REFERENCES `planstar_user`.`users` (`user_id`);
ALTER TABLE `planstar_order`.`orders` ADD CONSTRAINT `fk_orders_activity_id` FOREIGN KEY (`activity_id`) REFERENCES `planstar_catalog`.`activities` (`id`);
ALTER TABLE `planstar_order`.`order_events` ADD CONSTRAINT `fk_order_events_order_id` FOREIGN KEY (`order_id`) REFERENCES `planstar_order`.`orders` (`order_id`);
ALTER TABLE `planstar_order`.`order_items` ADD CONSTRAINT `fk_order_items_order_id` FOREIGN KEY (`order_id`) REFERENCES `planstar_order`.`orders` (`order_id`);
ALTER TABLE `planstar_order`.`order_goods` ADD CONSTRAINT `fk_order_goods_order_id` FOREIGN KEY (`order_id`) REFERENCES `planstar_order`.`orders` (`order_id`);
ALTER TABLE `planstar_order`.`order_goods_packages` ADD CONSTRAINT `fk_order_goods_packages_order_id` FOREIGN KEY (`order_id`) REFERENCES `planstar_order`.`orders` (`order_id`);
ALTER TABLE `planstar_order`.`order_deliveries` ADD CONSTRAINT `fk_order_deliveries_order_id` FOREIGN KEY (`order_id`) REFERENCES `planstar_order`.`orders` (`order_id`);
ALTER TABLE `planstar_order`.`order_confirm_snapshots` ADD CONSTRAINT `fk_order_confirm_snapshots_order_id` FOREIGN KEY (`order_id`) REFERENCES `planstar_order`.`orders` (`order_id`);
ALTER TABLE `planstar_payment`.`payments` ADD CONSTRAINT `fk_payments_order_id` FOREIGN KEY (`order_id`) REFERENCES `planstar_order`.`orders` (`order_id`);
ALTER TABLE `planstar_payment`.`payment_channels` ADD CONSTRAINT `fk_payment_channels_pay_id` FOREIGN KEY (`pay_id`) REFERENCES `planstar_payment`.`payments` (`pay_id`);
ALTER TABLE `planstar_payment`.`refunds` ADD CONSTRAINT `fk_refunds_order_id` FOREIGN KEY (`order_id`) REFERENCES `planstar_order`.`orders` (`order_id`);
ALTER TABLE `planstar_order`.`order_postage` ADD CONSTRAINT `fk_order_postage_order_id` FOREIGN KEY (`order_id`) REFERENCES `planstar_order`.`orders` (`order_id`);
ALTER TABLE `planstar_order`.`order_comments` ADD CONSTRAINT `fk_order_comments_order_id` FOREIGN KEY (`order_id`) REFERENCES `planstar_order`.`orders` (`order_id`);
ALTER TABLE `planstar_marketing`.`coupons` ADD CONSTRAINT `fk_coupons_user_id` FOREIGN KEY (`user_id`) REFERENCES `planstar_user`.`users` (`user_id`);
ALTER TABLE `planstar_payment`.`gift_cards` ADD CONSTRAINT `fk_gift_cards_user_id` FOREIGN KEY (`user_id`) REFERENCES `planstar_user`.`users` (`user_id`);
ALTER TABLE `planstar_payment`.`gift_card_usages` ADD CONSTRAINT `fk_gift_card_usages_order_id` FOREIGN KEY (`order_id`) REFERENCES `planstar_order`.`orders` (`order_id`);
ALTER TABLE `planstar_payment`.`gift_card_usages` ADD CONSTRAINT `fk_gift_card_usages_gift_card_id` FOREIGN KEY (`gift_card_id`) REFERENCES `planstar_payment`.`gift_cards` (`id`);
ALTER TABLE `planstar_payment`.`wallet_accounts` ADD CONSTRAINT `fk_wallet_accounts_user_id` FOREIGN KEY (`user_id`) REFERENCES `planstar_user`.`users` (`user_id`);
ALTER TABLE `planstar_payment`.`wallet_sequences` ADD CONSTRAINT `fk_wallet_sequences_account_id` FOREIGN KEY (`account_id`) REFERENCES `planstar_payment`.`wallet_accounts` (`account_id`);
ALTER TABLE `planstar_marketing`.`pin_tuan_campaigns` ADD CONSTRAINT `fk_pin_tuan_campaigns_activity_id` FOREIGN KEY (`activity_id`) REFERENCES `planstar_catalog`.`activities` (`id`);
ALTER TABLE `planstar_marketing`.`pin_tuan_campaign_items` ADD CONSTRAINT `fk_pin_tuan_campaign_items_pin_tuan_id` FOREIGN KEY (`pin_tuan_id`) REFERENCES `planstar_marketing`.`pin_tuan_campaigns` (`id`);
--  [sql_new] F-01 阻塞已由 R2 消除：子列 pin_tuan_orders.pin_tuan_id 现为 BIGINT，
--  与父列 pin_tuan_campaigns.id BIGINT 一致，本条约束不再因类型不匹配中断。
ALTER TABLE `planstar_marketing`.`pin_tuan_orders` ADD CONSTRAINT `fk_pin_tuan_orders_pin_tuan_id` FOREIGN KEY (`pin_tuan_id`) REFERENCES `planstar_marketing`.`pin_tuan_campaigns` (`id`);
ALTER TABLE `planstar_marketing`.`group_buy_members` ADD CONSTRAINT `fk_group_buy_members_group_id` FOREIGN KEY (`group_id`) REFERENCES `planstar_marketing`.`pin_tuan_campaigns` (`id`);
ALTER TABLE `planstar_marketing`.`proxy_buy_configs` ADD CONSTRAINT `fk_proxy_buy_configs_activity_id` FOREIGN KEY (`activity_id`) REFERENCES `planstar_catalog`.`activities` (`id`);
ALTER TABLE `planstar_marketing`.`to_watch` ADD CONSTRAINT `fk_to_watch_user_id` FOREIGN KEY (`user_id`) REFERENCES `planstar_user`.`users` (`user_id`);
ALTER TABLE `planstar_marketing`.`favorites` ADD CONSTRAINT `fk_favorites_user_id` FOREIGN KEY (`user_id`) REFERENCES `planstar_user`.`users` (`user_id`);
ALTER TABLE `planstar_marketing`.`favorites` ADD CONSTRAINT `fk_favorites_activity_id` FOREIGN KEY (`activity_id`) REFERENCES `planstar_catalog`.`activities` (`id`);
ALTER TABLE `planstar_community`.`reviews` ADD CONSTRAINT `fk_reviews_user_id` FOREIGN KEY (`user_id`) REFERENCES `planstar_user`.`users` (`user_id`);
ALTER TABLE `planstar_community`.`reviews` ADD CONSTRAINT `fk_reviews_activity_id` FOREIGN KEY (`activity_id`) REFERENCES `planstar_catalog`.`activities` (`id`);
ALTER TABLE `planstar_community`.`review_replies` ADD CONSTRAINT `fk_review_replies_review_id` FOREIGN KEY (`review_id`) REFERENCES `planstar_community`.`reviews` (`id`);
ALTER TABLE `planstar_community`.`review_todos` ADD CONSTRAINT `fk_review_todos_user_id` FOREIGN KEY (`user_id`) REFERENCES `planstar_user`.`users` (`user_id`);
ALTER TABLE `planstar_community`.`feeds` ADD CONSTRAINT `fk_feeds_user_id` FOREIGN KEY (`user_id`) REFERENCES `planstar_user`.`users` (`user_id`);
ALTER TABLE `planstar_community`.`feed_media` ADD CONSTRAINT `fk_feed_media_feed_id` FOREIGN KEY (`feed_id`) REFERENCES `planstar_community`.`feeds` (`id`);
ALTER TABLE `planstar_community`.`user_articles` ADD CONSTRAINT `fk_user_articles_user_id` FOREIGN KEY (`user_id`) REFERENCES `planstar_user`.`users` (`user_id`);
ALTER TABLE `planstar_community`.`likes` ADD CONSTRAINT `fk_likes_user_id` FOREIGN KEY (`user_id`) REFERENCES `planstar_user`.`users` (`user_id`);
ALTER TABLE `planstar_marketing`.`transfers` ADD CONSTRAINT `fk_transfers_user_id` FOREIGN KEY (`user_id`) REFERENCES `planstar_user`.`users` (`user_id`);
ALTER TABLE `planstar_content`.`share_logs` ADD CONSTRAINT `fk_share_logs_user_id` FOREIGN KEY (`user_id`) REFERENCES `planstar_user`.`users` (`user_id`);
ALTER TABLE `planstar_content`.`feedback` ADD CONSTRAINT `fk_feedback_user_id` FOREIGN KEY (`user_id`) REFERENCES `planstar_user`.`users` (`user_id`);
ALTER TABLE `planstar_content`.`tracking_events` ADD CONSTRAINT `fk_tracking_events_user_id` FOREIGN KEY (`user_id`) REFERENCES `planstar_user`.`users` (`user_id`);

--  校验查询需先 USE 对应服务库（dict_enums / _rev_* 在 planstar_meta）。


-- ---------------------------------------------------------------------------
--  校验查询（建完后执行，确认覆盖率）
-- ---------------------------------------------------------------------------
-- 1) 表数量（业务表 + 证据表 + 字典表；SDK 表已移除）
-- SELECT COUNT(*) AS table_count FROM information_schema.tables
--   WHERE table_schema = DATABASE();
-- 2) 逆向到的 DTO 字段总数 / 唯一字段名数
-- SELECT COUNT(*) AS field_rows, COUNT(DISTINCT field_name) AS unique_fields
--   FROM `_rev_dto_field_catalog`;
-- 3) 端点总数 / 按 HTTP 方法分布
-- SELECT http_method, COUNT(*) FROM `_rev_endpoint_catalog` GROUP BY http_method;
-- 4) 枚举字典分布
-- SELECT enum_name, COUNT(*) FROM `dict_enums` GROUP BY enum_name ORDER BY enum_name;
-- 5) 未落表的 DTO（需要人工确认是否补表）
-- SELECT dto_class, representation FROM `_rev_dto_table_mapping`
--   WHERE representation LIKE '%不单独建表%' OR representation LIKE '%内嵌对象%';
--
-- 落地建议：
--   a) 先拿真实库 information_schema 对齐表名 / 列名 / 类型，尤其是 DECIMAL 精度；
--   b) 时间字段若是 DATETIME，用 FROM_UNIXTIME(col/1000) 做一次迁移校验；
--   c) 状态类字段务必对照 dict_enums：OrderStatus 与 OrderUtils 前 4 个状态差 1，
--      WAITING_FOR_TICKETS 的描述还是从 INIT 复制错的（wiki_zh/27 §11.1）；
--   d) 94 个 DTO 的 ID 是 Java int，超过 2^31 会溢出（wiki_zh/27 §10），
--      本 schema 已放宽为 BIGINT；
--   e) ActivityBean.id 与 activityId 是两个独立字段，服务端有时只填其中一个，
--      导入时务必做 COALESCE 归一；
--   f) MakeOrderRequestBody.channelAttributeInfo 是自嵌套 JSON 字符串，
--      couponIds 与 campaignIds 互斥，giftCards 带实际抵扣额。

-- ========================================================================
--  EOF
-- ========================================================================


-- ------------------------------------------------------------------------
--  ↓ 以下为 13_query_indexes.sql
-- ------------------------------------------------------------------------

-- ========================================================================
--  sql_rebuildnew · 13_query_indexes.sql（忠实原样版，不做 R4）
--  SQL 正文与 docs/sql_new 逐语句一致，仅替换过时生成头部。
--  在 MySQL 8.4.10 上会产出 119 条 Warning 1681（TINYINT(1) 显示宽度弃用），0 ERROR。
-- ========================================================================

--  可选文件：不进入默认安装链；8 个条件索引候选，需 EXPLAIN 验证。

-- =======================================================================
--  Planstar 数据库评审 · 候选优化迁移
--  目标：MySQL 8.0 / InnoDB / utf8mb4 / 全新空库演示环境
--  生成：generate_review_sql.py（读取 01..11 根 DDL，不修改 000001 基线）
--  方言：MySQL 8.0。所有 DDL 均隐式提交，回滚依赖本包的 .down.sql 补偿脚本。
--  重要：本文件不是「已验证的性能优化」，索引项均为条件候选，
--        上线前必须用真实 EXPLAIN / 慢查询日志验证。
-- =======================================================================

-- -----------------------------------------------------------------------
-- 条件查询索引候选（全部为普通二级索引，可在空库安全回滚）
-- 每条都标注业务场景与证据；没有真实 QPS/执行计划，禁止声称性能收益。
-- 执行顺序：000002 之后、12_constraints.sql 之前或之后均可（索引名不与外键自动索引冲突）。
-- -----------------------------------------------------------------------

-- [planstar_order.orders] idx_order_user_addtime
--   场景：我的订单列表：按 user_id 过滤、按 add_time 倒序、以 order_id 做游标翻页
--   证据：lib/features/order/application/order_list_controller.dart → repository.findAll() 后在本地按 submittedAt/placedAt 排序
--   权衡：与现有 idx_order_user(user_id,status) 部分重叠；保留两者仅在「状态筛选」与「全量时间线」两种查询都真实存在时才合理
ALTER TABLE `planstar_order`.`orders` ADD INDEX `idx_order_user_addtime` (`user_id`,`add_time`,`order_id`);

-- [planstar_order.order_events] idx_orderev_order_time
--   场景：订单详情状态时间线：按 order_id 取事件并按发生时间排序
--   证据：order_events.order_id + add_time（05_order.sql:86,92）；现有 idx_orderev_order 仅覆盖 order_id
--   权衡：覆盖索引候选；若时间线永远按主键顺序读取则可不建
ALTER TABLE `planstar_order`.`order_events` ADD INDEX `idx_orderev_order_time` (`order_id`,`add_time`,`id`);

-- [planstar_catalog.activity_events] idx_event_act_start
--   场景：选票页场次日历：按演出取场次并按开演时间升序
--   证据：lib/features/catalog/ui/show_list_page.dart → firstSessionAt 升序排序
--   权衡：与现有 idx_event_act、idx_event_start 各重叠一部分；Alibaba 索引规约要求避免冗余，上线前用 EXPLAIN 二选一
ALTER TABLE `planstar_catalog`.`activity_events` ADD INDEX `idx_event_act_start` (`activity_id`,`start`,`id`);

-- [planstar_ticket.ticket_categories] idx_tc_event_status
--   场景：选票页票档列表：按场次 + 在售状态取票档
--   证据：04_ticket_inventory.sql:22,39；现有 idx_tc_event 仅覆盖场次
--   权衡：status 语义与取值尚未确认，条件索引；若 status 区分度极低则收益有限
ALTER TABLE `planstar_ticket`.`ticket_categories` ADD INDEX `idx_tc_event_status` (`activity_event_id`,`status`,`id`);

-- [planstar_marketing.coupons] idx_coupon_user_status_expiry
--   场景：我的优惠券：可用券按到期时间升序（临期排序）
--   证据：lib/features/marketing/coupon_page.dart / coupon.dart → daysUntilExpiry、isExpiringSoon 目前为客户端本地排序
--   权衡：当前 Flutter 为本地过滤排序，服务端尚无该查询契约；属预备候选，不应据此声称性能收益
ALTER TABLE `planstar_marketing`.`coupons` ADD INDEX `idx_coupon_user_status_expiry` (`user_id`,`status`,`effective_time_to`,`id`);

-- [planstar_payment.wallet_sequences] idx_walletseq_account_time
--   场景：钱包账户流水：按 account_id 分页并按时间倒序
--   证据：06_payment.sql:107,112；现有 idx_walletseq_user(user_id,time) 走用户维度
--   权衡：与现有索引口径不同（account_id vs user_id）；只有当服务端确实按账户查询时才需要
ALTER TABLE `planstar_payment`.`wallet_sequences` ADD INDEX `idx_walletseq_account_time` (`account_id`,`time`,`id`);

-- [planstar_community.reviews] idx_review_act_time
--   场景：演出详情评价列表：按演出取评价并按时间倒序 + 游标翻页
--   证据：08_community.sql:22-36（reviews 同时含 page_index/page_size/total_num/data 分页封装字段）
--   权衡：先要把分页封装字段从实体表中剥离，否则查询口径仍不明确
ALTER TABLE `planstar_community`.`reviews` ADD INDEX `idx_review_act_time` (`activity_id`,`add_time`,`id`);

-- [planstar_community.feeds] idx_feed_addtime_id  （替换 idx_feed_addtime）
--   场景：发现流游标分页：以 (add_time,id) 作为稳定游标，替代 OFFSET 深分页
--   证据：lib/features/discover/data/planstar_feed_api.dart → 客户端丢弃 next_page_key/has_more；基线已有 idx_feed_addtime(add_time)
--   权衡：替换而非新增；feeds 表目前把 next_page_key/has_more 当实体字段持久化，索引只解决排序，不解决建模问题
ALTER TABLE `planstar_community`.`feeds` DROP INDEX `idx_feed_addtime`;
ALTER TABLE `planstar_community`.`feeds` ADD INDEX `idx_feed_addtime_id` (`add_time`,`id`);

-- EOF
