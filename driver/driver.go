// Package driver holds the driver interface.
package driver

import (
	"fmt"
	neturl "net/url" // alias to allow `url string` func signature in New

	"github.com/db-journey/migrate/file"
)

// Driver is the interface type that needs to implemented by all drivers.
type Driver interface {

	// Initialize is the first function to be called.
	// Check the url string and open and verify any connection
	// that has to be made.
	Initialize(url string) error

	// Close is the last function to be called.
	// Close any open connection here.
	Close() error

	// FilenameExtension returns the extension of the migration files.
	// The returned string must not begin with a dot.
	FilenameExtension() string

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

// FileTemplater can be optionally implemented to
// fill newly created migration files with something useful.
type FileTemplater interface {
	// FileTemplate returns content that should be written
	// into newly-created migration file.
	FileTemplate() []byte
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

// FileTemplate returns migration file template
// for given driver if it implements FileTemplater
// or empty slice otherwise.
func FileTemplate(d Driver) []byte {
	if d, ok := d.(FileTemplater); ok {
		return d.FileTemplate()
	}
	return []byte{}
}

// New returns Driver and calls Initialize on it.
func New(url string) (Driver, error) {
	u, err := neturl.Parse(url)
	if err != nil {
		return nil, err
	}

	d := GetDriver(u.Scheme)
	if d == nil {
		return nil, fmt.Errorf("driver '%s' not found", u.Scheme)
	}
	verifyFilenameExtension(u.Scheme, d)
	if err := d.Initialize(url); err != nil {
		return nil, err
	}

	return d, nil
}

// verifyFilenameExtension panics if the driver's filename extension
// is not correct or empty.
func verifyFilenameExtension(driverName string, d Driver) {
	f := d.FilenameExtension()
	if f == "" {
		panic(fmt.Sprintf("%s.FilenameExtension() returns empty string.", driverName))
	}
	if f[0:1] == "." {
		panic(fmt.Sprintf("%s.FilenameExtension() returned string must not start with a dot.", driverName))
	}
}
