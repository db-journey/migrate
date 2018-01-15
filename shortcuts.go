package migrate

import (
	"context"
	"os"
	"os/signal"

	"github.com/db-journey/migrate/file"
)

// NOTE: funcs below are reconstruction of original package API.
// Presumably should be deprecated and removed later.
// Also, handling os signals is a responsibility of CLI, not this library.

// Up applies all available migrations.
// Up is a shortcut for Handle.Up
func Up(url, migrationsPath string) error {
	m, err := Open(url, migrationsPath)
	if err != nil {
		return err
	}
	defer m.Close()

	return m.Up(newOsInterruptCtx())
}

// Down rolls back all migrations.
// Down is a shortcut for Handle.Down
func Down(url, migrationsPath string) error {
	m, err := Open(url, migrationsPath)
	if err != nil {
		return err
	}
	defer m.Close()
	return m.Down(newOsInterruptCtx())
}

// Redo rolls back the most recently applied migration, then runs it again.
// Redo is a shortcut for Handle.Redo
func Redo(url, migrationsPath string) error {
	m, err := Open(url, migrationsPath)
	if err != nil {
		return err
	}
	defer m.Close()

	return m.Redo(newOsInterruptCtx())
}

// Reset runs the down and up migration function.
// Reset is a shortcut for Handle.Reset
func Reset(url, migrationsPath string) error {
	m, err := Open(url, migrationsPath)
	if err != nil {
		return err
	}
	defer m.Close()

	return m.Reset(newOsInterruptCtx())
}

// Migrate applies relative +n/-n migrations.
// Migrate is a shortcut for Handle.Migrate
func Migrate(url, migrationsPath string, relativeN int) error {
	m, err := Open(url, migrationsPath)
	if err != nil {
		return err
	}
	defer m.Close()

	return m.Migrate(newOsInterruptCtx(), relativeN)
}

// Version returns the current migration version.
// Version is a shortcut for Handle.Version
func Version(url, migrationsPath string) (file.Version, error) {
	m, err := Open(url, migrationsPath)
	if err != nil {
		return 0, err
	}
	defer m.Close()

	return m.Version(newOsInterruptCtx())
}

// Versions returns applied versions.
// Versions is a shortcut for Handle.Versions
func Versions(url, migrationsPath string) (file.Versions, error) {
	m, err := Open(url, migrationsPath)
	if err != nil {
		return nil, err
	}
	defer m.Close()

	return m.Versions(newOsInterruptCtx())
}

// PendingMigrations returns list of pending migration files
// PendingMigrations is a shortcut for Handle.PendingMigrations
func PendingMigrations(url, migrationsPath string) (file.Files, error) {
	m, err := Open(url, migrationsPath)
	if err != nil {
		return nil, err
	}
	defer m.Close()

	return m.PendingMigrations(newOsInterruptCtx())
}

// Create applies relative +n/-n migrations.
// Create is a shortcut for Handle.Create
func Create(url, migrationsPath, name string) (*file.MigrationFile, error) {
	m, err := Open(url, migrationsPath)
	if err != nil {
		return nil, err
	}
	defer m.Close()
	return m.Create(name)
}

// newOsInterruptCtx returns new context that will be cancelled
// on os.Interrupt signal.
func newOsInterruptCtx() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		cancel()
		signal.Stop(c)
	}()
	return ctx
}
