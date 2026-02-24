-- ============================================================================
-- TEST: reflect_node_gpu_count_change Trigger
-- ============================================================================
-- Tests for node_gpu INSERT, DELETE, and UPDATE operations

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

-- Test 1: INSERT node_gpu should update gpu count and gpus array
INSERT INTO node_gpu (id, node_twin_id, vendor, device, vram, contract)
VALUES ('gpu-1', 2001, 'NVIDIA', 'RTX 3090', 24000000000, 0);

SELECT pg_sleep(0.1);

SELECT is(
    (SELECT node_gpu_count FROM nodex WHERE node_id = 1001),
    1,
    'INSERT node_gpu should set node_gpu_count to 1'
);

SELECT is(
    (SELECT free_gpu_count FROM nodex WHERE node_id = 1001),
    1,
    'INSERT free gpu should set free_gpu_count to 1'
);

SELECT ok(
    (SELECT gpus FROM nodex WHERE node_id = 1001) IS NOT NULL,
    'INSERT node_gpu should populate gpus JSONB array'
);

SELECT ok(
    jsonb_array_length((SELECT gpus FROM nodex WHERE node_id = 1001)) = 1,
    'INSERT node_gpu should create gpus array with 1 element'
);

-- Test 2: INSERT multiple GPUs should aggregate correctly
INSERT INTO node_gpu (id, node_twin_id, vendor, device, vram, contract)
VALUES ('gpu-2', 2001, 'NVIDIA', 'RTX 4090', 24000000000, 0),
       ('gpu-3', 2001, 'AMD', 'RX 7900', 20000000000, 0);

SELECT pg_sleep(0.1);

SELECT is(
    (SELECT node_gpu_count FROM nodex WHERE node_id = 1001),
    3,
    'INSERT multiple GPUs should increment node_gpu_count'
);

SELECT is(
    (SELECT free_gpu_count FROM nodex WHERE node_id = 1001),
    3,
    'INSERT multiple free GPUs should set free_gpu_count to 3'
);

SELECT ok(
    jsonb_array_length((SELECT gpus FROM nodex WHERE node_id = 1001)) = 3,
    'INSERT multiple GPUs should aggregate all in gpus array'
);

-- Test 3: Verify GPU JSON structure
SELECT ok(
    (SELECT gpus->0->>'vendor' FROM nodex WHERE node_id = 1001) = 'NVIDIA',
    'GPU JSON should contain vendor field'
);

SELECT ok(
    (SELECT gpus->0->>'device' FROM nodex WHERE node_id = 1001) = 'RTX 3090',
    'GPU JSON should contain device field'
);

-- Test 4: DELETE node_gpu should update count and array
DELETE FROM node_gpu WHERE id = 'gpu-1';

SELECT pg_sleep(0.1);

SELECT is(
    (SELECT node_gpu_count FROM nodex WHERE node_id = 1001),
    2,
    'DELETE node_gpu should decrement node_gpu_count'
);

SELECT is(
    (SELECT free_gpu_count FROM nodex WHERE node_id = 1001),
    2,
    'DELETE free gpu should decrement free_gpu_count'
);

-- Test 5: GPU with contract assigned should not count as free
INSERT INTO node_gpu (id, node_twin_id, vendor, device, vram, contract)
VALUES ('gpu-4', 2001, 'NVIDIA', 'A100', 40000000000, 1001);

SELECT pg_sleep(0.1);

SELECT is(
    (SELECT free_gpu_count FROM nodex WHERE node_id = 1001),
    2,
    'GPU with contract should not increment free_gpu_count'
);

-- Cleanup
SELECT cleanup_test_data();

SELECT * FROM finish();

ROLLBACK;
