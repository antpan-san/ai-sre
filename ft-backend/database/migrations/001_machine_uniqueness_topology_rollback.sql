-- ============================================================================
-- ROLLBACK Migration 001: Machine Uniqueness & Master-Slave Topology
--
-- 日期 : 2026-02-13
-- 用法 : psql -d opsfleetpilot -f database/migrations/001_machine_uniqueness_topology_rollback.sql
--
-- ⚠️  警告: 此脚本会永久删除 machines 表的新增列及其数据!
--     执行前请确认已备份相关数据。
-- ============================================================================

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

-- 5. 删除新增列
ALTER TABLE machines DROP COLUMN IF EXISTS last_heartbeat_at;
ALTER TABLE machines DROP COLUMN IF EXISTS owner_user_id;
ALTER TABLE machines DROP COLUMN IF EXISTS master_machine_id;
ALTER TABLE machines DROP COLUMN IF EXISTS cluster_id;
ALTER TABLE machines DROP COLUMN IF EXISTS node_role;
ALTER TABLE machines DROP COLUMN IF EXISTS host_fingerprint;
ALTER TABLE machines DROP COLUMN IF EXISTS client_id;

COMMIT;
