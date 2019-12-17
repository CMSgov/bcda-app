#!/bin/bash
#
# This script is intended to be run from within the Docker "performance_test" container
#
set -e

echo echo "Running EOB"
go run performance.go -host=api:3000 -resourceType=ExplanationOfBenefit -endpoint=Patient
echo "Running Patient"
go run performance.go -host=api:3000 -resourceType=Patient -endpoint=Patient
echo "Running Coverage"
go run performance.go -host=api:3000 -resourceType=Coverage -endpoint=Patient
echo "Running Patient All"
go run performance.go -host=api:3000 -endpoint=Patient
echo "Running Group All"
go run performance.go -host=api:3000 -endpoint=Group
