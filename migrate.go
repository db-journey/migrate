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

// Up applies all available migrations.
func Up(url, migrationsPath string) error {
	d, files, versions, err := initDriverAndReadMigrationFilesAndGetVersions(url, migrationsPath)
	if err != nil {
		return err
	}
	defer d.Close()
	applyMigrationFiles, err := files.Pending(versions)
	if err != nil {
		return err
	}
	for _, f := range applyMigrationFiles {
		err := d.Migrate(f)
		if err != nil {
			return err
		}
	}
	return nil
}

// Down rolls back all migrations.
func Down(url, migrationsPath string) error {
	d, files, versions, err := initDriverAndReadMigrationFilesAndGetVersions(url, migrationsPath)
	if err != nil {
		return err
	}
	defer d.Close()

	applyMigrationFiles, err := files.Applied(versions)
	if err != nil {
		return err
	}

	for _, f := range applyMigrationFiles {
		err = d.Migrate(f)
		if err != nil {
			break
		}
	}
	return err
}

// Redo rolls back the most recently applied migration, then runs it again.
func Redo(url, migrationsPath string) error {
	err := Migrate(url, migrationsPath, -1)
	if err != nil {
		return err
	}
	return Migrate(url, migrationsPath, +1)
}

// Reset runs the down and up migration function.
func Reset(url, migrationsPath string) error {
	err := Down(url, migrationsPath)
	if err != nil {
		return err
	}
	return Up(url, migrationsPath)
}

// Migrate applies relative +n/-n migrations.
func Migrate(url, migrationsPath string, relativeN int) error {
	d, files, versions, err := initDriverAndReadMigrationFilesAndGetVersions(url, migrationsPath)
	if err != nil {
		return err
	}

	applyMigrationFiles, err := files.Relative(relativeN, versions)
	if err != nil {
		return err
	}

	for _, f := range applyMigrationFiles {
		err = d.Migrate(f)
		if err != nil {
			break
		}
	}
	return err
}

// Version returns the current migration version.
func Version(url, migrationsPath string) (version file.Version, err error) {
	d, err := driver.New(url)
	if err != nil {
		return 0, err
	}
	return d.Version()
}

// Versions returns applied versions.
func Versions(url, migrationsPath string) (versions file.Versions, err error) {
	d, err := driver.New(url)
	if err != nil {
		return file.Versions{}, err
	}
	return d.Versions()
}

// PendingMigrations returns list of pending migration files
func PendingMigrations(url, migrationsPath string) (file.Files, error) {
	_, files, versions, err := initDriverAndReadMigrationFilesAndGetVersions(url, migrationsPath)
	if err != nil {
		return nil, err
	}
	return files.Pending(versions)
}

// Create creates new migration files on disk.
func Create(url, migrationsPath, name string) (*file.MigrationFile, error) {
	d, files, _, err := initDriverAndReadMigrationFilesAndGetVersions(url, migrationsPath)
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
			Path:      migrationsPath,
			FileName:  fmt.Sprintf(filenamef, version, name, "up", d.FilenameExtension()),
			Name:      name,
			Content:   d.FileTemplate(),
			Direction: direction.Up,
		},
		DownFile: &file.File{
			Path:      migrationsPath,
			FileName:  fmt.Sprintf(filenamef, version, name, "down", d.FilenameExtension()),
			Name:      name,
			Content:   d.FileTemplate(),
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

// initDriverAndReadMigrationFilesAndGetVersionsAndGetVersion is a small helper
// function that is common to most of the migration funcs.
func initDriverAndReadMigrationFilesAndGetVersions(url, migrationsPath string) (driver.Driver, file.MigrationFiles, file.Versions, error) {
	d, err := driver.New(url)
	if err != nil {
		return nil, nil, file.Versions{}, err
	}
	defer d.Close()
	files, err := file.ReadMigrationFiles(migrationsPath, file.FilenameRegex(d.FilenameExtension()))
	if err != nil {
		return nil, nil, file.Versions{}, err
	}
	versions, err := d.Versions()
	return d, files, versions, err
}
