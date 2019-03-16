package utils

import (
	"os"

	"github.com/sirupsen/logrus"
)

func FromEnv(key, otherwise string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	logrus.Infof(`No %s value; using %s instead.`, key, otherwise)
	return otherwise
}
