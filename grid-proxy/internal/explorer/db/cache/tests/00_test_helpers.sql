-- ============================================================================
-- TEST HELPERS
-- ============================================================================
-- Helper functions for setting up test data and cleaning up after tests

-- Function to create minimal test data for a node
CREATE OR REPLACE FUNCTION create_test_node(
    p_node_id INTEGER,
    p_farm_id INTEGER,
    p_twin_id INTEGER,
    p_country TEXT DEFAULT 'US',
    p_certification TEXT DEFAULT NULL,
    p_extra_fee NUMERIC DEFAULT 0
) RETURNS VOID AS $$
BEGIN
    INSERT INTO node (
        id, grid_version, node_id, farm_id, twin_id, country, 
        created, farming_policy_id, secure, virtualized, created_at, updated_at, certification, extra_fee
    ) VALUES (
        'node-' || p_node_id::TEXT, 3, p_node_id, p_farm_id, p_twin_id, p_country,
        EXTRACT(EPOCH FROM NOW())::INTEGER, 1, false, false, 
        EXTRACT(EPOCH FROM NOW())::NUMERIC, EXTRACT(EPOCH FROM NOW())::NUMERIC,
        COALESCE(p_certification, 'Diy'), p_extra_fee
    ) ON CONFLICT (id) DO NOTHING;
END;
$$ LANGUAGE plpgsql;

-- Function to create test farm
CREATE OR REPLACE FUNCTION create_test_farm(
    p_farm_id INTEGER,
    p_twin_id INTEGER,
    p_name TEXT DEFAULT 'Test Farm',
    p_pricing_policy_id INTEGER DEFAULT 1
) RETURNS VOID AS $$
BEGIN
    INSERT INTO farm (
        id, grid_version, farm_id, name, twin_id, pricing_policy_id, dedicated_farm, certification
    ) VALUES (
        'farm-' || p_farm_id::TEXT, 3, p_farm_id, p_name, p_twin_id, p_pricing_policy_id, false, 'Certified'
    ) ON CONFLICT (id) DO NOTHING;
END;
$$ LANGUAGE plpgsql;

-- Function to create test node_resources_total
CREATE OR REPLACE FUNCTION create_test_node_resources_total(
    p_node_id VARCHAR,
    p_hru NUMERIC DEFAULT 1000000000000, -- 1TB
    p_mru NUMERIC DEFAULT 100000000000,  -- 100GB
    p_sru NUMERIC DEFAULT 100000000000,  -- 100GB
    p_cru NUMERIC DEFAULT 16              -- 16 cores
) RETURNS VOID AS $$
BEGIN
    INSERT INTO node_resources_total (
        id, hru, mru, sru, cru, node_id
    ) VALUES (
        'nrt-' || p_node_id, p_hru, p_mru, p_sru, p_cru, p_node_id
    ) ON CONFLICT (id) DO UPDATE SET
        hru = EXCLUDED.hru,
        mru = EXCLUDED.mru,
        sru = EXCLUDED.sru,
        cru = EXCLUDED.cru;
END;
$$ LANGUAGE plpgsql;

-- Function to create test node_contract
CREATE OR REPLACE FUNCTION create_test_node_contract(
    p_contract_id VARCHAR,
    p_node_id INTEGER,
    p_twin_id INTEGER,
    p_state TEXT DEFAULT 'Created',
    p_resources_used_id VARCHAR DEFAULT NULL
) RETURNS VARCHAR AS $$
DECLARE
    v_resources_id VARCHAR;
BEGIN
    IF p_resources_used_id IS NULL THEN
        v_resources_id := 'cr-' || p_contract_id;
    ELSE
        v_resources_id := p_resources_used_id;
    END IF;

    INSERT INTO node_contract (
        id, grid_version, contract_id, twin_id, node_id, deployment_data,
        deployment_hash, number_of_public_i_ps, created_at, resources_used_id, state
    ) VALUES (
        'nc-' || p_contract_id, 3, p_contract_id::NUMERIC, p_twin_id, p_node_id,
        '{}', 'hash', 0, EXTRACT(EPOCH FROM NOW())::NUMERIC, v_resources_id, p_state
    ) ON CONFLICT (id) DO UPDATE SET
        state = EXCLUDED.state,
        resources_used_id = EXCLUDED.resources_used_id;
    
    RETURN v_resources_id;
END;
$$ LANGUAGE plpgsql;

