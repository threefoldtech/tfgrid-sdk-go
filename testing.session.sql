INSERT INTO node (
    id,
    grid_version,
    node_id,
    farm_id,
    twin_id,
    country,
    city,
    -- uptime,
    created,
    farming_policy_id,
    -- secure,
    -- virtualized,
    -- serial_number,
    created_at,
    updated_at,
    location_id
    -- certification,
    -- connection_price,
    -- power,
    -- extra_fee
  )
VALUES (
    'node-1010',
    3,
    1010,
    100,
    3222,
    'country:text',
    'city:text',
    -- uptime:numeric,
    100000000,
    1,
    -- secure:boolean,
    -- virtualized:boolean,
    -- 'serial_number:text',
    12345678,
    12345678,
    'location-1'
    -- 'certification:character varying',
    -- connection_price:integer,
    -- 'power:jsonb',
    -- extra_fee:numeric
  );