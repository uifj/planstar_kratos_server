-- ========================================================================
--  sql · 02_user.sql（S2 全量规范重构版）
--  规范: 单数表名 / id BIGINT UNSIGNED 自增 / gmt_create,gmt_modified DATETIME
--        is_xxx TINYINT UNSIGNED / 金额 DECIMAL / 无物理外键 / 无冗余索引
--  表: user_info, login_session, real_name_info, user_address, user_follow, user_statistic, user_message, chat_thread, chat_message
-- ========================================================================

USE `planstar_user`;

-- ----------------------------------------------------------------------
--  域 1 · 账号与身份（account / identity）
-- ----------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS `user_info` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `user_id` BIGINT UNSIGNED NOT NULL COMMENT '用户ID',
  `user_name` VARCHAR(128) COMMENT '用户名',
  `avatar` VARCHAR(512) COMMENT '头像 URL',
  `badge_icon` VARCHAR(512) COMMENT '徽章图标',
  `bio` VARCHAR(1024) COMMENT '个人简介',
  `birthday` DATE COMMENT '生日',
  `gender` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '性别',
  `mobile_no` VARCHAR(32) NOT NULL COMMENT '手机号',
  `phone` VARCHAR(32) COMMENT '手机号（UserProfile 冗余字段）',
  `level` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '用户等级',
  `level_desc` VARCHAR(128) COMMENT '等级描述',
  `score` DECIMAL(12,4) COMMENT '评分 / 信用分',
  `status` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '账号状态',
  `shop_type` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '店铺类型（卖家标识）',
  `is_douli_user` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '是否斗牛用户',
  `is_blocked` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '是否被拉黑',
  `is_following` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '我关注他（原字段名 followed）',
  `is_followed_by` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '他关注我',
  `follower_num` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '粉丝数',
  `following_num` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '关注数（仅用户）',
  `is_force_follow` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '强制关注',
  `review_liked_num` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '获赞数',
  `want_watch_num` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '想看数',
  `watched_num` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '看过数',
  `ext` JSON COMMENT '扩展字段 | 预留',
  `gmt_create` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `gmt_modified` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_user_info_user_id` (`user_id`),
  UNIQUE KEY `uk_user_info_mobile_no` (`mobile_no`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户主档 | Flutter AuthUser（原 users）' ROW_FORMAT=DYNAMIC;

CREATE TABLE IF NOT EXISTS `login_session` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `session_id` BIGINT UNSIGNED NOT NULL COMMENT '代理主键',
  `user_id` BIGINT UNSIGNED COMMENT '用户ID',
  `login_type` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '登录方式（密码 / 短信 / 三方） | v1/user/login',
  `access_token` VARCHAR(512) NOT NULL COMMENT '访问令牌',
  `union_id` BIGINT UNSIGNED COMMENT '三方 UnionID',
  `is_bound` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '是否已绑定',
  `is_new_user` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '是否新用户',
  `is_need_captcha` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '是否需要图形验证码',
  `captcha_cooldown` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '验证码冷却秒数（客户端 60 秒倒计时依据）',
  `error_msg` VARCHAR(512) COMMENT '错误提示 | SendCodeResponseDto.errorMsg',
  `is_success` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '请求是否成功 | SendCodeResponseDto.success',
  `device_id` BIGINT UNSIGNED COMMENT '设备标识',
  `expire_time` DATETIME COMMENT '过期时间 | 推断',
  `gmt_create` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `gmt_modified` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_login_session_session_id` (`session_id`),
  UNIQUE KEY `uk_login_session_access_token` (`access_token`),
  KEY `idx_login_session_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='登录会话 / 三方登录 / 验证码 | SendCodeResponseDto（原 login_sessions）' ROW_FORMAT=DYNAMIC;

CREATE TABLE IF NOT EXISTS `real_name_info` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `real_name_id` BIGINT UNSIGNED NOT NULL COMMENT '实名信息ID',
  `user_id` BIGINT UNSIGNED NOT NULL COMMENT '所属用户',
  `name` VARCHAR(128) COMMENT '姓名',
  `id_card_no` VARCHAR(64) NOT NULL COMMENT '证件号 | Visitor.idCard.value',
  `id_card_type` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '证件类型（1 身份证 / 2 护照 / 3 港澳通行证 / 4 台湾通行证 / 5 港澳台居住证） | IdCardType',
  `country_area_id` BIGINT UNSIGNED COMMENT '国家/地区ID',
  `country_area_name` VARCHAR(128) COMMENT '国家/地区名',
  `status` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '状态 | 推断',
  `gmt_create` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `gmt_modified` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_real_name_info_real_name_id` (`real_name_id`),
  UNIQUE KEY `uk_real_name_info_user_id_id_card_no` (`user_id`,`id_card_no`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='实名信息 / 观演人 | Visitor（原 real_name_infos）' ROW_FORMAT=DYNAMIC;

CREATE TABLE IF NOT EXISTS `user_address` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `address_id` BIGINT UNSIGNED NOT NULL COMMENT '地址ID',
  `user_id` BIGINT UNSIGNED NOT NULL COMMENT '用户ID',
  `name` VARCHAR(128) COMMENT '收件人',
  `phone` VARCHAR(32) COMMENT '联系电话',
  `province` VARCHAR(64) COMMENT '省 | Flutter ShippingAddress.region',
  `city` VARCHAR(64) COMMENT '市',
  `district` VARCHAR(64) COMMENT '区',
  `district_id` BIGINT UNSIGNED COMMENT '区县ID',
  `detail_address` TEXT COMMENT '详细地址',
  `address_type` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '地址类型',
  `is_default` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '是否默认',
  `is_recent` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '是否最近使用',
  `latest_id` BIGINT UNSIGNED COMMENT '最近地址ID（MyAddressBean）',
  `receiver_name` VARCHAR(128) COMMENT '最近收件人',
  `receiver_phone` VARCHAR(32) COMMENT '最近收件电话',
  `status` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '状态',
  `gmt_create` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `gmt_modified` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_user_address_address_id` (`address_id`),
  KEY `idx_user_address_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='收货地址 | Flutter ShippingAddress（原 user_addresses）' ROW_FORMAT=DYNAMIC;

CREATE TABLE IF NOT EXISTS `user_follow` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '代理主键',
  `user_id` BIGINT UNSIGNED NOT NULL COMMENT '发起人',
  `target_type` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '目标类型（1 用户 / 2 艺人 / 3 群组） | 推断',
  `target_id` BIGINT UNSIGNED NOT NULL COMMENT '目标ID',
  `status` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '状态 | 推断',
  `gmt_create` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `gmt_modified` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_user_follow_user_id_target_type_target_id` (`user_id`,`target_type`,`target_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='关注关系（用户 / 艺人）| v1/user/follow · v1/actorFollow · following/actors · following/mine（原 user_follows）' ROW_FORMAT=DYNAMIC;

CREATE TABLE IF NOT EXISTS `user_statistic` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `user_id` BIGINT UNSIGNED NOT NULL COMMENT '用户ID',
  `actor_following` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '关注艺人数（与 following_num 相加为总关注数）',
  `balance` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '钱包余额（元）',
  `coupon_num` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '优惠券张数',
  `follower_num` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '粉丝数',
  `following_num` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '关注用户数',
  `getting_tickets_num` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '待收货',
  `gift_card_num` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '礼品卡张数',
  `msg_num` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '未读消息数',
  `paying_num` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '待付款',
  `review_num` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '评论数',
  `review_todo_num` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '待评论',
  `waiting_for_tickets_num` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '待发货',
  `want_watch_num` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '想看数',
  `gmt_create` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `gmt_modified` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_user_statistic_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户中心角标统计（原 user_statistics）' ROW_FORMAT=DYNAMIC;

CREATE TABLE IF NOT EXISTS `user_message` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `message_id` BIGINT UNSIGNED NOT NULL COMMENT '消息ID',
  `user_id` BIGINT UNSIGNED NOT NULL COMMENT '接收用户',
  `from_user_id` BIGINT UNSIGNED COMMENT '发送用户',
  `from_user` JSON COMMENT '发送人快照',
  `title` VARCHAR(512) COMMENT '标题',
  `content` TEXT COMMENT '正文',
  `msg_type` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '消息类型（见 dict_enum.UserMsgType）',
  `status` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '状态',
  `url` VARCHAR(1024) COMMENT '跳转链接',
  `unread_count` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '未读数',
  `gmt_create` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `gmt_modified` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_user_message_message_id` (`message_id`),
  KEY `idx_user_message_user_id_msg_type` (`user_id`,`msg_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='消息中心 | SystemMessageEntry（原 user_messages）' ROW_FORMAT=DYNAMIC;

CREATE TABLE IF NOT EXISTS `chat_thread` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `thread_id` BIGINT UNSIGNED NOT NULL COMMENT '会话ID | Flutter MessageThread.id',
  `user_id` BIGINT UNSIGNED NOT NULL COMMENT '用户ID',
  `title` VARCHAR(256) COMMENT '会话标题 | MessageThread.title',
  `preview` VARCHAR(1024) COMMENT '最后一条预览 | MessageThread.preview',
  `unread_count` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '未读数 | MessageThread.unreadCount',
  `avatar_url` VARCHAR(512) COMMENT '头像（开源图床 URL） | MessageThread.avatarFallbackUrl',
  `is_official` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '是否官方客服 | MessageThread.isOfficial',
  `visitor_token` VARCHAR(256) COMMENT '环信 visitor token',
  `gmt_create` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `gmt_modified` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_chat_thread_thread_id` (`thread_id`),
  KEY `idx_chat_thread_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='客服 / 会话线程 | Flutter MessageThread（原 chat_threads）' ROW_FORMAT=DYNAMIC;

CREATE TABLE IF NOT EXISTS `chat_message` (
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
