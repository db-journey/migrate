package migrate

import "github.com/db-journey/migrate/file"

// Up applies all available migrations.
// Up is a shortcut for Migrate.Up
func Up(url, migrationsPath string) error {
	m, err := New(url, migrationsPath)
	if err != nil {
		return err
	}
	return m.Up()
}

// Down rolls back all migrations.
// Down is a shortcut for Migrate.Down
func Down(url, migrationsPath string) error {
	m, err := New(url, migrationsPath)
	if err != nil {
		return err
	}
	return m.Down()
}

// Redo rolls back the most recently applied migration, then runs it again.
// Redo is a shortcut for Migrate.Redo
func Redo(url, migrationsPath string) error {
	m, err := New(url, migrationsPath)
	if err != nil {
		return err
	}
	return m.Redo()
}

// Reset runs the down and up migration function.
// Reset is a shortcut for Migrate.Reset
func Reset(url, migrationsPath string) error {
	m, err := New(url, migrationsPath)
	if err != nil {
		return err
	}
	return m.Reset()
}

// Migrate applies relative +n/-n migrations.
// Migrate is a shortcut for Migrate.Migrate
func Migrate(url, migrationsPath string, relativeN int) error {
	m, err := New(url, migrationsPath)
	if err != nil {
		return err
	}
	return m.Migrate(relativeN)
}

// Version returns the current migration version.
// Version is a shortcut for Version.Version
func Version(url, migrationsPath string) (file.Version, error) {
	m, err := New(url, migrationsPath)
	if err != nil {
		return 0, err
	}
	return m.Version()
}

// Versions returns applied versions.
// Versions is a shortcut for Versions.Versions
func Versions(url, migrationsPath string) (file.Versions, error) {
	m, err := New(url, migrationsPath)
	if err != nil {
		return nil, err
	}
	return m.Versions()
}

// PendingMigrations returns list of pending migration files
// PendingMigrations is a shortcut for PendingMigrations.PendingMigrations
func PendingMigrations(url, migrationsPath string) (file.Files, error) {
	m, err := New(url, migrationsPath)
	if err != nil {
		return nil, err
	}
	return m.PendingMigrations()
}

// Create applies relative +n/-n migrations.
// Create is a shortcut for Create.Create
func Create(url, migrationsPath, name string) (*file.MigrationFile, error) {
	m, err := New(url, migrationsPath)
	if err != nil {
		return nil, err
	}
	return m.Create(name)
}
