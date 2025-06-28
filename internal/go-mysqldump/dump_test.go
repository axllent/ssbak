// nolint:lll
package mysqldump_test

import (
	"bytes"
	"database/sql"
	"reflect"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/axllent/ssbak/internal/go-mysqldump"
	"github.com/stretchr/testify/assert"
)

func getMockData() (data *mysqldump.Data, mock sqlmock.Sqlmock, err error) {
	var db *sql.DB
	db, mock, err = sqlmock.New()

	if err != nil {
		return
	}

	mock.ExpectBegin()

	data = &mysqldump.Data{
		Connection: db,
	}
	err = data.Begin()

	return
}

func TestGetTablesOk(t *testing.T) {
	data, mock, err := getMockData()
	assert.NoError(t, err, "an error was not expected when opening a stub database connection")

	defer data.Close()

	rows := sqlmock.NewRows([]string{"Tables_in_Testdb"}).
		AddRow("Test_Table_1").
		AddRow("Test_Table_2")

	mock.ExpectQuery("^SHOW TABLES$").WillReturnRows(rows)

	result, err := data.GetTables()
	assert.NoError(t, err)

	// we make sure that all conditions were met
	assert.NoError(t, mock.ExpectationsWereMet(), "there were unfulfilled conditions")

	assert.EqualValues(t, []string{"Test_Table_1", "Test_Table_2"}, result)
}

func TestIgnoreTablesOk(t *testing.T) {
	data, mock, err := getMockData()
	assert.NoError(t, err, "an error was not expected when opening a stub database connection")

	defer data.Close()

	rows := sqlmock.NewRows([]string{"Tables_in_Testdb"}).
		AddRow("Test_Table_1").
		AddRow("Test_Table_2")

	mock.ExpectQuery("^SHOW TABLES$").WillReturnRows(rows)

	data.IgnoreTables = []string{"Test_Table_1"}

	result, err := data.GetTables()
	assert.NoError(t, err)

	// we make sure that all conditions were met
	assert.NoError(t, mock.ExpectationsWereMet(), "there were unfulfilled conditions")

	assert.EqualValues(t, []string{"Test_Table_2"}, result)
}

func TestGetTablesNil(t *testing.T) {
	data, mock, err := getMockData()
	assert.NoError(t, err, "an error was not expected when opening a stub database connection")

	defer data.Close()

	rows := sqlmock.NewRows([]string{"Tables_in_Testdb"}).
		AddRow("Test_Table_1").
		AddRow(nil).
		AddRow("Test_Table_3")

	mock.ExpectQuery("^SHOW TABLES$").WillReturnRows(rows)

	result, err := data.GetTables()
	assert.NoError(t, err)

	// we make sure that all conditions were met
	assert.NoError(t, mock.ExpectationsWereMet(), "there were unfulfilled conditions")

	assert.EqualValues(t, []string{"Test_Table_1", "Test_Table_3"}, result)
}

func TestGetServerVersionOk(t *testing.T) {
	data, mock, err := getMockData()
	assert.NoError(t, err, "an error was not expected when opening a stub database connection")

	defer data.Close()

	rows := sqlmock.NewRows([]string{"Version()"}).
		AddRow("test_version")

	mock.ExpectQuery("^SELECT version()").WillReturnRows(rows)

	meta := mysqldump.MetaData{}

	assert.NoError(t, meta.UpdateServerVersion(data), "error was not expected while updating stats")

	// we make sure that all expectations were met
	assert.NoError(t, mock.ExpectationsWereMet(), "there were unfulfilled conditions")

	assert.Equal(t, "test_version", meta.ServerVersion)
}

