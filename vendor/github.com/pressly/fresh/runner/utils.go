package runner

import (
	"os"
	"path/filepath"
	"strings"
)

func initFolders() {
	runnerLog("InitFolders")
	runnerLog("mkdir %s", settings.TmpPath)
	err := os.Mkdir(settings.TmpPath, 0755)
	if err != nil {
		runnerLog(err.Error())
	}
}

func isExcluded(path string) bool {
	absPath, _ := filepath.Abs(path)
	for _, excl := range settings.ExcludePaths {
		absExclPath, _ := filepath.Abs(excl)
		if absPath == absExclPath || strings.HasPrefix(absPath, absExclPath+"/") {
			return true
		}
	}
	return false
}

func isValidExt(path string) bool {
	ext := filepath.Ext(path)

	for _, e := range settings.ValidExtensions {
		if strings.TrimSpace(e) == ext {
			return true
		}
	}
	return false
}

func isWatchedFile(path string) bool {
	if isExcluded(path) {
		return false
	}

	return isValidExt(path)
}

func createBuildErrorsLog(message string) bool {
	file, err := os.Create(settings.BuildErrorPath)
	if err != nil {
		return false
	}

	_, err = file.WriteString(message)
	if err != nil {
		return false
	}

	return true
}

func removeBuildErrorsLog() error {
	err := os.Remove(settings.BuildErrorPath)

	return err
}
