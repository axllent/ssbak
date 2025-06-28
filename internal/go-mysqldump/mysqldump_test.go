// nolint:lll
package mysqldump_test

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/axllent/ssbak/internal/go-mysqldump"
	"github.com/stretchr/testify/assert"
)

const expectedTable = `--
-- Table structure for table ~Test_Table~
--


DROP TABLE IF EXISTS ~Test_Table~;

/*!40101 SET @saved_cs_client     = @@character_set_client */;
 SET character_set_client = utf8mb4 ;
CREATE TABLE 'Test_Table' (~id~ int(11) NOT NULL AUTO_INCREMENT,~email~ char(60) DEFAULT NULL, ~name~ char(60), PRIMARY KEY (~id~))ENGINE=InnoDB DEFAULT CHARSET=latin1;
/*!40101 SET character_set_client = @saved_cs_client */;


--
-- Dumping data for table ~Test_Table~
--

LOCK TABLES ~Test_Table~ WRITE;
/*!40000 ALTER TABLE ~Test_Table~ DISABLE KEYS */;
INSERT INTO ~Test_Table~ VALUES ('1',NULL,'Test Name 1'),('2','test2@test.de','Test Name 2');
/*!40000 ALTER TABLE ~Test_Table~ ENABLE KEYS */;
UNLOCK TABLES;

`

const expectedView = `
--
-- Table structure for table ~Test_View~
--


DROP VIEW IF EXISTS ~Test_View~;

/*!40101 SET @saved_cs_client     = @@character_set_client */;
 SET character_set_client = utf8mb4 ;
CREATE VIEW 'Test_View' AS SELECT ~id~, ~email~, ~name~ FROM 'Test_Table';
/*!40101 SET character_set_client = @saved_cs_client */;

`

const expectedHeader = `-- Go SQL Dump ` + mysqldump.Version + `
--
-- ------------------------------------------------------
-- Server version	test_version

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

const expectedFooter = `
/*!40103 SET TIME_ZONE=@OLD_TIME_ZONE */;

/*!40101 SET SQL_MODE=@OLD_SQL_MODE */;
/*!40014 SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS */;
/*!40014 SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
/*!40111 SET SQL_NOTES=@OLD_SQL_NOTES */;

`

func RunDump(t testing.TB, data *mysqldump.Data) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err, "an error was not expected when opening a stub database connection")

	defer func() { _ = db.Close() }()

	data.Connection = db
	showTablesRows := sqlmock.NewRows([]string{"Tables_in_Testdb"}).
		AddRow("Test_Table").
		AddRow("Test_View")

	serverVersionRows := sqlmock.NewRows([]string{"Version()"}).
		AddRow("test_version")

	createTableRows := sqlmock.NewRows([]string{"Table", "Create Table"}).
		AddRow("Test_Table", "CREATE TABLE 'Test_Table' (`id` int(11) NOT NULL AUTO_INCREMENT,`email` char(60) DEFAULT NULL, `name` char(60), PRIMARY KEY (`id`))ENGINE=InnoDB DEFAULT CHARSET=latin1")

	createTableValueRows := sqlmock.NewRows([]string{"id", "email", "name"}).
		AddRow(1, nil, "Test Name 1").
		AddRow(2, "test2@test.de", "Test Name 2")

	createViewRows := sqlmock.NewRows([]string{"View", "Create View"}).
		AddRow("Test_View", "CREATE VIEW 'Test_View' AS SELECT `id`, `email`, `name` FROM 'Test_Table'")

	mock.ExpectBegin()

	mock.ExpectQuery(`^SELECT version\(\)$`).WillReturnRows(serverVersionRows)
	mock.ExpectQuery(`^SHOW TABLES$`).WillReturnRows(showTablesRows)
	mock.ExpectExec("^LOCK TABLES `Test_Table` READ /\\*!32311 LOCAL \\*/,`Test_View` READ /\\*!32311 LOCAL \\*/$").WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectQuery(`SELECT table_type FROM information_schema.tables WHERE table_name = \?`).WithArgs("Test_Table").WillReturnRows(sqlmock.NewRows([]string{"table_type"}).AddRow("BASE TABLE"))
	mock.ExpectQuery("^SHOW CREATE TABLE `Test_Table`$").WillReturnRows(createTableRows)
	mock.ExpectQuery("^SELECT (.+) FROM `Test_Table`$").WillReturnRows(createTableValueRows)

	mock.ExpectQuery(`SELECT table_type FROM information_schema.tables WHERE table_name = \?`).WithArgs("Test_View").WillReturnRows(sqlmock.NewRows([]string{"table_type"}).AddRow("VIEW"))
	mock.ExpectQuery("^SHOW CREATE VIEW `Test_View`$").WillReturnRows(createViewRows)
	mock.ExpectQuery("^SELECT (.+) FROM `Test_View`$").WillReturnRows(createTableValueRows)

	mock.ExpectRollback()

	assert.NoError(t, data.Dump(), "an error was not expected when dumping a stub database connection")
}

func TestDumpOk(t *testing.T) {
	var buf bytes.Buffer

	RunDump(t, &mysqldump.Data{
		Out:        &buf,
		LockTables: true,
	})

	result := strings.ReplaceAll(strings.Split(buf.String(), "-- Dump completed")[0], "`", "~")

	expectedResult := expectedHeader + expectedTable + expectedView + expectedFooter
	assert.Equal(t, expectedResult, result)
}

