# Database Setup Documentation

This directory contains SQL scripts that set up a caching system for the TFGrid Explorer database. The system pre-computes and maintains frequently queried data to improve query performance.

## Overview

The setup consists of:
- **Constants**: Configuration values for resource calculations
- **Functions**: Utility functions for price calculations and data transformations
- **Views**: Base views that define the structure of cached data
- **Cache Tables**: Materialized tables storing pre-computed data
- **Triggers**: Automatic cache maintenance when source data changes
- **Error Logging**: Monitoring and debugging of cache operations

## Cached Tables

### `resources_cache`

Pre-computed node resource information for fast queries.

**Key Features:**
- Stores total, free, and used resources (HRU, MRU, SRU, CRU) in bytes
- Includes hardware information (DMI, GPUs, CPU benchmarks, network speeds)
- Contains rental information (renter, rent_contract_id)
- Tracks active contract counts
- Automatically calculates `price_usd` using a generated column

**Resource Reservations:**
- **MRU**: Reserved amount = `max(MRU/10, 2GB)` - automatically reserved and not available for contracts
- **SRU**: Fixed reservation of 20GB - automatically reserved

**Primary Key:** `node_id`
**Indexed on:** `farm_id` for fast farm-based queries

### `public_ips_cache`

Aggregated public IP information per farm.

**Key Features:**
- Tracks total IPs assigned to each farm
- Counts free IPs (where `contract_id = 0`)
- Stores complete IP details as JSONB array (id, ip, contract_id, gateway)

**Primary Key:** `farm_id`

## Triggers

Triggers automatically maintain cache tables when source data changes. Each trigger handles specific data types:

### Node Resources

| Trigger | Source Table | What It Does |
|---------|--------------|--------------|
| `tg_node` | `node` | On INSERT: Populates cache for new node. On DELETE: Removes node from cache. |
| `tg_node_resources_total` | `node_resources_total` | Updates total resources and adjusts free resources (handles MRU reserved amount changes). |
| `tg_contract_resources` | `contract_resources` | Updates used/free resources when contracts change (only processes 'Created'/'GracePeriod' contracts). |
| `tg_node_contract` | `node_contract` | On INSERT: Updates contract count. On UPDATE to 'Deleted': Releases contract resources and updates count. |

### Node Metadata

| Trigger | Source Table | What It Does |
|---------|--------------|--------------|
| `tg_node_gpu_count` | `node_gpu` | Recalculates GPU count and JSON array when GPUs are added/removed/updated. |
| `tg_rent_contract` | `rent_contract` | On INSERT: Sets renter and rent_contract_id. On UPDATE to 'Deleted': Clears rental info. |
| `tg_dmi` | `dmi` | Updates hardware information (bios, baseboard, processor, memory) in cache. |
| `tg_speed` | `speed` | Updates network speed test results (upload, download, IPv4/IPv6, TCP/UDP). |
| `tg_cpu_benchmark` | `cpu_benchmark` | Updates CPU benchmark results (single-threaded, multi-threaded, threads, workloads). |

### Public IPs

| Trigger | Source Table | What It Does |
|---------|--------------|--------------|
| `tg_public_ip` | `public_ip` | Updates free/total IP counts and re-aggregates IP JSON when IPs are inserted/updated/deleted or contract_id changes. |
| `tg_farm` | `farm` | On INSERT: Creates empty cache entry. On DELETE: Removes farm from cache. |

## Error Handling

All triggers include error handling that:
- Logs errors to `cache_errors` table for monitoring
- Raises warnings without failing the source transaction
- Includes context information (record IDs, old/new values) for debugging

## Cache Management

Manual cache refresh functions are available:
- `refresh_resources_cache()` - Refreshes entire resources cache
- `refresh_resources_cache_node(node_id)` - Refreshes a single node
- `refresh_public_ips_cache()` - Refreshes entire IP cache
- `refresh_public_ips_cache_farm(farm_id)` - Refreshes a single farm
- `validate_resources_cache()` - Validates cache consistency

The cache is also automatically refreshed nightly at midnight using `pg_cron`.

