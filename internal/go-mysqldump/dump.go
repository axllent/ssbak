package mysqldump

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"reflect"
	"text/template"
	"time"
)

// Data struct to configure dump behavior.
// Out:              Stream to write to
// Connection:       Database connection to dump
// IgnoreTables:     Mark sensitive tables to ignore
// MaxAllowedPacket: Sets the largest packet size to use in backups
// LockTables:       Lock all tables for the duration of the dump.
type Data struct {
	Out              io.Writer
	Connection       *sql.DB
	IgnoreTables     []string // TODO store in a map
	MaxAllowedPacket int
	LockTables       bool
	DBName           string

	tx           *sql.Tx
	headerTmpl   *template.Template
	recreateTmpl *template.Template
	tableTmpl    *template.Template
	footerTmpl   *template.Template
	err          error
}

type recreate struct {
	Database string
}

// Table struct contains variables for table template.
type Table struct {
	Name   string
	Err    error
	IsView bool

	data   *Data
	rows   *sql.Rows
	values []interface{}
}

// MetaData struct contains variables for header and footer templates.
type MetaData struct {
	DumpVersion   string
	ServerVersion string
	CompleteTime  string
}

const (
	// Version of this plugin for easy reference.
	Version = "0.5.1"

	defaultMaxAllowedPacket = 4194304
)

// takes a *MetaData.
const headerTmpl = `-- Go SQL Dump {{ .DumpVersion }}
--
-- ------------------------------------------------------
-- Server version	{{ .ServerVersion }}

/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
 SET NAMES utf8mb4 ;
/*!40103 SET @OLD_TIME_ZONE=@@TIME_ZONE */;
/*!40103 SET TIME_ZONE='+00:00' */;
/*!40014 SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0 */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;
/*!40111 SET @OLD_SQL_NOTES=@@SQL_NOTES, SQL_NOTES=0 */;
`

// takes a *MetaData.
const footerTmpl = `/*!40103 SET TIME_ZONE=@OLD_TIME_ZONE */;

/*!40101 SET SQL_MODE=@OLD_SQL_MODE */;
/*!40014 SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS */;
/*!40014 SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
/*!40111 SET SQL_NOTES=@OLD_SQL_NOTES */;

-- Dump completed on {{ .CompleteTime }}
`

const tableTmpl = `
--
-- Table structure for table {{ .NameEsc }}
--

{{ if .IsView }}
DROP VIEW IF EXISTS {{ .NameEsc }};
{{ else }}
DROP TABLE IF EXISTS {{ .NameEsc }};
{{ end }}
/*!40101 SET @saved_cs_client     = @@character_set_client */;
 SET character_set_client = utf8mb4 ;
{{ .CreateSQL }};
/*!40101 SET character_set_client = @saved_cs_client */;

{{ if not .IsView }}
--
-- Dumping data for table {{ .NameEsc }}
--

LOCK TABLES {{ .NameEsc }} WRITE;
/*!40000 ALTER TABLE {{ .NameEsc }} DISABLE KEYS */;
{{ range $value := .Stream }}
{{- $value }}
{{ end -}}
/*!40000 ALTER TABLE {{ .NameEsc }} ENABLE KEYS */;
UNLOCK TABLES;
{{ end }}
`

// Takes a *recreate.
const recreateTmpl = " " +
	"DROP DATABASE IF EXISTS `{{ .Database }}`;" +
	"CREATE DATABASE `{{ .Database }}` CHARACTER SET = 'utf8' COLLATE = 'utf8_general_ci';" +
	"USE `{{ .Database }}`;"

const nullType = "NULL"

var (
	errUnexpectedTable = errors.New("returned table is not the same as requested table")
	errDoubleInit      = errors.New("cannot init twice")
	errNoColumns       = errors.New("no columns in table")
)

