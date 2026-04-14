//go:build integration

package sspak

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/axllent/ssbak/app"
	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// configureDBFromEnv populates app.DB from environment variables and skips the
// test if TEST_DB_HOST is not set (allows running the suite locally without MariaDB).
func configureDBFromEnv(t *testing.T) {
	t.Helper()

	host := os.Getenv("TEST_DB_HOST")
	if host == "" {
		t.Skip("TEST_DB_HOST not set; skipping DB integration tests")
	}

	app.DB.Host = host
	app.DB.Port = os.Getenv("TEST_DB_PORT")
	app.DB.Username = os.Getenv("TEST_DB_USER")
	app.DB.Password = os.Getenv("TEST_DB_PASS")
	app.DB.Name = os.Getenv("TEST_DB_NAME")

	UseZSTD = false
	app.OnlyDB = false
	app.OnlyAssets = false
	t.Cleanup(func() {
		UseZSTD = false
		app.OnlyDB = false
		app.OnlyAssets = false
		app.TempDir = ""
	})
}

func adminDSN() string {
	cfg := genMySQLConfig()
	cfg.DBName = ""
	return cfg.FormatDSN()
}

func dbDSN() string {
	return genMySQLConfig().FormatDSN()
}

// seedDB drops and recreates the test database with a known table and rows.
func seedDB(t *testing.T) {
	t.Helper()

	admin, err := sql.Open("mysql", adminDSN())
	require.NoError(t, err)
	defer admin.Close()

	_, err = admin.Exec("DROP DATABASE IF EXISTS `" + app.DB.Name + "`")
	require.NoError(t, err)
	_, err = admin.Exec("CREATE DATABASE `" + app.DB.Name + "`")
	require.NoError(t, err)

	db, err := sql.Open("mysql", dbDSN())
	require.NoError(t, err)
	defer db.Close()

	_, err = db.Exec(`CREATE TABLE greetings (id INT PRIMARY KEY AUTO_INCREMENT, message VARCHAR(255))`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO greetings (message) VALUES ('hello'), ('world')`)
	require.NoError(t, err)
}

// rowCount returns the number of rows in table within the configured test database.
func rowCount(t *testing.T, table string) int {
	t.Helper()

	db, err := sql.Open("mysql", dbDSN())
	require.NoError(t, err)
	defer db.Close()

	var n int
	require.NoError(t, db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM `%s`", table)).Scan(&n))

	return n
}

func TestAddDatabaseGzipIntegration(t *testing.T) {
	configureDBFromEnv(t)
	seedDB(t)

	f := &File{TempFolder: t.TempDir()}
	require.NoError(t, f.AddDatabase())

	assert.FileExists(t, f.DatabaseFile)
	assert.Contains(t, f.DatabaseFile, "database.sql.gz")

	info, err := os.Stat(f.DatabaseFile)
	require.NoError(t, err)
	assert.Greater(t, info.Size(), int64(0))
}

func TestAddDatabaseZSTDIntegration(t *testing.T) {
	configureDBFromEnv(t)
	UseZSTD = true
	seedDB(t)

	f := &File{TempFolder: t.TempDir()}
	require.NoError(t, f.AddDatabase())

	assert.FileExists(t, f.DatabaseFile)
	assert.Contains(t, f.DatabaseFile, "database.sql.zst")
}

func TestLoadDatabaseGzipIntegration(t *testing.T) {
	configureDBFromEnv(t)
	seedDB(t)

	f := &File{TempFolder: t.TempDir()}
	require.NoError(t, f.AddDatabase())

	// Wipe the DB then restore
	admin, err := sql.Open("mysql", adminDSN())
	require.NoError(t, err)
	defer admin.Close()
	_, err = admin.Exec("DROP DATABASE IF EXISTS `" + app.DB.Name + "`")
	require.NoError(t, err)

	require.NoError(t, f.LoadDatabase(false))
	assert.Equal(t, 2, rowCount(t, "greetings"))
}

func TestLoadDatabaseDropIntegration(t *testing.T) {
	configureDBFromEnv(t)
	seedDB(t)

	f := &File{TempFolder: t.TempDir()}
	require.NoError(t, f.AddDatabase())

	// Restore with dropDatabase=true — should drop and recreate the DB
	require.NoError(t, f.LoadDatabase(true))
	assert.Equal(t, 2, rowCount(t, "greetings"))
}

func TestLoadDatabaseZSTDIntegration(t *testing.T) {
	configureDBFromEnv(t)
	UseZSTD = true
	seedDB(t)

	f := &File{TempFolder: t.TempDir()}
	require.NoError(t, f.AddDatabase())

	admin, err := sql.Open("mysql", adminDSN())
	require.NoError(t, err)
	defer admin.Close()
	_, err = admin.Exec("DROP DATABASE IF EXISTS `" + app.DB.Name + "`")
	require.NoError(t, err)

	require.NoError(t, f.LoadDatabase(false))
	assert.Equal(t, 2, rowCount(t, "greetings"))
}

func TestDatabaseRoundtripThroughSSPakIntegration(t *testing.T) {
	configureDBFromEnv(t)
	seedDB(t)

	tmpDir := t.TempDir()

	// Dump and pack into .sspak
	f := &File{TempFolder: tmpDir}
	require.NoError(t, f.AddDatabase())

	sspakPath := filepath.Join(tmpDir, "backup.sspak")
	require.NoError(t, f.Write(sspakPath))
	assert.FileExists(t, sspakPath)

	// Wipe the DB
	admin, err := sql.Open("mysql", adminDSN())
	require.NoError(t, err)
	defer admin.Close()
	_, err = admin.Exec("DROP DATABASE IF EXISTS `" + app.DB.Name + "`")
	require.NoError(t, err)

	// Open .sspak and restore
	app.TempDir = filepath.Join(t.TempDir(), "extracted")
	opened, err := Open(sspakPath)
	require.NoError(t, err)
	require.NoError(t, opened.LoadDatabase(false))

	// Verify restored row values
	db, err := sql.Open("mysql", dbDSN())
	require.NoError(t, err)
	defer db.Close()

	rows, err := db.Query("SELECT message FROM greetings ORDER BY id")
	require.NoError(t, err)
	defer rows.Close()

	var messages []string
	for rows.Next() {
		var msg string
		require.NoError(t, rows.Scan(&msg))
		messages = append(messages, msg)
	}
	require.NoError(t, rows.Err())
	assert.Equal(t, []string{"hello", "world"}, messages)
}
