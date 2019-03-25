package utils

import (
	"os"

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