func TestCreateTableSQLOk(t *testing.T) {
	data, mock, err := getMockData()
	assert.NoError(t, err, "an error was not expected when opening a stub database connection")

	defer data.Close()

	// Add expectation for table type check
	mock.ExpectQuery(`SELECT table_type FROM information_schema.tables WHERE table_name = \?`).WithArgs("Test_Table").WillReturnRows(sqlmock.NewRows([]string{"table_type"}).AddRow("BASE TABLE"))

	rows := sqlmock.NewRows([]string{"Table", "Create Table"}).
		AddRow("Test_Table", "CREATE TABLE 'Test_Table' (`id` int(11) NOT NULL AUTO_INCREMENT,`s` char(60) DEFAULT NULL, PRIMARY KEY (`id`))ENGINE=InnoDB DEFAULT CHARSET=latin1")

	mock.ExpectQuery("^SHOW CREATE TABLE `Test_Table`$").WillReturnRows(rows)

	table, err := data.CreateTable("Test_Table")
	assert.NoError(t, err)

	result, err := table.CreateSQL()
	assert.NoError(t, err)

	// we make sure that all expectations were met
	assert.NoError(t, mock.ExpectationsWereMet(), "there were unfulfilled conditions")

	expectedResult := "CREATE TABLE 'Test_Table' (`id` int(11) NOT NULL AUTO_INCREMENT,`s` char(60) DEFAULT NULL, PRIMARY KEY (`id`))ENGINE=InnoDB DEFAULT CHARSET=latin1"

	if !reflect.DeepEqual(result, expectedResult) {
		t.Fatalf("expected %#v, got %#v", expectedResult, result)
	}
}

func TestCreateTableRowValues(t *testing.T) {
	data, mock, err := getMockData()
	assert.NoError(t, err, "an error was not expected when opening a stub database connection")

	defer data.Close()

	rows := sqlmock.NewRows([]string{"id", "email", "name"}).
		AddRow(1, "test@test.de", "Test Name 1").
		AddRow(2, "test2@test.de", "Test Name 2")

	mock.ExpectQuery("SELECT table_type FROM information_schema.tables WHERE table_name = ?").WithArgs("test").WillReturnRows(sqlmock.NewRows([]string{"table_type"}).AddRow("BASE TABLE"))
	mock.ExpectQuery("^SELECT (.+) FROM `test`$").WillReturnRows(rows)

	table, err := data.CreateTable("test")
	assert.NoError(t, err)

	assert.True(t, table.Next())

	result := table.RowValues()
	assert.NoError(t, table.Err)

	// we make sure that all expectations were met
	assert.NoError(t, mock.ExpectationsWereMet(), "there were unfulfilled conditions")

	assert.EqualValues(t, "('1','test@test.de','Test Name 1')", result)
}

func TestCreateTableValuesSteam(t *testing.T) {
	data, mock, err := getMockData()
	assert.NoError(t, err, "an error was not expected when opening a stub database connection")

	defer data.Close()

	rows := sqlmock.NewRows([]string{"id", "email", "name"}).
		AddRow(1, "test@test.de", "Test Name 1").
		AddRow(2, "test2@test.de", "Test Name 2")

	mock.ExpectQuery("SELECT table_type FROM information_schema.tables WHERE table_name = ?").WithArgs("test").WillReturnRows(sqlmock.NewRows([]string{"table_type"}).AddRow("BASE TABLE"))
	mock.ExpectQuery("^SELECT (.+) FROM `test`$").WillReturnRows(rows)

	data.MaxAllowedPacket = 4096

	table, err := data.CreateTable("test")
	assert.NoError(t, err)

	s := table.Stream()
	assert.EqualValues(t, "INSERT INTO `test` VALUES ('1','test@test.de','Test Name 1'),('2','test2@test.de','Test Name 2');", <-s)

	// we make sure that all expectations were met
	assert.NoError(t, mock.ExpectationsWereMet(), "there were unfulfilled conditions")
}

