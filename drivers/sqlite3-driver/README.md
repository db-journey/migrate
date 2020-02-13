# Sqlite3 Driver

* Runs migrations in transactions.
  That means that if a migration fails, it will be safely rolled back.
* Tries to return helpful error messages.
* Stores migration version details in table ``schema_migrations``.
  This table will be auto-generated.


## Usage

```bash
journey -url sqlite3://database.sqlite -path ./db/migrations create add_field_to_table
journey -url sqlite3://database.sqlite -path ./db/migrations up
journey help # for more info
```

## Authors

* Matthias Kadenbach, https://github.com/mattes
* Caesar Wirth, https://github.com/cjwirth
