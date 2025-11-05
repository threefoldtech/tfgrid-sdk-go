-- ============================================================================
-- TEST: reflect_cpu_benchmark_changes Trigger
-- ============================================================================
-- Tests for cpu_benchmark INSERT and UPDATE operations

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

-- Test 1: INSERT cpu_benchmark should update all CPU fields
INSERT INTO cpu_benchmark (
    node_twin_id, single_threaded, multi_threaded, threads, workloads, updated_at
) VALUES (
    2001, 1000.5, 8000.5, 16, 8, EXTRACT(EPOCH FROM NOW())::BIGINT
);

SELECT pg_sleep(0.1);

SELECT is(
    (SELECT single_threaded_cpu FROM resources_cache WHERE node_id = 1001),
    1000.5,
    'INSERT cpu_benchmark should set single_threaded_cpu'
);

SELECT is(
    (SELECT multi_threaded_cpu FROM resources_cache WHERE node_id = 1001),
    8000.5,
    'INSERT cpu_benchmark should set multi_threaded_cpu'
);

SELECT is(
    (SELECT threads_cpu FROM resources_cache WHERE node_id = 1001),
    16,
    'INSERT cpu_benchmark should set threads_cpu'
);

SELECT is(
    (SELECT workloads_cpu FROM resources_cache WHERE node_id = 1001),
    8,
    'INSERT cpu_benchmark should set workloads_cpu'
);

-- Test 2: UPDATE cpu_benchmark should update all fields
UPDATE cpu_benchmark SET 
    single_threaded = 2000.5,
    multi_threaded = 16000.5,
    threads = 32,
    workloads = 16
WHERE node_twin_id = 2001;

SELECT pg_sleep(0.1);

SELECT is(
    (SELECT single_threaded_cpu FROM resources_cache WHERE node_id = 1001),
    2000.5,
    'UPDATE cpu_benchmark should update single_threaded_cpu'
);

SELECT is(
    (SELECT multi_threaded_cpu FROM resources_cache WHERE node_id = 1001),
    16000.5,
    'UPDATE cpu_benchmark should update multi_threaded_cpu'
);

-- Cleanup
SELECT cleanup_test_data();

SELECT * FROM finish();

ROLLBACK;

