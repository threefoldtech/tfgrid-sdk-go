-- node modifiers
INSERT INTO node (
    id,
    grid_version,
    node_id,
    farm_id,
    twin_id,
    country,
    city,
    uptime,
    created,
    farming_policy_id,
    secure,
    virtualized,
    serial_number,
    created_at,
    updated_at,
    location_id,
    certification,
    connection_price,
    power,
    extra_fee
  )
VALUES (
    'node-id-999999',
    3,
    999999,
    1,
    999999,
    'Egypt',
    'Cairo',
    1651769370,
    1651769370,
    1,
    false,
    false,
    '',
    1651769370,
    1651769370,
    'location-1',
    'not',
    0,
    '{}',
    0
  );
UPDATE node
SET country = 'Belgium'
WHERE node_id = 999999;
INSERT INTO node_resources_total (id, hru, sru, cru, mru, node_id)
VALUES (
    'total-resources-999999',
    10000000000000,
    10000000000000,
    1000,
    10000000000000,
    'node-id-999999'
  );
UPDATE node_resources_total
SET cru = 2000,
  hru = 20000000000000,
  mru = 20000000000000,
  sru = 20000000000000
WHERE node_id = 'node-id-999999';
-- capacity modifiers
INSERT INTO node_contract (
    id,
    grid_version,
    contract_id,
    twin_id,
    node_id,
    deployment_data,
    deployment_hash,
    number_of_public_i_ps,
    created_at,
    resources_used_id,
    state
  )
VALUES (
    'node-contract-999999',
    1,
    999999,
    99,
    999999,
    'deployment_data:text',
    'deployment_hash:text',
    0,
    1600000000,
    NULL,
    'Created'
  );
INSERT INTO node_contract (
    id,
    grid_version,
    contract_id,
    twin_id,
    node_id,
    deployment_data,
    deployment_hash,
    number_of_public_i_ps,
    created_at,
    resources_used_id,
    state
  )
VALUES (
    'node-contract-999998',
    1,
    999998,
    99,
    999999,
    'deployment_data:text',
    'deployment_hash:text',
    0,
    1600000000,
    NULL,
    'Created'
  );
INSERT INTO node_contract (
    id,
    grid_version,
    contract_id,
    twin_id,
    node_id,
    deployment_data,
    deployment_hash,
    number_of_public_i_ps,
    created_at,
    resources_used_id,
    state
  )
VALUES (
    'node-contract-999995',
    1,
    999995,
    99,
    999999,
    'deployment_data:text',
    'deployment_hash:text',
    0,
    1600000000,
    NULL,
    'Created'
  );
INSERT INTO contract_resources (id, hru, sru, cru, mru, contract_id)
VALUES (
    'contract-resources-999999',
    1,
    1,
    1,
    1,
    'node-contract-999999'
  );
INSERT INTO contract_resources (id, hru, sru, cru, mru, contract_id)
VALUES (
    'contract-resources-999998',
    1,
    1,
    1,
    1,
    'node-contract-999998'
  );
UPDATE contract_resources
SET cru = 1
where contract_id = 'node-contract-999999';
update node_contract
SET state = 'Deleted'
where contract_id = 999999;
-- renting modifiers
INSERT INTO rent_contract (
    id,
    grid_version,
    contract_id,
    twin_id,
    node_id,
    created_at,
    state
  )
VALUES (
    'rent-contract-999997',
    1,
    999997,
    99,
    999999,
    15000000,
    'Created'
  );
Update rent_contract
set state = 'Deleted'
where contract_id = 999997;
INSERT INTO rent_contract (
    id,
    grid_version,
    contract_id,
    twin_id,
    node_id,
    created_at,
    state
  )
VALUES (
    'rent-contract-999996',
    1,
    999996,
    99,
    999999,
    15000000,
    'Created'
  );
-- ips modifiers
INSERT INTO public_ip (id, gateway, ip, contract_id, farm_id)
VALUES (
    'public-ip-999999',
    'gateway:text',
    'ip:text',
    0,
    'farm-1'
  );
INSERT INTO public_ip (id, gateway, ip, contract_id, farm_id)
VALUES (
    'public-ip-999998',
    'gateway:text',
    'ip:text',
    0,
    'farm-1'
  );
update public_ip
set contract_id = 999998
where id = 'public-ip-999999';
update public_ip
set contract_id = 999995
where id = 'public-ip-999998';
update public_ip
set contract_id = 0
where id = 'public-ip-999999';
Delete from public_ip
where id = 'public-ip-999999';
Delete from public_ip
where id = 'public-ip-999998';