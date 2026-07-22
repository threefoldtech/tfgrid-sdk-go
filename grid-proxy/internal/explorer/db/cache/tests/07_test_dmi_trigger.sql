-- ============================================================================
-- TEST: reflect_dmi_changes Trigger
-- ============================================================================
-- Tests for dmi INSERT and UPDATE operations

BEGIN;

-- Load pgTAP
SELECT plan(6);

-- Setup: Create test data
SELECT create_test_farm(1001, 2001, 'Test Farm 1', 1);
SELECT create_test_pricing_policy(1, 1000000, 1000000);
SELECT create_test_twin(2001);
SELECT create_test_node(1001, 1001, 2001);
SELECT create_test_node_resources_total('node-1001', 1000000000000, 100000000000, 100000000000, 16);
SELECT pg_sleep(0.1);

-- Test 1: INSERT dmi should update cache fields
INSERT INTO dmi (node_twin_id, bios, baseboard, processor, memory, updated_at)
VALUES (
    2001,
    '{"vendor": "Test", "version": "1.0"}'::jsonb,
    '{"manufacturer": "TestMB"}'::jsonb,
    '[{"model": "TestCPU"}]'::jsonb,
    '[{"size": 16000000000}]'::jsonb,
    EXTRACT(EPOCH FROM NOW())::BIGINT
);

SELECT pg_sleep(0.1);

SELECT ok(
    (SELECT bios FROM nodex WHERE node_id = 1001) IS NOT NULL,
    'INSERT dmi should populate bios field'
);

SELECT ok(
    (SELECT baseboard FROM nodex WHERE node_id = 1001) IS NOT NULL,
    'INSERT dmi should populate baseboard field'
);

SELECT ok(
    (SELECT processor FROM nodex WHERE node_id = 1001) IS NOT NULL,
    'INSERT dmi should populate processor field'
);

SELECT ok(
    (SELECT memory FROM nodex WHERE node_id = 1001) IS NOT NULL,
    'INSERT dmi should populate memory field'
);

-- Test 2: Verify JSON structure
SELECT is(
    (SELECT bios->>'vendor' FROM nodex WHERE node_id = 1001),
    'Test',
    'bios JSON should contain correct vendor'
);

SELECT is(
    (SELECT baseboard->>'manufacturer' FROM nodex WHERE node_id = 1001),
    'TestMB',
    'baseboard JSON should contain correct manufacturer'
);

-- Test 3: UPDATE dmi should update cache
UPDATE dmi SET 
    bios = '{"vendor": "Updated", "version": "2.0"}'::jsonb,
    baseboard = '{"manufacturer": "UpdatedMB"}'::jsonb
WHERE node_twin_id = 2001;

SELECT pg_sleep(0.1);

SELECT is(
    (SELECT bios->>'vendor' FROM nodex WHERE node_id = 1001),
    'Updated',
    'UPDATE dmi should update bios field'
);

SELECT is(
    (SELECT baseboard->>'manufacturer' FROM nodex WHERE node_id = 1001),
    'UpdatedMB',
    'UPDATE dmi should update baseboard field'
);

-- Cleanup
SELECT cleanup_test_data();

SELECT * FROM finish();

ROLLBACK;

