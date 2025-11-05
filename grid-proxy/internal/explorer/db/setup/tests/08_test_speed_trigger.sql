-- ============================================================================
-- TEST: reflect_speed_changes Trigger
-- ============================================================================
-- Tests for speed INSERT and UPDATE operations

BEGIN;

-- Load pgTAP
SELECT plan(10);

-- Setup: Create test data
SELECT create_test_farm(1001, 2001, 'Test Farm 1', 1);
SELECT create_test_pricing_policy(1, 1000000, 1000000);
SELECT create_test_twin(2001);
SELECT create_test_node(1001, 1001, 2001);
SELECT create_test_node_resources_total('node-1001', 1000000000000, 100000000000, 100000000000, 16);
SELECT pg_sleep(0.1);

-- Test 1: INSERT speed should update all speed fields
INSERT INTO speed (
    node_twin_id, upload, download, 
    udp_download_ipv4, udp_upload_ipv4,
    tcp_download_ipv6, tcp_upload_ipv6,
    udp_download_ipv6, udp_upload_ipv6,
    updated_at
) VALUES (
    2001, 100.5, 200.5,
    150.5, 160.5,
    170.5, 180.5,
    190.5, 200.5,
    EXTRACT(EPOCH FROM NOW())::BIGINT
);

SELECT pg_sleep(0.1);

SELECT is(
    (SELECT upload_speed FROM resources_cache WHERE node_id = 1001),
    100.5,
    'INSERT speed should set upload_speed'
);

SELECT is(
    (SELECT download_speed FROM resources_cache WHERE node_id = 1001),
    200.5,
    'INSERT speed should set download_speed'
);

SELECT is(
    (SELECT udp_download_ipv4 FROM resources_cache WHERE node_id = 1001),
    150.5,
    'INSERT speed should set udp_download_ipv4'
);

SELECT is(
    (SELECT udp_upload_ipv4 FROM resources_cache WHERE node_id = 1001),
    160.5,
    'INSERT speed should set udp_upload_ipv4'
);

SELECT is(
    (SELECT tcp_download_ipv6 FROM resources_cache WHERE node_id = 1001),
    170.5,
    'INSERT speed should set tcp_download_ipv6'
);

SELECT is(
    (SELECT tcp_upload_ipv6 FROM resources_cache WHERE node_id = 1001),
    180.5,
    'INSERT speed should set tcp_upload_ipv6'
);

SELECT is(
    (SELECT udp_download_ipv6 FROM resources_cache WHERE node_id = 1001),
    190.5,
    'INSERT speed should set udp_download_ipv6'
);

SELECT is(
    (SELECT udp_upload_ipv6 FROM resources_cache WHERE node_id = 1001),
    200.5,
    'INSERT speed should set udp_upload_ipv6'
);

-- Test 2: UPDATE speed should update all fields
UPDATE speed SET 
    upload = 300.5,
    download = 400.5,
    udp_download_ipv4 = 350.5
WHERE node_twin_id = 2001;

SELECT pg_sleep(0.1);

SELECT is(
    (SELECT upload_speed FROM resources_cache WHERE node_id = 1001),
    300.5,
    'UPDATE speed should update upload_speed'
);

SELECT is(
    (SELECT download_speed FROM resources_cache WHERE node_id = 1001),
    400.5,
    'UPDATE speed should update download_speed'
);

-- Cleanup
SELECT cleanup_test_data();

SELECT * FROM finish();

ROLLBACK;

