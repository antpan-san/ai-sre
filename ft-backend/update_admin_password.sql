-- 更新admin用户的密码为123456
-- 使用兼容bcrypt算法的哈希值
UPDATE users 
SET password = '$2a$10$b9TQ5sZ84rbNJWQRYg7uKOxHK8m2jnPhI4P6ZS/8Orn.QEKxz/FR2'
WHERE username = 'admin';

-- 或者如果你使用的MySQL版本支持BCRYPT函数
-- UPDATE users 
-- SET password = BCRYPT('123456') 
-- WHERE username = 'admin';

-- 验证更新结果
SELECT username, password, role FROM users WHERE username = 'admin';