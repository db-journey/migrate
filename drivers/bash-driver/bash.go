// Package bash implements the Driver interface.
package bash

import (
	"os/exec"

	"github.com/db-journey/migrate/v2/driver"
	"github.com/db-journey/migrate/v2/file"
)

var fileTemplate = []byte(``)

func init() {
	driver.Register("bash", "sh", fileTemplate, Open)
}

type Driver struct {
}

func Open(url string) (driver.Driver, error) {
	return &Driver{}, nil
}

func (driver *Driver) Close() error {
	return nil
}

func (driver *Driver) Migrate(f file.File) error {
	return nil
}

// Version returns the current migration version.
func (driver *Driver) Version() (file.Version, error) {
	return file.Version(0), nil
}

// Versions returns the list of applied migrations.
func (driver *Driver) Versions() (file.Versions, error) {
	return file.Versions{0}, nil
}

// Execute shell script
func (driver *Driver) Execute(commands string) error {
	return exec.Command("sh", "-c", commands).Run()
}
