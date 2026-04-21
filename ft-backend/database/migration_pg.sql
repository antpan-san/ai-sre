-- ============================================================================
-- OpsFleetPilot  —  PostgreSQL 完整幂等迁移脚本
--
-- 版本 : 2.0
-- 日期 : 2026-02-13
-- 要求 : PostgreSQL 14+
-- 用法 : psql -d opsfleetpilot -f migration_pg.sql
--
-- 幂等保证:
--   • CREATE TABLE         IF NOT EXISTS
--   • CREATE INDEX         IF NOT EXISTS
--   • CREATE UNIQUE INDEX  IF NOT EXISTS
--   • CREATE TABLE … PARTITION OF  通过 DO 块 + pg_class 检查
--   • CREATE TRIGGER       通过 DROP IF EXISTS + CREATE
--   • 种子数据             通过 NOT EXISTS 子查询
--   • CREATE OR REPLACE    用于函数
--   • 整体包裹在单一事务中
-- ============================================================================

BEGIN;

-- ============================================================
-- 0. 扩展
-- ============================================================
CREATE EXTENSION IF NOT EXISTS "pgcrypto";          -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS "pg_trgm";           -- 三元组索引, 用于 ILIKE 加速

-- ============================================================
-- 1. tenants  —  多租户基础表
-- ============================================================
CREATE TABLE IF NOT EXISTS tenants (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(100) NOT NULL,
    code        VARCHAR(50)  NOT NULL,
    status      VARCHAR(20)  NOT NULL DEFAULT 'active',
    metadata    JSONB        NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_tenants_code   ON tenants (code);
CREATE INDEX IF NOT EXISTS        idx_tenants_status ON tenants (status);

-- ============================================================
-- 2. users
-- ============================================================
CREATE TABLE IF NOT EXISTS users (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID         NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001'
                             REFERENCES tenants(id),
    username    VARCHAR(50)  NOT NULL,
    email       VARCHAR(100) NOT NULL,
    phone       VARCHAR(20),
    password    VARCHAR(100) NOT NULL,
    full_name   VARCHAR(100),
    avatar      VARCHAR(255),
    role        VARCHAR(20)  NOT NULL DEFAULT 'user',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ
);

-- 部分唯一索引: 多租户安全, 忽略已软删除行
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username_tenant
    ON users (tenant_id, username) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email_tenant
    ON users (tenant_id, email)    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_users_tenant_id  ON users (tenant_id);
CREATE INDEX IF NOT EXISTS idx_users_role       ON users (role);
CREATE INDEX IF NOT EXISTS idx_users_created_at ON users (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users (deleted_at) WHERE deleted_at IS NOT NULL;

-- ============================================================
-- 3. machines
-- ============================================================
CREATE TABLE IF NOT EXISTS machines (
    id                UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID         NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001'
                                   REFERENCES tenants(id),
    name              VARCHAR(100) NOT NULL,
    ip                VARCHAR(50)  NOT NULL,
    cpu               INT          NOT NULL DEFAULT 0,
    memory            INT          NOT NULL DEFAULT 0,
    disk              INT          NOT NULL DEFAULT 0,
    status            VARCHAR(20)  NOT NULL DEFAULT 'offline',
    labels            JSONB        NOT NULL DEFAULT '{}',
    metadata          JSONB        NOT NULL DEFAULT '{}',
    -- uniqueness & identity
    client_id         VARCHAR(100),
    host_fingerprint  VARCHAR(255),
    -- cluster topology
    node_role         VARCHAR(20)  NOT NULL DEFAULT 'standalone',
    cluster_id        UUID,
    master_machine_id UUID,
    -- ownership
    owner_user_id     UUID,
    -- heartbeat
    last_heartbeat_at TIMESTAMPTZ,
    -- timestamps
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at        TIMESTAMPTZ,
    -- constraints
    CONSTRAINT chk_machines_node_role CHECK (node_role IN ('master', 'worker', 'standalone')),
    CONSTRAINT fk_machines_master_machine FOREIGN KEY (master_machine_id)
        REFERENCES machines(id) ON DELETE SET NULL,
    CONSTRAINT fk_machines_owner_user FOREIGN KEY (owner_user_id)
        REFERENCES users(id) ON DELETE SET NULL
);

-- basic indexes
CREATE INDEX IF NOT EXISTS idx_machines_tenant_id  ON machines (tenant_id);
CREATE INDEX IF NOT EXISTS idx_machines_status     ON machines (status);
CREATE INDEX IF NOT EXISTS idx_machines_created_at ON machines (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_machines_deleted_at ON machines (deleted_at) WHERE deleted_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_machines_labels     ON machines USING GIN (labels);
CREATE INDEX IF NOT EXISTS idx_machines_name_trgm  ON machines USING GIN (name gin_trgm_ops);

-- partial unique indexes (core uniqueness constraints)
CREATE UNIQUE INDEX IF NOT EXISTS idx_machines_tenant_fingerprint
    ON machines (tenant_id, host_fingerprint)
    WHERE deleted_at IS NULL AND host_fingerprint IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_machines_tenant_client_id
    ON machines (tenant_id, client_id)
    WHERE deleted_at IS NULL AND client_id IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_machines_tenant_cluster_master
    ON machines (tenant_id, cluster_id)
    WHERE node_role = 'master' AND deleted_at IS NULL AND cluster_id IS NOT NULL;

-- auxiliary indexes
CREATE INDEX IF NOT EXISTS idx_machines_client_id         ON machines (client_id);
CREATE INDEX IF NOT EXISTS idx_machines_cluster_id        ON machines (cluster_id);
CREATE INDEX IF NOT EXISTS idx_machines_master_machine_id ON machines (master_machine_id);
CREATE INDEX IF NOT EXISTS idx_machines_owner_user_id     ON machines (owner_user_id);
CREATE INDEX IF NOT EXISTS idx_machines_node_role         ON machines (node_role);
CREATE INDEX IF NOT EXISTS idx_machines_last_heartbeat_at ON machines (last_heartbeat_at DESC)
    WHERE last_heartbeat_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_machines_cluster_topology
    ON machines (cluster_id, node_role, created_at)
    WHERE deleted_at IS NULL AND cluster_id IS NOT NULL;

-- ============================================================
-- 4. files
-- ============================================================
CREATE TABLE IF NOT EXISTS files (
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID         NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001'
                                 REFERENCES tenants(id),
    user_id         UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    filename        VARCHAR(255) NOT NULL,
    original_name   VARCHAR(255) NOT NULL,
    size            BIGINT       NOT NULL DEFAULT 0,
    path            VARCHAR(500) NOT NULL,
    mime_type       VARCHAR(100),
    extension       VARCHAR(20),
    hash            VARCHAR(64),
    status          VARCHAR(20)  NOT NULL DEFAULT 'available',
    visibility      VARCHAR(20)  NOT NULL DEFAULT 'private',
    download_count  INT          NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_files_tenant_id   ON files (tenant_id);
CREATE INDEX IF NOT EXISTS idx_files_user_id     ON files (user_id);
CREATE INDEX IF NOT EXISTS idx_files_status      ON files (status);
CREATE INDEX IF NOT EXISTS idx_files_visibility  ON files (visibility);
CREATE INDEX IF NOT EXISTS idx_files_hash        ON files (hash);
CREATE INDEX IF NOT EXISTS idx_files_created_at  ON files (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_files_deleted_at  ON files (deleted_at) WHERE deleted_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_files_user_status ON files (user_id, status, created_at DESC)
    WHERE deleted_at IS NULL;

-- ============================================================
-- 5. transfers
-- ============================================================
CREATE TABLE IF NOT EXISTS transfers (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID         NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001'
                             REFERENCES tenants(id),
    user_id     UUID         NOT NULL REFERENCES users(id)  ON DELETE CASCADE,
    file_id     UUID         NOT NULL REFERENCES files(id)  ON DELETE CASCADE,
    type        VARCHAR(20)  NOT NULL,
    status      VARCHAR(20)  NOT NULL DEFAULT 'pending',
    progress    INT          NOT NULL DEFAULT 0,
    speed       BIGINT       NOT NULL DEFAULT 0,
    ip_address  VARCHAR(50),
    user_agent  VARCHAR(255),
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_transfers_tenant_id  ON transfers (tenant_id);
CREATE INDEX IF NOT EXISTS idx_transfers_user_id    ON transfers (user_id);
CREATE INDEX IF NOT EXISTS idx_transfers_file_id    ON transfers (file_id);
CREATE INDEX IF NOT EXISTS idx_transfers_type       ON transfers (type);
CREATE INDEX IF NOT EXISTS idx_transfers_status     ON transfers (status);
CREATE INDEX IF NOT EXISTS idx_transfers_created_at ON transfers (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_transfers_user_type  ON transfers (user_id, type, created_at DESC);

-- ============================================================
-- 6. shares
-- ============================================================
CREATE TABLE IF NOT EXISTS shares (
    id           UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID         NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001'
                              REFERENCES tenants(id),
    file_id      UUID         NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    share_key    VARCHAR(64)  NOT NULL,
    expires_at   TIMESTAMPTZ  NOT NULL,
    access_count INT          NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_shares_share_key  ON shares (share_key);
CREATE INDEX IF NOT EXISTS        idx_shares_tenant_id  ON shares (tenant_id);
CREATE INDEX IF NOT EXISTS        idx_shares_file_id    ON shares (file_id);
CREATE INDEX IF NOT EXISTS        idx_shares_expires_at ON shares (expires_at);
CREATE INDEX IF NOT EXISTS        idx_shares_active
    ON shares (file_id, created_at DESC) WHERE expires_at > NOW();

-- ============================================================
-- 7. k8s_clusters
-- ============================================================
CREATE TABLE IF NOT EXISTS k8s_clusters (
    id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID         NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001'
                               REFERENCES tenants(id),
    cluster_name  VARCHAR(100) NOT NULL,
    status        VARCHAR(20)  NOT NULL DEFAULT 'pending',
    version       VARCHAR(20),
    master_node   VARCHAR(255),
    worker_nodes  JSONB        NOT NULL DEFAULT '[]',
    config        JSONB        NOT NULL DEFAULT '{}',
    description   TEXT,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_k8s_clusters_name_tenant
    ON k8s_clusters (tenant_id, cluster_name);
CREATE INDEX IF NOT EXISTS idx_k8s_clusters_tenant_id ON k8s_clusters (tenant_id);
CREATE INDEX IF NOT EXISTS idx_k8s_clusters_status    ON k8s_clusters (status);
CREATE INDEX IF NOT EXISTS idx_k8s_clusters_worker    ON k8s_clusters USING GIN (worker_nodes);

-- ============================================================
-- 8. k8s_versions
-- ============================================================
CREATE TABLE IF NOT EXISTS k8s_versions (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID         NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001'
                             REFERENCES tenants(id),
    version     VARCHAR(20)  NOT NULL,
    is_active   BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_k8s_versions_version_tenant
    ON k8s_versions (tenant_id, version);
CREATE INDEX IF NOT EXISTS idx_k8s_versions_tenant_id ON k8s_versions (tenant_id);
CREATE INDEX IF NOT EXISTS idx_k8s_versions_active    ON k8s_versions (is_active) WHERE is_active = TRUE;

-- ============================================================
-- 9. operation_logs
-- ============================================================
CREATE TABLE IF NOT EXISTS operation_logs (
    id             UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      UUID         NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001'
                                REFERENCES tenants(id),
    username       VARCHAR(50)  NOT NULL,
    operation      VARCHAR(100) NOT NULL,
    resource       VARCHAR(100) NOT NULL,
    resource_id    VARCHAR(36),
    ip             VARCHAR(50)  NOT NULL,
    user_agent     VARCHAR(255),
    status         VARCHAR(20)  NOT NULL,
    error_message  VARCHAR(500),
    details        JSONB        NOT NULL DEFAULT '{}',
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_oplog_tenant_id  ON operation_logs (tenant_id);
CREATE INDEX IF NOT EXISTS idx_oplog_username   ON operation_logs (username);
CREATE INDEX IF NOT EXISTS idx_oplog_operation  ON operation_logs (operation);
CREATE INDEX IF NOT EXISTS idx_oplog_resource   ON operation_logs (resource);
CREATE INDEX IF NOT EXISTS idx_oplog_status     ON operation_logs (status);
CREATE INDEX IF NOT EXISTS idx_oplog_created_at ON operation_logs (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_oplog_audit      ON operation_logs (tenant_id, created_at DESC, status);
CREATE INDEX IF NOT EXISTS idx_oplog_details    ON operation_logs USING GIN (details);

-- ============================================================
-- 10. permissions
-- ============================================================
CREATE TABLE IF NOT EXISTS permissions (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID         NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001'
                             REFERENCES tenants(id),
    name        VARCHAR(100) NOT NULL,
    code        VARCHAR(100) NOT NULL,
    description VARCHAR(255),
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_permissions_code_tenant
    ON permissions (tenant_id, code) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_permissions_tenant_id  ON permissions (tenant_id);
CREATE INDEX IF NOT EXISTS idx_permissions_deleted_at ON permissions (deleted_at) WHERE deleted_at IS NOT NULL;

-- ============================================================
-- 11. role_permissions  (联合主键)
-- ============================================================
CREATE TABLE IF NOT EXISTS role_permissions (
    role_id       VARCHAR(20) NOT NULL,
    permission_id UUID        NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    tenant_id     UUID        NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001'
                              REFERENCES tenants(id),
    PRIMARY KEY (role_id, permission_id)
);

CREATE INDEX IF NOT EXISTS idx_rp_tenant_id     ON role_permissions (tenant_id);
CREATE INDEX IF NOT EXISTS idx_rp_role          ON role_permissions (role_id);
CREATE INDEX IF NOT EXISTS idx_rp_permission_id ON role_permissions (permission_id);

-- ============================================================
-- 12. performance_data
-- ============================================================
CREATE TABLE IF NOT EXISTS performance_data (
    id            UUID             PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID             NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001'
                                   REFERENCES tenants(id),
    machine_id    UUID             NOT NULL REFERENCES machines(id) ON DELETE CASCADE,
    machine_name  VARCHAR(100)     NOT NULL,
    cpu_usage     DOUBLE PRECISION NOT NULL DEFAULT 0,
    memory_usage  DOUBLE PRECISION NOT NULL DEFAULT 0,
    disk_usage    DOUBLE PRECISION NOT NULL DEFAULT 0,
    network_in    DOUBLE PRECISION NOT NULL DEFAULT 0,
    network_out   DOUBLE PRECISION NOT NULL DEFAULT 0,
    metrics       JSONB            NOT NULL DEFAULT '{}',
    "timestamp"   TIMESTAMPTZ      NOT NULL,
    created_at    TIMESTAMPTZ      NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_perf_tenant_id  ON performance_data (tenant_id);
CREATE INDEX IF NOT EXISTS idx_perf_machine_id ON performance_data (machine_id);
CREATE INDEX IF NOT EXISTS idx_perf_timestamp  ON performance_data ("timestamp" DESC);
CREATE INDEX IF NOT EXISTS idx_perf_machine_ts ON performance_data (machine_id, "timestamp" DESC);
CREATE INDEX IF NOT EXISTS idx_perf_metrics    ON performance_data USING GIN (metrics);

-- ============================================================
-- 13. heartbeats  —  按月 RANGE 分区表
--     GORM AutoMigrate 无法创建分区表, 必须由本脚本管理
--     主键必须包含分区键 created_at
-- ============================================================
CREATE TABLE IF NOT EXISTS heartbeats (
    id                UUID         NOT NULL DEFAULT gen_random_uuid(),
    tenant_id         UUID         NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001',
    client_id         VARCHAR(100) NOT NULL,
    client_version    VARCHAR(50),
    process_id        INT,
    status            VARCHAR(20)  NOT NULL DEFAULT 'unknown',
    local_ip          VARCHAR(50),
    business_module   VARCHAR(100),
    task_count        INT          NOT NULL DEFAULT 0,
    task_left         INT          NOT NULL DEFAULT 0,
    last_task_time    TIMESTAMPTZ,
    primary_host      JSONB        NOT NULL DEFAULT '{}',
    secondary_hosts   JSONB        NOT NULL DEFAULT '[]',
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

CREATE INDEX IF NOT EXISTS idx_hb_tenant_id       ON heartbeats (tenant_id);
CREATE INDEX IF NOT EXISTS idx_hb_client_id       ON heartbeats (client_id);
CREATE INDEX IF NOT EXISTS idx_hb_status          ON heartbeats (status);
CREATE INDEX IF NOT EXISTS idx_hb_created_at      ON heartbeats (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_hb_client_created  ON heartbeats (client_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_hb_primary_host    ON heartbeats USING GIN (primary_host);
CREATE INDEX IF NOT EXISTS idx_hb_secondary_hosts ON heartbeats USING GIN (secondary_hosts);

-- ---------- 幂等创建月分区 ----------
DO $$
DECLARE
    _year   INT;
    _month  INT;
    _name   TEXT;
    _start  TEXT;
    _end    TEXT;
BEGIN
    -- 2026 年 12 个月分区
    FOR _year IN 2026..2026 LOOP
        FOR _month IN 1..12 LOOP
            _name  := FORMAT('heartbeats_%s_%s', _year, LPAD(_month::TEXT, 2, '0'));
            _start := FORMAT('%s-%s-01', _year, LPAD(_month::TEXT, 2, '0'));
            IF _month = 12 THEN
                _end := FORMAT('%s-01-01', _year + 1);
            ELSE
                _end := FORMAT('%s-%s-01', _year, LPAD((_month + 1)::TEXT, 2, '0'));
            END IF;

            IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relname = _name) THEN
                EXECUTE FORMAT(
                    'CREATE TABLE %I PARTITION OF heartbeats FOR VALUES FROM (%L) TO (%L)',
                    _name, _start, _end
                );
                RAISE NOTICE 'Created partition: %', _name;
            END IF;
        END LOOP;
    END LOOP;

    -- default 分区 (兜底)
    IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relname = 'heartbeats_default') THEN
        CREATE TABLE heartbeats_default PARTITION OF heartbeats DEFAULT;
        RAISE NOTICE 'Created partition: heartbeats_default';
    END IF;
END;
$$;

-- ============================================================
-- 14. 自动创建下月分区的工具函数 (可配 pg_cron 定时调用)
-- ============================================================
CREATE OR REPLACE FUNCTION create_heartbeat_partition()
RETURNS void
LANGUAGE plpgsql
AS $$
DECLARE
    _date  DATE;
    _name  TEXT;
    _start TEXT;
    _end   TEXT;
BEGIN
    _date  := DATE_TRUNC('month', NOW() + INTERVAL '1 month');
    _name  := 'heartbeats_' || TO_CHAR(_date, 'YYYY_MM');
    _start := TO_CHAR(_date, 'YYYY-MM-DD');
    _end   := TO_CHAR(_date + INTERVAL '1 month', 'YYYY-MM-DD');

    IF NOT EXISTS (SELECT 1 FROM pg_class WHERE relname = _name) THEN
        EXECUTE FORMAT(
            'CREATE TABLE %I PARTITION OF heartbeats FOR VALUES FROM (%L) TO (%L)',
            _name, _start, _end
        );
        RAISE NOTICE 'Created partition: %', _name;
    END IF;
END;
$$;

-- ============================================================
-- 15. updated_at 自动更新触发器
-- ============================================================
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$;

-- 幂等: 先 DROP 再 CREATE 每张表的触发器
DO $$
DECLARE
    _tbl TEXT;
BEGIN
    FOR _tbl IN
        SELECT table_name
        FROM information_schema.columns
        WHERE column_name = 'updated_at'
          AND table_schema = 'public'
          AND table_name NOT IN ('heartbeats')      -- 分区表排除
        GROUP BY table_name
    LOOP
        EXECUTE FORMAT('DROP TRIGGER IF EXISTS trigger_updated_at ON %I', _tbl);
        EXECUTE FORMAT(
            'CREATE TRIGGER trigger_updated_at
             BEFORE UPDATE ON %I
             FOR EACH ROW EXECUTE FUNCTION update_updated_at_column()',
            _tbl
        );
    END LOOP;
END;
$$;

-- ============================================
-- 15.1 Roles
-- ============================================
CREATE TABLE IF NOT EXISTS roles (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID        NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001' REFERENCES tenants(id),
    name        VARCHAR(50)  NOT NULL,
    code        VARCHAR(50)  NOT NULL,
    description VARCHAR(255),
    is_system   BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_roles_code ON roles (code);
CREATE INDEX IF NOT EXISTS idx_roles_tenant_id ON roles (tenant_id);

-- ============================================
-- 15.2 Tasks (核心任务调度表)
-- ============================================
CREATE TABLE IF NOT EXISTS tasks (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID        NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001' REFERENCES tenants(id),
    name            VARCHAR(200) NOT NULL,
    type            VARCHAR(50)  NOT NULL,
    status          VARCHAR(20)  NOT NULL DEFAULT 'pending',
    priority        INT          NOT NULL DEFAULT 0,
    created_by      VARCHAR(50)  NOT NULL,
    description     VARCHAR(500),
    payload         JSONB        NOT NULL DEFAULT '{}',
    target_ids      JSONB        NOT NULL DEFAULT '[]',
    total_count     INT          NOT NULL DEFAULT 0,
    success_count   INT          NOT NULL DEFAULT 0,
    failed_count    INT          NOT NULL DEFAULT 0,
    timeout_sec     INT          NOT NULL DEFAULT 300,
    started_at      TIMESTAMPTZ,
    finished_at     TIMESTAMPTZ,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_tasks_tenant_id  ON tasks (tenant_id);
CREATE INDEX IF NOT EXISTS idx_tasks_type       ON tasks (type);
CREATE INDEX IF NOT EXISTS idx_tasks_status     ON tasks (status);
CREATE INDEX IF NOT EXISTS idx_tasks_created_by ON tasks (created_by);
CREATE INDEX IF NOT EXISTS idx_tasks_created_at ON tasks (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_tasks_composite  ON tasks (tenant_id, status, created_at DESC);

-- ============================================
-- 15.3 Sub-Tasks (子任务 - 分配给具体机器)
-- ============================================
CREATE TABLE IF NOT EXISTS sub_tasks (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID        NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001' REFERENCES tenants(id),
    task_id         UUID        NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    machine_id      UUID        NOT NULL,
    client_id       VARCHAR(100) NOT NULL,
    command         VARCHAR(50)  NOT NULL,
    status          VARCHAR(20)  NOT NULL DEFAULT 'pending',
    payload         JSONB        NOT NULL DEFAULT '{}',
    output          TEXT,
    exit_code       INT,
    error           VARCHAR(500),
    retry_count     INT          NOT NULL DEFAULT 0,
    max_retry       INT          NOT NULL DEFAULT 3,
    started_at      TIMESTAMPTZ,
    finished_at     TIMESTAMPTZ,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sub_tasks_tenant_id  ON sub_tasks (tenant_id);
CREATE INDEX IF NOT EXISTS idx_sub_tasks_task_id    ON sub_tasks (task_id);
CREATE INDEX IF NOT EXISTS idx_sub_tasks_machine_id ON sub_tasks (machine_id);
CREATE INDEX IF NOT EXISTS idx_sub_tasks_client_id  ON sub_tasks (client_id);
CREATE INDEX IF NOT EXISTS idx_sub_tasks_status     ON sub_tasks (status);
CREATE INDEX IF NOT EXISTS idx_sub_tasks_dispatch   ON sub_tasks (client_id, status, created_at ASC);

-- ============================================
-- 15.4 Task Logs (任务执行日志)
-- ============================================
CREATE TABLE IF NOT EXISTS task_logs (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID        NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001' REFERENCES tenants(id),
    task_id     UUID        NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    sub_task_id UUID,
    machine_id  UUID,
    client_id   VARCHAR(100),
    level       VARCHAR(20)  NOT NULL DEFAULT 'info',
    message     TEXT         NOT NULL,
    details     JSONB        NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_task_logs_task_id    ON task_logs (task_id);
CREATE INDEX IF NOT EXISTS idx_task_logs_sub_task_id ON task_logs (sub_task_id);
CREATE INDEX IF NOT EXISTS idx_task_logs_created_at ON task_logs (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_task_logs_level      ON task_logs (level);

-- ============================================
-- 15.5 Monitoring Configs (监控配置)
-- ============================================
CREATE TABLE IF NOT EXISTS monitoring_configs (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID        NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001' REFERENCES tenants(id),
    name        VARCHAR(100) NOT NULL,
    type        VARCHAR(50)  NOT NULL,
    status      VARCHAR(20)  NOT NULL DEFAULT 'inactive',
    config      JSONB        NOT NULL DEFAULT '{}',
    description VARCHAR(500),
    machine_id  VARCHAR(36),
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_monitoring_configs_type   ON monitoring_configs (type);
CREATE INDEX IF NOT EXISTS idx_monitoring_configs_status ON monitoring_configs (status);

-- ============================================
-- 15.6 Alert Rules (告警规则)
-- ============================================
CREATE TABLE IF NOT EXISTS alert_rules (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID        NOT NULL DEFAULT '00000000-0000-0000-0000-000000000001' REFERENCES tenants(id),
    name        VARCHAR(100) NOT NULL,
    type        VARCHAR(50)  NOT NULL,
    condition   VARCHAR(200) NOT NULL,
    threshold   VARCHAR(100) NOT NULL,
    severity    VARCHAR(20)  NOT NULL DEFAULT 'warning',
    status      VARCHAR(20)  NOT NULL DEFAULT 'active',
    config      JSONB        NOT NULL DEFAULT '{}',
    description VARCHAR(500),
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_alert_rules_type     ON alert_rules (type);
CREATE INDEX IF NOT EXISTS idx_alert_rules_severity ON alert_rules (severity);
CREATE INDEX IF NOT EXISTS idx_alert_rules_status   ON alert_rules (status);

-- ============================================================
-- 16. 种子数据  (全部幂等, 已存在则跳过)
-- ============================================================

-- 16.1  默认租户
INSERT INTO tenants (id, name, code)
SELECT '00000000-0000-0000-0000-000000000001', 'Default Tenant', 'default'
WHERE NOT EXISTS (
    SELECT 1 FROM tenants WHERE code = 'default'
);

-- 16.2  管理员用户  (password = admin123)
INSERT INTO users (username, email, password, full_name, role)
SELECT 'admin',
       'admin@example.com',
       crypt('password', gen_salt('bf', 10)),
       'Administrator',
       'admin'
WHERE NOT EXISTS (
    SELECT 1 FROM users WHERE username = 'admin' AND deleted_at IS NULL
);

-- 16.3  测试用户
INSERT INTO users (username, email, password, full_name, role)
SELECT 'testuser',
       'test@example.com',
       crypt('password', gen_salt('bf', 10)),
       'Test User',
       'user'
WHERE NOT EXISTS (
    SELECT 1 FROM users WHERE username = 'testuser' AND deleted_at IS NULL
);

-- 16.4  K8s 版本
INSERT INTO k8s_versions (version, is_active)
SELECT v, TRUE FROM (
    VALUES ('v1.35.4'), ('v1.34.3'), ('v1.32.11'),
           ('v1.32.6'), ('v1.30.0'), ('v1.28.15')
) AS t(v)
WHERE NOT EXISTS (
    SELECT 1 FROM k8s_versions WHERE k8s_versions.version = t.v
);

-- 16.5  默认权限
INSERT INTO permissions (name, code, description)
SELECT name, code, description FROM (
    VALUES
        ('Machine Management',    'machine_manage',    'Manage server machines'),
        ('User Management',       'user_manage',       'Manage system users'),
        ('File Management',       'file_manage',       'Manage file resources'),
        ('Security Audit',        'security_audit',    'View security audit logs'),
        ('Permission Management', 'permission_manage', 'Manage system permissions')
) AS t(name, code, description)
WHERE NOT EXISTS (
    SELECT 1 FROM permissions WHERE permissions.code = t.code AND permissions.deleted_at IS NULL
);

-- 16.6  默认角色
INSERT INTO roles (name, code, description, is_system)
SELECT name, code, description, is_system FROM (
    VALUES
        ('管理员',      'admin',    '系统管理员，拥有所有权限',           TRUE),
        ('普通用户',    'user',     '普通用户，拥有基础查看权限',         TRUE),
        ('运维工程师',  'operator', '运维人员，拥有机器管理和任务执行权限', TRUE),
        ('只读用户',    'viewer',   '只读用户，仅有查看权限',            TRUE)
) AS t(name, code, description, is_system)
WHERE NOT EXISTS (
    SELECT 1 FROM roles WHERE roles.code = t.code
);

-- 16.7  为 admin 角色分配所有权限
INSERT INTO role_permissions (role_id, permission_id, tenant_id)
SELECT 'admin', p.id, '00000000-0000-0000-0000-000000000001'::uuid
FROM permissions p
WHERE p.deleted_at IS NULL
  AND NOT EXISTS (
      SELECT 1 FROM role_permissions rp
      WHERE rp.role_id = 'admin' AND rp.permission_id = p.id
);

-- ============================================================
-- 17. 表注释
-- ============================================================
COMMENT ON TABLE tenants          IS '多租户注册表';
COMMENT ON TABLE users            IS '系统用户 (RBAC)';
COMMENT ON TABLE machines         IS '纳管服务器清单';
COMMENT ON TABLE files            IS '上传文件元数据';
COMMENT ON TABLE transfers        IS '文件传输记录';
COMMENT ON TABLE shares           IS '文件分享链接';
COMMENT ON TABLE k8s_clusters     IS 'Kubernetes 集群注册表';
COMMENT ON TABLE k8s_versions     IS '可用 Kubernetes 版本';
COMMENT ON TABLE operation_logs   IS '安全审计日志';
COMMENT ON TABLE permissions      IS 'RBAC 权限定义';
COMMENT ON TABLE role_permissions IS '角色-权限关联表';
COMMENT ON TABLE performance_data IS '机器性能指标';
COMMENT ON TABLE heartbeats         IS '客户端心跳 (按月分区)';
COMMENT ON TABLE roles              IS '系统角色定义';
COMMENT ON TABLE tasks              IS '任务调度主表';
COMMENT ON TABLE sub_tasks          IS '子任务 (分配到具体机器)';
COMMENT ON TABLE task_logs          IS '任务执行日志';
COMMENT ON TABLE monitoring_configs IS '监控工具配置';
COMMENT ON TABLE alert_rules        IS '告警规则';

COMMENT ON COLUMN machines.labels            IS '任意键值标签 (JSONB)';
COMMENT ON COLUMN machines.metadata          IS '可扩展元数据 (JSONB)';
COMMENT ON COLUMN machines.client_id         IS '客户端 Agent 唯一标识 (由 ft-client 生成)';
COMMENT ON COLUMN machines.host_fingerprint  IS '主机硬件/OS 指纹, 用于防止重复注册';
COMMENT ON COLUMN machines.node_role         IS '集群角色: master / worker / standalone';
COMMENT ON COLUMN machines.cluster_id        IS '所属集群 UUID (逻辑分组)';
COMMENT ON COLUMN machines.master_machine_id IS '自引用: worker 指向其 master machine';
COMMENT ON COLUMN machines.owner_user_id     IS '机器归属用户';
COMMENT ON COLUMN machines.last_heartbeat_at IS '最后一次心跳时间';
COMMENT ON COLUMN k8s_clusters.worker_nodes IS 'Worker 节点数组 (JSONB)';
COMMENT ON COLUMN k8s_clusters.config       IS '集群配置 (JSONB)';
COMMENT ON COLUMN heartbeats.primary_host   IS '主主机信息 (JSONB)';
COMMENT ON COLUMN heartbeats.secondary_hosts IS '副主机列表 (JSONB)';
COMMENT ON COLUMN operation_logs.details    IS '附加日志上下文 (JSONB)';
COMMENT ON COLUMN performance_data.metrics  IS '可扩展指标 (JSONB)';
COMMENT ON COLUMN tasks.payload             IS '任务参数 (JSONB)';
COMMENT ON COLUMN tasks.target_ids          IS '目标机器ID列表 (JSONB)';
COMMENT ON COLUMN sub_tasks.payload         IS '执行参数 (JSONB)';
COMMENT ON COLUMN task_logs.details         IS '日志附加信息 (JSONB)';

COMMIT;

-- ============================================================
-- 完成!  可通过 psql -d opsfleetpilot -f migration_pg.sql 反复执行
-- ============================================================
