// +build tools

package main

import (
	// bcda/worker/ssas dependencies
	_ "github.com/BurntSushi/toml"
	_ "github.com/go-delve/delve/cmd/dlv"
	_ "github.com/howeyc/fsnotify"
	_ "github.com/mattn/go-colorable"

	// end bcda/worker/ssas dependencies

	// test dependencies
	_ "github.com/securego/gosec/cmd/gosec"
	_ "github.com/tsenart/vegeta"
	_ "github.com/xo/usql"
	_ "gotest.tools/gotestsum"
	// end test dependencies
)
