/*
 node:
 free_hru, free_mru, free_sru, total_hru, total_mru, total_sru, total_cru,
 
 total_resources
 contract_resources -> used node resources
 renter -> to know who rented it, available for
 node_contracts -> to know whether it's rentable
 nodeid, farmid
 
 triggers:
 - trigger on node table (insert/update)
 - trigger on total resources (insert/update)
 - trigger on contract resources (insert/update)
 - trigger on rent contract (insert/update)
 - trigger on node contract (insert/update)
 
 triggers need to be in the same transaction with table creation
 */
BEGIN;

----
-- Helper functions
----
DROP FUNCTION IF EXISTS convert_to_decimal(v_input text);

CREATE
OR REPLACE FUNCTION convert_to_decimal(v_input text) RETURNS DECIMAL AS $ $ DECLARE v_dec_value DECIMAL DEFAULT NULL;

BEGIN BEGIN v_dec_value := v_input :: DECIMAL;

EXCEPTION
WHEN OTHERS THEN RAISE NOTICE 'Invalid decimal value: "%".  Returning NULL.',
v_input;

RETURN NULL;

END;

RETURN v_dec_value;

END;

$ $ LANGUAGE plpgsql;

----
-- Clean old triggers
----
DROP TRIGGER IF EXISTS node_added ON node;

----
-- Resources cache table
----
DROP TABLE IF EXISTS resources_cache;

DROP VIEW IF EXISTS resources_cache_view;

CREATE
OR REPLACE VIEW resources_cache_view AS
SELECT
    node.node_id as node_id,
    node.farm_id as farm_id,
    COALESCE(node_resources_total.hru, 0) as total_hru,
    COALESCE(node_resources_total.mru, 0) as total_mru,
    COALESCE(node_resources_total.sru, 0) as total_sru,
    COALESCE(node_resources_total.cru, 0) as total_cru,
    node_resources_total.hru - COALESCE(sum(contract_resources.hru), 0) as free_hru,
    node_resources_total.mru - COALESCE(sum(contract_resources.mru), 0) - GREATEST(
        CAST((node_resources_total.mru / 10) AS bigint),
        2147483648
    ) as free_mru,
    node_resources_total.sru - COALESCE(sum(contract_resources.sru), 0) - 21474836480 as free_sru,
    COALESCE(sum(contract_resources.hru), 0) as used_hru,
    COALESCE(sum(contract_resources.mru), 0) + GREATEST(
        CAST((node_resources_total.mru / 10) AS bigint),
        2147483648
    ) as used_mru,
    COALESCE(sum(contract_resources.sru), 0) + 21474836480 as used_sru,
    COALESCE(sum(contract_resources.cru), 0) as used_cru,
    rent_contract.twin_id as renter,
    rent_contract.contract_id as rent_contract_id,
    count(node_contract.contract_id) as node_contract_count,
    COALESCE(node_gpu.node_gpu_count, 0) as node_gpu_count,
    country.name as country,
    country.subregion as region
FROM
    contract_resources
    JOIN node_contract as node_contract ON node_contract.resources_used_id = contract_resources.id
    AND node_contract.state IN ('Created', 'GracePeriod')
    RIGHT JOIN node as node ON node.node_id = node_contract.node_id
    JOIN node_resources_total AS node_resources_total ON node_resources_total.node_id = node.id
    LEFT JOIN rent_contract on node.node_id = rent_contract.node_id
    AND rent_contract.state IN ('Created', 'GracePeriod')
    Left JOIN(
        SELECT
            node_twin_id,
            COUNT(id) as node_gpu_count
        FROM
            node_gpu
        GROUP BY
            node_twin_id
    ) AS node_gpu ON node.twin_id = node_gpu.node_twin_id
    LEFT JOIN country ON node.country = country.name
GROUP BY
    node.node_id,
    node_resources_total.mru,
    node_resources_total.sru,
    node_resources_total.hru,
    node_resources_total.cru,
    node.farm_id,
    rent_contract.contract_id,
    rent_contract.twin_id,
    COALESCE(node_gpu.node_gpu_count, 0),
    country.name,
    country.subregion;

