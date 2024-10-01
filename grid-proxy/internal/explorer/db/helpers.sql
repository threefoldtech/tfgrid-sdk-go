

-- by the time, updated_at gets outdated which break some functionalities
-- that depends on node status. this help update the nodes that
-- got updated in the last period to now.
CREATE OR REPLACE FUNCTION update_node_uptimes()
RETURNS void AS $$
DECLARE
    last_updated_at INT;
BEGIN
    -- Step 1: Get the latest uptime report's updated_at timestamp
    SELECT updated_at
    INTO last_updated_at
    FROM node
    ORDER BY updated_at DESC
    LIMIT 1;

    -- Step 2: Update nodes where updated_at is > last_updated_at - 39 minutes
    UPDATE node
    SET updated_at = CAST(EXTRACT(epoch FROM NOW()) AS INT)
    WHERE updated_at > last_updated_at - 2340;
END;
$$ LANGUAGE plpgsql;
