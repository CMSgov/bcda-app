package config

import (
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
		protect interface{}
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
		})
	}
}
