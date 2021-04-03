#!/usr/bin/env bash

set -euo pipefail

# Incomplete inotify support on Docker for Mac can cause entr to respond inconsistently.
# entr includes a workaround by setting the following enviornment variable
# https://github.com/eradman/entr#docker-and-windows-subsystem-for-linux
export ENTR_INOTIFY_WORKAROUND=true

# Watch all go extension files with entr and execute go run so it will recompile and 
# start the docker service up again if changes are detected.
if [ "$1" == "api" ]; then
    echo "Starting bcda api entr watcher..."
    find . -name '*.go' | entr -nr go run -v "${@:2}"
fi

if [ "$1" == "worker" ]; then
    echo "Starting bcda worker entr watcher..."
    find . ../bcda -name '*.go' | entr -nr sh -c "go install -v && bcdaworker"
fi
