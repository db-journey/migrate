# PostgreSQL Driver

[![Build Status](https://travis-ci.org/db-journey/postgresql-driver.svg?branch=master)](https://travis-ci.org/db-journey/postgresql-driver)
[![GoDoc](https://godoc.org/github.com/db-journey/journey?status.svg)](https://godoc.org/github.com/db-journey/journey)

* Runs migrations in transactions.
  That means that if a migration fails, it will be safely rolled back.
* Tries to return helpful error messages.
* Stores migration version details in table ``schema_migrations``.
  This table will be auto-generated.


## Usage

```bash
journey -url postgres://user@host:port/database -path ./db/migrations create add_field_to_table
journey -url postgres://user@host:port/database -path ./db/migrations up
journey help # for more info

## Disable DDL transactions

Some queries, like `alter type ... add value` cannot be executed inside a transaction block.
Since all migrations are executed in a transaction block by default (per migration file), a special option must be specified inside the migration file:

```sql
-- disable_ddl_transaction
alter type ...;
```

The option `disable_ddl_transaction` must be in a sql comment of the first line of the migration file.

Please note that you can't put several `alter type ... add value ...` in a single file. Doing so will result in a `ERROR 25001: ALTER TYPE ... ADD cannot be executed from a function or multi-command string` sql exception during migration.

Since the file will be executed without transaction, it's probably not a good idea to exec more than one statement anyway. If the last statement of the file fails, chances to run again the migration without error will be very limited.

