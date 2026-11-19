-- 生产版本占位文件：覆盖 base compose 的 14_mock_seed.sql 挂载点。
-- 按文件名 14_mock_seed.sql 执行，内容为空，确保生产库不注入任何演示业务数据。
-- 本地版本（docker-compose.local.yml）不使用本文件，仍挂载真实演示种子。
SELECT 'planstar production: demo seed intentionally skipped' AS init_note;
