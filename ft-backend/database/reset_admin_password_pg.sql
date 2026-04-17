-- ============================================================================
-- 强制重置 admin 为最高权限账号（默认租户），密码明文 123456
-- 用法: sudo -u postgres psql -d opsfleetpilot -f database/reset_admin_password_pg.sql
-- 执行后应显示 UPDATE 1；若 UPDATE 0 请检查 SELECT id, username, tenant_id FROM users;
-- ============================================================================

UPDATE users
SET password = crypt('123456', gen_salt('bf', 10)),
    updated_at = NOW(),
    deleted_at = NULL
WHERE username = 'admin'
  AND tenant_id = '00000000-0000-0000-0000-000000000001'::uuid;

-- 校验（可选）：
-- SELECT username, (password = crypt('123456', password)) AS password_ok
-- FROM users
-- WHERE username = 'admin' AND tenant_id = '00000000-0000-0000-0000-000000000001'::uuid;
