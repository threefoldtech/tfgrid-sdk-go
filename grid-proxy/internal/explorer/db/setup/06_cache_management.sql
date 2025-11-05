-- ============================================================================
-- CACHE MANAGEMENT FUNCTIONS
-- ============================================================================
-- Functions for manual cache refresh and validation.
-- Use these to fix inconsistencies or refresh after bulk operations.

/*
 * refresh_resources_cache_node
 * 
 * Refreshes cache for a single node by recalculating from the view.
 * Useful when cache becomes inconsistent for a specific node.
 * 
 * @param p_node_id - Node ID to refresh
 */
CREATE OR REPLACE FUNCTION refresh_resources_cache_node(p_node_id INTEGER) RETURNS VOID AS
$$
BEGIN
    DELETE FROM resources_cache WHERE node_id = p_node_id;
    
    INSERT INTO resources_cache
    SELECT *
    FROM resources_cache_view
    WHERE resources_cache_view.node_id = p_node_id;
END;
$$ LANGUAGE plpgsql;

/*
 * refresh_resources_cache
 * 
 * Refreshes entire resources_cache table from the view.
 * Use after bulk operations or when cache inconsistencies are detected.
 * WARNING: This truncates the table and recalculates all rows.
 */
CREATE OR REPLACE FUNCTION refresh_resources_cache() RETURNS VOID AS
$$
BEGIN
    TRUNCATE resources_cache;
    INSERT INTO resources_cache
    SELECT *
    FROM resources_cache_view;
END;
$$ LANGUAGE plpgsql;

/*
 * refresh_public_ips_cache_farm
 * 
 * Refreshes cache for a single farm by recalculating IP aggregations.
 * 
 * @param p_farm_id - Farm ID to refresh
 */
CREATE OR REPLACE FUNCTION refresh_public_ips_cache_farm(p_farm_id INTEGER) RETURNS VOID AS
$$
BEGIN
    DELETE FROM public_ips_cache WHERE farm_id = p_farm_id;
    
    INSERT INTO public_ips_cache
    SELECT
        farm.farm_id,
        COALESCE(public_ip_agg.free_ips, 0),
        COALESCE(public_ip_agg.total_ips, 0),
        COALESCE(public_ip_agg.ips, '[]')
    FROM farm
        LEFT JOIN(
            SELECT
                p1.farm_id,
                COUNT(p1.id) total_ips,
                COUNT(CASE WHEN p1.contract_id = 0 THEN 1 END) free_ips,
                jsonb_agg(jsonb_build_object('id', p1.id, 'ip', p1.ip, 'contract_id', p1.contract_id, 'gateway', p1.gateway)) as ips
            FROM public_ip AS p1
            GROUP BY
                p1.farm_id
        ) public_ip_agg on public_ip_agg.farm_id = farm.id
    WHERE farm.farm_id = p_farm_id;
END;
$$ LANGUAGE plpgsql;

/*
 * refresh_public_ips_cache
 * 
 * Refreshes entire public_ips_cache table from source.
 * Use after bulk IP operations or when cache inconsistencies are detected.
 * WARNING: This truncates the table and recalculates all rows.
 */
CREATE OR REPLACE FUNCTION refresh_public_ips_cache() RETURNS VOID AS
$$
BEGIN
    TRUNCATE public_ips_cache;
    
    INSERT INTO public_ips_cache
    SELECT
        farm.farm_id,
        COALESCE(public_ip_agg.free_ips, 0),
        COALESCE(public_ip_agg.total_ips, 0),
        COALESCE(public_ip_agg.ips, '[]')
    FROM farm
        LEFT JOIN(
            SELECT
                p1.farm_id,
                COUNT(p1.id) total_ips,
                COUNT(CASE WHEN p1.contract_id = 0 THEN 1 END) free_ips,
                jsonb_agg(jsonb_build_object('id', p1.id, 'ip', p1.ip, 'contract_id', p1.contract_id, 'gateway', p1.gateway)) as ips
            FROM public_ip AS p1
            GROUP BY
                p1.farm_id
        ) public_ip_agg on public_ip_agg.farm_id = farm.id;
END;
$$ LANGUAGE plpgsql;

/*
 * validate_resources_cache
 * 
 * Validates cache consistency by comparing cache table with view.
 * Checks all resource fields for mismatches.
 * 
 * @returns INTEGER - Count of mismatched records (0 = cache is consistent)
 */
CREATE OR REPLACE FUNCTION validate_resources_cache() RETURNS INTEGER AS
$$
DECLARE
    mismatch_count INTEGER;
BEGIN
    SELECT COUNT(*)
    INTO mismatch_count
    FROM resources_cache rc
    FULL OUTER JOIN resources_cache_view rcv ON rc.node_id = rcv.node_id
    WHERE rc.node_id IS NULL OR rcv.node_id IS NULL
       OR rc.total_hru != rcv.total_hru
       OR rc.total_mru != rcv.total_mru
       OR rc.total_sru != rcv.total_sru
       OR rc.total_cru != rcv.total_cru
       OR rc.free_hru != rcv.free_hru
       OR rc.free_mru != rcv.free_mru
       OR rc.free_sru != rcv.free_sru
       OR rc.used_hru != rcv.used_hru
       OR rc.used_mru != rcv.used_mru
       OR rc.used_sru != rcv.used_sru
       OR rc.used_cru != rcv.used_cru
       OR rc.node_contracts_count != rcv.node_contracts_count;
    
    RETURN mismatch_count;
END;
$$ LANGUAGE plpgsql;

/*
 * refresh_all_caches
 * 
 * Convenience function to refresh all cache tables.
 * Calls refresh_resources_cache() and refresh_public_ips_cache().
 * WARNING: This truncates and recalculates all cache tables.
 */
CREATE OR REPLACE FUNCTION refresh_all_caches() RETURNS VOID AS
$$
BEGIN
    PERFORM refresh_resources_cache();
    PERFORM refresh_public_ips_cache();
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- AUTOMATED CACHE REFRESH SCHEDULING
-- ============================================================================
-- Schedule automatic cache refresh using pg_cron extension.
-- The cache will be refreshed daily at midnight (00:00:00).

-- Install pg_cron extension if not already installed
CREATE EXTENSION IF NOT EXISTS pg_cron;

-- Schedule nightly cache refresh at midnight (00:00:00)
-- Cron expression: '0 0 * * *' = every day at 00:00:00 (midnight)
-- This will call refresh_all_caches() every night
DO $$
DECLARE
    job_id INTEGER;
BEGIN
    -- Check if job already exists and unschedule it (idempotent)
    SELECT jobid INTO job_id
    FROM cron.job
    WHERE jobname = 'refresh-cache-nightly';
    
    IF job_id IS NOT NULL THEN
        PERFORM cron.unschedule(job_id);
        RAISE NOTICE 'Removed existing cache refresh schedule';
    END IF;
    
    -- Schedule the nightly refresh
    PERFORM cron.schedule(
        'refresh-cache-nightly',           -- Job name
        '0 0 * * *',                       -- Cron: Every day at midnight (00:00:00)
        $$SELECT refresh_all_caches()$$    -- SQL to execute
    );
    
    RAISE NOTICE 'Scheduled cache refresh: Daily at midnight (00:00:00)';
EXCEPTION
    WHEN OTHERS THEN
        RAISE WARNING 'Failed to schedule cache refresh: %. pg_cron extension may not be available or may require superuser privileges.', SQLERRM;
END $$;

