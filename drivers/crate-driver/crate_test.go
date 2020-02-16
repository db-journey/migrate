package crate

import (
	"fmt"
	"os"
	"testing"

	"github.com/db-journey/migrate/v2/direction"
	"github.com/db-journey/migrate/v2/driver"
	"github.com/db-journey/migrate/v2/file"
)

func TestContentSplit(t *testing.T) {
	content := `CREATE TABLE users (user_id STRING primary key, first_name STRING, last_name STRING, email STRING, password_hash STRING) CLUSTERED INTO 3 shards WITH (number_of_replicas = 0);
CREATE TABLE units (unit_id STRING primary key, name STRING, members array(string)) CLUSTERED INTO 3 shards WITH (number_of_replicas = 0);
CREATE TABLE available_connectors (technology_id STRING primary key, description STRING, icon STRING, link STRING, configuration_parameters array(object as (name STRING, type STRING))) CLUSTERED INTO 3 shards WITH (number_of_replicas = 0);
	`

	lines := splitContent(content)
	if len(lines) != 3 {
		t.Errorf("Expected 3 lines, but got %d", len(lines))
	}

	if lines[0] != "CREATE TABLE users (user_id STRING primary key, first_name STRING, last_name STRING, email STRING, password_hash STRING) CLUSTERED INTO 3 shards WITH (number_of_replicas = 0)" {
		t.Error("Line does not match expected output")
	}

	if lines[1] != "CREATE TABLE units (unit_id STRING primary key, name STRING, members array(string)) CLUSTERED INTO 3 shards WITH (number_of_replicas = 0)" {
		t.Error("Line does not match expected output")
	}

	if lines[2] != "CREATE TABLE available_connectors (technology_id STRING primary key, description STRING, icon STRING, link STRING, configuration_parameters array(object as (name STRING, type STRING))) CLUSTERED INTO 3 shards WITH (number_of_replicas = 0)" {
		t.Error("Line does not match expected output")
	}
}

func TestMigrate(t *testing.T) {
	host := os.Getenv("CRATE_PORT_4200_TCP_ADDR")
	port := os.Getenv("CRATE_PORT_4200_TCP_PORT")

	url := fmt.Sprintf("crate://%s:%s", host, port)

	var err error
	var driver driver.Driver
	if driver, err = Open(url); err != nil {
		t.Fatal(err)
	}

	successFiles := []file.File{
		{
			Path:      "/foobar",
			FileName:  "20161122192905_foobar.up.sql",
			Version:   20161122192905,
			Name:      "foobar",
			Direction: direction.Up,
			Content: []byte(`
                CREATE TABLE yolo (
                    id integer primary key,
                    msg string
                );
            `),
		},
		{
			Path:      "/foobar",
			FileName:  "20161122192905_foobar.down.sql",
			Version:   20161122192905,
			Name:      "foobar",
			Direction: direction.Down,
			Content: []byte(`
                DROP TABLE yolo;
            `),
		},
	}

	failFiles := []file.File{
		{
			Path:      "/foobar",
			FileName:  "20161122193005_foobar.up.sql",
			Version:   20161122193005,
			Name:      "foobar",
			Direction: direction.Up,
			Content: []byte(`
                CREATE TABLE error (
                    id THIS WILL CAUSE AN ERROR
                )
            `),
		},
	}

	for _, file := range successFiles {
		err := driver.Migrate(file)
		if err != nil {
			t.Fatal(err)
		}
	}

	for _, file := range failFiles {
		err := driver.Migrate(file)
		if err == nil {
			t.Fatal("Migration should have failed but succeeded")
		}
	}

	if err := driver.Close(); err != nil {
		t.Fatal(err)
	}
}
