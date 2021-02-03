package conf

/*
   This is a package that wraps the viper, a package designed to handle config
   files, for the BCDA app. This package will go through three different stages.

   1. Local env looks primarily at conf package for variables, but will also look
   in the environment for any variables it is not tracking. PROD/TEST/DEV will
   only look in the environment. (WE ARE HERE NOW)

   2. Local env will only look at conf package. PROD/TEST/DEV will look at both.

   3. All env will look at the conf package.

   Assumptions:
   1. The configuration file is a env file (can be changed if needed).
   2. The configuration variables, once it has been ingested by the conf package,
   will stay immutable during the uptime of the application (exception is test).
*/

import (
	"os"
	"testing"

	"github.com/spf13/viper"
)

// Private global variable:

// An instance of the viper struct containing the config information. Only made
// accessible through public functions: GetEnv, SetEnv, etc.
var envVars viper.Viper

// Implementing a state machine that tracks the status of the config ingestion.
// This state machine should go away when in stage 3.
type configStatus uint8

const (
	configGood    configStatus = 0
	configBad     configStatus = 1
	noConfigFound configStatus = 2
)

// if config file found and loaded, doesn't changed
var state configStatus = configGood

/*
   This is the private helper function that sets up viper. This function is
   called by the init() function only once during initialization of the package.
*/
func setup(dir string) *viper.Viper {

	// Viper setup
	var v = viper.New()
	v.SetConfigName("local")
	v.SetConfigType("env")
	v.AddConfigPath(dir)
	// Viper is lazy, do the read and parse of the config file
	var err = v.ReadInConfig()

	// If viper cannot read the configuration file...
	if err != nil {
		state = configBad
	}

	return v

}

/*
   init:
   First thing to run when this package is loaded by the binary.
   Even if multiple packages import conf, this will be called and ran ONLY once.
*/
func init() {

	// Possible config file locations: local and PROD/DEV/TEST respectfully.
	var locationSlice = []string{
		"/go/src/github.com/CMSgov/bcda-app/shared_files/decrypted",
		// Placeholder for configuration location for TEST/DEV/PROD once available.
	}

	if success, loc := findEnv(locationSlice[:]); success {
		// A config file found, set up viper using that location
		envVars = *setup(loc)
	} else {
		// Checked both locations, no config file found
		state = noConfigFound
	}

}

/*
   findEnv is a helper function that will determine what environment the application
   is running in: local or PROD/TEST/DEV. Each environment should have a distinct
   path where the configuration file is located. First it check the local path,
   then the PROD/DEV/TEST. If both not found, defaults to just using environment vars.
*/
func findEnv(location []string) (bool, string) {

	for _, el := range location {
		// Check if the configuration file exists
		if _, err := os.Stat(el + "/local.env"); err == nil {
			return true, el
		}
	}

	return false, ""

}

// GetEnv() is a public function that retrieves values stored in conf. If it does not
// exist, an empty string (i.e., "") is returned.
func GetEnv(key string) string {

	// If the config file is loaded and ingested correctly, use the config file.
	if state == configGood {

		if value := envVars.GetString(key); value != "" {
			return value
		} else {
			// if it is blank, check environment variables. Just in case there are
			// variables that started off empty and is now available. Once made available,
			// that variable is immutable for the rest of the runtime.
			v, exist := os.LookupEnv(key)
			if exist {
				var _ = SetEnv(&testing.T{}, key, v)
			}
			return v
		}

	}

	// Config file not good, so default to environment variables.
	return os.Getenv(key)

}

// LookupEnv is a public function, like GetEnv, designed to replace os.LookupEnv() in code-base.
func LookupEnv(key string) (string, bool) {

	if state == configGood {
		if value := envVars.GetString(key); value != "" {
			return value, true
		} else {
			v, exist := os.LookupEnv(key)
			if exist {
				var _ = SetEnv(&testing.T{}, key, v)
			}
			return v, exist
		}
	}

	return os.LookupEnv(key)

}

// SetEnv is a public function that adds key values into conf. This function should only be used
// either in this package itself or testing. Protect parameter is type *testing.T, and it's there
// to ensure developers knowingly use it in the appropriate circumstances.
func SetEnv(protect *testing.T, key string, value string) error {

	var err error

	// If config is good, change the config in memory
	if state == configGood {
		envVars.Set(key, value) // This doesn't return anything...
	} else {
		// Config is bad, change the EV
		err = os.Setenv(key, value)
	}

	return err

}

// UnsetEnv is a public function that "unsets" a variable. Like SetEnv, this should only be used
// either in this package itself or testing.
func UnsetEnv(protect *testing.T, key string) error {
	var err error

	// If config is good, change the conf in memory
	if state == configGood {
		envVars.Set(key, "")
	}

	// Unset environment variable too, because GetEnv would copy it back over when it should not.
	err = os.Unsetenv(key)

	return err

}
