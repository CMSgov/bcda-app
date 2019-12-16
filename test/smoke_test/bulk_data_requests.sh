#!/bin/bash
#
# This script is intended to be run from within the Docker "smoke_test" container
#
set -e
echo "Running EOB"
go run bcda_client.go -host=api:3000 -clientID=$CLIENT_ID -clientSecret=$CLIENT_SECRET -endpoint=Patient -resourceType=ExplanationOfBenefit
echo "Running Patient"
go run bcda_client.go -host=api:3000 -clientID=$CLIENT_ID -clientSecret=$CLIENT_SECRET -endpoint=Patient -resourceType=Patient
echo "Running Coverage"
go run bcda_client.go -host=api:3000 -clientID=$CLIENT_ID -clientSecret=$CLIENT_SECRET -endpoint=Patient -resourceType=Coverage
echo "Running Patient All"
go run bcda_client.go -host=api:3000 -clientID=$CLIENT_ID -clientSecret=$CLIENT_SECRET -endpoint=Patient
echo "Running Patient, Coverage, EOB (explicitly)"
go run bcda_client.go -host=api:3000 -clientID=$CLIENT_ID -clientSecret=$CLIENT_SECRET -endpoint=Patient -resourceType=Patient,Coverage,ExplanationOfBenefit
echo "Running Patient, Coverage"
go run bcda_client.go -host=api:3000 -clientID=$CLIENT_ID -clientSecret=$CLIENT_SECRET -endpoint=Patient -resourceType=Patient,Coverage
echo "Running Patient, EOB"
go run bcda_client.go -host=api:3000 -clientID=$CLIENT_ID -clientSecret=$CLIENT_SECRET -endpoint=Patient -resourceType=Patient,ExplanationOfBenefit
echo "Running Coverage, EOB"
go run bcda_client.go -host=api:3000 -clientID=$CLIENT_ID -clientSecret=$CLIENT_SECRET -endpoint=Patient -resourceType=Coverage,ExplanationOfBenefit
echo "Running Group All"
go run bcda_client.go -host=api:3000 -clientID=$CLIENT_ID -clientSecret=$CLIENT_SECRET -endpoint=Group