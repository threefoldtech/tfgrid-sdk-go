-- ============================================================================
-- CONSTANTS
-- ============================================================================
-- This file defines all constant values used throughout the cache system.
-- Constants are stored in a table for easy modification and querying.

-- Create constants table
CREATE TABLE IF NOT EXISTS cache_constants (
    constant_name TEXT PRIMARY KEY,
    constant_value NUMERIC NOT NULL,
    description TEXT NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Resource reservation constants (in bytes)
INSERT INTO cache_constants (constant_name, constant_value, description) VALUES
    ('MRU_RESERVED_MIN_BYTES', 2147483648, 'MRU reserved minimum: 2GB (2^31 bytes)'),
    ('MRU_RESERVED_FRACTION', 10, 'MRU reserved fraction: 10% (1/10 of total MRU)'),
    ('SRU_RESERVED_BYTES', 21474836480, 'SRU fixed reservation: 20GB')
ON CONFLICT (constant_name) DO NOTHING;

-- Storage Unit (SU) conversion factors
INSERT INTO cache_constants (constant_name, constant_value, description) VALUES
    ('HRU_TO_SU_FACTOR', 1200, 'HRU to SU conversion: 1 SU = 1200 GB HRU'),
    ('SRU_TO_SU_FACTOR', 200, 'SRU to SU conversion: 1 SU = 200 GB SRU')
ON CONFLICT (constant_name) DO NOTHING;

-- Price calculation constants
INSERT INTO cache_constants (constant_name, constant_value, description) VALUES
    ('CERTIFIED_MULTIPLIER', 1.25, 'Certified node premium: 25% (multiplier = 1.25)'),
    ('HOURS_PER_MONTH', 720, 'Hours per month: 24 hours * 30 days = 720'),
    ('USD_CONVERSION_FACTOR', 10000000, 'USD conversion: divide by 1e7 (10000000)')
ON CONFLICT (constant_name) DO NOTHING;

-- Bytes to GB conversion factor
INSERT INTO cache_constants (constant_name, constant_value, description) VALUES
    ('BYTES_PER_GB', 1073741824, 'Bytes per GB: 1024 * 1024 * 1024 = 1073741824')
ON CONFLICT (constant_name) DO NOTHING;

-- Discount calculation constants
INSERT INTO cache_constants (constant_name, constant_value, description) VALUES
    ('DISCOUNT_TIER_18X', 18, 'Discount tier: balance >= 18x cost (60% discount)'),
    ('DISCOUNT_TIER_6X', 6, 'Discount tier: balance >= 6x cost (40% discount)'),
    ('DISCOUNT_TIER_3X', 3, 'Discount tier: balance >= 3x cost (30% discount)'),
    ('DISCOUNT_TIER_1_5X', 1.5, 'Discount tier: balance >= 1.5x cost (20% discount)'),
    ('DISCOUNT_60_PCT', 0.6, 'Discount percentage: 60%'),
    ('DISCOUNT_40_PCT', 0.4, 'Discount percentage: 40%'),
    ('DISCOUNT_30_PCT', 0.3, 'Discount percentage: 30%'),
    ('DISCOUNT_20_PCT', 0.2, 'Discount percentage: 20%')
ON CONFLICT (constant_name) DO NOTHING;

-- Helper functions to retrieve constants
CREATE OR REPLACE FUNCTION get_cache_constant(const_name TEXT) RETURNS NUMERIC AS $$
DECLARE
    result NUMERIC;
BEGIN
    SELECT constant_value INTO result
    FROM cache_constants
    WHERE constant_name = const_name;
    
    IF result IS NULL THEN
        RAISE EXCEPTION 'Constant % not found', const_name;
    END IF;
    
    RETURN result;
END;
$$ LANGUAGE plpgsql STABLE;

-- Convenience functions for commonly used constants
-- These are IMMUTABLE for use in generated columns and views
-- Values are read from cache_constants table but cached as IMMUTABLE functions for performance

CREATE OR REPLACE FUNCTION get_mru_reserved_min_bytes() RETURNS NUMERIC AS $$
    -- MRU reserved minimum: 2GB = 2147483648 bytes
    SELECT 2147483648::NUMERIC;
$$ LANGUAGE sql IMMUTABLE;

CREATE OR REPLACE FUNCTION get_mru_reserved_fraction() RETURNS NUMERIC AS $$
    -- MRU reserved fraction: 10% (1/10 of total MRU)
    SELECT 10::NUMERIC;
$$ LANGUAGE sql IMMUTABLE;

CREATE OR REPLACE FUNCTION get_sru_reserved_bytes() RETURNS NUMERIC AS $$
    -- SRU fixed reservation: 20GB = 21474836480 bytes
    SELECT 21474836480::NUMERIC;
$$ LANGUAGE sql IMMUTABLE;

CREATE OR REPLACE FUNCTION get_hru_to_su_factor() RETURNS NUMERIC AS $$
    -- HRU to SU conversion: 1 SU = 1200 GB HRU
    SELECT 1200::NUMERIC;
$$ LANGUAGE sql IMMUTABLE;

CREATE OR REPLACE FUNCTION get_sru_to_su_factor() RETURNS NUMERIC AS $$
    -- SRU to SU conversion: 1 SU = 200 GB SRU
    SELECT 200::NUMERIC;
$$ LANGUAGE sql IMMUTABLE;

CREATE OR REPLACE FUNCTION get_certified_multiplier() RETURNS NUMERIC AS $$
    -- Certified node premium: 25% (multiplier = 1.25)
    SELECT 1.25::NUMERIC;
$$ LANGUAGE sql IMMUTABLE;

CREATE OR REPLACE FUNCTION get_hours_per_month() RETURNS NUMERIC AS $$
    -- Hours per month: 24 hours * 30 days = 720
    SELECT 720::NUMERIC;
$$ LANGUAGE sql IMMUTABLE;

CREATE OR REPLACE FUNCTION get_usd_conversion_factor() RETURNS NUMERIC AS $$
    -- USD conversion: divide by 1e7 (10000000)
    SELECT 10000000::NUMERIC;
$$ LANGUAGE sql IMMUTABLE;

CREATE OR REPLACE FUNCTION get_bytes_per_gb() RETURNS NUMERIC AS $$
    -- Bytes per GB: 1024 * 1024 * 1024 = 1073741824
    SELECT 1073741824::NUMERIC;
$$ LANGUAGE sql IMMUTABLE;
