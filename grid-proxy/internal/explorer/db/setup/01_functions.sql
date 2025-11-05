-- ============================================================================
-- HELPER FUNCTIONS
-- ============================================================================
-- Utility functions used for calculations and data transformations

DROP FUNCTION IF EXISTS convert_to_decimal(v_input text);

/*
 * convert_to_decimal
 * 
 * Safely converts a text input to decimal, returning NULL if conversion fails.
 * This prevents errors from invalid numeric strings in the data.
 * 
 * @param v_input - Text value to convert
 * @returns DECIMAL or NULL if conversion fails
 */
CREATE OR REPLACE FUNCTION convert_to_decimal(v_input TEXT) RETURNS DECIMAL AS 
$$ 
DECLARE 
    v_dec_value DECIMAL DEFAULT NULL;
BEGIN     
    BEGIN 
        v_dec_value := v_input::DECIMAL;
    EXCEPTION
        WHEN OTHERS THEN 
            RAISE NOTICE 'Invalid decimal value: "%". Returning NULL.', v_input;
            RETURN NULL;
    END;
    RETURN v_dec_value;
END;
$$ LANGUAGE plpgsql;

/*
 * calc_discount
 * 
 * Calculates discount based on balance-to-cost ratio.
 * Higher balance relative to cost results in larger discounts.
 * 
 * Discount tiers:
 *   - 60% discount: balance >= 18x cost
 *   - 40% discount: balance >= 6x cost
 *   - 30% discount: balance >= 3x cost
 *   - 20% discount: balance >= 1.5x cost
 *   - 0% discount: otherwise
 * 
 * @param cost - Base cost amount
 * @param balance - Available balance
 * @returns NUMERIC - Final cost after discount
 */
CREATE OR REPLACE FUNCTION calc_discount(
    cost NUMERIC,
    balance NUMERIC
) RETURNS NUMERIC AS $$
DECLARE
    discount NUMERIC;
    -- Discount tier multipliers
    discount_tier_18x CONSTANT NUMERIC := 18;
    discount_tier_6x CONSTANT NUMERIC := 6;
    discount_tier_3x CONSTANT NUMERIC := 3;
    discount_tier_1_5x CONSTANT NUMERIC := 1.5;
    -- Discount percentages
    discount_60_pct CONSTANT NUMERIC := 0.6;
    discount_40_pct CONSTANT NUMERIC := 0.4;
    discount_30_pct CONSTANT NUMERIC := 0.3;
    discount_20_pct CONSTANT NUMERIC := 0.2;
BEGIN
    discount := (
        CASE 
            WHEN balance >= cost * discount_tier_18x THEN discount_60_pct
            WHEN balance >= cost * discount_tier_6x THEN discount_40_pct
            WHEN balance >= cost * discount_tier_3x THEN discount_30_pct
            WHEN balance >= cost * discount_tier_1_5x THEN discount_20_pct
            ELSE 0
        END
    );

    RETURN cost - cost * discount;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

/*
 * calc_price
 * 
 * Calculates the monthly price for node resources based on pricing policy.
 * 
 * Price calculation:
 *   1. Compute CU (Compute Unit) from CRU and MRU using the maximum of three formulas
 *   2. Compute SU (Storage Unit) from HRU and SRU
 *   3. Apply certified node multiplier (25% premium)
 *   4. Convert to USD (divide by 1e7)
 * 
 * @param cru - Compute Resource Units
 * @param sru - Storage Resource Units (in bytes)
 * @param hru - HDD Resource Units (in bytes)
 * @param mru - Memory Resource Units (in bytes)
 * @param certified - Whether node is certified (25% premium)
 * @param policy_id - Pricing policy ID
 * @param extra_fee - Additional fees
 * @returns NUMERIC - Price in USD per month
 */
CREATE OR REPLACE FUNCTION calc_price(
    cru NUMERIC,
    sru NUMERIC,
    hru NUMERIC,
    mru NUMERIC,
    certified BOOLEAN,
    policy_id INTEGER,
    extra_fee NUMERIC
) RETURNS NUMERIC AS $$
DECLARE
    su NUMERIC;
    cu NUMERIC;
    su_value NUMERIC;
    cu_value NUMERIC;
    cost_per_month NUMERIC;
BEGIN
    -- Fetch pricing values from policy (optimized single query)
    SELECT pricing_policy.cu->'value', pricing_policy.su->'value'
    INTO cu_value, su_value
    FROM pricing_policy
    WHERE pricing_policy_id = policy_id;

    IF cu_value IS NULL OR su_value IS NULL THEN
        RAISE EXCEPTION 'pricing values not found for policy_id: %', policy_id;
    END IF;

    -- Compute CU: take the minimum of three different calculation methods
    -- This ensures CU reflects the most constraining resource
    cu := LEAST(
        GREATEST(mru / 4, cru / 2),
        GREATEST(mru / 8, cru),
        GREATEST(mru / 2, cru / 4)
    );

    -- Compute SU: Storage Unit = HRU/HRU_TO_SU_FACTOR + SRU/SRU_TO_SU_FACTOR
    su := (hru / get_hru_to_su_factor() + sru / get_sru_to_su_factor());

    -- Calculate monthly cost:
    --   (CU * CU_price + SU * SU_price + extra_fee) * certified_multiplier * hours_per_month
    cost_per_month := (cu * cu_value + su * su_value + extra_fee) *
        (CASE certified WHEN true THEN get_certified_multiplier() ELSE 1 END) *
        get_hours_per_month();

    -- Convert to USD (divide by conversion factor)
    RETURN cost_per_month / get_usd_conversion_factor();
END;
$$ LANGUAGE plpgsql STABLE;

