USE shop_user;

-- =====================================================================
-- 用户表（与 user 服务 ent schema 对齐）
-- =====================================================================
CREATE TABLE IF NOT EXISTS users (
    id         BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    mobile     VARCHAR(11)  NOT NULL UNIQUE COMMENT '手机号码，用户唯一标识',
    password   VARCHAR(200) NOT NULL,
    nick_name  VARCHAR(25)  NOT NULL DEFAULT '',
    gender     VARCHAR(16)  NOT NULL DEFAULT 'male' COMMENT 'female: 女, male: 男',
    role       INT          NOT NULL DEFAULT 1 COMMENT '1: 普通用户, 2: 管理员',
    birthday   DATETIME,
    deleted    BOOLEAN      NOT NULL DEFAULT FALSE
);

CREATE INDEX idx_users_deleted ON users (deleted);

-- =====================================================================
-- 用户收货地址表（与 user 服务 ent schema 对齐）
-- =====================================================================
CREATE TABLE IF NOT EXISTS user_addresses (
    id         BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    user_id    BIGINT       NOT NULL DEFAULT 1,
    is_default SMALLINT     NOT NULL DEFAULT 0,
    mobile     VARCHAR(11)  NOT NULL,
    name       VARCHAR(25)  NOT NULL DEFAULT '',
    province   VARCHAR(25)  NOT NULL DEFAULT '',
    city       VARCHAR(25)  NOT NULL DEFAULT '',
    districts  VARCHAR(25)  NOT NULL DEFAULT '',
    address    VARCHAR(255) NOT NULL DEFAULT '',
    post_code  VARCHAR(25)  NOT NULL DEFAULT '',
    deleted    BOOLEAN      NOT NULL DEFAULT FALSE
);

CREATE INDEX idx_user_addresses_user_deleted ON user_addresses (user_id, deleted);

-- =====================================================================
-- Mock 基础数据
-- =====================================================================
-- 密码均为 12345678
INSERT IGNORE INTO users (id, mobile, password, nick_name, birthday, gender, role, deleted)
VALUES
    (1, '13800138000', '$pbkdf2-sha512$781f496fd9233c152837110ac2ce8a79$98de99589c597e7fc51130b01055463b5dc0383c8670968c4611cc067e5e1ea6',
     '商城用户', '1995-06-15 00:00:00', 'male', 1, FALSE),
    (2, '13501167215', '$pbkdf2-sha512$781f496fd9233c152837110ac2ce8a79$98de99589c597e7fc51130b01055463b5dc0383c8670968c4611cc067e5e1ea6',
     '管理员', '1990-01-01 00:00:00', 'male', 2, FALSE);

INSERT IGNORE INTO user_addresses (id, user_id, is_default, mobile, name, province, city, districts, address, post_code, deleted)
VALUES
    (1, 1, 1, '13800138000', '张三', '广东省', '深圳市', '南山区', '科技园路 1 号', '518000', FALSE),
    (2, 1, 0, '13800138000', '张三', '广东省', '广州市', '天河区', '体育西路 100 号', '510000', FALSE),
    (3, 2, 1, '13501167215', '李四', '上海市', '上海市', '浦东新区', '世纪大道 2000 号', '200000', FALSE);
