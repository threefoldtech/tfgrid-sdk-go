-- Error logging for cache maintenance

CREATE TABLE IF NOT EXISTS cache_errors (
    id BIGSERIAL PRIMARY KEY,
    error_timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    error_type TEXT NOT NULL,
    operation TEXT NOT NULL,
    table_name TEXT,
    record_id TEXT,
    error_message TEXT NOT NULL,
    error_context JSONB,
    resolved BOOLEAN DEFAULT FALSE,
    resolved_at TIMESTAMP,
    resolved_by TEXT,
    notes TEXT
);

CREATE OR REPLACE FUNCTION log_cache_error(
    p_error_type TEXT,
    p_operation TEXT,
    p_table_name TEXT DEFAULT NULL,
    p_record_id TEXT DEFAULT NULL,
    p_error_message TEXT DEFAULT NULL,
    p_error_context JSONB DEFAULT NULL
) RETURNS VOID AS $$
BEGIN
    INSERT INTO cache_errors (
        error_type,
        operation,
        table_name,
        record_id,
        error_message,
        error_context
    ) VALUES (
        p_error_type,
        p_operation,
        p_table_name,
        p_record_id,
        COALESCE(p_error_message, SQLERRM),
        p_error_context
    );
EXCEPTION
    WHEN OTHERS THEN
        RAISE WARNING 'Failed to log error to cache_errors: %', SQLERRM;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION get_unresolved_errors(p_error_type TEXT DEFAULT NULL) RETURNS INTEGER AS $$
DECLARE
    error_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO error_count
    FROM cache_errors
    WHERE resolved = FALSE
    AND (p_error_type IS NULL OR error_type = p_error_type);
    
    RETURN error_count;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION resolve_cache_error(
    p_error_id BIGINT,
    p_resolved_by TEXT DEFAULT NULL,
    p_notes TEXT DEFAULT NULL
) RETURNS VOID AS $$
BEGIN
    UPDATE cache_errors
    SET resolved = TRUE,
        resolved_at = CURRENT_TIMESTAMP,
        resolved_by = p_resolved_by,
        notes = p_notes
    WHERE id = p_error_id;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION cleanup_old_errors(p_days_to_keep INTEGER DEFAULT 30) RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM cache_errors
    WHERE resolved = TRUE
    AND resolved_at < CURRENT_TIMESTAMP - (p_days_to_keep || ' days')::INTERVAL;
    
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

