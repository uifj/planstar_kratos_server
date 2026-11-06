USE shop_order;

-- =====================================================================
-- 订单表
-- =====================================================================
CREATE TABLE IF NOT EXISTS orders (
    id             BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    `user`         BIGINT       NOT NULL DEFAULT 0,
    order_sn       VARCHAR(30)  NOT NULL DEFAULT '',
    order_amount   BIGINT       NOT NULL DEFAULT 0,
    goods_amount   BIGINT       NOT NULL DEFAULT 0,
    order_status   INT          NOT NULL DEFAULT 0 COMMENT '0: 库存处理中, 1: 待支付, 2: 已支付, 3: 已发货, 4: 已签收, 5: 已取消, 6: 交易完成, 7: 已退款',
    express_amount BIGINT       NOT NULL DEFAULT 0,
    delivery_at    DATETIME,
    refund_time    DATETIME,
    post           VARCHAR(200) NOT NULL DEFAULT '',
    created_at     DATETIME  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at     DATETIME  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at     DATETIME
);

CREATE INDEX idx_orders_user ON orders (`user`);
CREATE UNIQUE INDEX uk_orders_sn ON orders (order_sn);

-- =====================================================================
-- 订单收货地址表
-- =====================================================================
CREATE TABLE IF NOT EXISTS order_address (
    id               BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    `user`           BIGINT       NOT NULL DEFAULT 0,
    order_sn         VARCHAR(30)  NOT NULL DEFAULT '',
    recipient_name   VARCHAR(20)  NOT NULL DEFAULT '',
    recipient_mobile VARCHAR(20)  NOT NULL DEFAULT '',
    province         VARCHAR(25)  NOT NULL DEFAULT '',
    city             VARCHAR(25)  NOT NULL DEFAULT '',
    districts        VARCHAR(25)  NOT NULL DEFAULT '',
    address          VARCHAR(255) NOT NULL DEFAULT '',
    post_code        VARCHAR(25)  NOT NULL DEFAULT '',
    created_at       DATETIME  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at       DATETIME  NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_order_address_user ON order_address (`user`);
CREATE INDEX idx_order_address_sn ON order_address (order_sn);

-- =====================================================================
-- 订单商品表
-- =====================================================================
CREATE TABLE IF NOT EXISTS order_goods (
    id          BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    order_sn    VARCHAR(30)  NOT NULL DEFAULT '',
    user_id     BIGINT       NOT NULL DEFAULT 0,
    sku_id      BIGINT       NOT NULL DEFAULT 0,
    sku_name    VARCHAR(100) NOT NULL DEFAULT '',
    sku_price   BIGINT       NOT NULL DEFAULT 0,
    num         INT          NOT NULL DEFAULT 0,
    total_price BIGINT       NOT NULL DEFAULT 0,
    created_at  DATETIME  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME  NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_order_goods_sn ON order_goods (order_sn);
CREATE INDEX idx_order_goods_user ON order_goods (user_id);
CREATE INDEX idx_order_goods_sku ON order_goods (sku_id);

-- =====================================================================
-- 订单支付表
-- =====================================================================
CREATE TABLE IF NOT EXISTS order_pays (
    id         BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    `user`     BIGINT       NOT NULL DEFAULT 0,
    order_sn   VARCHAR(30)  NOT NULL DEFAULT '',
    trade_no   VARCHAR(100) NOT NULL DEFAULT '',
    pay_type   VARCHAR(20)  NOT NULL DEFAULT '' COMMENT 'alipay: 支付宝, wechat: 微信',
    pay_status INT          NOT NULL DEFAULT 0 COMMENT '0: 默认, 1: 待支付, 2: 成功, 3: 超时关闭, 4: 交易创建, 5: 交易结束',
    pay_time   DATETIME,
    created_at DATETIME  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME  NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_order_pays_user ON order_pays (`user`);
CREATE INDEX idx_order_pays_sn ON order_pays (order_sn);

-- =====================================================================
-- 订单事件 Outbox 表
-- =====================================================================
CREATE TABLE IF NOT EXISTS order_event_outbox (
    id           BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    event_id     VARCHAR(64) NOT NULL UNIQUE,
    event_type   VARCHAR(32) NOT NULL,
    order_sn     VARCHAR(30) NOT NULL DEFAULT '',
    payload      TEXT        NOT NULL,
    traceparent  VARCHAR(64) DEFAULT NULL,
    status       SMALLINT    NOT NULL DEFAULT 0,
    retry_count  INT         NOT NULL DEFAULT 0,
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    published_at DATETIME
);

CREATE INDEX idx_order_event_outbox_status ON order_event_outbox (status);

-- =====================================================================
-- Mock 基础数据
-- =====================================================================
INSERT IGNORE INTO orders (id, `user`, order_sn, order_amount, goods_amount, order_status, express_amount, post, created_at, updated_at)
VALUES
    (1, 1, '20260808000001', 1459800, 1449800, 2, 10000, '尽快发货', NOW(), NOW());

INSERT IGNORE INTO order_address (id, `user`, order_sn, recipient_name, recipient_mobile, province, city, districts, address, post_code)
VALUES
    (1, 1, '20260808000001', '张三', '13800138000', '广东省', '深圳市', '南山区', '科技园路 1 号', '518000');

INSERT IGNORE INTO order_goods (id, order_sn, user_id, sku_id, sku_name, sku_price, num, total_price)
VALUES
    (1, '20260808000001', 1, 1, 'iPhone 15 Pro 黑色 128G', 899900, 1, 899900),
    (2, '20260808000001', 1, 4, '联想小新 Pro 16 灰色 512G SSD', 549900, 1, 549900);

INSERT IGNORE INTO order_pays (id, `user`, order_sn, trade_no, pay_type, pay_status, pay_time)
VALUES
    (1, 1, '20260808000001', 'TRADE202608080001', 'alipay', 2, NOW());
