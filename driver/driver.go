// Package driver holds the driver interface.
package driver

import (
	"fmt"
	"reflect"
	"regexp"

	"github.com/db-journey/migrate/file"
)

// Driver is the interface type that needs to implemented by all drivers.
type Driver interface {
	// Close is the last function to be called.
	// Close any open connection here.
	Close() error

	// Migrate is the heart of the driver.
	// It will receive a file which the driver should apply
	// to its backend or whatever. The migration function should use
	// the pipe channel to return any errors or other useful information.
	Migrate(file file.File) error

	// Version returns the current migration version.
	Version() (file.Version, error)

	// Versions returns the list of applied migrations.
	Versions() (file.Versions, error)

	// Execute a statement
	Execute(statement string) error
}

// Lockable represents driver that supports database locking.
// Implement if possible to make it safe to run migrations concurrently.
// NOTE: Probably better to move into Driver interface to make sure it's not
// dismissed when locking is possible to implement.
type Lockable interface {
	Lock() error
	Unlock() error
}

// Lock calls Lock method if driver implements Lockable
func Lock(d Driver) error {
	if d, ok := d.(Lockable); ok {
		return d.Lock()
	}
	return nil
}

// Unlock calls Unlock method if driver implements Lockable
func Unlock(d Driver) error {
	if d, ok := d.(Lockable); ok {
		return d.Unlock()
	}
	return nil
}

// FileExtension returns extension of migration file for given driver.
// Panics if you provide instance of unregistered driver.
func FileExtension(d Driver) string {
	if d, ok := driversPkg[reflect.TypeOf(d).Elem().PkgPath()]; ok {
		return d.fileExtension
	}
	panic(fmt.Sprintf("unregistered driver instance: %#v (%s)", d, reflect.TypeOf(d).Elem().PkgPath()))
}

// FileTemplate returns initial content of migration file for given driver.
// Panics if you provide instance of unregistered driver.
func FileTemplate(d Driver) []byte {
	if d, ok := driversPkg[reflect.TypeOf(d).Elem().PkgPath()]; ok {
		return d.fileTemplate
	}
	panic(fmt.Sprintf("unregistered driver instance: %#v (%s)", d, reflect.TypeOf(d).Elem().PkgPath()))
}

// New returns Driver and calls Initialize on it.
func New(url string) (Driver, error) {
	scheme := getScheme(url)
	if scheme == "" {
		return nil, fmt.Errorf("no scheme found in %q", url)
	}

	drv := getDriver(scheme)
	if drv == nil {
		return nil, fmt.Errorf("driver '%s' not found", scheme)
	}
	return drv.new(url)
}

// getScheme will get the scheme of a URL-like connection string
func getScheme(url string) string {
	re := regexp.MustCompile(`(?m)^(\w+)://`)
	match := re.FindStringSubmatch(url)
	if len(match) < 2 {
		return ""
	}
	return match[1]
}