func TestNoLockOk(t *testing.T) {
	var buf bytes.Buffer

	data := &mysqldump.Data{
		Out:        &buf,
		LockTables: false,
	}

	db, mock, err := sqlmock.New()
	assert.NoError(t, err, "an error was not expected when opening a stub database connection")

	defer func() { _ = db.Close() }()

	data.Connection = db
	showTablesRows := sqlmock.NewRows([]string{"Tables_in_Testdb"}).
		AddRow("Test_Table").
		AddRow("Test_View")

	serverVersionRows := sqlmock.NewRows([]string{"Version()"}).
		AddRow("test_version")

	createTableRows := sqlmock.NewRows([]string{"Table", "Create Table"}).
		AddRow("Test_Table", "CREATE TABLE 'Test_Table' (`id` int(11) NOT NULL AUTO_INCREMENT,`email` char(60) DEFAULT NULL, `name` char(60), PRIMARY KEY (`id`))ENGINE=InnoDB DEFAULT CHARSET=latin1")

	createTableValueRows := sqlmock.NewRows([]string{"id", "email", "name"}).
		AddRow(1, nil, "Test Name 1").
		AddRow(2, "test2@test.de", "Test Name 2")

	createViewRows := sqlmock.NewRows([]string{"View", "Create View"}).
		AddRow("Test_View", "CREATE VIEW 'Test_View' AS SELECT `id`, `email`, `name` FROM 'Test_Table'")

	// Add expectation for table type check
	tableTypeRows := sqlmock.NewRows([]string{"table_type"}).
		AddRow("BASE TABLE").
		AddRow("VIEW")

	mock.ExpectBegin()
	mock.ExpectQuery(`^SELECT version\(\)$`).WillReturnRows(serverVersionRows)
	mock.ExpectQuery(`^SHOW TABLES$`).WillReturnRows(showTablesRows)

	mock.ExpectQuery(`^SELECT table_type FROM information_schema.tables WHERE table_name = \?$`).WithArgs("Test_Table").WillReturnRows(tableTypeRows)
	mock.ExpectQuery("^SHOW CREATE TABLE `Test_Table`$").WillReturnRows(createTableRows)
	mock.ExpectQuery("^SELECT (.+) FROM `Test_Table`$").WillReturnRows(createTableValueRows)

	mock.ExpectQuery(`^SELECT table_type FROM information_schema.tables WHERE table_name = \?$`).WithArgs("Test_View").WillReturnRows(tableTypeRows)
	mock.ExpectQuery("^SHOW CREATE VIEW `Test_View`$").WillReturnRows(createViewRows)
	mock.ExpectRollback()

	assert.NoError(t, data.Dump(), "an error was not expected when dumping a stub database connection")

	result := strings.ReplaceAll(strings.Split(buf.String(), "-- Dump completed")[0], "`", "~")

	expectedResult := expectedHeader + expectedTable + expectedView + expectedFooter
	assert.Equal(t, expectedResult, result)
}

func BenchmarkDump(b *testing.B) {
	data := &mysqldump.Data{
		Out:        io.Discard,
		LockTables: true,
	}

	for i := 0; i < b.N; i++ {
		RunDump(b, data)
	}
}
