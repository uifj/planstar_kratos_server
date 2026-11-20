-- ========================================================================
--  planstar · 22_app_release.sql
--  应用自更新版本发布表（幂等，可重复执行）
--    1) planstar_meta.app_release  各平台可更新包元数据（APK / EXE / IPA）
--  说明：包体放在自有 OSS/CDN，download_url 存直链；本表仅存元数据。
--        新增版本由运维/后台写入，客户端只读（GET /v1/update/check 与
--        /v1/update/appcast.xml 匿名访问）。
-- ========================================================================

SET NAMES utf8mb4;

-- ---------------------------------------------------------------------
-- 1. 应用版本发布表
--    uk_platform_channel_version 保证同一平台同一渠道同一版本号唯一，
--    重复写入触发唯一键冲突（幂等发布）。
--    force_upgrade 为显式强制升级；min_version_code 为「低于该版本即强制」
--    的软门槛，客户端比较当前 versionCode 判定。
-- ---------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS `planstar_meta`.`app_release` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `platform` VARCHAR(16) NOT NULL COMMENT '平台：android/windows/ios',
  `channel` VARCHAR(32) NOT NULL DEFAULT 'stable' COMMENT '渠道：stable/beta',
  `version_code` BIGINT NOT NULL COMMENT '版本号（整型递增，客户端据此比较）',
  `version_name` VARCHAR(64) NOT NULL COMMENT '版本名（展示用，如 1.0.2）',
  `download_url` VARCHAR(512) NOT NULL COMMENT '安装包直链（OSS/CDN）',
  `sha256` CHAR(64) NOT NULL COMMENT '安装包 SHA-256（小写十六进制）',
  `size_bytes` BIGINT NOT NULL COMMENT '包大小（字节）',
  `force_upgrade` TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否强制升级：1 是 / 0 否',
  `min_version_code` BIGINT NOT NULL DEFAULT 0 COMMENT '低于此版本号强制升级（0 表示不启用）',
  `changelog` TEXT COMMENT '更新日志（纯文本）',
  `released_at` DATETIME NOT NULL COMMENT '发布时间',
  `is_active` TINYINT(1) NOT NULL DEFAULT 1 COMMENT '是否生效：1 生效 / 0 下架',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_platform_channel_version` (`platform`,`channel`,`version_code`),
  KEY `idx_platform_channel_active` (`platform`,`channel`,`is_active`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='应用版本发布' ROW_FORMAT=DYNAMIC;
