
# DuckDB Migration Tool

A console-based migration tool for managing database schema changes in DuckDB. 
It provides commands for initializing the database, creating migrations, applying migrations, 
rolling back migrations, and listing applied migrations.

## Features

- Initialize the database with a migrations table.
- Create new migration files with an optional rollback section.
- Apply pending migrations to the database.
- Rollback the last migration or a specified number of migrations.
- List all applied migrations with timestamps.
- Sync data via migration. 
- Support for macros in migration files, substituting environment variables.
- Load environment variables from a `.env` file.

## Requirements

- Go (Golang) installed.
- DuckDB installed and accessible via the `github.com/marcboeker/go-duckdb` driver.

## Usage

### Build and Run

1. Clone the repository.
2. Run the application with:

   ```bash
   duckdbm [command] [options]
   ```

### Commands

#### 1. Initialize the Database
Creates the migrations table in the specified database file.

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
Applies all pending migrations to the database.

```bash
duckdbm -db=your_database.db apply
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
Displays all applied migrations.

```bash
duckdbm -db=your_database.db list
```

#### 6. Sync a Migration

The `sync` command applies a specific migration without recording it in the `migrations` table. Instead, the migration's application is recorded in a separate `sync` table.

"Possibility to update data from other sources/databases using the migration mechanism. Synchronization can be scheduled.

```bash
duckdbm -db=your_database.db sync migration_name
```

Example:
```bash
duckdbm -db=your_database.db sync 002_sync_users
```

Example migration to sync users from Mysql `002_sync_users.sql`:
```sql
-- MIGRATE
INSTALL mysql;
LOAD mysql;
CREATE
SECRET IF NOT EXISTS
( TYPE MYSQL,
    HOST '{{MYSQL_HOST}}',
    PORT 3306,
    DATABASE {{MYSQL_DB}},
    USER '{{MYSQL_USER}}',
    PASSWORD '{{MYSQL_PASSWORD}}');
ATTACH IF NOT EXISTS 'database={{MYSQL_DB}}' AS mysql_db (TYPE MYSQL);

INSERT OR
REPLACE INTO users
SELECT *
FROM mysql_db.default.users;

-- ROLLBACK
TRUNCATE TABLE users;
```

Also, this is useful for re-applying a migration without affecting the migration history.

### Migration Files
#### Using Macros in Migration Files

Migration files are `.sql` files located in the `migrations` directory. Can include macros in the format `{{ENV_VAR}}`.
These macros will be replaced with the values of the corresponding environment variables at runtime.

Each file can include a `-- ROLLBACK` section for rollback support.

#### Example Migration File with Macros

```sql
-- MIGRATE
CREATE TABLE {{TABLE_NAME}} (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL
);

-- ROLLBACK
DROP TABLE {{TABLE_NAME}};
```

If the environment variable `TABLE_NAME` is set to `users`, the macro `{{TABLE_NAME}}` will be replaced with `users`.

#### Setting Environment Variables

You can define environment variables directly in the terminal or use a `.env` file.

### Using a `.env` File

The tool supports loading environment variables from a `.env` file located in the root directory of your project.

#### Example `.env` File

```env
TABLE_NAME=users
COLUMN_NAME=username
```

When the tool runs, it will load the `.env` file and use the defined variables to replace macros in migration files.

#### Precedence of Variables

- Environment variables set in the system take precedence over `.env` variables.
- If a macro refers to an undefined variable, it will be replaced with an empty string.

### Example Workflow

1. **Set Up Your `.env` File**:
   ```env
   TABLE_NAME=users
   ```

2. **Run the Migration Tool**:
   ```bash
   duckdbm -db=your_database.db apply
   ```

The `{{TABLE_NAME}}` macro in migration files will now be replaced with the value from the `.env` file (`users`).

### Directory Structure

```
.
├── duckdbm
├── .env
├── migrations/
│   ├── 001_add_users_table.sql
│   ├── 003_sync_users.sql
│   └── ...
```

## License

This project is licensed under the MIT License.
