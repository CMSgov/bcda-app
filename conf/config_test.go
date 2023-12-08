package conf

import (
	"os"
	"reflect"
	"testing"

	"github.com/CMSgov/bcda-app/bcda/constants"
	"github.com/stretchr/testify/assert"
)

func TestGetEnv(t *testing.T) {
	type args struct {
		key string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			"Single Value",
			args{"TEST_HELLO"},
			"world",
		},
		{
			"Multi-value separated by commas",
			args{"TEST_LIST"},
			constants.TestListData,
		},
		{
			"Path",
			args{"TEST_SOMEPATH"},
			"../../FAKE/PATH",
		},
		{
			"Number",
			args{"TEST_NUM"},
			"1234",
		},
		{
			"Boolean",
			args{"TEST_BOOL"},
			"true",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.EqualValues(t, tt.want, GetEnv(tt.args.key))
		})
	}
}

func TestSetEnv(t *testing.T) {
	type args struct {
		protect *testing.T
		key     string
		value   string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			"Change Value",
			args{t, "TEST_SOMEPATH", "../somepath"},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := SetEnv(tt.args.protect, tt.args.key, tt.args.value); (err != nil) != tt.wantErr {
				t.Errorf("SetEnv() error = %v, wantErr %v", err, tt.wantErr)
			}
			assert.Equal(t, tt.args.value, GetEnv(tt.args.key))
		})
	}
}

func TestUnsetEnv(t *testing.T) {
	type args struct {
		protect *testing.T
		key     string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			"Remove Value",
			args{t, "TEST_HELLO"},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := UnsetEnv(tt.args.protect, tt.args.key); (err != nil) != tt.wantErr {
				t.Errorf("UnsetEnv() error = %v, wantErr %v, %v", err, tt.wantErr, state)
			}
			assert.Empty(t, GetEnv(tt.args.key))
			assert.Empty(t, os.Getenv(tt.args.key))
		})
	}
}

func TestLoadConfigs(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			"See if Viper sets up correctly",
			[]string{os.Getenv("GOPATH") + "/src/github.com/CMSgov/bcda-app/conf/test/local.env"},
			"true",
		},
		{
			"test multiple locations for config files",
			[]string{os.Getenv("GOPATH") + "/src/github.com/CMSgov/bcda-app/conf/test/local.env", os.Getenv("GOPATH") + "/src/github.com/CMSgov/bcda-app/conf/test/notlocal.env"},
			"false",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, state := loadConfigs(tt.args...)
			if !reflect.DeepEqual(v.Get("TEST").(string), tt.want) {
				t.Errorf("setup() = %v, want %v", state, tt.want)
			}
		})
	}
}

func TestGetConfigPaths(t *testing.T) {

	tests := []struct {
		name       string
		expected   []string
		envPath    string
		workerPath string
		apiPath    string
	}{
		{
			"Test case: no config paths set",
			[]string{""},
			"",
			"",
			"",
		},
		{
			"Test case: all three paths set",
			[]string{"/Somedir/foo/bar", "/test/helloworld", os.Getenv("GOPATH") + "/src/github.com/CMSgov/bcda-app/conf/test/.env"},
			os.Getenv("GOPATH") + "/src/github.com/CMSgov/bcda-app/conf/test/.env",
			"/Somedir/foo/bar",
			"/test/helloworld",
		},
		{
			"Test case: env file found, no api or worker dir found",
			[]string{os.Getenv("GOPATH") + "/src/github.com/CMSgov/bcda-app/conf/test/local.env"},
			os.Getenv("GOPATH") + "/src/github.com/CMSgov/bcda-app/conf/test/local.env",
			"",
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paths := configPaths(tt.envPath, tt.workerPath, tt.apiPath)
			assert.Equal(t, tt.expected, paths)
		})
	}
}

