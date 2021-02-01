package conf

import (
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
			args{"/go/src/github.com/CMSgov/bcda-app/conf/test"},
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
			args{[]string{"/go/src/github.com/CMSgov/bcda-app/conf/test", "/go/src/github.com/CMSgov/bcda-app/conf/FAKE"}},
			true,
			"/go/src/github.com/CMSgov/bcda-app/conf/test",
		},
		{
			"Test for prod (Doesn't exist yet)",
			args{[]string{"/go/src/github.com/CMSgov/bcda-app/conf/FAKE", "/go/src/github.com/CMSgov/bcda-app/conf/test"}},
			true,
			"/go/src/github.com/CMSgov/bcda-app/conf/test",
		},
		{
			"Test for both not existing",
			args{[]string{"/go/src/github.com/CMSgov/bcda-app/conf/FAKE", "/go/src/github.com/CMSgov/bcda-app/conf/FAKE"}},
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
			if got != tt.want {
				t.Errorf("LookupEnv() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("LookupEnv() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
