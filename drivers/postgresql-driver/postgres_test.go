package postgres

import (
	"database/sql"
	"os"
	"reflect"
	"testing"

	"github.com/db-journey/migrate/v2/direction"
	"github.com/db-journey/migrate/v2/driver"
	"github.com/db-journey/migrate/v2/file"
)

// TestMigrate runs some additional tests on Migrate().
// Basic testing is already done in migrate_test.go
func TestMigrate(t *testing.T) {
	host := getenvDefault("POSTGRES_PORT_5432_TCP_ADDR", "localhost")
	port := getenvDefault("POSTGRES_PORT_5432_TCP_PORT", "5432")
	driverURL := "postgres://postgres:p@" + host + ":" + port + "/template1?sslmode=disable"

	// prepare clean database
	connection, err := sql.Open("postgres", driverURL)
	if err != nil {
		t.Fatal(err)
	}

	dropTestTables(t, connection)

	migrate(t, driverURL)

	dropTestTables(t, connection)

	// Make an old-style `int` version column that we'll have to upgrade.
	_, err = connection.Exec("CREATE TABLE IF NOT EXISTS " + tableName + " (version bigint not null primary key)")
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
					id serial not null primary key
				);
				CREATE TYPE colors AS ENUM (
					'red',
					'green'
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
			FileName:  "20060102150406_foobar.up.sql",
			Version:   20060102150406,
			Name:      "foobar",
			Direction: direction.Up,
			Content: []byte(`-- disable_ddl_transaction
				ALTER TYPE colors ADD VALUE 'blue' AFTER 'red';
			`),
		},
		{
			Path:      "/foobar",
			FileName:  "20060102150406_foobar.down.sql",
			Version:   20060102150406,
			Name:      "foobar",
			Direction: direction.Down,
			Content: []byte(`
				DROP TYPE colors;
			`),
		},
		{
			Path:      "/foobar",
			FileName:  "20060102150407_foobar.up.sql",
			Version:   20060102150407,
			Name:      "foobar",
			Direction: direction.Up,
			Content: []byte(`
				CREATE TABLE error (
					id THIS WILL CAUSE AN ERROR
				)
			`),
		},
	}

	// should create table yolo
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

	// Check versions applied in DB
	expectedVersions := file.Versions{20060102150405}
	versions, err := d.Versions()
	if err != nil {
		t.Errorf("Could not fetch versions: %s", err)
	}

	if !reflect.DeepEqual(versions, expectedVersions) {
		t.Errorf("Expected versions to be: %v, got: %v", expectedVersions, versions)
	}

	// should alter type colors
	err = d.Migrate(files[2])
	if err != nil {
		t.Fatal(err)
	}

	colors := []string{}
	expectedColors := []string{"red", "blue", "green"}

	rows, err := d.(*Driver).db.Query("SELECT unnest(enum_range(NULL::colors));")
	if err != nil {
		t.Error(err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var color string
		if err = rows.Scan(&color); err != nil {
			t.Error(err)
			return
		}
		colors = append(colors, color)
	}

	if err = rows.Err(); err != nil {
		t.Error(err)
		return
	}

	if !reflect.DeepEqual(colors, expectedColors) {
		t.Errorf("Expected colors enum to be %q, got %q\n", expectedColors, colors)
	}

	err = d.Migrate(files[3])
	if err != nil {
		t.Fatal(err)
	}

	err = d.Migrate(files[1])
	if err != nil {
		t.Fatal(err)
	}

	err = d.Migrate(files[4])
	if err == nil {
		t.Error("Expected test case to fail")
	}

	// Check versions applied in DB
	expectedVersions = file.Versions{}
	versions, err = d.Versions()
	if err != nil {
		t.Errorf("Could not fetch versions: %s", err)
	}

	if !reflect.DeepEqual(versions, expectedVersions) {
		t.Errorf("Expected versions to be: %v, got: %v", expectedVersions, versions)
	}

	if err := d.Close(); err != nil {
		t.Fatal(err)
	}

}

func dropTestTables(t *testing.T, db *sql.DB) {
	if _, err := db.Exec(`
				DROP TYPE IF EXISTS colors;
				DROP TABLE IF EXISTS yolo;
				DROP TABLE IF EXISTS ` + tableName + `;`); err != nil {
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
