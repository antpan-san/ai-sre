-- ============================================================================
-- 强制重置 admin 密码为明文 password（用于登录 401 时修复）
-- 用法: sudo -u postgres psql -d opsfleetpilot -f database/reset_admin_password_pg.sql
-- 执行后应显示 UPDATE 1，然后用 用户名 admin / 密码 password 登录
-- ============================================================================

-- 1) 强制重置密码（含已软删除的 admin）
-- 使用 PostgreSQL 现场生成 bcrypt，避免手工拷贝哈希出错
UPDATE users
SET password = crypt('password', gen_salt('bf', 10)),
    updated_at = NOW(),
    deleted_at = NULL
WHERE username = 'admin'
  AND tenant_id = '00000000-0000-0000-0000-000000000001'::uuid;

-- 执行后应显示 UPDATE 1。若为 UPDATE 0，请检查：SELECT id, username FROM users;
-- 执行后可验证：
-- SELECT username, (password = crypt('password', password)) AS ok
-- FROM users
-- WHERE username='admin' AND tenant_id='00000000-0000-0000-0000-000000000001'::uuid;
