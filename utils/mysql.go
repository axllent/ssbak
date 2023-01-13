package utils

import (
	"bufio"
	"compress/gzip"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aliakseiz/go-mysqldump"
	"github.com/axllent/ssbak/app"
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

	f, err := os.Create(gzipFile)
	if err != nil {
		return fmt.Errorf("Error creating database backup: %s", err.Error())
	}

	defer func() {
		if err := f.Close(); err != nil {
			fmt.Printf("Error closing file: %s\n", err)
		}
	}()

	// Open connection to database
	db, err := sql.Open("mysql", config.FormatDSN())
	if err != nil {
		return fmt.Errorf("Error opening database: %s", err.Error())
	}

	defer db.Close()

	gzw := gzip.NewWriter(f)
	defer gzw.Close()
	defer gzw.Flush()

	app.Log(fmt.Sprintf("Dumping database to '%s'", gzipFile))

	dumper := mysqldump.Data{
		Connection:       db,
		Out:              gzw,
		MaxAllowedPacket: 512000, // 512KB
	}

	// Dump database to file
	if err = dumper.Dump(); err != nil {
		return fmt.Errorf("Error dumping: %s", err.Error())
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
		return fmt.Errorf("Error opening database: %s", err.Error())
	}

	defer db.Close()

	createMsg := `Creating database (if not exists)`

	if dropDatabase {
		app.Log(fmt.Sprintf("Dropping database `%s`", app.DB.Name))
		if _, err := db.Exec("DROP DATABASE IF EXISTS `" + app.DB.Name + "`"); err != nil {
			return err
		}
		createMsg = `Creating database`
	}

	app.Log(fmt.Sprintf("%s `%s`", createMsg, app.DB.Name))
	_, err = db.Exec("CREATE DATABASE IF NOT EXISTS `" + app.DB.Name + "`")

	return err
}

// MySQLLoadFromGz loads a GZ database file into the database,
// streaming the gz file to the mysql cli.
func MySQLLoadFromGz(gzipSQLFile string) error {
	if !IsFile(gzipSQLFile) {
		return fmt.Errorf("File '%s' does not exist", gzipSQLFile)
	}

	f, err := os.Open(filepath.Clean(gzipSQLFile))
	if err != nil {
		return err
	}

	defer func() {
		if err := f.Close(); err != nil {
			fmt.Printf("Error closing file: %s\n", err)
		}
	}()

	reader, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer reader.Close()

	config := mysqlConfig()

	// Open connection to database
	db, err := sql.Open("mysql", config.FormatDSN())
	if err != nil {
		return fmt.Errorf("Error opening database: %s", err.Error())
	}

	defer db.Close()

	fileScanner := bufio.NewScanner(reader)
	fileScanner.Split(bufio.ScanLines)
	cbuffer := make([]byte, 0, bufio.MaxScanTokenSize)
	fileScanner.Buffer(cbuffer, bufio.MaxScanTokenSize*50) // Otherwise long lines crash the scanner

	app.Log(fmt.Sprintf("Importing database `%s`", app.DB.Name))

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

	app.Log(fmt.Sprintf("Imported '%s' to `%s`", gzipSQLFile, app.DB.Name))

	return err
}
