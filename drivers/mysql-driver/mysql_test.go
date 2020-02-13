package mysql

import (
	"database/sql"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/db-journey/migrate/v2/direction"
	"github.com/db-journey/migrate/v2/driver"
	"github.com/db-journey/migrate/v2/file"
)

// TestMigrate runs some additional tests on Migrate().
// Basic testing is already done in migrate_test.go
func TestMigrate(t *testing.T) {
	host := getenvDefault("MYSQL_PORT_3306_TCP_ADDR", "localhost")
	port := getenvDefault("MYSQL_PORT_3306_TCP_PORT", "3306")
	driverURL := "mysql://root@tcp(" + host + ":" + port + ")/migratetest"

	// prepare clean database
	connection, err := sql.Open("mysql", strings.SplitN(driverURL, "mysql://", 2)[1])
	if err != nil {
		t.Fatal(err)
	}

	dropTestTables(t, connection)

	migrate(t, driverURL)

	dropTestTables(t, connection)

	// Make an old-style 32-bit int version column that we'll have to upgrade.
	_, err = connection.Exec("CREATE TABLE IF NOT EXISTS " + versionsTableName + " (version int not null primary key);")
	if err != nil {
		t.Fatal(err)
	}

	migrate(t, driverURL)
}

func migrate(t *testing.T, driverURL string) {
	var err error
	var d driver.Driver
	if d, err = Open(driverURL); err != nil {
		t.Fatal(err)
	}

	files := []file.File{
		{
			Path:      "/foobar",
			FileName:  "20060102150405_foobar.up.sql",
			Version:   20060102150405,
			Name:      "foobar",
			Direction: direction.Up,
			Content: []byte(`
        CREATE TABLE yolo (
          id int(11) not null primary key auto_increment
        );

				CREATE TABLE yolo1 (
				  id int(11) not null primary key auto_increment
				);
      `),
		},
		{
			Path:      "/foobar",
			FileName:  "20060102150405_foobar.down.sql",
			Version:   20060102150405,
			Name:      "foobar",
			Direction: direction.Down,
			Content: []byte(`
        DROP TABLE yolo;
      `),
		},
		{
			Path:      "/foobar",
			FileName:  "20070000000000_foobar.up.sql",
			Version:   20070000000000,
			Name:      "foobar",
			Direction: direction.Up,
			Content: []byte(`

      	// a comment
				CREATE TABLE error (
          id THIS WILL CAUSE AN ERROR
        );
      `),
		},
	}

	driver.Lock(d)
	err = d.Migrate(files[0])
	if err != nil {
		t.Fatal(err)
	}

	version, err := d.Version()
	if err != nil {
		t.Fatal(err)
	}

	if version != 20060102150405 {
		t.Errorf("Expected version to be: %d, got: %d", 20060102150405, version)
	}
	driver.Unlock(d)

	// Check versions applied in DB
	expectedVersions := file.Versions{20060102150405}
	versions, err := d.Versions()
	if err != nil {
		t.Errorf("Could not fetch versions: %s", err)
	}

	driver.Lock(d)
	err = d.Migrate(files[1])
	if err != nil {
		t.Fatal(err)
	}
	driver.Unlock(d)

	driver.Lock(d)
	err = d.Migrate(files[2])
	if err == nil {
		t.Error("Expected test case to fail")
	}
	driver.Unlock(d)

	// Check versions applied in DB
	driver.Lock(d)
	expectedVersions = file.Versions{}
	versions, err = d.Versions()
	if err != nil {
		t.Errorf("Could not fetch versions: %s", err)
	}
	driver.Unlock(d)

	if !reflect.DeepEqual(versions, expectedVersions) {
		t.Errorf("Expected versions to be: %v, got: %v", expectedVersions, versions)
	}

	if err := d.Close(); err != nil {
		t.Fatal(err)
	}
}

func dropTestTables(t *testing.T, db *sql.DB) {
	if _, err := db.Exec(`DROP TABLE IF EXISTS yolo, yolo1, ` + versionsTableName); err != nil {
		t.Fatal(err)
	}
}

func getenvDefault(varname, defaultValue string) string {
	v := os.Getenv(varname)
	if v == "" {
		return defaultValue
	}
	return v
}
