package app

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// ConfigFile struct
type configFile struct {
	Path string
	PHP  bool
}

// BootstrapEnv sets up the Silverstripe environment
func BootstrapEnv(dir string) error {
	if !isDir(dir) {
		return fmt.Errorf("%s is not a directory", dir)
	}

	d, err := filepath.Abs(dir)
	if err != nil {
		return err
	}

	ProjectRoot = d

	if !dotEnvIgnored() {
		conf, err := findConfig(ProjectRoot)
		if err == nil {
			Log(fmt.Sprintf("Parsing %s", conf.Path))
			if conf.PHP {
				if err := setFromSsEnvironmentFile(conf.Path); err != nil {
					return err
				}
			} else {
				if err := godotenv.Load(conf.Path); err != nil {
					return err
				}
			}
		}
	}

	// load/overwrite variables from environment if set
	setFromEnv()

	if DB.Name == "" {
		if !dotEnvIgnored() {
			fmt.Println("no .env file detected")
		}
		return errors.New("no database defined")
	}

	if DB.Username == "" {
		return errors.New("no database user defined")
	}

	// MySQLPDODatabase, MySQLDatabase, MSSQLDatabase, PostgreSQLDatabase
	if DB.Type == "" || strings.Contains(strings.ToLower(DB.Type), "mysql") {
		DB.Type = "MySQL"
	} else {
		return fmt.Errorf("database %s not supported", DB.Type)
	}

	return nil
}

// FindConfig will return a configuration file path & type if found
func findConfig(dir string) (configFile, error) {
	r := configFile{}
	if isFile(path.Join(dir, ".env")) {
		r.Path = RealPath(path.Join(dir, ".env"))
		return r, nil
	}
	if isFile(path.Join(filepath.Dir(dir), ".env")) {
		r.Path = RealPath(path.Join(filepath.Dir(dir), ".env"))
		return r, nil
	}
	if isFile(path.Join(dir, "_ss_environment.php")) {
		r.Path = RealPath(path.Join(dir, "_ss_environment.php"))
		r.PHP = true
		return r, nil
	}
	if isFile(path.Join(filepath.Dir(dir), "_ss_environment.php")) {
		r.Path = RealPath(path.Join(filepath.Dir(dir), "_ss_environment.php"))
		r.PHP = true
		return r, nil
	}

	return r, errors.New("config not found")
}

// Extract variables from the system environment if set
func setFromEnv() {
	if v, ok := os.LookupEnv("SS_DATABASE_SERVER"); ok {
		DB.Host = v
	}
	if v, ok := os.LookupEnv("SS_DATABASE_SERVER"); ok {
		DB.Host = v
	}
	if v, ok := os.LookupEnv("SS_DATABASE_USERNAME"); ok {
		DB.Username = v
	}
	if v, ok := os.LookupEnv("SS_DATABASE_PASSWORD"); ok {
		DB.Password = v
	}
	if v, ok := os.LookupEnv("SS_DATABASE_NAME"); ok {
		DB.Name = os.Getenv("SS_DATABASE_PREFIX") + v + os.Getenv("SS_DATABASE_SUFFIX")
	}
	if v, ok := os.LookupEnv("SS_DATABASE_CLASS"); ok {
		DB.Type = v
	}
	if v, ok := os.LookupEnv("SS_DATABASE_PORT"); ok {
		DB.Port = v
	}

	if DB.Name == "" && os.Getenv("SS_DATABASE_CHOOSE_NAME") != "" {
		DB.Name = dbChooseName(os.Getenv("SS_DATABASE_CHOOSE_NAME"))
	}
}

// return whether the boolean/int SS_IGNORE_DOT_ENV is set
func dotEnvIgnored() bool {
	v := strings.ToLower(os.Getenv("SS_IGNORE_DOT_ENV"))

	return v == "1" || v == "true"
}

// Extracts from a _ss_environment.php file
func setFromSsEnvironmentFile(file string) error {
	b, err := os.ReadFile(filepath.Clean(file))
	if err != nil {
		return err
	}

	rawPHP := string(b)

	// strip out php comments
	re := regexp.MustCompile("(?s)#.*?\n|(?s)//.*?\n|/\\*.*?\\*/")
	str := re.ReplaceAllString(rawPHP, "")

	DB.Host = matchFromPhp(str, "SS_DATABASE_SERVER")
	DB.Username = matchFromPhp(str, "SS_DATABASE_USERNAME")
	DB.Password = matchFromPhp(str, "SS_DATABASE_PASSWORD")
	DB.Name = matchFromPhp(str, "SS_DATABASE_PREFIX") +
		matchFromPhp(str, "SS_DATABASE_NAME") +
		matchFromPhp(str, "SS_DATABASE_SUFFIX")
	DB.Type = matchFromPhp(str, "SS_DATABASE_CLASS")
	DB.Port = matchFromPhp(str, "SS_DATABASE_PORT")

	if DB.Name == "" && matchFromPhp(str, "SS_DATABASE_CHOOSE_NAME") != "" {
		DB.Name = dbChooseName(matchFromPhp(str, "SS_DATABASE_CHOOSE_NAME"))
	}

	return nil
}

// MatchFromPhp uses regular expressions to detect variables in a string
func matchFromPhp(code, key string) string {
	// allow exported environment values to override
	if os.Getenv(key) != "" {
		return os.Getenv(key)
	}
	var re = regexp.MustCompile(`(?mi)define\s*?\(\s*?['"]` + key + `['"]\s*?,\s*?(['"](.*)['"]|(\d+|true))\s*?\)\s*?;`)

	matches := re.FindStringSubmatch(code)
	if len(matches) == 4 {
		if matches[2] == "" && matches[3] != "" {
			return matches[3] // unquoted variable
		}

		return matches[2]
	}

	return ""
}

// DBChooseName will translate the SS_DATABASE_CHOOSE_NAME variable
// into string based on the ProjectRoot.
func dbChooseName(v string) string {
	v = strings.ToLower(v)

	if v == "true" {
		v = "1"
	}

	i, err := strconv.Atoi(v)
	if err != nil {
		return ""
	}

	i = i - 1

	f := ProjectRoot

	// move up in the folder structure
	for x := i; x > 0; x-- {
		f = path.Dir(f)
	}

	return strings.ReplaceAll(fmt.Sprintf("SS_%s", path.Base(f)), ".", "")
}
