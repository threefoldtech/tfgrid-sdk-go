-- ============================================================================
-- TRIGGERS
-- ============================================================================
-- Trigger functions and triggers that automatically maintain cache tables
-- when source data changes. Each trigger handles a specific data type.

/*
 * reflect_node_changes
 * 
 * Maintains nodex when nodes are inserted or deleted.
 *   - INSERT: Populates cache from view for new node
 *   - DELETE: Removes node from cache
 */
CREATE OR REPLACE FUNCTION reflect_node_changes() RETURNS TRIGGER AS 
$$ 
BEGIN
    IF (TG_OP = 'INSERT') THEN
        BEGIN
            INSERT INTO nodex
            SELECT *
            FROM nodex_view 
            WHERE nodex_view.node_id = NEW.node_id;
        EXCEPTION
            WHEN OTHERS THEN
                PERFORM log_cache_error(
                    'reflect_node_changes',
                    'INSERT',
                    'node',
                    NEW.node_id::TEXT,
                    SQLERRM,
                    jsonb_build_object('node_id', NEW.node_id, 'farm_id', NEW.farm_id)
                );
                RAISE WARNING 'Error inserting nodex: %', SQLERRM;
        END;
    
    ELSIF (TG_OP = 'DELETE') THEN
        BEGIN
            DELETE FROM nodex WHERE node_id = OLD.node_id;
        EXCEPTION
            WHEN OTHERS THEN
                PERFORM log_cache_error(
                    'reflect_node_changes',
                    'DELETE',
                    'node',
                    OLD.node_id::TEXT,
                    SQLERRM,
                    jsonb_build_object('node_id', OLD.node_id)
                );
                RAISE WARNING 'Error deleting node from nodex: %', SQLERRM;
        END;
    END IF;

    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER tg_node
    AFTER INSERT OR DELETE
    ON node
    FOR EACH ROW EXECUTE PROCEDURE reflect_node_changes();

/*
 * reflect_total_resources_changes
 * 
 * Updates cache when node_resources_total changes.
 * 
 * For MRU: Adjusts both the reserved amount (MRU/10, min 2GB) and the total change.
 *   - Reserved amount change: OLD.reserved - NEW.reserved
 *   - Total change: NEW.mru - OLD.mru
 *   - Combined: free_mru = free_mru + (OLD.reserved - NEW.reserved) + (NEW.mru - OLD.mru)
 * 
 * For HRU/SRU: Simple incremental update based on total change.
 */
CREATE OR REPLACE FUNCTION reflect_total_resources_changes() RETURNS TRIGGER AS 
$$ 
BEGIN
    BEGIN
        UPDATE nodex
        SET
            -- Update total resources
            total_cru = NEW.cru,
            total_mru = NEW.mru,
            total_sru = NEW.sru,
            total_hru = NEW.hru,
            -- MRU: Adjust for reserved amount change + total change
            -- Reserved amount: GREATEST(MRU/get_mru_reserved_fraction(), get_mru_reserved_min_bytes())
            free_mru = free_mru + GREATEST(CAST((OLD.mru / get_mru_reserved_fraction()) AS bigint), get_mru_reserved_min_bytes()) -
                                    GREATEST(CAST((NEW.mru / get_mru_reserved_fraction()) AS bigint), get_mru_reserved_min_bytes()) + 
                                    (NEW.mru - COALESCE(OLD.mru, 0)),
            -- HRU: Simple incremental update
            free_hru = free_hru + (NEW.hru - COALESCE(OLD.hru, 0)),
            -- SRU: Simple incremental update
            free_sru = free_sru + (NEW.sru - COALESCE(OLD.sru, 0)),
            -- MRU used: Adjust reserved amount (used includes reserved)
            used_mru = used_mru - GREATEST(CAST((OLD.mru / get_mru_reserved_fraction()) AS bigint), get_mru_reserved_min_bytes()) +
                                    GREATEST(CAST((NEW.mru / get_mru_reserved_fraction()) AS bigint), get_mru_reserved_min_bytes())
            WHERE
            nodex.node_id = (
                SELECT node.node_id FROM node WHERE node.id = NEW.node_id
            );
    EXCEPTION
        WHEN OTHERS THEN
            PERFORM log_cache_error(
                'reflect_total_resources_changes',
                TG_OP,
                'node_resources_total',
                COALESCE(NEW.node_id, OLD.node_id)::TEXT,
                SQLERRM,
                jsonb_build_object(
                    'old_mru', OLD.mru,
                    'new_mru', NEW.mru,
                    'old_hru', OLD.hru,
                    'new_hru', NEW.hru,
                    'old_sru', OLD.sru,
                    'new_sru', NEW.sru
                )
            );
            RAISE WARNING 'Error reflecting total_resources changes: %', SQLERRM;
    END;    
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER tg_node_resources_total 
	AFTER INSERT OR UPDATE 
    ON node_resources_total FOR EACH ROW
	EXECUTE PROCEDURE reflect_total_resources_changes();


