-- ============================================================================
-- TEST: reflect_public_ip_changes Trigger
-- ============================================================================
-- Tests for public_ip INSERT, DELETE, and UPDATE contract_id operations

BEGIN;

-- Load pgTAP
SELECT plan(12);

-- Setup: Create test data
SELECT create_test_farm(1001, 2001, 'Test Farm 1', 1);
SELECT create_test_twin(2001);
SELECT pg_sleep(0.1);

-- Verify initial state (should have 0 IPs)
SELECT is(
    (SELECT free_ips FROM farmx WHERE farm_id = 1001),
    0,
    'Initial state should have 0 free IPs'
);

SELECT is(
    (SELECT total_ips FROM farmx WHERE farm_id = 1001),
    0,
    'Initial state should have 0 total IPs'
);

-- Test 1: INSERT public_ip with contract_id=0 should increment free_ips and total_ips
INSERT INTO public_ip (id, gateway, ip, contract_id, farm_id)
VALUES ('ip-1', '1.2.3.1', '1.2.3.4', 0, 'farm-1001');

SELECT pg_sleep(0.1);

SELECT is(
    (SELECT free_ips FROM farmx WHERE farm_id = 1001),
    1,
    'INSERT public_ip with contract_id=0 should increment free_ips'
);

SELECT is(
    (SELECT total_ips FROM farmx WHERE farm_id = 1001),
    1,
    'INSERT public_ip should increment total_ips'
);

-- Test 2: INSERT public_ip with contract_id!=0 should NOT increment free_ips
INSERT INTO public_ip (id, gateway, ip, contract_id, farm_id)
VALUES ('ip-2', '1.2.3.1', '1.2.3.5', 1001, 'farm-1001');

SELECT pg_sleep(0.1);

SELECT is(
    (SELECT free_ips FROM farmx WHERE farm_id = 1001),
    1,
    'INSERT public_ip with contract_id!=0 should NOT increment free_ips'
);

SELECT is(
    (SELECT total_ips FROM farmx WHERE farm_id = 1001),
    2,
    'INSERT public_ip should increment total_ips'
);

-- Test 3: UPDATE contract_id from 0 to non-zero should decrement free_ips
UPDATE public_ip SET contract_id = 1002 WHERE id = 'ip-1';

SELECT pg_sleep(0.1);

SELECT is(
    (SELECT free_ips FROM farmx WHERE farm_id = 1001),
    0,
    'UPDATE contract_id from 0 to non-zero should decrement free_ips'
);

-- Test 4: UPDATE contract_id from non-zero to 0 should increment free_ips
UPDATE public_ip SET contract_id = 0 WHERE id = 'ip-2';

SELECT pg_sleep(0.1);

SELECT is(
    (SELECT free_ips FROM farmx WHERE farm_id = 1001),
    1,
    'UPDATE contract_id from non-zero to 0 should increment free_ips'
);

-- Test 5: DELETE public_ip with contract_id=0 should decrement free_ips and total_ips
DELETE FROM public_ip WHERE id = 'ip-2';

SELECT pg_sleep(0.1);

SELECT is(
    (SELECT free_ips FROM farmx WHERE farm_id = 1001),
    0,
    'DELETE public_ip with contract_id=0 should decrement free_ips'
);

SELECT is(
    (SELECT total_ips FROM farmx WHERE farm_id = 1001),
    1,
    'DELETE public_ip should decrement total_ips'
);

-- Test 6: DELETE public_ip with contract_id!=0 should NOT decrement free_ips
DELETE FROM public_ip WHERE id = 'ip-1';

SELECT pg_sleep(0.1);

SELECT is(
    (SELECT free_ips FROM farmx WHERE farm_id = 1001),
    0,
    'DELETE public_ip with contract_id!=0 should NOT decrement free_ips'
);

SELECT is(
    (SELECT total_ips FROM farmx WHERE farm_id = 1001),
    0,
    'DELETE public_ip should decrement total_ips'
);

-- Test 7: Verify IPs JSON array is updated
INSERT INTO public_ip (id, gateway, ip, contract_id, farm_id)
VALUES 
    ('ip-3', '1.2.3.1', '1.2.3.6', 0, 'farm-1001'),
    ('ip-4', '1.2.3.1', '1.2.3.7', 1001, 'farm-1001');

SELECT pg_sleep(0.1);

SELECT ok(
    (SELECT ips FROM farmx WHERE farm_id = 1001) IS NOT NULL,
    'IPs JSON array should be populated'
);

SELECT ok(
    jsonb_array_length((SELECT ips FROM farmx WHERE farm_id = 1001)) = 2,
    'IPs JSON array should contain all IPs for the farm'
);

-- Cleanup
SELECT cleanup_test_data();

SELECT * FROM finish();

ROLLBACK;

