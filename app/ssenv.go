package app

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"regexp"

	"github.com/joho/godotenv"
)

// Env translates a SilverStripe env
func Env(dir string) error {
	if !isDir(dir) {
		return fmt.Errorf("%s is not a directory", dir)
	}

	ProjectRoot = dir

	if isFile(path.Join(dir, ".env")) {
		if err := fromEnv(path.Join(dir, ".env")); err != nil {
			return err
		}
	} else if isFile(path.Join(dir, "_ss_environment.php")) {
		if err := fromSsEnvironment(path.Join(dir, "_ss_environment.php")); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("Cannot find an SilverStripe config in %s", dir)
	}

	if DBName == "" {
		return errors.New("No database defined")
	}

	if DBUsername == "" {
		return errors.New("No database user defined")
	}

	return nil
}

// Extracts variables from an .env file
func fromEnv(file string) error {
	if err := godotenv.Load(file); err != nil {
		return err
	}

	DBHost = os.Getenv("SS_DATABASE_SERVER")
	DBUsername = os.Getenv("SS_DATABASE_USERNAME")
	DBPassword = os.Getenv("SS_DATABASE_PASSWORD")
	DBName = os.Getenv("SS_DATABASE_NAME")

	return nil
}

// Extracts from a _ss_environment.php file
func fromSsEnvironment(file string) error {
	phpb, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}

	php := string(phpb)

	DBHost = matchFromPhp(php, "SS_DATABASE_SERVER")
	DBUsername = matchFromPhp(php, "SS_DATABASE_USERNAME")
	DBPassword = matchFromPhp(php, "SS_DATABASE_PASSWORD")
	DBName = matchFromPhp(php, "SS_DATABASE_NAME")

	return nil
}

// MatchFromPhp uses regular expressiont to detect variables in a string
func matchFromPhp(code, key string) string {
	var re = regexp.MustCompile(`(?mi)define\s?\(\s?['"]` + key + `['"]\s?,\s?['"](.*)['"]\s?\)\s?;`)

	matches := re.FindStringSubmatch(code)
	if len(matches) == 2 {
		return matches[1]
	}
	return ""
}
