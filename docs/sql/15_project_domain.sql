-- ============================================================================
-- 15_project_domain.sql · 企划星领域增量 DDL
-- ----------------------------------------------------------------------------
-- 执行顺序：01..13 → 15_project_domain.sql → 14_mock_seed.sql
-- 定位说明：
--   1) 以 project（企划）为最小交易单元，reward_tier（回报档位）承载周边/内容/权益；
--   2) pledge（支持记录）替代票务订单成为核心交易凭证；
--   3) 既有 activity / ticket_* 表保留，降级为「线下活动型企划」的可选履约能力；
--   4) 本文件只新增表，不改写、不删除任何历史票务表。
-- ============================================================================

SET NAMES utf8mb4;

USE `planstar_catalog`;

-- ----------------------------------------------------------------------
-- project_category · 企划一级品类
-- ----------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `project_category` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `category_id` BIGINT UNSIGNED NOT NULL COMMENT '品类ID',
  `name` VARCHAR(64) NOT NULL COMMENT '品类名称',
  `en_name` VARCHAR(64) COMMENT '英文名',
  `icon` VARCHAR(1024) COMMENT '品类图标',
  `description` VARCHAR(512) COMMENT '品类说明',
  `sort` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '排序（越小越靠前）',
  `is_select` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '是否首页精选',
  `status` TINYINT UNSIGNED NOT NULL DEFAULT 1 COMMENT '状态：0隐藏 1启用',
  `gmt_create` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `gmt_modified` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_project_category_category_id` (`category_id`),
  KEY `idx_project_category_status_sort` (`status`, `sort`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='企划一级品类' ROW_FORMAT=DYNAMIC;

-- ----------------------------------------------------------------------
-- project_sub_tag · 企划二级标签
-- ----------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `project_sub_tag` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `sub_tag_id` BIGINT UNSIGNED NOT NULL COMMENT '标签ID',
  `category_id` BIGINT UNSIGNED NOT NULL COMMENT '所属一级品类ID',
  `name` VARCHAR(64) NOT NULL COMMENT '标签名称',
  `sort` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '排序',
  `status` TINYINT UNSIGNED NOT NULL DEFAULT 1 COMMENT '状态：0隐藏 1启用',
  `gmt_create` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `gmt_modified` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_project_sub_tag_sub_tag_id` (`sub_tag_id`),
  KEY `idx_project_sub_tag_category_id` (`category_id`, `status`, `sort`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='企划二级标签' ROW_FORMAT=DYNAMIC;

-- ----------------------------------------------------------------------
-- creator_profile · 创作者主页与信用
-- ----------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `creator_profile` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `creator_id` BIGINT UNSIGNED NOT NULL COMMENT '创作者ID',
  `user_id` BIGINT UNSIGNED NOT NULL COMMENT '关联用户ID',
  `display_name` VARCHAR(128) NOT NULL COMMENT '创作者昵称',
  `avatar` VARCHAR(1024) COMMENT '头像',
  `cover` VARCHAR(1024) COMMENT '主页封面',
  `bio` VARCHAR(512) COMMENT '简介',
  `is_verified` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '是否认证',
  `verified_type` VARCHAR(32) COMMENT '认证类型：studio/artist/writer/community/organization',
  `project_num` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '历史企划数',
  `success_num` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '达成企划数',
  `supporter_num` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '累计支持人数',
  `fulfillment_rate` DECIMAL(5,2) NOT NULL DEFAULT 0.00 COMMENT '履约率（百分比）',
  `avg_delivery_days` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '平均交付天数',
  `follower_num` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '关注者数',
  `status` TINYINT UNSIGNED NOT NULL DEFAULT 1 COMMENT '状态：0停用 1正常',
  `gmt_create` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `gmt_modified` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_creator_profile_creator_id` (`creator_id`),
  KEY `idx_creator_profile_user_id` (`user_id`),
  KEY `idx_creator_profile_verified` (`is_verified`, `status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='创作者主页与信用' ROW_FORMAT=DYNAMIC;

-- ----------------------------------------------------------------------
-- project · 企划主档
-- ----------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `project` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `project_id` BIGINT UNSIGNED NOT NULL COMMENT '企划ID',
  `project_no` VARCHAR(32) NOT NULL COMMENT '企划编号',
  `creator_id` BIGINT UNSIGNED NOT NULL COMMENT '创作者ID',
  `category_id` BIGINT UNSIGNED NOT NULL COMMENT '一级品类ID',
  `tag_ids` JSON COMMENT '二级标签ID列表',
  `title` VARCHAR(256) NOT NULL COMMENT '企划标题',
  `subtitle` VARCHAR(512) COMMENT '企划副标题',
  `cover` VARCHAR(1024) COMMENT '封面图',
  `gallery` JSON COMMENT '图集',
  `summary` VARCHAR(1024) COMMENT '一句话简介',
  `story` MEDIUMTEXT COMMENT '企划故事',
  `project_type` VARCHAR(32) NOT NULL COMMENT '企划类型：game/novel/fanwork/interview/boardgame/artbook/music/campus/other',
  `goal_amount` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '目标金额',
  `raised_amount` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '已筹金额',
  `supporter_count` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '支持人数',
  `start_time` DATETIME NOT NULL COMMENT '开始时间',
  `deadline` DATETIME NOT NULL COMMENT '截止时间',
  `funding_status` VARCHAR(24) NOT NULL DEFAULT 'ongoing' COMMENT '状态：preheating/ongoing/achieved/delivered/failed',
  `is_all_or_nothing` TINYINT UNSIGNED NOT NULL DEFAULT 1 COMMENT '未达成是否自动退款',
  `min_support_amount` DECIMAL(10,2) NOT NULL DEFAULT 0.00 COMMENT '最低支持金额',
  `risk_note` TEXT COMMENT '风险说明',
  `delivery_plan` TEXT COMMENT '交付计划',
  `fund_usage` TEXT COMMENT '资金用途',
  `refund_rule` TEXT COMMENT '未达成退款规则（对支持者可视化展示）',
  `refund_days` INT UNSIGNED NOT NULL DEFAULT 7 COMMENT '退款到账时限（天）',
  `faq` JSON COMMENT '常见问题',
  `is_featured` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '是否精选',
  `rank_weight` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '排序权重',
  `view_count` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '浏览数',
  `follow_count` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '关注数',
  `comment_count` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '评论数',
  `share_count` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '分享数',
  `publish_status` TINYINT UNSIGNED NOT NULL DEFAULT 1 COMMENT '发布状态：0草稿 1已发布 2审核中 3驳回',
  `status` TINYINT UNSIGNED NOT NULL DEFAULT 1 COMMENT '状态：0下线 1正常',
  `gmt_create` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `gmt_modified` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_project_project_id` (`project_id`),
  UNIQUE KEY `uk_project_project_no` (`project_no`),
  KEY `idx_project_category_status` (`category_id`, `funding_status`, `status`),
  KEY `idx_project_creator_id` (`creator_id`, `status`),
  KEY `idx_project_deadline` (`funding_status`, `deadline`),
  KEY `idx_project_rank_weight` (`funding_status`, `rank_weight`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='企划主档' ROW_FORMAT=DYNAMIC;

-- ----------------------------------------------------------------------
-- reward_tier · 回报档位
-- ----------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `reward_tier` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `reward_tier_id` BIGINT UNSIGNED NOT NULL COMMENT '回报档位ID',
  `project_id` BIGINT UNSIGNED NOT NULL COMMENT '企划ID',
  `tier_no` INT UNSIGNED NOT NULL COMMENT '档位序号',
  `title` VARCHAR(128) NOT NULL COMMENT '档位名称',
  `description` VARCHAR(1024) COMMENT '回报说明',
  `price` DECIMAL(10,2) NOT NULL COMMENT '支持价格',
  `reward_type` VARCHAR(24) NOT NULL COMMENT '回报类型：physical/digital/right/offline/mixed',
  `items` JSON COMMENT '回报明细',
  `limit_num` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '限量（0为不限）',
  `sold_num` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '已支持数',
  `remaining_num` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '剩余数量',
  `estimated_delivery_time` DATE COMMENT '预计交付日期',
  `is_early_bird` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '是否早鸟档',
  `is_limited` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '是否限量档',
  `sort` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '排序',
  `status` TINYINT UNSIGNED NOT NULL DEFAULT 1 COMMENT '状态：0停用 1启用',
  `gmt_create` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `gmt_modified` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_reward_tier_reward_tier_id` (`reward_tier_id`),
  KEY `idx_reward_tier_project_id` (`project_id`, `status`, `sort`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='回报档位' ROW_FORMAT=DYNAMIC;

-- ----------------------------------------------------------------------
-- project_update · 企划进度更新
-- ----------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `project_update` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `update_id` BIGINT UNSIGNED NOT NULL COMMENT '更新ID',
  `project_id` BIGINT UNSIGNED NOT NULL COMMENT '企划ID',
  `title` VARCHAR(256) NOT NULL COMMENT '标题',
  `content` TEXT COMMENT '正文',
  `images` JSON COMMENT '配图',
  `is_supporter_only` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '是否仅支持者可见',
  `view_count` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '浏览数',
  `like_count` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '点赞数',
  `publish_time` DATETIME NOT NULL COMMENT '发布时间',
  `status` TINYINT UNSIGNED NOT NULL DEFAULT 1 COMMENT '状态：0隐藏 1正常',
  `gmt_create` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `gmt_modified` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_project_update_update_id` (`update_id`),
  KEY `idx_project_update_project_time` (`project_id`, `publish_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='企划进度更新' ROW_FORMAT=DYNAMIC;

-- ----------------------------------------------------------------------
-- stretch_goal · 解锁目标
-- ----------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `stretch_goal` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `goal_id` BIGINT UNSIGNED NOT NULL COMMENT '解锁目标ID',
  `project_id` BIGINT UNSIGNED NOT NULL COMMENT '企划ID',
  `title` VARCHAR(256) NOT NULL COMMENT '目标标题',
  `description` VARCHAR(1024) COMMENT '目标说明',
  `target_amount` DECIMAL(12,2) NOT NULL COMMENT '解锁金额',
  `current_amount` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '当前金额',
  `is_achieved` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '是否已解锁',
  `sort` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '排序',
  `status` TINYINT UNSIGNED NOT NULL DEFAULT 1 COMMENT '状态',
  `gmt_create` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `gmt_modified` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_stretch_goal_goal_id` (`goal_id`),
  KEY `idx_stretch_goal_project_id` (`project_id`, `status`, `sort`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='企划解锁目标' ROW_FORMAT=DYNAMIC;

-- ----------------------------------------------------------------------
-- project_faq · 企划常见问题
-- ----------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `project_faq` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `faq_id` BIGINT UNSIGNED NOT NULL COMMENT 'FAQ ID',
  `project_id` BIGINT UNSIGNED NOT NULL COMMENT '企划ID',
  `question` VARCHAR(512) NOT NULL COMMENT '问题',
  `answer` TEXT NOT NULL COMMENT '回答',
  `sort` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '排序',
  `status` TINYINT UNSIGNED NOT NULL DEFAULT 1 COMMENT '状态',
  `gmt_create` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `gmt_modified` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_project_faq_faq_id` (`faq_id`),
  KEY `idx_project_faq_project_id` (`project_id`, `status`, `sort`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='企划常见问题' ROW_FORMAT=DYNAMIC;

-- ----------------------------------------------------------------------
-- project_review · 支持者评价与晒单
-- ----------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `project_review` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `review_id` BIGINT UNSIGNED NOT NULL COMMENT '评价ID',
  `project_id` BIGINT UNSIGNED NOT NULL COMMENT '企划ID',
  `pledge_id` BIGINT UNSIGNED COMMENT '关联支持记录ID（仅已支持用户可评价）',
  `user_id` BIGINT UNSIGNED NOT NULL COMMENT '评价用户ID',
  `rating` TINYINT UNSIGNED NOT NULL DEFAULT 5 COMMENT '评分 1-5',
  `content` VARCHAR(1024) COMMENT '评价内容',
  `images` JSON COMMENT '晒单图片',
  `is_show` TINYINT UNSIGNED NOT NULL DEFAULT 1 COMMENT '是否晒单（开箱/使用反馈）',
  `is_anonymous` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '是否匿名',
  `like_count` INT UNSIGNED NOT NULL DEFAULT 0 COMMENT '点赞数',
  `status` TINYINT UNSIGNED NOT NULL DEFAULT 1 COMMENT '状态：0隐藏 1正常 2审核中',
  `gmt_create` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `gmt_modified` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_project_review_review_id` (`review_id`),
  KEY `idx_project_review_project` (`project_id`, `status`, `gmt_create`),
  KEY `idx_project_review_pledge` (`pledge_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='支持者评价与晒单' ROW_FORMAT=DYNAMIC;

-- ----------------------------------------------------------------------
-- project_report · 举报与风控处理单
-- ----------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `project_report` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `report_id` BIGINT UNSIGNED NOT NULL COMMENT '举报ID',
  `target_type` VARCHAR(24) NOT NULL COMMENT '举报对象类型：project/review/feed/pledge',
  `target_id` BIGINT UNSIGNED NOT NULL COMMENT '举报对象ID',
  `project_id` BIGINT UNSIGNED COMMENT '关联企划ID',
  `user_id` BIGINT UNSIGNED NOT NULL COMMENT '举报人用户ID',
  `reason_code` VARCHAR(32) NOT NULL COMMENT '原因码：fraud/infringe/porn/violence/false_claim/other',
  `description` VARCHAR(1024) COMMENT '补充说明',
  `evidence` JSON COMMENT '证据图片',
  `status` VARCHAR(24) NOT NULL DEFAULT 'pending' COMMENT '状态：pending/accepted/rejected',
  `handle_note` VARCHAR(512) COMMENT '处理结论',
  `handle_time` DATETIME COMMENT '处理时间',
  `gmt_create` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `gmt_modified` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_project_report_report_id` (`report_id`),
  KEY `idx_project_report_target` (`target_type`, `target_id`),
  KEY `idx_project_report_status` (`status`, `gmt_create`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='举报与风控处理单' ROW_FORMAT=DYNAMIC;

-- ----------------------------------------------------------------------
-- support_team · 组队支持（原拼团心智改造）
-- ----------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `support_team` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `team_id` BIGINT UNSIGNED NOT NULL COMMENT '组队ID',
  `project_id` BIGINT UNSIGNED NOT NULL COMMENT '企划ID',
  `leader_user_id` BIGINT UNSIGNED NOT NULL COMMENT '队长用户ID',
  `title` VARCHAR(128) NOT NULL COMMENT '队伍名称',
  `target_amount` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '目标金额',
  `current_amount` DECIMAL(12,2) NOT NULL DEFAULT 0.00 COMMENT '当前金额',
  `member_num` INT UNSIGNED NOT NULL DEFAULT 1 COMMENT '成员数',
  `expire_time` DATETIME COMMENT '截止时间',
  `status` TINYINT UNSIGNED NOT NULL DEFAULT 1 COMMENT '状态：0关闭 1进行中 2已达成',
  `gmt_create` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `gmt_modified` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_support_team_team_id` (`team_id`),
  KEY `idx_support_team_project_id` (`project_id`, `status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='组队支持' ROW_FORMAT=DYNAMIC;

-- ----------------------------------------------------------------------
-- support_team_member · 组队成员
-- ----------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `support_team_member` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `member_id` BIGINT UNSIGNED NOT NULL COMMENT '成员ID',
  `team_id` BIGINT UNSIGNED NOT NULL COMMENT '组队ID',
  `user_id` BIGINT UNSIGNED NOT NULL COMMENT '用户ID',
  `pledge_id` BIGINT UNSIGNED COMMENT '关联支持记录ID',
  `support_amount` DECIMAL(10,2) NOT NULL DEFAULT 0.00 COMMENT '支持金额',
  `is_leader` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '是否队长',
  `join_time` DATETIME NOT NULL COMMENT '加入时间',
  `status` TINYINT UNSIGNED NOT NULL DEFAULT 1 COMMENT '状态',
  `gmt_create` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `gmt_modified` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_support_team_member_member_id` (`member_id`),
  KEY `idx_support_team_member_team_id` (`team_id`, `status`),
  KEY `idx_support_team_member_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='组队支持成员' ROW_FORMAT=DYNAMIC;

USE `planstar_order`;

-- ----------------------------------------------------------------------
-- pledge · 支持记录（企划支持单）
-- ----------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `pledge` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `pledge_id` BIGINT UNSIGNED NOT NULL COMMENT '支持记录ID',
  `pledge_no` VARCHAR(32) NOT NULL COMMENT '支持单号',
  `project_id` BIGINT UNSIGNED NOT NULL COMMENT '企划ID',
  `reward_tier_id` BIGINT UNSIGNED NOT NULL COMMENT '回报档位ID',
  `user_id` BIGINT UNSIGNED NOT NULL COMMENT '支持者用户ID',
  `support_amount` DECIMAL(10,2) NOT NULL COMMENT '支持金额',
  `quantity` INT UNSIGNED NOT NULL DEFAULT 1 COMMENT '数量',
  `is_anonymous` TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '是否匿名',
  `support_message` VARCHAR(255) COMMENT '支持留言',
  `contact_name` VARCHAR(64) COMMENT '联系人',
  `contact_phone` VARCHAR(32) COMMENT '联系电话',
  `address_id` BIGINT UNSIGNED COMMENT '收货地址ID',
  `delivery_type` VARCHAR(24) NOT NULL DEFAULT 'none' COMMENT '履约方式：physical/digital/right/offline/none',
  `pledge_status` VARCHAR(24) NOT NULL DEFAULT 'pending' COMMENT '状态：pending/paid/confirmed/shipped/delivered/refunded/cancelled',
  `payment_id` BIGINT UNSIGNED COMMENT '支付ID',
  `refund_id` BIGINT UNSIGNED COMMENT '退款ID',
  `tracking_no` VARCHAR(64) COMMENT '物流单号（仅 physical 履约使用）',
  `content_url` VARCHAR(1024) COMMENT '电子内容发放地址（仅 digital 履约使用）',
  `redeem_code` VARCHAR(64) COMMENT '权益核销码（仅 right/offline 履约使用）',
  `redeemed_time` DATETIME COMMENT '权益核销时间（仅 right/offline 履约使用）',
  `paid_time` DATETIME COMMENT '支付时间',
  `delivered_time` DATETIME COMMENT '交付时间',
  `gmt_create` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `gmt_modified` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_pledge_pledge_id` (`pledge_id`),
  UNIQUE KEY `uk_pledge_pledge_no` (`pledge_no`),
  KEY `idx_pledge_project_id` (`project_id`, `pledge_status`),
  KEY `idx_pledge_user_id` (`user_id`, `gmt_create`),
  KEY `idx_pledge_reward_tier_id` (`reward_tier_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='支持记录（企划支持单）' ROW_FORMAT=DYNAMIC;
