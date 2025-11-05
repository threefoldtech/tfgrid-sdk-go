-- ============================================================================
-- TEST: reflect_node_gpu_count_change Trigger
-- ============================================================================
-- Tests for node_gpu INSERT, DELETE, and UPDATE operations

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

-- Test 1: INSERT node_gpu should update gpu count and gpus array
INSERT INTO node_gpu (id, node_twin_id, vendor, device, vram, contract)
VALUES ('gpu-1', 2001, 'NVIDIA', 'RTX 3090', 24000000000, 0);

SELECT pg_sleep(0.1);

SELECT is(
    (SELECT node_gpu_count FROM resources_cache WHERE node_id = 1001),
    1,
    'INSERT node_gpu should set node_gpu_count to 1'
);

SELECT ok(
    (SELECT gpus FROM resources_cache WHERE node_id = 1001) IS NOT NULL,
    'INSERT node_gpu should populate gpus JSONB array'
);

SELECT ok(
    jsonb_array_length((SELECT gpus FROM resources_cache WHERE node_id = 1001)) = 1,
    'INSERT node_gpu should create gpus array with 1 element'
);

-- Test 2: INSERT multiple GPUs should aggregate correctly
INSERT INTO node_gpu (id, node_twin_id, vendor, device, vram, contract)
VALUES ('gpu-2', 2001, 'NVIDIA', 'RTX 4090', 24000000000, 0),
       ('gpu-3', 2001, 'AMD', 'RX 7900', 20000000000, 0);

SELECT pg_sleep(0.1);

SELECT is(
    (SELECT node_gpu_count FROM resources_cache WHERE node_id = 1001),
    3,
    'INSERT multiple GPUs should increment node_gpu_count'
);

SELECT ok(
    jsonb_array_length((SELECT gpus FROM resources_cache WHERE node_id = 1001)) = 3,
    'INSERT multiple GPUs should aggregate all in gpus array'
);

-- Test 3: Verify GPU JSON structure
SELECT ok(
    (SELECT gpus->0->>'vendor' FROM resources_cache WHERE node_id = 1001) = 'NVIDIA',
    'GPU JSON should contain vendor field'
);

SELECT ok(
    (SELECT gpus->0->>'device' FROM resources_cache WHERE node_id = 1001) = 'RTX 3090',
    'GPU JSON should contain device field'
);

-- Test 4: DELETE node_gpu should update count and array
DELETE FROM node_gpu WHERE id = 'gpu-1';

SELECT pg_sleep(0.1);

SELECT is(
    (SELECT node_gpu_count FROM resources_cache WHERE node_id = 1001),
    2,
    'DELETE node_gpu should decrement node_gpu_count'
);

SELECT ok(
    jsonb_array_length((SELECT gpus FROM resources_cache WHERE node_id = 1001)) = 2,
    'DELETE node_gpu should remove from gpus array'
);

-- Test 5: UPDATE node_gpu should re-aggregate
UPDATE node_gpu SET vram = 30000000000 WHERE id = 'gpu-2';

SELECT pg_sleep(0.1);

SELECT ok(
    EXISTS (
        SELECT 1 FROM resources_cache 
        WHERE node_id = 1001 
        AND gpus::jsonb @> '[{"id": "gpu-2", "vram": 30000000000}]'::jsonb
    ),
    'UPDATE node_gpu should update gpus array'
);

-- Test 6: DELETE all GPUs should set count to 0
DELETE FROM node_gpu WHERE node_twin_id = 2001;

SELECT pg_sleep(0.1);

SELECT is(
    (SELECT node_gpu_count FROM resources_cache WHERE node_id = 1001),
    0,
    'DELETE all GPUs should set node_gpu_count to 0'
);

SELECT ok(
    (SELECT gpus FROM resources_cache WHERE node_id = 1001) = '[]'::jsonb,
    'DELETE all GPUs should set gpus to empty array'
);

-- Test 7: GPU with contract assigned
INSERT INTO node_gpu (id, node_twin_id, vendor, device, vram, contract)
VALUES ('gpu-4', 2001, 'NVIDIA', 'A100', 40000000000, 1001);

SELECT pg_sleep(0.1);

SELECT ok(
    (SELECT gpus::jsonb @> '[{"contract": 1001}]'::jsonb FROM resources_cache WHERE node_id = 1001),
    'GPU with contract should be stored in gpus array'
);

-- Cleanup
SELECT cleanup_test_data();

SELECT * FROM finish();

ROLLBACK;