/*
 * reflect_contract_resources_changes
 * 
 * Updates cache when contract_resources change.
 * Only processes contracts in 'Created' or 'GracePeriod' states.
 * 
 *   - INSERT/UPDATE: Increments used, decrements free
 *   - DELETE: Decrements used, increments free
 */
CREATE OR REPLACE FUNCTION reflect_contract_resources_changes() RETURNS TRIGGER AS 
$$ 
BEGIN
    IF (TG_OP = 'DELETE') THEN
        BEGIN
            -- Handle DELETE: decrement used resources and increment free resources
            UPDATE nodex
            SET used_cru = used_cru - OLD.cru,
                used_mru = used_mru - OLD.mru,
                used_sru = used_sru - OLD.sru,
                used_hru = used_hru - OLD.hru,
                free_mru = free_mru + OLD.mru,
                free_hru = free_hru + OLD.hru,
                free_sru = free_sru + OLD.sru
            WHERE
            nodex.node_id = (
                    SELECT node_id FROM node_contract 
                    WHERE node_contract.id = OLD.contract_id 
                    AND node_contract.state IN ('Created', 'GracePeriod')
                );
        EXCEPTION
            WHEN OTHERS THEN
                PERFORM log_cache_error(
                    'reflect_contract_resources_changes',
                    'DELETE',
                    'contract_resources',
                    OLD.contract_id::TEXT,
                    SQLERRM,
                    jsonb_build_object('contract_id', OLD.contract_id, 'cru', OLD.cru, 'mru', OLD.mru, 'sru', OLD.sru, 'hru', OLD.hru)
                );
                RAISE WARNING 'Error reflecting contract_resources DELETE: %', SQLERRM;
        END;
    ELSE
        -- Handle INSERT and UPDATE
        BEGIN
            UPDATE nodex
            SET used_cru = used_cru + (NEW.cru - COALESCE(OLD.cru, 0)),
                used_mru = used_mru + (NEW.mru - COALESCE(OLD.mru, 0)),
                used_sru = used_sru + (NEW.sru - COALESCE(OLD.sru, 0)),
                used_hru = used_hru + (NEW.hru - COALESCE(OLD.hru, 0)),
                free_mru = free_mru - (NEW.mru - COALESCE(OLD.mru, 0)),
                free_hru = free_hru - (NEW.hru - COALESCE(OLD.hru, 0)),
                free_sru = free_sru - (NEW.sru - COALESCE(OLD.sru, 0))
            WHERE
            nodex.node_id = (
                    SELECT node_id FROM node_contract 
                    WHERE node_contract.id = NEW.contract_id 
                    AND node_contract.state IN ('Created', 'GracePeriod')
                );
        EXCEPTION
            WHEN OTHERS THEN
                PERFORM log_cache_error(
                    'reflect_contract_resources_changes',
                    TG_OP,
                    'contract_resources',
                    NEW.contract_id::TEXT,
                    SQLERRM,
                    jsonb_build_object('contract_id', NEW.contract_id, 'cru', NEW.cru, 'mru', NEW.mru, 'sru', NEW.sru, 'hru', NEW.hru)
                );
                RAISE WARNING 'Error reflecting contract_resources changes: %', SQLERRM;
        END;
    END IF;
RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER tg_contract_resources
    AFTER INSERT OR UPDATE OR DELETE ON contract_resources FOR EACH ROW 
    EXECUTE PROCEDURE reflect_contract_resources_changes();

/*
 * reflect_node_contract_changes
 * 
 * Updates cache when node_contract state changes.
 * 
 *   - INSERT: Recalculates node_contracts_count (active contracts only)
 *   - UPDATE to 'Deleted': Releases contract resources and updates count
 *     Uses row locking to prevent concurrent update conflicts
 */
CREATE OR REPLACE FUNCTION reflect_node_contract_changes() RETURNS TRIGGER AS 
$$ 
BEGIN
    IF (TG_OP = 'UPDATE' AND NEW.state = 'Deleted') THEN
        BEGIN
            -- Lock cache row to prevent concurrent updates during resource release
            PERFORM 1
            FROM nodex 
            WHERE node_id = NEW.node_id
            FOR UPDATE;

            UPDATE nodex
            SET 
                used_cru = nodex.used_cru - contract_resources.cru,
                used_mru = nodex.used_mru - contract_resources.mru,
                used_sru = nodex.used_sru - contract_resources.sru,
                used_hru = nodex.used_hru - contract_resources.hru,
                free_mru = nodex.free_mru + contract_resources.mru,
                free_sru = nodex.free_sru + contract_resources.sru,
                free_hru = nodex.free_hru + contract_resources.hru,
                node_contracts_count = COALESCE(ncc.count, 0)
            FROM contract_resources
            LEFT JOIN
                (SELECT node_id, COUNT(contract_id) as count
                FROM node_contract
                WHERE state IN ('Created', 'GracePeriod')
                GROUP BY node_id) AS ncc
                ON ncc.node_id = NEW.node_id
            WHERE 
                contract_resources.contract_id = NEW.id 
                AND nodex.node_id = NEW.node_id;
        EXCEPTION
            WHEN OTHERS THEN
                PERFORM log_cache_error(
                    'reflect_node_contract_changes',
                    'UPDATE',
                    'node_contract',
                    NEW.id::TEXT,
                    SQLERRM,
                    jsonb_build_object('contract_id', NEW.id, 'node_id', NEW.node_id, 'state', NEW.state)
                );
                RAISE WARNING 'Error reflecting node_contract updates: %', SQLERRM;
        END;

    ELSIF (TG_OP = 'INSERT') THEN
        BEGIN
            UPDATE nodex 
            SET node_contracts_count = (
                SELECT COALESCE(COUNT(contract_id), 0)
                FROM node_contract
                WHERE node_id = NEW.node_id
                    AND state IN ('Created', 'GracePeriod')
            )
            WHERE nodex.node_id = NEW.node_id;
        EXCEPTION
            WHEN OTHERS THEN
                PERFORM log_cache_error(
                    'reflect_node_contract_changes',
                    'INSERT',
                    'node_contract',
                    NEW.id::TEXT,
                    SQLERRM,
                    jsonb_build_object('contract_id', NEW.id, 'node_id', NEW.node_id)
                );
                RAISE WARNING 'Error calculating node_contracts_count: %', SQLERRM;
        END; 
    END IF;
RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER tg_node_contract
    AFTER INSERT OR UPDATE OF state ON node_contract FOR EACH ROW 
    EXECUTE PROCEDURE reflect_node_contract_changes();

/*
 * reflect_node_gpu_count_change
 * 
 * Updates GPU information in cache when node_gpu changes.
 * Re-aggregates all GPUs for the node and updates count + JSON array.
 * 
 *   - INSERT/DELETE/UPDATE: Recalculates GPU count and JSON array
 */
