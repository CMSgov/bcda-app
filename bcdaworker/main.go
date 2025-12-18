package main

import (
	"fmt"
	"os"

	"github.com/CMSgov/bcda-app/bcdaworker/cli"
	"github.com/CMSgov/bcda-app/log"
)

func main() {
	app := cli.GetApp()
	err := app.Run(os.Args)
	if err != nil {
		// Since the logs may be routed to a file,
		// ensure that the error makes it at least once to stdout
		fmt.Printf("Error occurred while executing app.Run from main, err %s\n", err)
		log.Worker.Fatal(err)
	}
}
