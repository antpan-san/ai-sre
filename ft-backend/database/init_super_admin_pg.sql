-- ============================================================================
-- OpsFleetPilot (PostgreSQL) — 初始化/重置超级管理员
--
-- 用法: psql -d <数据库名> -f init_super_admin_pg.sql
-- 幂等: 可重复执行；若 admin 已存在则跳过插入，仅可选地执行“重置密码”部分。
-- ============================================================================

-- ----------------------------------------------------------------------------
-- 一、当前超级用户信息（与 migration_pg.sql 中种子数据一致）
-- ----------------------------------------------------------------------------
-- 用户名: admin
-- 邮箱:   admin@example.com
-- 密码:   password  （下述 bcrypt 哈希对应的明文）
-- 角色:   admin
-- 说明:   哈希 $2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy
--         为 bcrypt 常见测试哈希，明文一般为 "password"。
--         若需改为 admin123 等，请执行下方「二、重置密码」并替换为对应哈希。
-- ----------------------------------------------------------------------------

-- 确保默认租户存在（若已执行过 migration_pg.sql 可忽略）
INSERT INTO tenants (id, name, code)
SELECT '00000000-0000-0000-0000-000000000001', 'Default Tenant', 'default'
WHERE NOT EXISTS (SELECT 1 FROM tenants WHERE code = 'default');

-- 若不存在则插入超级用户（幂等）
INSERT INTO users (tenant_id, username, email, password, full_name, role)
SELECT '00000000-0000-0000-0000-000000000001'::uuid,
       'admin',
       'admin@example.com',
       crypt('password', gen_salt('bf', 10)),
       'Administrator',
       'admin'
WHERE NOT EXISTS (
    SELECT 1 FROM users WHERE username = 'admin' AND deleted_at IS NULL
);

-- ----------------------------------------------------------------------------
-- 二、重置 admin 密码（可选）
--
-- 重要：在 shell 里用 psql -c 时不要对哈希里的 $ 转义，否则会写入前导反斜杠，
-- 导致登录一直 401。若已误用 '\$2a$10$...' 更新过，请先检查：
--   SELECT left(password, 5) FROM users WHERE username = 'admin';
-- 若显示 \$2a$ 说明已写坏，需用下面方式之一重新更新。
--
-- 推荐：把哈希写在 .sql 文件里用 -f 执行，例如本文件末尾的“重置为 admin123”示例。
-- 生成哈希：在 ft-backend 目录用 Go 调用 utils.HashPassword("你的密码") 打印结果。
-- ----------------------------------------------------------------------------
-- UPDATE users
-- SET password = '<YOUR_BCRYPT_HASH>',
--     updated_at = NOW()
-- WHERE username = 'admin' AND deleted_at IS NULL;

-- 若曾在 shell 里用 '\$2a$10$...' 更新导致登录 401，可先恢复默认密码（明文 password）：
-- UPDATE users SET password = crypt('password', gen_salt('bf', 10)), updated_at = NOW() WHERE username = 'admin' AND deleted_at IS NULL;
-- 然后使用 用户名 admin / 密码 password 登录。若需改为 admin123，用 Go 生成新哈希后写进本文件再执行上面 UPDATE。

-- ----------------------------------------------------------------------------
-- 三、验证
-- ----------------------------------------------------------------------------
SELECT id, username, email, full_name, role, created_at
FROM users
WHERE username = 'admin' AND deleted_at IS NULL;
