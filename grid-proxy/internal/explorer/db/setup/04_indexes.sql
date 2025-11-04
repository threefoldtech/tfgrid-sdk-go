-- ============================================================================
-- INDEXES
-- ============================================================================
-- Performance indexes to optimize cache table queries

-- Required PostgreSQL extensions for index types
CREATE EXTENSION IF NOT EXISTS pg_trgm;  -- For trigram similarity search
CREATE EXTENSION IF NOT EXISTS btree_gin; -- For GIN indexes on scalar values

-- Indexes on source tables (used by views and triggers)
CREATE INDEX IF NOT EXISTS idx_node_id ON public.node(node_id);
CREATE INDEX IF NOT EXISTS idx_twin_id ON public.twin(twin_id);
CREATE INDEX IF NOT EXISTS idx_farm_id ON public.farm(farm_id);
-- GIN indexes for UUID/string lookups (faster than B-tree for large datasets)
CREATE INDEX IF NOT EXISTS idx_node_contract_id ON public.node_contract USING gin(id);
CREATE INDEX IF NOT EXISTS idx_name_contract_id ON public.name_contract USING gin(id);
CREATE INDEX IF NOT EXISTS idx_rent_contract_id ON public.rent_contract USING gin(id);

-- Indexes on cache tables (for fast queries)
CREATE INDEX IF NOT EXISTS idx_resources_cache_farm_id ON resources_cache(farm_id);
CREATE INDEX IF NOT EXISTS idx_resources_cache_node_id ON resources_cache(node_id);
CREATE INDEX IF NOT EXISTS idx_public_ips_cache_farm_id ON public_ips_cache(farm_id);

-- Additional indexes for indexer tables
CREATE INDEX IF NOT EXISTS idx_location_id ON location USING gin(id);
CREATE INDEX IF NOT EXISTS idx_public_config_node_id ON public_config USING gin(node_id);

