-- Cache tables for pre-computed data

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
    price_usd NUMERIC GENERATED ALWAYS AS (
        calc_price(
            total_cru,
            total_sru / get_bytes_per_gb(),
            total_hru / get_bytes_per_gb(),
            total_mru / get_bytes_per_gb(),
            certified,
            policy_id,
            extra_fee
        )
    ) STORED
);

INSERT INTO nodex 
SELECT * 
FROM nodex_view;

DROP TABLE IF EXISTS farmx;
CREATE TABLE farmx(
    farm_id INTEGER PRIMARY KEY,
    free_ips INTEGER NOT NULL,
    total_ips INTEGER NOT NULL,
    ips jsonb
);

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

