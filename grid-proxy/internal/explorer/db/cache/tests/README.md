# Cache Triggers and Refresh Functions Tests

This directory contains pgTAP SQL tests for all cache triggers and cache refresher functions.

## Prerequisites

1. PostgreSQL with pgTAP extension installed
2. Database schema set up (all setup SQL files from parent directory)

To install pgTAP:
```sql
CREATE EXTENSION IF NOT EXISTS pgtap;
```

## Test Files

- `00_test_helpers.sql` - Helper functions for creating test data
- `01_test_node_trigger.sql` - Tests for `reflect_node_changes` trigger
- `02_test_node_resources_total_trigger.sql` - Tests for `reflect_total_resources_changes` trigger
- `03_test_contract_resources_trigger.sql` - Tests for `reflect_contract_resources_changes` trigger
- `04_test_node_contract_trigger.sql` - Tests for `reflect_node_contract_changes` trigger
- `05_test_node_gpu_trigger.sql` - Tests for `reflect_node_gpu_count_change` trigger
- `06_test_rent_contract_trigger.sql` - Tests for `reflect_rent_contract_changes` trigger
- `07_test_dmi_trigger.sql` - Tests for `reflect_dmi_changes` trigger
- `08_test_speed_trigger.sql` - Tests for `reflect_speed_changes` trigger
- `09_test_cpu_benchmark_trigger.sql` - Tests for `reflect_cpu_benchmark_changes` trigger
- `10_test_public_ip_trigger.sql` - Tests for `reflect_public_ip_changes` trigger
- `11_test_farm_trigger.sql` - Tests for `reflect_farm_changes` trigger
- `12_test_cache_refreshers.sql` - Tests for cache refresh functions

## Running Tests

### Using pg_prove (recommended)

```bash
# Run all tests
pg_prove -d your_database_name -h localhost -U postgres tests/*.sql

# Run specific test file
pg_prove -d your_database_name -h localhost -U postgres tests/01_test_node_trigger.sql
```

### Using psql

```bash
# Run all tests
psql -d your_database_name -h localhost -U postgres -f tests/01_test_node_trigger.sql

# Or run all tests in sequence
for file in tests/*.sql; do
    psql -d your_database_name -h localhost -U postgres -f "$file"
done
```

### Using SQL directly

```sql
-- Load helper functions first
\i tests/00_test_helpers.sql

-- Then run individual test files
\i tests/01_test_node_trigger.sql
```

## Test Structure

Each test file follows this pattern:

1. **BEGIN** - Start transaction for isolation
2. **SELECT plan(N)** - Declare number of tests
3. **Setup** - Create test data using helper functions
4. **Tests** - Fire triggers and verify cache table values
5. **Cleanup** - Remove test data
6. **SELECT * FROM finish()** - Complete pgTAP test run
7. **ROLLBACK** - Rollback transaction

## Test Coverage

### nodex Triggers

- ✅ Node INSERT/DELETE
- ✅ Node resources total INSERT/UPDATE
- ✅ Contract resources INSERT/UPDATE/DELETE
- ✅ Node contract INSERT/UPDATE to Deleted
- ✅ Node GPU INSERT/DELETE/UPDATE (including free_gpu_count)
- ✅ Rent contract INSERT/UPDATE to Deleted
- ✅ DMI INSERT/UPDATE
- ✅ Speed INSERT/UPDATE
- ✅ CPU benchmark INSERT/UPDATE

### farmx Triggers

- ✅ Public IP INSERT/DELETE/UPDATE contract_id
- ✅ Farm INSERT/DELETE

### Cache Refreshers

- ✅ refresh_nodex_node
- ✅ refresh_nodex
- ✅ refresh_farmx_farm
- ✅ refresh_farmx
- ✅ refresh_all

## Notes

- Tests use transactions (BEGIN/ROLLBACK) to ensure isolation
- Helper functions create test data with predictable IDs (node-1001, farm-1001, etc.)
- Tests include `pg_sleep(0.1)` to allow triggers to complete
- All test data is cleaned up using `cleanup_test_data()` function

## Troubleshooting

If tests fail:

1. Ensure pgTAP extension is installed: `CREATE EXTENSION pgtap;`
2. Ensure all setup SQL files have been run
3. Check that base tables exist (node, farm, contract_resources, etc.)
4. Verify triggers are created: `SELECT * FROM pg_trigger WHERE tgname LIKE 'tg_%';`