CREATE OR REPLACE FUNCTION reflect_node_gpu_count_change() RETURNS TRIGGER AS
$$
BEGIN
    BEGIN
        UPDATE nodex
            SET node_gpu_count = gpu.count, gpus = gpu.gpus
            FROM (
              SELECT COUNT(*) AS count,
                jsonb_agg(
                  jsonb_build_object(
                      'id', id,
                      'vendor', vendor,
                      'device', device,
                      'vram', vram,
                      'contract', contract
                )
              ) AS gpus
              FROM node_gpu 
              WHERE node_twin_id = COALESCE(NEW.node_twin_id, OLD.node_twin_id)
            ) AS gpu
        WHERE nodex.node_id = (
            SELECT node_id from node where node.twin_id = COALESCE(NEW.node_twin_id, OLD.node_twin_id)
        );
    EXCEPTION
        WHEN OTHERS THEN
            PERFORM log_cache_error(
                'reflect_node_gpu_count_change',
                TG_OP,
                'node_gpu',
                COALESCE(NEW.id, OLD.id)::TEXT,
                SQLERRM,
                jsonb_build_object('node_twin_id', COALESCE(NEW.node_twin_id, OLD.node_twin_id))
            );
                RAISE WARNING 'Error updating nodex gpu fields: %', SQLERRM;
    END;
RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER tg_node_gpu_count
    AFTER INSERT OR DELETE OR UPDATE ON node_gpu FOR EACH ROW
    EXECUTE PROCEDURE reflect_node_gpu_count_change();

/*
 * reflect_rent_contract_changes
 * 
 * Updates rental information in cache when rent_contract changes.
 * 
 *   - INSERT: Sets renter and rent_contract_id
 *   - UPDATE to 'Deleted': Clears renter and rent_contract_id
 */
CREATE OR REPLACE FUNCTION reflect_rent_contract_changes() RETURNS TRIGGER AS 
$$ 
BEGIN
    IF (TG_OP = 'UPDATE' AND NEW.state = 'Deleted') THEN
        BEGIN
            UPDATE nodex
            SET renter = NULL,
                rent_contract_id = NULL
            WHERE
                nodex.node_id = NEW.node_id;
        EXCEPTION
            WHEN OTHERS THEN
                PERFORM log_cache_error(
                    'reflect_rent_contract_changes',
                    'UPDATE',
                    'rent_contract',
                    NEW.contract_id::TEXT,
                    SQLERRM,
                    jsonb_build_object('contract_id', NEW.contract_id, 'node_id', NEW.node_id, 'state', NEW.state)
                );
                RAISE WARNING 'Error removing nodex rent fields: %', SQLERRM;
        END; 
    ELSIF (TG_OP = 'INSERT') THEN
        BEGIN
            UPDATE nodex 
            SET renter = NEW.twin_id,
                rent_contract_id = NEW.contract_id
            WHERE
                nodex.node_id = NEW.node_id;
        EXCEPTION
            WHEN OTHERS THEN
                PERFORM log_cache_error(
                    'reflect_rent_contract_changes',
                    'INSERT',
                    'rent_contract',
                    NEW.contract_id::TEXT,
                    SQLERRM,
                    jsonb_build_object('contract_id', NEW.contract_id, 'node_id', NEW.node_id, 'twin_id', NEW.twin_id)
                );
                RAISE WARNING 'Error reflecting rent_contract changes: %', SQLERRM;
        END; 
    END IF;
RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER tg_rent_contract
    AFTER INSERT OR UPDATE OF state ON rent_contract FOR EACH ROW
    EXECUTE PROCEDURE reflect_rent_contract_changes();

/*
 * reflect_dmi_changes
 * 
 * Updates DMI (hardware information) in cache when dmi table changes.
 * 
 *   - INSERT/UPDATE: Updates bios, baseboard, processor, memory fields
 */
