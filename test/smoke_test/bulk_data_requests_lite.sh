#!/bin/bash
#
# This script is intended to be run from within the Docker "smoke_test" container
#
set -e

pids=()

echo "Running Patient All" && \
go run bcda_client.go -host=api:3000 -clientID=$CLIENT_ID -clientSecret=$CLIENT_SECRET -endpoint=Patient &
pids+=($!)

echo "Running Group All" && \
go run bcda_client.go -host=api:3000 -clientID=$CLIENT_ID -clientSecret=$CLIENT_SECRET -endpoint=Group/all &
pids+=($!)

echo "Running Group Runout (EOB resource)" && \
go run bcda_client.go -host=api:3000 -clientID=$CLIENT_ID -clientSecret=$CLIENT_SECRET -endpoint=Group/runout -resourceType=ExplanationOfBenefit &
pids+=($!)


for pid in "${pids[@]}"; do
   wait "$pid"
done