// Dump data using struct.
func (data *Data) Dump() error {
	meta := MetaData{
		DumpVersion: Version,
	}

	if data.MaxAllowedPacket == 0 {
		data.MaxAllowedPacket = defaultMaxAllowedPacket
	}

	if err := data.GetTemplates(); err != nil {
		return err
	}

	// Start the read only transaction and defer the rollback until the end.
	// This way the database will have the exact state it did at the beginning of
	// the backup and nothing can be accidentally committed.
	if err := data.Begin(); err != nil {
		return err
	}
	defer data.rollback() // nolint:errcheck

	if err := meta.UpdateServerVersion(data); err != nil {
		return err
	}

	if err := data.headerTmpl.Execute(data.Out, meta); err != nil {
		return err
	}

	if data.DBName != "" {
		if err := data.recreateTmpl.Execute(data.Out, recreate{Database: data.DBName}); err != nil {
			return err
		}
	}

	tables, err := data.GetTables()
	if err != nil {
		return err
	}

	// Lock all tables before dumping if present.
	if data.LockTables && len(tables) > 0 {
		var b bytes.Buffer

		b.WriteString("LOCK TABLES ")

		for index, name := range tables {
			if index != 0 {
				b.WriteString(",")
			}

			b.WriteString("`" + name + "` READ /*!32311 LOCAL */")
		}

		if _, err := data.Connection.Exec(b.String()); err != nil {
			return err
		}

		defer data.Connection.Exec("UNLOCK TABLES") // nolint:errcheck
	}

	for _, name := range tables {
		if err = data.dumpTable(name); err != nil {
			return err
		}
	}

	if data.err != nil {
		return data.err
	}

	meta.CompleteTime = time.Now().String()

	return data.footerTmpl.Execute(data.Out, meta)
}

// MARK: - Private methods

// Begin starts a read only transaction that will be whatever the database was when it was called.
func (data *Data) Begin() (err error) {
	data.tx, err = data.Connection.BeginTx(context.Background(), &sql.TxOptions{
		Isolation: sql.LevelRepeatableRead,
		ReadOnly:  true,
	})

	return
}

// rollback cancels the transaction.
func (data *Data) rollback() error {
	return data.tx.Rollback()
}

func (data *Data) dumpTable(name string) error {
	if data.err != nil {
		return data.err
	}

	table, err := data.CreateTable(name)
	if err != nil {
		return err
	}

	return data.WriteTable(table)
}

// WriteTable fills table template with provided *table data.
func (data *Data) WriteTable(table *Table) error {
	if err := data.tableTmpl.Execute(data.Out, table); err != nil {
		return err
	}

	return table.Err
}

// GetTemplates initializes the templates on data from the constants in this file.
func (data *Data) GetTemplates() (err error) {
	data.headerTmpl, err = template.New("mysqldumpHeader").Parse(headerTmpl)
	if err != nil {
		return
	}

	data.recreateTmpl, err = template.New("mysqldumpRecreate").Parse(recreateTmpl)
	if err != nil {
		return
	}

	data.tableTmpl, err = template.New("mysqldumpTable").Parse(tableTmpl)
	if err != nil {
		return
	}

	data.footerTmpl, err = template.New("mysqldumpTable").Parse(footerTmpl)
	if err != nil {
		return
	}

	return
}

// GetTables returns database tables.
func (data *Data) GetTables() ([]string, error) {
	tables := make([]string, 0)

	rows, err := data.tx.Query("SHOW TABLES")
	if err != nil {
		return tables, err
	}
	defer rows.Close()

	for rows.Next() {
		var table sql.NullString

		if err := rows.Scan(&table); err != nil {
			return tables, err
		}

		if table.Valid && !data.isIgnoredTable(table.String) {
			tables = append(tables, table.String)
		}
	}

	return tables, rows.Err()
}

func (data *Data) isIgnoredTable(name string) bool {
	for _, item := range data.IgnoreTables {
		if item == name {
			return true
		}
	}

	return false
}

// UpdateServerVersion fetches server version and stores it in Data.
func (meta *MetaData) UpdateServerVersion(data *Data) (err error) {
	var serverVersion sql.NullString

	err = data.tx.QueryRow("SELECT version()").Scan(&serverVersion)
	meta.ServerVersion = serverVersion.String

	return
}

// CreateTable initializes a Table struct.
func (data *Data) CreateTable(name string) (*Table, error) {
	table := &Table{
		Name: name,
		data: data,
	}

	var tableType string

	err := data.tx.QueryRow("SELECT table_type FROM information_schema.tables WHERE table_name = ?", name).Scan(&tableType)
	if err != nil {
		return nil, err
	}

	if tableType == "VIEW" {
		table.IsView = true
	}

	return table, nil
}

func (table *Table) NameEsc() string {
	return "`" + table.Name + "`"
}

