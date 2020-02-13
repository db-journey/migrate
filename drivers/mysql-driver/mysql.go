// Package mysql implements the Driver interface.
package mysql

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/db-journey/migrate/v2/direction"
	"github.com/db-journey/migrate/v2/driver"
	"github.com/db-journey/migrate/v2/file"
	"github.com/go-sql-driver/mysql"
)

const versionsTableName = "schema_migrations"

// directives
const (
	directiveNotx    = "NOTX"
	directiveTxbegin = "TXBEGIN"
	directiveTxend   = "TXEND"
	directiveNoop    = ""
)

var fileTemplate = []byte(`
-- Each SQL statement MUST end with semicolon (;) FOLLOWED BY NEWLINE !
-- Whole migration will be executed inside transaction by default.
-- Place SQL between "-- TXBEGIN" and "-- TXEND" comments for custom transaction:
--   - you CAN have multiple separate transactions in single migration
--   - any SQL not wrapped into TXBEGIN - TXEND will be executed without transaction.
-- Add "-- NOTX" comment above all SQL to disable default migration. NOTE:
--   it's redundant when TXBEGIN/TXEND is used.
`)

func init() {
	driver.Register("mysql", "sql", fileTemplate, Open)
}

// Driver for MySQL
type Driver struct {
	db          *sql.DB
	versionConn *sql.Conn
}

// Open driver
func Open(url string) (driver.Driver, error) {
	drv := &Driver{}

	urlWithoutScheme := strings.SplitN(url, "mysql://", 2)
	if len(urlWithoutScheme) != 2 {
		return nil, errors.New("invalid mysql:// scheme")
	}

	// check if env vars vor mysql ssl connection are set and if yes use them
	// XXX: reading env vars in a library is not good, such stuff should be passed
	// from user (CLI code).
	if os.Getenv("MYSQL_SERVER_CA") != "" && os.Getenv("MYSQL_CLIENT_KEY") != "" && os.Getenv("MYSQL_CLIENT_CERT") != "" {
		rootCertPool := x509.NewCertPool()
		pem, err := ioutil.ReadFile(os.Getenv("MYSQL_SERVER_CA"))
		if err != nil {
			return nil, err
		}

		if ok := rootCertPool.AppendCertsFromPEM(pem); !ok {
			return nil, errors.New("Failed to append PEM")
		}

		clientCert := make([]tls.Certificate, 0, 1)
		certs, err := tls.LoadX509KeyPair(os.Getenv("MYSQL_CLIENT_CERT"), os.Getenv("MYSQL_CLIENT_KEY"))
		if err != nil {
			return nil, err
		}

		clientCert = append(clientCert, certs)
		mysql.RegisterTLSConfig("custom", &tls.Config{
			RootCAs:            rootCertPool,
			Certificates:       clientCert,
			InsecureSkipVerify: true,
		})

		urlWithoutScheme[1] += "&tls=custom"
	}

	db, err := sql.Open("mysql", urlWithoutScheme[1])
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	drv.db = db

	return drv, drv.ensureVersionTableExists()
}

// Close db connection
func (drv *Driver) Close() error {
	if drv.versionConn != nil {
		drv.versionConn.Close() // error is no big deal here.
	}
	return drv.db.Close()
}

// Execute sql
func (drv *Driver) Execute(sql string) error {
	_, err := drv.db.Exec(sql)
	return err
}

// Migrate runs migration.
// It locks schema_migrations table, so concurrent execution is safe.
func (drv *Driver) Migrate(f file.File) error {
	if drv.versionConn == nil {
		return errors.New("migrate must call Lock before Migrate")
	}
	if err := f.ReadContent(); err != nil {
		return err
	}

	migration, err := parseMigration(f.Content)
	if err != nil {
		return fmt.Errorf("failed to parse migration: %s", err)
	}

	err = migration.exec(drv.db)
	if err != nil {
		return err
	}

	versionUpdSQL := "INSERT INTO " + versionsTableName + " (version) VALUES (?)"
	if f.Direction == direction.Down {
		versionUpdSQL = "DELETE FROM " + versionsTableName + " WHERE version = ?"
	}
	if _, err = drv.versionConn.ExecContext(context.TODO(), versionUpdSQL, f.Version); err != nil {
		err = fmt.Errorf("migration %d was successfully applied, but failed to update schema_migrations table: %s", f.Version, err)
	}
	return err
}

// Version returns the current migration version.
func (drv *Driver) Version() (file.Version, error) {
	var version file.Version
	err := drv.versionConn.QueryRowContext(context.TODO(), "SELECT version FROM "+versionsTableName+" ORDER BY version DESC").Scan(&version)
	switch {
	case err == sql.ErrNoRows:
		return 0, nil
	case err != nil:
		return 0, err
	default:
		return version, nil
	}
}

// Versions returns the list of applied migrations.
func (drv *Driver) Versions() (file.Versions, error) {
	versions := file.Versions{}

	err := drv.initVersionConn()
	if err != nil {
		return nil, err
	}

	rows, err := drv.versionConn.QueryContext(context.TODO(), "SELECT version FROM "+versionsTableName+" ORDER BY version DESC")
	if err != nil {
		return versions, err
	}
	defer rows.Close()
	for rows.Next() {
		var version file.Version
		err := rows.Scan(&version)
		if err != nil {
			return versions, err
		}
		versions = append(versions, version)
	}
	err = rows.Err()
	return versions, err
}

// Lock schema_migrations table
func (drv *Driver) Lock() error {
	err := drv.initVersionConn()
	if err != nil {
		return err
	}
	_, err = drv.versionConn.ExecContext(context.TODO(), "LOCK TABLES "+versionsTableName+" WRITE")
	if err != nil {
		return fmt.Errorf("failed to lock %s table: %v", versionsTableName, err)
	}
	return nil
}

