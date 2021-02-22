package conf

import (
	"fmt"
	"os"
	"reflect"
	"testing"
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
		{ // Test Case #1
			"Single Value",
			args{"TEST_HELLO"},
			"world",
		},
		{ // Test Case #2
			"Multi-value separated by commas",
			args{"TEST_LIST"},
			"One,Two,Three,Four",
		},
		{ // Test Case #3
			"Path",
			args{"TEST_SOMEPATH"},
			"../../FAKE/PATH",
		},
		{ // Test Case #4
			"Number",
			args{"TEST_NUM"},
			"1234",
		},
		{ // Test Case #5
			"Boolean",
			args{"TEST_BOOL"},
			"true",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetEnv(tt.args.key); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetEnv() = %v, want %v", got, tt.want)
			}
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
			if val := GetEnv(tt.args.key); val != tt.args.value {
				t.Errorf("New value entered (%v) into conf does not match value provided.", tt.args.value)
			}
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
			if val := GetEnv(tt.args.key); val != "" {
				t.Errorf("UnsetEnv did not clear the key from conf. Value is %v", val)
			}
			if val := os.Getenv(tt.args.key); val != "" {
				t.Errorf("UnsetEnv did not clear the key from EV. Value is %v", val)
			}
		})
	}
}

func Test_setup(t *testing.T) {
	type args struct {
		dir string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			"See if Viper sets up correctly",
			args{envVars.gopath + "/src/github.com/CMSgov/bcda-app/conf/test"},
			"true",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var v = setup(tt.args.dir)
			if !reflect.DeepEqual(v.Get("TEST").(string), tt.want) {
				t.Errorf("setup() = %v, want %v", state, tt.want)
			}
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
			args{[]string{envVars.gopath + "/src/github.com/CMSgov/bcda-app/conf/test", envVars.gopath + "/src/github.com/CMSgov/bcda-app/conf/FAKE"}},
			true,
			envVars.gopath + "/src/github.com/CMSgov/bcda-app/conf/test",
		},
		{
			"Test for prod (Doesn't exist yet)",
			args{[]string{envVars.gopath + "/src/github.com/CMSgov/bcda-app/conf/FAKE", envVars.gopath + "/src/github.com/CMSgov/bcda-app/conf/test"}},
			true,
			envVars.gopath + "/src/github.com/CMSgov/bcda-app/conf/test",
		},
		{
			"Test for both not existing",
			args{[]string{envVars.gopath + "/src/github.com/CMSgov/bcda-app/conf/FAKE", envVars.gopath + "/src/github.com/CMSgov/bcda-app/conf/FAKE"}},
			false,
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := findEnv(tt.args.location)
			if got != tt.want {
				t.Errorf("findEnv() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("findEnv() got1 = %v, want %v", got1, tt.want1)
			}
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
		//{
		//"Query a variable that exists in local.env but does not have value",
		//args{"TEST_EMPTY"},
		//"",
		//true,
		//},
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
			if got != tt.want {
				t.Errorf("LookupEnv() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("LookupEnv() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

type inner struct {
	TestValue2 string
	TEST_NUM   string
}

type outer struct {
	TEST_LIST     string
	Test_tag      string `conf:"TEST_LIST"`
	TEST_SOMEPATH string `conf:"-"`
	TestValue1    int
	inner
}

func TestCheckout(t *testing.T) {

	t.Run("Traversing the nested struct", func(t *testing.T) {
		testStruct := outer{}
		err := Checkout(testStruct)
		// Check if copt of a struct is rejected
		if err == nil {
			t.Errorf("A copy of a struct was accepted.")
		}
		_ = Checkout(&testStruct)
		// Check if the appropriate fields were updated
		if val := testStruct.TEST_NUM; val != "1234" {
			t.Errorf("Wanted: %v Got: %v", "1234", val)
		}
		if val := testStruct.TEST_LIST; val != "One,Two,Three,Four" {
			t.Errorf("Wanted: %v Got: %v", "One,Two,Three,Four", val)
		}
		if val := testStruct.TestValue1; val != 0 {
			t.Errorf("Wanted: %v Got: %v", 0, val)
		}
		if val := testStruct.TestValue2; val != "" {
			t.Errorf("Wanted: %v Got: %v", "", val)
		}
		// Check if tags work
		if val := testStruct.Test_tag; val != "One,Two,Three,Four" {
			t.Errorf("Wanted: %v Got: %v", "One,Two,Three,Four", val)
		}
		// Check if explicit skip of tag works
		if val := testStruct.TEST_SOMEPATH; val != "" {
			t.Errorf("Wanted: %v Got: %v", "c", val)
		}
	})

	t.Run("Traversing a slice of strings.", func(t *testing.T) {
		testSlice := []string{"some", "SOME", "TEST_LIST"}
		err := Checkout(&testSlice)
		// Check if reference to a slice is rejected, since a slice is already a pointer
		if err == nil {
			t.Errorf("A reference to a slice string was accepted.")
		}
		_ = Checkout(testSlice)
		fmt.Println(testSlice)
		if val := testSlice[0]; val != "" {
			t.Errorf("Wanted: %v Got: %v", "", val)
		}
		if val := testSlice[1]; val != "" {
			t.Errorf("Wanted: %v Got: %v", "", val)
		}
		if val := testSlice[2]; val != "One,Two,Three,Four" {
			t.Errorf("Wanted: %v Got: %v", "One,Two,Three,Four", val)
		}
	})
}
