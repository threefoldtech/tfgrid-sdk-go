BEGIN;

----
-- Helper functions
----
DROP FUNCTION IF EXISTS convert_to_decimal(v_input text);

CREATE OR REPLACE FUNCTION convert_to_decimal(v_input TEXT) RETURNS DECIMAL AS 
$$ 
DECLARE v_dec_value DECIMAL DEFAULT NULL;
BEGIN     
    BEGIN 
        v_dec_value := v_input:: DECIMAL;
    EXCEPTION
        WHEN OTHERS THEN 
            RAISE NOTICE 'Invalid decimal value: "%".  Returning NULL.', v_input;
	    RETURN NULL;
	END;
RETURN v_dec_value;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION calc_discount(
    cost NUMERIC,
    balance NUMERIC
) RETURNS NUMERIC AS $$

DECLARE
    discount NUMERIC;

BEGIN
    discount := (
    CASE 
        WHEN balance >= cost * 18 THEN 0.6
        WHEN balance >= cost * 6 THEN 0.4
        WHEN balance >= cost * 3 THEN 0.3
        WHEN balance >= cost * 1.5 THEN 0.2
        ELSE 0
    END);

    RETURN cost - cost * discount;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

CREATE OR REPLACE FUNCTION calc_price(
    cru NUMERIC,
    sru NUMERIC,
    hru NUMERIC,
    mru NUMERIC,
    certified BOOLEAN,
    policy_id INTEGER,
    extra_fee NUMERIC
) RETURNS NUMERIC AS $$

DECLARE
    su NUMERIC;
    cu NUMERIC;
    su_value NUMERIC;
    cu_value NUMERIC;
    cost_per_month NUMERIC;

BEGIN
    SELECT pricing_policy.cu->'value'
    INTO cu_value
    FROM pricing_policy
    WHERE pricing_policy_id = policy_id;

    SELECT pricing_policy.su->'value'
    INTO su_value
    FROM pricing_policy
    WHERE pricing_policy_id = policy_id;

    IF cu_value IS NULL OR su_value IS NULL THEN
        RAISE EXCEPTION 'pricing values not found for policy_id: %', policy_id;
    END IF;

    cu := (LEAST(
        GREATEST(mru / 4, cru / 2),
        GREATEST(mru / 8, cru),
        GREATEST(mru / 2, cru / 4)
    ));

    su := (hru / 1200 + sru / 200);

    cost_per_month := (cu * cu_value + su * su_value + extra_fee) *
        (CASE certified WHEN true THEN 1.25 ELSE 1 END) *
        (24 * 30);

    RETURN cost_per_month / 10000000; -- 1e7
END;
$$ LANGUAGE plpgsql IMMUTABLE;

----
-- Clean old triggers
----
DROP TRIGGER IF EXISTS node_added ON node;

----
-- Resources cache table
----
DROP VIEW IF EXISTS resources_cache_view;

CREATE OR REPLACE VIEW resources_cache_view AS
SELECT
    node.node_id as node_id,
    node.farm_id as farm_id,
    COALESCE(node_resources_total.hru, 0) as total_hru,
    COALESCE(node_resources_total.mru, 0) as total_mru,
    COALESCE(node_resources_total.sru, 0) as total_sru,
    COALESCE(node_resources_total.cru, 0) as total_cru,
    COALESCE(node_resources_total.hru, 0) - COALESCE(sum(contract_resources.hru), 0) as free_hru,
    COALESCE(node_resources_total.mru, 0) - COALESCE(sum(contract_resources.mru), 0) - GREATEST(CAST((node_resources_total.mru / 10) AS bigint), 2147483648) as free_mru,
    COALESCE(node_resources_total.sru, 0) - COALESCE(sum(contract_resources.sru), 0) - 21474836480 as free_sru,
    COALESCE(sum(contract_resources.hru), 0) as used_hru,
    COALESCE(sum(contract_resources.mru), 0) + GREATEST(CAST( (node_resources_total.mru / 10) AS bigint), 2147483648 ) as used_mru,
    COALESCE(sum(contract_resources.sru), 0) + 21474836480 as used_sru,
    COALESCE(sum(contract_resources.cru), 0) as used_cru,
    rent_contract.twin_id as renter,
    rent_contract.contract_id as rent_contract_id,
    count(node_contract.contract_id) as node_contracts_count,
    COALESCE(node_gpu.node_gpu_count, 0) as node_gpu_count,
    node.country as country,
    country.region as region,
    CASE WHEN node.certification = 'Certified' THEN true ELSE false END as certified,
    CASE WHEN farm.pricing_policy_id = 0 THEN 1 ELSE farm.pricing_policy_id END as policy_id,
    COALESCE(node.extra_fee, 0) as extra_fee
