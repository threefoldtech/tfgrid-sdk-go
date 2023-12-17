CREATE MATERIALIZED VIEW node_materialized_view AS
SELECT node.node_id,
    node.twin_id,
    node.farm_id,
    node.power,
    node.updated_at,
    node.certification,
    node.country,
    nodes_resources_view.free_mru,
    nodes_resources_view.free_hru,
    nodes_resources_view.free_sru,
    COALESCE(rent_contract.contract_id, 0) as rent_contract_id,
    COALESCE(rent_contract.twin_id, 0) as renter,
    COALESCE(node_gpu.id, '') as gpu_id
FROM node
    LEFT JOIN nodes_resources_view ON node.node_id = nodes_resources_view.node_id
    LEFT JOIN rent_contract ON node.node_id = rent_contract.node_id
    AND rent_contract.state IN ('Created', 'GracePeriod')
    LEFT JOIN node_gpu ON node.twin_id = node_gpu.node_twin_id;
-- define a refresher function
CREATE OR REPLACE FUNCTION refresh_node_materialized_view() RETURNS TRIGGER AS $$ BEGIN REFRESH MATERIALIZED VIEW node_materialized_view;
RETURN NULL;
END;
$$ LANGUAGE plpgsql;
-- trigger the refresh on each node update
CREATE TRIGGER refresh_node_materialized_trigger
AFTER
UPDATE ON node FOR EACH ROW EXECUTE FUNCTION refresh_node_materialized_view();
-- trigger a backup refresh each 6 hours
SELECT cron.schedule(
        '0 */6 * * *',
        'SELECT refresh_node_materialized_view()'
    );
CREATE MATERIALIZED VIEW farm_materialized_view AS
SELECT farm.id,
    farm.farm_id,
    farm.name,
    farm.twin_id,
    farm.pricing_policy_id,
    farm.certification,
    farm.stellar_address,
    farm.dedicated_farm as dedicated,
    COALESCE(public_ips.public_ips, '[]') as public_ips,
    bool_or(node_materialized_view.rent_contract_id != 0) as has_rent_contract
FROM farm
    LEFT JOIN node_materialized_view ON node_materialized_view.farm_id = farm.farm_id
    LEFT JOIN country ON node_materialized_view.country = country.name
    LEFT JOIN (
        SELECT p1.farm_id,
            COUNT(p1.id) total_ips,
            COUNT(
                CASE
                    WHEN p2.contract_id = 0 THEN 1
                END
            ) free_ips
        FROM public_ip p1
            LEFT JOIN public_ip p2 ON p1.id = p2.id
        GROUP BY p1.farm_id
    ) public_ip_count on public_ip_count.farm_id = farm.id
    LEFT JOIN (
        SELECT farm_id,
            jsonb_agg(
                jsonb_build_object(
                    'id',
                    id,
                    'ip',
                    ip,
                    'contract_id',
                    contract_id,
                    'gateway',
                    gateway
                )
            ) as public_ips
        FROM public_ip
        GROUP BY farm_id
    ) public_ips on public_ips.farm_id = farm.id
GROUP BY farm.id,
    farm.farm_id,
    farm.name,
    farm.twin_id,
    farm.pricing_policy_id,
    farm.certification,
    farm.stellar_address,
    farm.dedicated_farm,
    COALESCE(public_ips.public_ips, '[]');