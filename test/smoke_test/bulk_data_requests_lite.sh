#!/bin/bash
#
# This script is intended to be run from within the Docker "smoke_test" container
#
set -e

PIDS=()

echo "Running Patient All" && \
go run bcda_client.go -host=api:3000 -clientID=$CLIENT_ID -clientSecret=$CLIENT_SECRET -endpoint=Patient -apiVersion=v2 &
PIDS+=($!)
echo "Running Group All" && \
go run bcda_client.go -host=api:3000 -clientID=$CLIENT_ID -clientSecret=$CLIENT_SECRET -endpoint=Group/all -apiVersion=v2  &
PIDS+=($!)
echo "Running Group Runout (EOB resource)" && \
go run bcda_client.go -host=api:3000 -clientID=$CLIENT_ID -clientSecret=$CLIENT_SECRET -endpoint=Group/runout -resourceType=ExplanationOfBenefit -apiVersion=v2  &
PIDS+=($!)

for PID in "${PIDS[@]}"; do
   wait "$PID"
done

