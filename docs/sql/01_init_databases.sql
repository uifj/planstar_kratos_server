-- ========================================================================
--  sql · 01_init_databases.sql（S2 全量规范重构版）
--  内容: 9 个服务库，统一 utf8mb4 / utf8mb4_unicode_ci
--  规范: 单数表名 / id BIGINT UNSIGNED 自增 / gmt_create,gmt_modified /
--        金额 DECIMAL / 无物理外键 / 索引内联到建表语句
-- ========================================================================

-- ========================================================================
--  01_init_databases.sql · 初始化企划星各服务库
-- ========================================================================
--  目录结构参考: github.com/arixbit/kratos-shop/sql/01_init_databases.sql
--  差异: 参考仓库用 PostgreSQL 的 \gexec 幂等建库；本文件保持 MySQL 8.0，
--        使用 CREATE DATABASE IF NOT EXISTS，语义等价且可直接执行。
--  本文件建 9 个基线库（见下）；planstar_shop 由 planstar/sql/18_star_coin_shop.sql 增建。
-- ========================================================================

-- planstar_user · 用户与账号
CREATE DATABASE IF NOT EXISTS `planstar_user` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- planstar_catalog · 基础字典与企划商品
CREATE DATABASE IF NOT EXISTS `planstar_catalog` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- planstar_ticket · 回报档位 / 座位 / 库存
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
