-- 初始化Kubernetes版本数据 (PostgreSQL 语法)
INSERT INTO k8s_versions (version, is_active, created_at, updated_at) VALUES
('v1.35.0', TRUE, NOW(), NOW()),
('v1.32.11', TRUE, NOW(), NOW()),
('v1.34.3', TRUE, NOW(), NOW()),
('v1.32.6', TRUE, NOW(), NOW()),
('v1.30.0', TRUE, NOW(), NOW()),
('v1.28.15', TRUE, NOW(), NOW())
ON CONFLICT (tenant_id, version) DO UPDATE SET
is_active = EXCLUDED.is_active,
updated_at = NOW();
