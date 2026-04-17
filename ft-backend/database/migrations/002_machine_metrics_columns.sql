-- Migration: 002_machine_metrics_columns
-- Add real-time metric columns to machines table.
-- These fields are updated on each heartbeat so the REST API returns fresh data
-- even when Redis is unavailable.

DO $$
BEGIN
    -- os_version
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'machines' AND column_name = 'os_version'
    ) THEN
        ALTER TABLE machines ADD COLUMN os_version VARCHAR(100) DEFAULT '';
        RAISE NOTICE 'Added column: machines.os_version';
    END IF;

    -- kernel_version
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'machines' AND column_name = 'kernel_version'
    ) THEN
        ALTER TABLE machines ADD COLUMN kernel_version VARCHAR(100) DEFAULT '';
        RAISE NOTICE 'Added column: machines.kernel_version';
    END IF;

    -- cpu_cores
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'machines' AND column_name = 'cpu_cores'
    ) THEN
        ALTER TABLE machines ADD COLUMN cpu_cores INTEGER NOT NULL DEFAULT 0;
        RAISE NOTICE 'Added column: machines.cpu_cores';
    END IF;

    -- cpu_usage
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'machines' AND column_name = 'cpu_usage'
    ) THEN
        ALTER TABLE machines ADD COLUMN cpu_usage DOUBLE PRECISION NOT NULL DEFAULT 0;
        RAISE NOTICE 'Added column: machines.cpu_usage';
    END IF;

    -- memory_total
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'machines' AND column_name = 'memory_total'
    ) THEN
        ALTER TABLE machines ADD COLUMN memory_total BIGINT NOT NULL DEFAULT 0;
        RAISE NOTICE 'Added column: machines.memory_total';
    END IF;

    -- memory_used
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'machines' AND column_name = 'memory_used'
    ) THEN
        ALTER TABLE machines ADD COLUMN memory_used BIGINT NOT NULL DEFAULT 0;
        RAISE NOTICE 'Added column: machines.memory_used';
    END IF;

    -- memory_usage
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'machines' AND column_name = 'memory_usage'
    ) THEN
        ALTER TABLE machines ADD COLUMN memory_usage DOUBLE PRECISION NOT NULL DEFAULT 0;
        RAISE NOTICE 'Added column: machines.memory_usage';
    END IF;

    -- disk_total
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'machines' AND column_name = 'disk_total'
    ) THEN
        ALTER TABLE machines ADD COLUMN disk_total BIGINT NOT NULL DEFAULT 0;
        RAISE NOTICE 'Added column: machines.disk_total';
    END IF;

    -- disk_used
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'machines' AND column_name = 'disk_used'
    ) THEN
        ALTER TABLE machines ADD COLUMN disk_used BIGINT NOT NULL DEFAULT 0;
        RAISE NOTICE 'Added column: machines.disk_used';
    END IF;

    -- disk_usage
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'machines' AND column_name = 'disk_usage'
    ) THEN
        ALTER TABLE machines ADD COLUMN disk_usage DOUBLE PRECISION NOT NULL DEFAULT 0;
        RAISE NOTICE 'Added column: machines.disk_usage';
    END IF;
END;
$$;

-- Rollback:
-- ALTER TABLE machines DROP COLUMN IF EXISTS os_version;
-- ALTER TABLE machines DROP COLUMN IF EXISTS kernel_version;
-- ALTER TABLE machines DROP COLUMN IF EXISTS cpu_cores;
-- ALTER TABLE machines DROP COLUMN IF EXISTS cpu_usage;
-- ALTER TABLE machines DROP COLUMN IF EXISTS memory_total;
-- ALTER TABLE machines DROP COLUMN IF EXISTS memory_used;
-- ALTER TABLE machines DROP COLUMN IF EXISTS memory_usage;
-- ALTER TABLE machines DROP COLUMN IF EXISTS disk_total;
-- ALTER TABLE machines DROP COLUMN IF EXISTS disk_used;
-- ALTER TABLE machines DROP COLUMN IF EXISTS disk_usage;
