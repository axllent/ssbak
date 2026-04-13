package sspak

import (
	"bufio"
	"compress/gzip"
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/aliakseiz/go-mysqldump"
	"github.com/axllent/ssbak/app"
	"github.com/axllent/ssbak/internal/utils"
	"github.com/go-sql-driver/mysql"
	"github.com/klauspost/compress/zstd"
)

// AddDatabase will dump a database and compress it using either gzip or zstd
func (f *File) AddDatabase() error {
	config := genMySQLConfig()

	f.DatabaseFile = filepath.Join(f.TempFolder, "database.sql.gz")
	if UseZSTD {
		f.DatabaseFile = filepath.Join(f.TempFolder, "database.sql.zst")
	}

	file, err := os.Create(f.DatabaseFile)
	if err != nil {
		return fmt.Errorf("error creating database backup: %s", err.Error())
	}

	defer func() {
		if err := file.Close(); err != nil {
			fmt.Printf("error closing file: %s\n", err)
		}
	}()

	// Open connection to database
	db, err := sql.Open("mysql", config.FormatDSN())
	if err != nil {
		return fmt.Errorf("error opening database: %s", err.Error())
	}

	defer func() { _ = db.Close() }()

	var compressor io.WriteCloser
	if UseZSTD {
		compressor, err = zstd.NewWriter(file)
		if err != nil {
			return fmt.Errorf("error creating zstd writer: %s", err.Error())
		}
	} else {
		compressor = gzip.NewWriter(file)
	}

	app.Log(fmt.Sprintf("Dumping database to '%s'", f.DatabaseFile))

	dumper := mysqldump.Data{
		Connection:       db,
		Out:              compressor,
		MaxAllowedPacket: 512000, // 512KB
	}

	// Dump database to file
	if err = dumper.Dump(); err != nil {
		_ = compressor.Close()
		return fmt.Errorf("error dumping: %s", err.Error())
	}

	// Close dumper first
	if err = dumper.Close(); err != nil {
		_ = compressor.Close()
		return fmt.Errorf("error closing dumper: %s", err.Error())
	}

	// Then close and flush compression writer
	if err = compressor.Close(); err != nil {
		return fmt.Errorf("error closing compressor: %s", err.Error())
	}

	outSize, _ := utils.CalcSize(f.DatabaseFile)
	app.Log(fmt.Sprintf("Wrote %s (%s)", f.DatabaseFile, utils.ByteToHr(outSize)))

	return nil
}

// AddDatabaseFromFile compresses an existing SQL file into the temp folder using
// either gzip or zstd (controlled by UseZSTD), and sets f.DatabaseFile.
func (f *File) AddDatabaseFromFile(sqlFile string) error {
	f.DatabaseFile = filepath.Join(f.TempFolder, "database.sql.gz")
	if UseZSTD {
		f.DatabaseFile = filepath.Join(f.TempFolder, "database.sql.zst")
	}

	src, err := os.Open(filepath.Clean(sqlFile))
	if err != nil {
		return err
	}
	defer func() {
		if err := src.Close(); err != nil {
			fmt.Printf("error closing file: %s\n", err)
		}
	}()

	outFile, err := os.Create(f.DatabaseFile)
	if err != nil {
		return err
	}

	inSize, _ := utils.CalcSize(sqlFile)
	app.Log(fmt.Sprintf("Compressing '%s' (%s) to '%s'", sqlFile, utils.ByteToHr(inSize), f.DatabaseFile))

	var compressor io.WriteCloser
	if UseZSTD {
		compressor, err = zstd.NewWriter(outFile)
		if err != nil {
			_ = outFile.Close()
			return fmt.Errorf("error creating zstd writer: %s", err.Error())
		}
	} else {
		compressor = gzip.NewWriter(outFile)
	}

	if _, err := io.Copy(compressor, src); err != nil {
		_ = compressor.Close()
		_ = outFile.Close()
		return err
	}

	if err := compressor.Close(); err != nil {
		_ = outFile.Close()
		return err
	}

	outSize, _ := utils.CalcSize(f.DatabaseFile)
	app.Log(fmt.Sprintf("Wrote '%s' (%s)", f.DatabaseFile, utils.ByteToHr(outSize)))

	return outFile.Close()
}

// LoadDatabase creates the target database (optionally dropping it first) and
// imports the SQL dump from f.DatabaseFile, supporting both gzip and zstd.
func (f *File) LoadDatabase(dropDatabase bool) error {
	config := genMySQLConfig()
	configNoDB := *config
	configNoDB.DBName = ""

	adminDB, err := sql.Open("mysql", configNoDB.FormatDSN())
	if err != nil {
		return fmt.Errorf("error opening database connection: %s", err.Error())
	}
	defer func() { _ = adminDB.Close() }()

	if dropDatabase {
		app.Log(fmt.Sprintf("Dropping database '%s'", app.DB.Name))
		if _, err := adminDB.Exec("DROP DATABASE IF EXISTS `" + app.DB.Name + "`"); err != nil {
			return err
		}
		app.Log(fmt.Sprintf("Creating database '%s'", app.DB.Name))
	} else {
		app.Log(fmt.Sprintf("Creating database (if not exists) '%s'", app.DB.Name))
	}

	if _, err := adminDB.Exec("CREATE DATABASE IF NOT EXISTS `" + app.DB.Name + "`"); err != nil {
		return err
	}

	// Import the dump
	file, err := os.Open(filepath.Clean(f.DatabaseFile))
	if err != nil {
		return err
	}
	defer func() {
		if err := file.Close(); err != nil {
			fmt.Printf("error closing file: %s\n", err)
		}
	}()

	var reader io.ReadCloser
	if strings.HasSuffix(f.DatabaseFile, ".zst") {
		zstdDecoder, err := zstd.NewReader(file)
		if err != nil {
			return fmt.Errorf("error creating zstd reader: %s", err.Error())
		}
		reader = zstdDecoder.IOReadCloser()
	} else {
		reader, err = gzip.NewReader(file)
		if err != nil {
			return fmt.Errorf("error creating gzip reader: %s", err.Error())
		}
	}
	defer func() { _ = reader.Close() }()

	db, err := sql.Open("mysql", config.FormatDSN())
	if err != nil {
		return fmt.Errorf("error opening database: %s", err.Error())
	}
	defer func() { _ = db.Close() }()

	scanner := bufio.NewScanner(reader)
	scanner.Split(bufio.ScanLines)
	buf := make([]byte, 0, bufio.MaxScanTokenSize)
	// ~32MB buffer to handle very long lines
	scanner.Buffer(buf, bufio.MaxScanTokenSize*500)

	app.Log(fmt.Sprintf("Importing database to '%s'", app.DB.Name))

	// Ensure compatibility with MySQL & MariaDB across strict mode variants
	if _, err := db.Exec("SET sql_mode = '';"); err != nil {
		return err
	}

	stmt := ""
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, "/*!") || strings.HasPrefix(line, "--") || line == "":
			// skip comments and blank lines
		case strings.HasSuffix(line, ";"):
			stmt += line + " "
			if strings.TrimSpace(stmt) != "" {
				if _, err := db.Exec(stmt); err != nil {
					return err
				}
			}
			stmt = ""
		default:
			stmt += "\n" + line
		}
	}

	if strings.TrimSpace(stmt) != "" {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}

	app.Log(fmt.Sprintf("Imported '%s' to '%s'", f.DatabaseFile, app.DB.Name))

	return nil
}

func genMySQLConfig() *mysql.Config {
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
