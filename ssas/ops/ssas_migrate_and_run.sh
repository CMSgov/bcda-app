#!/bin/bash
#
# This script is intended to be run from within the Docker "ssas" container

echo "Migrating ..."
cd /go/src/github.com/CMSgov/bcda-app/ssas
migrate -database $DATABASE_URL -path db/migrations up

echo "Starting SSAS..."
fresh -o ssas-service -p ./service/main -r --start