-- Debugging function: updates node uptimes

CREATE OR REPLACE FUNCTION update_node_uptimes()
RETURNS void AS $$
DECLARE
    last_updated_at INT;
BEGIN
    SELECT updated_at
    INTO last_updated_at
    FROM node
    ORDER BY updated_at DESC
    LIMIT 1;

    UPDATE node
    SET updated_at = CAST(EXTRACT(epoch FROM NOW()) AS INT)
    WHERE updated_at > last_updated_at - 2340;
END;
$$ LANGUAGE plpgsql;
