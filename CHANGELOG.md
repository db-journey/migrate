# Migrate Changelog

## 2.0.4 - 2019-11-29

- Use v2 by default


## 2.0.3 - 2019-11-27

- Publish v2 of Go module

## 2.0.2 - 2019-11-04

- Add support for go modules

## 2.0.1 - 2019-11-04

- Fix compilation for go >=1.11 (#9)

## 2.0.1 - 2019-11-04

- Fix compilation for go >=1.11 (#9)

## 2.0.0

All credits go to @josephbuchma

- Removed all "async" stuff.
- Added type for encapsulating migrations functionality for greater flexibility and performance.
- Changed driver registration and initialization approach, fixed #5
   - Removed `Initialize` method from Driver interface
   - Removed `FlenameExtension` from Driver interface
- Added driver.Locker interface, which can be optionally implemented to enable locking during migraitons.
   - Implemented for mysql driver
- Migrate now receives context.Context, and therefore can be cancelled.
- Added option to attach pre/post hooks for migrations.
- Added methods for applying / rolling back specific version.

## 1.5.0

- Add templating support in migration files

## 1.4.3

- Add an `Execute` command to the driver Interface

## 1.4.2

- Split drivers in their own repos

## v1.4.1 - 2016-12-16

* [cassandra] Add [disable_init_host_lookup](https://github.com/gocql/gocql/blob/master/cluster.go#L92) url param (@GeorgeMac / #17)

## v1.4.0 - 2016-11-22

* [crate] Add [Crate](https://crate.io) database support, based on the Crate sql driver by [herenow](https://github.com/herenow/go-crate) (@dereulenspiegel / #16)

## v1.3.2 - 2016-11-11

* [sqlite] Allow multiple statements per migration (dklimkin / #11)

## v1.3.1 - 2016-08-16

* Make MySQL driver aware of SSL certificates for TLS connection by scanning ENV variables (https://github.com/mattes/migrate/pull/117/files)

## v1.3.0 - 2016-08-15

* Initial changelog release
* Timestamp migration, instead of increments (https://github.com/mattes/migrate/issues/102)
* Versions will now be tagged
* Added consistency parameter to cassandra connection string (https://github.com/mattes/migrate/pull/114)
