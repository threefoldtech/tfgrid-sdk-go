-- ============================================================================
-- TEST: reflect_farm_changes Trigger
-- ============================================================================
-- Tests for farm INSERT and DELETE operations

BEGIN;

-- Load pgTAP
SELECT plan(4);

-- Test 1: INSERT farm should create entry in farmx
SELECT create_test_farm(1002, 2002, 'Test Farm 2', 1);
SELECT create_test_twin(2002);

SELECT pg_sleep(0.1);

SELECT ok(
    EXISTS (
        SELECT 1 FROM farmx WHERE farm_id = 1002
    ),
    'INSERT farm should create entry in farmx'
);

SELECT is(
    (SELECT free_ips FROM farmx WHERE farm_id = 1002),
    0,
    'INSERT farm should initialize free_ips to 0'
);

SELECT is(
    (SELECT total_ips FROM farmx WHERE farm_id = 1002),
    0,
    'INSERT farm should initialize total_ips to 0'
);

SELECT is(
    (SELECT ips FROM farmx WHERE farm_id = 1002),
    '[]'::jsonb,
    'INSERT farm should initialize ips to empty array'
);

-- Test 2: DELETE farm should remove entry from farmx
DELETE FROM farm WHERE id = 'farm-1002';

SELECT pg_sleep(0.1);

SELECT ok(
    NOT EXISTS (
        SELECT 1 FROM farmx WHERE farm_id = 1002
    ),
    'DELETE farm should remove entry from farmx'
);

-- Cleanup
SELECT cleanup_test_data();

SELECT * FROM finish();

ROLLBACK;

