#!/bin/bash
#
# This script is intended to be run from within the Docker "smoke_test" container
#
set -e
echo "Running EOB Encrypted"
go run bcda_client.go -host=api:3000 -clientID=$CLIENT_ID -clientSecret=$CLIENT_SECRET -endpoint=ExplanationOfBenefit
echo "Running Patient Encrypted"
go run bcda_client.go -host=api:3000 -clientID=$CLIENT_ID -clientSecret=$CLIENT_SECRET -endpoint=Patient
echo "Running Coverage Encrypted"
go run bcda_client.go -host=api:3000 -clientID=$CLIENT_ID -clientSecret=$CLIENT_SECRET -endpoint=Coverage
echo "Running EOB Unencrypted"
go run bcda_client.go -host=api:3000 -clientID=$CLIENT_ID -clientSecret=$CLIENT_SECRET -endpoint=ExplanationOfBenefit -encrypt=false
echo "Running Patient Unencrypted"
go run bcda_client.go -host=api:3000 -clientID=$CLIENT_ID -clientSecret=$CLIENT_SECRET -endpoint=Patient -encrypt=false
echo "Running Coverage Unencrypted"
go run bcda_client.go -host=api:3000 -clientID=$CLIENT_ID -clientSecret=$CLIENT_SECRET -endpoint=Coverage -encrypt=false