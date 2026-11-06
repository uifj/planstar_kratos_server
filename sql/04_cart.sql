USE shop_cart;

-- =====================================================================
-- 购物车表
-- =====================================================================
CREATE TABLE IF NOT EXISTS shop_carts (
    id          BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    user_id     BIGINT       NOT NULL,
    goods_id    BIGINT       NOT NULL,
    sku_id      BIGINT       NOT NULL,
    goods_price BIGINT       NOT NULL DEFAULT 0,
    goods_num   INT          NOT NULL DEFAULT 1,
    goods_sn    VARCHAR(500) NOT NULL DEFAULT '',
    goods_name  VARCHAR(500) NOT NULL DEFAULT '',
    is_select   BOOLEAN      NOT NULL DEFAULT FALSE,
    add_time    DATETIME  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_time DATETIME  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at  DATETIME
);

CREATE INDEX idx_shop_carts_user ON shop_carts (user_id);
CREATE INDEX idx_shop_carts_sku ON shop_carts (sku_id);

-- =====================================================================
-- Mock 基础数据
-- =====================================================================
INSERT IGNORE INTO shop_carts (id, user_id, goods_id, sku_id, goods_price, goods_num, goods_sn, goods_name, is_select)
VALUES
    (1, 1, 1, 1, 899900, 1, 'SN-IPHONE-001', 'iPhone 15 Pro 黑色 128G', TRUE),
    (2, 1, 3, 4, 549900, 1, 'SN-LENOVO-001', '联想小新 Pro 16 灰色 512G SSD', TRUE);
