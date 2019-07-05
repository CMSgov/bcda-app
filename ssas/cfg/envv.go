package cfg

import (
	"os"
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
