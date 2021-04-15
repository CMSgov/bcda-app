#!/usr/bin/env bash
set -euo pipefail
# Incomplete inotify support on Docker for Mac can cause entr to respond inconsistently.
# entr includes a workaround by setting the following enviornment variable
# https://github.com/eradman/entr#docker-and-windows-subsystem-for-linux
export ENTR_INOTIFY_WORKAROUND=true
# Watch all go extension files with entr and execute go run so it will recompile and 
# start the docker service up again if changes are detected.
entr_cmd="go install -v && ${@:2}"
ci=${CI:-false} # Default to not running in continuous integration
if [ "$1" == "api" ]; then
    if [ "$ci" == "true" ]; then
        "${@:2}"
    else
        echo "Starting bcda api entr watcher..."
        find . ../bcdaworker -name '*.go' | entr -nrs "$entr_cmd"
    fi
fi
if [ "$1" == "worker" ]; then
    if [ "$ci" == "true" ]; then
        "${@:2}"
    else
        echo "Starting bcda worker entr watcher..."
        find . ../bcda -name '*.go' | entr -nrs "$entr_cmd"
    fi
fi