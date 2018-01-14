package migrate

import (
	"context"
	"errors"
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

// Option for New
type Option func(h *Handle) error

// WithHooks allows to add pre/post migration hooks.
func WithHooks(pre, post func(f file.File) error) Option {
	return func(h *Handle) error {
		h.preHook = pre
		h.postHook = post
		return nil
	}
}

// Handle encapsulates migrations functionality
type Handle struct {
	drv            driver.Driver
	migrationsPath string
	locked         bool
	fatalErr       error

	preHook, postHook func(f file.File) error
}

// Open migrations Handle
func Open(url, migrationsPath string, opts ...Option) (*Handle, error) {
	d, err := driver.New(url)
	if err != nil {
		return nil, err
	}
	return New(d, migrationsPath, opts...)
}

// New migrations Handle
func New(drv driver.Driver, migrationsPath string, opts ...Option) (*Handle, error) {
	if drv == nil {
		return nil, errors.New("driver can't be nil")
	}
	h := &Handle{
		drv:            drv,
		migrationsPath: migrationsPath,
	}
	for _, configure := range opts {
		err := configure(h)
		if err != nil {
			return nil, err
		}
	}
	return h, nil
}

// Up applies all available migrations.
func (m *Handle) Up(ctx context.Context) error {
	return m.locking(ctx, func() error {
		files, versions, err := m.readFilesAndGetVersions()
		if err != nil {
			return err
		}
		applyMigrationFiles, err := files.Pending(versions)
		if err != nil {
			return err
		}
		for _, f := range applyMigrationFiles {
			err = m.drvMigrate(ctx, f)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

// Down rolls back all migrations.
func (m *Handle) Down(ctx context.Context) error {
	return m.locking(ctx, func() error {
		files, versions, err := m.readFilesAndGetVersions()
		if err != nil {
			return err
		}
		applyMigrationFiles, err := files.Applied(versions)
		if err != nil {
			return err
		}

		for _, f := range applyMigrationFiles {
			err = m.drvMigrate(ctx, f)
			if err != nil {
				break
			}
		}
		return err
	})
}

// Redo rolls back the most recently applied migration, then runs it again.
func (m *Handle) Redo(ctx context.Context) error {
	return m.locking(ctx, func() error {
		err := m.Migrate(ctx, -1)
		if err != nil {
			return err
		}
		return m.Migrate(ctx, +1)
	})
}

// Reset runs the Down and Up migration function.
func (m *Handle) Reset(ctx context.Context) error {
	return m.locking(ctx, func() error {
		err := m.Down(ctx)
		if err != nil {
			return err
		}
		return m.Up(ctx)
	})
}

// Migrate applies relative +n/-n migrations.
func (m *Handle) Migrate(ctx context.Context, relativeN int) error {
	return m.locking(ctx, func() error {
		files, versions, err := m.readFilesAndGetVersions()
		if err != nil {
			return err
		}

		applyMigrationFiles, err := files.Relative(relativeN, versions)
		if err != nil {
			return err
		}

		for _, f := range applyMigrationFiles {
			err = m.drvMigrate(ctx, f)
			if err != nil {
				break
			}
		}
		return err
	})
}

// Version returns the current migration version.
func (m *Handle) Version(ctx context.Context) (version file.Version, err error) {
	unlock, err := m.lock(ctx)
	if err != nil {
		return 0, err
	}
	defer unlock()
	return m.drv.Version()
}

// Versions returns applied versions.
func (m *Handle) Versions(ctx context.Context) (versions file.Versions, err error) {
	unlock, err := m.lock(ctx)
	if err != nil {
		return nil, err
	}
	defer unlock()
	return m.drv.Versions()
}

// PendingMigrations returns list of pending migration files
func (m *Handle) PendingMigrations(ctx context.Context) (file.Files, error) {
	unlock, err := m.lock(ctx)
	if err != nil {
		return nil, err
	}
	defer unlock()
	files, versions, err := m.readFilesAndGetVersions()
	if err != nil {
		return nil, err
	}
	return files.Pending(versions)
}

// Create creates new migration files on disk.
func (m *Handle) Create(name string) (*file.MigrationFile, error) {
	files, _, err := m.readFilesAndGetVersions()
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
			Content:   driver.FileTemplate(m.drv),
			Direction: direction.Up,
		},
		DownFile: &file.File{
			Path:      m.migrationsPath,
			FileName:  fmt.Sprintf(filenamef, version, name, "down", m.drv.FilenameExtension()),
			Name:      name,
			Content:   driver.FileTemplate(m.drv),
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

// ApplyVersion applies specific version.
func (m *Handle) ApplyVersion(ctx context.Context, version file.Version) error {
	return m.migrateVersion(ctx, version, direction.Up)
}

// RollbackVersion runs `down` migration for specific version.
func (m *Handle) RollbackVersion(ctx context.Context, version file.Version) error {
	return m.migrateVersion(ctx, version, direction.Down)
}

// Close database connection
func (m *Handle) Close() error {
	return m.drv.Close()
}

// migrateVersion runs up or down migration (depends on given direction)
// for given version.
func (m *Handle) migrateVersion(ctx context.Context, version file.Version, d direction.Direction) error {
	if d != direction.Up && d != direction.Down {
		return fmt.Errorf("invalid direction: %v", d)
	}
	return m.locking(ctx, func() error {
		files, versions, err := m.readFilesAndGetVersions()
		if err != nil {
			return err
		}
		if d == direction.Up && versions.Contains(version) {
			return fmt.Errorf("version %d is already applied", version)
		}
		if d == direction.Down && !versions.Contains(version) {
			return fmt.Errorf("version %d is not applied", version)
		}
		var migration *file.File
		for _, f := range files {
			if f.Version == version {
				if migration = getFileForDirection(f, d); migration != nil {
					return m.drvMigrate(ctx, *migration)
				}
				break
			}
		}
		// XXX: for some reason govet complains about Direction, so .String() is explicit
		return fmt.Errorf("no `%s` migration file for version %d", d.String(), version)
	})
}

func (m *Handle) lock(ctx context.Context) (unlock func(), err error) {
	if m.fatalErr != nil {
		return nil, m.fatalErr
	}
	if m.locked {
		return func() {}, nil
	}
	select {
	case err := <-drvLockChan(m.drv):
		if err != nil {
			return nil, err
		}
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	m.locked = true
	return m.unlock, nil
}

func (m *Handle) unlock() {
	err := driver.Unlock(m.drv)
	if err == nil {
		m.locked = false
		return
	}
	m.Close()
	m.fatalErr = fmt.Errorf("connection closed, this handle is no longer usable - failed to unlock database after last session: %s", err)
}

func (m *Handle) locking(ctx context.Context, f func() error) error {
	unlock, err := m.lock(ctx)
	if err != nil {
		return err
	}
	defer unlock()
	return f()
}

func (m *Handle) drvMigrate(ctx context.Context, f file.File) error {
	select {
	case <-ctx.Done():
		return fmt.Errorf("interrupted before applying version %d: %s", f.Version, ctx.Err())
	default:
		err := runHookIfNotNil(m.preHook, "pre", f)
		if err != nil {
			return err
		}
		err = m.drv.Migrate(f)
		if err != nil {
			return err
		}
		return runHookIfNotNil(m.postHook, "post", f)
	}
}

// readFilesAndGetVersions is a small helper
// function that is common to most of the migration funcs.
func (m *Handle) readFilesAndGetVersions() (file.MigrationFiles, file.Versions, error) {
	files, err := file.ReadMigrationFiles(m.migrationsPath, file.FilenameRegex(m.drv.FilenameExtension()))
	if err != nil {
		return nil, file.Versions{}, err
	}
	versions, err := m.drv.Versions()
	return files, versions, err
}

func runHookIfNotNil(hook func(f file.File) error, name string, f file.File) error {
	if hook == nil {
		return nil
	}
	err := hook(f)
	if err != nil {
		return fmt.Errorf("%s-hook for migration %q failed: %s", name, f.FileName, err)
	}
	return nil
}

func getFileForDirection(m file.MigrationFile, d direction.Direction) *file.File {
	if d == direction.Up {
		return m.UpFile
	}
	return m.DownFile
}

func drvLockChan(drv driver.Driver) <-chan error {
	ret := make(chan error)
	go func() {
		if err := driver.Lock(drv); err != nil {
			ret <- err
		}
		close(ret)
	}()
	return ret
}