CREATE OR REPLACE FUNCTION reflect_dmi_changes() RETURNS TRIGGER AS 
$$ 
BEGIN
    BEGIN
        UPDATE nodex
        SET bios = NEW.bios,
            baseboard = NEW.baseboard,
            processor = NEW.processor,
            memory = NEW.memory
        WHERE nodex.node_id = (
            SELECT node_id from node where node.twin_id = NEW.node_twin_id
        );
    EXCEPTION
        WHEN OTHERS THEN
            PERFORM log_cache_error(
                'reflect_dmi_changes',
                TG_OP,
                'dmi',
                NULL,
                SQLERRM,
                jsonb_build_object('node_twin_id', NEW.node_twin_id)
            );
                RAISE WARNING 'Error updating nodex dmi fields: %', SQLERRM;
    END; 
RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER tg_dmi
    AFTER INSERT OR UPDATE ON dmi FOR EACH ROW
    EXECUTE PROCEDURE reflect_dmi_changes();


/*
 * reflect_speed_changes
 * 
 * Updates network speed test results in cache when speed table changes.
 * 
 *   - INSERT/UPDATE: Updates all speed metrics (upload, download, IPv4/IPv6, TCP/UDP)
 */
CREATE OR REPLACE FUNCTION reflect_speed_changes() RETURNS TRIGGER AS 
$$ 
BEGIN
    BEGIN
        UPDATE nodex
        SET upload_speed = NEW.upload,
            download_speed = NEW.download,
            udp_download_ipv4 = NEW.udp_download_ipv4,
            udp_upload_ipv4 = NEW.udp_upload_ipv4,
            tcp_download_ipv6 = NEW.tcp_download_ipv6,
            tcp_upload_ipv6 = NEW.tcp_upload_ipv6,
            udp_download_ipv6 = NEW.udp_download_ipv6,
            udp_upload_ipv6 = NEW.udp_upload_ipv6
        WHERE nodex.node_id = (
            SELECT node_id from node where node.twin_id = NEW.node_twin_id
        );
    EXCEPTION
        WHEN OTHERS THEN
            PERFORM log_cache_error(
                'reflect_speed_changes',
                TG_OP,
                'speed',
                NULL,
                SQLERRM,
                jsonb_build_object('node_twin_id', NEW.node_twin_id)
            );
                RAISE WARNING 'Error updating nodex speed fields: %', SQLERRM;
    END; 
RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER tg_speed
    AFTER INSERT OR UPDATE ON speed FOR EACH ROW
    EXECUTE PROCEDURE reflect_speed_changes();

/*
 * reflect_cpu_benchmark_changes
 * 
 * Updates CPU benchmark results in cache when cpu_benchmark table changes.
 * 
 *   - INSERT/UPDATE: Updates single-threaded, multi-threaded, threads, workloads
 */
CREATE OR REPLACE FUNCTION reflect_cpu_benchmark_changes() RETURNS TRIGGER AS 
$$ 
BEGIN
    BEGIN
        UPDATE nodex
        SET single_threaded_cpu = NEW.single_threaded,
            multi_threaded_cpu = NEW.multi_threaded,
            threads_cpu = NEW.threads,
            workloads_cpu = NEW.workloads
        WHERE nodex.node_id = (
            SELECT node_id from node where node.twin_id = NEW.node_twin_id
        );
    EXCEPTION
        WHEN OTHERS THEN
            PERFORM log_cache_error(
                'reflect_cpu_benchmark_changes',
                TG_OP,
                'cpu_benchmark',
                NULL,
                SQLERRM,
                jsonb_build_object('node_twin_id', NEW.node_twin_id)
            );
                RAISE WARNING 'Error updating nodex cpu_benchmark fields: %', SQLERRM;
    END; 
RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER tg_cpu_benchmark
    AFTER INSERT OR UPDATE ON cpu_benchmark FOR EACH ROW
    EXECUTE PROCEDURE reflect_cpu_benchmark_changes();

