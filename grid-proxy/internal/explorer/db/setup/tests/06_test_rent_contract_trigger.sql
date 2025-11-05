-- ============================================================================
-- TEST: reflect_rent_contract_changes Trigger
-- ============================================================================
-- Tests for rent_contract INSERT and UPDATE state to 'Deleted'

BEGIN;

-- Load pgTAP
SELECT plan(6);

-- Setup: Create test data
SELECT create_test_farm(1001, 2001, 'Test Farm 1', 1);
SELECT create_test_pricing_policy(1, 1000000, 1000000);
SELECT create_test_twin(2001);
SELECT create_test_twin(3001); -- Renter twin
SELECT create_test_node(1001, 1001, 2001);
SELECT create_test_node_resources_total('node-1001', 1000000000000, 100000000000, 100000000000, 16);
SELECT pg_sleep(0.1);

-- Test 1: INSERT rent_contract should set renter and rent_contract_id
INSERT INTO rent_contract (
    id, grid_version, contract_id, twin_id, node_id, created_at, state
) VALUES (
    'rc-1001', 3, 1001, 3001, 1001, EXTRACT(EPOCH FROM NOW())::NUMERIC, 'Created'
);

SELECT pg_sleep(0.1);

SELECT is(
    (SELECT renter FROM resources_cache WHERE node_id = 1001),
    3001,
    'INSERT rent_contract should set renter field'
);

SELECT is(
    (SELECT rent_contract_id FROM resources_cache WHERE node_id = 1001),
    1001,
    'INSERT rent_contract should set rent_contract_id field'
);

-- Test 2: UPDATE rent_contract state to 'Deleted' should clear renter and rent_contract_id
UPDATE rent_contract SET state = 'Deleted' WHERE id = 'rc-1001';

SELECT pg_sleep(0.1);

SELECT is(
    (SELECT renter FROM resources_cache WHERE node_id = 1001),
    NULL,
    'UPDATE rent_contract to Deleted should clear renter field'
);

SELECT is(
    (SELECT rent_contract_id FROM resources_cache WHERE node_id = 1001),
    NULL,
    'UPDATE rent_contract to Deleted should clear rent_contract_id field'
);

-- Test 3: INSERT new rent_contract should update fields again
INSERT INTO rent_contract (
    id, grid_version, contract_id, twin_id, node_id, created_at, state
) VALUES (
    'rc-1002', 3, 1002, 3002, 1001, EXTRACT(EPOCH FROM NOW())::NUMERIC, 'Created'
);

SELECT create_test_twin(3002);

SELECT pg_sleep(0.1);

SELECT is(
    (SELECT renter FROM resources_cache WHERE node_id = 1001),
    3002,
    'INSERT new rent_contract should update renter field'
);

SELECT is(
    (SELECT rent_contract_id FROM resources_cache WHERE node_id = 1001),
    1002,
    'INSERT new rent_contract should update rent_contract_id field'
);

-- Test 4: UPDATE to non-Deleted state should not trigger
SELECT 
    (SELECT renter FROM resources_cache WHERE node_id = 1001) as renter_before
INTO TEMP renter_before;

UPDATE rent_contract SET solution_provider_id = 1 WHERE id = 'rc-1002';

SELECT pg_sleep(0.1);

SELECT is(
    (SELECT renter FROM resources_cache WHERE node_id = 1001),
    (SELECT renter_before FROM renter_before),
    'UPDATE non-state column should not trigger renter change'
);

-- Cleanup
SELECT cleanup_test_data();

SELECT * FROM finish();

ROLLBACK;

