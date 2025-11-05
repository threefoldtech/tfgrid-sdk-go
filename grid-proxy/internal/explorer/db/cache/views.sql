-- Base views for cache table structure

DROP TRIGGER IF EXISTS node_added ON node;
DROP VIEW IF EXISTS nodex_view;

CREATE OR REPLACE VIEW nodex_view AS
SELECT
    node.node_id as node_id,
    node.farm_id as farm_id,
    COALESCE(node_resources_total.hru, 0) as total_hru,
    COALESCE(node_resources_total.mru, 0) as total_mru,
    COALESCE(node_resources_total.sru, 0) as total_sru,
    COALESCE(node_resources_total.cru, 0) as total_cru,
    COALESCE(node_resources_total.hru, 0) - COALESCE(sum(contract_resources.hru), 0) as free_hru,
    COALESCE(node_resources_total.mru, 0) - COALESCE(sum(contract_resources.mru), 0) - GREATEST(CAST((node_resources_total.mru / get_mru_reserved_fraction()) AS bigint), get_mru_reserved_min_bytes()) as free_mru,
    COALESCE(node_resources_total.sru, 0) - COALESCE(sum(contract_resources.sru), 0) - get_sru_reserved_bytes() as free_sru,
    COALESCE(sum(contract_resources.hru), 0) as used_hru,
    COALESCE(sum(contract_resources.mru), 0) + GREATEST(CAST((node_resources_total.mru / get_mru_reserved_fraction()) AS bigint), get_mru_reserved_min_bytes()) as used_mru,
    COALESCE(sum(contract_resources.sru), 0) + get_sru_reserved_bytes() as used_sru,
    COALESCE(sum(contract_resources.cru), 0) as used_cru,
    rent_contract.twin_id as renter,
    rent_contract.contract_id as rent_contract_id,
    count(node_contract.contract_id) as node_contracts_count,
    node.country as country,
    COALESCE(dmi.bios, '{}') as bios,
    COALESCE(dmi.baseboard, '{}') as baseboard,
    COALESCE(dmi.processor, '[]') as processor,
    COALESCE(dmi.memory, '[]') as memory,
    COALESCE(speed.upload, 0) as upload_speed,
    COALESCE(speed.download, 0) as download_speed,
    COALESCE(speed.udp_download_ipv4, 0) as udp_download_ipv4,
    COALESCE(speed.udp_upload_ipv4, 0) as udp_upload_ipv4,
    COALESCE(speed.tcp_download_ipv6, 0) as tcp_download_ipv6,
    COALESCE(speed.tcp_upload_ipv6, 0) as tcp_upload_ipv6,
    COALESCE(speed.udp_download_ipv6, 0) as udp_download_ipv6,
    COALESCE(speed.udp_upload_ipv6, 0) as udp_upload_ipv6,
    COALESCE(cpu_benchmark.single_threaded, 0) as single_threaded_cpu,
    COALESCE(cpu_benchmark.multi_threaded, 0) as multi_threaded_cpu,
    COALESCE(cpu_benchmark.threads, 0) as threads_cpu,
    COALESCE(cpu_benchmark.workloads, 0) as workloads_cpu,
    CASE WHEN node.certification = 'Certified' THEN true ELSE false END as certified,
    CASE WHEN farm.pricing_policy_id = 0 THEN 1 ELSE farm.pricing_policy_id END as policy_id,
    COALESCE(node.extra_fee, 0) as extra_fee,
    COALESCE(node_gpu_agg.gpus, '[]'),
    COALESCE(node_gpu_agg.gpu_count, 0) as node_gpu_count
FROM node
    LEFT JOIN node_contract ON node.node_id = node_contract.node_id AND node_contract.state IN ('Created', 'GracePeriod')
    LEFT JOIN contract_resources ON node_contract.resources_used_id = contract_resources.id 
    LEFT JOIN node_resources_total AS node_resources_total ON node_resources_total.node_id = node.id
    LEFT JOIN rent_contract on node.node_id = rent_contract.node_id AND rent_contract.state IN ('Created', 'GracePeriod')
    LEFT JOIN speed ON node.twin_id = speed.node_twin_id
    LEFT JOIN cpu_benchmark ON node.twin_id = cpu_benchmark.node_twin_id
    LEFT JOIN dmi ON node.twin_id = dmi.node_twin_id
    LEFT JOIN farm ON farm.farm_id = node.farm_id
    LEFT JOIN(
        SELECT
            g1.node_twin_id,
            COUNT(g1.id) gpu_count,
            jsonb_agg(jsonb_build_object('id', g1.id, 'vendor', g1.vendor, 'vram', g1.vram, 'contract', g1.contract, 'device', g1.device)) as gpus
        FROM node_gpu AS g1
        GROUP BY
            g1.node_twin_id
    ) node_gpu_agg on node_gpu_agg.node_twin_id = node.twin_id
GROUP BY
    node.node_id,
    node_resources_total.mru,
    node_resources_total.sru,
    node_resources_total.hru,
    node_resources_total.cru,
    node.farm_id,
    rent_contract.contract_id,
    rent_contract.twin_id,
    COALESCE(node_gpu_agg.gpus, '[]'),
    COALESCE(node_gpu_agg.gpu_count, 0),
    node.country,
    COALESCE(dmi.bios, '{}'),
    COALESCE(dmi.baseboard, '{}'),
    COALESCE(dmi.processor, '[]'),
    COALESCE(dmi.memory, '[]'),
    COALESCE(speed.upload, 0),
    COALESCE(speed.download, 0),
    COALESCE(speed.udp_download_ipv4, 0),
    COALESCE(speed.udp_upload_ipv4, 0),
    COALESCE(speed.tcp_download_ipv6, 0),
    COALESCE(speed.tcp_upload_ipv6, 0),
    COALESCE(speed.udp_download_ipv6, 0),
    COALESCE(speed.udp_upload_ipv6, 0),
    COALESCE(cpu_benchmark.single_threaded, 0),
    COALESCE(cpu_benchmark.multi_threaded, 0),
    COALESCE(cpu_benchmark.threads, 0),
    COALESCE(cpu_benchmark.workloads, 0),
    node.certification,
    node.extra_fee,
    farm.pricing_policy_id;

