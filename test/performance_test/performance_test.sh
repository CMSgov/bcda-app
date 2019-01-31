#!/bin/bash
#
# This script is intended to be run from within the Docker "performance_test" container
#
set -e

go run performance.go --endpoint=Coverage --encrypt=false