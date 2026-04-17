-- ============================================================================
-- Migration 001: Machine Uniqueness & Master-Slave Topology
--
-- 日期 : 2026-02-13
-- 作者 : OpsFleetPilot
-- 要求 : PostgreSQL 14+
-- 用法 : psql -d opsfleetpilot -f database/migrations/001_machine_uniqueness_topology.sql
--
-- 变更内容:
--   1. machines 表新增字段:
--      client_id, host_fingerprint, node_role, cluster_id,
--      master_machine_id, owner_user_id, last_heartbeat_at
--   2. 添加三个部分唯一索引 (partial unique index):
--      - (tenant_id, host_fingerprint) WHERE deleted_at IS NULL
--      - (tenant_id, client_id)        WHERE deleted_at IS NULL
--      - (tenant_id, cluster_id)       WHERE node_role='master' AND deleted_at IS NULL
--   3. 添加辅助 B-tree 索引
--   4. 添加自引用外键 (master_machine_id -> machines.id)
--
-- 幂等保证: 全部使用 IF NOT EXISTS / DO $$ 检查
-- 回滚脚本: 见文件末尾 ROLLBACK 区域
-- ============================================================================

BEGIN;

-- ============================================================
-- 1. 新增字段 (幂等: 先检查列是否存在)
-- ============================================================
DO $$
BEGIN
    -- client_id: 客户端 Agent 标识
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'machines' AND column_name = 'client_id'
    ) THEN
        ALTER TABLE machines ADD COLUMN client_id VARCHAR(100);
        RAISE NOTICE 'Added column: machines.client_id';
    END IF;

    -- host_fingerprint: 主机硬件/OS 指纹 (用于唯一性去重)
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'machines' AND column_name = 'host_fingerprint'
    ) THEN
        ALTER TABLE machines ADD COLUMN host_fingerprint VARCHAR(255);
        RAISE NOTICE 'Added column: machines.host_fingerprint';
    END IF;

    -- node_role: 节点角色 (master / worker / standalone)
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'machines' AND column_name = 'node_role'
    ) THEN
        ALTER TABLE machines ADD COLUMN node_role VARCHAR(20) NOT NULL DEFAULT 'standalone';
        RAISE NOTICE 'Added column: machines.node_role';
    END IF;

    -- cluster_id: 所属集群 (逻辑分组, 可关联 k8s_clusters)
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'machines' AND column_name = 'cluster_id'
    ) THEN
        ALTER TABLE machines ADD COLUMN cluster_id UUID;
        RAISE NOTICE 'Added column: machines.cluster_id';
    END IF;

    -- master_machine_id: 自引用 FK, worker 指向其 master
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'machines' AND column_name = 'master_machine_id'
    ) THEN
        ALTER TABLE machines ADD COLUMN master_machine_id UUID;
        RAISE NOTICE 'Added column: machines.master_machine_id';
    END IF;

    -- owner_user_id: 机器归属用户
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'machines' AND column_name = 'owner_user_id'
    ) THEN
        ALTER TABLE machines ADD COLUMN owner_user_id UUID;
        RAISE NOTICE 'Added column: machines.owner_user_id';
    END IF;

    -- last_heartbeat_at: 最后心跳时间 (从 metadata 提升为一等字段)
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'machines' AND column_name = 'last_heartbeat_at'
    ) THEN
        ALTER TABLE machines ADD COLUMN last_heartbeat_at TIMESTAMPTZ;
        RAISE NOTICE 'Added column: machines.last_heartbeat_at';
    END IF;
END;
$$;

-- ============================================================
-- 2. CHECK 约束 (node_role 取值范围)
-- ============================================================
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'chk_machines_node_role' AND conrelid = 'machines'::regclass
    ) THEN
        ALTER TABLE machines ADD CONSTRAINT chk_machines_node_role
            CHECK (node_role IN ('master', 'worker', 'standalone'));
        RAISE NOTICE 'Added CHECK constraint: chk_machines_node_role';
    END IF;
END;
$$;

-- ============================================================
-- 3. 外键约束
-- ============================================================

-- 3.1 master_machine_id -> machines(id) 自引用
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'fk_machines_master_machine' AND conrelid = 'machines'::regclass
    ) THEN
        ALTER TABLE machines ADD CONSTRAINT fk_machines_master_machine
            FOREIGN KEY (master_machine_id) REFERENCES machines(id)
            ON DELETE SET NULL;
        RAISE NOTICE 'Added FK: fk_machines_master_machine';
    END IF;
END;
$$;

-- 3.2 owner_user_id -> users(id)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'fk_machines_owner_user' AND conrelid = 'machines'::regclass
    ) THEN
        ALTER TABLE machines ADD CONSTRAINT fk_machines_owner_user
            FOREIGN KEY (owner_user_id) REFERENCES users(id)
            ON DELETE SET NULL;
        RAISE NOTICE 'Added FK: fk_machines_owner_user';
    END IF;
END;
$$;

-- ============================================================
-- 4. 部分唯一索引 (Partial Unique Indexes)
--    核心: 仅对未软删除的行生效
-- ============================================================

-- 4.1 同一租户下, host_fingerprint 唯一 (排除已删除)
CREATE UNIQUE INDEX IF NOT EXISTS idx_machines_tenant_fingerprint
    ON machines (tenant_id, host_fingerprint)
    WHERE deleted_at IS NULL AND host_fingerprint IS NOT NULL;

