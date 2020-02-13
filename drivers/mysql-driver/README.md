# MySQL Driver

* Runs migrations in transactions.
  That means that if a migration fails, it will be safely rolled back.
* Tries to return helpful error messages.
* Stores migration version details in table `schema_migrations`.
  This table will be auto-generated.
* Safe to run concurrently (`schema_migrations` table is locked during migrations)

## Migrations SQL formatting

Each SQL statement MUST end with semicolon (;) FOLLOWED BY NEWLINE !
Whole migration will be executed inside transaction by default.
Place SQL between "-- TXBEGIN" and "-- TXEND" comments for custom transaction:
  - you CAN have multiple separate transactions in single migration
  - any SQL not wrapped into TXBEGIN - TXEND will be executed without transaction.
Add "-- NOTX" comment above all SQL to disable default transaction. NOTE:
  it's redundant when TXBEGIN/TXEND is used.

## Usage

```bash
migrate -url mysql://user@tcp(host:port)/database -path ./db/migrations create add_field_to_table
migrate -url mysql://user@tcp(host:port)/database -path ./db/migrations up
migrate help # for more info
```

See full [DSN (Data Source Name) documentation](https://github.com/go-sql-driver/mysql/#dsn-data-source-name).

### SSL

The MySQL driver will set a TLS config if the following env variables are set:

- `MYSQL_SERVER_CA`
- `MYSQL_CLIENT_KEY`
- `MYSQL_CLIENT_CERT`

__TODO: deprecate - library code should not rely on environment variables__


## Authors

* Matthias Kadenbach, https://github.com/mattes
* Joseph Buchma, https://github.com/josephbuchma
