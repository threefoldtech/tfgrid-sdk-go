-- ============================================================================
-- CACHE TABLES
-- ============================================================================
-- Materialized cache tables that store pre-computed data for fast queries.
-- These tables are automatically maintained by triggers.

/*
 * nodex
 * 
 * Materialized cache table storing pre-computed node resource information.
 * This table is automatically maintained by triggers when source data changes.
 * 
 * Key features:
 *   - price_usd is a generated column using calc_price() function
 *   - All resource amounts are stored in bytes
 *   - GPU information stored as JSONB array
 *   - Indexed on node_id (primary key) and farm_id for fast lookups
 */
DROP TABLE IF EXISTS nodex;
CREATE TABLE IF NOT EXISTS nodex(
    node_id INTEGER PRIMARY KEY,
    farm_id INTEGER NOT NULL,
    total_hru NUMERIC NOT NULL,
    total_mru NUMERIC NOT NULL,
    total_sru NUMERIC NOT NULL,
    total_cru NUMERIC NOT NULL,
    free_hru NUMERIC NOT NULL,
    free_mru NUMERIC NOT NULL,
    free_sru NUMERIC NOT NULL,
    used_hru NUMERIC NOT NULL,
    used_mru NUMERIC NOT NULL,
    used_sru NUMERIC NOT NULL,
    used_cru NUMERIC NOT NULL,
    renter INTEGER,
    rent_contract_id INTEGER,
    node_contracts_count INTEGER NOT NULL,
    country TEXT,
    bios jsonb,
    baseboard jsonb,
    processor jsonb,
    memory jsonb,
    upload_speed numeric,
    download_speed numeric,
    udp_download_ipv4 numeric,
    udp_upload_ipv4 numeric,
    tcp_download_ipv6 numeric,
    tcp_upload_ipv6 numeric,
    udp_download_ipv6 numeric,
    udp_upload_ipv6 numeric,
    single_threaded_cpu numeric,
    multi_threaded_cpu numeric,
    threads_cpu integer,
    workloads_cpu integer,
    certified BOOLEAN,
    policy_id INTEGER,
    extra_fee NUMERIC,
    gpus jsonb,
    node_gpu_count INTEGER NOT NULL,
    -- Generated column: automatically calculates price in USD
    -- Converts resource amounts from bytes to GB for calc_price function
    price_usd NUMERIC GENERATED ALWAYS AS (
        calc_price(
            total_cru,
            total_sru / get_bytes_per_gb(),  -- Convert bytes to GB
            total_hru / get_bytes_per_gb(),  -- Convert bytes to GB
            total_mru / get_bytes_per_gb(),  -- Convert bytes to GB
            certified,
            policy_id,
            extra_fee
        )
    ) STORED
);

-- Populate cache table from view
INSERT INTO nodex 
SELECT * 
FROM nodex_view;

/*
 * farmx
 * 
 * Materialized cache table storing aggregated public IP information per farm.
 * Tracks total IPs, free IPs (contract_id = 0), and IP details as JSONB.
 * 
 * Automatically maintained by triggers when public_ip table changes.
 */
DROP TABLE IF EXISTS farmx;
CREATE TABLE farmx(
    farm_id INTEGER PRIMARY KEY,
    free_ips INTEGER NOT NULL,      -- Count of IPs with contract_id = 0
    total_ips INTEGER NOT NULL,    -- Total IPs assigned to farm
    ips jsonb                       -- JSON array of all IPs with details
);

-- Populate cache table with aggregated IP data
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

