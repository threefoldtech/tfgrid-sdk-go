/*
 Rent events
 - [x] farm set to dedicated
 - [x] farm set to un dedicated
 
 - [ ] add/remove a first/last contract on unrented node
 - [ ] add/remove a first/last contract on rented node
 - [ ] add/remove a non first/last contract
 
 - [x] add rent contract
 - [x] remove rent contract
 */

BEGIN;
DO $$

DECLARE 
    rand_node_id NUMERIC;
    old_dedicated_node BOOLEAN;
    old_shared BOOLEAN;
    old_rentable BOOLEAN;
    old_rented BOOLEAN;
    old_renter NUMERIC;
    rand_id TEXT := gen_random_uuid()::text;
    rand_twin_id NUMERIC := 29;
BEGIN
    -- read data of random unrented node
    SELECT node_id, dedicated_node, shared, rentable, rented, renter
    INTO rand_node_id, old_dedicated_node, old_shared, old_rentable, old_rented, old_renter
    FROM resources_cache
    WHERE rented = FALSE
    ORDER BY random()
    LIMIT 1;
    
    -- create a rent contract
    INSERT INTO rent_contract (
        id, grid_version, contract_id, twin_id, node_id, created_at, state
    )
    VALUES (
        rand_id, 3, 99999, rand_twin_id::INTEGER, rand_node_id::INTEGER, 5646848616, 'Created'
    );

    -- assert the result
    ASSERT (
        SELECT (dedicated_node, shared, rentable, rented, renter::NUMERIC) 
        FROM resources_cache 
        WHERE node_id = rand_node_id
    ) = (TRUE, FALSE, FALSE, TRUE, rand_twin_id::NUMERIC);

    -- delete the rent contract 
    UPDATE rent_contract
        SET state = 'Deleted'
        WHERE id = rand_id;

    ASSERT (
        SELECT (dedicated_node, shared, rentable, rented, renter::NUMERIC) 
        FROM resources_cache 
        WHERE node_id = rand_node_id
    ) = (old_dedicated_node, old_shared, old_rentable, old_rented, old_renter::NUMERIC);
    
END $$;

DO $$
DECLARE
    old_dedicated_farm BOOLEAN;
    old_dedicated_node BOOLEAN;
    old_shared BOOLEAN;
    old_rentable BOOLEAN;
    rand_node_id NUMERIC;
    rand_farm_id NUMERIC;
    rand_id TEXT := gen_random_uuid()::text;
BEGIN
    SELECT node_id, resources_cache.farm_id, dedicated_node, shared, rentable
    INTO rand_node_id, rand_farm_id, old_dedicated_node, old_shared, old_rentable
    FROM resources_cache
    LEFT JOIN farm ON farm.farm_id = resources_cache.farm_id
    WHERE dedicated_farm = FALSE
    ORDER BY random()
    LIMIT 1;

    UPDATE farm
    SET dedicated_farm = TRUE
    WHERE farm_id = rand_farm_id;

    ASSERT (
        (
            SELECT (dedicated_node, shared, rentable)
            FROM resources_cache
            WHERE node_id = rand_node_id
        ) = (TRUE, FALSE, TRUE)
    );

    UPDATE farm
    SET dedicated_farm = false
    WHERE farm_id = rand_farm_id;

    ASSERT (
        (
            SELECT (dedicated_node, shared, rentable)
            FROM resources_cache
            WHERE node_id = rand_node_id
        ) = (old_dedicated_node, old_shared, old_rentable)
    );
END $$;

DO $$
DECLARE 
    rand_node_id NUMERIC;
    old_dedicated_node BOOLEAN;
    old_shared BOOLEAN;
    old_rentable BOOLEAN;
    rand_id TEXT := gen_random_uuid()::text;
    rand_twin_id NUMERIC := 29;
BEGIN
    SELECT node_id, dedicated_node, shared, rentable
    INTO rand_node_id, old_dedicated_node, old_shared, old_rentable
    FROM resources_cache
    WHERE node_contracts_count=0 and rented=false
    ORDER BY random()
    LIMIT 1;
    
    insert into 
    node_contract (
      id, 
      grid_version, 
      contract_id, 
      twin_id, 
      node_id, 
      deployment_data, 
      deployment_hash, 
      number_of_public_i_ps, 
      created_at, 
      state
    )
  values
    (
      rand_id, 
      3, 
      999999, 
      99999, 
      rand_node_id, 
      'deployment_data', 
      'deployment_hash', 
      0, 
      125457863, 
      'Created'
    );

    -- assert the result
    ASSERT (
        SELECT (dedicated_node, shared, rentable) 
        FROM resources_cache 
        WHERE node_id = rand_node_id
    ) = (FALSE, TRUE, FALSE);

    UPDATE node_contract
        SET state = 'Deleted'
        WHERE id = rand_id;

    ASSERT (
        SELECT (dedicated_node, shared, rentable) 
        FROM resources_cache 
        WHERE node_id = rand_node_id
    ) = (old_dedicated_node, old_shared, old_rentable);
    
END $$;
-- ROLLBACK;


-- BEGIN;

-- DO $$
-- DECLARE 
--     rand_node_id NUMERIC;
--     old_dedicated_node BOOLEAN;
--     old_shared BOOLEAN;
--     old_rentable BOOLEAN;
--     rand_id TEXT := gen_random_uuid()::text;
--     rand_twin_id NUMERIC := 29;
-- BEGIN
--     INTO rand_node_id, old_dedicated_node, old_shared, old_rentable
--     SELECT node_id, dedicated_node, shared, rentable
--     FROM resources_cache
--     WHERE node_contracts_count=0 and rented=true
--     ORDER BY random()
--     LIMIT 1;
    
--     insert into 
--     node_contract (
--       id, 
--       grid_version, 
--       contract_id, 
--       twin_id, 
--       node_id, 
--       deployment_data, 
--       deployment_hash, 
--       number_of_public_i_ps, 
--       created_at, 
--       state
--     )
--   values
--     (
--       rand_id, 
--       3, 
--       999999, 
--       99999, 
--       rand_node_id, 
--       'deployment_data', 
--       'deployment_hash', 
--       0, 
--       125457863, 
--       'Created'
--     );

--     -- assert the result
--     -- ASSERT (
--     --     SELECT (dedicated_node, shared, rentable) 
--     --     FROM resources_cache 
--     --     WHERE node_id = rand_node_id
--     -- ) = (old_dedicated_node, old_shared, old_rentable);

--     UPDATE node_contract
--         SET state = 'Deleted'
--         WHERE id = rand_id;

--     ASSERT (
--         SELECT (rentable) 
--         FROM resources_cache 
--         WHERE node_id = rand_node_id
--     ) = (old_rentable);
    
-- END $$;
ROLLBACK;

select * from resources_cache where node_id = 4534