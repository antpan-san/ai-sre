-- 创建数据库
CREATE DATABASE IF NOT EXISTS filetransfer CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

USE filetransfer;

-- 用户表
CREATE TABLE IF NOT EXISTS users (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    username VARCHAR(50) NOT NULL UNIQUE,
    email VARCHAR(100) NOT NULL UNIQUE,
    password VARCHAR(100) NOT NULL,
    full_name VARCHAR(100),
    avatar VARCHAR(255),
    role VARCHAR(20) NOT NULL DEFAULT 'user',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    PRIMARY KEY (id),
    INDEX idx_users_deleted_at (deleted_at),
    INDEX idx_users_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 文件表
CREATE TABLE IF NOT EXISTS files (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    user_id BIGINT UNSIGNED NOT NULL,
    filename VARCHAR(255) NOT NULL,
    original_name VARCHAR(255) NOT NULL,
    size BIGINT NOT NULL,
    path VARCHAR(255) NOT NULL,
    mime_type VARCHAR(100),
    extension VARCHAR(20),
    hash VARCHAR(64),
    status VARCHAR(20) NOT NULL DEFAULT 'available',
    visibility VARCHAR(20) NOT NULL DEFAULT 'private',
    download_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    PRIMARY KEY (id),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    INDEX idx_files_user_id (user_id),
    INDEX idx_files_status (status),
    INDEX idx_files_visibility (visibility),
    INDEX idx_files_created_at (created_at),
    INDEX idx_files_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 传输记录表
CREATE TABLE IF NOT EXISTS transfers (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    user_id BIGINT UNSIGNED NOT NULL,
    file_id BIGINT UNSIGNED NOT NULL,
    type VARCHAR(20) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    progress INT NOT NULL DEFAULT 0,
    speed BIGINT NOT NULL DEFAULT 0,
    ip_address VARCHAR(50),
    user_agent VARCHAR(255),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (file_id) REFERENCES files(id) ON DELETE CASCADE,
    INDEX idx_transfers_user_id (user_id),
    INDEX idx_transfers_file_id (file_id),
    INDEX idx_transfers_type (type),
    INDEX idx_transfers_status (status),
    INDEX idx_transfers_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 文件分享表
CREATE TABLE IF NOT EXISTS shares (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    file_id BIGINT UNSIGNED NOT NULL,
    share_key VARCHAR(64) NOT NULL UNIQUE,
    expires_at TIMESTAMP NOT NULL,
    access_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    FOREIGN KEY (file_id) REFERENCES files(id) ON DELETE CASCADE,
    INDEX idx_shares_file_id (file_id),
    INDEX idx_shares_share_key (share_key),
    INDEX idx_shares_expires_at (expires_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 插入测试数据
-- 注意：实际环境中密码应该使用bcrypt或argon2进行哈希处理
INSERT INTO users (username, email, password, full_name, role) VALUES 
('admin', 'admin@example.com', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', 'Administrator', 'admin'),
('testuser', 'test@example.com', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', 'Test User', 'user');

-- 插入测试文件
INSERT INTO files (user_id, filename, original_name, size, path, mime_type, extension, hash, visibility) VALUES 
(2, 'test_doc.pdf', 'sample_document.pdf', 1024000, 'uploads/test_doc.pdf', 'application/pdf', 'pdf', 'e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855', 'private'),
(2, 'test_image.jpg', 'sample_image.jpg', 512000, 'uploads/test_image.jpg', 'image/jpeg', 'jpg', 'e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855', 'public');

-- 机器表
CREATE TABLE IF NOT EXISTS machines (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    name VARCHAR(100) NOT NULL,
    ip VARCHAR(50) NOT NULL,
    cpu INT NOT NULL,
    memory INT NOT NULL,
    disk INT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'offline',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    PRIMARY KEY (id),
    INDEX idx_machines_status (status),
    INDEX idx_machines_created_at (created_at),
    INDEX idx_machines_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 操作日志表
CREATE TABLE IF NOT EXISTS operation_logs (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    username VARCHAR(50) NOT NULL,
    operation VARCHAR(100) NOT NULL,
    resource VARCHAR(100) NOT NULL,
    resource_id BIGINT UNSIGNED NOT NULL,
    ip VARCHAR(50) NOT NULL,
    user_agent VARCHAR(255),
    status VARCHAR(20) NOT NULL,
    error_message VARCHAR(255),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    INDEX idx_operation_logs_created_at (created_at),
    INDEX idx_operation_logs_status (status),
    INDEX idx_operation_logs_username (username)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 权限表
CREATE TABLE IF NOT EXISTS permissions (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    name VARCHAR(100) NOT NULL,
    code VARCHAR(100) NOT NULL UNIQUE,
    description VARCHAR(255),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    PRIMARY KEY (id),
    INDEX idx_permissions_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 角色权限表
CREATE TABLE IF NOT EXISTS role_permissions (
    role_id VARCHAR(20) NOT NULL,
    permission_id BIGINT UNSIGNED NOT NULL,
    PRIMARY KEY (role_id, permission_id),
    FOREIGN KEY (permission_id) REFERENCES permissions(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 性能数据表
CREATE TABLE IF NOT EXISTS performance_data (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    machine_id BIGINT UNSIGNED NOT NULL,
    machine_name VARCHAR(100) NOT NULL,
    cpu_usage FLOAT NOT NULL,
    memory_usage FLOAT NOT NULL,
    disk_usage FLOAT NOT NULL,
    network_in FLOAT,
    network_out FLOAT,
    timestamp TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    INDEX idx_performance_data_machine_id (machine_id),
    INDEX idx_performance_data_timestamp (timestamp)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 插入测试机器数据
INSERT INTO machines (name, ip, cpu, memory, disk, status) VALUES 
('测试机器1', '192.168.1.100', 4, 8192, 500, 'online'),
('测试机器2', '192.168.1.101', 8, 16384, 1000, 'offline');

-- 插入测试权限数据
INSERT INTO permissions (name, code, description) VALUES 
('机器管理', 'machine_manage', '管理服务器机器'),
('用户管理', 'user_manage', '管理系统用户'),
('文件管理', 'file_manage', '管理文件资源'),
('安全审计', 'security_audit', '查看安全审计日志'),
('权限管理', 'permission_manage', '管理系统权限');

-- 为管理员角色分配所有权限
INSERT INTO role_permissions (role_id, permission_id) VALUES 
('admin', 1),
('admin', 2),
('admin', 3),
('admin', 4),
('admin', 5);
