package utils

import (
	"fmt"
	"os"
	"regexp"
	"strconv"

	"github.com/CMSgov/bcda-app/conf"

	"github.com/sirupsen/logrus"
)

// FromEnv always returns a string that is either a non-empty value from the environment variable named by key or
// the string otherwise
func FromEnv(key, otherwise string) string {
	s := conf.GetEnv(key)
	if s == "" {
		logrus.Infof(`No %s value; using %s instead.`, key, otherwise)
		return otherwise
	}
	return s
}

func GetEnvInt(varName string, defaultVal int) int {
	v := conf.GetEnv(varName)
	if v != "" {
		i, err := strconv.Atoi(v)
		if err == nil {
			return i
		}
	}
	return defaultVal
}

func GetEnvBool(varName string, defaultVal bool) bool {
	v := conf.GetEnv(varName)
	if v != "" {
		b, err := strconv.ParseBool(v)
		if err == nil {
			return b
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

// CloseFileAndLogError closes a file and logs any errors
func CloseFileAndLogError(f *os.File) {
	CloseAndLog(logrus.ErrorLevel, f.Close)
}

func CloseAndLog(level logrus.Level, close func() error) {
	if err := close(); close != nil {
		logrus.StandardLogger().Log(level, err.Error())
	}
}

// Dedup is function that takes a string slice and removes any duplicates.
func Dedup(slice []string) []string {

	/*
	   While fast, hash map can be memory heavy. If the input data is very large (~100K)
	   and lower memory usage is a requirement, make changes to this function.
	   1. Work with reference of string slice instead of copy
	   2. Don't use map... maybe use merge sort deletion
	*/

	// Get the length of the slice
	var n = len(slice)

	// Use the length to allocate memory once for new slice and map
	var newSlice = make([]string, 0, n)
	var dupcheck = make(map[string]bool, n)

	for _, v := range slice {
		// If false, we have not encountered the string before
		// If it is true, do nothing
		if !dupcheck[v] {
			// Not a duplicate
			dupcheck[v] = true
			newSlice = append(newSlice, v)
		}
	}

	return newSlice
}
