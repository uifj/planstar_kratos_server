-- Demo data for the folded Kratos v3 mall.
--
-- Run with `make seed` AFTER the app container has completed `auto_migrate`
-- (tables are created by Ent at application startup). All statements are
-- idempotent (`INSERT IGNORE`), so re-running `make seed` is safe.
--
-- ids are deterministic UUIDv7-shaped strings; business keys are unique
-- strings. Prices are in cents. status = 1 means "active" everywhere.

-- ---------------------------------------------------------------------------
-- Catalogue: categories and brands
-- ---------------------------------------------------------------------------
INSERT IGNORE INTO categories
  (id, created_at, updated_at, name, parent_category, level, sort, status)
VALUES
  ('019a0000-0000-7000-8000-000000000001', NOW(), NOW(), '手机数码', '', 1, 1, 1),
  ('019a0000-0000-7000-8000-000000000002', NOW(), NOW(), '电脑办公', '', 1, 2, 1),
  ('019a0000-0000-7000-8000-000000000003', NOW(), NOW(), '智能家居', '', 1, 3, 1);

INSERT IGNORE INTO brands
  (id, created_at, updated_at, name, logo, status)
VALUES
  ('019b0000-0000-7000-8000-000000000001', NOW(), NOW(), 'Planstar', 'https://example.com/brand/planstar.png', 1),
  ('019b0000-0000-7000-8000-000000000002', NOW(), NOW(), 'Nova', 'https://example.com/brand/nova.png', 1);

-- ---------------------------------------------------------------------------
-- Goods + SKUs
-- ---------------------------------------------------------------------------
INSERT IGNORE INTO goods
  (id, created_at, updated_at, category_id, brand_id, type_id, name, name_alias,
   goods_sn, goods_brief, goods_front_image, market_price, on_sale, ship_free,
   is_new, is_hot, sold_num, status)
VALUES
  ('019c0000-0000-7000-8000-000000000001', NOW(), NOW(),
   '019a0000-0000-7000-8000-000000000001', '019b0000-0000-7000-8000-000000000001', '',
   'Planstar Pro 手机', 'Planstar Pro', 'GSN-1001', '旗舰影像，全天续航',
   'https://example.com/goods/1001.png', 399900, 1, 1, 1, 1, 0, 1),
  ('019c0000-0000-7000-8000-000000000002', NOW(), NOW(),
   '019a0000-0000-7000-8000-000000000002', '019b0000-0000-7000-8000-000000000002', '',
   'Nova Air 笔记本', 'Nova Air', 'GSN-1002', '轻薄便携，长时办公',
   'https://example.com/goods/1002.png', 599900, 1, 1, 1, 0, 0, 1),
  ('019c0000-0000-7000-8000-000000000003', NOW(), NOW(),
   '019a0000-0000-7000-8000-000000000003', '019b0000-0000-7000-8000-000000000001', '',
   'Planstar 智能音箱', 'Smart Speaker', 'GSN-1003', '语音助手，全屋互联',
   'https://example.com/goods/1003.png', 29900, 1, 0, 0, 1, 0, 1);

INSERT IGNORE INTO goods_skus
  (id, created_at, updated_at, goods_id, sku_name, sku_code, bar_code, price,
   promotion_price, inventory, on_sale, pic, specs, status)
VALUES
  ('019d0000-0000-7000-8000-000000000001', NOW(), NOW(),
   '019c0000-0000-7000-8000-000000000001', 'Planstar Pro 8+256G 星空黑', 'SKU-1001-BK',
   'BAR-1001-BK', 399900, 379900, 100, 1, 'https://example.com/sku/1001.png', NULL, 1),
  ('019d0000-0000-7000-8000-000000000002', NOW(), NOW(),
   '019c0000-0000-7000-8000-000000000002', 'Nova Air 16+512G 银', 'SKU-1002-SI',
   'BAR-1002-SI', 599900, 569900, 50, 1, 'https://example.com/sku/1002.png', NULL, 1),
  ('019d0000-0000-7000-8000-000000000003', NOW(), NOW(),
   '019c0000-0000-7000-8000-000000000003', 'Planstar 智能音箱 白色', 'SKU-1003-WH',
   'BAR-1003-WH', 29900, 27900, 200, 1, 'https://example.com/sku/1003.png', NULL, 1);

