package runner

import (
	"io"
	"io/ioutil"
	"os"
	"os/exec"
)

func build() (string, bool) {
	buildLog("Building...")
	var args = []string{
		"build",
		"-o",
		settings.OutputBinary,
	}

	if settings.BuildArgs != "" {
		args = append(args, settings.BuildArgs)
	}

	if settings.Root != "" {
		args = append(args, settings.Root)
	}

	cmd := exec.Command("go", args...)

	stderr, err := cmd.StderrPipe()
	if err != nil {
		fatal(err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fatal(err)
	}

	err = cmd.Start()
	if err != nil {
		fatal(err)
	}

	io.Copy(os.Stdout, stdout)
	errBuf, _ := ioutil.ReadAll(stderr)

	err = cmd.Wait()
	if err != nil {
		buildLog("Failed: \n%s", errBuf)
		return string(errBuf), false
	}

	return "", true
}