-- Function to create test contract_resources
CREATE OR REPLACE FUNCTION create_test_contract_resources(
    p_contract_id VARCHAR,
    p_hru NUMERIC DEFAULT 1000000000,    -- 1GB
    p_mru NUMERIC DEFAULT 1000000000,    -- 1GB
    p_sru NUMERIC DEFAULT 1000000000,    -- 1GB
    p_cru NUMERIC DEFAULT 1              -- 1 core
) RETURNS VOID AS $$
BEGIN
    INSERT INTO contract_resources (
        id, hru, mru, sru, cru, contract_id
    ) VALUES (
        'cr-' || p_contract_id, p_hru, p_mru, p_sru, p_cru, p_contract_id
    ) ON CONFLICT (id) DO UPDATE SET
        hru = EXCLUDED.hru,
        mru = EXCLUDED.mru,
        sru = EXCLUDED.sru,
        cru = EXCLUDED.cru;
END;
$$ LANGUAGE plpgsql;

-- Function to create test pricing_policy
CREATE OR REPLACE FUNCTION create_test_pricing_policy(
    p_policy_id INTEGER,
    p_cu_value NUMERIC DEFAULT 1000000,
    p_su_value NUMERIC DEFAULT 1000000
) RETURNS VOID AS $$
BEGIN
    INSERT INTO pricing_policy (
        id, grid_version, pricing_policy_id, name, su, cu, nu, ipu,
        foundation_account, certified_sales_account, dedicated_node_discount
    ) VALUES (
        'pp-' || p_policy_id::TEXT, 3, p_policy_id, 'Test Policy',
        jsonb_build_object('value', p_su_value),
        jsonb_build_object('value', p_cu_value),
        '{}'::jsonb, '{}'::jsonb, 'foundation', 'certified', 0
    ) ON CONFLICT (id) DO UPDATE SET
        su = EXCLUDED.su,
        cu = EXCLUDED.cu;
END;
$$ LANGUAGE plpgsql;

-- Function to create test public_ip
CREATE OR REPLACE FUNCTION create_test_public_ip(
    p_ip_id VARCHAR,
    p_farm_id VARCHAR,
    p_ip TEXT DEFAULT '1.2.3.4',
    p_gateway TEXT DEFAULT '1.2.3.1',
    p_contract_id NUMERIC DEFAULT 0
) RETURNS VOID AS $$
BEGIN
    INSERT INTO public_ip (
        id, gateway, ip, contract_id, farm_id
    ) VALUES (
        p_ip_id, p_gateway, p_ip, p_contract_id, p_farm_id
    ) ON CONFLICT (id) DO UPDATE SET
        contract_id = EXCLUDED.contract_id,
        ip = EXCLUDED.ip,
        gateway = EXCLUDED.gateway;
END;
$$ LANGUAGE plpgsql;

-- Function to create test twin
CREATE OR REPLACE FUNCTION create_test_twin(
    p_twin_id INTEGER,
    p_account_id TEXT DEFAULT 'account123'
) RETURNS VOID AS $$
BEGIN
    INSERT INTO twin (
        id, grid_version, twin_id, account_id, relay
    ) VALUES (
        'twin-' || p_twin_id::TEXT, 3, p_twin_id, p_account_id, ''
    ) ON CONFLICT (id) DO NOTHING;
END;
$$ LANGUAGE plpgsql;

-- Function to clean up test data
CREATE OR REPLACE FUNCTION cleanup_test_data() RETURNS VOID AS $$
BEGIN
    -- Clean in reverse order of dependencies
    DELETE FROM contract_resources WHERE id LIKE 'cr-%';
    DELETE FROM node_contract WHERE id LIKE 'nc-%';
    DELETE FROM node_gpu WHERE id LIKE 'gpu-%';
    DELETE FROM dmi WHERE node_twin_id IN (SELECT twin_id FROM node WHERE id LIKE 'node-%');
    DELETE FROM speed WHERE node_twin_id IN (SELECT twin_id FROM node WHERE id LIKE 'node-%');
    DELETE FROM cpu_benchmark WHERE node_twin_id IN (SELECT twin_id FROM node WHERE id LIKE 'node-%');
    DELETE FROM node_resources_total WHERE id LIKE 'nrt-%';
    DELETE FROM rent_contract WHERE id LIKE 'rc-%';
    DELETE FROM public_ip WHERE id LIKE 'ip-%';
    DELETE FROM node WHERE id LIKE 'node-%';
    DELETE FROM farm WHERE id LIKE 'farm-%';
    DELETE FROM twin WHERE id LIKE 'twin-%';
    DELETE FROM resources_cache WHERE node_id IN (SELECT node_id FROM node WHERE id LIKE 'node-%');
    DELETE FROM public_ips_cache WHERE farm_id IN (SELECT farm_id FROM farm WHERE id LIKE 'farm-%');
END;
$$ LANGUAGE plpgsql;

