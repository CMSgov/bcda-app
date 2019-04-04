#!/bin/bash
#
# This script is intended to be run from within the Docker "smoke_test" container
#
set -e
echo "Running EOB Encrypted"
go run bcda_client.go -host=api:3000 -endpoint=ExplanationOfBenefit
echo "Running Patient Encrypted"
go run bcda_client.go -host=api:3000 -endpoint=Patient
echo "Running Coverage Encrypted"
go run bcda_client.go -host=api:3000 -endpoint=Coverage
echo "Running EOB Unencrypted"
go run bcda_client.go -host=api:3000 -endpoint=ExplanationOfBenefit -encrypt=false
echo "Running Patient Unencrypted"
go run bcda_client.go -host=api:3000 -endpoint=Patient -encrypt=false
echo "Running Coverage Unencrypted"
go run bcda_client.go -host=api:3000 -endpoint=Coverage -encrypt=false