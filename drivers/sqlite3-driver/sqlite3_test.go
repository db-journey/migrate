package sqlite3

import (
	"io/ioutil"
	"os"
	"reflect"
	"testing"

	"github.com/db-journey/migrate/v2/direction"
	"github.com/db-journey/migrate/v2/driver"
	"github.com/db-journey/migrate/v2/file"
)

// TestMigrate runs some additional tests on Migrate()
// Basic testing is already done in migrate/migrate_test.go
func TestMigrate(t *testing.T) {
	f, err := ioutil.TempFile(os.TempDir(), "migrate_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())

	var d driver.Driver
	if d, err = Open("sqlite3://" + f.Name()); err != nil {
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
					id INTEGER PRIMARY KEY AUTOINCREMENT
				);
			`),
		},
		{
			Path:      "/foobar",
			FileName:  "20060102200405_alter_table.up.sql",
			Version:   20060102200405,
			Name:      "alter_table",
			Direction: direction.Up,
			Content: []byte(`
				ALTER TABLE yolo ADD COLUMN data1 VCHAR(255);
				ALTER TABLE yolo ADD COLUMN data2 VCHAR(255);
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
			FileName:  "20060102150406_failing.up.sql",
			Version:   20060103200406,
			Name:      "failing",
			Direction: direction.Down,
			Content: []byte(`
				CREATE TABLE error (
					THIS; WILL CAUSE; AN ERROR;
				)
			`),
		},
	}

	err = d.Migrate(files[0])
	if err != nil {
		t.Fatal(err)
	}

	version, err := d.Version()
	if err != nil {
		t.Fatal(err)
	}
	if version != files[0].Version {
		t.Errorf("Expected version to be: %d, got: %d", files[0].Version, version)
	}

	// Check versions applied in DB.
	expectedVersions := file.Versions{files[0].Version}
	versions, err := d.Versions()
	if err != nil {
		t.Errorf("Could not fetch versions: %s", err)
	}
	if !reflect.DeepEqual(versions, expectedVersions) {
		t.Errorf("Expected versions to be: %v, got: %v", expectedVersions, versions)
	}

	err = d.Migrate(files[1])
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.(*Driver).db.Query("SELECT id, data1, data2 FROM yolo"); err != nil {
		t.Errorf("Sequential migration failed: %v", err)
	}

	// Check versions applied in DB.
	expectedVersions = file.Versions{files[1].Version, files[0].Version}
	versions, err = d.Versions()
	if err != nil {
		t.Errorf("Could not fetch versions: %s", err)
	}
	if !reflect.DeepEqual(versions, expectedVersions) {
		t.Errorf("Expected versions to be: %v, got: %v", expectedVersions, versions)
	}

	err = d.Migrate(files[2])
	if err != nil {
		t.Fatal(err)
	}

	err = d.Migrate(files[3])
	if err == nil {
		t.Error("Expected test case to fail")
	}

	if err := d.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestSplitStatements(t *testing.T) {
	testCases := []struct {
		name string
		q    string
		want []string
	}{
		{"empty noop", "", []string{}},
		{"single query", "CREATE TABLE a id INT;", []string{"CREATE TABLE a id INT;"}},
		{"multiple queries", "CREATE TABLE a id INT; CREATE TABLE b id INT; ",
			[]string{"CREATE TABLE a id INT;", "CREATE TABLE b id INT;"},
		},
		{"with line breaks", "CREATE TABLE a id INT;\n\n\t CREATE TABLE b id INT; ",
			[]string{"CREATE TABLE a id INT;", "CREATE TABLE b id INT;"},
		},
	}
	for _, tc := range testCases {
		got := splitStatements(tc.q)
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("(%s) splitStatements(%q) = %q, want: %q", tc.name, tc.q, got, tc.want)
		}
	}
}
