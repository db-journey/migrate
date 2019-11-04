package migrate

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"testing"

	// Ensure imports for each driver we wish to test

	_ "github.com/db-journey/cassandra-driver"
	"github.com/db-journey/migrate/direction"
	"github.com/db-journey/migrate/driver"
	"github.com/db-journey/migrate/file"
	_ "github.com/db-journey/mysql-driver"
	_ "github.com/db-journey/postgresql-driver"
	_ "github.com/db-journey/sqlite3-driver"
)

// Add Driver URLs here to test basic Up, Down, .. functions.
var driverUrls = []string{
	//"postgres://postgres@" + os.Getenv("POSTGRES_PORT_5432_TCP_ADDR") + ":" + os.Getenv("POSTGRES_PORT_5432_TCP_PORT") + "/template1?sslmode=disable",
	"mysql://root@tcp(" + os.Getenv("MYSQL_PORT_3306_TCP_ADDR") + ":" + os.Getenv("MYSQL_PORT_3306_TCP_PORT") + ")/migratetest",
	// "cassandra://" + os.Getenv("CASSANDRA_PORT_9042_TCP_ADDR") + ":" + os.Getenv("CASSANDRA_PORT_9042_TCP_PORT") + "/migrate?protocol=4",
	"sqlite3:///tmp/migrate.db",
}

func TestCreate(t *testing.T) {
	for _, driverUrl := range driverUrls {
		ctx := context.Background()
		t.Logf("Test driver: %s", driverUrl)
		tmpdir, err := ioutil.TempDir("/tmp", "migrate-test")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpdir)

		m, err := Open(driverUrl, tmpdir)
		if err != nil {
			t.Fatalf("Failed to initialize Handle: %s", err)
		}

		file1, err := m.Create("test_migration")
		if err != nil {
			t.Fatal(err)
		}
		file2, err := m.Create("another migration")
		if err != nil {
			t.Fatal(err)
		}

		files, err := ioutil.ReadDir(tmpdir)
		if err != nil {
			t.Fatal(err)
		}
		if len(files) != 4 {
			t.Fatal("Expected 2 new files, got", len(files))
		}
		expectFiles := []string{
			file1.UpFile.FileName, file1.DownFile.FileName,
			file2.UpFile.FileName, file2.DownFile.FileName,
		}
		for _, expectFile := range expectFiles {
			filepath := path.Join(tmpdir, expectFile)
			if _, err := os.Stat(filepath); os.IsNotExist(err) {
				t.Errorf("Can't find migration file: %s", filepath)
			}
		}

		if file1.Version == file2.Version {
			t.Errorf("files can't same version: %d", file1.Version)
		}
		ensureClean(ctx, t, m)
	}
}

func TestReset(t *testing.T) {
	for _, driverUrl := range driverUrls {
		t.Logf("Test driver: %s", driverUrl)
		tmpdir, err := ioutil.TempDir("/tmp", "migrate-test")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpdir)

		ctx := context.Background()
		m, err := Open(driverUrl, tmpdir)
		if err != nil {
			t.Fatalf("Failed to initialize Handle: %s", err)
		}

		_, err = m.Create("migration1")
		if err != nil {
			t.Fatal(err)
		}
		file, err := m.Create("migration2")
		if err != nil {
			t.Fatal(err)
		}

		err = m.Reset(ctx)
		if err != nil {
			t.Fatal(err)
		}
		version, err := m.Version(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if version != file.Version {
			versions, _ := m.Versions(ctx)
			t.Logf("Versions in db: %v", versions)
			t.Fatalf("Expected version %d, got %v", file.Version, version)
		}

		ensureClean(ctx, t, m)
	}
}

