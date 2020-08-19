package app

var (
	// DB config
	DB = DBStruct{}

	// ProjectRoot var
	ProjectRoot string

	// Verbose logging
	Verbose bool

	// TempFiles get cleaned up on exit
	TempFiles []string

	// TempDir runtime variable can overridden with flags
	TempDir string

	// OnlyAssets runtime variable set with flags
	OnlyAssets bool

	// OnlyDB runtime variable set with flags
	OnlyDB bool

	// // DBType database type
	// DBType string
	// // IdentityFile SSH key
	// IdentityFile string
)

// DBStruct struct
type DBStruct struct {
	// Host database host
	Host string

	// Username database user
	Username string

	// Password database password
	Password string

	// Name database name
	Name string

	// Port database port (as string)
	Port string

	// Database type (mysql, postgres etc)
	Type string
}
