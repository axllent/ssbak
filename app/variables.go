package app

var (
	// ProjectRoot var
	ProjectRoot string

	// DBHost database host
	DBHost string

	// DBUsername database user
	DBUsername string

	// DBPassword database password
	DBPassword string

	// DBName database name
	DBName string

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
