-- Utility functions for calculations

DROP FUNCTION IF EXISTS convert_to_decimal(v_input text);

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

CREATE OR REPLACE FUNCTION calc_discount(
    cost NUMERIC,
    balance NUMERIC
) RETURNS NUMERIC AS $$
DECLARE
    discount NUMERIC;
    discount_tier_18x CONSTANT NUMERIC := 18;
    discount_tier_6x CONSTANT NUMERIC := 6;
    discount_tier_3x CONSTANT NUMERIC := 3;
    discount_tier_1_5x CONSTANT NUMERIC := 1.5;
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
    SELECT pricing_policy.cu->'value', pricing_policy.su->'value'
    INTO cu_value, su_value
    FROM pricing_policy
    WHERE pricing_policy_id = policy_id;

    IF cu_value IS NULL OR su_value IS NULL THEN
        RAISE EXCEPTION 'pricing values not found for policy_id: %', policy_id;
    END IF;

    cu := LEAST(
        GREATEST(mru / 4, cru / 2),
        GREATEST(mru / 8, cru),
        GREATEST(mru / 2, cru / 4)
    );

    su := (hru / get_hru_to_su_factor() + sru / get_sru_to_su_factor());

    cost_per_month := (cu * cu_value + su * su_value + extra_fee) *
        (CASE certified WHEN true THEN get_certified_multiplier() ELSE 1 END) *
        get_hours_per_month();

    RETURN cost_per_month / get_usd_conversion_factor();
END;
$$ LANGUAGE plpgsql STABLE;

