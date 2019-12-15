#!/bin/bash
#
# This script is intended to be run from within the Docker "performance_test" container
#
set -e

echo echo "Running EOB"
go run performance.go -host=api:3000 -resourceType=ExplanationOfBenefit -endpointBase=Patient
echo "Running Patient"
go run performance.go -host=api:3000 -resourceType=Patient -endpointBase=Patient
echo "Running Coverage"
go run performance.go -host=api:3000 -resourceType=Coverage -endpointBase=Patient
echo "Running Patient All"
go run performance.go -host=api:3000 -endpointBase=Patient
echo "Running Group All"
go run performance.go -host=api:3000 -endpointBase=Group
