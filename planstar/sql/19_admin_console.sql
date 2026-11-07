-- ========================================================================
--  planstar · 19_admin_console.sql
--  管理后台（planstar/admin）数据库与种子数据（幂等，可重复执行）
--    1) planstar_admin.admins          管理员账号
--    2) planstar_admin.admin_audit_log 管理操作审计
--    3) 种子管理员 admin / admin@planstar.top
--  说明：本文件只做增量补丁，不修改 docs/sql 基线。
--        admins.id 使用 CHAR(36)，与 ent/uuid driver.Valuer 的 36 字符文本一致；
--        应用层按 UUIDv7 生成并直接以文本写入。
-- ========================================================================

SET NAMES utf8mb4;

CREATE DATABASE IF NOT EXISTS `planstar_admin`
  DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- ---------------------------------------------------------------------
-- 1. 管理员账号
--    - name / email 唯一，登录与找回密码都按这两列定位账号。
--    - status: 1 ACTIVE / 2 DISABLED / 3 DELETED（软删除）。
--    - must_change_password 首次登录强制改密；请勿在生产关闭。
--    - 时间列使用 DATETIME(6)，对应 ent TimeMixin 的 time.Time。
-- ---------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `planstar_admin`.`admins` (
  `id` CHAR(36) NOT NULL COMMENT '管理员ID（UUIDv7 文本，与 ent/uuid driver.Valuer 一致）',
  `created_at` DATETIME(6) NOT NULL COMMENT '创建时间',
  `updated_at` DATETIME(6) NOT NULL COMMENT '更新时间',
  `name` VARCHAR(255) NOT NULL COMMENT '登录名',
  `email` VARCHAR(255) NOT NULL COMMENT '邮箱',
  `avatar` VARCHAR(512) NOT NULL DEFAULT '' COMMENT '头像 URL',
  `access` VARCHAR(32) NOT NULL DEFAULT '' COMMENT '访问角色（admin 等）',
  `password` VARCHAR(255) NOT NULL DEFAULT '' COMMENT 'bcrypt 密码哈希',
  `status` INT NOT NULL DEFAULT 1 COMMENT '状态：1 正常 / 2 停用 / 3 删除',
  `must_change_password` TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否强制改密：1 是 / 0 否',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_admins_name` (`name`),
  UNIQUE KEY `uk_admins_email` (`email`),
  KEY `admin_updated_at` (`updated_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='管理员账号' ROW_FORMAT=DYNAMIC;

-- ---------------------------------------------------------------------
-- 2. 管理操作审计
--    - admin_id 允许 NULL：未登录或系统任务产生的审计也保留。
--    - 两组联合索引分别支撑“按管理员回溯”和“按动作回溯”。
-- ---------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `planstar_admin`.`admin_audit_log` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `admin_id` CHAR(36) NULL COMMENT '管理员ID（可为空）',
  `actor` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '操作者标识',
  `action` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '动作',
  `target` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '目标资源',
  `result` VARCHAR(32) NOT NULL DEFAULT '' COMMENT '结果（success/failure 等）',
  `detail` TEXT NULL COMMENT '详情',
  `remote_addr` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '来源 IP',
  `created_at` DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) COMMENT '创建时间',
  PRIMARY KEY (`id`),
  KEY `idx_admin_audit_log_admin_created` (`admin_id`, `created_at`),
  KEY `idx_admin_audit_log_action_created` (`action`, `created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='管理操作审计' ROW_FORMAT=DYNAMIC;

-- ---------------------------------------------------------------------
-- 3. 种子管理员
--    账号：admin
--    初始密码明文：PlanStar@Init2026
--    密码哈希：bcrypt cost 12（$2a$12$...，由 Go x/crypto/bcrypt 生成）
--    首次登录必须改密（must_change_password=1）；该口令仅用于首次部署，
--    投入使用前必须由管理员在控制台修改，严禁长期保留。
--
--    INSERT ... SELECT ... WHERE NOT EXISTS 保证脚本可重复执行：
--    已存在同名或同邮箱账号时不会插入第二条记录。
-- ---------------------------------------------------------------------
INSERT INTO `planstar_admin`.`admins`
  (`id`, `created_at`, `updated_at`, `name`, `email`, `avatar`, `access`, `password`, `status`, `must_change_password`)
SELECT
  '01920000-0000-7000-8000-000000000001',
  NOW(6),
  NOW(6),
  'admin',
  'admin@planstar.top',
  '',
  'admin',
  '$2a$12$d3sioqDXZc2yM4Y8Y5L8FuqwMJtwvTbS2PjttxLVszZ8u7XlIZHjS',
  1,
  1
FROM DUAL
WHERE NOT EXISTS (
  SELECT 1
  FROM `planstar_admin`.`admins`
  WHERE `name` = 'admin' OR `email` = 'admin@planstar.top'
);

-- ========================================================================
--  最小权限账号示例（需人工执行；请先替换 <REPLACE_ME> 并确认来源网段）
--
--  背景：管理台需要读写 planstar_admin，同时只读访问各业务库用于运营查询；
--        仅 planstar_user.user_info 需要额外 UPDATE（例如人工修正用户资料）。
--  不建议直接复用业务库账号，也不建议授予 ALL ON *.*。
-- ========================================================================
--
-- CREATE USER IF NOT EXISTS 'planstar_admin'@'%' IDENTIFIED BY '<REPLACE_ME>';
--
-- -- 管理库：应用自身需要建表、读写与审计。
-- GRANT ALL PRIVILEGES ON `planstar_admin`.* TO 'planstar_admin'@'%';
--
-- -- 业务库：只读。
-- GRANT SELECT ON `planstar_user`.* TO 'planstar_admin'@'%';
-- GRANT SELECT ON `planstar_catalog`.* TO 'planstar_admin'@'%';
-- GRANT SELECT ON `planstar_order`.* TO 'planstar_admin'@'%';
-- GRANT SELECT ON `planstar_payment`.* TO 'planstar_admin'@'%';
-- GRANT SELECT ON `planstar_marketing`.* TO 'planstar_admin'@'%';
-- GRANT SELECT ON `planstar_community`.* TO 'planstar_admin'@'%';
-- GRANT SELECT ON `planstar_content`.* TO 'planstar_admin'@'%';
-- GRANT SELECT ON `planstar_meta`.* TO 'planstar_admin'@'%';
--
-- -- 额外写权限：仅用户资料表。
-- GRANT UPDATE ON `planstar_user`.`user_info` TO 'planstar_admin'@'%';
--
-- FLUSH PRIVILEGES;
