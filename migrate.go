package migrate

import (
	"fmt"
	"io/ioutil"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/db-journey/migrate/direction"
	"github.com/db-journey/migrate/driver"
	"github.com/db-journey/migrate/file"
)

// Handle encapsulates migrations functionality
type Handle struct {
	drv            driver.Driver
	migrationsPath string
}

// New Handle instance
func New(url, migrationsPath string) (*Handle, error) {
	d, err := driver.New(url)
	if err != nil {
		return nil, err
	}
	return &Handle{drv: d, migrationsPath: migrationsPath}, nil
}

// Up applies all available migrations.
func (m *Handle) Up() error {
	files, versions, err := m.readMigrationFilesAndGetVersions()
	if err != nil {
		return err
	}
	applyMigrationFiles, err := files.Pending(versions)
	if err != nil {
		return err
	}
	for _, f := range applyMigrationFiles {
		err := m.drv.Migrate(f)
		if err != nil {
			return err
		}
	}
	return nil
}

// Down rolls back all migrations.
func (m *Handle) Down() error {
	files, versions, err := m.readMigrationFilesAndGetVersions()
	if err != nil {
		return err
	}
	applyMigrationFiles, err := files.Applied(versions)
	if err != nil {
		return err
	}

	for _, f := range applyMigrationFiles {
		err = m.drv.Migrate(f)
		if err != nil {
			break
		}
	}
	return err
}

// Redo rolls back the most recently applied migration, then runs it again.
func (m *Handle) Redo() error {
	err := m.Migrate(-1)
	if err != nil {
		return err
	}
	return m.Migrate(+1)
}

// Reset runs the down and up migration function.
func (m *Handle) Reset() error {
	err := m.Down()
	if err != nil {
		return err
	}
	return m.Up()
}

// Migrate applies relative +n/-n migrations.
func (m *Handle) Migrate(relativeN int) error {
	files, versions, err := m.readMigrationFilesAndGetVersions()
	if err != nil {
		return err
	}

	applyMigrationFiles, err := files.Relative(relativeN, versions)
	if err != nil {
		return err
	}

	for _, f := range applyMigrationFiles {
		err = m.drv.Migrate(f)
		if err != nil {
			break
		}
	}
	return err
}

// Version returns the current migration version.
func (m *Handle) Version() (version file.Version, err error) {
	return m.drv.Version()
}

// Versions returns applied versions.
func (m *Handle) Versions() (versions file.Versions, err error) {
	return m.drv.Versions()
}

// PendingMigrations returns list of pending migration files
func (m *Handle) PendingMigrations() (file.Files, error) {
	files, versions, err := m.readMigrationFilesAndGetVersions()
	if err != nil {
		return nil, err
	}
	return files.Pending(versions)
}

// Create creates new migration files on disk.
func (m *Handle) Create(name string) (*file.MigrationFile, error) {
	files, _, err := m.readMigrationFilesAndGetVersions()
	if err != nil {
		return nil, err
	}

	versionStr := time.Now().UTC().Format("20060102150405")
	v, _ := strconv.ParseUint(versionStr, 10, 64)
	version := file.Version(v)

	filenamef := "%d_%s.%s.%s"
	name = strings.Replace(name, " ", "_", -1)

	// if latest version has the same timestamp, increment version
	if len(files) > 0 {
		latest := files[len(files)-1].Version
		if latest >= version {
			version = latest + 1
		}
	}

	mfile := &file.MigrationFile{
		Version: version,
		UpFile: &file.File{
			Path:      m.migrationsPath,
			FileName:  fmt.Sprintf(filenamef, version, name, "up", m.drv.FilenameExtension()),
			Name:      name,
			Content:   m.drv.FileTemplate(),
			Direction: direction.Up,
		},
		DownFile: &file.File{
			Path:      m.migrationsPath,
			FileName:  fmt.Sprintf(filenamef, version, name, "down", m.drv.FilenameExtension()),
			Name:      name,
			Content:   m.drv.FileTemplate(),
			Direction: direction.Down,
		},
	}

	if err := ioutil.WriteFile(path.Join(mfile.UpFile.Path, mfile.UpFile.FileName), mfile.UpFile.Content, 0644); err != nil {
		return nil, err
	}
	if err := ioutil.WriteFile(path.Join(mfile.DownFile.Path, mfile.DownFile.FileName), mfile.DownFile.Content, 0644); err != nil {
		return nil, err
	}

	return mfile, nil
}

// Close database connection
func (m *Handle) Close() error {
	return m.drv.Close()
}

// initDriverAndReadMigrationFilesAndGetVersionsAndGetVersion is a small helper
// function that is common to most of the migration funcs.
func (m *Handle) readMigrationFilesAndGetVersions() (file.MigrationFiles, file.Versions, error) {
	files, err := file.ReadMigrationFiles(m.migrationsPath, file.FilenameRegex(m.drv.FilenameExtension()))
	if err != nil {
		return nil, file.Versions{}, err
	}
	versions, err := m.drv.Versions()
	return files, versions, err
}
