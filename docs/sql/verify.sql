-- ========================================================================
--  sql · verify.sql（S2 全量规范重构版）
--  在 00_all.sql 执行完成后运行，复核 information_schema 是否符合阿里规约。
--  期望值：86 张业务表 + 3 张证据表 / 0 条物理外键 / 全 InnoDB+utf8mb4。
-- ========================================================================

SELECT VERSION() AS server_version;

SELECT SCHEMA_NAME, DEFAULT_CHARACTER_SET_NAME, DEFAULT_COLLATION_NAME
FROM information_schema.SCHEMATA
WHERE SCHEMA_NAME LIKE 'planstar_%' ORDER BY SCHEMA_NAME;

-- 业务表数（排除 3 张 _rev_* 证据表，期望 86）
SELECT COUNT(*) AS business_table_count FROM information_schema.TABLES
WHERE TABLE_SCHEMA LIKE 'planstar_%' AND TABLE_TYPE = 'BASE TABLE'
  AND TABLE_NAME NOT LIKE '\_rev\_%';

-- 全库表数（含证据表，期望 89）
SELECT COUNT(*) AS table_count FROM information_schema.TABLES
WHERE TABLE_SCHEMA LIKE 'planstar_%' AND TABLE_TYPE = 'BASE TABLE';

SELECT COUNT(*) AS column_count FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA LIKE 'planstar_%';

SELECT COUNT(DISTINCT TABLE_SCHEMA, TABLE_NAME, INDEX_NAME) AS index_count
FROM information_schema.STATISTICS WHERE TABLE_SCHEMA LIKE 'planstar_%';

-- 物理外键必须为 0
SELECT COUNT(*) AS fk_count FROM information_schema.TABLE_CONSTRAINTS
WHERE TABLE_SCHEMA LIKE 'planstar_%' AND CONSTRAINT_TYPE = 'FOREIGN KEY';

SELECT COUNT(*) AS fk_constraints_live FROM information_schema.REFERENTIAL_CONSTRAINTS
WHERE CONSTRAINT_SCHEMA LIKE 'planstar_%';

SELECT ENGINE, TABLE_COLLATION, COUNT(*) AS n FROM information_schema.TABLES
WHERE TABLE_SCHEMA LIKE 'planstar_%' AND TABLE_TYPE = 'BASE TABLE'
GROUP BY ENGINE, TABLE_COLLATION ORDER BY n DESC;

-- 缺注释列必须为 0
SELECT COUNT(*) AS uncommented_columns FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA LIKE 'planstar_%' AND (COLUMN_COMMENT IS NULL OR COLUMN_COMMENT = '');

-- 每张业务表必须具备 id / gmt_create / gmt_modified（结果为空即通过）
SELECT t.TABLE_SCHEMA, t.TABLE_NAME FROM information_schema.TABLES t
WHERE t.TABLE_SCHEMA LIKE 'planstar_%' AND t.TABLE_TYPE = 'BASE TABLE'
  AND t.TABLE_NAME NOT LIKE '\_rev\_%'
  AND (SELECT COUNT(*) FROM information_schema.COLUMNS c
       WHERE c.TABLE_SCHEMA = t.TABLE_SCHEMA AND c.TABLE_NAME = t.TABLE_NAME
         AND c.COLUMN_NAME IN ('id','gmt_create','gmt_modified')) < 3;

-- 单表二级索引数（不含主键）超过 5 的（结果为空即通过）
SELECT TABLE_SCHEMA, TABLE_NAME, COUNT(DISTINCT INDEX_NAME) AS idx_num
FROM information_schema.STATISTICS
WHERE TABLE_SCHEMA LIKE 'planstar_%' AND INDEX_NAME <> 'PRIMARY'
GROUP BY TABLE_SCHEMA, TABLE_NAME HAVING idx_num > 5;

-- 仍使用 FLOAT / DOUBLE 的列必须为 0
SELECT COUNT(*) AS float_double_columns FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA LIKE 'planstar_%' AND DATA_TYPE IN ('float','double');

-- 布尔列统一 TINYINT UNSIGNED
SELECT COUNT(*) AS bad_is_columns FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA LIKE 'planstar_%' AND COLUMN_NAME LIKE 'is\_%'
  AND DATA_TYPE <> 'tinyint';

-- 字典与证据种子行数
SELECT 'dict_enum' AS t, COUNT(*) AS n FROM `planstar_meta`.`dict_enum`
UNION ALL SELECT '_rev_dto_field_catalog', COUNT(*) FROM `planstar_meta`.`_rev_dto_field_catalog`
UNION ALL SELECT '_rev_endpoint_catalog', COUNT(*) FROM `planstar_meta`.`_rev_endpoint_catalog`
UNION ALL SELECT '_rev_dto_table_mapping', COUNT(*) FROM `planstar_meta`.`_rev_dto_table_mapping`;
