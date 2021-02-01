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
   1. The configuration file is a env file
   2. The configuration file, once it is made available to the application,
   will stay immutable during the uptime of the application (exception is test)
*/

import (
	"os"
	"testing"

	"github.com/spf13/viper"
)

// Private global variable:

// An instance of the viper struct containing the conf information. Only made
// accessible through public functions GetEnv, SetEnv, etc.
var envVars viper.Viper

// Implementing a state machine tracking how things are going in this package
// This state machine should go away when in stage 3.
const (
	configgood    uint8 = 0
	configbad     uint8 = 1
	noconfigfound uint8 = 2
)

var state uint8 = configgood // if config fie found and loaded, doesn't changed

/*
   This is the private helper function that sets up viper. This function is
   called by the init() function once during initialization of the package.
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
		state = configbad
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
	var locationSlice = [2]string{
		"/go/src/github.com/CMSgov/bcda-app/shared_files/decrypted",
		"/go/src/github.com/CMSgov/DoesNotExistYet", // This is a placeholder for now
	}

	if success, loc := findEnv(locationSlice[:]); success {
		// A config file found, set up viper using that location
		envVars = *setup(loc)
	} else {
		// Checked both locations, no config file found
		state = noconfigfound
	}
}

/*
   findEnv is a helper function that will determine what environment the application
   is running in: local or PROD/TEST/DEV. Each environment should have a distinct
   path where the configuration file is located. First it check the local path,
   then the PROD/DEV/TEST. If both not found, defaults to just using env vars.
*/
func findEnv(location []string) (bool, string) {

	// Check if the configuration file exists
	if _, err := os.Stat(location[0] + "/local.env"); err == nil {
		return true, location[0]
	}

	// Base case: checked both locations and no configurations found
	if len(location) == 1 {
		return false, ""
	}

	// Check the next index of slice location
	return findEnv(location[1:])
}

// GetEnv() is a public function that retrieves value stored in conf. If it does not exist
// "" empty string is returned.
func GetEnv(key string) string {

	// If the configuration file is good, use the config file
	if state == configgood {

		var value = envVars.GetString(key)
        var b bool

		// Even if the config file is load, if the key doesn't exist in conf,
		// try the environment. This technically makes the application mutable
		// and same as before. See doc-string at top of file to see why.
		if value == "" {
			// Copy it over to conf to prevent additional OS calls.
			// Remember to delete both from conf and environment var when UnsetEnv() called!
			value, b = os.LookupEnv(key)

            // Ensure the variables does exist before copy
            if b {
                test := &testing.T{}
                var _ = SetEnv(test, key, value)
            }

		}

		return value
	}

	// Config file not good, so default to environment... boo >:(
	return os.Getenv(key)

}

// LookupEnv is a public function that acts augments os.LookupEnv to look in viper struct first
func LookupEnv(key string) (string, bool) {


    if state == configgood {
        // If the key value exists in conf...
        if value := envVars.Get(key); value != nil && value != "" {
            return value.(string), true
        } else {
            // If it does not exist in conf, check os
            if v, exist := os.LookupEnv(key); exist {
                // bring value over to conf
                test := &testing.T{}
                var _ = SetEnv(test, key, v)
                return v, exist
            }
        }

        return "", false
    }
    
    return os.LookupEnv(key)

}

// SetEnv is a public function that adds key values into conf. This function should only be used
// either in this package itself or testing. Protect parameter is type *testing.T, and is there
// to ensure developers knowingly use it in the appropriate scope.
func SetEnv(protect *testing.T, key string, value string) error {

	var err error

	// If config is good, change the config in memory
	if state == configgood {
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
	if state == configgood {
		envVars.Set(key, "")
	} 

    // Why unset the environment variable too? See line 152.
    err = os.Unsetenv(key)

	return err

}
