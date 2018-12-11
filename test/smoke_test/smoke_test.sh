#!/bin/bash
#
# This script is intended to be run from within the Docker "smoke_test" container
#
set -e

go run bcda_client.go -host=api:3000 -endpoint=ExplanationOfBenefit
go run bcda_client.go -host=api:3000 -endpoint=Patient
