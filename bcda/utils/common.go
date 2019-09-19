package utils

import (
	"fmt"
	"os"
	"regexp"
	"strconv"

	"github.com/sirupsen/logrus"
)

// FromEnv always returns a string that is either a non-empty value from the environment variable named by key or
// the string otherwise
func FromEnv(key, otherwise string) string {
	s := os.Getenv(key)
	if s == "" {
		logrus.Infof(`No %s value; using %s instead.`, key, otherwise)
		return otherwise
	}
	return s
}

func GetEnvInt(varName string, defaultVal int) int {
	v := os.Getenv(varName)
	if v != "" {
		i, err := strconv.Atoi(v)
		if err == nil {
			return i
		}
	}
	return defaultVal
}

// Look for a directory by increasingly looking up the directory tree by appending '.../'
// It will look a max of 5 levels up before accepting failure and returning an empty string and an error
func GetDirPath(dir string) (string, error) {

	for i := 0; i <= 5; i++ {
		if _, err := os.Stat(dir); err == nil {
			return dir, nil
		} else {
			// look one more level up
			dir = "../" + dir
		}
	}
	return "", fmt.Errorf("unable to locate %s in file path", dir)
}

// ContainsString returns true if `os` is in the array `sa` and false if it is not.
func ContainsString(sa []string, os string) bool {
	for _, s := range sa {
		if s == os {
			return true
		}
	}
	return false
}

// IsUUID returns true if `s` is a UUID.
func IsUUID(s string) bool {
	re := regexp.MustCompile("^[a-fA-F0-9]{8}(?:-[a-fA-F0-9]{4}){3}-[a-fA-F0-9]{12}$")
	return re.MatchString(s)
}
