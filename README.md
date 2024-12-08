
# DuckDB Migration Tool

A console-based migration tool for managing database schema changes in DuckDB. 
It provides commands for initializing the database, creating migrations, applying migrations, 
rolling back migrations, and listing applied migrations.

## Features

- Initialize the database with a migrations table.
- Create new migration files with an optional rollback section.
- Apply pending migrations to the database.
- Rollback the last applied migration.
- List all applied migrations with timestamps.

## Requirements

- Go (Golang) installed.
- DuckDB installed and accessible via the `github.com/marcboeker/go-duckdb` driver.

## Usage

### Build and Run

1. Clone the repository.
2. Run the application with:

   ```bash
   go run main.go [command] [options]
   ```

### Commands

#### 1. Initialize the Database
Creates the migrations table in the specified database file.

```bash
go run main.go -db=your_database.db init
```

#### 2. Create a Migration
Generates a new migration file in the `migrations` directory.

```bash
go run main.go -db=your_database.db create migration_name
```

Example:
```bash
go run main.go -db=your_database.db create add_users_table
```

The file `migrations/001_add_users_table.sql` will be created.

#### 3. Apply Migrations
Applies all pending migrations in the `migrations` directory.

```bash
go run main.go -db=your_database.db apply
```

#### 4. Rollback the Last Migration
Rolls back the last applied migration.

```bash
go run main.go -db=your_database.db rollback
```

#### 5. List Applied Migrations
Displays all applied migrations.

```bash
go run main.go -db=your_database.db list
```

### Migration File Structure

Migration files are `.sql` files located in the `migrations` directory. 
Each file can include a `-- ROLLBACK` section for rollback support.

Example:
```sql
-- SQL migration
CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL
);

-- ROLLBACK
DROP TABLE users;
```

### Directory Structure

```
.
├── main.go
├── migrations/
│   ├── 001_add_users_table.sql
│   └── ...
```

## License

This project is licensed under the MIT License.