// Unlock schema_migrations table
func (drv *Driver) Unlock() error {
	if drv.versionConn == nil {
		return errors.New("not locked")
	}
	_, err := drv.versionConn.ExecContext(context.TODO(), "UNLOCK TABLES")
	if err != nil {
		return fmt.Errorf("failed to unlock %s table: %v", versionsTableName, err)
	}
	drv.versionConn.Close() // not a big deal if it fails to return connection to the pool
	drv.versionConn = nil
	return err
}

func (drv *Driver) initVersionConn() (err error) {
	if drv.versionConn == nil {
		drv.versionConn, err = drv.db.Conn(context.TODO())
	}
	return err
}

func (drv *Driver) ensureVersionTableExists() error {
	_, err := drv.db.Exec("CREATE TABLE IF NOT EXISTS " + versionsTableName + " (version bigint not null primary key);")
	if err != nil {
		return err
	}

	r := drv.db.QueryRow("SELECT data_type FROM information_schema.columns where table_name = ? and column_name = 'version'", versionsTableName)
	dataType := ""
	if err = r.Scan(&dataType); err != nil {
		return err
	}
	if dataType != "int" {
		return nil
	}
	_, err = drv.db.Exec("ALTER TABLE " + versionsTableName + " MODIFY version bigint")
	return err
}

func parseDirective(b []byte) string {
	b = bytes.TrimSpace(b)
	if !bytes.HasPrefix(b, []byte("-- ")) {
		return directiveNoop
	}
	return string(b[3 : len(b)-1])
}

type migrationSegment struct {
	statements     []string
	offsets        []int // line offset from beginning of file
	tx             bool
	txbegin, txend int // line numbers
}

type migration struct {
	// noTx determines if default transaction
	// should be disabled
	noTx     bool
	segments []migrationSegment
}

// parseMigration splits given SQL source into list of sql statements/transactions
// NOTE wrapping whole migration SQL into single transaction sucks,
// b/c some stuff like CREATE TABLE commits implicitly.
// Proper formatting is documented.
func parseMigration(b []byte) (*migration, error) {
	m := &migration{}
	lines := bytes.Split(b, []byte("\n"))
	for i := 0; i < len(lines); i++ {
		if len(bytes.TrimSpace(lines[i])) == 0 {
			continue
		}
		i = scrollEmpty(lines, i)
		if i < 0 {
			break
		}
		stmt := migrationSegment{}
		if !bytes.HasPrefix(bytes.TrimSpace(lines[i]), []byte("-- ")) {
			i = writeStmt(&stmt, lines, i)
			m.segments = append(m.segments, stmt)
			continue
		}
		i = scrollEmpty(lines, i)
		if i < 0 {
			break
		}
		switch parseDirective(lines[i]) {
		case directiveNotx:
			m.noTx = true
			break
		case directiveTxbegin:
			m.noTx = true
			stmt.tx = true
			stmt.txbegin = i + 1
			for ; i < len(lines); i++ {
				directive := parseDirective(lines[i])
				if directive != "" && directive != directiveTxend {
					return nil, fmt.Errorf("expected %q, got %q at line %d", directiveTxend, directive, i+1)
				}
				i = writeStmt(&stmt, lines, i)
			}
			stmt.txend = i + 1
			m.segments = append(m.segments, stmt)
			break
		case directiveNoop:
			break
		}
	}
	return m, nil
}

func (m migration) exec(db *sql.DB) (err error) {
	var tx *sql.Tx
	defer func() {
		if err != nil && tx != nil {
			tx.Rollback()
		}
	}()
	if !m.noTx {
		tx, err = db.Begin()
		if err != nil {
			return err
		}
		for _, seg := range m.segments {
			for i, stmt := range seg.statements {
				_, err = tx.Exec(stmt)
				if err != nil {
					return stmtExecErr(err, stmt, seg.offsets[i])
				}
			}
		}
		return tx.Commit()
	}
	for _, seg := range m.segments {
		if seg.tx {
			tx, err = db.Begin()
			if err != nil {
				return err
			}
			for i, stmt := range seg.statements {
				_, err = tx.Exec(stmt)
				if err != nil {
					return stmtExecErr(err, stmt, seg.offsets[i])
				}
			}
			return stmtCommitErr(tx.Commit(), seg)
		}
	}
	return nil
}

func stmtExecErr(err error, stmt string, stmtOffset int) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("Failed to exec SQL statement at line %d:\n%s\nError:%s", stmtOffset+1, stmt, err)
}

func stmtCommitErr(err error, s migrationSegment) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("Failed to commit lines %d-%d: %s", s.txbegin, s.txend, err)
}

// writeStmt is a DRYer for migration.parse
// returns last line index of statement.
func writeStmt(stmt *migrationSegment, lines [][]byte, i int) int {
	i = scrollEmpty(lines, i)
	if i < 0 {
		return len(lines) - 1
	}
	stmt.offsets = append(stmt.offsets, i)
	buf := &bytes.Buffer{}
	for ; i < len(lines); i++ {
		fmt.Fprintf(buf, "%s\n", lines[i])
		if bytes.HasSuffix(lines[i], []byte(";")) {
			break
		}
	}
	stmt.statements = append(stmt.statements, buf.String())
	return i
}

// scrollEmpty returns next non-empy line index.
func scrollEmpty(lines [][]byte, i int) int {
	for ; i < len(lines) && len(bytes.TrimSpace(lines[i])) == 0; i++ {
	}
	if i == len(lines)-1 && len(bytes.TrimSpace(lines[i])) == 0 {
		return -1
	}
	return i
}
