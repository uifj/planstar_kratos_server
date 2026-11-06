USE shop_payment;

CREATE TABLE IF NOT EXISTS payments (
    id          BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    payment_no  VARCHAR(64)  NOT NULL UNIQUE,
    order_sn    VARCHAR(64)  NOT NULL,
    user_id     BIGINT       NOT NULL DEFAULT 0,
    amount      BIGINT       NOT NULL DEFAULT 0,
    channel     VARCHAR(32)  NOT NULL DEFAULT 'mock',
    status      INT          NOT NULL DEFAULT 1,
    trade_no    VARCHAR(100) NOT NULL DEFAULT '',
    created_at  DATETIME  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    paid_at     DATETIME
);

CREATE INDEX idx_payments_order ON payments (order_sn);

INSERT IGNORE INTO payments (id, payment_no, order_sn, user_id, amount, channel, status)
VALUES (1, 'PAY-MOCK-00000001', '20260808013938373717', 1, 849900, 'mock', 2);
