package runner

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/howeyc/fsnotify"
)

func watch() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		fatal(err)
	}

	go func() {
		for {
			select {
			case ev := <-watcher.Event:
				if isWatchedFile(ev.Name) && !ev.IsAttrib() {
					watcherLog("sending event %s", ev)
					startChannel <- ev.String()
				}
			case err := <-watcher.Error:
				watcherLog("error: %s", err)
			}
		}
	}()

	for _, p := range settings.WatchPaths {
		p, _ = filepath.Abs(p)
		filepath.Walk(p, func(path string, info os.FileInfo, err error) error {
			if info.IsDir() {
				if isExcluded(path) {
					return filepath.SkipDir
				}
				if len(path) > 1 && strings.HasPrefix(filepath.Base(path), ".") {
					return filepath.SkipDir
				}
				watcherLog("Watching %s", path)
				watcher.Watch(path)
			}
			return err
		})
	}
}
