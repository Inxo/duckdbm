
# 🔁 Efficient Data Import into DuckDB Using SYNC SQL Files

DuckDB is a high-performance, embedded analytical database that's ideal for processing data locally. One convenient pattern for using DuckDB is the **automated import of data from external sources** via `SYNC` migration files.

[duckdbm](https://github.com/inxo/duckdbm) is a tool for migrating and synchronizing data in a DuckDB database.

In this article, we’ll explore how to use SQL files with the `SYNC` command to regularly import data from MySQL into DuckDB for analytical purposes.

---

## 🧩 What is SYNC?

`SYNC` is a special command in the migration tool that allows you to apply an SQL file like a normal migration, but **without recording it in the main `migrations` table**. Instead, execution records are stored in the `sync` table. This is useful when you need to **regularly re-run** migrations (e.g., updating data from external sources) without affecting the linear migration history.

---

## 📂 Example SYNC File for Importing Data from MySQL

```sql
-- SYNC
INSTALL mysql;
LOAD mysql;
ATTACH IF NOT EXISTS 'database=main' AS mysql_db (TYPE MYSQL);

INSTALL mysql_scanner;
LOAD mysql_scanner;

SET VARIABLE prime = (
  SELECT EPOCH(
    IFNULL(
      (SELECT (streams.curr_time::TIMESTAMP - INTERVAL 1 DAY)
       FROM streams
       ORDER BY streams.curr_time DESC
       LIMIT 1),
      'epoch'::TIMESTAMP
    )
  )
);
SELECT GETVARIABLE('prime');

INSERT OR REPLACE INTO streams
SELECT *
FROM mysql_query(
  'mysql_db',
  CONCAT('SELECT * FROM streams m WHERE m.curr_time > FROM_UNIXTIME(', GETVARIABLE('prime'), ');')
);
-- ROLLBACK
-- TRUNCATE TABLE streams;
```

---

## 🔍 Explanation of the SQL File

- `INSTALL` and `LOAD`: load the `mysql` and `mysql_scanner` extensions for accessing a MySQL database.
- `ATTACH`: create a connection to MySQL as a virtual database `mysql_db`.
- `SET VARIABLE prime`: define a timestamp `prime` as 1 day before the latest record in the `streams` table. If the table is empty, use the beginning of the epoch.
- `mysql_query(...)`: run a query in MySQL to select only records with `curr_time > prime`.
- `INSERT OR REPLACE INTO streams`: insert or update records in the local DuckDB table.
- `-- ROLLBACK`: a manual cleanup instruction, usually not needed for sync operations.

---

## 🧪 How to Use

Assuming you have the migration tool ([duckdbm](https://github.com/inxo/duckdbm)) with support for the `sync` command:

```bash
duckdbm sync 1689500000_import_streams_from_mysql
```

Each time you run it:
- It connects to the MySQL source.
- Calculates the sync checkpoint.
- Imports only new data.
- Records the execution in the `sync` table, not in `migrations`.

---

## 📈 Analytics in DuckDB

After data is imported, you can run any SQL queries for analysis:

```sql
SELECT date_trunc('day', curr_time) AS day, COUNT(*) AS event_count
FROM streams
GROUP BY 1
ORDER BY 1;
```

DuckDB provides fast processing even for large datasets, and you can save results or export them as Parquet, CSV, or Excel.

---

## ✅ Advantages of This Approach

- **Safety**: sync files don’t interfere with migration history.
- **Repeatability**: can be run multiple times without duplicating data.
- **Flexibility**: works with different sources (MySQL, Postgres, CSV, etc.).
- **Convenience**: declarative approach without the need for custom scripts.

---

## 📌 Conclusion

Using `SYNC` files with environment variables and external database integration is a powerful way to automate data loading into DuckDB for analytics. This approach is especially useful for BI tasks, reporting, and local development of analytical pipelines.

Ready to connect your sources? Just write a `SYNC` file and run it using `duckdbm`.
