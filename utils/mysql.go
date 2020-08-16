package utils

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"

	"github.com/axllent/ssbak/app"
)

// MySQLDumpToGz uses mysqldump to stream a database dump directly into a gzip file
func MySQLDumpToGz(gzipFile string) error {
	mysqldump, err := Which("mysqldump")
	if err != nil {
		return err
	}

	args := []string{"--skip-opt",
		"--add-drop-table",
		"--extended-insert",
		"--create-options",
		"--quick",
		"--set-charset",
		"--default-character-set=utf8",
		"--compress",
	}

	args = append(args, "-h", app.DBHost, "-u", app.DBUsername, app.DBName)

	cmd := exec.Command(mysqldump, args...)

	if app.DBPassword != "" {
		// Export MySQL password
		cmd.Env = append(os.Environ(), "MYSQL_PWD="+app.DBPassword)
	}

	app.Log(fmt.Sprintf("Dumping database to '%s'", gzipFile))

	f, err := os.Create(gzipFile)
	if err != nil {
		return err
	}
	defer f.Close()

	gzw := gzip.NewWriter(f)
	defer gzw.Close()
	defer gzw.Flush()

	var errbuf bytes.Buffer
	cmd.Stderr = &errbuf

	pipe, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}
	if _, err := io.Copy(gzw, pipe); err != nil {
		return err
	}

	if errbuf.String() != "" {
		return errors.New(errbuf.String())
	}

	outSize, _ := DirSize(gzipFile)
	app.Log(fmt.Sprintf("Wrote %s (%s)", gzipFile, outSize))

	return nil
}

// CreateDatabase creates a database, optionally dropping it
func CreateDatabase(dropDatabase bool) error {
	mysql, err := Which("mysql")
	if err != nil {
		return err
	}

	args := []string{
		"--default-character-set=utf8",
		"--compress",
	}

	sql := "CREATE DATABASE IF NOT EXISTS `" + app.DBName + "`"
	if dropDatabase {
		app.Log(fmt.Sprintf("Dropping database `%s`", app.DBName))
		sql = "DROP DATABASE IF EXISTS `" + app.DBName + "`; " + sql
	}

	app.Log(fmt.Sprintf("Creating database (if not exists) `%s`", app.DBName))

	args = append(args, "-h", app.DBHost, "-u", app.DBUsername, "-e", sql)

	cmd := exec.Command(mysql, args...)

	if app.DBPassword != "" {
		// Export MySQL password
		cmd.Env = append(os.Environ(), "MYSQL_PWD="+app.DBPassword)
	}

	var errbuf bytes.Buffer
	cmd.Stderr = &errbuf
	if err := cmd.Run(); err != nil {
		return err
	}

	if errbuf.String() != "" {
		return errors.New(errbuf.String())
	}

	return nil
}

// LoadDatabaseFromGz loads a GZ database file into the database,
// streaming the gz file to the mysql cli.
func LoadDatabaseFromGz(gzipSQLFile string) error {
	mysql, err := Which("mysql")
	if err != nil {
		return err
	}

	if !IsFile(gzipSQLFile) {
		return fmt.Errorf("File '%s' does not exist", gzipSQLFile)
	}

	args := []string{"--default-character-set=utf8"}

	args = append(args, "-h", app.DBHost, "-u", app.DBUsername, app.DBName)

	cmd := exec.Command(mysql, args...)

	if app.DBPassword != "" {
		// Export MySQL password
		cmd.Env = append(os.Environ(), "MYSQL_PWD="+app.DBPassword)
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		log.Fatal(err)
	}

	f, err := os.Open(gzipSQLFile)
	if err != nil {
		return err
	}

	defer f.Close()

	reader, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer reader.Close()

	go func() {
		defer stdin.Close()
		io.Copy(stdin, reader)
	}()

	if _, err := cmd.CombinedOutput(); err != nil {
		return err
	}

	app.Log(fmt.Sprintf("Imported '%s' to `%s`", gzipSQLFile, app.DBName))

	return nil
}
