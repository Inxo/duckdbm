
# DuckDB Migration Tool

A console-based migration tool for managing database schema changes in DuckDB.
It provides commands for initializing the database, creating migrations, applying migrations,
rolling back migrations, listing applied migrations, and importing data from external sources.

## Features

- Initialize the database with a migrations and sync table.
- Create new migration files with an optional rollback section.
- Apply pending migrations to the database with execution time tracking.
- Rollback the last migration or a specified number of migrations.
- List all applied migrations with timestamps and duration.
- [Sync data via migration](doc/duckdb_sync_import_guide.md) — import from MySQL, PostgreSQL, CSV, and more.
- Validate SQL syntax of migration files before applying.
- Progress spinner with elapsed time during sync operations.
- Webhook notifications on apply/sync completion (success or error).
- Support for macros in migration files, substituting environment variables.
- Load environment variables from a `.env` file.

## Requirements

- Go (Golang) installed.
- DuckDB driver: `github.com/duckdb/duckdb-go/v2`.

## Usage

### Build and Run

1. Clone the repository.
2. Run the application with:

   ```bash
   duckdbm [command] [options]
   ```

### Commands

#### 1. Initialize the Database

Creates the `migrations` and `sync` tables in the specified database file.

```bash
duckdbm -db=your_database.db init
```

#### 2. Create a Migration

Generates a new migration file in the `migrations` directory.

```bash
duckdbm -db=your_database.db create migration_name
```

Example:
```bash
duckdbm -db=your_database.db create add_users_table
```

The file `migrations/001_add_users_table.sql` will be created.

#### 3. Apply Migrations

Applies all pending migrations to the database. Execution time is recorded for each migration.

```bash
duckdbm -db=your_database.db apply
```

Output example:
```
Migration applied: 001_add_users_table.sql (12ms)
Migration applied: 002_add_orders_table.sql (8ms)
```

#### 4. Rollback Migrations

Rolls back the last applied migration or a specified number of migrations.

Rollback the last migration:
```bash
duckdbm -db=your_database.db rollback
```

Rollback the last 3 migrations:
```bash
duckdbm -db=your_database.db rollback 3
```

#### 5. List Applied Migrations

Displays applied migrations with timestamps and execution duration.

```bash
duckdbm -db=your_database.db list
```

Output example:
```
Applied migrations:
ID  Filename                    Applied At            Duration
----------------------------------------------------------------
2   002_add_orders_table.sql    2025-05-24 10:01:00   8ms
1   001_add_users_table.sql     2025-05-24 10:00:00   12ms
```

List the `sync` table instead:
```bash
duckdbm -db=your_database.db list sync
```

Limit results:
```bash
duckdbm -db=your_database.db list migrations 20
```

#### 6. Validate Migrations

Validates SQL syntax of all migration files without modifying the database.
Useful in CI pipelines to catch errors before deployment.

```bash
duckdbm validate
```

Validate a specific file (by partial name match):
```bash
duckdbm validate 003_import_orders
```

Output example:
```
Validating migrations...
  ✓ 001_add_users_table.sql
  ✓ 002_add_orders_table.sql
  ✗ 003_import_orders.sql — syntax error: ...

Validation failed.
```

Exits with code `1` if any file is invalid — CI-friendly.

#### 7. Sync a Migration

The `sync` command applies a specific migration without recording it in the `migrations` table.
Instead, the execution is recorded in a separate `sync` table.

Use this to import data from external sources on a schedule.
A progress spinner with elapsed time is shown during execution.

```bash
duckdbm -db=your_database.db sync migration_name
```

Example:
```bash
duckdbm -db=your_database.db sync 002_sync_users
```

Output example:
```
⠼ Syncing 002_sync_users... (3.2s)
✓ Successfully synced: 002_sync_users (5.841s)
```

Example migration to sync users from MySQL `002_sync_users.sql`:
```sql
-- MIGRATE
INSTALL mysql;
LOAD mysql;
CREATE SECRET IF NOT EXISTS (
    TYPE MYSQL,
    HOST '{{MYSQL_HOST}}',
    PORT 3306,
    DATABASE {{MYSQL_DB}},
    USER '{{MYSQL_USER}}',
    PASSWORD '{{MYSQL_PASSWORD}}'
);
ATTACH IF NOT EXISTS 'database={{MYSQL_DB}}' AS mysql_db (TYPE MYSQL);

INSERT OR REPLACE INTO users
SELECT * FROM mysql_db.default.users;

-- ROLLBACK
TRUNCATE TABLE users;
```

### Migration Files

#### File Format

Migration files are `.sql` files located in the `migrations` directory.
Each file has two sections separated by `-- ROLLBACK`:

```sql
-- MIGRATE
-- SQL statements to apply the migration

-- ROLLBACK
-- SQL statements to undo the migration
```

#### Using Macros

Migration files can include macros in the format `{{ENV_VAR}}`.
These are replaced with environment variable values at runtime.

```sql
-- MIGRATE
CREATE TABLE {{TABLE_NAME}} (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL
);

-- ROLLBACK
DROP TABLE {{TABLE_NAME}};
```

If `TABLE_NAME=users` is set, `{{TABLE_NAME}}` is replaced with `users`.

### Environment Variables

| Variable       | Description                                          |
|----------------|------------------------------------------------------|
| `DATABASE`     | Database file path (alternative to `-db` flag)       |
| `ENC_KEY`      | Encryption key for DuckDB encrypted databases        |
| `WEBHOOK_URL`  | HTTP endpoint for completion notifications (optional)|

#### Webhook Notifications

Set `WEBHOOK_URL` to receive an HTTP POST after each `apply` or `sync` completes.

```env
WEBHOOK_URL=https://hooks.slack.com/services/xxx
```

Payload format:
```json
{
  "event":       "sync",
  "status":      "success",
  "name":        "002_sync_users",
  "duration_ms": 5841,
  "timestamp":   "2025-05-24T10:00:00Z",
  "error":       ""
}
```

- `event`: `"apply"` or `"sync"`
- `status`: `"success"` or `"error"`
- Webhook failures are warnings only — they never fail the main operation.
- Timeout: 5 seconds.

Works with any HTTP endpoint: Slack incoming webhooks, [Healthchecks.io](https://healthchecks.io), custom APIs.

### Using a `.env` File

```env
DATABASE=mydata.db
TABLE_NAME=users
MYSQL_HOST=db.example.com
MYSQL_DB=production
MYSQL_USER=reader
MYSQL_PASSWORD=secret
WEBHOOK_URL=https://hooks.example.com/notify
```

System environment variables take precedence over `.env` values.
If a macro refers to an undefined variable, it is replaced with an empty string.

### Directory Structure

```
.
├── duckdbm
├── .env
├── migrations/
│   ├── 001_add_users_table.sql
│   ├── 002_sync_users.sql
│   └── ...
```

## License

This project is licensed under the MIT License.