func TestCreateTableValuesSteamSmallPackets(t *testing.T) {
	data, mock, err := getMockData()
	assert.NoError(t, err, "an error was not expected when opening a stub database connection")

	defer data.Close()

	rows := sqlmock.NewRows([]string{"id", "email", "name"}).
		AddRow(1, "test@test.de", "Test Name 1").
		AddRow(2, "test2@test.de", "Test Name 2")

	mock.ExpectQuery("SELECT table_type FROM information_schema.tables WHERE table_name = ?").WithArgs("test").WillReturnRows(sqlmock.NewRows([]string{"table_type"}).AddRow("BASE TABLE"))
	mock.ExpectQuery("^SELECT (.+) FROM `test`$").WillReturnRows(rows)

	data.MaxAllowedPacket = 64

	table, err := data.CreateTable("test")
	assert.NoError(t, err)

	s := table.Stream()
	assert.EqualValues(t, "INSERT INTO `test` VALUES ('1','test@test.de','Test Name 1');", <-s)
	assert.EqualValues(t, "INSERT INTO `test` VALUES ('2','test2@test.de','Test Name 2');", <-s)

	// we make sure that all expectations were met
	assert.NoError(t, mock.ExpectationsWereMet(), "there were unfulfilled conditions")
}

func TestCreateTableAllValuesWithNil(t *testing.T) {
	data, mock, err := getMockData()
	assert.NoError(t, err, "an error was not expected when opening a stub database connection")

	defer data.Close()

	rows := sqlmock.NewRows([]string{"id", "email", "name"}).
		AddRow(1, nil, "Test Name 1").
		AddRow(2, "test2@test.de", "Test Name 2").
		AddRow(3, "", "Test Name 3")

	mock.ExpectQuery("SELECT table_type FROM information_schema.tables WHERE table_name = ?").WithArgs("test").WillReturnRows(sqlmock.NewRows([]string{"table_type"}).AddRow("BASE TABLE"))
	mock.ExpectQuery("^SELECT (.+) FROM `test`$").WillReturnRows(rows)

	table, err := data.CreateTable("test")
	assert.NoError(t, err)

	results := make([]string, 0)

	for table.Next() {
		row := table.RowValues()
		assert.NoError(t, table.Err)

		results = append(results, row)
	}

	// we make sure that all expectations were met
	assert.NoError(t, mock.ExpectationsWereMet(), "there were unfulfilled conditions")

	expectedResults := []string{"('1',NULL,'Test Name 1')", "('2','test2@test.de','Test Name 2')", "('3','','Test Name 3')"}

	assert.EqualValues(t, expectedResults, results)
}

// nolint:dupl
func TestCreateTableOk(t *testing.T) {
	data, mock, err := getMockData()
	assert.NoError(t, err, "an error was not expected when opening a stub database connection")

	defer data.Close()

	// Add expectation for table type check
	mock.ExpectQuery(`SELECT table_type FROM information_schema.tables WHERE table_name = \?`).WithArgs("Test_Table").WillReturnRows(sqlmock.NewRows([]string{"table_type"}).AddRow("BASE TABLE"))

	createTableRows := sqlmock.NewRows([]string{"Table", "Create Table"}).
		AddRow("Test_Table", "CREATE TABLE 'Test_Table' (`id` int(11) NOT NULL AUTO_INCREMENT,`s` char(60) DEFAULT NULL, PRIMARY KEY (`id`))ENGINE=InnoDB DEFAULT CHARSET=latin1")

	createTableValueRows := sqlmock.NewRows([]string{"id", "email", "name"}).
		AddRow(1, nil, "Test Name 1").
		AddRow(2, "test2@test.de", "Test Name 2")

	mock.ExpectQuery("^SHOW CREATE TABLE `Test_Table`$").WillReturnRows(createTableRows)
	mock.ExpectQuery("^SELECT (.+) FROM `Test_Table`$").WillReturnRows(createTableValueRows)

	var buf bytes.Buffer

	data.Out = &buf
	data.MaxAllowedPacket = 4096

	assert.NoError(t, data.GetTemplates())

	table, err := data.CreateTable("Test_Table")
	assert.NoError(t, err)

	data.WriteTable(table) // nolint:errcheck

	// we make sure that all expectations were met
	assert.NoError(t, mock.ExpectationsWereMet(), "there were unfulfilled conditions")

	expectedResult := `
--
-- Table structure for table ~Test_Table~
--


DROP TABLE IF EXISTS ~Test_Table~;

/*!40101 SET @saved_cs_client     = @@character_set_client */;
 SET character_set_client = utf8mb4 ;
CREATE TABLE 'Test_Table' (~id~ int(11) NOT NULL AUTO_INCREMENT,~s~ char(60) DEFAULT NULL, PRIMARY KEY (~id~))ENGINE=InnoDB DEFAULT CHARSET=latin1;
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
	result := strings.ReplaceAll(buf.String(), "`", "~")
	assert.Equal(t, expectedResult, result)
}