FROM node
    LEFT JOIN node_contract ON node.node_id = node_contract.node_id AND node_contract.state IN ('Created', 'GracePeriod')
    LEFT JOIN contract_resources ON node_contract.resources_used_id = contract_resources.id 
    LEFT JOIN node_resources_total AS node_resources_total ON node_resources_total.node_id = node.id
    LEFT JOIN rent_contract on node.node_id = rent_contract.node_id AND rent_contract.state IN ('Created', 'GracePeriod')
    LEFT JOIN(
        SELECT
            node_twin_id,
            COUNT(id) as node_gpu_count
        FROM node_gpu
        GROUP BY
            node_twin_id
    ) AS node_gpu ON node.twin_id = node_gpu.node_twin_id
    LEFT JOIN country ON LOWER(node.country) = LOWER(country.name)
    LEFT JOIN farm ON farm.farm_id = node.farm_id
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
    node.country,
    node.certification,
    node.extra_fee,
    farm.pricing_policy_id,
    country.region;

DROP TABLE IF EXISTS resources_cache;
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
    region TEXT,
    certified BOOLEAN,
    policy_id INTEGER,
    extra_fee NUMERIC,
    price_usd NUMERIC GENERATED ALWAYS AS (
        calc_price(
            total_cru,
            total_sru / (1024*1024*1024),
            total_hru / (1024*1024*1024),
            total_mru / (1024*1024*1024),
            certified,
            policy_id,
            extra_fee
        )
    ) STORED
    );

INSERT INTO resources_cache 
SELECT * 
FROM resources_cache_view;


----
-- PublicIpsCache table
----
DROP TABLE IF EXISTS public_ips_cache;
CREATE TABLE public_ips_cache(
    farm_id INTEGER PRIMARY KEY,
    free_ips INTEGER NOT NULL,
    total_ips INTEGER NOT NULL,
    ips jsonb
);

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
            COUNT(CASE WHEN p2.contract_id = 0 THEN 1 END) free_ips,
            jsonb_agg(jsonb_build_object('id', p1.id, 'ip', p1.ip, 'contract_id', p1.contract_id, 'gateway', p1.gateway)) as ips
        FROM public_ip AS p1
            LEFT JOIN public_ip p2 ON p1.id = p2.id
        GROUP BY
            p1.farm_id
    ) public_ip_agg on public_ip_agg.farm_id = farm.id;

----
-- Create Indices
----
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE EXTENSION IF NOT EXISTS btree_gin;

CREATE INDEX IF NOT EXISTS idx_node_id ON public.node(node_id);
CREATE INDEX IF NOT EXISTS idx_twin_id ON public.twin(twin_id);
CREATE INDEX IF NOT EXISTS idx_farm_id ON public.farm(farm_id);
CREATE INDEX IF NOT EXISTS idx_node_contract_id ON public.node_contract USING gin(id);
CREATE INDEX IF NOT EXISTS idx_name_contract_id ON public.name_contract USING gin(id);
CREATE INDEX IF NOT EXISTS idx_rent_contract_id ON public.rent_contract USING gin(id);


CREATE INDEX IF NOT EXISTS idx_resources_cache_farm_id ON resources_cache (farm_id);
CREATE INDEX IF NOT EXISTS idx_resources_cache_node_id ON resources_cache(node_id);
CREATE INDEX IF NOT EXISTS idx_public_ips_cache_farm_id ON public_ips_cache(farm_id);

CREATE INDEX IF NOT EXISTS idx_location_id ON location USING gin(id);
CREATE INDEX IF NOT EXISTS idx_public_config_node_id ON public_config USING gin(node_id);

----
--create triggers
----

