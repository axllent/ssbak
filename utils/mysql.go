package utils

import (
	"bufio"
	"compress/gzip"
	"database/sql"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/axllent/ssbak/app"
	"github.com/axllent/ssbak/internal/go-mysqldump"
	"github.com/go-sql-driver/mysql"
)

func mysqlConfig() *mysql.Config {
	addr := app.DB.Host
	if app.DB.Port != "" {
		addr += ":" + app.DB.Port
	}

	// Open connection to database
	config := mysql.NewConfig()
	config.User = app.DB.Username
	config.Passwd = app.DB.Password
	config.DBName = app.DB.Name
	config.Net = "tcp"
	config.Addr = addr

	return config
}

// MySQLDumpToGz uses mysqldump to stream a database dump directly into a gzip file
func MySQLDumpToGz(gzipFile string) error {
	config := mysqlConfig()

	f, err := os.Create(path.Clean(gzipFile))
	if err != nil {
		return fmt.Errorf("error creating database backup: %s", err.Error())
	}

	defer func() {
		if err := f.Close(); err != nil {
			fmt.Printf("error closing file: %s\n", err)
		}
	}()

	// Open connection to database
	db, err := sql.Open("mysql", config.FormatDSN())
	if err != nil {
		return fmt.Errorf("error opening database: %s", err.Error())
	}

	defer func() { _ = db.Close() }()

	gzw := gzip.NewWriter(f)

	defer func() {
		_ = gzw.Close()
		_ = gzw.Flush()
	}()

	app.Log(fmt.Sprintf("Dumping database to '%s'", gzipFile))

	dumper := mysqldump.Data{
		Connection:       db,
		Out:              gzw,
		MaxAllowedPacket: 512000, // 512KB
	}

	// Dump database to file
	if err = dumper.Dump(); err != nil {
		return fmt.Errorf("error dumping: %s", err.Error())
	}

	outSize, _ := CalcSize(gzipFile)
	app.Log(fmt.Sprintf("Wrote %s (%s)", gzipFile, ByteToHr(outSize)))

	// Close dumper, connected database and file stream.
	return dumper.Close()
}

// MySQLCreateDB a database, optionally dropping it
func MySQLCreateDB(dropDatabase bool) error {
	config := mysqlConfig()
	config.DBName = "" // reset the database name

	// Open connection to database
	db, err := sql.Open("mysql", config.FormatDSN())
	if err != nil {
		return fmt.Errorf("error opening database: %s", err.Error())
	}

	defer func() { _ = db.Close() }()

	createMsg := `Creating database (if not exists)`

	if dropDatabase {
		app.Log(fmt.Sprintf("Dropping database '%s'", app.DB.Name))
		if _, err := db.Exec("DROP DATABASE IF EXISTS `" + app.DB.Name + "`"); err != nil {
			return err
		}
		createMsg = `Creating database`
	}

	app.Log(fmt.Sprintf("%s '%s'", createMsg, app.DB.Name))
	_, err = db.Exec("CREATE DATABASE IF NOT EXISTS `" + app.DB.Name + "`")

	return err
}

// MySQLLoadFromGz loads a GZ database file into the database,
// streaming the gz file to the mysql cli.
func MySQLLoadFromGz(gzipSQLFile string) error {
	if !IsFile(gzipSQLFile) {
		return fmt.Errorf("file '%s' does not exist", gzipSQLFile)
	}

	f, err := os.Open(filepath.Clean(gzipSQLFile))
	if err != nil {
		return err
	}

	defer func() {
		if err := f.Close(); err != nil {
			fmt.Printf("error closing file: %s\n", err)
		}
	}()

	reader, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer func() { _ = reader.Close() }()

	config := mysqlConfig()

	// Open connection to database
	db, err := sql.Open("mysql", config.FormatDSN())
	if err != nil {
		return fmt.Errorf("error opening database: %s", err.Error())
	}

	defer func() { _ = db.Close() }()

	fileScanner := bufio.NewScanner(reader)
	fileScanner.Split(bufio.ScanLines)
	cBuffer := make([]byte, 0, bufio.MaxScanTokenSize)
	fileScanner.Buffer(cBuffer, bufio.MaxScanTokenSize*50) // Otherwise long lines crash the scanner

	app.Log(fmt.Sprintf("Importing database to '%s'", app.DB.Name))

	// ensure compatibility between MySQL & Mariadb, including older versions caused by
	// `STRICT_TRANS_TABLES` and `STRICT_ALL_TABLES`
	if _, err := db.Exec("SET sql_mode = '';"); err != nil {
		return err
	}

	sql := ""

	for fileScanner.Scan() {
		line := fileScanner.Text()
		if strings.HasPrefix(line, "/*!") || strings.HasPrefix(line, "--") || line == "" {
			// ignore comments and blank lines
		} else if strings.HasSuffix(line, ";") {
			// end of line, append and insert
			sql = sql + line + " "
			if strings.TrimSpace(sql) != "" {
				if _, err := db.Exec(sql); err != nil {
					return err
				}
			}
			// reset sql
			sql = ""
		} else {
			// append sql
			sql = sql + "\n" + line
		}
	}

	// if any sql remains, execute
	if strings.TrimSpace(sql) != "" {
		if _, err := db.Exec(sql); err != nil {
			return err
		}
	}

	app.Log(fmt.Sprintf("Imported '%s' to '%s'", gzipSQLFile, app.DB.Name))

	return err
}
