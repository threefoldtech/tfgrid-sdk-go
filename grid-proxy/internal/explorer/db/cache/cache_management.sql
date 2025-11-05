-- Cache management functions for manual refresh and validation

CREATE OR REPLACE FUNCTION refresh_resources_cache_node(p_node_id INTEGER) RETURNS VOID AS
$$
BEGIN
    DELETE FROM nodex WHERE node_id = p_node_id;
    
    INSERT INTO nodex
    SELECT *
    FROM nodex_view
    WHERE nodex_view.node_id = p_node_id;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION refresh_resources_cache() RETURNS VOID AS
$$
BEGIN
    TRUNCATE nodex;
    INSERT INTO nodex
    SELECT *
    FROM nodex_view;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION refresh_public_ips_cache_farm(p_farm_id INTEGER) RETURNS VOID AS
$$
BEGIN
    DELETE FROM farmx WHERE farm_id = p_farm_id;
    
    INSERT INTO farmx
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

CREATE OR REPLACE FUNCTION refresh_public_ips_cache() RETURNS VOID AS
$$
BEGIN
    TRUNCATE farmx;
    
    INSERT INTO farmx
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

CREATE OR REPLACE FUNCTION validate_resources_cache() RETURNS INTEGER AS
$$
DECLARE
    mismatch_count INTEGER;
BEGIN
    SELECT COUNT(*)
    INTO mismatch_count
    FROM nodex rc
    FULL OUTER JOIN nodex_view rcv ON rc.node_id = rcv.node_id
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

CREATE OR REPLACE FUNCTION refresh_all_caches() RETURNS VOID AS
$$
BEGIN
    PERFORM refresh_resources_cache();
    PERFORM refresh_public_ips_cache();
END;
$$ LANGUAGE plpgsql;

-- Automated cache refresh scheduling with pg_cron
CREATE EXTENSION IF NOT EXISTS pg_cron;
DO $$
DECLARE
    job_id INTEGER;
BEGIN
    SELECT jobid INTO job_id
    FROM cron.job
    WHERE jobname = 'refresh-cache-nightly';
    
    IF job_id IS NOT NULL THEN
        PERFORM cron.unschedule(job_id);
        RAISE NOTICE 'Removed existing cache refresh schedule';
    END IF;
    
    PERFORM cron.schedule(
        'refresh-cache-nightly',
        '0 0 * * *',
        $$SELECT refresh_all_caches()$$
    );
    
    RAISE NOTICE 'Scheduled cache refresh: Daily at midnight (00:00:00)';
EXCEPTION
    WHEN OTHERS THEN
        RAISE WARNING 'Failed to schedule cache refresh: %. pg_cron extension may not be available or may require superuser privileges.', SQLERRM;
END $$;