/*
 Node Trigger:
    - Insert node record > Insert new resources_cache record
    - Update node country > update resources_cache country/region
*/
CREATE OR REPLACE FUNCTION reflect_node_changes() RETURNS TRIGGER AS 
$$ 
BEGIN
    IF (TG_OP = 'UPDATE') THEN
        BEGIN
            UPDATE resources_cache
            SET
                country = NEW.country,
                region = (
                    SELECT region FROM country WHERE LOWER(country.name) = LOWER(NEW.country)
                )
            WHERE
                resources_cache.node_id = NEW.node_id;
        EXCEPTION
            WHEN OTHERS THEN
                RAISE NOTICE 'Error updating resources_cache: %', SQLERRM;
        END;

    ELSIF (TG_OP = 'INSERT') THEN
        BEGIN
            INSERT INTO resources_cache
            SELECT *
            FROM resources_cache_view 
            WHERE resources_cache_view.node_id = NEW.node_id;
        EXCEPTION
            WHEN OTHERS THEN
                RAISE NOTICE 'Error inserting resources_cache: %', SQLERRM;
        END;
    
    ELSIF (TG_OP = 'DELETE') THEN
        BEGIN
            DELETE FROM resources_cache WHERE node_id = OLD.node_id;
        EXCEPTION
            WHEN OTHERS THEN
                RAISE NOTICE 'Error deleting node from resources_cache: %', SQLERRM;
        END;
    END IF;

    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER tg_node
    AFTER INSERT OR DELETE OR UPDATE OF country 
    ON node
    FOR EACH ROW EXECUTE PROCEDURE reflect_node_changes();

/*
 Total resources trigger
    - Insert/Update node_resources_total > Update equivalent resources_cache record.
 */
CREATE OR REPLACE FUNCTION reflect_total_resources_changes() RETURNS TRIGGER AS 
$$ 
BEGIN
    BEGIN
        UPDATE resources_cache
        SET
            total_cru = NEW.cru,
            total_mru = NEW.mru,
            total_sru = NEW.sru,
            total_hru = NEW.hru,
            free_mru = free_mru + GREATEST(CAST((OLD.mru / 10) AS bigint), 2147483648) -
                                    GREATEST(CAST((NEW.mru / 10) AS bigint), 2147483648) + (NEW.mru-COALESCE(OLD.mru, 0)),
            free_hru = free_hru + (NEW.hru-COALESCE(OLD.hru, 0)),
            free_sru = free_sru + (NEW.sru-COALESCE(OLD.sru, 0)),
            used_mru = used_mru - GREATEST(CAST((OLD.mru / 10) AS bigint), 2147483648) +
                                    GREATEST(CAST((NEW.mru / 10) AS bigint), 2147483648)
        WHERE
            resources_cache.node_id = (
                SELECT node.node_id FROM node WHERE node.id = New.node_id
            );
    EXCEPTION
        WHEN OTHERS THEN
            RAISE NOTICE 'Error reflecting total_resources changes %', SQLERRM;
    END;    
RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER tg_node_resources_total 
	AFTER INSERT OR UPDATE 
    ON node_resources_total FOR EACH ROW
	EXECUTE PROCEDURE reflect_total_resources_changes();


/*
 Contract resources
    - Insert/Update contract_resources report > update resources_cache used/free fields
 */

CREATE OR REPLACE FUNCTION reflect_contract_resources_changes() RETURNS TRIGGER AS 
$$ 
BEGIN
    BEGIN
        UPDATE resources_cache
        SET used_cru = used_cru + (NEW.cru - COALESCE(OLD.cru, 0)),
            used_mru = used_mru + (NEW.mru - COALESCE(OLD.mru, 0)),
            used_sru = used_sru + (NEW.sru - COALESCE(OLD.sru, 0)),
            used_hru = used_hru + (NEW.hru - COALESCE(OLD.hru, 0)),
            free_mru = free_mru - (NEW.mru - COALESCE(OLD.mru, 0)),
            free_hru = free_hru - (NEW.hru - COALESCE(OLD.hru, 0)),
            free_sru = free_sru - (NEW.sru - COALESCE(OLD.sru, 0))
        WHERE
            -- (SELECT state from node_contract where id = NEW.contract_id) != 'Deleted' AND
            resources_cache.node_id = (
                SELECT node_id FROM node_contract WHERE node_contract.id = NEW.contract_id
            );
    EXCEPTION
        WHEN OTHERS THEN
            RAISE NOTICE 'Error reflecting contract_resources changes %', SQLERRM;
    END;       
RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER tg_contract_resources
    AFTER INSERT OR UPDATE ON contract_resources FOR EACH ROW 
    EXECUTE PROCEDURE reflect_contract_resources_changes();

/*
    Node contract trigger
     - Insert new contract > increment resources_cache node_contracts_count
     - Update contract state to 'Deleted' > decrement used an increment free fields on resources_cache
*/
CREATE OR REPLACE FUNCTION reflect_node_contract_changes() RETURNS TRIGGER AS 
$$ 
BEGIN
    IF (TG_OP = 'UPDATE' AND NEW.state = 'Deleted') THEN
        BEGIN
            UPDATE resources_cache
            SET (used_cru, used_mru, used_sru, used_hru, free_mru, free_sru, free_hru, node_contracts_count) = 
            (
                SELECT
                    resources_cache.used_cru - cru,
                    resources_cache.used_mru - mru,
                    resources_cache.used_sru - sru,
                    resources_cache.used_hru - hru,
                    resources_cache.free_mru + mru,
                    resources_cache.free_sru + sru,
                    resources_cache.free_hru + hru,
                    resources_cache.node_contracts_count - 1
                FROM resources_cache
                LEFT JOIN contract_resources ON contract_resources.contract_id = NEW.id 
                WHERE resources_cache.node_id = NEW.node_id
            ) WHERE resources_cache.node_id = NEW.node_id;
        EXCEPTION
            WHEN OTHERS THEN
            RAISE NOTICE 'Error reflecting node_contract updates %', SQLERRM;
        END;    

    ELSIF (TG_OP = 'INSERT') THEN
        BEGIN
            UPDATE resources_cache 
            SET node_contracts_count = node_contracts_count + 1 
            WHERE resources_cache.node_id = NEW.node_id;
        EXCEPTION
            WHEN OTHERS THEN
            RAISE NOTICE 'Error incrementing node_contracts_count %', SQLERRM;
        END; 
    END IF;
RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER tg_node_contract
    AFTER INSERT OR UPDATE OF state ON node_contract FOR EACH ROW 
    EXECUTE PROCEDURE reflect_node_contract_changes();

/*
 Gpu trigger
    - Insert new node_gpu > increase the gpu_num in resources cache
    - Delete node_gpu > decrease the gpu_num in resources cache
*/
CREATE OR REPLACE FUNCTION reflect_node_gpu_count_change() RETURNS TRIGGER AS
$$
BEGIN
    BEGIN
        UPDATE resources_cache
        SET node_gpu_count = node_gpu_count + (
            CASE 
            WHEN TG_OP = 'INSERT' 
                THEN 1 
            WHEN TG_OP = 'DELETE'
                THEN -1
            ELSE 0
            END
        )
        WHERE resources_cache.node_id = (
            SELECT node_id from node where node.twin_id = COALESCE(NEW.node_twin_id, OLD.node_twin_id)
        );
    EXCEPTION
        WHEN OTHERS THEN
            RAISE NOTICE 'Error updating resources_cache gpu fields %', SQLERRM;
    END;
RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER tg_node_gpu_count
    AFTER INSERT OR DELETE ON node_gpu FOR EACH ROW
    EXECUTE PROCEDURE reflect_node_gpu_count_change();

/*
 Rent contract trigger
    - Insert new rent contract > Update resources_cache renter/rent_contract_id
    - Update (state to 'Deleted') > nullify resources_cache renter/rent_contract_id
*/

CREATE OR REPLACE FUNCTION reflect_rent_contract_changes() RETURNS TRIGGER AS 
$$ 
BEGIN
    IF (TG_OP = 'UPDATE' AND NEW.state = 'Deleted') THEN
        BEGIN
            UPDATE resources_cache
            SET renter = NULL,
                rent_contract_id = NULL
            WHERE
                resources_cache.node_id = NEW.node_id;
        EXCEPTION
            WHEN OTHERS THEN
                RAISE NOTICE 'Error removing resources_cache rent fields %', SQLERRM;
        END; 
    ELSIF (TG_OP = 'INSERT') THEN
        BEGIN
            UPDATE resources_cache 
            SET renter = NEW.twin_id,
                rent_contract_id = NEW.contract_id
            WHERE
                resources_cache.node_id = NEW.node_id;
        EXCEPTION
            WHEN OTHERS THEN
                RAISE NOTICE 'Error reflecting rent_contract changes %', SQLERRM;
        END; 
    END IF;
RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER tg_rent_contract
    AFTER INSERT OR UPDATE OF state ON rent_contract FOR EACH ROW
    EXECUTE PROCEDURE reflect_rent_contract_changes();

/*
 Public ips trigger
  - Insert new ip > increment free/total ips + re-aggregate ips object
  - Deleted > decrement total, decrement free ips (if it was used) + re-aggregate ips object
  - Update > increment/decrement free ips based on usage + re-aggregate ips object

  - reserve ip > free_ips decrease
  - unreserve ip > free_ips increase
  - insert new ip (expected be free) > free_ips increase
  - remove reserved ip > free_ips does not change
  - remove free ip > free_ips decrease
*/
CREATE OR REPLACE FUNCTION reflect_public_ip_changes() RETURNS TRIGGER AS 
$$ 
BEGIN

    BEGIN 
        UPDATE public_ips_cache
        SET free_ips = free_ips + (
                CASE 
                -- handles insertion/update by freeing ip
                WHEN TG_OP = 'INSERT' AND NEW.contract_id = 0 OR 
                     TG_OP = 'UPDATE' AND NEW.contract_id = 0 AND OLD.contract_id != 0
                    THEN 1 
                -- handles deletion/update by reserving ip
                WHEN TG_OP = 'DELETE' AND OLD.contract_id = 0 OR
                     TG_OP = 'UPDATE' AND OLD.contract_id = 0 AND NEW.contract_id != 0
                    THEN -1
                -- handles delete reserved ips
                ELSE 0
                END
            ),

            total_ips = total_ips + (
                CASE 
                WHEN TG_OP = 'INSERT' 
                    THEN 1 
                WHEn TG_OP = 'DELETE'
                    THEN -1
                ELSE 0
                END
            ),

            ips = (
                SELECT jsonb_agg(
                    jsonb_build_object(
                        'id',
                        public_ip.id,
                        'ip',
                        public_ip.ip,
                        'contract_id',
                        public_ip.contract_id,
                        'gateway',
                        public_ip.gateway
                    )
                )
                -- old/new farm_id are the same
                from public_ip where farm_id = COALESCE(NEW.farm_id, OLD.farm_id)
            )
        WHERE
            public_ips_cache.farm_id = (
                SELECT farm_id FROM farm WHERE farm.id = COALESCE(NEW.farm_id, OLD.farm_id)
            );
    EXCEPTION
        WHEN OTHERS THEN
            RAISE NOTICE 'Error reflect public_ips changes %s', SQLERRM;
    END;

RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER tg_public_ip
    AFTER INSERT OR DELETE OR UPDATE OF contract_id ON public_ip FOR EACH ROW 
    EXECUTE PROCEDURE reflect_public_ip_changes();


CREATE OR REPLACE FUNCTION reflect_farm_changes() RETURNS TRIGGER AS 
$$ 
BEGIN
    IF TG_OP = 'INSERT'  THEN
        BEGIN
            INSERT INTO public_ips_cache VALUES(
                NEW.farm_id,
                0,
                0,
                '[]'
            );
        EXCEPTION
            WHEN OTHERS THEN
                RAISE NOTICE 'Error inserting public_ips_cache record %s', SQLERRM;
        END;

    ELSIF (TG_OP = 'DELETE') THEN
        BEGIN
            DELETE FROM public_ips_cache WHERE public_ips_cache.farm_id = OLD.farm_id;
        EXCEPTION
            WHEN OTHERS THEN
                RAISE NOTICE 'Error deleting public_ips_cache record %s', SQLERRM;
        END; 
    END IF;

RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER tg_farm
    AFTER INSERT OR DELETE ON farm FOR EACH ROW 
    EXECUTE PROCEDURE reflect_farm_changes();

COMMIT;
