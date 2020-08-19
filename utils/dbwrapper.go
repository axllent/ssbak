package utils

// The DB wrappers serve to map the various database functions to their respective types.
// This provides flexibility in the calling function without having to duplicate code and
// wrap in a whole bunch of if/else statements.
var (
	// DBDumpWrapper is a map of database dump to gzip functions based on DB.Type
	DBDumpWrapper = map[string]func(string) error{
		"MySQL": MySQLDumpToGz,
	}

	// DBCreateWrapper is a map is database creation functions based on DB.Type
	DBCreateWrapper = map[string]func(bool) error{
		"MySQL": MySQLCreateDB,
	}

	// DBLoadWrapper is a map of database load-from-gzip functions based on DB.Type
	DBLoadWrapper = map[string]func(string) error{
		"MySQL": MySQLLoadFromGz,
	}
)
