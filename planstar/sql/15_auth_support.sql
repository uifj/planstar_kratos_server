-- ========================================================================
--  planstar · 15_auth_support.sql
--  认证支撑（docs/sql 基线未覆盖的部分，均为幂等脚本）
--    1) user_info.password_hash（密码登录）
--    2) login_code（短信验证码）
--    3) banner.title（契约 Banner.title 字段）
--    4) 演示数据补齐：user_statistic / banner / review / review_reply
--  演示账号：13800000001 / 13800000002，密码 123456
-- ========================================================================

-- 显式声明客户端字符集：若历史导入时连接为 latin1，中文会被双重编码。
SET NAMES utf8mb4;

-- ---------------------------------------------------------------------
-- 1. user_info.password_hash
-- ---------------------------------------------------------------------
SET @has_pwd := (
  SELECT COUNT(*) FROM information_schema.columns
   WHERE table_schema = 'planstar_user' AND table_name = 'user_info' AND column_name = 'password_hash'
);
SET @ddl := IF(@has_pwd = 0,
  'ALTER TABLE `planstar_user`.`user_info` ADD COLUMN `password_hash` VARCHAR(256) NOT NULL DEFAULT '''' COMMENT ''密码哈希（PBKDF2-SHA256）'' AFTER `mobile_no`',
  'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

UPDATE `planstar_user`.`user_info`
   SET `password_hash` = '$pbkdf2-sha256$10000$706c616e737461722d64656d6f2d3031$99af854a921442453fd86620f95b2618878727d5ab6640c158e0d74c7bae17d0'
 WHERE `password_hash` = '';

-- ---------------------------------------------------------------------
-- 2. login_code 短信验证码
-- ---------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `planstar_user`.`login_code` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `mobile_no` VARCHAR(32) NOT NULL COMMENT '手机号 | LoginRequest.ua',
  `code` VARCHAR(16) NOT NULL COMMENT '验证码',
  `expire_time` DATETIME NOT NULL COMMENT '过期时间',
  `gmt_create` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `gmt_modified` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  KEY `idx_login_code_mobile` (`mobile_no`, `code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='短信登录验证码（认证支撑表）' ROW_FORMAT=DYNAMIC;

-- ---------------------------------------------------------------------
-- 3. banner.title（契约 Banner.title；docs/sql 的 banner 无标题列）
-- ---------------------------------------------------------------------
SET @has_title := (
  SELECT COUNT(*) FROM information_schema.columns
   WHERE table_schema = 'planstar_content' AND table_name = 'banner' AND column_name = 'title'
);
SET @ddl := IF(@has_title = 0,
  'ALTER TABLE `planstar_content`.`banner` ADD COLUMN `title` VARCHAR(256) NOT NULL DEFAULT '''' COMMENT ''标题 | Banner.title'' AFTER `banner_id`',
  'SELECT 1');
PREPARE stmt FROM @ddl; EXECUTE stmt; DEALLOCATE PREPARE stmt;

-- ---------------------------------------------------------------------
-- 4. 演示数据补齐
-- ---------------------------------------------------------------------

-- 4.1 用户资产统计（profile 的余额/券/关注数来源）
INSERT INTO `planstar_user`.`user_statistic`
  (`user_id`, `actor_following`, `balance`, `coupon_num`, `follower_num`, `following_num`,
   `getting_tickets_num`, `gift_card_num`, `msg_num`, `paying_num`, `review_num`, `review_todo_num`,
   `waiting_for_tickets_num`, `want_watch_num`, `gmt_create`, `gmt_modified`)
SELECT u.`user_id`, 0, 100.00, 3, COALESCE(u.`follower_num`, 0), COALESCE(u.`following_num`, 0),
       0, 0, 0, 0, 0, 0, 0, COALESCE(u.`want_watch_num`, 0), NOW(), NOW()
  FROM `planstar_user`.`user_info` u
 WHERE NOT EXISTS (SELECT 1 FROM `planstar_user`.`user_statistic` s WHERE s.`user_id` = u.`user_id`);

-- 4.2 首页轮播
INSERT INTO `planstar_content`.`banner`
  (`banner_id`, `title`, `type`, `city_id`, `platform`, `poster`, `bg_color`, `jump_schema`, `url`,
   `sort`, `status`, `from_time`, `to_time`, `gmt_create`, `gmt_modified`)
