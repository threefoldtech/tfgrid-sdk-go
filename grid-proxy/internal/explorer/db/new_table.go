package db

/*
	node:
		free_hru, free_mru, free_sru, total_hru, total_mru, total_sru, total_cru,

		total_resources
		contract_resources -> used node resources
		renter -> to know who rented it, available for
		node_contracts -> to know whether it's rentable
		nodeid, farmid

	triggers:
		- trigger on node table (insert)
		- trigger on total resources (insert/update)
		- trigger on contract resources (insert/update)
		- trigger on rent contract (insert/update)
		- trigger on node contract (insert/update)

	triggers need to be in the same transaction with table creation
*/

var resourcesCache = `
	drop table if exists resources_cache;
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
		node_gpu_count INTEGER NOT NULL
	);

	INSERT INTO resources_cache
	SELECT *
	FROM (
        SELECT node.node_id as node_id,
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
            count(node_gpu.id) as node_gpu_count
        FROM contract_resources
            JOIN node_contract as node_contract ON node_contract.resources_used_id = contract_resources.id
            AND node_contract.state IN ('Created', 'GracePeriod')
            RIGHT JOIN node as node ON node.node_id = node_contract.node_id
            JOIN node_resources_total AS node_resources_total ON node_resources_total.node_id = node.id
            LEFT JOIN rent_contract on node.node_id = rent_contract.node_id
            AND rent_contract.state IN ('Created', 'GracePeriod')
            LEFT JOIN node_gpu ON node.twin_id = node_gpu.node_twin_id
        GROUP BY node.node_id,
            node_resources_total.mru,
            node_resources_total.sru,
            node_resources_total.hru,
            node_resources_total.cru,
            node.farm_id,
            rent_contract.contract_id,
            rent_contract.twin_id
    ) as node_resources;
`
