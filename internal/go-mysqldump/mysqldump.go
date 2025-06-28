// Package mysqldump provides interfaces to control database dumper.
package mysqldump

import (
	"database/sql"
	"errors"
	"io"
	"os"
	"path"
	"time"
)

var (
	errInvalidDir    = errors.New("invalid directory")
	errExistsAlready = errors.New("dump already exists")
)

// Register a new dumper.
// db: Database that will be dumped (https://golang.org/pkg/database/sql/#DB).
// dir: Path to the directory where the dumps will be stored.
// format: Format to be used to name each dump file.
//
//	Uses time.Time.Format (https://golang.org/pkg/time/#Time.Format). format appended with '.sql'.
//
// database: Database name to drop and create within a dump file. Recreate section omitted, when empty.
func Register(db *sql.DB, dir, format, database string) (*Data, error) {
	if !isDir(dir) {
		return nil, errInvalidDir
	}

	name := time.Now().Format(format)
	p := path.Join(dir, name+".sql")

	// Check dump directory
	if e, _ := exists(p); e {
		return nil, errExistsAlready
	}

	// Create .sql file
	f, err := os.Create(p)
	if err != nil {
		return nil, err
	}

	return &Data{
		Out:        f,
		Connection: db,
		DBName:     database,
	}, nil
}

// Dump Creates a MYSQL dump from the connection to the stream.
func Dump(db *sql.DB, out io.Writer) error {
	return (&Data{
		Connection: db,
		Out:        out,
	}).Dump()
}

// Close the dumper.
// Will also close the database the dumper is connected to as well as the out stream if it has a Close method.
func (data *Data) Close() error {
	defer func() {
		data.Connection = nil
		data.Out = nil
	}()

	if out, ok := data.Out.(io.Closer); ok {
		_ = out.Close()
	}

	return data.Connection.Close()
}

func exists(p string) (bool, os.FileInfo) {
	f, err := os.Open(p)
	if err != nil {
		return false, nil
	}

	defer func() { _ = f.Close() }()

	fi, err := f.Stat()
	if err != nil {
		return false, nil
	}

	return true, fi
}

func isDir(p string) bool {
	if e, fi := exists(p); e {
		return fi.Mode().IsDir()
	}

	return false
}
