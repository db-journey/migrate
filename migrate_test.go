package migrate

import (
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
	"postgres://postgres@" + os.Getenv("POSTGRES_PORT_5432_TCP_ADDR") + ":" + os.Getenv("POSTGRES_PORT_5432_TCP_PORT") + "/template1?sslmode=disable",
	"mysql://root@tcp(" + os.Getenv("MYSQL_PORT_3306_TCP_ADDR") + ":" + os.Getenv("MYSQL_PORT_3306_TCP_PORT") + ")/migratetest",
	// "cassandra://" + os.Getenv("CASSANDRA_PORT_9042_TCP_ADDR") + ":" + os.Getenv("CASSANDRA_PORT_9042_TCP_PORT") + "/migrate?protocol=4",
	"sqlite3:///tmp/migrate.db",
}

func TestCreate(t *testing.T) {
	for _, driverUrl := range driverUrls {
		t.Logf("Test driver: %s", driverUrl)
		tmpdir, err := ioutil.TempDir("/tmp", "migrate-test")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpdir)

		file1, err := Create(driverUrl, tmpdir, "test_migration")
		if err != nil {
			t.Fatal(err)
		}
		file2, err := Create(driverUrl, tmpdir, "another migration")
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
		ensureClean(t, tmpdir, driverUrl)
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

		_, err = Create(driverUrl, tmpdir, "migration1")
		if err != nil {
			t.Fatal(err)
		}
		file, err := Create(driverUrl, tmpdir, "migration2")
		if err != nil {
			t.Fatal(err)
		}

		err = Reset(driverUrl, tmpdir)
		if err != nil {
			t.Fatal(err)
		}
		version, err := Version(driverUrl, tmpdir)
		if err != nil {
			t.Fatal(err)
		}
		if version != file.Version {
			versions, _ := Versions(driverUrl, tmpdir)
			t.Logf("Versions in db: %v", versions)
			t.Fatalf("Expected version %d, got %v", file.Version, version)
		}

		ensureClean(t, tmpdir, driverUrl)
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

		Create(driverUrl, tmpdir, "migration1")
		file, _ := Create(driverUrl, tmpdir, "migration2")

		err = Reset(driverUrl, tmpdir)
		if err != nil {
			t.Fatal(err)
		}
		version, err := Version(driverUrl, tmpdir)
		if err != nil {
			t.Fatal(err)
		}
		if version != file.Version {
			t.Fatalf("Expected version %d, got %v", file.Version, version)
		}

		err = Down(driverUrl, tmpdir)
		if err != nil {
			t.Fatal(err)
		}
		version, err = Version(driverUrl, tmpdir)
		if err != nil {
			t.Fatal(err)
		}
		if version != 0 {
			t.Fatalf("Expected version 0, got %v", version)
		}
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

		Create(driverUrl, tmpdir, "migration1")
		file, _ := Create(driverUrl, tmpdir, "migration2")

		err = Down(driverUrl, tmpdir)
		if err != nil {
			t.Fatal(err)
		}
		version, err := Version(driverUrl, tmpdir)
		if err != nil {
			t.Fatal(err)
		}
		if version != 0 {
			t.Fatalf("Expected version 0, got %v", version)
		}

		err = Up(driverUrl, tmpdir)
		if err != nil {
			t.Fatal(err)
		}
		version, err = Version(driverUrl, tmpdir)
		if err != nil {
			t.Fatal(err)
		}
		if version != file.Version {
			t.Fatalf("Expected version %d, got %v", file.Version, version)
		}

		ensureClean(t, tmpdir, driverUrl)
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

		Create(driverUrl, tmpdir, "migration1")
		file, _ := Create(driverUrl, tmpdir, "migration2")

		err = Reset(driverUrl, tmpdir)
		if err != nil {
			t.Fatal(err)
		}
		version, err := Version(driverUrl, tmpdir)
		if err != nil {
			t.Fatal(err)
		}
		if version != file.Version {
			t.Fatalf("Expected version %d, got %v", file.Version, version)
		}

		err = Redo(driverUrl, tmpdir)
		if err != nil {
			t.Fatal(err)
		}
		version, err = Version(driverUrl, tmpdir)
		if err != nil {
			t.Fatal(err)
		}
		if version != file.Version {
			t.Fatalf("Expected version %d, got %v", file.Version, version)
		}
		ensureClean(t, tmpdir, driverUrl)
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

		file1, err := Create(driverUrl, tmpdir, "migration1")
		if err != nil {
			t.Fatal(err)
		}

		file2, err := Create(driverUrl, tmpdir, "migration2")
		if err != nil {
			t.Fatal(err)
		}

		err = Reset(driverUrl, tmpdir)
		if err != nil {
			t.Fatal(err)
		}
		version, err := Version(driverUrl, tmpdir)
		if err != nil {
			t.Fatal(err)
		}
		if version != file2.Version {
			t.Fatalf("Expected version %d, got %v", file2.Version, version)
		}

		err = Migrate(driverUrl, tmpdir, -2)
		if err != nil {
			t.Fatal(err)
		}
		version, err = Version(driverUrl, tmpdir)
		if err != nil {
			t.Fatal(err)
		}
		if version != 0 {
			versions, _ := Versions(driverUrl, tmpdir)
			t.Logf("Versions in db: %v", versions)
			t.Fatalf("Expected version 0, got %v", version)
		}

		err = Migrate(driverUrl, tmpdir, +1)
		if err != nil {
			t.Fatal(err)
		}
		version, err = Version(driverUrl, tmpdir)
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

		err = Up(driverUrl, tmpdir)
		if err != nil {
			t.Fatal(err)
		}
		expectedVersions := file.Versions{
			file2.Version,
			file1.Version,
			20060102150405,
		}

		versions, err := Versions(driverUrl, tmpdir)
		if err != nil {
			t.Errorf("Could not fetch versions: %s", err)
		}

		if !reflect.DeepEqual(versions, expectedVersions) {
			t.Errorf("Expected versions to be: %v, got: %v", expectedVersions, versions)
		}

		ensureClean(t, tmpdir, driverUrl)
	}
}

func ensureClean(t *testing.T, tmpdir, driverUrl string) {
	if err := Down(driverUrl, tmpdir); err != nil {
		t.Fatal(err)
	}
	version, err := Version(driverUrl, tmpdir)
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

	mfile := &file.MigrationFile{
		Version: version,
		UpFile: &file.File{
			Path:      migrationsPath,
			FileName:  fmt.Sprintf(filenamef, version, name, "up", d.FilenameExtension()),
			Name:      name,
			Content:   []byte(""),
			Direction: direction.Up,
		},
		DownFile: &file.File{
			Path:      migrationsPath,
			FileName:  fmt.Sprintf(filenamef, version, name, "down", d.FilenameExtension()),
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
