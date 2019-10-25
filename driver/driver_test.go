// Package driver holds the driver interface.
package driver

import "testing"

func Test_getScheme(t *testing.T) {
	type args struct {
		url string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "MySQL",
			args: args{
				url: "mysql://root:@(localhost:3306)/db",
			},
			want: "mysql",
		},
		{
			name: "PostgreSQL",
			args: args{
				url: "postgres://root@localhost:3306/db",
			},
			want: "postgres",
		},
		{
			name: "Cassandra",
			args: args{
				url: "cassandra://localhost:3306/keyspace",
			},
			want: "cassandra",
		},
		{
			name: "SQLite",
			args: args{
				url: "sqlite3://database.sqlite",
			},
			want: "sqlite3",
		},
		{
			name: "invalid",
			args: args{
				url: "root@localhost",
			},
			want: "",
		},
		{
			name: "malformed mysql",
			args: args{
				url: "mysql:/root:@localhost",
			},
			want: "",
		},
		{
			name: "malformed mysql",
			args: args{
				url: ":mysql://root:@localhost",
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getScheme(tt.args.url); got != tt.want {
				t.Errorf("getScheme() = %v, want %v", got, tt.want)
			}
		})
	}
}
