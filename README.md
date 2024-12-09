
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
- Support for macros in migration files, substituting environment variables.

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
Applies all pending migrations in the `migrations` directory.

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

### Using Macros in Migration Files

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

Set the required environment variables before running the migration tool:

```bash
export TABLE_NAME=users
```

Run the migration command:

```bash
go run main.go -db=your_database.db apply
```

#### Behavior with Undefined Macros

If a macro refers to an undefined environment variable, it will be replaced with an empty string. 
A warning will be printed to the console.

### Directory Structure

```
.
├── duckdbm
├── migrations/
│   ├── 001_add_users_table.sql
│   └── ...
```

## License

This project is licensed under the MIT License.