SELECT 1001, '本周热门 · 周杰伦上海站', 1, COALESCE(a.`city_id`, 0), 0, COALESCE(a.`poster`, a.`img`, ''),
       '#111111', 'planstar://activity/1001', '', 1, 1, NOW(), DATE_ADD(NOW(), INTERVAL 1 YEAR), NOW(), NOW()
  FROM `planstar_catalog`.`activity` a WHERE a.`activity_id` = 1001
ON DUPLICATE KEY UPDATE `title` = VALUES(`title`), `poster` = VALUES(`poster`), `status` = 1;

INSERT INTO `planstar_content`.`banner`
  (`banner_id`, `title`, `type`, `city_id`, `platform`, `poster`, `bg_color`, `jump_schema`, `url`,
   `sort`, `status`, `from_time`, `to_time`, `gmt_create`, `gmt_modified`)
SELECT 1002, '音乐剧季 · 剧院魅影中文版', 1, COALESCE(a.`city_id`, 0), 0, COALESCE(a.`poster`, a.`img`, ''),
       '#1a1a2e', 'planstar://activity/2001', '', 2, 1, NOW(), DATE_ADD(NOW(), INTERVAL 1 YEAR), NOW(), NOW()
  FROM `planstar_catalog`.`activity` a WHERE a.`activity_id` = 2001
ON DUPLICATE KEY UPDATE `title` = VALUES(`title`), `poster` = VALUES(`poster`), `status` = 1;

INSERT INTO `planstar_content`.`banner`
  (`banner_id`, `title`, `type`, `city_id`, `platform`, `poster`, `bg_color`, `jump_schema`, `url`,
   `sort`, `status`, `from_time`, `to_time`, `gmt_create`, `gmt_modified`)
SELECT 1003, '沉浸体验 · teamLab 无界', 1, COALESCE(a.`city_id`, 0), 0, COALESCE(a.`poster`, a.`img`, ''),
       '#0b3d2e', 'planstar://activity/5001', '', 3, 1, NOW(), DATE_ADD(NOW(), INTERVAL 1 YEAR), NOW(), NOW()
  FROM `planstar_catalog`.`activity` a WHERE a.`activity_id` = 5001
ON DUPLICATE KEY UPDATE `title` = VALUES(`title`), `poster` = VALUES(`poster`), `status` = 1;

-- 4.3 动态评论（userReply 依赖 review；activity 对应 feed 1/2/3）
INSERT INTO `planstar_community`.`review`
  (`review_id`, `user_id`, `activity_id`, `content`, `rating`, `images`, `videos`, `is_liked`, `likes`,
   `reply_count`, `status`, `user`, `gmt_create`, `gmt_modified`)
VALUES
  (1, 2, 2001, '吊灯落下的那一刻全场屏住呼吸，中文版的唱段也很稳，值回票价。', 5, '[]', '[]', 0, 12, 2, 1,
   JSON_OBJECT('userId', 2, 'userName', '小鹿看剧', 'avatar', ''), NOW(), NOW()),
  (2, 1, 1001, '按这个攻略卡点抢到了看台 219 区，视野比想象中好。', 5, '[]', '[]', 0, 9, 1, 1,
   JSON_OBJECT('userId', 1, 'userName', '阿哲', 'avatar', ''), NOW(), NOW()),
  (3, 2, 5001, '投影区确实最好拍，建议穿浅色衣服，出片率高很多。', 4, '[]', '[]', 0, 6, 0, 1,
   JSON_OBJECT('userId', 2, 'userName', '小鹿看剧', 'avatar', ''), NOW(), NOW())
ON DUPLICATE KEY UPDATE `content` = VALUES(`content`), `user` = VALUES(`user`), `status` = 1;

INSERT INTO `planstar_community`.`review_reply`
  (`reply_id`, `review_id`, `user_id`, `reply_to_user_id`, `reply_to_user_name`, `content`, `is_top`, `user`,
   `gmt_create`, `gmt_modified`)
VALUES
  (1, 1, 1, 2, '小鹿看剧', '同场！返场那首直接泪目。', 1,
   JSON_OBJECT('userId', 1, 'userName', '阿哲', 'avatar', ''), NOW(), NOW()),
  (2, 1, 2, 1, '阿哲', '你买的是池座还是楼座？', 0,
   JSON_OBJECT('userId', 2, 'userName', '小鹿看剧', 'avatar', ''), NOW(), NOW())
ON DUPLICATE KEY UPDATE `content` = VALUES(`content`), `reply_to_user_name` = VALUES(`reply_to_user_name`),
  `user` = VALUES(`user`), `is_top` = VALUES(`is_top`);