-- One goods_images row per goods so GoodsDetailReply.goods_images is populated.
INSERT IGNORE INTO goods_images
  (id, created_at, updated_at, goods_id, url, position, is_master, status)
VALUES
  ('019e0000-0000-7000-8000-000000000001', NOW(), NOW(),
   '019c0000-0000-7000-8000-000000000001', 'https://example.com/goods/1001.png', 1, 1, 1),
  ('019e0000-0000-7000-8000-000000000002', NOW(), NOW(),
   '019c0000-0000-7000-8000-000000000002', 'https://example.com/goods/1002.png', 1, 1, 1),
  ('019e0000-0000-7000-8000-000000000003', NOW(), NOW(),
   '019c0000-0000-7000-8000-000000000003', 'https://example.com/goods/1003.png', 1, 1, 1);

-- ---------------------------------------------------------------------------
-- Inventory (one row per SKU, required for the FOR UPDATE row lock)
-- ---------------------------------------------------------------------------
INSERT IGNORE INTO inventories
  (id, created_at, updated_at, sku_id, total, locked, sold, version)
VALUES
  ('019f0000-0000-7000-8000-000000000001', NOW(), NOW(),
   '019d0000-0000-7000-8000-000000000001', 100, 0, 0, 0),
  ('019f0000-0000-7000-8000-000000000002', NOW(), NOW(),
   '019d0000-0000-7000-8000-000000000002', 50, 0, 0, 0),
  ('019f0000-0000-7000-8000-000000000003', NOW(), NOW(),
   '019d0000-0000-7000-8000-000000000003', 200, 0, 0, 0);

-- ---------------------------------------------------------------------------
-- RBAC: permission definitions and the admin role grant
-- ---------------------------------------------------------------------------
INSERT IGNORE INTO permissions
  (id, created_at, updated_at, code, name, group_name, sort, status)
VALUES
  ('01a00000-0000-7000-8000-000000000001', NOW(), NOW(), 'dashboard:view', '看板查看', '仪表盘', 1, 1),
  ('01a00000-0000-7000-8000-000000000002', NOW(), NOW(), 'goods:manage', '商品管理', '商品', 2, 1),
  ('01a00000-0000-7000-8000-000000000003', NOW(), NOW(), 'order:manage', '订单管理', '订单', 3, 1),
  ('01a00000-0000-7000-8000-000000000004', NOW(), NOW(), 'system:manage', '系统管理', '系统', 4, 1);

INSERT IGNORE INTO role_permissions
  (id, created_at, updated_at, role, permission_code, status)
VALUES
  ('01a10000-0000-7000-8000-000000000001', NOW(), NOW(), 2, 'dashboard:view', 1),
  ('01a10000-0000-7000-8000-000000000002', NOW(), NOW(), 2, 'goods:manage', 1),
  ('01a10000-0000-7000-8000-000000000003', NOW(), NOW(), 2, 'order:manage', 1),
  ('01a10000-0000-7000-8000-000000000004', NOW(), NOW(), 2, 'system:manage', 1),
  ('01a10000-0000-7000-8000-000000000005', NOW(), NOW(), 1, 'goods:view', 1);

-- ---------------------------------------------------------------------------
-- Admin account
--
-- username: admin
-- password: admin123456
-- hash format: pbkdf2_sha512$<iterations>$<salt-hex>$<key-hex>
-- (matches internal/data/password_hasher.go, 10000 iterations, 64-byte key)
-- ---------------------------------------------------------------------------
INSERT IGNORE INTO users
  (id, created_at, updated_at, mobile, username, password_hash, nick_name,
   birthday, gender, role, status)
VALUES
  ('01a20000-0000-7000-8000-000000000001', NOW(), NOW(),
   '13900000000', 'admin',
   'pbkdf2_sha512$10000$00112233445566778899aabbccddeeff$20353d6c3c5ff73d6216b6c95d8c73b7fec861f4254d367cd439253e39ac2dd50cbd55f06f9e9f204250d1c118712186db008a5d02afe719ce62852c7909c466',
   'Administrator', 0, '', 2, 1);
