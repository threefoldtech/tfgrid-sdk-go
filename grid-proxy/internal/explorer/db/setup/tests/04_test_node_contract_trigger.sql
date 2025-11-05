-- ============================================================================
-- TEST: reflect_node_contract_changes Trigger
-- ============================================================================
-- Tests for node_contract INSERT and UPDATE state to 'Deleted'

BEGIN;

-- Load pgTAP
SELECT plan(8);

-- Setup: Create test data
SELECT create_test_farm(1001, 2001, 'Test Farm 1', 1);
SELECT create_test_pricing_policy(1, 1000000, 1000000);
SELECT create_test_twin(2001);
SELECT create_test_node(1001, 1001, 2001);
SELECT create_test_node_resources_total('node-1001', 1000000000000, 100000000000, 100000000000, 16);
SELECT pg_sleep(0.1);

-- Test 1: INSERT node_contract with 'Created' state should increment node_contracts_count
SELECT create_test_node_contract('1001', 1001, 2001, 'Created', 'cr-1001');
SELECT create_test_contract_resources('1001', 1000000000, 2000000000, 3000000000, 2);

SELECT pg_sleep(0.1);

SELECT is(
    (SELECT node_contracts_count FROM resources_cache WHERE node_id = 1001),
    1,
    'INSERT node_contract with Created state should set node_contracts_count to 1'
);

-- Test 2: INSERT another contract should increment count
SELECT create_test_node_contract('1002', 1001, 2001, 'Created', 'cr-1002');
SELECT create_test_contract_resources('1002', 500000000, 500000000, 500000000, 1);

SELECT pg_sleep(0.1);

SELECT is(
    (SELECT node_contracts_count FROM resources_cache WHERE node_id = 1001),
    2,
    'INSERT multiple contracts should increment node_contracts_count'
);

-- Test 3: INSERT contract with 'GracePeriod' state should be counted
SELECT create_test_node_contract('1003', 1001, 2001, 'GracePeriod', 'cr-1003');
SELECT create_test_contract_resources('1003', 1000000000, 1000000000, 1000000000, 1);

SELECT pg_sleep(0.1);

SELECT is(
    (SELECT node_contracts_count FROM resources_cache WHERE node_id = 1001),
    3,
    'INSERT contract with GracePeriod state should be counted'
);

-- Test 4: UPDATE contract state to 'Deleted' should decrement count and release resources
-- Get initial values before deletion
SELECT 
    (SELECT used_hru FROM resources_cache WHERE node_id = 1001) as before_used_hru,
    (SELECT free_hru FROM resources_cache WHERE node_id = 1001) as before_free_hru,
    (SELECT node_contracts_count FROM resources_cache WHERE node_id = 1001) as before_count
INTO TEMP before_delete;

UPDATE node_contract SET state = 'Deleted' WHERE id = 'nc-1001';

SELECT pg_sleep(0.1);

SELECT is(
    (SELECT node_contracts_count FROM resources_cache WHERE node_id = 1001),
    (SELECT before_count FROM before_delete) - 1,
    'UPDATE contract to Deleted state should decrement node_contracts_count'
);

-- Test 5: UPDATE to 'Deleted' should release contract resources
SELECT is(
    (SELECT used_hru FROM resources_cache WHERE node_id = 1001),
    (SELECT before_used_hru FROM before_delete) - 1000000000,
    'UPDATE to Deleted should release contract resources (decrement used_hru)'
);

SELECT is(
    (SELECT free_hru FROM resources_cache WHERE node_id = 1001),
    (SELECT before_free_hru FROM before_delete) + 1000000000,
    'UPDATE to Deleted should release contract resources (increment free_hru)'
);

-- Test 6: UPDATE multiple contracts to 'Deleted'
UPDATE node_contract SET state = 'Deleted' WHERE id = 'nc-1002';

SELECT pg_sleep(0.1);

SELECT is(
    (SELECT node_contracts_count FROM resources_cache WHERE node_id = 1001),
    1,
    'Multiple contract deletions should update count correctly'
);

-- Test 7: INSERT contract with 'Deleted' state should NOT increment count
SELECT create_test_node_contract('1004', 1001, 2001, 'Deleted', 'cr-1004');
SELECT create_test_contract_resources('1004', 1000000000, 1000000000, 1000000000, 1);

SELECT pg_sleep(0.1);

SELECT is(
    (SELECT node_contracts_count FROM resources_cache WHERE node_id = 1001),
    1,
    'INSERT contract with Deleted state should NOT increment count'
);

-- Test 8: UPDATE to non-Deleted state should not trigger resource release
-- (Only UPDATE of state column triggers, and only to 'Deleted')
SELECT create_test_node_contract('1005', 1001, 2001, 'Created', 'cr-1005');
SELECT create_test_contract_resources('1005', 1000000000, 1000000000, 1000000000, 1);
SELECT pg_sleep(0.1);

SELECT 
    (SELECT node_contracts_count FROM resources_cache WHERE node_id = 1001) as count_before
INTO TEMP count_before_update;

UPDATE node_contract SET deployment_data = '{}' WHERE id = 'nc-1005';

SELECT pg_sleep(0.1);

SELECT is(
    (SELECT node_contracts_count FROM resources_cache WHERE node_id = 1001),
    (SELECT count_before FROM count_before_update),
    'UPDATE non-state column should not trigger count change'
);

-- Cleanup
SELECT cleanup_test_data();

SELECT * FROM finish();

ROLLBACK;

