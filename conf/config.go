package conf

/*
   This is a package that wraps viper, a package designed to handle configuration
   files, for BCDA and BCDAWORKER. This package will go through a number of stages.

   1. Local env looks primarily at conf package for variables, but will also look
   in the environment for any variables it is not tracking. PROD/TEST/DEV will
   only look in the environment. (WE ARE HERE NOW)

   2. Local env will only look at conf package. PROD/TEST/DEV still the same.

   3. All env will look at the conf package.

   4. Make conf package less ubiquitous by implementing some solution that removes
   all the repeated calls to the conf package for information.

   Assumptions:
   1. The configuration file is a env file (can be changed if needed).
   2. The configuration variables, once it has been ingested by the conf package,
   will stay immutable during the uptime of the application (exception is test).
*/

import (
	"errors"
	"go/build"
	"os"
	"reflect"
	"testing"

	"github.com/spf13/viper"
)

// Private global variable:

// config is a conf package struct that wraps the viper struct with one other field
type config struct {
	gopath string
	viper.Viper
}

// envVars is an uninitialized config struct at this point, and it is private.
// It is initialized by the init func when the configuration file is found.
var envVars config

// Implementing a state machine that tracks the status of the configuration loading.
// This state machine should go away when in stage 3.
type configStatus uint8

const (
	configGood    configStatus = 0
	configBad     configStatus = 1
	noConfigFound configStatus = 2
)

// if configuration file found and loaded, state doesn't changed
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

	// Find the gopath on the machine running the application
	gopath := os.Getenv("GOPATH")

	if gopath == "" {
		gopath = build.Default.GOPATH
	}

	// Possible configuration file locations: local and PROD/DEV/TEST respectfully.
	var locationSlice = []string{
		gopath + "/src/github.com/CMSgov/bcda-app/shared_files/decrypted",
		// Placeholder for configuration location for TEST/DEV/PROD once available.
	}

	if success, loc := findEnv(locationSlice[:]); success {
		// A configuration file found, initialize config struct
		envVars = config{gopath, *setup(loc)}
	} else {
		// Checked both locations, no configuration file found
		state = noConfigFound
	}

}

/*
   findEnv is a helper function that will determine what environment the application
   is running in: local or PROD/TEST/DEV env. Each environment should have a distinct
   path where the configuration file is located. First it check the local path,
   then the PROD/DEV/TEST. If both not found, defaults to just using environment vars.

   Later iterations of this package will phase out this "defaulting" behavior.
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
// This function will be phased out in later versions of the package.
func GetEnv(key string) string {

	// If the configuration file is loaded correctly, check config struct for info
	if state == configGood {

		if value := envVars.GetString(key); value != "" {
			return value
		} else {
			// if it is blank, check environment variables. Just in case there are
			// variables that started off empty and is now available. Once made available,
			// that variable is immutable for the rest of the runtime.
			// This functionality will be phased out in later versions of the package
			v, exist := os.LookupEnv(key)
			if exist {
				var _ = SetEnv(&testing.T{}, key, v)
			}
			return v
		}

	}

	// Configuration file not good, so default to environment variables.
	return os.Getenv(key)

}

// LookupEnv is a public function, like GetEnv, designed to replace os.LookupEnv() in code-base.
// This function will most likely become a private function in later versions of the package.
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
// This function will most likely become a private function in later versions of the package.
func SetEnv(protect *testing.T, key string, value string) error {

	var err error

	// If configuration file is good, change key value in conf struct
	if state == configGood {
		envVars.Set(key, value) // This doesn't return anything...
	} else {
		// Configuration file is bad, change the EV
		err = os.Setenv(key, value)
	}

	return err

}

// UnsetEnv is a public function that "unsets" a variable. Like SetEnv, this should only be used
// either in this package itself or testing.
// This function will most likely become a private function in later versions of the package.
func UnsetEnv(protect *testing.T, key string) error {
	var err error

	// If configuration file is good
	if state == configGood {
		envVars.Set(key, "")
	}

	// Unset environment variable too, because GetEnv would copy it back over when it should not.
	err = os.Unsetenv(key)

	return err

}

/***********************************************************************************************

    CODE BELOW THIS BLOCK IS EXPERIMENTAL AND ONLY BEING USED INTERNALLY BY CONF PACKAGE

***********************************************************************************************/

// Gopath function exposes the gopath of the application while keeping it immutable. Golang does
// not allow const to be strings, and a var would make it mutable if made public. This could be
// something the config / viper struct keeps track. For now it's separate.
//func Gopath() string {
	//return envVars.gopath
//}

/*
Checkout function takes a reference to a struct or a slice of string. It will traverse both
data structures and look up key value pairs in the conf / viper struct by the name of the field
for structs and string values for the slice. The function works with pointers so no value is
returned. An error is returned if the wrong data structure is provided.
*/
func Checkout(v interface{}) error {

	// Check if the data type provided is supported
	switch v := v.(type) {
	// If it's a slice of strings
	case []string:

		for n, key := range v {
			if val, exists := LookupEnv(key); exists {
				v[n] = val
			} else {
                v[n] = ""
            }
		}

		return nil
	// Checking the rest of data types through reflection
	default:
		// Get the concrete value from the interface
		check := reflect.ValueOf(v)
		// Is it a pointer?
		if check.Kind() == reflect.Ptr {

			// Dereference the pointer
			el := check.Elem()

			// Is the data type a struct?
			if el.Kind() == reflect.Struct {

                // Recursively walk the struct
				walk(el)

				return nil
			}
		}
	}

	return errors.New("The data type provided to Checkout func is not supported.")

}

// walk is a private function used by Checkout to recursively walk a possible nested struct.
func walk(field reflect.Value) {

    // Get the number of fields in the struct
	num := field.NumField()
    // Used to get Tag information
	fieldType := field.Type()

    // Traverse the struct
	for i := 0; i < num; i++ {
		innerField := field.Field(i)
        // Ensure the field is "exportable" and is a type string
		if innerField.CanInterface() && innerField.IsValid() && innerField.Type().Name() == "string" {
            // Get the field name
			name := fieldType.Field(i).Name
            // Tags are supported, but currently they do nothing
			_ = fieldType.Field(i).Tag.Get("conf")
            // Look up in the conf struct and set if possible
			if val, exists := LookupEnv(name); exists {
				innerField.SetString(val)
			}
        // Is the field of the struct a nested struct?
		} else if innerField.Kind() == reflect.Struct {
			walk(innerField)
		}
	}

}
