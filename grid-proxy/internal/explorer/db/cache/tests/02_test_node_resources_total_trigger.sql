-- ============================================================================
-- TEST: reflect_total_resources_changes Trigger
-- ============================================================================
-- Tests for node_resources_total INSERT and UPDATE operations

BEGIN;

-- Load pgTAP
SELECT plan(12);

-- Setup: Create test data
SELECT create_test_farm(1001, 2001, 'Test Farm 1', 1);
SELECT create_test_pricing_policy(1, 1000000, 1000000);
SELECT create_test_twin(2001);
SELECT create_test_node(1001, 1001, 2001);
SELECT create_test_node_resources_total('node-1001', 1000000000000, 100000000000, 100000000000, 16);
SELECT pg_sleep(0.1);

-- Get initial cache values
SELECT 
    (SELECT total_hru FROM nodex WHERE node_id = 1001) as initial_hru,
    (SELECT total_mru FROM nodex WHERE node_id = 1001) as initial_mru,
    (SELECT total_sru FROM nodex WHERE node_id = 1001) as initial_sru,
    (SELECT total_cru FROM nodex WHERE node_id = 1001) as initial_cru,
    (SELECT free_hru FROM nodex WHERE node_id = 1001) as initial_free_hru,
    (SELECT free_mru FROM nodex WHERE node_id = 1001) as initial_free_mru,
    (SELECT free_sru FROM nodex WHERE node_id = 1001) as initial_free_sru,
    (SELECT used_mru FROM nodex WHERE node_id = 1001) as initial_used_mru
INTO TEMP initial_cache;

-- Test 1: UPDATE total_hru should update cache
UPDATE node_resources_total 
SET hru = 2000000000000 
WHERE node_id = 'node-1001';

SELECT pg_sleep(0.1);

SELECT is(
    (SELECT total_hru FROM nodex WHERE node_id = 1001),
    2000000000000,
    'UPDATE total_hru should update nodex total_hru'
);

-- Calculate expected free_hru change: NEW.hru - OLD.hru
SELECT is(
    (SELECT free_hru FROM nodex WHERE node_id = 1001),
    (SELECT initial_free_hru FROM initial_cache) + 1000000000000,
    'UPDATE total_hru should increment free_hru by the difference'
);

-- Test 2: UPDATE total_mru should update cache and adjust reserved amount
UPDATE node_resources_total 
SET mru = 200000000000 
WHERE node_id = 'node-1001';

SELECT pg_sleep(0.1);

SELECT is(
    (SELECT total_mru FROM nodex WHERE node_id = 1001),
    200000000000,
    'UPDATE total_mru should update nodex total_mru'
);

-- MRU reserved calculation: GREATEST(MRU/10, 2147483648)
-- Old reserved: GREATEST(100000000000/10, 2147483648) = GREATEST(10000000000, 2147483648) = 10000000000
-- New reserved: GREATEST(200000000000/10, 2147483648) = GREATEST(20000000000, 2147483648) = 20000000000
-- free_mru change = (old_reserved - new_reserved) + (new_mru - old_mru)
--                 = (10000000000 - 20000000000) + (200000000000 - 100000000000)
--                 = -10000000000 + 100000000000 = 90000000000
-- But we need to check the actual calculation...

-- Test 3: UPDATE total_sru should update cache
UPDATE node_resources_total 
SET sru = 200000000000 
WHERE node_id = 'node-1001';

SELECT pg_sleep(0.1);

SELECT is(
    (SELECT total_sru FROM nodex WHERE node_id = 1001),
    200000000000,
    'UPDATE total_sru should update nodex total_sru'
);

-- Test 4: UPDATE total_cru should update cache
UPDATE node_resources_total 
SET cru = 32 
WHERE node_id = 'node-1001';

SELECT pg_sleep(0.1);

SELECT is(
    (SELECT total_cru FROM nodex WHERE node_id = 1001),
    32,
    'UPDATE total_cru should update nodex total_cru'
);

-- Test 5: INSERT new node_resources_total should update cache
SELECT create_test_node(1002, 1001, 2002);
SELECT create_test_twin(2002);
SELECT create_test_node_resources_total('node-1002', 500000000000, 50000000000, 50000000000, 8);
SELECT pg_sleep(0.1);

SELECT ok(
    EXISTS (
        SELECT 1 FROM nodex 
        WHERE node_id = 1002 
        AND total_hru = 500000000000
        AND total_mru = 50000000000
        AND total_sru = 50000000000
        AND total_cru = 8
    ),
    'INSERT node_resources_total should update nodex'
);

-- Test 6: UPDATE MRU below reserved minimum threshold
-- MRU reserved minimum is 2147483648 (2GB)
-- If MRU is 10000000000 (10GB), reserved = GREATEST(10GB/10, 2GB) = GREATEST(1GB, 2GB) = 2GB
-- If MRU is 100000000000 (100GB), reserved = GREATEST(100GB/10, 2GB) = GREATEST(10GB, 2GB) = 10GB
UPDATE node_resources_total 
SET mru = 10000000000  -- 10GB, below threshold so reserved should be 2GB
WHERE node_id = 'node-1001';

SELECT pg_sleep(0.1);

-- Verify used_mru includes reserved amount (at least 2GB)
SELECT ok(
    (SELECT used_mru FROM nodex WHERE node_id = 1001) >= 2147483648,
    'used_mru should include at least minimum reserved amount (2GB)'
);

-- Test 7: Multiple simultaneous updates
UPDATE node_resources_total 
SET hru = 3000000000000, mru = 300000000000, sru = 300000000000, cru = 64
WHERE node_id = 'node-1001';

SELECT pg_sleep(0.1);

SELECT is(
    (SELECT total_hru FROM nodex WHERE node_id = 1001),
    3000000000000,
    'Multiple UPDATE should update all total fields'
);

SELECT is(
    (SELECT total_cru FROM nodex WHERE node_id = 1001),
    64,
    'Multiple UPDATE should update total_cru'
);

-- Cleanup
SELECT cleanup_test_data();

SELECT * FROM finish();

ROLLBACK;