func Test_findEnv(t *testing.T) {
	type args struct {
		location []string
	}
	tests := []struct {
		name  string
		args  args
		want  bool
		want1 string
	}{
		{
			"Test for local",
			args{[]string{os.Getenv("GOPATH") + constants.TestConfPath, os.Getenv("GOPATH") + constants.TestFakePath}},
			true,
			os.Getenv("GOPATH") + constants.TestConfPath,
		},
		{
			"Test for prod (Doesn't exist yet)",
			args{[]string{os.Getenv("GOPATH") + constants.TestFakePath, os.Getenv("GOPATH") + constants.TestConfPath}},
			true,
			os.Getenv("GOPATH") + constants.TestConfPath,
		},
		{
			"Test for both not existing",
			args{[]string{os.Getenv("GOPATH") + constants.TestFakePath, os.Getenv("GOPATH") + constants.TestFakePath}},
			false,
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := findEnv(tt.args.location)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.want1, got1)
		})
	}
}

func TestLookupEnv(t *testing.T) {
	type args struct {
		key string
	}
	tests := []struct {
		name  string
		args  args
		want  string
		want1 bool
	}{
		{
			"Query a variable that exists in local.env but does not have value",
			args{"TEST_EMPTY"},
			"",
			true,
		},
		{
			"Query a variable that does not exist",
			args{"TEST_DOESNOTEXIST"},
			"",
			false,
		},
		{
			"Query a variable that exists but was unset",
			args{"TEST_CHANGE"},
			"",
			false,
		},
		{
			"Query a variable that only exist as environment var and not conf",
			args{"TEST_EVONLY"},
			"",
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			if tt.args.key == "TEST_CHANGE" {
				// For this test, unset
				var _ = UnsetEnv(t, tt.args.key)
			}

			if tt.args.key == "TEST_EVONLY" {
				os.Setenv("TEST_EVONLY", "")
			}

			got, got1 := LookupEnv(tt.args.key)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.want1, got1)
		})
	}
}

type inner struct {
	TestValue2 string `conf:"TestValue2" conf_default:"ABC"`
	TEST_NUM   string
}

type outer struct {
	TEST_LIST        string
	Test_tag         string `conf:"TEST_LIST"`
	TEST_SOMEPATH    string `conf:"-"`
	TestDefaultValue int    `conf:"TEST_DEFAULT_VALUE" conf_default:"123"`
	TestValue1       int
	inner            `conf:",squash"`
	Inner            struct {
		A string `conf:"A"`
	} `conf:"INNER"`
	A string `conf:"A"`
}

func TestCheckout(t *testing.T) {
	t.Cleanup(func() {
		assert.NoError(t, os.Unsetenv("A"))
		assert.NoError(t, os.Unsetenv("INNER.A"))
	})
	assert.NoError(t, os.Setenv("A", "DEF"))
	assert.NoError(t, os.Setenv("INNER.A", "GHI"))
	t.Run("Traversing the nested struct", func(t *testing.T) {
		testStruct := outer{}
		err := Checkout(testStruct)
		assert.Error(t, err)

		assert.NoError(t, Checkout(&testStruct))
		assert.Equal(t, testStruct.TEST_NUM, "1234")
		assert.Equal(t, testStruct.TEST_LIST, constants.TestListData)
		assert.Equal(t, testStruct.TestValue1, 0)
		assert.Equal(t, testStruct.TestValue2, "ABC")
		// Check if tags work
		assert.Equal(t, testStruct.Test_tag, constants.TestListData)
		// Check if explicit skip of tag works
		assert.Equal(t, testStruct.TEST_SOMEPATH, "")
		assert.Equal(t, testStruct.TestDefaultValue, 123, "Default value should be honored")
		assert.Equal(t, testStruct.A, "DEF")
		assert.Equal(t, testStruct.Inner.A, "GHI")
	})

	t.Run("Traversing a slice of strings.", func(t *testing.T) {
		testSlice := []string{"some", "SOME", "TEST_LIST"}
		err := Checkout(&testSlice)

		// Check if reference to a slice is rejected, since a slice is already a pointer
		assert.Error(t, err)

		assert.NoError(t, Checkout(testSlice))
		assert.Equal(t, testSlice[0], "")
		assert.Equal(t, testSlice[1], "")
		assert.Equal(t, testSlice[2], constants.TestListData)
	})
}