func (table *Table) CreateSQL() (string, error) {
	var (
		nameReturn, createSQL sql.NullString
		createSQLQuery        string
	)

	if table.IsView {
		createSQLQuery = "SHOW CREATE VIEW " + table.NameEsc()
	} else {
		createSQLQuery = "SHOW CREATE TABLE " + table.NameEsc()
	}

	if err := table.data.tx.QueryRow(createSQLQuery).Scan(&nameReturn, &createSQL); err != nil {
		return "", err
	}

	if nameReturn.String != table.Name {
		return "", errUnexpectedTable
	}

	return createSQL.String, nil
}

func (table *Table) Init() (err error) {
	if len(table.values) != 0 {
		return errDoubleInit
	}

	// nolint:rowserrcheck, sqlclosecheck
	table.rows, err = table.data.tx.Query(fmt.Sprintf("SELECT * FROM %s", table.NameEsc()))
	if err != nil {
		return err
	}

	columns, err := table.rows.Columns()
	if err != nil {
		return err
	}

	if len(columns) == 0 {
		return errNoColumns
	}

	tt, err := table.rows.ColumnTypes()
	if err != nil {
		return err
	}

	var t reflect.Type

	table.values = make([]interface{}, len(tt))

	for i, tp := range tt {
		st := tp.ScanType()

		switch {
		case tp.DatabaseTypeName() == "BLOB":
			t = reflect.TypeOf(sql.RawBytes{})
		case st != nil && (st.Kind() == reflect.Int ||
			st.Kind() == reflect.Int8 ||
			st.Kind() == reflect.Int16 ||
			st.Kind() == reflect.Int32 ||
			st.Kind() == reflect.Int64):
			t = reflect.TypeOf(sql.NullInt64{})
		default:
			t = reflect.TypeOf(sql.NullString{})
		}

		table.values[i] = reflect.New(t).Interface()
	}

	return nil
}

func (table *Table) Next() bool {
	if table.rows == nil {
		if err := table.Init(); err != nil {
			table.Err = err

			return false
		}
	}
	// Fallthrough
	if table.rows.Next() {
		if err := table.rows.Scan(table.values...); err != nil {
			table.Err = err

			return false
		} else if err := table.rows.Err(); err != nil {
			table.Err = err

			return false
		}
	} else {
		table.rows.Close()
		table.rows = nil

		return false
	}

	return true
}

func (table *Table) RowValues() string {
	return table.RowBuffer().String()
}

func (table *Table) RowBuffer() *bytes.Buffer {
	var b bytes.Buffer

	b.WriteString("(")

	for key, value := range table.values {
		if key != 0 {
			b.WriteString(",")
		}

		switch s := value.(type) {
		case nil:
			b.WriteString(nullType)
		case *sql.NullString:
			if s.Valid {
				fmt.Fprintf(&b, "'%s'", Sanitize(s.String))
			} else {
				b.WriteString(nullType)
			}
		case *sql.NullInt64:
			if s.Valid {
				fmt.Fprintf(&b, "%d", s.Int64)
			} else {
				b.WriteString(nullType)
			}
		case *sql.RawBytes:
			if len(*s) == 0 {
				b.WriteString(nullType)
			} else {
				fmt.Fprintf(&b, "_binary '%s'", Sanitize(string(*s)))
			}
		default:
			fmt.Fprintf(&b, "'%s'", value)
		}
	}

	b.WriteString(")")

	return &b
}

func (table *Table) Stream() <-chan string {
	valueOut := make(chan string, 1)

	go func() {
		defer close(valueOut)

		var insert bytes.Buffer

		for table.Next() {
			b := table.RowBuffer()
			// Truncate our insert if it won't fit
			if insert.Len() != 0 && insert.Len()+b.Len() > table.data.MaxAllowedPacket-1 {
				insert.WriteString(";")
				valueOut <- insert.String()

				insert.Reset()
			}

			if insert.Len() == 0 {
				fmt.Fprintf(&insert, "INSERT INTO %s VALUES ", table.NameEsc())
			} else {
				insert.WriteString(",")
			}

			b.WriteTo(&insert) // nolint:errcheck
		}

		if insert.Len() != 0 {
			insert.WriteString(";")
			valueOut <- insert.String()
		}
	}()

	return valueOut
}

// SetData assigns the data pointer to the table.
func (table *Table) SetData(data *Data) {
	table.data = data
}
