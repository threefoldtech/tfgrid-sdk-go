## Data Preparation for Testing

Starting the postgres container

```bash
make db-start
```

Populate the database using one of the two available methods:
Generate a mock database:
  - `./schema.sql` is a database schema taken from graphql processor on dev net, it creates all the needed tables for the database.
  - the `data-crafter` have a various of methods to create/update/delete the main tables on the database. it can be customized by the params passed to the `NewCrafter`.
  ```bash
  make db-fill
  ```

Or use a data dump file (contains both schema and data)

```bash
make db-dump p=<path-to-dump.sql-file>
```
