package runner

import (
	"os"
	"os/exec"
	"strings"
)

func run() bool {
	runnerLog("Running...")

	cmd := exec.Command(settings.OutputBinary, settings.RunArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Start()
	if err != nil {
		fatal(err)
	}
	runnerLog(strings.Repeat("-", 20))

	go func() {
		<-stopChannel
		pid := cmd.Process.Pid
		runnerLog("Killing PID %d", pid)
		cmd.Process.Kill()
	}()

	return true
}
