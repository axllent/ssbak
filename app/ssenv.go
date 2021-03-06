package app

import (
	"errors"
	"fmt"
	"io/ioutil"
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

// BoostrapEnv sets up the Silverstripe environment
func BoostrapEnv(dir string) error {
	if !isDir(dir) {
		return fmt.Errorf("%s is not a directory", dir)
	}

	d, err := filepath.Abs(dir)
	if err != nil {
		return err
	}

	ProjectRoot = d

	conf, err := findConfig(ProjectRoot)
	if err == nil {
		Log(fmt.Sprintf("Parsing %s", conf.Path))
		if conf.PHP {
			if err := setFromSsEnvironmentFile(conf.Path); err != nil {
				return err
			}
		} else {
			if err := setFromEnvFile(conf.Path); err != nil {
				return err
			}
		}
	} else {
		// show warning, but continue as the DB variables could have been exported
		fmt.Printf("Cannot find an Silverstripe config in %s\n", dir)
	}

	if DB.Name == "" {
		return errors.New("No database defined")
	}

	if DB.Username == "" {
		return errors.New("No database user defined")
	}

	// MySQLPDODatabase, MySQLDatabase, MSSQLDatabase, PostgreSQLDatabase
	if DB.Type == "" || strings.Contains(strings.ToLower(DB.Type), "mysql") {
		DB.Type = "MySQL"
	} else {
		return fmt.Errorf("Database %s not supported", DB.Type)
	}

	return nil
}

// FindConfig will return a configuration file path & type if found
func findConfig(dir string) (configFile, error) {
	r := configFile{}
	if isFile(path.Join(dir, ".env")) {
		r.Path = path.Join(dir, ".env")
		return r, nil
	}
	if isFile(path.Join(filepath.Dir(dir), ".env")) {
		r.Path = path.Join(filepath.Dir(dir), ".env")
		return r, nil
	}
	if isFile(path.Join(dir, "_ss_environment.php")) {
		r.Path = path.Join(dir, "_ss_environment.php")
		r.PHP = true
		return r, nil
	}
	if isFile(path.Join(filepath.Dir(dir), "_ss_environment.php")) {
		r.Path = path.Join(filepath.Dir(dir), "_ss_environment.php")
		r.PHP = true
		return r, nil
	}

	return r, errors.New("Config not found")
}

// Extracts variables from an .env file
func setFromEnvFile(file string) error {
	if err := godotenv.Load(file); err != nil {
		return err
	}

	DB.Host = os.Getenv("SS_DATABASE_SERVER")
	DB.Username = os.Getenv("SS_DATABASE_USERNAME")
	DB.Password = os.Getenv("SS_DATABASE_PASSWORD")
	DB.Name = os.Getenv("SS_DATABASE_PREFIX") +
		os.Getenv("SS_DATABASE_NAME") +
		os.Getenv("SS_DATABASE_SUFFIX")
	DB.Type = os.Getenv("SS_DATABASE_CLASS")
	DB.Port = os.Getenv("SS_DATABASE_PORT")

	if DB.Name == "" && os.Getenv("SS_DATABASE_CHOOSE_NAME") != "" {
		DB.Name = dbChooseName(os.Getenv("SS_DATABASE_CHOOSE_NAME"))
	}

	return nil
}

// Extracts from a _ss_environment.php file
func setFromSsEnvironmentFile(file string) error {
	phpb, err := ioutil.ReadFile(filepath.Clean(file))
	if err != nil {
		return err
	}

	rawPHP := string(phpb)

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

// MatchFromPhp uses regular expressiont to detect variables in a string
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

	return strings.Replace(fmt.Sprintf("SS_%s", path.Base(f)), ".", "", -1)
}