-- 4.2 同一租户下, client_id 唯一 (排除已删除)
CREATE UNIQUE INDEX IF NOT EXISTS idx_machines_tenant_client_id
    ON machines (tenant_id, client_id)
    WHERE deleted_at IS NULL AND client_id IS NOT NULL;

-- 4.3 同一租户+集群下, 只能有一个 master (排除已删除)
CREATE UNIQUE INDEX IF NOT EXISTS idx_machines_tenant_cluster_master
    ON machines (tenant_id, cluster_id)
    WHERE node_role = 'master' AND deleted_at IS NULL AND cluster_id IS NOT NULL;

-- ============================================================
-- 5. 辅助索引 (加速常见查询)
-- ============================================================
CREATE INDEX IF NOT EXISTS idx_machines_client_id         ON machines (client_id);
CREATE INDEX IF NOT EXISTS idx_machines_cluster_id        ON machines (cluster_id);
CREATE INDEX IF NOT EXISTS idx_machines_master_machine_id ON machines (master_machine_id);
CREATE INDEX IF NOT EXISTS idx_machines_owner_user_id     ON machines (owner_user_id);
CREATE INDEX IF NOT EXISTS idx_machines_node_role         ON machines (node_role);
CREATE INDEX IF NOT EXISTS idx_machines_last_heartbeat_at ON machines (last_heartbeat_at DESC)
    WHERE last_heartbeat_at IS NOT NULL;

-- 复合索引: 集群拓扑查询 (给定 cluster_id 查所有节点, 按角色排序)
CREATE INDEX IF NOT EXISTS idx_machines_cluster_topology
    ON machines (cluster_id, node_role, created_at)
    WHERE deleted_at IS NULL AND cluster_id IS NOT NULL;

-- ============================================================
-- 6. 列注释
-- ============================================================
COMMENT ON COLUMN machines.client_id         IS '客户端 Agent 唯一标识 (由 ft-client 生成)';
COMMENT ON COLUMN machines.host_fingerprint  IS '主机硬件/OS 指纹, 用于防止重复注册';
COMMENT ON COLUMN machines.node_role         IS '集群角色: master / worker / standalone';
COMMENT ON COLUMN machines.cluster_id        IS '所属集群 UUID (逻辑分组)';
COMMENT ON COLUMN machines.master_machine_id IS '自引用: worker 指向其 master machine';
COMMENT ON COLUMN machines.owner_user_id     IS '机器归属用户';
COMMENT ON COLUMN machines.last_heartbeat_at IS '最后一次心跳时间';

COMMIT;

-- ============================================================================
-- ██████  ██████  ██      ██      ██████   █████   ██████ ██   ██
-- ██   ██ ██    ██ ██      ██      ██   ██ ██   ██ ██      ██  ██
-- ██████  ██    ██ ██      ██      ██████  ███████ ██      █████
-- ██   ██ ██    ██ ██      ██      ██   ██ ██   ██ ██      ██  ██
-- ██   ██  ██████  ███████ ███████ ██████  ██   ██  ██████ ██   ██
--
-- 回滚脚本 — 请在单独事务中执行
-- 用法: psql -d opsfleetpilot -f database/migrations/001_machine_uniqueness_topology_rollback.sql
-- 或取消下方注释直接执行
-- ============================================================================

/*
BEGIN;

-- 1. 删除辅助索引
DROP INDEX IF EXISTS idx_machines_cluster_topology;
DROP INDEX IF EXISTS idx_machines_last_heartbeat_at;
DROP INDEX IF EXISTS idx_machines_owner_user_id;
DROP INDEX IF EXISTS idx_machines_master_machine_id;
DROP INDEX IF EXISTS idx_machines_cluster_id;
DROP INDEX IF EXISTS idx_machines_client_id;
DROP INDEX IF EXISTS idx_machines_node_role;

-- 2. 删除唯一索引
DROP INDEX IF EXISTS idx_machines_tenant_cluster_master;
DROP INDEX IF EXISTS idx_machines_tenant_client_id;
DROP INDEX IF EXISTS idx_machines_tenant_fingerprint;

-- 3. 删除外键约束
ALTER TABLE machines DROP CONSTRAINT IF EXISTS fk_machines_owner_user;
ALTER TABLE machines DROP CONSTRAINT IF EXISTS fk_machines_master_machine;

-- 4. 删除 CHECK 约束
ALTER TABLE machines DROP CONSTRAINT IF EXISTS chk_machines_node_role;

-- 5. 删除新增列 (顺序无关, 但保持对称)
ALTER TABLE machines DROP COLUMN IF EXISTS last_heartbeat_at;
ALTER TABLE machines DROP COLUMN IF EXISTS owner_user_id;
ALTER TABLE machines DROP COLUMN IF EXISTS master_machine_id;
ALTER TABLE machines DROP COLUMN IF EXISTS cluster_id;
ALTER TABLE machines DROP COLUMN IF EXISTS node_role;
ALTER TABLE machines DROP COLUMN IF EXISTS host_fingerprint;
ALTER TABLE machines DROP COLUMN IF EXISTS client_id;

COMMIT;
*/
