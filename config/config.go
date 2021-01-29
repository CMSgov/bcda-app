package config

/*
   This is a package that wraps the viper, a package designed to handle config
   files, for the BCDA app.

   Assumptions:
   1. The configuration file is a env file
   2. The configuration file, once it is made available to the application,
   will stay immutable during the uptime of the application (exception is test)
*/

import (
	"os"
	"testing"
    "time"

	"github.com/spf13/viper"
)

// Global variable made available to any other package importing this package
var envVars viper.Viper
const testmode bool = false

// Implementing a state machine tracking how things are going in this package
const (
	configgood    uint8 = 0
	configbad     uint8 = 1
	noconfigfound uint8 = 2
)
// State is a public variable for diagnostic purposes.
var State uint8 = configgood // if config found and loaded, value not changed

/*
   This is the private helper function that sets up viper. This function is
   called by the "Start" function below once, and it should be inlined by the
   compiler, so no expense of an extra function call:
   https://medium.com/@felipedutratine/does-golang-inline-functions-b41ee2d743fa
*/
func setup(dir string) *viper.Viper {

	/*  TIP:
	    The viper package allocates the Viper struct as "var v" during the import
	    of the package. The line below is actually a function that calls a method
	    using "var v".

	    viper.SetConfigFile("yaml")

	    However, we want the Viper struct to be unreachable after the
	    values from the yaml file is retrieved, so the GC can clean it up after.
	*/

	// Print statement for diagnostic purposes only... to see the viper struct
	// instantiated once. Comment out for production.
	//println("Creating the viper struct!")

	// Viper setup
	var v = viper.New()
	v.SetConfigName("local")
	v.SetConfigType("env")
	v.AddConfigPath(dir)
	// Viper is lazy, do the read and parse of the env now
	var err = v.ReadInConfig()

	// If viper cannot read the configuration file...
	if err != nil {
		State = configbad
	}

	return v

}

/*
   init:
   When the packages is imported, try a couple of locations to find the local.env
   configuration file. If the configuration file is not found, which is the case
   with PROD, it will default the previous behavior of calling os.Getenv.
*/
func init() {

	// Possible config locations. If there are more places to look, add here:
	var locationSlice = []string{
        // TEST DEV Location
        "/go/src/github.com/CMSgov/bcda-app/shared_files/decrypted",
        // This will be the location on PROD, which is currently not set
        // So for now, it will wait 30 seconds and just default to os calls
		".",
	}

    // TEST MODE
    if testmode {
        State = configbad
        return
    }

	// Iterate through the possible locations
	for i, v := range locationSlice {

		// If the file exists
		if success := fileexistattempt(3, v, i); success {
			envVars = *setup(v)
			break
		}

		// Exhausted the for loop, config file not found :/
		if i == len(locationSlice) - 1 {
			State = noconfigfound
		}
	}

}

/*
   This is a helper function to help address the issue with the aco-api application
   starting before the EVs are available, panicing and restarting over and over.
   It's not perfect, and this function should be improved in the future.
*/
func fileexistattempt(maxattempt uint8, v string, i int) bool {

	// Check if the configuration file exists
	if _, err := os.Stat(v + "/local.env"); err == nil {
		return true
	}

	// If test / dev, just make one attempt and report false if the top block fails
	if i == 0 {
		return false
	}

	// Base case for prod configuration... if already done three times, stop
	if maxattempt == 1 {
		return false
	}

	// Wait 10 seconds before retrying... This probably isn't the fanciest way and may
	// need to be revamp in the future.
    time.Sleep(time.Second * 10) // POSSIBLE REVAMP NEEDED WHEN GOING TO PROD

	// Attempt to see if file exists maxattempt times recursively
	return fileexistattempt(maxattempt-1, v, i)
}

// A public function that acts like a Getter
func GetEnv(key string) string {

	// If the configuration file is good, use the config file
	if State == configgood {

		var value = envVars.GetString(key)

		// Even if the config file is load, if the key doesn't exist in config
		// try the environment. This technically makes the application mutable 
        // and same as before. But put this here for now b/c there are environ
        // variables that are not in config.
        if value == "" {
            // Copy it over to config to prevent additional OS calls.
            // Remember to delete both config mem and environ var when unsetenv!
            value = os.Getenv(key)
            os.Setenv(key, value)
        }

		return value
	}

    // Config file not good, so default to environment... boo >:(
	return os.Getenv(key)

}

// A public function that wraps GenEnv and returns a bool if string is not empty
func LookupEnv(key string) (string, bool) {
    
    if String := GetEnv(key); String != "" {
        return String, true
    }

    return "", false

}

// A public function that acts like a Setter. This function should only be used
// in testing. Protect parameter is an interface, but it's really a testing
// T struct. Any other type entered for protect will panic.
func SetEnv(protect interface{}, key string, value string) error {

    var err error

	// Check if the protect type is Testing T struct
	if _, ok := protect.(*testing.T); ok {
		// If config is good, change the config in memory
		if State == configgood {
			envVars.Set(key, value) // This doesn't return anything...
		} else {
			// Config is bad, change the EV
			err = os.Setenv(key, value)
		}
	} else {
		// Not a testing T struct, most likely not in testing... PANIC!
		panic("You cannot use SetEnv function outside testing!")
	}

    return err

}

// A public function that unsets a variable. This function should also only be
// used in testing.
func UnsetEnv(protect interface{}, key string) error {
    var err error

	// Check if the protect type is Testing T struct
	if _, ok := protect.(*testing.T); ok {
		// If config is good, change the config in memory
		if State == configgood {
            // Why? See line 152
			err = os.Unsetenv(key)
            envVars.Set(key, "")
		} else {
			// Config is bad, change the EV
			err = os.Unsetenv(key)
		}
	} else {
		// Not a testing T struct, most likely not in testing... PANIC!
		panic("You cannot use UnsetEnv function outside testing!")
	}

    return err
}
