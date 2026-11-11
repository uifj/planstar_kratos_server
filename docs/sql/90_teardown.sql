-- ========================================================================
--  sql · 90_teardown.sql —— 逆序清理（破坏性）
--  会删除下列 9 个基线库及其数据（不含 planstar_shop / planstar_admin）；
--  仅在确认无业务数据后执行。
-- ========================================================================

SET FOREIGN_KEY_CHECKS = 0;
DROP DATABASE IF EXISTS `planstar_meta`;
DROP DATABASE IF EXISTS `planstar_content`;
DROP DATABASE IF EXISTS `planstar_community`;
DROP DATABASE IF EXISTS `planstar_marketing`;
DROP DATABASE IF EXISTS `planstar_payment`;
DROP DATABASE IF EXISTS `planstar_order`;
DROP DATABASE IF EXISTS `planstar_ticket`;
DROP DATABASE IF EXISTS `planstar_catalog`;
DROP DATABASE IF EXISTS `planstar_user`;
SET FOREIGN_KEY_CHECKS = 1;
