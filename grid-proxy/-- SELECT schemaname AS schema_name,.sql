





-- CREATE INDEX idx_state_node_contract on node_contract USING gin(state);

-- CREATE UNIQUE INDEX idx_rent_contract_node_id on rent_contract( node_id);
-- CREATE INDEX idx_rent_twin_id ON public.rent_contract(twin_id);
-- CREATE INDEX idx_node_country ON public.node(LOWER(country));
-- CREATE INDEX idx_null_la ON rent_contract ( twin_id ASC NULLS LAST,twin_id);
-- CREATE INDEX idx_rent_contract_contract_id ON public.rent_contract(contract_id);

-- DROP INDEX idx_state_node_contract;
-- DROP INDEX idx_rent_contract_node_id;
-- DROP INDEX idx_rent_twin_id;






-- CREATE INDEX idx_node_id ON node(node_id);
-- CREATE INDEX idx_null_NOTNULL ON public.rent_contract(twin_id) WHERE rent_contract.twin_id IS NULL;
-- DROP INDEX idx_rent_contract_node_id;
-- CREATE INDEX idx2 ON rent_contract ((CASE WHEN rent_contract IS NOT NULL THEN 1 ELSE 2 END));
-- CREATE INDEX node_id_index ON node (node_id);
-- CREATE INDEX idx_rent_contract_twin_id ON rent_contract (twin_id);
-- create index idx_farm_dedicated ON public.farm(dedicated_farm)
-- CREATE INDEX idx_rent_contract_id_twin_id ON public.rent_contract(contract_id,twin_id);
-- CREATE INDEX idx_public_config_node_id ON public_config(node_id);
-- CREATE INDEX idx_farm_farm_id ON farm(farm_id);
-- CREATE INDEX idx_node_location_id ON node(location_id);
-- CREATE INDEX idx_node_twin_id ON node(twin_id);
-- CREATE INDEX idx_node_resources_total_node_id ON public.node_resources_total (node_id);
-- CREATE INDEX idx_contract_resources_contract_id ON public.contract_resources (contract_id);
-- CREATE INDEX idx_node_id ON public.node(node_id);
-- CREATE INDEX idx_farm_id ON public.node(farm_id);
-- CREATE INDEX idx_rent_contrac_twin_id ON public.rent_contract(twin_id);
-- CREATE INDEX Idx_rent_contract_state ON public.rent_contract(state);
-- -- CREATE INDEX idx_farm_farm_id ON public.farm(farm_id);
-- DROP INDEX idx_rent_contract_node_id;
-- DROP INDEX idx_state_node_contract;