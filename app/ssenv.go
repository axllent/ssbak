package app

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/joho/godotenv"
)

// BoostrapEnv sets up the SilverStripe environment
func BoostrapEnv(dir string) error {
	if !isDir(dir) {
		return fmt.Errorf("%s is not a directory", dir)
	}

	ProjectRoot = dir

	if isFile(path.Join(dir, ".env")) {
		if err := setFromEnvFile(path.Join(dir, ".env")); err != nil {
			return err
		}
	} else if isFile(path.Join(dir, "_ss_environment.php")) {
		if err := setFromSsEnvironmentFile(path.Join(dir, "_ss_environment.php")); err != nil {
			return err
		}
	} else {
		// show warning, but continue as the DB variables could have been exported
		fmt.Printf("Cannot find an SilverStripe config in %s\n", dir)
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

// Extracts variables from an .env file
func setFromEnvFile(file string) error {
	if err := godotenv.Load(file); err != nil {
		return err
	}

	DB.Host = os.Getenv("SS_DATABASE_SERVER")
	DB.Username = os.Getenv("SS_DATABASE_USERNAME")
	DB.Password = os.Getenv("SS_DATABASE_PASSWORD")
	DB.Name = os.Getenv("SS_DATABASE_NAME")
	DB.Type = os.Getenv("SS_DATABASE_CLASS")

	if os.Getenv("SS_DATABASE_PORT") != "" {
		DB.Port = os.Getenv("SS_DATABASE_PORT")
	}

	return nil
}

// Extracts from a _ss_environment.php file
func setFromSsEnvironmentFile(file string) error {
	phpb, err := ioutil.ReadFile(file)
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
	DB.Name = matchFromPhp(str, "SS_DATABASE_NAME")
	DB.Type = matchFromPhp(str, "SS_DATABASE_CLASS")

	if matchFromPhp(str, "SS_DATABASE_PORT") != "" {
		DB.Port = matchFromPhp(str, "SS_DATABASE_PORT")
	}

	return nil
}

// MatchFromPhp uses regular expressiont to detect variables in a string
func matchFromPhp(code, key string) string {
	// allow exported environment values to override
	if os.Getenv(key) != "" {
		return os.Getenv(key)
	}
	var re = regexp.MustCompile(`(?mi)define\s*?\(\s*?['"]` + key + `['"]\s*?,\s*?(['"](.*)['"]|(\d+))\s*?\)\s*?;`)

	matches := re.FindStringSubmatch(code)
	if len(matches) == 4 {
		if matches[2] == "" && matches[3] != "" {
			return matches[3] // unquoted variable
		}
		return matches[2]
	}

	return ""
}
