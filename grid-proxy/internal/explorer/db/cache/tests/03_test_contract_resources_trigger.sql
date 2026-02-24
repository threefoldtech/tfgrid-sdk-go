-- ============================================================================
-- TEST: reflect_contract_resources_changes Trigger
-- ============================================================================
-- Tests for contract_resources INSERT, UPDATE, and DELETE operations

BEGIN;

-- Load pgTAP
SELECT plan(15);

-- Setup: Create test data
SELECT create_test_farm(1001, 2001, 'Test Farm 1', 1);
SELECT create_test_pricing_policy(1, 1000000, 1000000);
SELECT create_test_twin(2001);
SELECT create_test_node(1001, 1001, 2001);
SELECT create_test_node_resources_total('node-1001', 1000000000000, 100000000000, 100000000000, 16);
SELECT pg_sleep(0.1);

-- Get initial cache values
SELECT 
    (SELECT free_hru FROM nodex WHERE node_id = 1001) as initial_free_hru,
    (SELECT free_mru FROM nodex WHERE node_id = 1001) as initial_free_mru,
    (SELECT free_sru FROM nodex WHERE node_id = 1001) as initial_free_sru,
    (SELECT used_hru FROM nodex WHERE node_id = 1001) as initial_used_hru,
    (SELECT used_mru FROM nodex WHERE node_id = 1001) as initial_used_mru,
    (SELECT used_sru FROM nodex WHERE node_id = 1001) as initial_used_sru,
    (SELECT used_cru FROM nodex WHERE node_id = 1001) as initial_used_cru
INTO TEMP initial_cache;

-- Create node_contract and contract_resources
SELECT create_test_node_contract('1001', 1001, 2001, 'Created', 'cr-1001');
SELECT create_test_contract_resources('1001', 1000000000, 2000000000, 3000000000, 2);

SELECT pg_sleep(0.1);

-- Test 1: INSERT contract_resources should increment used and decrement free
SELECT is(
    (SELECT used_hru FROM nodex WHERE node_id = 1001),
    (SELECT initial_used_hru FROM initial_cache) + 1000000000,
    'INSERT contract_resources should increment used_hru'
);

SELECT is(
    (SELECT used_mru FROM nodex WHERE node_id = 1001),
    (SELECT initial_used_mru FROM initial_cache) + 2000000000,
    'INSERT contract_resources should increment used_mru'
);

SELECT is(
    (SELECT free_hru FROM nodex WHERE node_id = 1001),
    (SELECT initial_free_hru FROM initial_cache) - 1000000000,
    'INSERT contract_resources should decrement free_hru'
);

SELECT is(
    (SELECT free_mru FROM nodex WHERE node_id = 1001),
    (SELECT initial_free_mru FROM initial_cache) - 2000000000,
    'INSERT contract_resources should decrement free_mru'
);

-- Test 2: UPDATE contract_resources should adjust differences
UPDATE contract_resources 
SET hru = 2000000000, mru = 4000000000, sru = 5000000000, cru = 4
WHERE id = 'cr-1001';

SELECT pg_sleep(0.1);

SELECT is(
    (SELECT used_hru FROM nodex WHERE node_id = 1001),
    (SELECT initial_used_hru FROM initial_cache) + 2000000000,
    'UPDATE contract_resources should update used_hru to new value'
);

SELECT is(
    (SELECT used_mru FROM nodex WHERE node_id = 1001),
    (SELECT initial_used_mru FROM initial_cache) + 4000000000,
    'UPDATE contract_resources should update used_mru to new value'
);

SELECT is(
    (SELECT free_hru FROM nodex WHERE node_id = 1001),
    (SELECT initial_free_hru FROM initial_cache) - 2000000000,
    'UPDATE contract_resources should adjust free_hru'
);

-- Test 3: DELETE contract_resources should decrement used and increment free
DELETE FROM contract_resources WHERE id = 'cr-1001';

SELECT pg_sleep(0.1);

SELECT is(
    (SELECT used_hru FROM nodex WHERE node_id = 1001),
    (SELECT initial_used_hru FROM initial_cache),
    'DELETE contract_resources should decrement used_hru back to initial'
);

SELECT is(
    (SELECT used_mru FROM nodex WHERE node_id = 1001),
    (SELECT initial_used_mru FROM initial_cache),
    'DELETE contract_resources should decrement used_mru back to initial'
);

SELECT is(
    (SELECT free_hru FROM nodex WHERE node_id = 1001),
    (SELECT initial_free_hru FROM initial_cache),
    'DELETE contract_resources should increment free_hru back to initial'
);

-- Test 4: Contract with 'GracePeriod' state should be counted
SELECT create_test_node_contract('1002', 1001, 2001, 'GracePeriod', 'cr-1002');
SELECT create_test_contract_resources('1002', 500000000, 500000000, 500000000, 1);

SELECT pg_sleep(0.1);

SELECT ok(
    (SELECT used_hru FROM nodex WHERE node_id = 1001) > (SELECT initial_used_hru FROM initial_cache),
    'Contract in GracePeriod state should be counted in used resources'
);

-- Test 5: Contract with 'Deleted' state should NOT be counted
DELETE FROM contract_resources WHERE id = 'cr-1002';
UPDATE node_contract SET state = 'Deleted' WHERE id = 'nc-1002';
SELECT create_test_node_contract('1003', 1001, 2001, 'Deleted', 'cr-1003');
SELECT create_test_contract_resources('1003', 1000000000, 1000000000, 1000000000, 1);

SELECT pg_sleep(0.1);

-- Should not affect cache because state is 'Deleted'
SELECT is(
    (SELECT used_hru FROM nodex WHERE node_id = 1001),
    (SELECT initial_used_hru FROM initial_cache),
    'Contract in Deleted state should NOT affect cache'
);

-- Test 6: Multiple contracts on same node
SELECT create_test_node_contract('1004', 1001, 2001, 'Created', 'cr-1004');
SELECT create_test_contract_resources('1004', 1000000000, 1000000000, 1000000000, 1);
SELECT create_test_node_contract('1005', 1001, 2001, 'Created', 'cr-1005');
SELECT create_test_contract_resources('1005', 1000000000, 1000000000, 1000000000, 1);

SELECT pg_sleep(0.1);

SELECT is(
    (SELECT used_hru FROM nodex WHERE node_id = 1001),
    (SELECT initial_used_hru FROM initial_cache) + 2000000000,
    'Multiple contracts should sum up used resources'
);

-- Cleanup
SELECT cleanup_test_data();

SELECT * FROM finish();

ROLLBACK;