/*
 * reflect_public_ip_changes
 * 
 * Updates farmx when public_ip changes.
 * Tracks free_ips (contract_id = 0) and total_ips, re-aggregates IP JSON.
 * 
 * Logic:
 *   - INSERT with contract_id=0: free_ips++, total_ips++
 *   - DELETE with contract_id=0: free_ips--, total_ips--
 *   - DELETE with contract_id!=0: free_ips unchanged, total_ips--
 *   - UPDATE: free/reserved: free_ips--, reserved/free: free_ips++
 *   - Always re-aggregates entire IP JSON array for the farm
 */
CREATE OR REPLACE FUNCTION reflect_public_ip_changes() RETURNS TRIGGER AS 
$$ 
BEGIN

    BEGIN 
        UPDATE farmx
        SET free_ips = free_ips + (
                CASE 
                -- handles insertion/update by freeing ip
                WHEN (TG_OP = 'INSERT' AND NEW.contract_id = 0) OR 
                     (TG_OP = 'UPDATE' AND NEW.contract_id = 0 AND OLD.contract_id != 0)
                    THEN 1 
                -- handles deletion/update by reserving ip
                WHEN (TG_OP = 'DELETE' AND OLD.contract_id = 0) OR
                     (TG_OP = 'UPDATE' AND OLD.contract_id = 0 AND NEW.contract_id != 0)
                    THEN -1
                -- handles delete reserved ips (no change to free_ips)
                ELSE 0
                END
            ),

            total_ips = total_ips + (
                CASE 
                WHEN TG_OP = 'INSERT'
                    THEN 1 
                WHEN TG_OP = 'DELETE'
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
            farmx.farm_id = (
                SELECT farm_id FROM farm WHERE farm.id = COALESCE(NEW.farm_id, OLD.farm_id)
            );
    EXCEPTION
        WHEN OTHERS THEN
            PERFORM log_cache_error(
                'reflect_public_ip_changes',
                TG_OP,
                'public_ip',
                COALESCE(NEW.id, OLD.id)::TEXT,
                SQLERRM,
                jsonb_build_object(
                    'farm_id', COALESCE(NEW.farm_id, OLD.farm_id),
                    'contract_id_old', OLD.contract_id,
                    'contract_id_new', NEW.contract_id
                )
            );
            RAISE WARNING 'Error reflecting public_ips changes: %', SQLERRM;
    END;

RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER tg_public_ip
    AFTER INSERT OR DELETE OR UPDATE OF contract_id ON public_ip FOR EACH ROW 
    EXECUTE PROCEDURE reflect_public_ip_changes();


/*
 * reflect_farm_changes
 * 
 * Maintains farmx when farms are inserted or deleted.
 * 
 *   - INSERT: Creates empty cache entry (0 IPs)
 *   - DELETE: Removes farm from cache
 */
CREATE OR REPLACE FUNCTION reflect_farm_changes() RETURNS TRIGGER AS 
$$ 
BEGIN
    IF TG_OP = 'INSERT' THEN
        BEGIN
            INSERT INTO farmx VALUES(
                NEW.farm_id,
                0,
                0,
                '[]'
            );
        EXCEPTION
            WHEN OTHERS THEN
                PERFORM log_cache_error(
                    'reflect_farm_changes',
                    'INSERT',
                    'farm',
                    NEW.farm_id::TEXT,
                    SQLERRM,
                    jsonb_build_object('farm_id', NEW.farm_id)
                );
                RAISE WARNING 'Error inserting farmx record: %', SQLERRM;
        END;

    ELSIF (TG_OP = 'DELETE') THEN
        BEGIN
            DELETE FROM farmx WHERE farmx.farm_id = OLD.farm_id;
        EXCEPTION
            WHEN OTHERS THEN
                PERFORM log_cache_error(
                    'reflect_farm_changes',
                    'DELETE',
                    'farm',
                    OLD.farm_id::TEXT,
                    SQLERRM,
                    jsonb_build_object('farm_id', OLD.farm_id)
                );
                RAISE WARNING 'Error deleting farmx record: %', SQLERRM;
        END; 
    END IF;

RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER tg_farm
    AFTER INSERT OR DELETE ON farm FOR EACH ROW 
    EXECUTE PROCEDURE reflect_farm_changes();

