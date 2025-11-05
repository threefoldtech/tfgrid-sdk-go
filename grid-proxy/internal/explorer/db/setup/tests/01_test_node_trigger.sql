-- ============================================================================
-- TEST: reflect_node_changes Trigger
-- ============================================================================
-- Tests for node INSERT and DELETE operations on resources_cache

BEGIN;

-- Load pgTAP
SELECT plan(6);

-- Setup: Create test farm and pricing policy
SELECT create_test_farm(1001, 2001, 'Test Farm 1', 1);
SELECT create_test_pricing_policy(1, 1000000, 1000000);
SELECT create_test_twin(2001);

-- Test 1: INSERT node should populate resources_cache
SELECT create_test_node(1001, 1001, 2001, 'US', 'Certified', 0);
SELECT create_test_node_resources_total('node-1001', 1000000000000, 100000000000, 100000000000, 16);

-- Wait for trigger to fire
SELECT pg_sleep(0.1);

SELECT ok(
    EXISTS (
        SELECT 1 FROM resources_cache WHERE node_id = 1001
    ),
    'Node INSERT should create entry in resources_cache'
);

-- Test 2: Verify resources_cache values match expected from view
SELECT is(
    (SELECT total_hru FROM resources_cache WHERE node_id = 1001),
    1000000000000,
    'resources_cache total_hru should match node_resources_total'
);

SELECT is(
    (SELECT total_mru FROM resources_cache WHERE node_id = 1001),
    100000000000,
    'resources_cache total_mru should match node_resources_total'
);

-- Test 3: DELETE node should remove entry from resources_cache
DELETE FROM node_resources_total WHERE node_id = 'node-1001';
DELETE FROM node WHERE id = 'node-1001';

-- Wait for trigger to fire
SELECT pg_sleep(0.1);

SELECT ok(
    NOT EXISTS (
        SELECT 1 FROM resources_cache WHERE node_id = 1001
    ),
    'Node DELETE should remove entry from resources_cache'
);

-- Test 4: INSERT node with all fields populated
SELECT create_test_node(1002, 1001, 2002, 'DE', 'Certified', 100);
SELECT create_test_twin(2002);
SELECT create_test_node_resources_total('node-1002', 2000000000000, 200000000000, 200000000000, 32);

SELECT pg_sleep(0.1);

SELECT ok(
    EXISTS (
        SELECT 1 FROM resources_cache 
        WHERE node_id = 1002 
        AND farm_id = 1001
        AND country = 'DE'
        AND certified = true
        AND extra_fee = 100
    ),
    'Node INSERT should populate all fields in resources_cache'
);

-- Cleanup
SELECT cleanup_test_data();

SELECT * FROM finish();

ROLLBACK;

