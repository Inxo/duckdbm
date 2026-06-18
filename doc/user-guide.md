# duckdbm User Guide

duckdbm is a command-line migration tool for [DuckDB](https://duckdb.org). It manages schema evolution through versioned SQL files and supports scheduled data synchronization from external sources.

---

## Table of Contents

1. [Installation](#installation)
2. [Quick Start](#quick-start)
3. [Configuration](#configuration)
4. [Commands](#commands)
   - [init](#init)
   - [create](#create)
   - [apply](#apply)
   - [rollback](#rollback)
   - [list](#list)
   - [validate](#validate)
   - [sync](#sync)
5. [Migration Files](#migration-files)
6. [Macros (Environment Variable Substitution)](#macros)
7. [Webhook Notifications](#webhook-notifications)
8. [Data Sync Pattern](#data-sync-pattern)
9. [Docker](#docker)
10. [CI/CD Integration](#cicd-integration)

---

## Installation

### Build from Source

Requirements: Go 1.24+ with CGO enabled.

```bash
git clone https://github.com/inxo/duckdbm.git
cd duckdbm
make build
```

The binary is placed at `build/duckdbm`. Add it to your `PATH`:

```bash
sudo cp build/duckdbm /usr/local/bin/duckdbm
```

### Platform-specific builds

```bash
make build-linux    # Linux amd64
make build          # Current platform
```

---

## Quick Start

```bash
# 1. Initialize the database
duckdbm -db=mydata.db init

# 2. Create your first migration
duckdbm -db=mydata.db create create_users_table

# 3. Edit the generated file
vim migrations/001_create_users_table.sql

# 4. Validate SQL syntax
duckdbm validate

# 5. Apply migrations
duckdbm -db=mydata.db apply

# 6. Check what was applied
duckdbm -db=mydata.db list
```

---

## Configuration

### Command-line flag

| Flag | Description | Default |
|------|-------------|---------|
| `-db=<path>` | Path to the DuckDB database file | `duckdb` |

### Environment Variables

| Variable | Description |
|----------|-------------|
| `DATABASE` | Database file path — alternative to `-db` |
| `ENC_KEY` | Encryption key for encrypted DuckDB databases |
| `WEBHOOK_URL` | HTTP endpoint for completion notifications |

Any additional variables you define are available as macros in migration files (see [Macros](#macros)).

### .env File

duckdbm automatically loads a `.env` file from the current directory:

```env
DATABASE=mydata.db
ENC_KEY=my_secret_key
WEBHOOK_URL=https://hooks.example.com/notify

# Custom variables used as macros in SQL files
MYSQL_HOST=db.example.com
MYSQL_DB=production
MYSQL_USER=reader
MYSQL_PASSWORD=secret
TABLE_NAME=users
```

System environment variables take precedence over `.env` values.

---

## Commands

### init

Creates the `migrations` and `sync` tracking tables in the database.

```bash
duckdbm -db=mydata.db init
```

Run this once before using any other commands. `apply` and `rollback` will call `init` automatically if needed.

---

### create

Generates a new numbered migration file in the `migrations/` directory.

```bash
duckdbm -db=mydata.db create <migration_name>
```

**Example:**

```bash
duckdbm -db=mydata.db create add_orders_table
# Creates: migrations/002_add_orders_table.sql
```

Files are numbered sequentially (`001_`, `002_`, …) based on the count of existing files.

The generated file contains a ready-to-fill template:

```sql
-- MIGRATE
-- Write your forward migration SQL here

-- ROLLBACK
-- Write your rollback SQL here
```

---

### apply

Applies all pending migrations in alphabetical order.

```bash
duckdbm -db=mydata.db apply
```

**Output:**

```
Migration applied: 001_create_users_table.sql (12ms)
Migration applied: 002_add_orders_table.sql (8ms)
```

- Only migrations not yet recorded in the `migrations` table are applied.
- Each migration is recorded with a timestamp and duration.
- Stops on the first error.

---

### rollback

Reverts the last applied migration, or a specified number of migrations.

```bash
# Rollback the last migration
duckdbm -db=mydata.db rollback

# Rollback the last 3 migrations
duckdbm -db=mydata.db rollback 3
```

The `-- ROLLBACK` section of each migration file is executed. The corresponding row is removed from the `migrations` table on success.

---

### list

Displays applied migrations with timestamps and execution duration.

```bash
# Show last 10 applied migrations (default)
duckdbm -db=mydata.db list

# Show the sync table instead
duckdbm -db=mydata.db list sync

# Show last 50 migrations
duckdbm -db=mydata.db list migrations 50
```

**Output:**

```
Applied migrations:
ID   Filename                        Applied At             Duration
----------------------------------------------------------------------
2    002_add_orders_table.sql        2025-05-24 10:01:00    8ms
1    001_create_users_table.sql      2025-05-24 10:00:00    12ms
```

---

### validate

Checks SQL syntax of migration files without executing them.

```bash
# Validate all migration files
duckdbm validate

# Validate files matching a pattern
duckdbm validate 003_import
```

**Output:**

```
Validating migrations...
  ✓ 001_create_users_table.sql
  ✓ 002_add_orders_table.sql
  ✗ 003_import_orders.sql — syntax error near "SELEC"

Validation failed.
```

- Uses DuckDB's `EXPLAIN` statement internally — no data is modified.
- Exits with code `1` on failure, making it suitable for CI pipelines.
- Does not require a `-db` flag.

---

### sync

Applies a migration file as a data synchronization operation. Execution is recorded in the `sync` table, **not** in the `migrations` table, so it can be run repeatedly.

```bash
duckdbm -db=mydata.db sync <migration_name>
```

**Example:**

```bash
duckdbm -db=mydata.db sync 002_sync_users
```

**Output:**

```
⠼ Syncing 002_sync_users... (3.2s)
✓ Successfully synced: 002_sync_users (5.841s)
```

A progress spinner with elapsed time is shown during execution. Use `sync` for scheduled data imports (e.g., via cron).

---

## Migration Files

### Location

All migration files live in the `migrations/` directory next to the binary or in the current working directory.

```
.
├── duckdbm
├── .env
└── migrations/
    ├── 001_create_users_table.sql
    ├── 002_add_orders_table.sql
    └── 003_sync_users.sql
```

### File Format

Each file has two labeled sections:

```sql
-- MIGRATE
CREATE TABLE users (
    id   INTEGER PRIMARY KEY,
    name TEXT    NOT NULL,
    email TEXT   UNIQUE
);

-- ROLLBACK
DROP TABLE users;
```

- **`-- MIGRATE`** — SQL executed by `apply` and `sync`.
- **`-- ROLLBACK`** — SQL executed by `rollback`.

Both sections are required. If rollback is not applicable, leave the section empty or add a comment.

### Ordering

Migrations are applied in **alphabetical order** by filename. The numeric prefix (`001_`, `002_`, …) enforces the correct sequence. Never rename applied migration files.

---

## Macros

Migration files support `{{VAR_NAME}}` placeholders that are replaced with environment variable values at runtime.

**Migration file:**

```sql
-- MIGRATE
CREATE TABLE {{TABLE_NAME}} (
    id   INTEGER PRIMARY KEY,
    name TEXT NOT NULL
);

-- ROLLBACK
DROP TABLE {{TABLE_NAME}};
```

**`.env` file:**

```env
TABLE_NAME=users
```

At runtime `{{TABLE_NAME}}` is replaced with `users`.

**Rules:**

- Pattern: `{{[A-Z0-9_]+}}` (uppercase letters, digits, underscores).
- Values come from `.env` or system environment variables.
- System variables override `.env` values.
- If a variable is not defined, it is replaced with an empty string and a warning is printed.

**Practical use — connecting to MySQL:**

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

---

## Webhook Notifications

Set `WEBHOOK_URL` to receive an HTTP POST notification after each `apply` or `sync` completes.

```env
WEBHOOK_URL=https://hooks.slack.com/services/T00000000/B00000000/XXXXXXXX
```

### Payload

```json
{
  "event":       "apply",
  "status":      "success",
  "name":        "002_add_orders_table",
  "duration_ms": 8,
  "timestamp":   "2025-05-24T10:01:00Z",
  "error":       ""
}
```

| Field | Values |
|-------|--------|
| `event` | `"apply"` or `"sync"` |
| `status` | `"success"` or `"error"` |
| `error` | Error message, or empty string on success |

**Behavior:**

- Timeout: 5 seconds.
- Webhook failures are non-fatal — they print a warning and do not affect the exit code.
- Works with Slack incoming webhooks, [Healthchecks.io](https://healthchecks.io), PagerDuty, or any custom HTTP endpoint.

---

## Data Sync Pattern

`sync` is designed for importing data from external sources on a schedule. Unlike `apply`, the same sync file can be run multiple times without polluting the migration history.

### Incremental Import from MySQL

The following example imports only rows added since the last sync run by tracking a timestamp checkpoint:

```sql
-- MIGRATE
INSTALL mysql;
LOAD mysql;
INSTALL mysql_scanner;
LOAD mysql_scanner;

ATTACH IF NOT EXISTS 'database={{MYSQL_DB}}' AS mysql_db (TYPE MYSQL);

-- Calculate checkpoint: 1 day before the latest local record, or epoch if empty
SET VARIABLE prime = (
  SELECT EPOCH(
    IFNULL(
      (SELECT streams.curr_time::TIMESTAMP - INTERVAL 1 DAY
       FROM streams
       ORDER BY streams.curr_time DESC
       LIMIT 1),
      'epoch'::TIMESTAMP
    )
  )
);

-- Import only new rows
INSERT OR REPLACE INTO streams
SELECT *
FROM mysql_query(
  'mysql_db',
  CONCAT('SELECT * FROM streams WHERE curr_time > FROM_UNIXTIME(', GETVARIABLE('prime'), ')')
);

-- ROLLBACK
TRUNCATE TABLE streams;
```

**Running on a schedule (cron):**

```cron
# Every hour
0 * * * * cd /app && duckdbm -db=analytics.db sync 001_sync_streams
```

### Other Sources

DuckDB supports many external sources via extensions. The same pattern works for:

| Source | Extension |
|--------|-----------|
| MySQL | `mysql` / `mysql_scanner` |
| PostgreSQL | `postgres` |
| SQLite | `sqlite` |
| CSV / Parquet / JSON | Built-in |
| S3 / GCS | `httpfs` |

---

## Docker

A `Dockerfile` and `docker-compose.yml` are included.

**Build the image:**

```bash
docker-compose build
```

**Run commands in a container:**

```bash
docker-compose run duckdbm duckdbm -db=/data/analytics.db init
docker-compose run duckdbm duckdbm -db=/data/analytics.db apply
docker-compose run duckdbm duckdbm -db=/data/analytics.db sync 001_sync_users
```

Mount your data directory and migrations folder as volumes in `docker-compose.yml` to persist the database and access migration files.

---

## CI/CD Integration

### Validate on every push

Add a validation step to catch SQL syntax errors before merging:

**GitHub Actions:**

```yaml
- name: Validate migrations
  run: duckdbm validate
```

**GitLab CI:**

```yaml
validate:
  stage: test
  script:
    - duckdbm validate
```

`validate` exits with code `1` on any syntax error, which fails the pipeline.

### Apply on deploy

```bash
duckdbm -db=$DATABASE apply
```

Set `DATABASE` (or `ENC_KEY`, `WEBHOOK_URL`) as CI/CD secrets/environment variables — duckdbm reads them automatically.

### Full deploy workflow example

```bash
#!/bin/bash
set -e

# Validate first — fail fast
duckdbm validate

# Apply schema migrations
duckdbm -db="$DATABASE" apply

echo "Migrations applied successfully."
```

---

## Internal Tables

duckdbm maintains two tables in the database, created by `init`:

### migrations

Tracks applied schema migrations. Each filename must be unique.

| Column | Type | Description |
|--------|------|-------------|
| `id` | INTEGER | Auto-increment primary key |
| `filename` | TEXT | Migration filename (unique) |
| `applied_at` | TIMESTAMP | When the migration was applied |
| `duration_ms` | INTEGER | Execution time in milliseconds |

### sync

Tracks data sync executions. The same filename can appear multiple times.

| Column | Type | Description |
|--------|------|-------------|
| `id` | INTEGER | Auto-increment primary key |
| `filename` | TEXT | Migration filename |
| `applied_at` | TIMESTAMP | When the sync ran |
| `duration_ms` | INTEGER | Execution time in milliseconds |
