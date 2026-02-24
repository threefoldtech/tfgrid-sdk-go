-- ============================================================================
-- TEST: Cache Refresher Functions
-- ============================================================================
-- Tests for cache refresh functions

BEGIN;

-- Load pgTAP
SELECT plan(10);

-- Setup: Create test data
SELECT create_test_farm(1001, 2001, 'Test Farm 1', 1);
SELECT create_test_pricing_policy(1, 1000000, 1000000);
SELECT create_test_twin(2001);
SELECT create_test_node(1001, 1001, 2001);
SELECT create_test_node_resources_total('node-1001', 1000000000000, 100000000000, 100000000000, 16);
SELECT create_test_node_contract('1001', 1001, 2001, 'Created', 'cr-1001');
SELECT create_test_contract_resources('1001', 1000000000, 2000000000, 3000000000, 2);
SELECT pg_sleep(0.1);

-- Get initial cache values from view
SELECT
    total_hru, total_mru, total_sru, total_cru,
    free_hru, free_mru, free_sru,
    used_hru, used_mru, used_sru, used_cru,
    node_contracts_count
FROM nodex_view WHERE node_id = 1001
INTO TEMP expected_cache;

-- Test 1: refresh_nodex_node should refresh single node
-- First, corrupt the cache
UPDATE nodex SET total_hru = 999999999 WHERE node_id = 1001;

SELECT refresh_nodex_node(1001);

SELECT is(
    (SELECT total_hru FROM nodex WHERE node_id = 1001),
    (SELECT total_hru FROM expected_cache),
    'refresh_nodex_node should restore correct total_hru'
);

SELECT is(
    (SELECT free_hru FROM nodex WHERE node_id = 1001),
    (SELECT free_hru FROM expected_cache),
    'refresh_nodex_node should restore correct free_hru'
);

SELECT is(
    (SELECT used_hru FROM nodex WHERE node_id = 1001),
    (SELECT used_hru FROM expected_cache),
    'refresh_nodex_node should restore correct used_hru'
);

-- Test 2: refresh_nodex should refresh all nodes
-- Create another node
SELECT create_test_node(1002, 1001, 2002);
SELECT create_test_twin(2002);
SELECT create_test_node_resources_total('node-1002', 2000000000000, 200000000000, 200000000000, 32);
SELECT pg_sleep(0.1);

-- Corrupt both caches
UPDATE nodex SET total_hru = 111111111 WHERE node_id IN (1001, 1002);

SELECT refresh_nodex();

SELECT ok(
    (SELECT COUNT(*) FROM nodex) >= 2,
    'refresh_nodex should refresh all nodes'
);

SELECT ok(
    EXISTS (
        SELECT 1 FROM nodex
        WHERE node_id = 1001 AND total_hru = (SELECT total_hru FROM nodex_view WHERE node_id = 1001)
    ),
    'refresh_nodex should refresh node 1001 correctly'
);

SELECT ok(
    EXISTS (
        SELECT 1 FROM nodex
        WHERE node_id = 1002 AND total_hru = (SELECT total_hru FROM nodex_view WHERE node_id = 1002)
    ),
    'refresh_nodex should refresh node 1002 correctly'
);

-- Test 3: refresh_farmx_farm should refresh single farm
SELECT create_test_public_ip('ip-1', 'farm-1001', '1.2.3.4', '1.2.3.1', 0);
SELECT pg_sleep(0.1);

-- Get expected values
SELECT
    COALESCE(COUNT(*), 0) as expected_total,
    COALESCE(COUNT(CASE WHEN contract_id = 0 THEN 1 END), 0) as expected_free
FROM public_ip WHERE farm_id = 'farm-1001'
INTO TEMP expected_ips;

-- Corrupt cache
UPDATE farmx SET free_ips = 999, total_ips = 999 WHERE farm_id = 1001;

SELECT refresh_farmx_farm(1001);

SELECT is(
    (SELECT total_ips FROM farmx WHERE farm_id = 1001),
    (SELECT expected_total FROM expected_ips),
    'refresh_farmx_farm should restore correct total_ips'
);

SELECT is(
    (SELECT free_ips FROM farmx WHERE farm_id = 1001),
    (SELECT expected_free FROM expected_ips),
    'refresh_farmx_farm should restore correct free_ips'
);

-- Test 4: refresh_farmx should refresh all farms
SELECT create_test_farm(1003, 2003, 'Test Farm 3', 1);
SELECT create_test_twin(2003);
SELECT create_test_public_ip('ip-2', 'farm-1003', '2.3.4.5', '2.3.4.1', 0);
SELECT pg_sleep(0.1);

-- Corrupt cache
UPDATE farmx SET free_ips = 888 WHERE farm_id IN (1001, 1003);

SELECT refresh_farmx();

SELECT ok(
    (SELECT COUNT(*) FROM farmx) >= 2,
    'refresh_farmx should refresh all farms'
);

SELECT ok(
    EXISTS (
        SELECT 1 FROM farmx
        WHERE farm_id = 1001 AND free_ips = (SELECT COUNT(*) FROM public_ip WHERE farm_id = 'farm-1001' AND contract_id = 0)
    ),
    'refresh_farmx should refresh farm 1001 correctly'
);

-- Test 5: refresh_all should refresh both cache tables
UPDATE nodex SET total_hru = 777777777 WHERE node_id = 1001;
UPDATE farmx SET free_ips = 777 WHERE farm_id = 1001;

SELECT refresh_all();

SELECT ok(
    (SELECT total_hru FROM nodex WHERE node_id = 1001) != 777777777,
    'refresh_all should refresh nodex'
);

SELECT ok(
    (SELECT free_ips FROM farmx WHERE farm_id = 1001) != 777,
    'refresh_all should refresh farmx'
);

-- Cleanup
SELECT cleanup_test_data();

SELECT * FROM finish();

ROLLBACK;
