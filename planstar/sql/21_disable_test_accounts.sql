-- ---------------------------------------------------------------------------
-- 生产版本专用：禁用本地演示/测试口令账号。
-- 按 docker-entrypoint-initdb.d 文件名字典序，本文件在 15_auth_support.sql 之后执行。
-- 本地版本（docker-compose.local.yml）**不挂载本文件**。
--
-- 对应《阿里巴巴Java开发手册（黄山版）》安全规约：
--   用户敏感数据/口令不得硬编码，生产不得保留可直接登录的测试账号。
-- 与应用层双重保险：cmd/planstar/main.go 的 AssertNoTestCredentials
-- 会扫描所有启用账号的 password_hash，命中测试口令即终止启动。
-- ---------------------------------------------------------------------------

-- 1. 演示账号：清空口令哈希并置为停用（密码登录分支校验空哈希必然失败）。
UPDATE `planstar_user`.`user_info`
   SET `password_hash` = '',
       `status` = 0,
       `gmt_modified` = NOW()
 WHERE `mobile_no` IN ('13800000001', '13800000002');

-- 2. 清理这些账号可能残留的会话与验证码。
DELETE s FROM `planstar_user`.`login_session` s
  JOIN `planstar_user`.`user_info` u ON u.`user_id` = s.`user_id`
 WHERE u.`mobile_no` IN ('13800000001', '13800000002');

DELETE FROM `planstar_user`.`login_code`;

-- 注意：生产新注册用户本来就是 password_hash=''（只允许短信码登录），
-- 不能按「空哈希」批量停用，否则会误伤正常用户。
-- 测试口令的兜底检测由应用启动断言 AssertNoTestCredentials 完成。