func TestDown(t *testing.T) {
	for _, driverUrl := range driverUrls {
		t.Logf("Test driver: %s", driverUrl)
		tmpdir, err := ioutil.TempDir("/tmp", "migrate-test")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpdir)

		ctx := context.Background()
		m, err := Open(driverUrl, tmpdir)
		if err != nil {
			t.Fatalf("Failed to initialize Handle: %s", err)
		}

		m.Create("migration1")
		file, _ := m.Create("migration2")

		err = m.Reset(ctx)
		if err != nil {
			t.Fatal(err)
		}
		version, err := m.Version(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if version != file.Version {
			t.Fatalf("Expected version %d, got %v", file.Version, version)
		}

		err = m.Down(ctx)
		if err != nil {
			t.Fatal(err)
		}
		version, err = m.Version(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if version != 0 {
			t.Fatalf("Expected version 0, got %v", version)
		}
		ensureClean(ctx, t, m)
	}
}

func TestUp(t *testing.T) {
	for _, driverUrl := range driverUrls {
		t.Logf("Test driver: %s", driverUrl)
		tmpdir, err := ioutil.TempDir("/tmp", "migrate-test")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpdir)

		ctx := context.Background()
		m, err := Open(driverUrl, tmpdir)
		if err != nil {
			t.Fatalf("Failed to initialize Handle: %s", err)
		}

		m.Create("migration1")
		file, _ := m.Create("migration2")

		err = m.Down(ctx)
		if err != nil {
			t.Fatal(err)
		}
		version, err := m.Version(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if version != 0 {
			t.Fatalf("Expected version 0, got %v", version)
		}

		err = m.Up(ctx)
		if err != nil {
			t.Fatal(err)
		}
		version, err = m.Version(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if version != file.Version {
			t.Fatalf("Expected version %d, got %v", file.Version, version)
		}

		ensureClean(ctx, t, m)
	}
}

func TestRedo(t *testing.T) {
	for _, driverUrl := range driverUrls {
		t.Logf("Test driver: %s", driverUrl)
		tmpdir, err := ioutil.TempDir("/tmp", "migrate-test")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpdir)

		ctx := context.Background()
		m, err := Open(driverUrl, tmpdir)
		if err != nil {
			t.Fatalf("Failed to initialize Handle: %s", err)
		}

		m.Create("migration1")
		file, _ := m.Create("migration2")

		err = m.Reset(ctx)
		if err != nil {
			t.Fatal(err)
		}
		version, err := m.Version(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if version != file.Version {
			t.Fatalf("Expected version %d, got %v", file.Version, version)
		}

		err = m.Redo(ctx)
		if err != nil {
			t.Fatal(err)
		}
		version, err = m.Version(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if version != file.Version {
			t.Fatalf("Expected version %d, got %v", file.Version, version)
		}
		ensureClean(ctx, t, m)
	}
}

func TestMigrate(t *testing.T) {
	for _, driverUrl := range driverUrls {
		t.Logf("Test driver: %s", driverUrl)
		tmpdir, err := ioutil.TempDir("/tmp", "migrate-test")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpdir)

		ctx := context.Background()
		m, err := Open(driverUrl, tmpdir)
		if err != nil {
			t.Fatalf("Failed to initialize Handle: %s", err)
		}

		file1, err := m.Create("migration1")
		if err != nil {
			t.Fatal(err)
		}

		file2, err := m.Create("migration2")
		if err != nil {
			t.Fatal(err)
		}

		err = m.Reset(ctx)
		if err != nil {
			t.Fatal(err)
		}
		version, err := m.Version(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if version != file2.Version {
			t.Fatalf("Expected version %d, got %v", file2.Version, version)
		}

		err = m.Migrate(ctx, -2)
		if err != nil {
			t.Fatal(err)
		}
		version, err = m.Version(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if version != 0 {
			versions, _ := m.Versions(ctx)
			t.Logf("Versions in db: %v", versions)
			t.Fatalf("Expected version 0, got %v", version)
		}

		err = m.Migrate(ctx, +1)
		if err != nil {
			t.Fatal(err)
		}
		version, err = m.Version(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if version != file1.Version {
			t.Fatalf("Expected version %d, got %v", file1.Version, version)
		}

		err = createOldMigrationFile(driverUrl, tmpdir)
		if err != nil {
			t.Fatal(err)
		}

		err = m.Up(ctx)
		if err != nil {
			t.Fatal(err)
		}
		expectedVersions := file.Versions{
			file2.Version,
			file1.Version,
			20060102150405,
		}

		versions, err := m.Versions(ctx)
		if err != nil {
			t.Errorf("Could not fetch versions: %s", err)
		}

		if !reflect.DeepEqual(versions, expectedVersions) {
			t.Errorf("Expected versions to be: %v, got: %v", expectedVersions, versions)
		}

		ensureClean(ctx, t, m)
	}
}

func ensureClean(ctx context.Context, t *testing.T, m *Handle) {
	if err := m.Down(ctx); err != nil {
		t.Fatal(err)
	}
	version, err := m.Version(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if version != 0 {
		t.Fatalf("Expected version 0, got %v", version)
	}
}

func createOldMigrationFile(url, migrationsPath string) error {
	version := file.Version(20060102150405)
	filenamef := "%d_%s.%s.%s"
	name := "old"
	d, err := driver.New(url)
	if err != nil {
		return err
	}

	ext := driver.FileExtension(d)

	mfile := &file.MigrationFile{
		Version: version,
		UpFile: &file.File{
			Path:      migrationsPath,
			FileName:  fmt.Sprintf(filenamef, version, name, "up", ext),
			Name:      name,
			Content:   []byte(""),
			Direction: direction.Up,
		},
		DownFile: &file.File{
			Path:      migrationsPath,
			FileName:  fmt.Sprintf(filenamef, version, name, "down", ext),
			Name:      name,
			Content:   []byte(""),
			Direction: direction.Down,
		},
	}

	err = ioutil.WriteFile(path.Join(mfile.UpFile.Path, mfile.UpFile.FileName), mfile.UpFile.Content, 0644)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path.Join(mfile.DownFile.Path, mfile.DownFile.FileName), mfile.DownFile.Content, 0644)
}
