USE shop_inventory;

-- =====================================================================
-- 实时库存表
-- =====================================================================
CREATE TABLE IF NOT EXISTS inventories (
    id          BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    sku_id      BIGINT      NOT NULL UNIQUE,
    inventory   BIGINT      NOT NULL DEFAULT 0,
    locked      BIGINT      NOT NULL DEFAULT 0,
    version     BIGINT      NOT NULL DEFAULT 0,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- =====================================================================
-- 库存预占记录表
-- =====================================================================
CREATE TABLE IF NOT EXISTS inventory_locks (
    id          BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    order_sn    VARCHAR(64) NOT NULL,
    sku_id      BIGINT      NOT NULL,
    num         INT         NOT NULL DEFAULT 0,
    status      INT         NOT NULL DEFAULT 1,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uk_inventory_locks UNIQUE (order_sn, sku_id)
);

CREATE INDEX idx_inventory_locks_sku ON inventory_locks (sku_id);

-- =====================================================================
-- 库存流水表
-- =====================================================================
CREATE TABLE IF NOT EXISTS inventory_flows (
    id          BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    order_sn    VARCHAR(64) NOT NULL,
    sku_id      BIGINT      NOT NULL,
    `change`    BIGINT      NOT NULL DEFAULT 0,
    type        VARCHAR(32) NOT NULL,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_inventory_flows_order ON inventory_flows (order_sn);
CREATE INDEX idx_inventory_flows_sku ON inventory_flows (sku_id);

-- =====================================================================
-- 消息消费幂等表
-- =====================================================================
CREATE TABLE IF NOT EXISTS consumed_event (
    event_id    VARCHAR(64) PRIMARY KEY,
    order_sn    VARCHAR(64) NOT NULL DEFAULT '',
    consumed_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- =====================================================================
-- Mock 库存数据（与 shop_goods.goods_skus 对应）
-- =====================================================================
INSERT IGNORE INTO inventories (id, sku_id, inventory, locked, version)
VALUES
    (1, 1, 50, 0, 0),
    (2, 2, 30, 0, 0),
    (3, 3, 60, 0, 0),
    (4, 4, 40, 0, 0);