CREATE TABLE IF NOT EXISTS resources_cache(
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
    node_gpu_count INTEGER NOT NULL,
    country TEXT,
    region TEXT
);

INSERT INTO
    resources_cache
SELECT
    *
FROM
    resources_cache_view;

----
-- PublicIpsCache table
----
DROP TABLE IF EXISTS public_ips_cache;

CREATE TABLE IF NOT EXISTS public_ips_cache(
    farm_id INTEGER PRIMARY KEY,
    free_ips INTEGER NOT NULL,
    total_ips INTEGER NOT NULL,
    ips jsonb
);

INSERT INTO
    public_ips_cache
SELECT
    farm.farm_id,
    COALESCE(public_ip_agg.free_ips, 0),
    COALESCE(public_ip_agg.total_ips, 0),
    COALESCE(public_ip_agg.ips, '[]')
FROM
    farm
    LEFT JOIN(
        SELECT
            p1.farm_id,
            COUNT(p1.id) total_ips,
            COUNT(
                CASE
                    WHEN p2.contract_id = 0 THEN 1
                END
            ) free_ips,
            jsonb_agg(
                jsonb_build_object(
                    'id',
                    p1.id,
                    'ip',
                    p1.ip,
                    'contract_id',
                    p1.contract_id,
                    'gateway',
                    p1.gateway
                )
            ) as ips
        FROM
            public_ip p1
            LEFT JOIN public_ip p2 ON p1.id = p2.id
        GROUP BY
            p1.farm_id
    ) public_ip_agg on public_ip_agg.farm_id = farm.id;

----
-- Create Indices 
----
CREATE EXTENSION pg_trgm;

CREATE EXTENSION btree_gin;

CREATE INDEX IF NOT EXISTS idx_node_id ON public.node(node_id);

CREATE INDEX IF NOT EXISTS idx_twin_id ON public.twin(twin_id);

CREATE INDEX IF NOT EXISTS idx_farm_id ON public.farm(farm_id);

CREATE INDEX IF NOT EXISTS idx_contract_id ON public.node_contract(contract_id);

CREATE INDEX IF NOT EXISTS resources_cache_farm_id ON resources_cache (farm_id);

CREATE INDEX IF NOT EXISTS location_id ON location USING gin(id);

CREATE INDEX IF NOT EXISTS resources_cache_node_id ON resources_cache(node_id);

CREATE INDEX IF NOT EXISTS public_ips_cache_farm_id ON public_ips_cache(farm_id);

CREATE INDEX IF NOT EXISTS public_config_node_id ON public_config USING gin(node_id);

----
--create triggers
----
CREATE OR REPLACE FUNCTION node_upsert() RETURNS TRIGGER AS $$ 
    BEGIN 

        IF (TG_OP = 'UPDATE') THEN
            UPDATE resources_cache SET country = NEW.country
            WHERE resources_cache.node_id = NEW.node_id;
        ELSIF (TG_OP = 'INSERT') THEN
        INSERT INTO
            resources_cache SELECT * FROM resources_cache_view;

        END IF;
    END;

$$ LANGUAGE plpgsql;

CREATE TRIGGER node_trigger
AFTER INSERT OR UPDATE
    ON node FOR EACH ROW EXECUTE PROCEDURE node_upsert();

CREATE OR REPLACE FUNCTION total_resources_upsert() RETURNS TRIGGER AS $$ 
    BEGIN 

        UPDATE resources_cache 
            SET total_cru = NEW.total_cru, 
            total_mru = NEW.total_mru, 
            total_sru = New.total_sru, 
            total_hru = NEW.total_hru
        WHERE resources_cache.id = NEW.node_id;

    END;

$$ LANGUAGE plpgsql;

CREATE TRIGGER total_resources_trigger
AFTER INSERT OR UPDATE
    ON total_resources FOR EACH ROW EXECUTE PROCEDURE total_resources_trigger();


COMMIT;