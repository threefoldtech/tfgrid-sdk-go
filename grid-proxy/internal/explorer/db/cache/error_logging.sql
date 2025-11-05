-- ============================================================================
-- ERROR LOGGING
-- ============================================================================
-- Error logging table and functions to track cache maintenance errors.
-- This allows monitoring and debugging of trigger failures without losing
-- error information.

/*
 * cache_errors
 * 
 * Table to log errors from cache triggers and maintenance functions.
 * Errors are logged here instead of just raising warnings, allowing
 * for monitoring, alerting, and debugging.
 */
CREATE TABLE IF NOT EXISTS cache_errors (
    id BIGSERIAL PRIMARY KEY,
    error_timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    error_type TEXT NOT NULL,              -- Type of error (trigger name, function name, etc.)
    operation TEXT NOT NULL,              -- Operation that failed (INSERT, UPDATE, DELETE, etc.)
    table_name TEXT,                      -- Table where error occurred
    record_id TEXT,                       -- ID of the record (can be node_id, farm_id, etc.)
    error_message TEXT NOT NULL,          -- Error message from SQLERRM
    error_context JSONB,                  -- Additional context (OLD/NEW values, etc.)
    resolved BOOLEAN DEFAULT FALSE,       -- Whether error has been resolved
    resolved_at TIMESTAMP,
    resolved_by TEXT,
    notes TEXT
);

-- Note: Indexes are created in 04_indexes.sql to maintain proper ordering

/*
 * log_cache_error
 * 
 * Helper function to log errors to cache_errors table.
 * 
 * @param p_error_type - Type of error (e.g., 'reflect_node_changes')
 * @param p_operation - Operation that failed (INSERT, UPDATE, DELETE)
 * @param p_table_name - Table where error occurred
 * @param p_record_id - ID of the record
 * @param p_error_message - Error message
 * @param p_error_context - Additional context as JSONB
 */
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
        -- If logging fails, raise warning (we don't want to fail silently)
        RAISE WARNING 'Failed to log error to cache_errors: %', SQLERRM;
END;
$$ LANGUAGE plpgsql;

/*
 * get_unresolved_errors
 * 
 * Returns count of unresolved errors, optionally filtered by type.
 * 
 * @param p_error_type - Optional filter by error type
 * @returns INTEGER - Count of unresolved errors
 */
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

/*
 * resolve_cache_error
 * 
 * Marks an error as resolved.
 * 
 * @param p_error_id - ID of error to resolve
 * @param p_resolved_by - Who resolved it (optional)
 * @param p_notes - Notes about resolution (optional)
 */
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

/*
 * cleanup_old_errors
 * 
 * Removes resolved errors older than specified days.
 * 
 * @param p_days_to_keep - Number of days to keep resolved errors (default 30)
 */
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

