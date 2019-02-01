#!/bin/bash
#
# This script is intended to be run from within the Docker "performance_test" container
#
set -e

go run performance.go -host=api:3000 -endpoint=ExplanationOfBenefit
go run performance.go -host=api:3000 -endpoint=Patient
go run performance.go -host=api:3000 -endpoint=Coverage
go run performance.go -host=api:3000 -endpoint=ExplanationOfBenefit -encrypt=false
go run performance.go -host=api:3000 -endpoint=Patient -encrypt=false
go run performance.go -host=api:3000 -endpoint=Coverage -encrypt=false