// nolint:dupl
func TestCreateTableOkSmallPackets(t *testing.T) {
	data, mock, err := getMockData()
	assert.NoError(t, err, "an error was not expected when opening a stub database connection")

	defer data.Close()

	// Add expectation for table type check
	mock.ExpectQuery(`SELECT table_type FROM information_schema.tables WHERE table_name = \?`).WithArgs("Test_Table").WillReturnRows(sqlmock.NewRows([]string{"table_type"}).AddRow("BASE TABLE"))

	createTableRows := sqlmock.NewRows([]string{"Table", "Create Table"}).
		AddRow("Test_Table", "CREATE TABLE 'Test_Table' (`id` int(11) NOT NULL AUTO_INCREMENT,`s` char(60) DEFAULT NULL, PRIMARY KEY (`id`))ENGINE=InnoDB DEFAULT CHARSET=latin1")

	createTableValueRows := sqlmock.NewRows([]string{"id", "email", "name"}).
		AddRow(1, nil, "Test Name 1").
		AddRow(2, "test2@test.de", "Test Name 2")

	mock.ExpectQuery("^SHOW CREATE TABLE `Test_Table`$").WillReturnRows(createTableRows)
	mock.ExpectQuery("^SELECT (.+) FROM `Test_Table`$").WillReturnRows(createTableValueRows)

	var buf bytes.Buffer

	data.Out = &buf
	data.MaxAllowedPacket = 64

	assert.NoError(t, data.GetTemplates())

	table, err := data.CreateTable("Test_Table")
	assert.NoError(t, err)

	data.WriteTable(table) // nolint:errcheck

	// we make sure that all expectations were met
	assert.NoError(t, mock.ExpectationsWereMet(), "there were unfulfilled conditions")

	expectedResult := `
--
-- Table structure for table ~Test_Table~
--


DROP TABLE IF EXISTS ~Test_Table~;

/*!40101 SET @saved_cs_client     = @@character_set_client */;
 SET character_set_client = utf8mb4 ;
CREATE TABLE 'Test_Table' (~id~ int(11) NOT NULL AUTO_INCREMENT,~s~ char(60) DEFAULT NULL, PRIMARY KEY (~id~))ENGINE=InnoDB DEFAULT CHARSET=latin1;
/*!40101 SET character_set_client = @saved_cs_client */;


--
-- Dumping data for table ~Test_Table~
--

LOCK TABLES ~Test_Table~ WRITE;
/*!40000 ALTER TABLE ~Test_Table~ DISABLE KEYS */;
INSERT INTO ~Test_Table~ VALUES ('1',NULL,'Test Name 1');
INSERT INTO ~Test_Table~ VALUES ('2','test2@test.de','Test Name 2');
/*!40000 ALTER TABLE ~Test_Table~ ENABLE KEYS */;
UNLOCK TABLES;

`
	result := strings.ReplaceAll(buf.String(), "`", "~")
	assert.Equal(t, expectedResult, result)
}
